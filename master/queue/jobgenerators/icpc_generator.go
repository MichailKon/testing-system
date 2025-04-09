package jobgenerators

import (
	"fmt"
	"github.com/google/uuid"
	"slices"
	"sync"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
)

type ICPCGenerator struct {
	problem            *models.Problem
	mutex              sync.RWMutex
	jobs               []*GeneratorJob
	blockedJobs        []*GeneratorJob
	givenJobs          map[string]*GeneratorJob // given, but completed jobs
	hasFailedJobs      bool
	isTestingCompleted bool
}

func (i *ICPCGenerator) GivenJobs() map[string]*GeneratorJob {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.givenJobs
}

func (i *ICPCGenerator) CanGiveJob() bool {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return len(i.jobs) > 0
}

func (i *ICPCGenerator) IsTestingCompleted() bool {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.isTestingCompleted
}

func (i *ICPCGenerator) Score() (float64, error) {
	if !i.IsTestingCompleted() {
		return 0, fmt.Errorf("testing is not completed")
	}
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	if i.hasFailedJobs {
		return 0, nil
	} else {
		return 1, nil
	}
}

func (i *ICPCGenerator) RescheduleJob(jobID string) {
	//TODO implement me
	panic("implement me")
}

func (i *ICPCGenerator) NextJob() (*GeneratorJob, error) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	if len(i.jobs) == 0 {
		return nil, fmt.Errorf("can't give any job")
	}
	job := i.jobs[0]
	i.jobs = i.jobs[1:]
	i.givenJobs[job.InvokerJob.ID] = job
	return job, nil
}

// jobCompletedSuccessfully must be done with acquired mutex
func (i *ICPCGenerator) jobCompletedSuccessfully(job *GeneratorJob) error {
	for _, nextJobID := range job.requiredBy {
		nextJobInd := slices.IndexFunc(i.blockedJobs, func(job *GeneratorJob) bool {
			return job.InvokerJob.ID == nextJobID
		})
		if nextJobInd == -1 {
			return fmt.Errorf("job %v is blocked by %v, but is not in blocked list",
				nextJobID, job.InvokerJob.ID)
		}
		nextJob := i.blockedJobs[nextJobInd]
		curJobInd := slices.Index(nextJob.blockedBy, job.InvokerJob.ID)
		if curJobInd == -1 {
			return fmt.Errorf("job %v is blocked by %v, but the first one doesn't contain the latter one",
				nextJob.InvokerJob.ID, job.InvokerJob.ID)
		}
		nextJob.blockedBy[curJobInd] = nextJob.blockedBy[len(nextJob.blockedBy)-1]
		nextJob.blockedBy = nextJob.blockedBy[:len(nextJob.blockedBy)-1]
		if len(nextJob.blockedBy) == 0 {
			i.jobs = append(i.jobs, nextJob)
			i.blockedJobs = slices.DeleteFunc(i.blockedJobs, func(job *GeneratorJob) bool {
				return job.InvokerJob.ID == nextJobID
			})
		}
	}
	job.requiredBy = nil
	if len(i.givenJobs) == 0 && len(i.jobs) == 0 && len(i.blockedJobs) == 0 {
		i.isTestingCompleted = true
	}
	return nil
}

// jobFailed must be done with acquired mutex
func (i *ICPCGenerator) jobFailed() error {
	i.isTestingCompleted = true
	i.hasFailedJobs = true
	i.jobs = nil
	i.blockedJobs = nil
	return nil
}

// compileJobCompleted must be done with acquired mutex
func (i *ICPCGenerator) compileJobCompleted(job *GeneratorJob, result *masterconn.InvokerJobResult) error {
	switch result.Verdict {
	case verdict.CD:
		return i.jobCompletedSuccessfully(job)
	case verdict.CE:
		return i.jobFailed()
	default:
		return fmt.Errorf("unknown verdict for compilation completed: %v", result.Verdict)
	}
}

// testJobCompleted must be done with acquired mutex
func (i *ICPCGenerator) testJobCompleted(job *GeneratorJob, result *masterconn.InvokerJobResult) error {
	switch result.Verdict {
	case verdict.OK:
		return i.jobCompletedSuccessfully(job)
	case verdict.PT, verdict.WA, verdict.PE, verdict.RT, verdict.ML, verdict.TL, verdict.WL, verdict.SE, verdict.CF:
		return i.jobFailed()
	default:
		return fmt.Errorf("unknown verdict for testing completed: %v", result.Verdict)
	}
}

func (i *ICPCGenerator) JobCompleted(result *masterconn.InvokerJobResult) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	job, ok := i.givenJobs[result.JobID]
	if !ok {
		return fmt.Errorf("job %v does not exist", result.JobID)
	}
	delete(i.givenJobs, result.JobID)
	switch job.InvokerJob.Type {
	case invokerconn.CompileJob:
		return i.compileJobCompleted(job, result)
	case invokerconn.TestJob:
		return i.testJobCompleted(job, result)
	default:
		return fmt.Errorf("unknown job type %d", job.InvokerJob.Type)
	}
}

func newICPCGenerator(problem *models.Problem, submitID uint) (Generator, error) {
	if problem.ProblemType != models.ProblemType_ICPC {
		return nil, fmt.Errorf("problem type is not ICPC")
	}
	// prepare compile job
	compileJob := &GeneratorJob{
		InvokerJob: nil,
		blockedBy:  nil,
		requiredBy: make([]string, 0),
	}
	compileJobUUID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	compileJob.InvokerJob = &invokerconn.Job{
		ID:       compileJobUUID.String(),
		SubmitID: submitID,
		Type:     invokerconn.CompileJob,
	}
	// prepare testing jobs
	testingJobs := make([]*GeneratorJob, 0)
	for i := range problem.TestsNumber {
		testJobUUID, err := uuid.NewV7()
		if err != nil {
			return nil, err
		}
		compileJob.requiredBy = append(compileJob.requiredBy, testJobUUID.String())
		testingJobs = append(testingJobs, &GeneratorJob{
			InvokerJob: &invokerconn.Job{
				ID:       testJobUUID.String(),
				SubmitID: submitID,
				Type:     invokerconn.TestJob,
				Test:     i + 1,
			},
			blockedBy:  []string{compileJob.InvokerJob.ID},
			requiredBy: nil,
		})
	}

	return &ICPCGenerator{
		problem:       problem,
		jobs:          []*GeneratorJob{compileJob},
		blockedJobs:   testingJobs,
		givenJobs:     make(map[string]*GeneratorJob),
		hasFailedJobs: false,
	}, nil
}
