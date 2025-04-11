package invoker

import (
	"fmt"
	"sync"
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/invoker/compiler"
	"testing_system/invoker/storage"
	"testing_system/lib/logger"
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

	// TODO Add master initial connection and invoker keepalive thread

	logger.Info("Configured invoker")
	return nil
}

func (i *Invoker) getStatus() *invokerconn.StatusResponse {
	i.Mutex.Lock()
	defer i.Mutex.Unlock()
	status := new(invokerconn.StatusResponse)
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
