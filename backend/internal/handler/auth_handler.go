package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"github.com/yourusername/noshirvani-academy/backend/internal/infrastructure/auth"
	"github.com/yourusername/noshirvani-academy/backend/internal/infrastructure/sms"
	"gorm.io/gorm"
)

type AuthHandler struct {
	db          *gorm.DB
	jwtService  *auth.JWTService
	otpStore    *sms.OTPStore
	otpProvider string
	exposeMockOTP bool
	adminPhones map[string]bool
}

// RequestOTPInput درخواست OTP را نشان می‌دهد
type RequestOTPInput struct {
	Phone string `json:"phone" binding:"required" example:"+989123456789" description:"شماره تلفن کاربر"`
}

// VerifyOTPInput تأیید OTP را نشان می‌دهد
type VerifyOTPInput struct {
	Phone string `json:"phone" binding:"required" example:"+989123456789" description:"شماره تلفن کاربر"`
	Code  string `json:"code" binding:"required" example:"123456" description:"کد OTP 6 رقمی"`
}

// RefreshTokenInput تازه‌سازی توکن را نشان می‌دهد
type RefreshTokenInput struct {
	RefreshToken string `json:"refresh_token" binding:"required" description:"توکن تازه‌سازی معتبر"`
}

// OTPResponse پاسخ درخواست OTP
type OTPResponse struct {
	Message string `json:"message" example:"otp sent" description:"پیام تأیید"`
	OTP     string `json:"otp,omitempty" example:"123456" description:"کد OTP (فقط در حالت تست)"`
}

// AuthResponse پاسخ احراز هویت
type AuthResponse struct {
	AccessToken  string      `json:"access_token" description:"توکن دسترسی JWT"`
	RefreshToken string      `json:"refresh_token" description:"توکن تازه‌سازی JWT"`
	User         domain.User `json:"user" description:"اطلاعات کاربر"`
	ExpiresIn    int64       `json:"expires_in" example:"3600" description:"مدت اعتبار توکن (ثانیه)"`
}

// TokenResponse پاسخ توکن جدید
type TokenResponse struct {
	AccessToken string `json:"access_token" description:"توکن دسترسی JWT جدید"`
	ExpiresIn   int64  `json:"expires_in" example:"3600" description:"مدت اعتبار توکن (ثانیه)"`
}

// Deprecated: استفاده از RequestOTPInput کنید
type requestOTPInput struct {
	Phone string `json:"phone" binding:"required"`
}

// Deprecated: استفاده از VerifyOTPInput کنید
type verifyOTPInput struct {
	Phone string `json:"phone" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

// Deprecated: استفاده از RefreshTokenInput کنید
type refreshInput struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func NewAuthHandler(db *gorm.DB, jwtService *auth.JWTService, otpStore *sms.OTPStore, otpProvider string, exposeMockOTP bool, adminPhones map[string]bool) *AuthHandler {
	return &AuthHandler{db: db, jwtService: jwtService, otpStore: otpStore, otpProvider: otpProvider, exposeMockOTP: exposeMockOTP, adminPhones: adminPhones}
}

// RequestOTP godoc
// @Summary درخواست کد OTP
// @Description کد OTP را برای شماره تلفن کاربر ارسال می‌کند
// @Tags احراز هویت
// @Accept json
// @Produce json
// @Param input body RequestOTPInput true "شماره تلفن"
// @Success 200 {object} OTPResponse "کد OTP با موفقیت ارسال شد"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /auth/request-otp [post]
func (h *AuthHandler) RequestOTP(c *gin.Context) {
	var input RequestOTPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}

	otp, err := h.otpStore.GenerateOTP(input.Phone)
	if err != nil {
		// Check if it's a rate limit error
		var rateLimitErr *sms.RateLimitError
		if errors.As(err, &rateLimitErr) {
			c.JSON(http.StatusTooManyRequests, ErrorResponse{
				Error: rateLimitErr.Message,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to generate otp"})
		return
	}

	resp := OTPResponse{Message: "otp sent"}
	if h.otpProvider == "mock" && h.exposeMockOTP {
		resp.OTP = otp
	}
	c.JSON(http.StatusOK, resp)
}

// VerifyOTP godoc
// @Summary تأیید کد OTP و ورود کاربر
// @Description کد OTP را تأیید می‌کند و توکن‌های JWT برای کاربر ایجاد می‌کند
// @Tags احراز هویت
// @Accept json
// @Produce json
// @Param input body VerifyOTPInput true "شماره تلفن و کد OTP"
// @Success 200 {object} AuthResponse "احراز هویت موفق"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 401 {object} ErrorResponse "کد OTP نامعتبر یا منقضی"
// @Failure 403 {object} ErrorResponse "کاربر غیرفعال است"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /auth/verify-otp [post]
func (h *AuthHandler) VerifyOTP(c *gin.Context) {
	var input VerifyOTPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}

	if !h.otpStore.VerifyOTP(input.Phone, input.Code) {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid or expired otp"})
		return
	}

	var user domain.User
	if err := h.db.Where("phone = ?", input.Phone).First(&user).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load user"})
			return
		}
		role := "student"
		if h.adminPhones[input.Phone] {
			role = "admin"
		}
		user = domain.User{Phone: input.Phone, Role: role, IsActive: true}
		if err := h.db.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create user"})
			return
		}
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "user is inactive"})
		return
	}

	accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to generate access token"})
		return
	}

	refreshToken, err := h.jwtService.GenerateRefreshToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to generate refresh token"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
		ExpiresIn:    h.jwtService.AccessTTLSeconds(),
	})
}

// RefreshToken godoc
// @Summary تازه‌سازی توکن دسترسی
// @Description با استفاده از توکن تازه‌سازی، توکن دسترسی جدید ایجاد می‌کند
// @Tags احراز هویت
// @Accept json
// @Produce json
// @Param input body RefreshTokenInput true "توکن تازه‌سازی"
// @Success 200 {object} TokenResponse "توکن جدید با موفقیت ایجاد شد"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 401 {object} ErrorResponse "توکن تازه‌سازی نامعتبر"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var input RefreshTokenInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}

	claims, err := h.jwtService.ValidateRefreshToken(input.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid refresh token"})
		return
	}

	userID, ok := claims["user_id"].(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid refresh token"})
		return
	}

	var user domain.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "user not found"})
		return
	}
	if !user.IsActive {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "user is inactive"})
		return
	}

	accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to generate access token"})
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		AccessToken: accessToken,
		ExpiresIn:   h.jwtService.AccessTTLSeconds(),
	})
}
