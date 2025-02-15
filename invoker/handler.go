package invoker

import (
	"errors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/db/models"
	"testing_system/lib/connector"
	"testing_system/lib/logger"
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
	case invokerconn.Compile:
		i.newCompileJob(c, job)
	case invokerconn.Test:
		i.newTestJob(c, job)
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
	i.JobQueue <- job
}

func (i *Invoker) newTestJob(c *gin.Context, job *Job) {
	i.Storage.Binary.Lock(uint64(job.Submission.ID))
	i.Storage.Test.Lock(uint64(job.Problem.ID), job.Test)
	i.Storage.Checker.Lock(uint64(job.Problem.ID))
	// TODO: interactor
	i.JobQueue <- job
}
