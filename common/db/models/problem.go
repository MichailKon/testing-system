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

	TimeLimit     customfields.TimeLimit
	MemoryLimit   customfields.MemoryLimit
	WallTimeLimit *customfields.TimeLimit

	TestsNumber uint64

	MaxOpenFiles  *uint64
	MaxThreads    *uint64
	MaxOutputSize *customfields.MemoryLimit
}
