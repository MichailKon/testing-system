package queue

import (
	"fmt"
	"slices"
	"sync"
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/lib/logger"
	"testing_system/master/queue/jobgenerators"
)

type generatorState int

const (
	generatorReady generatorState = iota
	generatorAwaiting
)

type Queue struct {
	ts *common.TestingSystem

	mutex              sync.Mutex
	activeGenerators   []jobgenerators.Generator
	awaitingGenerators []jobgenerators.Generator
	generatorToState   map[jobgenerators.Generator]generatorState
	// tasks, which were produced, but generator already ended its life
	jobsWithoutGenerator map[string]struct{}
	jobIDToGenerator     map[string]jobgenerators.Generator
}

func (q *Queue) Submit(submission *SubmissionHolder) error {
	generator, err := jobgenerators.NewGenerator(submission.Problem, submission.Submission.ID)
	if err != nil {
		return err
	}
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.activeGenerators = append(q.activeGenerators, generator)
	q.generatorToState[generator] = generatorReady
	return nil
}

func (q *Queue) JobCompleted(job *masterconn.InvokerJobResult) (submission *SubmissionHolder, err error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	_, ok := q.jobsWithoutGenerator[job.JobID]
	if ok {
		delete(q.jobsWithoutGenerator, job.JobID)
		return nil, err
	}
	generator, ok := q.jobIDToGenerator[job.JobID]
	if !ok {
		return nil, fmt.Errorf("can't find job %v", job.JobID)
	}
	err = generator.JobCompleted(job)
	switch q.generatorToState[generator] {
	case generatorReady:
		if !generator.CanGiveJob() {
			q.generatorToState[generator] = generatorAwaiting
			generatorInd := slices.Index(q.activeGenerators, generator)
			if generatorInd == -1 {
				return nil, fmt.Errorf("generator with job %v is active but not in map", job.JobID)
			}
			q.activeGenerators = append(q.activeGenerators[:generatorInd], q.activeGenerators[generatorInd+1:]...)
			q.awaitingGenerators = append(q.awaitingGenerators, generator)
		}
	case generatorAwaiting:
		if generator.CanGiveJob() {
			q.generatorToState[generator] = generatorReady
			generatorInd := slices.Index(q.awaitingGenerators, generator)
			if generatorInd == -1 {
				return nil, fmt.Errorf("generator with job %v is awaiting but not in map", job.JobID)
			}
			q.awaitingGenerators = append(q.awaitingGenerators[:generatorInd], q.awaitingGenerators[generatorInd+1:]...)
			q.activeGenerators = append(q.activeGenerators, generator)
		}
	}
	return nil, err
}

func (q *Queue) RescheduleJob(jobID string) error {
	//TODO implement me
	panic("implement me")
}

func (q *Queue) NextJob() *invokerconn.Job {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	attempts := len(q.activeGenerators)
	for range attempts {
		curGenerator := q.activeGenerators[0]
		q.activeGenerators = q.activeGenerators[1:]
		if !curGenerator.CanGiveJob() {
			q.generatorToState[curGenerator] = generatorAwaiting
			q.awaitingGenerators = append(q.awaitingGenerators, curGenerator)
			continue
		}
		job, err := curGenerator.NextJob()
		if err != nil {
			logger.Error("Generator is in active state and can give a task, but there is an error in it: %v",
				err)
			q.activeGenerators = append(q.activeGenerators, curGenerator)
			continue
		}
		if curGenerator.CanGiveJob() {
			q.activeGenerators = append(q.activeGenerators, curGenerator)
			return job.InvokerJob
		}
		q.generatorToState[curGenerator] = generatorReady
		q.awaitingGenerators = append(q.awaitingGenerators, curGenerator)
		return job.InvokerJob
	}
	return nil
}
