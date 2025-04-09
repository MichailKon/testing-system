package jobgenerators

import (
	"fmt"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/db/models"
)

type GeneratorJob struct {
	InvokerJob *invokerconn.Job

	blockedBy  []string
	requiredBy []string
}

type Generator interface {
	// GivenJobs returns jobs, that has been given, but has not been completed yet
	GivenJobs() map[string]*GeneratorJob

	// CanGiveJob returns true iff Generator can give any job
	CanGiveJob() bool

	IsTestingCompleted() bool

	// Score returns score (wow) if HaveJobs() == false; it returns an error if HaveJobs() == true
	Score() (float64, error)

	// RescheduleJob reschedules a job and changes it's ID
	RescheduleJob(jobID string)

	// NextJob returns _some_ job from this generator
	NextJob() (*GeneratorJob, error)

	// JobCompleted returns an errors, if it couldn't complete a job for some reason
	JobCompleted(result *masterconn.InvokerJobResult) error
}

func NewGenerator(problem *models.Problem, submitID uint) (Generator, error) {
	switch problem.ProblemType {
	case models.ProblemType_ICPC:
		return newICPCGenerator(problem, submitID)
	default:
		return nil, fmt.Errorf("unknown problem type %v", problem.ProblemType)
	}
}
