package invoker

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"testing_system/common/connectors/invokerconn"
	"testing_system/lib/connector"
)

func (i *Invoker) handleStatus(c *gin.Context) {
	connector.RespOK(c, i.getStatus())
}

func (i *Invoker) handleNewJob(c *gin.Context) {
	job := new(Job)
	err := c.BindJSON(&job.Job)
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Can not parse invoker job, error: %s", err.Error())
		return
	}
	if !i.initJob(c, job) {
		return
	}
	switch job.Type {
	case invokerconn.CompileJob:
		if !i.newCompileJob(c, job) {
			return
		}
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

func (i *Invoker) resetCache(c *gin.Context) {
	i.Storage.Reset()
	connector.RespOK(c, nil)
}

func (i *Invoker) stopJob(c *gin.Context) {
	jobIDBytes, err := c.GetRawData()
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Can not get job id, error: %s", err.Error())
	}
	jobID := string(jobIDBytes)

	i.Mutex.Lock()

	job, ok := i.ActiveJobs[jobID]
	if ok {
		if job.Type == invokerconn.TestJob {
			job.stopFunc()
		}
	}
	i.Mutex.Unlock()
	connector.RespOK(c, nil)
}
