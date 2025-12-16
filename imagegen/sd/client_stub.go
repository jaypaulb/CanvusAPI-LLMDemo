//go:build !sd || !cgo
// +build !sd !cgo

// Package sd provides Stable Diffusion image generation.
//
// This file contains the stub implementation for platforms without CUDA
// or when building without CGo support. It provides a functional but
// limited client that returns appropriate errors.
//
// Build with -tags sd and CGO_ENABLED=1 for real CUDA support.
package sd

import (
	"context"
	"fmt"
)

// NewClient creates a new Stable Diffusion client.
// This stub implementation returns a client that always fails with
// an error indicating CUDA is not available.
//
// For real CUDA support, build with:
//
//	CGO_ENABLED=1 go build -tags sd
func NewClient(config ClientConfig) (Client, error) {
	return &stubClient{config: config}, nil
}

// NewClientFromPath creates a client using the model at the given path.
// This is a convenience wrapper around NewClient.
//
// Stub implementation: Returns a stub client regardless of path.
func NewClientFromPath(modelPath string) (Client, error) {
	config := DefaultClientConfig()
	config.ModelPath = modelPath
	return NewClient(config)
}

// IsCUDAAvailable returns whether CUDA is available for image generation.
// Stub implementation: Always returns false.
func IsCUDAAvailable() bool {
	return false
}

// stubClient is a Client implementation for non-CUDA builds.
type stubClient struct {
	config ClientConfig
	closed bool
}

// Generate returns an error indicating CUDA is required.
func (c *stubClient) Generate(ctx context.Context, request GenerationRequest) (*GenerationResponse, error) {
	if c.closed {
		return nil, NewGenerationError(
			ErrCodeGenerationFailed,
			"client is closed",
			false,
			nil,
		)
	}

	// Validate request first (same behavior as real client)
	if err := request.Validate(); err != nil {
		return nil, NewGenerationError(
			ErrCodeInvalidRequest,
			err.Error(),
			false,
			err,
		)
	}

	return nil, NewGenerationError(
		ErrCodeCUDAUnavailable,
		"Stable Diffusion requires CUDA. Build with: CGO_ENABLED=1 go build -tags sd",
		false,
		nil,
	)
}

// IsReady returns false for stub client.
func (c *stubClient) IsReady() bool {
	return false
}

// GetModelInfo returns nil for stub client.
func (c *stubClient) GetModelInfo() *ModelInfo {
	if c.config.ModelPath != "" {
		return &ModelInfo{
			Path:   c.config.ModelPath,
			Name:   "Stub Model (not loaded)",
			Loaded: false,
		}
	}
	return nil
}

// GetBackendInfo returns stub backend info.
func (c *stubClient) GetBackendInfo() BackendInfo {
	return BackendInfo{
		Name:      "Stub (CUDA not available)",
		Available: false,
	}
}

// Close marks the client as closed.
func (c *stubClient) Close() error {
	c.closed = true
	return nil
}

// Ensure stubClient implements Client.
var _ Client = (*stubClient)(nil)

// GetBuildInfo returns information about how this package was built.
func GetBuildInfo() string {
	return fmt.Sprintf(
		"imagegen/sd: stub build (CUDA support disabled)\n" +
			"To enable CUDA support, rebuild with:\n" +
			"  CGO_ENABLED=1 go build -tags sd\n" +
			"Prerequisites:\n" +
			"  - NVIDIA GPU with CUDA support\n" +
			"  - CUDA Toolkit 11.8+\n" +
			"  - stable-diffusion.cpp built and in lib/",
	)
}
