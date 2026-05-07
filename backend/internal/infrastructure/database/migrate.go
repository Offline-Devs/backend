package database

import (
    "github.com/yourusername/noshirvani-academy/backend/internal/domain"
    "gorm.io/gorm"
)

func AutoMigrate(db *gorm.DB) {
    _ = db.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto;")
    _ = db.AutoMigrate(
        &domain.User{},
        &domain.Student{},
        &domain.Exam{},
        &domain.SubjectExam{},
        &domain.Mistake{},
        &domain.PerformanceHistory{},
        &domain.DynamicFieldDefinition{},
        &domain.DynamicFieldValue{},
        &domain.BlogPost{},
    )
}
