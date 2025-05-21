package jobgenerators

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/xorcare/pointer"
	"slices"
	"sync"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/lib/logger"
	"testing_system/master/queue/queuestatus"
)

type IOIGenerator struct {
	id                      string
	mutex                   sync.Mutex
	submission              *models.Submission
	problem                 *models.Problem
	state                   generatorState
	givenJobs               map[string]*invokerconn.Job
	groupNameToGroupInfo    map[string]*models.TestGroup
	groupNameToInternalInfo map[string]*internalGroupInfo
	testNumberToGroupName   map[uint64]string
	internalTestResults     []*internalTestResult

	// firstNotCompletedTest = the longest prefix of the tests, for which we know verdict; 1-based indexing
	firstNotCompletedTest uint64
	// firstNotCompletedGroup = the longest prefix of the groups, for which we know verdict; 1-based indexing
	firstNotCompletedGroup uint64
	// firstNotGivenTest = first test with internalTestState = testNotGiven; 1-based indexing
	firstNotGivenTest uint64

	statusUpdater *queuestatus.QueueStatus
}

type internalTestState int

const (
	testNotGiven internalTestState = iota
	testRunning
	testFinished
)

type internalGroupInfo struct {
	shouldGiveNewJobs           bool
	shouldMarkFinalTestsSkipped bool
}

type internalTestResult struct {
	result *models.TestResult
	state  internalTestState
}

func (i *IOIGenerator) checkIfGroupsDependenciesOK(problem *models.Problem) error {
	if !slices.IsSortedFunc(problem.TestGroups, func(a, b *models.TestGroup) int {
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
		if group.FirstTest > group.LastTest {
			return fmt.Errorf("group first test is greater than the last one")
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
		if group.FirstTest > group.LastTest {
			return fmt.Errorf("group %v has FirstTest > LastTest", group.Name)
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
		i.groupNameToGroupInfo[group.Name] = group.Copy()
		i.groupNameToInternalInfo[group.Name] = &internalGroupInfo{
			shouldGiveNewJobs:           true,
			shouldMarkFinalTestsSkipped: false,
		}
	}
	return i.checkIfGroupsDependenciesOK(problem)
}

func (i *IOIGenerator) ID() string {
	return i.id
}

func (i *IOIGenerator) doesGroupDependOnJob(groupName string, job *invokerconn.Job) bool {
	groupInfo := i.groupNameToGroupInfo[groupName]
	jobGroupName := i.testNumberToGroupName[job.Test-1]
	if jobGroupName == groupName && groupInfo.ScoringType != models.TestGroupScoringTypeEachTest {
		return true
	}
	return slices.Contains(groupInfo.RequiredGroupNames, jobGroupName)
}

func (i *IOIGenerator) NextJob() *invokerconn.Job {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	if i.state == compilationFinished && i.firstNotGivenTest > i.problem.TestsNumber {
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
	for i.firstNotGivenTest <= i.problem.TestsNumber {
		groupName := i.testNumberToGroupName[i.firstNotGivenTest-1]
		groupInfo := i.groupNameToInternalInfo[groupName]
		if !groupInfo.shouldGiveNewJobs {
			i.internalTestResults[i.firstNotGivenTest-1].state = testFinished
			i.firstNotGivenTest++
			continue
		}
		i.internalTestResults[i.firstNotGivenTest-1].state = testRunning
		job.Test = i.firstNotGivenTest

		for givenJobID, testingJob := range i.givenJobs {
			if i.doesGroupDependOnJob(groupName, testingJob) {
				job.RequiredJobIDs = append(job.RequiredJobIDs, givenJobID)
			}
		}
		i.givenJobs[job.ID] = job
		i.firstNotGivenTest++
		return job
	}
	return nil
}

func doesTestPreventTestingGroup(testGroupInfo *models.TestGroup, testVerdict verdict.Verdict) bool {
	switch testGroupInfo.ScoringType {
	case models.TestGroupScoringTypeComplete:
		return testVerdict != verdict.OK
	case models.TestGroupScoringTypeEachTest:
		return false
	case models.TestGroupScoringTypeMin:
		return testVerdict != verdict.OK && testVerdict != verdict.PT
	default:
		logger.Panic("unknown testGroupInfo.ScoringType %v", testGroupInfo.ScoringType)
	}
	return false
}

func (i *IOIGenerator) calcGroupVerdict(groupInfo *models.TestGroup) verdict.Verdict {
	firstTest, lastTest := groupInfo.FirstTest, groupInfo.LastTest
	if slices.IndexFunc(i.submission.TestResults[firstTest-1:lastTest], func(result *models.TestResult) bool {
		return result.Verdict != verdict.OK
	}) != -1 {
		return verdict.PT
	} else {
		return verdict.OK
	}
}

func (i *IOIGenerator) completeGroupTesting(groupInfo *models.TestGroup) {
	if i.firstNotCompletedTest != groupInfo.LastTest {
		logger.Panic("completeGroupTesting called, but wasn't finished right now")
	}
	score := 0.0
	groupVerdict := i.calcGroupVerdict(groupInfo)
	switch groupInfo.ScoringType {
	case models.TestGroupScoringTypeComplete:
		if groupVerdict == verdict.OK {
			score = *groupInfo.GroupScore
		}
	case models.TestGroupScoringTypeEachTest:
		for testNumber := groupInfo.FirstTest; testNumber <= groupInfo.LastTest; testNumber++ {
			testScore := i.submission.TestResults[testNumber-1].Points
			if testScore == nil {
				logger.Panic("test %v has <nil> points in group %v", testNumber, groupInfo.Name)
			} else {
				score += *testScore
			}
		}
	case models.TestGroupScoringTypeMin:
		score = *groupInfo.GroupScore
		for testNumber := groupInfo.FirstTest; testNumber <= groupInfo.LastTest; testNumber++ {
			testScore := i.submission.TestResults[testNumber-1].Points
			if testScore == nil {
				if i.submission.TestResults[testNumber-1].Verdict != verdict.SK {
					logger.Panic("test %v has <nil> points in group %v", testNumber, groupInfo.Name)
				} else {
					score = 0
				}
			} else {
				score = min(score, *testScore)
			}
		}
	}
	i.submission.GroupResults = append(i.submission.GroupResults, models.GroupResult{
		GroupName: groupInfo.Name,
		Points:    score,
		Passed:    groupVerdict == verdict.OK,
	})
	i.firstNotCompletedGroup++
}

// updateSubmissionResult must be done with acquired mutex
func (i *IOIGenerator) updateSubmissionResult() (*models.Submission, error) {
	updated := false
	defer func() {
		if updated {
			i.statusUpdater.UpdateSubmission(i.submission)
		}
	}()

TestsLoop:
	for i.firstNotCompletedTest <= i.problem.TestsNumber {
		testGroupName := i.testNumberToGroupName[i.firstNotCompletedTest-1]
		testGroupInfo := i.groupNameToGroupInfo[testGroupName]
		testInternalGroupInfo := i.groupNameToInternalInfo[testGroupName]
		switch i.internalTestResults[i.firstNotCompletedTest-1].state {
		case testNotGiven:
			if !testInternalGroupInfo.shouldMarkFinalTestsSkipped {
				break TestsLoop
			}
		case testRunning:
			break TestsLoop
		case testFinished:
			break
		}
		updated = true

		if testInternalGroupInfo.shouldMarkFinalTestsSkipped {
			i.submission.TestResults = append(i.submission.TestResults,
				&models.TestResult{
					TestNumber: i.internalTestResults[i.firstNotCompletedTest-1].result.TestNumber,
					Verdict:    verdict.SK,
				})
		} else {
			testResult := i.internalTestResults[i.firstNotCompletedTest-1].result
			if testResult.Verdict == verdict.SK {
				testResult.Verdict = verdict.CF
				testResult.Error = "invoker returned verdict SK, but no required test failed"
			}

			i.submission.TestResults = append(
				i.submission.TestResults,
				testResult,
			)
			if doesTestPreventTestingGroup(testGroupInfo, testResult.Verdict) {
				testInternalGroupInfo.shouldMarkFinalTestsSkipped = true
				for _, group := range i.problem.TestGroups {
					for _, requiredGroupName := range group.RequiredGroupNames {
						if i.groupNameToInternalInfo[requiredGroupName].shouldMarkFinalTestsSkipped {
							i.groupNameToInternalInfo[group.Name].shouldMarkFinalTestsSkipped = true
						}
					}
				}
			}
		}

		if i.firstNotCompletedTest == testGroupInfo.LastTest {
			i.completeGroupTesting(testGroupInfo)
		}
		i.firstNotCompletedTest++
	}
	if i.firstNotCompletedTest > i.problem.TestsNumber && len(i.givenJobs) == 0 {
		updated = true
		i.setFinalScoreAndVerdict()
		if int(i.firstNotCompletedGroup) <= len(i.problem.TestGroups) {
			logger.Panic("not all the groups were filled, but the problem is considered tested")
		}
		return i.submission, nil
	}
	return nil, nil
}

// compileJobCompleted must be done with acquired mutex
func (i *IOIGenerator) compileJobCompleted(
	job *invokerconn.Job,
	result *masterconn.InvokerJobResult,
) {
	if job.Type != invokerconn.CompileJob {
		logger.Warn("job type %s is %v; treating is as a compile job", job.ID, job.Type)
	}
	switch result.Verdict {
	case verdict.CD:
		i.state = compilationFinished
	case verdict.CE:
		i.submission.Verdict = verdict.CE
		for _, group := range i.problem.TestGroups {
			i.groupNameToInternalInfo[group.Name].shouldMarkFinalTestsSkipped = true
		}
	default:
		if result.Verdict != verdict.CF {
			result.Verdict = verdict.CF
			result.Error = fmt.Sprintf("unknown verdict for compile job: %v", result.Verdict)
		}
		i.submission.Verdict = verdict.CF
		for _, group := range i.problem.TestGroups {
			i.groupNameToInternalInfo[group.Name].shouldMarkFinalTestsSkipped = true
		}
	}
	i.submission.CompilationResult = buildTestResult(job, result)
}

func populateTestJobResult(groupInfo *models.TestGroup, result *masterconn.InvokerJobResult) error {
	switch groupInfo.ScoringType {
	case models.TestGroupScoringTypeComplete:
		break
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
		logger.Panic("unknown group scoring type: %v", groupInfo.ScoringType)
	}
	return nil
}

func (i *IOIGenerator) stopGivingNewTestsIfNeeded(
	testInternalGroupInfo *internalGroupInfo,
	testGroupInfo *models.TestGroup,
	testVerdict verdict.Verdict,
) {
	if !doesTestPreventTestingGroup(testGroupInfo, testVerdict) {
		return
	}

	testInternalGroupInfo.shouldGiveNewJobs = false
	for _, group := range i.problem.TestGroups {
		for _, requiredGroupName := range group.RequiredGroupNames {
			if !i.groupNameToInternalInfo[requiredGroupName].shouldGiveNewJobs {
				i.groupNameToInternalInfo[group.Name].shouldGiveNewJobs = false
				break
			}
		}
	}
}

func (i *IOIGenerator) setFinalScoreAndVerdict() {
	if len(i.givenJobs) != 0 {
		logger.Panic("setFinalScoreAndVerdict called, but there are some jobs still")
	}
	if i.firstNotCompletedTest != i.problem.TestsNumber+1 {
		logger.Panic("setFinalScoreAndVerdict called, but not all the tests were completed")
	}
	if len(i.submission.GroupResults) != len(i.problem.TestGroups) {
		logger.Panic("for some reason \"len(i.submission.GroupResults) != len(i.problem.TestGroups)\"")
	}
	if len(i.submission.TestResults) != int(i.problem.TestsNumber) {
		logger.Panic("for some reason \"len(i.submission.TestResults) != int(i.problem.TestsNumber)\"")
	}
	for _, groupResult := range i.submission.GroupResults {
		i.submission.Score += groupResult.Points
	}
	if i.submission.Verdict != verdict.RU {
		logger.Trace("submission %v already has verdict %v", i.submission, i.submission.Verdict)
		return
	}

	hasSkipped := false
	hasVerdict := false
	for _, testResult := range i.submission.TestResults {
		if testResult.Verdict == verdict.CF {
			i.submission.Verdict = verdict.CF
			hasVerdict = true
		} else if testResult.Verdict == verdict.SK {
			hasSkipped = true
		} else if testResult.Verdict != verdict.OK {
			i.submission.Verdict = verdict.PT
			hasVerdict = true
		}
	}
	if !hasVerdict {
		if hasSkipped {
			logger.Panic("submission does not have verdict, but has skipped tests")
		}
		i.submission.Verdict = verdict.OK
	}
	return
}

// testJobCompleted must be done with acquired mutex
func (i *IOIGenerator) testJobCompleted(
	job *invokerconn.Job,
	result *masterconn.InvokerJobResult,
) {
	if job.Type != invokerconn.TestJob {
		logger.Warn("job type %s is %v; treating is as a testing job", job.ID, job.Type)
	}
	testGroupName := i.testNumberToGroupName[job.Test-1]
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
	i.internalTestResults[job.Test-1].result = buildTestResult(job, result)
	i.internalTestResults[job.Test-1].state = testFinished
	i.stopGivingNewTestsIfNeeded(testInternalGroupInfo, testGroupInfo, result.Verdict)
}

func (i *IOIGenerator) JobCompleted(jobResult *masterconn.InvokerJobResult) (*models.Submission, error) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	job, ok := i.givenJobs[jobResult.Job.ID]
	if !ok {
		return nil, fmt.Errorf("job %s does not exist", jobResult.Job.ID)
	}
	delete(i.givenJobs, jobResult.Job.ID)
	switch job.Type {
	case invokerconn.CompileJob:
		i.compileJobCompleted(job, jobResult)
	case invokerconn.TestJob:
		i.testJobCompleted(job, jobResult)
	default:
		return nil, fmt.Errorf("unknown job type for IOI problem: %v", job.Type)
	}
	return i.updateSubmissionResult()
}

func NewIOIGenerator(
	problem *models.Problem,
	submission *models.Submission,
	status *queuestatus.QueueStatus,
) (Generator, error) {
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
		groupNameToGroupInfo:    make(map[string]*models.TestGroup),
		groupNameToInternalInfo: make(map[string]*internalGroupInfo),
		testNumberToGroupName:   make(map[uint64]string),
		internalTestResults:     make([]*internalTestResult, 0),
		firstNotCompletedTest:   1,
		firstNotCompletedGroup:  1,
		firstNotGivenTest:       1,
		statusUpdater:           status,
	}
	generator.submission.Verdict = verdict.RU
	if err = generator.prepareGenerator(); err != nil {
		return nil, err
	}
	return generator, nil
}
