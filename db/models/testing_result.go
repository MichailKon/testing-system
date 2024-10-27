package models

import "github.com/lib/pq"

type Verdict int

const (
	Verdict_OK int = iota
	Verdict_WA
	Verdict_TL
	Verdict_ML
	Verdict_RE
	Verdict_UNK
)

type TestingResult struct {
	Verdicts pq.Int64Array `gorm:"type:int[]"`
}
