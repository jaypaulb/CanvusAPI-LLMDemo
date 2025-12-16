// Package logging provides structured logging unit tests using zaptest/observer.
// These tests verify JSON serialization, field sanitization, log levels, and
// ObjectMarshaler implementations.
package logging

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"go_backend/core"
)

// newObservedCore creates a zapcore.Core with an observer for testing.
// Returns the core and the observer logs for verification.
func newObservedCore(level zapcore.Level) (zapcore.Core, *observer.ObservedLogs) {
	return observer.New(level)
}

// TestJSONOutputFormat_StructuredFields verifies that structured fields are
// captured correctly in JSON format via the observer.
func TestJSONOutputFormat_StructuredFields(t *testing.T) {
	observerCore, logs := newObservedCore(zapcore.InfoLevel)
	logger := zap.New(observerCore)

	// Log with various field types
	logger.Info("test message",
		zap.String("string_field", "test_value"),
		zap.Int("int_field", 42),
		zap.Float64("float_field", 3.14),
		zap.Bool("bool_field", true),
		zap.Duration("duration_field", 2*time.Second),
	)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]

	// Verify message
	if entry.Message != "test message" {
		t.Errorf("message = %q, want %q", entry.Message, "test message")
	}

	// Verify level
	if entry.Level != zapcore.InfoLevel {
		t.Errorf("level = %v, want %v", entry.Level, zapcore.InfoLevel)
	}

	// Verify context fields are captured
	contextMap := entry.ContextMap()

	tests := []struct {
		key      string
		expected interface{}
	}{
		{"string_field", "test_value"},
		{"int_field", int64(42)},
		{"float_field", float64(3.14)},
		{"bool_field", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			val, ok := contextMap[tt.key]
			if !ok {
				t.Errorf("field %q not found in context", tt.key)
				return
			}
			if val != tt.expected {
				t.Errorf("field %q = %v (%T), want %v (%T)",
					tt.key, val, val, tt.expected, tt.expected)
			}
		})
	}
}

// TestLogLevelFiltering_DebugFilteredAtInfoLevel verifies that log level
// filtering works correctly - Debug messages should not appear at Info level.
func TestLogLevelFiltering_DebugFilteredAtInfoLevel(t *testing.T) {
	observerCore, logs := newObservedCore(zapcore.InfoLevel)
	logger := zap.New(observerCore)

	// Log at various levels
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	entries := logs.All()

	// Debug should be filtered out at InfoLevel
	if len(entries) != 3 {
		t.Errorf("expected 3 log entries (info, warn, error), got %d", len(entries))
	}

	// Verify the messages that made it through
	expectedMessages := []string{"info message", "warn message", "error message"}
	for i, msg := range expectedMessages {
		if i >= len(entries) {
			t.Errorf("missing entry %d: %q", i, msg)
			continue
		}
		if entries[i].Message != msg {
			t.Errorf("entry[%d].Message = %q, want %q", i, entries[i].Message, msg)
		}
	}
}

// TestLogLevelFiltering_AllLevelsAtDebug verifies that all levels are captured
// when the minimum level is Debug.
func TestLogLevelFiltering_AllLevelsAtDebug(t *testing.T) {
	observerCore, logs := newObservedCore(zapcore.DebugLevel)
	logger := zap.New(observerCore)

	// Log at all levels
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	entries := logs.All()

	if len(entries) != 4 {
		t.Fatalf("expected 4 log entries, got %d", len(entries))
	}

	expectedLevels := []zapcore.Level{
		zapcore.DebugLevel,
		zapcore.InfoLevel,
		zapcore.WarnLevel,
		zapcore.ErrorLevel,
	}

	for i, level := range expectedLevels {
		if entries[i].Level != level {
			t.Errorf("entry[%d].Level = %v, want %v", i, entries[i].Level, level)
		}
	}
}

// TestGPUMetrics_ObjectMarshalerEncoding verifies that GPUMetrics correctly
// implements zapcore.ObjectMarshaler and produces the expected JSON field names.
func TestGPUMetrics_ObjectMarshalerEncoding(t *testing.T) {
	observerCore, logs := newObservedCore(zapcore.InfoLevel)
	logger := zap.New(observerCore)

	gpu := core.GPUMetrics{
		VRAMUsedMB:     4096,
		VRAMTotalMB:    8192,
		GPUUtilization: 85.5,
		Temperature:    72.0,
	}

	logger.Info("gpu status", zap.Object("gpu", gpu))

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	// The object should be in the context as a map
	contextMap := entries[0].ContextMap()
	gpuData, ok := contextMap["gpu"]
	if !ok {
		t.Fatal("gpu field not found in context")
	}

	// Convert to map for field verification
	gpuMap, ok := gpuData.(map[string]interface{})
	if !ok {
		t.Fatalf("gpu data is not a map, got %T", gpuData)
	}

	// Verify field names match JSON struct tags from core.GPUMetrics
	expectedFields := map[string]interface{}{
		"vram_used_mb":    int64(4096),
		"vram_total_mb":   int64(8192),
		"gpu_utilization": float64(85.5),
		"temperature":     float64(72.0),
	}

	for key, expected := range expectedFields {
		t.Run(key, func(t *testing.T) {
			val, ok := gpuMap[key]
			if !ok {
				t.Errorf("field %q not found in GPU data", key)
				return
			}
			if val != expected {
				t.Errorf("field %q = %v (%T), want %v (%T)",
					key, val, val, expected, expected)
			}
		})
	}
}

// TestInferenceMetrics_JSONFieldNames verifies that InferenceMetrics JSON field
// names match the struct tags when marshaled via zapcore.ObjectMarshaler.
func TestInferenceMetrics_JSONFieldNames(t *testing.T) {
	metrics := InferenceMetrics{
		ModelName:        "llama-3.2-8b",
		PromptTokens:     150,
		CompletionTokens: 200,
		TotalTokens:      350,
		Duration:         2500 * time.Millisecond,
		TokensPerSecond:  140.0,
		GPU: core.GPUMetrics{
			VRAMUsedMB:     4096,
			VRAMTotalMB:    8192,
			GPUUtilization: 85.5,
			Temperature:    72.0,
		},
	}

	// Use the mock encoder to capture fields
	enc := newMockObjectEncoder()
	err := metrics.MarshalLogObject(enc)
	if err != nil {
		t.Fatalf("MarshalLogObject returned error: %v", err)
	}

	// Verify all expected JSON field names are present
	expectedStrings := map[string]string{
		"model_name": "llama-3.2-8b",
	}

	expectedInts := map[string]int{
		"prompt_tokens":     150,
		"completion_tokens": 200,
		"total_tokens":      350,
	}

	expectedInt64s := map[string]int64{
		"duration_ms": 2500,
	}

	expectedFloat64s := map[string]float64{
		"tokens_per_second": 140.0,
	}

	for key, expected := range expectedStrings {
		if got := enc.strings[key]; got != expected {
			t.Errorf("string field %q = %q, want %q", key, got, expected)
		}
	}

	for key, expected := range expectedInts {
		if got := enc.ints[key]; got != expected {
			t.Errorf("int field %q = %d, want %d", key, got, expected)
		}
	}

	for key, expected := range expectedInt64s {
		if got := enc.int64s[key]; got != expected {
			t.Errorf("int64 field %q = %d, want %d", key, got, expected)
		}
	}

	for key, expected := range expectedFloat64s {
		if got := enc.float64s[key]; got != expected {
			t.Errorf("float64 field %q = %f, want %f", key, got, expected)
		}
	}

	// Verify nested GPU object was called
	if _, ok := enc.objects["gpu"]; !ok {
		t.Error("gpu object field not found")
	}
}

// TestSensitiveFieldRedaction_APIKeyInFieldName verifies that fields with
// sensitive names are redacted by the Logger wrapper.
func TestSensitiveFieldRedaction_APIKeyInFieldName(t *testing.T) {
	// Create a Logger that will redact sensitive fields
	logger := &Logger{
		zap:           zap.NewNop(),
		sugar:         zap.NewNop().Sugar(),
		isDevelopment: false,
	}

	// Test redaction of fields with sensitive names
	fields := []zap.Field{
		zap.String("OPENAI_API_KEY", "sk-secret123456789012345678901234567890"),
		zap.String("user_api_key", "secret-value"),
		zap.String("password", "mysecretpassword"),
		zap.String("username", "john"), // Not sensitive
	}

	redacted := logger.redactFields(fields)

	// Verify sensitive fields are redacted
	for _, field := range redacted {
		switch field.Key {
		case "OPENAI_API_KEY", "user_api_key", "password":
			if field.String != RedactedPlaceholder {
				t.Errorf("field %q should be redacted, got %q", field.Key, field.String)
			}
		case "username":
			if field.String != "john" {
				t.Errorf("field %q should NOT be redacted, got %q", field.Key, field.String)
			}
		}
	}
}

// TestSensitiveFieldRedaction_PatternInValue verifies that values containing
// sensitive patterns are redacted even when the field name is not sensitive.
func TestSensitiveFieldRedaction_PatternInValue(t *testing.T) {
	// Create a Logger that will redact sensitive fields
	logger := &Logger{
		zap:           zap.NewNop(),
		sugar:         zap.NewNop().Sugar(),
		isDevelopment: false,
	}

	tests := []struct {
		name         string
		fieldName    string
		fieldValue   string
		shouldRedact bool
	}{
		{
			name:         "OpenAI key pattern in value",
			fieldName:    "config",
			fieldValue:   "key=sk-proj-abc123def456ghi789jkl012mno345",
			shouldRedact: true,
		},
		{
			name:         "Bearer token in value",
			fieldName:    "header",
			fieldValue:   "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.abc",
			shouldRedact: true,
		},
		{
			name:         "Normal value",
			fieldName:    "message",
			fieldValue:   "Hello, this is a normal message",
			shouldRedact: false,
		},
		{
			name:         "GitHub token in value",
			fieldName:    "config",
			fieldValue:   "token: ghp_abcdefghijklmnopqrstuvwxyz1234567890",
			shouldRedact: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := []zap.Field{zap.String(tt.fieldName, tt.fieldValue)}
			redacted := logger.redactFields(fields)

			if len(redacted) != 1 {
				t.Fatalf("expected 1 field, got %d", len(redacted))
			}

			containsRedacted := strings.Contains(redacted[0].String, RedactedPlaceholder)
			if tt.shouldRedact && !containsRedacted {
				t.Errorf("value should be redacted but wasn't: %q", redacted[0].String)
			}
			if !tt.shouldRedact && containsRedacted {
				t.Errorf("value should NOT be redacted but was: %q", redacted[0].String)
			}
		})
	}
}

// TestSensitiveFieldRedaction_SugaredLogger verifies that the sugared logger
// (key-value pairs) also redacts sensitive data correctly.
func TestSensitiveFieldRedaction_SugaredLogger(t *testing.T) {
	logger := &Logger{
		zap:           zap.NewNop(),
		sugar:         zap.NewNop().Sugar(),
		isDevelopment: false,
	}

	keysAndValues := []interface{}{
		"API_KEY", "sk-supersecret123456789012345678901234567890",
		"username", "john",
		"TOKEN", "some-secret-token-value123456789012345",
		"message", "normal message",
	}

	redacted := logger.redactKeysAndValues(keysAndValues)

	// Verify API_KEY is redacted (index 1)
	if redacted[1] != RedactedPlaceholder {
		t.Errorf("API_KEY value should be redacted, got %v", redacted[1])
	}

	// Verify username is NOT redacted (index 3)
	if redacted[3] != "john" {
		t.Errorf("username value should NOT be redacted, got %v", redacted[3])
	}

	// Verify TOKEN is redacted (index 5)
	if redacted[5] != RedactedPlaceholder {
		t.Errorf("TOKEN value should be redacted, got %v", redacted[5])
	}

	// Verify message is NOT redacted (index 7)
	if redacted[7] != "normal message" {
		t.Errorf("message value should NOT be redacted, got %v", redacted[7])
	}
}

// TestInferenceMetrics_JSONRoundTrip verifies that InferenceMetrics can be
// properly serialized to JSON and the output matches expected structure.
func TestInferenceMetrics_JSONRoundTrip(t *testing.T) {
	// This test verifies the JSON struct tags are correctly defined
	// by doing a standard JSON marshal/unmarshal roundtrip

	metrics := InferenceMetrics{
		ModelName:        "test-model",
		PromptTokens:     100,
		CompletionTokens: 200,
		TotalTokens:      300,
		Duration:         2 * time.Second,
		TokensPerSecond:  150.0,
		GPU: core.GPUMetrics{
			VRAMUsedMB:     4096,
			VRAMTotalMB:    8192,
			GPUUtilization: 85.5,
			Temperature:    72.0,
		},
	}

	// Marshal to JSON
	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Verify expected field names appear in JSON
	jsonStr := string(data)
	expectedFields := []string{
		`"model_name"`,
		`"prompt_tokens"`,
		`"completion_tokens"`,
		`"total_tokens"`,
		`"duration"`,
		`"tokens_per_second"`,
		`"vram_used_mb"`,
		`"vram_total_mb"`,
		`"gpu_utilization"`,
		`"temperature"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON output missing field %s, got: %s", field, jsonStr)
		}
	}

	// Unmarshal back and verify values
	var decoded InferenceMetrics
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	if decoded.ModelName != metrics.ModelName {
		t.Errorf("ModelName = %q, want %q", decoded.ModelName, metrics.ModelName)
	}
	if decoded.TotalTokens != metrics.TotalTokens {
		t.Errorf("TotalTokens = %d, want %d", decoded.TotalTokens, metrics.TotalTokens)
	}
	if decoded.GPU.VRAMUsedMB != metrics.GPU.VRAMUsedMB {
		t.Errorf("GPU.VRAMUsedMB = %d, want %d", decoded.GPU.VRAMUsedMB, metrics.GPU.VRAMUsedMB)
	}
}
