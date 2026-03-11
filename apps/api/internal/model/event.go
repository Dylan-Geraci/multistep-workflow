package model

import "encoding/json"

const (
	EventRunStatusChanged = "run.status_changed"
	EventStepStarted      = "step.started"
	EventStepCompleted    = "step.completed"
	EventStepFailed       = "step.failed"
	EventRunCompleted     = "run.completed"
	EventRunFailed        = "run.failed"
)

type WSIncomingMessage struct {
	Type   string   `json:"type"`
	RunIDs []string `json:"run_ids,omitempty"`
}

type WSEvent struct {
	Type  string          `json:"type"`
	RunID string          `json:"run_id"`
	Data  json.RawMessage `json:"data"`
}
