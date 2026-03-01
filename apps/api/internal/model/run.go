package model

import (
	"encoding/json"
	"time"
)

type Run struct {
	ID           string          `json:"id"`
	WorkflowID   string          `json:"workflow_id"`
	UserID       string          `json:"user_id"`
	Status       string          `json:"status"`
	Context      json.RawMessage `json:"context"`
	CurrentStep  int             `json:"current_step"`
	ErrorMessage *string         `json:"error_message,omitempty"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	Steps        []StepExecution `json:"steps,omitempty"`
}

type StepExecution struct {
	ID            string          `json:"id"`
	RunID         string          `json:"run_id"`
	StepIndex     int             `json:"step_index"`
	AttemptID     string          `json:"attempt_id"`
	AttemptNumber int             `json:"attempt_number"`
	Action        string          `json:"action"`
	Status        string          `json:"status"`
	Input         json.RawMessage `json:"input"`
	Output        json.RawMessage `json:"output"`
	ErrorMessage  *string         `json:"error_message,omitempty"`
	DurationMs    *int            `json:"duration_ms,omitempty"`
	StartedAt     *time.Time      `json:"started_at,omitempty"`
	CompletedAt   *time.Time      `json:"completed_at,omitempty"`
}

type CreateRunRequest struct {
	Context json.RawMessage `json:"context,omitempty"`
}

type StepMessage struct {
	RunID     string `json:"run_id"`
	StepIndex int    `json:"step_index"`
	AttemptID string `json:"attempt_id"`
}
