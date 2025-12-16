package sdruntime

import (
	"fmt"
	"strings"
)

// ValidatePrompt validates a prompt string for image generation.
// Returns an error if the prompt is invalid.
// This is a pure function with no side effects.
func ValidatePrompt(prompt string) error {
	// Check for empty prompt
	if strings.TrimSpace(prompt) == "" {
		return fmt.Errorf("%w: prompt cannot be empty", ErrInvalidPrompt)
	}

	// Check for null bytes (security concern for C interop)
	if strings.ContainsRune(prompt, '\x00') {
		return fmt.Errorf("%w: prompt contains null bytes", ErrInvalidPrompt)
	}

	// Check length
	if len(prompt) > MaxPromptLength {
		return fmt.Errorf("%w: prompt length %d exceeds maximum %d",
			ErrInvalidPrompt, len(prompt), MaxPromptLength)
	}

	return nil
}

// SanitizePrompt cleans a prompt by trimming whitespace.
// This is a pure function that transforms input to output.
func SanitizePrompt(prompt string) string {
	return strings.TrimSpace(prompt)
}
