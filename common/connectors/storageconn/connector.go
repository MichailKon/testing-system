package storageconn

import (
	"fmt"
	"os"
	"path/filepath"
	"testing_system/common/config"
	"testing_system/common/connectors"
	"testing_system/common/constants/resource"

	"github.com/go-resty/resty/v2"
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

func (s *Connector) Download(request *Request) *ResponseFiles {
	if err := os.MkdirAll(request.BaseFolder, 0775); err != nil {
		return &ResponseFiles{Response: Response{R: *request, Error: fmt.Errorf("failed to create base folder: %v", err)}}
	}

	path := fmt.Sprintf("/storage/get?id=%d&dataType=%s&filepath=%s",
		getIDForResource(request),
		getDataTypeForResource(request.Resource),
		request.Resource.String(),
	)

	r := s.connection.R()
	resp, err := r.Execute(resty.MethodGet, path)
	if err != nil {
		return &ResponseFiles{Response: Response{R: *request, Error: fmt.Errorf("failed to send request: %v", err)}}
	}

	if resp.IsError() {
		return &ResponseFiles{Response: Response{R: *request, Error: fmt.Errorf("request failed with status: %v", resp.Status())}}
	}

	filename := request.Resource.String()
	if request.CustomFilename != "" {
		filename = request.CustomFilename
	}

	filepath := filepath.Join(request.BaseFolder, filename)
	err = os.WriteFile(filepath, resp.Body(), 0644)
	if err != nil {
		return &ResponseFiles{Response: Response{R: *request, Error: fmt.Errorf("failed to write file: %v", err)}}
	}

	responseFiles := NewResponseFiles(*request)
	responseFiles.fileNames = []string{filename}
	responseFiles.Size = uint64(len(resp.Body()))
	return responseFiles
}

func (s *Connector) Upload(request *Request) *Response {
	if len(request.Files) == 0 {
		return &Response{R: *request, Error: fmt.Errorf("no files to upload")}
	}

	path := fmt.Sprintf("/storage/upload?id=%d&dataType=%s&filepath=%s",
		getIDForResource(request),
		getDataTypeForResource(request.Resource),
		request.Resource.String(),
	)

	r := s.connection.R()
	for filename, reader := range request.Files {
		r.SetFileReader("file", filename, reader)
	}

	resp, err := r.Execute(resty.MethodPost, path)
	if err != nil {
		return &Response{R: *request, Error: fmt.Errorf("failed to send request: %v", err)}
	}

	if resp.IsError() {
		return &Response{R: *request, Error: fmt.Errorf("request failed with status: %s, body: %s", resp.Status(), resp.String())}
	}

	return &Response{R: *request}
}

func (s *Connector) Delete(request *Request) *Response {
	path := fmt.Sprintf("/storage/remove?id=%d&dataType=%s&filepath=%s",
		getIDForResource(request),
		getDataTypeForResource(request.Resource),
		request.Resource.String(),
	)

	r := s.connection.R()
	resp, err := r.Execute(resty.MethodDelete, path)
	if err != nil {
		return &Response{R: *request, Error: fmt.Errorf("failed to send request: %v", err)}
	}

	if resp.IsError() {
		return &Response{R: *request, Error: fmt.Errorf("request failed with status: %s, body: %s", resp.Status(), resp.String())}
	}

	return &Response{R: *request}
}

func getIDForResource(request *Request) uint64 {
	switch request.Resource {
	case resource.SourceCode, resource.CompiledBinary, resource.CompileOutput:
		return request.SubmitID
	case resource.Checker, resource.Interactor:
		return request.ProblemID
	case resource.Test:
		if request.TestID > 0 {
			return request.TestID
		}
		return request.ProblemID
	default:
		return 0
	}
}

func getDataTypeForResource(resourceType resource.Type) string {
	switch resourceType {
	case resource.SourceCode, resource.CompiledBinary, resource.CompileOutput:
		return "submission"
	case resource.Checker, resource.Interactor, resource.Test:
		return "problem"
	default:
		return "unknown"
	}
}
