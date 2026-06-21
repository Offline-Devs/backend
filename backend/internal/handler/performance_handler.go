package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"gorm.io/gorm"
)

type PerformanceHandler struct {
	db *gorm.DB
}

type CreateStudyPlanInput struct {
	Title           string     `json:"title" description:"عنوان برنامه مطالعاتی"`
	Description     string     `json:"description" description:"توضیحات برنامه مطالعاتی"`
	StartDate       *time.Time `json:"start_date" description:"تاریخ شروع میلادی"`
	EndDate         *time.Time `json:"end_date" description:"تاریخ پایان میلادی"`
	JalaliStartDate string     `json:"jalali_start_date" description:"تاریخ شروع جلالی"`
	JalaliEndDate   string     `json:"jalali_end_date" description:"تاریخ پایان جلالی"`
	Attachments     string     `json:"attachments" description:"فایل های پیوست"`
}

type UpdateStudyPlanInput struct {
	Title           *string    `json:"title"`
	Description     *string    `json:"description"`
	StartDate       *time.Time `json:"start_date"`
	EndDate         *time.Time `json:"end_date"`
	JalaliStartDate *string    `json:"jalali_start_date"`
	JalaliEndDate   *string    `json:"jalali_end_date"`
	Attachments     *string    `json:"attachments"`
}

type CreateAdminNoteInput struct {
	Title         string     `json:"title"`
	Body          string     `json:"body"`
	Date          *time.Time `json:"date"`
	JalaliDate    string     `json:"jalali_date"`
	AttachmentURL string     `json:"attachment_url"`
}

type UpdateAdminNoteInput struct {
	Title         *string    `json:"title"`
	Body          *string    `json:"body"`
	Date          *time.Time `json:"date"`
	JalaliDate    *string    `json:"jalali_date"`
	AttachmentURL *string    `json:"attachment_url"`
}

func NewPerformanceHandler(db *gorm.DB) *PerformanceHandler {
	return &PerformanceHandler{db: db}
}

func performanceStudentParam(c *gin.Context) string {
	if id := c.Param("id"); id != "" {
		return id
	}
	return c.Param("student_id")
}

func (h *PerformanceHandler) GetStudentPerformance(c *gin.Context) {
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

	var studyPlans []domain.StudyPlan
	var adminNotes []domain.AdminNote
	if err := h.db.Where("student_id = ?", profile.ID).Order("start_date desc").Find(&studyPlans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load study plans"})
		return
	}
	if err := h.db.Where("student_id = ?", profile.ID).Order("date desc").Find(&adminNotes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load admin notes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"study_plans": studyPlans,
		"admin_notes": adminNotes,
	})
}

func (h *PerformanceHandler) AdminCreateStudyPlan(c *gin.Context) {
	studentID := performanceStudentParam(c)
	adminID, ok := getAuthenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}

	var input CreateStudyPlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}
	if !isSafeAttachmentURL(input.Attachments) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid attachment url"})
		return
	}

	var student domain.StudentProfile
	if err := h.db.First(&student, "id = ?", studentID).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student not found"})
		return
	}

	startDate, jalaliStart, err := parseFlexibleDateInput(input.StartDate, input.JalaliStartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid start date"})
		return
	}
	endDate, jalaliEnd, err := parseFlexibleDateInput(input.EndDate, input.JalaliEndDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid end date"})
		return
	}
	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "end date cannot be before start date"})
		return
	}

	plan := domain.StudyPlan{
		StudentProfileID: studentID,
		AssignedBy:       adminID,
		Title:            strings.TrimSpace(input.Title),
		Description:      strings.TrimSpace(input.Description),
		StartDate:        startDate,
		EndDate:          endDate,
		JalaliStartDate:  jalaliStart,
		JalaliEndDate:    jalaliEnd,
		Attachments:      strings.TrimSpace(input.Attachments),
	}
	if plan.Title == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "title is required"})
		return
	}

	if err := h.db.Create(&plan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create study plan"})
		return
	}

	c.JSON(http.StatusCreated, plan)
}

func (h *PerformanceHandler) AdminUpdateStudyPlan(c *gin.Context) {
	id := c.Param("id")
	var input UpdateStudyPlanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}

	var plan domain.StudyPlan
	if err := h.db.First(&plan, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "study plan not found"})
		return
	}

	updates := map[string]interface{}{}
	if input.Title != nil {
		updates["title"] = strings.TrimSpace(*input.Title)
	}
	if input.Description != nil {
		updates["description"] = strings.TrimSpace(*input.Description)
	}
	if input.Attachments != nil {
		if !isSafeAttachmentURL(*input.Attachments) {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid attachment url"})
			return
		}
		updates["attachments"] = strings.TrimSpace(*input.Attachments)
	}
	if input.JalaliStartDate != nil || input.StartDate != nil {
		jalali := ""
		if input.JalaliStartDate != nil {
			jalali = *input.JalaliStartDate
		}
		date, normalized, err := parseFlexibleDateInput(input.StartDate, jalali)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid start date"})
			return
		}
		updates["start_date"] = date
		updates["jalali_start_date"] = normalized
	}
	if input.JalaliEndDate != nil || input.EndDate != nil {
		jalali := ""
		if input.JalaliEndDate != nil {
			jalali = *input.JalaliEndDate
		}
		date, normalized, err := parseFlexibleDateInput(input.EndDate, jalali)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid end date"})
			return
		}
		updates["end_date"] = date
		updates["jalali_end_date"] = normalized
	}

	if err := h.db.Model(&plan).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update study plan"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (h *PerformanceHandler) AdminDeleteStudyPlan(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&domain.StudyPlan{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to delete study plan"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *PerformanceHandler) AdminListStudentPerformance(c *gin.Context) {
	studentID := performanceStudentParam(c)

	var studyPlans []domain.StudyPlan
	var adminNotes []domain.AdminNote
	if err := h.db.Where("student_id = ?", studentID).Order("start_date desc").Find(&studyPlans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load study plans"})
		return
	}
	if err := h.db.Where("student_id = ?", studentID).Order("date desc").Find(&adminNotes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load admin notes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"study_plans": studyPlans,
		"admin_notes": adminNotes,
	})
}

func (h *PerformanceHandler) AdminCreateNote(c *gin.Context) {
	studentID := performanceStudentParam(c)
	adminID, ok := getAuthenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "missing user id"})
		return
	}

	var input CreateAdminNoteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}
	if !isSafeAttachmentURL(input.AttachmentURL) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid attachment url"})
		return
	}

	var student domain.StudentProfile
	if err := h.db.First(&student, "id = ?", studentID).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "student not found"})
		return
	}

	noteDate, jalaliDate, err := parseFlexibleDateInput(input.Date, input.JalaliDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid note date"})
		return
	}

	note := domain.AdminNote{
		StudentProfileID: studentID,
		AdminID:          adminID,
		Title:            strings.TrimSpace(input.Title),
		Body:             strings.TrimSpace(input.Body),
		NoteDate:         noteDate,
		JalaliDate:       jalaliDate,
		AttachmentURL:    strings.TrimSpace(input.AttachmentURL),
	}
	if note.Title == "" || note.Body == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "title and body are required"})
		return
	}

	if err := h.db.Create(&note).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create admin note"})
		return
	}

	c.JSON(http.StatusCreated, note)
}

func (h *PerformanceHandler) AdminUpdateNote(c *gin.Context) {
	id := c.Param("id")
	var input UpdateAdminNoteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid payload"})
		return
	}

	var note domain.AdminNote
	if err := h.db.First(&note, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "admin note not found"})
		return
	}

	updates := map[string]interface{}{}
	if input.Title != nil {
		updates["title"] = strings.TrimSpace(*input.Title)
	}
	if input.Body != nil {
		updates["body"] = strings.TrimSpace(*input.Body)
	}
	if input.AttachmentURL != nil {
		if !isSafeAttachmentURL(*input.AttachmentURL) {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid attachment url"})
			return
		}
		updates["attachment_url"] = strings.TrimSpace(*input.AttachmentURL)
	}
	if input.JalaliDate != nil || input.Date != nil {
		jalali := ""
		if input.JalaliDate != nil {
			jalali = *input.JalaliDate
		}
		date, normalized, err := parseFlexibleDateInput(input.Date, jalali)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid note date"})
			return
		}
		updates["date"] = date
		updates["jalali_date"] = normalized
	}

	if err := h.db.Model(&note).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update admin note"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "updated"})
}

func (h *PerformanceHandler) AdminDeleteNote(c *gin.Context) {
	id := c.Param("id")
	if err := h.db.Delete(&domain.AdminNote{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to delete admin note"})
		return
	}
	c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}
