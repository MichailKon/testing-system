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

	// For uploads, Files should be specified. Filename is key in map, file data should be read from value
	Files map[string]io.Reader `json:"-"`
}

type Response struct {
	R     Request
	Error error
}

type ResponseFiles struct {
	Response
	fileNames []string
	Size      uint64
}

func (r *ResponseFiles) File() (string, bool) {
	if len(r.fileNames) == 0 {
		return "", false
	}
	return filepath.Join(r.R.BaseFolder, r.fileNames[0]), true
}

func (r *ResponseFiles) Get(fileName string) (string, bool) {
	for _, f := range r.fileNames {
		if fileName == f {
			return filepath.Join(r.R.BaseFolder, f), true
		}
	}
	return "", false
}

func (r *ResponseFiles) CleanUp() {
	if r.Error != nil {
		return
	}
	if len(r.R.BaseFolder) == 0 {
		return
	}
	err := os.RemoveAll(r.R.BaseFolder)
	if err != nil {
		logger.Panic("Can not remove resource folder, error: %s", err.Error())
	}
}
