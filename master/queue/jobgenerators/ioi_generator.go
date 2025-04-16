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
)

type IOIGenerator struct {
	id         string
	mutex      sync.Mutex
	submission *models.Submission
	problem    *models.Problem
	state      generatorState
	givenJobs  map[string]*invokerconn.Job

	// groupNameToOrigGroup should be set once at the creating of the generator and should not be changed further
	groupNameToOrigGroup map[string]*models.TestGroup
	// origGroupNamesToBeGiven should be set once at the creating of the generator and should not be changed further
	origGroupNamesToBeGiven []string

	groupNameToGroup       map[string]*models.TestGroup
	groupNameToGroupResult map[string]*models.GroupResult
	groupNamesToBeGiven    []string
}

type problemWalkthroughResults struct {
	order    []string
	hasCycle bool
	used     map[string]int
}

func (i *IOIGenerator) walkThroughGroups(curGroup *models.TestGroup, result *problemWalkthroughResults) {
	result.used[curGroup.Name] = 1
	for _, requiredGroupName := range curGroup.RequiredGroupNames {
		requiredGroup := i.groupNameToGroup[requiredGroupName]
		if val, ok := result.used[requiredGroup.Name]; !ok {
			i.walkThroughGroups(requiredGroup, result)
		} else if val == 1 {
			result.hasCycle = true
		}
	}
	result.order = append(result.order, curGroup.Name)
	result.used[curGroup.Name] = 2
}

func (i *IOIGenerator) getWalkthroughResults() *problemWalkthroughResults {
	result := &problemWalkthroughResults{
		order:    make([]string, 0),
		hasCycle: false,
		used:     make(map[string]int),
	}
	for _, group := range i.groupNameToGroup {
		if _, ok := result.used[group.Name]; !ok {
			i.walkThroughGroups(group, result)
		}
	}
	return result
}

func (i *IOIGenerator) prepareGenerator() error {
	problem := i.problem
	if problem.ProblemType != models.ProblemTypeIOI {
		return fmt.Errorf("problem %v is not an IOI problem", problem.ID)
	}
	// must copy groups in order to not break the problem
	for _, group := range problem.TestGroups {
		if _, ok := i.groupNameToGroup[group.Name]; ok {
			return fmt.Errorf("group %v presented twice in problem", problem.ID)
		}
		group1 := group
		i.groupNameToGroup[group1.Name] = &group1
		group2 := group1
		i.groupNameToOrigGroup[group2.Name] = &group2
	}
	// each group with TestGroupScoringTypeEachTest must have TestScore
	for _, group := range problem.TestGroups {
		switch group.ScoringType {
		case models.TestGroupScoringTypeComplete:
			if group.Score == nil {
				return fmt.Errorf("group %v in problem %v doesn't have Score", group.Name, problem.ID)
			}
		case models.TestGroupScoringTypeEachTest:
			if group.TestScore == nil {
				return fmt.Errorf("group %v in problem %v doesn't have TestScore", group.Name, problem.ID)
			}
		case models.TestGroupScoringTypeMin:
		default:
			return fmt.Errorf("unknown TestGroupScoringType %v", group.ScoringType)
		}
	}
	{
		// simple check that each test is in exactly one group
		for testNumber := range i.problem.TestsNumber {
			cnt := 0
			for _, group := range i.problem.TestGroups {
				if uint64(group.FirstTest) <= testNumber+1 && testNumber+1 <= uint64(group.LastTest) {
					cnt += 1
				}
			}
			if cnt != 1 {
				return fmt.Errorf("test %v is presented in %v groups in problem %v", testNumber, problem.ID, cnt)
			}
		}
	}
	{
		// check for cycles and build topsort of the groups
		result := i.getWalkthroughResults()
		if result.hasCycle {
			return fmt.Errorf("cycle detected in problem %v", problem.ID)
		}
		i.groupNamesToBeGiven = result.order
		i.origGroupNamesToBeGiven = result.order
	}
	{
		testResults := make([]models.TestResult, 0, problem.TestsNumber)
		for testNumber := range problem.TestsNumber {
			testResults = append(testResults, models.TestResult{
				TestNumber: testNumber + 1,
				Verdict:    verdict.SK,
				Time:       0,
				Memory:     0,
			})
		}
		i.submission.TestResults = testResults
	}
	return nil
}

func (i *IOIGenerator) finalizeResults() {
	totalScore := 0.0
	i.submission.Verdict = verdict.OK
	for _, groupName := range i.origGroupNamesToBeGiven {
		group := i.groupNameToOrigGroup[groupName]
		if _, ok := i.groupNameToGroupResult[group.Name]; !ok {
			i.groupNameToGroupResult[group.Name] = &models.GroupResult{
				GroupName: group.Name,
				Passed:    false,
			}
		}
		groupResult := i.groupNameToGroupResult[group.Name]

		allRequiredPassed := true
		for _, requiredGroupName := range group.RequiredGroupNames {
			if !i.groupNameToGroupResult[requiredGroupName].Passed {
				allRequiredPassed = false
				break
			}
		}
		if !allRequiredPassed {
			groupResult.Passed = false
			groupResult.Points = 0.
			for j := group.FirstTest - 1; j < group.LastTest; j++ {
				i.submission.TestResults[j].Verdict = verdict.SK
				i.submission.TestResults[j].Points = pointer.Float64(0.0)
				i.submission.TestResults[j].Error = ""
			}
			i.groupNameToGroupResult[groupName] = groupResult
			continue
		}

		{
			groupTestResults := i.submission.TestResults[group.FirstTest-1 : group.LastTest]
			allOK := slices.IndexFunc(groupTestResults, func(result models.TestResult) bool {
				return result.Verdict != verdict.OK
			}) == -1
			groupResult.Passed = allOK
			if !allOK {
				i.submission.Verdict = verdict.PT
			}
		}
		if groupResult.Passed && group.ScoringType == models.TestGroupScoringTypeComplete {
			totalScore += *group.Score
			groupResult.Points = *group.Score
		}

		switch group.ScoringType {
		case models.TestGroupScoringTypeComplete:
			setSkipped := false
			for j := group.FirstTest - 1; j < group.LastTest; j++ {
				if setSkipped {
					i.submission.TestResults[j].Verdict = verdict.SK
					continue
				}
				if i.submission.TestResults[j].Verdict == verdict.OK ||
					i.submission.TestResults[j].Verdict == verdict.SK {
					continue
				}
				setSkipped = true
			}
		case models.TestGroupScoringTypeEachTest:
			fallthrough
		case models.TestGroupScoringTypeMin:
			for j := group.FirstTest - 1; j < group.LastTest; j++ {
				groupResult.Points += *i.submission.TestResults[j].Points
				totalScore += *i.submission.TestResults[j].Points
			}
		}
		i.groupNameToGroupResult[groupName] = groupResult
	}
	i.submission.Score = totalScore
	for _, group := range i.problem.TestGroups {
		i.submission.GroupResults = append(i.submission.GroupResults, *i.groupNameToGroupResult[group.Name])
	}
}

func (i *IOIGenerator) testNumberToGroup(testNumber uint64) *models.TestGroup {
	for _, group := range i.groupNameToOrigGroup {
		if group.FirstTest <= int(testNumber) && int(testNumber) <= group.LastTest {
			return group
		}
	}
	return nil
}

func (i *IOIGenerator) ID() string {
	return i.id
}

func (i *IOIGenerator) NextJob() *invokerconn.Job {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	if i.state == compilationFinished && len(i.groupNamesToBeGiven) == 0 {
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
	for len(i.groupNamesToBeGiven) > 0 {
		curGroup := i.groupNamesToBeGiven[0]
		shouldSkip := false
		group := i.groupNameToGroup[curGroup]
		for _, requiredGroupName := range group.RequiredGroupNames {
			if result, ok := i.groupNameToGroupResult[requiredGroupName]; ok && !result.Passed {
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			i.groupNamesToBeGiven = i.groupNamesToBeGiven[1:]
			continue
		}
		job.Test = uint64(group.FirstTest)
		if group.FirstTest == group.LastTest {
			i.groupNamesToBeGiven = i.groupNamesToBeGiven[1:]
		} else {
			group.FirstTest++
		}
		i.givenJobs[job.ID] = job
		return job
	}
	return nil
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
		i.finalizeResults()
		i.submission.Verdict = result.Verdict
		return i.submission, nil
	default:
		return nil, fmt.Errorf("unknown verdict for compilation completed: %v", result.Verdict)
	}
}

// testJobCompleted must be done with acquired mutex
func (i *IOIGenerator) testJobCompleted(job *invokerconn.Job, result *masterconn.InvokerJobResult) (*models.Submission, error) {
	if job.Type != invokerconn.TestJob {
		return nil, fmt.Errorf("job type %v is not test job", job.ID)
	}
	group := i.testNumberToGroup(job.Test)
	if group == nil {
		logger.Panic("Can't get group of test %v in job %v, problemId=%v", job.Test, job.ID, i.problem.ID)
		return nil, nil // just for goland to chill
	}

	if group.ScoringType == models.TestGroupScoringTypeEachTest &&
		(result.Points == nil || *result.Points > *group.TestScore) {
		if len(result.Error) > 0 {
			result.Error += "; "
		}
		result.Error += fmt.Sprintf("checker returned score=%v, but testScore=%v in problemId=%v, group=%v",
			result.Points, *group.TestScore, job.ID, i.problem.ID)
		result.Verdict = verdict.CF
	}
	if group.ScoringType == models.TestGroupScoringTypeMin && result.Points == nil {
		if len(result.Error) > 0 {
			result.Error += "; "
		}
		result.Error += fmt.Sprintf("checker returned score=%v, but testScore=%v in problemId=%v, group=%v",
			result.Points, *group.TestScore, job.ID, i.problem.ID)
		result.Verdict = verdict.CF
	}

	i.submission.TestResults[job.Test-1].Verdict = result.Verdict

	switch result.Verdict {
	case verdict.OK:
	case verdict.CF:
		i.submission.TestResults[job.Test-1].Error = result.Error
		fallthrough
	case verdict.PT, verdict.WA, verdict.RT, verdict.ML, verdict.TL, verdict.WL, verdict.SE:
		if _, ok := i.groupNameToGroupResult[group.Name]; !ok {
			i.groupNameToGroupResult[group.Name] = &models.GroupResult{
				GroupName: group.Name,
				Passed:    false,
			}
		}
	default:
		return nil, fmt.Errorf("unknown verdict for test %v in job %v", job.Test, job.ID)
	}

	if result.Verdict == verdict.OK || result.Verdict == verdict.PT {
		switch group.ScoringType {
		case models.TestGroupScoringTypeEachTest, models.TestGroupScoringTypeMin:
			i.submission.TestResults[job.Test-1].Points = group.Score
		case models.TestGroupScoringTypeComplete:
		}
	}

	/*
		  scenario:
			group1: complete, tests: 1-1
			group2: complete, tests: 2-2
		  	1. NextJob returns the test from the first group
			2. WA it
			3. It is expected that we already know testing result
		  so we have to go through all the groups and check for their required groups if they are not passed already
	*/
	{
		newGroupNamesToBeGiven := make([]string, 0)
		for _, groupName := range i.groupNamesToBeGiven {
			group = i.groupNameToOrigGroup[groupName]
			shouldTest := true
			for _, requiredGroupName := range group.RequiredGroupNames {
				if _, ok := i.groupNameToGroupResult[requiredGroupName]; ok {
					// if the group is presented in this map, then it is failed already
					shouldTest = false
					break
				}
			}
			if _, ok := i.groupNameToGroupResult[groupName]; ok &&
				group.ScoringType == models.TestGroupScoringTypeComplete {
				shouldTest = false
			}
			if shouldTest {
				newGroupNamesToBeGiven = append(newGroupNamesToBeGiven, groupName)
			}
		}
		i.groupNamesToBeGiven = newGroupNamesToBeGiven
	}

	if len(i.givenJobs) == 0 && len(i.groupNamesToBeGiven) == 0 {
		i.finalizeResults()
		return i.submission, nil
	}
	return nil, nil
}

func (i *IOIGenerator) JobCompleted(jobResult *masterconn.InvokerJobResult) (*models.Submission, error) {
	i.mutex.Lock()
	defer i.mutex.Unlock()
	job, ok := i.givenJobs[jobResult.JobID]
	if !ok {
		return nil, fmt.Errorf("job %s not exist", jobResult.JobID)
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
		id:                     id.String(),
		submission:             submission,
		problem:                problem,
		state:                  compilationNotStarted,
		givenJobs:              make(map[string]*invokerconn.Job),
		groupNameToGroupResult: make(map[string]*models.GroupResult),
		// these fields will be filled in prepareGenerator
		groupNameToOrigGroup: make(map[string]*models.TestGroup),
		groupNameToGroup:     make(map[string]*models.TestGroup),
		groupNamesToBeGiven:  nil,
	}
	if err = generator.prepareGenerator(); err != nil {
		return nil, err
	}
	return generator, nil
}
