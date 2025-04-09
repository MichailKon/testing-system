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

func nextJob(t *testing.T, g Generator, SubmitID uint, jobType invokerconn.JobType, test uint64) *GeneratorJob {
	job, err := g.NextJob()
	assert.Nil(t, err)
	assert.NotNil(t, job)
	assert.Equal(t, job.InvokerJob.Type, jobType)
	assert.Equal(t, SubmitID, job.InvokerJob.SubmitID)
	assert.Equal(t, test, job.InvokerJob.Test)
	return job
}

func TestStraightTasksFinishing(t *testing.T) {
	problem := fixtureProblem()
	generator, err := NewGenerator(problem, 1)
	require.Nil(t, err)
	job := nextJob(t, generator, 1, invokerconn.CompileJob, 0)
	require.False(t, generator.CanGiveJob())
	require.Nil(t, generator.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.InvokerJob.ID,
		Verdict: verdict.CD,
	}))
	for i := range 9 {
		job = nextJob(t, generator, 1, invokerconn.TestJob, uint64(i)+1)
		require.False(t, generator.IsTestingCompleted())
		require.NoError(t, generator.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   job.InvokerJob.ID,
			Verdict: verdict.OK,
		}))
	}
	job = nextJob(t, generator, 1, invokerconn.TestJob, 10)
	require.False(t, generator.IsTestingCompleted())
	require.Nil(t, generator.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.InvokerJob.ID,
		Verdict: verdict.OK,
	}))
	require.True(t, generator.IsTestingCompleted())
	score, err := generator.Score()
	require.Nil(t, err)
	require.Equal(t, 1., score)
}

func TestTasksFinishing(t *testing.T) {
	prepare := func() (Generator, []string) {
		g, err := NewGenerator(fixtureProblem(), 1)
		require.Nil(t, err)
		job := nextJob(t, g, 1, invokerconn.CompileJob, 0)
		require.Nil(t, g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   job.InvokerJob.ID,
			Verdict: verdict.CD,
		}))
		firstTwoJobIds := make([]string, 0)
		for i := range 2 {
			job = nextJob(t, g, 1, invokerconn.TestJob, uint64(i)+1)
			firstTwoJobIds = append(firstTwoJobIds, job.InvokerJob.ID)
		}
		return g, firstTwoJobIds
	}
	finishOtherTests := func(g Generator) {
		for i := 2; i < 9; i++ {
			job := nextJob(t, g, 1, invokerconn.TestJob, uint64(i)+1)
			require.Nil(t, g.JobCompleted(&masterconn.InvokerJobResult{
				JobID:   job.InvokerJob.ID,
				Verdict: verdict.OK,
			}))
		}
		job := nextJob(t, g, 1, invokerconn.TestJob, 10)
		require.Nil(t, g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   job.InvokerJob.ID,
			Verdict: verdict.OK,
		}))
		score, err := g.Score()
		require.Nil(t, err)
		require.Equal(t, 1., score)
	}

	t.Run("right order", func(t *testing.T) {
		g, firstTwoJobIds := prepare()
		for _, id := range firstTwoJobIds {
			require.Nil(t, g.JobCompleted(&masterconn.InvokerJobResult{
				JobID:   id,
				Verdict: verdict.OK,
			}))
		}
		finishOtherTests(g)
	})

	t.Run("wrong order + both ok", func(t *testing.T) {
		g, firstTwoJobIds := prepare()
		require.Nil(t, g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   firstTwoJobIds[1],
			Verdict: verdict.OK,
		}))
		require.Nil(t, g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   firstTwoJobIds[0],
			Verdict: verdict.OK,
		}))

		finishOtherTests(g)
	})

	t.Run("wrong order + 2nd fail", func(t *testing.T) {
		g, firstTwoJobIds := prepare()
		require.Nil(t, g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   firstTwoJobIds[1],
			Verdict: verdict.WA,
		}))
		// this task no longer exists, so any result may be here
		_ = g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   firstTwoJobIds[0],
			Verdict: verdict.OK,
		})

		require.True(t, g.IsTestingCompleted())
		score, err := g.Score()
		require.Nil(t, err)
		require.Equal(t, 0., score)
	})

	t.Run("wrong order + 1st fail", func(t *testing.T) {
		g, firstTwoJobIds := prepare()
		require.Nil(t, g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   firstTwoJobIds[0],
			Verdict: verdict.WA,
		}))
		// this task no longer exists, so any result may be here
		_ = g.JobCompleted(&masterconn.InvokerJobResult{
			JobID:   firstTwoJobIds[1],
			Verdict: verdict.OK,
		})

		require.True(t, g.IsTestingCompleted())
		score, err := g.Score()
		require.Nil(t, err)
		require.Equal(t, 0., score)
	})
}

func TestFailedCompilation(t *testing.T) {
	g, err := NewGenerator(fixtureProblem(), 1)
	require.Nil(t, err)
	job := nextJob(t, g, 1, invokerconn.CompileJob, 0)
	require.Nil(t, g.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.InvokerJob.ID,
		Verdict: verdict.CE,
	}))
	require.True(t, g.IsTestingCompleted())
	score, err := g.Score()
	require.Nil(t, err)
	require.Equal(t, 0., score)
}

func TestFinishSameJobTwice(t *testing.T) {
	g, err := NewGenerator(fixtureProblem(), 1)
	require.Nil(t, err)
	job := nextJob(t, g, 1, invokerconn.CompileJob, 0)
	require.Nil(t, g.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.InvokerJob.ID,
		Verdict: verdict.CE,
	}))
	require.True(t, g.IsTestingCompleted())
	require.NotNil(t, g.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.InvokerJob.ID,
		Verdict: verdict.CE,
	}))
	score, err := g.Score()
	require.Nil(t, err)
	require.Equal(t, 0., score)
}

func TestICPCGenerator_IsTestingCompleted(t *testing.T) {
	problem := fixtureProblem()
	problem.TestsNumber = 1
	g, err := NewGenerator(problem, 1)
	require.Nil(t, err)
	job := nextJob(t, g, 1, invokerconn.CompileJob, 0)
	require.Nil(t, g.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.InvokerJob.ID,
		Verdict: verdict.CD,
	}))
	require.False(t, g.IsTestingCompleted())
	job = nextJob(t, g, 1, invokerconn.TestJob, 1)
	require.False(t, g.IsTestingCompleted())
	require.Nil(t, g.JobCompleted(&masterconn.InvokerJobResult{
		JobID:   job.InvokerJob.ID,
		Verdict: verdict.WA,
	}))
	require.True(t, g.IsTestingCompleted())
	score, err := g.Score()
	require.Nil(t, err)
	require.Equal(t, 0., score)
}
