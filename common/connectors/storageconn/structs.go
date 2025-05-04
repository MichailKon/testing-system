package storageconn

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing_system/common/constants/resource"
	"testing_system/lib/logger"
)

type Request struct {
	// Should be always specified
	Resource resource.Type `json:"resource"`

	/*
		Resource must always have exactly one of ProblemId and SubmitID
		Including, only TestID cannot be specified
		ID=0 is considered absent
	*/

	// If resource is part of problem, ProblemID is used
	ProblemID uint64 `json:"problemID"`
	// If resource is part of submit, SubmitID is used
	SubmitID uint64 `json:"submitID"`
	// If resource is a test, TestID should be specified
	TestID uint64 `json:"testID"`

	// For any download, either a new file will be created or the file will be downloaded as []byte
	DownloadBytes bool `json:"-"`

	// For any download, if DownloadBytes == false, the DownloadFolder should be specified
	DownloadFolder string `json:"-"`

	// Specify a custom filename for the downloaded file. Can be empty
	DownloadFilename *string `json:"-"`

	// For downloads, DownloadHead can be specified so that only first DownloadHead bytes will be loaded
	DownloadHead *int64 `json:"downloadHead"`

	// For uploads, File should be specified
	File io.Reader `json:"-"`

	// If StorageFilename is not specified, Storage tries to get the filename automatically
	StorageFilename string `json:"storageFilename"`

	// Context may be specified for requests
	Ctx context.Context `json:"-"`
}

type Response struct {
	R     Request
	Error error
}

type FileResponse struct {
	Response
	IsBytesArray bool   `json:"isFile"`
	RawData      []byte `json:"rawData"`

	Filename   string `json:"filename"`
	BaseFolder string `json:"baseFolder"`
	Size       uint64 `json:"size"`
}

func NewFileResponse(request Request) *FileResponse {
	return &FileResponse{
		Response:   Response{R: request, Error: nil},
		Filename:   "",
		BaseFolder: "",
		Size:       0,
	}
}

func (r *FileResponse) GetFilePath() (string, bool) {
	if r.BaseFolder == "" || r.Filename == "" || !r.IsBytesArray {
		return "", false
	}
	return filepath.Join(r.BaseFolder, r.Filename), true
}

// CleanUp Removes BaseFolder with all files
func (r *FileResponse) CleanUp() {
	if r.Error != nil {
		logger.Error("CleanUp was called for failed FileResponse: %v", r.Error)
		return
	}
	if r.BaseFolder == "" {
		logger.Error("CleanUp was called for empty BaseFolder name")
		return
	}
	if r.IsBytesArray {
		logger.Error("CleanUp was called for file downloaded as []bytes")
		return
	}

	err := os.RemoveAll(r.BaseFolder)
	if err != nil {
		logger.Error("Cannot remove resource folder %s: %s", r.BaseFolder, err.Error())
	}
}
