package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/dylangeraci/flowforge/internal/metrics"
	"github.com/go-chi/chi/v5"
)

// Metrics records HTTP request count and duration using route patterns to avoid cardinality explosion.
func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)

		routePattern := chi.RouteContext(r.Context()).RoutePattern()
		if routePattern == "" {
			routePattern = "unmatched"
		}

		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, routePattern, strconv.Itoa(sw.status)).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(r.Method, routePattern).Observe(time.Since(start).Seconds())
	})
}
