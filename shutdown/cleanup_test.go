package shutdown

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

func TestCleanupDownloads_RemovesTempFiles(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create a temp directory for testing
	tempDir := t.TempDir()

	// Create some temp_* files
	tempFiles := []string{
		"temp_abc123.pdf",
		"temp_def456.jpg",
		"temp_ghi789.txt",
	}
	for _, f := range tempFiles {
		path := filepath.Join(tempDir, f)
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", f, err)
		}
	}

	// Create a non-temp file that should NOT be deleted
	keepFile := filepath.Join(tempDir, "keep_me.txt")
	if err := os.WriteFile(keepFile, []byte("keep this"), 0644); err != nil {
		t.Fatalf("Failed to create keep file: %v", err)
	}

	// Execute cleanup
	cleanupFn := CleanupDownloads(logger, tempDir)
	err := cleanupFn(context.Background())
	if err != nil {
		t.Errorf("CleanupDownloads returned unexpected error: %v", err)
	}

	// Verify temp files are deleted
	for _, f := range tempFiles {
		path := filepath.Join(tempDir, f)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("Temp file %s should have been deleted", f)
		}
	}

	// Verify non-temp file still exists
	if _, err := os.Stat(keepFile); os.IsNotExist(err) {
		t.Error("Non-temp file should not have been deleted")
	}
}

func TestCleanupDownloads_HandlesEmptyDirectory(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create an empty temp directory
	tempDir := t.TempDir()

	// Execute cleanup - should succeed without errors
	cleanupFn := CleanupDownloads(logger, tempDir)
	err := cleanupFn(context.Background())
	if err != nil {
		t.Errorf("CleanupDownloads on empty directory returned error: %v", err)
	}

	// Directory should still exist
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Error("Directory should still exist after cleanup")
	}
}

func TestCleanupDownloads_HandlesMissingDirectory(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Use a path that doesn't exist
	nonExistentDir := filepath.Join(t.TempDir(), "does_not_exist")

	// Execute cleanup - should succeed (filepath.Glob handles missing dirs gracefully)
	cleanupFn := CleanupDownloads(logger, nonExistentDir)
	err := cleanupFn(context.Background())
	if err != nil {
		t.Errorf("CleanupDownloads on missing directory returned error: %v", err)
	}
}

func TestCleanupDownloadsAndDir_RemovesDirectory(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create a temp directory for testing
	parentDir := t.TempDir()
	downloadsDir := filepath.Join(parentDir, "downloads")
	if err := os.Mkdir(downloadsDir, 0755); err != nil {
		t.Fatalf("Failed to create downloads directory: %v", err)
	}

	// Create some files in the directory
	files := []string{"temp_abc.pdf", "other_file.txt"}
	for _, f := range files {
		path := filepath.Join(downloadsDir, f)
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", f, err)
		}
	}

	// Execute cleanup
	cleanupFn := CleanupDownloadsAndDir(logger, downloadsDir)
	err := cleanupFn(context.Background())
	if err != nil {
		t.Errorf("CleanupDownloadsAndDir returned unexpected error: %v", err)
	}

	// Verify directory is deleted
	if _, err := os.Stat(downloadsDir); !os.IsNotExist(err) {
		t.Error("Downloads directory should have been deleted")
	}

	// Parent directory should still exist
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		t.Error("Parent directory should still exist")
	}
}

func TestCleanupDownloadsAndDir_HandlesMissingDirectory(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Use a path that doesn't exist
	nonExistentDir := filepath.Join(t.TempDir(), "does_not_exist")

	// Execute cleanup - should succeed
	cleanupFn := CleanupDownloadsAndDir(logger, nonExistentDir)
	err := cleanupFn(context.Background())
	if err != nil {
		t.Errorf("CleanupDownloadsAndDir on missing directory returned error: %v", err)
	}
}

func TestCleanupDownloads_RespectsContextCancellation(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create a temp directory with many files
	tempDir := t.TempDir()
	for i := 0; i < 10; i++ {
		path := filepath.Join(tempDir, "temp_file_"+string(rune('a'+i))+".txt")
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Execute cleanup with cancelled context
	cleanupFn := CleanupDownloads(logger, tempDir)
	err := cleanupFn(ctx)

	// Should return nil (cleanup doesn't block on cancellation)
	if err != nil {
		t.Errorf("CleanupDownloads with cancelled context returned error: %v", err)
	}
}

func TestCleanupDownloadsAndDir_RespectsContextCancellation(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create a temp directory
	parentDir := t.TempDir()
	downloadsDir := filepath.Join(parentDir, "downloads")
	if err := os.Mkdir(downloadsDir, 0755); err != nil {
		t.Fatalf("Failed to create downloads directory: %v", err)
	}

	// Create some files
	for i := 0; i < 5; i++ {
		path := filepath.Join(downloadsDir, "temp_file_"+string(rune('a'+i))+".txt")
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Create an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Execute cleanup with cancelled context
	cleanupFn := CleanupDownloadsAndDir(logger, downloadsDir)
	err := cleanupFn(ctx)

	// Should return nil (cleanup doesn't block on cancellation)
	if err != nil {
		t.Errorf("CleanupDownloadsAndDir with cancelled context returned error: %v", err)
	}
}

func TestCleanupDownloads_ReturnsShutdownFunc(t *testing.T) {
	logger := zaptest.NewLogger(t)
	tempDir := t.TempDir()

	// Verify return type is compatible with core.ShutdownFunc
	fn := CleanupDownloads(logger, tempDir)

	// Should be callable with context and return error
	err := fn(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCleanupDownloadsAndDir_ReturnsShutdownFunc(t *testing.T) {
	logger := zaptest.NewLogger(t)
	parentDir := t.TempDir()
	downloadsDir := filepath.Join(parentDir, "downloads")
	if err := os.Mkdir(downloadsDir, 0755); err != nil {
		t.Fatalf("Failed to create downloads directory: %v", err)
	}

	// Verify return type is compatible with core.ShutdownFunc
	fn := CleanupDownloadsAndDir(logger, downloadsDir)

	// Should be callable with context and return error
	err := fn(context.Background())
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCleanupDownloads_HandlesSubdirectories(t *testing.T) {
	logger := zaptest.NewLogger(t)
	tempDir := t.TempDir()

	// Create a temp_* subdirectory (should NOT be removed by CleanupDownloads)
	subDir := filepath.Join(tempDir, "temp_subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create a file inside the subdirectory
	subFile := filepath.Join(subDir, "file.txt")
	if err := os.WriteFile(subFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file in subdirectory: %v", err)
	}

	// Create a regular temp file (should be removed)
	tempFile := filepath.Join(tempDir, "temp_file.txt")
	if err := os.WriteFile(tempFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Execute cleanup
	cleanupFn := CleanupDownloads(logger, tempDir)
	err := cleanupFn(context.Background())
	if err != nil {
		t.Errorf("CleanupDownloads returned error: %v", err)
	}

	// Regular temp file should be removed
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Error("Temp file should have been removed")
	}

	// Subdirectory should remain (os.Remove doesn't remove directories with contents)
	// This is expected behavior - we don't recursively delete directories
	if _, err := os.Stat(subDir); os.IsNotExist(err) {
		t.Error("Subdirectory should still exist (os.Remove doesn't delete non-empty dirs)")
	}
}

// ============================================================================
// Integration Tests - Testing with shutdown.Manager
// ============================================================================

func TestCleanupDownloads_IntegrationWithManager(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create a temp directory
	tempDir := t.TempDir()

	// Create some temp files
	tempFile := filepath.Join(tempDir, "temp_integration.pdf")
	if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Create manager and register cleanup
	manager := NewManager(logger, WithTimeout(5*time.Second))
	manager.Register("cleanup-downloads", 45, CleanupDownloads(logger, tempDir))

	// Execute shutdown
	err := manager.Shutdown()
	if err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}

	// Verify file was cleaned up
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Error("Temp file should have been cleaned up during shutdown")
	}
}

func TestCleanupDownloadsAndDir_IntegrationWithManager(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create a temp directory structure
	parentDir := t.TempDir()
	downloadsDir := filepath.Join(parentDir, "downloads")
	if err := os.Mkdir(downloadsDir, 0755); err != nil {
		t.Fatalf("Failed to create downloads directory: %v", err)
	}

	// Create some files
	tempFile := filepath.Join(downloadsDir, "temp_file.pdf")
	if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	// Create manager and register cleanup
	manager := NewManager(logger, WithTimeout(5*time.Second))
	manager.Register("cleanup-downloads-dir", 50, CleanupDownloadsAndDir(logger, downloadsDir))

	// Execute shutdown
	err := manager.Shutdown()
	if err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}

	// Verify directory was removed
	if _, err := os.Stat(downloadsDir); !os.IsNotExist(err) {
		t.Error("Downloads directory should have been removed during shutdown")
	}
}

func TestCleanupDownloads_ExecutesInPriorityOrder(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Create a temp directory
	tempDir := t.TempDir()

	// Create a temp file
	tempFile := filepath.Join(tempDir, "temp_order_test.txt")
	if err := os.WriteFile(tempFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	var executionOrder []string

	// Create manager
	manager := NewManager(logger, WithTimeout(5*time.Second))

	// Register cleanup with high priority (executes last)
	manager.Register("cleanup-downloads", 45, func(ctx context.Context) error {
		executionOrder = append(executionOrder, "cleanup-downloads")
		return CleanupDownloads(logger, tempDir)(ctx)
	})

	// Register another handler with lower priority (executes first)
	manager.Register("pre-cleanup", 10, func(ctx context.Context) error {
		executionOrder = append(executionOrder, "pre-cleanup")
		return nil
	})

	// Execute shutdown
	err := manager.Shutdown()
	if err != nil {
		t.Errorf("Shutdown returned error: %v", err)
	}

	// Verify execution order
	if len(executionOrder) != 2 {
		t.Fatalf("Expected 2 handlers executed, got %d", len(executionOrder))
	}
	if executionOrder[0] != "pre-cleanup" {
		t.Errorf("Expected pre-cleanup first, got %s", executionOrder[0])
	}
	if executionOrder[1] != "cleanup-downloads" {
		t.Errorf("Expected cleanup-downloads second, got %s", executionOrder[1])
	}

	// Verify cleanup happened
	if _, err := os.Stat(tempFile); !os.IsNotExist(err) {
		t.Error("Temp file should have been cleaned up")
	}
}
