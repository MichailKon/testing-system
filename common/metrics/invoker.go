package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"testing_system/common/connectors/invokerconn"
	"testing_system/common/connectors/masterconn"
)

func (c *Collector) setupInvokerMetrics() {
	c.InvokerJobResults = c.createInvokerCounter(
		"job_results_count",
		"Number of job results received from invoker",
	)

	c.InvokerTestingWaitDuration = c.createInvokerCounter(
		"testing_wait_duration_sum",
		"Time submission waits for testing in invoker",
	)

	c.InvokerSandboxOccupationDuration = c.createInvokerCounter(
		"sandbox_occupation_duration_sum",
		"Total sandbox time for submission testing in invoker",
	)

	c.InvokerResourceWaitDuration = c.createInvokerCounter(
		"resource_wait_duration_sum",
		"Total time spent waiting for resources for submissions to load in invokers",
	)

	c.InvokerFileActionsDuration = c.createInvokerCounter(
		"file_actions_duration_sum",
		"Total time spent waiting for file copy to sandbox in invoker",
	)

	c.InvokerExecutionWaitDuration = c.createInvokerCounter(
		"execution_wait_duration_sum",
		"Total time spent waiting for execution of process on invoker when sandbox is set up",
	)

	c.InvokerExecutionDuration = c.createInvokerCounter(
		"execution_duration_sum",
		"Total time spent on executing processes in sandboxes",
	)

	c.InvokerSendResultDuration = c.createInvokerCounter(
		"send_result_duration_sum",
		"Total time spent on sending results from invoker to storage",
	)

	c.InvokerLifetimeDuration = c.createInvokerGauge(
		"lifetime_duration",
		"Total time the invoker is running",
		[]string{invokerLabel},
	)

	c.InvokerSandboxCount = c.createInvokerGauge(
		"sandbox_count",
		"Number of sandboxes in invoker",
		[]string{invokerLabel},
	)

	c.InvokerSandboxWaitDuration = c.createInvokerGauge(
		"sandbox_wait_duration",
		"The duration while the sandbox is free (not processing jobs)",
		[]string{invokerLabel, indexLabel},
	)

	c.InvokerThreadCount = c.createInvokerGauge(
		"threads_count",
		"Number of threads that are executing processes (parallelization)",
		[]string{invokerLabel},
	)

	c.InvokerThreadWaitDuration = c.createInvokerGauge(
		"threads_wait_duration",
		"The duration while threads are free (not executing any processes)",
		[]string{invokerLabel, indexLabel},
	)
}

func (c *Collector) createInvokerCounter(
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
	c.Registerer.MustRegister(counter)
	return counter
}

func (c *Collector) createInvokerGauge(
	name string,
	help string,
	labelNames []string,
) *prometheus.GaugeVec {
	gauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ts",
			Subsystem: "invoker",
			Name:      name,
			Help:      help,
		},
		labelNames,
	)
	c.Registerer.MustRegister(gauge)
	return gauge
}

func (c *Collector) ProcessJobResult(result *masterconn.InvokerJobResult) {
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

func (c *Collector) ProcessInvokerStatus(status *invokerconn.Status) {
	labels := prometheus.Labels{
		invokerLabel: status.Address,
	}

	c.InvokerLifetimeDuration.With(labels).Set(status.Metrics.Lifetime.Seconds())
	c.InvokerThreadCount.With(labels).Set(float64(status.Metrics.ThreadMetrics.Count))
	c.InvokerSandboxCount.With(labels).Set(float64(status.Metrics.SandboxMetrics.Count))

	for i, duration := range status.Metrics.ThreadMetrics.TotalWaitDuration {
		c.InvokerThreadWaitDuration.With(prometheus.Labels{
			invokerLabel: status.Address,
			indexLabel:   strconv.Itoa(i + 1),
		}).Set(duration.Seconds())
	}

	for i, duration := range status.Metrics.SandboxMetrics.TotalWaitDuration {
		c.InvokerSandboxWaitDuration.With(prometheus.Labels{
			invokerLabel: status.Address,
			indexLabel:   strconv.Itoa(i + 1),
		}).Set(duration.Seconds())
	}
}
