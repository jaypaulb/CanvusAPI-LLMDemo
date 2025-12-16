package imagegen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go_backend/core"
)

// TestNewDownloader_NilConfig tests that NewDownloader returns error for nil config.
func TestNewDownloader_NilConfig(t *testing.T) {
	downloader, err := NewDownloader(nil)

	if err == nil {
		t.Error("expected error for nil config, got nil")
	}
	if downloader != nil {
		t.Error("expected nil downloader for nil config")
	}
}

// TestNewDownloader_ValidConfig tests successful downloader creation.
func TestNewDownloader_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &core.Config{
		DownloadsDir: tmpDir,
	}

	downloader, err := NewDownloader(cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if downloader == nil {
		t.Fatal("expected non-nil downloader")
	}
	if downloader.DownloadsDir() != tmpDir {
		t.Errorf("expected downloads dir %s, got %s", tmpDir, downloader.DownloadsDir())
	}
}

// TestNewDownloader_DefaultDownloadsDir tests default downloads directory.
func TestNewDownloader_DefaultDownloadsDir(t *testing.T) {
	cfg := &core.Config{
		DownloadsDir: "", // Empty should default to "downloads"
	}

	downloader, err := NewDownloader(cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if downloader.DownloadsDir() != "downloads" {
		t.Errorf("expected default downloads dir 'downloads', got %s", downloader.DownloadsDir())
	}

	// Clean up
	os.RemoveAll("downloads")
}

// TestNewDownloaderWithConfig_ValidConfig tests explicit config creation.
func TestNewDownloaderWithConfig_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := DownloaderConfig{
		DownloadsDir: tmpDir,
		Timeout:      30 * time.Second,
	}

	downloader, err := NewDownloaderWithConfig(cfg, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if downloader == nil {
		t.Fatal("expected non-nil downloader")
	}
	if downloader.DownloadsDir() != tmpDir {
		t.Errorf("expected downloads dir %s, got %s", tmpDir, downloader.DownloadsDir())
	}
}

// TestDefaultDownloaderConfig tests default configuration values.
func TestDefaultDownloaderConfig(t *testing.T) {
	cfg := DefaultDownloaderConfig()

	if cfg.DownloadsDir != "downloads" {
		t.Errorf("expected default DownloadsDir 'downloads', got %s", cfg.DownloadsDir)
	}
	if cfg.Timeout != 60*time.Second {
		t.Errorf("expected default Timeout 60s, got %v", cfg.Timeout)
	}
}

// TestDownload_EmptyURL tests that empty URL returns error.
func TestDownload_EmptyURL(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &core.Config{
		DownloadsDir: tmpDir,
	}

	downloader, err := NewDownloader(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating downloader: %v", err)
	}

	result, err := downloader.Download(context.Background(), "", "test")

	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
	if result != nil {
		t.Error("expected nil result for empty URL")
	}
}

// TestDownload_EmptyFilename tests that empty filename returns error.
func TestDownload_EmptyFilename(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &core.Config{
		DownloadsDir: tmpDir,
	}

	downloader, err := NewDownloader(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating downloader: %v", err)
	}

	result, err := downloader.Download(context.Background(), "http://example.com/image.png", "")

	if err == nil {
		t.Error("expected error for empty filename, got nil")
	}
	if result != nil {
		t.Error("expected nil result for empty filename")
	}
}

// TestDownload_Success tests successful image download.
func TestDownload_Success(t *testing.T) {
	// Create test server
	imageData := []byte("fake image data for testing")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(imageData)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	cfg := &core.Config{
		DownloadsDir: tmpDir,
	}

	downloader, err := NewDownloader(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating downloader: %v", err)
	}

	result, err := downloader.Download(context.Background(), server.URL, "test-image")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Size != int64(len(imageData)) {
		t.Errorf("expected size %d, got %d", len(imageData), result.Size)
	}
	if result.ContentType != "image/png" {
		t.Errorf("expected content type image/png, got %s", result.ContentType)
	}
	if filepath.Ext(result.Path) != ".png" {
		t.Errorf("expected .png extension, got %s", filepath.Ext(result.Path))
	}

	// Verify file exists and has correct content
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(data) != string(imageData) {
		t.Error("downloaded file content doesn't match")
	}
}

// TestDownload_JPEGContentType tests JPEG content type handling.
func TestDownload_JPEGContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("jpeg data"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	downloader, err := NewDownloaderWithConfig(DownloaderConfig{
		DownloadsDir: tmpDir,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result, err := downloader.Download(context.Background(), server.URL, "test")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filepath.Ext(result.Path) != ".jpg" {
		t.Errorf("expected .jpg extension, got %s", filepath.Ext(result.Path))
	}
}

// TestDownload_NonOKStatus tests error handling for non-200 status codes.
func TestDownload_NonOKStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	cfg := &core.Config{
		DownloadsDir: tmpDir,
	}

	downloader, err := NewDownloader(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating downloader: %v", err)
	}

	result, err := downloader.Download(context.Background(), server.URL, "test")

	if err == nil {
		t.Error("expected error for 404 status, got nil")
	}
	if result != nil {
		t.Error("expected nil result for 404 status")
	}
}

// TestDownloadBytes_EmptyURL tests that empty URL returns error for DownloadBytes.
func TestDownloadBytes_EmptyURL(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &core.Config{
		DownloadsDir: tmpDir,
	}

	downloader, err := NewDownloader(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating downloader: %v", err)
	}

	data, contentType, err := downloader.DownloadBytes(context.Background(), "")

	if err == nil {
		t.Error("expected error for empty URL, got nil")
	}
	if data != nil {
		t.Error("expected nil data for empty URL")
	}
	if contentType != "" {
		t.Errorf("expected empty content type, got %s", contentType)
	}
}

// TestDownloadBytes_Success tests successful byte download.
func TestDownloadBytes_Success(t *testing.T) {
	imageData := []byte("image bytes for testing")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/webp")
		w.WriteHeader(http.StatusOK)
		w.Write(imageData)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	cfg := &core.Config{
		DownloadsDir: tmpDir,
	}

	downloader, err := NewDownloader(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating downloader: %v", err)
	}

	data, contentType, err := downloader.DownloadBytes(context.Background(), server.URL)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != string(imageData) {
		t.Error("downloaded bytes don't match")
	}
	if contentType != "image/webp" {
		t.Errorf("expected content type image/webp, got %s", contentType)
	}
}

// TestExtensionFromContentType tests content type to extension mapping.
func TestExtensionFromContentType(t *testing.T) {
	tests := []struct {
		contentType string
		expected    string
	}{
		{"image/png", ".png"},
		{"image/jpeg", ".jpg"},
		{"image/jpg", ".jpg"},
		{"image/gif", ".gif"},
		{"image/webp", ".webp"},
		{"image/bmp", ".bmp"},
		{"IMAGE/PNG", ".png"},                     // case insensitive
		{"image/png; charset=utf-8", ".png"},      // with parameters
		{"image/unknown", ".png"},                 // unknown image type defaults to png
		{"text/plain", ""},                        // non-image type
		{"", ""},                                  // empty
		{"application/octet-stream", ""},          // binary
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := extensionFromContentType(tt.contentType)
			if result != tt.expected {
				t.Errorf("extensionFromContentType(%q) = %q, expected %q", tt.contentType, result, tt.expected)
			}
		})
	}
}

// TestSanitizeFilename tests filename sanitization.
func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with space", "with space"},
		{"path/separator", "path_separator"},
		{"back\\slash", "back_slash"},
		{"colon:char", "colon_char"},
		{"star*char", "star_char"},
		{"question?mark", "question_mark"},
		{"quote\"char", "quote_char"},
		{"less<than", "less_than"},
		{"greater>than", "greater_than"},
		{"pipe|char", "pipe_char"},
		{"new\nline", "new_line"},
		{"carriage\rreturn", "carriage_return"},
		{"tab\tchar", "tab_char"},
		{"", "image"}, // empty defaults to "image"
		{"/multiple//unsafe\\chars", "_multiple__unsafe_chars"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSanitizeFilename_LongFilename tests truncation of long filenames.
func TestSanitizeFilename_LongFilename(t *testing.T) {
	// Create a filename longer than 200 characters
	longFilename := ""
	for i := 0; i < 250; i++ {
		longFilename += "a"
	}

	result := sanitizeFilename(longFilename)

	if len(result) > 200 {
		t.Errorf("expected filename to be truncated to 200 chars, got %d", len(result))
	}
	if len(result) != 200 {
		t.Errorf("expected exactly 200 chars, got %d", len(result))
	}
}

// TestDownload_ContextCancellation tests that context cancellation is respected.
func TestDownload_ContextCancellation(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	cfg := &core.Config{
		DownloadsDir: tmpDir,
	}

	downloader, err := NewDownloader(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating downloader: %v", err)
	}

	// Create a context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := downloader.Download(ctx, server.URL, "test")

	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
	if result != nil {
		t.Error("expected nil result for cancelled context")
	}
}
