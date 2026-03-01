package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type delayConfig struct {
	DurationMs int `json:"duration_ms"`
}

func executeDelay(ctx context.Context, config json.RawMessage, _ json.RawMessage) (json.RawMessage, error) {
	var cfg delayConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("invalid delay config: %w", err)
	}

	if cfg.DurationMs <= 0 || cfg.DurationMs > 3600000 {
		return nil, fmt.Errorf("duration_ms must be between 1 and 3600000, got %d", cfg.DurationMs)
	}

	select {
	case <-time.After(time.Duration(cfg.DurationMs) * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	out, _ := json.Marshal(map[string]interface{}{
		"delayed_ms": cfg.DurationMs,
	})
	return out, nil
}
