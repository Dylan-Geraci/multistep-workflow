package ws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dylangeraci/flowforge/internal/model"
	"github.com/redis/go-redis/v9"
)

func PublishEvent(ctx context.Context, rdb *redis.Client, runID string, event model.WSEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	return rdb.Publish(ctx, "flowforge:events:"+runID, data).Err()
}
