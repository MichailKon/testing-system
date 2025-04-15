package invoker

import (
	"fmt"
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

	JobQueue chan *Job
	RunQueue chan func()

	ActiveJobs map[string]*Job
	MaxJobs    uint64
	Mutex      sync.Mutex

	Address string
}

func SetupInvoker(ts *common.TestingSystem) error {
	if ts.Config.Invoker == nil {
		return fmt.Errorf("invoker is not configured")
	}
	invoker := &Invoker{
		TS:         ts,
		Storage:    storage.NewInvokerStorage(ts),
		Compiler:   compiler.NewCompiler(ts),
		RunQueue:   make(chan func(), ts.Config.Invoker.Threads),
		ActiveJobs: make(map[string]*Job),
	}
	invoker.setupAddress()

	invoker.MaxJobs = ts.Config.Invoker.QueueSize + ts.Config.Invoker.Sandboxes
	invoker.JobQueue = make(chan *Job, invoker.MaxJobs*2)

	for i := range ts.Config.Invoker.Sandboxes {
		jobExecutor := NewJobExecutor(ts, i)
		ts.AddProcess(func() { invoker.RunJobExecutorThread(jobExecutor) })
		ts.AddDefer(jobExecutor.Delete)
	}

	invoker.StartRunners()

	r := ts.Router.Group("/invoker/")
	r.GET("/status", invoker.HandleStatus)
	r.POST("/job/new", invoker.HandleNewJob)

	ts.AddProcess(invoker.runStatusLoop)

	// TODO Add master initial connection and invoker keepalive thread

	logger.Info("Configured invoker")
	return nil
}

func (i *Invoker) setupAddress() {
	if i.TS.Config.Invoker.PublicAddress != nil {
		i.Address = *i.TS.Config.Invoker.PublicAddress
	} else {
		// TODO: Handle master and invoker on same server
		var host string
		if i.TS.Config.Host != nil {
			host = *i.TS.Config.Host
		} else {
			host = "localhost"
		}
		i.Address = host + ":" + strconv.Itoa(i.TS.Config.Port)
	}
}

func (i *Invoker) getStatus() *invokerconn.Status {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()
	status := new(invokerconn.Status)
	status.Address = i.Address

	// V6 uid is slower than v7, but gives us better ordering
	epochID, err := uuid.NewV6()
	if err != nil {
		logger.Panic("Can not generate status ID, error: %v", err.Error())
	}
	status.Epoch = epochID.String()
	for jobID := range i.ActiveJobs {
		status.ActiveJobIDs = append(status.ActiveJobIDs, jobID)
	}
	if uint64(len(status.ActiveJobIDs)) > i.MaxJobs {
		status.MaxNewJobs = 0
	} else {
		status.MaxNewJobs = i.MaxJobs - uint64(len(status.ActiveJobIDs))
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
