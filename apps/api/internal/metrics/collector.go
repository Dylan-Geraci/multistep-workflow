package metrics

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	streamName = "flowforge:steps"
	groupName  = "workers"
)

// StartCollector runs a background loop that polls Redis for queue depth
// and pending claims, updating the corresponding Prometheus gauges.
func StartCollector(ctx context.Context, rdb *redis.Client, m *Metrics) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			collect(ctx, rdb, m)
		}
	}
}

func collect(ctx context.Context, rdb *redis.Client, m *Metrics) {
	// Queue depth: total messages in stream
	length, err := rdb.XLen(ctx, streamName).Result()
	if err != nil {
		slog.ErrorContext(ctx, "metrics collector: XLEN error", "error", err)
	} else {
		m.RedisQueueDepth.Set(float64(length))
	}

	// Pending claims: messages delivered but not ACKed
	pending, err := rdb.XPending(ctx, streamName, groupName).Result()
	if err != nil {
		slog.ErrorContext(ctx, "metrics collector: XPENDING error", "error", err)
	} else {
		m.RedisPendingClaims.Set(float64(pending.Count))
	}
}
