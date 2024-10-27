package models

import "gorm.io/gorm"

type Submission struct {
	gorm.Model
	TestingResult
	ContentId     uint64 // id of submission in the storage
	TaskId        uint64
	TaskVersionId uint64
}
