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

type CreateDynamicFieldInput struct {
	EntityType string `json:"entity_type" binding:"required" description:"نوع موجودیت (student, exam, etc)"`
	Name       string `json:"name" binding:"required" description:"نام فیلد (نامای برنامه‌نویسی)"`
	Label      string `json:"label" description:"برچسب فیلد (برای نمایش)"`
	FieldType  string `json:"field_type" binding:"required" description:"نوع فیلد (text, number, select, etc)"`
	Options    string `json:"options" description:"گزینه‌های فیلد (JSON format)"`
	IsRequired bool   `json:"is_required" description:"آیا فیلد اجباری است"`
}

type UpdateStudentInput struct {
	FirstName       *string `json:"first_name" description:"نام کاربر"`
	LastName        *string `json:"last_name" description:"نام خانوادگی کاربر"`
	City            *string `json:"city" description:"شهر"`
	School          *string `json:"school" description:"نام مدرسه"`
	Major           *string `json:"major" description:"رشته تحصیلی"`
	Status          *string `json:"status" description:"وضعیت تایید (pending, approved, rejected)"`
	RejectionReason *string `json:"rejection_reason" description:"دلیل رد"`
}

type createDynamicFieldInput = CreateDynamicFieldInput

type updateStudentInput = UpdateStudentInput

func NewAdminHandler(db *gorm.DB) *AdminHandler {
	return &AdminHandler{db: db}
}

func (h *AdminHandler) ListStudents(c *gin.Context) {
	var students []domain.StudentProfile
	if err := h.db.Preload("User").Limit(100).Order("created_at desc").Find(&students).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load students"})
		return
	}
	c.JSON(http.StatusOK, students)
}

func (h *AdminHandler) GetStudent(c *gin.Context) {
	id := c.Param("id")
	var student domain.StudentProfile
	if err := h.db.Preload("User").First(&student, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student not found"})
		return
	}
	c.JSON(http.StatusOK, student)
}

func (h *AdminHandler) UpdateStudent(c *gin.Context) {
	id := c.Param("id")
	adminID, ok := getAuthenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}

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
	if input.Status != nil {
		status := strings.TrimSpace(*input.Status)
		switch status {
		case domain.StudentProfileStatusPending:
			updates["status"] = status
			updates["is_approved"] = false
			updates["approval_date"] = nil
			updates["reviewed_at"] = nil
			updates["reviewed_by"] = nil
			updates["rejection_reason"] = ""
		case domain.StudentProfileStatusApproved:
			now := time.Now().UTC()
			updates["status"] = status
			updates["is_approved"] = true
			updates["approval_date"] = &now
			updates["reviewed_at"] = &now
			updates["reviewed_by"] = &adminID
			updates["rejection_reason"] = ""
		case domain.StudentProfileStatusRejected:
			now := time.Now().UTC()
			updates["status"] = status
			updates["is_approved"] = false
			updates["approval_date"] = nil
			updates["reviewed_at"] = &now
			updates["reviewed_by"] = &adminID
			if input.RejectionReason == nil || strings.TrimSpace(*input.RejectionReason) == "" {
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "rejection_reason is required when rejecting a profile"})
				return
			}
			updates["rejection_reason"] = strings.TrimSpace(*input.RejectionReason)
		default:
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid status"})
			return
		}
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no fields to update"})
		return
	}

	if err := h.db.Model(&domain.StudentProfile{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update student"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (h *AdminHandler) DeleteStudent(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&domain.StudentProfile{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to delete student"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AdminHandler) ApproveStudent(c *gin.Context) {
	id := c.Param("id")
	adminID, ok := getAuthenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}
	adminIDCopy := adminID
	now := time.Now().UTC()
	if err := h.db.Model(&domain.StudentProfile{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":           domain.StudentProfileStatusApproved,
		"is_approved":      true,
		"approval_date":    &now,
		"reviewed_at":      &now,
		"reviewed_by":      &adminIDCopy,
		"rejection_reason": "",
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to approve student"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "approved"})
}

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

func (h *AdminHandler) DeleteDynamicField(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&domain.DynamicFieldDefinition{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to delete field"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *AdminHandler) GetStudentExams(c *gin.Context) {
	studentID := studentParam(c)

	var exams []domain.Exam
	if err := h.db.Where("student_id = ?", studentID).Order("exam_date desc").Preload("Subjects").Find(&exams).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load exams"})
		return
	}

	c.JSON(http.StatusOK, exams)
}

func (h *AdminHandler) GetStudentMistakes(c *gin.Context) {
	studentID := studentParam(c)

	var mistakes []domain.MistakeAnalysis
	if err := h.db.Where("student_id = ?", studentID).Order("created_at desc").Find(&mistakes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load mistakes"})
		return
	}

	c.JSON(http.StatusOK, mistakes)
}

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
	query := h.db.Model(&domain.StudentProfile{}).Preload("User")

	if approved := c.Query("approved"); approved != "" && approved != "all" {
		if approved == "true" {
			query = query.Where("is_approved = ?", true)
		} else if approved == "false" {
			query = query.Where("is_approved = ?", false)
		}
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var total int64
	query.Count(&total)

	var students []domain.StudentProfile
	if err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&students).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load students"})
		return
	}

	type StudentWithStats struct {
		domain.StudentProfile
		ExamCount      int64 `json:"exam_count"`
		MistakeCount   int64 `json:"mistake_count"`
		StudyPlanCount int64 `json:"study_plan_count"`
		AdminNoteCount int64 `json:"admin_note_count"`
	}

	studentsWithStats := make([]StudentWithStats, 0, len(students))
	for _, student := range students {
		var examCount, mistakeCount, studyPlanCount, adminNoteCount int64
		h.db.Model(&domain.Exam{}).Where("student_id = ?", student.ID).Count(&examCount)
		h.db.Model(&domain.MistakeAnalysis{}).Where("student_id = ?", student.ID).Count(&mistakeCount)
		h.db.Model(&domain.StudyPlan{}).Where("student_id = ?", student.ID).Count(&studyPlanCount)
		h.db.Model(&domain.AdminNote{}).Where("student_id = ?", student.ID).Count(&adminNoteCount)

		studentsWithStats = append(studentsWithStats, StudentWithStats{
			StudentProfile: student,
			ExamCount:      examCount,
			MistakeCount:   mistakeCount,
			StudyPlanCount: studyPlanCount,
			AdminNoteCount: adminNoteCount,
		})
	}

	c.JSON(http.StatusOK, ListResponse{
		Data:  studentsWithStats,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}
