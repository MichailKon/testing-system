package jobgenerators

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/xorcare/pointer"
	"math"
	"slices"
	"sync"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/lib/logger"
)

type IOIGenerator struct {
	id                      string
	mutex                   sync.Mutex
	submission              *models.Submission
	problem                 *models.Problem
	state                   generatorState
	givenJobs               map[string]*invokerconn.Job
	groupNameToGroupInfo    map[string]models.TestGroup
	groupNameToInternalInfo map[string]*internalGroupInfo
	testNumberToGroupName   map[uint64]string
	internalTestResults     []*internalTestResult

	// firstNotCompletedTest = the longest prefix of the tests, for which we know verdict
	firstNotCompletedTest uint64
	// firstNotCompletedGroup = the longest prefix of the groups, for which we know verdict
	firstNotCompletedGroup uint64
	firstUnseenTest        uint64
}

type internalGroupState int

const (
	groupRunning internalGroupState = iota
	groupFailed                     // if the group isn't failed and completed, it will still have groupRunning state
)

type internalTestState int

const (
	testNotGiven internalTestState = iota
	testRunning
	testFinished
)

type internalGroupInfo struct {
	state           internalGroupState
	shouldSkipTests bool
}

type internalTestResult struct {
	result *models.TestResult
	state  internalTestState
}

func (i *IOIGenerator) checkIfGroupsDependenciesOK(problem *models.Problem) error {
	if !slices.IsSortedFunc(problem.TestGroups, func(a, b models.TestGroup) int {
		return int(a.FirstTest) - int(b.FirstTest)
	}) {
		return fmt.Errorf("groups are not sorted by tests")
	}
	// each test should be in exactly one group, and each group should depend on the earlier ones
	lastTest := uint64(0)
	encounteredGroups := make(map[string]struct{})
	for _, group := range problem.TestGroups {
		if group.FirstTest != lastTest+1 {
			return fmt.Errorf("at least one test is not in exactly one group")
		}
		lastTest = group.LastTest
		if _, ok := encounteredGroups[group.Name]; ok {
			return fmt.Errorf("duplicate test group: %s", group.Name)
		}
		for _, requiredGroupName := range group.RequiredGroupNames {
			if _, ok := encounteredGroups[requiredGroupName]; !ok {
				return fmt.Errorf("missing required group (maybe it is later) %v for group %v",
					requiredGroupName, group.Name)
			}
			requiredGroupInfo := i.groupNameToGroupInfo[requiredGroupName]
			if requiredGroupInfo.ScoringType != models.TestGroupScoringTypeComplete {
				return fmt.Errorf("group %v depends on group %v with scoringType=%v",
					group.Name, requiredGroupInfo.Name, requiredGroupInfo.ScoringType)
			}
		}
		encounteredGroups[group.Name] = struct{}{}
	}
	if lastTest != problem.TestsNumber {
		return fmt.Errorf("at least one test is not in exactly one group")
	}
	return nil
}

func (i *IOIGenerator) prepareGenerator() error {
	problem := i.problem
	if problem.ProblemType != models.ProblemTypeIOI {
		return fmt.Errorf("problem %v is not an IOI problem", problem.ID)
	}
	// each group with TestGroupScoringTypeEachTest must have TestScore
	for _, group := range problem.TestGroups {
		switch group.ScoringType {
		case models.TestGroupScoringTypeComplete, models.TestGroupScoringTypeMin:
			if group.GroupScore == nil {
				return fmt.Errorf("group %v in problem %v doesn't have GroupScore", group.Name, problem.ID)
			}
		case models.TestGroupScoringTypeEachTest:
			if group.TestScore == nil {
				return fmt.Errorf("group %v in problem %v doesn't have TestScore", group.Name, problem.ID)
			}
		default:
			return fmt.Errorf("unknown TestGroupScoringType %v", group.ScoringType)
		}
		for testNumber := group.FirstTest; testNumber <= group.LastTest; testNumber++ {
			i.testNumberToGroupName[testNumber-1] = group.Name
			i.internalTestResults = append(i.internalTestResults, &internalTestResult{
				result: &models.TestResult{
					TestNumber: testNumber,
					Verdict:    verdict.RU,
				},
				state: testNotGiven,
			})
		}
		i.groupNameToGroupInfo[group.Name] = group
		i.groupNameToInternalInfo[group.Name] = &internalGroupInfo{
			state:           groupRunning,
			shouldSkipTests: false,
		}
	}
	return i.checkIfGroupsDependenciesOK(problem)
}

func (i *IOIGenerator) ID() string {
	return i.id
}

func (i *IOIGenerator) NextJob() *invokerconn.Job {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	if i.state == compilationFinished && i.firstUnseenTest > i.problem.TestsNumber {
		return nil
	}
	if i.state == compilationStarted {
		return nil
	}
	id, err := uuid.NewV7()
	if err != nil {
		logger.Panic("Can't generate id for job: %w", err)
	}
	job := &invokerconn.Job{
		ID:       id.String(),
		SubmitID: i.submission.ID,
	}
	if i.state == compilationNotStarted {
		job.Type = invokerconn.CompileJob
		i.state = compilationStarted
		i.givenJobs[job.ID] = job
		return job
	}
	job.Type = invokerconn.TestJob
	for i.firstUnseenTest < i.problem.TestsNumber {
		groupName := i.testNumberToGroupName[i.firstUnseenTest]
		groupInfo := i.groupNameToInternalInfo[groupName]
		if groupInfo.shouldSkipTests {
			i.firstUnseenTest++
			continue
		}
		i.internalTestResults[i.firstUnseenTest].state = testRunning
		i.firstUnseenTest++
		job.Test = i.firstUnseenTest
		i.givenJobs[job.ID] = job
		return job
	}
	return nil
}

func isTestFailed(testGroupInfo models.TestGroup, testVerdict verdict.Verdict) bool {
	switch testGroupInfo.ScoringType {
	case models.TestGroupScoringTypeComplete:
		return testVerdict != verdict.OK
	case models.TestGroupScoringTypeEachTest:
		return testVerdict != verdict.OK
	case models.TestGroupScoringTypeMin:
		return testVerdict != verdict.OK
	default:
		panic(fmt.Sprintf("unknown testGroupInfo.ScoringType %v", testGroupInfo.ScoringType))
	}
}

func shouldSkipNewTests(testGroupInfo models.TestGroup, testVerdict verdict.Verdict) bool {
	switch testGroupInfo.ScoringType {
	case models.TestGroupScoringTypeComplete:
		return testVerdict != verdict.OK
	case models.TestGroupScoringTypeEachTest:
		return false
	case models.TestGroupScoringTypeMin:
		return testVerdict != verdict.OK && testVerdict != verdict.PT
	default:
		panic(fmt.Sprintf("unknown testGroupInfo.ScoringType %v", testGroupInfo.ScoringType))
	}
}

// increaseCompletedTestsAndGroups must be done with acquired mutex
func (i *IOIGenerator) increaseCompletedTestsAndGroups() {
TestsLoop:
	for i.firstNotCompletedTest != i.problem.TestsNumber {
		testGroupName := i.testNumberToGroupName[i.firstNotCompletedTest]
		testGroupInfo := i.groupNameToGroupInfo[testGroupName]
		testInternalGroupInfo := i.groupNameToInternalInfo[testGroupName]
		switch i.internalTestResults[i.firstNotCompletedTest].state {
		case testNotGiven:
			if !testInternalGroupInfo.shouldSkipTests {
				break TestsLoop
			}
		case testRunning:
			break TestsLoop
		case testFinished:
		}
		i.submission.TestResults = append(i.submission.TestResults,
			*i.internalTestResults[i.firstNotCompletedTest].result)
		if !testInternalGroupInfo.shouldSkipTests {
			if shouldSkipNewTests(testGroupInfo, i.submission.TestResults[i.firstNotCompletedTest].Verdict) {
				testInternalGroupInfo.shouldSkipTests = true
				// update shouldSkipTests
				for _, group := range i.problem.TestGroups {
					for _, requiredGroupName := range group.RequiredGroupNames {
						if i.groupNameToInternalInfo[requiredGroupName].shouldSkipTests {
							i.groupNameToInternalInfo[group.Name].shouldSkipTests = true
						}
					}
				}
			}
		} else {
			i.submission.TestResults[i.firstNotCompletedTest].Verdict = verdict.SK
		}
		if i.firstNotCompletedTest+1 == testGroupInfo.LastTest {
			score := 0.0
			switch testGroupInfo.ScoringType {
			case models.TestGroupScoringTypeComplete:
				if testInternalGroupInfo.state != groupFailed {
					score = *testGroupInfo.GroupScore
				}
			case models.TestGroupScoringTypeEachTest:
				for testNumber := testGroupInfo.FirstTest - 1; testNumber < testGroupInfo.LastTest; testNumber++ {
					if testScore := i.submission.TestResults[testNumber].Points; testScore != nil {
						score += *testScore
					}
				}
			case models.TestGroupScoringTypeMin:
				score = math.Inf(+1)
				for testNumber := testGroupInfo.FirstTest - 1; testNumber < testGroupInfo.LastTest; testNumber++ {
					if testScore := i.submission.TestResults[testNumber].Points; testScore != nil {
						score = min(score, *testScore)
					} else {
						score = 0.0
					}
				}
				if score == math.Inf(+1) {
					score = 0
				}
			}
			i.submission.GroupResults = append(i.submission.GroupResults, models.GroupResult{
				GroupName: testGroupName,
				Points:    score,
				Passed:    testInternalGroupInfo.state != groupFailed,
			})
			i.firstNotCompletedGroup++
		}
		i.firstNotCompletedTest++
	}
}

// compileJobCompleted must be done with acquired mutex
func (i *IOIGenerator) compileJobCompleted(job *invokerconn.Job, result *masterconn.InvokerJobResult) (*models.Submission, error) {
	if job.Type != invokerconn.CompileJob {
		return nil, fmt.Errorf("job type %s is not compile job", job.ID)
	}
	switch result.Verdict {
	case verdict.CD:
		i.state = compilationFinished
		return nil, nil
	case verdict.CE:
		i.submission.Verdict = verdict.CE
		for _, group := range i.problem.TestGroups {
			i.submission.GroupResults = append(i.submission.GroupResults, models.GroupResult{
				GroupName: group.Name,
				Points:    0,
				Passed:    false,
			})
		}
		for t := range i.problem.TestsNumber {
			i.submission.TestResults = append(i.submission.TestResults, models.TestResult{
				TestNumber: t + 1,
				Verdict:    verdict.SK,
			})
		}
		return i.submission, nil
	default:
		return nil, fmt.Errorf("unknown verdict for compilation completed: %v", result.Verdict)
	}
}

func populateTestJobResult(groupInfo models.TestGroup, result *masterconn.InvokerJobResult) error {
	switch groupInfo.ScoringType {
	case models.TestGroupScoringTypeComplete:
	case models.TestGroupScoringTypeEachTest:
		if result.Points == nil {
			if result.Verdict == verdict.OK {
				result.Points = groupInfo.TestScore
			} else if result.Verdict == verdict.PT {
				return fmt.Errorf("checker returned nil points, but verdict=%v", result.Verdict)
			} else {
				result.Points = pointer.Float64(0)
			}
		}
	case models.TestGroupScoringTypeMin:
		if result.Points == nil {
			if result.Verdict == verdict.OK {
				result.Points = groupInfo.GroupScore
				return nil
			} else if result.Verdict == verdict.PT {
				return fmt.Errorf("checker returned nil points, but verdict=%v", result.Verdict)
			} else {
				result.Points = pointer.Float64(0)
			}
		}
	default:
		panic(fmt.Sprintf("unknown group scoring type: %v", groupInfo.ScoringType))
	}
	return nil
}

// testJobCompleted must be done with acquired mutex
func (i *IOIGenerator) testJobCompleted(job *invokerconn.Job, result *masterconn.InvokerJobResult) (*models.Submission, error) {
	if job.Type != invokerconn.TestJob {
		return nil, fmt.Errorf("job type %v is not test job", job.ID)
	}
	test := job.Test - 1
	testGroupName := i.testNumberToGroupName[test]
	testGroupInfo := i.groupNameToGroupInfo[testGroupName]
	testInternalGroupInfo := i.groupNameToInternalInfo[testGroupName]
	// move info from result to internal state
	if err := populateTestJobResult(testGroupInfo, result); err != nil {
		result.Verdict = verdict.CF
		if result.Points != nil {
			result.Points = pointer.Float64(0)
		}
		if result.Error != "" {
			result.Error += "; "
		}
		result.Error += err.Error()
	}
	i.internalTestResults[test].result.Points = result.Points
	i.internalTestResults[test].result.Verdict = result.Verdict
	i.internalTestResults[test].result.Error = result.Error
	if result.Statistics != nil {
		i.internalTestResults[test].result.Time = result.Statistics.Time
		i.internalTestResults[test].result.Memory = result.Statistics.Memory
	}
	i.internalTestResults[test].state = testFinished

	if isTestFailed(testGroupInfo, result.Verdict) {
		testInternalGroupInfo.state = groupFailed
		for _, group := range i.problem.TestGroups {
			for _, requiredGroupName := range group.RequiredGroupNames {
				if i.groupNameToInternalInfo[requiredGroupName].state == groupFailed {
					i.groupNameToInternalInfo[group.Name].state = groupFailed
					break
				}
			}
		}
	}
	i.increaseCompletedTestsAndGroups()
	if i.firstNotCompletedTest == i.problem.TestsNumber && len(i.givenJobs) == 0 {
		if len(i.submission.GroupResults) != len(i.problem.TestGroups) {
			panic("for some reason \"len(i.submission.GroupResults) != len(i.problem.TestGroups)\"")
		}
		if len(i.submission.TestResults) != int(i.problem.TestsNumber) {
			panic("for some reason \"len(i.submission.TestResults) != int(i.problem.TestsNumber)\"")
		}
		for _, groupResult := range i.submission.GroupResults {
			i.submission.Score += groupResult.Points
		}

		haveSkipped := false
		haveVerdict := false
		for _, testResult := range i.submission.TestResults {
			if testResult.Verdict != verdict.OK && testResult.Verdict != verdict.SK {
				i.submission.Verdict = verdict.PT
				haveVerdict = true
			} else if testResult.Verdict == verdict.SK {
				haveSkipped = true
			}
		}
		if !haveVerdict {
			if haveSkipped {
				panic("problem does not have verdict, but have skipped tests")
			}
			i.submission.Verdict = verdict.OK
		}
		return i.submission, nil
	}
	return nil, nil
}

func (i *IOIGenerator) JobCompleted(jobResult *masterconn.InvokerJobResult) (*models.Submission, error) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	job, ok := i.givenJobs[jobResult.JobID]
	if !ok {
		return nil, fmt.Errorf("job %s does not exist", jobResult.JobID)
	}
	delete(i.givenJobs, jobResult.JobID)
	switch job.Type {
	case invokerconn.CompileJob:
		return i.compileJobCompleted(job, jobResult)
	case invokerconn.TestJob:
		return i.testJobCompleted(job, jobResult)
	default:
		return nil, fmt.Errorf("unknown job type for IOI problem: %v", job.Type)
	}
}

func NewIOIGenerator(problem *models.Problem, submission *models.Submission) (Generator, error) {
	id, err := uuid.NewV7()
	if err != nil {
		logger.Panic("Can't generate generator id: %w", err)
	}
	generator := &IOIGenerator{
		id:                      id.String(),
		submission:              submission,
		problem:                 problem,
		state:                   compilationNotStarted,
		givenJobs:               make(map[string]*invokerconn.Job),
		groupNameToGroupInfo:    make(map[string]models.TestGroup),
		groupNameToInternalInfo: make(map[string]*internalGroupInfo),
		testNumberToGroupName:   make(map[uint64]string),
		internalTestResults:     make([]*internalTestResult, 0),
		firstNotCompletedTest:   0,
		firstNotCompletedGroup:  0,
		firstUnseenTest:         0,
	}
	if err = generator.prepareGenerator(); err != nil {
		return nil, err
	}
	return generator, nil
}
