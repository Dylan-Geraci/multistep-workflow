package model

import (
	"encoding/json"
	"time"
)

type Workflow struct {
	ID          string          `json:"id"`
	UserID      string          `json:"user_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	RetryPolicy json.RawMessage `json:"retry_policy"`
	IsActive    bool            `json:"is_active"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	Steps       []WorkflowStep  `json:"steps,omitempty"`
}

type WorkflowStep struct {
	ID        string          `json:"id"`
	StepIndex int             `json:"step_index"`
	Action    string          `json:"action"`
	Config    json.RawMessage `json:"config"`
	Name      string          `json:"name"`
}

type CreateWorkflowRequest struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	RetryPolicy json.RawMessage     `json:"retry_policy,omitempty"`
	Steps       []CreateStepRequest `json:"steps"`
}

type CreateStepRequest struct {
	Action string          `json:"action"`
	Config json.RawMessage `json:"config"`
	Name   string          `json:"name"`
}

var ValidActions = map[string]bool{
	"http_call": true,
	"delay":     true,
	"log":       true,
	"transform": true,
}
