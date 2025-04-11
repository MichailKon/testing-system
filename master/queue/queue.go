package queue

import (
	"container/list"
	"fmt"
	"github.com/google/uuid"
	"sync"
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/db/models"
	"testing_system/lib/logger"
	"testing_system/master/queue/jobgenerators"
)

type Queue struct {
	ts *common.TestingSystem

	mutex            sync.Mutex
	activeGenerators list.List
	// in case of reschedule, new ID will be mapped to the first one
	jobIdToOriginalJobId map[string]string
	newFailedJobs        []*invokerconn.Job
	originalJobIDToJob   map[string]*invokerconn.Job

	originalJobIDToGenerator map[string]jobgenerators.Generator
	activeGeneratorIds       map[string]struct{}
}

func (q *Queue) Submit(problem *models.Problem, submission *models.Submission) error {
	generator, err := jobgenerators.NewGenerator(problem, submission)
	if err != nil {
		return err
	}
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.activeGenerators.PushBack(generator)
	q.activeGeneratorIds[generator.ID()] = struct{}{}
	return nil
}

func (q *Queue) JobCompleted(jobResult *masterconn.InvokerJobResult) (submission *models.Submission, err error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	wasID := jobResult.JobID
	if origId, ok := q.jobIdToOriginalJobId[jobResult.JobID]; ok {
		delete(q.jobIdToOriginalJobId, jobResult.JobID)
		jobResult.JobID = origId
		defer func() {
			jobResult.JobID = wasID
		}()
	}

	generator, ok := q.originalJobIDToGenerator[jobResult.JobID]
	if !ok {
		if wasID != jobResult.JobID {
			logger.Panic("Job has id=%v and origId=%v; was not found in originalJobIDToGenerator",
				wasID, jobResult.JobID)
		}
		return nil, fmt.Errorf("no job with id=%v (origId=%v)", jobResult.JobID, wasID)
	}
	delete(q.originalJobIDToJob, jobResult.JobID)
	delete(q.originalJobIDToGenerator, jobResult.JobID)
	return generator.JobCompleted(jobResult)
}

func (q *Queue) RescheduleJob(jobID string) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	var origJob *invokerconn.Job
	wasId := jobID
	if origId, ok := q.jobIdToOriginalJobId[jobID]; ok {
		delete(q.jobIdToOriginalJobId, jobID)
		jobID = origId
		if origJob, ok = q.originalJobIDToJob[jobID]; !ok {
			logger.Panic("Job has id=%v and origId=%v; was not found in originalJobIDToGenerator",
				wasId, jobID)
		}
	} else {
		origJob, ok = q.originalJobIDToJob[jobID]
		if !ok {
			return fmt.Errorf("no job with id=%v (origId=%v)", wasId, jobID)
		}
	}
	newUUID, err := uuid.NewV7()
	if err != nil {
		logger.Panic("Can't generate job id: %v", err)
	}
	origJob.ID = newUUID.String()
	q.jobIdToOriginalJobId[origJob.ID] = jobID
	q.newFailedJobs = append(q.newFailedJobs, origJob)
	return nil
}

func (q *Queue) NextJob() *invokerconn.Job {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if len(q.newFailedJobs) > 0 {
		job := q.newFailedJobs[0]
		q.newFailedJobs = q.newFailedJobs[1:]
		return job
	}
	attempts := q.activeGenerators.Len()
	for range attempts {
		generatorListElement := q.activeGenerators.Front()
		generator := generatorListElement.Value.(jobgenerators.Generator)
		job := generator.NextJob()
		if job == nil {
			q.activeGenerators.Remove(generatorListElement)
			delete(q.activeGeneratorIds, generator.ID())
			continue
		}
		q.activeGenerators.MoveToBack(generatorListElement)
		q.originalJobIDToGenerator[job.ID] = generator
		q.originalJobIDToJob[job.ID] = job
		return job
	}
	return nil
}
