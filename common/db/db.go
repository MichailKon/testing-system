package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"testing_system/common/config"
	"testing_system/common/db/models"
	"testing_system/lib/logger"
)

func NewDB(config config.DBConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(config.Dsn), &gorm.Config{})
	if err != nil {
		return nil, logger.Error("Can't open database with dsn=\"%v\" because of %v:", config.Dsn, err)
	}
	if err = db.AutoMigrate(&models.Problem{}); err != nil {
		return nil, logger.Error("Can't migrate Problem: %v", err)
	}
	if err = db.AutoMigrate(&models.Submission{}); err != nil {
		return nil, logger.Error("Can't migrate Submission: %v", err)
	}
	return db, err
}
