// Package imagegen provides image generation utilities for the Canvus canvas.
//
// downloader.go implements the Downloader molecule that downloads generated
// images from temporary URLs returned by image generation providers.
//
// This molecule composes:
//   - core.Config: for HTTP/TLS configuration
//   - net/http: for HTTP downloads
package imagegen

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go_backend/core"
)

// Downloader handles downloading generated images from URLs.
//
// Image generation providers (OpenAI, Azure) return temporary URLs
// that expire after about 1 hour. This molecule downloads the image
// data and saves it locally before uploading to Canvus.
//
// Thread Safety: Downloader is safe for concurrent use.
// Each download creates its own HTTP request.
type Downloader struct {
	client       *http.Client
	downloadsDir string
}

// DownloaderConfig holds configuration for the Downloader.
type DownloaderConfig struct {
	// HTTPClient is the HTTP client for downloads (optional)
	// If nil, a default client will be created
	HTTPClient *http.Client

	// DownloadsDir is the directory for temporary image files
	// Default: "downloads"
	DownloadsDir string

	// Timeout for download operations
	// Default: 60 seconds
	Timeout time.Duration
}

// DefaultDownloaderConfig returns sensible defaults for downloading images.
func DefaultDownloaderConfig() DownloaderConfig {
	return DownloaderConfig{
		DownloadsDir: "downloads",
		Timeout:      60 * time.Second,
	}
}

// NewDownloader creates a new image downloader.
//
// Parameters:
//   - cfg: core.Config with HTTP/TLS configuration
//
// The downloader uses the HTTP client settings from core.Config,
// including TLS certificate validation settings.
//
// Example:
//
//	downloader, err := NewDownloader(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	path, err := downloader.Download(ctx, imageURL, "generated-123")
func NewDownloader(cfg *core.Config) (*Downloader, error) {
	if cfg == nil {
		return nil, fmt.Errorf("imagegen: config cannot be nil")
	}

	// Create HTTP client with TLS settings
	httpClient := core.GetDefaultHTTPClient(cfg)

	// Ensure downloads directory exists
	downloadsDir := cfg.DownloadsDir
	if downloadsDir == "" {
		downloadsDir = "downloads"
	}
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		return nil, fmt.Errorf("imagegen: failed to create downloads directory: %w", err)
	}

	return &Downloader{
		client:       httpClient,
		downloadsDir: downloadsDir,
	}, nil
}

// NewDownloaderWithConfig creates a downloader with explicit configuration.
// This is useful for testing or when you need fine-grained control.
//
// Parameters:
//   - cfg: DownloaderConfig with downloader-specific settings
//   - coreCfg: core.Config for HTTP client (optional, can be nil)
func NewDownloaderWithConfig(cfg DownloaderConfig, coreCfg *core.Config) (*Downloader, error) {
	// Use provided HTTP client or create one
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		if coreCfg != nil {
			httpClient = core.GetDefaultHTTPClient(coreCfg)
		} else {
			httpClient = &http.Client{
				Timeout: cfg.Timeout,
			}
		}
	}

	// Determine downloads directory
	downloadsDir := cfg.DownloadsDir
	if downloadsDir == "" {
		downloadsDir = "downloads"
	}

	// Ensure downloads directory exists
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		return nil, fmt.Errorf("imagegen: failed to create downloads directory: %w", err)
	}

	return &Downloader{
		client:       httpClient,
		downloadsDir: downloadsDir,
	}, nil
}

// DownloadResult contains information about the downloaded image.
type DownloadResult struct {
	// Path is the local file path of the downloaded image
	Path string

	// Size is the size of the downloaded image in bytes
	Size int64

	// ContentType is the MIME type of the image (if available)
	ContentType string
}

// Download downloads an image from the given URL and saves it locally.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - url: the URL of the image to download (typically a temporary URL from OpenAI/Azure)
//   - filename: base filename (without extension) for the downloaded image
//
// The method:
//  1. Creates an HTTP request with the provided context
//  2. Downloads the image data
//  3. Determines the file extension from Content-Type
//  4. Saves the image to the downloads directory
//  5. Returns the path and metadata
//
// Returns:
//   - *DownloadResult: information about the downloaded file
//   - error: if download fails
//
// The caller is responsible for cleaning up the downloaded file after use.
func (d *Downloader) Download(ctx context.Context, url string, filename string) (*DownloadResult, error) {
	if url == "" {
		return nil, fmt.Errorf("imagegen: URL cannot be empty")
	}
	if filename == "" {
		return nil, fmt.Errorf("imagegen: filename cannot be empty")
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("imagegen: failed to create download request: %w", err)
	}

	// Execute request
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("imagegen: failed to download image: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("imagegen: download failed with status %d", resp.StatusCode)
	}

	// Determine file extension from Content-Type
	ext := extensionFromContentType(resp.Header.Get("Content-Type"))
	if ext == "" {
		// Default to .png if we can't determine the type
		ext = ".png"
	}

	// Sanitize filename and create full path
	safeFilename := sanitizeFilename(filename)
	fullPath := filepath.Join(d.downloadsDir, safeFilename+ext)

	// Create the file
	file, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("imagegen: failed to create image file: %w", err)
	}
	defer file.Close()

	// Copy the data
	size, err := io.Copy(file, resp.Body)
	if err != nil {
		// Try to clean up the partial file
		os.Remove(fullPath)
		return nil, fmt.Errorf("imagegen: failed to write image data: %w", err)
	}

	return &DownloadResult{
		Path:        fullPath,
		Size:        size,
		ContentType: resp.Header.Get("Content-Type"),
	}, nil
}

// DownloadBytes downloads an image and returns the raw bytes.
// This is useful when you need the data in memory without saving to disk.
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - url: the URL of the image to download
//
// Returns:
//   - []byte: the image data
//   - string: the Content-Type header value
//   - error: if download fails
func (d *Downloader) DownloadBytes(ctx context.Context, url string) ([]byte, string, error) {
	if url == "" {
		return nil, "", fmt.Errorf("imagegen: URL cannot be empty")
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("imagegen: failed to create download request: %w", err)
	}

	// Execute request
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("imagegen: failed to download image: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("imagegen: download failed with status %d", resp.StatusCode)
	}

	// Read all data
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("imagegen: failed to read image data: %w", err)
	}

	return data, resp.Header.Get("Content-Type"), nil
}

// DownloadsDir returns the configured downloads directory.
func (d *Downloader) DownloadsDir() string {
	return d.downloadsDir
}

// extensionFromContentType returns the file extension for a given Content-Type.
func extensionFromContentType(contentType string) string {
	if contentType == "" {
		return ""
	}

	// Normalize to lowercase and strip parameters
	lower := strings.ToLower(contentType)
	if idx := strings.Index(lower, ";"); idx != -1 {
		lower = lower[:idx]
	}
	lower = strings.TrimSpace(lower)

	switch lower {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/bmp":
		return ".bmp"
	default:
		// Check for generic image/* types
		if strings.HasPrefix(lower, "image/") {
			return ".png" // Default to PNG for unknown image types
		}
		return ""
	}
}

// sanitizeFilename removes or replaces characters that are unsafe for filenames.
func sanitizeFilename(filename string) string {
	// Replace path separators and other unsafe characters
	unsafe := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|", "\n", "\r", "\t"}
	result := filename
	for _, char := range unsafe {
		result = strings.ReplaceAll(result, char, "_")
	}

	// Limit length
	if len(result) > 200 {
		result = result[:200]
	}

	// Ensure not empty
	if result == "" {
		result = "image"
	}

	return result
}
