package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"gorm.io/gorm"
)

type MistakeHandler struct {
	db *gorm.DB
}

type CreateMistakeInput struct {
	ExamID         *string                `json:"exam_id" description:"شناسه آزمون (اختیاری)"`
	SubjectExamID  *string                `json:"subject_exam_id" description:"شناسه درس آزمون (اختیاری)"`
	QuestionNumber int                    `json:"question_number" description:"شماره سؤال"`
	Category       string                 `json:"category" description:"دسته‌بندی اشتباه"`
	Notes          string                 `json:"notes" description:"یادداشت‌های توضیحی"`
	DynamicFields  map[string]interface{} `json:"dynamic_fields" description:"فیلدهای سفارشی"`
}

type UpdateMistakeInput struct {
	ExamID         *string                `json:"exam_id" description:"شناسه آزمون (اختیاری)"`
	SubjectExamID  *string                `json:"subject_exam_id" description:"شناسه درس آزمون (اختیاری)"`
	QuestionNumber *int                   `json:"question_number" description:"شماره سؤال"`
	Category       *string                `json:"category" description:"دسته‌بندی اشتباه"`
	Notes          *string                `json:"notes" description:"یادداشت‌های توضیحی"`
	DynamicFields  map[string]interface{} `json:"dynamic_fields" description:"فیلدهای سفارشی"`
}

type createMistakeInput = CreateMistakeInput

func NewMistakeHandler(db *gorm.DB) *MistakeHandler {
	return &MistakeHandler{db: db}
}

func (h *MistakeHandler) Create(c *gin.Context) {
	var input CreateMistakeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}
	if input.QuestionNumber <= 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "question_number must be greater than zero"})
		return
	}
	if strings.TrimSpace(input.Category) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "category is required"})
		return
	}

	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}

	profile, err := loadStudentProfileByUserID(h.db, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student profile not found"})
		return
	}
	if err := validateMistakeRelationships(h.db, profile.ID, input.ExamID, input.SubjectExamID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	mistake := domain.MistakeAnalysis{
		StudentProfileID: profile.ID,
		ExamID:           input.ExamID,
		ExamSubjectID:    input.SubjectExamID,
		QuestionNumber:   input.QuestionNumber,
		Category:         strings.TrimSpace(input.Category),
		Notes:            strings.TrimSpace(input.Notes),
		DynamicFields:    input.DynamicFields,
	}

	if err := h.db.Create(&mistake).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create mistake"})
		return
	}

	c.JSON(http.StatusCreated, mistake)
}

func (h *MistakeHandler) List(c *gin.Context) {
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}

	profile, err := loadStudentProfileByUserID(h.db, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student profile not found"})
		return
	}

	var mistakes []domain.MistakeAnalysis
	if err := h.db.Where("student_id = ?", profile.ID).Order("created_at desc").Find(&mistakes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load mistakes"})
		return
	}
	c.JSON(http.StatusOK, mistakes)
}

func (h *MistakeHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}

	profile, err := loadStudentProfileByUserID(h.db, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student profile not found"})
		return
	}

	if err := h.db.Where("id = ? AND student_id = ?", id, profile.ID).Delete(&domain.MistakeAnalysis{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to delete mistake"})
		return
	}

	c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *MistakeHandler) Update(c *gin.Context) {
	id := c.Param("id")
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}

	var input UpdateMistakeInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}
	if input.QuestionNumber != nil && *input.QuestionNumber <= 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "question_number must be greater than zero"})
		return
	}

	profile, err := loadStudentProfileByUserID(h.db, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student profile not found"})
		return
	}

	var mistake domain.MistakeAnalysis
	if err := h.db.First(&mistake, "id = ? AND student_id = ?", id, profile.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "mistake not found"})
		return
	}

	examID := mistake.ExamID
	subjectID := mistake.ExamSubjectID
	if input.ExamID != nil {
		examID = input.ExamID
	}
	if input.SubjectExamID != nil {
		subjectID = input.SubjectExamID
	}
	if err := validateMistakeRelationships(h.db, profile.ID, examID, subjectID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if input.ExamID != nil {
		updates["exam_id"] = *input.ExamID
	}
	if input.SubjectExamID != nil {
		updates["subject_exam_id"] = *input.SubjectExamID
	}
	if input.QuestionNumber != nil {
		updates["question_number"] = *input.QuestionNumber
	}
	if input.Category != nil {
		updates["category"] = strings.TrimSpace(*input.Category)
	}
	if input.Notes != nil {
		updates["notes"] = strings.TrimSpace(*input.Notes)
	}
	if input.DynamicFields != nil {
		updates["dynamic_fields"] = input.DynamicFields
	}

	if err := h.db.Model(&mistake).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update mistake"})
		return
	}

	if err := h.db.First(&mistake, "id = ?", mistake.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to reload mistake"})
		return
	}

	c.JSON(http.StatusOK, mistake)
}
