package invoker

import (
	"fmt"
	"github.com/stretchr/testify/require"
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
	}
	require.NoError(t, os.CopyFS(ts.FilesDir, os.DirFS("testdata/files")))
	ts.Executor = NewJobExecutor(ts.TS, 1)
	return ts
}

func (ts *testState) testCompile(submitID uint) (file *string, v verdict.Verdict) {
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
	require.NoError(ts.t, ts.Executor.Sandbox.Init())

	j := compileJob{
		invoker: ts.Invoker,
		tester:  ts.Executor,
		job:     job,
	}

	require.NoError(ts.t, j.invoker.Storage.Source.Insert(
		fmt.Sprintf("%s/source/%d/%d.cpp", ts.FilesDir, submitID, submitID),
		uint64(submitID),
	))

	require.NoError(ts.t, j.Prepare())

	j.wg.Add(1)
	j.Execute()

	require.NoError(ts.t, j.compileResult.Err)

	_, err := j.FinalizeVerdict()
	require.NoError(ts.t, err)

	v = j.compileResult.Verdict
	if j.compileResult.Verdict == verdict.CD {
		file = pointer.String(filepath.Join(j.tester.Sandbox.Dir(), j.binaryName))
	}
	return
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

	file, v := ts.testCompile(1)
	require.Equal(t, verdict.CD, v)

	cmd := exec.Command(*file)
	var stdout strings.Builder
	cmd.Stdout = &stdout

	require.NoError(t, cmd.Run())
	require.Equal(t, "1", strings.TrimSpace(stdout.String()))

	ts.Executor.Sandbox.Cleanup()

	_, v = ts.testCompile(2)
	require.Equal(t, verdict.CE, v)
}

func (ts *testState) addProblem(problemID uint) {
	require.NoError(ts.t, ts.Invoker.Storage.TestInput.Insert(
		fmt.Sprintf("%s/test_input/%d-1/1", ts.FilesDir, problemID),
		uint64(problemID), 1,
	))

	require.NoError(ts.t, ts.Invoker.Storage.TestAnswer.Insert(
		fmt.Sprintf("%s/test_answer/%d-1/1.a", ts.FilesDir, problemID),
		uint64(problemID), 1,
	))

	checkerDir := fmt.Sprintf("%s/checker/%d", ts.FilesDir, problemID)
	cmd := exec.Command("g++", "check.cpp", "-std=c++17", "-o", "check")
	cmd.Dir = checkerDir
	require.NoError(ts.t, cmd.Run())

	require.NoError(ts.t, ts.Invoker.Storage.Checker.Insert(filepath.Join(checkerDir, "check"), uint64(problemID)))
}

func (ts *testState) testRun(submitID uint, problemID uint) *sandbox.RunResult {
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
	require.NoError(ts.t, cmd.Run())

	require.NoError(ts.t, ts.Invoker.Storage.Binary.Insert(filepath.Join(sourceDir, "binary"), uint64(submitID)))

	require.NoError(ts.t, ts.Executor.Sandbox.Init())
	defer ts.Executor.Sandbox.Cleanup()

	j := testJob{
		invoker: ts.Invoker,
		tester:  ts.Executor,
		job:     job,
	}

	require.NoError(ts.t, j.PrepareRun())

	j.wg.Add(1)
	j.RunOnTest()
	require.NoError(ts.t, j.runResult.Err)

	if j.runResult.Verdict != verdict.OK {
		return j.runResult
	}

	require.NoError(ts.t, j.PrepareCheck())

	j.wg.Add(1)
	j.RunChecker()

	require.NoError(ts.t, j.checkResult.Err)

	switch j.checkResult.Verdict {
	case verdict.OK, verdict.RT:
		j.ParseCheckerOutput()
	default:
		ts.t.Fatalf("Wrong checker verdict: %v", j.checkResult.Verdict)
	}

	return j.runResult
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

	res = ts.testRun(7, 1)
	require.Equal(t, verdict.ML, res.Verdict)

	ts.addProblem(2)
	res = ts.testRun(8, 2)
	require.Equal(t, verdict.PT, res.Verdict)
	require.EqualValues(t, 5, *res.Points)
}
