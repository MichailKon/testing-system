package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing_system/common/config"
	"testing_system/common/db/models"
	"testing_system/lib/logger"
)

func NewDB(config config.DBConfig) (*gorm.DB, error) {
	var db *gorm.DB
	var err error
	if config.InMemory {
		db, err = newInMemoryDB()
	} else {
		db, err = newPostgresDB(config)
	}
	if err != nil {
		return nil, err
	}

	if err = db.AutoMigrate(&models.Problem{}); err != nil {
		return nil, logger.Error("Can't migrate Problem: %v", err)
	}
	if err = db.AutoMigrate(&models.Submission{}); err != nil {
		return nil, logger.Error("Can't migrate Submission: %v", err)
	}
	logger.Info("Configured DB successfully")
	return db, err
}

func newPostgresDB(config config.DBConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(config.Dsn), &gorm.Config{})
	if err != nil {
		return nil, logger.Error("Can't open database with dsn=\"%v\" because of %v:", config.Dsn, err)
	}
	return db, nil
}

func newInMemoryDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open("file:ts?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	logger.Warn("InMemory DB should not be used in production, consider using postgres db instead")
	return db, nil
}
