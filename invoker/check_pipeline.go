package invoker

import (
	"encoding/xml"
	"errors"
	"fmt"
	"golang.org/x/net/html/charset"
	"os"
	"path/filepath"
	"strings"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/constants/resource"
	"testing_system/common/constants/verdict"
	"testing_system/invoker/sandbox"
	"testing_system/lib/logger"
)

func (s *JobPipelineState) fullCheckPipeline() error {
	err := s.loadCheckerBinaryFile()
	if err != nil {
		return err
	}

	err = s.loadTestAnswerFile()
	if err != nil {
		return err
	}

	err = s.generateCheckerRunConfig()
	if err != nil {
		return err
	}

	err = s.executeCheckerRunCommand()
	if err != nil {
		return err
	}

	err = s.parseCheckerResult()
	if err != nil {
		return err
	}
	return nil
}

func (s *JobPipelineState) generateCheckerRunConfig() error {
	s.test.checkConfig = &sandbox.ExecuteConfig{
		RunLimitsConfig: *s.invoker.TS.Config.Invoker.CheckerLimits,
	}

	s.test.checkConfig.Command = checkerBinaryFile
	s.test.checkConfig.Args = []string{
		testInputFile, testOutputFile, testAnswerFile, checkResultFile, checkResultFileArg,
	}
	logger.Trace("Generated checker run config for %s", s.loggerData)
	return nil
}

func (s *JobPipelineState) executeCheckerRunCommand() error {
	s.executeWaitGroup.Add(1)
	s.invoker.RunQueue <- s.runChecker
	s.executeWaitGroup.Wait()

	if s.test.checkResult.Err != nil {
		return fmt.Errorf("can not run checker in sandbox, error: %v", s.test.checkResult.Err)
	}

	switch s.test.checkResult.Verdict {
	case verdict.OK, verdict.RT:
		logger.Trace("Finished checker run for %s", s.loggerData)
		return nil
	case verdict.TL:
		return fmt.Errorf("checker running took more than %v time", s.test.checkConfig.TimeLimit)
	case verdict.ML:
		return fmt.Errorf("checker running took more than %v memory", s.test.checkConfig.MemoryLimit)
	case verdict.WL:
		return fmt.Errorf("checker running took more than %v wall time", s.test.checkConfig.WallTimeLimit)
	case verdict.SE:
		return fmt.Errorf("checker security violation")
	default:
		return fmt.Errorf("unknown checker sandbox run verdict: %s", s.test.checkResult.Verdict)
	}
}

func (s *JobPipelineState) runChecker() {
	s.test.checkResult = s.sandbox.Run(s.test.checkConfig)
	s.executeWaitGroup.Done()
}

func (s *JobPipelineState) parseCheckerResult() error {
	_, err := os.Stat(filepath.Join(s.sandbox.Dir(), checkResultFile))
	if err == nil {
		return s.parseTestlibCheckerResult()
	} else if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("no checker result file found, non testlib checkers are not supported")
	} else {
		return fmt.Errorf("can not stat checker result file")
	}
}

func (s *JobPipelineState) parseTestlibCheckerResult() error {
	checkResultReader, err := os.Open(filepath.Join(s.sandbox.Dir(), checkResultFile))
	if err != nil {
		return fmt.Errorf(
			"checker exited with exit code %d, can not parse checker result xml file in appes mode",
			s.test.checkResult.Statistics.ExitCode,
		)
	}
	defer checkResultReader.Close()

	var checkerResult CheckerResultXML
	xmlReader := xml.NewDecoder(checkResultReader)
	xmlReader.CharsetReader = charset.NewReaderLabel
	err = xmlReader.Decode(&checkerResult)
	if err != nil {
		return fmt.Errorf(
			"can not parse checker result xml file in appes mode: %s",
			err.Error(),
		)
	}

	s.test.checkerOutputReader = s.limitedReader(strings.NewReader(checkerResult.Value))
	switch checkerResult.Outcome {
	case "accepted":
		s.test.runResult.Verdict = verdict.OK
	case "wrong-answer", "presentation-error", "unexpected-eof": // We will treat PE as WA as all testing systems now do
		s.test.runResult.Verdict = verdict.WA
	case "fail":
		// Only in case of checker CF verdict we accept job as successful
		s.test.runResult.Verdict = verdict.CF
	case "points", "relative-scoring":
		if checkerResult.Points == nil {
			return fmt.Errorf(
				"checker exited with exit code %d and verdict %s, but no points specified",
				s.test.checkResult.Statistics.ExitCode,
				checkerResult.Outcome,
			)
		} else {
			s.test.runResult.Verdict = verdict.PT
			s.test.runResult.Points = checkerResult.Points
		}
	default:
		return fmt.Errorf(
			"unknown checker verdict %s, checker exited with exit code %d",
			checkerResult.Outcome,
			s.test.checkResult.Statistics.ExitCode,
		)
	}

	logger.Trace(
		"Parsed testlib checker result for %s, checker verdict is %s",
		s.loggerData, s.test.checkResult.Verdict,
	)
	return nil
}

func (s *JobPipelineState) uploadTestRunResources() error {
	err := s.uploadOutput(testOutputFile, resource.TestOutput)
	if err != nil {
		return err
	}

	err = s.uploadOutput(testErrorFile, resource.TestStderr)
	if err != nil {
		return err
	}

	err = s.uploadCheckerOutput()
	if err != nil {
		return err
	}
	return nil
}

func (s *JobPipelineState) uploadCheckerOutput() error {
	checkerOutputRequest := &storageconn.Request{
		Resource: resource.CheckerOutput,
		SubmitID: uint64(s.job.Submission.ID),
		TestID:   s.job.Test,
		File:     s.test.checkerOutputReader,
	}
	resp := s.invoker.TS.StorageConn.Upload(checkerOutputRequest)
	if resp.Error != nil {
		return fmt.Errorf("can not upload checker output to storage, error: %s", resp.Error.Error())
	}
	logger.Trace("Uploaded checker output for %s", s.loggerData)
	return nil
}

type CheckerResultXML struct {
	Outcome string   `xml:"outcome,attr"`
	Points  *float64 `xml:"points,attr,omitempty"`
	Value   string   `xml:",chardata"`
}
