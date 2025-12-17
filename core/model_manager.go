package core

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ModelConfig holds configuration for a specific AI model.
// This is a data structure that defines model properties without behavior.
type ModelConfig struct {
	// Name is the friendly name of the model (e.g., "bunny-v1.1")
	Name string
	// URL is the download URL for the model file
	URL string
	// Filename is the local filename for the model
	Filename string
	// ExpectedSHA256 is the expected SHA256 checksum for verification
	ExpectedSHA256 string
	// SizeBytes is the expected file size in bytes (for disk space checks)
	SizeBytes int64
}

// DefaultTextModel is the default text/vision model configuration.
// Bunny-v1.1-llama-3.2-4b is a compact multimodal model suitable for most tasks.
var DefaultTextModel = ModelConfig{
	Name:           "bunny-v1.1-llama-3.2-4b",
	URL:            "https://huggingface.co/BAAI/Bunny-v1_1-4B/resolve/main/ggml-model-Q4_K_M.gguf",
	Filename:       "bunny-v1.1-llama-3.2-4b-Q4_K_M.gguf",
	ExpectedSHA256: "",             // To be filled with actual checksum when known
	SizeBytes:      3 * BytesPerGB, // ~3GB for 4B Q4_K_M model
}

// DefaultVisionModel is the vision encoder model configuration.
var DefaultVisionModel = ModelConfig{
	Name:           "bunny-mmproj",
	URL:            "https://huggingface.co/BAAI/Bunny-v1_1-4B/resolve/main/mmproj-model-f16.gguf",
	Filename:       "bunny-mmproj-f16.gguf",
	ExpectedSHA256: "",               // To be filled with actual checksum when known
	SizeBytes:      600 * BytesPerMB, // ~600MB for vision projector
}

// DefaultSDModel is the Stable Diffusion model configuration for image generation.
var DefaultSDModel = ModelConfig{
	Name:           "sd-turbo",
	URL:            "https://huggingface.co/stabilityai/sd-turbo/resolve/main/sd_turbo.safetensors",
	Filename:       "sd-turbo.safetensors",
	ExpectedSHA256: "",             // To be filled with actual checksum when known
	SizeBytes:      2 * BytesPerGB, // ~2GB for SD Turbo
}

// ModelManager manages AI model availability and downloading.
// This is an organism that composes download molecules and disk space atoms
// to provide model lifecycle management.
type ModelManager struct {
	// modelDir is the directory where models are stored
	modelDir string
	// httpClient is the HTTP client for downloads
	httpClient *http.Client
	// models holds configurations for all managed models
	models map[string]ModelConfig
	// maxRetries is the number of download retry attempts
	maxRetries int
	// baseRetryDelay is the initial delay between retries (doubles each attempt)
	baseRetryDelay time.Duration
	// diskSpaceBuffer is the percentage buffer for disk space checks
	diskSpaceBuffer int
}

// ModelManagerOption is a functional option for configuring ModelManager.
type ModelManagerOption func(*ModelManager)

// WithMaxRetries sets the maximum number of download retry attempts.
func WithMaxRetries(n int) ModelManagerOption {
	return func(mm *ModelManager) {
		if n > 0 {
			mm.maxRetries = n
		}
	}
}

// WithBaseRetryDelay sets the base delay between retry attempts.
func WithBaseRetryDelay(d time.Duration) ModelManagerOption {
	return func(mm *ModelManager) {
		if d > 0 {
			mm.baseRetryDelay = d
		}
	}
}

// WithDiskSpaceBuffer sets the disk space buffer percentage.
func WithDiskSpaceBuffer(percent int) ModelManagerOption {
	return func(mm *ModelManager) {
		if percent >= 0 {
			mm.diskSpaceBuffer = percent
		}
	}
}

// WithModel registers a model configuration.
func WithModel(model ModelConfig) ModelManagerOption {
	return func(mm *ModelManager) {
		mm.models[model.Name] = model
	}
}

// NewModelManager creates a new ModelManager with the given configuration.
// The modelDir parameter specifies where models are stored.
// The httpClient parameter is used for downloads (if nil, a default client is created).
//
// Default behavior:
//   - 3 retry attempts with exponential backoff (2s, 4s, 8s)
//   - 10% disk space buffer
//   - Default text, vision, and SD models registered
func NewModelManager(modelDir string, httpClient *http.Client, opts ...ModelManagerOption) *ModelManager {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 0, // No timeout for large downloads; context handles cancellation
		}
	}

	mm := &ModelManager{
		modelDir:        modelDir,
		httpClient:      httpClient,
		models:          make(map[string]ModelConfig),
		maxRetries:      3,
		baseRetryDelay:  2 * time.Second,
		diskSpaceBuffer: DefaultBufferPercent,
	}

	// Register default models
	mm.models[DefaultTextModel.Name] = DefaultTextModel
	mm.models[DefaultVisionModel.Name] = DefaultVisionModel
	mm.models[DefaultSDModel.Name] = DefaultSDModel

	// Apply options
	for _, opt := range opts {
		opt(mm)
	}

	return mm
}

// EnsureModelAvailable checks if a model exists and downloads it if missing.
// This is the main entry point for model management.
//
// Parameters:
//   - ctx: context for cancellation
//   - modelName: name of the model to ensure (must be registered)
//
// Returns:
//   - error: if model cannot be made available (disk space, download failure, checksum mismatch)
//
// The function:
//  1. Checks if model file already exists (returns nil if found)
//  2. Verifies sufficient disk space
//  3. Downloads with retries (3 attempts, exponential backoff)
//  4. Verifies checksum after download (if provided)
func (mm *ModelManager) EnsureModelAvailable(ctx context.Context, modelName string) error {
	// Lookup model configuration
	modelCfg, ok := mm.models[modelName]
	if !ok {
		return fmt.Errorf("unknown model: %q (available: %v)", modelName, mm.availableModelNames())
	}

	modelPath := filepath.Join(mm.modelDir, modelCfg.Filename)

	// Check if model already exists
	exists, err := mm.checkModelExists(modelPath, modelCfg.ExpectedSHA256)
	if err != nil {
		return fmt.Errorf("check model exists: %w", err)
	}
	if exists {
		return nil
	}

	// Download the model
	return mm.downloadModel(ctx, modelCfg, modelPath)
}

// checkModelExists verifies if a model file exists and optionally validates checksum.
//
// Parameters:
//   - modelPath: full path to the model file
//   - expectedChecksum: expected SHA256 (optional, empty string skips verification)
//
// Returns:
//   - bool: true if model exists (and checksum matches if provided)
//   - error: if file exists but checksum verification fails
func (mm *ModelManager) checkModelExists(modelPath string, expectedChecksum string) (bool, error) {
	// Check if file exists
	info, err := os.Stat(modelPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("stat model file: %w", err)
	}

	// File exists but is empty or directory
	if info.IsDir() {
		return false, fmt.Errorf("model path is a directory: %s", modelPath)
	}
	if info.Size() == 0 {
		// Empty file, treat as not existing
		return false, nil
	}

	// If no checksum provided, existence is sufficient
	if expectedChecksum == "" {
		return true, nil
	}

	// Verify checksum
	valid, err := VerifyChecksum(modelPath, expectedChecksum)
	if err != nil {
		return false, fmt.Errorf("verify checksum: %w", err)
	}
	if !valid {
		// File exists but checksum doesn't match - corrupt file
		return false, fmt.Errorf("model file corrupted: checksum mismatch for %s", modelPath)
	}

	return true, nil
}

// downloadModel downloads a model with retries and verification.
//
// Parameters:
//   - ctx: context for cancellation
//   - modelCfg: model configuration with URL, checksum, etc.
//   - destPath: destination path for the downloaded file
//
// Returns:
//   - error: if download fails after all retries
func (mm *ModelManager) downloadModel(ctx context.Context, modelCfg ModelConfig, destPath string) error {
	// Check disk space before download
	if modelCfg.SizeBytes > 0 {
		if err := CheckDiskSpaceForModel(mm.modelDir, modelCfg.SizeBytes, mm.diskSpaceBuffer); err != nil {
			return &ModelDownloadError{
				ModelName: modelCfg.Name,
				Cause:     err,
				Message:   fmt.Sprintf("insufficient disk space: need %s with %d%% buffer", FormatBytes(modelCfg.SizeBytes), mm.diskSpaceBuffer),
			}
		}
	}

	// Ensure model directory exists
	if err := os.MkdirAll(mm.modelDir, 0755); err != nil {
		return fmt.Errorf("create model directory: %w", err)
	}

	// Attempt download with retries
	var lastErr error
	for attempt := 1; attempt <= mm.maxRetries; attempt++ {
		// Check context cancellation before each attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Calculate retry delay (exponential backoff: 2s, 4s, 8s)
		if attempt > 1 {
			delay := mm.baseRetryDelay * time.Duration(1<<(attempt-2)) // 2^(attempt-2) * baseDelay
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		// Attempt download
		err := mm.attemptDownload(ctx, modelCfg, destPath, attempt)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is retryable
		if !mm.isRetryableError(err) {
			break
		}
	}

	// All retries exhausted
	return &ModelDownloadError{
		ModelName: modelCfg.Name,
		Cause:     lastErr,
		Message:   fmt.Sprintf("download failed after %d attempts", mm.maxRetries),
		URL:       modelCfg.URL,
		DestPath:  destPath,
		Checksum:  modelCfg.ExpectedSHA256,
	}
}

// attemptDownload performs a single download attempt.
func (mm *ModelManager) attemptDownload(ctx context.Context, modelCfg ModelConfig, destPath string, attempt int) error {
	opts := DownloadOptions{
		URL:            modelCfg.URL,
		DestPath:       destPath,
		ExpectedSHA256: modelCfg.ExpectedSHA256,
		HTTPClient:     mm.httpClient,
		Resume:         true, // Enable resume for large model files
	}

	_, err := DownloadWithProgress(ctx, opts)
	return err
}

// isRetryableError determines if an error is worth retrying.
// Network errors and timeouts are retryable; checksum mismatches are not.
func (mm *ModelManager) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Context cancellation is not retryable
	if err == context.Canceled || err == context.DeadlineExceeded {
		return false
	}

	// Checksum mismatch is not retryable (file is corrupt)
	errStr := err.Error()
	if strings.Contains(errStr, "checksum mismatch") {
		return false
	}

	// Disk space error is not retryable
	if _, ok := err.(*DiskSpaceError); ok {
		return false
	}

	// Most other errors (network, HTTP) are retryable
	return true
}

// availableModelNames returns the names of all registered models.
func (mm *ModelManager) availableModelNames() []string {
	names := make([]string, 0, len(mm.models))
	for name := range mm.models {
		names = append(names, name)
	}
	return names
}

// GetModelPath returns the full path to a model file.
// Does not verify the file exists.
func (mm *ModelManager) GetModelPath(modelName string) (string, error) {
	modelCfg, ok := mm.models[modelName]
	if !ok {
		return "", fmt.Errorf("unknown model: %q", modelName)
	}
	return filepath.Join(mm.modelDir, modelCfg.Filename), nil
}

// ModelDownloadError provides detailed information about a download failure.
// Implements error interface with actionable guidance.
type ModelDownloadError struct {
	// ModelName is the name of the model that failed to download
	ModelName string
	// Cause is the underlying error
	Cause error
	// Message is a human-readable description
	Message string
	// URL is the download URL (for manual download instructions)
	URL string
	// DestPath is where the model should be saved
	DestPath string
	// Checksum is the expected checksum (for verification)
	Checksum string
}

func (e *ModelDownloadError) Error() string {
	if e.URL != "" && e.DestPath != "" {
		return fmt.Sprintf(`model download failed: %s

%s

Manual download instructions:
  1. Visit: %s
  2. Save to: %s
  3. Verify SHA256: %s
  4. Restart the application

For help, see: https://github.com/yourusername/CanvusLocalLLM/wiki/manual-model-download`,
			e.ModelName, e.Message, e.URL, e.DestPath, e.Checksum)
	}
	return fmt.Sprintf("model download failed: %s: %s", e.ModelName, e.Message)
}

func (e *ModelDownloadError) Unwrap() error {
	return e.Cause
}
