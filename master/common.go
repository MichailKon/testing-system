package master

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/constants/resource"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/lib/connector"
	"testing_system/lib/logger"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

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
		connector.RespErr(c, http.StatusInternalServerError, "internal error")
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
		connector.RespErr(c, http.StatusInternalServerError, "internal error")
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
		connector.RespErr(c, http.StatusInternalServerError, "internal error")
		return nil
	}

	return submission
}
