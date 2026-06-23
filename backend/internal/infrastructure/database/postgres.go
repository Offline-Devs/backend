package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewPostgresDB(dsn, environment string) (*gorm.DB, error) {
	logLevel := logger.Warn
	switch environment {
	case "development":
		logLevel = logger.Info
	case "test":
		logLevel = logger.Silent
	case "production":
		logLevel = logger.Error
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return nil, err
	}
	return db, nil
}
