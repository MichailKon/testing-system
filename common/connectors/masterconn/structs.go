package masterconn

import (
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/constants/verdict"
	"testing_system/common/db/models"
	"testing_system/lib/customfields"
	"time"
)

type InvokerJobResult struct {
	Job *invokerconn.Job `json:"job"`

	Verdict verdict.Verdict `json:"verdict"`
	Points  *float64        `json:"points,omitempty"`

	Error string `json:"error,omitempty"` // Is set only in case of check failed caused by invoker problems

	Statistics *JobResultStatistics `json:"statistics,omitempty"`

	InvokerStatus *invokerconn.Status `json:"invoker_status"`

	Metrics *InvokerJobMetrics `json:"metrics"`
}

type JobResultStatistics struct {
	Time     customfields.Time   `json:"time"`
	Memory   customfields.Memory `json:"memory"`
	WallTime customfields.Time   `json:"wall_time"`

	ExitCode int `json:"ExitCode"`
	// TODO: Add more statistics
}

type SubmissionResponse struct {
	SubmissionID uint `json:"submission_id"`
}

type Status struct {
	Epoch              string               `json:"epoch"`
	TestingSubmissions []uint               `json:"testing_submissions"`
	UpdatedSubmissions []*models.Submission `json:"updated_submissions"`

	Invokers []*InvokerStatus `json:"invokers"`
}

type InvokerStatus struct {
	Address     string             `json:"address"`
	TimeAdded   time.Time          `json:"time_added"`
	MaxNewJobs  int                `json:"max_new_jobs"`
	TestingJobs []*invokerconn.Job `json:"active_jobs"`
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
