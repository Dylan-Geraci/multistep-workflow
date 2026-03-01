# FlowForge — Project Plan

## Problem Statement

There's no lightweight, self-hosted workflow engine that combines durable execution, real-time visibility, and clean API design. FlowForge fills this gap as a production-quality backend system.

## Goals

- Durable multi-step workflow execution with crash recovery
- Built-in step actions (http_call, delay, log, transform) — no arbitrary code
- Real-time execution streaming via WebSockets
- JWT auth (access + refresh token rotation)
- Full observability stack (metrics, traces, dashboards)
- Load tested with documented performance targets

## MVP Features

- Workflow CRUD (linear step chains)
- Run triggering + background execution via Redis Streams workers
- 4 built-in actions: `http_call`, `delay`, `log`, `transform`
- Exponential backoff retries with max attempts
- Crash recovery via XPENDING/XCLAIM
- JWT auth with refresh token rotation
- WebSocket real-time run updates
- SvelteKit dashboard (workflow list, run viewer, live step timeline)
- Prometheus metrics + Grafana dashboard
- OpenTelemetry tracing

## Stretch Goals

- DAG execution (parallel branches)
- Workflow versioning (copy-on-edit, runs snapshot)
- Run history CSV export
- Rate limiting (token bucket, per-user)

## Architecture

```
                    ┌──────────────────────────────────┐
                    │         Client Browser           │
                    │  SvelteKit SSR+SPA  │  WebSocket │
                    └────────┬────────────┬────────────┘
                             │ HTTPS      │ WSS
                    ┌────────┴────────────┴────────────┐
                    │       Go API (Chi router)         │
                    │  REST Handlers  │  WS Hub         │
                    │       Service Layer               │
                    │  sqlc queries │ Redis Streams      │
                    └───────┬──────────────┬────────────┘
                            │              │
                    ┌───────┴───┐   ┌──────┴──────┐
                    │ PostgreSQL │   │    Redis    │
                    │  (state of │   │  (queue +   │
                    │   record)  │   │   pub/sub)  │
                    └───────────┘   └─────────────┘

  Observability: Go API → OTel Collector → Prometheus ← Grafana
                 Go API /metrics ← Prometheus scrape
```

Single binary runs both API server and worker pool (N goroutines consuming Redis Streams). Shared DB pool and Redis client.

## Execution Model

**Flow:** POST /runs → create run (pending) → XADD to `flowforge:steps` → worker XREADGROUP → execute step → write step_execution → XACK → enqueue next step or complete run.

**Redis Streams durability:**
- Stream: `flowforge:steps`, consumer group: `workers`
- Consumer name: `worker-<hostname>-<pid>`
- Message payload: `{ run_id, step_index, attempt_id (ULID), enqueued_at, delay_until }`
- Crash recovery: background goroutine every 30s runs XPENDING, XCLAIMs messages idle >60s

**Retry policy** (per-workflow JSONB):
```json
{ "max_retries": 3, "initial_delay_ms": 1000, "max_delay_ms": 30000, "multiplier": 2.0 }
```
Delay for attempt N: `min(initial_delay_ms * multiplier^(N-1), max_delay_ms)`

**Idempotency:** Each message carries unique `attempt_id`. Before executing, worker does `INSERT ... ON CONFLICT (attempt_id) DO NOTHING` — if 0 rows affected, another worker won the race, skip.

**Built-in actions:**

| Action | Input | Behavior |
|--------|-------|----------|
| `http_call` | `{ url, method, headers, body, timeout_ms }` | HTTP request, returns `{ status, headers, body }` |
| `delay` | `{ duration_ms }` | Sets `delay_until` on next enqueue (max 1hr) |
| `log` | `{ message, level }` | Writes to step output, always succeeds |
| `transform` | `{ expression, input_path, output_path }` | JSONPath field extraction via gjson/sjson |

Run context flows between steps — each step receives prior step's output.

## Data Model

All IDs: ULIDs (TEXT). Timestamps: TIMESTAMPTZ. Enums: TEXT + CHECK (not PG ENUM).

**Tables:**
- `users` (id, email, password_hash, display_name, created_at, updated_at)
- `refresh_tokens` (id, user_id FK, token_hash UNIQUE, expires_at, revoked_at, created_at)
- `workflows` (id, user_id FK, name, description, retry_policy JSONB, is_active, created_at, updated_at)
- `workflow_steps` (id, workflow_id FK, step_index, action CHECK, config JSONB, name, UNIQUE(workflow_id, step_index))
- `workflow_runs` (id, workflow_id FK, user_id FK, status CHECK [pending/running/completed/failed/cancelled], context JSONB, current_step, error_message, started_at, completed_at)
- `step_executions` (id, run_id FK, step_index, attempt_id UNIQUE, attempt_number, action, status CHECK, input JSONB, output JSONB, error_message, duration_ms, started_at, completed_at)

## API Surface

Base: `/api/v1`, JSON, `Authorization: Bearer <token>` (except auth routes).

**Auth:** POST /auth/register, /auth/login, /auth/refresh, /auth/logout, GET /auth/me
- Access: JWT HS256, 15min TTL, claims `{ sub, exp, iat }`
- Refresh: opaque 32-byte, SHA-256 stored, 30-day TTL, rotation on use

**Workflows:** POST/GET/GET:id/PUT/DELETE /workflows (steps replaced as batch on PUT)

**Runs:** POST /workflows/:id/runs, GET /workflows/:id/runs, GET /runs/:id, POST /runs/:id/cancel

**WebSocket:** `GET /api/v1/ws` → subscribe `{ type: "subscribe", run_ids: [...] }`
Events: `run.status_changed`, `step.started`, `step.completed`, `step.failed`, `run.completed`, `run.failed`
Internal: workers PUBLISH to Redis `flowforge:events:<run_id>`, WS hub subscribes and fans out.

**Error format:** `{ "error": { "code": "VALIDATION_FAILED", "message": "...", "details": {} } }`

## Observability

**Metrics** (Prometheus, `/metrics`):
- `flowforge_http_requests_total` (method, path, status)
- `flowforge_http_request_duration_sec` (histogram)
- `flowforge_workflow_runs_total` (status), `_active` (gauge)
- `flowforge_step_executions_total` (action, status), `_duration_ms` (histogram)
- `flowforge_redis_queue_depth`, `_pending_claims`
- `flowforge_ws_connections_active`

**Traces:** OTel SDK → Collector. Span hierarchy: `workflow.run` → `step.execute` → `http.client`.

**Grafana dashboards:** API health (req rate, latency, errors), execution engine (active runs, step durations, retry rate), infra (queue depth, WS connections).

## Performance Targets

| Metric | Target |
|--------|--------|
| API p95 latency | < 50ms |
| API p99 latency | < 150ms |
| Step pickup latency | < 100ms |
| Concurrent active runs | >= 500 |
| WS event latency (end-to-end) | < 200ms |
| Run throughput (3-step) | >= 200 runs/min |

## Load Testing (k6)

Scripts in `/infra/loadtest/`. 5 scenarios: api_crud_baseline (50 VU/2min), run_throughput (100 VU/5min), ws_fanout (200 VU/3min), burst_spike (500 VU/30s), sustained_concurrent (50 VU/10min). Thresholds: p95<50ms, p99<150ms, error rate<1%.

## 4-Week Roadmap

**Week 1 — Foundation + Core Execution:** Repo scaffold, Docker Compose, migrations, health endpoint, workflow CRUD, run creation, worker loop, all 4 step actions, integration tests.

**Week 2 — Auth + Real-Time + Retries:** JWT auth + refresh rotation, protect endpoints, retry logic with backoff, crash recovery (XPENDING/XCLAIM), idempotency guards, WebSocket hub + Redis pub/sub bridge, heartbeat.

**Week 3 — Frontend + Observability:** SvelteKit + Tailwind, auth pages, workflow list/create/edit, run viewer with live WS timeline, Prometheus metrics, OTel tracing, Grafana provisioning, structured logging.

**Week 4 — Load Testing + Polish:** k6 scripts, bottleneck tuning, rate limiting, validation hardening, error handling audit, README, stretch goals if time permits.
