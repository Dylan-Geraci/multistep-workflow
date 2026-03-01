package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/dylangeraci/flowforge/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewHealthHandler(db *pgxpool.Pool, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

func (h *HealthHandler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	pgStatus := "ok"
	if err := h.db.Ping(ctx); err != nil {
		pgStatus = "error"
	}

	redisStatus := "ok"
	if err := h.redis.Ping(ctx).Err(); err != nil {
		redisStatus = "error"
	}

	status := "ok"
	httpStatus := http.StatusOK
	if pgStatus != "ok" || redisStatus != "ok" {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	model.WriteJSON(w, httpStatus, map[string]string{
		"status":  status,
		"pg":      pgStatus,
		"redis":   redisStatus,
		"version": "0.1.0",
	})
}
