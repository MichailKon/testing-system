package queue

import (
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/db/models"
	"testing_system/master/queue/jobgenerators"
)

/*
Queue is responsible for:
 * generating and scheduling invoker jobs
 * handling finished jobs
 * updating submission statuses (calculating score, updating verdicts, etc.)

It does not directly connect to the DB.
When the submission status is finalized, it returns to the Master, which saves the results to the DB.
*/

type IQueue interface {
	// Submit processes a new submission; you SHOULD NOT submit the same pointer twice
	Submit(problem *models.Problem, submission *models.Submission) error

	// JobCompleted returns not nil if submission status is finalized
	JobCompleted(jobResult *masterconn.InvokerJobResult) (submission *models.Submission, err error)

	// RescheduleJob puts job back into the queue in case of failure
	RescheduleJob(jobID string) error

	// NextJob returns a new job or nil if no jobs to do; each job should be completed or rescheduled
	NextJob() *invokerconn.Job
}

func NewQueue(ts *common.TestingSystem) IQueue {
	return &Queue{
		ts:               ts,
		jobIDToGenerator: make(map[string]jobgenerators.Generator),
		generatorsInList: make(map[jobgenerators.Generator]struct{}),
	}
}
