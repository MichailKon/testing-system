package invoker

import (
	"fmt"
	"testing_system/common/config"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/invoker/sandbox"
	"testing_system/lib/logger"
)

const (
	checkResultFileArg = "-appes"
)

func (i *Invoker) fullTestingPipeline(sandbox sandbox.ISandbox, job *Job) {
	s := i.newPipeline(sandbox, job)
	s.test = new(pipelineTestData)
	s.loggerData = fmt.Sprintf(
		"test job: %s submission: %d problem %d test %d",
		job.ID,
		job.submission.ID,
		job.problem.ID,
		job.Test,
	)
	defer s.checkFinish()

	logger.Trace("Starting testing for %s", s.loggerData)

	err := s.testingProcessPipeline()
	if err != nil {
		logger.Error("Error in %s error: %v", s.loggerData, err)
		s.failJob("job %s error: %v", job.ID, err)
		return
	}

	if s.test.hasResources {
		err = s.uploadTestRunResources()
		if err != nil {
			logger.Error("Error in %s error: %v", s.loggerData, err)
			s.failJob("job %s error: %v", job.ID, err)
			return
		}
	}

	s.successJob(s.test.runResult)
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

	err = s.fullCheckPipeline()
	if err != nil {
		return err
	}

	return nil
}

func (s *JobPipelineState) generateTestRunConfig() error {
	s.test.runConfig = new(sandbox.ExecuteConfig)
	fillInTestRunConfigLimits(s.test.runConfig, s.job.problem)

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
	s.runProcess(s.runSolution)
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
