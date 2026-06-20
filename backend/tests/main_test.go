package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/yourusername/noshirvani-academy/backend/internal/config"
	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"github.com/yourusername/noshirvani-academy/backend/internal/infrastructure/auth"
	"github.com/yourusername/noshirvani-academy/backend/internal/infrastructure/database"
	"github.com/yourusername/noshirvani-academy/backend/internal/router"
)

// Shared test fixtures, initialised once in TestMain.
var (
	testDB     *gorm.DB
	testRouter *gin.Engine
	testCfg    *config.Config
	jwtSvc     *auth.JWTService

	// ipCounter hands every request a unique client IP so the global
	// in-memory rate limiter (60 req/min per IP) never trips during the suite.
	ipCounter uint32
)

const (
	defaultTestDB    = "postgres://postgres:postgres@localhost:5433/noshirvani_test?sslmode=disable"
	defaultTestRedis = "localhost:6379"
	adminPhone       = "09120000001"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	dsn := getenv("TEST_DATABASE_URL", defaultTestDB)
	db, err := database.NewPostgresDB(dsn)
	if err != nil {
		fmt.Printf("cannot connect to test database (%s): %v\n", dsn, err)
		os.Exit(1)
	}
	if err := database.AutoMigrate(db); err != nil {
		fmt.Printf("migration failed: %v\n", err)
		os.Exit(1)
	}
	testDB = db

	testCfg = &config.Config{
		DatabaseURL:      dsn,
		JWTSecret:        "test-access-secret",
		JWTRefreshSecret: "test-refresh-secret",
		JWTAccessTTL:     3600,
		JWTRefreshTTL:    1296000,
		OTPProvider:      "mock",
		UploadPath:       os.TempDir() + "/noshirvani_test_uploads",
		ServerAddr:       ":0",
		CORSOrigins:      []string{"http://localhost:3000"},
		AdminPhones:      map[string]bool{adminPhone: true},
		SMSIRAPIKey:      "", // mock SMS (just logs)
		SMSIRTemplateID:  "",
		RedisAddr:        getenv("TEST_REDIS_ADDR", defaultTestRedis),
	}

	jwtSvc = auth.NewJWTService(testCfg.JWTSecret, testCfg.JWTRefreshSecret, testCfg.JWTAccessTTL, testCfg.JWTRefreshTTL)
	testRouter = router.Setup(testDB, testCfg)

	code := m.Run()
	os.Exit(code)
}

// resetDB truncates every table so each test starts from a clean slate.
func resetDB(t *testing.T) {
	t.Helper()
	tables := []string{
		"mistakes", "subject_exams", "exams", "performance_histories",
		"dynamic_field_values", "dynamic_field_definitions", "blog_posts",
		"students", "users",
	}
	for _, tbl := range tables {
		if err := testDB.Exec("TRUNCATE TABLE " + tbl + " CASCADE").Error; err != nil {
			t.Fatalf("failed to truncate %s: %v", tbl, err)
		}
	}
}

// redisClient returns a client pointed at the same Redis the router uses.
func redisClient() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: testCfg.RedisAddr})
}

// clearOTPLimits removes any lingering OTP / rate-limit keys for a phone so
// OTP tests are deterministic regardless of leftover Redis state from prior runs.
func clearOTPLimits(t *testing.T, phone string) {
	t.Helper()
	rdb := redisClient()
	defer rdb.Close()
	ctx := context.Background()
	rdb.Del(ctx, "otp:"+phone, "otp:last:"+phone, "otp:rate:"+phone)
}

// uniquePhone returns a distinct, valid-looking phone for each call so that
// per-phone OTP rate limits never collide between tests.
func uniquePhone() string {
	n := atomic.AddUint32(&ipCounter, 1)
	// 0913 prefix keeps these distinct from the fixed adminPhone (0912...).
	return fmt.Sprintf("0913%07d", n)
}

// --- HTTP helpers ---------------------------------------------------------

type apiResponse struct {
	Code int
	Body []byte
}

func (r apiResponse) JSON(t *testing.T, dst interface{}) {
	t.Helper()
	if err := json.Unmarshal(r.Body, dst); err != nil {
		t.Fatalf("failed to decode response (%s): %v", string(r.Body), err)
	}
}

// do performs an HTTP request against the full router with a unique client IP.
func do(t *testing.T, method, path, token string, body interface{}) apiResponse {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal body: %v", err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}

	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	// Unique source IP per request -> dodge the global rate limiter.
	req.RemoteAddr = fmt.Sprintf("10.%d.%d.%d:12345",
		atomic.AddUint32(&ipCounter, 1)%256, ipCounter%256, ipCounter%251+1)

	rec := httptest.NewRecorder()
	testRouter.ServeHTTP(rec, req)
	return apiResponse{Code: rec.Code, Body: rec.Body.Bytes()}
}

// --- fixture builders -----------------------------------------------------

// createUser inserts a user row and returns its id and a freshly minted access token.
func createUser(t *testing.T, role string) (string, string) {
	t.Helper()
	user := domain.User{
		Phone:    uniquePhone(),
		Role:     role,
		IsActive: true,
	}
	if err := testDB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	token, err := jwtSvc.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		t.Fatalf("failed to mint token: %v", err)
	}
	return user.ID, token
}

// createStudent inserts a user + student profile and returns ids + token.
func createStudent(t *testing.T) (userID, studentID, token string) {
	t.Helper()
	userID, token = createUser(t, "student")
	student := domain.Student{
		UserID:    userID,
		FirstName: "Test",
		LastName:  "Student",
		Major:     "ریاضی",
	}
	if err := testDB.Create(&student).Error; err != nil {
		t.Fatalf("failed to create student: %v", err)
	}
	return userID, student.ID, token
}

// createAdmin returns an admin user id + token.
func createAdmin(t *testing.T) (string, string) {
	t.Helper()
	return createUser(t, "admin")
}

// mustField checks a JSON map contains an expected value.
func ymd(year int, month time.Month, day int) string {
	return fmt.Sprintf("%04d/%02d/%02d", year, int(month), day)
}

var _ = http.StatusOK // keep net/http imported for readability in test files
