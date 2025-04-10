package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"testing_system/common/constants/verdict"
)

type TestResult struct {
	TestNumber     uint64          `json:"testNumber"`
	Verdict        verdict.Verdict `json:"verdict"`
	TimeConsumed   int64           `json:"timeConsumed"`
	MemoryConsumed int64           `json:"memoryConsumed"`
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
	return json.Unmarshal(bytes, &t)
}

type Submission struct {
	gorm.Model
	ProblemID uint64
	Language  string

	Score       float64
	Verdict     verdict.Verdict
	TestResults []TestResult `gorm:"type:jsonb"`
}
