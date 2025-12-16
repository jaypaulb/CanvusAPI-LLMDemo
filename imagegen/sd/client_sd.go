//go:build sd && cgo
// +build sd,cgo

// Package sd provides Stable Diffusion image generation.
//
// This file contains the real CGo-based implementation that uses
// stable-diffusion.cpp for CUDA-accelerated image generation.
//
// NOTE: This is a stub file showing the intended integration pattern.
// Full implementation requires completing the sdruntime package integration.
//
// Build requirements:
//   - NVIDIA GPU with CUDA support
//   - CUDA Toolkit 11.8+
//   - stable-diffusion.cpp compiled as shared library in lib/
//   - CGO_ENABLED=1
//   - Build tag: -tags sd
//
// Build steps:
//  1. cd deps/stable-diffusion.cpp && ./build-linux.sh  # or build-windows.ps1
//  2. CGO_ENABLED=1 go build -tags sd
package sd

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"go_backend/sdruntime"
)

// NewClient creates a new Stable Diffusion client with CUDA support.
//
// This implementation uses sdruntime (CGo bindings to stable-diffusion.cpp)
// for actual image generation.
func NewClient(config ClientConfig) (Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Check if model file exists
	if config.ModelPath != "" {
		if _, err := os.Stat(config.ModelPath); os.IsNotExist(err) {
			return nil, NewGenerationError(
				ErrCodeModelNotFound,
				fmt.Sprintf("model not found: %s", config.ModelPath),
				false,
				err,
			)
		}
	}

	// Determine number of threads
	numThreads := config.NumThreads
	if numThreads <= 0 {
		numThreads = runtime.NumCPU()
	}

	// Create context pool using existing sdruntime API
	pool, err := sdruntime.NewContextPool(config.MaxConcurrent, config.ModelPath)
	if err != nil {
		return nil, NewGenerationError(
			ErrCodeModelLoadFailed,
			"failed to create SD context pool",
			false,
			err,
		)
	}

	client := &sdClient{
		config:            config,
		pool:              pool,
		numThreads:        numThreads,
		acquireTimeout:    time.Duration(config.AcquireTimeout) * time.Second,
		generationTimeout: time.Duration(config.GenerationTimeout) * time.Second,
	}

	return client, nil
}

// NewClientFromPath creates a client using the model at the given path.
func NewClientFromPath(modelPath string) (Client, error) {
	config := DefaultClientConfig()
	config.ModelPath = modelPath
	return NewClient(config)
}

// IsCUDAAvailable returns whether CUDA is available for image generation.
func IsCUDAAvailable() bool {
	// Check via sdruntime's backend info
	info := sdruntime.GetBackendInfo()
	return info != "stub" && info != ""
}

// sdClient is the real Client implementation using stable-diffusion.cpp.
type sdClient struct {
	config            ClientConfig
	pool              *sdruntime.ContextPool
	numThreads        int
	acquireTimeout    time.Duration
	generationTimeout time.Duration

	mu      sync.RWMutex
	closed  bool
	metrics ClientMetrics
}

// Generate creates an image from a text prompt using CUDA.
func (c *sdClient) Generate(ctx context.Context, request GenerationRequest) (*GenerationResponse, error) {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return nil, NewGenerationError(
			ErrCodeGenerationFailed,
			"client is closed",
			false,
			nil,
		)
	}
	c.mu.RUnlock()

	// Validate request
	if err := request.Validate(); err != nil {
		c.recordFailure()
		return nil, NewGenerationError(
			ErrCodeInvalidRequest,
			err.Error(),
			false,
			err,
		)
	}

	// Create timeout context for acquisition
	acquireCtx, acquireCancel := context.WithTimeout(ctx, c.acquireTimeout)
	defer acquireCancel()

	// Acquire context from pool
	sdCtx, err := c.pool.Acquire(acquireCtx)
	if err != nil {
		c.recordFailure()
		return nil, NewGenerationError(
			ErrCodeTimeout,
			"timeout waiting for generation slot",
			true,
			err,
		)
	}
	defer c.pool.Release(sdCtx)

	// Convert request to sdruntime params
	params := sdruntime.GenerateParams{
		Prompt:         request.Prompt,
		NegativePrompt: request.NegativePrompt,
		Width:          request.Width,
		Height:         request.Height,
		Steps:          request.Steps,
		CFGScale:       request.CFGScale,
		Seed:           request.Seed,
	}

	// Create timeout context for generation
	genCtx, genCancel := context.WithTimeout(ctx, c.generationTimeout)
	defer genCancel()

	// Generate image
	start := time.Now()

	// Use the SDContext from the pool to generate
	result, err := sdruntime.GenerateImage(sdCtx.SDContext, params)
	duration := time.Since(start)

	// Check for context cancellation
	select {
	case <-genCtx.Done():
		c.recordFailure()
		return nil, NewGenerationError(ErrCodeTimeout, "generation timed out", true, genCtx.Err())
	default:
	}

	if err != nil {
		c.recordFailure()
		// Translate error messages
		errMsg := err.Error()
		if contains(errMsg, "out of memory") || contains(errMsg, "VRAM") {
			return nil, NewGenerationError(ErrCodeOutOfMemory, errMsg, true, err)
		}
		return nil, NewGenerationError(ErrCodeGenerationFailed, errMsg, false, err)
	}

	c.recordSuccess(duration)

	// Build response
	response := &GenerationResponse{
		ImageData: result.ImageData,
		Width:     result.Width,
		Height:    result.Height,
		Format:    FormatPNG,
		Seed:      result.Seed,
		Duration:  duration,
		Request:   &request,
	}

	return response, nil
}

// contains checks if s contains substr (simple helper to avoid strings import).
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// IsReady returns true if the client has a loaded model and available backend.
func (c *sdClient) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return false
	}

	return c.pool != nil && !c.pool.IsClosed()
}

// GetModelInfo returns information about the loaded model.
func (c *sdClient) GetModelInfo() *ModelInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed || c.pool == nil {
		return nil
	}

	modelPath := c.pool.ModelPath()
	if modelPath == "" {
		return nil
	}

	// Get file info if available
	var size int64
	if info, err := os.Stat(modelPath); err == nil {
		size = info.Size()
	}

	return &ModelInfo{
		Path:   modelPath,
		Name:   "Stable Diffusion",
		Size:   size,
		Loaded: c.pool.Created() > 0,
	}
}

// GetBackendInfo returns information about the compute backend.
func (c *sdClient) GetBackendInfo() BackendInfo {
	info := sdruntime.GetBackendInfo()
	return BackendInfo{
		Name:      info,
		Available: info != "stub" && info != "",
	}
}

// Close releases all resources held by the client.
func (c *sdClient) Close() error {
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

// recordSuccess updates metrics for a successful generation.
func (c *sdClient) recordSuccess(duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics.TotalGenerations++
	c.metrics.SuccessfulGenerations++
	c.metrics.TotalDuration += duration.Milliseconds()

	if c.metrics.SuccessfulGenerations > 0 {
		c.metrics.AverageDuration = c.metrics.TotalDuration / c.metrics.SuccessfulGenerations
	}
}

// recordFailure updates metrics for a failed generation.
func (c *sdClient) recordFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.metrics.TotalGenerations++
	c.metrics.FailedGenerations++
}

// GetMetrics returns generation statistics.
func (c *sdClient) GetMetrics() ClientMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	metrics := c.metrics
	if c.pool != nil {
		metrics.QueueDepth = c.pool.Size()
		metrics.ActiveGenerations = c.pool.Created() - c.pool.Size()
	}

	return metrics
}

// Ensure sdClient implements Client and ClientWithMetrics.
var _ Client = (*sdClient)(nil)
var _ ClientWithMetrics = (*sdClient)(nil)

// GetBuildInfo returns information about how this package was built.
func GetBuildInfo() string {
	backend := sdruntime.GetBackendInfo()
	return fmt.Sprintf(
		"imagegen/sd: CUDA build enabled\n"+
			"Backend: %s\n"+
			"Build: CGo with stable-diffusion.cpp",
		backend,
	)
}
