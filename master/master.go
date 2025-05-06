//go:generate go run github.com/swaggo/swag/cmd/swag@latest init -g master.go --parseDependency -o ../swag

package master

import (
	"errors"
	"testing_system/common"
	"testing_system/lib/logger"
	"testing_system/master/queue"
	"testing_system/master/registry"
	"time"
)

type Master struct {
	ts              *common.TestingSystem
	queue           queue.IQueue
	invokerRegistry *registry.InvokerRegistry
}

func SetupMaster(ts *common.TestingSystem) error {
	if ts.Config.Master == nil {
		return errors.New("master is not configured")
	}

	queue := queue.NewQueue(ts)
	master := Master{
		ts:              ts,
		queue:           queue,
		invokerRegistry: registry.NewInvokerRegistry(queue, ts),
	}

	ts.AddProcess(master.sendingJobsLoop)

	router := ts.Router.Group("/master")

	// invoker handlers
	r := router.Group("/invoker")
	r.POST("/job-result", master.handleInvokerJobResult)
	r.POST("/status", master.handleInvokerStatus)

	// client handlers
	router.POST("/submit", master.handleNewSubmission)
	router.GET("/status", master.handleStatus)

	return nil
}

func (m *Master) sendingJobsLoop() {
	logger.Info("starting jobs sending loop")

	t := time.Tick(m.ts.Config.Master.SendJobInterval)

	for {
		select {
		case <-m.ts.StopCtx.Done():
			logger.Info("stopping jobs sending loop")
			return
		case <-t:
			m.invokerRegistry.SendJobs()
		}
	}
}
