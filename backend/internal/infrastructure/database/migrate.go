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
		&domain.Student{},
		&domain.Exam{},
		&domain.SubjectExam{},
		&domain.Mistake{},
		&domain.PerformanceHistory{},
		&domain.Notification{},
		&domain.DynamicFieldDefinition{},
		&domain.DynamicFieldValue{},
		&domain.BlogPost{},
	)
}
