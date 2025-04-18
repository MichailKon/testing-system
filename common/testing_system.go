package common

import (
	"context"
	"net/http"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"testing_system/common/config"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/db"
	"testing_system/lib/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	_ "testing_system/swag"

	swaggo "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
)

type TestingSystem struct {
	Config *config.Config
	Router *gin.Engine
	DB     *gorm.DB

	MasterConn  *masterconn.Connector
	StorageConn *storageconn.Connector

	processes []func()
	defers    []func()

	StopCtx  context.Context
	stopFunc context.CancelFunc
	stopWG   sync.WaitGroup
}

func InitTestingSystem(configPath string) *TestingSystem {
	ts := &TestingSystem{
		Config: config.ReadConfig(configPath),
	}
	logger.InitLogger(ts.Config)

	ts.Router = gin.Default() // TODO: Add router options (e.g debug)
	ts.Router.GET("/swagger/*any", ginswagger.WrapHandler(swaggo.Handler))

	var err error
	ts.DB, err = db.NewDB(ts.Config.DB)
	if err != nil {
		logger.Panic("Can not set up db connection, error: %s", err.Error())
	}

	ts.MasterConn = masterconn.NewConnector(ts.Config.MasterConnection)
	ts.StorageConn = storageconn.NewConnector(ts.Config.StorageConnection)

	return ts
}

func (ts *TestingSystem) AddProcess(f func()) {
	ts.processes = append(ts.processes, f)
}

func (ts *TestingSystem) AddDefer(f func()) {
	ts.defers = append(ts.defers, f)
}

func (ts *TestingSystem) Run() {
	var ctx context.Context
	var cancel context.CancelFunc
	ctx, ts.stopFunc = context.WithCancel(context.Background())
	ts.StopCtx, cancel = signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	for _, process := range ts.processes {
		ts.Go(process)
	}

	ts.runServer()

	ts.stopWG.Wait()

	for _, d := range ts.defers {
		d()
	}
}

func (ts *TestingSystem) runServer() {
	addr := ":" + strconv.Itoa(ts.Config.Port)
	if ts.Config.Host != nil {
		addr = *ts.Config.Host + addr
	}
	logger.Info("Starting server at " + addr)
	server := http.Server{
		Addr:    addr,
		Handler: ts.Router,
	}
	go func() {
		<-ts.StopCtx.Done()
		logger.Info("Shutting down server")
		server.Shutdown(context.Background())
	}()
	server.ListenAndServe()
}

func (ts *TestingSystem) Go(f func()) {
	ts.stopWG.Add(1)
	go ts.runProcess(f)
}

func (ts *TestingSystem) runProcess(f func()) {
	defer func() {
		v := recover()
		if v != nil {
			logger.Error("One process got panic, shutting down all processes gracefully")
			ts.stopFunc()
		}
		ts.stopWG.Done()
	}()

	f()
}
