package invokerconn

import (
	"testing_system/common/db/models"
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

	Submit  *models.Submission `json:"submit,omitempty"`
	Problem *models.Problem    `json:"problem,omitempty"`
}
