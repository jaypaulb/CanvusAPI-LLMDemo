// Package sd provides Stable Diffusion image generation interfaces.
//
// This file defines the Client interface and factory functions for
// creating image generation clients. The interface allows for multiple
// implementations:
//   - Real CGo-based implementation (requires CUDA)
//   - Stub implementation for testing/development
//   - Mock implementation for unit tests
//
// Build constraints: This file compiles on all platforms.
package sd

import (
	"context"
	"io"
)

// Client defines the interface for Stable Diffusion image generation.
//
// Implementations of this interface handle:
//   - Model loading and context management
//   - Image generation from text prompts
//   - Resource cleanup
//
// The interface follows the atomic design pattern as a template,
// defining the contract without implementation.
type Client interface {
	// Generate creates an image from a text prompt.
	//
	// The context can be used for cancellation and timeout.
	// The request must pass Validate() or an error is returned.
	//
	// Returns the generated image response or an error.
	// On error, the response may be nil or partially filled.
	Generate(ctx context.Context, request GenerationRequest) (*GenerationResponse, error)

	// IsReady returns true if the client is ready to generate images.
	// This means the model is loaded and the backend is available.
	IsReady() bool

	// GetModelInfo returns information about the loaded model.
	// Returns nil if no model is loaded.
	GetModelInfo() *ModelInfo

	// GetBackendInfo returns information about the compute backend.
	GetBackendInfo() BackendInfo

	// Close releases all resources held by the client.
	// After Close, the client should not be used.
	// Close is safe to call multiple times.
	Close() error
}

// ClientConfig holds configuration for creating a Client.
type ClientConfig struct {
	// ModelPath is the path to the Stable Diffusion model file.
	// Required for real implementations.
	ModelPath string

	// VAEPath is an optional path to a separate VAE model.
	// If empty, the built-in VAE is used.
	VAEPath string

	// NumThreads is the number of CPU threads for non-GPU operations.
	// Default: runtime.NumCPU()
	NumThreads int

	// MaxConcurrent is the maximum number of concurrent generations.
	// Default: 2
	MaxConcurrent int

	// AcquireTimeout is how long to wait for a generation slot.
	// Default: 30s
	AcquireTimeout int // seconds

	// GenerationTimeout is the maximum time for a single generation.
	// Default: 60s
	GenerationTimeout int // seconds

	// VAETiling enables VAE tiling for lower memory usage.
	// Default: false
	VAETiling bool

	// FreeParamsImmediately frees model params after loading to save memory.
	// Default: false
	FreeParamsImmediately bool
}

// DefaultClientConfig returns a ClientConfig with sensible defaults.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		ModelPath:             "",
		VAEPath:               "",
		NumThreads:            0, // Use runtime.NumCPU()
		MaxConcurrent:         2,
		AcquireTimeout:        30,
		GenerationTimeout:     60,
		VAETiling:             false,
		FreeParamsImmediately: false,
	}
}

// Validate checks if the configuration is valid.
func (c ClientConfig) Validate() error {
	// ModelPath is required for real implementations
	// (stub implementations may not need it)

	if c.MaxConcurrent < 1 {
		return NewGenerationError(ErrCodeInvalidRequest, "max_concurrent must be at least 1", false, nil)
	}
	if c.MaxConcurrent > 10 {
		return NewGenerationError(ErrCodeInvalidRequest, "max_concurrent must be at most 10", false, nil)
	}

	if c.AcquireTimeout < 1 {
		return NewGenerationError(ErrCodeInvalidRequest, "acquire_timeout must be at least 1 second", false, nil)
	}

	if c.GenerationTimeout < 1 {
		return NewGenerationError(ErrCodeInvalidRequest, "generation_timeout must be at least 1 second", false, nil)
	}

	return nil
}

// ClientFactory is a function type for creating Client instances.
// This allows dependency injection and testing.
type ClientFactory func(config ClientConfig) (Client, error)

// Progress represents generation progress updates.
// Used for streaming progress during long-running generations.
type Progress struct {
	// Step is the current diffusion step (0-indexed).
	Step int

	// TotalSteps is the total number of steps.
	TotalSteps int

	// Percentage is the completion percentage (0-100).
	Percentage float64

	// ETA is the estimated time remaining.
	ETA string

	// Preview may contain a preview image at this step.
	// Can be nil if preview is not available.
	Preview []byte
}

// ProgressCallback is called during generation to report progress.
type ProgressCallback func(progress Progress)

// StreamingClient extends Client with streaming progress support.
type StreamingClient interface {
	Client

	// GenerateWithProgress creates an image with progress updates.
	// The callback is called periodically during generation.
	GenerateWithProgress(ctx context.Context, request GenerationRequest, callback ProgressCallback) (*GenerationResponse, error)
}

// BatchClient extends Client with batch generation support.
type BatchClient interface {
	Client

	// GenerateBatch creates multiple images from the same prompt.
	// Returns a slice of responses, one per image.
	// Some responses may contain errors while others succeed.
	GenerateBatch(ctx context.Context, request GenerationRequest, count int) ([]*GenerationResponse, error)
}

// ClientWithMetrics extends Client with metrics collection.
type ClientWithMetrics interface {
	Client

	// GetMetrics returns generation statistics.
	GetMetrics() ClientMetrics
}

// ClientMetrics holds statistics about client usage.
type ClientMetrics struct {
	// TotalGenerations is the total number of generation attempts.
	TotalGenerations int64

	// SuccessfulGenerations is the number of successful generations.
	SuccessfulGenerations int64

	// FailedGenerations is the number of failed generations.
	FailedGenerations int64

	// TotalDuration is the cumulative generation time.
	TotalDuration int64 // milliseconds

	// AverageDuration is the average generation time.
	AverageDuration int64 // milliseconds

	// QueueDepth is the current number of queued requests.
	QueueDepth int

	// ActiveGenerations is the current number of active generations.
	ActiveGenerations int
}

// NullClient is a Client implementation that does nothing.
// Useful for testing and as a default when SD is not available.
type NullClient struct{}

// Generate returns an error indicating SD is not available.
func (c *NullClient) Generate(ctx context.Context, request GenerationRequest) (*GenerationResponse, error) {
	return nil, NewGenerationError(
		ErrCodeCUDAUnavailable,
		"Stable Diffusion is not available (null client)",
		false,
		nil,
	)
}

// IsReady always returns false for NullClient.
func (c *NullClient) IsReady() bool {
	return false
}

// GetModelInfo returns nil for NullClient.
func (c *NullClient) GetModelInfo() *ModelInfo {
	return nil
}

// GetBackendInfo returns unavailable backend info.
func (c *NullClient) GetBackendInfo() BackendInfo {
	return BackendInfo{
		Name:      "None",
		Available: false,
	}
}

// Close does nothing for NullClient.
func (c *NullClient) Close() error {
	return nil
}

// Ensure NullClient implements Client.
var _ Client = (*NullClient)(nil)

// WriterClient wraps a Client to write generated images to an io.Writer.
// This is a utility adapter, not a full Client implementation.
type WriterClient struct {
	client Client
}

// NewWriterClient creates a WriterClient wrapping the given Client.
func NewWriterClient(client Client) *WriterClient {
	return &WriterClient{client: client}
}

// GenerateToWriter generates an image and writes it to the provided writer.
func (w *WriterClient) GenerateToWriter(ctx context.Context, request GenerationRequest, out io.Writer) (*GenerationResponse, error) {
	resp, err := w.client.Generate(ctx, request)
	if err != nil {
		return nil, err
	}

	if resp != nil && len(resp.ImageData) > 0 {
		_, writeErr := out.Write(resp.ImageData)
		if writeErr != nil {
			return resp, NewGenerationError(
				ErrCodeGenerationFailed,
				"failed to write image data",
				false,
				writeErr,
			)
		}
	}

	return resp, nil
}

// Inner returns the underlying Client.
func (w *WriterClient) Inner() Client {
	return w.client
}
