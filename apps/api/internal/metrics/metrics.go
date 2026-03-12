package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// HTTP metrics
	HTTPRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "flowforge_http_requests_total",
		Help: "Total HTTP requests by method, path, and status.",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "flowforge_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds by method and path.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})

	// Run metrics
	RunsStartedTotal   = prometheus.NewCounter(prometheus.CounterOpts{Name: "flowforge_runs_started_total", Help: "Total runs started."})
	RunsCompletedTotal = prometheus.NewCounter(prometheus.CounterOpts{Name: "flowforge_runs_completed_total", Help: "Total runs completed successfully."})
	RunsFailedTotal    = prometheus.NewCounter(prometheus.CounterOpts{Name: "flowforge_runs_failed_total", Help: "Total runs failed."})

	// Step metrics
	StepExecutionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "flowforge_step_executions_total",
		Help: "Total step executions by action and status.",
	}, []string{"action", "status"})

	StepExecutionDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "flowforge_step_execution_duration_seconds",
		Help:    "Step execution duration in seconds by action.",
		Buckets: prometheus.DefBuckets,
	}, []string{"action"})

	// Worker/queue metrics
	WorkerActive = prometheus.NewGauge(prometheus.GaugeOpts{Name: "flowforge_worker_active", Help: "Number of active workers."})
	QueueDepth   = prometheus.NewGauge(prometheus.GaugeOpts{Name: "flowforge_queue_depth", Help: "Number of messages in the step queue."})

	// Recovery
	RecoveryClaimsTotal = prometheus.NewCounter(prometheus.CounterOpts{Name: "flowforge_recovery_claims_total", Help: "Total orphaned messages recovered."})

	// Health gauges
	PGUp    = prometheus.NewGauge(prometheus.GaugeOpts{Name: "flowforge_pg_up", Help: "PostgreSQL is reachable (1) or not (0)."})
	RedisUp = prometheus.NewGauge(prometheus.GaugeOpts{Name: "flowforge_redis_up", Help: "Redis is reachable (1) or not (0)."})
)

// Init registers all metrics with the default Prometheus registry.
func Init() {
	prometheus.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		RunsStartedTotal,
		RunsCompletedTotal,
		RunsFailedTotal,
		StepExecutionsTotal,
		StepExecutionDuration,
		WorkerActive,
		QueueDepth,
		RecoveryClaimsTotal,
		PGUp,
		RedisUp,
	)
}
