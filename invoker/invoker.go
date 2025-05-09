package invoker

import (
	"context"
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

	RunnerStop   func()
	RunnerCtx    context.Context
	SandboxCount uint64

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
		TS:           ts,
		Storage:      storage.NewInvokerStorage(ts),
		Compiler:     compiler.NewCompiler(ts),
		RunQueue:     make(chan func(), ts.Config.Invoker.Threads),
		SandboxCount: ts.Config.Invoker.Sandboxes,
		ActiveJobs:   make(map[string]*Job),
	}
	invoker.setupAddress()

	invoker.MaxJobs = ts.Config.Invoker.QueueSize + ts.Config.Invoker.Sandboxes
	invoker.JobQueue = make(chan *Job, invoker.MaxJobs*2)
	invoker.RunnerCtx, invoker.RunnerStop = context.WithCancel(context.Background())

	for i := range ts.Config.Invoker.Sandboxes {
		sandbox := newSandbox(ts, i)
		ts.AddProcess(func() { invoker.runSandboxThread(sandbox, i) })
		ts.AddDefer(sandbox.Delete)
	}

	invoker.startRunners()

	r := ts.Router.Group("/invoker")
	r.GET("/status", invoker.HandleStatus)
	r.POST("/job/new", invoker.HandleNewJob)

	ts.AddProcess(invoker.runStatusLoop)

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
		i.Address = "http://" + host + ":" + strconv.Itoa(i.TS.Config.Port)
	}
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
