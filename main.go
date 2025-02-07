package main

import (
	"os"
	"testing_system/common"
	"testing_system/invoker"
)

func main() {
	configPath := os.Args[1]
	ts := common.InitTestingSystem(configPath)
	if ts.Config.Invoker != nil {
		invoker.SetupInvoker(ts)
	}
	ts.Run()
}
