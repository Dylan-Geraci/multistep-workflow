package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dylangeraci/flowforge/internal/config"
	"github.com/dylangeraci/flowforge/internal/db"
	"github.com/dylangeraci/flowforge/internal/logging"
	"github.com/dylangeraci/flowforge/internal/metrics"
	"github.com/dylangeraci/flowforge/internal/router"
	"github.com/dylangeraci/flowforge/internal/worker"
	"github.com/dylangeraci/flowforge/internal/ws"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	logger := logging.New(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx := context.Background()

	// Connect to PostgreSQL
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("unable to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("unable to ping database", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to PostgreSQL")

	// Run migrations
	if err := db.RunMigrations(cfg.DatabaseURL); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")

	// Connect to Redis
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		slog.Error("invalid REDIS_URL", "error", err)
		os.Exit(1)
	}
	rdb := redis.NewClient(opts)
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("unable to connect to Redis", "error", err)
		os.Exit(1)
	}
	slog.Info("connected to Redis")

	// Initialize metrics
	m := metrics.New()
	go metrics.StartCollector(ctx, rdb, m)

	// Start WebSocket hub
	hub := ws.NewHub(rdb, m)
	go hub.Run(ctx)

	// Start worker pool
	wp := worker.NewPool(pool, rdb, cfg.WorkerCount, cfg.RecoveryIntervalSecs, cfg.RecoveryIdleThresholdSecs, m)
	if err := wp.Start(ctx); err != nil {
		slog.Error("failed to start worker pool", "error", err)
		os.Exit(1)
	}

	// Setup router
	r := router.New(pool, rdb, cfg, hub, m)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("shutting down")

	// Stop worker pool first
	wp.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
