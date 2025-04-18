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

	groupNameToGroupInfo  map[string]*internalGroupInfo
	groupNamesToBeGiven   []string
	testNumberToGroupName map[uint64]string
}

type groupState int

const (
	onlyOK groupState = iota
	okOrPT
	hasFails
)

type internalGroupInfo struct {
	group       models.TestGroup
	minPoints   float64
	sumPoints   float64
	testsPasses int
	groupState  groupState
}

type problemWalkthroughResults struct {
	order []string
	used  map[string]int
	err   error
}

func (i *IOIGenerator) walkThroughGroups(groupInfo *internalGroupInfo, result *problemWalkthroughResults) {
	result.used[groupInfo.group.Name] = 1
	for _, requiredGroupName := range groupInfo.group.RequiredGroupNames {
		requiredGroupInfo := i.groupNameToGroupInfo[requiredGroupName]
		if requiredGroupInfo.group.ScoringType == models.TestGroupScoringTypeEachTest ||
			requiredGroupInfo.group.ScoringType == models.TestGroupScoringTypeMin {
			result.err = fmt.Errorf("group can't depend on group with scoring type each test or min")
		} else if val, ok := result.used[requiredGroupInfo.group.Name]; !ok {
			i.walkThroughGroups(requiredGroupInfo, result)
		} else if val == 1 {
			result.err = fmt.Errorf("cycle detected in dependencies")
		}
	}
	result.order = append(result.order, groupInfo.group.Name)
	result.used[groupInfo.group.Name] = 2
}

func (i *IOIGenerator) getWalkthroughResults() *problemWalkthroughResults {
	result := &problemWalkthroughResults{
		order: make([]string, 0),
		used:  make(map[string]int),
		err:   nil,
	}
	for _, groupInfo := range i.groupNameToGroupInfo {
		if _, ok := result.used[groupInfo.group.Name]; !ok {
			i.walkThroughGroups(groupInfo, result)
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
		if _, ok := i.groupNameToGroupInfo[group.Name]; ok {
			return fmt.Errorf("group %v presented twice in problem", problem.ID)
		}
		i.groupNameToGroupInfo[group.Name] = &internalGroupInfo{
			group:       group,
			minPoints:   math.Inf(1),
			sumPoints:   0,
			testsPasses: 0,
			groupState:  onlyOK,
		}
		group1 := group
		i.groupNameToOrigGroup[group1.Name] = &group1
		for testNumber := group.FirstTest - 1; testNumber < group.LastTest; testNumber++ {
			i.testNumberToGroupName[uint64(testNumber)] = group.Name
		}
	}
	// each group with TestGroupScoringTypeEachTest must have TestScore
	for _, group := range problem.TestGroups {
		switch group.ScoringType {
		case models.TestGroupScoringTypeComplete, models.TestGroupScoringTypeMin:
			if group.Score == nil {
				return fmt.Errorf("group %v in problem %v doesn't have Score", group.Name, problem.ID)
			}
		case models.TestGroupScoringTypeEachTest:
			if group.TestScore == nil {
				return fmt.Errorf("group %v in problem %v doesn't have TestScore", group.Name, problem.ID)
			}
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
		// check for cycles and build order of groups
		result := i.getWalkthroughResults()
		if result.err != nil {
			return result.err
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

// finalizeResultsAfterTestJobCompleted must be done with acquired mutex and only ONCE (in case of no CE)
func (i *IOIGenerator) finalizeResultsAfterTestJobCompleted() {
	totalScore := 0.0
	i.submission.Verdict = verdict.OK
	for _, groupName := range i.origGroupNamesToBeGiven {
		groupInfo := i.groupNameToGroupInfo[groupName]
		origGroup := i.groupNameToOrigGroup[groupName]
		// if at least one required group is failed, then current group is fully SK
		haveFail := false
		for _, requiredGroupName := range origGroup.RequiredGroupNames {
			if i.groupNameToGroupInfo[requiredGroupName].groupState == hasFails {
				haveFail = true
			}
		}
		if haveFail {
			for testNumber := origGroup.FirstTest - 1; testNumber < origGroup.LastTest; testNumber++ {
				i.submission.TestResults[testNumber].Verdict = verdict.SK
				i.submission.TestResults[testNumber].Points = pointer.Float64(0)
			}
			i.submission.GroupResults = append(i.submission.GroupResults, models.GroupResult{
				GroupName: groupName,
				Points:    0.0,
				Passed:    false,
			})
			continue
		}
		// otherwise calculate score for this group and set appropriate verdicts
		if groupInfo.groupState != onlyOK {
			i.submission.Verdict = verdict.PT
		}
		curPoints := 0.0
		setSKVerdicts := func(notAcceptableVerdicts ...verdict.Verdict) {
			// should set SK for all the tests after the first one without OK
			setSK := false
			for testNumber := origGroup.FirstTest - 1; testNumber < origGroup.LastTest; testNumber++ {
				if !slices.Contains(notAcceptableVerdicts, i.submission.TestResults[testNumber].Verdict) {
					setSK = true
					continue
				}
				if setSK {
					i.submission.TestResults[testNumber].Verdict = verdict.SK
				}
			}
		}
		switch origGroup.ScoringType {
		case models.TestGroupScoringTypeComplete:
			if groupInfo.groupState == onlyOK {
				curPoints = *origGroup.Score
			} else {
				setSKVerdicts(verdict.OK)
			}
		case models.TestGroupScoringTypeEachTest:
			curPoints = groupInfo.sumPoints
		case models.TestGroupScoringTypeMin:
			if groupInfo.groupState == hasFails {
				setSKVerdicts(verdict.OK, verdict.PT)
				curPoints = 0.0
			} else {
				if groupInfo.minPoints != math.Inf(1) {
					curPoints = groupInfo.minPoints
				}
			}
		}
		totalScore += curPoints
		i.submission.GroupResults = append(i.submission.GroupResults, models.GroupResult{
			GroupName: groupName,
			Points:    curPoints,
			Passed:    groupInfo.groupState == onlyOK,
		})
	}
	i.submission.Score = totalScore
}

// finalizeResultsAfterCompileJobFailed must be done with acquired mutex and only ONCE (in case of CE)
func (i *IOIGenerator) finalizeResultsAfterCompileJobFailed() {
	i.submission.Verdict = verdict.CE
	for _, groupName := range i.origGroupNamesToBeGiven {
		origGroup := i.groupNameToOrigGroup[groupName]
		for testNumber := origGroup.FirstTest - 1; testNumber < origGroup.LastTest; testNumber++ {
			i.submission.TestResults[testNumber].Verdict = verdict.SK
		}
		i.submission.GroupResults = append(i.submission.GroupResults, models.GroupResult{
			GroupName: groupName,
			Points:    0.0,
			Passed:    false,
		})
	}
	i.submission.Score = 0.0
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
	for len(i.groupNameToGroupInfo) > 0 {
		groupName := i.groupNamesToBeGiven[0]
		groupInfo := i.groupNameToGroupInfo[groupName]
		job.Test = uint64(groupInfo.group.FirstTest)
		groupInfo.group.FirstTest++
		if groupInfo.group.FirstTest > groupInfo.group.LastTest {
			i.groupNamesToBeGiven = i.groupNamesToBeGiven[1:]
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
		i.finalizeResultsAfterCompileJobFailed()
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
	groupName := i.testNumberToGroupName[job.Test-1]
	groupInfo := i.groupNameToGroupInfo[groupName]

	// a lot of checks for fucking IOI problems
	setResultCF := func(checkerScore *float64, problemId uint, group string) {
		if len(result.Error) > 0 {
			result.Error += "; "
		}
		result.Error += fmt.Sprintf(
			"checker returned score=%v, but testScore=%v and score=%v in problemId=%v, group=%v, type=%v",
			checkerScore, groupInfo.group.TestScore, groupInfo.group.Score,
			problemId, group, groupInfo.group.ScoringType)
		result.Verdict = verdict.CF
		result.Points = pointer.Float64(0)
	}
	if groupInfo.group.ScoringType == models.TestGroupScoringTypeEachTest {
		if result.Points == nil {
			if result.Verdict == verdict.OK {
				result.Points = groupInfo.group.TestScore
			} else if result.Verdict == verdict.PT {
				setResultCF(result.Points, i.problem.ID, groupName)
			}
		} else if *result.Points > *groupInfo.group.TestScore {
			setResultCF(result.Points, i.problem.ID, groupName)
		}
	} else if groupInfo.group.ScoringType == models.TestGroupScoringTypeMin {
		if result.Points == nil {
			if result.Verdict == verdict.OK {
				result.Points = groupInfo.group.Score
			} else if result.Verdict == verdict.PT {
				setResultCF(result.Points, i.problem.ID, groupName)
			}
		} else if *result.Points > *groupInfo.group.Score {
			setResultCF(result.Points, i.problem.ID, groupName)
		}
	} else if groupInfo.group.ScoringType == models.TestGroupScoringTypeComplete {
		if result.Verdict == verdict.PT {
			result.Verdict = verdict.CF
			if len(result.Error) > 0 {
				result.Error += "; "
			}
			result.Error += fmt.Sprintf("checker returned PT on test=%v in problemId=%v, group=%v, type=%v",
				job.Test, i.problem.ID, groupName, groupInfo.group.ScoringType)
		}
	}

	i.submission.TestResults[job.Test-1].Verdict = result.Verdict
	if result.Points != nil {
		groupInfo.minPoints = min(groupInfo.minPoints, *result.Points)
		groupInfo.sumPoints += *result.Points
	}
	if result.Statistics != nil {
		i.submission.TestResults[job.Test-1].Time = result.Statistics.Time
		i.submission.TestResults[job.Test-1].Memory = result.Statistics.Memory
	}

	switch result.Verdict {
	case verdict.PT:
		if groupInfo.groupState != hasFails {
			groupInfo.groupState = okOrPT
		}
		fallthrough
	case verdict.OK:
		if groupInfo.group.ScoringType == models.TestGroupScoringTypeEachTest ||
			groupInfo.group.ScoringType == models.TestGroupScoringTypeMin {
			i.submission.TestResults[job.Test-1].Points = result.Points
		}
	case verdict.CF:
		i.submission.TestResults[job.Test-1].Error = result.Error
		fallthrough
	case verdict.WA, verdict.RT, verdict.ML, verdict.TL, verdict.WL, verdict.SE:
		groupInfo.groupState = hasFails
	default:
		return nil, fmt.Errorf("unknown verdict for test %v in job %v", job.Test, job.ID)
	}

	/*
		  scenario:
			group1: any scoring type, tests: 1-1
			group2: any scoring type, tests: 2-2
		  	1. NextJob returns the test from the first group
			2. WA it
			3. It is expected that we already know testing result
		  so we have to go through all the groups and check for their required groups if they are not passed already
	*/
	{
		newGroupNamesToBeGiven := make([]string, 0)
		for _, groupName = range i.groupNamesToBeGiven {
			groupInfo = i.groupNameToGroupInfo[groupName]
			shouldTest := true
			for _, requiredGroupName := range groupInfo.group.RequiredGroupNames {
				if i.groupNameToGroupInfo[requiredGroupName].groupState != onlyOK {
					shouldTest = false
					break
				}
			}
			// this is the only case, since we assume that checker can not return PT in type complete
			if groupInfo.groupState == hasFails {
				shouldTest = false
			}
			if shouldTest {
				newGroupNamesToBeGiven = append(newGroupNamesToBeGiven, groupName)
			}
		}
		i.groupNamesToBeGiven = newGroupNamesToBeGiven
	}

	if len(i.givenJobs) == 0 && len(i.groupNamesToBeGiven) == 0 {
		i.finalizeResultsAfterTestJobCompleted()
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
		id:                   id.String(),
		submission:           submission,
		problem:              problem,
		state:                compilationNotStarted,
		givenJobs:            make(map[string]*invokerconn.Job),
		groupNameToGroupInfo: make(map[string]*internalGroupInfo),
		// these fields will be filled in prepareGenerator
		groupNameToOrigGroup:    make(map[string]*models.TestGroup),
		groupNamesToBeGiven:     nil,
		origGroupNamesToBeGiven: nil,
		testNumberToGroupName:   make(map[uint64]string),
	}
	if err = generator.prepareGenerator(); err != nil {
		return nil, err
	}
	return generator, nil
}
