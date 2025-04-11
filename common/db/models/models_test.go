package models

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
	"testing_system/common/constants/verdict"
	"testing_system/lib/customfields"
	"time"
)

func fixtureDb(t *testing.T) *gorm.DB {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	err := db.AutoMigrate(&Submission{})
	assert.NoError(t, err)
	return db
}

func TestTestResultSerialization(t *testing.T) {
	var time customfields.Time
	require.Nil(t, time.FromStr("5s"))
	var memory customfields.Memory
	require.Nil(t, memory.FromStr("5m"))

	testResult := TestResult{
		TestNumber: 1,
		Points:     nil,
		Verdict:    verdict.OK,
		Time:       time,
		Memory:     memory,
	}

	t.Run("json", func(t *testing.T) {
		b, err := json.Marshal(testResult)
		require.Nil(t, err)
		require.Equal(t, `{"testNumber":1,"verdict":"OK","time":"5s","memory":"5m"}`, string(b))

		var newTestResult TestResult
		err = json.Unmarshal(b, &newTestResult)
		require.Nil(t, err)
		require.Equal(t, testResult, newTestResult)
	})

	t.Run("yaml", func(t *testing.T) {
		b, err := yaml.Marshal(testResult)
		require.Nil(t, err)
		require.Equal(t, `testNumber: 1
verdict: OK
time: 5s
memory: 5m
`, string(b))
		var newTestResult TestResult
		err = yaml.Unmarshal(b, &newTestResult)
		require.Nil(t, err)
		require.Equal(t, testResult, newTestResult)
	})
}

func TestTestResultsDB(t *testing.T) {
	t.Run("sqlite", func(t *testing.T) {
		db := fixtureDb(t)
		submission := Submission{
			ProblemID: 1,
			Language:  "cpp",
			Score:     1,
			Verdict:   verdict.TL,
			TestResults: []TestResult{
				{
					TestNumber: 1,
					Points:     nil,
					Verdict:    verdict.OK,
					Time:       customfields.Time(1 * time.Second),
					Memory:     customfields.Memory(10 * 1024 * 1024),
				},
				{
					TestNumber: 0,
					Points:     nil,
					Verdict:    "",
					Time:       customfields.Time(10 * time.Second),
					Memory:     customfields.Memory(10 * 1024 * 1024),
				},
			},
		}
		tx := db.Create(&submission)
		require.Nil(t, tx.Error)
		var newSubmission Submission
		require.Nil(t, tx.First(&newSubmission).Error)
		require.Equal(t, submission.ProblemID, newSubmission.ProblemID)
		require.Equal(t, submission.Language, newSubmission.Language)
		require.Equal(t, submission.Score, newSubmission.Score)
		require.Equal(t, submission.Verdict, newSubmission.Verdict)
		require.Equal(t, submission.TestResults, newSubmission.TestResults)
	})
}
