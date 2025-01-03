package models

import "gorm.io/gorm"

type Submission struct {
	gorm.Model
	ContentId       uint64 // id of submission in the storage
	TaskId          uint64
	TaskVersionId   uint64
	TestingResultId int
	TestingResult   TestingResult
}
