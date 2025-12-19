package core

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGetDiskSpace(t *testing.T) {
	// Test with current directory (should always work)
	info, err := GetDiskSpace(".")
	if err != nil {
		t.Fatalf("GetDiskSpace(\".\") error: %v", err)
	}

	// Basic sanity checks
	if info.Total <= 0 {
		t.Errorf("Total = %d, want > 0", info.Total)
	}
	if info.Free < 0 {
		t.Errorf("Free = %d, want >= 0", info.Free)
	}
	if info.Used < 0 {
		t.Errorf("Used = %d, want >= 0", info.Used)
	}
	if info.Total != info.Free+info.Used {
		t.Errorf("Total (%d) != Free (%d) + Used (%d)", info.Total, info.Free, info.Used)
	}
	if info.UsedPercent < 0 || info.UsedPercent > 100 {
		t.Errorf("UsedPercent = %.2f, want 0-100", info.UsedPercent)
	}

	// Check formatted values are not empty
	if info.TotalFormatted == "" {
		t.Error("TotalFormatted is empty")
	}
	if info.FreeFormatted == "" {
		t.Error("FreeFormatted is empty")
	}
	if info.UsedFormatted == "" {
		t.Error("UsedFormatted is empty")
	}
}

func TestGetDiskSpace_WithFile(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "diskspace_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tmpPath)

	// GetDiskSpace should work with a file path (uses parent directory)
	info, err := GetDiskSpace(tmpPath)
	if err != nil {
		t.Fatalf("GetDiskSpace(%q) error: %v", tmpPath, err)
	}

	if info.Total <= 0 {
		t.Errorf("Total = %d, want > 0", info.Total)
	}
}

func TestGetDiskSpace_WithDirectory(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "diskspace_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	info, err := GetDiskSpace(tmpDir)
	if err != nil {
		t.Fatalf("GetDiskSpace(%q) error: %v", tmpDir, err)
	}

	if info.Total <= 0 {
		t.Errorf("Total = %d, want > 0", info.Total)
	}
	if info.Path != tmpDir {
		t.Errorf("Path = %q, want %q", info.Path, tmpDir)
	}
}

func TestGetDiskSpace_RootPath(t *testing.T) {
	// Test with root path
	var rootPath string
	if os.PathSeparator == '/' {
		rootPath = "/"
	} else {
		rootPath = "C:\\"
	}

	info, err := GetDiskSpace(rootPath)
	if err != nil {
		t.Fatalf("GetDiskSpace(%q) error: %v", rootPath, err)
	}

	if info.Total <= 0 {
		t.Errorf("Total = %d, want > 0", info.Total)
	}
}

func TestGetDiskSpace_NonExistentPath(t *testing.T) {
	// Test with a non-existent path - should try parent directory
	tmpDir, err := os.MkdirTemp("", "diskspace_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	nonExistentPath := filepath.Join(tmpDir, "does_not_exist", "subdir")

	// This should work by walking up to the existing parent
	info, err := GetDiskSpace(nonExistentPath)
	if err != nil {
		t.Fatalf("GetDiskSpace(%q) error: %v", nonExistentPath, err)
	}

	if info.Total <= 0 {
		t.Errorf("Total = %d, want > 0", info.Total)
	}
}

func TestCheckDiskSpace_SufficientSpace(t *testing.T) {
	// Get current disk info
	info, err := GetDiskSpace(".")
	if err != nil {
		t.Fatalf("GetDiskSpace error: %v", err)
	}

	// Request less than available
	halfFree := info.Free / 2
	err = CheckDiskSpace(".", halfFree)
	if err != nil {
		t.Errorf("CheckDiskSpace(half of free) error: %v", err)
	}

	// Request 0 bytes (should always succeed)
	err = CheckDiskSpace(".", 0)
	if err != nil {
		t.Errorf("CheckDiskSpace(0) error: %v", err)
	}
}

func TestCheckDiskSpace_InsufficientSpace(t *testing.T) {
	// Get current disk info
	info, err := GetDiskSpace(".")
	if err != nil {
		t.Fatalf("GetDiskSpace error: %v", err)
	}

	// Request more than available
	doubleFree := info.Free * 2
	err = CheckDiskSpace(".", doubleFree)
	if err == nil {
		t.Error("CheckDiskSpace(double of free) should error, but didn't")
	}

	// Verify it's a core.DiskSpaceError
	var diskErr *core.DiskSpaceError
	if !errors.As(err, &diskErr) {
		t.Errorf("Error type = %T, want *DiskSpaceError", err)
	} else {
		if diskErr.Required != doubleFree {
			t.Errorf("Required = %d, want %d", diskErr.Required, doubleFree)
		}
		if diskErr.Available != info.Free {
			t.Errorf("Available = %d, want %d", diskErr.Available, info.Free)
		}
	}
}

func TestCheckDiskSpaceForModel(t *testing.T) {
	// Get current disk info
	info, err := GetDiskSpace(".")
	if err != nil {
		t.Fatalf("GetDiskSpace error: %v", err)
	}

	// Test with small model size that should fit
	smallSize := info.Free / 4
	err = CheckDiskSpaceForModel(".", smallSize, 10)
	if err != nil {
		t.Errorf("CheckDiskSpaceForModel with small size error: %v", err)
	}

	// Test with oversized model
	hugeSize := info.Total * 2
	err = CheckDiskSpaceForModel(".", hugeSize, 10)
	if err == nil {
		t.Error("CheckDiskSpaceForModel with huge size should error")
	}
}

func TestCheckDiskSpaceForDefaultModel(t *testing.T) {
	// This test may pass or fail depending on available disk space
	// We just verify it doesn't panic and returns a sensible error type
	err := CheckDiskSpaceForDefaultModel(".")
	if err != nil {
		var diskErr *core.DiskSpaceError
		if !errors.As(err, &diskErr) {
			t.Errorf("Unexpected error type: %T", err)
		}
	}
}

func TestDiskSpaceError(t *testing.T) {
	err := &core.DiskSpaceError{
		Path:      "/some/path",
		Required:  core.BytesPerGB * 8,
		Available: core.BytesPerGB * 2,
		Message:   "insufficient disk space",
	}

	// Test Error() method
	if err.Error() != "insufficient disk space" {
		t.Errorf("Error() = %q, want %q", err.Error(), "insufficient disk space")
	}

	// Test that it implements error interface
	var _ error = err
}

func TestGetParentPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"unix root", "/", ""},
		{"unix single level", "/foo", "/"},
		{"unix two levels", "/foo/bar", "/foo"},
		{"unix three levels", "/foo/bar/baz", "/foo/bar"},
		{"current dir", ".", ""},
		{"relative path", "foo/bar", "foo"},
	}

	// Only run Unix tests on Unix
	if os.PathSeparator == '/' {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := getParentPath(tt.path)
				if result != tt.expected {
					t.Errorf("getParentPath(%q) = %q, want %q", tt.path, result, tt.expected)
				}
			})
		}
	}
}

func TestDiskSpaceConstants(t *testing.T) {
	// Verify constants are reasonable
	if DefaultModelSizeBytes != 8*core.BytesPerGB {
		t.Errorf("DefaultModelSizeBytes = %d, want %d", DefaultModelSizeBytes, 8*core.BytesPerGB)
	}
	if core.DefaultBufferPercent != 10 {
		t.Errorf("core.DefaultBufferPercent = %d, want 10", core.DefaultBufferPercent)
	}
}

// Benchmark tests
func BenchmarkGetDiskSpace(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GetDiskSpace(".")
	}
}

func BenchmarkCheckDiskSpace(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = CheckDiskSpace(".", core.BytesPerGB)
	}
}
