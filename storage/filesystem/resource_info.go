package filesystem

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/constants/resource"
)

type ResourceInfo struct {
	Request              *storageconn.Request
	ID                   uint64
	Filepath             string
	DataType             resource.DataType
	EmptyStorageFilename bool
}

// FilepathFolderMapping should contain all types
var FilepathFolderMapping = map[resource.Type]string{
	resource.SourceCode:     "source",
	resource.CompiledBinary: "",
	resource.CompileOutput:  "",
	resource.TestInput:      "tests",
	resource.TestAnswer:     "tests",
	resource.TestOutput:     "tests",
	resource.TestStderr:     "tests",
	resource.Checker:        "checker",
	resource.CheckerOutput:  "tests",
	resource.Interactor:     "interactor",
}

var FilepathFilenameMapping = map[resource.Type]string{
	resource.CompiledBinary: "solution",
	resource.CompileOutput:  "compile.out",
	resource.TestInput:      "%02d",
	resource.TestAnswer:     "%02d.a",
	resource.TestOutput:     "%02d.out",
	resource.TestStderr:     "%02d.err",
	resource.CheckerOutput:  "%02d.check",
}

func (resourseInfo *ResourceInfo) ParseDataType() error {
	request := resourseInfo.Request

	switch request.Resource {
	case resource.Checker, resource.Interactor:
		resourseInfo.DataType = resource.Problem
		return nil
	case resource.TestInput, resource.TestAnswer:
		resourseInfo.DataType = resource.Problem
		return nil
	case resource.SourceCode, resource.CompiledBinary, resource.CompileOutput:
		resourseInfo.DataType = resource.Submission
		return nil
	case resource.TestOutput, resource.TestStderr, resource.CheckerOutput:
		resourseInfo.DataType = resource.Submission
		return nil
	default:
		return errors.New("unknown resource type")
	}
}

func (resourseInfo *ResourceInfo) ParseDataID() error {
	request := resourseInfo.Request

	switch resourseInfo.DataType {
	case resource.Problem:
		if request.ProblemID == 0 {
			return errors.New("ProblemID is not specified for problem resource")
		}
		resourseInfo.ID = request.ProblemID
		return nil
	case resource.Submission:
		if request.SubmitID == 0 {
			return errors.New("SubmitID is not specified for submission resource")
		}
		resourseInfo.ID = request.SubmitID
		return nil
	default:
		return errors.New("unavailable to get data id")
	}
}

func (resourseInfo *ResourceInfo) ParseFilepath() error {
	filepathFolder, err := resourseInfo.parseFilepathFolder()

	if err != nil {
		return fmt.Errorf("unavailable to parse filepathFolder: %v", err)
	}

	filepathFilename, err := resourseInfo.parseFilepathFilename()

	if err != nil {
		return fmt.Errorf("unavailable to parse filepathFilename: %v", err)
	}

	resourseInfo.Filepath = filepath.Join(filepathFolder, filepathFilename)

	return nil
}

func (resourseInfo *ResourceInfo) parseFilepathFolder() (string, error) {
	request := resourseInfo.Request

	filepathFolder, ok := FilepathFolderMapping[request.Resource]
	if !ok {
		return "", fmt.Errorf("there is no type %s in FilepathFolderMapping", request.Resource.String())
	}

	return filepathFolder, nil
}

func (resourseInfo *ResourceInfo) parseFilepathFilename() (string, error) {
	request := resourseInfo.Request

	if filepathFilename, ok := FilepathFilenameMapping[request.Resource]; ok {
		switch request.Resource {
		case resource.TestInput, resource.TestAnswer, resource.TestOutput, resource.TestStderr, resource.CheckerOutput:
			if request.TestID == 0 {
				return "", errors.New("TestID is not specified for test resource")
			}
			return fmt.Sprintf(filepathFilename, request.TestID), nil
		default:
			return filepathFilename, nil
		}
	}

	if request.StorageFilename != "" {
		return request.StorageFilename, nil
	}
	resourseInfo.EmptyStorageFilename = true

	return "", nil
}
