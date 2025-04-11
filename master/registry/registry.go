package registry

import (
	"slices"
	"sync"
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/lib/logger"
	"testing_system/master/queue"
)

type InvokerRegistry struct {
	ts *common.TestingSystem

	queue queue.IQueue
	mutex sync.Mutex

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

func (r *InvokerRegistry) RegisterNewInvoker(connector *invokerconn.Connector) {
	invoker := newInvoker(connector, r, r.ts)

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.invokers = append(r.invokers, invoker)
}

func (r *InvokerRegistry) JobTested(jobID string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	invoker, exists := r.invokerByJobID[jobID]

	if !exists {
		logger.Info("old or unknown job %s is tested", jobID)
		return false
	}

	delete(r.invokerByJobID, jobID)
	return invoker.JobTested(jobID)
}

func (r *InvokerRegistry) trySendJob(job *invokerconn.Job) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, invoker := range r.invokers {
		if invoker.TrySendJob(job) {
			return true
		}
	}

	return false
}

func (r *InvokerRegistry) SendJobs() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

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
