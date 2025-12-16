package core

import (
	"encoding/json"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// TestGPUMetrics_Instantiation tests that GPUMetrics can be instantiated correctly.
func TestGPUMetrics_Instantiation(t *testing.T) {
	metrics := GPUMetrics{
		VRAMUsedMB:     4096,
		VRAMTotalMB:    8192,
		GPUUtilization: 75.5,
		Temperature:    65.0,
	}

	if metrics.VRAMUsedMB != 4096 {
		t.Errorf("VRAMUsedMB = %d, want 4096", metrics.VRAMUsedMB)
	}
	if metrics.VRAMTotalMB != 8192 {
		t.Errorf("VRAMTotalMB = %d, want 8192", metrics.VRAMTotalMB)
	}
	if metrics.GPUUtilization != 75.5 {
		t.Errorf("GPUUtilization = %f, want 75.5", metrics.GPUUtilization)
	}
	if metrics.Temperature != 65.0 {
		t.Errorf("Temperature = %f, want 65.0", metrics.Temperature)
	}
}

// TestGPUMetrics_JSONMarshal tests that GPUMetrics can be marshaled to JSON.
func TestGPUMetrics_JSONMarshal(t *testing.T) {
	metrics := GPUMetrics{
		VRAMUsedMB:     4096,
		VRAMTotalMB:    8192,
		GPUUtilization: 75.5,
		Temperature:    65.0,
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("Failed to marshal GPUMetrics: %v", err)
	}

	var decoded GPUMetrics
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal GPUMetrics: %v", err)
	}

	if decoded != metrics {
		t.Errorf("Round-trip mismatch: got %+v, want %+v", decoded, metrics)
	}
}

// TestGPUMetrics_MarshalLogObject tests the zapcore.ObjectMarshaler implementation.
func TestGPUMetrics_MarshalLogObject(t *testing.T) {
	core, observed := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	metrics := GPUMetrics{
		VRAMUsedMB:     4096,
		VRAMTotalMB:    8192,
		GPUUtilization: 75.5,
		Temperature:    65.0,
	}

	logger.Info("gpu metrics", zap.Object("metrics", metrics))

	logs := observed.All()
	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}

	// Verify the metrics field exists
	fields := logs[0].Context
	found := false
	for _, f := range fields {
		if f.Key == "metrics" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'metrics' field in log context")
	}
}

// TestProcessingHistory_Instantiation tests that ProcessingHistory can be instantiated correctly.
func TestProcessingHistory_Instantiation(t *testing.T) {
	now := time.Now()
	history := ProcessingHistory{
		ID:            1,
		CorrelationID: "corr-123",
		CanvasID:      "canvas-456",
		WidgetID:      "widget-789",
		OperationType: "note",
		Prompt:        "What is the meaning of life?",
		Response:      "42",
		ModelName:     "gpt-4",
		InputTokens:   10,
		OutputTokens:  1,
		DurationMs:    150,
		Status:        "success",
		ErrorMessage:  "",
		CreatedAt:     now,
	}

	if history.CorrelationID != "corr-123" {
		t.Errorf("CorrelationID = %s, want corr-123", history.CorrelationID)
	}
	if history.Status != "success" {
		t.Errorf("Status = %s, want success", history.Status)
	}
	if history.DurationMs != 150 {
		t.Errorf("DurationMs = %d, want 150", history.DurationMs)
	}
}

// TestProcessingHistory_JSONMarshal tests that ProcessingHistory can be marshaled to JSON.
func TestProcessingHistory_JSONMarshal(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	history := ProcessingHistory{
		ID:            1,
		CorrelationID: "corr-123",
		CanvasID:      "canvas-456",
		WidgetID:      "widget-789",
		OperationType: "note",
		Prompt:        "Test prompt",
		Response:      "Test response",
		ModelName:     "gpt-4",
		InputTokens:   10,
		OutputTokens:  5,
		DurationMs:    150,
		Status:        "success",
		CreatedAt:     now,
	}

	data, err := json.Marshal(history)
	if err != nil {
		t.Fatalf("Failed to marshal ProcessingHistory: %v", err)
	}

	var decoded ProcessingHistory
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ProcessingHistory: %v", err)
	}

	// Compare truncated times for consistency
	if !decoded.CreatedAt.Equal(history.CreatedAt) {
		t.Errorf("CreatedAt mismatch: got %v, want %v", decoded.CreatedAt, history.CreatedAt)
	}
	if decoded.CorrelationID != history.CorrelationID {
		t.Errorf("CorrelationID mismatch: got %s, want %s", decoded.CorrelationID, history.CorrelationID)
	}
}

// TestCanvasEvent_Instantiation tests that CanvasEvent can be instantiated correctly.
func TestCanvasEvent_Instantiation(t *testing.T) {
	now := time.Now()
	event := CanvasEvent{
		ID:             1,
		CanvasID:       "canvas-123",
		WidgetID:       "widget-456",
		EventType:      EventTypeCreated,
		WidgetType:     WidgetTypeNote,
		ContentPreview: "Hello world...",
		CreatedAt:      now,
	}

	if event.EventType != EventTypeCreated {
		t.Errorf("EventType = %s, want %s", event.EventType, EventTypeCreated)
	}
	if event.WidgetType != WidgetTypeNote {
		t.Errorf("WidgetType = %s, want %s", event.WidgetType, WidgetTypeNote)
	}
}

// TestCanvasEvent_JSONMarshal tests that CanvasEvent can be marshaled to JSON.
func TestCanvasEvent_JSONMarshal(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	event := CanvasEvent{
		ID:             1,
		CanvasID:       "canvas-123",
		WidgetID:       "widget-456",
		EventType:      EventTypeUpdated,
		WidgetType:     WidgetTypeImage,
		ContentPreview: "Image preview",
		CreatedAt:      now,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal CanvasEvent: %v", err)
	}

	var decoded CanvasEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal CanvasEvent: %v", err)
	}

	if decoded.EventType != event.EventType {
		t.Errorf("EventType mismatch: got %s, want %s", decoded.EventType, event.EventType)
	}
	if decoded.WidgetType != event.WidgetType {
		t.Errorf("WidgetType mismatch: got %s, want %s", decoded.WidgetType, event.WidgetType)
	}
}

// TestCanvasEvent_Constants tests the event and widget type constants.
func TestCanvasEvent_Constants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"EventTypeCreated", EventTypeCreated, "created"},
		{"EventTypeUpdated", EventTypeUpdated, "updated"},
		{"EventTypeDeleted", EventTypeDeleted, "deleted"},
		{"WidgetTypeNote", WidgetTypeNote, "note"},
		{"WidgetTypeImage", WidgetTypeImage, "image"},
		{"WidgetTypePDF", WidgetTypePDF, "pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %s, want %s", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestPerformanceMetric_Instantiation tests that PerformanceMetric can be instantiated correctly.
func TestPerformanceMetric_Instantiation(t *testing.T) {
	now := time.Now()
	metric := PerformanceMetric{
		ID:          1,
		MetricType:  MetricTypeInference,
		MetricName:  "tokens_per_second",
		MetricValue: 42.5,
		Metadata:    `{"model": "gpt-4"}`,
		CreatedAt:   now,
	}

	if metric.MetricType != MetricTypeInference {
		t.Errorf("MetricType = %s, want %s", metric.MetricType, MetricTypeInference)
	}
	if metric.MetricValue != 42.5 {
		t.Errorf("MetricValue = %f, want 42.5", metric.MetricValue)
	}
}

// TestPerformanceMetric_JSONMarshal tests that PerformanceMetric can be marshaled to JSON.
func TestPerformanceMetric_JSONMarshal(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	metric := PerformanceMetric{
		ID:          1,
		MetricType:  MetricTypeGPU,
		MetricName:  "vram_utilization",
		MetricValue: 85.5,
		Metadata:    `{"device": 0}`,
		CreatedAt:   now,
	}

	data, err := json.Marshal(metric)
	if err != nil {
		t.Fatalf("Failed to marshal PerformanceMetric: %v", err)
	}

	var decoded PerformanceMetric
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal PerformanceMetric: %v", err)
	}

	if decoded.MetricType != metric.MetricType {
		t.Errorf("MetricType mismatch: got %s, want %s", decoded.MetricType, metric.MetricType)
	}
	if decoded.MetricValue != metric.MetricValue {
		t.Errorf("MetricValue mismatch: got %f, want %f", decoded.MetricValue, metric.MetricValue)
	}
}

// TestPerformanceMetric_Constants tests the metric type constants.
func TestPerformanceMetric_Constants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"MetricTypeInference", MetricTypeInference, "inference"},
		{"MetricTypeGPU", MetricTypeGPU, "gpu"},
		{"MetricTypeSystem", MetricTypeSystem, "system"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %s, want %s", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestErrorLog_Instantiation tests that ErrorLog can be instantiated correctly.
func TestErrorLog_Instantiation(t *testing.T) {
	now := time.Now()
	errLog := ErrorLog{
		ID:            1,
		CorrelationID: "corr-123",
		ErrorType:     ErrorTypeAIInference,
		ErrorMessage:  "Model timeout",
		StackTrace:    "main.go:42\nhandlers.go:150",
		Context:       `{"model": "gpt-4", "timeout": 30}`,
		CreatedAt:     now,
	}

	if errLog.ErrorType != ErrorTypeAIInference {
		t.Errorf("ErrorType = %s, want %s", errLog.ErrorType, ErrorTypeAIInference)
	}
	if errLog.ErrorMessage != "Model timeout" {
		t.Errorf("ErrorMessage = %s, want Model timeout", errLog.ErrorMessage)
	}
}

// TestErrorLog_JSONMarshal tests that ErrorLog can be marshaled to JSON.
func TestErrorLog_JSONMarshal(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	errLog := ErrorLog{
		ID:            1,
		CorrelationID: "corr-456",
		ErrorType:     ErrorTypeCanvasAPI,
		ErrorMessage:  "API rate limit exceeded",
		Context:       `{"endpoint": "/api/widgets"}`,
		CreatedAt:     now,
	}

	data, err := json.Marshal(errLog)
	if err != nil {
		t.Fatalf("Failed to marshal ErrorLog: %v", err)
	}

	var decoded ErrorLog
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal ErrorLog: %v", err)
	}

	if decoded.ErrorType != errLog.ErrorType {
		t.Errorf("ErrorType mismatch: got %s, want %s", decoded.ErrorType, errLog.ErrorType)
	}
	if decoded.ErrorMessage != errLog.ErrorMessage {
		t.Errorf("ErrorMessage mismatch: got %s, want %s", decoded.ErrorMessage, errLog.ErrorMessage)
	}
}

// TestErrorLog_Constants tests the error type constants.
func TestErrorLog_Constants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"ErrorTypeAIInference", ErrorTypeAIInference, "ai_inference"},
		{"ErrorTypeCanvasAPI", ErrorTypeCanvasAPI, "canvas_api"},
		{"ErrorTypeDatabase", ErrorTypeDatabase, "database"},
		{"ErrorTypeSystem", ErrorTypeSystem, "system"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %s, want %s", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

// TestErrorLog_OmitEmpty tests that optional fields are omitted when empty.
func TestErrorLog_OmitEmpty(t *testing.T) {
	errLog := ErrorLog{
		ID:           1,
		ErrorType:    ErrorTypeSystem,
		ErrorMessage: "System error",
		CreatedAt:    time.Now(),
	}

	data, err := json.Marshal(errLog)
	if err != nil {
		t.Fatalf("Failed to marshal ErrorLog: %v", err)
	}

	// Check that optional fields are omitted
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Failed to unmarshal to map: %v", err)
	}

	if _, exists := raw["correlation_id"]; exists {
		t.Error("Expected correlation_id to be omitted when empty")
	}
	if _, exists := raw["stack_trace"]; exists {
		t.Error("Expected stack_trace to be omitted when empty")
	}
	if _, exists := raw["context"]; exists {
		t.Error("Expected context to be omitted when empty")
	}
}
