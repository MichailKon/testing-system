package storageconn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"testing_system/common/config"
	"testing_system/common/connectors"
	"testing_system/lib/connector"
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

	if !request.DownloadBytes {
		if err := os.MkdirAll(request.DownloadFolder, 0775); err != nil {
			response.Error = fmt.Errorf("failed to create base folder: %v", err)
			return response
		}
	}

	r := s.connection.R()

	if request.Ctx != nil {
		r.SetContext(request.Ctx)
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		response.Error = fmt.Errorf("failed to form request to storage: %v", err)
		return response
	}

	r.SetQueryParams(map[string]string{
		"request": string(requestJSON),
	})
	r.SetDoNotParseResponse(true)

	resp, err := r.Get("/storage/get")
	if err != nil {
		response.Error = fmt.Errorf("failed to send request: %v", err)
		return response
	}
	defer resp.RawBody().Close()

	if resp.StatusCode() != http.StatusOK {
		if resp.StatusCode() == http.StatusNotFound {
			response.Error = ErrStorageFileNotFound
		} else {
			body, err := io.ReadAll(resp.RawBody())
			if err != nil {
				response.Error = &connector.Error{
					Code:    resp.StatusCode(),
					Message: err.Error(),
					Path:    resp.Request.URL,
				}
			}
			response.Error = connector.ParseRespError(body, resp)
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

	var written int64
	if request.DownloadBytes {
		buf := &bytes.Buffer{}
		written, err = io.Copy(buf, resp.RawBody())
		if err != nil {
			response.Error = fmt.Errorf("failed to load file as []byte, error: %v", err)
			return response
		}
		response.IsBytesArray = true
		response.RawData = buf.Bytes()
	} else {
		filePath := filepath.Join(request.DownloadFolder, filename)
		file, err := os.Create(filePath)
		if err != nil {
			response.Error = fmt.Errorf("failed to create file: %v", err)
			return response
		}
		defer file.Close()

		written, err = io.Copy(file, resp.RawBody())
		if err != nil {
			response.Error = fmt.Errorf("failed to write to file: %v", err)
			return response
		}
		response.BaseFolder = request.DownloadFolder
	}

	response.Filename = filename
	response.Size = uint64(written)
	return response
}

func (s *Connector) Upload(request *Request) *Response {
	response := &Response{R: *request}

	if request.File == nil {
		response.Error = fmt.Errorf("file for upload is not specified")
		return response
	}

	r := s.connection.R()
	if request.Ctx != nil {
		r.SetContext(request.Ctx)
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		response.Error = fmt.Errorf("failed to form request to storage: %v", err)
		return response
	}

	r.SetFormData(map[string]string{
		"request": string(requestJSON),
	})

	requestFileName := request.StorageFilename
	// request.StorageFilename can be empty but http requires filename to be specified
	if requestFileName == "" {
		requestFileName = "noname"
	}
	r.SetFileReader("file", requestFileName, request.File)

	response.Error = connector.ReceiveEmpty(r, "/storage/upload", resty.MethodPost)
	return response
}

func (s *Connector) Delete(request *Request) *Response {
	response := &Response{R: *request}

	r := s.connection.R()
	if request.Ctx != nil {
		r.SetContext(request.Ctx)
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		response.Error = fmt.Errorf("failed to form request to storage: %v", err)
		return response
	}

	r.SetFormData(map[string]string{
		"request": string(requestJSON),
	})

	response.Error = connector.ReceiveEmpty(r, "/storage/remove", resty.MethodDelete)

	return response
}
