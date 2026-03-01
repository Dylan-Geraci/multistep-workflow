package router

import (
	"github.com/dylangeraci/flowforge/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func New(db *pgxpool.Pool, rdb *redis.Client) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	health := handler.NewHealthHandler(db, rdb)
	r.Get("/health", health.Check)

	workflows := handler.NewWorkflowHandler(db)
	runs := handler.NewRunHandler(db, rdb)
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/workflows", workflows.Create)
		r.Get("/workflows", workflows.List)
		r.Get("/workflows/{id}", workflows.GetByID)

		r.Post("/workflows/{id}/runs", runs.Create)
		r.Get("/workflows/{id}/runs", runs.List)
		r.Get("/runs/{id}", runs.GetByID)
	})

	return r
}
