package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

type logConfig struct {
	Message string `json:"message"`
	Level   string `json:"level"`
}

func executeLog(_ context.Context, config json.RawMessage, _ json.RawMessage) (json.RawMessage, error) {
	var cfg logConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("invalid log config: %w", err)
	}

	if cfg.Level == "" {
		cfg.Level = "info"
	}

	log.Printf("[%s] %s", cfg.Level, cfg.Message)

	out, _ := json.Marshal(map[string]interface{}{
		"message": cfg.Message,
		"level":   cfg.Level,
		"logged":  true,
	})
	return out, nil
}
