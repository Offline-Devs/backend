package handler

import (
    "log"
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
    adminPhones map[string]bool
}

type requestOTPInput struct {
    Phone string `json:"phone" binding:"required"`
}

type verifyOTPInput struct {
    Phone string `json:"phone" binding:"required"`
    Code  string `json:"code" binding:"required"`
}

type refreshInput struct {
    RefreshToken string `json:"refresh_token" binding:"required"`
}

func NewAuthHandler(db *gorm.DB, jwtService *auth.JWTService, otpStore *sms.OTPStore, otpProvider string, adminPhones map[string]bool) *AuthHandler {
    return &AuthHandler{db: db, jwtService: jwtService, otpStore: otpStore, otpProvider: otpProvider, adminPhones: adminPhones}
}

func (h *AuthHandler) RequestOTP(c *gin.Context) {
    var input requestOTPInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
        return
    }

    otp, err := h.otpStore.GenerateOTP(input.Phone)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate otp"})
        return
    }

    log.Printf("OTP requested for %s: %s", input.Phone, otp)

    resp := gin.H{"message": "otp sent"}
    if h.otpProvider == "mock" {
        resp["otp"] = otp
    }
    c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) VerifyOTP(c *gin.Context) {
    var input verifyOTPInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
        return
    }

    if !h.otpStore.VerifyOTP(input.Phone, input.Code) {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired otp"})
        return
    }

    var user domain.User
    if err := h.db.Where("phone = ?", input.Phone).First(&user).Error; err != nil {
        if err != gorm.ErrRecordNotFound {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user"})
            return
        }
        role := "student"
        if h.adminPhones[input.Phone] {
            role = "admin"
        }
        user = domain.User{Phone: input.Phone, Role: role, IsActive: true}
        if err := h.db.Create(&user).Error; err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
            return
        }
    }

    if !user.IsActive {
        c.JSON(http.StatusForbidden, gin.H{"error": "user is inactive"})
        return
    }

    accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Role)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
        return
    }

    refreshToken, err := h.jwtService.GenerateRefreshToken(user.ID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "access_token":  accessToken,
        "refresh_token": refreshToken,
        "user":          user,
    })
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
    var input refreshInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
        return
    }

    claims, err := h.jwtService.ValidateRefreshToken(input.RefreshToken)
    if err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
        return
    }

    userID, ok := claims["user_id"].(string)
    if !ok || userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid refresh token"})
        return
    }

    var user domain.User
    if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
        return
    }

    accessToken, err := h.jwtService.GenerateAccessToken(user.ID, user.Role)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate access token"})
        return
    }

    refreshToken, err := h.jwtService.GenerateRefreshToken(user.ID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate refresh token"})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "access_token":  accessToken,
        "refresh_token": refreshToken,
        "user":          user,
    })
}
