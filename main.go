package main

import (
	"os"
	"testing_system/common"
	"testing_system/invoker"
	"testing_system/lib/logger"
	"testing_system/master"
	"testing_system/storage"
)

func main() {
	configPath := os.Args[1]
	ts := common.InitTestingSystem(configPath)

	if ts.Config.Invoker != nil {
		err := invoker.SetupInvoker(ts)
		if err != nil {
			logger.Panic("Can not setup invoker, error: %v", err.Error())
		}
	} else {
		logger.Info("Invoker is not configured, skipping invoker component")
	}

	if ts.Config.Master != nil {
		err := master.SetupMaster(ts)
		if err != nil {
			logger.Panic("Can not setup master, error: %v", err.Error())
		}
	} else {
		logger.Info("master is not configured, skipping master component")
	}

	if ts.Config.Storage != nil {
		err := storage.SetupStorage(ts)
		if err != nil {
			logger.Panic("Can not setup storage, error: %v", err.Error())
		}
	} else {
		logger.Info("storage is not configured, skipping storage component")
	}

	ts.Run()
}
