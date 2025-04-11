package jobgenerators

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
)

func fixtureProblem() *models.Problem {
	return &models.Problem{
		ProblemType: models.ProblemType_ICPC,
		TestsNumber: 10,
	}
}

func fixtureSubmission() *models.Submission {
	submission := &models.Submission{}
	submission.ID = 1
	return submission
}

func nextJob(t *testing.T, g Generator, SubmitID uint, jobType invokerconn.JobType, test uint64) *invokerconn.Job {
	job := g.NextJob()
	assert.NotNil(t, job)
	assert.Equal(t, job.Type, jobType)
	assert.Equal(t, SubmitID, job.SubmitID)
	assert.Equal(t, test, job.Test)
	return job
}

func noJobs(t *testing.T, g Generator) {
	job := g.NextJob()
	assert.Nil(t, job)
}

func TestStraightTasksFinishing(t *testing.T) {
	problem := fixtureProblem()
	submission := fixtureSubmission()
	generator, err := NewGenerator(problem, submission)
	require.Nil(t, err)
	job := nextJob(t, generator, 1, invokerconn.CompileJob, 0)
	noJobs(t, generator)
	sub, err := generator.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.ID,
		Verdict: verdict.CD,
	})
	require.Nil(t, sub)
	require.Nil(t, err)
	for i := range 9 {
		job = nextJob(t, generator, 1, invokerconn.TestJob, uint64(i)+1)
		sub, err = generator.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   job.ID,
			Verdict: verdict.OK,
		})
		require.Nil(t, sub)
		require.Nil(t, err)
	}
	job = nextJob(t, generator, 1, invokerconn.TestJob, 10)
	sub, err = generator.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.ID,
		Verdict: verdict.OK,
	})
	require.NotNil(t, sub)
	require.Nil(t, err)

	require.Equal(t, verdict.OK, sub.Verdict)
	require.Equal(t, 1., sub.Score)
	for i, result := range sub.TestResults {
		require.Equal(t, verdict.OK, result.Verdict)
		require.Equal(t, uint64(i)+1, result.TestNumber)
	}
}

func TestTasksFinishing(t *testing.T) {
	prepare := func() (Generator, []string) {
		problem := fixtureProblem()
		submission := fixtureSubmission()
		g, err := NewGenerator(problem, submission)
		require.Nil(t, err)
		job := nextJob(t, g, 1, invokerconn.CompileJob, 0)
		sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   job.ID,
			Verdict: verdict.CD,
		})
		require.Nil(t, sub)
		require.Nil(t, err)
		firstTwoJobIds := make([]string, 0)
		for i := range 2 {
			job = nextJob(t, g, 1, invokerconn.TestJob, uint64(i)+1)
			firstTwoJobIds = append(firstTwoJobIds, job.ID)
		}
		return g, firstTwoJobIds
	}
	finishOtherTests := func(g Generator) {
		for i := 2; i < 9; i++ {
			job := nextJob(t, g, 1, invokerconn.TestJob, uint64(i)+1)
			sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
				JobID:   job.ID,
				Verdict: verdict.OK,
			})
			require.Nil(t, sub)
			require.Nil(t, err)
		}
		job := nextJob(t, g, 1, invokerconn.TestJob, 10)
		sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   job.ID,
			Verdict: verdict.OK,
		})
		require.NotNil(t, sub)
		require.Nil(t, err)

		require.Equal(t, verdict.OK, sub.Verdict)
		require.Equal(t, 1., sub.Score)
		for i, result := range sub.TestResults {
			require.Equal(t, verdict.OK, result.Verdict)
			require.Equal(t, uint64(i)+1, result.TestNumber)
		}
	}

	t.Run("right order", func(t *testing.T) {
		g, firstTwoJobIds := prepare()
		for _, id := range firstTwoJobIds {
			sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
				JobID:   id,
				Verdict: verdict.OK,
			})
			require.Nil(t, sub)
			require.Nil(t, err)
		}
		finishOtherTests(g)
	})

	t.Run("wrong order + both ok", func(t *testing.T) {
		g, firstTwoJobIds := prepare()
		sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   firstTwoJobIds[1],
			Verdict: verdict.OK,
		})
		require.Nil(t, sub)
		require.Nil(t, err)

		sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   firstTwoJobIds[0],
			Verdict: verdict.OK,
		})
		require.Nil(t, sub)
		require.Nil(t, err)

		finishOtherTests(g)
	})

	t.Run("wrong order + 2nd fail", func(t *testing.T) {
		g, firstTwoJobIds := prepare()
		sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   firstTwoJobIds[1],
			Verdict: verdict.WA,
		})
		require.Nil(t, sub)
		require.Nil(t, err)

		sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   firstTwoJobIds[0],
			Verdict: verdict.OK,
		})
		require.NotNil(t, sub)
		require.Nil(t, err)

		require.Equal(t, verdict.WA, sub.Verdict)
		require.Equal(t, 0., sub.Score)
		require.Equal(t, verdict.OK, sub.TestResults[0].Verdict)
		require.Equal(t, verdict.WA, sub.TestResults[1].Verdict)
		for i, result := range sub.TestResults[2:] {
			require.Equal(t, verdict.SK, result.Verdict)
			require.Equal(t, uint64(i)+3, result.TestNumber)
		}
	})

	t.Run("wrong order + 1st fail", func(t *testing.T) {
		g, firstTwoJobIds := prepare()
		sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   firstTwoJobIds[1],
			Verdict: verdict.OK,
		})
		require.Nil(t, sub)
		require.Nil(t, err)

		sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   firstTwoJobIds[0],
			Verdict: verdict.WA,
		})
		require.NotNil(t, sub)
		require.Nil(t, err)

		require.Equal(t, verdict.WA, sub.Verdict)
		require.Equal(t, 0., sub.Score)
		require.Equal(t, verdict.WA, sub.TestResults[0].Verdict)
		require.Equal(t, verdict.SK, sub.TestResults[1].Verdict)
		for i, result := range sub.TestResults[2:] {
			require.Equal(t, verdict.SK, result.Verdict)
			require.Equal(t, uint64(i)+3, result.TestNumber)
		}
	})
}

func TestFailedCompilation(t *testing.T) {
	problem, submission := fixtureProblem(), fixtureSubmission()
	g, err := NewGenerator(problem, submission)
	require.Nil(t, err)
	job := nextJob(t, g, 1, invokerconn.CompileJob, 0)
	sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.ID,
		Verdict: verdict.CE,
	})
	require.NotNil(t, sub)
	require.Nil(t, err)

	require.Equal(t, verdict.CE, sub.Verdict)
	require.Equal(t, 0., sub.Score)
	for i, result := range sub.TestResults {
		require.Equal(t, verdict.SK, result.Verdict)
		require.Equal(t, uint64(i)+1, result.TestNumber)
	}
}

func TestFinishSameJobTwice(t *testing.T) {
	problem, submission := fixtureProblem(), fixtureSubmission()
	g, err := NewGenerator(problem, submission)
	require.Nil(t, err)
	job := nextJob(t, g, 1, invokerconn.CompileJob, 0)
	sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.ID,
		Verdict: verdict.CE,
	})
	require.NotNil(t, sub)
	require.Nil(t, err)

	require.Equal(t, verdict.CE, sub.Verdict)
	require.Equal(t, 0., sub.Score)
	for i, result := range sub.TestResults {
		require.Equal(t, verdict.SK, result.Verdict)
		require.Equal(t, uint64(i)+1, result.TestNumber)
	}

	sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.ID,
		Verdict: verdict.CE,
	})
	require.Nil(t, sub)
	require.NotNil(t, err)
}
