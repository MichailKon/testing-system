package registry

import (
	"fmt"
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

	invoker, exists := r.invokerByJobID[result.Job.ID]

	if !exists {
		logger.Trace("old or unknown job %s is tested", result.Job.ID)
		return false
	}

	delete(r.invokerByJobID, result.Job.ID)
	return invoker.JobTested(result.Job.ID)
}

func (r *InvokerRegistry) SendJobs() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	logger.Trace("Sending new jobs from master to invoker")

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
			r.invokerByJobID[r.nextJob.ID] = invoker
			r.nextJob = nil
		}
	}
}

func (r *InvokerRegistry) Status() []*masterconn.InvokerStatus {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	status := make([]*masterconn.InvokerStatus, 0, len(r.invokers))
	for _, invoker := range r.invokers {
		status = append(status, invoker.StatusForMaster())
	}
	return status
}

func (r *InvokerRegistry) invokersAction(f func(i *Invoker) error) error {
	r.mutex.Lock()
	invokersCount := len(r.invokers)
	receiveChan := make(chan error, invokersCount)
	for _, invoker := range r.invokers {
		r.ts.Go(func() {
			err := f(invoker)
			if err != nil {
				err = fmt.Errorf("invoker %s error: %v", invoker.ID(), err)
			}
			receiveChan <- err
		})
	}
	r.mutex.Unlock()
	for range invokersCount {
		err := <-receiveChan
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *InvokerRegistry) ResetCache() error {
	return r.invokersAction(func(i *Invoker) error {
		return i.ResetCache()
	})
}
