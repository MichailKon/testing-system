package registry

import (
	"context"
	"sync"
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/lib/logger"
	"time"
)

type JobType int

const (
	SendingJob JobType = iota // job is sending
	TestingJob                // job is testing
	NoReplyJob                // job is tested, but not veryfied
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

	status *invokerconn.StatusResponse
	failed bool
}

func newInvoker(connector *invokerconn.Connector, registry *InvokerRegistry, ts *common.TestingSystem) *Invoker {
	invoker := Invoker{
		connector:     connector,
		registry:      registry,
		jobTypeByID:   make(map[string]JobType),
		jobTypesCount: make(map[JobType]int),
	}

	var ctx context.Context
	ctx, invoker.cancel = context.WithCancel(ts.StopCtx)
	ts.Go(func() { invoker.PingLoop(ctx) })

	return &invoker
}

func (i *Invoker) ping() {
	status, err := i.connector.Status()

	i.mutex.Lock()
	defer i.mutex.Unlock()

	if err != nil {
		i.markFailed()
		return
	}

	i.updateStatus(status)
}

func (i *Invoker) PingLoop(ctx context.Context) {
	logger.Info("starting invoker ping loop")

	t := time.Tick(i.ts.Config.Master.InvokersPingInterval)

	for {
		i.ping()

		select {
		case <-ctx.Done():
			logger.Info("stopping invoker ping loop")
			return
		case <-t:
		}
	}
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
		i.markFailed()
		return
	}

	jobType := i.getJobType(job.ID)
	switch jobType {
	case SendingJob:
		i.setJobType(job.ID, TestingJob)
	case TestingJob, NoReplyJob:
		panic(logger.Error("SendingJob unexpectedly changed its status"))
	case UnknownJob:
		// job has been already tested
	}

	i.updateStatus(status)
}

func (i *Invoker) TrySendJob(job *invokerconn.Job) bool {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if i.failed || (i.status != nil && i.jobTypesCount[SendingJob]+1 > int(i.status.MaxNewJobs)) {
		return false
	}

	i.addJob(job.ID, SendingJob)
	i.ts.Go(func() { i.completeSendJob(job) })

	return true
}

func isJobTesting(jobID string, status *invokerconn.StatusResponse) bool {
	for _, id := range status.ActiveJobIDs {
		if jobID == id {
			return true
		}
	}
	return false
}

func (i *Invoker) ensureJobIsNotLost(jobID string) {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if i.getJobType(jobID) == NoReplyJob {
		i.markFailed()
	}
}

func (i *Invoker) updateStatus(status *invokerconn.StatusResponse) {
	if i.failed || (i.status != nil && i.status.Epoch >= status.Epoch) {
		return
	}

	i.status = status
	newNoReplyJobs := make([]string, 0)

	for jobID, jobType := range i.jobTypeByID {
		switch jobType {
		case SendingJob:
			// pass
		case TestingJob:
			if !isJobTesting(jobID, status) {
				newNoReplyJobs = append(newNoReplyJobs, jobID)
				i.setJobType(jobID, NoReplyJob)
			}
		case NoReplyJob:
			if isJobTesting(jobID, status) {
				logger.Info("job %s diappeared from status, but then reappeared", jobID)
				i.markFailed()
				return
			}
		case UnknownJob:
			panic(logger.Error("invoker shouldn't store job of type UnknownJob"))
		}
	}

	for _, jobID := range newNoReplyJobs {
		time.AfterFunc(i.ts.Config.Master.LostJobTimeout, func() { i.ensureJobIsNotLost(jobID) })
	}
}

func (i *Invoker) markFailed() {
	if i.failed {
		return
	}

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

	i.removeJob(jobID)
	return true
}
