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

// CompleteProfileInput اطلاعات پروفایل دانشجو
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

// Deprecated: استفاده از CompleteProfileInput کنید
type studentProfileInput struct {
	FirstName       string                 `json:"first_name"`
	LastName        string                 `json:"last_name"`
	City            string                 `json:"city"`
	BirthDate       *time.Time             `json:"birth_date"`
	JalaliBirthDate string                 `json:"jalali_birth_date"`
	School          string                 `json:"school"`
	Major           string                 `json:"major"`
	ProfilePhoto    string                 `json:"profile_photo"`
	DynamicFields   map[string]interface{} `json:"dynamic_fields"`
}

func NewStudentHandler(db *gorm.DB) *StudentHandler {
	return &StudentHandler{db: db}
}

// CompleteProfile godoc
// @Summary تکمیل یا بروزرسانی پروفایل دانشجو
// @Description اطلاعات پروفایل دانشجویی را تکمیل یا به‌روز می‌کند
// @Tags دانشجویان
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body CompleteProfileInput true "اطلاعات پروفایل"
// @Success 200 {object} domain.Student "پروفایل با موفقیت ایجاد/بروزرسانی شد"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /students/profile [post]
func (h *StudentHandler) CompleteProfile(c *gin.Context) {
	var input CompleteProfileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}
	if strings.TrimSpace(input.FirstName) == "" || strings.TrimSpace(input.LastName) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "first_name and last_name are required"})
		return
	}
	if input.JalaliBirthDate != "" && input.BirthDate != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "birth_date and jalali_birth_date are mutually exclusive"})
		return
	}

	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}
	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid user id"})
		return
	}

	var student domain.Student
	err := h.db.Where("user_id = ?", userIDStr).First(&student).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load profile"})
		return
	}
	notFound := err == gorm.ErrRecordNotFound

	if input.JalaliBirthDate != "" {
		canonicalDate, err := pkg.CanonicalJalaliDate(input.JalaliBirthDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid jalali_birth_date format"})
			return
		}
		t, _ := pkg.JalaliToGregorian(canonicalDate)
		input.BirthDate = &t
		input.JalaliBirthDate = canonicalDate
	} else if input.BirthDate != nil {
		input.JalaliBirthDate = pkg.GregorianToJalaliString(*input.BirthDate)
	}

	dynamicFields, err := validateAndCleanDynamicValues(h.db, "student", input.DynamicFields)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	if notFound {
		student = domain.Student{
			UserID:        userIDStr,
			FirstName:     strings.TrimSpace(input.FirstName),
			LastName:      strings.TrimSpace(input.LastName),
			City:          strings.TrimSpace(input.City),
			School:        strings.TrimSpace(input.School),
			Major:         strings.TrimSpace(input.Major),
			ProfilePhoto:  strings.TrimSpace(input.ProfilePhoto),
			DynamicFields: dynamicFields,
		}
		if input.BirthDate != nil {
			student.BirthDate = *input.BirthDate
		}
		if input.JalaliBirthDate != "" {
			student.JalaliBirthDate = input.JalaliBirthDate
		}

		if err := h.db.Create(&student).Error; err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create profile"})
			return
		}
		c.JSON(http.StatusOK, student)
		return
	}

	updates := map[string]interface{}{
		"first_name":     strings.TrimSpace(input.FirstName),
		"last_name":      strings.TrimSpace(input.LastName),
		"city":           strings.TrimSpace(input.City),
		"school":         strings.TrimSpace(input.School),
		"major":          strings.TrimSpace(input.Major),
		"profile_photo":  strings.TrimSpace(input.ProfilePhoto),
		"dynamic_fields": dynamicFields,
	}
	if input.JalaliBirthDate != "" {
		updates["jalali_birth_date"] = input.JalaliBirthDate
	}
	if input.BirthDate != nil {
		updates["birth_date"] = *input.BirthDate
	}

	if student.IsApproved {
		updates["is_approved"] = false
		updates["approval_date"] = nil
	}

	if err := h.db.Model(&student).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update profile"})
		return
	}
	if err := h.db.Preload("User").First(&student, "id = ?", student.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to reload profile"})
		return
	}

	c.JSON(http.StatusOK, student)
}

// GetProfile godoc
// @Summary دریافت پروفایل دانشجو
// @Description اطلاعات پروفایل دانشجویی را دریافت می‌کند
// @Tags دانشجویان
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.Student "پروفایل دانشجو"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "پروفایل یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /students/profile [get]
func (h *StudentHandler) GetProfile(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}

	var student domain.Student
	if err := h.db.Preload("User").Where("user_id = ?", userID).First(&student).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "profile not found"})
		return
	}

	c.JSON(http.StatusOK, student)
}
