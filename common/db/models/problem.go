package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"testing_system/lib/customfields"
)

type ProblemType int

// TestGroupScoringType sets how should scheduler set points for a group
type TestGroupScoringType int

// TestGroupFeedbackType sets which info about tests in a group would be shown
type TestGroupFeedbackType int

const (
	ProblemTypeICPC ProblemType = iota + 1
	ProblemTypeIOI
)
const (
	// TestGroupScoringTypeComplete means that group costs TestGroup.Score (all the tests should be OK)
	TestGroupScoringTypeComplete TestGroupScoringType = iota + 1
	// TestGroupScoringTypeEachTest means that group score = TestGroup.TestScore * (number of tests with OK)
	TestGroupScoringTypeEachTest
	// TestGroupScoringTypeMin means that group score = min(checker's scores among all the tests)
	TestGroupScoringTypeMin
)
const (
	// TestGroupFeedbackTypeNone won't show anything
	TestGroupFeedbackTypeNone TestGroupFeedbackType = iota + 1
	// TestGroupFeedbackTypePoints will show points only
	TestGroupFeedbackTypePoints
	// TestGroupFeedbackTypeICPC will show verdict, time and memory usage for the first test with no OK
	TestGroupFeedbackTypeICPC
	// TestGroupFeedbackTypeComplete same as TestGroupFeedbackTypeICPC, but for every test
	TestGroupFeedbackTypeComplete
	// TestGroupFeedbackTypeFull same as TestGroupFeedbackTypeComplete, but with input, output, stderr, etc.
	TestGroupFeedbackTypeFull
)

type TestGroup struct {
	Name      string `json:"name" yaml:"name"`
	FirstTest int    `json:"FirstTest" yaml:"FirstTest"`
	LastTest  int    `json:"LastTest" yaml:"LastTest"`
	// TestScore meaningful only in case of TestGroupScoringTypeEachTest
	TestScore *float64 `json:"TestScore" yaml:"TestScore"`
	// Score meaningful only in case of TestGroupScoringTypeComplete
	Score              *float64              `json:"Score" yaml:"Score"`
	ScoringType        TestGroupScoringType  `json:"ScoringType" yaml:"ScoringType"`
	FeedbackType       TestGroupFeedbackType `json:"FeedbackType" yaml:"FeedbackType"`
	RequiredGroupNames []string              `json:"RequiredGroupNames" yaml:"RequiredGroupNames"`
}

func (t TestGroup) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (t *TestGroup) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, t)
}

func (t TestGroup) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql", "sqlite":
		return "JSON"
	case "postgres":
		return "JSONB"
	}
	return ""
}

type Problem struct {
	gorm.Model

<<<<<<< HEAD
	ProblemType ProblemType `yaml:"ProblemType"`

	TimeLimit   customfields.Time   `yaml:"TimeLimit"`
	MemoryLimit customfields.Memory `yaml:"MemoryLimit"`

	TestsNumber uint64 `yaml:"TestsNumber"`
	// TestGroups ignored for ICPC problems
	TestGroups []TestGroup
=======
	ProblemType ProblemType
>>>>>>> f1bf2b0 (fixes and tests)

	// WallTimeLimit specifies maximum execution and wait time.
	// By default, it is max(5s, TimeLimit * 2)
	WallTimeLimit *customfields.Time `yaml:"WallTimeLimit,omitempty"`

<<<<<<< HEAD
	// MaxOpenFiles specifies the maximum number of files, opened by testing system.
	// By default, it is 64
	MaxOpenFiles *uint64 `yaml:"MaxOpenFiles,omitempty"`
=======
	TestsNumber uint64
	// TestGroups ignored for ICPC problems
	TestGroups []TestGroup
>>>>>>> f1bf2b0 (fixes and tests)

	// MaxThreads specifies the maximum number of threads and/or processes;
	// By default, it is a single thread
	// If MaxThreads equals to -1, any number of threads allowed
	MaxThreads *int64 `yaml:"MaxThreads,omitempty"`

	// MaxOutputSize specifies maximum output in EACH file.
	// By default, it is 1g
	MaxOutputSize *customfields.Memory `yaml:"MaxOutputSize,omitempty"`
}
