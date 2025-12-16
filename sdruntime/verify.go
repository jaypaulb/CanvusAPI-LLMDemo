// Package sdruntime provides Stable Diffusion image generation capabilities.
package sdruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ModelChecksums maps model filenames to their expected SHA256 checksums.
// These checksums are used to verify model integrity before loading.
var ModelChecksums = map[string]string{
	// Stable Diffusion v1.5 safetensors format
	// Source: https://huggingface.co/runwayml/stable-diffusion-v1-5
	"sd-v1-5.safetensors": "6ce0161689b3853acaa03779ec93eafe75a02f4ced659bee03f50797806fa2fa",

	// Add additional model checksums as needed
	// "model-name.safetensors": "expected_sha256_checksum",
}

// bufferSize defines the chunk size for streaming file reads during checksum calculation.
// 32KB provides a good balance between memory usage and I/O efficiency.
const bufferSize = 32 * 1024

// VerifyModelChecksum validates a model file's SHA256 checksum against known values.
// It composes the CalculateChecksum atom and GetExpectedChecksum lookup.
//
// Returns:
//   - nil if checksum matches
//   - ErrModelNotFound if file doesn't exist
//   - ErrModelCorrupted if checksum mismatch
//   - wrapped error for other I/O failures
func VerifyModelChecksum(modelPath string) error {
	// Check if file exists first
	if _, err := os.Stat(modelPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrModelNotFound, modelPath)
		}
		return fmt.Errorf("failed to access model file: %w", err)
	}

	// Extract model name from path
	modelName := filepath.Base(modelPath)

	// Look up expected checksum
	expectedChecksum, ok := GetExpectedChecksum(modelName)
	if !ok {
		// If no checksum registered, skip verification with warning
		// This allows loading models that haven't been added to the registry
		return nil
	}

	// Calculate actual checksum
	actualChecksum, err := CalculateChecksum(modelPath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Compare checksums
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("%w: expected %s, got %s", ErrModelCorrupted, expectedChecksum, actualChecksum)
	}

	return nil
}

// CalculateChecksum computes the SHA256 hash of a file.
// It streams the file in chunks to avoid loading the entire file into memory,
// making it suitable for large model files (several GB).
//
// Returns the lowercase hex-encoded SHA256 hash string.
func CalculateChecksum(filePath string) (string, error) {
	// Open file for reading
	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%w: %s", ErrModelNotFound, filePath)
		}
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create SHA256 hasher
	hasher := sha256.New()

	// Stream file through hasher using io.Copy for efficiency
	// io.Copy uses an internal buffer and is more efficient than manual buffering
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Return hex-encoded hash
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// GetExpectedChecksum returns the expected SHA256 checksum for a model file.
// The modelName should be just the filename (e.g., "sd-v1-5.safetensors").
//
// Returns the checksum string and true if found, or empty string and false if not registered.
func GetExpectedChecksum(modelName string) (string, bool) {
	checksum, ok := ModelChecksums[modelName]
	return checksum, ok
}

// RegisterModelChecksum adds or updates a model checksum in the registry.
// This allows runtime registration of additional models.
func RegisterModelChecksum(modelName, checksum string) {
	ModelChecksums[modelName] = checksum
}

// IsModelCorrupted checks if an error indicates model corruption.
// This is a convenience function for error handling.
func IsModelCorrupted(err error) bool {
	return errors.Is(err, ErrModelCorrupted)
}

// IsModelNotFound checks if an error indicates a missing model file.
// This is a convenience function for error handling.
func IsModelNotFound(err error) bool {
	return errors.Is(err, ErrModelNotFound)
}
