//go:generate go run golang.org/x/tools/cmd/stringer@latest -type=JobType

package invokerconn

import "fmt"

type JobType int

const (
	CompileJob JobType = iota + 1
	TestJob
)

type Job struct {
	ID       string  `json:"ID"`
	SubmitID uint    `json:"SubmitID" binding:"required"`
	Type     JobType `json:"JobType" binding:"required"`
	Test     uint64  `json:"Test"`

	// TODO: Add job dependency
}

func (j Job) String() string {
	return fmt.Sprintf("ID: %s Submit: %d Type %v Test: %d", j.ID, j.SubmitID, j.Type, j.Test)
}

type Status struct {
	MaxNewJobs   uint64   `json:"MaxNewJobs"`
	ActiveJobIDs []string `json:"ActiveJobIDs"`
	Epoch        string   `json:"Epoch"`
	Address      string   `json:"Address"`
}
