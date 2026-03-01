package worker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type transformConfig struct {
	Expression string `json:"expression"`
	InputPath  string `json:"input_path"`
	OutputPath string `json:"output_path"`
}

func executeTransform(_ context.Context, config json.RawMessage, runContext json.RawMessage) (json.RawMessage, error) {
	var cfg transformConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("invalid transform config: %w", err)
	}

	if cfg.OutputPath == "" {
		cfg.OutputPath = "result"
	}

	source := string(runContext)
	if len(source) == 0 {
		source = "{}"
	}

	// Narrow source via input_path
	if cfg.InputPath != "" {
		result := gjson.Get(source, cfg.InputPath)
		if !result.Exists() {
			source = "{}"
		} else {
			source = result.Raw
		}
	}

	// Extract value with expression
	var extracted interface{}
	if cfg.Expression != "" {
		result := gjson.Get(source, cfg.Expression)
		if result.Exists() {
			extracted = result.Value()
		}
	} else {
		// No expression = pass through the source
		extracted = gjson.Parse(source).Value()
	}

	// Set value at output_path
	out, err := sjson.Set("{}", cfg.OutputPath, extracted)
	if err != nil {
		return nil, fmt.Errorf("failed to set output: %w", err)
	}

	return json.RawMessage(out), nil
}
