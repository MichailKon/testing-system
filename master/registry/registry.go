package registry

import (
	"sync"
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/lib/logger"
	"testing_system/master/queue"
	"time"
)

type InvokerRegistry struct {
	queue queue.IQueue
	ts    *common.TestingSystem
	mutex sync.Mutex

	invokers       []*Invoker
	invokerByJobID map[string]*Invoker

	jobRescheduleEpoch map[string]int
	rescheduleEpoch    int
}

func NewInvokerRegistry(ts *common.TestingSystem) *InvokerRegistry {
	return &InvokerRegistry{
		ts:                 ts,
		invokerByJobID:     make(map[string]*Invoker),
		jobRescheduleEpoch: make(map[string]int),
	}
}

func (r *InvokerRegistry) rescheduleJobs(jobsToReschedule []string) {
	r.mutex.Lock()
	for _, jobID := range jobsToReschedule {
		delete(r.invokerByJobID, jobID)
	}
	r.mutex.Unlock()

	for _, jobID := range jobsToReschedule {
		r.queue.RescheduleJob(jobID)
	}
}

func (r *InvokerRegistry) rescheduleJobIfNeeded(jobID string, epoch int) {
	r.mutex.Lock()

	curEpoch, exists := r.jobRescheduleEpoch[jobID]
	if !exists || curEpoch != epoch {
		r.mutex.Unlock()
		return
	}
	delete(r.jobRescheduleEpoch, jobID)

	invoker, exists := r.invokerByJobID[jobID]
	r.mutex.Unlock()

	if !exists {
		return
	}

	invoker.Mutex.Lock()
	jobType := invoker.GetJobType(jobID)
	if jobType != NoReplyJob {
		invoker.Mutex.Unlock()
		return
	}

	invoker.RemoveJob(jobID)
	invoker.Mutex.Unlock()

	r.queue.RescheduleJob(jobID)

	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.invokerByJobID, jobID)
}

func (r *InvokerRegistry) nextRescheduleEpoch() int {
	r.rescheduleEpoch++
	return r.rescheduleEpoch
}

func (r *InvokerRegistry) updateInvokerStatus(invoker *Invoker, status *invokerconn.StatusResponse, epoch int) {
	invoker.Mutex.Lock()
	jobsToReschedule, newNoReplyJobs := invoker.SetStatus(status, epoch)
	rescheduleEpoch := r.nextRescheduleEpoch()
	invoker.Mutex.Unlock()

	r.rescheduleJobs(jobsToReschedule)

	for _, jobID := range newNoReplyJobs {
		time.AfterFunc(r.ts.Config.Master.LostJobTimeout, func() { r.rescheduleJobIfNeeded(jobID, rescheduleEpoch) })
	}
}

func (r *InvokerRegistry) pingInvokerIfNeeded(invoker *Invoker) {
	invoker.Mutex.Lock()
	if !invoker.NeedToPing(r.ts.Config.Master.InvokersPingInterval / 2) {
		invoker.Mutex.Unlock()
		return
	}

	epoch := invoker.NextStatusEpoch()
	invoker.Mutex.Unlock()

	status, err := invoker.Connector.Status()
	if err != nil {
		r.handleInvokerFailure(invoker)
		return
	}

	r.updateInvokerStatus(invoker, status, epoch)
}

func (r *InvokerRegistry) RegisterNewInvoker(connector *invokerconn.Connector) {
	invoker := newInvoker(connector)

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.invokers = append(r.invokers, invoker)
	go r.pingInvokerIfNeeded(invoker)
}

func (r *InvokerRegistry) PingAll() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	for _, invoker := range r.invokers {
		go r.pingInvokerIfNeeded(invoker)
	}
}

func (r *InvokerRegistry) handleInvokerFailure(invoker *Invoker) {
	invoker.Mutex.Lock()
	jobsToReschedule := invoker.MarkFailed()
	invoker.Mutex.Unlock()

	r.rescheduleJobs(jobsToReschedule)
}

func (r *InvokerRegistry) completeSendingJobToInvoker(job *invokerconn.Job, epoch int, invoker *Invoker) {
	r.mutex.Lock()
	r.invokerByJobID[job.ID] = invoker
	r.mutex.Unlock()

	status, err := invoker.Connector.NewJob(job)

	if err != nil {
		logger.Info("failed to send job to invoker: %v", err)
		r.handleInvokerFailure(invoker)
		return
	}

	invoker.Mutex.Lock()

	jobType := invoker.GetJobType(job.ID)
	switch jobType {
	case SendingJob:
		invoker.SetJobType(job.ID, TestingJob)
	case TestingJob, NoReplyJob:
		logger.Error("SendingJob unexpectedly changed its status")
	case UnknownJob:
		// job has been already tested
	}

	invoker.Mutex.Unlock()
	r.updateInvokerStatus(invoker, status, epoch)
}

func (r *InvokerRegistry) trySendJobToInvoker(job *invokerconn.Job, invoker *Invoker) bool {
	invoker.Mutex.Lock()
	defer invoker.Mutex.Unlock()

	if !invoker.CanSendJob() {
		return false
	}

	invoker.SetJobType(job.ID, SendingJob)
	epoch := invoker.NextStatusEpoch()

	go r.completeSendingJobToInvoker(job, epoch, invoker)

	return true
}

func (r *InvokerRegistry) TrySendJob(job *invokerconn.Job) bool {
	r.mutex.Lock()
	invokersList := make([]*Invoker, len(r.invokers))
	copy(invokersList, r.invokers)
	r.mutex.Unlock()

	for _, invoker := range invokersList {
		if r.trySendJobToInvoker(job, invoker) {
			return true
		}
	}

	return false
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

	invoker.Mutex.Lock()
	defer invoker.Mutex.Unlock()

	jobType := invoker.GetJobType(jobID)
	switch jobType {
	case UnknownJob:
		return false
	case SendingJob, TestingJob, NoReplyJob:
		invoker.RemoveJob(jobID)
	}

	return true
}
