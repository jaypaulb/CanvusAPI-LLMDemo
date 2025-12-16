package sdruntime

import (
	"os"
	"strconv"
	"time"
)

// SDConfig holds configuration for Stable Diffusion image generation.
type SDConfig struct {
	// Image generation defaults
	ImageSize      int     // Default image size (512, 768, or 1024)
	InferenceSteps int     // Default inference steps (1-100)
	GuidanceScale  float64 // Default CFG scale (1.0-30.0)
	NegativePrompt string  // Default negative prompt

	// Runtime configuration
	Timeout       time.Duration // Generation timeout
	MaxConcurrent int           // Maximum concurrent generations

	// Model configuration
	ModelPath string // Path to SD model file
}

// Default configuration values
const (
	DefaultImageSize      = 512
	DefaultInferenceSteps = 20
	DefaultGuidanceScale  = 7.5
	DefaultTimeoutSeconds = 120
	DefaultMaxConcurrent  = 1
)

// LoadSDConfig loads SD configuration from environment variables.
// This is a pure parsing function that reads from env vars.
func LoadSDConfig() *SDConfig {
	return &SDConfig{
		ImageSize:      parseImageSize(os.Getenv("SD_IMAGE_SIZE")),
		InferenceSteps: parseInferenceSteps(os.Getenv("SD_INFERENCE_STEPS")),
		GuidanceScale:  parseGuidanceScale(os.Getenv("SD_GUIDANCE_SCALE")),
		NegativePrompt: os.Getenv("SD_NEGATIVE_PROMPT"),
		Timeout:        parseTimeout(os.Getenv("SD_TIMEOUT_SECONDS")),
		MaxConcurrent:  parseMaxConcurrent(os.Getenv("SD_MAX_CONCURRENT")),
		ModelPath:      os.Getenv("SD_MODEL_PATH"),
	}
}

// parseImageSize parses and validates image size from string.
// Valid values: 512, 768, 1024 (common SD sizes).
// Returns default if invalid or empty.
func parseImageSize(s string) int {
	if s == "" {
		return DefaultImageSize
	}

	size, err := strconv.Atoi(s)
	if err != nil {
		return DefaultImageSize
	}

	// Validate against common SD sizes
	switch size {
	case 512, 768, 1024:
		return size
	default:
		// For custom sizes, validate range and divisibility
		if size >= MinImageSize && size <= MaxImageSize && size%ImageSizeMultple == 0 {
			return size
		}
		return DefaultImageSize
	}
}

// parseInferenceSteps parses and validates inference steps from string.
// Returns default if invalid or out of range.
func parseInferenceSteps(s string) int {
	if s == "" {
		return DefaultInferenceSteps
	}

	steps, err := strconv.Atoi(s)
	if err != nil {
		return DefaultInferenceSteps
	}

	if steps < MinSteps || steps > MaxSteps {
		return DefaultInferenceSteps
	}

	return steps
}

// parseGuidanceScale parses and validates CFG scale from string.
// Returns default if invalid or out of range.
func parseGuidanceScale(s string) float64 {
	if s == "" {
		return DefaultGuidanceScale
	}

	scale, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return DefaultGuidanceScale
	}

	if scale < MinCFGScale || scale > MaxCFGScale {
		return DefaultGuidanceScale
	}

	return scale
}

// parseTimeout parses timeout in seconds from string.
// Returns default if invalid.
func parseTimeout(s string) time.Duration {
	if s == "" {
		return time.Duration(DefaultTimeoutSeconds) * time.Second
	}

	seconds, err := strconv.Atoi(s)
	if err != nil || seconds <= 0 {
		return time.Duration(DefaultTimeoutSeconds) * time.Second
	}

	return time.Duration(seconds) * time.Second
}

// parseMaxConcurrent parses max concurrent generations from string.
// Returns default if invalid.
func parseMaxConcurrent(s string) int {
	if s == "" {
		return DefaultMaxConcurrent
	}

	concurrent, err := strconv.Atoi(s)
	if err != nil || concurrent < 1 {
		return DefaultMaxConcurrent
	}

	return concurrent
}
