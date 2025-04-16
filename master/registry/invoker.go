package registry

import (
	"context"
	"slices"
	"sync"
	"testing_system/common"
	"testing_system/common/config"
	"testing_system/common/connectors/invokerconn"
	"testing_system/lib/logger"
	"time"
)

type JobType int

const (
	SendingJob JobType = iota // job is sending
	TestingJob                // job is testing
	NoReplyJob                // job is tested, but not verified
	UnknownJob                // no information about the job
)

type Invoker struct {
	ts        *common.TestingSystem
	connector *invokerconn.Connector
	registry  *InvokerRegistry
	cancel    context.CancelFunc

	mutex sync.Mutex

	jobTypeByID   map[string]JobType
	jobTypesCount map[JobType]int

	status *invokerconn.Status
	failed bool
}

func newInvoker(status *invokerconn.Status, registry *InvokerRegistry, ts *common.TestingSystem) *Invoker {
	logger.Info("registering new invoker: %s", status.Address)

	invoker := Invoker{
		connector:     invokerconn.NewConnector(&config.Connection{Address: status.Address}),
		registry:      registry,
		jobTypeByID:   make(map[string]JobType),
		jobTypesCount: make(map[JobType]int),
		status:        status,
	}

	var ctx context.Context
	ctx, invoker.cancel = context.WithCancel(ts.StopCtx)
	ts.Go(func() { invoker.pingLoop(ctx) })

	return &invoker
}

func (i *Invoker) ping() {
	status, err := i.connector.Status()

	i.mutex.Lock()
	defer i.mutex.Unlock()

	if err != nil {
		logger.Info("invoker %s did not response on ping, error: %s", i.status.Address, err.Error())
		i.markFailed()
		return
	}

	i.updateStatus(status)
}

func (i *Invoker) pingLoop(ctx context.Context) {
	logger.Info("starting invoker ping loop")

	t := time.Tick(i.ts.Config.Master.InvokersPingInterval)

	for {
		select {
		case <-ctx.Done():
			logger.Info("stopping invoker ping loop")
			return
		case <-t:
			i.ping()
		}
	}
}

func (i *Invoker) address() string {
	return i.status.Address
}

func (i *Invoker) getJobType(jobID string) JobType {
	jobType, ok := i.jobTypeByID[jobID]
	if !ok {
		return UnknownJob
	}
	return jobType
}

func (i *Invoker) addJob(jobID string, jobType JobType) {
	i.jobTypeByID[jobID] = jobType
	i.jobTypesCount[jobType]++
}

func (i *Invoker) setJobType(jobID string, jobType JobType) {
	i.jobTypesCount[i.getJobType(jobID)]--
	i.jobTypeByID[jobID] = jobType
	i.jobTypesCount[jobType]++
}

func (i *Invoker) removeJob(jobID string) {
	i.jobTypesCount[i.getJobType(jobID)]--
	delete(i.jobTypeByID, jobID)
}

func (i *Invoker) completeSendJob(job *invokerconn.Job) {
	status, err := i.connector.NewJob(job)

	i.mutex.Lock()
	defer i.mutex.Unlock()

	if err != nil {
		logger.Info("failed to send job to invoker %s, error: %s", i.address(), err.Error())
		i.markFailed()
		return
	}

	jobType := i.getJobType(job.ID)
	switch jobType {
	case SendingJob:
		i.setJobType(job.ID, TestingJob)
	case TestingJob, NoReplyJob:
		logger.Panic("SendingJob unexpectedly changed its status")
	case UnknownJob:
		// job has been already tested
	}

	i.updateStatus(status)
}

func (i *Invoker) TrySendJob(job *invokerconn.Job) bool {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if i.failed || i.jobTypesCount[SendingJob]+1 > int(i.status.MaxNewJobs) {
		return false
	}

	logger.Trace("sending job %s to invoker %s", job.ID, i.address())
	i.addJob(job.ID, SendingJob)
	i.ts.Go(func() { i.completeSendJob(job) })

	return true
}

func isJobTesting(jobID string, status *invokerconn.Status) bool {
	return slices.Contains(status.ActiveJobIDs, jobID)
}

func (i *Invoker) ensureJobIsNotLost(jobID string) {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if i.getJobType(jobID) == NoReplyJob {
		logger.Info("invoker %s has lost job %s", i.address(), jobID)
		i.markFailed()
	}
}

func (i *Invoker) updateStatus(status *invokerconn.Status) {
	if i.failed || i.status.Epoch >= status.Epoch {
		return
	}

	i.status = status

	for jobID, jobType := range i.jobTypeByID {
		switch jobType {
		case SendingJob:
			// pass
		case TestingJob:
			if !isJobTesting(jobID, status) {
				logger.Trace("job %s disappeared from the status of invoker %s", jobID, i.address())
				i.setJobType(jobID, NoReplyJob)
				time.AfterFunc(i.ts.Config.Master.LostJobTimeout, func() { i.ensureJobIsNotLost(jobID) })
			}
		case NoReplyJob:
			if isJobTesting(jobID, status) {
				logger.Info("job %s disappeared from status, but then reappeared, invoker: %s", jobID, i.address())
				i.markFailed()
				return
			}
		case UnknownJob:
			logger.Panic("invoker shouldn't store job of type UnknownJob")
		}
	}
}

func (i *Invoker) markFailed() {
	if i.failed {
		return
	}

	logger.Info("marking invoker %s as failed", i.address())
	i.failed = true
	i.cancel()
	i.ts.Go(func() { i.registry.OnInvokerFailure(i) })
}

func (i *Invoker) ExtractJobs() []string {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	jobs := make([]string, 0, len(i.jobTypeByID))
	for k := range i.jobTypeByID {
		jobs = append(jobs, k)
	}

	i.jobTypeByID = make(map[string]JobType)
	i.jobTypesCount = make(map[JobType]int)

	return jobs
}

func (i *Invoker) JobTested(jobID string) bool {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	jobType := i.getJobType(jobID)
	if jobType == UnknownJob {
		return false
	}

	logger.Trace("job %s successfully tested on invoker %s", jobID, i.address())
	i.removeJob(jobID)
	return true
}

func (i *Invoker) VerifyAndUpdateStatus(status *invokerconn.Status) bool {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if i.address() != status.Address {
		return false
	}

	i.updateStatus(status)
	return !i.failed
}
