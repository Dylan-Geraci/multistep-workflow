package router

import (
	"github.com/dylangeraci/flowforge/internal/config"
	"github.com/dylangeraci/flowforge/internal/handler"
	authmw "github.com/dylangeraci/flowforge/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func New(db *pgxpool.Pool, rdb *redis.Client, cfg config.Config) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	health := handler.NewHealthHandler(db, rdb)
	r.Get("/health", health.Check)

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

	// Protected: /api/v1/*
	workflows := handler.NewWorkflowHandler(db)
	runs := handler.NewRunHandler(db, rdb)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authmw.RequireAuth(cfg.JWTSecret))

		r.Post("/workflows", workflows.Create)
		r.Get("/workflows", workflows.List)
		r.Get("/workflows/{id}", workflows.GetByID)

		r.Post("/workflows/{id}/runs", runs.Create)
		r.Get("/workflows/{id}/runs", runs.List)
		r.Get("/runs/{id}", runs.GetByID)
	})

	return r
}
