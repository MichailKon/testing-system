package invoker

import (
	"fmt"
	"github.com/xorcare/pointer"
	"gorm.io/gorm"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"testing_system/common"
	"testing_system/common/config"
	_ "testing_system/common/config"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/invoker/compiler"
	"testing_system/invoker/sandbox"
	"testing_system/invoker/storage"
)

type testState struct {
	t        *testing.T
	TS       *common.TestingSystem
	Invoker  *Invoker
	Executor *JobExecutor
	Dir      string
	FilesDir string
}

func newTestState(t *testing.T) *testState {
	ts := &testState{
		t:   t,
		Dir: t.TempDir(),
	}
	ts.FilesDir = filepath.Join(ts.Dir, "files")
	ts.TS = &common.TestingSystem{
		Config: &config.Config{
			Invoker: &config.InvokerConfig{
				SandboxType:           "simple",
				SandboxHomePath:       filepath.Join(ts.Dir, "sandbox"),
				CacheSize:             1000 * 1000 * 1000,
				CachePath:             ts.FilesDir,
				CompilerConfigsFolder: "testdata/compiler",
			},
		},
	}
	config.FillInInvokerConfig(ts.TS.Config.Invoker)
	ts.Invoker = &Invoker{
		TS:       ts.TS,
		Storage:  storage.NewInvokerStorage(ts.TS),
		Compiler: compiler.NewCompiler(ts.TS),
	}
	err := os.CopyFS(ts.FilesDir, os.DirFS("testdata/files"))
	if err != nil {
		t.Fatal(err)
	}
	ts.Executor = NewJobExecutor(ts.TS, 1)
	return ts
}

func (ts *testState) TestCompile(submitID uint) (file *string, v verdict.Verdict) {
	job := &Job{
		Job: invokerconn.Job{
			ID:       "JOB",
			SubmitID: submitID,
			Type:     invokerconn.CompileJob,
		},
		Problem: &models.Problem{
			Model: gorm.Model{
				ID: 1,
			},
		},
		Submission: &models.Submission{
			Model: gorm.Model{
				ID: submitID,
			},
			ProblemID: 1,
			Language:  "cpp",
		},
	}
	err := ts.Executor.Sandbox.Init()
	if err != nil {
		ts.t.Fatalf("failed to init executor, error: %v", err)
	}

	j := compileJob{
		invoker: ts.Invoker,
		tester:  ts.Executor,
		job:     job,
	}

	err = j.invoker.Storage.Source.Insert(fmt.Sprintf("%s/source/%d/%d.cpp", ts.FilesDir, submitID, submitID), uint64(submitID))
	if err != nil {
		ts.t.Fatalf("failed to insert source to storage, error: %v", err)
	}

	err = j.Prepare()
	if err != nil {
		ts.t.Fatalf("failed to prepare job, error: %v", err)
	}

	j.wg.Add(1)
	j.Execute()

	if j.compileResult.Err != nil {
		ts.t.Fatalf("failed to execute compile job, error: %v", j.compileResult.Err)
	}

	v = j.compileResult.Verdict
	if j.compileResult.Verdict == verdict.OK {
		file = pointer.String(filepath.Join(j.tester.Sandbox.Dir(), j.binaryName))
	}
	return
}

func TestCompile(t *testing.T) {
	ts := newTestState(t)

	file, v := ts.TestCompile(1)
	if v != verdict.OK {
		t.Fatalf("Compilation verdict wrong, error: %v", v)
	}

	cmd := exec.Command(*file)
	var stdout strings.Builder
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		t.Fatalf("failed to execute compiled source, error: %v", err)
	}
	if strings.TrimSpace(stdout.String()) != "1" {
		t.Fatalf("failed to execute compiled source, wrong stdout: %v", stdout.String())
	}

	ts.Executor.Sandbox.Cleanup()

	file, v = ts.TestCompile(2)
	if v != verdict.RT {
		t.Fatalf("Compilation verdict wrong, error: %v", v)
	}
}

func (ts *testState) AddProblem(problemID uint) {
	err := ts.Invoker.Storage.TestInput.Insert(fmt.Sprintf("%s/test_input/%d-1/1", ts.FilesDir, problemID), uint64(problemID), 1)
	if err != nil {
		ts.t.Fatalf("failed to add problem test input, error: %v", err)
	}

	err = ts.Invoker.Storage.TestAnswer.Insert(fmt.Sprintf("%s/test_answer/%d-1/1.a", ts.FilesDir, problemID), uint64(problemID), 1)
	if err != nil {
		ts.t.Fatalf("failed to add problem test answer, error: %v", err)
	}

	checkerDir := fmt.Sprintf("%s/checker/%d", ts.FilesDir, problemID)
	cmd := exec.Command("g++", "check.cpp", "-std=c++17", "-o", "check")
	cmd.Dir = checkerDir
	err = cmd.Run()
	if err != nil {
		ts.t.Fatalf("failed to compile checker, error: %v", err)
	}
	err = ts.Invoker.Storage.Checker.Insert(filepath.Join(checkerDir, "check"), uint64(problemID))
	if err != nil {
		ts.t.Fatalf("failed to add problem checker, error: %v", err)
	}
}

func (ts *testState) TestRun(submitID uint, problemID uint) *sandbox.RunResult {
	job := &Job{
		Job: invokerconn.Job{
			ID:       "JOB",
			SubmitID: submitID,
			Type:     invokerconn.TestJob,
			Test:     1,
		},
		Problem: &models.Problem{
			Model: gorm.Model{
				ID: problemID,
			},
			TestsNumber: 1,
		},
		Submission: &models.Submission{
			Model: gorm.Model{
				ID: submitID,
			},
			ProblemID: 1,
			Language:  "cpp",
		},
	}
	job.Problem.TimeLimit.FromStr("1s")
	job.Problem.MemoryLimit.FromStr("100m")

	sourceDir := fmt.Sprintf("%s/binary/%d", ts.FilesDir, submitID)
	cmd := exec.Command("g++", "source.cpp", "-std=c++17", "-o", "binary")
	cmd.Dir = sourceDir
	err := cmd.Run()
	if err != nil {
		ts.t.Fatalf("failed to compile source, error: %v", err)
	}
	err = ts.Invoker.Storage.Binary.Insert(filepath.Join(sourceDir, "binary"), uint64(submitID))
	if err != nil {
		ts.t.Fatalf("failed to insert submit binary, error: %v", err)
	}

	err = ts.Executor.Sandbox.Init()
	if err != nil {
		ts.t.Fatalf("failed to init executor, error: %v", err)
	}
	defer ts.Executor.Sandbox.Cleanup()

	j := testJob{
		invoker: ts.Invoker,
		tester:  ts.Executor,
		job:     job,
	}

	err = j.PrepareRun()
	if err != nil {
		ts.t.Fatalf("failed to prepare run, error: %v", err)
	}

	j.wg.Add(1)
	j.RunOnTest()

	if j.runResult.Err != nil {
		ts.t.Fatalf("failed to run, error: %v", j.runResult.Err)
	}

	if j.runResult.Verdict != verdict.OK {
		return j.runResult
	}

	err = j.PrepareCheck()
	if err != nil {
		ts.t.Fatalf("failed to prepare check, error: %v", err)
	}

	j.wg.Add(1)
	j.RunChecker()

	if j.checkResult.Err != nil {
		ts.t.Fatalf("failed to check, error: %v", j.checkResult.Err)
	}

	switch j.checkResult.Verdict {
	case verdict.OK, verdict.RT:
		j.ParseCheckerOutput()
	default:
		ts.t.Fatalf("Wrong checker verdict: %v", j.checkResult.Verdict)
	}

	return j.runResult
}

func TestRun(t *testing.T) {
	ts := newTestState(t)
	ts.AddProblem(1)

	res := ts.TestRun(3, 1)
	if res.Verdict != verdict.OK {
		t.Fatalf("Wrong verdict: %v", res.Verdict)
	}

	res = ts.TestRun(4, 1)
	if res.Verdict != verdict.RT {
		t.Fatalf("Wrong verdict: %v", res.Verdict)
	}

	res = ts.TestRun(5, 1)
	if res.Verdict != verdict.TL {
		t.Fatalf("Wrong verdict: %v", res.Verdict)
	}

	res = ts.TestRun(6, 1)
	if res.Verdict != verdict.WA {
		t.Fatalf("Wrong verdict: %v", res.Verdict)
	}

	res = ts.TestRun(7, 1)
	if res.Verdict != verdict.ML {
		t.Fatalf("Wrong verdict: %v", res.Verdict)
	}

	ts.AddProblem(2)
	res = ts.TestRun(8, 2)
	if res.Verdict != verdict.PT {
		t.Fatalf("Wrong verdict: %v", res.Verdict)
	}
	if *res.Points != 5 {
		t.Fatalf("Wrong points: %v", res.Points)
	}
}
