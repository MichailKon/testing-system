package main

import (
	"os"
	"testing_system/common"
	"testing_system/invoker"
	"testing_system/storage"
)

func main() {
	configPath := os.Args[1]
	ts := common.InitTestingSystem(configPath)
	if ts.Config.Invoker != nil {
		invoker.SetupInvoker(ts)
		storage.SetupStorage(ts)
	}
	ts.Run()
}
