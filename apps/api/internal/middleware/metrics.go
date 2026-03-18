package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dylangeraci/flowforge/internal/metrics"
	"github.com/go-chi/chi/v5"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func HTTPMetrics(m *metrics.Metrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			// Use Chi route pattern to avoid cardinality explosion
			pattern := chi.RouteContext(r.Context()).RoutePattern()
			if pattern == "" {
				pattern = "unmatched"
			}

			duration := time.Since(start).Seconds()
			status := fmt.Sprintf("%d", sw.status)

			m.HTTPRequestsTotal.WithLabelValues(r.Method, pattern, status).Inc()
			m.HTTPRequestDuration.WithLabelValues(r.Method, pattern).Observe(duration)
		})
	}
}
