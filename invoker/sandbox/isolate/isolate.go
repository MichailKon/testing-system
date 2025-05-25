package isolate

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/invoker/sandbox"
	"testing_system/lib/customfields"
	"testing_system/lib/logger"
	"time"
)

const (
	isolateCommand   = "/usr/local/bin/isolate"
	bytesInKB        = 1024
	memoryAdditional = 1024
	timeAdditional   = 0.001
	extraTime        = 0.5
)

type Sandbox struct {
	id           int
	localHomeDir string
	initialized  bool
}

func NewSandbox(id int, localHomeDir string) (*Sandbox, error) {
	s := &Sandbox{
		id:           id,
		localHomeDir: localHomeDir,
	}

	_, err := os.Stat("/usr/local/bin/isolate")
	if err != nil {
		return nil, fmt.Errorf("can not find isolate, error: %v", err)
	}

	_, err = os.Stat(s.Dir())
	if err == nil {
		logger.Warn("Isolate box %d already existed, cleaning up", s.id)
		s.Cleanup()
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("can not stat isolate box %d, error: %v", s.id, err)
	}
	logger.Info("Initialized isolate box %d", id)
	return s, nil
}

func (s *Sandbox) Dir() string {
	return fmt.Sprintf("/var/local/lib/isolate/%d/box", s.id)
}

func (s *Sandbox) command(ctx context.Context) *exec.Cmd {
	return exec.CommandContext(ctx, isolateCommand, "--cg", fmt.Sprintf("--box-id=%d", s.id), "-s")
}

func (s *Sandbox) metaPath() string {
	return filepath.Join(s.localHomeDir, "meta-"+strconv.Itoa(s.id))
}

func (s *Sandbox) Init() error {
	cmd := s.command(context.Background())
	cmd.Args = append(cmd.Args, "--init")
	err := cmd.Run()
	if err != nil {
		return err
	}
	s.initialized = true
	return nil
}

func (s *Sandbox) Cleanup() {
	if !s.initialized {
		logger.Error("Cleaning up uninitialized sandbox")
		return
	}
	cmd := s.command(context.Background())
	cmd.Args = append(cmd.Args, "--cleanup")
	err := cmd.Run()
	if err != nil {
		logger.Error("Can not clean up sandbox, error: %v", err)
	}
	s.initialized = false
}

func (s *Sandbox) Delete() {
	if s.initialized {
		logger.Error("sandbox %d was initialized before delete", s.id)
		s.Cleanup()
	}
}

func (s *Sandbox) Run(config *sandbox.ExecuteConfig) *sandbox.RunResult {
	cmd := s.prepareRun(config)

	result := &sandbox.RunResult{
		Statistics: &masterconn.JobResultStatistics{},
	}

	skipped := false
	cmd.Cancel = func() error {
		skipped = true
		return cmd.Process.Kill()
	}

	err := cmd.Run()

	if skipped {
		result.Verdict = verdict.SK
		return result
	}

	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			result.Err = fmt.Errorf("error running isolate box %d, error: %v", s.id, err)
			return result
		}
	}

	s.parseMeta(config, result)
	return result
}

func (s *Sandbox) prepareRun(config *sandbox.ExecuteConfig) *exec.Cmd {
	ctx := context.Background()
	if config.Ctx != nil {
		ctx = config.Ctx
	}
	cmd := s.command(ctx)
	cmd.Args = append(cmd.Args, "--run")

	// We are changing workdir, just in case
	cmd.Dir = s.localHomeDir
	cmd.Args = append(cmd.Args, "--meta="+s.metaPath())

	// We initialize path ENV so that compilers will work
	cmd.Args = append(cmd.Args, "--env=PATH=/usr/bin")

	cmd.Args = append(cmd.Args, fmt.Sprintf("--time=%f", float64(config.TimeLimit)/float64(time.Second)+timeAdditional))
	cmd.Args = append(cmd.Args, fmt.Sprintf("--extra-time=%f", extraTime))
	cmd.Args = append(cmd.Args, fmt.Sprintf("--wall-time=%f", float64(config.WallTimeLimit)/float64(time.Second)))

	cmd.Args = append(cmd.Args, fmt.Sprintf("--cg-mem=%d", uint64(config.MemoryLimit)/bytesInKB+memoryAdditional))

	if config.MaxThreads != 0 {
		if config.MaxThreads == -1 {
			cmd.Args = append(cmd.Args, "--processes")
		} else {
			cmd.Args = append(cmd.Args, "--processes", fmt.Sprintf("%d", config.MaxThreads))
		}
	}

	cmd.Args = append(cmd.Args, fmt.Sprintf("--open-files=%d", config.MaxOpenFiles))
	cmd.Args = append(cmd.Args, fmt.Sprintf("--fsize=%d", config.MaxOutputSize/bytesInKB))

	if config.Stdin != nil {
		if config.Stdin.Input != nil {
			cmd.Stdin = config.Stdin.Input
		} else {
			cmd.Args = append(cmd.Args, "--stdin="+config.Stdin.FileName)
		}
	}

	if config.Stdout != nil {
		if config.Stdout.Output != nil {
			cmd.Stdout = config.Stdout.Output
		} else {
			cmd.Args = append(cmd.Args, "--stdout="+config.Stdout.FileName)
		}
	}

	if config.StderrToStdout {
		cmd.Args = append(cmd.Args, "--stderr-to-stdout")
	} else if config.Stderr != nil {
		if config.Stderr.Output != nil {
			cmd.Stderr = config.Stderr.Output
		} else {
			cmd.Args = append(cmd.Args, "--stderr="+config.Stderr.FileName)
		}
	}

	cmd.Args = append(cmd.Args, "--")
	cmd.Args = append(cmd.Args, config.Command)
	cmd.Args = append(cmd.Args, config.Args...)

	return cmd
}

func (s *Sandbox) parseMeta(config *sandbox.ExecuteConfig, result *sandbox.RunResult) {
	reader, err := os.Open(s.metaPath())
	if err != nil {
		result.Err = fmt.Errorf("can not open meta file, error: %v", err)
		return
	}
	defer reader.Close()

	result.Verdict = verdict.OK

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 2)
		if len(parts) != 2 {
			result.Err = fmt.Errorf("can not parse meta file line %s", scanner.Text())
			return
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		switch key {
		case "cg-mem":
			err = result.Statistics.Memory.FromStr(value + "k")
			if err != nil {
				result.Err = fmt.Errorf("can not parse meta file line %s", scanner.Text())
				return
			}
		case "cg-oom-killed":
			if value == "1" {
				result.Verdict = verdict.ML
			}
		case "exitcode":
			result.Statistics.ExitCode, err = strconv.Atoi(value)
			if err != nil {
				result.Err = fmt.Errorf("can not parse meta file line %s", scanner.Text())
				return
			}
		case "exitsig":
			result.Statistics.ExitCode, err = strconv.Atoi(value)
			if err != nil {
				result.Err = fmt.Errorf("can not parse meta file line %s", scanner.Text())
				return
			}
			if result.Verdict == verdict.OK {
				result.Verdict = verdict.RT
			}
		case "status":
			switch value {
			case "RE", "SG":
				if result.Verdict == verdict.OK {
					result.Verdict = verdict.RT
				}
			case "TO":
				result.Verdict = verdict.WL
			case "XX":
				result.Err = fmt.Errorf("unknown error from isolate")
				return
			default:
				result.Err = fmt.Errorf("can not parse meta file line %s", scanner.Text())
				return
			}
		case "time":
			timeVal, err := strconv.ParseFloat(value, 64)
			if err != nil {
				result.Err = fmt.Errorf("can not parse meta file line %s", scanner.Text())
				return
			}
			result.Statistics.Time = customfields.Time(timeVal * float64(time.Second))
		case "time-wall":
			timeVal, err := strconv.ParseFloat(value, 64)
			if err != nil {
				result.Err = fmt.Errorf("can not parse meta file line %s", scanner.Text())
				return
			}
			result.Statistics.WallTime = customfields.Time(timeVal * float64(time.Second))
		case "csw-forced", "csw-voluntary", "killed", "max-rss", "message":
			// skip
		default:
			result.Err = fmt.Errorf("can not parse meta file line %s", scanner.Text())
			return
		}
	}

	setVerdict := func(toSet verdict.Verdict, condition bool, canHaveVerdicts ...verdict.Verdict) {
		if slices.Contains(canHaveVerdicts, result.Verdict) && condition {
			result.Verdict = toSet
		}
	}

	setVerdict(verdict.ML, result.Statistics.Memory > config.MemoryLimit, verdict.OK, verdict.RT)
	setVerdict(verdict.TL, result.Statistics.Time > config.TimeLimit, verdict.OK, verdict.WL)
	setVerdict(verdict.WL, result.Statistics.WallTime > config.WallTimeLimit, verdict.OK)
	setVerdict(verdict.RT, result.Statistics.ExitCode != 0, verdict.OK)

	// TODO: Support SE
}
