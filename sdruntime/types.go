package sdruntime

import "fmt"

// GenerateParams holds parameters for image generation.
type GenerateParams struct {
	Prompt         string  // Required: text description of the image to generate
	NegativePrompt string  // Optional: what to avoid in the image
	Width          int     // Image width in pixels (128-2048, must be divisible by 8)
	Height         int     // Image height in pixels (128-2048, must be divisible by 8)
	Steps          int     // Number of inference steps (1-100)
	CFGScale       float64 // Classifier-free guidance scale (1.0-30.0)
	Seed           int64   // Random seed for reproducibility (-1 for random)
}

// Parameter validation constants
const (
	MinImageSize     = 128
	MaxImageSize     = 2048
	ImageSizeMultple = 8 // Image dimensions must be divisible by this

	MinSteps = 1
	MaxSteps = 100

	MinCFGScale = 1.0
	MaxCFGScale = 30.0

	MaxPromptLength = 1000
)

// ValidateParams validates generation parameters and returns an error if invalid.
// This is a pure function with no side effects.
func ValidateParams(p GenerateParams) error {
	// Validate prompt
	if err := ValidatePrompt(p.Prompt); err != nil {
		return err
	}

	// Validate width
	if p.Width < MinImageSize || p.Width > MaxImageSize {
		return fmt.Errorf("%w: width %d must be between %d and %d",
			ErrInvalidParams, p.Width, MinImageSize, MaxImageSize)
	}
	if p.Width%ImageSizeMultple != 0 {
		return fmt.Errorf("%w: width %d must be divisible by %d",
			ErrInvalidParams, p.Width, ImageSizeMultple)
	}

	// Validate height
	if p.Height < MinImageSize || p.Height > MaxImageSize {
		return fmt.Errorf("%w: height %d must be between %d and %d",
			ErrInvalidParams, p.Height, MinImageSize, MaxImageSize)
	}
	if p.Height%ImageSizeMultple != 0 {
		return fmt.Errorf("%w: height %d must be divisible by %d",
			ErrInvalidParams, p.Height, ImageSizeMultple)
	}

	// Validate steps
	if p.Steps < MinSteps || p.Steps > MaxSteps {
		return fmt.Errorf("%w: steps %d must be between %d and %d",
			ErrInvalidParams, p.Steps, MinSteps, MaxSteps)
	}

	// Validate CFG scale
	if p.CFGScale < MinCFGScale || p.CFGScale > MaxCFGScale {
		return fmt.Errorf("%w: CFGScale %.2f must be between %.1f and %.1f",
			ErrInvalidParams, p.CFGScale, MinCFGScale, MaxCFGScale)
	}

	// Negative prompt is optional, but if provided, validate length
	if len(p.NegativePrompt) > MaxPromptLength {
		return fmt.Errorf("%w: negative prompt length %d exceeds maximum %d",
			ErrInvalidParams, len(p.NegativePrompt), MaxPromptLength)
	}

	return nil
}
