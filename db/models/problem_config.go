package models

import "gorm.io/gorm"

type ProblemType int

const (
	ProblemType_ICPC ProblemType = iota + 1
	ProblemType_IOI
)

type ProblemConfig struct {
	gorm.Model
	ProblemType ProblemType
	// everything we need here
}
