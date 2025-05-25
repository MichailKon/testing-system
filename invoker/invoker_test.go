package invoker

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing_system/common"
	"testing_system/common/config"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/invoker/compiler"
	"testing_system/invoker/sandbox"
	"testing_system/invoker/storage"
	"time"
)

type testState struct {
	t        *testing.T
	TS       *common.TestingSystem
	Invoker  *Invoker
	Sandbox  sandbox.ISandbox
	Dir      string
	FilesDir string
}

func newTestState(t *testing.T, sandboxType string) *testState {
	ts := &testState{
		t:   t,
		Dir: t.TempDir(),
	}
	ts.FilesDir = filepath.Join(ts.Dir, "files")
	ts.TS = &common.TestingSystem{
		Config: &config.Config{
			Invoker: &config.InvokerConfig{
				SandboxType:           sandboxType,
				SandboxHomePath:       filepath.Join(ts.Dir, "sandbox"),
				CacheSize:             1000 * 1000 * 1000,
				CachePath:             ts.FilesDir,
				CompilerConfigsFolder: "testdata/compiler",
			},
		},
	}
	require.NoError(t, os.MkdirAll(ts.TS.Config.Invoker.SandboxHomePath, 0o755))
	config.FillInInvokerConfig(ts.TS.Config.Invoker)
	ts.Invoker = &Invoker{
		TS:       ts.TS,
		Storage:  storage.NewInvokerStorage(ts.TS),
		Compiler: compiler.NewCompiler(ts.TS),
		RunnerThreads: &threadsExecutor[func()]{
			values:         make([]func(), 0),
			closed:         false,
			threadsCount:   1,
			name:           "test",
			waitDuration:   make([]time.Duration, 1),
			lastActiveTime: make([]time.Time, 1),
			isActive:       make([]bool, 1),
		},
	}
	ts.Invoker.RunnerThreads.cond = sync.NewCond(&ts.Invoker.RunnerThreads.mutex)
	go func() {
		ts.Invoker.RunnerThreads.runThread(1, func(f func()) {
			f()
		})
	}()
	require.NoError(t, os.CopyFS(ts.FilesDir, os.DirFS("testdata/files")))
	ts.Sandbox = ts.Invoker.newSandbox(1)
	return ts
}

func (ts *testState) testCompile(submitID uint) *JobPipelineState {
	job := &Job{
		Job: invokerconn.Job{
			ID:       "JOB",
			SubmitID: submitID,
			Type:     invokerconn.CompileJob,
		},
		problem: &models.Problem{
			ID: 1,
		},
		submission: &models.Submission{
			ID:        submitID,
			ProblemID: 1,
			Language:  "cpp",
		},
	}

	require.NoError(ts.t, ts.Invoker.Storage.Source.Insert(
		ts.Invoker.Storage.GetEpoch(),
		fmt.Sprintf("%s/source/%d/%d.cpp", ts.FilesDir, submitID, submitID),
		uint64(submitID),
	))

	s := ts.Invoker.newPipelineState(ts.Sandbox, job)
	s.compile = new(pipelineCompileData)
	s.loggerData = fmt.Sprintf("compile job: %s submission: %d", job.ID, job.submission.ID)

	require.NoError(ts.t, s.compilationProcessPipeline())

	return s
}

func TestCompile(t *testing.T) {
	t.Run("Simple sandbox", func(t *testing.T) { testCompileSandbox(t, "simple") })

	t.Run("Isolate sandbox", func(t *testing.T) {
		_, err := os.Stat("/usr/local/bin/isolate")
		if err != nil {
			t.Skip("No isolate installed on current device, skipping isolate tests")
		} else {
			testCompileSandbox(t, "isolate")
		}
	})

}

func testCompileSandbox(t *testing.T, sandboxType string) {
	ts := newTestState(t, sandboxType)

	s := ts.testCompile(1)
	require.Equal(t, verdict.CD, s.compile.result.Verdict)

	cmd := exec.Command(filepath.Join(s.sandbox.Dir(), solutionBinaryFile))
	var stdout strings.Builder
	cmd.Stdout = &stdout

	require.NoError(t, cmd.Run())
	require.Equal(t, "1", strings.TrimSpace(stdout.String()))

	s.finish()

	s = ts.testCompile(2)
	require.Equal(t, verdict.CE, s.compile.result.Verdict)
	s.finish()

	ts.Invoker.RunnerThreads.stop()
}

func (ts *testState) addProblem(problemID uint) {
	require.NoError(ts.t, ts.Invoker.Storage.TestInput.Insert(
		ts.Invoker.Storage.GetEpoch(),
		fmt.Sprintf("%s/test_input/%d-1/1", ts.FilesDir, problemID),
		uint64(problemID), 1,
	))

	require.NoError(ts.t, ts.Invoker.Storage.TestAnswer.Insert(
		ts.Invoker.Storage.GetEpoch(),
		fmt.Sprintf("%s/test_answer/%d-1/1.a", ts.FilesDir, problemID),
		uint64(problemID), 1,
	))

	checkerDir := fmt.Sprintf("%s/checker/%d", ts.FilesDir, problemID)

	testlib, err := os.ReadFile(filepath.Join(ts.FilesDir, "checker", "testlib.h"))
	require.NoError(ts.t, err)
	require.NoError(ts.t, os.WriteFile(filepath.Join(checkerDir, "testlib.h"), testlib, 0666))

	cmd := exec.Command("g++", "check.cpp", "-std=c++17", "-o", "check")
	cmd.Dir = checkerDir
	require.NoError(ts.t, cmd.Run())

	require.NoError(ts.t, ts.Invoker.Storage.Checker.Insert(
		ts.Invoker.Storage.GetEpoch(),
		filepath.Join(checkerDir, "check"),
		uint64(problemID),
	))
}

func (ts *testState) prepareTestRun(submitID uint, problemID uint) *JobPipelineState {
	job := &Job{
		Job: invokerconn.Job{
			ID:       "JOB",
			SubmitID: submitID,
			Type:     invokerconn.TestJob,
			Test:     1,
		},
		problem: &models.Problem{
			ID:          problemID,
			TestsNumber: 1,
		},
		submission: &models.Submission{
			ID:        submitID,
			ProblemID: 1,
			Language:  "cpp",
		},
	}
	job.problem.TimeLimit.FromStr("1s")
	job.problem.MemoryLimit.FromStr("100m")

	sourceDir := fmt.Sprintf("%s/binary/%d", ts.FilesDir, submitID)
	cmd := exec.Command("g++", "source.cpp", "-std=c++17", "-o", "binary")
	cmd.Dir = sourceDir
	require.NoError(ts.t, cmd.Run())

	require.NoError(ts.t, ts.Invoker.Storage.Binary.Insert(
		ts.Invoker.Storage.GetEpoch(),
		filepath.Join(sourceDir, "binary"),
		uint64(submitID),
	))

	s := ts.Invoker.newPipelineState(ts.Sandbox, job)
	s.test = new(pipelineTestData)
	s.loggerData = fmt.Sprintf(
		"test job: %s submission: %d problem %d test %d",
		job.ID,
		job.submission.ID,
		job.problem.ID,
		job.Test,
	)

	return s
}

func (ts *testState) testRun(submitID uint, problemID uint) *sandbox.RunResult {
	s := ts.prepareTestRun(submitID, problemID)
	defer s.finish()

	require.NoError(ts.t, s.testingProcessPipeline())

	return s.test.runResult
}

func TestRun(t *testing.T) {
	t.Run("Simple sandbox", func(t *testing.T) { testRunSandbox(t, "simple") })

	t.Run("Isolate sandbox", func(t *testing.T) {
		_, err := os.Stat("/usr/local/bin/isolate")
		if err != nil {
			t.Skip("No isolate installed on current device, skipping isolate tests")
		} else {
			testRunSandbox(t, "isolate")
		}
	})

}

func testRunSandbox(t *testing.T, sandboxType string) {
	ts := newTestState(t, sandboxType)
	ts.addProblem(1)

	res := ts.testRun(3, 1)
	require.Equal(t, verdict.OK, res.Verdict)

	res = ts.testRun(4, 1)
	require.Equal(t, verdict.RT, res.Verdict)

	res = ts.testRun(5, 1)
	require.Equal(t, verdict.TL, res.Verdict)

	res = ts.testRun(6, 1)
	require.Equal(t, verdict.WA, res.Verdict)

	if sandboxType == "isolate" {
		// Simple sandbox does not support ML verdict
		res = ts.testRun(7, 1)
		require.Equal(t, verdict.ML, res.Verdict)
	}

	ts.addProblem(2)
	res = ts.testRun(8, 2)
	require.Equal(t, verdict.PT, res.Verdict)
	require.EqualValues(t, 5, *res.Points)

	{
		s := ts.prepareTestRun(9, 1)
		defer s.finish()
		startTime := time.Now()
		s.job.stopCtx, s.job.stopFunc = context.WithCancel(context.Background())
		go func() {
			time.Sleep(100 * time.Millisecond)
			s.job.stopFunc() // I can not use context.WithDeadline here as the ctx.Error() should be cancelled
		}()
		require.NoError(t, s.testingProcessPipeline())
		require.Less(t, time.Since(startTime), time.Millisecond*900)
		require.Equal(t, verdict.SK, s.test.runResult.Verdict)
	}

	ts.Invoker.RunnerThreads.stop()
}
