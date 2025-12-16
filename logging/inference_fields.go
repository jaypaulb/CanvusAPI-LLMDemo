// Package logging provides structured logging utilities for the CanvusLocalLLM application.
// This file contains molecule-level helper functions that compose InferenceMetrics
// and GPUMetrics atoms into convenient zap.Field helpers for structured logging.
package logging

import (
	"time"

	"go.uber.org/zap"
	"go_backend/core"
)

// InferenceFields creates a structured zap field from inference metrics.
// This is a molecule that composes the InferenceMetrics atom into a ready-to-use zap.Field.
//
// Example:
//
//	metrics := logging.InferenceMetrics{
//		ModelName:        "llama-3.2-8b",
//		PromptTokens:     150,
//		CompletionTokens: 200,
//		TotalTokens:      350,
//		Duration:         2 * time.Second,
//		TokensPerSecond:  100.0,
//	}
//	logger.Info("inference complete", logging.InferenceFields(metrics))
func InferenceFields(metrics InferenceMetrics) zap.Field {
	return zap.Object("inference", metrics)
}

// GPUFields creates a structured zap field from GPU metrics.
// This is a molecule that composes the core.GPUMetrics atom into a ready-to-use zap.Field.
//
// Example:
//
//	gpuMetrics := core.GPUMetrics{
//		VRAMUsedMB:     4096,
//		VRAMTotalMB:    8192,
//		GPUUtilization: 85.5,
//		Temperature:    72.0,
//	}
//	logger.Info("GPU status", logging.GPUFields(gpuMetrics))
func GPUFields(metrics core.GPUMetrics) zap.Field {
	return zap.Object("gpu", metrics)
}

// TokenFields creates a slice of zap fields for token counts.
// This is a convenience function for logging token metrics without a full InferenceMetrics struct.
//
// Example:
//
//	logger.Info("token usage", logging.TokenFields(150, 200, 350)...)
func TokenFields(prompt, completion, total int) []zap.Field {
	return []zap.Field{
		zap.Int("prompt_tokens", prompt),
		zap.Int("completion_tokens", completion),
		zap.Int("total_tokens", total),
	}
}

// TimingFields creates a slice of zap fields for inference timing.
// This is a convenience function for logging timing metrics with automatic duration calculation.
//
// Example:
//
//	start := time.Now()
//	// ... perform inference ...
//	end := time.Now()
//	tokensPerSec := float64(totalTokens) / end.Sub(start).Seconds()
//	logger.Info("timing", logging.TimingFields(start, end, tokensPerSec)...)
func TimingFields(startTime, endTime time.Time, tokensPerSecond float64) []zap.Field {
	return []zap.Field{
		zap.Time("start_time", startTime),
		zap.Time("end_time", endTime),
		zap.Duration("duration", endTime.Sub(startTime)),
		zap.Float64("tokens_per_second", tokensPerSecond),
	}
}
