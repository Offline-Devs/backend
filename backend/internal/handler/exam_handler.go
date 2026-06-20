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

// CreateExamInput داده‌های ورودی برای ایجاد آزمون
type CreateExamInput struct {
	Title         string                 `json:"title" description:"عنوان آزمون"`
	ExamDate      *time.Time             `json:"exam_date" description:"تاریخ و زمان آزمون"`
	JalaliDate    string                 `json:"jalali_date" example:"1400/01/01" description:"تاریخ جلالی آزمون"`
	Major         string                 `json:"major" description:"رشته تحصیلی"`
	TotalSubjects int                    `json:"total_subjects" description:"تعداد کل دروس"`
	DynamicFields map[string]interface{} `json:"dynamic_fields" description:"فیلدهای سفارشی"`
	Subjects      []domain.SubjectExam   `json:"subjects" description:"دروس آزمون"`
}

// UpdateExamInput داده‌های ورودی برای بروزرسانی آزمون
type UpdateExamInput struct {
	Title         *string                `json:"title" description:"عنوان آزمون"`
	ExamDate      *time.Time             `json:"exam_date" description:"تاریخ و زمان آزمون"`
	JalaliDate    *string                `json:"jalali_date" example:"1400/01/01" description:"تاریخ جلالی آزمون"`
	Major         *string                `json:"major" description:"رشته تحصیلی"`
	TotalSubjects *int                   `json:"total_subjects" description:"تعداد کل دروس"`
	DynamicFields map[string]interface{} `json:"dynamic_fields" description:"فیلدهای سفارشی"`
	Subjects      []domain.SubjectExam   `json:"subjects" description:"دروس آزمون"`
}

// Deprecated: استفاده از CreateExamInput کنید
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

// CreateExam godoc
// @Summary ایجاد آزمون جدید
// @Description یک آزمون جدید برای دانشجو ایجاد می‌کند
// @Tags آزمون‌ها
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body CreateExamInput true "اطلاعات آزمون"
// @Success 201 {object} domain.Exam "آزمون با موفقیت ایجاد شد"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "پروفایل دانشجو یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /exams [post]
func (h *ExamHandler) CreateExam(c *gin.Context) {
	var input CreateExamInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}

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

	examDate := time.Now()
	if input.JalaliDate != "" {
		t, err := pkg.JalaliToGregorian(input.JalaliDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid jalali_date format"})
			return
		}
		examDate = t
	} else if input.ExamDate != nil {
		examDate = *input.ExamDate
		input.JalaliDate = pkg.GregorianToJalaliString(examDate)
	} else {
		input.JalaliDate = pkg.GregorianToJalaliString(examDate)
	}

	for i := range input.Subjects {
		input.Subjects[i].ID = ""
		input.Subjects[i].ExamID = ""
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
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create exam"})
		return
	}

	c.JSON(http.StatusCreated, exam)
}

// ListExams godoc
// @Summary دریافت لیست آزمون‌های دانشجو
// @Description تمام آزمون‌های دانشجو را دریافت می‌کند
// @Tags آزمون‌ها
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.Exam "لیست آزمون‌ها"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "پروفایل دانشجو یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /exams [get]
func (h *ExamHandler) ListExams(c *gin.Context) {
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

	var exams []domain.Exam
	if err := h.db.Preload("Subjects").Where("student_id = ?", student.ID).Order("created_at desc").Find(&exams).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load exams"})
		return
	}

	c.JSON(http.StatusOK, exams)
}

// GetExam godoc
// @Summary دریافت جزئیات آزمون
// @Description جزئیات یک آزمون خاص را دریافت می‌کند
// @Tags آزمون‌ها
// @Security BearerAuth
// @Produce json
// @Param id path string true "شناسه آزمون"
// @Success 200 {object} domain.Exam "جزئیات آزمون"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "آزمون یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /exams/{id} [get]
func (h *ExamHandler) GetExam(c *gin.Context) {
	id := c.Param("id")
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

	var exam domain.Exam
	if err := h.db.Preload("Subjects").First(&exam, "id = ? AND student_id = ?", id, student.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "exam not found"})
		return
	}

	c.JSON(http.StatusOK, exam)
}

// DeleteExam godoc
// @Summary حذف آزمون
// @Description یک آزمون را حذف می‌کند
// @Tags آزمون‌ها
// @Security BearerAuth
// @Param id path string true "شناسه آزمون"
// @Success 200 {object} map[string]string "آزمون با موفقیت حذف شد"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "پروفایل دانشجو یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /exams/{id} [delete]
func (h *ExamHandler) DeleteExam(c *gin.Context) {
	id := c.Param("id")
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

	if err := h.db.Where("id = ? AND student_id = ?", id, student.ID).Delete(&domain.Exam{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to delete exam"})
		return
	}

	c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// UpdateExam godoc
// @Summary بروزرسانی آزمون
// @Description یک آزمون موجود را بروزرسانی می‌کند
// @Tags آزمون‌ها
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "شناسه آزمون"
// @Param input body UpdateExamInput true "اطلاعات آزمون"
// @Success 200 {object} domain.Exam "آزمون با موفقیت بروزرسانی شد"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "آزمون یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /exams/{id} [put]
func (h *ExamHandler) UpdateExam(c *gin.Context) {
	id := c.Param("id")
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}

	var input UpdateExamInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}

	var student domain.Student
	if err := h.db.Where("user_id = ?", userID).First(&student).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student profile not found"})
		return
	}

	var exam domain.Exam
	if err := h.db.Preload("Subjects").First(&exam, "id = ? AND student_id = ?", id, student.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "exam not found"})
		return
	}

	updates := map[string]interface{}{}
	if input.Title != nil {
		updates["title"] = *input.Title
	}
	if input.Major != nil {
		updates["major"] = *input.Major
	}
	if input.TotalSubjects != nil {
		updates["total_subjects"] = *input.TotalSubjects
	}
	if input.DynamicFields != nil {
		updates["dynamic_fields"] = input.DynamicFields
	}
	if input.JalaliDate != nil {
		t, err := pkg.JalaliToGregorian(*input.JalaliDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid jalali_date format"})
			return
		}
		updates["jalali_date"] = *input.JalaliDate
		updates["exam_date"] = t
	} else if input.ExamDate != nil {
		updates["exam_date"] = *input.ExamDate
		updates["jalali_date"] = pkg.GregorianToJalaliString(*input.ExamDate)
	}

	if err := h.db.Model(&exam).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update exam"})
		return
	}

	if input.Subjects != nil {
		if err := h.db.Where("exam_id = ?", exam.ID).Delete(&domain.SubjectExam{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update exam subjects"})
			return
		}
		for i := range input.Subjects {
			input.Subjects[i].ID = ""
			input.Subjects[i].ExamID = exam.ID
		}
		if len(input.Subjects) > 0 {
			if err := h.db.Create(&input.Subjects).Error; err != nil {
				c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update exam subjects"})
				return
			}
		}
	}

	if err := h.db.Preload("Subjects").First(&exam, "id = ?", exam.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to reload exam"})
		return
	}

	c.JSON(http.StatusOK, exam)
}
