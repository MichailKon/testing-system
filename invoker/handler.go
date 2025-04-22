package invoker

import (
	"testing_system/common/connectors/invokerconn"
	"testing_system/lib/connector"

	"github.com/gin-gonic/gin"
	"net/http"
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
