package invoker

import (
	"errors"
	"net/http"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/db/models"
	"testing_system/lib/connector"
	"testing_system/lib/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (i *Invoker) HandleStatus(c *gin.Context) {
	connector.RespOK(c, i.getStatus())
}

func (i *Invoker) HandleNewJob(c *gin.Context) {
	job := new(Job)
	err := c.BindJSON(&job.Job)
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Can not parse invoker jobs, error: %s", err.Error())
		return
	}
	if !i.initJob(c, job) {
		return
	}
	switch job.Type {
	case invokerconn.CompileJob:
		i.newCompileJob(c, job)
	case invokerconn.TestJob:
		if !i.newTestJob(c, job) {
			return
		}
	default:
		connector.RespErr(c, http.StatusBadRequest, "Can not parse job type %v", job.Type)
		return
	}
	i.Mutex.Lock()
	i.ActiveJobs[job.ID] = job
	i.Mutex.Unlock() // We unlock mutex without defer because getStatus uses mutex
	connector.RespOK(c, i.getStatus())
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

	i.Storage.TestInput.Lock(uint64(job.Problem.ID), job.Test)
	job.Defers = append(job.Defers, func() { i.Storage.TestInput.Unlock(uint64(job.Problem.ID), job.Test) })

	i.Storage.TestAnswer.Lock(uint64(job.Problem.ID), job.Test)
	job.Defers = append(job.Defers, func() { i.Storage.TestAnswer.Unlock(uint64(job.Problem.ID), job.Test) })

	i.Storage.Checker.Lock(uint64(job.Problem.ID))
	job.Defers = append(job.Defers, func() { i.Storage.Checker.Unlock(uint64(job.Problem.ID)) })

	// TODO: interactor
	i.JobQueue <- job
	return true
}
