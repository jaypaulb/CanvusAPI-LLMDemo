package ocrprocessor

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go_backend/logging"
)

// Test constants for valid API keys
const testAPIKey = "AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY"

// mockVisionServer creates a test server that returns configurable responses
func mockVisionServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// successHandler returns a handler that responds with extracted text
func successHandler(text string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := visionResponse{
			Responses: []visionResponseItem{
				{
					FullTextAnnotation: struct {
						Text string `json:"text"`
					}{
						Text: text,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// TestDefaultProcessorConfig verifies default configuration values
func TestDefaultProcessorConfig(t *testing.T) {
	config := DefaultProcessorConfig()

	if config.MaxImageSize != 20*1024*1024 {
		t.Errorf("MaxImageSize = %d, want 20MB", config.MaxImageSize)
	}

	if config.DownloadTimeout != 30*time.Second {
		t.Errorf("DownloadTimeout = %v, want 30s", config.DownloadTimeout)
	}

	if len(config.SupportedFormats) == 0 {
		t.Error("SupportedFormats should not be empty")
	}

	// Verify common formats are supported
	formats := map[string]bool{
		"image/jpeg":      false,
		"image/png":       false,
		"application/pdf": false,
	}
	for _, f := range config.SupportedFormats {
		if _, ok := formats[f]; ok {
			formats[f] = true
		}
	}
	for format, found := range formats {
		if !found {
			t.Errorf("Expected format %s not in SupportedFormats", format)
		}
	}
}

// TestNewProcessor tests processor creation with various inputs
func TestNewProcessor(t *testing.T) {
	logger := testLogger(t)
	httpClient := &http.Client{}

	tests := []struct {
		name        string
		apiKey      string
		httpClient  *http.Client
		logger      *logging.Logger
		config      ProcessorConfig
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid inputs",
			apiKey:     testAPIKey,
			httpClient: httpClient,
			logger:     logger,
			config:     DefaultProcessorConfig(),
			wantErr:    false,
		},
		{
			name:        "nil HTTP client",
			apiKey:      testAPIKey,
			httpClient:  nil,
			logger:      logger,
			config:      DefaultProcessorConfig(),
			wantErr:     true,
			errContains: "nil",
		},
		{
			name:        "nil logger",
			apiKey:      testAPIKey,
			httpClient:  httpClient,
			logger:      nil,
			config:      DefaultProcessorConfig(),
			wantErr:     true,
			errContains: "nil",
		},
		{
			name:        "empty API key",
			apiKey:      "",
			httpClient:  httpClient,
			logger:      logger,
			config:      DefaultProcessorConfig(),
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "invalid API key (too short)",
			apiKey:      "short",
			httpClient:  httpClient,
			logger:      logger,
			config:      DefaultProcessorConfig(),
			wantErr:     true,
			errContains: "short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			processor, err := NewProcessor(tt.apiKey, tt.httpClient, tt.logger, tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("NewProcessor() expected error, got nil")
				} else if tt.errContains != "" && !containsStr(err.Error(), tt.errContains) {
					t.Errorf("NewProcessor() error = %q, want to contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewProcessor() unexpected error: %v", err)
				}
				if processor == nil {
					t.Error("NewProcessor() returned nil processor without error")
				}
			}
		})
	}
}

// TestNewProcessorWithProgress tests creation with progress callback
func TestNewProcessorWithProgress(t *testing.T) {
	logger := testLogger(t)
	httpClient := &http.Client{}

	var progressCalls []string
	progressCallback := func(stage string, progress float64, msg string) {
		progressCalls = append(progressCalls, stage)
	}

	processor, err := NewProcessorWithProgress(testAPIKey, httpClient, logger, DefaultProcessorConfig(), progressCallback)
	if err != nil {
		t.Fatalf("NewProcessorWithProgress() unexpected error: %v", err)
	}
	if processor == nil {
		t.Fatal("NewProcessorWithProgress() returned nil processor")
	}

	// Verify progress callback is set by triggering it
	processor.reportProgress("test", 0.5, "test message")
	if len(progressCalls) != 1 || progressCalls[0] != "test" {
		t.Errorf("Progress callback not set correctly, calls = %v", progressCalls)
	}
}

// TestNewProcessorWithProgress_Error tests error propagation
func TestNewProcessorWithProgress_Error(t *testing.T) {
	logger := testLogger(t)

	_, err := NewProcessorWithProgress("", &http.Client{}, logger, DefaultProcessorConfig(), nil)
	if err == nil {
		t.Error("NewProcessorWithProgress() with empty API key should return error")
	}
}

// TestProcessor_SetProgressCallback tests setting progress callback after creation
func TestProcessor_SetProgressCallback(t *testing.T) {
	logger := testLogger(t)
	httpClient := &http.Client{}

	processor, err := NewProcessor(testAPIKey, httpClient, logger, DefaultProcessorConfig())
	if err != nil {
		t.Fatalf("NewProcessor() unexpected error: %v", err)
	}

	var called bool
	processor.SetProgressCallback(func(stage string, progress float64, msg string) {
		called = true
	})

	processor.reportProgress("test", 0.5, "test")
	if !called {
		t.Error("Progress callback was not called after SetProgressCallback")
	}
}

// TestProcessor_ProcessImage tests image processing
func TestProcessor_ProcessImage(t *testing.T) {
	server := mockVisionServer(t, successHandler("Extracted text from image"))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()
	config.VisionClientConfig.Endpoint = server.URL

	processor, err := NewProcessor(testAPIKey, server.Client(), logger, config)
	if err != nil {
		t.Fatalf("NewProcessor() error: %v", err)
	}

	imageData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header

	result, err := processor.ProcessImage(context.Background(), imageData)
	if err != nil {
		t.Fatalf("ProcessImage() error: %v", err)
	}

	if result.Text != "Extracted text from image" {
		t.Errorf("ProcessImage().Text = %q, want %q", result.Text, "Extracted text from image")
	}
	if result.ProcessingTime <= 0 {
		t.Error("ProcessImage().ProcessingTime should be positive")
	}
	if result.ImageSize != int64(len(imageData)) {
		t.Errorf("ProcessImage().ImageSize = %d, want %d", result.ImageSize, len(imageData))
	}
}

// TestProcessor_ProcessImage_NotConfigured tests nil client error
func TestProcessor_ProcessImage_NotConfigured(t *testing.T) {
	processor := &Processor{} // Empty processor

	_, err := processor.ProcessImage(context.Background(), []byte{0x89, 0x50, 0x4E, 0x47})
	if !errors.Is(err, ErrProcessorNotConfigured) {
		t.Errorf("ProcessImage() error = %v, want ErrProcessorNotConfigured", err)
	}
}

// TestProcessor_ProcessImage_TooLarge tests image size limit enforcement
func TestProcessor_ProcessImage_TooLarge(t *testing.T) {
	server := mockVisionServer(t, successHandler("text"))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()
	config.VisionClientConfig.Endpoint = server.URL
	config.MaxImageSize = 10 // Very small limit

	processor, err := NewProcessor(testAPIKey, server.Client(), logger, config)
	if err != nil {
		t.Fatalf("NewProcessor() error: %v", err)
	}

	largeData := make([]byte, 100) // Exceeds limit

	_, err = processor.ProcessImage(context.Background(), largeData)
	if !errors.Is(err, ErrImageTooLarge) {
		t.Errorf("ProcessImage() error = %v, want ErrImageTooLarge", err)
	}
}

// TestProcessor_ProcessImage_NoSizeLimit tests processing without size limit
func TestProcessor_ProcessImage_NoSizeLimit(t *testing.T) {
	server := mockVisionServer(t, successHandler("text"))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()
	config.VisionClientConfig.Endpoint = server.URL
	config.MaxImageSize = 0 // No limit

	processor, err := NewProcessor(testAPIKey, server.Client(), logger, config)
	if err != nil {
		t.Fatalf("NewProcessor() error: %v", err)
	}

	largeData := make([]byte, 1000)
	largeData[0] = 0x89 // Make it look like image data

	result, err := processor.ProcessImage(context.Background(), largeData)
	if err != nil {
		t.Fatalf("ProcessImage() with no size limit should succeed: %v", err)
	}
	if result.Text != "text" {
		t.Errorf("ProcessImage().Text = %q, want %q", result.Text, "text")
	}
}

// TestProcessor_ProcessImage_ProgressReporting tests progress callback invocation
func TestProcessor_ProcessImage_ProgressReporting(t *testing.T) {
	server := mockVisionServer(t, successHandler("text"))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()
	config.VisionClientConfig.Endpoint = server.URL

	var stages []string
	var progresses []float64

	processor, err := NewProcessorWithProgress(testAPIKey, server.Client(), logger, config,
		func(stage string, progress float64, msg string) {
			stages = append(stages, stage)
			progresses = append(progresses, progress)
		})
	if err != nil {
		t.Fatalf("NewProcessorWithProgress() error: %v", err)
	}

	_, err = processor.ProcessImage(context.Background(), []byte{0x89, 0x50, 0x4E, 0x47})
	if err != nil {
		t.Fatalf("ProcessImage() error: %v", err)
	}

	// Should have reported progress at multiple stages
	if len(stages) < 2 {
		t.Errorf("Expected at least 2 progress reports, got %d", len(stages))
	}

	// Last progress should be 1.0 (complete)
	if len(progresses) > 0 && progresses[len(progresses)-1] != 1.0 {
		t.Errorf("Final progress = %v, want 1.0", progresses[len(progresses)-1])
	}
}

// TestProcessor_ProcessFile tests file processing
func TestProcessor_ProcessFile(t *testing.T) {
	server := mockVisionServer(t, successHandler("File text"))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()
	config.VisionClientConfig.Endpoint = server.URL

	processor, err := NewProcessor(testAPIKey, server.Client(), logger, config)
	if err != nil {
		t.Fatalf("NewProcessor() error: %v", err)
	}

	// Create temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_image.png")
	if err := os.WriteFile(testFile, []byte{0x89, 0x50, 0x4E, 0x47}, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err := processor.ProcessFile(context.Background(), testFile)
	if err != nil {
		t.Fatalf("ProcessFile() error: %v", err)
	}

	if result.Text != "File text" {
		t.Errorf("ProcessFile().Text = %q, want %q", result.Text, "File text")
	}
}

// TestProcessor_ProcessFile_NotFound tests missing file error
func TestProcessor_ProcessFile_NotFound(t *testing.T) {
	server := mockVisionServer(t, successHandler("text"))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()
	config.VisionClientConfig.Endpoint = server.URL

	processor, err := NewProcessor(testAPIKey, server.Client(), logger, config)
	if err != nil {
		t.Fatalf("NewProcessor() error: %v", err)
	}

	_, err = processor.ProcessFile(context.Background(), "/nonexistent/file.png")
	if !errors.Is(err, ErrImageLoadFailed) {
		t.Errorf("ProcessFile() error = %v, want ErrImageLoadFailed", err)
	}
}

// TestProcessor_ProcessFile_NotConfigured tests nil client error
func TestProcessor_ProcessFile_NotConfigured(t *testing.T) {
	processor := &Processor{} // Empty processor

	_, err := processor.ProcessFile(context.Background(), "/some/file.png")
	if !errors.Is(err, ErrProcessorNotConfigured) {
		t.Errorf("ProcessFile() error = %v, want ErrProcessorNotConfigured", err)
	}
}

// TestProcessor_ProcessURL tests URL download and processing
func TestProcessor_ProcessURL(t *testing.T) {
	// Create image server
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte{0x89, 0x50, 0x4E, 0x47})
	}))
	defer imageServer.Close()

	// Create Vision API server
	visionServer := mockVisionServer(t, successHandler("URL text"))
	defer visionServer.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()
	config.VisionClientConfig.Endpoint = visionServer.URL

	processor, err := NewProcessor(testAPIKey, imageServer.Client(), logger, config)
	if err != nil {
		t.Fatalf("NewProcessor() error: %v", err)
	}

	result, err := processor.ProcessURL(context.Background(), imageServer.URL)
	if err != nil {
		t.Fatalf("ProcessURL() error: %v", err)
	}

	if result.Text != "URL text" {
		t.Errorf("ProcessURL().Text = %q, want %q", result.Text, "URL text")
	}
}

// TestProcessor_ProcessURL_HTTPError tests HTTP error handling
func TestProcessor_ProcessURL_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()

	processor, err := NewProcessor(testAPIKey, server.Client(), logger, config)
	if err != nil {
		t.Fatalf("NewProcessor() error: %v", err)
	}

	_, err = processor.ProcessURL(context.Background(), server.URL)
	if !errors.Is(err, ErrImageLoadFailed) {
		t.Errorf("ProcessURL() error = %v, want ErrImageLoadFailed", err)
	}
	if !containsStr(err.Error(), "404") {
		t.Errorf("ProcessURL() error should mention status code: %v", err)
	}
}

// TestProcessor_ProcessURL_ContentLengthExceeded tests size limit via Content-Length
func TestProcessor_ProcessURL_ContentLengthExceeded(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100000000") // Large Content-Length
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte{0x89, 0x50, 0x4E, 0x47})
	}))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()
	config.MaxImageSize = 1000 // Small limit

	processor, err := NewProcessor(testAPIKey, server.Client(), logger, config)
	if err != nil {
		t.Fatalf("NewProcessor() error: %v", err)
	}

	_, err = processor.ProcessURL(context.Background(), server.URL)
	if !errors.Is(err, ErrImageTooLarge) {
		t.Errorf("ProcessURL() error = %v, want ErrImageTooLarge", err)
	}
}

// TestProcessor_ProcessURL_BodyTooLarge tests size limit enforcement during read
func TestProcessor_ProcessURL_BodyTooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't set Content-Length header, send large body
		w.Header().Set("Content-Type", "image/png")
		largeData := make([]byte, 2000)
		w.Write(largeData)
	}))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()
	config.MaxImageSize = 100 // Small limit

	processor, err := NewProcessor(testAPIKey, server.Client(), logger, config)
	if err != nil {
		t.Fatalf("NewProcessor() error: %v", err)
	}

	_, err = processor.ProcessURL(context.Background(), server.URL)
	if !errors.Is(err, ErrImageTooLarge) {
		t.Errorf("ProcessURL() error = %v, want ErrImageTooLarge", err)
	}
}

// TestProcessor_ProcessURL_NotConfigured tests nil client error
func TestProcessor_ProcessURL_NotConfigured(t *testing.T) {
	processor := &Processor{} // Empty processor

	_, err := processor.ProcessURL(context.Background(), "http://example.com/image.png")
	if !errors.Is(err, ErrProcessorNotConfigured) {
		t.Errorf("ProcessURL() error = %v, want ErrProcessorNotConfigured", err)
	}
}

// TestProcessor_ProcessURL_InvalidURL tests invalid URL handling
func TestProcessor_ProcessURL_InvalidURL(t *testing.T) {
	logger := testLogger(t)
	config := DefaultProcessorConfig()

	processor, err := NewProcessor(testAPIKey, &http.Client{}, logger, config)
	if err != nil {
		t.Fatalf("NewProcessor() error: %v", err)
	}

	_, err = processor.ProcessURL(context.Background(), "://invalid-url")
	if !errors.Is(err, ErrImageLoadFailed) {
		t.Errorf("ProcessURL() error = %v, want ErrImageLoadFailed", err)
	}
}

// TestProcessor_ProcessURL_ContextCancelled tests context cancellation
func TestProcessor_ProcessURL_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte{0x89, 0x50, 0x4E, 0x47})
	}))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()

	processor, err := NewProcessor(testAPIKey, server.Client(), logger, config)
	if err != nil {
		t.Fatalf("NewProcessor() error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = processor.ProcessURL(ctx, server.URL)
	if err == nil {
		t.Error("ProcessURL() with cancelled context should return error")
	}
}

// TestProcessor_ProcessURL_NoSizeLimit tests URL processing without size limit
func TestProcessor_ProcessURL_NoSizeLimit(t *testing.T) {
	// Create image server
	imageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x00, 0x00, 0x00})
	}))
	defer imageServer.Close()

	// Create Vision API server
	visionServer := mockVisionServer(t, successHandler("text"))
	defer visionServer.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()
	config.VisionClientConfig.Endpoint = visionServer.URL
	config.MaxImageSize = 0 // No limit

	processor, err := NewProcessor(testAPIKey, imageServer.Client(), logger, config)
	if err != nil {
		t.Fatalf("NewProcessor() error: %v", err)
	}

	result, err := processor.ProcessURL(context.Background(), imageServer.URL)
	if err != nil {
		t.Fatalf("ProcessURL() with no size limit should succeed: %v", err)
	}
	if result.Text != "text" {
		t.Errorf("ProcessURL().Text = %q, want %q", result.Text, "text")
	}
}

// TestProcessor_ValidateAPIKey tests API key validation
func TestProcessor_ValidateAPIKey(t *testing.T) {
	server := mockVisionServer(t, func(w http.ResponseWriter, r *http.Request) {
		response := visionResponse{
			Responses: []visionResponseItem{{}},
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()
	config.VisionClientConfig.Endpoint = server.URL

	processor, err := NewProcessor(testAPIKey, server.Client(), logger, config)
	if err != nil {
		t.Fatalf("NewProcessor() error: %v", err)
	}

	err = processor.ValidateAPIKey(context.Background())
	if err != nil {
		t.Errorf("ValidateAPIKey() unexpected error: %v", err)
	}
}

// TestProcessor_ValidateAPIKey_NotConfigured tests validation with nil client
func TestProcessor_ValidateAPIKey_NotConfigured(t *testing.T) {
	processor := &Processor{} // Empty processor

	err := processor.ValidateAPIKey(context.Background())
	if !errors.Is(err, ErrProcessorNotConfigured) {
		t.Errorf("ValidateAPIKey() error = %v, want ErrProcessorNotConfigured", err)
	}
}

// TestProcessor_GetMaskedAPIKey tests API key masking
func TestProcessor_GetMaskedAPIKey(t *testing.T) {
	server := mockVisionServer(t, successHandler("text"))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()
	config.VisionClientConfig.Endpoint = server.URL

	processor, err := NewProcessor(testAPIKey, server.Client(), logger, config)
	if err != nil {
		t.Fatalf("NewProcessor() error: %v", err)
	}

	masked := processor.GetMaskedAPIKey()
	if masked == testAPIKey {
		t.Error("GetMaskedAPIKey() should not return full key")
	}
	if !strings.Contains(masked, "****") {
		t.Errorf("GetMaskedAPIKey() = %q, want masked characters", masked)
	}
}

// TestProcessor_GetMaskedAPIKey_NotConfigured tests masking with nil client
func TestProcessor_GetMaskedAPIKey_NotConfigured(t *testing.T) {
	processor := &Processor{} // Empty processor

	masked := processor.GetMaskedAPIKey()
	if masked != "[not configured]" {
		t.Errorf("GetMaskedAPIKey() = %q, want '[not configured]'", masked)
	}
}

// TestProcessImageWithConfig tests the convenience function
func TestProcessImageWithConfig(t *testing.T) {
	server := mockVisionServer(t, successHandler("Config text"))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()
	config.VisionClientConfig.Endpoint = server.URL

	result, err := ProcessImageWithConfig(context.Background(), testAPIKey, server.Client(), logger, config, []byte{0x89, 0x50, 0x4E, 0x47})
	if err != nil {
		t.Fatalf("ProcessImageWithConfig() error: %v", err)
	}

	if result.Text != "Config text" {
		t.Errorf("ProcessImageWithConfig().Text = %q, want %q", result.Text, "Config text")
	}
}

// TestProcessImageWithConfig_InvalidKey tests error propagation
func TestProcessImageWithConfig_InvalidKey(t *testing.T) {
	logger := testLogger(t)

	_, err := ProcessImageWithConfig(context.Background(), "", &http.Client{}, logger, DefaultProcessorConfig(), []byte{0x89})
	if err == nil {
		t.Error("ProcessImageWithConfig() with empty key should return error")
	}
}

// TestExtractText tests the simple convenience function
func TestExtractText(t *testing.T) {
	server := mockVisionServer(t, successHandler("Simple text"))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultProcessorConfig()
	config.VisionClientConfig.Endpoint = server.URL

	// We need to override the default config - create processor directly
	processor, err := NewProcessor(testAPIKey, server.Client(), logger, config)
	if err != nil {
		t.Fatalf("NewProcessor() error: %v", err)
	}

	result, err := processor.ProcessImage(context.Background(), []byte{0x89, 0x50, 0x4E, 0x47})
	if err != nil {
		t.Fatalf("ProcessImage() error: %v", err)
	}

	if result.Text != "Simple text" {
		t.Errorf("ExtractText() = %q, want %q", result.Text, "Simple text")
	}
}

// TestExtractText_InvalidKey tests error propagation
func TestExtractText_InvalidKey(t *testing.T) {
	logger := testLogger(t)

	_, err := ExtractText(context.Background(), "", &http.Client{}, logger, []byte{0x89})
	if err == nil {
		t.Error("ExtractText() with empty key should return error")
	}
}

// TestProcessResult_Fields verifies ProcessResult structure
func TestProcessResult_Fields(t *testing.T) {
	result := ProcessResult{
		Text:           "test",
		ProcessingTime: 100 * time.Millisecond,
		VisionAPITime:  50 * time.Millisecond,
		ImageSize:      1024,
	}

	if result.Text != "test" {
		t.Errorf("ProcessResult.Text = %q, want %q", result.Text, "test")
	}
	if result.ProcessingTime != 100*time.Millisecond {
		t.Errorf("ProcessResult.ProcessingTime = %v, want 100ms", result.ProcessingTime)
	}
	if result.VisionAPITime != 50*time.Millisecond {
		t.Errorf("ProcessResult.VisionAPITime = %v, want 50ms", result.VisionAPITime)
	}
	if result.ImageSize != 1024 {
		t.Errorf("ProcessResult.ImageSize = %d, want 1024", result.ImageSize)
	}
}

// TestProcessor_reportProgress_NilCallback tests nil callback safety
func TestProcessor_reportProgress_NilCallback(t *testing.T) {
	processor := &Processor{
		progress: nil,
	}

	// Should not panic
	processor.reportProgress("test", 0.5, "message")
}

// TestSentinelErrors verifies sentinel errors are defined correctly
func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrImageTooLarge", ErrImageTooLarge},
		{"ErrUnsupportedFormat", ErrUnsupportedFormat},
		{"ErrProcessorNotConfigured", ErrProcessorNotConfigured},
		{"ErrImageLoadFailed", ErrImageLoadFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}
			if tt.err.Error() == "" {
				t.Errorf("%s.Error() should not be empty", tt.name)
			}
			if !strings.HasPrefix(tt.err.Error(), "ocrprocessor:") {
				t.Errorf("%s.Error() = %q, want prefix 'ocrprocessor:'", tt.name, tt.err.Error())
			}
		})
	}
}
