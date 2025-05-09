package metrics

import "github.com/prometheus/client_golang/prometheus"

func (c *Collector) setupMasterMetrics() {
	c.MasterQueueSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "ts",
		Subsystem: "master",
		Name:      "queue_size",
		Help:      "Number of submissions currently testing",
	})
	c.Registerer.MustRegister(c.MasterQueueSize)

	c.MasterInvokerFails = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "ts",
		Subsystem: "master",
		Name:      "invoker_fails_count",
		Help:      "Number of invoker failures detected on master",
	})
	c.Registerer.MustRegister(c.MasterInvokerFails)

	c.MasterJobReschedules = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "ts",
		Subsystem: "master",
		Name:      "job_reschedule_count",
		Help:      "Number times the jobs are rescheduled (because of invoker failure)",
	})
	c.Registerer.MustRegister(c.MasterJobReschedules)
}
