package models

type ProblemType int

const (
	ProblemType_ICPC ProblemType = iota
	ProblemType_IOI
)

type ProblemConfig struct {
	ProblemType ProblemType
	// everything we need here
}
