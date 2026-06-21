package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"gorm.io/gorm"
)

type StatisticsHandler struct {
	db *gorm.DB
}

type ExamStatistics struct {
	TotalExams       int64               `json:"total_exams"`
	AverageScore     float64             `json:"average_score"`
	SubjectStats     []SubjectStatistics `json:"subject_stats"`
	TrendData        []TrendPoint        `json:"trend_data"`
	MistakesByReason map[string]int      `json:"mistakes_by_reason"`
}

type SubjectStatistics struct {
	SubjectName    string  `json:"subject_name"`
	TotalQuestions int     `json:"total_questions"`
	Correct        int     `json:"correct"`
	Wrong          int     `json:"wrong"`
	Blank          int     `json:"blank"`
	Percentage     float64 `json:"percentage"`
}

type TrendPoint struct {
	Date       string  `json:"date"`
	JalaliDate string  `json:"jalali_date"`
	Score      float64 `json:"score"`
	ExamCount  int     `json:"exam_count"`
}

func NewStatisticsHandler(db *gorm.DB) *StatisticsHandler {
	return &StatisticsHandler{db: db}
}

func statisticsStudentParam(c *gin.Context) string {
	if id := c.Param("id"); id != "" {
		return id
	}
	return c.Param("student_id")
}

func (h *StatisticsHandler) GetStudentStatistics(c *gin.Context) {
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

	stats, err := h.calculateStatistics(profile.ID, c.Query("from"), c.Query("to"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to calculate statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *StatisticsHandler) AdminGetStudentStatistics(c *gin.Context) {
	studentID := statisticsStudentParam(c)

	var student domain.StudentProfile
	if err := h.db.First(&student, "id = ?", studentID).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student not found"})
		return
	}

	stats, err := h.calculateStatistics(studentID, c.Query("from"), c.Query("to"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to calculate statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *StatisticsHandler) calculateStatistics(studentID, fromDate, toDate string) (*ExamStatistics, error) {
	query := h.db.Where("student_id = ?", studentID)
	if fromDate != "" {
		query = query.Where("jalali_date >= ?", fromDate)
	}
	if toDate != "" {
		query = query.Where("jalali_date <= ?", toDate)
	}

	var exams []domain.Exam
	if err := query.Preload("Subjects").Order("exam_date asc").Find(&exams).Error; err != nil {
		return nil, err
	}

	stats := &ExamStatistics{
		TotalExams:       int64(len(exams)),
		SubjectStats:     make([]SubjectStatistics, 0),
		TrendData:        make([]TrendPoint, 0),
		MistakesByReason: make(map[string]int),
	}
	if len(exams) == 0 {
		return stats, nil
	}

	subjectMap := make(map[string]*SubjectStatistics)
	var totalScore float64
	var examCount int

	for _, exam := range exams {
		examScore := 0.0
		examTotal := 0
		for _, subject := range exam.Subjects {
			if _, exists := subjectMap[subject.SubjectName]; !exists {
				subjectMap[subject.SubjectName] = &SubjectStatistics{SubjectName: subject.SubjectName}
			}
			s := subjectMap[subject.SubjectName]
			s.TotalQuestions += subject.TotalQuestions
			s.Correct += subject.Correct
			s.Wrong += subject.Wrong
			s.Blank += subject.Blank
			if subject.TotalQuestions > 0 {
				examScore += subject.Percentage
				examTotal++
			}
		}
		if examTotal > 0 {
			examPercentage := examScore / float64(examTotal)
			totalScore += examPercentage
			examCount++
			stats.TrendData = append(stats.TrendData, TrendPoint{
				Date:       exam.Date.Format("2006-01-02"),
				JalaliDate: exam.JalaliDate,
				Score:      examPercentage,
				ExamCount:  examCount,
			})
		}
	}

	for _, subjectStat := range subjectMap {
		if subjectStat.TotalQuestions > 0 {
			subjectStat.Percentage = (float64(subjectStat.Correct) / float64(subjectStat.TotalQuestions)) * 100
		}
		stats.SubjectStats = append(stats.SubjectStats, *subjectStat)
	}
	if examCount > 0 {
		stats.AverageScore = totalScore / float64(examCount)
	}

	var mistakes []domain.MistakeAnalysis
	if err := h.db.Where("student_id = ?", studentID).Find(&mistakes).Error; err == nil {
		for _, mistake := range mistakes {
			if mistake.Category != "" {
				stats.MistakesByReason[mistake.Category]++
			}
		}
	}

	return stats, nil
}

func (h *StatisticsHandler) GetDashboardSummary(c *gin.Context) {
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

	var examCount int64
	h.db.Model(&domain.Exam{}).Where("student_id = ?", profile.ID).Count(&examCount)

	var mistakeCount int64
	h.db.Model(&domain.MistakeAnalysis{}).Where("student_id = ?", profile.ID).Count(&mistakeCount)

	var recentExams []domain.Exam
	h.db.Where("student_id = ?", profile.ID).Order("exam_date desc").Limit(5).Preload("Subjects").Find(&recentExams)

	stats, _ := h.calculateStatistics(profile.ID, "", "")

	var studyPlanCount int64
	h.db.Model(&domain.StudyPlan{}).Where("student_id = ?", profile.ID).Count(&studyPlanCount)

	summary := map[string]interface{}{
		"total_exams":    examCount,
		"total_mistakes": mistakeCount,
		"recent_exams":   recentExams,
		"average_score":  stats.AverageScore,
		"is_approved":    profile.IsApproved,
		"status":         profile.Status,
		"has_study_plan": studyPlanCount > 0,
	}

	c.JSON(http.StatusOK, summary)
}
