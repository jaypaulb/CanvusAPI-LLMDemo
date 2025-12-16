package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateChecksumFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "release_checksum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testContent := []byte("hello world")
	testFile := filepath.Join(tmpDir, "test-artifact.tar.gz")
	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	expectedChecksum := "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	t.Run("generates checksum file", func(t *testing.T) {
		checksumFile, checksum, err := GenerateChecksumFile(testFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify checksum value
		if checksum != expectedChecksum {
			t.Errorf("checksum = %q, want %q", checksum, expectedChecksum)
		}

		// Verify checksum file path
		expectedPath := testFile + ".sha256"
		if checksumFile != expectedPath {
			t.Errorf("checksumFile = %q, want %q", checksumFile, expectedPath)
		}

		// Verify checksum file exists and has correct content
		content, err := os.ReadFile(checksumFile)
		if err != nil {
			t.Fatalf("failed to read checksum file: %v", err)
		}

		expectedContent := expectedChecksum + "  test-artifact.tar.gz\n"
		if string(content) != expectedContent {
			t.Errorf("checksum file content = %q, want %q", string(content), expectedContent)
		}
	})

	t.Run("empty path error", func(t *testing.T) {
		_, _, err := GenerateChecksumFile("")
		if err == nil {
			t.Error("expected error for empty path, got nil")
		}
	})

	t.Run("nonexistent file error", func(t *testing.T) {
		_, _, err := GenerateChecksumFile(filepath.Join(tmpDir, "nonexistent.bin"))
		if err == nil {
			t.Error("expected error for nonexistent file, got nil")
		}
	})
}

func TestFormatChecksumLine(t *testing.T) {
	tests := []struct {
		name     string
		checksum string
		filename string
		expected string
	}{
		{
			name:     "standard format",
			checksum: "abc123def456",
			filename: "file.tar.gz",
			expected: "abc123def456  file.tar.gz",
		},
		{
			name:     "empty filename",
			checksum: "abc123",
			filename: "",
			expected: "abc123  ",
		},
		{
			name:     "filename with spaces",
			checksum: "abc123",
			filename: "my file.zip",
			expected: "abc123  my file.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatChecksumLine(tt.checksum, tt.filename)
			if result != tt.expected {
				t.Errorf("FormatChecksumLine(%q, %q) = %q, want %q",
					tt.checksum, tt.filename, result, tt.expected)
			}
		})
	}
}

func TestGenerateCombinedChecksumFile(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "combined_checksum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	files := map[string][]byte{
		"file1.bin": []byte("content1"),
		"file2.bin": []byte("content2"),
	}

	var filePaths []string
	for name, content := range files {
		fp := filepath.Join(tmpDir, name)
		if err := os.WriteFile(fp, content, 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
		filePaths = append(filePaths, fp)
	}

	outputPath := filepath.Join(tmpDir, "CHECKSUMS.sha256")

	t.Run("generates combined file", func(t *testing.T) {
		checksums, err := GenerateCombinedChecksumFile(outputPath, filePaths)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify we got checksums for all files
		if len(checksums) != len(files) {
			t.Errorf("got %d checksums, want %d", len(checksums), len(files))
		}

		// Verify output file exists
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		// Verify content has correct number of lines
		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		if len(lines) != len(files) {
			t.Errorf("got %d lines, want %d", len(lines), len(files))
		}

		// Verify each line has correct format (hash + two spaces + filename)
		for _, line := range lines {
			parts := strings.SplitN(line, "  ", 2)
			if len(parts) != 2 {
				t.Errorf("invalid line format: %q", line)
				continue
			}
			if len(parts[0]) != 64 {
				t.Errorf("invalid hash length in line: %q", line)
			}
		}
	})

	t.Run("empty output path error", func(t *testing.T) {
		_, err := GenerateCombinedChecksumFile("", filePaths)
		if err == nil {
			t.Error("expected error for empty output path, got nil")
		}
	})

	t.Run("no files error", func(t *testing.T) {
		_, err := GenerateCombinedChecksumFile(outputPath, []string{})
		if err == nil {
			t.Error("expected error for empty file list, got nil")
		}
	})
}

func TestGenerateWindowsChecksumCommand(t *testing.T) {
	result := GenerateWindowsChecksumCommand("path/to/file.exe")
	if !strings.Contains(result, "certutil") {
		t.Errorf("command should contain certutil: %q", result)
	}
	if !strings.Contains(result, "SHA256") {
		t.Errorf("command should contain SHA256: %q", result)
	}
	if !strings.Contains(result, "file.exe") {
		t.Errorf("command should contain filename: %q", result)
	}
}

func TestGenerateLinuxChecksumCommand(t *testing.T) {
	result := GenerateLinuxChecksumCommand("path/to/file.tar.gz")
	if !strings.Contains(result, "sha256sum") {
		t.Errorf("command should contain sha256sum: %q", result)
	}
	if !strings.Contains(result, "file.tar.gz") {
		t.Errorf("command should contain filename: %q", result)
	}
}

func TestGenerateMacChecksumCommand(t *testing.T) {
	result := GenerateMacChecksumCommand("path/to/file.dmg")
	if !strings.Contains(result, "shasum") {
		t.Errorf("command should contain shasum: %q", result)
	}
	if !strings.Contains(result, "-a 256") {
		t.Errorf("command should contain -a 256: %q", result)
	}
	if !strings.Contains(result, "file.dmg") {
		t.Errorf("command should contain filename: %q", result)
	}
}

func TestGenerateChecksumFiles(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "multi_checksum_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.bin")
	file2 := filepath.Join(tmpDir, "file2.bin")
	nonexistent := filepath.Join(tmpDir, "nonexistent.bin")

	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}
	if err := os.WriteFile(file2, []byte("content2"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	t.Run("processes all files", func(t *testing.T) {
		results, err := GenerateChecksumFiles([]string{file1, file2})
		// Should return nil error when all succeed
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if len(results) != 2 {
			t.Errorf("got %d results, want 2", len(results))
		}

		for fp, result := range results {
			if result.Error != nil {
				t.Errorf("unexpected error for %q: %v", fp, result.Error)
			}
			if result.Checksum == "" {
				t.Errorf("empty checksum for %q", fp)
			}
		}
	})

	t.Run("continues on error", func(t *testing.T) {
		results, err := GenerateChecksumFiles([]string{file1, nonexistent, file2})
		// Should return the first error
		if err == nil {
			t.Error("expected error for nonexistent file, got nil")
		}

		// Should still have results for all files
		if len(results) != 3 {
			t.Errorf("got %d results, want 3", len(results))
		}

		// Check that valid files succeeded
		if results[file1].Error != nil {
			t.Errorf("unexpected error for file1: %v", results[file1].Error)
		}
		if results[file2].Error != nil {
			t.Errorf("unexpected error for file2: %v", results[file2].Error)
		}
		// Check that nonexistent file failed
		if results[nonexistent].Error == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}
