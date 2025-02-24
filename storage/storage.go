package storage

import (
	"testing_system/common"
	"testing_system/common/config"
	"testing_system/lib/logger"
	"testing_system/storage/filesystem"
)

type Storage struct {
	TS *common.TestingSystem

	filesystem filesystem.IFilesystem
}

func SetupStorage(ts *common.TestingSystem) {
	if ts.Config.Storage == nil {
		logger.Info("Storage is not configured, skipping storage start")
		return
	}
	config.FillInStorageConfig(ts.Config.Storage)

	r := ts.Router.Group("/storage/")

	storage := &Storage{TS: ts, filesystem: filesystem.CreateFilesystem(ts.Config.Storage.StoragePath)}

	r.POST("/upload", storage.HandleUpload)
	r.DELETE("/remove", storage.HandleRemove)
	r.GET("/get", storage.HandleGet)
}
