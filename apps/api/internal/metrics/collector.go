package metrics

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const streamName = "flowforge:steps"

// StartCollector runs a background goroutine that updates queue depth and health gauges every 15 seconds.
func StartCollector(ctx context.Context, pool *pgxpool.Pool, rdb *redis.Client) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	collect(ctx, pool, rdb) // initial collection
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			collect(ctx, pool, rdb)
		}
	}
}

func collect(ctx context.Context, pool *pgxpool.Pool, rdb *redis.Client) {
	// Queue depth via XLEN
	length, err := rdb.XLen(ctx, streamName).Result()
	if err != nil {
		slog.Error("metrics collector: XLEN failed", "error", err)
	} else {
		QueueDepth.Set(float64(length))
	}

	// Redis health
	if err := rdb.Ping(ctx).Err(); err != nil {
		RedisUp.Set(0)
	} else {
		RedisUp.Set(1)
	}

	// PostgreSQL health
	if err := pool.Ping(ctx); err != nil {
		PGUp.Set(0)
	} else {
		PGUp.Set(1)
	}
}
