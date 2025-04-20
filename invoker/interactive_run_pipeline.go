package invoker

import (
	"fmt"
	"os"
	"sync"
	"testing_system/common/constants/verdict"
	"testing_system/invoker/sandbox"
	"testing_system/lib/logger"
)

// For interactive problems, we use two sandboxes, one for interactor and another one for solution.
// Interactor creates most of the files, so all the work is associated with interactor sandbox.
// Solution sandbox just initializes and gives ownership to interactor sandbox,
// which will release check sandbox when all the work is done.
//
// We use following synchronisation points:
//
// Job.InteractiveData.PipelineReadyWG - used for InteractiveInteractorJob to wait for InteractiveSolutionJob to initialize itself.
// After this WaitGroup is ready, the ownership of InteractiveSolutionJob is transfered to InteractiveInteractorJob.
//
// JobPipelineState.interaction.solutionReleaseWaitGroup - used for InteractiveSolutionJob
// to wait until the job is released by InteractiveInteractorJob.
//
// JobPipelineState.interaction.runWaitGroup - used to start running of both interactor and solution processes simultaneously.
// The running of processes is done in runner threads, and this threads can be free at different points of time.
// So before the running begins, we wait until other process is also ready to run.

func (i *Invoker) fullInteractiveSolutionPipeline(sandbox sandbox.ISandbox, job *Job) {
	s := &JobPipelineState{
		job:         job,
		invoker:     i,
		sandbox:     sandbox,
		test:        new(pipelineTestData),
		interaction: new(pipelineInteractionData),
		loggerData: fmt.Sprintf(
			"interactive solution job: %s submission: %d problem %d test %d",
			job.ID,
			job.Submission.ID,
			job.Problem.ID,
			job.Test,
		),
	}

	s.defers = append(s.defers, s.job.deferFunc)
	defer s.deferFunc()

	s.job.InteractiveData.SolutionPipelineState = s
	s.interaction.solution = s

	s.interaction.solutionReleaseWaitGroup.Add(1)
	// WaitGroup panics if done is called multiple times
	s.interaction.solutionRelease = sync.OnceFunc(func() {
		s.interaction.solutionReleaseWaitGroup.Done()
	})

	// Ownership moves to interactor sandbox
	s.job.InteractiveData.PipelineReadyWG.Done()

	// Wait for interactor sandbox to release solution sandbox
	s.interaction.solutionReleaseWaitGroup.Wait()
}

func (i *Invoker) fullInteractiveInteractorPipeline(sandbox sandbox.ISandbox, job *Job) {
	s := &JobPipelineState{
		job:     job,
		invoker: i,
		sandbox: sandbox,
		test:    new(pipelineTestData),
		loggerData: fmt.Sprintf(
			"interactive job: %s submission: %d problem %d test %d",
			job.ID,
			job.Submission.ID,
			job.Problem.ID,
			job.Test,
		),
	}
	s.job.InteractiveData.PipelineReadyWG.Wait()
	logger.Trace("Starting testing for %s", s.loggerData)

	// Two jobs share common interaction field to wait each other before execution begins
	s.interaction = s.job.InteractiveData.SolutionPipelineState.interaction
	s.defers = append(s.defers, job.deferFunc)
	s.defers = append(s.defers, s.interaction.solutionRelease)
	defer s.deferFunc()

	err := s.interactiveTestingPipeline()
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

func (s *JobPipelineState) interactiveTestingPipeline() error {
	defer s.fixInteractiveVerdict()
	err := s.initSandbox()
	if err != nil {
		return err
	}

	err = s.interaction.solution.initSandbox()
	if err != nil {
		return err
	}

	err = s.interaction.solution.loadSolutionBinary()
	if err != nil {
		return err
	}

	err = s.loadInteractorBinaryFile()
	if err != nil {
		return err
	}

	err = s.loadTestInput()
	if err != nil {
		return err
	}

	err = s.loadTestAnswerFile()
	if err != nil {
		return err
	}
	err = s.generateInteractiveTestRunConfig()
	if err != nil {
		return err
	}

	err = s.executeInteractiveTestRunCommand()
	if err != nil {
		return err
	}

	// We do not need solution anymore, so we can release its sandbox.
	s.interaction.solutionRelease()

	if s.test.runResult.Verdict != verdict.OK {
		s.test.hasResources = false
		return nil
	}
	s.test.hasResources = true

	err = s.parseTestlibResult("interactor", s.interaction.interactorResult)
	if err != nil {
		return err
	}

	if s.test.runResult.Verdict != verdict.OK {
		return nil
	}

	// Isolate sandbox doesn't like to write two times in the same file, and we will use this file for checker result
	err = s.removeSandboxFile(testlibResultFile)
	if err != nil {
		return err
	}

	err = s.fullCheckPipeline(false)
	if err != nil {
		return err
	}

	return nil
}

func (s *JobPipelineState) generateInteractiveTestRunConfig() error {
	solutionConfig := new(sandbox.ExecuteConfig)
	fillInTestRunConfigLimits(solutionConfig, s.job.Problem)
	solutionConfig.Command = solutionBinaryFile

	interactorConfig := new(sandbox.ExecuteConfig)
	interactorConfig.RunLimitsConfig = *s.invoker.TS.Config.Invoker.InteractorLimits
	interactorConfig.Command = interactorBinaryFile
	interactorConfig.Args = []string{
		testInputFile, testOutputFile, testAnswerFile, testlibResultFile, xmlTestlibCommandOption,
	}

	err := s.createPipe(solutionConfig, interactorConfig)
	if err != nil {
		return err
	}
	err = s.createPipe(interactorConfig, solutionConfig)
	if err != nil {
		return err
	}

	solutionConfig.Interactive = true
	interactorConfig.Interactive = true
	s.interaction.interactorConfig = interactorConfig
	s.interaction.solution.test.runConfig = solutionConfig

	logger.Trace("Generated test run config for %s", s.loggerData)
	return nil
}

func (s *JobPipelineState) createPipe(
	stdinCfg *sandbox.ExecuteConfig,
	stdoutCfg *sandbox.ExecuteConfig,
) error {
	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("error creating pipe: %v", err)
	}
	stdinCfg.Stdin = &sandbox.IORedirect{
		Input: r,
	}

	stdoutCfg.Stdout = &sandbox.IORedirect{
		Output: w,
	}

	stdinCfg.Defers = append(stdinCfg.Defers, func() { r.Close() })
	stdoutCfg.Defers = append(stdoutCfg.Defers, func() { w.Close() })

	return nil
}

func (s *JobPipelineState) executeInteractiveTestRunCommand() error {
	s.interaction.runWaitGroup.Add(2)

	s.interaction.solution.executeWaitGroup.Add(1)
	s.invoker.RunQueue <- s.interaction.solution.runInteractiveSolution

	s.executeWaitGroup.Add(1)
	// If multiple threads are enabled on invoker, we use different thread for interactor
	// Otherwise, we just run interactor in separate goroutine
	if s.invoker.TS.Config.Invoker.Threads > 1 {
		s.invoker.RunQueue <- s.runInteractiveInteractor
	} else {
		s.invoker.TS.Go(s.runInteractiveInteractor)
	}

	s.executeWaitGroup.Wait()
	s.interaction.solution.executeWaitGroup.Wait()

	if s.interaction.solution.test.runResult.Err != nil {
		return fmt.Errorf(
			"can not run interactive solution in sandbox, error: %v",
			s.interaction.solution.test.runResult.Err,
		)
	}

	if s.interaction.interactorResult.Err != nil {
		return fmt.Errorf("can not run interactor in sandbox, error: %v", s.interaction.interactorResult.Err)
	}

	err := s.verifyHelperCommandVerdict("interactor", s.interaction.interactorResult.Verdict)
	if err != nil {
		return err
	}

	// Dirty hack, but most of the work for problem goes inside interactor sandbox, so this makes code much simpler
	s.test.runResult = s.interaction.solution.test.runResult

	logger.Trace("Finished running solution and interactor for %s", s.loggerData)
	return nil
}

func (s *JobPipelineState) runInteractiveSolution() {
	// Wait until other job execution begins
	s.interaction.runWaitGroup.Done()
	s.interaction.runWaitGroup.Wait()

	s.test.runResult = s.interaction.solution.sandbox.Run(s.interaction.solution.test.runConfig)
	s.executeWaitGroup.Done()
}

func (s *JobPipelineState) runInteractiveInteractor() {
	// Wait until other job execution begins
	s.interaction.runWaitGroup.Done()
	s.interaction.runWaitGroup.Wait()

	s.interaction.interactorResult = s.sandbox.Run(s.interaction.interactorConfig)
	s.executeWaitGroup.Done()
}

// Interactive problems can have two verdicts: OK and Wrong!
func (s *JobPipelineState) fixInteractiveVerdict() {
	if s.test.runResult != nil {
		switch s.test.runResult.Verdict {
		case verdict.OK, verdict.PT, verdict.CF:
			// skip
		default:
			s.test.runResult.Verdict = verdict.WR
		}
	}
}
