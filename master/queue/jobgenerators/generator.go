package jobgenerators

import (
	"fmt"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/db/models"
)

type Generator interface {
	// RescheduleJob reschedules a job and changes it's ID
	RescheduleJob(jobID string) error

	// NextJob returns _some_ job from this generator, or an error if there are no jobs
	NextJob() (*invokerconn.Job, error)

	// JobCompleted returns an errors, if it couldn't complete a job for some reason;
	// submission is not nil if status is finalized
	JobCompleted(jobResult *masterconn.InvokerJobResult) (*models.Submission, error)
}

func NewGenerator(problem *models.Problem, submission *models.Submission) (Generator, error) {
	switch problem.ProblemType {
	case models.ProblemType_ICPC:
		return newICPCGenerator(problem, submission)
	default:
		return nil, fmt.Errorf("unknown problem type %v", problem.ProblemType)
	}
}
