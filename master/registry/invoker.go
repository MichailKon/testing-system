package registry

import (
	"context"
	"slices"
	"sync"
	"testing_system/common"
	"testing_system/common/config"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
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

	jobHolderByID map[string]*jobHolder
	jobTypesCount map[JobType]int

	status    *invokerconn.Status
	failed    bool
	timeAdded time.Time
}

type jobHolder struct {
	job     *invokerconn.Job
	jobType JobType
}

func newInvoker(status *invokerconn.Status, registry *InvokerRegistry, ts *common.TestingSystem) *Invoker {
	logger.Info("registering new invoker: %s", status.Address)

	invoker := Invoker{
		ts:            ts,
		connector:     invokerconn.NewConnector(&config.Connection{Address: status.Address}),
		registry:      registry,
		jobHolderByID: make(map[string]*jobHolder),
		jobTypesCount: make(map[JobType]int),
		status:        status,
		timeAdded:     time.Now(),
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
		logger.Warn("invoker %s did not response on ping, error: %s", i.status.Address, err.Error())
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
	holder, ok := i.jobHolderByID[jobID]
	if !ok {
		return UnknownJob
	}
	return holder.jobType
}

func (i *Invoker) addJob(job *invokerconn.Job, jobType JobType) {
	i.jobHolderByID[job.ID] = &jobHolder{
		job:     job,
		jobType: jobType,
	}
	i.jobTypesCount[jobType]++
}

func (i *Invoker) setJobType(jobID string, jobType JobType) {
	holder, ok := i.jobHolderByID[jobID]
	if !ok {
		logger.Panic("Changing job type in invoker for job %s which was not sent to invoker", jobID)
	}
	i.jobTypesCount[holder.jobType]--
	holder.jobType = jobType
	i.jobTypesCount[jobType]++
}

func (i *Invoker) removeJob(jobID string) {
	holder, ok := i.jobHolderByID[jobID]
	if !ok {
		logger.Panic("Removing job %s in invoker which was not sent to invoker", jobID)
	}
	i.jobTypesCount[holder.jobType]--
	delete(i.jobHolderByID, jobID)
}

func (i *Invoker) completeSendJob(job *invokerconn.Job) {
	status, err := i.connector.NewJob(job)

	i.mutex.Lock()
	defer i.mutex.Unlock()

	if err != nil {
		logger.Warn("failed to send job to invoker %s, error: %s", i.address(), err.Error())
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

	logger.Trace("job %s is sended to invoker %s", job.ID, i.address())
}

func (i *Invoker) TrySendJob(job *invokerconn.Job) bool {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	if i.failed || i.jobTypesCount[SendingJob]+1 > int(i.status.MaxNewJobs) {
		return false
	}

	logger.Trace("sending job %s to invoker %s", job.ID, i.address())
	i.addJob(job, SendingJob)
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
		logger.Warn("invoker %s has lost job %s", i.address(), jobID)
		i.markFailed()
	}
}

func (i *Invoker) updateStatus(status *invokerconn.Status) {
	if i.failed || i.status.Epoch >= status.Epoch {
		return
	}

	i.status = status

	for jobID, holder := range i.jobHolderByID {
		switch holder.jobType {
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
				logger.Warn("job %s disappeared from status, but then reappeared, invoker: %s", jobID, i.address())
				i.markFailed()
				return
			}
		case UnknownJob:
			logger.Panic("invoker shouldn't store job of type UnknownJob")
		}
	}

	logger.Trace(
		"status of invoker %s is updated; active jobs: %d, max new jobs: %d",
		i.address(),
		len(status.ActiveJobIDs),
		status.MaxNewJobs,
	)
}

func (i *Invoker) markFailed() {
	if i.failed {
		return
	}

	logger.Warn("marking invoker %s as failed", i.address())
	i.failed = true
	i.cancel()
	i.ts.Go(func() { i.registry.OnInvokerFailure(i) })
}

func (i *Invoker) ExtractJobs() []string {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	jobs := make([]string, 0, len(i.jobHolderByID))
	for jobID := range i.jobHolderByID {
		jobs = append(jobs, jobID)
	}

	i.jobHolderByID = make(map[string]*jobHolder)
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

func (i *Invoker) StatusForMaster() *masterconn.InvokerStatus {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	status := &masterconn.InvokerStatus{
		Address:     i.address(),
		MaxNewJobs:  int(i.status.MaxNewJobs) - i.jobTypesCount[SendingJob],
		TimeAdded:   i.timeAdded,
		TestingJobs: make([]*invokerconn.Job, 0),
	}

	for _, holder := range i.jobHolderByID {
		status.TestingJobs = append(status.TestingJobs, holder.job)
	}
	return status
}
