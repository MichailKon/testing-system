package models

import "gorm.io/gorm"

type Submission struct {
	gorm.Model
	ProblemID uint64
	Language  string

	Verdict string
}
