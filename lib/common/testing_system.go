package common

import (
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
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

// NewMasterRequest creates new request to master with authentication and address set up.
// master address will be taken from config (it is either to same process, or to the other microservice)
func (ts *TestingSystem) NewMasterRequest() *resty.Request {
	// TODO: Add request creation
}

// NewStorageRequest creates new request to storage with authentication and address set up.
// master address will be taken from config (it is either to same process, or to the other microservice)
func (ts *TestingSystem) NewStorageRequest() *resty.Request {
	// TODO: Add request creation
}

// NewInvokerRequest creates new request to invoker with authentication set up.
// params may be changed depending on how invokers are stored, probably address will be set up by caller
func (ts *TestingSystem) NewInvokerRequest(invokerID string) *resty.Request {
	// TODO: Add request creation
}
