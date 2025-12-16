// Package sdruntime provides CGo bindings for stable-diffusion.cpp.
//
// This file contains wrapper functions for the stable-diffusion.cpp C library.
// When the library is not available, build with the "stub" tag to use mock implementations.
//
// Build requirements for real CGo implementation:
//   - stable-diffusion.cpp compiled as shared library (libstable-diffusion.so/dylib/dll)
//   - Header file: stable-diffusion.h
//   - Set CGO_CFLAGS and CGO_LDFLAGS appropriately
//
// Example build with real library:
//
//	CGO_CFLAGS="-I/path/to/stable-diffusion.cpp" \
//	CGO_LDFLAGS="-L/path/to/stable-diffusion.cpp/build -lstable-diffusion" \
//	go build -tags sd
//
// Example build without library (stub mode):
//
//	go build -tags stub
package sdruntime

// SDContext represents an opaque handle to a stable-diffusion context.
// In the real implementation, this wraps a C pointer to sd_ctx_t.
// The stub implementation uses an internal ID for tracking.
type SDContext struct {
	// id is used for stub implementation tracking
	id uint64
	// modelPath stores the path used to load this context
	modelPath string
	// valid indicates if this context is usable
	valid bool
}

// IsValid returns whether this context is valid and usable.
func (c *SDContext) IsValid() bool {
	if c == nil {
		return false
	}
	return c.valid
}

// ModelPath returns the model path used to create this context.
func (c *SDContext) ModelPath() string {
	if c == nil {
		return ""
	}
	return c.modelPath
}

// GenerateResult holds the result of an image generation operation.
type GenerateResult struct {
	// ImageData contains the raw PNG image bytes
	ImageData []byte
	// Width of the generated image
	Width int
	// Height of the generated image
	Height int
	// Seed used for generation (may differ from input if -1 was specified)
	Seed int64
}

// LoadModel loads a Stable Diffusion model and returns a context for generation.
// The modelPath should point to a valid .safetensors or .ckpt model file.
//
// This function composes:
//   - ErrModelNotFound: when modelPath does not exist
//   - ErrModelLoadFailed: when the C library fails to load the model
//   - ErrModelCorrupted: when the model file is invalid
//
// The returned SDContext must be freed with FreeContext when no longer needed.
//
// Real implementation requirements:
//   - Call C.sd_ctx_create() or equivalent
//   - Handle C string allocation with C.CString
//   - Defer C.free for allocated C strings
//   - Check return value for NULL (indicates failure)
func LoadModel(modelPath string) (*SDContext, error) {
	return loadModelImpl(modelPath)
}

// GenerateImage generates an image using the provided context and parameters.
// The context must be valid (created via LoadModel and not yet freed).
//
// This function composes:
//   - ErrInvalidParams: when params fail validation (via ValidateParams)
//   - ErrGenerationFailed: when the C library fails to generate
//   - ErrGenerationTimeout: when generation exceeds configured timeout
//   - ErrOutOfVRAM: when GPU memory is exhausted
//
// The returned GenerateResult contains PNG image data.
//
// Real implementation requirements:
//   - Validate params using ValidateParams (atom)
//   - Convert Go strings to C strings with C.CString
//   - Defer C.free for all allocated C strings
//   - Call C.txt2img() or equivalent
//   - Convert C image buffer to Go []byte
//   - Free C image buffer after copying
func GenerateImage(ctx *SDContext, params GenerateParams) (*GenerateResult, error) {
	// Validate parameters using atom
	if err := ValidateParams(params); err != nil {
		return nil, err
	}

	return generateImageImpl(ctx, params)
}

// FreeContext releases resources associated with an SDContext.
// This must be called when the context is no longer needed to prevent memory leaks.
// Calling FreeContext on a nil or already-freed context is safe (no-op).
//
// After calling FreeContext, the context is invalid and must not be used.
//
// Real implementation requirements:
//   - Check for nil/invalid context
//   - Call C.sd_ctx_free() or equivalent
//   - Mark context as invalid to prevent double-free
func FreeContext(ctx *SDContext) {
	freeContextImpl(ctx)
}

// GetBackendInfo returns information about the available compute backend.
// This can be used to determine if CUDA/Metal/CPU is being used.
//
// Real implementation requirements:
//   - Query C library for backend information
//   - Return human-readable string
func GetBackendInfo() string {
	return getBackendInfoImpl()
}
