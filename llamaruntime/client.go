// Package llamaruntime provides Go bindings to llama.cpp for local LLM inference.
// This file contains the Client molecule - the high-level Go API for inference.
//
// The Client provides a simple, high-level interface for loading models and
// performing inference. It manages the context pool internally and handles
// all resource lifecycle management.
//
// Architecture:
// - Client owns a ContextPool which manages multiple inference contexts
// - Thread-safe: multiple goroutines can call Infer/InferVision concurrently
// - Resource management: proper cleanup via Close() method
//
// Example usage:
//
//	config := llamaruntime.DefaultClientConfig()
//	config.ModelPath = "models/bunny-v1.1.gguf"
//
//	client, err := llamaruntime.NewClient(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Close()
//
//	// Text inference
//	params := llamaruntime.DefaultInferenceParams()
//	params.Prompt = "What is the capital of France?"
//	result, err := client.Infer(context.Background(), params)
//
//	// Vision inference (with Bunny model)
//	visionParams := llamaruntime.DefaultVisionParams()
//	visionParams.Prompt = "Describe this image"
//	visionParams.ImagePath = "image.jpg"
//	result, err := client.InferVision(context.Background(), visionParams)
package llamaruntime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

// =============================================================================
// Client Configuration
// =============================================================================

// ClientConfig contains configuration for the Client.
type ClientConfig struct {
	// ModelPath is the path to the GGUF model file.
	// Required - no default.
	ModelPath string

	// NumContexts is the number of inference contexts to maintain in the pool.
	// This determines maximum concurrent inference operations.
	// Defaults to 3.
	NumContexts int

	// ContextSize is the context window size in tokens.
	// Defaults to DefaultContextSize.
	ContextSize int

	// BatchSize is the batch size for inference.
	// Defaults to DefaultBatchSize.
	BatchSize int

	// NumGPULayers is the number of model layers to offload to GPU.
	// -1 means all layers (recommended). Defaults to DefaultNumGPULayers.
	NumGPULayers int

	// NumThreads is the number of CPU threads for inference.
	// Defaults to DefaultNumThreads.
	NumThreads int

	// UseMMap enables memory-mapped model loading.
	// Defaults to true.
	UseMMap bool

	// UseMlock enables memory locking to prevent swapping.
	// Defaults to false.
	UseMlock bool

	// AcquireTimeout is the maximum time to wait for a context.
	// Defaults to 30 seconds.
	AcquireTimeout time.Duration

	// VerboseLogging enables verbose llama.cpp logging.
	// Defaults to false.
	VerboseLogging bool
}

// DefaultClientConfig returns a ClientConfig with sensible defaults.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		NumContexts:    3,
		ContextSize:    DefaultContextSize,
		BatchSize:      DefaultBatchSize,
		NumGPULayers:   DefaultNumGPULayers,
		NumThreads:     DefaultNumThreads,
		UseMMap:        true,
		UseMlock:       false,
		AcquireTimeout: 30 * time.Second,
		VerboseLogging: false,
	}
}

// =============================================================================
// Client
// =============================================================================

// Client provides a high-level API for llama.cpp inference.
// It manages a pool of inference contexts for concurrent operations.
//
// Thread Safety:
// - All public methods are thread-safe
// - Multiple goroutines can call Infer/InferVision concurrently
// - Concurrency is bounded by NumContexts in the configuration
type Client struct {
	pool      *ContextPool
	config    ClientConfig
	modelInfo *ModelInfo
	mu        sync.RWMutex
	closed    bool

	// Statistics
	startTime         time.Time
	totalInferences   int64
	totalVisionInfers int64
	totalTokensGen    int64
	totalTokensPrompt int64
	totalDuration     int64 // nanoseconds
	errorCount        int64
	lastInference     time.Time
	lastInferenceMu   sync.RWMutex
}

// NewClient creates a new Client with the given configuration.
// It loads the model and initializes the context pool.
//
// Returns an error if:
// - ModelPath is empty or file doesn't exist
// - Model loading fails
// - Context pool creation fails
func NewClient(config ClientConfig) (*Client, error) {
	// Validate model path
	if config.ModelPath == "" {
		return nil, &LlamaError{
			Op:      "NewClient",
			Code:    -1,
			Message: "ModelPath is required",
			Err:     ErrModelNotFound,
		}
	}

	// Check if model file exists
	absPath, err := filepath.Abs(config.ModelPath)
	if err != nil {
		return nil, &LlamaError{
			Op:      "NewClient",
			Code:    -1,
			Message: fmt.Sprintf("invalid model path: %s", config.ModelPath),
			Err:     err,
		}
	}

	fileInfo, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &LlamaError{
				Op:      "NewClient",
				Code:    -1,
				Message: fmt.Sprintf("model file not found: %s", absPath),
				Err:     ErrModelNotFound,
			}
		}
		return nil, &LlamaError{
			Op:      "NewClient",
			Code:    -1,
			Message: fmt.Sprintf("cannot access model file: %s", absPath),
			Err:     err,
		}
	}

	// Apply defaults
	if config.NumContexts <= 0 {
		config.NumContexts = 3
	}
	if config.ContextSize <= 0 {
		config.ContextSize = DefaultContextSize
	}
	if config.BatchSize <= 0 {
		config.BatchSize = DefaultBatchSize
	}
	if config.NumGPULayers == 0 {
		config.NumGPULayers = DefaultNumGPULayers
	}
	if config.NumThreads <= 0 {
		config.NumThreads = DefaultNumThreads
	}
	if config.AcquireTimeout <= 0 {
		config.AcquireTimeout = 30 * time.Second
	}

	// Create context pool configuration
	poolConfig := ContextPoolConfig{
		ModelPath:      absPath,
		NumContexts:    config.NumContexts,
		ContextSize:    config.ContextSize,
		BatchSize:      config.BatchSize,
		NumGPULayers:   config.NumGPULayers,
		NumThreads:     config.NumThreads,
		UseMMap:        config.UseMMap,
		UseMlock:       config.UseMlock,
		AcquireTimeout: config.AcquireTimeout,
	}

	// Create context pool
	pool, err := NewContextPool(poolConfig)
	if err != nil {
		return nil, &LlamaError{
			Op:      "NewClient",
			Code:    -1,
			Message: "failed to create context pool",
			Err:     err,
		}
	}

	// Extract model info
	modelInfo := &ModelInfo{
		Path:          absPath,
		Name:          filepath.Base(absPath),
		Size:          fileInfo.Size(),
		Format:        "GGUF",
		ContextLength: config.ContextSize,
		LoadedAt:      time.Now(),
	}

	return &Client{
		pool:      pool,
		config:    config,
		modelInfo: modelInfo,
		startTime: time.Now(),
	}, nil
}

// Infer performs text inference with the given parameters.
// It acquires a context from the pool, runs inference, and releases the context.
//
// The ctx parameter controls cancellation and timeout. If no timeout is set,
// the AcquireTimeout from configuration is used for context acquisition,
// and the params.Timeout is used for inference.
//
// Returns InferenceResult containing the generated text and statistics.
//
// Thread-safe: multiple goroutines can call Infer concurrently.
func (c *Client) Infer(ctx context.Context, params InferenceParams) (*InferenceResult, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, &LlamaError{
			Op:      "Infer",
			Code:    -1,
			Message: "client is closed",
		}
	}
	c.mu.RUnlock()

	// Apply defaults
	if params.MaxTokens <= 0 {
		params.MaxTokens = DefaultMaxTokens
	}
	if params.Temperature <= 0 {
		params.Temperature = DefaultTemperature
	}
	if params.TopP <= 0 {
		params.TopP = DefaultTopP
	}
	if params.TopK <= 0 {
		params.TopK = DefaultTopK
	}
	if params.RepeatPenalty <= 0 {
		params.RepeatPenalty = DefaultRepeatPenalty
	}
	if params.Timeout <= 0 {
		params.Timeout = DefaultTimeout
	}

	// Acquire context from pool
	llamaCtx, err := c.pool.Acquire(ctx)
	if err != nil {
		atomic.AddInt64(&c.errorCount, 1)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return nil, &LlamaError{
				Op:      "Infer",
				Code:    -1,
				Message: "timeout waiting for inference context",
				Err:     ErrTimeout,
			}
		}
		return nil, &LlamaError{
			Op:      "Infer",
			Code:    -1,
			Message: "failed to acquire context",
			Err:     err,
		}
	}
	defer c.pool.Release(llamaCtx)

	// Create inference context with timeout
	inferCtx, cancel := context.WithTimeout(ctx, params.Timeout)
	defer cancel()

	// Run inference
	startTime := time.Now()

	samplingParams := SamplingParams{
		Temperature:   params.Temperature,
		TopK:          params.TopK,
		TopP:          params.TopP,
		RepeatPenalty: params.RepeatPenalty,
	}

	text, err := inferText(inferCtx, llamaCtx, params.Prompt, params.MaxTokens, samplingParams)
	if err != nil {
		atomic.AddInt64(&c.errorCount, 1)
		return nil, &LlamaError{
			Op:      "Infer",
			Code:    -1,
			Message: "inference failed",
			Err:     err,
		}
	}

	duration := time.Since(startTime)

	// Estimate token counts (approximation - actual counts from llama.cpp would be better)
	tokensPrompt := len(params.Prompt) / 4 // ~4 chars per token
	tokensGenerated := len(text) / 4       // ~4 chars per token
	if tokensGenerated < 1 {
		tokensGenerated = 1
	}

	tokensPerSecond := float64(tokensGenerated) / duration.Seconds()
	if duration.Seconds() < 0.001 {
		tokensPerSecond = 0
	}

	// Update statistics
	atomic.AddInt64(&c.totalInferences, 1)
	atomic.AddInt64(&c.totalTokensGen, int64(tokensGenerated))
	atomic.AddInt64(&c.totalTokensPrompt, int64(tokensPrompt))
	atomic.AddInt64(&c.totalDuration, int64(duration))

	c.lastInferenceMu.Lock()
	c.lastInference = time.Now()
	c.lastInferenceMu.Unlock()

	return &InferenceResult{
		Text:            text,
		TokensGenerated: tokensGenerated,
		TokensPrompt:    tokensPrompt,
		Duration:        duration,
		TokensPerSecond: tokensPerSecond,
		StopReason:      determineStopReason(text, params),
	}, nil
}

// InferVision performs vision (multimodal) inference with the given parameters.
// This is designed for use with models like Bunny that support image input.
//
// Either ImageData (raw bytes) or ImagePath must be provided.
// If both are provided, ImageData takes precedence.
//
// Thread-safe: multiple goroutines can call InferVision concurrently.
func (c *Client) InferVision(ctx context.Context, params VisionParams) (*InferenceResult, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, &LlamaError{
			Op:      "InferVision",
			Code:    -1,
			Message: "client is closed",
		}
	}
	c.mu.RUnlock()

	// Validate image input
	imageData := params.ImageData
	if len(imageData) == 0 {
		if params.ImagePath == "" {
			return nil, &LlamaError{
				Op:      "InferVision",
				Code:    -1,
				Message: "either ImageData or ImagePath is required",
				Err:     ErrInvalidImage,
			}
		}

		// Load image from file
		data, err := os.ReadFile(params.ImagePath)
		if err != nil {
			return nil, &LlamaError{
				Op:      "InferVision",
				Code:    -1,
				Message: fmt.Sprintf("failed to read image file: %s", params.ImagePath),
				Err:     err,
			}
		}
		imageData = data
	}

	// Apply defaults
	if params.MaxTokens <= 0 {
		params.MaxTokens = DefaultMaxTokens
	}
	if params.Temperature <= 0 {
		params.Temperature = DefaultTemperature
	}
	if params.Timeout <= 0 {
		params.Timeout = DefaultTimeout
	}

	// Acquire context from pool
	llamaCtx, err := c.pool.Acquire(ctx)
	if err != nil {
		atomic.AddInt64(&c.errorCount, 1)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return nil, &LlamaError{
				Op:      "InferVision",
				Code:    -1,
				Message: "timeout waiting for inference context",
				Err:     ErrTimeout,
			}
		}
		return nil, &LlamaError{
			Op:      "InferVision",
			Code:    -1,
			Message: "failed to acquire context",
			Err:     err,
		}
	}
	defer c.pool.Release(llamaCtx)

	// Create inference context with timeout
	inferCtx, cancel := context.WithTimeout(ctx, params.Timeout)
	defer cancel()

	// Run vision inference
	startTime := time.Now()

	samplingParams := SamplingParams{
		Temperature:   params.Temperature,
		TopK:          DefaultTopK,
		TopP:          DefaultTopP,
		RepeatPenalty: DefaultRepeatPenalty,
	}

	text, err := inferVision(inferCtx, llamaCtx, params.Prompt, imageData, params.MaxTokens, samplingParams)
	if err != nil {
		atomic.AddInt64(&c.errorCount, 1)
		return nil, &LlamaError{
			Op:      "InferVision",
			Code:    -1,
			Message: "vision inference failed",
			Err:     err,
		}
	}

	duration := time.Since(startTime)

	// Estimate token counts
	tokensPrompt := len(params.Prompt) / 4
	tokensGenerated := len(text) / 4
	if tokensGenerated < 1 {
		tokensGenerated = 1
	}

	tokensPerSecond := float64(tokensGenerated) / duration.Seconds()
	if duration.Seconds() < 0.001 {
		tokensPerSecond = 0
	}

	// Update statistics
	atomic.AddInt64(&c.totalInferences, 1)
	atomic.AddInt64(&c.totalVisionInfers, 1)
	atomic.AddInt64(&c.totalTokensGen, int64(tokensGenerated))
	atomic.AddInt64(&c.totalTokensPrompt, int64(tokensPrompt))
	atomic.AddInt64(&c.totalDuration, int64(duration))

	c.lastInferenceMu.Lock()
	c.lastInference = time.Now()
	c.lastInferenceMu.Unlock()

	return &InferenceResult{
		Text:            text,
		TokensGenerated: tokensGenerated,
		TokensPrompt:    tokensPrompt,
		Duration:        duration,
		TokensPerSecond: tokensPerSecond,
		StopReason:      "eos",
	}, nil
}

// GetGPUMemoryUsage returns current GPU memory usage.
// Returns nil if GPU is not available.
func (c *Client) GetGPUMemoryUsage() (*GPUMemoryInfo, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, &LlamaError{
			Op:      "GetGPUMemoryUsage",
			Code:    -1,
			Message: "client is closed",
		}
	}
	c.mu.RUnlock()

	return getGPUMemory()
}

// HealthCheck returns the current health status of the client.
// This includes model info, GPU status, and inference statistics.
func (c *Client) HealthCheck() (*HealthStatus, error) {
	c.mu.RLock()
	closed := c.closed
	c.mu.RUnlock()

	status := &HealthStatus{
		CheckedAt: time.Now(),
		Uptime:    time.Since(c.startTime),
	}

	if closed {
		status.Healthy = false
		status.Status = "client closed"
		return status, nil
	}

	// Get pool stats
	poolStats := c.pool.Stats()

	// Get GPU memory
	gpuMem, err := getGPUMemory()
	if err == nil && gpuMem != nil {
		status.GPUStatus = &GPUStatus{
			Available:   true,
			GPUCount:    1,
			FreeMemory:  gpuMem.Free,
			TotalMemory: gpuMem.Total,
			LastChecked: gpuMem.LastUpdate,
			GPUs: []GPUInfo{{
				TotalMemory: gpuMem.Total,
				FreeMemory:  gpuMem.Free,
				UsedMemory:  gpuMem.Used,
				IsAvailable: true,
			}},
		}
	}

	// Build statistics
	c.lastInferenceMu.RLock()
	lastInfer := c.lastInference
	c.lastInferenceMu.RUnlock()

	totalInferences := atomic.LoadInt64(&c.totalInferences)
	totalTokensGen := atomic.LoadInt64(&c.totalTokensGen)
	totalDuration := time.Duration(atomic.LoadInt64(&c.totalDuration))
	errorCount := atomic.LoadInt64(&c.errorCount)

	avgTokensPerSecond := float64(0)
	if totalDuration > 0 {
		avgTokensPerSecond = float64(totalTokensGen) / totalDuration.Seconds()
	}

	status.Stats = &InferenceStats{
		TotalInferences:        totalInferences,
		TotalTokensGenerated:   totalTokensGen,
		TotalTokensPrompt:      atomic.LoadInt64(&c.totalTokensPrompt),
		TotalDuration:          totalDuration,
		AverageTokensPerSecond: avgTokensPerSecond,
		ErrorCount:             errorCount,
	}

	status.ModelLoaded = c.modelInfo != nil
	status.ModelInfo = c.modelInfo
	status.LastInference = lastInfer

	// Determine overall health
	status.Healthy = true
	if poolStats.TotalAcquires > 0 && float64(poolStats.AcquireTimeouts)/float64(poolStats.TotalAcquires) > 0.1 {
		status.Healthy = false
		status.Status = "high timeout rate - pool may be undersized"
	} else if errorCount > 0 && float64(errorCount)/float64(totalInferences+1) > 0.2 {
		status.Healthy = false
		status.Status = "high error rate"
	} else {
		status.Status = "healthy"
	}

	return status, nil
}

// ModelInfo returns information about the loaded model.
func (c *Client) ModelInfo() *ModelInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.modelInfo
}

// Stats returns inference statistics.
func (c *Client) Stats() InferenceStats {
	totalInferences := atomic.LoadInt64(&c.totalInferences)
	totalTokensGen := atomic.LoadInt64(&c.totalTokensGen)
	totalDuration := time.Duration(atomic.LoadInt64(&c.totalDuration))

	avgTokensPerSecond := float64(0)
	if totalDuration > 0 {
		avgTokensPerSecond = float64(totalTokensGen) / totalDuration.Seconds()
	}

	return InferenceStats{
		TotalInferences:        totalInferences,
		TotalTokensGenerated:   totalTokensGen,
		TotalTokensPrompt:      atomic.LoadInt64(&c.totalTokensPrompt),
		TotalDuration:          totalDuration,
		AverageTokensPerSecond: avgTokensPerSecond,
		ErrorCount:             atomic.LoadInt64(&c.errorCount),
	}
}

// Close releases all resources held by the client.
// After Close, no inference operations can be performed.
//
// Close is safe to call multiple times (idempotent).
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	if c.pool != nil {
		return c.pool.Close()
	}

	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// determineStopReason determines why generation stopped.
func determineStopReason(text string, params InferenceParams) string {
	// Check for stop sequences
	for _, seq := range params.StopSequences {
		if len(text) >= len(seq) && text[len(text)-len(seq):] == seq {
			return "stop_sequence"
		}
	}

	// Check if we hit max tokens (rough estimate)
	tokenCount := len(text) / 4
	if tokenCount >= params.MaxTokens-1 {
		return "max_tokens"
	}

	return "eos"
}

// =============================================================================
// Convenience Functions
// =============================================================================

// QuickInfer is a convenience function for simple one-shot inference.
// It creates a client, runs inference, and closes the client.
//
// This is useful for scripts and one-off operations, but not recommended
// for production use where you want to reuse the client.
func QuickInfer(modelPath, prompt string) (string, error) {
	config := DefaultClientConfig()
	config.ModelPath = modelPath
	config.NumContexts = 1

	client, err := NewClient(config)
	if err != nil {
		return "", err
	}
	defer client.Close()

	params := DefaultInferenceParams()
	params.Prompt = prompt

	result, err := client.Infer(context.Background(), params)
	if err != nil {
		return "", err
	}

	return result.Text, nil
}

// InferWithTimeout is a convenience wrapper that adds a timeout to inference.
func (c *Client) InferWithTimeout(timeout time.Duration, params InferenceParams) (*InferenceResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.Infer(ctx, params)
}

// InferStream is a placeholder for future streaming inference support.
// Currently, it just calls Infer and returns the full result.
//
// In the future, this will return a channel that yields tokens as they're generated.
func (c *Client) InferStream(ctx context.Context, params InferenceParams, tokenChan chan<- string) (*InferenceResult, error) {
	result, err := c.Infer(ctx, params)
	if err != nil {
		close(tokenChan)
		return nil, err
	}

	// For now, just send the full text as a single token
	tokenChan <- result.Text
	close(tokenChan)

	return result, nil
}

// =============================================================================
// io.Closer Interface
// =============================================================================

// Ensure Client implements io.Closer
var _ io.Closer = (*Client)(nil)
