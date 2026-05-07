package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/yourusername/noshirvani-academy/backend/internal/domain"
    "gorm.io/gorm"
)

type MistakeHandler struct {
    db *gorm.DB
}

type createMistakeInput struct {
    ExamID        *string                `json:"exam_id"`
    SubjectExamID *string                `json:"subject_exam_id"`
    QuestionNumber int                   `json:"question_number"`
    Category      string                 `json:"category"`
    Notes         string                 `json:"notes"`
    DynamicFields map[string]interface{} `json:"dynamic_fields"`
}

func NewMistakeHandler(db *gorm.DB) *MistakeHandler {
    return &MistakeHandler{db: db}
}

func (h *MistakeHandler) Create(c *gin.Context) {
    var input createMistakeInput
    if err := c.ShouldBindJSON(&input); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
        return
    }

    userID, ok := c.Get("user_id")
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
        return
    }

    var student domain.Student
    if err := h.db.Where("user_id = ?", userID).First(&student).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "student profile not found"})
        return
    }

    mistake := domain.Mistake{
        StudentID:     student.ID,
        ExamID:        input.ExamID,
        SubjectExamID: input.SubjectExamID,
        QuestionNumber: input.QuestionNumber,
        Category:      input.Category,
        Notes:         input.Notes,
        DynamicFields: input.DynamicFields,
    }

    if err := h.db.Create(&mistake).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create mistake"})
        return
    }

    c.JSON(http.StatusOK, mistake)
}

func (h *MistakeHandler) List(c *gin.Context) {
    userID, ok := c.Get("user_id")
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
        return
    }

    var student domain.Student
    if err := h.db.Where("user_id = ?", userID).First(&student).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "student profile not found"})
        return
    }

    var mistakes []domain.Mistake
    if err := h.db.Where("student_id = ?", student.ID).Order("created_at desc").Find(&mistakes).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load mistakes"})
        return
    }
    c.JSON(http.StatusOK, mistakes)
}

func (h *MistakeHandler) Delete(c *gin.Context) {
    id := c.Param("id")
    userID, ok := c.Get("user_id")
    if !ok {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "missing user id"})
        return
    }

    var student domain.Student
    if err := h.db.Where("user_id = ?", userID).First(&student).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "student profile not found"})
        return
    }

    if err := h.db.Where("id = ? AND student_id = ?", id, student.ID).Delete(&domain.Mistake{}).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete mistake"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
