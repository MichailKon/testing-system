package models

import (
	"gorm.io/gorm"
	"testing_system/lib/customfields"
)

type Problem struct {
	gorm.Model

	TimeLimit     customfields.TimeLimit
	MemoryLimit   customfields.MemoryLimit
	WallTimeLimit *customfields.TimeLimit

	TestsNumber uint64

	MaxOpenFiles  *uint64
	MaxThreads    *uint64
	MaxOutputSize *customfields.MemoryLimit
}
