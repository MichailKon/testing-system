package tester

import (
	"testing_system/common"
	"testing_system/common/db/models"
	"testing_system/invoker/storage"
)

type Tester struct {
	ID      int
	Storage *storage.InvokerStorage
}

func (t *Tester) Test(submit *models.Submission) {

}

func (t *Tester) Cleanup() {

}

func NewTester(
	ts *common.TestingSystem,
	id int,
	storage *storage.InvokerStorage,
) *Tester {
	return &Tester{
		ID:      id,
		Storage: storage,
	}
}
