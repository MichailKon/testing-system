package invoker

import (
	"fmt"
	"testing_system/common/config"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/invoker/sandbox"
	"testing_system/lib/logger"
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

	err := s.standardTestingPipeline()
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

func (s *JobPipelineState) standardTestingPipeline() error {
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

	err = s.generateStandartTestRunConfig()
	if err != nil {
		return err
	}

	err = s.executeStandardTestRunCommand()
	if err != nil {
		return err
	}

	if s.test.runResult.Verdict != verdict.OK {
		s.test.hasResources = false
		return nil
	}
	s.test.hasResources = true

	err = s.fullCheckPipeline(true)
	if err != nil {
		return err
	}

	return nil
}

func (s *JobPipelineState) generateStandartTestRunConfig() error {
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

func (s *JobPipelineState) executeStandardTestRunCommand() error {
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
