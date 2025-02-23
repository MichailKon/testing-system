package problem_queue

import (
	"container/list"
	"fmt"
	"gorm.io/gorm"
	"sync"
	"testing_system/common/problem_queue/problem_graph"
	"testing_system/db/models"
)

type TestingItem struct {
	ProblemID uint64
	Solution  uint64
}

type queueElement struct {
	problemGraph *problem_graph.Graph
	testingItem  TestingItem
	testingNow   bool
}

type Queue struct {
	db                   *gorm.DB
	problems             *list.List
	testingItemToProblem map[TestingItem]*list.Element
	mutex                sync.Mutex
}

func CreateQueue(db *gorm.DB) *Queue {
	return &Queue{
		db:                   db,
		problems:             list.New(),
		testingItemToProblem: make(map[TestingItem]*list.Element),
		mutex:                sync.Mutex{},
	}
}

func (q *Queue) AddProblem(testingItem TestingItem) error {
	var problem models.Problem
	if err := q.db.First(&problem, "problem_id = ?", testingItem.ProblemID).Error; err != nil {
		return fmt.Errorf("can't fetch problem %v with error %w", testingItem.ProblemID, err)
	}

	if graph, err := problem_graph.MakeGraph(problem); err == nil {
		q.problems.PushBack(&queueElement{
			problemGraph: graph,
			testingItem:  testingItem,
			testingNow:   false,
		})
		q.testingItemToProblem[testingItem] = q.problems.Back()
		return nil
	} else {
		return err
	}
}

func (q *Queue) GetNextTestAndSolution() (testingItem TestingItem, testNumber uint64, err error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	itemsCount := q.problems.Len()
	if itemsCount == 0 {
		return TestingItem{}, 0, fmt.Errorf("queue is empty")
	}
	for range itemsCount {
		problem := q.problems.Remove(q.problems.Front()).(*queueElement)
		if problem.testingNow {
			q.problems.PushBack(problem)
			continue
		}
		nextTest, err := problem.problemGraph.GetNextTestNumber()
		if err != nil {
			panic(err)
		}
		problem.testingNow = true
		q.problems.PushBack(problem)
		return problem.testingItem, nextTest, nil
	}
	return TestingItem{}, 0, fmt.Errorf("can't take any of the problems for testing")
}

func (q *Queue) SetTestStatus(testingItem TestingItem, testNumber uint64, testStatus problem_graph.TestStatus) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	node, ok := q.testingItemToProblem[testingItem]
	if !ok {
		return fmt.Errorf("can't find item=%+v, test=%v testing queue",
			testingItem, testNumber)
	}
	cur := node.Value.(*queueElement)
	if !cur.testingNow {
		return fmt.Errorf("test is already done (item=%+v, test=%v)",
			testingItem, testNumber)
	}
	cur.testingNow = false
	cur.problemGraph.SetTestStatus(testNumber, testStatus)
	if cur.problemGraph.IsProblemTestingCompleted() {
		delete(q.testingItemToProblem, testingItem)
		q.problems.Remove(node)
	}
	return nil
}
