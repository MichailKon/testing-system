package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"testing_system/common/constants/verdict"
	"testing_system/lib/customfields"
	"time"
)

type TestResult struct {
	TestNumber uint64               `json:"test_number" yaml:"test_number"`
	Verdict    verdict.Verdict      `json:"verdict" yaml:"verdict"`
	Points     *float64             `json:"points,omitempty" yaml:"points,omitempty"`
	Time       *customfields.Time   `json:"time,omitempty" yaml:"time,omitempty"`
	Memory     *customfields.Memory `json:"memory,omitempty" yaml:"memory,omitempty"`
	WallTime   *customfields.Time   `json:"wall_time,omitempty" yaml:"wall_time,omitempty"`
	Error      string               `json:"error,omitempty" yaml:"error,omitempty"`
	ExitCode   *int                 `json:"exit_code,omitempty" yaml:"exit_code,omitempty"`
}

func (t TestResult) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (t *TestResult) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, t)
}

func (t TestResult) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql", "sqlite":
		return "JSON"
	case "postgres":
		return "JSONB"
	}
	return ""
}

type TestResults []*TestResult

func (t TestResults) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (t *TestResults) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, t)
}

func (t TestResults) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql", "sqlite":
		return "JSON"
	case "postgres":
		return "JSONB"
	}
	return ""
}

type GroupResult struct {
	GroupName string  `json:"group_name" yaml:"group_name"`
	Points    float64 `json:"points" yaml:"points"`
	Passed    bool    `json:"passed" yaml:"passed"`
	// TODO maybe more fields
}

type GroupResults []GroupResult

func (t GroupResults) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (t *GroupResults) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed while scanning GroupResults")
	}
	return json.Unmarshal(bytes, t)
}

func (t GroupResults) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "mysql", "sqlite":
		return "JSON"
	case "postgres":
		return "JSONB"
	}
	return ""
}

type Submission struct {
	ID        uint           `gorm:"primarykey; index:problem_submission,priority:2,sort:desc" json:"id" yaml:"id"`
	CreatedAt time.Time      `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" yaml:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-" yaml:"-"`

	ProblemID uint    `gorm:"index:problem_submission,priority:1" json:"problem_id" yaml:"problem_id"`
	Problem   Problem `gorm:"constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;" json:"-" yaml:"-"`
	Language  string  `json:"language" yaml:"language"`

	Score             float64         `json:"score" yaml:"score"`
	Verdict           verdict.Verdict `json:"verdict" yaml:"verdict"`
	TestResults       TestResults     `json:"test_results" yaml:"test_results"`
	CompilationResult *TestResult     `json:"compilation_result" yaml:"compilation_result"`
	GroupResults      GroupResults    `json:"group_results,omitempty" yaml:"group_results,omitempty"`
}
