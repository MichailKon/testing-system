package storageconn

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"testing_system/common/config"
	"testing_system/common/connectors"
	"testing_system/common/constants/resource"
)

type Connector struct {
	connection *connectors.ConnectorBase
}

func NewConnector(connection *config.Connection) *Connector {
	if connection == nil {
		return nil
	}
	return &Connector{connectors.NewConnectorBase(connection)}
}

func (s *Connector) Download(request *Request) *FileResponse {
	response := NewFileResponse(*request)

	if err := os.MkdirAll(request.BaseFolder, 0775); err != nil {
		response.Error = fmt.Errorf("failed to create base folder: %v", err)
		return response
	}

	path := "/storage/get"
	r := s.connection.R()

	params, err := getStorageParams(request)

	if err != nil {
		response.Error = fmt.Errorf("failed to form storage request: %v", err)
		return response
	}

	r.SetQueryParams(map[string]string{
		"id":       params.id,
		"dataType": params.dataType,
		"filepath": params.filepath,
	})

	resp, err := r.Get(path)
	if err != nil {
		response.Error = fmt.Errorf("failed to send request: %v", err)
		return response
	}

	if resp.IsError() {
		response.Error = fmt.Errorf("get request failed with status: %v", resp.Status())
		return response
	}

	filename := ""
	if request.CustomFilename != "" {
		filename = request.CustomFilename
	} else {
		// Extract filename from Content-Disposition header
		contentDisposition := resp.Header().Get("Content-Disposition")
		if contentDisposition != "" {
			_, params, err := mime.ParseMediaType(contentDisposition)
			if err == nil && params["filename"] != "" {
				filename = params["filename"]
			}
		}
	}

	if filename == "" {
		response.Error = fmt.Errorf("can't extract filename from CustomFilename or Content-Disposition header")
		return response
	}

	filePath := filepath.Join(request.BaseFolder, filename)
	err = os.WriteFile(filePath, resp.Body(), 0644)
	if err != nil {
		response.Error = fmt.Errorf("failed to write file: %v", err)
		return response
	}

	response.Filename = filename
	response.BaseFolder = request.BaseFolder
	response.Size = uint64(len(resp.Body()))
	return response
}

func (s *Connector) Upload(request *Request) *Response {
	response := &Response{R: *request}

	if request.File == nil {
		response.Error = fmt.Errorf("file for upload is not specified")
		return response
	}

	path := "/storage/upload"
	r := s.connection.R()

	params, err := getStorageParams(request)

	if err != nil {
		response.Error = fmt.Errorf("failed to form storage request: %v", err)
		return response
	}

	r.SetFormData(map[string]string{
		"id":       params.id,
		"dataType": params.dataType,
		"filepath": params.filepath,
	})

	r.SetFileReader("file", request.Filename, request.File)

	resp, err := r.Post(path)
	if err != nil {
		response.Error = fmt.Errorf("failed to send request: %v", err)
		return response
	}

	if resp.IsError() {
		response.Error = fmt.Errorf("upload failed with status: %v", resp.Status())
		return response
	}

	return response
}

func (s *Connector) Delete(request *Request) *Response {
	response := &Response{R: *request}

	path := "/storage/remove"
	r := s.connection.R()

	params, err := getStorageParams(request)

	if err != nil {
		response.Error = fmt.Errorf("failed to form storage request: %v", err)
		return response
	}

	r.SetFormData(map[string]string{
		"id":       params.id,
		"dataType": params.dataType,
		"filepath": params.filepath,
	})

	resp, err := r.Delete(path)
	if err != nil {
		response.Error = fmt.Errorf("failed to send request: %v", err)
		return response
	}

	if resp.IsError() {
		response.Error = fmt.Errorf("delete failed with status: %v", resp.Status())
		return response
	}

	return response
}

type storageParams struct {
	id       string
	dataType string
	filepath string
}

func getStorageParams(request *Request) (storageParams, error) {
	params := storageParams{}
	switch request.Resource {
	case resource.Checker, resource.Interactor:
		params.dataType = "problem"
		params.filepath = request.Resource.String()
		params.id = strconv.FormatUint(request.ProblemID, 10)
		return params, nil
	case resource.SourceCode, resource.CompiledBinary, resource.CompileOutput:
		params.dataType = "submission"
		params.filepath = request.Resource.String()
		params.id = strconv.FormatUint(request.SubmitID, 10)
		return params, nil
	case resource.Test:
		params.dataType = "problem"
		params.filepath = "tests/" + strconv.FormatUint(request.TestID, 10)
		params.id = strconv.FormatUint(request.ProblemID, 10)
		return params, nil
	default:
		return params, fmt.Errorf("unknown resource type: %s", request.Resource.String())
	}
}
