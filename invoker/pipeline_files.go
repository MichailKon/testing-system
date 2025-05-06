package invoker

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/constants/resource"
	"testing_system/invoker/storage"
	"testing_system/lib/logger"
	"time"
)

const (
	solutionBinaryFile     = "solution"
	compileScriptFile      = "compile.sh"
	compilationMessageFile = "compile_message.txt"
	testInputFile          = "input.txt"
	testOutputFile         = "output.txt"
	testErrorFile          = "stderr.txt"
	testAnswerFile         = "answer.txt"
	checkerBinaryFile      = "check"
	checkResultFile        = "check_result.xml"
	checkOutputFile        = "checker_output.txt"
)

func (s *JobPipelineState) loadSolutionBinary() error {
	binary, err := s.loadResource(s.invoker.Storage.Binary, uint64(s.job.submission.ID))
	if err != nil {
		return fmt.Errorf("can not get solution binary, error: %v", err)
	}
	err = s.copyFileToSandbox(*binary, solutionBinaryFile, 0755)
	if err != nil {
		return fmt.Errorf("can not copy solution binary to sandbox, error: %v", err)
	}
	logger.Trace("Loaded solution binary to sandbox for %s", s.loggerData)
	return nil
}

func (s *JobPipelineState) loadTestInput() error {
	testInput, err := s.loadResource(s.invoker.Storage.TestInput, uint64(s.job.problem.ID), s.job.Test)
	if err != nil {
		return fmt.Errorf("can not get test input, error: %v", err)
	}
	err = s.copyFileToSandbox(*testInput, testInputFile, 0644)
	if err != nil {
		return fmt.Errorf("can not copy test input to sandbox, error: %v", err)
	}
	logger.Trace("Loaded test input to sandbox for %s", s.loggerData)
	return nil
}

func (s *JobPipelineState) loadCheckerBinaryFile() error {
	checker, err := s.loadResource(s.invoker.Storage.Checker, uint64(s.job.problem.ID))
	if err != nil {
		return fmt.Errorf("can not get checker binary, error: %v", err)
	}
	err = s.copyFileToSandbox(*checker, checkerBinaryFile, 0755)
	if err != nil {
		return fmt.Errorf("can not copy checker binary to sandbox, error: %v", err)
	}
	logger.Trace("Loaded checker binary to sandbox for %s", s.loggerData)
	return nil
}

func (s *JobPipelineState) loadTestAnswerFile() error {
	testAnswer, err := s.loadResource(s.invoker.Storage.TestAnswer, uint64(s.job.problem.ID), s.job.Test)
	if err != nil {
		return fmt.Errorf("can not get test answer, error: %s", err.Error())
	}
	err = s.copyFileToSandbox(*testAnswer, testAnswerFile, 0644)
	if err != nil {
		return fmt.Errorf("can not copy test answer to sandbox, error: %s", err.Error())
	}
	logger.Trace("Loaded test answer to sandbox for %s", s.loggerData)
	return nil
}

func (s *JobPipelineState) loadSolutionSourceFile() error {
	source, err := s.loadResource(s.invoker.Storage.Source, uint64(s.job.submission.ID))
	if err != nil {
		return fmt.Errorf("can not get submission source, error: %v", err)
	}
	if s.compile == nil {
		return fmt.Errorf("can not save solution source, pipeline compile field not initialized")
	}
	s.compile.sourceName = "source_" + filepath.Base(*source)
	err = s.copyFileToSandbox(*source, s.compile.sourceName, 0644)
	if err != nil {
		return fmt.Errorf("can not copy submission source to sandbox, error: %v", err)
	}
	logger.Trace("Loaded source to sandbox for %s", s.loggerData)
	return nil
}

func (s *JobPipelineState) uploadBinary() error {
	reader, err := s.openSandboxFile(solutionBinaryFile, false)
	if err != nil {
		return fmt.Errorf("can not open solution binary file, error: %v", err)
	}
	binaryStoreRequest := &storageconn.Request{
		Resource: resource.CompiledBinary,
		SubmitID: uint64(s.job.submission.ID),
		File:     reader,
	}
	resp := s.uploadResource(binaryStoreRequest)
	if resp.Error != nil {
		return fmt.Errorf("can not send solution binary file to storage, error: %v", resp.Error)
	}
	logger.Trace("Uploaded solution binary file to storage for %s", s.loggerData)
	return nil
}

func (s *JobPipelineState) uploadOutput(fileName string, resourceType resource.Type) error {
	reader, err := s.openSandboxFile(fileName, true)
	if err != nil {
		return fmt.Errorf("can not open %v file, error: %v", resourceType, err)
	}

	outputStoreRequest := &storageconn.Request{
		Resource: resourceType,
		SubmitID: uint64(s.job.submission.ID),
		TestID:   s.job.Test,
		File:     reader,
	}
	resp := s.uploadResource(outputStoreRequest)
	if resp.Error != nil {
		return fmt.Errorf("can not upload %v file to storage, error: %v", resourceType, resp.Error)
	}
	logger.Trace("Sent %v file to storage for %s", resourceType, s.loggerData)
	return nil
}

func (s *JobPipelineState) copyFileToSandbox(src string, dst string, perm os.FileMode) error {
	defer updateMetrics(&s.metrics.FileActionsDuration, time.Now())
	srcReader, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcReader.Close()
	dstWriter, err := os.OpenFile(filepath.Join(s.sandbox.Dir(), dst), os.O_RDWR|os.O_CREATE|os.O_TRUNC|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	defer dstWriter.Close()
	_, err = io.Copy(dstWriter, srcReader)

	return nil
}

func (s *JobPipelineState) openSandboxFile(fileName string, limit bool) (io.Reader, error) {
	file, err := os.Open(filepath.Join(s.sandbox.Dir(), fileName))
	if err != nil {
		return nil, err
	}
	s.defers = append(s.defers, func() { file.Close() })
	if limit {
		return s.limitedReader(file), nil
	}
	return file, nil
}

func (s *JobPipelineState) limitedReader(r io.Reader) io.Reader {
	if s.invoker.TS.Config.Invoker.SaveOutputHead == nil {
		return r
	} else {
		return io.LimitReader(r, int64(*s.invoker.TS.Config.Invoker.SaveOutputHead))
	}
}

func (s *JobPipelineState) loadResource(getter *storage.CacheGetter, args ...uint64) (*string, error) {
	defer updateMetrics(&s.metrics.ResourceWaitDuration, time.Now())
	res, err := getter.Get(args...)
	return res, err
}

func (s *JobPipelineState) uploadResource(request *storageconn.Request) *storageconn.Response {
	defer updateMetrics(&s.metrics.SendResultDuration, time.Now())
	resp := s.invoker.TS.StorageConn.Upload(request)
	return resp
}
