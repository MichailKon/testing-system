package queue

import "testing_system/common/db/models"

type SubmissionHolder struct {
	Submission *models.Submission
	Problem    *models.Problem
}
