package database

import (
	"github.com/yourusername/noshirvani-academy/backend/internal/domain"
	"gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) error {
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto;").Error; err != nil {
		return err
	}
	return db.AutoMigrate(
		&domain.User{},
		&domain.StudentProfile{},
		&domain.Exam{},
		&domain.ExamSubject{},
		&domain.MistakeAnalysis{},
		&domain.StudyPlan{},
		&domain.AdminNote{},
		&domain.DynamicFieldDefinition{},
		&domain.DynamicFieldValue{},
		&domain.BlogPost{},
	)
}
