package invoker

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"slices"
	"sync"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/invoker/sandbox"
	"testing_system/lib/connector"
	"testing_system/lib/logger"
)

type Job struct {
	invokerconn.Job

	Submission *models.Submission
	Problem    *models.Problem

	Defers []func()

	InteractiveData *InteractiveJobData
}

type InteractiveJobData struct {
	PipelineReadyWG       sync.WaitGroup
	SolutionPipelineState *JobPipelineState
}

func (j *Job) deferFunc() {
	slices.Reverse(j.Defers)
	for _, f := range j.Defers {
		f()
	}
	j.Defers = nil
}

func (i *Invoker) failJob(j *Job, errf string, args ...interface{}) {
	request := &masterconn.InvokerJobResult{
		JobID:         j.ID,
		Verdict:       verdict.CF,
		Error:         fmt.Sprintf(errf, args...),
		InvokerStatus: i.getStatus(),
	}
	err := i.TS.MasterConn.SendInvokerJobResult(request)
	if err != nil {
		logger.Panic("Can not send invoker request, error: %s", err.Error())
		// TODO: Add normal handling of this error
	}
	i.Mutex.Lock()
	defer i.Mutex.Unlock()
	delete(i.ActiveJobs, j.ID)
}

func (i *Invoker) successJob(j *Job, runResult *sandbox.RunResult) {
	request := &masterconn.InvokerJobResult{
		JobID:         j.ID,
		Verdict:       runResult.Verdict,
		Points:        runResult.Points,
		Statistics:    runResult.Statistics,
		InvokerStatus: i.getStatus(),
	}
	err := i.TS.MasterConn.SendInvokerJobResult(request)
	if err != nil {
		logger.Panic("Can not send invoker request, error: %s", err.Error())
		// TODO: Add normal handling of this error
	}
	i.Mutex.Lock()
	defer i.Mutex.Unlock()
	delete(i.ActiveJobs, j.ID)
}

func (i *Invoker) initJob(c *gin.Context, job *Job) bool {
	var submission models.Submission
	if err := i.TS.DB.WithContext(c).Find(&submission, job.SubmitID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			connector.RespErr(c, http.StatusBadRequest, "Submission %d not found", job.SubmitID)
		} else {
			logger.Error("Error while finding submission in db, error: %s", err.Error())
			connector.RespErr(c, http.StatusInternalServerError, "DB Error")
			return false
		}
	}
	job.Submission = &submission

	var problem models.Problem
	if err := i.TS.DB.WithContext(c).Find(&problem, job.Submission.ProblemID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			connector.RespErr(c, http.StatusBadRequest, "Problem %d not found", job.Submission.ProblemID)
		} else {
			logger.Error("Error while finding problem in db, error: %s", err.Error())
			connector.RespErr(c, http.StatusInternalServerError, "DB Error")
			return false
		}
	}
	job.Problem = &problem
	return true
}

func (i *Invoker) newCompileJob(c *gin.Context, job *Job) {
	i.Storage.Source.Lock(uint64(job.Submission.ID))
	job.Defers = append(job.Defers, func() { i.Storage.Source.Unlock(uint64(job.Submission.ID)) })

	i.JobQueue <- job
}

func (i *Invoker) newTestJob(c *gin.Context, job *Job) bool {
	if job.Test <= 0 || job.Test > job.Problem.TestsNumber {
		connector.RespErr(c,
			http.StatusBadRequest,
			"%d test required, tests in problem %d are numbered from 1 to %d",
			job.Test, job.Problem.ID, job.Problem.TestsNumber)
		return false
	}

	i.Storage.Binary.Lock(uint64(job.Submission.ID))
	job.Defers = append(job.Defers, func() { i.Storage.Binary.Unlock(uint64(job.Submission.ID)) })

	switch job.Problem.ProblemType {
	case models.ProblemTypeStandard:
		return i.finishCreatingStandardJob(c, job)
	case models.ProblemTypeInteractive:
		return i.finishCreatingInteractiveJob(c, job)
	default:
		logger.Warn("Unknown problem type %s for problem %d", job.Problem.ProblemType, job.Problem.ID)
		connector.RespErr(c, http.StatusInternalServerError, "Unknown problem type")
		return false
	}
}

func (i *Invoker) finishCreatingStandardJob(c *gin.Context, job *Job) bool {
	i.Storage.TestInput.Lock(uint64(job.Problem.ID), job.Test)
	job.Defers = append(job.Defers, func() { i.Storage.TestInput.Unlock(uint64(job.Problem.ID), job.Test) })

	i.Storage.TestAnswer.Lock(uint64(job.Problem.ID), job.Test)
	job.Defers = append(job.Defers, func() { i.Storage.TestAnswer.Unlock(uint64(job.Problem.ID), job.Test) })

	i.Storage.Checker.Lock(uint64(job.Problem.ID))
	job.Defers = append(job.Defers, func() { i.Storage.Checker.Unlock(uint64(job.Problem.ID)) })

	i.JobQueue <- job
	return true
}

func (i *Invoker) finishCreatingInteractiveJob(c *gin.Context, solutionJob *Job) bool {
	if i.TS.Config.Invoker.Sandboxes < 2 {
		logger.Error("Can not run interactive problems as they require at least two sandboxes")
		connector.RespErr(c, http.StatusInternalServerError, "Can not run interactive problems")
		return false
	}

	interactorJob := createInteractorJob(solutionJob)

	i.Storage.TestInput.Lock(uint64(interactorJob.Problem.ID), interactorJob.Test)
	interactorJob.Defers = append(interactorJob.Defers, func() {
		i.Storage.TestInput.Unlock(uint64(interactorJob.Problem.ID), interactorJob.Test)
	})

	i.Storage.TestAnswer.Lock(uint64(interactorJob.Problem.ID), interactorJob.Test)
	interactorJob.Defers = append(interactorJob.Defers, func() {
		i.Storage.TestAnswer.Unlock(uint64(interactorJob.Problem.ID), interactorJob.Test)
	})

	i.Storage.Checker.Lock(uint64(interactorJob.Problem.ID))
	interactorJob.Defers = append(interactorJob.Defers, func() {
		i.Storage.Checker.Unlock(uint64(interactorJob.Problem.ID))
	})

	i.Storage.Interactor.Lock(uint64(interactorJob.Problem.ID))
	interactorJob.Defers = append(interactorJob.Defers, func() {
		i.Storage.Interactor.Unlock(uint64(interactorJob.Problem.ID))
	})

	i.JobQueue <- interactorJob
	i.JobQueue <- solutionJob
	return true
}

func createInteractorJob(solutionJob *Job) *Job {
	interactorJob := &Job{
		Job:        solutionJob.Job,
		Submission: solutionJob.Submission,
		Problem:    solutionJob.Problem,
	}
	solutionJob.Type = invokerconn.InteractiveSolutionJob
	interactorJob.Type = invokerconn.InteractiveInteractorJob

	interactorJob.InteractiveData = new(InteractiveJobData)
	solutionJob.InteractiveData = interactorJob.InteractiveData
	interactorJob.InteractiveData.PipelineReadyWG.Add(1)
	return interactorJob
}
