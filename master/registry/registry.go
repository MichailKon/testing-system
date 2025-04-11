package registry

import (
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

	for i, curInvoker := range r.invokers {
		if curInvoker == invoker {
			r.invokers[i] = nil
			r.invokers = append(r.invokers[:i], r.invokers[i+1:]...)
			break
		}
	}
}

func (r *InvokerRegistry) RegisterNewInvoker(connector *invokerconn.Connector) {
	invoker := newInvoker(connector, r, r.ts)

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.invokers = append(r.invokers, invoker)
}

func (r *InvokerRegistry) JobTested(jobID string) bool {
	r.mutex.Lock()
	invoker, exists := r.invokerByJobID[jobID]

	if !exists {
		logger.Info("old or unknown job %s is tested", jobID)
		r.mutex.Unlock()
		return false
	}

	delete(r.invokerByJobID, jobID)
	r.mutex.Unlock()

	return invoker.JobTested(jobID)
}

func (r *InvokerRegistry) trySendJob(job *invokerconn.Job) bool {
	r.mutex.Lock()
	invokersList := make([]*Invoker, len(r.invokers))
	copy(invokersList, r.invokers)
	r.mutex.Unlock()

	for _, invoker := range invokersList {
		if invoker.TrySendJob(job) {
			return true
		}
	}

	return false
}

func (r *InvokerRegistry) SendJobs() {
	for {
		job := r.queue.NextJob()
		if job == nil {
			return
		}
		if !r.trySendJob(job) {
			return
		}
	}
}
