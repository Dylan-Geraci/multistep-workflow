-- Queries for run operations (reference for sqlc, used manually for now)

-- name: InsertRun :exec
INSERT INTO workflow_runs (id, workflow_id, user_id, status, context, current_step, created_at)
VALUES ($1, $2, $3, 'pending', $4, 0, $5);

-- name: GetRunByID :one
SELECT id, workflow_id, user_id, status, context, current_step, error_message, started_at, completed_at, created_at
FROM workflow_runs WHERE id = $1 AND user_id = $2;

-- name: ListRunsByWorkflow :many
SELECT id, workflow_id, user_id, status, context, current_step, error_message, started_at, completed_at, created_at
FROM workflow_runs WHERE workflow_id = $1 AND user_id = $2
ORDER BY created_at DESC LIMIT $3 OFFSET $4;

-- name: CountRunsByWorkflow :one
SELECT COUNT(*) FROM workflow_runs WHERE workflow_id = $1 AND user_id = $2;

-- name: UpdateRunStatus :exec
UPDATE workflow_runs SET status = $2, current_step = $3, started_at = COALESCE(started_at, now())
WHERE id = $1;

-- name: UpdateRunContext :exec
UPDATE workflow_runs SET context = $2 WHERE id = $1;

-- name: MarkRunCompleted :exec
UPDATE workflow_runs SET status = 'completed', completed_at = $2 WHERE id = $1;

-- name: MarkRunFailed :exec
UPDATE workflow_runs SET status = 'failed', error_message = $2, completed_at = $3 WHERE id = $1;

-- name: InsertStepExecution :exec
INSERT INTO step_executions (id, run_id, step_index, attempt_id, attempt_number, action, status, input, output, error_message, duration_ms, started_at, completed_at)
VALUES ($1, $2, $3, $4, 1, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: ListStepExecutions :many
SELECT id, run_id, step_index, attempt_id, attempt_number, action, status, input, output, error_message, duration_ms, started_at, completed_at
FROM step_executions WHERE run_id = $1 ORDER BY step_index, attempt_number;
