package master

import (
	"errors"
	"sync"
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
	sendJobsMutex   sync.Mutex
}

func SetupMaster(ts *common.TestingSystem) error {
	if ts.Config.Master == nil {
		return errors.New("master is not configured")
	}

	master := Master{
		ts:              ts,
		queue:           queue.NewQueue(ts),
		invokerRegistry: registry.NewInvokerRegistry(ts),
	}

	ts.AddProcess(master.pingInvokersLoop)
	ts.AddProcess(master.sendingJobsLoop)

	// TODO: register invoker connectors
	// TODO: register http handlers

	return nil
}

func (m *Master) sendJobs() {
	// Trying to lock mutex since it's useless to run it concurrently.
	if !m.sendJobsMutex.TryLock() {
		return
	}
	// Between the end of this function and this Unlock, a new job might have come in and not run due to the previous TryLock().
	// In that case, the job will be scheduled during the next m.sendJobs call, which will happen in at most m.ts.Config.Master.SendJobInterval.
	// I think it's okay, since if this case does happen (which is already very unlikely), the system is probably under load,
	// and the next m.sendJobs call will likely happen even earlier than m.ts.Config.Master.SendJobInterval.

	// Maybe it's okay to call this function without any mutexes cause it should finish pretty fast.
	// Opinion?
	defer m.sendJobsMutex.Unlock()

	for {
		select {
		case <-m.ts.StopCtx.Done():
			return
		default:
		}

		job := m.queue.NextJob()
		if job == nil {
			return
		}
		if !m.invokerRegistry.TrySendJob(job) {
			return
		}
	}
}

func (m *Master) pingInvokersLoop() {
	logger.Info("starting ping invokers loop")

	t := time.Tick(m.ts.Config.Master.InvokersPingInterval)

	for {
		select {
		case <-m.ts.StopCtx.Done():
			logger.Info("stopping ping invokers loop")
			return
		case <-t:
			m.invokerRegistry.PingAll()
		}
	}
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
			go m.sendJobs()
		}
	}
}
