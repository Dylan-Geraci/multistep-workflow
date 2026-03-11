package worker

import (
	"testing"

	"github.com/dylangeraci/flowforge/internal/model"
)

func TestCalculateBackoffDelay(t *testing.T) {
	policy := model.RetryPolicy{
		MaxRetries:     3,
		InitialDelayMs: 1000,
		MaxDelayMs:     60000,
		Multiplier:     2.0,
	}

	tests := []struct {
		attempt  int
		expected int
	}{
		{1, 1000},  // 1000 * 2^0 = 1000
		{2, 2000},  // 1000 * 2^1 = 2000
		{3, 4000},  // 1000 * 2^2 = 4000
		{4, 8000},  // 1000 * 2^3 = 8000
		{5, 16000}, // 1000 * 2^4 = 16000
	}

	for _, tt := range tests {
		got := CalculateBackoffDelay(policy, tt.attempt)
		if got != tt.expected {
			t.Errorf("attempt %d: got %d, want %d", tt.attempt, got, tt.expected)
		}
	}
}

func TestCalculateBackoffDelay_CappedAtMax(t *testing.T) {
	policy := model.RetryPolicy{
		MaxRetries:     10,
		InitialDelayMs: 1000,
		MaxDelayMs:     5000,
		Multiplier:     2.0,
	}

	// 1000 * 2^5 = 32000, should be capped at 5000
	got := CalculateBackoffDelay(policy, 6)
	if got != 5000 {
		t.Errorf("expected cap at 5000, got %d", got)
	}
}

func TestCalculateBackoffDelay_CustomMultiplier(t *testing.T) {
	policy := model.RetryPolicy{
		MaxRetries:     5,
		InitialDelayMs: 500,
		MaxDelayMs:     100000,
		Multiplier:     3.0,
	}

	tests := []struct {
		attempt  int
		expected int
	}{
		{1, 500},   // 500 * 3^0 = 500
		{2, 1500},  // 500 * 3^1 = 1500
		{3, 4500},  // 500 * 3^2 = 4500
		{4, 13500}, // 500 * 3^3 = 13500
	}

	for _, tt := range tests {
		got := CalculateBackoffDelay(policy, tt.attempt)
		if got != tt.expected {
			t.Errorf("attempt %d: got %d, want %d", tt.attempt, got, tt.expected)
		}
	}
}
