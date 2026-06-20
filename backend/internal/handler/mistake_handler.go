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

// CreateMistakeInput داده‌های ورودی برای ایجاد اشتباه
type CreateMistakeInput struct {
	ExamID         *string                `json:"exam_id" description:"شناسه آزمون (اختیاری)"`
	SubjectExamID  *string                `json:"subject_exam_id" description:"شناسه درس آزمون (اختیاری)"`
	QuestionNumber int                    `json:"question_number" description:"شماره سؤال"`
	Category       string                 `json:"category" description:"دسته‌بندی اشتباه"`
	Notes          string                 `json:"notes" description:"یادداشت‌های توضیحی"`
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

	c.JSON(http.StatusOK, mistake)
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
