package sandbox

import (
	"io"
	"slices"
	"testing_system/common/config"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
)

type ExecuteConfig struct {
	config.RunConfig

	Command string   `yaml:"-"`
	Args    []string `yaml:"-"` // Except zero argument (command name itself)

	Stdin  io.Reader `yaml:"-"`
	Stdout io.Writer `yaml:"-"`
	Stderr io.Writer `yaml:"-"`

	Defers []func() `yaml:"-"`
}

func (e *ExecuteConfig) DeferFunc() {
	slices.Reverse(e.Defers)
	for _, f := range e.Defers {
		f()
	}
	e.Defers = nil
}

type RunResult struct {
	Err error

	Verdict verdict.Verdict

	Statistics *masterconn.JobResultStatistics

	Points *float64 // Not used by sandbox, but can be set up by other components
}
