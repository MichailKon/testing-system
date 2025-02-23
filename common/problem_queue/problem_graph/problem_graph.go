package problem_graph

import (
	"fmt"
	"testing_system/db/models"
)

type TestStatus int

const (
	TestFailed TestStatus = iota
	TestOk
	TestUnknown
)

// Graph provides testing structure of a problem
// It can give you next test that should be tested
// NB: Not thread-safe
type Graph struct {
	graph          [][]uint64 // graph[testInd] = {tests, which needs testInd to be OK in order to be tested}
	needToBeTested []int      // needToBeTested[testInd] = how many tests must be done to start this test
	testsStatus    []TestStatus
}

func generateIcpcProblemGraph(problem models.Problem) *Graph {
	g := &Graph{
		graph:          make([][]uint64, problem.TestsCount),
		testsStatus:    make([]TestStatus, problem.TestsCount),
		needToBeTested: make([]int, problem.TestsCount),
	}
	for i := range g.testsStatus {
		g.testsStatus[i] = TestUnknown
	}
	for i := uint64(0); i+1 < problem.TestsCount; i++ {
		g.graph[i] = []uint64{i + 1}
		g.needToBeTested[i+1] = 1
	}
	g.needToBeTested[0] = 0
	return g
}

func MakeGraph(problem models.Problem) (*Graph, error) {
	switch problem.ProblemType {
	case models.ProblemType_ICPC:
		return generateIcpcProblemGraph(problem), nil
	case models.ProblemType_IOI:
		return nil, fmt.Errorf("unsupported problem type IOI")
	default:
		return nil, fmt.Errorf("unknown ProblemType for problemID=%v", problem.ID)
	}
}

func (g *Graph) SetTestStatus(testNumber uint64, testStatus TestStatus) {
	g.testsStatus[testNumber] = testStatus
	if testStatus == TestOk || testStatus == TestFailed {
		for _, to := range g.graph[testNumber] {
			g.needToBeTested[to]--
		}
	}
}

func (g *Graph) IsProblemTestingCompleted() bool {
	for _, testType := range g.testsStatus {
		if testType == TestUnknown {
			return false
		}
	}
	return true
}

func (g *Graph) GetNextTestNumber() (uint64, error) {
	for i := range g.graph {
		if g.needToBeTested[i] == 0 && g.testsStatus[i] == TestUnknown {
			return uint64(i), nil
		}
	}
	return 0, fmt.Errorf("no tests needed for testing")
}
