package invoker

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing_system/common/config"
	"testing_system/common/connectors/storageconn"
	"testing_system/common/constants/resource"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/invoker/sandbox"
	"testing_system/lib/logger"
)

const (
	BinaryFile         = "solution"
	InputFile          = "input.txt"
	OutputFile         = "output.txt"
	ErrorFile          = "stderr.txt"
	AnswerFile         = "answer.txt"
	CheckerFile        = "check"
	CheckResultFile    = "check_result.xml"
	CheckResultFileArg = "-appes"
	CheckerOutputFile  = "check.txt"
)

type testJob struct {
	invoker *Invoker
	tester  *JobExecutor

	job *Job

	runConfig *sandbox.ExecuteConfig
	runResult *sandbox.RunResult

	checkConfig *sandbox.ExecuteConfig
	checkResult *sandbox.RunResult

	checkerOutputReader io.Reader

	wg sync.WaitGroup
}

func (i *Invoker) Test(tester *JobExecutor, job *Job) {
	logger.Trace("Starting testing of submit %d, test %d, job %s", job.Submission.ID, job.Test, job.ID)
	defer job.DeferFunc()

	tester.Sandbox.Init()
	defer tester.Sandbox.Cleanup()

	j := testJob{
		invoker: i,
		tester:  tester,
		job:     job,
	}

	err := j.PrepareRun()
	if err != nil {
		logger.Error("Prepare running of submit %d on problem %d test %d job %s fail, error: %s", job.Submission.ID, job.Problem, job.Test, job.ID, err.Error())
		j.invoker.FailJob(job, "can not prepare run of job %s, error: %s", job.ID, err.Error())
		return
	}
	logger.Trace("Prepared running of submit %d on problem %d test %d job %s", job.Submission.ID, job.Problem.ID, job.Test, job.ID)

	j.wg.Add(1)
	i.RunQueue <- j.RunOnTest
	j.wg.Wait()

	if j.runResult.Err != nil {
		logger.Error("Can not run submit %d on problem %d test %d in job %s error: %s", job.Submission.ID, job.Problem.ID, job.Test, job.ID, err)
		j.invoker.FailJob(job, "can not run submit in job %s, error: %s", job.ID, j.runResult.Err.Error())
		return
	}

	if j.runResult.Verdict != verdict.OK {
		logger.Trace("Finished running process of submit %d on problem %d test %d job %s with verdict %v, uploading result", job.Submission.ID, job.Problem.ID, job.Test, job.ID, j.runResult.Verdict)
		j.invoker.SuccessJob(job, j.runResult)
		return
	}

	logger.Trace("Successfully finished running process of submit %d on problem %d test %d job %s, preparing checker run", job.Submission.ID, job.Problem.ID, job.Test, job.ID)
	err = j.PrepareCheck()
	if err != nil {
		logger.Error("Prepare checking of submit %d on problem %d test %d job %s fail, error: %s", job.Submission.ID, job.Problem.ID, job.Test, job.ID, err.Error())
		j.invoker.FailJob(job, "can not prepare check of job %s, error: %s", job.ID, err.Error())
		return
	}
	logger.Trace("Prepared checking of submit %d on problem %d test %d job %s", job.Submission.ID, job.Problem, job.Test, job.ID)

	j.wg.Add(1)
	i.RunQueue <- j.RunChecker
	j.wg.Wait()

	if j.runResult.Err != nil {
		logger.Error("Can not check submit %d on problem %d test %d in job %s error: %s", job.Submission.ID, job.Problem.ID, job.Test, job.ID, j.runResult.Err.Error())
		j.invoker.FailJob(job, "can not check job %s, error: %s", job.ID, j.runResult.Err.Error())
		return
	}
	logger.Trace("Finished checking of submit %d on problem %d test %d in job %s", job.Submission.ID, job.Problem.ID, job.Test, job.ID)

	err = j.Finish()
	if err == nil {
		logger.Trace("Uploaded testing and checking result of submit %d on problem %d test %d in job %s", job.Submission.ID, job.Problem.ID, job.Test, job.ID)
		j.invoker.SuccessJob(job, j.runResult)
	} else {
		logger.Error("Upload testing and checking result of submit %d on problem %d test %d in job %s error %s", job.Submission.ID, job.Problem.ID, job.Test, job.ID, err.Error())
		j.invoker.FailJob(job, "can not upload testing and checking result of job %s, error: %s", job.ID, err.Error())
	}
}

func (j *testJob) PrepareRun() error {
	j.runConfig = problemRunConfig(j.job.Problem)

	// This defers will be called during sandbox execution. However, in case of error we will call them with job defers.
	// The RunConfig defers are cleaned up after each call, so we can call runConfig defer function multiple times with no harm
	j.job.Defers = append(j.job.Defers, j.runConfig.DeferFunc)

	binary, err := j.invoker.Storage.Binary.Get(uint64(j.job.Submission.ID))
	if err != nil {
		return fmt.Errorf("can not get binary, error: %s", err.Error())
	}
	err = j.tester.CopyFileToSandbox(*binary, BinaryFile, 0755)
	if err != nil {
		return fmt.Errorf("can not copy binary, error: %s", err.Error())
	}
	j.runConfig.Command = BinaryFile

	testInput, err := j.invoker.Storage.TestInput.Get(uint64(j.job.Problem.ID), j.job.Test)
	if err != nil {
		return fmt.Errorf("can not get test input, error: %s", err.Error())
	}
	err = j.tester.CopyFileToSandbox(*testInput, InputFile, 0644)
	if err != nil {
		return fmt.Errorf("can not copy test input to sandbox, error: %s", err.Error())
	}

	stdin, err := os.Open(filepath.Join(j.tester.Sandbox.Dir(), InputFile))
	if err != nil {
		return fmt.Errorf("can not open test for reading, path: %s, error: %s", *testInput, err.Error())
	}
	j.runConfig.Defers = append(j.runConfig.Defers, func() { stdin.Close() })
	j.runConfig.Stdin = stdin

	stdout, err := os.OpenFile(filepath.Join(j.tester.Sandbox.Dir(), OutputFile), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("can not open output file for writing, path: %s, error: %s", *testInput, err.Error())
	}
	j.runConfig.Defers = append(j.runConfig.Defers, func() { stdout.Close() })
	j.runConfig.Stdout = stdout

	stderr, err := os.OpenFile(filepath.Join(j.tester.Sandbox.Dir(), ErrorFile), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("can not open stderr file for writing, path: %s, error: %s", *testInput, err.Error())
	}
	j.runConfig.Defers = append(j.runConfig.Defers, func() { stderr.Close() })
	j.runConfig.Stderr = stderr

	return nil
}

func problemRunConfig(problem *models.Problem) *sandbox.ExecuteConfig {
	c := sandbox.ExecuteConfig{
		RunConfig: config.RunConfig{
			TimeLimit:   problem.TimeLimit,
			MemoryLimit: problem.MemoryLimit,
		},
	}

	if problem.WallTimeLimit != nil {
		c.WallTimeLimit = *problem.WallTimeLimit
	} else {
		c.WallTimeLimit.FromStr("5s")
		if c.WallTimeLimit < c.TimeLimit*2 {
			c.WallTimeLimit = c.TimeLimit * 2
		}
	}

	if problem.MaxOpenFiles == nil {
		c.MaxOpenFiles = 64
	}
	if problem.MaxThreads == nil {
		c.MaxThreads = 1
	}
	if problem.MaxOutputSize == nil {
		c.MaxOutputSize.FromStr("1g")
	}
	return &c
}

func (j *testJob) RunOnTest() {
	j.runResult = j.tester.Sandbox.Run(j.runConfig)
	j.wg.Done()
}

func (j *testJob) PrepareCheck() error {
	j.checkConfig = &sandbox.ExecuteConfig{
		RunConfig: *j.invoker.TS.Config.Invoker.CheckerLimits,
	}

	checker, err := j.invoker.Storage.Checker.Get(uint64(j.job.Problem.ID))
	if err != nil {
		return fmt.Errorf("can not get checker, error: %s", err.Error())
	}
	err = j.tester.CopyFileToSandbox(*checker, CheckerFile, 0755)
	if err != nil {
		return fmt.Errorf("can not copy checker to sandbox, error: %s", err.Error())
	}
	j.checkConfig.Command = CheckerFile

	testAnswer, err := j.invoker.Storage.TestAnswer.Get(uint64(j.job.Problem.ID))
	if err != nil {
		return fmt.Errorf("can not get test answer, error: %s", err.Error())
	}
	err = j.tester.CopyFileToSandbox(*testAnswer, AnswerFile, 0644)
	if err != nil {
		return fmt.Errorf("can not copy test answer to sandbox, error: %s", err.Error())
	}

	j.checkConfig.Args = []string{InputFile, OutputFile, AnswerFile, CheckResultFile, CheckResultFileArg}
	return nil
}

func (j *testJob) RunChecker() {
	j.checkResult = j.tester.Sandbox.Run(j.checkConfig)
	j.wg.Done()
}

func (j *testJob) Finish() error {
	err := j.UploadOutput(OutputFile, resource.TestOutput)
	if err != nil {
		return err
	}

	err = j.UploadOutput(ErrorFile, resource.TestStderr)
	if err != nil {
		return err
	}

	switch j.checkResult.Verdict {
	case verdict.OK, verdict.RT:
		j.ParseCheckerOutput()
	case verdict.TL:
		j.runResult.Verdict = verdict.CF
		j.checkerOutputReader = strings.NewReader(fmt.Sprintf("Checker running took more than %v time", j.checkConfig.TimeLimit))
	case verdict.ML:
		j.runResult.Verdict = verdict.CF
		j.checkerOutputReader = strings.NewReader(fmt.Sprintf("Checker running took more than %v memory", j.checkConfig.MemoryLimit))
	case verdict.WL:
		j.runResult.Verdict = verdict.CF
		j.checkerOutputReader = strings.NewReader(fmt.Sprintf("Checker running took more than %v wall time", j.checkConfig.WallTimeLimit))
	case verdict.SE:
		j.runResult.Verdict = verdict.CF
		j.checkerOutputReader = strings.NewReader(fmt.Sprintf("Checker security violation"))
	default:
		return fmt.Errorf("unknown checker run sandbox verdict: %s", j.checkResult.Verdict)
	}

	checkerOutputRequest := &storageconn.Request{
		Resource: resource.CheckerOutput,
		SubmitID: uint64(j.job.Submission.ID),
		TestID:   j.job.Test,
		Files: map[string]io.Reader{
			CheckerOutputFile: j.invoker.limitedReader(j.checkerOutputReader),
		},
	}
	resp := j.invoker.TS.StorageConn.Upload(checkerOutputRequest)
	if resp.Error != nil {
		return fmt.Errorf("can not upload checker output to storage, error: %s", resp.Error.Error())
	}
	return nil
}

func (j *testJob) UploadOutput(fileName string, resourceType resource.Type) error {
	outputF, err := os.Open(filepath.Join(j.tester.Sandbox.Dir(), fileName))
	if err != nil {
		return fmt.Errorf("can not open %v file for reading, error: %s", resourceType, err.Error())
	}
	defer outputF.Close()

	outputStoreRequest := &storageconn.Request{
		Resource: resourceType,
		SubmitID: uint64(j.job.Submission.ID),
		TestID:   j.job.Test,
		Files: map[string]io.Reader{
			fileName: j.invoker.limitedReader(outputF),
		},
	}
	resp := j.invoker.TS.StorageConn.Upload(outputStoreRequest)
	if resp.Error != nil {
		return fmt.Errorf("can not upload %v to storage, error: %s", resourceType, resp.Error.Error())
	}
	return nil
}

func (j *testJob) ParseCheckerOutput() {
	checkResultData, err := os.ReadFile(filepath.Join(j.tester.Sandbox.Dir(), CheckResultFile))
	if err != nil {
		j.runResult.Verdict = verdict.CF
		j.checkerOutputReader = strings.NewReader(fmt.Sprintf("Checker exited with exit code %d, can not parse checker result xml file in appes mode", j.checkResult.Statistics.ExitCode))
		return
	}
	var checkerResult CheckerResultXML
	err = xml.Unmarshal(checkResultData, &checkerResult)
	if err != nil {
		j.runResult.Verdict = verdict.CF
		j.checkerOutputReader = strings.NewReader(fmt.Sprintf("Can not parse checker result xml file in appes mode: %s", err.Error()))
		return
	}
	j.checkerOutputReader = strings.NewReader(checkerResult.Value)
	switch checkerResult.Outcome {
	case "accepted":
		j.runResult.Verdict = verdict.OK
	case "wrong-answer", "presentation-error", "unexpected-eof": // We will treat PE as WA as all testing systems now do
		j.runResult.Verdict = verdict.WA
	case "fail":
		j.runResult.Verdict = verdict.CF
	case "points", "relative-scoring":
		if checkerResult.Points == nil {
			j.runResult.Verdict = verdict.CF
			j.checkerOutputReader = strings.NewReader(fmt.Sprintf("Checker exited with exit code %d and verdict %s, but no points specified", j.checkResult.Statistics.ExitCode, checkerResult.Outcome))
		} else {
			j.runResult.Verdict = verdict.PT
			j.runResult.Points = checkerResult.Points
		}
	default:
		j.runResult.Verdict = verdict.CF
		j.checkerOutputReader = strings.NewReader(fmt.Sprintf("Unknown checker verdict %s, checker exited with exit code %d", checkerResult.Outcome, j.checkResult.Statistics.ExitCode))
	}
}

type CheckerResultXML struct {
	Outcome string   `xml:"outcome,attr"`
	Points  *float64 `xml:"points,attr,omitempty"`
	Value   string   `xml:",chardata"`
}
