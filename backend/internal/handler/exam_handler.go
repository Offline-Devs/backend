package handler

import (
	"net/http"
	"strings"
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
	NegativeMark  float64                `json:"negative_mark" description:"نمره منفی هر پاسخ غلط"`
	TotalSubjects int                    `json:"total_subjects" description:"تعداد کل دروس"`
	DynamicFields map[string]interface{} `json:"dynamic_fields" description:"فیلدهای سفارشی"`
	Subjects      []ExamSubjectInput     `json:"subjects" description:"دروس آزمون"`
}

// UpdateExamInput داده‌های ورودی برای بروزرسانی آزمون
type UpdateExamInput struct {
	Title         *string                `json:"title" description:"عنوان آزمون"`
	ExamDate      *time.Time             `json:"exam_date" description:"تاریخ و زمان آزمون"`
	JalaliDate    *string                `json:"jalali_date" example:"1400/01/01" description:"تاریخ جلالی آزمون"`
	Major         *string                `json:"major" description:"رشته تحصیلی"`
	NegativeMark  *float64               `json:"negative_mark" description:"نمره منفی هر پاسخ غلط"`
	TotalSubjects *int                   `json:"total_subjects" description:"تعداد کل دروس"`
	DynamicFields map[string]interface{} `json:"dynamic_fields" description:"فیلدهای سفارشی"`
	Subjects      []ExamSubjectInput     `json:"subjects" description:"دروس آزمون"`
}

type ExamSubjectInput struct {
	SubjectName    string `json:"subject_name" description:"نام درس"`
	TotalQuestions int    `json:"total_questions" description:"تعداد کل سوالات"`
	Correct        int    `json:"correct" description:"تعداد پاسخ صحیح"`
	Wrong          int    `json:"wrong" description:"تعداد پاسخ غلط"`
}

// Deprecated: استفاده از CreateExamInput کنید
type createExamInput struct {
	Title         string                 `json:"title"`
	ExamDate      *time.Time             `json:"exam_date"`
	JalaliDate    string                 `json:"jalali_date"`
	Major         string                 `json:"major"`
	NegativeMark  float64                `json:"negative_mark"`
	TotalSubjects int                    `json:"total_subjects"`
	DynamicFields map[string]interface{} `json:"dynamic_fields"`
	Subjects      []domain.SubjectExam   `json:"subjects"`
}

func NewExamHandler(db *gorm.DB) *ExamHandler {
	return &ExamHandler{db: db}
}

func normalizeSubjects(inputs []ExamSubjectInput) ([]domain.SubjectExam, error) {
	subjects := make([]domain.SubjectExam, len(inputs))
	for i, input := range inputs {
		name := strings.TrimSpace(input.SubjectName)
		if name == "" {
			return nil, gorm.ErrInvalidData
		}
		if input.TotalQuestions < 0 || input.Correct < 0 || input.Wrong < 0 {
			return nil, gorm.ErrInvalidData
		}
		if input.Correct+input.Wrong > input.TotalQuestions {
			return nil, gorm.ErrInvalidData
		}
		subjects[i] = domain.SubjectExam{
			SubjectName:    name,
			TotalQuestions: input.TotalQuestions,
			Correct:        input.Correct,
			Wrong:          input.Wrong,
		}
	}
	return subjects, nil
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
	if input.NegativeMark < 0 || input.NegativeMark > 1 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "negative_mark must be between 0 and 1"})
		return
	}
	subjects, err := normalizeSubjects(input.Subjects)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid subject counts"})
		return
	}

	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}
	if input.JalaliDate != "" && input.ExamDate != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "exam_date and jalali_date are mutually exclusive"})
		return
	}

	var student domain.Student
	if err := h.db.Where("user_id = ?", userID).First(&student).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student profile not found"})
		return
	}

	examDate := time.Now()
	if input.JalaliDate != "" {
		canonicalDate, err := pkg.CanonicalJalaliDate(input.JalaliDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid jalali_date format"})
			return
		}
		t, _ := pkg.JalaliToGregorian(canonicalDate)
		examDate = t
		input.JalaliDate = canonicalDate
	} else if input.ExamDate != nil {
		examDate = *input.ExamDate
		input.JalaliDate = pkg.GregorianToJalaliString(examDate)
	} else {
		input.JalaliDate = pkg.GregorianToJalaliString(examDate)
	}

	exam := domain.Exam{
		StudentID:     student.ID,
		Title:         input.Title,
		ExamDate:      examDate,
		JalaliDate:    input.JalaliDate,
		Major:         input.Major,
		NegativeMark:  input.NegativeMark,
		TotalSubjects: input.TotalSubjects,
		DynamicFields: input.DynamicFields,
		Subjects:      subjects,
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

	result := h.db.Where("id = ? AND student_id = ?", id, student.ID).Delete(&domain.Exam{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to delete exam"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "exam not found"})
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
	if input.JalaliDate != nil && input.ExamDate != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "exam_date and jalali_date are mutually exclusive"})
		return
	}
	var subjects []domain.SubjectExam
	if input.Subjects != nil {
		var err error
		subjects, err = normalizeSubjects(input.Subjects)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid subject counts"})
			return
		}
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
	if input.NegativeMark != nil {
		if *input.NegativeMark < 0 || *input.NegativeMark > 1 {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "negative_mark must be between 0 and 1"})
			return
		}
		updates["negative_mark"] = *input.NegativeMark
	}
	if input.TotalSubjects != nil {
		updates["total_subjects"] = *input.TotalSubjects
	}
	if input.DynamicFields != nil {
		updates["dynamic_fields"] = input.DynamicFields
	}
	if input.JalaliDate != nil {
		canonicalDate, err := pkg.CanonicalJalaliDate(*input.JalaliDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid jalali_date format"})
			return
		}
		t, _ := pkg.JalaliToGregorian(canonicalDate)
		updates["jalali_date"] = canonicalDate
		updates["exam_date"] = t
	} else if input.ExamDate != nil {
		updates["exam_date"] = *input.ExamDate
		updates["jalali_date"] = pkg.GregorianToJalaliString(*input.ExamDate)
	}

	if err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&exam).Updates(updates).Error; err != nil {
			return err
		}

		if input.Subjects != nil {
			if err := tx.Model(&domain.Mistake{}).
				Where("subject_exam_id IN (SELECT id FROM subject_exams WHERE exam_id = ?)", exam.ID).
				Update("subject_exam_id", nil).Error; err != nil {
				return err
			}
			if err := tx.Where("exam_id = ?", exam.ID).Delete(&domain.SubjectExam{}).Error; err != nil {
				return err
			}
			for i := range subjects {
				subjects[i].ExamID = exam.ID
			}
			if len(subjects) > 0 {
				if err := tx.Create(&subjects).Error; err != nil {
					return err
				}
			}
		}

		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update exam"})
		return
	}

	if err := h.db.Preload("Subjects").First(&exam, "id = ?", exam.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to reload exam"})
		return
	}

	c.JSON(http.StatusOK, exam)
}
