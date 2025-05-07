package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"testing_system/lib/customfields"
	"time"
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
	FirstTest uint64 `json:"first_test" yaml:"first_test"`
	LastTest  uint64 `json:"last_test" yaml:"last_test"`
	// TestScore meaningful only in case of TestGroupScoringTypeEachTest
	TestScore *float64 `json:"test_score" yaml:"test_score"`
	// GroupScore meaningful only in case of TestGroupScoringTypeComplete
	GroupScore         *float64              `json:"group_score" yaml:"group_score"`
	ScoringType        TestGroupScoringType  `json:"scoring_type" yaml:"scoring_type"`
	FeedbackType       TestGroupFeedbackType `json:"feedback_type" yaml:"feedback_type"`
	RequiredGroupNames []string              `json:"required_group_names" yaml:"required_group_names"`
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
	ID        uint           `gorm:"primarykey" json:"ID" yaml:"ID"`
	CreatedAt time.Time      `json:"created_at" yaml:"CreatedAt"`
	UpdatedAt time.Time      `json:"updated_at" yaml:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-" yaml:"-"`

	Name string `yaml:"name" json:"name" binding:"required"`

	ProblemType ProblemType `yaml:"problem_type" json:"problem_type" binding:"required"`

	// TestGroups ignored for ICPC problems
	TestGroups TestGroups `yaml:"test_groups" json:"test_groups"`

	TimeLimit   customfields.Time   `yaml:"time_limit" json:"time_limit" binding:"required"`
	MemoryLimit customfields.Memory `yaml:"memory_limit" json:"memory_limit" binding:"required"`

	TestsNumber uint64 `yaml:"tests_number" json:"tests_number" binding:"required"`

	// WallTimeLimit specifies maximum execution and wait time.
	// By default, it is max(5s, TimeLimit * 2)
	WallTimeLimit *customfields.Time `yaml:"wall_time_limit,omitempty" json:"wall_time_limit,omitempty"`

	// MaxOpenFiles specifies maximum number of files, opened by testing system.
	// By default, it is 64
	MaxOpenFiles *uint64 `yaml:"max_open_files,omitempty" json:"max_open_files,omitempty"`

	// MaxThreads specifies maximum number of threads and/or processes
	// By default, it is single thread
	// If MaxThreads equals to -1, any number of threads allowed
	MaxThreads *int64 `yaml:"max_threads,omitempty" json:"max_threads,omitempty"`

	// MaxOutputSize specifies maximum output in EACH file.
	// By default, it is 1g
	MaxOutputSize *customfields.Memory `yaml:"max_output_size,omitempty" json:"max_output_size,omitempty"`
}
