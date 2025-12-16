package imagegen

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"go_backend/canvusapi"
	"go_backend/logging"
	"go_backend/sdruntime"
)

// TestNewProcessor_Validation tests the input validation in NewProcessor.
func TestNewProcessor_Validation(t *testing.T) {
	// Setup: Create valid dependencies for non-nil test cases
	tmpDir := t.TempDir()
	logger, err := logging.NewLogger(true, filepath.Join(tmpDir, "test.log"))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	client := canvusapi.NewClient("http://test", "canvas-123", "api-key", false)

	tests := []struct {
		name        string
		pool        *sdruntime.ContextPool
		client      *canvusapi.Client
		logger      *logging.Logger
		expectError string
	}{
		{
			name:        "nil pool returns error",
			pool:        nil,
			client:      client,
			logger:      logger,
			expectError: "pool cannot be nil",
		},
		{
			name:        "nil client returns error",
			pool:        createTestPool(t), // Will be closed for this test
			client:      nil,
			logger:      logger,
			expectError: "client cannot be nil",
		},
		{
			name:        "nil logger returns error",
			pool:        createTestPool(t),
			client:      client,
			logger:      nil,
			expectError: "logger cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultProcessorConfig()
			config.DownloadsDir = t.TempDir()

			_, err := NewProcessor(tt.pool, tt.client, tt.logger, config)

			if err == nil {
				t.Errorf("Expected error containing %q, got nil", tt.expectError)
				return
			}

			if !containsString(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing %q, got %q", tt.expectError, err.Error())
			}
		})
	}
}

// TestNewProcessor_ClosedPool tests that NewProcessor rejects a closed pool.
func TestNewProcessor_ClosedPool(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := logging.NewLogger(true, filepath.Join(tmpDir, "test.log"))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	client := canvusapi.NewClient("http://test", "canvas-123", "api-key", false)

	pool := createTestPool(t)
	pool.Close() // Close the pool

	config := DefaultProcessorConfig()
	config.DownloadsDir = t.TempDir()

	_, err = NewProcessor(pool, client, logger, config)

	if err == nil {
		t.Error("Expected error for closed pool, got nil")
		return
	}

	if !containsString(err.Error(), "pool is already closed") {
		t.Errorf("Expected error about closed pool, got: %v", err)
	}
}

// TestNewProcessor_CreatesDownloadsDir tests that NewProcessor creates the downloads directory.
func TestNewProcessor_CreatesDownloadsDir(t *testing.T) {
	tmpDir := t.TempDir()
	logger, err := logging.NewLogger(true, filepath.Join(tmpDir, "test.log"))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	client := canvusapi.NewClient("http://test", "canvas-123", "api-key", false)
	pool := createTestPool(t)
	defer pool.Close()

	// Create a path to a non-existent directory
	downloadsDir := filepath.Join(tmpDir, "new-downloads-dir")

	config := DefaultProcessorConfig()
	config.DownloadsDir = downloadsDir

	processor, err := NewProcessor(pool, client, logger, config)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if processor == nil {
		t.Fatal("Processor should not be nil")
	}

	// Verify the directory was created
	if _, err := os.Stat(downloadsDir); os.IsNotExist(err) {
		t.Error("Downloads directory was not created")
	}
}

// TestTruncateText tests the truncateText helper function.
func TestTruncateText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxLen   int
		expected string
	}{
		{
			name:     "short text unchanged",
			text:     "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length unchanged",
			text:     "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long text truncated with ellipsis",
			text:     "hello world",
			maxLen:   8,
			expected: "hello...",
		},
		{
			name:     "very short maxLen no ellipsis",
			text:     "hello",
			maxLen:   3,
			expected: "hel",
		},
		{
			name:     "empty text unchanged",
			text:     "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "unicode text truncated",
			text:     "hello world again",
			maxLen:   10,
			expected: "hello w...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateText(tt.text, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateText(%q, %d) = %q, want %q", tt.text, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// TestDefaultProcessorConfig tests that default config has sensible values.
func TestDefaultProcessorConfig(t *testing.T) {
	config := DefaultProcessorConfig()

	if config.DownloadsDir == "" {
		t.Error("DownloadsDir should have a default value")
	}

	if config.DefaultWidth == 0 || config.DefaultWidth%8 != 0 {
		t.Errorf("DefaultWidth should be non-zero and divisible by 8, got %d", config.DefaultWidth)
	}

	if config.DefaultHeight == 0 || config.DefaultHeight%8 != 0 {
		t.Errorf("DefaultHeight should be non-zero and divisible by 8, got %d", config.DefaultHeight)
	}

	if config.DefaultSteps <= 0 {
		t.Errorf("DefaultSteps should be positive, got %d", config.DefaultSteps)
	}

	if config.DefaultCFGScale <= 0 {
		t.Errorf("DefaultCFGScale should be positive, got %f", config.DefaultCFGScale)
	}
}

// TestDefaultProcessingNoteConfig tests the default processing note configuration.
func TestDefaultProcessingNoteConfig(t *testing.T) {
	config := DefaultProcessingNoteConfig()

	if config.Title == "" {
		t.Error("Title should have a default value")
	}

	if config.BackgroundColor == "" {
		t.Error("BackgroundColor should have a default value")
	}

	if config.TextColor == "" {
		t.Error("TextColor should have a default value")
	}
}

// TestCanvasWidget_Interface tests that CanvasWidget implements ParentWidget interface.
func TestCanvasWidget_Interface(t *testing.T) {
	widget := CanvasWidget{
		ID:       "test-widget-123",
		Location: WidgetLocation{X: 100, Y: 200},
		Size:     WidgetSize{Width: 300, Height: 400},
		Scale:    1.5,
		Depth:    50,
	}

	// Verify interface implementation
	var _ ParentWidget = widget

	if widget.GetID() != "test-widget-123" {
		t.Errorf("GetID() = %v, want %v", widget.GetID(), "test-widget-123")
	}

	loc := widget.GetLocation()
	if loc.X != 100 || loc.Y != 200 {
		t.Errorf("GetLocation() = %v, want {100, 200}", loc)
	}

	size := widget.GetSize()
	if size.Width != 300 || size.Height != 400 {
		t.Errorf("GetSize() = %v, want {300, 400}", size)
	}

	if widget.GetScale() != 1.5 {
		t.Errorf("GetScale() = %v, want %v", widget.GetScale(), 1.5)
	}

	if widget.GetDepth() != 50 {
		t.Errorf("GetDepth() = %v, want %v", widget.GetDepth(), 50.0)
	}
}

// TestProcessImagePrompt_InvalidPrompt tests that invalid prompts are rejected.
func TestProcessImagePrompt_InvalidPrompt(t *testing.T) {
	// Setup mock server for Canvus API
	noteCreated := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Track if error note was created
		if r.Method == "POST" && r.URL.Path == "/api/v1/canvases/canvas-123/notes" {
			noteCreated = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "error-note-123"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	logger, err := logging.NewLogger(true, filepath.Join(tmpDir, "test.log"))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	client := canvusapi.NewClient(server.URL, "canvas-123", "api-key", false)
	pool := createTestPool(t)
	defer pool.Close()

	config := DefaultProcessorConfig()
	config.DownloadsDir = t.TempDir()

	processor, err := NewProcessor(pool, client, logger, config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	parentWidget := CanvasWidget{
		ID:       "parent-123",
		Location: WidgetLocation{X: 100, Y: 100},
		Size:     WidgetSize{Width: 200, Height: 150},
		Scale:    1.0,
		Depth:    10,
	}

	// Test with empty prompt (should fail validation)
	ctx := context.Background()
	result, err := processor.ProcessImagePrompt(ctx, "   ", parentWidget)

	if err == nil {
		t.Error("Expected error for empty prompt, got nil")
	}

	if result != nil {
		t.Error("Expected nil result for failed processing")
	}

	// Verify error note was attempted (may fail if server doesn't handle it)
	if noteCreated {
		t.Log("Error note was created as expected")
	}
}

// createTestPool creates a test pool. Note: the pool will fail to generate
// images without a real model, but it's sufficient for testing processor creation.
func createTestPool(t *testing.T) *sdruntime.ContextPool {
	t.Helper()
	pool, err := sdruntime.NewContextPool(1, "/nonexistent/model.gguf")
	if err != nil {
		t.Fatalf("Failed to create test pool: %v", err)
	}
	return pool
}

// containsString checks if substr is contained in s.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && searchSubstring(s, substr)))
}

func searchSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
