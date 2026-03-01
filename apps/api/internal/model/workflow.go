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

type RetryPolicy struct {
	MaxRetries     int     `json:"max_retries"`
	InitialDelayMs int     `json:"initial_delay_ms"`
	MaxDelayMs     int     `json:"max_delay_ms"`
	Multiplier     float64 `json:"multiplier"`
}

func (rp RetryPolicy) WithDefaults() RetryPolicy {
	if rp.MaxRetries == 0 {
		rp.MaxRetries = 3
	}
	if rp.InitialDelayMs == 0 {
		rp.InitialDelayMs = 1000
	}
	if rp.MaxDelayMs == 0 {
		rp.MaxDelayMs = 60000
	}
	if rp.Multiplier == 0 {
		rp.Multiplier = 2.0
	}
	return rp
}

var ValidActions = map[string]bool{
	"http_call": true,
	"delay":     true,
	"log":       true,
	"transform": true,
}
