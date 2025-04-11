package invoker

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/constants/resource"
	"testing_system/common/constants/verdict"
	"testing_system/invoker/compiler"
	"testing_system/invoker/sandbox"
	"testing_system/lib/logger"
)

type compileJob struct {
	invoker *Invoker
	tester  *JobExecutor

	job      *Job
	language *compiler.Language

	binaryName string
	stdout     bytes.Buffer

	compileConfig *sandbox.ExecuteConfig
	compileResult *sandbox.RunResult

	wg sync.WaitGroup
}

func (i *Invoker) Compile(tester *JobExecutor, job *Job) {
	logger.Trace("Starting compilation of submit %d, job %s", job.Submission.ID, job.ID)
	defer job.DeferFunc()

	tester.Sandbox.Init()
	defer tester.Sandbox.Cleanup()

	j := compileJob{
		invoker: i,
		tester:  tester,
		job:     job,
	}

	err := j.Prepare()
	if err != nil {
		logger.Error("Compilation of submit %d in job %s prepare error: %s", job.Submission.ID, job.ID, err.Error())
		j.invoker.FailJob(job, "can not prepare compilation of job %s, error: %s", job.ID, err.Error())
		return
	}
	logger.Trace("Prepared compilation of submit %d, job %s", job.Submission.ID, job.ID)

	j.wg.Add(1)
	i.RunQueue <- j.Execute
	j.wg.Wait()

	if j.compileResult.Err != nil {
		logger.Error("Can not compile submit %d in job %s error: %s",
			job.Submission.ID, job.ID, j.compileResult.Err.Error())
		j.invoker.FailJob(job, "can not compile submit in job %s, error: %s", job.ID, j.compileResult.Err.Error())
		return
	}
	logger.Trace("Finished compilation process of submit %d in job %s with verdict %v",
		job.Submission.ID, job.ID, j.compileResult.Verdict)

	err = j.Finish()
	if err == nil {
		logger.Trace("Uploaded result of compilation of submit %d in job %s", job.Submission.ID, job.ID)
		j.invoker.SuccessJob(job, j.compileResult)
	} else {
		logger.Error("Compilation of submit %d in job %s send result error: %s",
			job.Submission.ID, job.ID, err.Error())
		j.invoker.FailJob(job, "can not upload compilation result of submit %d, job %s, error: %s",
			job.Submission.ID, job.ID, err.Error())
	}
}

func (j *compileJob) Prepare() error {
	source, err := j.invoker.Storage.Source.Get(uint64(j.job.Submission.ID))
	if err != nil {
		return fmt.Errorf("can not get source, error: %s", err.Error())
	}
	sourceName := "source_" + filepath.Base(*source)
	err = j.tester.CopyFileToSandbox(*source, sourceName, 0644)
	if err != nil {
		return fmt.Errorf("can not copy file to sandbox, error: %s", err.Error())
	}

	var ok bool
	j.language, ok = j.invoker.Compiler.Languages[j.job.Submission.Language]
	if !ok {
		return fmt.Errorf("language %s does not exist", j.job.Submission.Language)
	}
	j.binaryName = binaryName(sourceName)
	script, err := j.language.GenerateScript(sourceName, j.binaryName)
	if err != nil {
		return fmt.Errorf("can not generate compile script, error: %s", err.Error())
	}
	err = os.WriteFile(filepath.Join(j.tester.Sandbox.Dir(), "compile.sh"), script, 0755)
	if err != nil {
		return fmt.Errorf("can not create compile script, error: %s", err.Error())
	}

	j.compileConfig = j.language.GenerateExecuteConfig(&j.stdout)
	j.compileConfig.Command = "compile.sh"

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
	j.compileResult = j.tester.Sandbox.Run(j.compileConfig)
	j.wg.Done()
}

func (j *compileJob) Finish() error {
	var outputReader io.Reader

	switch j.compileResult.Verdict {
	case verdict.OK:
		j.compileResult.Verdict = verdict.CD
		outputReader = j.invoker.limitedReader(&j.stdout)
	case verdict.RT:
		j.compileResult.Verdict = verdict.CE
		outputReader = j.invoker.limitedReader(&j.stdout)
	case verdict.TL:
		j.compileResult.Verdict = verdict.CE
		outputReader = strings.NewReader(fmt.Sprintf("Compilation took more than %v time",
			j.language.Limits.TimeLimit))
	case verdict.ML:
		j.compileResult.Verdict = verdict.CE
		outputReader = strings.NewReader(fmt.Sprintf("Compilation took more than %v memory",
			j.language.Limits.MemoryLimit))
	case verdict.WL:
		j.compileResult.Verdict = verdict.CE
		outputReader = strings.NewReader(fmt.Sprintf("Compilation took more than %v wall time",
			j.language.Limits.WallTimeLimit))
	case verdict.SE:
		j.compileResult.Verdict = verdict.CE
		outputReader = strings.NewReader(fmt.Sprintf("Security violation"))
	default:
		return fmt.Errorf("unknown sandbox verdict: %s", j.compileResult.Verdict)
	}

	compileOutputStoreRequest := &storageconn.Request{
		Resource: resource.CompileOutput,
		SubmitID: uint64(j.job.Submission.ID),
		Files: map[string]io.Reader{
			"compile_output.txt": outputReader,
		},
	}
	resp := j.invoker.TS.StorageConn.Upload(compileOutputStoreRequest)
	if resp.Error != nil {
		return fmt.Errorf("can not upload compile output to storage, error: %s", resp.Error.Error())
	}

	if j.compileResult.Verdict == verdict.CD {
		compiledReader, err := os.Open(filepath.Join(j.tester.Sandbox.Dir(), j.binaryName))
		if err != nil {
			return fmt.Errorf("can not open compiled binary, error: %s", err.Error())
		}
		defer compiledReader.Close()

		binaryStoreRequest := &storageconn.Request{
			Resource: resource.CompiledBinary,
			SubmitID: uint64(j.job.Submission.ID),
			Files: map[string]io.Reader{
				j.binaryName: compiledReader,
			},
		}
		resp = j.invoker.TS.StorageConn.Upload(binaryStoreRequest)
		if resp.Error != nil {
			return fmt.Errorf("can not upload compiled binary, error: %s", resp.Error.Error())
		}
	}

	return nil
}
