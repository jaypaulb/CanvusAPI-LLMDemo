// Package llamaruntime provides Go bindings to llama.cpp for local LLM inference.
// This file contains pure Go types and constants - no CGo dependencies.
package llamaruntime

import (
	"time"
)

// =============================================================================
// Default Constants
// =============================================================================

const (
	// DefaultContextSize is the default context window size in tokens.
	// Bunny v1.1 supports up to 2048 tokens, but we use a conservative default.
	DefaultContextSize = 2048

	// DefaultBatchSize is the default batch size for inference.
	// Smaller batches use less memory but may be slower.
	DefaultBatchSize = 512

	// DefaultNumGPULayers is the default number of layers to offload to GPU.
	// -1 means offload all layers to GPU (recommended for CUDA builds).
	DefaultNumGPULayers = -1

	// DefaultNumThreads is the default number of CPU threads for inference.
	// Used for hybrid GPU/CPU inference or CPU fallback.
	DefaultNumThreads = 4

	// DefaultMaxTokens is the default maximum number of tokens to generate.
	DefaultMaxTokens = 512

	// DefaultTemperature is the default sampling temperature.
	// Lower values make output more deterministic.
	DefaultTemperature = 0.7

	// DefaultTopP is the default top-p (nucleus) sampling parameter.
	// Lower values make output more focused.
	DefaultTopP = 0.9

	// DefaultTopK is the default top-k sampling parameter.
	// Limits the number of tokens considered for sampling.
	DefaultTopK = 40

	// DefaultRepeatPenalty is the default repeat penalty.
	// Higher values discourage repetitive output.
	DefaultRepeatPenalty = 1.1

	// DefaultTimeout is the default timeout for inference operations.
	DefaultTimeout = 2 * time.Minute

	// MinContextSize is the minimum allowed context size.
	MinContextSize = 256

	// MaxContextSize is the maximum allowed context size.
	// Limited by model architecture and VRAM.
	MaxContextSize = 8192

	// MinBatchSize is the minimum allowed batch size.
	MinBatchSize = 1

	// MaxBatchSize is the maximum allowed batch size.
	MaxBatchSize = 2048
)

// =============================================================================
// Configuration Types
// =============================================================================

// Config contains the configuration for initializing the llama.cpp runtime.
// All fields have reasonable defaults and can be overridden as needed.
type Config struct {
	// ModelPath is the path to the GGUF model file.
	// Required - no default.
	ModelPath string

	// ContextSize is the context window size in tokens.
	// Defaults to DefaultContextSize.
	ContextSize int

	// BatchSize is the batch size for inference.
	// Defaults to DefaultBatchSize.
	BatchSize int

	// NumGPULayers is the number of layers to offload to GPU.
	// -1 means all layers (recommended). Defaults to DefaultNumGPULayers.
	NumGPULayers int

	// NumThreads is the number of CPU threads for inference.
	// Defaults to DefaultNumThreads.
	NumThreads int

	// UseMMap enables memory-mapped model loading.
	// Recommended for faster startup. Defaults to true.
	UseMMap bool

	// UseMlock enables memory locking to prevent swapping.
	// May require elevated privileges. Defaults to false.
	UseMlock bool

	// VerboseLogging enables verbose llama.cpp logging.
	// Useful for debugging. Defaults to false.
	VerboseLogging bool

	// Seed is the random seed for reproducible output.
	// -1 means random seed. Defaults to -1.
	Seed int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		ContextSize:    DefaultContextSize,
		BatchSize:      DefaultBatchSize,
		NumGPULayers:   DefaultNumGPULayers,
		NumThreads:     DefaultNumThreads,
		UseMMap:        true,
		UseMlock:       false,
		VerboseLogging: false,
		Seed:           -1,
	}
}

// InferenceParams contains parameters for a single inference request.
type InferenceParams struct {
	// Prompt is the input text prompt.
	// Required.
	Prompt string

	// MaxTokens is the maximum number of tokens to generate.
	// Defaults to DefaultMaxTokens.
	MaxTokens int

	// Temperature controls randomness in sampling.
	// Higher values (e.g., 1.0) make output more random.
	// Lower values (e.g., 0.1) make output more deterministic.
	// Defaults to DefaultTemperature.
	Temperature float32

	// TopP is the nucleus sampling parameter.
	// Only tokens with cumulative probability <= TopP are considered.
	// Defaults to DefaultTopP.
	TopP float32

	// TopK is the top-k sampling parameter.
	// Only the top K most likely tokens are considered.
	// Defaults to DefaultTopK.
	TopK int

	// RepeatPenalty penalizes repeated tokens.
	// Higher values (e.g., 1.5) discourage repetition.
	// Defaults to DefaultRepeatPenalty.
	RepeatPenalty float32

	// StopSequences are sequences that stop generation when encountered.
	// Common examples: ["</s>", "\n\n", "User:"]
	StopSequences []string

	// Timeout is the maximum time allowed for inference.
	// Defaults to DefaultTimeout.
	Timeout time.Duration
}

// DefaultInferenceParams returns InferenceParams with sensible defaults.
func DefaultInferenceParams() InferenceParams {
	return InferenceParams{
		MaxTokens:     DefaultMaxTokens,
		Temperature:   DefaultTemperature,
		TopP:          DefaultTopP,
		TopK:          DefaultTopK,
		RepeatPenalty: DefaultRepeatPenalty,
		Timeout:       DefaultTimeout,
	}
}

// =============================================================================
// Vision Types (for Bunny multimodal model)
// =============================================================================

// VisionParams contains parameters for vision (image) inference.
// Used with multimodal models like Bunny.
type VisionParams struct {
	// ImageData is the raw image bytes.
	// Supported formats: JPEG, PNG.
	ImageData []byte

	// ImagePath is an alternative to ImageData - path to image file.
	// If both are provided, ImageData takes precedence.
	ImagePath string

	// Prompt is the text prompt to accompany the image.
	// For Bunny, this typically describes what to analyze in the image.
	Prompt string

	// MaxTokens is the maximum number of tokens to generate.
	// Defaults to DefaultMaxTokens.
	MaxTokens int

	// Temperature controls randomness in sampling.
	// Defaults to DefaultTemperature.
	Temperature float32

	// Timeout is the maximum time allowed for inference.
	// Vision inference may take longer due to image processing.
	// Defaults to DefaultTimeout.
	Timeout time.Duration
}

// DefaultVisionParams returns VisionParams with sensible defaults.
func DefaultVisionParams() VisionParams {
	return VisionParams{
		MaxTokens:   DefaultMaxTokens,
		Temperature: DefaultTemperature,
		Timeout:     DefaultTimeout,
	}
}

// =============================================================================
// Result Types
// =============================================================================

// InferenceResult contains the result of an inference operation.
type InferenceResult struct {
	// Text is the generated text output.
	Text string

	// TokensGenerated is the number of tokens generated.
	TokensGenerated int

	// TokensPrompt is the number of tokens in the prompt.
	TokensPrompt int

	// Duration is the total time taken for inference.
	Duration time.Duration

	// TokensPerSecond is the generation speed.
	TokensPerSecond float64

	// StopReason indicates why generation stopped.
	// Possible values: "max_tokens", "stop_sequence", "eos"
	StopReason string
}

// InferenceStats contains detailed performance statistics.
type InferenceStats struct {
	// TotalInferences is the total number of inference calls.
	TotalInferences int64

	// TotalTokensGenerated is the total tokens generated across all calls.
	TotalTokensGenerated int64

	// TotalTokensPrompt is the total prompt tokens processed.
	TotalTokensPrompt int64

	// TotalDuration is the cumulative inference time.
	TotalDuration time.Duration

	// AverageTokensPerSecond is the average generation speed.
	AverageTokensPerSecond float64

	// PeakMemoryUsage is the peak GPU memory usage in bytes.
	PeakMemoryUsage int64

	// ErrorCount is the number of failed inference calls.
	ErrorCount int64
}

// =============================================================================
// GPU Types
// =============================================================================

// GPUInfo contains information about a CUDA GPU.
type GPUInfo struct {
	// Index is the GPU index (0-based).
	Index int

	// Name is the GPU model name (e.g., "NVIDIA GeForce RTX 3080").
	Name string

	// TotalMemory is the total GPU memory in bytes.
	TotalMemory int64

	// FreeMemory is the available GPU memory in bytes.
	FreeMemory int64

	// UsedMemory is the used GPU memory in bytes.
	UsedMemory int64

	// ComputeCapability is the CUDA compute capability (e.g., "8.6").
	ComputeCapability string

	// DriverVersion is the NVIDIA driver version.
	DriverVersion string

	// CUDAVersion is the CUDA runtime version.
	CUDAVersion string

	// IsAvailable indicates if the GPU is available for use.
	IsAvailable bool

	// Temperature is the GPU temperature in Celsius (if available).
	Temperature int

	// Utilization is the GPU utilization percentage (if available).
	Utilization int

	// PowerDraw is the current power draw in watts (if available).
	PowerDraw float32

	// PowerLimit is the power limit in watts (if available).
	PowerLimit float32
}

// GPUStatus represents the overall GPU status.
type GPUStatus struct {
	// Available indicates if any CUDA GPU is available.
	Available bool

	// GPUCount is the number of available GPUs.
	GPUCount int

	// GPUs contains information about each GPU.
	GPUs []GPUInfo

	// TotalMemory is the total memory across all GPUs.
	TotalMemory int64

	// FreeMemory is the available memory across all GPUs.
	FreeMemory int64

	// LastChecked is when the GPU status was last updated.
	LastChecked time.Time
}

// =============================================================================
// Model Types
// =============================================================================

// ModelInfo contains information about a loaded model.
type ModelInfo struct {
	// Path is the path to the model file.
	Path string

	// Name is the model name (derived from filename).
	Name string

	// Size is the model file size in bytes.
	Size int64

	// Format is the model format (e.g., "GGUF").
	Format string

	// Quantization is the quantization type (e.g., "Q4_K_M", "Q8_0").
	Quantization string

	// Parameters is the estimated number of model parameters.
	Parameters int64

	// ContextLength is the maximum context length supported.
	ContextLength int

	// EmbeddingLength is the embedding dimension.
	EmbeddingLength int

	// VocabSize is the vocabulary size.
	VocabSize int

	// IsMultimodal indicates if the model supports vision input.
	IsMultimodal bool

	// LoadedAt is when the model was loaded.
	LoadedAt time.Time

	// LoadDuration is how long it took to load the model.
	LoadDuration time.Duration
}

// =============================================================================
// Health Types
// =============================================================================

// HealthStatus represents the health of the runtime.
type HealthStatus struct {
	// Healthy indicates if the runtime is healthy and ready for inference.
	Healthy bool

	// Status is a human-readable status message.
	Status string

	// ModelLoaded indicates if a model is loaded.
	ModelLoaded bool

	// ModelInfo contains information about the loaded model (if any).
	ModelInfo *ModelInfo

	// GPUStatus contains GPU status information.
	GPUStatus *GPUStatus

	// Stats contains inference statistics.
	Stats *InferenceStats

	// LastInference is when the last inference was performed.
	LastInference time.Time

	// Uptime is how long the runtime has been running.
	Uptime time.Duration

	// CheckedAt is when this health check was performed.
	CheckedAt time.Time
}
