package masterstatus

import (
	"context"
	"sync"
	"testing_system/clients/common"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/lib/logger"
	"time"
)

type MasterStatus struct {
	base  *common.ClientBase
	mutex sync.Mutex

	submissions map[uint]*models.Submission

	status *masterconn.Status
}

func NewMasterStatus(clientBase *common.ClientBase) (*MasterStatus, error) {
	s := &MasterStatus{
		base:        clientBase,
		submissions: make(map[uint]*models.Submission),
	}

	var updateInterval time.Duration
	if clientBase.Config.TestingSystemAPI.StatusUpdateInterval > 0 {
		updateInterval = clientBase.Config.TestingSystemAPI.StatusUpdateInterval
	} else {
		updateInterval = time.Second
	}

	go s.runUpdateThread(updateInterval)

	return s, nil
}

func (s *MasterStatus) runUpdateThread(interval time.Duration) {
	logger.Info("Starting master status update thread")

	t := time.Tick(interval)
	for {
		select {
		case <-t:
			s.updateStatus()
		}
	}
}

func (s *MasterStatus) updateStatus() {
	s.mutex.Lock()
	var epoch string
	if s.status != nil {
		epoch = s.status.Epoch
	}
	s.mutex.Unlock()

	status, err := s.base.MasterConnection.GetStatus(context.Background(), epoch)
	if err != nil {
		logger.Error("Can not update master status, error: %v", err)
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.status = status

	for _, submission := range status.UpdatedSubmissions {
		s.submissions[submission.ID] = submission
	}

	s.status.UpdatedSubmissions = nil

	logger.Trace("Updated master status")
}

func (s *MasterStatus) GetSubmission(ctx context.Context, id uint) (*models.Submission, error) {
	var submission models.Submission
	err := s.base.DB.WithContext(ctx).First(&submission, id).Error
	if err != nil {
		return nil, err
	}
	if submission.Verdict == verdict.RU {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		updatedSubmission, ok := s.submissions[submission.ID]
		if ok {
			return updatedSubmission, nil
		} else {
			submission.TestResults = make(models.TestResults, 0)
		}
	}

	return &submission, nil
}

type SubmissionInList struct {
	ID          uint            `json:"id"`
	ProblemID   uint            `json:"problem_id"`
	Language    string          `json:"language"`
	Score       float64         `json:"score"`
	Verdict     verdict.Verdict `json:"verdict"`
	CurrentTest int             `json:"current_test,omitempty" gorm:"-"`
}

type SubmissionsFilter struct {
	Count int `form:"count" binding:"required"`
	Page  int `form:"page,default=1"`

	ProblemID *uint            `form:"problem_id,omitempty"`
	Verdict   *verdict.Verdict `form:"verdict,omitempty"`
	Language  *string          `form:"language,omitempty"`
}

func (s *MasterStatus) GetSubmissions(ctx context.Context, filter *SubmissionsFilter) ([]SubmissionInList, error) {
	request := s.base.DB.
		WithContext(ctx).
		Model(&models.Submission{}).
		Limit(filter.Count).
		Offset((filter.Page - 1) * filter.Count)

	if filter.ProblemID != nil {
		request = request.Where("problem_id=?", *filter.ProblemID)
	}
	if filter.Verdict != nil {
		request = request.Where("verdict=?", *filter.Verdict)
	}
	if filter.Language != nil {
		request = request.Where("language=?", *filter.Language)
	}
	var submissions []SubmissionInList

	err := request.
		Order("id desc").
		Find(&submissions).
		Error

	if err != nil {
		return nil, err
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for i := range submissions {
		if submissions[i].Verdict == verdict.RU {
			updatedSubmission, ok := s.submissions[submissions[i].ID]
			if ok {
				submissions[i].CurrentTest = len(updatedSubmission.TestResults)
			}
		}
	}

	return submissions, err
}

func (s *MasterStatus) Status() *masterconn.Status {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.status
}
