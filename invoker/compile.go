package invoker

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/constants/resource"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/invoker/compiler"
	"testing_system/invoker/sandbox"
	"testing_system/lib/logger"
)

type compileJob struct {
	invoker *Invoker
	tester  *JobExecutor

	job        *Job
	language   *compiler.Language
	submission *models.Submission

	binaryName string

	runResult *sandbox.RunResult

	wg *sync.WaitGroup
}

func (i *Invoker) Compile(tester *JobExecutor, job *Job) {
	logger.Trace("Starting compilation of submit %d, job %s", job.Submission.ID, job.ID)
	defer i.Storage.Source.Unlock(uint64(job.Submission.ID))

	tester.Sandbox.Init()
	defer tester.Sandbox.Cleanup()

	j := compileJob{
		tester: tester,
		job:    job,
	}

	err := j.Prepare()
	if err != nil {
		logger.Trace("Compilation of submit %d in job %s prepare error: %s", job.Submission.ID, job.ID, err)
		j.invoker.FailJob(j.job, "can not start compilation of submit %d, job %s, error: %s", j.submission.ID, j.job.ID, err.Error())
		return
	}
	logger.Trace("Prepared compilation of submit %d, job %s", j.submission.ID, j.job.ID)

	j.wg = new(sync.WaitGroup)
	j.wg.Add(1)
	i.RunQueue <- j.Execute
	j.wg.Wait()

	if j.runResult.Err != nil {
		logger.Error("Compilation of submit %d in job %s error: %s", job.Submission.ID, job.ID, err)
		j.invoker.FailJob(j.job, "can not compile submit %d, job %s, error: %s ", j.submission.ID, j.job.ID, j.runResult.Err.Error())
		return
	}
	logger.Trace("Finished compilation process of submit %d in job %s", j.submission.ID, j.job.ID)

	err = j.Finish()
	if err == nil {
		logger.Trace("Finished compilation of submit %d in job %s", j.submission.ID, j.job.ID)
		j.invoker.SuccessJob(j.job, j.runResult)
	} else {
		logger.Trace("Compilation of submit %d in job %s send result error: %s", job.Submission.ID, job.ID, err)
		j.invoker.FailJob(j.job, "can not upload compilation result of submit %d, job %s, error: %s", j.submission.ID, j.job.ID, err.Error())
	}
}

func (j *compileJob) Prepare() error {
	j.submission = j.job.Submission

	source, err := j.invoker.Storage.Source.Get(uint64(j.submission.ID))
	if err != nil {
		return logger.Error("can not get source, error: %s", err.Error())
	}
	sourceFile, ok := source.File()
	if !ok {
		return logger.Error("can not get source, file not found")
	}
	sourceName := "source_" + filepath.Base(sourceFile)
	err = j.tester.CopyFileToSandbox(sourceFile, sourceName, 0644)
	if err != nil {
		return logger.Error("can not copy file to sandbox, error: %s", err.Error())
	}

	lang, ok := j.invoker.Compiler.Languages[j.submission.Language]
	if !ok {
		return fmt.Errorf("language %s does not exist", j.submission.Language)
	}
	j.binaryName = binaryName(sourceName)
	script, err := lang.GenerateScript(sourceName, j.binaryName)
	if err != nil {
		logger.Warn("Can not generate compile script for submit %d, job %s, error: %s", j.submission.ID, j.job.ID, err.Error())
		return err
	}
	err = j.tester.CreateSandboxFile("compile.sh", 0755, script)
	if err != nil {
		return logger.Error("can not create compile script, error: %s", err.Error())
	}
	return nil
}

func binaryName(sourceName string) string {
	name := "solution"
	if name == sourceName {
		name += "_binary"
	}
	return name
}

func (j *compileJob) Execute() {
	j.runResult = j.tester.Sandbox.Run("compile.sh", []string{"compile.sh"}, j.language.Limits)
	j.wg.Done()
}

func (j *compileJob) Finish() error {
	var outputReader io.Reader

	switch j.runResult.Verdict {
	case verdict.OK:
		j.runResult.Verdict = verdict.CD
	case verdict.RT:
		j.runResult.Verdict = verdict.CE
		if j.invoker.TS.Config.Invoker.SaveOutputHead == nil {
			outputReader = j.runResult.Stdout
		} else {
			outputReader = io.LimitReader(j.runResult.Stdout, int64(*j.invoker.TS.Config.Invoker.SaveOutputHead))
		}
	case verdict.TL:
		j.runResult.Verdict = verdict.CE
		outputReader = strings.NewReader(fmt.Sprintf("Compilation took more than %v time", j.language.Limits.TL))
	case verdict.ML:
		j.runResult.Verdict = verdict.CE
		outputReader = strings.NewReader(fmt.Sprintf("Compilation took more than %v memory", j.language.Limits.ML))
	case verdict.WL:
		j.runResult.Verdict = verdict.CE
		outputReader = strings.NewReader(fmt.Sprintf("Compilation took more than %v wall time", j.language.Limits.WL))
	case verdict.SE:
		j.runResult.Verdict = verdict.CE
		outputReader = strings.NewReader(fmt.Sprintf("Security violation"))
	default:
		return logger.Error("unknown sandbox verdict: %d", j.runResult.Verdict)
	}

	compileOuputStoreRequest := &storageconn.Request{
		Resource: resource.CompileOutput,
		SubmitID: uint64(j.submission.ID),
		Files: map[string]io.Reader{
			"compile_output.txt": outputReader,
		},
	}
	resp := j.invoker.TS.StorageConn.Upload(compileOuputStoreRequest)
	if resp.Error != nil {
		return logger.Error("can not upload compile output to storage, error: %s", resp.Error.Error())
	}

	if j.runResult.Verdict == verdict.CD {
		compiledReader, err := os.Open(filepath.Join(j.tester.Sandbox.Dir(), j.binaryName))
		if err != nil {
			return logger.Error("can not open compiled binary, error: %s", err.Error())
		}
		defer compiledReader.Close()

		binaryStoreRequest := &storageconn.Request{
			Resource: resource.CompiledBinary,
			SubmitID: uint64(j.submission.ID),
			Files: map[string]io.Reader{
				j.binaryName: compiledReader,
			},
		}
		resp = j.invoker.TS.StorageConn.Upload(binaryStoreRequest)
		if resp.Error != nil {
			return logger.Error("can not upload compiled binary, error: %s", resp.Error.Error())
		}
	}

	return nil
}
