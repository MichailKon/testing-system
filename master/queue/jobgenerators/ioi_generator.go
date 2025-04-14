package jobgenerators

import (
	"fmt"
	"github.com/google/uuid"
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
	state      state
	givenJobs  map[string]*invokerconn.Job

	groupNameToGroup        map[string]*models.TestGroup
	groupNamesToBeGiven     []string
	groupNameToDependencies map[string][]*models.TestGroup
}

func (i *IOIGenerator) finalizeResults() {
	for _, group := range i.groupNameToGroup {
		switch group.ScoringType {
		case models.TestGroupScoringTypeComplete:
			setSkipped := false
			for j := range i.submission.TestResults[group.FirstTest-1 : group.LastTest] {
				if setSkipped {
					i.submission.TestResults[j].Verdict = verdict.SK
					continue
				}
				if i.submission.TestResults[j].Verdict == verdict.OK || i.submission.TestResults[j].Verdict == verdict.SK {
					continue
				}
				setSkipped = true
				i.submission.Verdict = i.submission.TestResults[j].Verdict
			}
			if !setSkipped {
				if group.Score == nil {
					logger.Panic("Group '%v' in problemId=%v has TypeComplete, but score is nil",
						group.Name, i.problem.ID)
				}
				i.submission.TestResults[group.LastTest-1].Points = group.Score
			}
		case models.TestGroupScoringTypeEachTest:
			// TODO
		case models.TestGroupScoringTypeMin:
			// TODO
		}
	}
	notOkInd := slices.IndexFunc(i.submission.TestResults, func(result models.TestResult) bool {
		return result.Verdict != verdict.OK
	})
	if notOkInd == -1 {
		i.submission.Verdict = verdict.OK
	} else {
		i.submission.Verdict = i.submission.TestResults[notOkInd].Verdict
	}
	totalScore := 0.
	for _, result := range i.submission.TestResults {
		if result.Points != nil {
			totalScore += *result.Points
		}
	}
	i.submission.Score = totalScore
}

func (i *IOIGenerator) fetchGroupDependencies(curGroup *models.TestGroup) map[string]struct{} {
	used := make(map[string]struct{})
	q := make([]*models.TestGroup, 0)
	q = append(q, curGroup)
	used[curGroup.Name] = struct{}{}
	for len(q) > 0 {
		group := q[0]
		q = q[1:]
		for _, nextGroup := range i.groupNameToDependencies[group.Name] {
			if _, ok := used[nextGroup.Name]; !ok {
				q = append(q, nextGroup)
				used[nextGroup.Name] = struct{}{}
			}
		}
	}
	return used
}

func (i *IOIGenerator) testNumberToGroup(testNumber uint64) *models.TestGroup {
	for _, group := range i.groupNameToGroup {
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
		return job
	}
	curGroupName := i.groupNamesToBeGiven[0]
	group := i.groupNameToGroup[curGroupName]
	job.Type = invokerconn.TestJob
	job.Test = uint64(group.FirstTest)
	if group.FirstTest == group.LastTest {
		i.groupNamesToBeGiven = i.groupNamesToBeGiven[1:]
	} else {
		group.FirstTest++
	}
	i.givenJobs[job.ID] = job
	return job
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
		i.submission.Verdict = result.Verdict
		return i.submission, nil
	default:
		return nil, fmt.Errorf("unknown verdict for compilation completed: %v", result.Verdict)
	}
}

// testJobCompleted must be done with acquired mutex
func (i *IOIGenerator) testJobCompleted(job *invokerconn.Job, result *masterconn.InvokerJobResult) (*models.Submission, error) {
	if job.Type != invokerconn.TestJob {
		return nil, fmt.Errorf("job type %s is not test job", job.ID)
	}
	group := i.testNumberToGroup(job.Test)
	if group == nil {
		logger.Panic("Can't get group of test %v in job %v, problemId=%v", job.Test, job.ID, i.problem.ID)
		return nil, nil // just for goland to chill
	}

	i.submission.TestResults[job.Test-1].Verdict = result.Verdict

	switch result.Verdict {
	case verdict.OK:
		if group.ScoringType == models.TestGroupScoringTypeEachTest {
			if group.TestScore == nil {
				logger.Panic("Group '%v' has type EachTest, but TestScore is nil in problemId=%v",
					group.Name, i.problem.ID)
			}
			i.submission.TestResults[job.Test-1].Points = group.TestScore
		} else if group.ScoringType == models.TestGroupScoringTypeMin {
			if result.Points == nil {
				logger.Panic("Group '%v' has type Min, but checker didn't set points in problemId=%v, jobId=%v",
					group.Name, i.problem.ID, job.ID)
			}
			i.submission.TestResults[job.Test-1].Points = result.Points
		}
		if len(i.givenJobs) == 0 && len(i.groupNamesToBeGiven) == 0 {
			i.finalizeResults()
			return i.submission, nil
		}
		return nil, nil
	case verdict.PT, verdict.WA, verdict.RT, verdict.ML, verdict.TL, verdict.WL, verdict.SE, verdict.CF:
		dependencies := i.fetchGroupDependencies(group)
		newGroupNamesToBeGiven := make([]string, 0)
		for _, s := range i.groupNamesToBeGiven {
			if _, ok := dependencies[s]; !ok {
				newGroupNamesToBeGiven = append(newGroupNamesToBeGiven, s)
			} else if s == group.Name && group.ScoringType != models.TestGroupScoringTypeComplete {
				newGroupNamesToBeGiven = append(newGroupNamesToBeGiven, s)
			}
		}
		i.groupNamesToBeGiven = newGroupNamesToBeGiven
		if len(i.givenJobs) == 0 && len(i.groupNamesToBeGiven) == 0 {
			i.finalizeResults()
			return i.submission, nil
		}
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown verdict for testing completed: %v", result.Verdict)
	}
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
	return &IOIGenerator{}, nil
}
