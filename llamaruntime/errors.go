// Package llamaruntime provides Go bindings to llama.cpp for local LLM inference.
package llamaruntime

import (
	"errors"
	"fmt"
)

// LlamaError represents an error from llama.cpp operations.
// It provides structured error information including the operation that failed,
// an error code from the C layer, and a descriptive message.
type LlamaError struct {
	Op      string // Operation that failed (e.g., "loadModel", "infer")
	Code    int    // Error code from C layer (0 = success, non-zero = error)
	Message string // Human-readable error message
	Err     error  // Wrapped underlying error (if any)
}

// Error implements the error interface.
// It returns a formatted error string with operation, message, and wrapped error.
func (e *LlamaError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("llama.cpp %s: %s (code: %d): %v", e.Op, e.Message, e.Code, e.Err)
	}
	return fmt.Sprintf("llama.cpp %s: %s (code: %d)", e.Op, e.Message, e.Code)
}

// Unwrap implements the error unwrapping interface.
// It returns the wrapped error, allowing use with errors.Is and errors.As.
func (e *LlamaError) Unwrap() error {
	return e.Err
}

// Sentinel errors for common failure conditions.
// These are used for error checking with errors.Is().
var (
	// ErrModelNotFound indicates the model file was not found at the specified path.
	ErrModelNotFound = errors.New("model file not found")

	// ErrModelLoadFailed indicates the model file exists but failed to load.
	// This may be due to corruption, incompatible format, or insufficient resources.
	ErrModelLoadFailed = errors.New("failed to load model")

	// ErrContextCreateFailed indicates failure to create an inference context.
	// This may be due to insufficient GPU memory or invalid parameters.
	ErrContextCreateFailed = errors.New("failed to create inference context")

	// ErrInferenceFailed indicates the inference operation failed.
	// This may be due to invalid input, timeout, or internal llama.cpp errors.
	ErrInferenceFailed = errors.New("inference failed")

	// ErrGPUNotAvailable indicates CUDA GPU is not available or not detected.
	// This is a critical error as CPU-only mode is not supported in Phase 2.
	ErrGPUNotAvailable = errors.New("CUDA GPU not available")

	// ErrInsufficientVRAM indicates insufficient GPU VRAM to load the model.
	// The user may need to use a smaller quantization or upgrade hardware.
	ErrInsufficientVRAM = errors.New("insufficient GPU VRAM")

	// ErrInvalidImage indicates the provided image data is invalid or unsupported.
	// This may be due to unsupported format, corrupted data, or encoding issues.
	ErrInvalidImage = errors.New("invalid or unsupported image format")

	// ErrTimeout indicates the inference operation timed out.
	// This may occur with very long prompts or insufficient GPU resources.
	ErrTimeout = errors.New("inference timeout")
)
