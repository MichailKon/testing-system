package tests

import (
	"context"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"strconv"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/lib/logger"
	"time"
)

type submitTest struct {
	innerID uint   `yaml:"-"`
	dir     string `yaml:"-"`
	ID      uint   `yaml:"-"`

	ProblemID  uint   `yaml:"problem_id"`
	Language   string `yaml:"language"`
	SourceFile string `yaml:"source_file"`

	result         *models.Submission `yaml:"-"`
	RequiredResult *models.Submission `yaml:"required_result"`
}

func (h *TSHolder) newSubmit(id uint) {
	s := &submitTest{
		innerID: id,
		dir:     filepath.Join(h.submitsDir, strconv.FormatUint(uint64(id), 10)),
	}

	sData, err := os.ReadFile(filepath.Join(s.dir, "cfg.yaml"))
	require.NoError(h.t, err)
	require.NoError(h.t, yaml.Unmarshal(sData, s))

	var ok bool
	for _ = range 5 {
		ok = h.sendSubmit(s)
		if ok {
			break
		}
	}
	// We retry sending submits because in memory sqlite sucks and may lock db for each request
	require.Equal(h.t, true, ok)

	h.submits = append(h.submits, s)
}

func (h *TSHolder) sendSubmit(s *submitTest) bool {
	sourceReader, err := os.Open(filepath.Join(s.dir, s.SourceFile))
	require.NoError(h.t, err)
	defer sourceReader.Close()
	s.ID, err = h.ts.MasterConn.SendNewSubmission(
		context.Background(),
		s.ProblemID,
		s.Language,
		s.SourceFile,
		sourceReader,
	)
	require.NoError(h.t, err)
	return true
}

func (h *TSHolder) waitSubmits() {
	for {
		status, err := h.ts.MasterConn.GetStatus(context.Background(), "")
		require.NoError(h.t, err)
		if len(status.TestingSubmissions) == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}
	for _, s := range h.submits {
		h.verifySubmit(s)
	}
	h.submits = nil
}

func (h *TSHolder) verifySubmit(s *submitTest) {
	h.waitTesting(s)

	require.Equal(h.t, s.RequiredResult.Verdict, s.result.Verdict)
	if s.RequiredResult.Score != -1 {
		require.Equal(h.t, s.RequiredResult.Score, s.result.Score)
	}

	h.verifyTestResults(s)

	logger.Trace("Verified submit %d", s.ID)
}

func (h *TSHolder) verifyTestResults(s *submitTest) {
	var problem models.Problem
	require.NoError(h.t, h.ts.DB.Find(&problem, s.ProblemID).Error)

	for testID := uint64(1); testID <= problem.TestsNumber; testID++ {
		count := 0
		for _, test := range s.result.TestResults {
			if test.TestNumber == testID {
				count++
			}
		}

		require.Equal(h.t, 1, count)
	}

	for _, requiredTest := range s.RequiredResult.TestResults {
		for _, test := range s.result.TestResults {
			if test.TestNumber == requiredTest.TestNumber {
				require.Equal(h.t, requiredTest.Verdict, test.Verdict)
				if requiredTest.Points != nil {
					require.NotNil(h.t, test.Points)
					require.Equal(h.t, *requiredTest.Points, *test.Points)
				}
			}
		}
	}
}

func (h *TSHolder) waitTesting(s *submitTest) {
	for {
		submission := new(models.Submission)

		// We retry sending submits because in memory sqlite sucks and may lock db for each request
		var err error
		for _ = range 1000 {
			if err = h.ts.DB.Find(submission, s.ID).Error; err == nil {
				break
			}
		}
		require.NoError(h.t, h.ts.DB.Find(submission, s.ID).Error)
		if submission.Verdict != verdict.RU {
			s.result = submission
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}
