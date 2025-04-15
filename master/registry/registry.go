package registry

import (
	"slices"
	"sync"
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/lib/logger"
	"testing_system/master/queue"
)

type InvokerRegistry struct {
	ts *common.TestingSystem

	queue queue.IQueue
	mutex sync.RWMutex

	invokers       []*Invoker
	invokerByJobID map[string]*Invoker

	nextJob *invokerconn.Job
}

func NewInvokerRegistry(queue queue.IQueue, ts *common.TestingSystem) *InvokerRegistry {
	return &InvokerRegistry{
		ts:             ts,
		queue:          queue,
		invokerByJobID: make(map[string]*Invoker),
	}
}

func (r *InvokerRegistry) OnInvokerFailure(invoker *Invoker) {
	jobsToReschedule := invoker.ExtractJobs()

	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, jobID := range jobsToReschedule {
		_, exists := r.invokerByJobID[jobID]
		if exists {
			r.queue.RescheduleJob(jobID)
			delete(r.invokerByJobID, jobID)
		}
	}

	r.invokers = slices.DeleteFunc(r.invokers, func(i *Invoker) bool { return i == invoker })
}

func (r *InvokerRegistry) UpsertInvoker(status *invokerconn.Status) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, invoker := range r.invokers {
		if invoker.VerifyAndUpdateStatus(status) {
			return
		}
	}

	invoker := newInvoker(status, r, r.ts)
	r.invokers = append(r.invokers, invoker)
}

func (r *InvokerRegistry) HandleInvokerJobResult(result *masterconn.InvokerJobResult) bool {
	defer r.UpsertInvoker(result.InvokerStatus)

	r.mutex.Lock()
	defer r.mutex.Unlock()

	invoker, exists := r.invokerByJobID[result.JobID]

	if !exists {
		logger.Trace("old or unknown job %s is tested", result.JobID)
		return false
	}

	delete(r.invokerByJobID, result.JobID)
	return invoker.JobTested(result.JobID)
}

func (r *InvokerRegistry) SendJobs() {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, invoker := range r.invokers {
		for {
			if r.nextJob == nil {
				r.nextJob = r.queue.NextJob()
			}
			if r.nextJob == nil {
				return
			}

			if !invoker.TrySendJob(r.nextJob) {
				break
			}
			r.nextJob = nil
		}
	}
}
