package queue

import (
	"container/list"
	"fmt"
	"github.com/google/uuid"
	"slices"
	"sync"
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/lib/logger"
)

type queueElement struct {
	invokerJob       *invokerconn.Job
	submissionHolder *SubmissionHolder
	// ids that should be done for the corresponding move of the current task
	requiredIdsLPToHP      []string
	requiredIdsBlockedToLP []string
	// ids that wait this task to be done for the corresponding move
	waitersIdsLPToHP      []string
	waitersIdsBlockedToLP []string
}

type taskInfo struct {
	listElement       *list.Element
	correspondingList *list.List
	submissionHolder  *SubmissionHolder
}

type Queue struct {
	ts    *common.TestingSystem
	mutex sync.Mutex
	// firstly, the jobs will be taken from the highPriorityJobs (HP then)
	// if it's empty, then the tasks from the lowPriorityJobs will be taken (LP then)
	// initially, only the Compile task is in the HP; the other ones are in the blockedJobs
	// when a JobCompleted method is called, then some tasks may move either blockedJobs -> LP, or LP -> HP, or both
	highPriorityJobs list.List
	lowPriorityJobs  list.List
	blockedJobs      list.List
	// helper maps for easier moves between lists
	stringUUIDToInfo         map[string]*taskInfo
	submissionHolderToJobIDs map[*SubmissionHolder][]string
	//
	givenTasks Set[string]
}

func (q *Queue) processICPCTask(submission *SubmissionHolder) error {
	// TODO!!!
	const testCount = 10
	// prepare tasks
	var compileTask *queueElement

	if UUID, err := uuid.NewV7(); err == nil {
		compileTask = &queueElement{
			invokerJob: &invokerconn.Job{
				ID:       UUID.String(),
				SubmitID: submission.Submission.ID,
				Type:     invokerconn.CompileJob,
				Test:     0,
			},
			submissionHolder:       submission,
			requiredIdsLPToHP:      make([]string, 0),
			requiredIdsBlockedToLP: make([]string, 0),
			waitersIdsLPToHP:       make([]string, 0),
			waitersIdsBlockedToLP:  make([]string, 0),
		}
	} else {
		return err
	}

	testingTasks := make([]*queueElement, 0, testCount)
	for i := 1; i <= testCount; i++ {
		if UUID, err := uuid.NewV7(); err == nil {
			testingTasks = append(testingTasks, &queueElement{
				invokerJob: &invokerconn.Job{
					ID:       UUID.String(),
					SubmitID: submission.Submission.ID,
					Type:     invokerconn.TestJob,
					Test:     uint64(i),
				},
				submissionHolder:       submission,
				requiredIdsLPToHP:      make([]string, 0),
				requiredIdsBlockedToLP: make([]string, 0),
				waitersIdsLPToHP:       make([]string, 0),
				waitersIdsBlockedToLP:  make([]string, 0),
			})
		} else {
			return err
		}
	}
	// fill up required and waiting IDs
	// construct a graph with O(n^2) edges for now (bc for any test number i tests with numbers 1..i should be finished)
	for i, task := range testingTasks {
		task.requiredIdsBlockedToLP = append(task.requiredIdsBlockedToLP, compileTask.invokerJob.ID)
		compileTask.waitersIdsBlockedToLP = append(compileTask.waitersIdsBlockedToLP, task.invokerJob.ID)
		for j := 0; j < i; j++ {
			prevTask := testingTasks[j]
			prevTask.waitersIdsLPToHP = append(prevTask.waitersIdsLPToHP, task.invokerJob.ID)
			task.requiredIdsLPToHP = append(task.requiredIdsLPToHP, prevTask.invokerJob.ID)
		}
	}
	// add tasks
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.highPriorityJobs.PushBack(compileTask)
	q.stringUUIDToInfo[compileTask.invokerJob.ID] = &taskInfo{
		listElement:       q.highPriorityJobs.Back(),
		correspondingList: &q.highPriorityJobs,
		submissionHolder:  submission,
	}
	q.submissionHolderToJobIDs[submission] = append(q.submissionHolderToJobIDs[submission], compileTask.invokerJob.ID)
	for _, task := range testingTasks {
		q.blockedJobs.PushBack(task)
		q.stringUUIDToInfo[task.invokerJob.ID] = &taskInfo{
			listElement:       q.blockedJobs.Back(),
			correspondingList: &q.blockedJobs,
			submissionHolder:  submission,
		}
		q.submissionHolderToJobIDs[submission] = append(q.submissionHolderToJobIDs[submission], task.invokerJob.ID)
	}
	return nil
}

func (q *Queue) Submit(submission *SubmissionHolder) error {
	switch submission.Problem.ProblemType {
	case models.ProblemType_ICPC:
		return q.processICPCTask(submission)
	default:
		return logger.Error("unknown problem type")
	}
}

// cleanupAfterFailedTask should be performed with already locked mutex
func (q *Queue) cleanupAfterFailedTask(taskId string) error {
	submissionHolder := q.stringUUIDToInfo[taskId].submissionHolder
	for _, nextTaskID := range q.submissionHolderToJobIDs[submissionHolder] {
		q.givenTasks.SafeRemove(nextTaskID)
		if nextTaskInfo, ok := q.stringUUIDToInfo[nextTaskID]; !ok {
			// take HP(1) and LP(2); get OK on LP(2) -> it is removed; get WA on HP(1)
			continue
		} else {
			nextTaskInfo.correspondingList.Remove(nextTaskInfo.listElement)
			delete(q.stringUUIDToInfo, nextTaskID)
		}
	}
	delete(q.submissionHolderToJobIDs, submissionHolder)
	return nil
}

// moveTasksAfterFinishing should be performed with already locked mutex
func (q *Queue) moveTasksAfterFinishing(currentTaskInfo *taskInfo) error {
	moveTaskToNextStage := func(nextTaskInfo *taskInfo, newList *list.List) {
		nextTaskInfo.correspondingList.Remove(nextTaskInfo.listElement)
		newList.PushBack(nextTaskInfo.listElement.Value)
		nextTaskInfo.listElement = newList.Back()
		nextTaskInfo.correspondingList = newList
	}
	removeDependency := func(from []string, nextTaskInfo *taskInfo, ind int, to *list.List) []string {
		from[ind] = from[len(from)-1]
		from = from[:len(from)-1]
		if len(from) == 0 {
			moveTaskToNextStage(nextTaskInfo, to)
		}
		return from
	}

	qe := currentTaskInfo.listElement.Value.(*queueElement)
	// firstly, blocked -> LP
	for _, nextTaskId := range qe.waitersIdsBlockedToLP {
		nextTaskInfo := q.stringUUIDToInfo[nextTaskId]
		nextTaskQe := nextTaskInfo.listElement.Value.(*queueElement)
		// TODO: make this faster mb
		ind := slices.Index(nextTaskQe.requiredIdsBlockedToLP, qe.invokerJob.ID)
		if ind == -1 {
			return fmt.Errorf("taskId %v is a dep for %v, but not vice versa",
				qe.invokerJob.ID, nextTaskQe.invokerJob.ID)
		}
		if q.givenTasks.Contains(nextTaskId) {
			return fmt.Errorf("blocked task %v already given", nextTaskId)
		}
		nextTaskQe.requiredIdsBlockedToLP =
			removeDependency(nextTaskQe.requiredIdsBlockedToLP, nextTaskInfo, ind, &q.lowPriorityJobs)
	}
	qe.waitersIdsBlockedToLP = make([]string, 0)
	// then LP -> HP
	for _, nextTaskID := range qe.waitersIdsLPToHP {
		nextTaskInfo, ok := q.stringUUIDToInfo[nextTaskID]
		if !ok {
			// can continue, because nextJob could be finished before this one
			continue
		}
		nextTaskQe := nextTaskInfo.listElement.Value.(*queueElement)
		if q.givenTasks.Contains(nextTaskID) {
			// the same as upper one
			continue
		}
		// TODO: make this faster mb
		ind := slices.Index(nextTaskQe.requiredIdsLPToHP, qe.invokerJob.ID)
		if ind == -1 {
			// the same as upper one
			continue
		}
		nextTaskQe.requiredIdsLPToHP =
			removeDependency(nextTaskQe.requiredIdsLPToHP, nextTaskInfo, ind, &q.highPriorityJobs)
	}
	qe.waitersIdsLPToHP = make([]string, 0)
	return nil
}

func (q *Queue) jobCompletedSuccessfully(info *taskInfo) (submission *SubmissionHolder, err error) {
	err = q.moveTasksAfterFinishing(info)
	if haveTasks, ok := q.submissionHolderToJobIDs[info.submissionHolder]; ok && len(haveTasks) == 0 {
		delete(q.submissionHolderToJobIDs, info.submissionHolder)
		return info.submissionHolder, err
	} else {
		return nil, err
	}
}

// compileJobCompleted should be performed with already locked mutex
func (q *Queue) compileJobCompleted(job *masterconn.InvokerJobResult) (submission *SubmissionHolder, err error) {
	info, ok := q.stringUUIDToInfo[job.JobID]
	if !ok {
		return nil, fmt.Errorf("jobID %s not exist", job.JobID)
	}

	switch job.Verdict {
	case verdict.CD:
		return q.jobCompletedSuccessfully(info)
	case verdict.CE:
		return info.listElement.Value.(*queueElement).submissionHolder,
			q.cleanupAfterFailedTask(info.listElement.Value.(*queueElement).invokerJob.ID)
	default:
		return nil, fmt.Errorf("unknown verdict for compilation completed: %v", job.Verdict)
	}
}

// testJobCompleted should be performed with already locked mutex
func (q *Queue) testJobCompleted(job *masterconn.InvokerJobResult) (submission *SubmissionHolder, err error) {
	info, ok := q.stringUUIDToInfo[job.JobID]
	if !ok {
		return nil, fmt.Errorf("jobID %s not exist", job.JobID)
	}

	switch job.Verdict {
	case verdict.OK:
		return q.jobCompletedSuccessfully(info)
	case verdict.PT, verdict.WA, verdict.PE, verdict.RT, verdict.ML, verdict.TL, verdict.WL, verdict.SE:
		return info.listElement.Value.(*queueElement).submissionHolder, q.cleanupAfterFailedTask(job.JobID)
	case verdict.CF:
		// TODO think about this one
		panic("implement me")
	default:
		return nil, fmt.Errorf("unknown verdict for testing completed: %v", job.Verdict)
	}
}

func (q *Queue) JobCompleted(job *masterconn.InvokerJobResult) (submission *SubmissionHolder, err error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.givenTasks.SafeRemove(job.JobID)
	info, ok := q.stringUUIDToInfo[job.JobID]
	if !ok {
		return nil, nil
	}

	defer func() {
		delete(q.stringUUIDToInfo, job.JobID)
	}()
	{
		ind := slices.Index(q.submissionHolderToJobIDs[info.submissionHolder], job.JobID)
		if ind == -1 {
			return nil, nil
		}
		ids := q.submissionHolderToJobIDs[info.submissionHolder]
		ids[ind] = ids[len(ids)-1]
		ids = ids[:len(ids)-1]
		q.submissionHolderToJobIDs[info.submissionHolder] = ids
	}
	task := info.listElement.Value.(*queueElement)
	switch task.invokerJob.Type {
	case invokerconn.CompileJob:
		return q.compileJobCompleted(job)
	case invokerconn.TestJob:
		return q.testJobCompleted(job)
	default:
		return nil, fmt.Errorf("unknown job type: %v", task.invokerJob.Type)
	}
}

func (q *Queue) RescheduleJob(jobID string) error {
	panic("implement me")
}

func (q *Queue) NextJob() *invokerconn.Job {
	giveTask := func(from *list.List) *invokerconn.Job {
		task := from.Front()
		from.Remove(task)
		ID := task.Value.(*queueElement).invokerJob.ID
		q.givenTasks.Add(ID)
		return task.Value.(*queueElement).invokerJob
	}

	q.mutex.Lock()
	defer q.mutex.Unlock()
	if q.highPriorityJobs.Len() > 0 {
		return giveTask(&q.highPriorityJobs)
	} else if q.lowPriorityJobs.Len() > 0 {
		return giveTask(&q.lowPriorityJobs)
	}
	return nil
}
