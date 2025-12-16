// Package logging provides structured logging utilities for the CanvusLocalLLM application.
//
// gpu_metrics.go is an organism that provides a unified API for GPU and inference metrics logging.
// It composes:
//   - core.GPUMetrics atom (GPU resource metrics)
//   - InferenceMetrics atom (inference operation metrics)
//   - InferenceFields, GPUFields, TokenFields, TimingFields molecules (zap field helpers)
//
// This organism provides high-level functions for logging AI inference operations
// with automatic GPU metrics capture and structured output.
package logging

import (
	"time"

	"go.uber.org/zap"
	"go_backend/core"
)

// MetricsLogger provides structured logging for AI inference and GPU metrics.
// It wraps a Logger and provides convenience methods for inference logging.
//
// This organism composes:
//   - Logger organism (structured logging with redaction)
//   - InferenceMetrics atom (metrics struct)
//   - InferenceFields molecule (zap field conversion)
//
// Example:
//
//	logger, _ := NewLogger(true, "app.log")
//	ml := NewMetricsLogger(logger)
//
//	metrics := ml.StartInference("llama-3.2-8b")
//	// ... perform inference ...
//	ml.EndInference(metrics, 150, 200)
type MetricsLogger struct {
	logger *Logger
}

// NewMetricsLogger creates a MetricsLogger wrapping the given Logger.
func NewMetricsLogger(logger *Logger) *MetricsLogger {
	return &MetricsLogger{logger: logger}
}

// InferenceTimer tracks timing for an inference operation.
// Use StartInference to create and EndInference to complete.
type InferenceTimer struct {
	ModelName string
	StartTime time.Time
	GPU       core.GPUMetrics
}

// StartInference begins timing an inference operation.
// Call EndInference when the operation completes.
//
// Example:
//
//	timer := ml.StartInference("llama-3.2-8b")
//	// ... perform inference ...
//	ml.EndInference(timer, promptTokens, completionTokens)
func (ml *MetricsLogger) StartInference(modelName string) *InferenceTimer {
	return &InferenceTimer{
		ModelName: modelName,
		StartTime: time.Now(),
	}
}

// StartInferenceWithGPU begins timing with GPU metrics snapshot.
// Use this when you have GPU metrics available at start time.
func (ml *MetricsLogger) StartInferenceWithGPU(modelName string, gpu core.GPUMetrics) *InferenceTimer {
	return &InferenceTimer{
		ModelName: modelName,
		StartTime: time.Now(),
		GPU:       gpu,
	}
}

// EndInference completes timing and logs the inference metrics.
// Returns the completed InferenceMetrics for further use if needed.
//
// Example:
//
//	timer := ml.StartInference("model")
//	// ... inference ...
//	metrics := ml.EndInference(timer, 150, 200)
func (ml *MetricsLogger) EndInference(timer *InferenceTimer, promptTokens, completionTokens int) InferenceMetrics {
	return ml.EndInferenceWithGPU(timer, promptTokens, completionTokens, timer.GPU)
}

// EndInferenceWithGPU completes timing with updated GPU metrics.
// Use this when GPU metrics may have changed during inference.
func (ml *MetricsLogger) EndInferenceWithGPU(timer *InferenceTimer, promptTokens, completionTokens int, gpu core.GPUMetrics) InferenceMetrics {
	endTime := time.Now()
	duration := endTime.Sub(timer.StartTime)
	totalTokens := promptTokens + completionTokens

	// Calculate tokens per second, avoiding division by zero
	var tokensPerSecond float64
	if duration.Seconds() > 0 {
		tokensPerSecond = float64(totalTokens) / duration.Seconds()
	}

	metrics := InferenceMetrics{
		ModelName:        timer.ModelName,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		Duration:         duration,
		TokensPerSecond:  tokensPerSecond,
		GPU:              gpu,
	}

	ml.logger.Info("inference complete", InferenceFields(metrics))
	return metrics
}

// LogInference logs a complete inference operation in a single call.
// Use this when you have all metrics available at once.
//
// Example:
//
//	ml.LogInference("llama-3.2-8b", 150, 200, 2*time.Second, gpu)
func (ml *MetricsLogger) LogInference(modelName string, promptTokens, completionTokens int, duration time.Duration, gpu core.GPUMetrics) InferenceMetrics {
	totalTokens := promptTokens + completionTokens

	var tokensPerSecond float64
	if duration.Seconds() > 0 {
		tokensPerSecond = float64(totalTokens) / duration.Seconds()
	}

	metrics := InferenceMetrics{
		ModelName:        modelName,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		Duration:         duration,
		TokensPerSecond:  tokensPerSecond,
		GPU:              gpu,
	}

	ml.logger.Info("inference complete", InferenceFields(metrics))
	return metrics
}

// LogGPU logs current GPU metrics.
//
// Example:
//
//	ml.LogGPU(core.GPUMetrics{
//	    VRAMUsedMB:     4096,
//	    VRAMTotalMB:    8192,
//	    GPUUtilization: 85.5,
//	    Temperature:    72.0,
//	})
func (ml *MetricsLogger) LogGPU(gpu core.GPUMetrics) {
	ml.logger.Info("GPU metrics", GPUFields(gpu))
}

// LogGPUWarn logs GPU metrics at warn level (for high resource usage).
//
// Example:
//
//	if gpu.GPUUtilization > 90 {
//	    ml.LogGPUWarn(gpu, "high GPU utilization")
//	}
func (ml *MetricsLogger) LogGPUWarn(gpu core.GPUMetrics, msg string) {
	ml.logger.Warn(msg, GPUFields(gpu))
}

// LogTokens logs token counts without full inference metrics.
// Useful for partial operations or streaming.
//
// Example:
//
//	ml.LogTokens("streaming chunk", 0, 50, 50)
func (ml *MetricsLogger) LogTokens(msg string, prompt, completion, total int) {
	ml.logger.Info(msg, TokenFields(prompt, completion, total)...)
}

// Debug logs a debug message with inference context.
func (ml *MetricsLogger) Debug(msg string, fields ...zap.Field) {
	ml.logger.Debug(msg, fields...)
}

// Info logs an info message with inference context.
func (ml *MetricsLogger) Info(msg string, fields ...zap.Field) {
	ml.logger.Info(msg, fields...)
}

// Warn logs a warning message with inference context.
func (ml *MetricsLogger) Warn(msg string, fields ...zap.Field) {
	ml.logger.Warn(msg, fields...)
}

// Error logs an error message with inference context.
func (ml *MetricsLogger) Error(msg string, fields ...zap.Field) {
	ml.logger.Error(msg, fields...)
}

// WithModel returns a MetricsLogger with model name context.
// All subsequent logs will include the model name.
//
// Example:
//
//	llama := ml.WithModel("llama-3.2-8b")
//	llama.Info("loading model")
//	llama.Info("model loaded")
func (ml *MetricsLogger) WithModel(modelName string) *MetricsLogger {
	return &MetricsLogger{
		logger: ml.logger.With(zap.String("model", modelName)),
	}
}

// WithCanvas returns a MetricsLogger with canvas context.
// All subsequent logs will include canvas and widget IDs.
//
// Example:
//
//	canvasLogger := ml.WithCanvas("canvas-123", "widget-456")
//	canvasLogger.Info("processing widget")
func (ml *MetricsLogger) WithCanvas(canvasID, widgetID string) *MetricsLogger {
	return &MetricsLogger{
		logger: ml.logger.With(
			zap.String("canvas_id", canvasID),
			zap.String("widget_id", widgetID),
		),
	}
}

// WithCorrelation returns a MetricsLogger with correlation ID.
// Use for tracing related operations.
//
// Example:
//
//	reqLogger := ml.WithCorrelation("req-abc123")
//	reqLogger.Info("request started")
func (ml *MetricsLogger) WithCorrelation(correlationID string) *MetricsLogger {
	return &MetricsLogger{
		logger: ml.logger.With(zap.String("correlation_id", correlationID)),
	}
}

// Logger returns the underlying Logger for direct access.
func (ml *MetricsLogger) Logger() *Logger {
	return ml.logger
}

// Utility functions for creating metrics without MetricsLogger

// NewInferenceMetrics creates InferenceMetrics with calculated fields.
// This is a convenience function for creating metrics outside of MetricsLogger.
//
// Example:
//
//	metrics := NewInferenceMetrics("llama", 150, 200, 2*time.Second, gpu)
func NewInferenceMetrics(modelName string, promptTokens, completionTokens int, duration time.Duration, gpu core.GPUMetrics) InferenceMetrics {
	totalTokens := promptTokens + completionTokens

	var tokensPerSecond float64
	if duration.Seconds() > 0 {
		tokensPerSecond = float64(totalTokens) / duration.Seconds()
	}

	return InferenceMetrics{
		ModelName:        modelName,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		Duration:         duration,
		TokensPerSecond:  tokensPerSecond,
		GPU:              gpu,
	}
}

// CalculateTokensPerSecond calculates throughput from tokens and duration.
// Returns 0 if duration is zero or negative.
func CalculateTokensPerSecond(totalTokens int, duration time.Duration) float64 {
	if duration.Seconds() <= 0 {
		return 0
	}
	return float64(totalTokens) / duration.Seconds()
}

// VRAMUtilization calculates VRAM usage percentage from GPUMetrics.
// Returns 0 if total VRAM is 0.
func VRAMUtilization(gpu core.GPUMetrics) float64 {
	if gpu.VRAMTotalMB == 0 {
		return 0
	}
	return float64(gpu.VRAMUsedMB) / float64(gpu.VRAMTotalMB) * 100
}

// IsGPUHighUtilization returns true if GPU utilization exceeds threshold.
// Default threshold is 90%.
func IsGPUHighUtilization(gpu core.GPUMetrics, threshold ...float64) bool {
	t := 90.0
	if len(threshold) > 0 {
		t = threshold[0]
	}
	return gpu.GPUUtilization > t
}

// IsVRAMHighUsage returns true if VRAM usage exceeds threshold.
// Default threshold is 90%.
func IsVRAMHighUsage(gpu core.GPUMetrics, threshold ...float64) bool {
	t := 90.0
	if len(threshold) > 0 {
		t = threshold[0]
	}
	return VRAMUtilization(gpu) > t
}

// IsGPUHot returns true if GPU temperature exceeds threshold.
// Default threshold is 80°C.
func IsGPUHot(gpu core.GPUMetrics, threshold ...float64) bool {
	t := 80.0
	if len(threshold) > 0 {
		t = threshold[0]
	}
	return gpu.Temperature > t
}
