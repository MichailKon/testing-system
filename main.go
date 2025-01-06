package main

import (
	"os"
	"testing_system/lib/common"
)

func main() {
	configPath := os.Args[1]
	ts := common.InitTestingSystem(configPath)
	// TODO: Add all components initialisation
	ts.Run()
}
