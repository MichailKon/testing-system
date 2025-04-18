package jobgenerators

import (
	"fmt"
	"github.com/google/uuid"
	"sync"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/lib/logger"
)

type state int

const (
	compilationNotStarted state = iota
	compilationStarted
	compilationFinished
)

type ICPCGenerator struct {
	id          string
	mutex       sync.Mutex
	submission  *models.Submission
	problem     *models.Problem
	state       state
	hasFails    bool
	givenJobs   map[string]*invokerconn.Job
	futureTests []uint64
}

func (i *ICPCGenerator) ID() string {
	return i.id
}

// finalizeResults must be done with acquired mutex
func (i *ICPCGenerator) finalizeResults() {
	setSkipped := false
	for j := range i.submission.TestResults {
		if setSkipped {
			i.submission.TestResults[j].Verdict = verdict.SK
			continue
		}
		if i.submission.TestResults[j].Verdict == verdict.OK || i.submission.TestResults[j].Verdict == verdict.SK {
			continue
		}
		setSkipped = true
		i.submission.Verdict = i.submission.TestResults[j].Verdict
	}
	if !setSkipped {
		i.submission.Score = 1
		i.submission.Verdict = verdict.OK
	}
}

func (i *ICPCGenerator) NextJob() *invokerconn.Job {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	if i.state == compilationFinished && len(i.futureTests) == 0 {
		return nil
	}
	if i.state == compilationStarted {
		return nil
	}
	id, err := uuid.NewV7()
	if err != nil {
		logger.Panic("Can't generate id for job: %w", err)
	}
	job := &invokerconn.Job{
		ID:       id.String(),
		SubmitID: i.submission.ID,
	}
	if i.state == compilationNotStarted {
		job.Type = invokerconn.CompileJob
		i.state = compilationStarted
	} else {
		if i.hasFails {
			return nil
		}
		job.Test = i.futureTests[0]
		i.futureTests = i.futureTests[1:]
		job.Type = invokerconn.TestJob
	}
	i.givenJobs[job.ID] = job
	return job
}

// compileJobCompleted must be done with acquired mutex
func (i *ICPCGenerator) compileJobCompleted(job *invokerconn.Job, result *masterconn.InvokerJobResult) (*models.Submission, error) {
	if job.Type != invokerconn.CompileJob {
		return nil, fmt.Errorf("job type %s is not compile job", job.ID)
	}
	switch result.Verdict {
	case verdict.CD:
		i.state = compilationFinished
		return nil, nil
	case verdict.CE:
		i.submission.Verdict = result.Verdict
		return i.submission, nil
	default:
		return nil, fmt.Errorf("unknown verdict for compilation completed: %v", result.Verdict)
	}
}

// testJobCompleted must be done with acquired mutex
func (i *ICPCGenerator) testJobCompleted(job *invokerconn.Job, result *masterconn.InvokerJobResult) (*models.Submission, error) {
	i.submission.TestResults[job.Test-1].Verdict = result.Verdict
	switch result.Verdict {
	case verdict.OK:
		if len(i.givenJobs) == 0 && (len(i.futureTests) == 0 || i.hasFails) {
			i.finalizeResults()
			return i.submission, nil
		}
		return nil, nil
	case verdict.PT, verdict.WA, verdict.RT, verdict.ML, verdict.TL, verdict.WL, verdict.SE, verdict.CF:
		i.hasFails = true
		if len(i.givenJobs) == 0 {
			i.finalizeResults()
			return i.submission, nil
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown verdict for testing completed: %v", result.Verdict)
	}
}

func (i *ICPCGenerator) JobCompleted(result *masterconn.InvokerJobResult) (*models.Submission, error) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	job, ok := i.givenJobs[result.JobID]
	if !ok {
		return nil, fmt.Errorf("job not found")
	}
	delete(i.givenJobs, result.JobID)

	switch job.Type {
	case invokerconn.CompileJob:
		return i.compileJobCompleted(job, result)
	case invokerconn.TestJob:
		return i.testJobCompleted(job, result)
	default:
		return nil, fmt.Errorf("unknown job type for ICPC problem: %v", job.Type)
	}
}

func newICPCGenerator(problem *models.Problem, submission *models.Submission) (Generator, error) {
	id, err := uuid.NewV7()
	if err != nil {
		logger.Panic("Can't generate generator id: %w", err)
	}

	if problem.ScoringType != models.ScoringTypeICPC {
		return nil, fmt.Errorf("problem %v is not ICPC", problem.ID)
	}
	futureTests := make([]uint64, 0, problem.TestsNumber)
	testResults := make([]models.TestResult, 0, problem.TestsNumber)
	for i := range problem.TestsNumber {
		futureTests = append(futureTests, i+1)
		testResults = append(testResults, models.TestResult{
			TestNumber: i + 1,
			Verdict:    verdict.SK,
			Time:       0,
			Memory:     0,
		})
	}

	submission.TestResults = testResults

	return &ICPCGenerator{
		id:          id.String(),
		submission:  submission,
		problem:     problem,
		givenJobs:   make(map[string]*invokerconn.Job),
		futureTests: futureTests,
	}, nil
}
