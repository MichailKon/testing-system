package jobgenerators

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xorcare/pointer"
	"testing"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
)

func fixtureSubmission(ID uint) *models.Submission {
	submission := &models.Submission{}
	submission.ID = ID
	return submission
}

func nextJob(t *testing.T, g Generator, SubmitID uint, jobType invokerconn.JobType, test uint64) *invokerconn.Job {
	job := g.NextJob()
	require.NotNil(t, job)
	require.Equal(t, jobType, job.Type)
	require.Equal(t, SubmitID, job.SubmitID)
	require.Equal(t, test, job.Test)
	return job
}

func noJobs(t *testing.T, g Generator) {
	job := g.NextJob()
	require.Nil(t, job)
}

func TestICPCGenerator(t *testing.T) {
	const fixtureICPCProblemTestsNumber = 10
	fixtureICPCProblem := func() *models.Problem {
		return &models.Problem{
			ProblemType: models.ProblemTypeICPC,
			TestsNumber: fixtureICPCProblemTestsNumber,
		}
	}

	t.Run("Fail compilation", func(t *testing.T) {
		problem, submission := fixtureICPCProblem(), fixtureSubmission(1)
		g, err := NewGenerator(problem, submission)
		require.Nil(t, err)
		job := nextJob(t, g, 1, invokerconn.CompileJob, 0)
		sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
			Job:     job,
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
	})

	t.Run("Straight tasks finishing", func(t *testing.T) {
		problem := fixtureICPCProblem()
		submission := fixtureSubmission(1)
		generator, err := NewGenerator(problem, submission)
		require.Nil(t, err)
		job := nextJob(t, generator, 1, invokerconn.CompileJob, 0)
		noJobs(t, generator)
		sub, err := generator.JobCompleted(&masterconn.InvokerJobResult{
			Job:     job,
			Verdict: verdict.CD,
		})
		require.Nil(t, sub)
		require.Nil(t, err)
		for i := range fixtureICPCProblemTestsNumber - 1 {
			job = nextJob(t, generator, 1, invokerconn.TestJob, uint64(i)+1)
			sub, err = generator.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.OK,
			})
			require.Nil(t, sub)
			require.Nil(t, err)
		}
		job = nextJob(t, generator, 1, invokerconn.TestJob, fixtureICPCProblemTestsNumber)
		sub, err = generator.JobCompleted(&masterconn.InvokerJobResult{
			Job:     job,
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
	})

	t.Run("Tasks finishing", func(t *testing.T) {
		prepare := func() (Generator, []*invokerconn.Job) {
			problem := fixtureICPCProblem()
			submission := fixtureSubmission(1)
			g, err := NewGenerator(problem, submission)
			require.Nil(t, err)
			job := nextJob(t, g, 1, invokerconn.CompileJob, 0)
			sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.CD,
			})
			require.Nil(t, sub)
			require.Nil(t, err)
			firstTwoJobs := make([]*invokerconn.Job, 0)
			for i := range 2 {
				job = nextJob(t, g, 1, invokerconn.TestJob, uint64(i)+1)
				firstTwoJobs = append(firstTwoJobs, job)
			}
			return g, firstTwoJobs
		}
		finishOtherTests := func(g Generator) {
			for i := 2; i < fixtureICPCProblemTestsNumber-1; i++ {
				job := nextJob(t, g, 1, invokerconn.TestJob, uint64(i)+1)
				sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
					Job:     job,
					Verdict: verdict.OK,
				})
				require.Nil(t, sub)
				require.Nil(t, err)
			}
			job := nextJob(t, g, 1, invokerconn.TestJob, 10)
			sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
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
			g, firstTwoJobs := prepare()
			for _, job := range firstTwoJobs {
				sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
					Job:     job,
					Verdict: verdict.OK,
				})
				require.Nil(t, sub)
				require.Nil(t, err)
			}
			finishOtherTests(g)
		})

		t.Run("wrong order + both ok", func(t *testing.T) {
			g, firstTwoJobs := prepare()
			sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     firstTwoJobs[1],
				Verdict: verdict.OK,
			})
			require.Nil(t, sub)
			require.Nil(t, err)

			sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     firstTwoJobs[0],
				Verdict: verdict.OK,
			})
			require.Nil(t, sub)
			require.Nil(t, err)

			finishOtherTests(g)
		})

		t.Run("wrong order + 2nd fail", func(t *testing.T) {
			g, firstTwoJobs := prepare()
			sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     firstTwoJobs[1],
				Verdict: verdict.WA,
			})
			require.Nil(t, sub)
			require.Nil(t, err)

			sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     firstTwoJobs[0],
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
			g, firstTwoJobs := prepare()
			sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     firstTwoJobs[1],
				Verdict: verdict.OK,
			})
			require.Nil(t, sub)
			require.Nil(t, err)

			sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     firstTwoJobs[0],
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
	})

	t.Run("Finish same job twice", func(t *testing.T) {
		problem, submission := fixtureICPCProblem(), fixtureSubmission(1)
		g, err := NewGenerator(problem, submission)
		require.Nil(t, err)
		job := nextJob(t, g, 1, invokerconn.CompileJob, 0)
		sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
			Job:     job,
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
			Job:     job,
			Verdict: verdict.CE,
		})
		require.Nil(t, sub)
		require.NotNil(t, err)
	})
}

func TestIOIGenerator(t *testing.T) {
	t.Run("Bad problem configuration", func(t *testing.T) {
		badProblems := []models.Problem{
			// type EachTest, but TestScore is nil
			{
				ProblemType: models.ProblemTypeIOI,
				TestGroups: []models.TestGroup{
					{
						Name:               "name",
						FirstTest:          1,
						LastTest:           1,
						TestScore:          nil,
						ScoringType:        models.TestGroupScoringTypeEachTest,
						RequiredGroupNames: make([]string, 0),
					},
				},
				TestsNumber: 1,
			},
			// type Complete, but GroupScore is nil
			{
				ProblemType: models.ProblemTypeIOI,
				TestGroups: []models.TestGroup{
					{
						Name:               "name",
						FirstTest:          1,
						LastTest:           1,
						GroupScore:         nil,
						ScoringType:        models.TestGroupScoringTypeComplete,
						RequiredGroupNames: make([]string, 0),
					},
				},
				TestsNumber: 1,
			},
			// type Min, but GroupScore is nil
			{
				ProblemType: models.ProblemTypeIOI,
				TestGroups: []models.TestGroup{
					{
						Name:               "name",
						FirstTest:          1,
						LastTest:           1,
						GroupScore:         nil,
						ScoringType:        models.TestGroupScoringTypeMin,
						RequiredGroupNames: make([]string, 0),
					},
				},
				TestsNumber: 1,
			},
			// cyclic groups
			{
				ProblemType: models.ProblemTypeIOI,
				TestGroups: []models.TestGroup{
					{
						Name:               "name1",
						FirstTest:          1,
						LastTest:           1,
						GroupScore:         pointer.Float64(1.0),
						ScoringType:        models.TestGroupScoringTypeMin,
						RequiredGroupNames: []string{"name2"},
					},
					{
						Name:               "name2",
						FirstTest:          2,
						LastTest:           2,
						GroupScore:         pointer.Float64(1.0),
						ScoringType:        models.TestGroupScoringTypeMin,
						RequiredGroupNames: []string{"name1"},
					},
				},
			},
			// test is not covered
			{
				ProblemType: models.ProblemTypeIOI,
				TestGroups: []models.TestGroup{
					{
						Name:               "name",
						FirstTest:          1,
						LastTest:           1,
						TestScore:          pointer.Float64(1.0),
						ScoringType:        models.TestGroupScoringTypeEachTest,
						RequiredGroupNames: make([]string, 0),
					},
				},
				TestsNumber: 2,
			},
			// test in several groups
			{
				ProblemType: models.ProblemTypeIOI,
				TestGroups: []models.TestGroup{
					{
						Name:               "name",
						FirstTest:          1,
						LastTest:           2,
						TestScore:          pointer.Float64(1.0),
						ScoringType:        models.TestGroupScoringTypeEachTest,
						RequiredGroupNames: make([]string, 0),
					},
					{
						Name:               "name1",
						FirstTest:          2,
						LastTest:           3,
						TestScore:          pointer.Float64(1.0),
						ScoringType:        models.TestGroupScoringTypeEachTest,
						RequiredGroupNames: make([]string, 0),
					},
				},
				TestsNumber: 3,
			},
			// the same group name
			{
				ProblemType: models.ProblemTypeIOI,
				TestGroups: []models.TestGroup{
					{
						Name:               "name",
						FirstTest:          1,
						LastTest:           2,
						TestScore:          pointer.Float64(1.0),
						ScoringType:        models.TestGroupScoringTypeEachTest,
						RequiredGroupNames: make([]string, 0),
					},
					{
						Name:               "name",
						FirstTest:          3,
						LastTest:           3,
						TestScore:          pointer.Float64(1.0),
						ScoringType:        models.TestGroupScoringTypeEachTest,
						RequiredGroupNames: make([]string, 0),
					},
				},
				TestsNumber: 3,
			},
			// first test > last test
			{
				ProblemType: models.ProblemTypeIOI,
				TestGroups: []models.TestGroup{
					{
						Name:               "name1",
						FirstTest:          1,
						LastTest:           2,
						TestScore:          pointer.Float64(1.0),
						ScoringType:        models.TestGroupScoringTypeEachTest,
						RequiredGroupNames: make([]string, 0),
					},
					{
						Name:               "name2",
						FirstTest:          2,
						LastTest:           1,
						TestScore:          pointer.Float64(1.0),
						ScoringType:        models.TestGroupScoringTypeEachTest,
						RequiredGroupNames: make([]string, 0),
					},
					{
						Name:               "name3",
						FirstTest:          3,
						LastTest:           3,
						TestScore:          pointer.Float64(1.0),
						ScoringType:        models.TestGroupScoringTypeEachTest,
						RequiredGroupNames: make([]string, 0),
					},
				},
			},
		}
		for _, problem := range badProblems {
			_, err := NewIOIGenerator(&problem, &models.Submission{})
			require.Error(t, err)
		}
	})

	problemWithOneGroup := models.Problem{
		ProblemType: models.ProblemTypeIOI,
		TestsNumber: 10,
		TestGroups: []models.TestGroup{
			{
				Name:               "group1",
				FirstTest:          1,
				LastTest:           10,
				TestScore:          nil,
				GroupScore:         pointer.Float64(100),
				ScoringType:        models.TestGroupScoringTypeComplete,
				RequiredGroupNames: make([]string, 0),
			},
		},
	}

	t.Run("Fail compilation", func(t *testing.T) {
		submission := fixtureSubmission(1)
		wasProblem := problemWithOneGroup
		g, err := NewGenerator(&problemWithOneGroup, submission)
		require.Nil(t, err)
		job := nextJob(t, g, 1, invokerconn.CompileJob, 0)
		sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
			Job:     job,
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
		require.Equal(t, wasProblem, problemWithOneGroup)
		require.Equal(t, models.GroupResults{
			{
				GroupName: "group1",
				Points:    0.,
				Passed:    false,
			},
		}, sub.GroupResults)
	})

	t.Run("Straight task finishing", func(t *testing.T) {
		wasProblem := problemWithOneGroup
		submission := fixtureSubmission(1)
		generator, err := NewGenerator(&problemWithOneGroup, submission)
		require.Nil(t, err)
		job := nextJob(t, generator, 1, invokerconn.CompileJob, 0)
		noJobs(t, generator)
		sub, err := generator.JobCompleted(&masterconn.InvokerJobResult{
			Job:     job,
			Verdict: verdict.CD,
		})
		require.Nil(t, sub)
		require.Nil(t, err)
		for i := range 9 {
			job = nextJob(t, generator, 1, invokerconn.TestJob, uint64(i)+1)
			sub, err = generator.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.OK,
			})
			require.Nil(t, sub)
			require.Nil(t, err)
		}
		job = nextJob(t, generator, 1, invokerconn.TestJob, 10)
		sub, err = generator.JobCompleted(&masterconn.InvokerJobResult{
			Job:     job,
			Verdict: verdict.OK,
		})
		require.NotNil(t, sub)
		require.Nil(t, err)

		require.Equal(t, verdict.OK, sub.Verdict)
		require.Equal(t, 100., sub.Score)
		for i, result := range sub.TestResults {
			require.Equal(t, verdict.OK, result.Verdict)
			require.Equal(t, uint64(i)+1, result.TestNumber)
		}
		require.Equal(t, sub.GroupResults, models.GroupResults{
			{
				GroupName: "group1",
				Points:    100.,
				Passed:    true,
			},
		})

		require.Equal(t, wasProblem, problemWithOneGroup)
	})

	t.Run("Fails in TestGroupScoringTypeComplete", func(t *testing.T) {
		prepare := func() Generator {
			problem := models.Problem{
				ProblemType: models.ProblemTypeIOI,
				TestsNumber: 2,
				TestGroups: []models.TestGroup{
					{
						Name:               "group1",
						FirstTest:          1,
						LastTest:           1,
						TestScore:          nil,
						GroupScore:         pointer.Float64(50),
						ScoringType:        models.TestGroupScoringTypeComplete,
						RequiredGroupNames: make([]string, 0),
					},
					{
						Name:               "group2",
						FirstTest:          2,
						LastTest:           2,
						TestScore:          nil,
						GroupScore:         pointer.Float64(50),
						ScoringType:        models.TestGroupScoringTypeComplete,
						RequiredGroupNames: []string{"group1"},
					},
				},
			}
			gen, err := NewGenerator(&problem, &models.Submission{})
			require.NoError(t, err)
			job := nextJob(t, gen, 0, invokerconn.CompileJob, 0)
			sub, err := gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.CD,
			})
			require.Nil(t, sub)
			require.NoError(t, err)
			return gen
		}

		t.Run("WA 1st, OK 2nd", func(t *testing.T) {
			gen := prepare()
			job1 := nextJob(t, gen, 0, invokerconn.TestJob, 1)
			job2 := nextJob(t, gen, 0, invokerconn.TestJob, 2)
			sub, err := gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job1,
				Verdict: verdict.WA,
			})
			require.Nil(t, sub)
			require.NoError(t, err)
			sub, err = gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job2,
				Verdict: verdict.OK,
			})
			require.Equal(t, verdict.PT, sub.Verdict)
			require.Equal(t, 0., sub.Score)
			require.Equal(t, verdict.WA, sub.TestResults[0].Verdict)
			require.Equal(t, verdict.SK, sub.TestResults[1].Verdict)
			require.Equal(t, sub.GroupResults, models.GroupResults{
				{
					GroupName: "group1",
					Points:    0.,
					Passed:    false,
				},
				{
					GroupName: "group2",
					Points:    0.,
					Passed:    false,
				},
			})
		})

		t.Run("OK 2nd, WA 1st", func(t *testing.T) {
			gen := prepare()
			job1 := nextJob(t, gen, 0, invokerconn.TestJob, 1)
			job2 := nextJob(t, gen, 0, invokerconn.TestJob, 2)
			sub, err := gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job2,
				Verdict: verdict.OK,
			})
			require.Nil(t, sub)
			require.NoError(t, err)
			sub, err = gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job1,
				Verdict: verdict.WA,
			})
			require.Equal(t, verdict.PT, sub.Verdict)
			require.Equal(t, 0., sub.Score)
			require.Equal(t, verdict.WA, sub.TestResults[0].Verdict)
			require.Equal(t, verdict.SK, sub.TestResults[1].Verdict)
			require.Equal(t, sub.GroupResults, models.GroupResults{
				{
					GroupName: "group1",
					Points:    0.,
					Passed:    false,
				},
				{
					GroupName: "group2",
					Points:    0.,
					Passed:    false,
				},
			})
		})

		t.Run("OK 1st, WA 2nd", func(t *testing.T) {
			gen := prepare()
			job1 := nextJob(t, gen, 0, invokerconn.TestJob, 1)
			job2 := nextJob(t, gen, 0, invokerconn.TestJob, 2)
			sub, err := gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job1,
				Verdict: verdict.OK,
			})
			require.Nil(t, sub)
			require.NoError(t, err)
			sub, err = gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job2,
				Verdict: verdict.WA,
			})
			require.Equal(t, verdict.PT, sub.Verdict)
			require.Equal(t, 50., sub.Score)
			require.Equal(t, verdict.OK, sub.TestResults[0].Verdict)
			require.Equal(t, verdict.WA, sub.TestResults[1].Verdict)
			require.Equal(t, sub.GroupResults, models.GroupResults{
				{
					GroupName: "group1",
					Points:    50.,
					Passed:    true,
				},
				{
					GroupName: "group2",
					Points:    0.,
					Passed:    false,
				},
			})
		})

		t.Run("WA 2nd, OK 1st", func(t *testing.T) {
			gen := prepare()
			job1 := nextJob(t, gen, 0, invokerconn.TestJob, 1)
			job2 := nextJob(t, gen, 0, invokerconn.TestJob, 2)
			sub, err := gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job2,
				Verdict: verdict.WA,
			})
			require.Nil(t, sub)
			require.NoError(t, err)
			sub, err = gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job1,
				Verdict: verdict.OK,
			})
			require.Equal(t, verdict.PT, sub.Verdict)
			require.Equal(t, 50., sub.Score)
			require.Equal(t, verdict.OK, sub.TestResults[0].Verdict)
			require.Equal(t, verdict.WA, sub.TestResults[1].Verdict)
			require.Equal(t, sub.GroupResults, models.GroupResults{
				{
					GroupName: "group1",
					Points:    50.,
					Passed:    true,
				},
				{
					GroupName: "group2",
					Points:    0.,
					Passed:    false,
				},
			})
		})

		t.Run("WA 1st, then take 2nd", func(t *testing.T) {
			gen := prepare()
			job := nextJob(t, gen, 0, invokerconn.TestJob, 1)
			sub, err := gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.WA,
			})
			require.NotNil(t, sub)
			require.NoError(t, err)
			assert.Nil(t, gen.NextJob())

			require.Equal(t, verdict.PT, sub.Verdict)
			require.Equal(t, 0., sub.Score)
			require.Equal(t, verdict.WA, sub.TestResults[0].Verdict)
			require.Equal(t, verdict.SK, sub.TestResults[1].Verdict)
			require.Equal(t, sub.GroupResults, models.GroupResults{
				{
					GroupName: "group1",
					Points:    0.,
					Passed:    false,
				},
				{
					GroupName: "group2",
					Points:    0.,
					Passed:    false,
				},
			})
		})

		t.Run("Fail in group with >1 tests", func(t *testing.T) {
			problem := models.Problem{
				ProblemType: models.ProblemTypeIOI,
				TestsNumber: 2,
				TestGroups: []models.TestGroup{
					{
						Name:               "group1",
						FirstTest:          1,
						LastTest:           2,
						TestScore:          nil,
						GroupScore:         pointer.Float64(100),
						ScoringType:        models.TestGroupScoringTypeComplete,
						RequiredGroupNames: make([]string, 0),
					},
				},
			}
			gen, err := NewGenerator(&problem, &models.Submission{})
			require.NoError(t, err)
			job := nextJob(t, gen, 0, invokerconn.CompileJob, 0)
			sub, err := gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.CD,
			})
			require.Nil(t, sub)
			require.NoError(t, err)
			job = nextJob(t, gen, 0, invokerconn.TestJob, 1)
			sub, err = gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.WA,
			})
			require.Nil(t, gen.NextJob())
			require.NotNil(t, sub)
			require.NoError(t, err)

			require.Equal(t, verdict.PT, sub.Verdict)
			require.Equal(t, 0., sub.Score)
			require.Equal(t, verdict.WA, sub.TestResults[0].Verdict)
			require.Equal(t, verdict.SK, sub.TestResults[1].Verdict)
			require.Equal(t, sub.GroupResults, models.GroupResults{
				{
					GroupName: "group1",
					Points:    0.,
					Passed:    false,
				},
			})
		})

		t.Run("OK, run, FAIL, get", func(t *testing.T) {
			baseStat := &masterconn.JobResultStatistics{
				Time:     100,
				Memory:   100,
				WallTime: 100,
				ExitCode: 0,
			}
			problem := models.Problem{
				ProblemType: models.ProblemTypeIOI,
				TestsNumber: 4,
				TestGroups: []models.TestGroup{
					{
						Name:               "group1",
						FirstTest:          1,
						LastTest:           4,
						TestScore:          nil,
						GroupScore:         pointer.Float64(100),
						ScoringType:        models.TestGroupScoringTypeComplete,
						RequiredGroupNames: make([]string, 0),
					},
				},
			}
			gen, err := NewGenerator(&problem, fixtureSubmission(1))
			require.NoError(t, err)
			job := nextJob(t, gen, 1, invokerconn.CompileJob, 0)
			sub, err := gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.CD,
			})
			require.Nil(t, sub)
			require.NoError(t, err)
			job1 := nextJob(t, gen, 1, invokerconn.TestJob, 1)
			job2 := nextJob(t, gen, 1, invokerconn.TestJob, 2)
			job3 := nextJob(t, gen, 1, invokerconn.TestJob, 3)
			// now finish 1 and 3
			sub, err = gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job1,
				Verdict: verdict.OK,
			})
			require.NoError(t, err)
			require.Nil(t, sub)
			sub, err = gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:        job3,
				Verdict:    verdict.WA,
				Statistics: baseStat,
			})
			// this group is already failed, so the generator should not return any job
			require.Nil(t, gen.NextJob())
			sub, err = gen.JobCompleted(&masterconn.InvokerJobResult{
				Job:        job2,
				Verdict:    verdict.TL,
				Statistics: baseStat,
			})
			require.NoError(t, err)
			require.Nil(t, gen.NextJob())
			require.NotNil(t, sub)

			require.Equal(t, verdict.PT, sub.Verdict)
			require.Equal(t, 0., sub.Score)
			require.Equal(t, sub.GroupResults, models.GroupResults{
				{
					GroupName: "group1",
					Points:    0.,
					Passed:    false,
				},
			})
			require.Equal(t, models.TestResult{
				TestNumber: 1,
				Points:     nil,
				Verdict:    verdict.OK,
			}, sub.TestResults[0])
			require.Equal(t, models.TestResult{
				TestNumber: 2,
				Points:     nil,
				Verdict:    verdict.TL,
				Time:       100,
				Memory:     100,
			}, sub.TestResults[1])
			require.Equal(t, models.TestResult{
				TestNumber: 3,
				Points:     nil,
				Verdict:    verdict.SK,
			}, sub.TestResults[2])
			require.Equal(t, models.TestResult{
				TestNumber: 4,
				Points:     nil,
				Verdict:    verdict.SK,
			}, sub.TestResults[3])
		})
	})

	t.Run("Fails in TestGroupScoringTypeEachTest", func(t *testing.T) {
		t.Run("WA in the middle of the group", func(t *testing.T) {
			problem := models.Problem{
				ProblemType: models.ProblemTypeIOI,
				TestsNumber: 3,
				TestGroups: []models.TestGroup{
					{
						Name:               "group1",
						FirstTest:          1,
						LastTest:           3,
						TestScore:          pointer.Float64(20),
						ScoringType:        models.TestGroupScoringTypeEachTest,
						RequiredGroupNames: make([]string, 0),
					},
				},
			}
			g, err := NewIOIGenerator(&problem, fixtureSubmission(1))
			require.NoError(t, err)
			job := nextJob(t, g, 1, invokerconn.CompileJob, 0)
			require.NotNil(t, job)
			require.Nil(t, g.NextJob())
			sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.CD,
			})
			require.NoError(t, err)
			require.Nil(t, sub)
			// test
			job = nextJob(t, g, 1, invokerconn.TestJob, 1)
			require.NotNil(t, job)
			sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.OK,
			})
			job2 := nextJob(t, g, 1, invokerconn.TestJob, 2)
			require.NotNil(t, job2)
			job3 := nextJob(t, g, 1, invokerconn.TestJob, 3)
			require.NotNil(t, job3)
			require.Nil(t, g.NextJob())
			sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job2,
				Verdict: verdict.WA,
			})
			require.NoError(t, err)
			require.Nil(t, sub)
			sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job3,
				Verdict: verdict.OK,
			})
			require.NoError(t, err)
			require.NotNil(t, sub)

			require.Equal(t, verdict.PT, sub.Verdict)
			require.Equal(t, 40., sub.Score)
			require.Equal(t, verdict.OK, sub.TestResults[0].Verdict)
			require.Equal(t, verdict.WA, sub.TestResults[1].Verdict)
			require.Equal(t, verdict.OK, sub.TestResults[2].Verdict)
			require.Equal(t, sub.GroupResults, models.GroupResults{
				{
					GroupName: "group1",
					Points:    40.,
					Passed:    false,
				},
			})
		})
	})

	t.Run("TestGroupScoringTypeMin", func(t *testing.T) {
		prepare := func() models.Problem {
			return models.Problem{
				ProblemType: models.ProblemTypeIOI,
				TestsNumber: 3,
				TestGroups: []models.TestGroup{
					{
						Name:               "group1",
						FirstTest:          1,
						LastTest:           3,
						GroupScore:         pointer.Float64(20),
						ScoringType:        models.TestGroupScoringTypeMin,
						RequiredGroupNames: make([]string, 0),
					},
				},
			}
		}

		t.Run("WA in the middle of the group", func(t *testing.T) {
			problem := prepare()
			g, err := NewIOIGenerator(&problem, fixtureSubmission(1))
			require.NoError(t, err)
			job := nextJob(t, g, 1, invokerconn.CompileJob, 0)
			require.NotNil(t, job)
			require.Nil(t, g.NextJob())
			sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.CD,
			})
			require.NoError(t, err)
			require.Nil(t, sub)
			// test
			job = nextJob(t, g, 1, invokerconn.TestJob, 1)
			require.NotNil(t, job)
			sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.OK,
			})
			job2 := nextJob(t, g, 1, invokerconn.TestJob, 2)
			require.NotNil(t, job2)
			job3 := nextJob(t, g, 1, invokerconn.TestJob, 3)
			require.NotNil(t, job3)
			require.Nil(t, g.NextJob())
			sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job2,
				Verdict: verdict.WA,
			})
			require.NoError(t, err)
			require.Nil(t, sub)
			sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job3,
				Verdict: verdict.OK,
			})
			require.NoError(t, err)
			require.NotNil(t, sub)

			require.Equal(t, verdict.PT, sub.Verdict)
			require.Equal(t, 0., sub.Score)
			require.Equal(t, verdict.OK, sub.TestResults[0].Verdict)
			require.Equal(t, verdict.WA, sub.TestResults[1].Verdict)
			require.Equal(t, verdict.SK, sub.TestResults[2].Verdict)
			require.Equal(t, sub.GroupResults, models.GroupResults{
				{
					GroupName: "group1",
					Points:    0.,
					Passed:    false,
				},
			})
		})

		t.Run("PT in the middle of the group", func(t *testing.T) {
			problem := prepare()
			g, err := NewIOIGenerator(&problem, fixtureSubmission(1))
			require.NoError(t, err)
			job := nextJob(t, g, 1, invokerconn.CompileJob, 0)
			require.NotNil(t, job)
			require.Nil(t, g.NextJob())
			sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.CD,
			})
			require.NoError(t, err)
			require.Nil(t, sub)
			// test
			job = nextJob(t, g, 1, invokerconn.TestJob, 1)
			require.NotNil(t, job)
			sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.OK,
			})
			job2 := nextJob(t, g, 1, invokerconn.TestJob, 2)
			require.NotNil(t, job2)
			job3 := nextJob(t, g, 1, invokerconn.TestJob, 3)
			require.NotNil(t, job3)
			require.Nil(t, g.NextJob())
			sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job2,
				Verdict: verdict.PT,
				Points:  pointer.Float64(10),
			})
			require.NoError(t, err)
			require.Nil(t, sub)
			sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job3,
				Verdict: verdict.OK,
			})
			require.NoError(t, err)
			require.NotNil(t, sub)

			require.Equal(t, verdict.PT, sub.Verdict)
			require.Equal(t, 10., sub.Score)
			require.Equal(t, verdict.OK, sub.TestResults[0].Verdict)
			require.Equal(t, verdict.PT, sub.TestResults[1].Verdict)
			require.Equal(t, verdict.OK, sub.TestResults[2].Verdict)
			require.Equal(t, sub.GroupResults, models.GroupResults{
				{
					GroupName: "group1",
					Points:    10.,
					Passed:    false,
				},
			})
		})

		t.Run("no fails", func(t *testing.T) {
			problem := prepare()
			g, err := NewIOIGenerator(&problem, fixtureSubmission(1))
			require.NoError(t, err)
			job := nextJob(t, g, 1, invokerconn.CompileJob, 0)
			require.NotNil(t, job)
			require.Nil(t, g.NextJob())
			sub, err := g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.CD,
			})
			require.NoError(t, err)
			require.Nil(t, sub)
			// test
			job = nextJob(t, g, 1, invokerconn.TestJob, 1)
			require.NotNil(t, job)
			sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job,
				Verdict: verdict.OK,
			})
			job2 := nextJob(t, g, 1, invokerconn.TestJob, 2)
			require.NotNil(t, job2)
			job3 := nextJob(t, g, 1, invokerconn.TestJob, 3)
			require.NotNil(t, job3)
			require.Nil(t, g.NextJob())
			sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job2,
				Verdict: verdict.OK,
			})
			require.NoError(t, err)
			require.Nil(t, sub)
			sub, err = g.JobCompleted(&masterconn.InvokerJobResult{
				Job:     job3,
				Verdict: verdict.OK,
			})
			require.NoError(t, err)
			require.NotNil(t, sub)

			require.Equal(t, verdict.OK, sub.Verdict)
			require.Equal(t, 20., sub.Score)
			require.Equal(t, verdict.OK, sub.TestResults[0].Verdict)
			require.Equal(t, verdict.OK, sub.TestResults[1].Verdict)
			require.Equal(t, verdict.OK, sub.TestResults[2].Verdict)
			require.Equal(t, sub.GroupResults, models.GroupResults{
				{
					GroupName: "group1",
					Points:    20.,
					Passed:    true,
				},
			})
		})
	})
}
