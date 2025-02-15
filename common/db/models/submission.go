package models

import "gorm.io/gorm"

type Submission struct {
	gorm.Model
	ProblemID        uint64
	ProblemVersionID uint64
	TestingResultID  int
	TestingResult    TestingResult
}
