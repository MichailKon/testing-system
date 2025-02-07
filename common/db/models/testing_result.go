package models

import (
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Verdict int

const (
	Verdict_OK Verdict = iota + 1
	Verdict_WA
	Verdict_TL
	Verdict_ML
	Verdict_RE
	Verdict_UNK
)

type TestingResult struct {
	gorm.Model
	Verdicts pq.Int64Array `gorm:"type:int[]"`
}
