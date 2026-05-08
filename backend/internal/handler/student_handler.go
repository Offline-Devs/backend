package handler

import (
	"net/http"
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

	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}

	var student domain.Student
	err := h.db.Where("user_id = ?", userID).First(&student).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load profile"})
		return
	}

	if input.JalaliBirthDate != "" {
		if t, err := pkg.JalaliToGregorian(input.JalaliBirthDate); err == nil {
			input.BirthDate = &t
		}
	}

	if err == gorm.ErrRecordNotFound {
		student = domain.Student{
			UserID:        userID.(string),
			FirstName:     input.FirstName,
			LastName:      input.LastName,
			City:          input.City,
			School:        input.School,
			Major:         input.Major,
			ProfilePhoto:  input.ProfilePhoto,
			DynamicFields: input.DynamicFields,
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
		"first_name":        input.FirstName,
		"last_name":         input.LastName,
		"city":              input.City,
		"school":            input.School,
		"major":             input.Major,
		"profile_photo":     input.ProfilePhoto,
		"dynamic_fields":    input.DynamicFields,
		"jalali_birth_date": input.JalaliBirthDate,
	}
	if input.BirthDate != nil {
		updates["birth_date"] = *input.BirthDate
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
