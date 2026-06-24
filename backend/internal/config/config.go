package config

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	phoneutil "github.com/yourusername/noshirvani-academy/backend/internal/phone"
)

type Config struct {
	DatabaseURL      string
	JWTSecret        string
	JWTRefreshSecret string
	JWTAccessTTL     int64
	JWTRefreshTTL    int64
	Environment      string
	OTPProvider      string
	ExposeMockOTP    bool
	UploadPath       string
	ServerAddr       string
	CORSOrigins      []string
	AdminPhones      map[string]bool
	SMSIRAPIKey      string
	SMSIRTemplateID  string
	RedisAddr        string
}

func Load() *Config {
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if refreshSecret == "" {
		refreshSecret = os.Getenv("JWT_SECRET")
	}

	cfg := &Config{
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		JWTRefreshSecret: refreshSecret,
		JWTAccessTTL:     getEnvDurationSeconds("JWT_ACCESS_TTL", 3600),
		JWTRefreshTTL:    getEnvDurationSeconds("JWT_REFRESH_TTL", 15*24*3600),
		Environment:      getEnv("ENVIRONMENT", "development"),
		OTPProvider:      getEnv("OTP_PROVIDER", "mock"),
		ExposeMockOTP:    getEnvBool("EXPOSE_MOCK_OTP", false),
		UploadPath:       getEnv("UPLOAD_PATH", "./uploads"),
		ServerAddr:       getEnv("SERVER_ADDR", ":8080"),
		CORSOrigins:      splitCSV(os.Getenv("CORS_ORIGINS")),
		AdminPhones:      phoneSet(os.Getenv("ADMIN_PHONES")),
		SMSIRAPIKey:      os.Getenv("SMSIR_API_KEY"),
		SMSIRTemplateID:  os.Getenv("SMSIR_TEMPLATE_ID"),
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
	}

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" || cfg.JWTRefreshSecret == "" {
		log.Fatal("JWT_SECRET and JWT_REFRESH_SECRET are required")
	}
	if cfg.Environment != "development" && len(cfg.CORSOrigins) == 0 {
		log.Fatal("CORS_ORIGINS is required outside development")
	}
	if cfg.OTPProvider != "mock" && cfg.OTPProvider != "smsir" {
		log.Fatal("OTP_PROVIDER must be either mock or smsir")
	}
	if cfg.Environment == "production" && cfg.OTPProvider == "mock" {
		log.Fatal("OTP_PROVIDER=mock is not allowed when ENVIRONMENT=production")
	}
	if cfg.Environment == "production" && cfg.ExposeMockOTP {
		log.Fatal("EXPOSE_MOCK_OTP=true is not allowed when ENVIRONMENT=production")
	}
	if cfg.OTPProvider == "smsir" && (cfg.SMSIRAPIKey == "" || cfg.SMSIRTemplateID == "") {
		log.Fatal("SMSIR_API_KEY and SMSIR_TEMPLATE_ID are required when OTP_PROVIDER=smsir")
	}

	return cfg
}

func getEnv(name, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(name string, fallback int64) int64 {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDurationSeconds(name string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
		return parsed
	}
	if strings.HasSuffix(value, "d") {
		days, err := strconv.ParseInt(strings.TrimSuffix(value, "d"), 10, 64)
		if err == nil {
			return int64((time.Duration(days) * 24 * time.Hour) / time.Second)
		}
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return int64(parsed / time.Second)
}

func getEnvBool(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	output := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			output = append(output, trimmed)
		}
	}
	return output
}

func phoneSet(value string) map[string]bool {
	phones := map[string]bool{}
	for _, rawPhone := range splitCSV(value) {
		normalized := phoneutil.Normalize(rawPhone)
		if normalized != "" {
			phones[normalized] = true
		}
	}
	return phones
}
