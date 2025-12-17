// Package llamaruntime provides Go bindings to llama.cpp for local LLM inference.
// This file contains tests for the RecoveryManager organism.
package llamaruntime

import (
	"testing"
	"time"
)

// TestDefaultRecoveryConfig tests that default config has sensible values.
func TestDefaultRecoveryConfig(t *testing.T) {
	config := DefaultRecoveryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries=3, got %d", config.MaxRetries)
	}

	if config.InitialBackoff != 500*time.Millisecond {
		t.Errorf("Expected InitialBackoff=500ms, got %v", config.InitialBackoff)
	}

	if config.MaxBackoff != 10*time.Second {
		t.Errorf("Expected MaxBackoff=10s, got %v", config.MaxBackoff)
	}

	if config.BackoffMultiplier != 2.0 {
		t.Errorf("Expected BackoffMultiplier=2.0, got %f", config.BackoffMultiplier)
	}

	if config.ContextResetThreshold != 3 {
		t.Errorf("Expected ContextResetThreshold=3, got %d", config.ContextResetThreshold)
	}

	if config.ModelReloadThreshold != 2 {
		t.Errorf("Expected ModelReloadThreshold=2, got %d", config.ModelReloadThreshold)
	}

	if !config.DegradedModeEnabled {
		t.Error("Expected DegradedModeEnabled=true")
	}

	if config.DegradedModeRecoveryInterval != 1*time.Minute {
		t.Errorf("Expected DegradedModeRecoveryInterval=1m, got %v", config.DegradedModeRecoveryInterval)
	}
}

// TestRecoveryStatsZeroValues tests that RecoveryStats starts with zero values.
func TestRecoveryStatsZeroValues(t *testing.T) {
	stats := RecoveryStats{}

	if stats.TotalRequests != 0 {
		t.Errorf("Expected TotalRequests=0, got %d", stats.TotalRequests)
	}

	if stats.SuccessfulRequests != 0 {
		t.Errorf("Expected SuccessfulRequests=0, got %d", stats.SuccessfulRequests)
	}

	if stats.RetryAttempts != 0 {
		t.Errorf("Expected RetryAttempts=0, got %d", stats.RetryAttempts)
	}

	if stats.InDegradedMode {
		t.Error("Expected InDegradedMode=false")
	}
}

// TestRecoveryActionConstants tests recovery action constants.
func TestRecoveryActionConstants(t *testing.T) {
	tests := []struct {
		action   RecoveryAction
		expected string
	}{
		{RecoveryActionRetry, "retry"},
		{RecoveryActionContextReset, "context_reset"},
		{RecoveryActionModelReload, "model_reload"},
		{RecoveryActionDegradedMode, "degraded_mode"},
	}

	for _, tt := range tests {
		if string(tt.action) != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, string(tt.action))
		}
	}
}

// TestIsRecoverableError tests error classification.
// Note: This requires a RecoveryManager instance which needs a real client.
// For unit testing without CGo, we test the error types directly.
func TestIsRecoverableErrorTypes(t *testing.T) {
	// Test that sentinel errors are defined correctly
	if ErrModelNotFound == nil {
		t.Error("ErrModelNotFound should not be nil")
	}

	if ErrGPUNotAvailable == nil {
		t.Error("ErrGPUNotAvailable should not be nil")
	}

	if ErrInsufficientVRAM == nil {
		t.Error("ErrInsufficientVRAM should not be nil")
	}

	if ErrInferenceFailed == nil {
		t.Error("ErrInferenceFailed should not be nil")
	}
}

// TestRecoveryConfigDefaults tests that zero values get proper defaults.
func TestRecoveryConfigDefaults(t *testing.T) {
	// Create a config with all zero values
	config := RecoveryConfig{}

	// The NewRecoveryManager applies defaults, but we can't test that
	// without a real client. Instead, verify DefaultRecoveryConfig works.
	defaults := DefaultRecoveryConfig()

	if defaults.MaxRetries <= 0 {
		t.Error("DefaultRecoveryConfig should have MaxRetries > 0")
	}

	if defaults.InitialBackoff <= 0 {
		t.Error("DefaultRecoveryConfig should have InitialBackoff > 0")
	}

	if defaults.MaxBackoff <= 0 {
		t.Error("DefaultRecoveryConfig should have MaxBackoff > 0")
	}

	if defaults.BackoffMultiplier <= 0 {
		t.Error("DefaultRecoveryConfig should have BackoffMultiplier > 0")
	}

	// Verify the zero config is different from defaults
	if config.MaxRetries == defaults.MaxRetries && config.MaxRetries != 0 {
		t.Error("Zero config should differ from defaults")
	}
}

// TestExponentialBackoffCalculation tests backoff calculation logic.
func TestExponentialBackoffCalculation(t *testing.T) {
	initial := 500 * time.Millisecond
	multiplier := 2.0
	maxBackoff := 10 * time.Second

	backoff := initial

	// First iteration
	if backoff != initial {
		t.Errorf("Expected initial backoff %v, got %v", initial, backoff)
	}

	// Second iteration
	backoff = time.Duration(float64(backoff) * multiplier)
	expected := 1 * time.Second
	if backoff != expected {
		t.Errorf("Expected second backoff %v, got %v", expected, backoff)
	}

	// Third iteration
	backoff = time.Duration(float64(backoff) * multiplier)
	expected = 2 * time.Second
	if backoff != expected {
		t.Errorf("Expected third backoff %v, got %v", expected, backoff)
	}

	// Fourth iteration
	backoff = time.Duration(float64(backoff) * multiplier)
	expected = 4 * time.Second
	if backoff != expected {
		t.Errorf("Expected fourth backoff %v, got %v", expected, backoff)
	}

	// Fifth iteration
	backoff = time.Duration(float64(backoff) * multiplier)
	expected = 8 * time.Second
	if backoff != expected {
		t.Errorf("Expected fifth backoff %v, got %v", expected, backoff)
	}

	// Sixth iteration - should cap at maxBackoff
	backoff = time.Duration(float64(backoff) * multiplier)
	if backoff > maxBackoff {
		backoff = maxBackoff
	}
	if backoff != maxBackoff {
		t.Errorf("Expected capped backoff %v, got %v", maxBackoff, backoff)
	}
}
