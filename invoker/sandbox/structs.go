package sandbox

import (
	"bytes"
	"testing_system/common/constants/verdict"
	"testing_system/lib/customfields"
)

type RunConfig struct {
	TL customfields.TimeLimit   `yaml:"TL" json:"TL"`
	ML customfields.MemoryLimit `yaml:"ML" json:"ML"`
	WL customfields.TimeLimit   `yaml:"WL" json:"WL"`
}

type RunResult struct {
	Err error

	Verdict verdict.Verdict

	Stdout *bytes.Buffer
	Stderr *bytes.Buffer
}
