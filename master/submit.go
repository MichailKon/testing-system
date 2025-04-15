//go:generate go run github.com/swaggo/swag/cmd/swag@latest init -g master.go --parseDependency -o ../swag

package master

import (
	"net/http"
	"strconv"
	"testing_system/lib/connector"
	"testing_system/lib/logger"

	"github.com/gin-gonic/gin"
)

type SubmissionResponse struct {
	SubmissionID uint `json:"SubmissionID"`
}

// @Summary Submit
// @Description Submit a solution
// @Tags Client
// @Accept multipart/form-data
// @Produce json
// @Param ProblemId formData uint true "Problem ID" example:"228"
// @Param Language formData string true "Programming language" example:"g++"
// @Param Solution formData file true "Source code"
// @Success 200 {object} SubmissionResponse
// @Failure 400 {object} string
// @Failure 404 {object} string
// @Failure 500 "Internal error"
// @Router /client/submit [post]
func (m *Master) handleNewSubmission(c *gin.Context) {
	problemIDStr := c.PostForm("ProblemId")
	language := c.PostForm("Language")

	problemID, err := strconv.ParseUint(problemIDStr, 10, 0)
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "ProblemID is not uint")
		return
	}

	file, err := c.FormFile("Solution")
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "No source code")
		return
	}

	problem := m.loadProblem(c, uint(problemID))
	if problem == nil {
		return
	}

	if !m.saveSubmissionInStorage(c, problemID, file) {
		return
	}

	submission := m.saveSubmissionInDB(c, problemID, language)
	if submission == nil {
		return
	}

	if err := m.queue.Submit(problem, submission); err != nil {
		logger.Error("failed to submit to queue, error: %s", err.Error())
		connector.RespErr(c, http.StatusInternalServerError, "internal error")
		return
	}

	m.invokerRegistry.SendJobs()

	connector.RespOK(c, &SubmissionResponse{
		SubmissionID: submission.ID,
	})
}
