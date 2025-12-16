package core

import (
	"time"

	"go.uber.org/zap/zapcore"
)

// GPUMetrics represents GPU resource utilization metrics.
// Implements zapcore.ObjectMarshaler for structured logging.
type GPUMetrics struct {
	// VRAMUsedMB is the amount of VRAM currently in use (megabytes)
	VRAMUsedMB int64 `json:"vram_used_mb"`
	// VRAMTotalMB is the total available VRAM (megabytes)
	VRAMTotalMB int64 `json:"vram_total_mb"`
	// GPUUtilization is the GPU utilization percentage (0-100)
	GPUUtilization float64 `json:"gpu_utilization"`
	// Temperature is the GPU temperature in Celsius
	Temperature float64 `json:"temperature"`
}

// MarshalLogObject implements zapcore.ObjectMarshaler for structured logging.
func (g GPUMetrics) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt64("vram_used_mb", g.VRAMUsedMB)
	enc.AddInt64("vram_total_mb", g.VRAMTotalMB)
	enc.AddFloat64("gpu_utilization", g.GPUUtilization)
	enc.AddFloat64("temperature", g.Temperature)
	return nil
}

// ProcessingHistory represents a record of AI processing operations.
// Used for tracking and auditing AI inference requests.
type ProcessingHistory struct {
	// ID is the unique identifier for this record
	ID int64 `json:"id"`
	// CorrelationID links related operations together
	CorrelationID string `json:"correlation_id"`
	// CanvasID identifies the canvas where processing occurred
	CanvasID string `json:"canvas_id"`
	// WidgetID identifies the specific widget processed
	WidgetID string `json:"widget_id"`
	// OperationType describes the type of operation (e.g., "note", "pdf", "image", "canvas_analysis")
	OperationType string `json:"operation_type"`
	// Prompt is the input prompt sent to the AI model
	Prompt string `json:"prompt"`
	// Response is the AI model's response
	Response string `json:"response"`
	// ModelName identifies which AI model was used
	ModelName string `json:"model_name"`
	// InputTokens is the count of tokens in the input
	InputTokens int64 `json:"input_tokens"`
	// OutputTokens is the count of tokens in the output
	OutputTokens int64 `json:"output_tokens"`
	// DurationMs is the processing duration in milliseconds
	DurationMs int64 `json:"duration_ms"`
	// Status indicates the result status (e.g., "success", "error", "timeout")
	Status string `json:"status"`
	// ErrorMessage contains error details if Status is not "success"
	ErrorMessage string `json:"error_message,omitempty"`
	// CreatedAt is when the record was created
	CreatedAt time.Time `json:"created_at"`
}

// CanvasEvent represents an event that occurred on a canvas widget.
// Used for tracking canvas activity and changes.
type CanvasEvent struct {
	// ID is the unique identifier for this event
	ID int64 `json:"id"`
	// CanvasID identifies the canvas where the event occurred
	CanvasID string `json:"canvas_id"`
	// WidgetID identifies the widget involved in the event
	WidgetID string `json:"widget_id"`
	// EventType describes what happened (e.g., "created", "updated", "deleted")
	EventType string `json:"event_type"`
	// WidgetType identifies the type of widget (e.g., "note", "image", "pdf")
	WidgetType string `json:"widget_type"`
	// ContentPreview is a truncated preview of the widget content
	ContentPreview string `json:"content_preview,omitempty"`
	// CreatedAt is when the event occurred
	CreatedAt time.Time `json:"created_at"`
}

// EventType constants for CanvasEvent
const (
	EventTypeCreated = "created"
	EventTypeUpdated = "updated"
	EventTypeDeleted = "deleted"
)

// WidgetType constants for CanvasEvent
const (
	WidgetTypeNote  = "note"
	WidgetTypeImage = "image"
	WidgetTypePDF   = "pdf"
)

// PerformanceMetric represents a single performance measurement.
// Used for tracking system and inference performance over time.
type PerformanceMetric struct {
	// ID is the unique identifier for this metric
	ID int64 `json:"id"`
	// MetricType categorizes the metric (e.g., "inference", "gpu", "system")
	MetricType string `json:"metric_type"`
	// MetricName is the specific metric being measured
	MetricName string `json:"metric_name"`
	// MetricValue is the measured value
	MetricValue float64 `json:"metric_value"`
	// Metadata is optional JSON-encoded additional context
	Metadata string `json:"metadata,omitempty"`
	// CreatedAt is when the metric was recorded
	CreatedAt time.Time `json:"created_at"`
}

// MetricType constants for PerformanceMetric
const (
	MetricTypeInference = "inference"
	MetricTypeGPU       = "gpu"
	MetricTypeSystem    = "system"
)

// ErrorLog represents a logged error with context.
// Used for error tracking and debugging.
type ErrorLog struct {
	// ID is the unique identifier for this error log
	ID int64 `json:"id"`
	// CorrelationID links this error to related operations
	CorrelationID string `json:"correlation_id,omitempty"`
	// ErrorType categorizes the error (e.g., "ai_inference", "canvas_api", "database", "system")
	ErrorType string `json:"error_type"`
	// ErrorMessage is the error description
	ErrorMessage string `json:"error_message"`
	// StackTrace contains the stack trace if available
	StackTrace string `json:"stack_trace,omitempty"`
	// Context is optional JSON-encoded additional context
	Context string `json:"context,omitempty"`
	// CreatedAt is when the error occurred
	CreatedAt time.Time `json:"created_at"`
}

// ErrorType constants for ErrorLog
const (
	ErrorTypeAIInference = "ai_inference"
	ErrorTypeCanvasAPI   = "canvas_api"
	ErrorTypeDatabase    = "database"
	ErrorTypeSystem      = "system"
)
