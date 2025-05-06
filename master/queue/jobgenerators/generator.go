package jobgenerators

import (
	"fmt"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/db/models"
	"testing_system/master/queue/queuestatus"
)

type Generator interface {
	// ID returns Generator's unique(!) ID
	ID() string

	// NextJob returns _some_ job from this generator
	NextJob() *invokerconn.Job

	// JobCompleted returns an errors, if it couldn't complete a job for some reason;
	// submission is not nil if status is finalized
	JobCompleted(jobResult *masterconn.InvokerJobResult) (*models.Submission, error)
}

func NewGenerator(problem *models.Problem, submission *models.Submission, status *queuestatus.QueueStatus) (Generator, error) {
	switch problem.ProblemType {
	case models.ProblemTypeICPC:
		return newICPCGenerator(problem, submission, status)
	case models.ProblemTypeIOI:
		return NewIOIGenerator(problem, submissionm, status)
	default:
		return nil, fmt.Errorf("unknown problem type %v", problem.ProblemType)
	}
}
