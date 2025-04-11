package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/constants/resource"

	"github.com/gin-gonic/gin"
)

type resourceInfo struct {
	id       uint64
	filepath string
	dataType resource.DataType
}

func getInfoFromRequest(c *gin.Context) (*resourceInfo, error) {
	request, err := parseRequest(c)

	if err != nil {
		return nil, fmt.Errorf("unavailable to get request: %v", err)
	}

	info := &resourceInfo{}

	info.dataType, err = getDataType(request)
	if err != nil {
		return nil, err
	}

	info.id, err = getDataId(info.dataType, request)
	if err != nil {
		return nil, err
	}

	info.filepath, err = getFilepath(request)
	if err != nil {
		return nil, err
	}

	return info, nil
}

func parseRequest(c *gin.Context) (*storageconn.Request, error) {
	var requestJSON string

	switch c.Request.Method {
	case http.MethodGet, http.MethodDelete:
		requestJSON = c.Query("request")
		if requestJSON == "" {
			return nil, errors.New("missing request parameter in query")
		}
	case http.MethodPost, http.MethodPut:
		requestJSON = c.PostForm("request")
		if requestJSON == "" {
			return nil, errors.New("missing request parameter in form data")
		}
	default:
		return nil, errors.New("unsupported HTTP method")
	}

	req := storageconn.Request{}
	err := json.Unmarshal([]byte(requestJSON), &req)
	if err != nil {
		return nil, errors.New("invalid request format: " + err.Error())
	}

	return &req, nil
}

func getDataType(request *storageconn.Request) (resource.DataType, error) {
	switch request.Resource {
	case resource.Checker, resource.Interactor:
		return resource.Problem, nil
	case resource.TestInput, resource.TestAnswer:
		return resource.Problem, nil
	case resource.SourceCode, resource.CompiledBinary, resource.CompileOutput:
		return resource.Submission, nil
	case resource.TestOutput, resource.TestStderr, resource.CheckerOutput:
		return resource.Submission, nil
	default:
		return resource.UnknownDataType, errors.New("unknown resource type")
	}
}

func getDataId(dataType resource.DataType, request *storageconn.Request) (uint64, error) {
	switch dataType {
	case resource.Problem:
		if request.ProblemID == 0 {
			return 0, errors.New("ProblemID is not specified for promlem resource")
		}
		return request.ProblemID, nil
	case resource.Submission:
		if request.SubmitID == 0 {
			return 0, errors.New("SubmitID is not specified for submission resource")
		}
		return request.SubmitID, nil
	default:
		return 0, errors.New("unavailable to get data id")
	}
}

func getFilepath(request *storageconn.Request) (string, error) {
	if request.StorageFilename == "" {
		return "", errors.New("StorageFilename is not specified")
	}
	switch request.Resource {
	case resource.TestInput, resource.TestOutput, resource.TestAnswer, resource.TestStderr:
		if request.TestID == 0 {
			return "", errors.New("TestID is not specified for test resource")
		}
		return fmt.Sprintf("tests/%s/%s", strconv.FormatUint(request.TestID, 10), request.StorageFilename), nil
	default:
		return fmt.Sprintf("%s", request.StorageFilename), nil
	}
}
