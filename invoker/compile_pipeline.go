package invoker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/constants/resource"
	"testing_system/common/constants/verdict"
	"testing_system/invoker/sandbox"
	"testing_system/lib/logger"
)

func (i *Invoker) fullCompilationPipeline(sandbox sandbox.ISandbox, job *Job) {
	s := &JobPipelineState{
		job:        job,
		invoker:    i,
		sandbox:    sandbox,
		compile:    new(pipelineCompileData),
		loggerData: fmt.Sprintf("compile job: %s submission: %d", job.ID, job.Submission.ID),
	}

	s.defers = append(s.defers, job.deferFunc)
	defer s.deferFunc()

	logger.Trace("Starting compilation for %s", s.loggerData)

	err := s.compilationProcessPipeline()
	if err != nil {
		logger.Error("Error in %s error: %v", s.loggerData, err)
		i.failJob(job, "job %s error: %v", job.ID, err)
		return
	}

	err = s.uploadCompilationResources()
	if err != nil {
		logger.Error("Error in %s error: %v", s.loggerData, err)
		i.failJob(job, "job %s error: %v", job.ID, err)
		return
	}

	i.successJob(job, s.compile.result)
}

func (s *JobPipelineState) compilationProcessPipeline() error {
	err := s.initSandbox()
	if err != nil {
		return err
	}

	err = s.loadSource()
	if err != nil {
		return err
	}

	err = s.setupCompileScript()
	if err != nil {
		return err
	}

	err = s.executeCompilationCommand()
	if err != nil {
		return err
	}

	err = s.processCompilationResult()
	if err != nil {
		return err
	}
	return nil
}

func (s *JobPipelineState) loadSource() error {
	source, err := s.invoker.Storage.Source.Get(uint64(s.job.Submission.ID))
	if err != nil {
		return fmt.Errorf("can not get submission source, error: %v", err)
	}
	s.compile.sourceName = "source_" + filepath.Base(*source)
	err = s.copyFileToSandbox(*source, s.compile.sourceName, 0644)
	if err != nil {
		return fmt.Errorf("can not copy submission source to sandbox, error: %v", err)
	}
	logger.Trace("Loaded source to sandbox for %s", s.loggerData)
	return nil
}

func (s *JobPipelineState) setupCompileScript() error {
	var ok bool
	s.compile.language, ok = s.invoker.Compiler.Languages[s.job.Submission.Language]
	if !ok {
		return fmt.Errorf("submission language %s does not exist", s.job.Submission.Language)
	}
	script, err := s.compile.language.GenerateScript(s.compile.sourceName, solutionBinaryFile)
	if err != nil {
		return fmt.Errorf("can not generate compile script, error: %v", err)
	}
	err = os.WriteFile(filepath.Join(s.sandbox.Dir(), compileScriptFile), script, 0755)
	if err != nil {
		return fmt.Errorf("can not write compile script to sandbox, error: %v", err)
	}

	s.compile.config = s.compile.language.GenerateExecuteConfig(compilationMessageFile)
	s.compile.config.Command = "compile.sh"
	logger.Trace("Prepared compilation for %s", s.loggerData)
	return nil
}

func (s *JobPipelineState) executeCompilationCommand() error {
	s.executeWaitGroup.Add(1)
	s.invoker.RunQueue <- s.runCompilationCommand
	s.executeWaitGroup.Wait()

	if s.compile.result.Err != nil {
		return fmt.Errorf("can not run compilation in sandbox, error: %v", s.compile.result.Err)
	}
	logger.Trace("Executed compilation for %s with verdict %s", s.loggerData, s.compile.result.Verdict)
	return nil
}

func (s *JobPipelineState) runCompilationCommand() {
	s.compile.result = s.sandbox.Run(s.compile.config)
	s.executeWaitGroup.Done()
}

func (s *JobPipelineState) processCompilationResult() error {
	switch s.compile.result.Verdict {
	case verdict.OK:
		s.compile.result.Verdict = verdict.CD
		err := s.openCompilationMessage()
		if err != nil {
			return err
		}
	case verdict.RT:
		s.compile.result.Verdict = verdict.CE
		err := s.openCompilationMessage()
		if err != nil {
			return err
		}
	case verdict.TL:
		s.compile.result.Verdict = verdict.CE
		s.compile.messageReader = strings.NewReader(
			fmt.Sprintf("Compilation took more than %v time", s.compile.language.Limits.TimeLimit),
		)
	case verdict.ML:
		s.compile.result.Verdict = verdict.CE
		s.compile.messageReader = strings.NewReader(
			fmt.Sprintf("Compilation took more than %v memory", s.compile.language.Limits.MemoryLimit),
		)
	case verdict.WL:
		s.compile.result.Verdict = verdict.CE
		s.compile.messageReader = strings.NewReader(
			fmt.Sprintf("Compilation took more than %v wall time", s.compile.language.Limits.WallTimeLimit),
		)
	case verdict.SE:
		s.compile.result.Verdict = verdict.CE
		s.compile.messageReader = strings.NewReader(fmt.Sprintf("Security violation"))
		return nil
	default:
		return fmt.Errorf("unknown sandbox verdict: %s", s.compile.result.Verdict)
	}
	logger.Trace("Processed compilation result for %s", s.loggerData)
	return nil
}

func (s *JobPipelineState) openCompilationMessage() error {
	var err error
	s.compile.messageReader, err = s.openSandboxFile(compilationMessageFile, true)
	if err != nil {
		return fmt.Errorf("can not open compilation message file, error: %v", err)
	}
	logger.Trace("Opened compilation message for reading")
	return nil
}

func (s *JobPipelineState) uploadCompilationResources() error {
	err := s.uploadCompileResult()
	if err != nil {
		return err
	}

	if s.compile.result.Verdict != verdict.CD {
		return nil
	}
	return s.uploadBinary()
}

func (s *JobPipelineState) uploadCompileResult() error {
	compileOutputStoreRequest := &storageconn.Request{
		Resource: resource.CompileOutput,
		SubmitID: uint64(s.job.Submission.ID),
		File:     s.compile.messageReader,
	}
	resp := s.invoker.TS.StorageConn.Upload(compileOutputStoreRequest)
	if resp.Error != nil {
		return fmt.Errorf("can not upload compile output to storage, error: %v", resp.Error.Error())
	}
	logger.Trace("Uploaded compilation result to storage for %s", s.loggerData)
	return nil
}
