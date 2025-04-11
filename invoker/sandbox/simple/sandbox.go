package simple

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/invoker/sandbox"
	"testing_system/lib/customfields"
	"testing_system/lib/logger"
	"time"
)

type Sandbox struct {
	dir         string
	initialized bool
}

func NewSandbox(dir string) (*Sandbox, error) {
	err := os.RemoveAll(dir)
	if err != nil {
		return nil, err
	}

	logger.Warn("Using simple sandbox is not safe. Consider using isolate sandbox")

	return &Sandbox{
		dir:         dir,
		initialized: false,
	}, nil
}

func (s *Sandbox) Init() error {
	if s.initialized {
		return fmt.Errorf("sandbox already initialized")
	}
	err := os.MkdirAll(s.dir, 0777)
	if err != nil {
		return err
	}
	s.initialized = true
	return nil
}

func (s *Sandbox) Dir() string {
	return s.dir
}

func (s *Sandbox) Run(config *sandbox.ExecuteConfig) *sandbox.RunResult {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.WallTimeLimit))
	defer cancel()
	cmd := exec.CommandContext(
		ctx,
		filepath.Join(s.dir, config.Command),
		config.Args...,
	)

	cmd.Dir = s.dir
	cmd.Stdin = config.Stdin
	cmd.Stdout = config.Stdout
	cmd.Stderr = config.Stderr

	wallTimeLimit := false
	cmd.Cancel = func() error {
		wallTimeLimit = true
		return cmd.Process.Kill()
	}

	err := cmd.Run()

	result := &sandbox.RunResult{
		Statistics: &masterconn.JobResultStatistics{},
	}
	if cmd.ProcessState == nil {
		result.Err = fmt.Errorf("sandbox process state is nil, something wrong with sandbox, process error: %v", err)
		return result
	}

	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		result.Err = fmt.Errorf("sandbox process exited with unknown error: %v", err)
		return result
	}

	rusage := cmd.ProcessState.SysUsage().(*syscall.Rusage)
	result.Statistics.Time = customfields.Time(rusage.Utime.Nano())
	result.Statistics.Memory = customfields.Memory(rusage.Maxrss)
	if runtime.GOOS != "darwin" { // We have macOS defined for tests!
		result.Statistics.Memory *= 1024
	}
	result.Statistics.WallTime = result.Statistics.Time
	result.Statistics.ExitCode = cmd.ProcessState.ExitCode()

	if result.Statistics.ExitCode != 0 && !wallTimeLimit {
		result.Verdict = verdict.RT
	} else if result.Statistics.Time > config.TimeLimit {
		result.Verdict = verdict.TL
	} else if result.Statistics.Memory > config.MemoryLimit {
		result.Verdict = verdict.ML
	} else if wallTimeLimit {
		result.Verdict = verdict.WL
	} else {
		result.Verdict = verdict.OK
	}

	return result
}

func (s *Sandbox) Cleanup() {
	err := os.RemoveAll(s.dir)
	if err != nil {
		logger.Error("Can not clean up sandbox, error: %v", err)
	} else {
		s.initialized = false
	}
}

func (s *Sandbox) Delete() {
	if s.initialized {
		logger.Error("sandbox %s was initialized before delete", s.dir)
	}
}
