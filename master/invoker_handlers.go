package master

import (
	"net/http"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/lib/connector"
	"testing_system/lib/logger"

	"github.com/gin-gonic/gin"
)

func (m *Master) handleInvokerStatus(c *gin.Context) {
	status := new(invokerconn.Status)
	err := c.BindJSON(&status)
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "can not parse invoker status, error: %s", err.Error())
		return
	}

	m.invokerRegistry.UpsertInvoker(status)
	connector.RespOK(c, nil)
}

func (m *Master) handleInvokerJobResult(c *gin.Context) {
	result := new(masterconn.InvokerJobResult)
	if err := c.BindJSON(result); err != nil {
		connector.RespErr(c, http.StatusBadRequest, "can not parse invoker job result, error: %s", err.Error())
		return
	}

	logger.Trace("new job result received, job id: %s", result.JobID)
	if !m.invokerRegistry.HandleInvokerJobResult(result) {
		logger.Trace("job %s is unknown or was rescheduled, skipping", result.JobID)
		connector.RespOK(c, nil)
		return
	}

	submission, err := m.queue.JobCompleted(result)
	if err != nil {
		logger.Error("failed to handle tested job, error: %s", err.Error())
		connector.RespErr(c, http.StatusInternalServerError, "Queue error")
		return
	}

	m.invokerRegistry.SendJobs()

	if submission != nil {
		logger.Trace("submission #%d is tested, saving results to db", submission.ID)
		if err := m.ts.DB.WithContext(c).Save(submission).Error; err != nil {
			logger.Error("while saving submission to the database error happened: %s", err.Error())
			connector.RespErr(c, http.StatusInternalServerError, "DB error")
			return
		}
	}

	connector.RespOK(c, nil)
}
