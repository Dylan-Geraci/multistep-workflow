# Changelog

## Unreleased

### Added
- Project PLAN.md with architecture, data model, API surface, observability plan
- 4-week milestone roadmap
- Repo scaffold with Go API, Docker Compose (PostgreSQL + Redis)
- Database migrations for all 6 tables (users, refresh_tokens, workflows, workflow_steps, workflow_runs, step_executions)
- Health endpoint with PostgreSQL and Redis connectivity checks
- Workflow CRUD endpoints (create, list, get by ID)
- Multi-stage Dockerfile for Go API
