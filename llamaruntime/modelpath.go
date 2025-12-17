// Package llamaruntime provides Go bindings to llama.cpp for local LLM inference.
// This file contains atom functions for model path configuration and validation.
//
// These are pure functions with no dependencies on external state.
// They validate model paths and download models if needed.
package llamaruntime

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ModelPathConfig contains configuration for model path resolution.
type ModelPathConfig struct {
	// ModelPath is the path to the GGUF model file.
	// Can be absolute or relative.
	ModelPath string

	// ModelURL is an optional URL to download the model from if not found locally.
	ModelURL string

	// ModelsDir is the directory where models are stored.
	// Defaults to "models" in the current directory.
	ModelsDir string

	// AllowDownload enables automatic model download if the model is not found.
	AllowDownload bool
}

// DefaultModelPathConfig returns a ModelPathConfig with sensible defaults.
func DefaultModelPathConfig() ModelPathConfig {
	return ModelPathConfig{
		ModelsDir:     "models",
		AllowDownload: false,
	}
}

// =============================================================================
// Model Path Validation Atoms
// =============================================================================

// ValidateModelPath checks if a model path points to a valid GGUF file.
// Returns nil if valid, or an error describing what's wrong.
//
// Validation checks:
// 1. Path is not empty
// 2. File exists
// 3. File has .gguf extension
// 4. File is readable
func ValidateModelPath(path string) error {
	if path == "" {
		return fmt.Errorf("model path is empty")
	}

	// Check if file exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("model file not found: %s", path)
	}
	if err != nil {
		return fmt.Errorf("cannot access model file: %w", err)
	}

	// Check if it's a regular file (not a directory)
	if info.IsDir() {
		return fmt.Errorf("model path is a directory, not a file: %s", path)
	}

	// Check for .gguf extension
	if !IsGGUFFile(path) {
		return fmt.Errorf("model file does not have .gguf extension: %s", path)
	}

	// Check if file is readable
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("cannot open model file: %w", err)
	}
	defer f.Close()

	// Read first 4 bytes to verify it's a valid GGUF file
	header := make([]byte, 4)
	n, err := f.Read(header)
	if err != nil || n < 4 {
		return fmt.Errorf("cannot read model file header: %w", err)
	}

	// GGUF magic number: "GGUF" (0x46554747)
	if string(header) != "GGUF" {
		return fmt.Errorf("invalid GGUF file: magic number mismatch (got %q)", string(header))
	}

	return nil
}

// IsGGUFFile returns true if the path has a .gguf extension.
func IsGGUFFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".gguf")
}

// ResolveModelPath resolves a model path to an absolute path.
// If the path is relative, it's resolved relative to modelsDir.
// If the path is absolute, it's returned as-is.
func ResolveModelPath(path, modelsDir string) string {
	if path == "" {
		return ""
	}

	// If path is absolute, return as-is
	if filepath.IsAbs(path) {
		return path
	}

	// If modelsDir is empty, use current directory
	if modelsDir == "" {
		modelsDir = "."
	}

	return filepath.Join(modelsDir, path)
}

// ModelExists checks if a model file exists at the given path.
func ModelExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// GetModelSize returns the size of the model file in bytes.
// Returns 0 if the file doesn't exist or can't be accessed.
func GetModelSize(path string) int64 {
	if path == "" {
		return 0
	}
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// ExtractModelName extracts the model name from a file path.
// For "models/bunny-v1.1-q4_k_m.gguf", returns "bunny-v1.1-q4_k_m".
func ExtractModelName(path string) string {
	if path == "" {
		return ""
	}
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// =============================================================================
// Model Download Atoms
// =============================================================================

// DownloadProgressCallback is called during model download with progress updates.
type DownloadProgressCallback func(downloaded, total int64)

// DownloadModel downloads a model from the given URL to the specified path.
// If progressCallback is not nil, it's called with progress updates.
//
// This is a blocking operation that downloads the entire model to disk.
// The model is first downloaded to a temporary file, then renamed to the
// target path to ensure atomicity.
func DownloadModel(url, destPath string, progressCallback DownloadProgressCallback) error {
	if url == "" {
		return fmt.Errorf("download URL is empty")
	}
	if destPath == "" {
		return fmt.Errorf("destination path is empty")
	}

	// Create parent directory if needed
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create model directory: %w", err)
	}

	// Create temporary file in the same directory (for atomic rename)
	tmpFile, err := os.CreateTemp(dir, "model-download-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up on error
	success := false
	defer func() {
		if !success {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	// Start download
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download model: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Get content length for progress reporting
	totalSize := resp.ContentLength

	// Copy with progress
	var downloaded int64
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := tmpFile.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("failed to write model data: %w", writeErr)
			}
			downloaded += int64(n)
			if progressCallback != nil {
				progressCallback(downloaded, totalSize)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading download: %w", err)
		}
	}

	// Close temp file before rename
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, destPath); err != nil {
		return fmt.Errorf("failed to move model to final location: %w", err)
	}

	success = true
	return nil
}

// =============================================================================
// Model Path Resolution
// =============================================================================

// ResolveModelPathConfig resolves the model path from config, optionally downloading.
// Returns the resolved absolute path to the model file.
//
// Resolution order:
// 1. If ModelPath is set and valid, use it
// 2. If model doesn't exist and AllowDownload is true, download from ModelURL
// 3. Return error if model can't be found or downloaded
func ResolveModelPathConfig(cfg ModelPathConfig, progressCallback DownloadProgressCallback) (string, error) {
	if cfg.ModelPath == "" {
		return "", fmt.Errorf("model path not configured")
	}

	// Resolve path
	resolvedPath := ResolveModelPath(cfg.ModelPath, cfg.ModelsDir)

	// Check if model exists
	if ModelExists(resolvedPath) {
		// Validate the existing model
		if err := ValidateModelPath(resolvedPath); err != nil {
			return "", err
		}
		return resolvedPath, nil
	}

	// Model doesn't exist - try to download if allowed
	if !cfg.AllowDownload {
		return "", fmt.Errorf("model file not found: %s", resolvedPath)
	}

	if cfg.ModelURL == "" {
		return "", fmt.Errorf("model file not found and no download URL configured: %s", resolvedPath)
	}

	// Download the model
	if err := DownloadModel(cfg.ModelURL, resolvedPath, progressCallback); err != nil {
		return "", fmt.Errorf("failed to download model: %w", err)
	}

	// Validate the downloaded model
	if err := ValidateModelPath(resolvedPath); err != nil {
		// Remove invalid download
		os.Remove(resolvedPath)
		return "", fmt.Errorf("downloaded model is invalid: %w", err)
	}

	return resolvedPath, nil
}
