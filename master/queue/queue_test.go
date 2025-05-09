package queue

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"slices"
	"strings"
	"testing"
	"testing_system/common"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/common/metrics"
)

func doQueueCycles(t *testing.T, q *Queue, cycles int, maxNoJobs int) int {
	finishedTasks := 0
	for range cycles {
		job := q.NextJob()
		if job == nil {
			require.Greater(t, maxNoJobs, 0)
			maxNoJobs--
			continue
		}
		if job.Type == invokerconn.CompileJob {
			sub, err := q.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.CD,
			})
			assert.Nil(t, err)
			assert.Nil(t, sub)
		} else {
			sub, err := q.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.OK,
			})
			assert.Nil(t, err)
			if sub != nil {
				finishedTasks++
			}
		}
	}
	return finishedTasks
}

func isQueueEmpty(q *Queue) bool {
	return len(q.jobIDToOriginalJobID) == 0 &&
		len(q.newFailedJobs) == 0 &&
		len(q.originalJobIDToJob) == 0 &&
		len(q.originalJobIDToGenerator) == 0 &&
		q.activeGenerators.Len() == 0
}

func createQueue() *Queue {
	ts := &common.TestingSystem{
		Metrics: metrics.NewCollector(),
	}
	return NewQueue(ts).(*Queue)
}

func TestQueueWork(t *testing.T) {
	q := createQueue()
	problem1 := models.Problem{
		TestsNumber: 2,
		ProblemType: models.ProblemTypeICPC,
	}
	problem2 := models.Problem{
		TestsNumber: 2,
		ProblemType: models.ProblemTypeICPC,
	}
	problem1.ID, problem2.ID = 1, 2
	submission1 := models.Submission{}
	submission2 := models.Submission{}
	submission1.ID, submission2.ID = 1, 2
	err := q.Submit(&problem1, &submission1)
	require.Nil(t, err)
	err = q.Submit(&problem2, &submission2)
	require.Nil(t, err)
	// 7 cycles because we need 1 more cycle to remove generators (since they are not empty now)
	require.Equal(t, 2, doQueueCycles(t, q, 7, 1))
	require.True(t, isQueueEmpty(q))
}

func TestQueueFairness(t *testing.T) {
	q := createQueue()
	problem1 := models.Problem{
		TestsNumber: 500,
		ProblemType: models.ProblemTypeICPC,
	}
	problem2 := models.Problem{
		TestsNumber: 10,
		ProblemType: models.ProblemTypeICPC,
	}
	problem1.ID, problem2.ID = 1, 2
	submission1 := models.Submission{}
	submission2 := models.Submission{}
	submission1.ID, submission2.ID = 1, 2
	err := q.Submit(&problem1, &submission1)
	require.Nil(t, err)
	err = q.Submit(&problem2, &submission2)
	require.Nil(t, err)
	require.Equal(t, 1, doQueueCycles(t, q, 22, 1))
	require.False(t, isQueueEmpty(q))
	// one more cycle to remove generator from the 1st task
	require.Equal(t, 1, doQueueCycles(t, q, 491, 1))
	require.True(t, isQueueEmpty(q))
}

func TestQueue_RescheduleJob(t *testing.T) {
	prepare := func() *Queue {
		q := createQueue()
		require.True(t, isQueueEmpty(q))
		problem1 := models.Problem{
			TestsNumber: 1,
			ProblemType: models.ProblemTypeICPC,
		}
		problem2 := problem1
		submission1 := models.Submission{}
		submission2 := models.Submission{}
		submission1.ID, submission2.ID = 1, 2
		err := q.Submit(&problem1, &submission1)
		require.Nil(t, err)
		err = q.Submit(&problem2, &submission2)
		require.Nil(t, err)
		return q
	}

	t.Run("normal", func(t *testing.T) {
		q := prepare()
		job := *q.NextJob() // this job may change since it's a pointer, so we need to dereference it
		require.NotNil(t, job)

		// reschedule
		err := q.RescheduleJob(job.ID)
		require.Nil(t, err)

		newJob := q.NextJob()
		require.NotNil(t, newJob)
		require.NotEqual(t, job.ID, newJob.ID)
		require.Equal(t, job.SubmitID, newJob.SubmitID)

		_, err = q.JobCompleted(&masterconn.InvokerJobResult{
			Job:     newJob,
			Verdict: verdict.CD,
		})
		assert.Nil(t, err)
	})

	t.Run("next -> reschedule -> next -> next", func(t *testing.T) {
		q := prepare()
		job := *q.NextJob()
		require.NotNil(t, job)
		err := q.RescheduleJob(job.ID)
		require.Nil(t, err)
		newJob := q.NextJob()
		require.NotNil(t, newJob)
		require.NotEqual(t, job.ID, newJob.ID)
		require.Equal(t, uint(1), newJob.SubmitID)
		anotherJob := q.NextJob()
		require.Equal(t, uint(2), anotherJob.SubmitID)
	})

	t.Run("finish first job", func(t *testing.T) {
		q := prepare()
		job := *q.NextJob() // this job may change since it's a pointer, so we need to dereference it
		require.NotNil(t, job)

		// reschedule
		err := q.RescheduleJob(job.ID)
		require.Nil(t, err)

		newJob := q.NextJob()
		require.NotNil(t, newJob)
		require.NotEqual(t, job.ID, newJob.ID)
		require.Equal(t, job.SubmitID, newJob.SubmitID)

		_, err = q.JobCompleted(&masterconn.InvokerJobResult{
			Job:     newJob,
			Verdict: verdict.CD,
		})
		assert.Nil(t, err) // this job should not be found

		_, err = q.JobCompleted(&masterconn.InvokerJobResult{
			Job:     &job,
			Verdict: verdict.CD,
		})
		assert.NotNil(t, err)
	})

	t.Run("spam reschedules", func(t *testing.T) {
		spamJobs := func(q *Queue, submitID uint) invokerconn.Job {
			jobs := make([]invokerconn.Job, 0)
			for i := range 100 {
				job := *q.NextJob()
				jobs = append(jobs, job)
				require.NotNil(t, job)
				require.Equal(t, job.SubmitID, submitID)
				if i != 99 {
					require.Nil(t, q.RescheduleJob(job.ID))
				}
			}
			lastJob := jobs[len(jobs)-1]
			slices.SortFunc(jobs, func(a, b invokerconn.Job) int {
				return strings.Compare(a.ID, b.ID)
			})
			jobs = slices.CompactFunc(jobs, func(job invokerconn.Job, job2 invokerconn.Job) bool {
				return job.ID == job2.ID
			})
			require.Equal(t, 100, len(jobs))
			return lastJob
		}
		q := prepare()

		// compile 1
		lastJob := spamJobs(q, 1)
		_, err := q.JobCompleted(&masterconn.InvokerJobResult{
			Job:     &lastJob,
			Verdict: verdict.CD,
		})
		require.Nil(t, err)

		// compile 2
		lastJob = spamJobs(q, 2)
		_, err = q.JobCompleted(&masterconn.InvokerJobResult{
			Job:     &lastJob,
			Verdict: verdict.CD,
		})
		require.Nil(t, err)

		// test1
		lastJob = spamJobs(q, 1)
		_, err = q.JobCompleted(&masterconn.InvokerJobResult{
			Job:     &lastJob,
			Verdict: verdict.OK,
		})
		require.Nil(t, err)

		// test2
		lastJob = spamJobs(q, 2)
		_, err = q.JobCompleted(&masterconn.InvokerJobResult{
			Job:     &lastJob,
			Verdict: verdict.OK,
		})
		require.Nil(t, err)

		// clean up generators
		require.Nil(t, q.NextJob())

		require.True(t, isQueueEmpty(q))
	})
}

func TestQueueWrongJobID(t *testing.T) {
	q := createQueue()
	sub, err := q.JobCompleted(&masterconn.InvokerJobResult{
		Job:     &invokerconn.Job{},
		Verdict: verdict.CD,
	})
	assert.Nil(t, sub)
	assert.NotNil(t, err)
}
