package queuestatus

import (
	"container/list"
	"sync"
	"testing_system/common/db/models"
)

type QueueStatus struct {
	mutex sync.Mutex

	activeSubmissions          map[uint]*submissionHolder
	submissionsOrderedByUpdate *list.List
}

type submissionHolder struct {
	submission *models.Submission
}
