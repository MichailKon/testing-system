package queue

import (
	"container/list"
	"fmt"
	"sync"
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/db/models"
	"testing_system/master/queue/jobgenerators"
)

type Queue struct {
	ts *common.TestingSystem

	mutex            sync.Mutex
	activeGenerators list.List
	jobIDToGenerator map[string]jobgenerators.Generator
	generatorsInList map[jobgenerators.Generator]struct{}
}

func (q *Queue) Submit(problem *models.Problem, submission *models.Submission) error {
	generator, err := jobgenerators.NewGenerator(problem, submission)
	if err != nil {
		return err
	}
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.activeGenerators.PushBack(generator)
	q.generatorsInList[generator] = struct{}{}
	return nil
}

func (q *Queue) JobCompleted(jobResult *masterconn.InvokerJobResult) (submission *models.Submission, err error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	generator, ok := q.jobIDToGenerator[jobResult.JobID]
	if !ok {
		return nil, fmt.Errorf("no job with id %v", jobResult.JobID)
	}
	return generator.JobCompleted(jobResult)
}

func (q *Queue) RescheduleJob(jobID string) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	generator := q.jobIDToGenerator[jobID]
	// should check if this generator in activeGenerators, or it may appear several times there
	// that will affect queue's fairness
	if _, ok := q.generatorsInList[generator]; !ok {
		q.activeGenerators.PushBack(generator)
		q.generatorsInList[generator] = struct{}{}
	}
	return generator.RescheduleJob(jobID)
}

func (q *Queue) NextJob() *invokerconn.Job {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	attempts := q.activeGenerators.Len()
	for range attempts {
		generatorListElement := q.activeGenerators.Front()
		q.activeGenerators.Remove(generatorListElement)
		generator := generatorListElement.Value.(jobgenerators.Generator)
		delete(q.generatorsInList, generator)
		job, err := generator.NextJob()
		if err != nil {
			continue
		}
		q.activeGenerators.PushBack(generator)
		q.generatorsInList[generator] = struct{}{}
		q.jobIDToGenerator[job.ID] = generator
		return job
	}
	return nil
}
