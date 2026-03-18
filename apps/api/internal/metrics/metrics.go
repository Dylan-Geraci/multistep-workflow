package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	HTTPRequestsTotal     *prometheus.CounterVec
	HTTPRequestDuration   *prometheus.HistogramVec
	WorkflowRunsTotal     *prometheus.CounterVec
	StepExecutionsTotal   *prometheus.CounterVec
	StepExecutionDuration *prometheus.HistogramVec
	RedisQueueDepth       prometheus.Gauge
	RedisPendingClaims    prometheus.Gauge
	WSConnectionsActive   prometheus.Gauge
}

func New() *Metrics {
	return &Metrics{
		HTTPRequestsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "flowforge_http_requests_total",
			Help: "Total number of HTTP requests",
		}, []string{"method", "path", "status"}),

		HTTPRequestDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "flowforge_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}, []string{"method", "path"}),

		WorkflowRunsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "flowforge_workflow_runs_total",
			Help: "Total number of workflow runs by final status",
		}, []string{"status"}),

		StepExecutionsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "flowforge_step_executions_total",
			Help: "Total number of step executions",
		}, []string{"action", "status"}),

		StepExecutionDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "flowforge_step_execution_duration_ms",
			Help:    "Step execution duration in milliseconds",
			Buckets: []float64{10, 50, 100, 250, 500, 1000, 5000, 10000, 30000},
		}, []string{"action"}),

		RedisQueueDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "flowforge_redis_queue_depth",
			Help: "Number of messages in the Redis steps stream",
		}),

		RedisPendingClaims: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "flowforge_redis_pending_claims",
			Help: "Number of pending messages in the consumer group",
		}),

		WSConnectionsActive: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "flowforge_ws_connections_active",
			Help: "Number of active WebSocket connections",
		}),
	}
}
