package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/dylangeraci/flowforge/internal/middleware"
	"github.com/dylangeraci/flowforge/internal/model"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oklog/ulid/v2"
)

type WorkflowHandler struct {
	db *pgxpool.Pool
}

func NewWorkflowHandler(db *pgxpool.Pool) *WorkflowHandler {
	return &WorkflowHandler{db: db}
}

func newULID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)).String()
}

func (h *WorkflowHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var req model.CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		model.WriteError(w, http.StatusBadRequest, "INVALID_JSON", "Request body is not valid JSON")
		return
	}

	if req.Name == "" {
		model.WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "name is required")
		return
	}
	if len(req.Steps) == 0 {
		model.WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "steps must not be empty")
		return
	}
	for i, s := range req.Steps {
		if !model.ValidActions[s.Action] {
			model.WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED",
				fmt.Sprintf("step %d: invalid action %q, must be one of: http_call, delay, log, transform", i, s.Action))
			return
		}
	}

	retryPolicy := req.RetryPolicy
	if retryPolicy == nil {
		retryPolicy = json.RawMessage(`{"max_retries":3,"initial_delay_ms":1000,"max_delay_ms":30000,"multiplier":2.0}`)
	}

	workflowID := newULID()
	now := time.Now().UTC()

	tx, err := h.db.Begin(r.Context())
	if err != nil {
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to begin transaction")
		return
	}
	defer tx.Rollback(r.Context())

	_, err = tx.Exec(r.Context(),
		`INSERT INTO workflows (id, user_id, name, description, retry_policy, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		workflowID, userID, req.Name, req.Description, retryPolicy, now, now,
	)
	if err != nil {
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to create workflow")
		return
	}

	steps := make([]model.WorkflowStep, len(req.Steps))
	for i, s := range req.Steps {
		stepID := newULID()
		config := s.Config
		if config == nil {
			config = json.RawMessage(`{}`)
		}
		_, err = tx.Exec(r.Context(),
			`INSERT INTO workflow_steps (id, workflow_id, step_index, action, config, name)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			stepID, workflowID, i, s.Action, config, s.Name,
		)
		if err != nil {
			model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to create workflow step")
			return
		}
		steps[i] = model.WorkflowStep{
			ID:        stepID,
			StepIndex: i,
			Action:    s.Action,
			Config:    config,
			Name:      s.Name,
		}
	}

	if err := tx.Commit(r.Context()); err != nil {
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to commit transaction")
		return
	}

	wf := model.Workflow{
		ID:          workflowID,
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		RetryPolicy: retryPolicy,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
		Steps:       steps,
	}
	model.WriteJSON(w, http.StatusCreated, wf)
}

func (h *WorkflowHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	rows, err := h.db.Query(r.Context(),
		`SELECT id, user_id, name, description, retry_policy, is_active, created_at, updated_at
		 FROM workflows WHERE user_id = $1
		 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to list workflows")
		return
	}
	defer rows.Close()

	workflows := []model.Workflow{}
	for rows.Next() {
		var wf model.Workflow
		if err := rows.Scan(&wf.ID, &wf.UserID, &wf.Name, &wf.Description, &wf.RetryPolicy, &wf.IsActive, &wf.CreatedAt, &wf.UpdatedAt); err != nil {
			model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to scan workflow")
			return
		}
		workflows = append(workflows, wf)
	}

	var total int
	h.db.QueryRow(r.Context(), `SELECT COUNT(*) FROM workflows WHERE user_id = $1`, userID).Scan(&total)

	model.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"data":   workflows,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *WorkflowHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var wf model.Workflow
	err := h.db.QueryRow(r.Context(),
		`SELECT id, user_id, name, description, retry_policy, is_active, created_at, updated_at
		 FROM workflows WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&wf.ID, &wf.UserID, &wf.Name, &wf.Description, &wf.RetryPolicy, &wf.IsActive, &wf.CreatedAt, &wf.UpdatedAt)
	if err != nil {
		model.WriteError(w, http.StatusNotFound, "NOT_FOUND", "Workflow not found")
		return
	}

	wf.Steps, err = h.getSteps(r.Context(), id)
	if err != nil {
		model.WriteError(w, http.StatusInternalServerError, "INTERNAL", "Failed to load workflow steps")
		return
	}

	model.WriteJSON(w, http.StatusOK, wf)
}

func (h *WorkflowHandler) getSteps(ctx context.Context, workflowID string) ([]model.WorkflowStep, error) {
	rows, err := h.db.Query(ctx,
		`SELECT id, step_index, action, config, name
		 FROM workflow_steps WHERE workflow_id = $1 ORDER BY step_index`,
		workflowID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []model.WorkflowStep
	for rows.Next() {
		var s model.WorkflowStep
		if err := rows.Scan(&s.ID, &s.StepIndex, &s.Action, &s.Config, &s.Name); err != nil {
			return nil, err
		}
		steps = append(steps, s)
	}
	if steps == nil {
		steps = []model.WorkflowStep{}
	}
	return steps, nil
}
