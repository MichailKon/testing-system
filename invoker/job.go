package invoker

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"slices"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/db/models"
	"testing_system/lib/connector"
	"testing_system/lib/logger"
	"time"
)

type Job struct {
	invokerconn.Job

	submission   *models.Submission
	problem      *models.Problem
	storageEpoch int

	defers     []func()
	createTime time.Time

	stopCtx  context.Context
	stopFunc context.CancelFunc
}

func (j *Job) deferFunc() {
	slices.Reverse(j.defers)
	for _, f := range j.defers {
		f()
	}
	j.defers = nil
}

func (i *Invoker) initJob(c *gin.Context, job *Job) bool {
	job.createTime = time.Now()
	job.storageEpoch = i.Storage.GetEpoch()
	job.stopCtx, job.stopFunc = context.WithCancel(context.Background())

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
	job.submission = &submission

	var problem models.Problem
	if err := i.TS.DB.WithContext(c).Find(&problem, job.submission.ProblemID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			connector.RespErr(c, http.StatusBadRequest, "Problem %d not found", job.submission.ProblemID)
		} else {
			logger.Error("Error while finding problem in db, error: %s", err.Error())
			connector.RespErr(c, http.StatusInternalServerError, "DB Error")
			return false
		}
	}
	job.problem = &problem
	return true
}

func (i *Invoker) newCompileJob(c *gin.Context, job *Job) bool {
	i.Storage.Source.Lock(job.storageEpoch, uint64(job.submission.ID))
	job.defers = append(job.defers, func() { i.Storage.Source.Unlock(job.storageEpoch, uint64(job.submission.ID)) })

	err := i.SandboxThreads.add(job)
	if err != nil {
		logger.Error("Error while adding compile job %s to sandbox queue, error: %s", job.ID, err.Error())
		connector.RespErr(c, http.StatusInternalServerError, "server error")
		return false
	}
	return true
}

func (i *Invoker) newTestJob(c *gin.Context, job *Job) bool {
	if job.Test <= 0 || job.Test > job.problem.TestsNumber {
		connector.RespErr(c,
			http.StatusBadRequest,
			"%d test required, tests in problem %d are numbered from 1 to %d",
			job.Test, job.problem.ID, job.problem.TestsNumber)
		return false
	}

	i.Storage.Binary.Lock(job.storageEpoch, uint64(job.submission.ID))
	job.defers = append(job.defers, func() { i.Storage.Binary.Unlock(job.storageEpoch, uint64(job.submission.ID)) })

	i.Storage.TestInput.Lock(job.storageEpoch, uint64(job.problem.ID), job.Test)
	job.defers = append(job.defers, func() { i.Storage.TestInput.Unlock(job.storageEpoch, uint64(job.problem.ID), job.Test) })

	i.Storage.TestAnswer.Lock(job.storageEpoch, uint64(job.problem.ID), job.Test)
	job.defers = append(job.defers, func() { i.Storage.TestAnswer.Unlock(job.storageEpoch, uint64(job.problem.ID), job.Test) })

	i.Storage.Checker.Lock(job.storageEpoch, uint64(job.problem.ID))
	job.defers = append(job.defers, func() { i.Storage.Checker.Unlock(job.storageEpoch, uint64(job.problem.ID)) })

	// TODO: interactor

	err := i.SandboxThreads.add(job)
	if err != nil {
		logger.Error("Error while adding test job %s to sandbox queue, error: %s", job.ID, err.Error())
		connector.RespErr(c, http.StatusInternalServerError, "server error")
		return false
	}
	return true
}
