package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing_system/common/connectors/storageconn"
	"testing_system/storage/filesystem"

	"github.com/gin-gonic/gin"
)

func getInfoFromRequest(c *gin.Context) (*filesystem.ResourceInfo, error) {
	request, err := parseRequest(c)

	if err != nil {
		return nil, fmt.Errorf("unavailable to parse request: %v", err)
	}

	resourseInfo := &filesystem.ResourceInfo{Request: request}

	err = resourseInfo.ParseDataType()

	if err != nil {
		return nil, fmt.Errorf("unavailable to parse dataType: %v", err)
	}

	err = resourseInfo.ParseDataID()

	if err != nil {
		return nil, fmt.Errorf("unavailable to parse dataID: %v", err)
	}

	err = resourseInfo.ParseFilepath()

	if err != nil {
		return nil, fmt.Errorf("unavailable to parse filepath: %v", err)
	}

	return resourseInfo, nil
}

func parseRequest(c *gin.Context) (*storageconn.Request, error) {
	var requestJSON string

	switch c.Request.Method {
	case http.MethodGet:
		requestJSON = c.Query("request")
		if requestJSON == "" {
			return nil, errors.New("missing request parameter in query")
		}
	case http.MethodPost, http.MethodPut, http.MethodDelete:
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
