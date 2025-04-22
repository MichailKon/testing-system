package models

import (
	"gorm.io/gorm"
	"testing_system/lib/customfields"
)

type ProblemType int

const (
	ProblemType_ICPC ProblemType = iota + 1
	ProblemType_IOI
)

type Problem struct {
	gorm.Model

	ProblemType ProblemType `yaml:"ProblemType"`

	TimeLimit   customfields.Time   `yaml:"TimeLimit"`
	MemoryLimit customfields.Memory `yaml:"MemoryLimit"`

	TestsNumber uint64 `yaml:"TestsNumber"`

	// WallTimeLimit specifies maximum execution and wait time.
	// By default, it is max(5s, TimeLimit * 2)
	WallTimeLimit *customfields.Time `yaml:"WallTimeLimit,omitempty"`

	// MaxOpenFiles specifies maximum number of files, opened by testing system.
	// By default, it is 64
	MaxOpenFiles *uint64 `yaml:"MaxOpenFiles,omitempty"`

	// MaxThreads specifies maximum number of threads and/or processes
	// By default, it is single thread
	// If MaxThreads equals to -1, any number of threads allowed
	MaxThreads *int64 `yaml:"MaxThreads,omitempty"`

	// MaxOutputSize specifies maximum output in EACH file.
	// By default, it is 1g
	MaxOutputSize *customfields.Memory `yaml:"MaxOutputSize,omitempty"`
}
