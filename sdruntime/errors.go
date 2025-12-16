// Package sdruntime provides Stable Diffusion image generation capabilities.
package sdruntime

import "errors"

// Sentinel errors for SD runtime operations.
// These are domain-specific errors that provide clear failure modes.
var (
	// Model-related errors
	ErrModelNotFound   = errors.New("sdruntime: model file not found")
	ErrModelLoadFailed = errors.New("sdruntime: failed to load model")
	ErrModelCorrupted  = errors.New("sdruntime: model file is corrupted or invalid")

	// Generation errors
	ErrGenerationFailed  = errors.New("sdruntime: image generation failed")
	ErrGenerationTimeout = errors.New("sdruntime: image generation timed out")

	// Input validation errors
	ErrInvalidPrompt = errors.New("sdruntime: invalid prompt")
	ErrInvalidParams = errors.New("sdruntime: invalid generation parameters")

	// Hardware/resource errors
	ErrCUDANotAvailable = errors.New("sdruntime: CUDA not available")
	ErrOutOfVRAM        = errors.New("sdruntime: out of VRAM")

	// Context pool errors
	ErrContextPoolClosed = errors.New("sdruntime: context pool is closed")
	ErrAcquireTimeout    = errors.New("sdruntime: timeout acquiring context from pool")
)
