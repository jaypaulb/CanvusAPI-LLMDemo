package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckFileExists(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "validate_file_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a test directory
	testDir := filepath.Join(tmpDir, "testdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errType string
	}{
		{
			name:    "existing file",
			path:    testFile,
			wantErr: false,
		},
		{
			name:    "non-existent file",
			path:    filepath.Join(tmpDir, "nonexistent.txt"),
			wantErr: true,
			errType: "not found",
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
			errType: "empty",
		},
		{
			name:    "directory instead of file",
			path:    testDir,
			wantErr: true,
			errType: "directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := core.CheckFileExists(tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("core.CheckFileExists(%q) expected error but got nil", tt.path)
					return
				}
				// Verify it's a FileExistsError
				if _, ok := err.(*FileExistsError); !ok {
					t.Errorf("core.CheckFileExists(%q) expected *FileExistsError, got %T", tt.path, err)
				}
			} else {
				if err != nil {
					t.Errorf("core.CheckFileExists(%q) unexpected error: %v", tt.path, err)
				}
			}
		})
	}
}

func TestCheckEnvFileExists(t *testing.T) {
	// Save current working directory
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(originalWd)

	// Test 1: Directory without .env file
	t.Run("missing .env file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "env_test_missing")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change to temp dir: %v", err)
		}

		err = CheckEnvFileExists()
		if err == nil {
			t.Error("CheckEnvFileExists() expected error for missing .env, got nil")
		}
	})

	// Test 2: Directory with .env file
	t.Run("existing .env file", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "env_test_exists")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		envFile := filepath.Join(tmpDir, ".env")
		if err := os.WriteFile(envFile, []byte("TEST=value"), 0644); err != nil {
			t.Fatalf("Failed to create .env file: %v", err)
		}

		if err := os.Chdir(tmpDir); err != nil {
			t.Fatalf("Failed to change to temp dir: %v", err)
		}

		err = CheckEnvFileExists()
		if err != nil {
			t.Errorf("CheckEnvFileExists() unexpected error: %v", err)
		}
	})
}
