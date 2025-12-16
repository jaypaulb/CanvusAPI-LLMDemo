package core

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetDataDirectory_ReturnsNonEmpty(t *testing.T) {
	dir := GetDataDirectory()
	if dir == "" {
		t.Error("GetDataDirectory() returned empty string")
	}
}

func TestGetDataDirectory_PlatformAppropriate(t *testing.T) {
	dir := GetDataDirectory()

	switch runtime.GOOS {
	case "windows":
		// Windows: should contain CanvusLocalLLM
		if !strings.Contains(dir, AppName) {
			t.Errorf("Windows path %q should contain %q", dir, AppName)
		}
	default:
		// Unix-like: should end with .canvuslocallm
		if !strings.HasSuffix(dir, ".canvuslocallm") {
			t.Errorf("Unix path %q should end with .canvuslocallm", dir)
		}
	}
}

func TestGetDataFilePath_JoinsCorrectly(t *testing.T) {
	filename := "test.db"
	path := GetDataFilePath(filename)

	expectedSuffix := filepath.Join(filepath.Base(GetDataDirectory()), filename)
	if !strings.HasSuffix(path, expectedSuffix) {
		t.Errorf("GetDataFilePath(%q) = %q, should end with %q", filename, path, expectedSuffix)
	}
}
