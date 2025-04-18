package models

import (
	"gorm.io/gorm"
	"testing_system/lib/customfields"
)

type ProblemType int

const (
	ProblemTypeStandard ProblemType = iota + 1
	ProblemTypeInteractive
)

type ScoringType int

const (
	ScoringTypeICPC ScoringType = iota + 1
	ScoringTypeIOI
)

type Problem struct {
	gorm.Model

	ProblemType ProblemType `gorm:"not null"`
	ScoringType ScoringType `gorm:"not null"`

	TimeLimit   customfields.Time
	MemoryLimit customfields.Memory

	TestsNumber uint64

	// WallTimeLimit specifies maximum execution and wait time.
	// By default, it is max(5s, TimeLimit * 2)
	WallTimeLimit *customfields.Time

	// MaxOpenFiles specifies maximum number of files, opened by testing system.
	// By default, it is 64
	MaxOpenFiles *uint64

	// MaxThreads specifies maximum number of threads and/or processes
	// By default, it is single thread
	// If MaxThreads equals to -1, any number of threads allowed
	MaxThreads *int64

	// MaxOutputSize specifies maximum output in EACH file.
	// By default, it is 1g
	MaxOutputSize *customfields.Memory
}
