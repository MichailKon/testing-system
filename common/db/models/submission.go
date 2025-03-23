package models

import (
	"gorm.io/gorm"
	"testing_system/common/constants/verdict"
)

type Submission struct {
	gorm.Model
	ProblemID uint64
	Language  string

	Verdict verdict.Verdict
}
