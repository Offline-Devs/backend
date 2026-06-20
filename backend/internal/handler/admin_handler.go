package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"gorm.io/gorm"
)

type AdminHandler struct {
	db *gorm.DB
}

func studentParam(c *gin.Context) string {
	if id := c.Param("id"); id != "" {
		return id
	}
	return c.Param("student_id")
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
		updates["first_name"] = strings.TrimSpace(*input.FirstName)
	}
	if input.LastName != nil {
		updates["last_name"] = strings.TrimSpace(*input.LastName)
	}
	if input.City != nil {
		updates["city"] = strings.TrimSpace(*input.City)
	}
	if input.School != nil {
		updates["school"] = strings.TrimSpace(*input.School)
	}
	if input.Major != nil {
		updates["major"] = strings.TrimSpace(*input.Major)
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

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no fields to update"})
		return
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
		EntityType: strings.TrimSpace(input.EntityType),
		Name:       strings.TrimSpace(input.Name),
		Label:      strings.TrimSpace(input.Label),
		FieldType:  strings.TrimSpace(input.FieldType),
		Options:    strings.TrimSpace(input.Options),
		IsRequired: input.IsRequired,
	}

	if err := h.db.Create(&field).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create field"})
		return
	}

	c.JSON(http.StatusCreated, field)
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
		"entity_type": strings.TrimSpace(input.EntityType),
		"name":        strings.TrimSpace(input.Name),
		"label":       strings.TrimSpace(input.Label),
		"field_type":  strings.TrimSpace(input.FieldType),
		"options":     strings.TrimSpace(input.Options),
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

// GetStudentExams godoc
// @Summary دریافت آزمون‌های یک دانشجو (مدیر)
// @Description مدیر می‌تواند تمام آزمون‌های یک دانشجو را مشاهده کند
// @Tags مدیریت
// @Security BearerAuth
// @Produce json
// @Param id path string true "شناسه دانشجو"
// @Success 200 {array} domain.Exam "لیست آزمون‌ها"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/students/{id}/exams [get]
func (h *AdminHandler) GetStudentExams(c *gin.Context) {
	studentID := studentParam(c)

	var exams []domain.Exam
	if err := h.db.Where("student_id = ?", studentID).
		Order("exam_date desc").
		Preload("Subjects").
		Find(&exams).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load exams"})
		return
	}

	c.JSON(http.StatusOK, exams)
}

// GetStudentMistakes godoc
// @Summary دریافت اشتباهات یک دانشجو (مدیر)
// @Description مدیر می‌تواند تمام اشتباهات ثبت‌شده یک دانشجو را مشاهده کند
// @Tags مدیریت
// @Security BearerAuth
// @Produce json
// @Param id path string true "شناسه دانشجو"
// @Success 200 {array} domain.Mistake "لیست اشتباهات"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/students/{id}/mistakes [get]
func (h *AdminHandler) GetStudentMistakes(c *gin.Context) {
	studentID := studentParam(c)

	var mistakes []domain.Mistake
	if err := h.db.Where("student_id = ?", studentID).
		Order("created_at desc").
		Find(&mistakes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load mistakes"})
		return
	}

	c.JSON(http.StatusOK, mistakes)
}

// GetAllStudentsWithStats godoc
// @Summary دریافت لیست دانشجویان با آمار (مدیر)
// @Description لیست تمام دانشجویان به همراه آمار خلاصه
// @Tags مدیریت
// @Security BearerAuth
// @Produce json
// @Param page query int false "شماره صفحه" default(1)
// @Param limit query int false "تعداد نتایج در هر صفحه" default(20)
// @Param approved query string false "فیلتر وضعیت تایید (true, false, all)" default(all)
// @Success 200 {object} ListResponse "لیست دانشجویان"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/students/with-stats [get]
func (h *AdminHandler) GetAllStudentsWithStats(c *gin.Context) {
	page := 1
	limit := 20
	if p, ok := c.GetQuery("page"); ok {
		if pInt, err := strconv.Atoi(p); err == nil && pInt > 0 {
			page = pInt
		}
	}
	if l, ok := c.GetQuery("limit"); ok {
		if lInt, err := strconv.Atoi(l); err == nil && lInt > 0 && lInt <= 100 {
			limit = lInt
		}
	}

	offset := (page - 1) * limit

	query := h.db.Model(&domain.Student{}).Preload("User")

	// Filter by approval status
	if approved := c.Query("approved"); approved != "" && approved != "all" {
		if approved == "true" {
			query = query.Where("is_approved = ?", true)
		} else if approved == "false" {
			query = query.Where("is_approved = ?", false)
		}
	}

	var total int64
	query.Count(&total)

	var students []domain.Student
	if err := query.Order("created_at desc").
		Offset(offset).
		Limit(limit).
		Find(&students).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load students"})
		return
	}

	// Add statistics for each student
	type StudentWithStats struct {
		domain.Student
		ExamCount    int64 `json:"exam_count"`
		MistakeCount int64 `json:"mistake_count"`
	}

	studentsWithStats := make([]StudentWithStats, 0, len(students))
	for _, student := range students {
		var examCount, mistakeCount int64
		h.db.Model(&domain.Exam{}).Where("student_id = ?", student.ID).Count(&examCount)
		h.db.Model(&domain.Mistake{}).Where("student_id = ?", student.ID).Count(&mistakeCount)

		studentsWithStats = append(studentsWithStats, StudentWithStats{
			Student:      student,
			ExamCount:    examCount,
			MistakeCount: mistakeCount,
		})
	}

	c.JSON(http.StatusOK, ListResponse{
		Data:  studentsWithStats,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}
