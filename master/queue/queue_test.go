package queue

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"testing"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
)

const fixtureSubmissionID = 2

func fixtureSubmission() *SubmissionHolder {
	submission := &SubmissionHolder{
		Submission: &models.Submission{
			ProblemID: 1,
			Language:  "",
			Verdict:   "",
		},
		Problem: &models.Problem{
			Model:       gorm.Model{},
			ProblemType: models.ProblemType_ICPC,
		},
	}
	submission.Submission.ID = fixtureSubmissionID
	return submission
}

func checkQueueEmptiness(t *testing.T, q *Queue) {
	assert.Equal(t, 0, q.highPriorityJobs.Len())
	assert.Equal(t, 0, q.lowPriorityJobs.Len())
	assert.Equal(t, 0, q.blockedJobs.Len())

	assert.Equal(t, make(map[string]*taskInfo), q.stringUUIDToInfo)
	assert.Equal(t, make(map[*SubmissionHolder][]string), q.submissionHolderToJobIDs)

	assert.Equal(t, NewSet[string](), q.givenTasks)
}

func nextJob(t *testing.T, q IQueue, ID uint, jobType invokerconn.JobType) *invokerconn.Job {
	job := q.NextJob()
	assert.NotNil(t, job)
	assert.Equal(t, job.Type, jobType)
	assert.Equal(t, ID, job.SubmitID)
	return job
}

func TestStraightTasksFinishing(t *testing.T) {
	q := NewQueue(nil)
	submission := fixtureSubmission()
	err := q.Submit(submission)
	assert.Nil(t, err)
	job := nextJob(t, q, fixtureSubmissionID, invokerconn.CompileJob)
	handler, err := q.JobCompleted(&masterconn.InvokerJobResult{JobID: job.ID, Verdict: verdict.CD})
	assert.Nil(t, err)
	assert.Nil(t, handler)
	for i := range 9 {
		job = nextJob(t, q, fixtureSubmissionID, invokerconn.TestJob)
		assert.Equal(t, i+1, int(job.Test))
		handler, err = q.JobCompleted(&masterconn.InvokerJobResult{JobID: job.ID, Verdict: verdict.OK})
		assert.Nil(t, err)
		assert.Nil(t, handler)
	}
	job = nextJob(t, q, fixtureSubmissionID, invokerconn.TestJob)
	assert.Equal(t, 10, int(job.Test))
	handler, err = q.JobCompleted(&masterconn.InvokerJobResult{JobID: job.ID, Verdict: verdict.OK})
	assert.Nil(t, err)
	assert.NotNil(t, handler)
}

func TestTasksFinishing(t *testing.T) {
	prepare := func() (IQueue, *SubmissionHolder, []string) {
		q := NewQueue(nil)
		submission := fixtureSubmission()
		err := q.Submit(submission)
		assert.Nil(t, err)
		job := nextJob(t, q, fixtureSubmissionID, invokerconn.CompileJob)
		handler, err := q.JobCompleted(&masterconn.InvokerJobResult{JobID: job.ID, Verdict: verdict.CD})
		assert.Nil(t, err)
		assert.Nil(t, handler)
		firstTwoJobIds := make([]string, 0)
		for i := range 2 {
			job = nextJob(t, q, fixtureSubmissionID, invokerconn.TestJob)
			assert.Equal(t, i+1, int(job.Test))
			firstTwoJobIds = append(firstTwoJobIds, job.ID)
		}
		return q, submission, firstTwoJobIds
	}

	finishOtherTests := func(q IQueue, submission *SubmissionHolder) {
		for i := 2; i < 9; i++ {
			job := nextJob(t, q, fixtureSubmissionID, invokerconn.TestJob)
			assert.Equal(t, i+1, int(job.Test))
			handler, err := q.JobCompleted(&masterconn.InvokerJobResult{JobID: job.ID, Verdict: verdict.OK})
			assert.Nil(t, err)
			assert.Nil(t, handler)
		}
		job := nextJob(t, q, fixtureSubmissionID, invokerconn.TestJob)
		assert.Equal(t, 10, int(job.Test))
		handler, err := q.JobCompleted(&masterconn.InvokerJobResult{JobID: job.ID, Verdict: verdict.OK})
		assert.Nil(t, err)
		assert.NotNil(t, handler)
	}

	t.Run("right order", func(t *testing.T) {
		q, submission, firstTwoJobIds := prepare()
		for _, id := range firstTwoJobIds {
			handler, err := q.JobCompleted(&masterconn.InvokerJobResult{JobID: id, Verdict: verdict.OK})
			assert.Nil(t, err)
			assert.Nil(t, handler)
		}
		finishOtherTests(q, submission)
		checkQueueEmptiness(t, q.(*Queue))
	})

	t.Run("wrong order + both ok", func(t *testing.T) {
		q, submission, firstTwoJobIds := prepare()
		handler, err := q.JobCompleted(&masterconn.InvokerJobResult{JobID: firstTwoJobIds[1], Verdict: verdict.OK})
		assert.Nil(t, err)
		assert.Nil(t, handler)
		handler, err = q.JobCompleted(&masterconn.InvokerJobResult{JobID: firstTwoJobIds[0], Verdict: verdict.OK})
		assert.Nil(t, err)
		assert.Nil(t, handler)

		finishOtherTests(q, submission)
		checkQueueEmptiness(t, q.(*Queue))
	})

	t.Run("wrong order + 2nd fail", func(t *testing.T) {
		q, submission, firstTwoJobIds := prepare()
		handler, err := q.JobCompleted(&masterconn.InvokerJobResult{JobID: firstTwoJobIds[1], Verdict: verdict.WA})
		assert.Nil(t, err)
		assert.Equal(t, submission, handler)
		handler, err = q.JobCompleted(&masterconn.InvokerJobResult{JobID: firstTwoJobIds[0], Verdict: verdict.OK})
		assert.Nil(t, err)
		assert.Nil(t, handler)

		assert.Nil(t, q.NextJob())
		checkQueueEmptiness(t, q.(*Queue))
	})

	t.Run("wrong order + 1st fail", func(t *testing.T) {
		q, submission, firstTwoJobIds := prepare()
		fmt.Println(firstTwoJobIds)
		handler, err := q.JobCompleted(&masterconn.InvokerJobResult{JobID: firstTwoJobIds[1], Verdict: verdict.OK})
		assert.Nil(t, err)
		assert.Nil(t, handler)
		handler, err = q.JobCompleted(&masterconn.InvokerJobResult{JobID: firstTwoJobIds[0], Verdict: verdict.WA})
		assert.Nil(t, err)
		assert.Equal(t, submission, handler)

		assert.Nil(t, q.NextJob())
		checkQueueEmptiness(t, q.(*Queue))
	})
}

func TestFailedCompilation(t *testing.T) {
	q := NewQueue(nil)
	submission := fixtureSubmission()
	assert.NoError(t, q.Submit(submission))
	job := nextJob(t, q, fixtureSubmissionID, invokerconn.CompileJob)
	handler, err := q.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.ID,
		Verdict: verdict.CE,
	})
	assert.Nil(t, err)
	assert.NotNil(t, handler)
	assert.Nil(t, q.NextJob())
	checkQueueEmptiness(t, q.(*Queue))
}

func TestFailedTesting(t *testing.T) {
	q := NewQueue(nil)
	submission := fixtureSubmission()
	assert.NoError(t, q.Submit(submission))
	job := nextJob(t, q, fixtureSubmissionID, invokerconn.CompileJob)
	assert.Nil(t, q.NextJob())
	handler, err := q.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.ID,
		Verdict: verdict.CD,
	})
	assert.Nil(t, err)
	assert.Nil(t, handler)

	ids := make([]string, 0)
	for i := range 2 {
		job = q.NextJob()
		assert.NotNil(t, job)
		assert.Equal(t, job.Type, invokerconn.TestJob)
		assert.Equal(t, i+1, int(job.Test))
		ids = append(ids, job.ID)
	}
	job = q.NextJob()
	handler, err = q.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.ID,
		Verdict: verdict.WA,
	})
	assert.Nil(t, err)
	assert.NotNil(t, handler)
	// no jobs should be in the queue
	assert.Nil(t, q.NextJob())
	checkQueueEmptiness(t, q.(*Queue))
	// now finish that tasks
	for _, id := range ids {
		handler, err = q.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   id,
			Verdict: verdict.OK,
		})
		assert.Nil(t, err)
		assert.Nil(t, handler)
	}
	checkQueueEmptiness(t, q.(*Queue))
	// one more time just for fun
	handler, err = q.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   ids[0],
		Verdict: verdict.WA,
	})
	assert.Nil(t, err)
	assert.Nil(t, handler)
	checkQueueEmptiness(t, q.(*Queue))
}

func TestFinishJobTwice(t *testing.T) {
	q := NewQueue(nil)
	submission := fixtureSubmission()
	assert.NoError(t, q.Submit(submission))
	job := nextJob(t, q, fixtureSubmissionID, invokerconn.CompileJob)
	assert.Nil(t, q.NextJob())
	handler, err := q.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.ID,
		Verdict: verdict.CD,
	})
	assert.Nil(t, err)
	assert.Nil(t, handler)
	handler, err = q.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.ID,
		Verdict: verdict.CD,
	})
	assert.Nil(t, err)
	assert.Nil(t, handler)
}
