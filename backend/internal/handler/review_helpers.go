package handler

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"github.com/yourusername/noshirvani-academy/backend/pkg"
	"gorm.io/gorm"
)

var ErrInvalidDateInput = errors.New("either jalali date or gregorian date is required")

func getAuthenticatedUserID(c *gin.Context) (string, bool) {
	value, ok := c.Get("user_id")
	if !ok {
		return "", false
	}
	userID, ok := value.(string)
	if !ok || strings.TrimSpace(userID) == "" {
		return "", false
	}
	return userID, true
}

func loadStudentProfileByUserID(db *gorm.DB, userID string) (*domain.StudentProfile, error) {
	var profile domain.StudentProfile
	if err := db.Where("user_id = ?", userID).First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

func parseFlexibleDateInput(gregorian *time.Time, jalali string) (time.Time, string, error) {
	if strings.TrimSpace(jalali) != "" {
		t, err := pkg.JalaliToGregorian(jalali)
		if err != nil {
			return time.Time{}, "", fmt.Errorf("invalid jalali date")
		}
		return t.UTC(), strings.TrimSpace(jalali), nil
	}
	if gregorian != nil && !gregorian.IsZero() {
		t := gregorian.UTC()
		return t, pkg.GregorianToJalaliString(t), nil
	}
	return time.Time{}, "", ErrInvalidDateInput
}

func computeExamSubjectMetrics(subjects []domain.ExamSubject) ([]domain.ExamSubject, error) {
	computed := make([]domain.ExamSubject, len(subjects))
	for i, subject := range subjects {
		subject.SubjectName = strings.TrimSpace(subject.SubjectName)
		if subject.SubjectName == "" {
			return nil, fmt.Errorf("subject_name is required")
		}
		if subject.TotalQuestions <= 0 {
			return nil, fmt.Errorf("total_questions must be greater than zero")
		}
		if subject.Correct < 0 || subject.Wrong < 0 || subject.Blank < 0 {
			return nil, fmt.Errorf("correct, wrong, and blank must be non-negative")
		}
		answered := subject.Correct + subject.Wrong
		if answered > subject.TotalQuestions {
			return nil, fmt.Errorf("answered questions exceed total_questions")
		}
		blank := subject.TotalQuestions - answered
		if subject.Blank > 0 && subject.Blank != blank {
			return nil, fmt.Errorf("blank must match total_questions - correct - wrong")
		}
		subject.Answered = answered
		subject.Blank = blank
		subject.Percentage = (float64(subject.Correct) / float64(subject.TotalQuestions)) * 100
		computed[i] = subject
	}
	return computed, nil
}

func validateMistakeRelationships(db *gorm.DB, profileID string, examID, subjectID *string) error {
	if examID == nil && subjectID == nil {
		return nil
	}
	if examID != nil {
		var exam domain.Exam
		if err := db.Where("id = ? AND student_id = ?", *examID, profileID).First(&exam).Error; err != nil {
			return fmt.Errorf("exam not found for student")
		}
	}
	if subjectID != nil {
		query := db.Table("exam_subjects").Joins("JOIN exams ON exams.id = exam_subjects.exam_id").Where("exam_subjects.id = ? AND exams.student_id = ?", *subjectID, profileID)
		if examID != nil {
			query = query.Where("exam_subjects.exam_id = ?", *examID)
		}
		var count int64
		if err := query.Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return fmt.Errorf("exam subject not found for student")
		}
	}
	return nil
}

func isSafeAttachmentURL(url string) bool {
	url = strings.TrimSpace(url)
	if url == "" {
		return true
	}
	if !strings.HasPrefix(url, "/uploads/") {
		return false
	}
	ext := strings.ToLower(filepath.Ext(url))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".pdf", ".doc", ".docx", ".txt", ".xls", ".xlsx":
		return true
	default:
		return false
	}
}
