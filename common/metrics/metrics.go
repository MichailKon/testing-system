package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"testing_system/common/connectors/masterconn"
)

const (
	invokerLabel = "invoker"
	jobTypeLabel = "job_type"
)

type Collector struct {
	InvokerJobResults                *prometheus.CounterVec
	InvokerTestingWaitDuration       *prometheus.CounterVec
	InvokerSandboxOccupationDuration *prometheus.CounterVec
	InvokerResourceWaitDuration      *prometheus.CounterVec
	InvokerFileActionsDuration       *prometheus.CounterVec
	InvokerExecutionWaitDuration     *prometheus.CounterVec
	InvokerExecutionDuration         *prometheus.CounterVec
	InvokerSendResultDuration        *prometheus.CounterVec
}

func NewCollector() *Collector {
	c := &Collector{}
	c.InvokerJobResults = createInvokerCounter(
		"job_results_count",
		"Number of job results received from invoker",
	)

	c.InvokerTestingWaitDuration = createInvokerCounter(
		"testing_wait_duration_sum",
		"Time submission waits for testing in invoker",
	)

	c.InvokerSandboxOccupationDuration = createInvokerCounter(
		"sandbox_occupation_duration_sun",
		"Total sandbox time for submission testing in invoker",
	)

	c.InvokerResourceWaitDuration = createInvokerCounter(
		"resource_wait_duration_sum",
		"Total time spent waiting for resources for submissions to load in invokers",
	)

	c.InvokerFileActionsDuration = createInvokerCounter(
		"file_actions_duration_sum",
		"Total time spent waiting for file copy to sandbox in invoker",
	)

	c.InvokerExecutionWaitDuration = createInvokerCounter(
		"execution_wait_duration_sum",
		"Total time spent waiting for execution of process on invoker when sandbox is set up",
	)

	c.InvokerExecutionDuration = createInvokerCounter(
		"execution_duration_sum",
		"Total time spent on executing processes in sandboxes",
	)

	c.InvokerSendResultDuration = createInvokerCounter(
		"send_result_duration_sum",
		"Total time spent on sending results from invoker to storage",
	)
	return c
}

func createInvokerCounter(
	name string,
	help string,
) *prometheus.CounterVec {
	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ts",
			Subsystem: "invoker",
			Name:      name,
			Help:      help,
		},
		[]string{invokerLabel, jobTypeLabel},
	)
	prometheus.MustRegister(counter)
	return counter
}

func (c *Collector) NewJobResult(result *masterconn.InvokerJobResult) {
	labels := prometheus.Labels{
		invokerLabel: result.InvokerStatus.Address,
		jobTypeLabel: result.Job.Type.String(),
	}

	c.InvokerJobResults.With(labels).Inc()
	c.InvokerTestingWaitDuration.With(labels).Add(result.Metrics.TestingWaitDuration.Seconds())
	c.InvokerSandboxOccupationDuration.With(labels).Add(result.Metrics.TotalSandboxOccupation.Seconds())
	c.InvokerResourceWaitDuration.With(labels).Add(result.Metrics.ResourceWaitDuration.Seconds())
	c.InvokerFileActionsDuration.With(labels).Add(result.Metrics.FileActionsDuration.Seconds())
	c.InvokerExecutionWaitDuration.With(labels).Add(result.Metrics.ExecutionWaitDuration.Seconds())
	c.InvokerExecutionDuration.With(labels).Add(result.Metrics.ExecutionDuration.Seconds())
	c.InvokerSendResultDuration.With(labels).Add(result.Metrics.SendResultDuration.Seconds())
}
