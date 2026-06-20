package tests

import (
	"net/http"
	"testing"

	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
)

// requestOTP hits /auth/request-otp and returns the mock OTP code.
func requestOTP(t *testing.T, phone string) string {
	t.Helper()
	clearOTPLimits(t, phone)
	resp := do(t, http.MethodPost, "/auth/request-otp", "", map[string]string{"phone": phone})
	if resp.Code != http.StatusOK {
		t.Fatalf("request-otp expected 200, got %d: %s", resp.Code, resp.Body)
	}
	var body struct {
		Message string `json:"message"`
		OTP     string `json:"otp"`
	}
	resp.JSON(t, &body)
	if body.OTP == "" {
		t.Fatalf("expected mock OTP in response, got %s", resp.Body)
	}
	return body.OTP
}

// POST /auth/request-otp
func TestRequestOTP(t *testing.T) {
	t.Run("valid phone returns mock otp", func(t *testing.T) {
		requestOTP(t, uniquePhone())
	})

	t.Run("missing phone -> 400", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/auth/request-otp", "", map[string]string{})
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})
}

// POST /auth/verify-otp
func TestVerifyOTP(t *testing.T) {
	resetDB(t)

	t.Run("new student login auto-creates user", func(t *testing.T) {
		phone := uniquePhone()
		code := requestOTP(t, phone)

		resp := do(t, http.MethodPost, "/auth/verify-otp", "", map[string]string{"phone": phone, "code": code})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var body struct {
			AccessToken  string      `json:"access_token"`
			RefreshToken string      `json:"refresh_token"`
			User         domain.User `json:"user"`
			ExpiresIn    int64       `json:"expires_in"`
		}
		resp.JSON(t, &body)
		if body.AccessToken == "" || body.RefreshToken == "" {
			t.Fatalf("expected tokens, got %s", resp.Body)
		}
		if body.User.Role != "student" {
			t.Fatalf("expected role student, got %q", body.User.Role)
		}
	})

	t.Run("admin phone gets admin role", func(t *testing.T) {
		code := requestOTP(t, adminPhone)
		resp := do(t, http.MethodPost, "/auth/verify-otp", "", map[string]string{"phone": adminPhone, "code": code})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var body struct {
			User domain.User `json:"user"`
		}
		resp.JSON(t, &body)
		if body.User.Role != "admin" {
			t.Fatalf("expected admin role for admin phone, got %q", body.User.Role)
		}
	})

	t.Run("wrong code -> 401", func(t *testing.T) {
		phone := uniquePhone()
		requestOTP(t, phone)
		resp := do(t, http.MethodPost, "/auth/verify-otp", "", map[string]string{"phone": phone, "code": "000000"})
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("missing fields -> 400", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/auth/verify-otp", "", map[string]string{"phone": uniquePhone()})
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("inactive user -> 403", func(t *testing.T) {
		phone := uniquePhone()
		// Pre-create a user, then force is_active=false. A plain struct Create
		// cannot do this: the `default:true` tag makes GORM substitute the DB
		// default for the bool zero value, so we update with raw SQL.
		u := domain.User{Phone: phone, Role: "student", IsActive: true}
		if err := testDB.Create(&u).Error; err != nil {
			t.Fatalf("seed user: %v", err)
		}
		if err := testDB.Exec("UPDATE users SET is_active = false WHERE id = ?", u.ID).Error; err != nil {
			t.Fatalf("deactivate user: %v", err)
		}
		code := requestOTP(t, phone)
		resp := do(t, http.MethodPost, "/auth/verify-otp", "", map[string]string{"phone": phone, "code": code})
		if resp.Code != http.StatusForbidden {
			t.Fatalf("expected 403 for inactive user, got %d: %s", resp.Code, resp.Body)
		}
	})
}

// POST /auth/refresh
func TestRefreshToken(t *testing.T) {
	resetDB(t)

	t.Run("valid refresh token returns new access token", func(t *testing.T) {
		userID, _ := createUser(t, "student")
		refresh, err := jwtSvc.GenerateRefreshToken(userID)
		if err != nil {
			t.Fatalf("mint refresh: %v", err)
		}
		resp := do(t, http.MethodPost, "/auth/refresh", "", map[string]string{"refresh_token": refresh})
		if resp.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body)
		}
		var body struct {
			AccessToken string `json:"access_token"`
		}
		resp.JSON(t, &body)
		if body.AccessToken == "" {
			t.Fatalf("expected access token, got %s", resp.Body)
		}
	})

	t.Run("invalid refresh token -> 401", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/auth/refresh", "", map[string]string{"refresh_token": "garbage.token.value"})
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("access token used as refresh -> 401", func(t *testing.T) {
		// Access token is signed with a different secret, so it must be rejected.
		_, access := createUser(t, "student")
		resp := do(t, http.MethodPost, "/auth/refresh", "", map[string]string{"refresh_token": access})
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 when using access token as refresh, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("refresh for deleted user -> 401", func(t *testing.T) {
		userID, _ := createUser(t, "student")
		refresh, _ := jwtSvc.GenerateRefreshToken(userID)
		if err := testDB.Exec("DELETE FROM users WHERE id = ?", userID).Error; err != nil {
			t.Fatalf("delete user: %v", err)
		}
		resp := do(t, http.MethodPost, "/auth/refresh", "", map[string]string{"refresh_token": refresh})
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for deleted user, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("missing body -> 400", func(t *testing.T) {
		resp := do(t, http.MethodPost, "/auth/refresh", "", map[string]string{})
		if resp.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d: %s", resp.Code, resp.Body)
		}
	})
}

// Auth middleware behaviour on a protected route.
func TestAuthMiddleware(t *testing.T) {
	t.Run("missing header -> 401", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/students/profile", "", nil)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", resp.Code, resp.Body)
		}
	})

	t.Run("malformed header -> 401", func(t *testing.T) {
		req := do(t, http.MethodGet, "/students/profile", "not-a-bearer-token", nil)
		if req.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", req.Code, req.Body)
		}
	})

	t.Run("invalid token -> 401", func(t *testing.T) {
		resp := do(t, http.MethodGet, "/students/profile", "aaa.bbb.ccc", nil)
		if resp.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", resp.Code, resp.Body)
		}
	})
}
