package invokerconn

import (
	models2 "testing_system/common/db/models"
)

// This will be changed in later commits

const (
	JobTypeCompile = iota
	JobTypeTest
)

type Job struct {
	SubmitID uint `json:"submitID" binding:"required"`
	JobType  int  `json:"jobType" binding:"required"`
	Test     int  `json:"test"`

	Submit  *models2.Submission `json:"submit,omitempty"`
	Problem *models2.Problem    `json:"problem,omitempty"`
}
