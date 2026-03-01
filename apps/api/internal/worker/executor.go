package worker

import (
	"context"
	"encoding/json"
	"fmt"
)

func ExecuteAction(ctx context.Context, action string, config json.RawMessage, runContext json.RawMessage) (json.RawMessage, error) {
	switch action {
	case "log":
		return executeLog(ctx, config, runContext)
	case "delay":
		return executeDelay(ctx, config, runContext)
	case "http_call":
		return executeHTTPCall(ctx, config, runContext)
	case "transform":
		return executeTransform(ctx, config, runContext)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}
