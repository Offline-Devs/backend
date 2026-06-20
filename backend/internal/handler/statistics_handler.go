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

// ExamStatistics آمار آزمون‌ها
type ExamStatistics struct {
	TotalExams       int64               `json:"total_exams"`
	AverageScore     float64             `json:"average_score"`
	SubjectStats     []SubjectStatistics `json:"subject_stats"`
	TrendData        []TrendPoint        `json:"trend_data"`
	MistakesByReason map[string]int      `json:"mistakes_by_reason"`
}

// SubjectStatistics آمار دروس
type SubjectStatistics struct {
	SubjectName    string  `json:"subject_name"`
	TotalQuestions int     `json:"total_questions"`
	Correct        int     `json:"correct"`
	Wrong          int     `json:"wrong"`
	Blank          int     `json:"blank"`
	Percentage     float64 `json:"percentage"`
}

// TrendPoint نقطه روند عملکرد
type TrendPoint struct {
	Date       string  `json:"date"`
	JalaliDate string  `json:"jalali_date"`
	Score      float64 `json:"score"`
	ExamCount  int     `json:"exam_count"`
}

func NewStatisticsHandler(db *gorm.DB) *StatisticsHandler {
	return &StatisticsHandler{db: db}
}

// GetStudentStatistics godoc
// @Summary دریافت آمار عملکرد دانشجو
// @Description آمار کامل عملکرد دانشجو در آزمون‌ها را دریافت می‌کند
// @Tags آمار
// @Security BearerAuth
// @Produce json
// @Param from query string false "تاریخ شروع (جلالی YYYY/MM/DD)"
// @Param to query string false "تاریخ پایان (جلالی YYYY/MM/DD)"
// @Success 200 {object} ExamStatistics "آمار عملکرد"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "پروفایل دانشجو یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /students/statistics [get]
func (h *StatisticsHandler) GetStudentStatistics(c *gin.Context) {
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

	stats, err := h.calculateStatistics(student.ID, c.Query("from"), c.Query("to"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to calculate statistics"})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// AdminGetStudentStatistics godoc
// @Summary دریافت آمار عملکرد یک دانشجو (مدیر)
// @Description مدیر می‌تواند آمار کامل یک دانشجو را مشاهده کند
// @Tags مدیریت
// @Security BearerAuth
// @Produce json
// @Param student_id path string true "شناسه دانشجو"
// @Param from query string false "تاریخ شروع (جلالی YYYY/MM/DD)"
// @Param to query string false "تاریخ پایان (جلالی YYYY/MM/DD)"
// @Success 200 {object} ExamStatistics "آمار عملکرد"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "دانشجو یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /admin/students/{student_id}/statistics [get]
func (h *StatisticsHandler) AdminGetStudentStatistics(c *gin.Context) {
	studentID := c.Param("student_id")

	// Verify student exists
	var student domain.Student
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

	// Apply date filters if provided
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

	// Calculate subject statistics
	subjectMap := make(map[string]*SubjectStatistics)
	var totalScore float64
	var examCount int

	for _, exam := range exams {
		examScore := 0.0
		examTotal := 0

		for _, subject := range exam.Subjects {
			if _, exists := subjectMap[subject.SubjectName]; !exists {
				subjectMap[subject.SubjectName] = &SubjectStatistics{
					SubjectName: subject.SubjectName,
				}
			}

			s := subjectMap[subject.SubjectName]
			s.TotalQuestions += subject.TotalQuestions
			s.Correct += subject.Correct
			s.Wrong += subject.Wrong
			s.Blank += subject.Blank

			if subject.TotalQuestions > 0 {
				examScore += float64(subject.Correct)
				examTotal += subject.TotalQuestions
			}
		}

		if examTotal > 0 {
			examPercentage := (examScore / float64(examTotal)) * 100
			totalScore += examPercentage
			examCount++

			// Add trend point
			stats.TrendData = append(stats.TrendData, TrendPoint{
				Date:       exam.ExamDate.Format("2006-01-02"),
				JalaliDate: exam.JalaliDate,
				Score:      examPercentage,
				ExamCount:  examCount,
			})
		}
	}

	// Calculate percentages for subjects
	for _, subjectStat := range subjectMap {
		if subjectStat.TotalQuestions > 0 {
			subjectStat.Percentage = (float64(subjectStat.Correct) / float64(subjectStat.TotalQuestions)) * 100
		}
		stats.SubjectStats = append(stats.SubjectStats, *subjectStat)
	}

	// Calculate average score
	if examCount > 0 {
		stats.AverageScore = totalScore / float64(examCount)
	}

	// Get mistake statistics
	var mistakes []domain.Mistake
	mistakeQuery := h.db.Where("student_id = ?", studentID)
	if err := mistakeQuery.Find(&mistakes).Error; err == nil {
		for _, mistake := range mistakes {
			if mistake.Category != "" {
				stats.MistakesByReason[mistake.Category]++
			}
		}
	}

	return stats, nil
}

// GetDashboardSummary godoc
// @Summary دریافت خلاصه داشبورد دانشجو
// @Description خلاصه‌ای از آمار کلی دانشجو برای نمایش در داشبورد
// @Tags آمار
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{} "خلاصه داشبورد"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "پروفایل دانشجو یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /students/dashboard [get]
func (h *StatisticsHandler) GetDashboardSummary(c *gin.Context) {
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

	// Get total exams count
	var examCount int64
	h.db.Model(&domain.Exam{}).Where("student_id = ?", student.ID).Count(&examCount)

	// Get total mistakes count
	var mistakeCount int64
	h.db.Model(&domain.Mistake{}).Where("student_id = ?", student.ID).Count(&mistakeCount)

	// Get recent exams
	var recentExams []domain.Exam
	h.db.Where("student_id = ?", student.ID).
		Order("exam_date desc").
		Limit(5).
		Preload("Subjects").
		Find(&recentExams)

	stats, _ := h.calculateStatistics(student.ID, "", "")

	summary := map[string]interface{}{
		"total_exams":    examCount,
		"total_mistakes": mistakeCount,
		"recent_exams":   recentExams,
		"average_score":  stats.AverageScore,
		"is_approved":    student.IsApproved,
		"has_study_plan": false,
	}

	// Check if student has any study plans
	var performanceCount int64
	h.db.Model(&domain.PerformanceHistory{}).
		Where("student_id = ? AND study_plan != ''", student.ID).
		Count(&performanceCount)
	summary["has_study_plan"] = performanceCount > 0

	c.JSON(http.StatusOK, summary)
}
