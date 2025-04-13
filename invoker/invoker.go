package invoker

import (
	"fmt"
	"github.com/google/uuid"
	"sync"
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/invoker/compiler"
	"testing_system/invoker/storage"
	"testing_system/lib/logger"
	"time"
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

	Epoch string
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

func (i *Invoker) getStatus() *invokerconn.StatusResponse {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()
	status := new(invokerconn.StatusResponse)

	// V6 uid is slower than v7, but v7 is compared within milliseconds,
	// and v6 is compared by seconds and clock sequence, which will give us better ordering if milliseconds are same
	epochID, err := uuid.NewV6()
	if err != nil {
		logger.Panic("Can not status ID, error: %v", err.Error())
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
