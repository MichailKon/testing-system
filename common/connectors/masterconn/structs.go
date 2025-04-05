package masterconn

import (
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/constants/verdict"
)

type InvokerJobResult struct {
	JobID   string          `json:"JobID"`
	Verdict verdict.Verdict `json:"Verdict"`
	Error   string          `json:"Error,omitempty"` // Is set only in case of check failed

	InvokerStatus *invokerconn.StatusResponse `json:"InvokerStatus"`

	// TODO: Add some statistics (e.g time, memory, etc)
}
