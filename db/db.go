package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log/slog"
	"testing_system/db/models"
)

func NewDb(config Config) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(config.Dsn), &gorm.Config{})
	if err != nil {
		slog.Error("Can't open database", "dsn", config.Dsn, "err", err)
		return nil, err
	}
	if err = db.AutoMigrate(&models.Problem{}); err != nil {
		slog.Error("Can't migrate Problem", "err", err)
		return nil, err
	}
	if err = db.AutoMigrate(&models.Submission{}); err != nil {
		slog.Error("Can't migrate Submission", "err", err)
		return nil, err
	}
	return db, err
}
