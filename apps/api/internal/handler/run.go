package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dylangeraci/flowforge/internal/metrics"
	"github.com/dylangeraci/flowforge/internal/middleware"
	"github.com/dylangeraci/flowforge/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const streamName = "flowforge:steps"

type RunHandler struct {
	db      *pgxpool.Pool
	rdb     *redis.Client
	metrics *metrics.Metrics
}

func NewRunHandler(db *pgxpool.Pool, rdb *redis.Client, m *metrics.Metrics) *RunHandler {
	return &RunHandler{db: db, rdb: rdb, metrics: m}
}

func (h *RunHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	workflowID := chi.URLParam(r, "id")

	// Validate workflow exists and belongs to user
	var wfID string
	err := h.db.QueryRow(r.Context(),
		`SELECT id FROM workflows WHERE id = $1 AND user_id = $2`,
		workflowID, userID,
	).Scan(&wfID)
	if err != nil {
		model.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Workflow not found")
		return
	}

	// Validate workflow has steps
	var stepCount int
	h.db.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM workflow_steps WHERE workflow_id = $1`, workflowID,
	).Scan(&stepCount)
	if stepCount == 0 {
		model.WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "Workflow has no steps")
		return
	}

	var req model.CreateRunRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			model.WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Request body is not valid JSON")
			return
		}
	}

	runContext := req.Context
	if runContext == nil {
		runContext = json.RawMessage(`{}`)
	}

	runID := newULID()
	now := time.Now().UTC()

	_, err = h.db.Exec(r.Context(),
		`INSERT INTO workflow_runs (id, workflow_id, user_id, status, context, current_step, created_at)
		 VALUES ($1, $2, $3, 'pending', $4, 0, $5)`,
		runID, workflowID, userID, runContext, now,
	)
	if err != nil {
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to create run")
		return
	}

	// XADD first step to Redis stream with trace context
	attemptID := newULID()
	xaddValues := map[string]interface{}{
		"run_id":         runID,
		"step_index":     "0",
		"attempt_id":     attemptID,
		"attempt_number": "1",
	}
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(r.Context(), carrier)
	if v := carrier.Get("traceparent"); v != "" {
		xaddValues["traceparent"] = v
	}
	err = h.rdb.XAdd(r.Context(), &redis.XAddArgs{
		Stream: streamName,
		Values: xaddValues,
	}).Err()
	if err != nil {
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to enqueue step")
		return
	}

	h.metrics.WorkflowRunsTotal.WithLabelValues("pending").Inc()

	run := model.Run{
		ID:          runID,
		WorkflowID:  workflowID,
		UserID:      userID,
		Status:      "pending",
		Context:     runContext,
		CurrentStep: 0,
		CreatedAt:   now,
	}
	model.WriteJSON(w, http.StatusCreated, run)
}

func (h *RunHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	workflowID := chi.URLParam(r, "id")

	// Validate workflow exists
	var wfID string
	err := h.db.QueryRow(r.Context(),
		`SELECT id FROM workflows WHERE id = $1 AND user_id = $2`,
		workflowID, userID,
	).Scan(&wfID)
	if err != nil {
		model.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Workflow not found")
		return
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	rows, err := h.db.Query(r.Context(),
		`SELECT id, workflow_id, user_id, status, context, current_step, error_message, started_at, completed_at, created_at
		 FROM workflow_runs WHERE workflow_id = $1 AND user_id = $2
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		workflowID, userID, limit, offset,
	)
	if err != nil {
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to list runs")
		return
	}
	defer rows.Close()

	runs := []model.Run{}
	for rows.Next() {
		var run model.Run
		if err := rows.Scan(&run.ID, &run.WorkflowID, &run.UserID, &run.Status, &run.Context, &run.CurrentStep, &run.ErrorMessage, &run.StartedAt, &run.CompletedAt, &run.CreatedAt); err != nil {
			model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to scan run")
			return
		}
		runs = append(runs, run)
	}

	var total int
	h.db.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM workflow_runs WHERE workflow_id = $1 AND user_id = $2`,
		workflowID, userID,
	).Scan(&total)

	model.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":   runs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *RunHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	runID := chi.URLParam(r, "id")

	var status string
	err := h.db.QueryRow(r.Context(),
		`SELECT status FROM workflow_runs WHERE id = $1 AND user_id = $2`,
		runID, userID,
	).Scan(&status)
	if err != nil {
		model.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Run not found")
		return
	}

	if status != "pending" && status != "running" {
		model.WriteError(w, http.StatusConflict, "CONFLICT", fmt.Sprintf("Cannot cancel run with status %q", status))
		return
	}

	now := time.Now().UTC()
	_, err = h.db.Exec(r.Context(),
		`UPDATE workflow_runs SET status = 'cancelled', completed_at = $2 WHERE id = $1`,
		runID, now,
	)
	if err != nil {
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to cancel run")
		return
	}

	var run model.Run
	h.db.QueryRow(r.Context(),
		`SELECT id, workflow_id, user_id, status, context, current_step, error_message, started_at, completed_at, created_at
		 FROM workflow_runs WHERE id = $1`,
		runID,
	).Scan(&run.ID, &run.WorkflowID, &run.UserID, &run.Status, &run.Context, &run.CurrentStep, &run.ErrorMessage, &run.StartedAt, &run.CompletedAt, &run.CreatedAt)

	model.WriteJSON(w, http.StatusOK, run)
}

func (h *RunHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	runID := chi.URLParam(r, "id")

	var run model.Run
	err := h.db.QueryRow(r.Context(),
		`SELECT id, workflow_id, user_id, status, context, current_step, error_message, started_at, completed_at, created_at
		 FROM workflow_runs WHERE id = $1 AND user_id = $2`,
		runID, userID,
	).Scan(&run.ID, &run.WorkflowID, &run.UserID, &run.Status, &run.Context, &run.CurrentStep, &run.ErrorMessage, &run.StartedAt, &run.CompletedAt, &run.CreatedAt)
	if err != nil {
		model.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Run not found")
		return
	}

	// Load step executions
	stepRows, err := h.db.Query(r.Context(),
		`SELECT id, run_id, step_index, attempt_id, attempt_number, action, status, input, output, error_message, duration_ms, started_at, completed_at
		 FROM step_executions WHERE run_id = $1 ORDER BY step_index, attempt_number`,
		runID,
	)
	if err != nil {
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to load step executions")
		return
	}
	defer stepRows.Close()

	steps := []model.StepExecution{}
	for stepRows.Next() {
		var se model.StepExecution
		if err := stepRows.Scan(&se.ID, &se.RunID, &se.StepIndex, &se.AttemptID, &se.AttemptNumber, &se.Action, &se.Status, &se.Input, &se.Output, &se.ErrorMessage, &se.DurationMs, &se.StartedAt, &se.CompletedAt); err != nil {
			model.WriteError(w, http.StatusInternalServerError, "INTERNAL", fmt.Sprintf("Failed to scan step execution: %v", err))
			return
		}
		steps = append(steps, se)
	}
	run.Steps = steps

	model.WriteJSON(w, http.StatusOK, run)
}
