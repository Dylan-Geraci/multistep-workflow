-- Queries for workflow operations (reference for sqlc, used manually for now)

-- name: InsertWorkflow :exec
INSERT INTO workflows (id, user_id, name, description, retry_policy, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: InsertWorkflowStep :exec
INSERT INTO workflow_steps (id, workflow_id, step_index, action, config, name)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: ListWorkflows :many
SELECT id, user_id, name, description, retry_policy, is_active, created_at, updated_at
FROM workflows WHERE user_id = $1
ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: GetWorkflowByID :one
SELECT id, user_id, name, description, retry_policy, is_active, created_at, updated_at
FROM workflows WHERE id = $1 AND user_id = $2;

-- name: GetWorkflowSteps :many
SELECT id, step_index, action, config, name
FROM workflow_steps WHERE workflow_id = $1 ORDER BY step_index;
