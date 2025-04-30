package common

import (
	"context"
	"fmt"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"os/signal"
	"runtime"
	"slices"
	"sync"
	"syscall"
	"testing_system/common/config"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/db"
	"testing_system/common/metrics"
	"testing_system/lib/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TestingSystem struct {
	Config  *config.Config
	Router  *gin.Engine
	DB      *gorm.DB
	Metrics *metrics.Collector

	MasterConn  *masterconn.Connector
	StorageConn *storageconn.Connector

	processes []func()
	defers    []func()

	StopCtx  context.Context
	stopFunc context.CancelFunc
	stopWG   sync.WaitGroup

	panics     []any
	panicsLock sync.Mutex
}

func InitTestingSystem(configPath string) *TestingSystem {
	ts := &TestingSystem{
		Config: config.ReadConfig(configPath),
	}
	logger.InitLogger(ts.Config)

	ts.InitServer()

	var err error
	ts.DB, err = db.NewDB(ts.Config.DB)
	if err != nil {
		logger.Panic("Can not set up db connection, error: %s", err.Error())
	}

	ts.MasterConn = masterconn.NewConnector(ts.Config.MasterConnection)
	ts.StorageConn = storageconn.NewConnector(ts.Config.StorageConnection)

	ts.Metrics = metrics.NewCollector()
	ts.Router.GET("/metrics", gin.WrapH(promhttp.Handler()))

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

	slices.Reverse(ts.defers)
	for _, d := range ts.defers {
		d()
	}

	ts.panicsLock.Lock()
	defer ts.panicsLock.Unlock()
	if len(ts.panics) > 0 {
		logger.Panic("Server running finished with panic: %v", ts.panics)
	}
}

func (ts *TestingSystem) Stop() {
	ts.stopFunc()
}

func (ts *TestingSystem) Go(f func()) {
	ts.stopWG.Add(1)
	go ts.runProcess(f)
}

func (ts *TestingSystem) runProcess(f func()) {
	defer func() {
		v := recover()
		if v != nil {
			stackTrace := ""
			for i := 1; ; i++ {
				_, file, line, ok := runtime.Caller(i)
				if !ok {
					break
				}
				stackTrace += fmt.Sprintf("%s:%d\n", file, line)
			}
			logger.Error("One process got panic: %v, stack trace:\n%s", v, stackTrace)
			logger.Error("Shutting down all processes gracefully")
			ts.stopFunc()

			ts.panicsLock.Lock()
			defer ts.panicsLock.Unlock()
			ts.panics = append(ts.panics, v)
		}
		ts.stopWG.Done()
	}()

	f()
}
