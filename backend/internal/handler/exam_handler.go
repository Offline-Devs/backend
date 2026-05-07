package handler

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/yourusername/noshirvani-academy/backend/internal/domain"
    "github.com/yourusername/noshirvani-academy/backend/pkg"
    "gorm.io/gorm"
)

type ExamHandler struct {
    db *gorm.DB
}

type createExamInput struct {
    Title         string                 `json:"title"`
    ExamDate      *time.Time             `json:"exam_date"`
    JalaliDate    string                 `json:"jalali_date"`
    Major         string                 `json:"major"`
    TotalSubjects int                    `json:"total_subjects"`
    DynamicFields map[string]interface{} `json:"dynamic_fields"`
    Subjects      []domain.SubjectExam   `json:"subjects"`
}

func NewExamHandler(db *gorm.DB) *ExamHandler {
    return &ExamHandler{db: db}
}

func (h *ExamHandler) CreateExam(c *gin.Context) {
    var input createExamInput
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

    examDate := time.Now()
    if input.JalaliDate != "" {
        if t, err := pkg.JalaliToGregorian(input.JalaliDate); err == nil {
            examDate = t
        }
    } else if input.ExamDate != nil {
        examDate = *input.ExamDate
        input.JalaliDate = pkg.GregorianToJalaliString(examDate)
    }

    exam := domain.Exam{
        StudentID:     student.ID,
        Title:         input.Title,
        ExamDate:      examDate,
        JalaliDate:    input.JalaliDate,
        Major:         input.Major,
        TotalSubjects: input.TotalSubjects,
        DynamicFields: input.DynamicFields,
        Subjects:      input.Subjects,
    }

    if err := h.db.Create(&exam).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create exam"})
        return
    }

    c.JSON(http.StatusOK, exam)
}

func (h *ExamHandler) ListExams(c *gin.Context) {
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

    var exams []domain.Exam
    if err := h.db.Preload("Subjects").Where("student_id = ?", student.ID).Order("created_at desc").Find(&exams).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load exams"})
        return
    }

    c.JSON(http.StatusOK, exams)
}

func (h *ExamHandler) GetExam(c *gin.Context) {
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

    var exam domain.Exam
    if err := h.db.Preload("Subjects").First(&exam, "id = ? AND student_id = ?", id, student.ID).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "exam not found"})
        return
    }

    c.JSON(http.StatusOK, exam)
}

func (h *ExamHandler) DeleteExam(c *gin.Context) {
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

    if err := h.db.Where("id = ? AND student_id = ?", id, student.ID).Delete(&domain.Exam{}).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete exam"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
