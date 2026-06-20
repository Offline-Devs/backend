package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db *gorm.DB
}

// CreateDynamicFieldInput داده‌های ورودی برای ایجاد فیلد سفارشی
type CreateDynamicFieldInput struct {
	EntityType string `json:"entity_type" binding:"required" description:"نوع موجودیت (student, exam, etc)"`
	Name       string `json:"name" binding:"required" description:"نام فیلد (نامای برنامه‌نویسی)"`
	Label      string `json:"label" description:"برچسب فیلد (برای نمایش)"`
	FieldType  string `json:"field_type" binding:"required" description:"نوع فیلد (text, number, select, etc)"`
	Options    string `json:"options" description:"گزینه‌های فیلد (JSON format)"`
	IsRequired bool   `json:"is_required" description:"آیا فیلد اجباری است"`
}

// UpdateStudentInput داده‌های ورودی برای بروزرسانی دانشجو
type UpdateStudentInput struct {
	FirstName  *string `json:"first_name" description:"نام کاربر"`
	LastName   *string `json:"last_name" description:"نام خانوادگی کاربر"`
	City       *string `json:"city" description:"شهر"`
	School     *string `json:"school" description:"نام مدرسه"`
	Major      *string `json:"major" description:"رشته تحصیلی"`
	IsApproved *bool   `json:"is_approved" description:"وضعیت تایید دانشجو"`
}

// Deprecated: استفاده از CreateDynamicFieldInput کنید
type createDynamicFieldInput struct {
	EntityType string `json:"entity_type" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Label      string `json:"label"`
	FieldType  string `json:"field_type" binding:"required"`
	Options    string `json:"options"`
	IsRequired bool   `json:"is_required"`
}

// Deprecated: استفاده از UpdateStudentInput کنید
type updateStudentInput struct {
	FirstName  *string `json:"first_name"`
	LastName   *string `json:"last_name"`
	City       *string `json:"city"`
	School     *string `json:"school"`
	Major      *string `json:"major"`
	IsApproved *bool   `json:"is_approved"`
}

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

// ListStudents godoc
// @Summary دریافت لیست دانشجویان
// @Description لیست تمام دانشجویان را دریافت می‌کند (فقط برای مدیران)
// @Tags مدیریت
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.Student "لیست دانشجویان"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/students [get]
func (h *AdminHandler) ListStudents(c *gin.Context) {
	var students []domain.Student
	if err := h.db.Preload("User").Limit(100).Order("created_at desc").Find(&students).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load students"})
		return
	}
	c.JSON(http.StatusOK, students)
}

// GetStudent godoc
// @Summary دریافت جزئیات دانشجو
// @Description جزئیات یک دانشجو را دریافت می‌کند (فقط برای مدیران)
// @Tags مدیریت
// @Security BearerAuth
// @Produce json
// @Param id path string true "شناسه دانشجو"
// @Success 200 {object} domain.Student "جزئیات دانشجو"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "دانشجو یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/students/{id} [get]
func (h *AdminHandler) GetStudent(c *gin.Context) {
	id := c.Param("id")
	var student domain.Student
	if err := h.db.Preload("User").First(&student, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student not found"})
		return
	}
	c.JSON(http.StatusOK, student)
}

// UpdateStudent godoc
// @Summary بروزرسانی اطلاعات دانشجو
// @Description اطلاعات یک دانشجو را بروزرسانی می‌کند (فقط برای مدیران)
// @Tags مدیریت
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "شناسه دانشجو"
// @Param input body UpdateStudentInput true "اطلاعات جدید دانشجو"
// @Success 200 {object} map[string]string "دانشجو با موفقیت بروزرسانی شد"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/students/{id} [put]
func (h *AdminHandler) UpdateStudent(c *gin.Context) {
	id := c.Param("id")
	var input UpdateStudentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}

	updates := map[string]interface{}{}
	if input.FirstName != nil {
		updates["first_name"] = *input.FirstName
	}
	if input.LastName != nil {
		updates["last_name"] = *input.LastName
	}
	if input.City != nil {
		updates["city"] = *input.City
	}
	if input.School != nil {
		updates["school"] = *input.School
	}
	if input.Major != nil {
		updates["major"] = *input.Major
	}
	if input.IsApproved != nil {
		updates["is_approved"] = *input.IsApproved
		if *input.IsApproved {
			now := time.Now()
			updates["approval_date"] = &now
		} else {
			updates["approval_date"] = nil
		}
	}

	if err := h.db.Model(&domain.Student{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update student"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

// DeleteStudent godoc
// @Summary حذف دانشجو
// @Description یک دانشجو و تمام اطلاعات مرتبط آن را حذف می‌کند (فقط برای مدیران)
// @Tags مدیریت
// @Security BearerAuth
// @Param id path string true "شناسه دانشجو"
// @Success 200 {object} map[string]string "دانشجو با موفقیت حذف شد"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/students/{id} [delete]
func (h *AdminHandler) DeleteStudent(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&domain.Student{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to delete student"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// ApproveStudent godoc
// @Summary تایید دانشجو
// @Description یک دانشجو را تایید می‌کند (فقط برای مدیران)
// @Tags مدیریت
// @Security BearerAuth
// @Param id path string true "شناسه دانشجو"
// @Success 200 {object} map[string]string "دانشجو با موفقیت تایید شد"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/students/{id}/approve [put]
func (h *AdminHandler) ApproveStudent(c *gin.Context) {
	id := c.Param("id")
	now := time.Now()
	if err := h.db.Model(&domain.Student{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_approved":   true,
		"approval_date": &now,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to approve student"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "approved"})
}

// GetDynamicFields godoc
// @Summary دریافت فیلدهای سفارشی
// @Description فیلدهای سفارشی را برای یک موجودیت دریافت می‌کند (فقط برای مدیران)
// @Tags مدیریت
// @Security BearerAuth
// @Produce json
// @Param entity_type query string false "نوع موجودیت (اختیاری)"
// @Success 200 {array} domain.DynamicFieldDefinition "لیست فیلدهای سفارشی"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/dynamic-fields [get]
func (h *AdminHandler) GetDynamicFields(c *gin.Context) {
	entityType := c.Query("entity_type")
	q := h.db.Model(&domain.DynamicFieldDefinition{})
	if entityType != "" {
		q = q.Where("entity_type = ?", entityType)
	}

	var fields []domain.DynamicFieldDefinition
	if err := q.Order("created_at desc").Find(&fields).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load fields"})
		return
	}
	c.JSON(http.StatusOK, fields)
}

// CreateDynamicField godoc
// @Summary ایجاد فیلد سفارشی
// @Description یک فیلد سفارشی جدید ایجاد می‌کند (فقط برای مدیران)
// @Tags مدیریت
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body CreateDynamicFieldInput true "اطلاعات فیلد سفارشی"
// @Success 201 {object} domain.DynamicFieldDefinition "فیلد با موفقیت ایجاد شد"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/dynamic-fields [post]
func (h *AdminHandler) CreateDynamicField(c *gin.Context) {
	var input CreateDynamicFieldInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}

	field := domain.DynamicFieldDefinition{
		EntityType: input.EntityType,
		Name:       input.Name,
		Label:      input.Label,
		FieldType:  input.FieldType,
		Options:    input.Options,
		IsRequired: input.IsRequired,
	}

	if err := h.db.Create(&field).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create field"})
		return
	}

	c.JSON(http.StatusOK, field)
}

// UpdateDynamicField godoc
// @Summary بروزرسانی فیلد سفارشی
// @Description یک فیلد سفارشی را بروزرسانی می‌کند (فقط برای مدیران)
// @Tags مدیریت
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "شناسه فیلد"
// @Param input body CreateDynamicFieldInput true "اطلاعات جدید فیلد"
// @Success 200 {object} map[string]string "فیلد با موفقیت بروزرسانی شد"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/dynamic-fields/{id} [put]
func (h *AdminHandler) UpdateDynamicField(c *gin.Context) {
	id := c.Param("id")
	var input CreateDynamicFieldInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}

	updates := map[string]interface{}{
		"entity_type": input.EntityType,
		"name":        input.Name,
		"label":       input.Label,
		"field_type":  input.FieldType,
		"options":     input.Options,
		"is_required": input.IsRequired,
	}

	if err := h.db.Model(&domain.DynamicFieldDefinition{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update field"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

// DeleteDynamicField godoc
// @Summary حذف فیلد سفارشی
// @Description یک فیلد سفارشی را حذف می‌کند (فقط برای مدیران)
// @Tags مدیریت
// @Security BearerAuth
// @Param id path string true "شناسه فیلد"
// @Success 200 {object} map[string]string "فیلد با موفقیت حذف شد"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/dynamic-fields/{id} [delete]
func (h *AdminHandler) DeleteDynamicField(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&domain.DynamicFieldDefinition{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to delete field"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// ListStudentExams godoc
// @Summary دریافت تمام آزمون‌های یک دانشجو
// @Description مدیر می‌تواند تمام آزمون‌های یک دانشجو را مشاهده کند
// @Tags مدیریت
// @Security BearerAuth
// @Produce json
// @Param student_id path string true "شناسه دانشجو"
// @Success 200 {array} domain.Exam "لیست آزمون‌های دانشجو"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "دانشجو یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/students/{student_id}/exams [get]
func (h *AdminHandler) ListStudentExams(c *gin.Context) {
	studentID := c.Param("student_id")

	// Verify student exists
	var student domain.Student
	if err := h.db.First(&student, "id = ?", studentID).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student not found"})
		return
	}

	var exams []domain.Exam
	if err := h.db.Where("student_id = ?", studentID).
		Preload("Subjects").
		Order("exam_date desc").
		Find(&exams).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load exams"})
		return
	}

	c.JSON(http.StatusOK, exams)
}

// ListStudentMistakes godoc
// @Summary دریافت تمام اشتباهات یک دانشجو
// @Description مدیر می‌تواند تمام اشتباهات یک دانشجو را مشاهده کند
// @Tags مدیریت
// @Security BearerAuth
// @Produce json
// @Param student_id path string true "شناسه دانشجو"
// @Success 200 {array} domain.Mistake "لیست اشتباهات دانشجو"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "دانشجو یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/students/{student_id}/mistakes [get]
func (h *AdminHandler) ListStudentMistakes(c *gin.Context) {
	studentID := c.Param("student_id")

	// Verify student exists
	var student domain.Student
	if err := h.db.First(&student, "id = ?", studentID).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student not found"})
		return
	}

	var mistakes []domain.Mistake
	if err := h.db.Where("student_id = ?", studentID).
		Order("created_at desc").
		Find(&mistakes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load mistakes"})
		return
	}

	c.JSON(http.StatusOK, mistakes)
}
