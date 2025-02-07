package invoker

import (
	"testing_system/common"
	"testing_system/common/db/models"
	"testing_system/invoker/storage"
	"testing_system/invoker/tester"
	"testing_system/lib/logger"
)

type Invoker struct {
	TS *common.TestingSystem

	Storage *storage.InvokerStorage
	Testers []*tester.Tester

	Queue        chan *models.Submission // This will be changed in later commits
	MaxQueueSize int
}

func SetupInvoker(ts *common.TestingSystem) {
	if ts.Config.Invoker == nil {
		logger.Info("Invoker is not configured, skipping invoker start")
		return
	}
	invoker := &Invoker{
		TS:           ts,
		MaxQueueSize: ts.Config.Invoker.Instances * *ts.Config.Invoker.QueueSize,
	}
	invoker.Queue = make(chan *models.Submission, invoker.MaxQueueSize)
	invoker.Storage = storage.NewInvokerStorage(ts)

	for i := range ts.Config.Invoker.Instances {
		invoker.Testers = append(invoker.Testers, tester.NewTester(ts, i, invoker.Storage))
		testerID := i
		ts.AddProcess(func() { invoker.RunTesterThread(testerID) })
		ts.AddDefer(invoker.Testers[i].Cleanup)
	}

	r := ts.Router.Group("/invoker/")
	r.GET("/ping", invoker.HandlePing)
	r.POST("/test", invoker.HandleTest)
	// TODO
}

func (i *Invoker) RunTesterThread(id int) {
	t := i.Testers[id]
	defer t.Cleanup()
	logger.Info("Starting invoker %d", id)
	for {
		select {
		case <-i.TS.StopCtx.Done():
			logger.Info("Stopped invoker %d", id)
			break
		case s := <-i.Queue:
			t.Test(s)
		}
	}
}
