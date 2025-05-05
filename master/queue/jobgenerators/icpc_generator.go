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
	"testing_system/master/queue/queuestatus"
)

type ICPCGenerator struct {
	id    string
	mutex sync.Mutex

	submission *models.Submission
	problem    *models.Problem

	state       generatorState
	firstTestToGive    uint64
	testedPrefixLength uint64

	givenJobs           map[string]*invokerconn.Job
	internalTestResults map[uint64]*models.TestResult

	statusUpdater *queuestatus.QueueStatus
}

func (i *ICPCGenerator) ID() string {
	return i.id
}

func (i *ICPCGenerator) NextJob() *invokerconn.Job {
	i.mutex.Lock()
	defer i.mutex.Unlock()
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
	} else if i.firstTestToGive > i.problem.TestsNumber {
		return nil
	} else {
		job.Type = invokerconn.TestJob
		job.Test = i.firstTestToGive
		i.firstTestToGive++
	}

	i.givenJobs[job.ID] = job
	return job
}

func (i *ICPCGenerator) setFail() {
	for i.firstTestToGive <= i.problem.TestsNumber {
		i.internalTestResults[i.firstTestToGive] = &models.TestResult{
			TestNumber: i.firstTestToGive,
			Verdict:    verdict.SK,
		}
		i.firstTestToGive++
	}
}

func (i *ICPCGenerator) updateSubmissionResult() (*models.Submission, error) {
	updated := false
	defer func() {
		if updated {
			i.statusUpdater.UpdateSubmission(i.submission)
		}
	}()

	for i.testedPrefixLength < i.problem.TestsNumber {
		result, ok := i.internalTestResults[i.testedPrefixLength+1]
		if !ok {
			return nil, nil
		}
		updated = true
		i.testedPrefixLength++
		if i.submission.Verdict != verdict.RU {
			result = &models.TestResult{
				TestNumber: i.testedPrefixLength,
				Verdict:    verdict.SK,
			}
		}
		switch result.Verdict {
		case verdict.OK, verdict.SK:
			// skip
		default:
			if i.submission.Verdict != verdict.RU {
				logger.Panic("Trying to change bad verdict in ICPC problem")
			}
			i.submission.Verdict = result.Verdict
		}
		i.submission.TestResults = append(i.submission.TestResults, result)
	}
	// If we went here, then submission is tested
	if i.submission.Verdict == verdict.RU {
		i.submission.Verdict = verdict.OK
		i.submission.Score = 1
	}
	return i.submission, nil
}

// compileJobCompleted must be done with acquired mutex
func (i *ICPCGenerator) compileJobCompleted(job *invokerconn.Job, result *masterconn.InvokerJobResult) {
	if job.Type != invokerconn.CompileJob {
		logger.Panic("Treating job %s of type %v as compile job", job.ID, job.Type)
	}
	i.state = compilationFinished
	switch result.Verdict {
	case verdict.CD:
		// skip
	case verdict.CE, verdict.CF:
		i.submission.Verdict = result.Verdict
		i.setFail()
	default:
		result.Verdict = verdict.CF
		result.Error = fmt.Sprintf("unknown verdict for compile job: %v", result.Verdict)
		i.submission.Verdict = result.Verdict
		i.setFail()
	}
	i.submission.CompilationResult = buildTestResult(job, result)
}

// testJobCompleted must be done with acquired mutex
func (i *ICPCGenerator) testJobCompleted(job *invokerconn.Job, result *masterconn.InvokerJobResult) {
	if job.Type != invokerconn.TestJob {
		logger.Panic("Treating job %s of type %v as test job", job.ID, job.Type)
	}
	switch result.Verdict {
	case verdict.OK:
		// skip
	case verdict.PT, verdict.WA, verdict.RT, verdict.ML, verdict.TL, verdict.WL, verdict.SE, verdict.CF:
		i.setFail()
	default:
		result.Verdict = verdict.CF
		result.Error = fmt.Sprintf("unknown verdict for test job: %v", result.Verdict)
		i.setFail()
	}
	i.internalTestResults[job.Test] = buildTestResult(job, result)
}

func (i *ICPCGenerator) JobCompleted(result *masterconn.InvokerJobResult) (*models.Submission, error) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	job, ok := i.givenJobs[result.Job.ID]
	if !ok {
		logger.Panic("Wrong job %s is passed to ICPC generator", job.ID)
		return nil, nil
	}
	delete(i.givenJobs, result.Job.ID)

	switch job.Type {
	case invokerconn.CompileJob:
		i.compileJobCompleted(job, result)
	case invokerconn.TestJob:
		i.testJobCompleted(job, result)
	default:
		logger.Panic("unknown job type for ICPC problem: %v", job.Type)
		// never pass here
	}
	return i.updateSubmissionResult()
}

func newICPCGenerator(problem *models.Problem, submission *models.Submission, status *queuestatus.QueueStatus) (Generator, error) {
	id, err := uuid.NewV7()
	if err != nil {
		logger.Panic("Can't generate generator id: %w", err)
	}

	if problem.ProblemType != models.ProblemTypeICPC {
		return nil, fmt.Errorf("problem %v is not ICPC", problem.ID)
	}
	submission.Verdict = verdict.RU

	return &ICPCGenerator{
		id: id.String(),

		submission: submission,
		problem:    problem,

		state:              compilationNotStarted,
		firstTestToGive:    1,
		testedPrefixLength: 0,

		givenJobs:           make(map[string]*invokerconn.Job),
		internalTestResults: make(map[uint64]*models.TestResult),

		statusUpdater: status,
	}, nil
}
