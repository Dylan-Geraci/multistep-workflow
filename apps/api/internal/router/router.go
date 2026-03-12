package router

import (
	"github.com/dylangeraci/flowforge/internal/config"
	"github.com/dylangeraci/flowforge/internal/handler"
	authmw "github.com/dylangeraci/flowforge/internal/middleware"
	"github.com/dylangeraci/flowforge/internal/ws"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
)

func New(db *pgxpool.Pool, rdb *redis.Client, cfg config.Config, hub *ws.Hub) *chi.Mux {
	r := chi.NewRouter()
	r.Use(authmw.RequestID)
	r.Use(authmw.StructuredLogger)
	r.Use(authmw.Metrics)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	// Unauthenticated endpoints
	health := handler.NewHealthHandler(db, rdb)
	r.Get("/health", health.Check)
	r.Handle("/metrics", promhttp.Handler())

	// Public auth routes
	auth := handler.NewAuthHandler(db, cfg)
	r.Post("/auth/register", auth.Register)
	r.Post("/auth/login", auth.Login)
	r.Post("/auth/refresh", auth.Refresh)
	r.Post("/auth/logout", auth.Logout)

	// Protected: /auth/me
	r.Group(func(r chi.Router) {
		r.Use(authmw.RequireAuth(cfg.JWTSecret))
		r.Get("/auth/me", auth.Me)
	})

	// Protected: WebSocket (auth required but no Content-Type enforcement)
	wsHandler := handler.NewWSHandler(hub)
	r.Group(func(r chi.Router) {
		r.Use(authmw.RequireAuth(cfg.JWTSecret))
		r.Get("/api/v1/ws", wsHandler.Handle)
	})

	// Protected: /api/v1/*
	workflows := handler.NewWorkflowHandler(db)
	runs := handler.NewRunHandler(db, rdb)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authmw.RequireAuth(cfg.JWTSecret))

		r.Post("/workflows", workflows.Create)
		r.Get("/workflows", workflows.List)
		r.Get("/workflows/{id}", workflows.GetByID)
		r.Put("/workflows/{id}", workflows.Update)
		r.Delete("/workflows/{id}", workflows.Delete)

		r.Post("/workflows/{id}/runs", runs.Create)
		r.Get("/workflows/{id}/runs", runs.List)
		r.Get("/runs/{id}", runs.GetByID)
		r.Post("/runs/{id}/cancel", runs.Cancel)
	})

	return r
}
