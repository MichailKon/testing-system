package storageconn

//go:generate stringer -type=ResourceType

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing_system/lib/logger"
)

type ResourceType int

const (
	SourceCode ResourceType = iota
	CompiledBinary
	Test
	Checker
	Interactor
	// Will be increased
)

type Request struct {
	// Should be always specified
	Resource ResourceType `json:"resource"`

	// If resource is part of problem, ProblemID is used
	ProblemID uint64 `json:"problemID"`
	// If resource is part of submit, SubmitID is used
	SubmitID uint64 `json:"submitID"`
	// If resource is a test, TestID should be specified
	TestID uint64 `json:"testID"`

	// For any download, BaseFolder should be specified. The files with original filenames will be placed there
	BaseFolder string `json:"-"`

	// For uploads, FilePath or FilePaths should be specified (depending on whether the resource is single-file or not).
	// Filename will be taken from filename inside the path.
	FilePath  string   `json:"-"`
	FilePaths []string `json:"-"`
}

func (s *Request) FillBaseFolder(parent string) {
	s.BaseFolder = filepath.Join(parent, s.Resource.String())
	switch s.Resource {
	case SourceCode, CompiledBinary:
		s.BaseFolder = filepath.Join(s.BaseFolder, strconv.FormatUint(s.SubmitID, 10))
	case Checker, Interactor:
		s.BaseFolder = filepath.Join(s.BaseFolder, strconv.FormatUint(s.ProblemID, 10))
	case Test:
		s.BaseFolder = filepath.Join(s.BaseFolder, fmt.Sprintf("%d-%d", s.SubmitID, s.TestID))
	default:
		logger.Panic("Can not fill base folder for storageconn request of type %s", s.Resource)
	}
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
