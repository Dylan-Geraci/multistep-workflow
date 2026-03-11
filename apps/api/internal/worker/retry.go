package worker

import (
	"math"

	"github.com/dylangeraci/flowforge/internal/model"
)

// CalculateBackoffDelay returns the backoff delay in milliseconds for the given
// attempt number (1-based) using the retry policy's exponential backoff formula.
func CalculateBackoffDelay(policy model.RetryPolicy, attemptNumber int) int {
	delay := float64(policy.InitialDelayMs) * math.Pow(policy.Multiplier, float64(attemptNumber-1))
	if delay > float64(policy.MaxDelayMs) {
		delay = float64(policy.MaxDelayMs)
	}
	return int(delay)
}
