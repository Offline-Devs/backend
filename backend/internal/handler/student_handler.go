package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"github.com/yourusername/noshirvani-academy/backend/pkg"
	"gorm.io/gorm"
)

type StudentHandler struct {
	db *gorm.DB
}

type CompleteProfileInput struct {
	FirstName       string                 `json:"first_name" description:"نام کاربر"`
	LastName        string                 `json:"last_name" description:"نام خانوادگی کاربر"`
	City            string                 `json:"city" description:"شهر"`
	BirthDate       *time.Time             `json:"birth_date" description:"تاریخ تولد میلادی"`
	JalaliBirthDate string                 `json:"jalali_birth_date" example:"1400/01/01" description:"تاریخ تولد جلالی (YYYY/MM/DD)"`
	School          string                 `json:"school" description:"نام مدرسه"`
	Major           string                 `json:"major" description:"رشته تحصیلی"`
	ProfilePhoto    string                 `json:"profile_photo" description:"URL عکس پروفایل"`
	DynamicFields   map[string]interface{} `json:"dynamic_fields" description:"فیلدهای سفارشی"`
}

type studentProfileInput = CompleteProfileInput

func NewStudentHandler(db *gorm.DB) *StudentHandler {
	return &StudentHandler{db: db}
}

func (h *StudentHandler) CompleteProfile(c *gin.Context) {
	var input CompleteProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}

	input.FirstName = strings.TrimSpace(input.FirstName)
	input.LastName = strings.TrimSpace(input.LastName)
	input.City = strings.TrimSpace(input.City)
	input.School = strings.TrimSpace(input.School)
	input.Major = strings.TrimSpace(input.Major)
	input.ProfilePhoto = strings.TrimSpace(input.ProfilePhoto)

	if input.FirstName == "" || input.LastName == "" || input.City == "" || input.School == "" || input.Major == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "first_name, last_name, city, school, and major are required"})
		return
	}

	userIDValue, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}
	userID, ok := userIDValue.(string)
	if !ok || userID == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid user id"})
		return
	}

	birthDate, jalaliBirthDate, err := normalizeJalaliDateInput(input.BirthDate, input.JalaliBirthDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	now := time.Now().UTC()
	var profile domain.StudentProfile
	err = h.db.Where("user_id = ?", userID).First(&profile).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load profile"})
		return
	}

	if err == gorm.ErrRecordNotFound {
		profile = domain.StudentProfile{
			UserID:          userID,
			FirstName:       input.FirstName,
			LastName:        input.LastName,
			City:            input.City,
			School:          input.School,
			Major:           input.Major,
			BirthDate:       birthDate,
			JalaliBirthDate: jalaliBirthDate,
			ProfilePhoto:    input.ProfilePhoto,
			Status:          domain.StudentProfileStatusPending,
			IsApproved:      false,
			LastSubmittedAt: &now,
			DynamicFields:   input.DynamicFields,
		}
		if err := h.db.Create(&profile).Error; err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create profile"})
			return
		}
		c.JSON(http.StatusOK, profile)
		return
	}

	if profile.Status == domain.StudentProfileStatusApproved {
		c.JSON(http.StatusConflict, ErrorResponse{Error: "approved profiles cannot be edited by students"})
		return
	}

	updates := map[string]interface{}{
		"first_name":        input.FirstName,
		"last_name":         input.LastName,
		"city":              input.City,
		"school":            input.School,
		"major":             input.Major,
		"birth_date":        birthDate,
		"jalali_birth_date": jalaliBirthDate,
		"profile_photo":     input.ProfilePhoto,
		"dynamic_fields":    input.DynamicFields,
		"status":            domain.StudentProfileStatusPending,
		"is_approved":       false,
		"approval_date":     nil,
		"reviewed_at":       nil,
		"reviewed_by":       nil,
		"rejection_reason":  "",
		"last_submitted_at": &now,
	}

	if err := h.db.Model(&profile).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update profile"})
		return
	}
	if err := h.db.Preload("User").First(&profile, "id = ?", profile.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to reload profile"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (h *StudentHandler) GetProfile(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}

	var profile domain.StudentProfile
	if err := h.db.Preload("User").Where("user_id = ?", userID).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "profile not found"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func normalizeJalaliDateInput(gregorian *time.Time, jalali string) (time.Time, string, error) {
	if strings.TrimSpace(jalali) != "" {
		t, err := pkg.JalaliToGregorian(jalali)
		if err != nil {
			return time.Time{}, "", err
		}
		return t.UTC(), strings.TrimSpace(jalali), nil
	}
	if gregorian != nil && !gregorian.IsZero() {
		t := gregorian.UTC()
		return t, pkg.GregorianToJalaliString(t), nil
	}
	return time.Time{}, "", ErrInvalidDateInput
}
