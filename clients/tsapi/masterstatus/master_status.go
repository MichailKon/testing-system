package masterstatus

import (
	"context"
	"testing_system/clients/common"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
)

type MasterStatus struct {
	base *common.ClientBase
}

func NewMasterStatus(clientBase *common.ClientBase) (*MasterStatus, error) {
	return &MasterStatus{
		base: clientBase,
	}, nil
}

func (m *MasterStatus) GetSubmission(ctx context.Context, id uint, submission *models.Submission) error {
	// TODO: Get submission from master get while testing is not completed
	return m.base.DB.WithContext(ctx).First(submission, id).Error
}

type SubmissionInList struct {
	ID        uint            `json:"id"`
	ProblemID uint            `json:"problem_id"`
	Language  string          `json:"language"`
	Score     float64         `json:"score"`
	Verdict   verdict.Verdict `json:"verdict"`
}

type SubmissionsFilter struct {
	Count int `form:"count" binding:"required"`
	Page  int `form:"page,default=1"`

	ProblemID *uint            `form:"problem_id,omitempty"`
	Verdict   *verdict.Verdict `form:"verdict,omitempty"`
	Language  *string          `form:"language,omitempty"`
}

func (m *MasterStatus) GetSubmissions(ctx context.Context, filter *SubmissionsFilter) ([]SubmissionInList, error) {
	reqest := m.base.DB.
		WithContext(ctx).
		Model(&models.Submission{}).
		Limit(filter.Count).
		Offset((filter.Page - 1) * filter.Count)

	if filter.ProblemID != nil {
		reqest = reqest.Where("problem_id=?", *filter.ProblemID)
	}
	if filter.Verdict != nil {
		reqest = reqest.Where("verdict=?", *filter.Verdict)
	}
	if filter.Language != nil {
		reqest = reqest.Where("language=?", *filter.Language)
	}
	var submissions []SubmissionInList

	err := reqest.
		Order("id desc").
		Find(&submissions).
		Error
	return submissions, err
}
