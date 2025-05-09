package main

import (
	"os"
	"testing_system/clients/admin"
	"testing_system/clients/common"
	"testing_system/clients/tsapi"
	"testing_system/lib/logger"
)

func main() {
	configPath := os.Args[1]
	clientBase := common.NewClientBase(configPath)

	if clientBase.Config.TestingSystemAPI != nil {
		err := tsapi.SetupHandler(clientBase)
		if err != nil {
			logger.Panic("Can not create testing system api handler, error: %v", err)
		}
	} else {
		logger.Info("Not creating testing system api handler, config is not specified")
	}

	if clientBase.Config.Admin {
		err := admin.SetupHandler(clientBase)
		if err != nil {
			logger.Panic("Can not create admin handler, error: %v", err)
		}
	} else {
		logger.Info("Not creating admin handler, config is not specified")
	}

	clientBase.Run()
}
