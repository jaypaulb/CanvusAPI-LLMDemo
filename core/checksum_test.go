package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestComputeSHA256FromBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "empty data",
			input:    []byte{},
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "hello world",
			input:    []byte("hello world"),
			expected: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:     "single byte",
			input:    []byte{0x00},
			expected: "6e340b9cffb37a989ca544e6bb780a2c78901d3fb33738768511a30617afa01d",
		},
		{
			name:     "binary data",
			input:    []byte{0xDE, 0xAD, 0xBE, 0xEF},
			expected: "5f78c33274e43fa9de5659265c1d917e25c03722dcb0b8d27db8d5feaa813953",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeSHA256FromBytes(tt.input)
			if result != tt.expected {
				t.Errorf("ComputeSHA256FromBytes() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestComputeSHA256FromReader(t *testing.T) {
	tests := []struct {
		name        string
		input       []byte
		expected    string
		expectError bool
	}{
		{
			name:        "empty data",
			input:       []byte{},
			expected:    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			expectError: false,
		},
		{
			name:        "hello world",
			input:       []byte("hello world"),
			expected:    "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.input)
			result, err := ComputeSHA256FromReader(reader)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("ComputeSHA256FromReader() = %q, want %q", result, tt.expected)
			}
		})
	}

	// Test nil reader
	t.Run("nil reader", func(t *testing.T) {
		_, err := ComputeSHA256FromReader(nil)
		if err == nil {
			t.Error("expected error for nil reader, got nil")
		}
	})
}

func TestComputeSHA256(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "checksum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test cases with actual files
	tests := []struct {
		name        string
		content     []byte
		expected    string
		expectError bool
	}{
		{
			name:        "empty file",
			content:     []byte{},
			expected:    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			expectError: false,
		},
		{
			name:        "text file",
			content:     []byte("hello world"),
			expected:    "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tmpDir, tt.name+".bin")
			if err := os.WriteFile(testFile, tt.content, 0644); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			result, err := ComputeSHA256(testFile)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("ComputeSHA256() = %q, want %q", result, tt.expected)
			}
		})
	}

	// Test error cases
	t.Run("nonexistent file", func(t *testing.T) {
		_, err := ComputeSHA256(filepath.Join(tmpDir, "nonexistent.txt"))
		if err == nil {
			t.Error("expected error for nonexistent file, got nil")
		}
	})

	t.Run("empty filepath", func(t *testing.T) {
		_, err := ComputeSHA256("")
		if err == nil {
			t.Error("expected error for empty filepath, got nil")
		}
	})
}

func TestVerifyChecksum(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "checksum_verify_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testContent := []byte("hello world")
	testFile := filepath.Join(tmpDir, "test.bin")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	correctHash := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	wrongHash := "0000000000000000000000000000000000000000000000000000000000000000"

	tests := []struct {
		name         string
		filepath     string
		expectedHash string
		wantMatch    bool
		expectError  bool
	}{
		{
			name:         "correct hash lowercase",
			filepath:     testFile,
			expectedHash: correctHash,
			wantMatch:    true,
			expectError:  false,
		},
		{
			name:         "correct hash uppercase",
			filepath:     testFile,
			expectedHash: "B94D27B9934D3E08A52E52D7DA7DABFAC484EFE37A5380EE9088F7ACE2EFCDE9",
			wantMatch:    true,
			expectError:  false,
		},
		{
			name:         "wrong hash",
			filepath:     testFile,
			expectedHash: wrongHash,
			wantMatch:    false,
			expectError:  false,
		},
		{
			name:         "empty expected hash",
			filepath:     testFile,
			expectedHash: "",
			wantMatch:    false,
			expectError:  true,
		},
		{
			name:         "invalid hash length",
			filepath:     testFile,
			expectedHash: "abc123",
			wantMatch:    false,
			expectError:  true,
		},
		{
			name:         "invalid hex characters",
			filepath:     testFile,
			expectedHash: "zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
			wantMatch:    false,
			expectError:  true,
		},
		{
			name:         "nonexistent file",
			filepath:     filepath.Join(tmpDir, "nonexistent.txt"),
			expectedHash: correctHash,
			wantMatch:    false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := VerifyChecksum(tt.filepath, tt.expectedHash)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if match != tt.wantMatch {
				t.Errorf("VerifyChecksum() = %v, want %v", match, tt.wantMatch)
			}
		})
	}
}

func TestToLowerHex(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"abc123", "abc123"},
		{"ABC123", "abc123"},
		{"AbCdEf", "abcdef"},
		{"0123456789ABCDEF", "0123456789abcdef"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toLowerHex(tt.input)
			if result != tt.expected {
				t.Errorf("toLowerHex(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
