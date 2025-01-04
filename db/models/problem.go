package models

import "gorm.io/gorm"

type ProblemType int

const (
	ProblemType_ICPC ProblemType = iota + 1
	ProblemType_IOI
)

type Problem struct {
	gorm.Model
	ProblemType ProblemType
}
