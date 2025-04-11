package storage

import (
	"fmt"
	"testing_system/common"
	"testing_system/lib/logger"
	"testing_system/storage/filesystem"
)

type Storage struct {
	TS *common.TestingSystem

	filesystem filesystem.IFilesystem
}

func SetupStorage(ts *common.TestingSystem) error {
	if ts.Config.Storage == nil {
		return fmt.Errorf("storage is not configured")
	}

	r := ts.Router.Group("/storage/")

	storage := NewStorage(ts)

	r.POST("/upload", storage.HandleUpload)
	r.DELETE("/remove", storage.HandleRemove)
	r.GET("/get", storage.HandleGet)

	logger.Info("Configured invoker")
	return nil
}

func NewStorage(ts *common.TestingSystem) *Storage {
	return &Storage{TS: ts, filesystem: filesystem.NewFilesystem(ts.Config.Storage)}
}
