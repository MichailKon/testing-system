package masterconn

import (
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/constants/verdict"
	"testing_system/lib/customfields"
)

type InvokerJobResult struct {
	JobID string `json:"JobID"`

	Verdict verdict.Verdict `json:"Verdict"`
	Points  *float64        `json:"Points,omitempty"`

	Error string `json:"Error,omitempty"` // Is set only in case of check failed caused by invoker problems

	Statistics *JobResultStatistics `json:"Statistics"`

	InvokerStatus *invokerconn.StatusResponse `json:"InvokerStatus"`
}

type JobResultStatistics struct {
	Time     customfields.TimeLimit   `json:"Time"`
	Memory   customfields.MemoryLimit `json:"Memory"`
	WallTime customfields.TimeLimit   `json:"WallTime"`

	ExitCode int `json:"ExitCode"`
	// TODO: Add more statistics
}
