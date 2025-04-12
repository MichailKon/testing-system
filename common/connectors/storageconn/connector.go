package storageconn

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"testing_system/common/config"
	"testing_system/common/connectors"
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

	requestJSON, err := json.Marshal(request)
	if err != nil {
		response.Error = fmt.Errorf("failed to form request to storage: %v", err)
		return response
	}

	r.SetQueryParams(map[string]string{
		"request": string(requestJSON),
	})

	resp, err := r.SetDoNotParseResponse(true).Execute("GET", path)
	if err != nil {
		response.Error = fmt.Errorf("failed to send request: %v", err)
		return response
	}
	defer resp.RawBody().Close()

	if resp.StatusCode() != http.StatusOK {
		if resp.StatusCode() == http.StatusNotFound {
			response.Error = ErrStorageFileNotFound
		} else {
			response.Error = fmt.Errorf("get request failed with status: %v", resp.Status())
		}
		return response
	}

	var filename string
	if request.DownloadFilename != nil && *request.DownloadFilename != "" {
		filename = *request.DownloadFilename
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
		response.Error = fmt.Errorf("can't extract filename from DownloadFilename or Content-Disposition header")
		return response
	}

	filePath := filepath.Join(request.BaseFolder, filename)
	file, err := os.Create(filePath)
	if err != nil {
		response.Error = fmt.Errorf("failed to create file: %v", err)
		return response
	}
	defer file.Close()

	written, err := io.Copy(file, resp.RawBody())
	if err != nil {
		response.Error = fmt.Errorf("failed to write to file: %v", err)
		return response
	}

	response.Filename = filename
	response.BaseFolder = request.BaseFolder
	response.Size = uint64(written)
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

	requestJSON, err := json.Marshal(request)
	if err != nil {
		response.Error = fmt.Errorf("failed to form request to storage: %v", err)
		return response
	}

	r.SetFormData(map[string]string{
		"request": string(requestJSON),
	})

	// request.StorageFilename can be empty
	r.SetFileReader("file", request.StorageFilename, request.File)

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

	requestJSON, err := json.Marshal(request)
	if err != nil {
		response.Error = fmt.Errorf("failed to form request to storage: %v", err)
		return response
	}

	r.SetFormData(map[string]string{
		"request": string(requestJSON),
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
