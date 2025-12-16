package sdruntime

import (
	"errors"
	"testing"
)

func TestValidateParams_ValidInput(t *testing.T) {
	params := GenerateParams{
		Prompt:         "a beautiful sunset over the ocean",
		NegativePrompt: "blurry, low quality",
		Width:          512,
		Height:         512,
		Steps:          20,
		CFGScale:       7.5,
		Seed:           12345,
	}

	err := ValidateParams(params)
	if err != nil {
		t.Errorf("expected no error for valid params, got: %v", err)
	}
}

func TestValidateParams_InvalidWidth(t *testing.T) {
	tests := []struct {
		name  string
		width int
	}{
		{"too small", 64},
		{"too large", 4096},
		{"not divisible by 8", 513},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := GenerateParams{
				Prompt:   "test prompt",
				Width:    tt.width,
				Height:   512,
				Steps:    20,
				CFGScale: 7.5,
			}

			err := ValidateParams(params)
			if err == nil {
				t.Errorf("expected error for width %d", tt.width)
			}
			if !errors.Is(err, ErrInvalidParams) {
				t.Errorf("expected ErrInvalidParams, got: %v", err)
			}
		})
	}
}

func TestValidateParams_InvalidHeight(t *testing.T) {
	tests := []struct {
		name   string
		height int
	}{
		{"too small", 100},
		{"too large", 3000},
		{"not divisible by 8", 515},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := GenerateParams{
				Prompt:   "test prompt",
				Width:    512,
				Height:   tt.height,
				Steps:    20,
				CFGScale: 7.5,
			}

			err := ValidateParams(params)
			if err == nil {
				t.Errorf("expected error for height %d", tt.height)
			}
			if !errors.Is(err, ErrInvalidParams) {
				t.Errorf("expected ErrInvalidParams, got: %v", err)
			}
		})
	}
}

func TestValidateParams_InvalidSteps(t *testing.T) {
	tests := []struct {
		name  string
		steps int
	}{
		{"zero", 0},
		{"negative", -5},
		{"too high", 150},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := GenerateParams{
				Prompt:   "test prompt",
				Width:    512,
				Height:   512,
				Steps:    tt.steps,
				CFGScale: 7.5,
			}

			err := ValidateParams(params)
			if err == nil {
				t.Errorf("expected error for steps %d", tt.steps)
			}
			if !errors.Is(err, ErrInvalidParams) {
				t.Errorf("expected ErrInvalidParams, got: %v", err)
			}
		})
	}
}

func TestValidateParams_InvalidCFGScale(t *testing.T) {
	tests := []struct {
		name  string
		scale float64
	}{
		{"too low", 0.5},
		{"too high", 35.0},
		{"negative", -1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := GenerateParams{
				Prompt:   "test prompt",
				Width:    512,
				Height:   512,
				Steps:    20,
				CFGScale: tt.scale,
			}

			err := ValidateParams(params)
			if err == nil {
				t.Errorf("expected error for CFGScale %.2f", tt.scale)
			}
			if !errors.Is(err, ErrInvalidParams) {
				t.Errorf("expected ErrInvalidParams, got: %v", err)
			}
		})
	}
}

func TestValidateParams_EmptyPrompt(t *testing.T) {
	params := GenerateParams{
		Prompt:   "",
		Width:    512,
		Height:   512,
		Steps:    20,
		CFGScale: 7.5,
	}

	err := ValidateParams(params)
	if err == nil {
		t.Error("expected error for empty prompt")
	}
	if !errors.Is(err, ErrInvalidPrompt) {
		t.Errorf("expected ErrInvalidPrompt, got: %v", err)
	}
}

func TestValidateParams_BoundaryValues(t *testing.T) {
	tests := []struct {
		name   string
		params GenerateParams
	}{
		{
			"minimum values",
			GenerateParams{
				Prompt:   "x",
				Width:    128,
				Height:   128,
				Steps:    1,
				CFGScale: 1.0,
			},
		},
		{
			"maximum values",
			GenerateParams{
				Prompt:   "x",
				Width:    2048,
				Height:   2048,
				Steps:    100,
				CFGScale: 30.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParams(tt.params)
			if err != nil {
				t.Errorf("expected boundary values to be valid, got: %v", err)
			}
		})
	}
}
