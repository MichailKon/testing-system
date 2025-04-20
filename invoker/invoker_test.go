package invoker

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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
)

type testState struct {
	t        *testing.T
	TS       *common.TestingSystem
	Invoker  *Invoker
	Sandbox  sandbox.ISandbox
	Sandbox2 sandbox.ISandbox
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
		RunQueue: make(chan func()),
	}
	go func() {
		for {
			f := <-ts.Invoker.RunQueue
			f()
		}
	}()
	require.NoError(t, os.CopyFS(ts.FilesDir, os.DirFS("testdata/files")))
	ts.Sandbox = newSandbox(ts.TS, 1)
	ts.Sandbox2 = newSandbox(ts.TS, 2)
	return ts
}

func (ts *testState) testCompile(submitID uint) *JobPipelineState {
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

	require.NoError(ts.t, ts.Invoker.Storage.Source.Insert(
		fmt.Sprintf("%s/source/%d/%d.cpp", ts.FilesDir, submitID, submitID),
		uint64(submitID),
	))

	s := &JobPipelineState{
		invoker:    ts.Invoker,
		sandbox:    ts.Sandbox,
		job:        job,
		compile:    new(pipelineCompileData),
		loggerData: fmt.Sprintf("compile job: %s submission: %d", job.ID, job.Submission.ID),
	}

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

	s.deferFunc()

	s = ts.testCompile(2)
	require.Equal(t, verdict.CE, s.compile.result.Verdict)
	s.deferFunc()
}

func (ts *testState) addProblem(problemID uint, problemType models.ProblemType) {
	require.NoError(ts.t, ts.Invoker.Storage.TestInput.Insert(
		fmt.Sprintf("%s/test_input/%d-1/1", ts.FilesDir, problemID),
		uint64(problemID), 1,
	))

	require.NoError(ts.t, ts.Invoker.Storage.TestAnswer.Insert(
		fmt.Sprintf("%s/test_answer/%d-1/1.a", ts.FilesDir, problemID),
		uint64(problemID), 1,
	))

	checkerDir := fmt.Sprintf("%s/checker/%d", ts.FilesDir, problemID)

	testlib, err := os.ReadFile(filepath.Join(ts.FilesDir, "testlib.h"))
	require.NoError(ts.t, err)
	require.NoError(ts.t, os.WriteFile(filepath.Join(checkerDir, "testlib.h"), testlib, fileModeText))

	cmd := exec.Command("g++", "check.cpp", "-std=c++17", "-o", "check")
	cmd.Dir = checkerDir
	require.NoError(ts.t, cmd.Run())
	require.NoError(ts.t, ts.Invoker.Storage.Checker.Insert(filepath.Join(checkerDir, "check"), uint64(problemID)))

	if problemType == models.ProblemTypeInteractive {
		interactorDir := fmt.Sprintf("%s/interactor/%d", ts.FilesDir, problemID)
		require.NoError(ts.t, os.WriteFile(filepath.Join(interactorDir, "testlib.h"), testlib, fileModeText))

		cmd = exec.Command("g++", "interactor.cpp", "-std=c++17", "-o", "interactor")
		cmd.Dir = interactorDir
		require.NoError(ts.t, cmd.Run())
		require.NoError(ts.t, ts.Invoker.Storage.Interactor.Insert(filepath.Join(interactorDir, "interactor"), uint64(problemID)))
	}
}

func (ts *testState) testRun(submitID uint, problemID uint, problemType models.ProblemType) *sandbox.RunResult {
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
			ProblemType: problemType,
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

	switch problemType {
	case models.ProblemTypeStandard:
		return ts.runStandardJob(job)
	case models.ProblemTypeInteractive:
		return ts.runInteractiveJob(job)
	default:
		ts.t.Fatalf("Unknown problem type")
	}
	return nil
}

func (ts *testState) runStandardJob(job *Job) *sandbox.RunResult {
	s := &JobPipelineState{
		job:     job,
		invoker: ts.Invoker,
		sandbox: ts.Sandbox,
		test:    new(pipelineTestData),
		loggerData: fmt.Sprintf(
			"test job: %s submission: %d problem %d test %d",
			job.ID,
			job.Submission.ID,
			job.Problem.ID,
			job.Test,
		),
	}

	defer s.deferFunc()

	require.NoError(ts.t, s.standardTestingPipeline())

	return s.test.runResult
}

func (ts *testState) runInteractiveJob(solutionJob *Job) *sandbox.RunResult {
	interactorJob := createInteractorJob(solutionJob)
	var solutionWait sync.WaitGroup
	solutionWait.Add(1)
	go func() {
		ts.Invoker.fullInteractiveSolutionPipeline(ts.Sandbox2, solutionJob)
		solutionWait.Done()
	}()

	s := &JobPipelineState{
		job:     interactorJob,
		invoker: ts.Invoker,
		sandbox: ts.Sandbox,
		test:    new(pipelineTestData),
	}
	s.job.InteractiveData.PipelineReadyWG.Wait()

	// Two jobs share common interaction field to wait each other before execution begins
	s.interaction = s.job.InteractiveData.SolutionPipelineState.interaction
	s.defers = append(s.defers, s.interaction.solutionRelease)
	defer s.deferFunc()

	require.NoError(ts.t, s.interactiveTestingPipeline())

	solutionWait.Wait()

	return s.test.runResult
}

func TestStandardRun(t *testing.T) {
	t.Run("Simple sandbox", func(t *testing.T) { testSandboxStandardRun(t, "simple") })

	t.Run("Isolate sandbox", func(t *testing.T) {
		_, err := os.Stat("/usr/local/bin/isolate")
		if err != nil {
			t.Skip("No isolate installed on current device, skipping isolate tests")
		} else {
			testSandboxStandardRun(t, "isolate")
		}
	})

}

func testSandboxStandardRun(t *testing.T, sandboxType string) {
	ts := newTestState(t, sandboxType)
	ts.addProblem(1, models.ProblemTypeStandard)

	res := ts.testRun(3, 1, models.ProblemTypeStandard)
	require.Equal(t, verdict.OK, res.Verdict)

	res = ts.testRun(4, 1, models.ProblemTypeStandard)
	require.Equal(t, verdict.RT, res.Verdict)

	res = ts.testRun(5, 1, models.ProblemTypeStandard)
	require.Equal(t, verdict.TL, res.Verdict)

	res = ts.testRun(6, 1, models.ProblemTypeStandard)
	require.Equal(t, verdict.WA, res.Verdict)

	res = ts.testRun(7, 1, models.ProblemTypeStandard)
	require.Equal(t, verdict.ML, res.Verdict)

	ts.addProblem(2, models.ProblemTypeStandard)
	res = ts.testRun(8, 2, models.ProblemTypeStandard)
	require.Equal(t, verdict.PT, res.Verdict)
	require.EqualValues(t, 5, *res.Points)
}

func TestInteractiveRun(t *testing.T) {
	t.Run("Simple sandbox", func(t *testing.T) { testSandboxInteractiveRun(t, "simple") })

	t.Run("Isolate sandbox", func(t *testing.T) {
		_, err := os.Stat("/usr/local/bin/isolate")
		if err != nil {
			t.Skip("No isolate installed on current device, skipping isolate tests")
		} else {
			testSandboxInteractiveRun(t, "isolate")
		}
	})
}

func testSandboxInteractiveRun(t *testing.T, sandboxType string) {
	ts := newTestState(t, sandboxType)
	ts.addProblem(3, models.ProblemTypeInteractive)

	res := ts.testRun(9, 3, models.ProblemTypeInteractive)
	require.Equal(t, verdict.OK, res.Verdict)

	// Not enough printed
	res = ts.testRun(10, 3, models.ProblemTypeInteractive)
	require.Equal(t, verdict.WR, res.Verdict)

	// Interaction is ok, but wrong answer is printed
	res = ts.testRun(11, 3, models.ProblemTypeInteractive)
	require.Equal(t, verdict.WR, res.Verdict)

	// RT
	res = ts.testRun(4, 3, models.ProblemTypeInteractive)
	require.Equal(t, verdict.WR, res.Verdict)

	// TL
	res = ts.testRun(5, 3, models.ProblemTypeInteractive)
	require.Equal(t, verdict.WR, res.Verdict)

	// ML
	res = ts.testRun(7, 3, models.ProblemTypeInteractive)
	require.Equal(t, verdict.WR, res.Verdict)
}
