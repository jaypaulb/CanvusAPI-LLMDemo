// Package ocrprocessor provides OCR (Optical Character Recognition) functionality
// for CanvusLocalLLM using Google Cloud Vision API.
//
// atoms.go contains pure validation and utility functions with no dependencies.
package ocrprocessor

import (
	"errors"
	"regexp"
	"strings"
)

// Common validation errors for OCR processing.
var (
	// ErrEmptyAPIKey indicates that the API key is empty or contains only whitespace.
	ErrEmptyAPIKey = errors.New("API key is empty")

	// ErrInvalidAPIKeyFormat indicates the API key does not match expected format.
	ErrInvalidAPIKeyFormat = errors.New("API key has invalid format")

	// ErrAPIKeyTooShort indicates the API key is shorter than minimum length.
	ErrAPIKeyTooShort = errors.New("API key is too short (minimum 20 characters)")

	// ErrAPIKeyTooLong indicates the API key exceeds maximum length.
	ErrAPIKeyTooLong = errors.New("API key is too long (maximum 100 characters)")
)

// googleAPIKeyPattern matches the format of Google API keys.
// Google API keys are typically 39 characters, containing alphanumeric chars,
// underscores, and dashes. Pattern: AIza[0-9A-Za-z_-]{35}
var googleAPIKeyPattern = regexp.MustCompile(`^AIza[0-9A-Za-z_-]{35}$`)

// ValidateGoogleAPIKey validates a Google Cloud API key format.
// It checks for:
//   - Non-empty value
//   - Minimum/maximum length (20-100 chars as a reasonable range)
//   - Expected Google API key pattern (AIza prefix + 35 chars)
//
// This is a pure function with no dependencies - it only performs string validation.
//
// Returns nil if the API key is valid, or an appropriate error if not.
//
// Example:
//
//	err := ValidateGoogleAPIKey("AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY")  // nil
//	err := ValidateGoogleAPIKey("")                                          // ErrEmptyAPIKey
//	err := ValidateGoogleAPIKey("short")                                     // ErrAPIKeyTooShort
func ValidateGoogleAPIKey(apiKey string) error {
	// Trim whitespace
	trimmed := strings.TrimSpace(apiKey)

	// Check for empty
	if trimmed == "" {
		return ErrEmptyAPIKey
	}

	// Check minimum length
	if len(trimmed) < 20 {
		return ErrAPIKeyTooShort
	}

	// Check maximum length
	if len(trimmed) > 100 {
		return ErrAPIKeyTooLong
	}

	// Check pattern (optional - only if it looks like it should be a Google API key)
	// If it starts with "AIza", it must match the full pattern
	if strings.HasPrefix(trimmed, "AIza") && !googleAPIKeyPattern.MatchString(trimmed) {
		return ErrInvalidAPIKeyFormat
	}

	return nil
}

// IsGoogleAPIKey checks if the given string appears to be a Google API key.
// It performs a pattern match against the expected Google API key format.
//
// This is a pure function with no dependencies - it only performs pattern matching.
//
// Returns true if the string matches Google API key format (AIza prefix + 35 chars).
//
// Example:
//
//	IsGoogleAPIKey("AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY")  // true
//	IsGoogleAPIKey("sk-1234567890abcdef")                       // false
//	IsGoogleAPIKey("")                                           // false
func IsGoogleAPIKey(s string) bool {
	return googleAPIKeyPattern.MatchString(strings.TrimSpace(s))
}

// SanitizeAPIKey removes leading/trailing whitespace and validates the key.
// If validation fails, it returns an empty string and the validation error.
// If successful, it returns the sanitized key and nil error.
//
// This is a pure function with no dependencies.
//
// Example:
//
//	key, err := SanitizeAPIKey("  AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY  ")
//	// key = "AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY", err = nil
func SanitizeAPIKey(apiKey string) (string, error) {
	trimmed := strings.TrimSpace(apiKey)
	if err := ValidateGoogleAPIKey(trimmed); err != nil {
		return "", err
	}
	return trimmed, nil
}

// MaskAPIKey returns a masked version of the API key for safe logging.
// Only the first 8 and last 4 characters are shown, with the middle replaced by asterisks.
// Returns "[empty]" for empty keys.
//
// This is a pure function with no dependencies.
//
// Example:
//
//	MaskAPIKey("AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY")  // "AIzaSyD-****MBWY"
//	MaskAPIKey("")                                         // "[empty]"
func MaskAPIKey(apiKey string) string {
	if apiKey == "" {
		return "[empty]"
	}
	if len(apiKey) <= 12 {
		// Too short to meaningfully mask, show first 4 chars only
		if len(apiKey) <= 4 {
			return strings.Repeat("*", len(apiKey))
		}
		return apiKey[:4] + strings.Repeat("*", len(apiKey)-4)
	}
	return apiKey[:8] + "****" + apiKey[len(apiKey)-4:]
}
