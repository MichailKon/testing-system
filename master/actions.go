package master

import (
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/constants/resource"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/lib/connector"
	"testing_system/lib/logger"

	"github.com/cenkalti/backoff/v5"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Some functions should always be executed even when fail happens. We will retry these functions here
func (m *Master) retryUntilOK(
	firstTryCtx context.Context,
	f func(ctx context.Context, submission *models.Submission) error,
	submission *models.Submission,
) {
	err := f(firstTryCtx, submission)
	if err == nil {
		return
	}
	m.ts.Go(func() {
		_, err = backoff.Retry(
			m.ts.StopCtx,
			func() (*struct{}, error) {
				return nil, f(m.ts.StopCtx, submission)
			},
			backoff.WithBackOff(backoff.NewExponentialBackOff()),
		)

		if err != nil {
			logger.Panic("Master retry operation has failed too many times, error: %v", err)
		}
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
		connector.RespErr(c, http.StatusInternalServerError, "internal error")
	}
	return nil
}

func (m *Master) saveSubmissionInStorage(c *gin.Context, submission *models.Submission, file *multipart.FileHeader) bool {
	reader, err := file.Open()
	if err != nil {
		connector.RespErr(c, http.StatusBadRequest, "failed to read source code")
		return false
	}
	defer reader.Close()

	request := &storageconn.Request{
		Resource:        resource.SourceCode,
		SubmitID:        uint64(submission.ID),
		StorageFilename: file.Filename,
		File:            reader,
		Ctx:             c,
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
		Verdict:   verdict.RU,
	}

	if err := m.ts.DB.WithContext(c).Save(submission).Error; err != nil {
		logger.Error("failed to save submission to db, error: %s", err.Error())
		connector.RespErr(c, http.StatusInternalServerError, "internal error")
		return nil
	}

	return submission
}

func (m *Master) removeSubmissionFromDB(ctx context.Context, submission *models.Submission) error {
	err := m.ts.DB.WithContext(ctx).Delete(submission).Error
	if err != nil {
		logger.Error("failed to remove submission %d, error: %v", submission.ID, err)
		return err
	}
	return nil
}

func (m *Master) removeSubmissionFromStorage(ctx context.Context, submission *models.Submission) error {
	request := &storageconn.Request{
		Resource: resource.SourceCode,
		SubmitID: uint64(submission.ID),
		Ctx:      ctx,
	}

	if err := m.ts.StorageConn.Delete(request).Error; err != nil {
		logger.Error("failed to remove submission %d from storage, error: %v", submission.ID, err)
		return err
	}
	return nil
}

func (m *Master) updateSubmission(ctx context.Context, submission *models.Submission) error {
	if err := m.ts.DB.WithContext(ctx).Save(submission).Error; err != nil {
		logger.Error("failed to save submission %d to db, error: %v", submission.ID, err)
		return err
	}
	return nil
}
