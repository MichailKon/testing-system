package master

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"testing_system/common/connectors/masterconn"
	"testing_system/lib/logger"
)

// @Summary Submit
// @Description Submit a solution
// @Tags Client
// @Accept multipart/form-data
// @Produce json
// @Param ProblemID formData uint true "Problem ID" example:"228"
// @Param Language formData string true "Programming language" example:"g++"
// @Param Solution formData file true "Source code"
// @Success 200 {object} masterconn.SubmissionResponse
// @Failure 400 {object} string
// @Failure 404 {object} string
// @Failure 500 {object} string
// @Router /master/submit [post]
func (m *Master) handleNewSubmission(c *gin.Context) {
	problemIDStr := c.PostForm("ProblemID")
	language := c.PostForm("Language")

	problemID, err := strconv.ParseUint(problemIDStr, 10, 0)
	if err != nil {
		c.String(http.StatusBadRequest, "ProblemID is not uint")
		return
	}

	file, err := c.FormFile("Solution")
	if err != nil {
		c.String(http.StatusBadRequest, "No source code")
		return
	}

	problem := m.loadProblem(c, uint(problemID))
	if problem == nil {
		return
	}

	submission := m.saveSubmissionInDB(c, uint(problemID), language)
	if submission == nil {
		return
	}

	if !m.saveSubmissionInStorage(c, submission, file) {
		m.retryUntilOK(m.removeSubmissionFromDB, submission)
		return
	}

	logger.Trace("new submission, id: %d, problem: %d, language: %s", submission.ID, problem.ID, language)

	if err = m.queue.Submit(problem, submission); err != nil {
		m.retryUntilOK(m.removeSubmissionFromDB, submission)
		m.retryUntilOK(m.removeSubmissionFromStorage, submission)

		logger.Error("failed to submit to queue, error: %s", err.Error())
		c.String(http.StatusInternalServerError, "internal server error")
		return
	}
	m.ts.Metrics.MasterQueueSize.Inc()

	m.invokerRegistry.SendJobs()

	c.JSON(http.StatusOK, masterconn.SubmissionResponse{submission.ID})
}

// @Summary Status
// @Description Status of master
// @Tags Client
// @Produce json
// @Param prevEpoch query string false "epoch of previous update" example:"long-epoch-name"
// @Success 200 {object} masterconn.Status
// @Failure 500 {object} string
// @Router /master/status [get]
func (m *Master) handleStatus(c *gin.Context) {
	prevEpoch := c.Query("prevEpoch")
	status := m.queue.Status().GetStatus(prevEpoch)
	status.Invokers = m.invokerRegistry.Status()
	c.JSON(http.StatusOK, status)
}

// @Summary Reset invoker cache
// @Description Resetting cache in all invokers
// @Tags Client
// @Produce plain
// @Success 200 {string} Helloworld
// @Failure 500 {object} string
// @Router /master/reset_invoker_cache [post]
func (m *Master) handleResetInvokerCache(c *gin.Context) {
	err := m.invokerRegistry.ResetCache()
	if err != nil {
		logger.Error("failed to reset invoker cache, error: %v", err)
		c.String(http.StatusInternalServerError, "internal server error")
		return
	}
	c.String(http.StatusOK, "OK")
}
