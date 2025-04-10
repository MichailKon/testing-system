package main

import (
	"os"
	"testing_system/common"
	"testing_system/invoker"
	"testing_system/lib/logger"
)

func main() {
	configPath := os.Args[1]
	ts := common.InitTestingSystem(configPath)
	if ts.Config.Invoker != nil {
		err := invoker.SetupInvoker(ts)
		if err != nil {
			logger.Panic(err.Error())
		}
	} else {
		logger.Info("Invoker is not configured, skipping invoker component")
	}
	ts.Run()
}
