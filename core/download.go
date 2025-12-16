package core

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// DownloadOptions configures the download behavior.
type DownloadOptions struct {
	// URL to download from
	URL string
	// DestPath is the local file path to save to
	DestPath string
	// ExpectedSHA256 is the optional expected SHA256 checksum (lowercase hex, 64 chars)
	// If provided, the downloaded file will be verified against this checksum
	ExpectedSHA256 string
	// HTTPClient is the HTTP client to use (creates default if nil)
	HTTPClient *http.Client
	// OnProgress is called periodically with progress updates (optional)
	OnProgress func(ProgressInfo)
	// Resume enables resuming from partial downloads if the file exists
	Resume bool
}

// DownloadResult contains information about a completed download.
type DownloadResult struct {
	// BytesDownloaded is the number of bytes downloaded in this session
	BytesDownloaded int64
	// TotalBytes is the total file size (from server)
	TotalBytes int64
	// Resumed indicates whether the download was resumed from a partial file
	Resumed bool
	// ChecksumValid is true if checksum was provided and verified
	ChecksumValid bool
	// Path is the final file path
	Path string
}

// DownloadWithProgress downloads a file with progress tracking and optional resume support.
// This molecule composes:
//   - HTTP client (from config or provided)
//   - Range headers (for resume support)
//   - ProgressTracker (for speed and ETA calculation)
//   - SHA256 checksum verification (optional)
//
// Parameters:
//   - ctx: context for cancellation
//   - opts: download configuration options
//
// Returns:
//   - *DownloadResult: download statistics and verification status
//   - error: if download fails or checksum doesn't match
//
// The function supports:
//   - Resumable downloads (if opts.Resume is true and server supports Range)
//   - Progress callbacks with speed and ETA
//   - Post-download checksum verification
func DownloadWithProgress(ctx context.Context, opts DownloadOptions) (*DownloadResult, error) {
	// Validate required options
	if opts.URL == "" {
		return nil, fmt.Errorf("URL is required")
	}
	if opts.DestPath == "" {
		return nil, fmt.Errorf("DestPath is required")
	}

	// Use provided HTTP client or create a default one
	client := opts.HTTPClient
	if client == nil {
		// Create a basic HTTP client with reasonable timeout for downloads
		client = &http.Client{
			Timeout: 0, // No timeout for downloads (handled by context)
		}
	}

	// Ensure destination directory exists
	destDir := filepath.Dir(opts.DestPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Check for existing partial file if resume is enabled
	var resumeFrom int64
	if opts.Resume {
		if info, err := os.Stat(opts.DestPath); err == nil {
			resumeFrom = info.Size()
		}
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", opts.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add Range header if resuming
	if resumeFrom > 0 {
		req.Header.Set("Range", BuildRangeHeader(resumeFrom))
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle response status
	var totalSize int64
	var resumed bool

	switch resp.StatusCode {
	case http.StatusOK: // 200 - Full content
		totalSize = resp.ContentLength
		resumeFrom = 0 // Server sent full file, start fresh

	case http.StatusPartialContent: // 206 - Partial content (resume supported)
		resumed = true
		// Parse Content-Range to get total size
		if contentRange := resp.Header.Get("Content-Range"); contentRange != "" {
			_, _, total, parseErr := ParseContentRange(contentRange)
			if parseErr == nil && total > 0 {
				totalSize = total
			}
		}
		// Fallback: use Content-Length + resumeFrom
		if totalSize == 0 && resp.ContentLength > 0 {
			totalSize = resumeFrom + resp.ContentLength
		}

	case http.StatusRequestedRangeNotSatisfiable: // 416 - Range not satisfiable
		// File might be complete already, verify checksum if provided
		if opts.ExpectedSHA256 != "" {
			valid, verifyErr := VerifyChecksum(opts.DestPath, opts.ExpectedSHA256)
			if verifyErr != nil {
				return nil, fmt.Errorf("range not satisfiable and checksum verification failed: %w", verifyErr)
			}
			if valid {
				// File is already complete and verified
				info, _ := os.Stat(opts.DestPath)
				return &DownloadResult{
					BytesDownloaded: 0,
					TotalBytes:      info.Size(),
					Resumed:         true,
					ChecksumValid:   true,
					Path:            opts.DestPath,
				}, nil
			}
		}
		// Delete partial file and retry without resume
		_ = os.Remove(opts.DestPath)
		opts.Resume = false
		return DownloadWithProgress(ctx, opts)

	default:
		return nil, fmt.Errorf("unexpected status code: %d %s", resp.StatusCode, resp.Status)
	}

	// Open destination file
	var file *os.File
	if resumed {
		// Append to existing file
		file, err = os.OpenFile(opts.DestPath, os.O_APPEND|os.O_WRONLY, 0644)
	} else {
		// Create or truncate file
		file, err = os.Create(opts.DestPath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to open destination file: %w", err)
	}
	defer file.Close()

	// Initialize progress tracker
	tracker := NewProgressTracker(totalSize)
	if resumed {
		tracker.SetDownloaded(resumeFrom)
	}

	// Create a reader that updates progress
	reader := &progressReader{
		reader:     resp.Body,
		tracker:    tracker,
		onProgress: opts.OnProgress,
	}

	// Copy data to file
	bytesWritten, err := io.Copy(file, reader)
	if err != nil {
		return nil, fmt.Errorf("download interrupted: %w", err)
	}

	// Ensure file is synced to disk
	if err := file.Sync(); err != nil {
		return nil, fmt.Errorf("failed to sync file: %w", err)
	}

	// Build result
	result := &DownloadResult{
		BytesDownloaded: bytesWritten,
		TotalBytes:      totalSize,
		Resumed:         resumed,
		ChecksumValid:   false,
		Path:            opts.DestPath,
	}

	// Verify checksum if provided
	if opts.ExpectedSHA256 != "" {
		// Close file before computing checksum
		file.Close()

		valid, verifyErr := VerifyChecksum(opts.DestPath, opts.ExpectedSHA256)
		if verifyErr != nil {
			return nil, fmt.Errorf("checksum verification failed: %w", verifyErr)
		}
		if !valid {
			return nil, fmt.Errorf("checksum mismatch: file may be corrupted")
		}
		result.ChecksumValid = true
	}

	return result, nil
}

// progressReader wraps an io.Reader to track download progress.
type progressReader struct {
	reader     io.Reader
	tracker    *ProgressTracker
	onProgress func(ProgressInfo)
	// For rate-limiting progress callbacks
	lastCallback int64
}

func (r *progressReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if n > 0 {
		r.tracker.Update(int64(n))

		// Call progress callback (rate-limited to avoid excessive calls)
		if r.onProgress != nil {
			downloaded := r.tracker.Downloaded()
			// Call every ~100KB or when done
			if downloaded-r.lastCallback >= 102400 || err == io.EOF {
				r.onProgress(r.tracker.Progress())
				r.lastCallback = downloaded
			}
		}
	}
	return n, err
}

// DownloadWithProgressSimple is a convenience wrapper for simple downloads.
// It downloads a file with progress tracking but without resume or checksum.
//
// Parameters:
//   - ctx: context for cancellation
//   - url: URL to download from
//   - destPath: local file path to save to
//   - onProgress: optional callback for progress updates
//
// Returns:
//   - error: if download fails
func DownloadWithProgressSimple(ctx context.Context, url, destPath string, onProgress func(ProgressInfo)) error {
	_, err := DownloadWithProgress(ctx, DownloadOptions{
		URL:        url,
		DestPath:   destPath,
		OnProgress: onProgress,
	})
	return err
}

// DownloadWithResume downloads a file with resume support.
// If the file exists, it will attempt to resume from where it left off.
//
// Parameters:
//   - ctx: context for cancellation
//   - url: URL to download from
//   - destPath: local file path to save to
//   - expectedSHA256: expected SHA256 checksum for verification (optional, pass "" to skip)
//   - onProgress: optional callback for progress updates
//
// Returns:
//   - *DownloadResult: download statistics
//   - error: if download fails or checksum doesn't match
func DownloadWithResume(ctx context.Context, url, destPath, expectedSHA256 string, onProgress func(ProgressInfo)) (*DownloadResult, error) {
	return DownloadWithProgress(ctx, DownloadOptions{
		URL:            url,
		DestPath:       destPath,
		ExpectedSHA256: expectedSHA256,
		OnProgress:     onProgress,
		Resume:         true,
	})
}
