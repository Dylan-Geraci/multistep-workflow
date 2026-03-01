# FlowForge

A durable workflow execution engine with real-time dashboard.

## Quick Start

```bash
cd infra/docker
docker compose up --build
```

### Verify

```bash
# Health check
curl http://localhost:8080/health

# Create a workflow
curl -X POST http://localhost:8080/api/v1/workflows \
  -H 'Content-Type: application/json' \
  -d '{"name":"Test","steps":[{"action":"log","config":{"message":"hello","level":"info"},"name":"Step 1"}]}'

# List workflows
curl http://localhost:8080/api/v1/workflows
```

## Architecture

- **Go API** (Chi router) — REST + WebSocket
- **PostgreSQL** — state of record
- **Redis** — queue (Streams) + pub/sub

See [docs/PLAN.md](docs/PLAN.md) for the full project plan.
