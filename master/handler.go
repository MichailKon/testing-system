//go:generate swag init -g master.go --parseDependency -o ../swag

package master

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/constants/resource"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/lib/connector"
	"testing_system/lib/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (m *Master) handleInvokerPing(c *gin.Context) {
	status := new(invokerconn.StatusResponse)
	err := c.BindJSON(&status)
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Can not parse invoker status, error: %s", err.Error())
		return
	}

	m.invokerRegistry.UpsertInvoker(status)
	connector.RespOK(c, nil)
}

func (m *Master) handleInvokerJobResult(c *gin.Context) {
	result := new(masterconn.InvokerJobResult)
	if err := c.BindJSON(result); err != nil {
		connector.RespErr(c, http.StatusBadRequest, "Can not parse invoker job result, error: %s", err.Error())
		return
	}

	if !m.invokerRegistry.HandleInvokerJobResult(result) {
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

	if err := m.ts.DB.WithContext(c).Save(submission).Error; err != nil {
		logger.Error("while saving submission to the database error happened: %s", err.Error())
		connector.RespErr(c, http.StatusInternalServerError, "DB error")
		return
	}

	connector.RespOK(c, nil)
}

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
// @Success 200 {object} SubmissionResponse "Returns upload confirmation with details"
// @Failure 400 {object} string "Invalid request"
// @Failure 404 {object} string "Problem not found"
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
		connector.RespErr(c, http.StatusInternalServerError, "")
		return
	}

	m.invokerRegistry.SendJobs()

	connector.RespOK(c, &SubmissionResponse{
		SubmissionID: submission.ID,
	})
}

func (m *Master) loadProblem(c *gin.Context, problemID uint) *models.Problem {
	// TODO: cache

	problem := new(models.Problem)
	err := m.ts.DB.WithContext(c).First(problem, problemID).Error
	if err == nil {
		return problem
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		connector.RespErr(c, http.StatusNotFound, "problem not found")
	} else {
		logger.Error("failed to find problem in db, error: %s", err.Error())
		connector.RespErr(c, http.StatusInternalServerError, "")
	}
	return nil
}

func (m *Master) saveSubmissionInStorage(c *gin.Context, problemID uint64, file *multipart.FileHeader) bool {
	reader, err := file.Open()
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "failed to read source code")
		return false
	}
	defer reader.Close()

	request := &storageconn.Request{
		Resource:  resource.SourceCode,
		ProblemID: problemID,
		Files: map[string]io.Reader{
			file.Filename: reader,
		},
	}

	if err := m.ts.StorageConn.Upload(request).Error; err != nil {
		logger.Error("failed to save solution file, error: %s", err.Error())
		connector.RespErr(c, http.StatusInternalServerError, "")
		return false
	}

	return true
}

func (m *Master) saveSubmissionInDB(c *gin.Context, problemID uint64, language string) *models.Submission {
	submission := &models.Submission{
		ProblemID: problemID,
		Language:  language,
		Verdict:   verdict.CL,
	}

	if err := m.ts.DB.WithContext(c).Save(submission).Error; err != nil {
		logger.Error("failed to save submission to db, error: %s", err.Error())
		connector.RespErr(c, http.StatusInternalServerError, "")
		return nil
	}
	return submission
}
