package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"gorm.io/gorm"
)

type ExamHandler struct {
	db *gorm.DB
}

type CreateExamInput struct {
	Title         string                 `json:"title" description:"عنوان آزمون"`
	ExamDate      *time.Time             `json:"exam_date" description:"تاریخ و زمان آزمون"`
	JalaliDate    string                 `json:"jalali_date" example:"1400/01/01" description:"تاریخ جلالی آزمون"`
	Major         string                 `json:"major" description:"رشته تحصیلی"`
	TotalSubjects int                    `json:"total_subjects" description:"تعداد کل دروس"`
	DynamicFields map[string]interface{} `json:"dynamic_fields" description:"فیلدهای سفارشی"`
	Subjects      []domain.ExamSubject   `json:"subjects" description:"دروس آزمون"`
}

type UpdateExamInput struct {
	Title         *string                `json:"title" description:"عنوان آزمون"`
	ExamDate      *time.Time             `json:"exam_date" description:"تاریخ و زمان آزمون"`
	JalaliDate    *string                `json:"jalali_date" example:"1400/01/01" description:"تاریخ جلالی آزمون"`
	Major         *string                `json:"major" description:"رشته تحصیلی"`
	TotalSubjects *int                   `json:"total_subjects" description:"تعداد کل دروس"`
	DynamicFields map[string]interface{} `json:"dynamic_fields" description:"فیلدهای سفارشی"`
	Subjects      []domain.ExamSubject   `json:"subjects" description:"دروس آزمون"`
}

type createExamInput = CreateExamInput

func NewExamHandler(db *gorm.DB) *ExamHandler {
	return &ExamHandler{db: db}
}

func (h *ExamHandler) CreateExam(c *gin.Context) {
	var input CreateExamInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
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

	if len(input.Subjects) == 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "at least one subject is required"})
		return
	}

	examDate, jalaliDate, err := parseFlexibleDateInput(input.ExamDate, input.JalaliDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	computedSubjects, err := computeExamSubjectMetrics(input.Subjects)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	totalSubjects := len(computedSubjects)
	if input.TotalSubjects > 0 {
		totalSubjects = input.TotalSubjects
	}

	exam := domain.Exam{
		StudentProfileID: profile.ID,
		Title:            strings.TrimSpace(input.Title),
		Date:             examDate,
		JalaliDate:       jalaliDate,
		Major:            strings.TrimSpace(input.Major),
		TotalSubjects:    totalSubjects,
		DynamicFields:    input.DynamicFields,
		Subjects:         computedSubjects,
	}

	if exam.Title == "" || exam.Major == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "title and major are required"})
		return
	}

	if err := h.db.Create(&exam).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create exam"})
		return
	}

	c.JSON(http.StatusCreated, exam)
}

func (h *ExamHandler) ListExams(c *gin.Context) {
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

	var exams []domain.Exam
	if err := h.db.Preload("Subjects").Where("student_id = ?", profile.ID).Order("exam_date desc").Find(&exams).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load exams"})
		return
	}

	c.JSON(http.StatusOK, exams)
}

func (h *ExamHandler) GetExam(c *gin.Context) {
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

	var exam domain.Exam
	if err := h.db.Preload("Subjects").First(&exam, "id = ? AND student_id = ?", id, profile.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "exam not found"})
		return
	}

	c.JSON(http.StatusOK, exam)
}

func (h *ExamHandler) DeleteExam(c *gin.Context) {
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

	if err := h.db.Where("id = ? AND student_id = ?", id, profile.ID).Delete(&domain.Exam{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to delete exam"})
		return
	}

	c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *ExamHandler) UpdateExam(c *gin.Context) {
	id := c.Param("id")
	userID, ok := getAuthenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}

	var input UpdateExamInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}

	profile, err := loadStudentProfileByUserID(h.db, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student profile not found"})
		return
	}

	var exam domain.Exam
	if err := h.db.Preload("Subjects").First(&exam, "id = ? AND student_id = ?", id, profile.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "exam not found"})
		return
	}

	updates := map[string]interface{}{}
	if input.Title != nil {
		updates["title"] = strings.TrimSpace(*input.Title)
	}
	if input.Major != nil {
		updates["major"] = strings.TrimSpace(*input.Major)
	}
	if input.TotalSubjects != nil {
		updates["total_subjects"] = *input.TotalSubjects
	}
	if input.DynamicFields != nil {
		updates["dynamic_fields"] = input.DynamicFields
	}
	if input.JalaliDate != nil || input.ExamDate != nil {
		jalali := ""
		if input.JalaliDate != nil {
			jalali = *input.JalaliDate
		}
		examDate, normalizedJalali, err := parseFlexibleDateInput(input.ExamDate, jalali)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}
		updates["exam_date"] = examDate
		updates["jalali_date"] = normalizedJalali
	}

	if len(updates) > 0 {
		if err := h.db.Model(&exam).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update exam"})
			return
		}
	}

	if input.Subjects != nil {
		computedSubjects, err := computeExamSubjectMetrics(input.Subjects)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
			return
		}
		if err := h.db.Where("exam_id = ?", exam.ID).Delete(&domain.ExamSubject{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update exam subjects"})
			return
		}
		for i := range computedSubjects {
			computedSubjects[i].ID = ""
			computedSubjects[i].ExamID = exam.ID
		}
		if len(computedSubjects) > 0 {
			if err := h.db.Create(&computedSubjects).Error; err != nil {
				c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update exam subjects"})
				return
			}
			if input.TotalSubjects == nil {
				h.db.Model(&exam).Update("total_subjects", len(computedSubjects))
			}
		}
	}

	if err := h.db.Preload("Subjects").First(&exam, "id = ?", exam.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to reload exam"})
		return
	}

	c.JSON(http.StatusOK, exam)
}
