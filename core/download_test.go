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
)

func TestDownloadWithProgress_BasicDownload(t *testing.T) {
	// Create test content
	content := []byte("Hello, World! This is test content for download.")
	checksum := sha256.Sum256(content)
	checksumHex := hex.EncodeToString(checksum[:])

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "download_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	destPath := filepath.Join(tmpDir, "test_download.txt")

	// Track progress calls
	var progressCalls int32
	onProgress := func(info ProgressInfo) {
		atomic.AddInt32(&progressCalls, 1)
	}

	// Execute download
	result, err := DownloadWithProgress(context.Background(), DownloadOptions{
		URL:            server.URL,
		DestPath:       destPath,
		ExpectedSHA256: checksumHex,
		OnProgress:     onProgress,
	})

	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	// Verify result
	if result.BytesDownloaded != int64(len(content)) {
		t.Errorf("BytesDownloaded = %d, want %d", result.BytesDownloaded, len(content))
	}
	if result.TotalBytes != int64(len(content)) {
		t.Errorf("TotalBytes = %d, want %d", result.TotalBytes, len(content))
	}
	if result.Resumed {
		t.Error("Resumed = true, want false")
	}
	if !result.ChecksumValid {
		t.Error("ChecksumValid = false, want true")
	}

	// Verify file content
	downloaded, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(downloaded) != string(content) {
		t.Errorf("Downloaded content mismatch")
	}
}

func TestDownloadWithProgress_Resume(t *testing.T) {
	// Create test content
	content := []byte("Hello, World! This is test content for resume download. More content here.")

	// Create test server that supports Range requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			// Parse range header
			start, _, _, parseErr := ParseContentRange("bytes " + strings.TrimPrefix(rangeHeader, "bytes=") + "/" + strconv.Itoa(len(content)))
			if parseErr != nil {
				// Simple parsing for "bytes=N-" format
				var s int64
				_, _ = fmt.Sscanf(rangeHeader, "bytes=%d-", &s)
				start = s
			}

			// Return partial content
			w.Header().Set("Content-Range", "bytes "+strconv.FormatInt(start, 10)+"-"+strconv.Itoa(len(content)-1)+"/"+strconv.Itoa(len(content)))
			w.Header().Set("Content-Length", strconv.FormatInt(int64(len(content))-start, 10))
			w.WriteHeader(http.StatusPartialContent)
			w.Write(content[start:])
		} else {
			w.Header().Set("Content-Length", strconv.Itoa(len(content)))
			w.WriteHeader(http.StatusOK)
			w.Write(content)
		}
	}))
	defer server.Close()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "download_resume_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	destPath := filepath.Join(tmpDir, "test_resume.txt")

	// Create partial file (simulate interrupted download)
	partialContent := content[:20]
	if err := os.WriteFile(destPath, partialContent, 0644); err != nil {
		t.Fatalf("Failed to create partial file: %v", err)
	}

	// Execute download with resume
	result, err := DownloadWithProgress(context.Background(), DownloadOptions{
		URL:      server.URL,
		DestPath: destPath,
		Resume:   true,
	})

	if err != nil {
		t.Fatalf("Resume download failed: %v", err)
	}

	// Verify result
	if !result.Resumed {
		t.Error("Resumed = false, want true")
	}
	expectedDownloaded := int64(len(content) - 20)
	if result.BytesDownloaded != expectedDownloaded {
		t.Errorf("BytesDownloaded = %d, want %d", result.BytesDownloaded, expectedDownloaded)
	}

	// Verify complete file
	downloaded, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(downloaded) != string(content) {
		t.Errorf("Downloaded content mismatch. Got %d bytes, want %d", len(downloaded), len(content))
	}
}

func TestDownloadWithProgress_ChecksumMismatch(t *testing.T) {
	content := []byte("Test content")
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "download_checksum_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	destPath := filepath.Join(tmpDir, "test_checksum.txt")

	_, err = DownloadWithProgress(context.Background(), DownloadOptions{
		URL:            server.URL,
		DestPath:       destPath,
		ExpectedSHA256: wrongChecksum,
	})

	if err == nil {
		t.Error("Expected checksum mismatch error, got nil")
	}
	if !strings.Contains(err.Error(), "checksum mismatch") {
		t.Errorf("Expected checksum mismatch error, got: %v", err)
	}
}

func TestDownloadWithProgress_ContextCancellation(t *testing.T) {
	// Create a server that sends data slowly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000000")
		w.WriteHeader(http.StatusOK)
		// Write some data but not all
		w.Write([]byte("start"))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// Wait for context cancellation
		<-r.Context().Done()
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "download_cancel_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	destPath := filepath.Join(tmpDir, "test_cancel.txt")

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Start download in goroutine
	errCh := make(chan error, 1)
	go func() {
		_, err := DownloadWithProgress(ctx, DownloadOptions{
			URL:      server.URL,
			DestPath: destPath,
		})
		errCh <- err
	}()

	// Cancel after a short delay
	cancel()

	err = <-errCh
	if err == nil {
		t.Error("Expected error from cancelled download, got nil")
	}
}

func TestDownloadWithProgress_InvalidOptions(t *testing.T) {
	tests := []struct {
		name    string
		opts    DownloadOptions
		wantErr string
	}{
		{
			name:    "empty URL",
			opts:    DownloadOptions{DestPath: "/tmp/test"},
			wantErr: "URL is required",
		},
		{
			name:    "empty DestPath",
			opts:    DownloadOptions{URL: "http://example.com/file"},
			wantErr: "DestPath is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DownloadWithProgress(context.Background(), tt.opts)
			if err == nil {
				t.Errorf("Expected error containing %q, got nil", tt.wantErr)
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Expected error containing %q, got %q", tt.wantErr, err.Error())
			}
		})
	}
}

func TestDownloadWithProgressSimple(t *testing.T) {
	content := []byte("Simple download test content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "download_simple_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	destPath := filepath.Join(tmpDir, "simple.txt")

	err = DownloadWithProgressSimple(context.Background(), server.URL, destPath, nil)
	if err != nil {
		t.Fatalf("Simple download failed: %v", err)
	}

	downloaded, _ := os.ReadFile(destPath)
	if string(downloaded) != string(content) {
		t.Errorf("Content mismatch")
	}
}

func TestDownloadWithResume(t *testing.T) {
	content := []byte("Resumable download test content")
	checksum := sha256.Sum256(content)
	checksumHex := hex.EncodeToString(checksum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "download_resume_func_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	destPath := filepath.Join(tmpDir, "resume.txt")

	result, err := DownloadWithResume(context.Background(), server.URL, destPath, checksumHex, nil)
	if err != nil {
		t.Fatalf("Resume download failed: %v", err)
	}

	if !result.ChecksumValid {
		t.Error("ChecksumValid = false, want true")
	}
}
