package queue

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
)

func doQueueCycles(t *testing.T, q IQueue, cycles int) int {
	finishedTasks := 0
	for range cycles {
		job := q.NextJob()
		if job.Type == invokerconn.CompileJob {
			sub, err := q.JobCompleted(&masterconn.InvokerJobResult{
				JobID:   job.ID,
				Verdict: verdict.CD,
			})
			assert.Nil(t, err)
			assert.Nil(t, sub)
		} else {
			sub, err := q.JobCompleted(&masterconn.InvokerJobResult{
				JobID:   job.ID,
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

func TestQueueWork(t *testing.T) {
	q := NewQueue(nil)
	problem1 := models.Problem{
		TestsNumber: 1,
		ProblemType: models.ProblemType_ICPC,
	}
	problem2 := models.Problem{
		TestsNumber: 1,
		ProblemType: models.ProblemType_ICPC,
	}
	problem1.ID, problem2.ID = 1, 2
	submission1 := models.Submission{}
	submission2 := models.Submission{}
	submission1.ID, submission2.ID = 1, 2
	err := q.Submit(&problem1, &submission1)
	require.Nil(t, err)
	err = q.Submit(&problem2, &submission2)
	require.Nil(t, err)
	assert.Equal(t, 2, doQueueCycles(t, q, 4))
}

func TestQueueFairness(t *testing.T) {
	q := NewQueue(nil)
	problem1 := models.Problem{
		TestsNumber: 500,
		ProblemType: models.ProblemType_ICPC,
	}
	problem2 := models.Problem{
		TestsNumber: 10,
		ProblemType: models.ProblemType_ICPC,
	}
	problem1.ID, problem2.ID = 1, 2
	submission1 := models.Submission{}
	submission2 := models.Submission{}
	submission1.ID, submission2.ID = 1, 2
	err := q.Submit(&problem1, &submission1)
	require.Nil(t, err)
	err = q.Submit(&problem2, &submission2)
	require.Nil(t, err)
	assert.Equal(t, 1, doQueueCycles(t, q, 22))
}
