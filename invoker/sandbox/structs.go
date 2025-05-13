package sandbox

import (
	"context"
	"io"
	"testing_system/common/config"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
)

type ExecuteConfig struct {
	config.RunLimitsConfig `yaml:",inline"`

	Command string   `yaml:"-"`
	Args    []string `yaml:"-"` // Except zero argument (command name itself)

	Stdin          *IORedirect `yaml:"-"`
	Stdout         *IORedirect `yaml:"-"`
	Stderr         *IORedirect `yaml:"-"`
	StderrToStdout bool        `yaml:"-"`

	Ctx context.Context
}

// IORedirect specifies files to read/write to.
// Either Input, Output or FileName should be specified
// FileName should be relative inside sandbox
type IORedirect struct {
	Input    io.Reader `yaml:"-"`
	Output   io.Writer `yaml:"-"`
	FileName string    `yaml:"-"`
}

type RunResult struct {
	Err error

	Verdict verdict.Verdict

	Statistics *masterconn.JobResultStatistics

	Points *float64 // Not used by sandbox, but can be set up by other components
}
