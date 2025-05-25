package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	invokerLabel = "invoker"
	jobTypeLabel = "job_type"
	indexLabel   = "index"
)

type Collector struct {
	Registerer *prometheus.Registry

	InvokerJobResults                *prometheus.CounterVec
	InvokerTestingWaitDuration       *prometheus.CounterVec
	InvokerSandboxOccupationDuration *prometheus.CounterVec
	InvokerResourceWaitDuration      *prometheus.CounterVec
	InvokerFileActionsDuration       *prometheus.CounterVec
	InvokerExecutionWaitDuration     *prometheus.CounterVec
	InvokerExecutionDuration         *prometheus.CounterVec
	InvokerSendResultDuration        *prometheus.CounterVec
	InvokerSkippedJobs               *prometheus.CounterVec

	InvokerLifetimeDuration    *prometheus.GaugeVec
	InvokerSandboxCount        *prometheus.GaugeVec
	InvokerThreadCount         *prometheus.GaugeVec
	InvokerSandboxWaitDuration *prometheus.GaugeVec
	InvokerThreadWaitDuration  *prometheus.GaugeVec

	MasterQueueSize      prometheus.Gauge
	MasterInvokerFails   prometheus.Counter
	MasterJobReschedules prometheus.Counter
}

func NewCollector() *Collector {
	c := &Collector{
		Registerer: prometheus.NewRegistry(),
	}

	c.setupInvokerMetrics()
	c.setupMasterMetrics()

	return c
}
