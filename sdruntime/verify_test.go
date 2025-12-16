package sdruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCalculateChecksum(t *testing.T) {
	// Create a temporary file with known content
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("Hello, World!")

	err := os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate expected checksum manually
	hasher := sha256.New()
	hasher.Write(testContent)
	expectedChecksum := hex.EncodeToString(hasher.Sum(nil))

	// Test CalculateChecksum
	actualChecksum, err := CalculateChecksum(testFile)
	if err != nil {
		t.Fatalf("CalculateChecksum returned error: %v", err)
	}

	if actualChecksum != expectedChecksum {
		t.Errorf("Checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}
}

func TestCalculateChecksum_NonExistentFile(t *testing.T) {
	_, err := CalculateChecksum("/nonexistent/path/to/file.txt")
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}

	if !errors.Is(err, ErrModelNotFound) {
		t.Errorf("Expected ErrModelNotFound, got: %v", err)
	}
}

func TestCalculateChecksum_EmptyFile(t *testing.T) {
	// Create an empty file
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.txt")

	err := os.WriteFile(emptyFile, []byte{}, 0644)
	if err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	// SHA256 of empty content
	hasher := sha256.New()
	expectedChecksum := hex.EncodeToString(hasher.Sum(nil))

	actualChecksum, err := CalculateChecksum(emptyFile)
	if err != nil {
		t.Fatalf("CalculateChecksum returned error: %v", err)
	}

	if actualChecksum != expectedChecksum {
		t.Errorf("Checksum mismatch for empty file: expected %s, got %s", expectedChecksum, actualChecksum)
	}
}

func TestVerifyModelChecksum_ValidChecksum(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-model.safetensors")
	testContent := []byte("test model content")

	err := os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Calculate checksum and register it
	checksum, err := CalculateChecksum(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}

	// Register the checksum
	RegisterModelChecksum("test-model.safetensors", checksum)
	defer delete(ModelChecksums, "test-model.safetensors") // Cleanup

	// Verify should pass
	err = VerifyModelChecksum(testFile)
	if err != nil {
		t.Errorf("VerifyModelChecksum failed for valid checksum: %v", err)
	}
}

func TestVerifyModelChecksum_Mismatch(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "corrupted-model.safetensors")
	testContent := []byte("actual content")

	err := os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Register a wrong checksum
	RegisterModelChecksum("corrupted-model.safetensors", "0000000000000000000000000000000000000000000000000000000000000000")
	defer delete(ModelChecksums, "corrupted-model.safetensors") // Cleanup

	// Verify should return ErrModelCorrupted
	err = VerifyModelChecksum(testFile)
	if err == nil {
		t.Fatal("Expected error for checksum mismatch, got nil")
	}

	if !errors.Is(err, ErrModelCorrupted) {
		t.Errorf("Expected ErrModelCorrupted, got: %v", err)
	}
}

func TestVerifyModelChecksum_NonExistentFile(t *testing.T) {
	err := VerifyModelChecksum("/nonexistent/path/to/model.safetensors")
	if err == nil {
		t.Fatal("Expected error for non-existent file, got nil")
	}

	if !errors.Is(err, ErrModelNotFound) {
		t.Errorf("Expected ErrModelNotFound, got: %v", err)
	}
}

func TestVerifyModelChecksum_UnregisteredModel(t *testing.T) {
	// Create a temporary file with unregistered model name
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "unregistered-model.safetensors")
	testContent := []byte("some content")

	err := os.WriteFile(testFile, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify should pass (skip verification for unregistered models)
	err = VerifyModelChecksum(testFile)
	if err != nil {
		t.Errorf("VerifyModelChecksum should skip unregistered models, got error: %v", err)
	}
}

func TestGetExpectedChecksum_Exists(t *testing.T) {
	// Register a test checksum
	testChecksum := "abcdef1234567890"
	RegisterModelChecksum("get-test-model.safetensors", testChecksum)
	defer delete(ModelChecksums, "get-test-model.safetensors")

	checksum, ok := GetExpectedChecksum("get-test-model.safetensors")
	if !ok {
		t.Error("Expected to find registered checksum")
	}

	if checksum != testChecksum {
		t.Errorf("Checksum mismatch: expected %s, got %s", testChecksum, checksum)
	}
}

func TestGetExpectedChecksum_NotExists(t *testing.T) {
	checksum, ok := GetExpectedChecksum("nonexistent-model.safetensors")
	if ok {
		t.Error("Expected not to find unregistered checksum")
	}

	if checksum != "" {
		t.Errorf("Expected empty string for missing checksum, got %s", checksum)
	}
}

func TestRegisterModelChecksum(t *testing.T) {
	modelName := "register-test-model.safetensors"
	testChecksum := "fedcba0987654321"

	// Ensure not already registered
	delete(ModelChecksums, modelName)

	// Register
	RegisterModelChecksum(modelName, testChecksum)
	defer delete(ModelChecksums, modelName)

	// Verify registration
	checksum, ok := GetExpectedChecksum(modelName)
	if !ok {
		t.Error("Failed to register checksum")
	}

	if checksum != testChecksum {
		t.Errorf("Registered checksum mismatch: expected %s, got %s", testChecksum, checksum)
	}
}

func TestIsModelCorrupted(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"ErrModelCorrupted", ErrModelCorrupted, true},
		{"wrapped ErrModelCorrupted", errors.Join(ErrModelCorrupted, errors.New("additional context")), true},
		{"ErrModelNotFound", ErrModelNotFound, false},
		{"other error", errors.New("some error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsModelCorrupted(tt.err)
			if result != tt.expected {
				t.Errorf("IsModelCorrupted(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestIsModelNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"ErrModelNotFound", ErrModelNotFound, true},
		{"wrapped ErrModelNotFound", errors.Join(ErrModelNotFound, errors.New("additional context")), true},
		{"ErrModelCorrupted", ErrModelCorrupted, false},
		{"other error", errors.New("some error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsModelNotFound(tt.err)
			if result != tt.expected {
				t.Errorf("IsModelNotFound(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestCalculateChecksum_LargeFile(t *testing.T) {
	// Create a file larger than the buffer size to test streaming
	tmpDir := t.TempDir()
	largeFile := filepath.Join(tmpDir, "large.bin")

	// Create a 100KB file (larger than 32KB buffer)
	largeContent := make([]byte, 100*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	err := os.WriteFile(largeFile, largeContent, 0644)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// Calculate expected checksum
	hasher := sha256.New()
	hasher.Write(largeContent)
	expectedChecksum := hex.EncodeToString(hasher.Sum(nil))

	// Test CalculateChecksum
	actualChecksum, err := CalculateChecksum(largeFile)
	if err != nil {
		t.Fatalf("CalculateChecksum returned error: %v", err)
	}

	if actualChecksum != expectedChecksum {
		t.Errorf("Checksum mismatch for large file: expected %s, got %s", expectedChecksum, actualChecksum)
	}
}
