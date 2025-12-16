package logging

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"go_backend/core"
)

// createTestLogger creates a logger and observer for testing zap fields.
func createTestLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.DebugLevel)
	return zap.New(core), logs
}

func TestInferenceFields_ReturnsValidZapField(t *testing.T) {
	metrics := InferenceMetrics{
		ModelName:        "llama-3.2-8b",
		PromptTokens:     150,
		CompletionTokens: 200,
		TotalTokens:      350,
		Duration:         2 * time.Second,
		TokensPerSecond:  175.0,
		GPU: core.GPUMetrics{
			VRAMUsedMB:     4096,
			VRAMTotalMB:    8192,
			GPUUtilization: 85.5,
			Temperature:    72.0,
		},
	}

	logger, logs := createTestLogger()
	field := InferenceFields(metrics)

	// Should not panic when used with logger
	logger.Info("test message", field)

	// Verify the log entry was created
	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]

	// Verify the field key is "inference"
	if field.Key != "inference" {
		t.Errorf("field key = %q, want %q", field.Key, "inference")
	}

	// Verify the field type is Object
	if field.Type != zapcore.ObjectMarshalerType {
		t.Errorf("field type = %v, want ObjectMarshalerType", field.Type)
	}

	// Verify log message
	if entry.Message != "test message" {
		t.Errorf("log message = %q, want %q", entry.Message, "test message")
	}
}

func TestGPUFields_ReturnsValidZapField(t *testing.T) {
	gpuMetrics := core.GPUMetrics{
		VRAMUsedMB:     4096,
		VRAMTotalMB:    8192,
		GPUUtilization: 85.5,
		Temperature:    72.0,
	}

	logger, logs := createTestLogger()
	field := GPUFields(gpuMetrics)

	// Should not panic when used with logger
	logger.Info("gpu status", field)

	// Verify the log entry was created
	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	// Verify the field key is "gpu"
	if field.Key != "gpu" {
		t.Errorf("field key = %q, want %q", field.Key, "gpu")
	}

	// Verify the field type is Object
	if field.Type != zapcore.ObjectMarshalerType {
		t.Errorf("field type = %v, want ObjectMarshalerType", field.Type)
	}
}

func TestTokenFields_ReturnsCorrectSlice(t *testing.T) {
	prompt := 150
	completion := 200
	total := 350

	fields := TokenFields(prompt, completion, total)

	// Verify we get exactly 3 fields
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}

	// Verify field keys and values
	expectedFields := map[string]int{
		"prompt_tokens":     prompt,
		"completion_tokens": completion,
		"total_tokens":      total,
	}

	for _, field := range fields {
		expected, ok := expectedFields[field.Key]
		if !ok {
			t.Errorf("unexpected field key: %s", field.Key)
			continue
		}

		if field.Type != zapcore.Int64Type {
			t.Errorf("field %s type = %v, want Int64Type", field.Key, field.Type)
		}

		// zap.Int stores value as int64 in Integer field
		if int(field.Integer) != expected {
			t.Errorf("field %s value = %d, want %d", field.Key, field.Integer, expected)
		}
	}

	// Verify fields work with logger
	logger, logs := createTestLogger()
	logger.Info("token usage", fields...)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
}

func TestTimingFields_CalculatesDurationCorrectly(t *testing.T) {
	startTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 15, 10, 0, 2, 500000000, time.UTC) // 2.5 seconds later
	tokensPerSecond := 140.0
	expectedDuration := 2500 * time.Millisecond

	fields := TimingFields(startTime, endTime, tokensPerSecond)

	// Verify we get exactly 4 fields
	if len(fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(fields))
	}

	// Verify field keys are present
	foundKeys := make(map[string]bool)
	for _, field := range fields {
		foundKeys[field.Key] = true
	}

	expectedKeys := []string{"start_time", "end_time", "duration", "tokens_per_second"}
	for _, key := range expectedKeys {
		if !foundKeys[key] {
			t.Errorf("missing expected field key: %s", key)
		}
	}

	// Verify duration calculation
	for _, field := range fields {
		if field.Key == "duration" {
			// Duration is stored as int64 nanoseconds in the Integer field
			gotDuration := time.Duration(field.Integer)
			if gotDuration != expectedDuration {
				t.Errorf("duration = %v, want %v", gotDuration, expectedDuration)
			}
		}

		if field.Key == "tokens_per_second" {
			// Float64 is stored in Interface field when using zap.Float64
			// The type will be Float64Type
			if field.Type != zapcore.Float64Type {
				t.Errorf("tokens_per_second type = %v, want Float64Type", field.Type)
			}
		}
	}

	// Verify fields work with logger
	logger, logs := createTestLogger()
	logger.Info("timing info", fields...)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
}

func TestTimingFields_ZeroDuration(t *testing.T) {
	now := time.Now()
	fields := TimingFields(now, now, 0.0)

	// Verify we still get 4 fields with zero duration
	if len(fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(fields))
	}

	// Find duration field and verify it's zero
	for _, field := range fields {
		if field.Key == "duration" {
			gotDuration := time.Duration(field.Integer)
			if gotDuration != 0 {
				t.Errorf("duration = %v, want 0", gotDuration)
			}
		}
	}
}
