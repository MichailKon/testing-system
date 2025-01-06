package common

import (
	"github.com/gin-gonic/gin"
	"strconv"
	"testing_system/lib/config"
	"testing_system/lib/logger"
)

type TestingSystem struct {
	Config *config.Config
	Router *gin.Engine
	// TODO: Add common params for all testing system components
}

func InitTestingSystem(configPath string) *TestingSystem {
	ts := &TestingSystem{
		Config: config.ReadConfig(configPath),
	}
	logger.InitLogger(ts.Config)
	ts.Router = gin.Default() // TODO: Add router options (e.g debug)

	// TODO: Add all TestingSystem common components initialization

	return ts
}

func (ts *TestingSystem) Run() {
	addr := ":" + strconv.Itoa(ts.Config.Port)
	if ts.Config.Host != nil {
		addr = *ts.Config.Host + addr
	}
	logger.Info("Starting server at " + addr)
	err := ts.Router.Run(addr)
	if err != nil {
		logger.Panic(err.Error())
	}
}
