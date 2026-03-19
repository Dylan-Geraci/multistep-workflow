# Changelog

## Unreleased

### Added

#### Week 1 — Foundation + Core Execution
- Project PLAN.md with architecture, data model, API surface, observability plan, 4-week roadmap
- Repo scaffold with Go API (Chi router), Docker Compose (PostgreSQL 16 + Redis 7)
- Database migrations for all 6 tables (users, refresh_tokens, workflows, workflow_steps, workflow_runs, step_executions)
- Health endpoint with PostgreSQL and Redis connectivity checks
- Workflow CRUD endpoints (create, list, get, update, delete) with step management
- Multi-stage Dockerfile for Go API
- Run creation, listing, detail view, and cancellation endpoints
- Redis Streams worker pool with configurable worker count
- All 4 built-in step actions: `http_call`, `delay`, `log`, `transform`
- Exponential backoff retry with configurable policy per workflow
- Crash recovery via XPENDING/XCLAIM for orphaned messages
- Idempotent step execution via unique attempt IDs

#### Week 2 — Auth + Real-Time
- JWT authentication with HS256 access tokens (15-min TTL)
- Opaque refresh tokens with SHA-256 storage, rotation on use, 30-day TTL
- User registration, login, refresh, logout, and `/auth/me` endpoints
- Protected API routes via Bearer token middleware
- WebSocket endpoint (`/api/v1/ws`) with subscription-based event streaming
- Redis pub/sub bridge for real-time run/step events
- Event types: run.status_changed, step.started, step.completed, step.failed, run.completed, run.failed
- Handler tests (workflow CRUD, run lifecycle, cancellation)
- Worker tests (all 4 actions, context cancellation, retry backoff)
- WebSocket hub tests (client lifecycle, message routing)

#### Week 3 — Observability
- Structured JSON logging via `log/slog` with context-aware handler
- Request ID middleware (ULID-based) with `X-Request-ID` response header
- Automatic `request_id` and `trace_id` injection into all log records
- Prometheus metrics endpoint (`/metrics`) with 8 custom metrics:
  - `flowforge_http_requests_total` (method, path, status)
  - `flowforge_http_request_duration_seconds` (histogram)
  - `flowforge_workflow_runs_total` (status)
  - `flowforge_step_executions_total` (action, status)
  - `flowforge_step_execution_duration_ms` (histogram)
  - `flowforge_redis_queue_depth` (gauge)
  - `flowforge_redis_pending_claims` (gauge)
  - `flowforge_ws_connections_active` (gauge)
- Background Redis metrics collector (queue depth + pending claims, 15s interval)
- HTTP metrics middleware using Chi route patterns (bounded cardinality)
- OpenTelemetry tracing with OTLP gRPC exporter
- Trace spans: `workflow.run` → `step.execute` → `http.client`
- Trace context propagation through Redis Streams via `traceparent` field
- Traced HTTP client for `http_call` action via `otelhttp.NewTransport`
- Docker Compose additions: Prometheus, Grafana, OTel Collector
- Prometheus scrape config targeting API `/metrics`
- OTel Collector pipeline (OTLP gRPC → batch → debug/prometheus exporters)
- 3 pre-provisioned Grafana dashboards:
  - API Health (request rate, latency percentiles, error rate, WS connections)
  - Execution Engine (runs by status, step executions, duration percentiles, failure rate)
  - Infrastructure (Redis queue depth, pending claims, goroutines, memory, CPU)
- Grafana auto-provisioning with Prometheus datasource
