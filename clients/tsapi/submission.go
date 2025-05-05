package tsapi

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/xorcare/pointer"
	"gorm.io/gorm"
	"io"
	"net/http"
	"strings"
	"testing_system/clients/tsapi/masterstatus"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/constants/resource"
	"testing_system/common/db/models"
)

func (h *Handler) getSubmissions(c *gin.Context) {
	var filter masterstatus.SubmissionsFilter
	err := c.ShouldBindQuery(&filter)
	if err != nil {
		respError(c, http.StatusBadRequest, "%v", err)
	}

	submissions, err := h.masterStatus.GetSubmissions(c, &filter)

	if err != nil {
		respServerError(c, "Can not load submissions, error: %v", err)
		return
	}
	respSuccess(c, submissions)
}

func (h *Handler) getSubmission(c *gin.Context) {
	submission, ok := h.findSubmission(c)
	if !ok {
		return
	}
	respSuccess(c, submission)
}

type fileData struct {
	Filename string `json:"filename"`
	Data     string `json:"data"`
	Size     uint64 `json:"size"`
}

func (h *Handler) submissionResourceGetter(resourceType resource.Type, limit bool) func(c *gin.Context) {
	return func(c *gin.Context) {
		submission, ok := h.findSubmission(c)
		if !ok {
			return
		}

		request := &storageconn.Request{
			Resource:      resourceType,
			SubmitID:      uint64(submission.ID),
			DownloadBytes: true,
			Ctx:           c,
		}
		if limit {
			request.DownloadHead = pointer.Int64(h.config.LoadFilesHead)
		}
		resp := h.base.StorageConnection.Download(request)
		if resp.Error != nil {
			if errors.Is(resp.Error, storageconn.ErrStorageFileNotFound) {
				respError(c, http.StatusNotFound, "%v for submission %d does not exist", resourceType, submission.ID)
				return
			}
			respServerError(c, "Can not load %v, error: %v", resourceType, resp.Error)
			return
		}

		respSuccess(c, fileData{
			Filename: resp.Filename,
			Data:     string(resp.RawData),
			Size:     resp.Size,
		})
	}

}

func (h *Handler) submissionTestResourceGetter(resourceType resource.Type) func(c *gin.Context) {
	return func(c *gin.Context) {
		submission, ok := h.findSubmission(c)
		if !ok {
			return
		}

		problem, ok := h.findProblemByID(c, submission.ProblemID)

		testID, ok := h.getProblemTestID(c, problem)
		if !ok {
			return
		}

		resp := h.base.StorageConnection.Download(&storageconn.Request{
			Resource:      resourceType,
			SubmitID:      uint64(submission.ID),
			TestID:        testID,
			DownloadBytes: true,
			DownloadHead:  pointer.Int64(h.config.LoadFilesHead),
			Ctx:           c,
		})
		if resp.Error != nil {
			if errors.Is(resp.Error, storageconn.ErrStorageFileNotFound) {
				respError(c, http.StatusNotFound, "%v for submission %d test %d does not exist", resourceType, submission.ID, testID)
				return
			}
			respServerError(c, "Can not load submission %d %v, error: %v", submission.ID, resourceType, resp.Error)
			return
		}

		respSuccess(c, fileData{
			Filename: resp.Filename,
			Data:     string(resp.RawData),
			Size:     resp.Size,
		})
	}
}

func (h *Handler) addSubmission(c *gin.Context) {
	language, ok := c.GetPostForm("language")
	if !ok {
		respError(c, http.StatusBadRequest, "Can not parse language")
		return
	}
	problem, ok := h.findProblem(c, c.PostForm("problem_id"))
	if !ok {
		return
	}

	var reader io.Reader
	var filename string
	file, err := c.FormFile("solution")

	if err == nil {
		fd, err := file.Open()
		if err != nil {
			respError(c, http.StatusBadRequest, "Can not parse solution")
			return
		}
		defer fd.Close()
		reader = fd
		filename = file.Filename
	} else {
		if !errors.Is(err, http.ErrMissingFile) {
			respError(c, http.StatusBadRequest, "Can not parse solution file")
			return
		}
		solutionBytes, ok := c.GetPostForm("solution_text")
		if !ok {
			respError(c, http.StatusBadRequest, "No solution file or text provided")
			return
		}
		reader = strings.NewReader(solutionBytes)
		filename = fmt.Sprintf("solution.%s", language)
	}

	submissionID, err := h.base.MasterConnection.SendNewSubmission(c, problem.ID, language, filename, reader)
	if err != nil {
		respServerError(c, "Can not send new submission, error: %v", err)
		return
	}
	respSuccess(c, submissionID)
}

type submissionIDHolder struct {
	ID uint `uri:"id" binding:"required"`
}

func (h *Handler) findSubmission(c *gin.Context) (*models.Submission, bool) {
	submitID := new(submissionIDHolder)
	err := c.ShouldBindUri(&submitID)
	if err != nil {
		respError(c, http.StatusBadRequest, "%v", err)
		return nil, false
	}

	submission := new(models.Submission)
	err = h.masterStatus.GetSubmission(c, submitID.ID, submission)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			respError(c, http.StatusNotFound, "Submission with id %d not found", submitID.ID)
		} else {
			respServerError(c, "Can not load submission %d, error: %v", submitID.ID, err)
		}
		return nil, false
	}
	return submission, true
}
