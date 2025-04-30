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
	jobIDToOriginalJobID map[string]string
	newFailedJobs        []*invokerconn.Job
	originalJobIDToJob   map[string]*invokerconn.Job

	originalJobIDToGenerator map[string]jobgenerators.Generator
	activeGeneratorIDs       map[string]struct{}
}

func (q *Queue) Submit(problem *models.Problem, submission *models.Submission) error {
	generator, err := jobgenerators.NewGenerator(problem, submission)
	if err != nil {
		return err
	}
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.activeGenerators.PushBack(generator)
	q.activeGeneratorIDs[generator.ID()] = struct{}{}
	logger.Trace("Registered submission %d for problem %d in queue", submission.ID, problem.ID)
	return nil
}

func (q *Queue) JobCompleted(jobResult *masterconn.InvokerJobResult) (submission *models.Submission, err error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	wasID := jobResult.Job.ID
	if origID, ok := q.jobIDToOriginalJobID[jobResult.Job.ID]; ok {
		delete(q.jobIDToOriginalJobID, jobResult.Job.ID)
		jobResult.Job.ID = origID
		defer func() {
			jobResult.Job.ID = wasID
		}()
	}

	generator, ok := q.originalJobIDToGenerator[jobResult.Job.ID]
	if !ok {
		if wasID != jobResult.Job.ID {
			logger.Panic("Job has id=%v and origID=%v; was not found in originalJobIDToGenerator",
				wasID, jobResult.Job.ID)
		}
		return nil, fmt.Errorf("no job with id=%v (origID=%v)", jobResult.Job.ID, wasID)
	}
	delete(q.originalJobIDToJob, jobResult.Job.ID)
	delete(q.originalJobIDToGenerator, jobResult.Job.ID)

	if _, ok = q.activeGeneratorIDs[generator.ID()]; !ok {
		q.activeGenerators.PushBack(generator)
	}

	logger.Trace("Job %s result is received by queue", wasID)
	return generator.JobCompleted(jobResult)
}

func (q *Queue) RescheduleJob(jobID string) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	var origJob *invokerconn.Job
	wasID := jobID
	if origID, ok := q.jobIDToOriginalJobID[jobID]; ok {
		delete(q.jobIDToOriginalJobID, jobID)
		jobID = origID
		if origJob, ok = q.originalJobIDToJob[jobID]; !ok {
			logger.Panic("Job has id=%v and origID=%v; was not found in originalJobIDToGenerator",
				wasID, jobID)
		}
	} else {
		origJob, ok = q.originalJobIDToJob[jobID]
		if !ok {
			return fmt.Errorf("no job with id=%v (origID=%v)", wasID, jobID)
		}
	}
	newUUID, err := uuid.NewV7()
	if err != nil {
		logger.Panic("Can't generate job id: %v", err)
	}
	origJob.ID = newUUID.String()
	q.jobIDToOriginalJobID[origJob.ID] = jobID
	q.newFailedJobs = append(q.newFailedJobs, origJob)

	logger.Trace("Rescheduled job %s, new id: %s", wasID, newUUID)
	return nil
}

func (q *Queue) NextJob() *invokerconn.Job {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	if len(q.newFailedJobs) > 0 {
		job := q.newFailedJobs[0]
		q.newFailedJobs = q.newFailedJobs[1:]
		logger.Trace("Queue returns rescheduled job %v", job)
		return job
	}
	attempts := q.activeGenerators.Len()
	for range attempts {
		generatorListElement := q.activeGenerators.Front()
		generator := generatorListElement.Value.(jobgenerators.Generator)
		job := generator.NextJob()
		if job == nil {
			q.activeGenerators.Remove(generatorListElement)
			delete(q.activeGeneratorIDs, generator.ID())
			continue
		}
		q.activeGenerators.MoveToBack(generatorListElement)
		q.originalJobIDToGenerator[job.ID] = generator
		q.originalJobIDToJob[job.ID] = job
		logger.Trace("Queue returns new job %v", job)
		return job
	}
	return nil
}
