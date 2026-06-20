package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"github.com/yourusername/noshirvani-academy/backend/pkg"
	"gorm.io/gorm"
)

type PerformanceHandler struct {
	db *gorm.DB
}

// CreatePerformanceInput داده‌های ورودی برای ایجاد رکورد عملکرد
type CreatePerformanceInput struct {
	Date       *time.Time `json:"date" description:"تاریخ میلادی"`
	JalaliDate string     `json:"jalali_date" example:"1400/01/01" description:"تاریخ جلالی"`
	Notes      string     `json:"notes" description:"یادداشت‌ها و توضیحات"`
	StudyPlan  string     `json:"study_plan" description:"برنامه مطالعاتی"`
	Files      string     `json:"files" description:"فایل‌های پیوست (JSON array of URLs)"`
}

// UpdatePerformanceInput داده‌های ورودی برای بروزرسانی رکورد عملکرد
type UpdatePerformanceInput struct {
	Notes     *string `json:"notes" description:"یادداشت‌ها و توضیحات"`
	StudyPlan *string `json:"study_plan" description:"برنامه مطالعاتی"`
	Files     *string `json:"files" description:"فایل‌های پیوست (JSON array of URLs)"`
}

func NewPerformanceHandler(db *gorm.DB) *PerformanceHandler {
	return &PerformanceHandler{db: db}
}

// GetStudentPerformance godoc
// @Summary دریافت تاریخچه عملکرد دانشجو
// @Description تاریخچه عملکرد و برنامه‌های مطالعاتی دانشجو را دریافت می‌کند (فقط خواندنی برای دانشجو)
// @Tags عملکرد
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.PerformanceHistory "لیست رکوردهای عملکرد"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "پروفایل دانشجو یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /students/performance [get]
func (h *PerformanceHandler) GetStudentPerformance(c *gin.Context) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}

	var student domain.Student
	if err := h.db.Where("user_id = ?", userID).First(&student).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student profile not found"})
		return
	}

	var performances []domain.PerformanceHistory
	if err := h.db.Where("student_id = ?", student.ID).Order("date desc").Find(&performances).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load performance history"})
		return
	}

	c.JSON(http.StatusOK, performances)
}

// AdminCreatePerformance godoc
// @Summary ایجاد رکورد عملکرد برای دانشجو (مدیر)
// @Description مدیر می‌تواند برنامه مطالعاتی، یادداشت و فایل برای دانشجو اضافه کند
// @Tags مدیریت
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param student_id path string true "شناسه دانشجو"
// @Param input body CreatePerformanceInput true "اطلاعات رکورد عملکرد"
// @Success 201 {object} domain.PerformanceHistory "رکورد با موفقیت ایجاد شد"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "دانشجو یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/students/{student_id}/performance [post]
func (h *PerformanceHandler) AdminCreatePerformance(c *gin.Context) {
	studentID := c.Param("student_id")
	var input CreatePerformanceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}

	// Verify student exists
	var student domain.Student
	if err := h.db.First(&student, "id = ?", studentID).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student not found"})
		return
	}

	recordDate := time.Now()
	jalaliDate := pkg.GregorianToJalaliString(recordDate)

	if input.JalaliDate != "" {
		if t, err := pkg.JalaliToGregorian(input.JalaliDate); err == nil {
			recordDate = t
			jalaliDate = input.JalaliDate
		}
	} else if input.Date != nil {
		recordDate = *input.Date
		jalaliDate = pkg.GregorianToJalaliString(recordDate)
	}

	performance := domain.PerformanceHistory{
		StudentID:  studentID,
		Date:       recordDate,
		JalaliDate: jalaliDate,
		Notes:      input.Notes,
		StudyPlan:  input.StudyPlan,
		Files:      input.Files,
	}

	if err := h.db.Create(&performance).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create performance record"})
		return
	}

	c.JSON(http.StatusCreated, performance)
}

// AdminUpdatePerformance godoc
// @Summary بروزرسانی رکورد عملکرد (مدیر)
// @Description مدیر می‌تواند رکورد عملکرد دانشجو را بروزرسانی کند
// @Tags مدیریت
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "شناسه رکورد عملکرد"
// @Param input body UpdatePerformanceInput true "اطلاعات جدید"
// @Success 200 {object} map[string]string "رکورد با موفقیت بروزرسانی شد"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/performance/{id} [put]
func (h *PerformanceHandler) AdminUpdatePerformance(c *gin.Context) {
	id := c.Param("id")
	var input UpdatePerformanceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}

	updates := map[string]interface{}{}
	if input.Notes != nil {
		updates["notes"] = *input.Notes
	}
	if input.StudyPlan != nil {
		updates["study_plan"] = *input.StudyPlan
	}
	if input.Files != nil {
		updates["files"] = *input.Files
	}

	if err := h.db.Model(&domain.PerformanceHistory{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update performance record"})
		return
	}

	c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

// AdminDeletePerformance godoc
// @Summary حذف رکورد عملکرد (مدیر)
// @Description مدیر می‌تواند رکورد عملکرد را حذف کند
// @Tags مدیریت
// @Security BearerAuth
// @Param id path string true "شناسه رکورد عملکرد"
// @Success 200 {object} map[string]string "رکورد با موفقیت حذف شد"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/performance/{id} [delete]
func (h *PerformanceHandler) AdminDeletePerformance(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&domain.PerformanceHistory{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to delete performance record"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// AdminListStudentPerformance godoc
// @Summary دریافت تاریخچه عملکرد یک دانشجو (مدیر)
// @Description مدیر می‌تواند تمام رکوردهای عملکرد یک دانشجو را مشاهده کند
// @Tags مدیریت
// @Security BearerAuth
// @Produce json
// @Param student_id path string true "شناسه دانشجو"
// @Success 200 {array} domain.PerformanceHistory "لیست رکوردهای عملکرد"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/students/{student_id}/performance [get]
func (h *PerformanceHandler) AdminListStudentPerformance(c *gin.Context) {
	studentID := c.Param("student_id")

	var performances []domain.PerformanceHistory
	if err := h.db.Where("student_id = ?", studentID).Order("date desc").Find(&performances).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load performance history"})
		return
	}

	c.JSON(http.StatusOK, performances)
}
