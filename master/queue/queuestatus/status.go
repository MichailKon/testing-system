package queuestatus

import (
	"container/list"
	"github.com/google/uuid"
	"sync"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/db/models"
	"testing_system/lib/logger"
)

type QueueStatus struct {
	mutex sync.Mutex

	activeSubmissions          map[uint]*submissionHolder
	submissionsOrderedByUpdate *list.List
	isTesting                  bool
}

type submissionHolder struct {
	submission       *models.Submission
	lastUpdatedEpoch string
	listPosition     *list.Element
}

// NewQueueStatus creates new queue status.
// isTesting should be specified only for tests
func NewQueueStatus(isTesting bool) *QueueStatus {
	return &QueueStatus{
		activeSubmissions:          make(map[uint]*submissionHolder),
		submissionsOrderedByUpdate: list.New(),
		isTesting:                  isTesting,
	}
}

func (s *QueueStatus) AddSubmission(submission *models.Submission) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	_, ok := s.activeSubmissions[submission.ID]
	if ok {
		logger.Panic("submission %d is added to status twice", submission.ID)
	}
	holder := &submissionHolder{
		submission:       submission,
		lastUpdatedEpoch: s.newEpoch(),
	}
	s.activeSubmissions[submission.ID] = holder
	holder.listPosition = s.submissionsOrderedByUpdate.PushFront(holder)
}

func (s *QueueStatus) FinishSubmissionTesting(id uint) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	holder, ok := s.activeSubmissions[id]
	if !ok {
		logger.Panic("Removing submission %d from queue status that is not present in list", id)
	}
	delete(s.activeSubmissions, id)
	s.submissionsOrderedByUpdate.Remove(holder.listPosition)
}

func (s *QueueStatus) UpdateSubmission(submission *models.Submission) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.isTesting {
		return
	}

	holder, ok := s.activeSubmissions[submission.ID]
	if !ok {
		logger.Panic("Updating submission %d that is not added to queue status", submission.ID)
	}

	// We will not have data race with pointers here.
	// All pointers of types: TestResults, CompilationResult, GroupResult
	// can be only assigned to new values, but they never change
	submissionCopy := *submission
	copy(submissionCopy.TestResults, submission.TestResults)

	holder.lastUpdatedEpoch = s.newEpoch()
	holder.submission = &submissionCopy
	s.submissionsOrderedByUpdate.MoveToFront(holder.listPosition)
}

func (s *QueueStatus) GetStatus(prevEpoch string) *masterconn.Status {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	status := &masterconn.Status{
		Epoch:              s.newEpoch(),
		TestingSubmissions: make([]uint, 0),
		UpdatedSubmissions: make([]*models.Submission, 0),
	}

	for submissionID := range s.activeSubmissions {
		status.TestingSubmissions = append(status.TestingSubmissions, submissionID)
	}

	for element := s.submissionsOrderedByUpdate.Front(); element != nil; element = element.Next() {
		holder, ok := element.Value.(*submissionHolder)
		if !ok {
			logger.Panic("Status submissions list has invalid element %v", element.Value)
		}
		if holder.lastUpdatedEpoch < prevEpoch { // We use strict less here, as uuid theoretically may have same value
			break
		}
		status.UpdatedSubmissions = append(status.UpdatedSubmissions, holder.submission)
	}
	return status
}

func (s *QueueStatus) newEpoch() string {
	epoch, err := uuid.NewV7()
	if err != nil {
		logger.Panic("Can not generate new uuidv7")
		return "" // Never reach, just for linter
	}
	return epoch.String()
}
