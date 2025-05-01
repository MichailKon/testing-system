package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"testing_system/common/constants/verdict"
	"testing_system/lib/customfields"
)

type TestResult struct {
	TestNumber uint64              `json:"TestNumber" yaml:"TestNumber"`
	Points     *float64            `json:"Points,omitempty" yaml:"Points,omitempty"`
	Verdict    verdict.Verdict     `json:"Verdict" yaml:"Verdict"`
	Time       customfields.Time   `json:"Time" yaml:"Time"`
	Memory     customfields.Memory `json:"Memory" yaml:"Memory"`
	Error      string              `json:"Error,omitempty" yaml:"Error,omitempty"`
}

type TestResults []TestResult

func (t TestResults) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (t *TestResults) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed while scanning TestResults")
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
	GroupName string  `json:"GroupName" yaml:"GroupName"`
	Points    float64 `json:"Points" yaml:"Points"`
	Passed    bool    `json:"Passed" yaml:"Passed"`
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
	gorm.Model
	ProblemID uint64 `json:"ProblemID" yaml:"ProblemID"`
	Language  string `json:"Language" yaml:"Language"`

	Score       float64         `json:"Score" yaml:"Score"`
	Verdict     verdict.Verdict `json:"Verdict" yaml:"Verdict"`
	TestResults TestResults     `gorm:"type:jsonb" json:"TestResults" yaml:"TestResults"`
	GroupResults GroupResults
}
