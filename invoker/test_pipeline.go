package invoker

import (
	"encoding/xml"
	"errors"
	"fmt"
	"golang.org/x/net/html/charset"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing_system/common/config"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/constants/resource"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/invoker/sandbox"
	"testing_system/lib/logger"
)

const (
	checkResultFileArg = "-appes"
)

func (i *Invoker) fullTestingPipeline(sandbox sandbox.ISandbox, job *Job) {
	s := &JobPipelineState{
		job:     job,
		invoker: i,
		sandbox: sandbox,
		test:    new(pipelineTestData),
		loggerData: fmt.Sprintf(
			"test job: %s submission: %d problem %d test %d",
			job.ID,
			job.Submission.ID,
			job.Problem.ID,
			job.Test,
		),
	}

	logger.Trace("Starting testing for %s", s.loggerData)

	s.defers = append(s.defers, job.deferFunc)
	defer s.deferFunc()

	err := s.testingProcessPipeline()
	if err != nil {
		logger.Error("Error in %s error: %v", s.loggerData, err)
		i.failJob(job, "job %s error: %v", job.ID, err)
		return
	}

	if s.test.hasResources {
		err = s.uploadTestRunResources()
		if err != nil {
			logger.Error("Error in %s error: %v", s.loggerData, err)
			i.failJob(job, "job %s error: %v", job.ID, err)
			return
		}
	}

	i.successJob(job, s.test.runResult)
}

func (s *JobPipelineState) testingProcessPipeline() error {
	err := s.initSandbox()
	if err != nil {
		return err
	}

	err = s.loadSolutionBinary()
	if err != nil {
		return err
	}

	err = s.loadTestInput()
	if err != nil {
		return err
	}

	err = s.generateTestRunConfig()
	if err != nil {
		return err
	}

	err = s.executeTestRunCommand()
	if err != nil {
		return err
	}

	if s.test.runResult.Verdict != verdict.OK {
		s.test.hasResources = false
		return nil
	}
	s.test.hasResources = true

	err = s.loadCheckerBinaryFile()
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

func (s *JobPipelineState) generateTestRunConfig() error {
	s.test.runConfig = new(sandbox.ExecuteConfig)
	fillInTestRunConfigLimits(s.test.runConfig, s.job.Problem)

	s.test.runConfig.Command = solutionBinaryFile
	s.test.runConfig.Stdin = &sandbox.IORedirect{FileName: testInputFile}
	s.test.runConfig.Stdout = &sandbox.IORedirect{FileName: testOutputFile}
	s.test.runConfig.Stderr = &sandbox.IORedirect{FileName: testErrorFile}

	// TODO: support interactive problems

	logger.Trace("Generated test run config for %s", s.loggerData)
	return nil
}

func fillInTestRunConfigLimits(c *sandbox.ExecuteConfig, problem *models.Problem) {
	c.RunLimitsConfig = config.RunLimitsConfig{
		TimeLimit:   problem.TimeLimit,
		MemoryLimit: problem.MemoryLimit,
	}

	if problem.WallTimeLimit != nil {
		c.WallTimeLimit = *problem.WallTimeLimit
	} else {
		c.WallTimeLimit.FromStr("5s")
		if c.WallTimeLimit < c.TimeLimit*2 {
			c.WallTimeLimit = c.TimeLimit * 2
		}
	}

	if problem.MaxOpenFiles != nil {
		c.MaxOpenFiles = *problem.MaxOpenFiles
	} else {
		c.MaxOpenFiles = 64
	}

	if problem.MaxThreads != nil {
		c.MaxThreads = *problem.MaxThreads
	} else {
		c.MaxThreads = 0
	}

	if problem.MaxOutputSize != nil {
		c.MaxOutputSize = *problem.MaxOutputSize
	} else {
		c.MaxOutputSize.FromStr("1g")
	}
}

func (s *JobPipelineState) executeTestRunCommand() error {
	s.executeWaitGroup.Add(1)
	s.invoker.RunQueue <- s.runSolution
	s.executeWaitGroup.Wait()

	if s.test.runResult.Err != nil {
		return fmt.Errorf("can not run solution in sandbox, error: %v", s.test.runResult.Err)
	}
	logger.Trace("Finished test run for %s with verdict %s", s.loggerData, s.test.runResult.Verdict)
	return nil
}

func (s *JobPipelineState) runSolution() {
	s.test.runResult = s.sandbox.Run(s.test.runConfig)
	s.executeWaitGroup.Done()
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
		Files: map[string]io.Reader{
			checkOutputFile: s.test.checkerOutputReader,
		},
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
