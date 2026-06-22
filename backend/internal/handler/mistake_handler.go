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

// CreateMistakeInput داده‌های ورودی برای ایجاد اشتباه
type CreateMistakeInput struct {
	ExamID         *string                `json:"exam_id" description:"شناسه آزمون (اختیاری)"`
	SubjectExamID  *string                `json:"subject_exam_id" description:"شناسه درس آزمون (اختیاری)"`
	QuestionNumber int                    `json:"question_number" description:"شماره سؤال"`
	Category       string                 `json:"category" description:"دسته‌بندی اشتباه"`
	Notes          string                 `json:"notes" description:"یادداشت‌های توضیحی"`
	DynamicFields  map[string]interface{} `json:"dynamic_fields" description:"فیلدهای سفارشی"`
}

// UpdateMistakeInput داده‌های ورودی برای بروزرسانی اشتباه
type UpdateMistakeInput struct {
	ExamID         *string                `json:"exam_id" description:"شناسه آزمون (اختیاری)"`
	SubjectExamID  *string                `json:"subject_exam_id" description:"شناسه درس آزمون (اختیاری)"`
	QuestionNumber *int                   `json:"question_number" description:"شماره سؤال"`
	Category       *string                `json:"category" description:"دسته‌بندی اشتباه"`
	Notes          *string                `json:"notes" description:"یادداشت‌های توضیحی"`
	DynamicFields  map[string]interface{} `json:"dynamic_fields" description:"فیلدهای سفارشی"`
}

// Deprecated: استفاده از CreateMistakeInput کنید
type createMistakeInput struct {
	ExamID         *string                `json:"exam_id"`
	SubjectExamID  *string                `json:"subject_exam_id"`
	QuestionNumber int                    `json:"question_number"`
	Category       string                 `json:"category"`
	Notes          string                 `json:"notes"`
	DynamicFields  map[string]interface{} `json:"dynamic_fields"`
}

func NewMistakeHandler(db *gorm.DB) *MistakeHandler {
	return &MistakeHandler{db: db}
}

func (h *MistakeHandler) validateReferences(studentID string, examID, subjectExamID *string) error {
	var validatedExamID *string
	if examID != nil {
		trimmed := strings.TrimSpace(*examID)
		if trimmed == "" {
			return gorm.ErrRecordNotFound
		}
		var exam domain.Exam
		if err := h.db.Select("id").Where("id = ? AND student_id = ?", trimmed, studentID).First(&exam).Error; err != nil {
			return err
		}
		validatedExamID = &exam.ID
	}

	if subjectExamID != nil {
		trimmed := strings.TrimSpace(*subjectExamID)
		if trimmed == "" {
			return gorm.ErrRecordNotFound
		}
		var subject domain.SubjectExam
		query := h.db.
			Table("subject_exams").
			Select("subject_exams.id, subject_exams.exam_id").
			Joins("JOIN exams ON exams.id = subject_exams.exam_id").
			Where("subject_exams.id = ? AND exams.student_id = ?", trimmed, studentID)
		if err := query.First(&subject).Error; err != nil {
			return err
		}
		if validatedExamID != nil && subject.ExamID != *validatedExamID {
			return gorm.ErrRecordNotFound
		}
	}

	return nil
}

// Create godoc
// @Summary ایجاد اشتباه جدید
// @Description یک اشتباه جدید برای دانشجو ثبت می‌کند
// @Tags اشتباهات
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body CreateMistakeInput true "اطلاعات اشتباه"
// @Success 201 {object} domain.Mistake "اشتباه با موفقیت ایجاد شد"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "پروفایل دانشجو یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /mistakes [post]
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
	if err := h.validateReferences(student.ID, input.ExamID, input.SubjectExamID); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "referenced exam or subject not found"})
		return
	}

	mistake := domain.Mistake{
		StudentID:      student.ID,
		ExamID:         input.ExamID,
		SubjectExamID:  input.SubjectExamID,
		QuestionNumber: input.QuestionNumber,
		Category:       input.Category,
		Notes:          input.Notes,
		DynamicFields:  input.DynamicFields,
	}

	if err := h.db.Create(&mistake).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create mistake"})
		return
	}

	c.JSON(http.StatusCreated, mistake)
}

// List godoc
// @Summary دریافت لیست اشتباهات دانشجو
// @Description تمام اشتباهات ثبت‌شده برای دانشجو را دریافت می‌کند
// @Tags اشتباهات
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.Mistake "لیست اشتباهات"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "پروفایل دانشجو یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /mistakes [get]
func (h *MistakeHandler) List(c *gin.Context) {
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

	var mistakes []domain.Mistake
	if err := h.db.Where("student_id = ?", student.ID).Order("created_at desc").Find(&mistakes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load mistakes"})
		return
	}
	c.JSON(http.StatusOK, mistakes)
}

// Delete godoc
// @Summary حذف اشتباه
// @Description یک اشتباه را حذف می‌کند
// @Tags اشتباهات
// @Security BearerAuth
// @Param id path string true "شناسه اشتباه"
// @Success 200 {object} map[string]string "اشتباه با موفقیت حذف شد"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "پروفایل دانشجو یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /mistakes/{id} [delete]
func (h *MistakeHandler) Delete(c *gin.Context) {
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

	if err := h.db.Where("id = ? AND student_id = ?", id, student.ID).Delete(&domain.Mistake{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to delete mistake"})
		return
	}

	c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// Update godoc
// @Summary بروزرسانی اشتباه
// @Description یک اشتباه ثبت‌شده را بروزرسانی می‌کند
// @Tags اشتباهات
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path string true "شناسه اشتباه"
// @Param input body UpdateMistakeInput true "اطلاعات اشتباه"
// @Success 200 {object} domain.Mistake "اشتباه با موفقیت بروزرسانی شد"
// @Failure 400 {object} ErrorResponse "درخواست نامعتبر"
// @Failure 401 {object} ErrorResponse "عدم اجازه دسترسی"
// @Failure 404 {object} ErrorResponse "اشتباه یافت نشد"
// @Failure 500 {object} ErrorResponse "خطای سرور"
// @Router /mistakes/{id} [put]
func (h *MistakeHandler) Update(c *gin.Context) {
	id := c.Param("id")
	userID, ok := c.Get("user_id")
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

	var student domain.Student
	if err := h.db.Where("user_id = ?", userID).First(&student).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student profile not found"})
		return
	}

	var mistake domain.Mistake
	if err := h.db.First(&mistake, "id = ? AND student_id = ?", id, student.ID).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "mistake not found"})
		return
	}

	examID := mistake.ExamID
	if input.ExamID != nil {
		examID = input.ExamID
	}
	subjectExamID := mistake.SubjectExamID
	if input.SubjectExamID != nil {
		subjectExamID = input.SubjectExamID
	}
	if err := h.validateReferences(student.ID, examID, subjectExamID); err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "referenced exam or subject not found"})
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
		updates["category"] = *input.Category
	}
	if input.Notes != nil {
		updates["notes"] = *input.Notes
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
