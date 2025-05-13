package invoker

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/invoker/compiler"
	"testing_system/invoker/storage"
	"testing_system/lib/logger"
	"time"

	"github.com/google/uuid"
)

type Invoker struct {
	TS *common.TestingSystem

	Storage  *storage.InvokerStorage
	Compiler *compiler.Compiler

	SandboxThreads *threadsExecutor[*Job]
	RunnerThreads  *threadsExecutor[func()]

	ActiveJobs map[string]*Job
	MaxJobs    int
	Mutex      sync.Mutex

	Address     string
	TimeStarted time.Time
}

func SetupInvoker(ts *common.TestingSystem) error {
	if ts.Config.Invoker == nil {
		return fmt.Errorf("invoker is not configured")
	}
	invoker := &Invoker{
		TS:         ts,
		Storage:    storage.NewInvokerStorage(ts),
		Compiler:   compiler.NewCompiler(ts),
		ActiveJobs: make(map[string]*Job),

		TimeStarted: time.Now(),
	}
	invoker.setupAddress()

	invoker.MaxJobs = ts.Config.Invoker.QueueSize + ts.Config.Invoker.Sandboxes

	invoker.initializeSandboxThreads()
	invoker.initializeRunnerThreads()

	r := ts.Router.Group("/invoker")
	r.GET("/status", invoker.handleStatus)
	r.POST("/job/new", invoker.handleNewJob)
	r.POST("/reset_cache", invoker.resetCache)
	r.POST("/job/stop", invoker.stopJob)

	ts.AddProcess(invoker.runStatusLoop)

	logger.Info("Configured invoker")
	return nil
}

func (i *Invoker) setupAddress() {
	if i.TS.Config.Invoker.PublicAddress != nil {
		i.Address = *i.TS.Config.Invoker.PublicAddress
	} else if i.TS.Config.Invoker.AutodetectPublicAddress != nil && *i.TS.Config.Invoker.AutodetectPublicAddress {
		host, err := getPublicIP()
		if err != nil {
			logger.Panic("Can not detect public address for invoker, error: %v", err.Error())
		}
		i.Address = fmt.Sprintf("http://%s:%d", host, i.TS.Config.Port)
	} else {
		// TODO: Handle master and invoker on same server
		var host string
		if i.TS.Config.Host != nil {
			host = *i.TS.Config.Host
		} else {
			host = "localhost"
		}
		i.Address = "http://" + host + ":" + strconv.Itoa(i.TS.Config.Port)
	}
}

func getPublicIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}
	return "", errors.New("public ip address is not detected")
}

func (i *Invoker) getStatus() *invokerconn.Status {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()
	status := new(invokerconn.Status)
	status.Address = i.Address

	epochID, err := uuid.NewV7()
	if err != nil {
		logger.Panic("Can not generate status ID, error: %v", err.Error())
	}
	status.Epoch = epochID.String()
	for jobID := range i.ActiveJobs {
		status.ActiveJobIDs = append(status.ActiveJobIDs, jobID)
	}
	if len(status.ActiveJobIDs) > i.MaxJobs {
		status.MaxNewJobs = 0
	} else {
		status.MaxNewJobs = i.MaxJobs - len(status.ActiveJobIDs)
	}

	status.Metrics = &invokerconn.StatusMetrics{
		Lifetime:       time.Since(i.TimeStarted),
		SandboxMetrics: i.SandboxThreads.metrics(),
		ThreadMetrics:  i.SandboxThreads.metrics(),
	}

	return status
}

func (i *Invoker) runStatusLoop() {
	logger.Info("Starting master ping loop")

	t := time.Tick(i.TS.Config.Invoker.MasterPingInterval)
	for {
		err := i.TS.MasterConn.SendInvokerStatus(i.getStatus())
		if err != nil {
			logger.Warn("Can not send invoker status, error: %v", err.Error())
		}

		select {
		case <-i.TS.StopCtx.Done():
			logger.Info("Stopping master ping loop")
			return
		case <-t:
		}
	}
}
