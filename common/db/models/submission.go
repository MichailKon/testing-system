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
	TestNumber uint64              `json:"testNumber" yaml:"testNumber"`
	Points     *float64            `json:"points,omitempty" yaml:"points,omitempty"`
	Verdict    verdict.Verdict     `json:"verdict" yaml:"verdict"`
	Time       customfields.Time   `json:"time" yaml:"time"`
	Memory     customfields.Memory `json:"memory" yaml:"memory"`
}

type TestResults []TestResult

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

type Submission struct {
	gorm.Model
	ProblemID uint64
	Language  string

	Score       float64
	Verdict     verdict.Verdict
	TestResults TestResults `gorm:"type:jsonb"`
}
