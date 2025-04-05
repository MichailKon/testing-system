package queue

import (
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
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
	// Processes a new submission
	Submit(submission *SubmissionHolder) error

	// Returns not nil if submission status is finalized
	JobCompleted(job *masterconn.InvokerJobResult) (submission *SubmissionHolder, err error)

	// Puts job back into the queue in case of failure
	RescheduleJob(jobID string) error

	// Returns a new job or nil if no jobs are queued
	NextJob() *invokerconn.Job
}

func NewQueue(ts *common.TestingSystem) IQueue {
	// TODO
	return nil
}
