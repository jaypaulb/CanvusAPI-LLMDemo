// Package sd provides Stable Diffusion image generation types and interfaces.
//
// This package defines the data structures for image generation requests and
// responses, following atomic design principles as atoms (pure data types).
//
// The types here are designed to be:
//   - Platform-independent (no CGo dependencies)
//   - Serializable (for potential IPC/API use)
//   - Validated (with validation methods)
//
// Build constraints: This file compiles on all platforms.
package sd

import (
	"fmt"
	"time"
)

// ImageFormat represents the output format for generated images.
type ImageFormat string

const (
	// FormatPNG is PNG format (default, lossless).
	FormatPNG ImageFormat = "png"
	// FormatJPEG is JPEG format (lossy, smaller files).
	FormatJPEG ImageFormat = "jpeg"
)

// SampleMethod represents the sampling algorithm for diffusion.
type SampleMethod int

const (
	// SampleEulerA is Euler Ancestral sampling (fast, good quality).
	SampleEulerA SampleMethod = iota
	// SampleEuler is deterministic Euler sampling.
	SampleEuler
	// SampleHeun is Heun's method (slower, higher quality).
	SampleHeun
	// SampleDPM2 is DPM2 sampling.
	SampleDPM2
	// SampleDPMPP2SA is DPM++ 2S Ancestral sampling.
	SampleDPMPP2SA
	// SampleDPMPP2M is DPM++ 2M sampling (recommended).
	SampleDPMPP2M
	// SampleLCM is LCM sampling (very fast, requires LCM model).
	SampleLCM
)

// String returns the human-readable name of the sample method.
func (s SampleMethod) String() string {
	names := []string{
		"euler_a",
		"euler",
		"heun",
		"dpm2",
		"dpmpp_2s_a",
		"dpmpp_2m",
		"lcm",
	}
	if int(s) < 0 || int(s) >= len(names) {
		return "unknown"
	}
	return names[s]
}

// ParseSampleMethod converts a string to SampleMethod.
func ParseSampleMethod(s string) (SampleMethod, error) {
	methods := map[string]SampleMethod{
		"euler_a":    SampleEulerA,
		"euler":      SampleEuler,
		"heun":       SampleHeun,
		"dpm2":       SampleDPM2,
		"dpmpp_2s_a": SampleDPMPP2SA,
		"dpmpp_2m":   SampleDPMPP2M,
		"lcm":        SampleLCM,
	}
	if m, ok := methods[s]; ok {
		return m, nil
	}
	return SampleEulerA, fmt.Errorf("unknown sample method: %s", s)
}

// GenerationRequest contains parameters for image generation.
// This is an atom-level type with no dependencies.
type GenerationRequest struct {
	// Prompt is the text description of the image to generate (required).
	Prompt string `json:"prompt"`

	// NegativePrompt describes what to avoid in the image (optional).
	NegativePrompt string `json:"negative_prompt,omitempty"`

	// Width of the output image in pixels.
	// Must be 128-2048 and divisible by 8.
	// Default: 512
	Width int `json:"width"`

	// Height of the output image in pixels.
	// Must be 128-2048 and divisible by 8.
	// Default: 512
	Height int `json:"height"`

	// Steps is the number of diffusion steps.
	// More steps = higher quality but slower.
	// Must be 1-100. Default: 25
	Steps int `json:"steps"`

	// CFGScale is the classifier-free guidance scale.
	// Higher values follow the prompt more closely.
	// Must be 1.0-30.0. Default: 7.5
	CFGScale float64 `json:"cfg_scale"`

	// Seed for random number generation.
	// Use -1 for random seed.
	// Default: -1
	Seed int64 `json:"seed"`

	// SampleMethod is the sampling algorithm to use.
	// Default: SampleEulerA
	SampleMethod SampleMethod `json:"sample_method"`

	// Format is the output image format.
	// Default: FormatPNG
	Format ImageFormat `json:"format,omitempty"`

	// ClipSkip is the number of CLIP layers to skip.
	// Use -1 for model default.
	// Default: -1
	ClipSkip int `json:"clip_skip,omitempty"`
}

// Validation constants
const (
	MinWidth     = 128
	MaxWidth     = 2048
	MinHeight    = 128
	MaxHeight    = 2048
	SizeMultiple = 8 // Dimensions must be divisible by this

	MinSteps = 1
	MaxSteps = 100

	MinCFGScale = 1.0
	MaxCFGScale = 30.0

	MaxPromptLength = 1000
)

// DefaultRequest returns a GenerationRequest with sensible defaults.
// This is a pure function (no side effects).
func DefaultRequest() GenerationRequest {
	return GenerationRequest{
		Prompt:         "",
		NegativePrompt: "ugly, blurry, low quality, deformed",
		Width:          512,
		Height:         512,
		Steps:          25,
		CFGScale:       7.5,
		Seed:           -1, // Random
		SampleMethod:   SampleEulerA,
		Format:         FormatPNG,
		ClipSkip:       -1, // Use default
	}
}

// Validate checks if the request parameters are valid.
// Returns nil if valid, or an error describing the problem.
// This is a pure function (no side effects).
func (r GenerationRequest) Validate() error {
	// Validate prompt
	if r.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	if len(r.Prompt) > MaxPromptLength {
		return fmt.Errorf("prompt length %d exceeds maximum %d", len(r.Prompt), MaxPromptLength)
	}

	// Validate negative prompt
	if len(r.NegativePrompt) > MaxPromptLength {
		return fmt.Errorf("negative prompt length %d exceeds maximum %d", len(r.NegativePrompt), MaxPromptLength)
	}

	// Validate width
	if r.Width < MinWidth || r.Width > MaxWidth {
		return fmt.Errorf("width %d must be between %d and %d", r.Width, MinWidth, MaxWidth)
	}
	if r.Width%SizeMultiple != 0 {
		return fmt.Errorf("width %d must be divisible by %d", r.Width, SizeMultiple)
	}

	// Validate height
	if r.Height < MinHeight || r.Height > MaxHeight {
		return fmt.Errorf("height %d must be between %d and %d", r.Height, MinHeight, MaxHeight)
	}
	if r.Height%SizeMultiple != 0 {
		return fmt.Errorf("height %d must be divisible by %d", r.Height, SizeMultiple)
	}

	// Validate steps
	if r.Steps < MinSteps || r.Steps > MaxSteps {
		return fmt.Errorf("steps %d must be between %d and %d", r.Steps, MinSteps, MaxSteps)
	}

	// Validate CFG scale
	if r.CFGScale < MinCFGScale || r.CFGScale > MaxCFGScale {
		return fmt.Errorf("cfg_scale %.2f must be between %.1f and %.1f", r.CFGScale, MinCFGScale, MaxCFGScale)
	}

	return nil
}

// WithPrompt returns a copy with the specified prompt.
// Builder pattern for immutable updates.
func (r GenerationRequest) WithPrompt(prompt string) GenerationRequest {
	r.Prompt = prompt
	return r
}

// WithSize returns a copy with the specified dimensions.
// Builder pattern for immutable updates.
func (r GenerationRequest) WithSize(width, height int) GenerationRequest {
	r.Width = width
	r.Height = height
	return r
}

// WithSteps returns a copy with the specified step count.
// Builder pattern for immutable updates.
func (r GenerationRequest) WithSteps(steps int) GenerationRequest {
	r.Steps = steps
	return r
}

// WithSeed returns a copy with the specified seed.
// Builder pattern for immutable updates.
func (r GenerationRequest) WithSeed(seed int64) GenerationRequest {
	r.Seed = seed
	return r
}

// GenerationResponse contains the result of image generation.
// This is an atom-level type with no dependencies.
type GenerationResponse struct {
	// ImageData contains the raw image bytes (PNG or JPEG).
	ImageData []byte `json:"image_data,omitempty"`

	// Width of the generated image in pixels.
	Width int `json:"width"`

	// Height of the generated image in pixels.
	Height int `json:"height"`

	// Format of the image data.
	Format ImageFormat `json:"format"`

	// Seed that was used for generation.
	// Useful for reproducing results.
	Seed int64 `json:"seed"`

	// Duration is how long generation took.
	Duration time.Duration `json:"duration"`

	// Request is the original request that produced this response.
	// Useful for debugging and logging.
	Request *GenerationRequest `json:"request,omitempty"`
}

// IsValid checks if the response contains valid image data.
func (r GenerationResponse) IsValid() bool {
	return len(r.ImageData) > 0 && r.Width > 0 && r.Height > 0
}

// GenerationError represents an error during image generation.
type GenerationError struct {
	// Code is a machine-readable error code.
	Code string `json:"code"`

	// Message is a human-readable error description.
	Message string `json:"message"`

	// Retryable indicates if the operation might succeed on retry.
	Retryable bool `json:"retryable"`

	// Cause is the underlying error (if any).
	Cause error `json:"-"`
}

// Error implements the error interface.
func (e GenerationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e GenerationError) Unwrap() error {
	return e.Cause
}

// Common error codes
const (
	ErrCodeInvalidRequest   = "invalid_request"
	ErrCodeModelNotFound    = "model_not_found"
	ErrCodeModelLoadFailed  = "model_load_failed"
	ErrCodeOutOfMemory      = "out_of_memory"
	ErrCodeTimeout          = "timeout"
	ErrCodeCUDAUnavailable  = "cuda_unavailable"
	ErrCodeGenerationFailed = "generation_failed"
)

// NewGenerationError creates a new GenerationError.
func NewGenerationError(code, message string, retryable bool, cause error) GenerationError {
	return GenerationError{
		Code:      code,
		Message:   message,
		Retryable: retryable,
		Cause:     cause,
	}
}

// ModelInfo contains information about a loaded model.
type ModelInfo struct {
	// Path is the filesystem path to the model.
	Path string `json:"path"`

	// Name is a human-readable model name.
	Name string `json:"name"`

	// Size is the model file size in bytes.
	Size int64 `json:"size"`

	// Loaded indicates if the model is currently loaded.
	Loaded bool `json:"loaded"`

	// VRAMUsage is estimated VRAM usage in bytes (0 if unknown).
	VRAMUsage int64 `json:"vram_usage,omitempty"`
}

// BackendInfo contains information about the compute backend.
type BackendInfo struct {
	// Name is the backend name (e.g., "CUDA", "CPU").
	Name string `json:"name"`

	// Available indicates if the backend is functional.
	Available bool `json:"available"`

	// DeviceName is the GPU/device name if applicable.
	DeviceName string `json:"device_name,omitempty"`

	// VRAMTotal is total VRAM in bytes (0 for CPU).
	VRAMTotal int64 `json:"vram_total,omitempty"`

	// VRAMFree is free VRAM in bytes (0 for CPU).
	VRAMFree int64 `json:"vram_free,omitempty"`
}
