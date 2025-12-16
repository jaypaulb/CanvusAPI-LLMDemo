package sdruntime

import (
	"os"
	"testing"
	"time"
)

func TestLoadSDConfig_Defaults(t *testing.T) {
	// Clear relevant env vars
	os.Unsetenv("SD_IMAGE_SIZE")
	os.Unsetenv("SD_INFERENCE_STEPS")
	os.Unsetenv("SD_GUIDANCE_SCALE")
	os.Unsetenv("SD_TIMEOUT_SECONDS")
	os.Unsetenv("SD_MAX_CONCURRENT")

	cfg := LoadSDConfig()

	if cfg.ImageSize != DefaultImageSize {
		t.Errorf("expected default ImageSize %d, got %d", DefaultImageSize, cfg.ImageSize)
	}
	if cfg.InferenceSteps != DefaultInferenceSteps {
		t.Errorf("expected default InferenceSteps %d, got %d", DefaultInferenceSteps, cfg.InferenceSteps)
	}
	if cfg.GuidanceScale != DefaultGuidanceScale {
		t.Errorf("expected default GuidanceScale %.1f, got %.1f", DefaultGuidanceScale, cfg.GuidanceScale)
	}
	if cfg.Timeout != time.Duration(DefaultTimeoutSeconds)*time.Second {
		t.Errorf("expected default Timeout %v, got %v", time.Duration(DefaultTimeoutSeconds)*time.Second, cfg.Timeout)
	}
	if cfg.MaxConcurrent != DefaultMaxConcurrent {
		t.Errorf("expected default MaxConcurrent %d, got %d", DefaultMaxConcurrent, cfg.MaxConcurrent)
	}
}

func TestLoadSDConfig_FromEnv(t *testing.T) {
	// Set env vars
	os.Setenv("SD_IMAGE_SIZE", "768")
	os.Setenv("SD_INFERENCE_STEPS", "50")
	os.Setenv("SD_GUIDANCE_SCALE", "12.5")
	os.Setenv("SD_TIMEOUT_SECONDS", "180")
	os.Setenv("SD_MAX_CONCURRENT", "3")
	os.Setenv("SD_NEGATIVE_PROMPT", "blurry, low quality")
	os.Setenv("SD_MODEL_PATH", "/models/sd-v1.5.gguf")

	defer func() {
		os.Unsetenv("SD_IMAGE_SIZE")
		os.Unsetenv("SD_INFERENCE_STEPS")
		os.Unsetenv("SD_GUIDANCE_SCALE")
		os.Unsetenv("SD_TIMEOUT_SECONDS")
		os.Unsetenv("SD_MAX_CONCURRENT")
		os.Unsetenv("SD_NEGATIVE_PROMPT")
		os.Unsetenv("SD_MODEL_PATH")
	}()

	cfg := LoadSDConfig()

	if cfg.ImageSize != 768 {
		t.Errorf("expected ImageSize 768, got %d", cfg.ImageSize)
	}
	if cfg.InferenceSteps != 50 {
		t.Errorf("expected InferenceSteps 50, got %d", cfg.InferenceSteps)
	}
	if cfg.GuidanceScale != 12.5 {
		t.Errorf("expected GuidanceScale 12.5, got %.1f", cfg.GuidanceScale)
	}
	if cfg.Timeout != 180*time.Second {
		t.Errorf("expected Timeout 180s, got %v", cfg.Timeout)
	}
	if cfg.MaxConcurrent != 3 {
		t.Errorf("expected MaxConcurrent 3, got %d", cfg.MaxConcurrent)
	}
	if cfg.NegativePrompt != "blurry, low quality" {
		t.Errorf("expected NegativePrompt 'blurry, low quality', got %q", cfg.NegativePrompt)
	}
	if cfg.ModelPath != "/models/sd-v1.5.gguf" {
		t.Errorf("expected ModelPath '/models/sd-v1.5.gguf', got %q", cfg.ModelPath)
	}
}

func TestParseImageSize_ValidValues(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"512", 512},
		{"768", 768},
		{"1024", 1024},
		{"256", 256}, // Custom valid size
		{"", DefaultImageSize},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseImageSize(tt.input)
			if result != tt.expected {
				t.Errorf("parseImageSize(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseImageSize_InvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"not a number", "abc"},
		{"too small", "64"},
		{"too large", "4096"},
		{"not divisible by 8", "513"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseImageSize(tt.input)
			if result != DefaultImageSize {
				t.Errorf("parseImageSize(%q) = %d, expected default %d", tt.input, result, DefaultImageSize)
			}
		})
	}
}

func TestParseInferenceSteps_ValidValues(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"1", 1},
		{"20", 20},
		{"100", 100},
		{"", DefaultInferenceSteps},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseInferenceSteps(tt.input)
			if result != tt.expected {
				t.Errorf("parseInferenceSteps(%q) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseInferenceSteps_InvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"not a number", "abc"},
		{"zero", "0"},
		{"negative", "-5"},
		{"too high", "150"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInferenceSteps(tt.input)
			if result != DefaultInferenceSteps {
				t.Errorf("parseInferenceSteps(%q) = %d, expected default %d", tt.input, result, DefaultInferenceSteps)
			}
		})
	}
}

func TestParseGuidanceScale_ValidValues(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"1.0", 1.0},
		{"7.5", 7.5},
		{"30.0", 30.0},
		{"", DefaultGuidanceScale},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseGuidanceScale(tt.input)
			if result != tt.expected {
				t.Errorf("parseGuidanceScale(%q) = %.1f, expected %.1f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseGuidanceScale_InvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"not a number", "abc"},
		{"too low", "0.5"},
		{"too high", "35.0"},
		{"negative", "-1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseGuidanceScale(tt.input)
			if result != DefaultGuidanceScale {
				t.Errorf("parseGuidanceScale(%q) = %.1f, expected default %.1f", tt.input, result, DefaultGuidanceScale)
			}
		})
	}
}
