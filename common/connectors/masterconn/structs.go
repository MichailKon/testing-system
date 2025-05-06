package masterconn

import (
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/constants/verdict"
	"testing_system/lib/customfields"
	"time"
)

type InvokerJobResult struct {
	Job *invokerconn.Job `json:"Job"`

	Verdict verdict.Verdict `json:"Verdict"`
	Points  *float64        `json:"Points,omitempty"`

	Error string `json:"Error,omitempty"` // Is set only in case of check failed caused by invoker problems

	Statistics *JobResultStatistics `json:"Statistics,omitempty"`

	InvokerStatus *invokerconn.Status `json:"InvokerStatus"`

	Metrics *InvokerJobMetrics `json:"Metrics"`
}

type JobResultStatistics struct {
	Time     customfields.Time   `json:"Time"`
	Memory   customfields.Memory `json:"Memory"`
	WallTime customfields.Time   `json:"WallTime"`

	ExitCode int `json:"ExitCode"`
	// TODO: Add more statistics
}

type InvokerJobMetrics struct {
	TestingWaitDuration    time.Duration `json:"InvokerWaitDuration"`
	TotalSandboxOccupation time.Duration `json:"TotalSandboxOccupation"`
	ResourceWaitDuration   time.Duration `json:"ResourceWaitDuration"`
	FileActionsDuration    time.Duration `json:"FileActionsDuration"`
	ExecutionWaitDuration  time.Duration `json:"ExecutionWaitDuration"`
	ExecutionDuration      time.Duration `json:"ExecutionDuration"`
	SendResultDuration     time.Duration `json:"SendResultDuration"`
}
