package logging

import (
	"time"

	"go.uber.org/zap/zapcore"
	"go_backend/core"
)

// InferenceMetrics represents metrics collected during AI inference operations.
// Implements zapcore.ObjectMarshaler for structured logging.
//
// This is a pure data structure with no dependencies on other logging atoms.
// It embeds core.GPUMetrics to capture GPU resource utilization during inference.
//
// Example:
//
//	metrics := InferenceMetrics{
//		ModelName:        "llama-3.2-8b",
//		PromptTokens:     150,
//		CompletionTokens: 200,
//		TotalTokens:      350,
//		Duration:         2 * time.Second,
//		TokensPerSecond:  100.0,
//		GPU: core.GPUMetrics{
//			VRAMUsedMB:     4096,
//			VRAMTotalMB:    8192,
//			GPUUtilization: 85.5,
//			Temperature:    72.0,
//		},
//	}
//	logger.Info("inference complete", zap.Object("metrics", metrics))
type InferenceMetrics struct {
	// ModelName identifies which AI model was used for inference
	ModelName string `json:"model_name"`

	// PromptTokens is the count of tokens in the input prompt
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the count of tokens generated in the response
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the sum of PromptTokens and CompletionTokens
	TotalTokens int `json:"total_tokens"`

	// Duration is the total time taken for the inference operation
	Duration time.Duration `json:"duration"`

	// TokensPerSecond is the throughput rate (TotalTokens / Duration.Seconds())
	TokensPerSecond float64 `json:"tokens_per_second"`

	// GPU contains GPU resource utilization metrics captured during inference
	GPU core.GPUMetrics `json:"gpu"`
}

// MarshalLogObject implements zapcore.ObjectMarshaler for structured logging.
// This allows InferenceMetrics to be logged as a nested JSON object in zap logs.
//
// The method encodes all fields with consistent JSON key names matching the struct tags.
// Duration is encoded in milliseconds for readability.
func (m InferenceMetrics) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("model_name", m.ModelName)
	enc.AddInt("prompt_tokens", m.PromptTokens)
	enc.AddInt("completion_tokens", m.CompletionTokens)
	enc.AddInt("total_tokens", m.TotalTokens)
	enc.AddInt64("duration_ms", m.Duration.Milliseconds())
	enc.AddFloat64("tokens_per_second", m.TokensPerSecond)

	// Embed GPU metrics using its own ObjectMarshaler
	if err := enc.AddObject("gpu", m.GPU); err != nil {
		return err
	}

	return nil
}
