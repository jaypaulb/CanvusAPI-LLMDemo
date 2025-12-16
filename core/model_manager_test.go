package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewModelManager(t *testing.T) {
	t.Run("creates with defaults", func(t *testing.T) {
		mm := NewModelManager("/tmp/models", nil)

		if mm.modelDir != "/tmp/models" {
			t.Errorf("modelDir = %q, want %q", mm.modelDir, "/tmp/models")
		}
		if mm.httpClient == nil {
			t.Error("httpClient is nil, want non-nil")
		}
		if mm.maxRetries != 3 {
			t.Errorf("maxRetries = %d, want 3", mm.maxRetries)
		}
		if mm.baseRetryDelay != 2*time.Second {
			t.Errorf("baseRetryDelay = %v, want 2s", mm.baseRetryDelay)
		}
		if len(mm.models) != 3 {
			t.Errorf("len(models) = %d, want 3 (default models)", len(mm.models))
		}
	})

	t.Run("applies options", func(t *testing.T) {
		customClient := &http.Client{Timeout: 5 * time.Second}
		customModel := ModelConfig{
			Name:     "test-model",
			URL:      "http://example.com/test.gguf",
			Filename: "test.gguf",
		}

		mm := NewModelManager("/tmp/models", customClient,
			WithMaxRetries(5),
			WithBaseRetryDelay(1*time.Second),
			WithDiskSpaceBuffer(20),
			WithModel(customModel),
		)

		if mm.httpClient != customClient {
			t.Error("httpClient not set to custom client")
		}
		if mm.maxRetries != 5 {
			t.Errorf("maxRetries = %d, want 5", mm.maxRetries)
		}
		if mm.baseRetryDelay != 1*time.Second {
			t.Errorf("baseRetryDelay = %v, want 1s", mm.baseRetryDelay)
		}
		if mm.diskSpaceBuffer != 20 {
			t.Errorf("diskSpaceBuffer = %d, want 20", mm.diskSpaceBuffer)
		}
		if _, ok := mm.models["test-model"]; !ok {
			t.Error("custom model not registered")
		}
	})

	t.Run("ignores invalid options", func(t *testing.T) {
		mm := NewModelManager("/tmp/models", nil,
			WithMaxRetries(0),       // Invalid: should be ignored
			WithMaxRetries(-1),      // Invalid: should be ignored
			WithBaseRetryDelay(0),   // Invalid: should be ignored
			WithDiskSpaceBuffer(-1), // Invalid: should be ignored
		)

		if mm.maxRetries != 3 {
			t.Errorf("maxRetries = %d, want 3 (unchanged)", mm.maxRetries)
		}
		if mm.baseRetryDelay != 2*time.Second {
			t.Errorf("baseRetryDelay = %v, want 2s (unchanged)", mm.baseRetryDelay)
		}
	})
}

func TestModelManager_checkModelExists(t *testing.T) {
	// Create temp directory for tests
	tmpDir, err := os.MkdirTemp("", "model_manager_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mm := NewModelManager(tmpDir, nil)

	t.Run("returns false for non-existent file", func(t *testing.T) {
		exists, err := mm.checkModelExists(filepath.Join(tmpDir, "nonexistent.gguf"), "")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if exists {
			t.Error("exists = true, want false")
		}
	})

	t.Run("returns true for existing file without checksum", func(t *testing.T) {
		// Create a test file
		testPath := filepath.Join(tmpDir, "test_model.gguf")
		if err := os.WriteFile(testPath, []byte("test model content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		exists, err := mm.checkModelExists(testPath, "")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !exists {
			t.Error("exists = false, want true")
		}
	})

	t.Run("returns false for empty file", func(t *testing.T) {
		testPath := filepath.Join(tmpDir, "empty_model.gguf")
		if err := os.WriteFile(testPath, []byte{}, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		exists, err := mm.checkModelExists(testPath, "")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if exists {
			t.Error("exists = true for empty file, want false")
		}
	})

	t.Run("returns error for directory", func(t *testing.T) {
		dirPath := filepath.Join(tmpDir, "model_dir")
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			t.Fatalf("Failed to create test dir: %v", err)
		}

		_, err := mm.checkModelExists(dirPath, "")
		if err == nil {
			t.Error("expected error for directory, got nil")
		}
		if !strings.Contains(err.Error(), "directory") {
			t.Errorf("error should mention directory: %v", err)
		}
	})

	t.Run("verifies checksum when provided", func(t *testing.T) {
		content := []byte("test model content for checksum")
		checksum := sha256.Sum256(content)
		checksumHex := hex.EncodeToString(checksum[:])

		testPath := filepath.Join(tmpDir, "checksum_model.gguf")
		if err := os.WriteFile(testPath, content, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Valid checksum
		exists, err := mm.checkModelExists(testPath, checksumHex)
		if err != nil {
			t.Errorf("unexpected error with valid checksum: %v", err)
		}
		if !exists {
			t.Error("exists = false with valid checksum, want true")
		}

		// Invalid checksum
		wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"
		_, err = mm.checkModelExists(testPath, wrongChecksum)
		if err == nil {
			t.Error("expected error with invalid checksum, got nil")
		}
		if !strings.Contains(err.Error(), "checksum mismatch") {
			t.Errorf("error should mention checksum mismatch: %v", err)
		}
	})
}

func TestModelManager_EnsureModelAvailable(t *testing.T) {
	t.Run("returns error for unknown model", func(t *testing.T) {
		tmpDir, _ := os.MkdirTemp("", "model_manager_test_*")
		defer os.RemoveAll(tmpDir)

		mm := NewModelManager(tmpDir, nil)

		err := mm.EnsureModelAvailable(context.Background(), "nonexistent-model")
		if err == nil {
			t.Error("expected error for unknown model, got nil")
		}
		if !strings.Contains(err.Error(), "unknown model") {
			t.Errorf("error should mention unknown model: %v", err)
		}
	})

	t.Run("returns nil if model already exists", func(t *testing.T) {
		tmpDir, _ := os.MkdirTemp("", "model_manager_test_*")
		defer os.RemoveAll(tmpDir)

		// Create model file
		modelContent := []byte("existing model content")
		modelPath := filepath.Join(tmpDir, "test-model.gguf")
		if err := os.WriteFile(modelPath, modelContent, 0644); err != nil {
			t.Fatalf("Failed to create model file: %v", err)
		}

		mm := NewModelManager(tmpDir, nil,
			WithModel(ModelConfig{
				Name:     "test-model",
				URL:      "http://example.com/model.gguf",
				Filename: "test-model.gguf",
			}),
		)

		err := mm.EnsureModelAvailable(context.Background(), "test-model")
		if err != nil {
			t.Errorf("unexpected error when model exists: %v", err)
		}
	})

	t.Run("downloads model if missing", func(t *testing.T) {
		// Create test content
		content := []byte("downloaded model content")
		checksum := sha256.Sum256(content)
		checksumHex := hex.EncodeToString(checksum[:])

		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.WriteHeader(http.StatusOK)
			w.Write(content)
		}))
		defer server.Close()

		tmpDir, _ := os.MkdirTemp("", "model_manager_test_*")
		defer os.RemoveAll(tmpDir)

		mm := NewModelManager(tmpDir, nil,
			WithModel(ModelConfig{
				Name:           "test-model",
				URL:            server.URL + "/model.gguf",
				Filename:       "test-model.gguf",
				ExpectedSHA256: checksumHex,
				SizeBytes:      int64(len(content)),
			}),
		)

		err := mm.EnsureModelAvailable(context.Background(), "test-model")
		if err != nil {
			t.Fatalf("EnsureModelAvailable failed: %v", err)
		}

		// Verify file was downloaded
		modelPath := filepath.Join(tmpDir, "test-model.gguf")
		downloaded, err := os.ReadFile(modelPath)
		if err != nil {
			t.Fatalf("Failed to read downloaded file: %v", err)
		}
		if string(downloaded) != string(content) {
			t.Error("downloaded content mismatch")
		}
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		// Create a server that delays response
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(10 * time.Second):
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("content"))
			}
		}))
		defer server.Close()

		tmpDir, _ := os.MkdirTemp("", "model_manager_test_*")
		defer os.RemoveAll(tmpDir)

		mm := NewModelManager(tmpDir, nil,
			WithModel(ModelConfig{
				Name:      "test-model",
				URL:       server.URL + "/model.gguf",
				Filename:  "test-model.gguf",
				SizeBytes: 100,
			}),
		)

		ctx, cancel := context.WithCancel(context.Background())

		// Start download in goroutine
		errCh := make(chan error, 1)
		go func() {
			errCh <- mm.EnsureModelAvailable(ctx, "test-model")
		}()

		// Cancel immediately
		cancel()

		err := <-errCh
		if err == nil {
			t.Error("expected error from cancelled context, got nil")
		}
	})
}

func TestModelManager_RetryLogic(t *testing.T) {
	t.Run("retries on transient failure", func(t *testing.T) {
		var requestCount int32
		content := []byte("model content")
		checksum := sha256.Sum256(content)
		checksumHex := hex.EncodeToString(checksum[:])

		// Server fails first 2 requests, succeeds on 3rd
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			count := atomic.AddInt32(&requestCount, 1)
			if count < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.WriteHeader(http.StatusOK)
			w.Write(content)
		}))
		defer server.Close()

		tmpDir, _ := os.MkdirTemp("", "model_manager_retry_test_*")
		defer os.RemoveAll(tmpDir)

		mm := NewModelManager(tmpDir, nil,
			WithMaxRetries(3),
			WithBaseRetryDelay(10*time.Millisecond), // Fast retries for test
			WithModel(ModelConfig{
				Name:           "test-model",
				URL:            server.URL + "/model.gguf",
				Filename:       "test-model.gguf",
				ExpectedSHA256: checksumHex,
				SizeBytes:      int64(len(content)),
			}),
		)

		err := mm.EnsureModelAvailable(context.Background(), "test-model")
		if err != nil {
			t.Fatalf("EnsureModelAvailable failed: %v", err)
		}

		// Should have made 3 requests
		finalCount := atomic.LoadInt32(&requestCount)
		if finalCount != 3 {
			t.Errorf("requestCount = %d, want 3", finalCount)
		}
	})

	t.Run("fails after max retries", func(t *testing.T) {
		var requestCount int32

		// Server always fails
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		tmpDir, _ := os.MkdirTemp("", "model_manager_retry_test_*")
		defer os.RemoveAll(tmpDir)

		mm := NewModelManager(tmpDir, nil,
			WithMaxRetries(3),
			WithBaseRetryDelay(10*time.Millisecond),
			WithModel(ModelConfig{
				Name:      "test-model",
				URL:       server.URL + "/model.gguf",
				Filename:  "test-model.gguf",
				SizeBytes: 100,
			}),
		)

		err := mm.EnsureModelAvailable(context.Background(), "test-model")
		if err == nil {
			t.Error("expected error after max retries, got nil")
		}

		// Should be a ModelDownloadError
		var downloadErr *ModelDownloadError
		if !isModelDownloadError(err) {
			t.Errorf("expected ModelDownloadError, got %T", err)
		} else {
			downloadErr = err.(*ModelDownloadError)
			if !strings.Contains(downloadErr.Message, "3 attempts") {
				t.Errorf("error should mention attempts: %v", downloadErr.Message)
			}
		}

		// Should have made 3 requests
		finalCount := atomic.LoadInt32(&requestCount)
		if finalCount != 3 {
			t.Errorf("requestCount = %d, want 3", finalCount)
		}
	})

	t.Run("does not retry checksum mismatch", func(t *testing.T) {
		var requestCount int32
		content := []byte("model content")
		wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&requestCount, 1)
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.WriteHeader(http.StatusOK)
			w.Write(content)
		}))
		defer server.Close()

		tmpDir, _ := os.MkdirTemp("", "model_manager_checksum_test_*")
		defer os.RemoveAll(tmpDir)

		mm := NewModelManager(tmpDir, nil,
			WithMaxRetries(3),
			WithBaseRetryDelay(10*time.Millisecond),
			WithModel(ModelConfig{
				Name:           "test-model",
				URL:            server.URL + "/model.gguf",
				Filename:       "test-model.gguf",
				ExpectedSHA256: wrongChecksum,
				SizeBytes:      int64(len(content)),
			}),
		)

		err := mm.EnsureModelAvailable(context.Background(), "test-model")
		if err == nil {
			t.Error("expected checksum mismatch error, got nil")
		}

		// Should only make 1 request (no retry for checksum mismatch)
		finalCount := atomic.LoadInt32(&requestCount)
		if finalCount != 1 {
			t.Errorf("requestCount = %d, want 1 (no retry for checksum)", finalCount)
		}
	})
}

func TestModelManager_GetModelPath(t *testing.T) {
	mm := NewModelManager("/opt/models", nil,
		WithModel(ModelConfig{
			Name:     "custom-model",
			Filename: "custom.gguf",
		}),
	)

	t.Run("returns path for known model", func(t *testing.T) {
		path, err := mm.GetModelPath("custom-model")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		expected := "/opt/models/custom.gguf"
		if path != expected {
			t.Errorf("path = %q, want %q", path, expected)
		}
	})

	t.Run("returns error for unknown model", func(t *testing.T) {
		_, err := mm.GetModelPath("unknown-model")
		if err == nil {
			t.Error("expected error for unknown model, got nil")
		}
	})
}

func TestModelDownloadError(t *testing.T) {
	t.Run("formats error with manual download instructions", func(t *testing.T) {
		err := &ModelDownloadError{
			ModelName: "test-model",
			Message:   "download failed after 3 attempts",
			URL:       "http://example.com/model.gguf",
			DestPath:  "/opt/models/model.gguf",
			Checksum:  "abc123",
		}

		errStr := err.Error()
		if !strings.Contains(errStr, "test-model") {
			t.Error("error should contain model name")
		}
		if !strings.Contains(errStr, "Manual download instructions") {
			t.Error("error should contain manual download instructions")
		}
		if !strings.Contains(errStr, "http://example.com/model.gguf") {
			t.Error("error should contain URL")
		}
		if !strings.Contains(errStr, "/opt/models/model.gguf") {
			t.Error("error should contain destination path")
		}
	})

	t.Run("formats simple error without URL", func(t *testing.T) {
		err := &ModelDownloadError{
			ModelName: "test-model",
			Message:   "insufficient disk space",
		}

		errStr := err.Error()
		if strings.Contains(errStr, "Manual download") {
			t.Error("error should not contain manual download instructions without URL")
		}
		if !strings.Contains(errStr, "test-model") {
			t.Error("error should contain model name")
		}
	})

	t.Run("unwraps cause", func(t *testing.T) {
		cause := fmt.Errorf("underlying error")
		err := &ModelDownloadError{
			ModelName: "test-model",
			Cause:     cause,
			Message:   "download failed",
		}

		unwrapped := err.Unwrap()
		if unwrapped != cause {
			t.Error("Unwrap should return cause")
		}
	})
}

func TestModelManager_isRetryableError(t *testing.T) {
	mm := NewModelManager("/tmp", nil)

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"context canceled", context.Canceled, false},
		{"context deadline", context.DeadlineExceeded, false},
		{"checksum mismatch", fmt.Errorf("checksum mismatch: file may be corrupted"), false},
		{"disk space error", &DiskSpaceError{Message: "insufficient space"}, false},
		{"network error", fmt.Errorf("connection refused"), true},
		{"http error", fmt.Errorf("HTTP 500"), true},
		{"generic error", fmt.Errorf("something went wrong"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mm.isRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("isRetryableError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestDefaultModels(t *testing.T) {
	t.Run("DefaultTextModel has required fields", func(t *testing.T) {
		if DefaultTextModel.Name == "" {
			t.Error("Name is empty")
		}
		if DefaultTextModel.URL == "" {
			t.Error("URL is empty")
		}
		if DefaultTextModel.Filename == "" {
			t.Error("Filename is empty")
		}
		if DefaultTextModel.SizeBytes <= 0 {
			t.Error("SizeBytes should be positive")
		}
	})

	t.Run("DefaultVisionModel has required fields", func(t *testing.T) {
		if DefaultVisionModel.Name == "" {
			t.Error("Name is empty")
		}
		if DefaultVisionModel.URL == "" {
			t.Error("URL is empty")
		}
		if DefaultVisionModel.Filename == "" {
			t.Error("Filename is empty")
		}
	})

	t.Run("DefaultSDModel has required fields", func(t *testing.T) {
		if DefaultSDModel.Name == "" {
			t.Error("Name is empty")
		}
		if DefaultSDModel.URL == "" {
			t.Error("URL is empty")
		}
		if DefaultSDModel.Filename == "" {
			t.Error("Filename is empty")
		}
	})
}

// Helper to check if error is ModelDownloadError
func isModelDownloadError(err error) bool {
	_, ok := err.(*ModelDownloadError)
	return ok
}
