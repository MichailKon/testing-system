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

	ProblemType ProblemType

	TimeLimit     customfields.Time
	MemoryLimit   customfields.Memory
	WallTimeLimit *customfields.Time

	TestsNumber uint64

	MaxOpenFiles  *uint64
	MaxThreads    *uint64
	MaxOutputSize *customfields.Memory
}
