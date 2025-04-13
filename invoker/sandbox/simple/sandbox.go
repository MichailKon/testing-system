package simple

import (
	"context"
	"errors"
	"fmt"
	"io"
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

func (s *Sandbox) parseReader(r *io.Reader, conf *sandbox.IORedirect) (func() error, error) {
	if conf == nil {
		return nil, nil
	}
	if conf.Input != nil {
		*r = conf.Input
		return nil, nil
	}
	if conf.Output != nil {
		return nil, fmt.Errorf("writer is specified for reading")
	}
	if len(conf.FileName) == 0 {
		return nil, fmt.Errorf("no source is specified for IORedirect")
	}

	file := filepath.Join(s.dir, conf.FileName)
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	*r = fd
	return fd.Close, nil
}

func (s *Sandbox) parseWriter(r *io.Writer, conf *sandbox.IORedirect) (func() error, error) {
	if conf == nil {
		return nil, nil
	}
	if conf.Input != nil {
		return nil, fmt.Errorf("reader is specified for writing")
	}
	if conf.Output != nil {
		*r = conf.Output
		return nil, nil
	}
	if len(conf.FileName) == 0 {
		return nil, fmt.Errorf("no source is specified for IORedirect")
	}

	file := filepath.Join(s.dir, conf.FileName)
	fd, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	*r = fd
	return fd.Close, nil
}

func (s *Sandbox) Run(config *sandbox.ExecuteConfig) *sandbox.RunResult {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.WallTimeLimit))
	defer cancel()
	cmd := exec.CommandContext(
		ctx,
		filepath.Join(s.dir, config.Command),
		config.Args...,
	)

	result := &sandbox.RunResult{
		Statistics: &masterconn.JobResultStatistics{},
	}

	cmd.Dir = s.dir
	closer, err := s.parseReader(&cmd.Stdin, config.Stdin)
	if err != nil {
		result.Err = fmt.Errorf("can not parse stdin: %v", err)
	}
	if closer != nil {
		defer closer()
	}

	closer, err = s.parseWriter(&cmd.Stdout, config.Stdout)
	if err != nil {
		result.Err = fmt.Errorf("can not parse stdout: %v", err)
	}
	if closer != nil {
		defer closer()
	}

	closer, err = s.parseWriter(&cmd.Stderr, config.Stderr)
	if err != nil {
		result.Err = fmt.Errorf("can not parse stderr: %v", err)
	}
	if closer != nil {
		defer closer()
	}

	wallTimeLimit := false
	cmd.Cancel = func() error {
		wallTimeLimit = true
		return cmd.Process.Kill()
	}

	err = cmd.Run()

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
	if !s.initialized {
		logger.Error("Cleaning up uninitialized sandbox")
		return
	}
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
