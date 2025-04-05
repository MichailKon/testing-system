package storageconn

import (
	"io"
	"os"
	"path/filepath"
	"testing_system/common/constants/resource"
	"testing_system/lib/logger"
)

type Request struct {
	// Should be always specified
	Resource resource.Type `json:"resource"`

	// If resource is part of problem, ProblemID is used
	ProblemID uint64 `json:"problemID"`
	// If resource is part of submit, SubmitID is used
	SubmitID uint64 `json:"submitID"`
	// If resource is a test, TestID should be specified
	TestID uint64 `json:"testID"`

	// For any download, BaseFolder should be specified. The files with original filenames will be placed there
	BaseFolder string `json:"-"`

	// Specify a custom filename for the downloaded file
	CustomFilename string `json:"-"`

	// For uploads, File should be specified
	File io.Reader `json:"-"`

	// For uploads, Filename should be specified
	Filename string `json:"-"`
}

type Response struct {
	R     Request
	Error error
}

type FileResponse struct {
	Response
	Filename   string `json:"filename"`
	BaseFolder string `json:"basefolder"`
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
	if r.BaseFolder == "" || r.Filename == "" {
		return "", false
	}
	return filepath.Join(r.BaseFolder, r.Filename), true
}

func (r *FileResponse) CleanUp() {
	if r.Error != nil {
		return
	}
	if r.BaseFolder == "" {
		return
	}

	// CleanUp should be called after using all needed files
	err := os.RemoveAll(r.BaseFolder)
	if err != nil {
		logger.Error("Cannot remove resource folder %s: %s", r.BaseFolder, err.Error())
	}
}
