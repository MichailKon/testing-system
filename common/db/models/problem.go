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

const (
	ProblemTypeICPC ProblemType = iota + 1
	ProblemTypeIOI
)

// TestGroupScoringType sets how should scheduler set points for a group
type TestGroupScoringType int

const (
	// TestGroupScoringTypeComplete means that group costs TestGroup.GroupScore (all the tests should be OK)
	TestGroupScoringTypeComplete TestGroupScoringType = iota + 1
	// TestGroupScoringTypeEachTest means that group score = TestGroup.TestScore * (number of tests with OK)
	TestGroupScoringTypeEachTest
	// TestGroupScoringTypeMin means that group score = min(checker's scores among all the tests)
	TestGroupScoringTypeMin
)

// TestGroupFeedbackType sets which info about tests in a group would be shown
type TestGroupFeedbackType int

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
	FirstTest uint64 `json:"FirstTest" yaml:"FirstTest"`
	LastTest  uint64 `json:"LastTest" yaml:"LastTest"`
	// TestScore meaningful only in case of TestGroupScoringTypeEachTest
	TestScore *float64 `json:"TestScore" yaml:"TestScore"`
	// GroupScore meaningful only in case of TestGroupScoringTypeComplete
	GroupScore         *float64              `json:"GroupScore" yaml:"GroupScore"`
	ScoringType        TestGroupScoringType  `json:"ScoringType" yaml:"ScoringType"`
	FeedbackType       TestGroupFeedbackType `json:"FeedbackType" yaml:"FeedbackType"`
	RequiredGroupNames []string              `json:"RequiredGroupNames" yaml:"RequiredGroupNames"`
}

type TestGroups []TestGroup

func (t TestGroups) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (t *TestGroups) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed while scanning TestGroups")
	}
	return json.Unmarshal(bytes, t)
}

func (t TestGroups) GormDBDataType(db *gorm.DB, field *schema.Field) string {
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

	ProblemType ProblemType `yaml:"ProblemType"`

	TimeLimit   customfields.Time   `yaml:"TimeLimit"`
	MemoryLimit customfields.Memory `yaml:"MemoryLimit"`

	TestsNumber uint64 `yaml:"TestsNumber"`
	// TestGroups ignored for ICPC problems
	TestGroups TestGroups `yaml:"TestGroups"`

	// WallTimeLimit specifies maximum execution and wait time.
	// By default, it is max(5s, TimeLimit * 2)
	WallTimeLimit *customfields.Time `yaml:"WallTimeLimit,omitempty"`

	// MaxOpenFiles specifies the maximum number of files, opened by testing system.
	// By default, it is 64
	MaxOpenFiles *uint64 `yaml:"MaxOpenFiles,omitempty"`

	// MaxThreads specifies the maximum number of threads and/or processes;
	// By default, it is a single thread
	// If MaxThreads equals to -1, any number of threads allowed
	MaxThreads *int64 `yaml:"MaxThreads,omitempty"`

	// MaxOutputSize specifies maximum output in EACH file.
	// By default, it is 1g
	MaxOutputSize *customfields.Memory `yaml:"MaxOutputSize,omitempty"`
}
