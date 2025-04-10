package registry

import (
	"sync"
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
	Connector *invokerconn.Connector
	Mutex     sync.Mutex

	jobTypeByID   map[string]JobType
	jobTypesCount map[JobType]int

	status           *invokerconn.StatusResponse
	curStatusEpoch   int
	lastStatusEpoch  int
	lastStatusUpdate time.Time
	failed           bool
}

func (i *Invoker) NeedToPing(pingInterval time.Duration) bool {
	return !i.failed && (i.status == nil || time.Now().Sub(i.lastStatusUpdate) > pingInterval)
}

func (i *Invoker) NextStatusEpoch() int {
	i.curStatusEpoch++
	return i.curStatusEpoch
}

func (i *Invoker) GetJobType(jobID string) JobType {
	jobType, ok := i.jobTypeByID[jobID]
	if !ok {
		return UnknownJob
	}
	return jobType
}

func (i *Invoker) SetJobType(jobID string, jobType JobType) {
	i.jobTypesCount[i.GetJobType(jobID)]--
	i.jobTypeByID[jobID] = jobType
	i.jobTypesCount[i.GetJobType(jobID)]++
}

func (i *Invoker) RemoveJob(jobID string) {
	i.jobTypesCount[i.GetJobType(jobID)]--
	delete(i.jobTypeByID, jobID)
}

func (i *Invoker) CanSendJob() bool {
	return !i.failed && i.status != nil && i.jobTypesCount[SendingJob]+1 <= int(i.status.MaxNewJobs)
}

func isJobTesting(jobID string, status *invokerconn.StatusResponse) bool {
	for _, id := range status.ActiveJobIDs {
		if jobID == id {
			return true
		}
	}
	return false
}

func (i *Invoker) SetStatus(status *invokerconn.StatusResponse, epoch int) (jobsToReschedule, newNoReplyJobs []string) {
	if i.failed || (status != nil && epoch <= i.lastStatusEpoch) {
		return
	}

	i.status = status
	i.lastStatusEpoch = epoch
	i.lastStatusUpdate = time.Now()

	for jobID, jobType := range i.jobTypeByID {
		switch jobType {
		case SendingJob:
			// pass
		case TestingJob:
			if !isJobTesting(jobID, status) {
				newNoReplyJobs = append(newNoReplyJobs, jobID)
				i.SetJobType(jobID, NoReplyJob)
			}
		case NoReplyJob:
			if isJobTesting(jobID, status) {
				logger.Info("job %s diappeared from status, but then reappeared", jobID)
				return i.MarkFailed(), nil
			}
		case UnknownJob:
			logger.Error("invoker shouldn't store job of type UnknownJob")
		}
	}
	return
}

func (i *Invoker) MarkFailed() []string {
	if i.failed {
		return nil
	}
	i.failed = true

	jobs := make([]string, 0, len(i.jobTypeByID))
	for jobID := range i.jobTypeByID {
		jobs = append(jobs, jobID)
	}

	i.jobTypeByID = nil
	i.jobTypesCount = nil

	return jobs
}

func newInvoker(connector *invokerconn.Connector) *Invoker {
	return &Invoker{
		Connector:     connector,
		jobTypeByID:   make(map[string]JobType),
		jobTypesCount: make(map[JobType]int),
	}
}
