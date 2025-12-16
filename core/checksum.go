package core

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// ComputeSHA256 computes the SHA256 hash of a file and returns it as a hexadecimal string.
// This is a pure function used for verifying downloaded model files and release artifacts.
//
// Parameters:
//   - filepath: absolute or relative path to the file to hash
//
// Returns:
//   - string: lowercase hexadecimal representation of the SHA256 hash (64 characters)
//   - error: if file cannot be opened or read
//
// Examples:
//   - ComputeSHA256("model.gguf") returns "a1b2c3d4...", nil for valid file
//   - ComputeSHA256("nonexistent.txt") returns "", error
//
// This is a pure function with deterministic output for any given file contents.
func ComputeSHA256(filepath string) (string, error) {
	if filepath == "" {
		return "", fmt.Errorf("filepath cannot be empty")
	}

	file, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %q: %w", filepath, err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to read file %q: %w", filepath, err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ComputeSHA256FromReader computes the SHA256 hash from an io.Reader.
// Useful for computing checksums without needing a file on disk.
//
// Parameters:
//   - r: an io.Reader providing the data to hash
//
// Returns:
//   - string: lowercase hexadecimal representation of the SHA256 hash (64 characters)
//   - error: if reading from the reader fails
//
// This is a pure function with deterministic output for any given input data.
func ComputeSHA256FromReader(r io.Reader) (string, error) {
	if r == nil {
		return "", fmt.Errorf("reader cannot be nil")
	}

	hasher := sha256.New()
	if _, err := io.Copy(hasher, r); err != nil {
		return "", fmt.Errorf("failed to read data: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ComputeSHA256FromBytes computes the SHA256 hash of a byte slice.
// Useful for computing checksums of in-memory data.
//
// Parameters:
//   - data: byte slice to hash
//
// Returns:
//   - string: lowercase hexadecimal representation of the SHA256 hash (64 characters)
//
// This is a pure function with deterministic output for any given input.
func ComputeSHA256FromBytes(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	return hex.EncodeToString(hasher.Sum(nil))
}

// VerifyChecksum computes the SHA256 hash of a file and compares it against an expected value.
// Comparison is case-insensitive (handles both uppercase and lowercase hex).
//
// Parameters:
//   - filepath: path to the file to verify
//   - expectedHash: expected SHA256 hash in hexadecimal format
//
// Returns:
//   - bool: true if the computed hash matches the expected hash
//   - error: if file cannot be read or the expected hash is invalid
//
// This is a pure function useful for validating downloaded model files.
func VerifyChecksum(filepath string, expectedHash string) (bool, error) {
	if expectedHash == "" {
		return false, fmt.Errorf("expected hash cannot be empty")
	}

	// Validate expected hash format (should be 64 hex characters for SHA256)
	if len(expectedHash) != 64 {
		return false, fmt.Errorf("invalid SHA256 hash length: expected 64 characters, got %d", len(expectedHash))
	}

	// Verify it's valid hex
	if _, err := hex.DecodeString(expectedHash); err != nil {
		return false, fmt.Errorf("invalid SHA256 hash format: %w", err)
	}

	computed, err := ComputeSHA256(filepath)
	if err != nil {
		return false, err
	}

	// Case-insensitive comparison using lowercase
	return toLowerHex(computed) == toLowerHex(expectedHash), nil
}

// toLowerHex converts a hex string to lowercase without importing strings package.
func toLowerHex(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'F' {
			c += 32 // Convert to lowercase
		}
		b[i] = c
	}
	return string(b)
}
