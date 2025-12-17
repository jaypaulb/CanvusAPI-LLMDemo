// Package llamaruntime provides Go bindings to llama.cpp for local LLM inference.
// This file contains the ModelLoader molecule - model loading and validation.
//
// The ModelLoader composes atoms from modelpath.go to load and validate
// GGUF models with proper metadata logging and startup verification.
package llamaruntime

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

// =============================================================================
// Model Loader Configuration
// =============================================================================

// ModelLoaderConfig contains configuration for the ModelLoader.
type ModelLoaderConfig struct {
	// ModelPath is the path to the GGUF model file.
	// Can be absolute or relative to ModelsDir.
	ModelPath string

	// ModelURL is an optional URL to download the model from.
	ModelURL string

	// ModelsDir is the directory for storing models.
	ModelsDir string

	// AllowDownload enables automatic model download if not found.
	AllowDownload bool

	// RunStartupTest enables a startup inference test after loading.
	RunStartupTest bool

	// StartupTestPrompt is the prompt for the startup test.
	StartupTestPrompt string

	// StartupTestTimeout is the timeout for the startup test.
	StartupTestTimeout time.Duration

	// Logger is an optional logger for model loading events.
	// If nil, uses standard log.
	Logger *log.Logger
}

// DefaultModelLoaderConfig returns a ModelLoaderConfig with sensible defaults.
func DefaultModelLoaderConfig() ModelLoaderConfig {
	return ModelLoaderConfig{
		ModelsDir:          "models",
		AllowDownload:      false,
		RunStartupTest:     true,
		StartupTestPrompt:  "Hello",
		StartupTestTimeout: 30 * time.Second,
	}
}

// =============================================================================
// Model Metadata
// =============================================================================

// ModelMetadata contains information about a loaded model.
type ModelMetadata struct {
	// Path is the absolute path to the model file.
	Path string

	// Name is the extracted model name (without extension).
	Name string

	// Size is the model file size in bytes.
	Size int64

	// SizeHuman is the human-readable size (e.g., "4.5 GB").
	SizeHuman string

	// VocabSize is the vocabulary size.
	VocabSize int

	// ContextSize is the context window size.
	ContextSize int

	// EmbeddingSize is the embedding dimension.
	EmbeddingSize int

	// LoadedAt is when the model was loaded.
	LoadedAt time.Time

	// StartupTestPassed indicates if the startup test passed (if run).
	StartupTestPassed bool

	// StartupTestDuration is how long the startup test took.
	StartupTestDuration time.Duration
}

// =============================================================================
// Model Loader
// =============================================================================

// ModelLoader handles model loading, validation, and initialization.
type ModelLoader struct {
	config   ModelLoaderConfig
	logger   *log.Logger
	metadata *ModelMetadata
}

// NewModelLoader creates a new ModelLoader with the given configuration.
func NewModelLoader(config ModelLoaderConfig) *ModelLoader {
	logger := config.Logger
	if logger == nil {
		logger = log.New(os.Stdout, "[ModelLoader] ", log.LstdFlags)
	}

	return &ModelLoader{
		config: config,
		logger: logger,
	}
}

// Load loads and validates the model, returning a configured Client.
// If RunStartupTest is enabled, it runs a startup inference test.
func (m *ModelLoader) Load(ctx context.Context) (*Client, error) {
	m.logger.Printf("Starting model load: %s", m.config.ModelPath)

	// Step 1: Resolve the model path
	resolvedPath, err := m.resolveModelPath()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve model path: %w", err)
	}
	m.logger.Printf("Resolved model path: %s", resolvedPath)

	// Step 2: Validate the model file
	if err := ValidateModelPath(resolvedPath); err != nil {
		return nil, fmt.Errorf("model validation failed: %w", err)
	}
	m.logger.Printf("Model validation passed")

	// Step 3: Extract metadata
	metadata := m.extractMetadata(resolvedPath)
	m.metadata = metadata
	m.logModelMetadata(metadata)

	// Step 4: Create the Client
	clientConfig := DefaultClientConfig()
	clientConfig.ModelPath = resolvedPath

	client, err := NewClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Step 5: Update metadata with model info from client
	m.updateMetadataFromClient(client)

	// Step 6: Run startup test if enabled
	if m.config.RunStartupTest {
		if err := m.runStartupTest(ctx, client); err != nil {
			client.Close()
			return nil, fmt.Errorf("startup test failed: %w", err)
		}
	}

	m.logger.Printf("Model loaded successfully: %s", metadata.Name)
	return client, nil
}

// Metadata returns the metadata of the loaded model.
// Returns nil if the model hasn't been loaded yet.
func (m *ModelLoader) Metadata() *ModelMetadata {
	return m.metadata
}

// resolveModelPath resolves the model path, downloading if necessary.
func (m *ModelLoader) resolveModelPath() (string, error) {
	cfg := ModelPathConfig{
		ModelPath:     m.config.ModelPath,
		ModelURL:      m.config.ModelURL,
		ModelsDir:     m.config.ModelsDir,
		AllowDownload: m.config.AllowDownload,
	}

	// If download is allowed, provide a progress callback
	var progressCallback DownloadProgressCallback
	if m.config.AllowDownload && m.config.ModelURL != "" {
		progressCallback = func(downloaded, total int64) {
			if total > 0 {
				pct := float64(downloaded) / float64(total) * 100
				m.logger.Printf("Downloading model: %.1f%% (%s / %s)",
					pct, formatBytes(downloaded), formatBytes(total))
			}
		}
	}

	return ResolveModelPathConfig(cfg, progressCallback)
}

// extractMetadata extracts model metadata from the file.
func (m *ModelLoader) extractMetadata(path string) *ModelMetadata {
	size := GetModelSize(path)
	return &ModelMetadata{
		Path:      path,
		Name:      ExtractModelName(path),
		Size:      size,
		SizeHuman: formatBytes(size),
		LoadedAt:  time.Now(),
	}
}

// updateMetadataFromClient updates metadata with info from the loaded model.
func (m *ModelLoader) updateMetadataFromClient(client *Client) {
	if m.metadata == nil {
		return
	}

	// Get model info from health check
	health, err := client.HealthCheck()
	if err == nil && health != nil && health.ModelInfo != nil {
		m.metadata.VocabSize = health.ModelInfo.VocabSize
		m.metadata.ContextSize = health.ModelInfo.ContextLength
		m.metadata.EmbeddingSize = health.ModelInfo.EmbeddingLength
	}
}

// logModelMetadata logs the model metadata.
func (m *ModelLoader) logModelMetadata(metadata *ModelMetadata) {
	m.logger.Printf("=== Model Information ===")
	m.logger.Printf("  Name: %s", metadata.Name)
	m.logger.Printf("  Path: %s", metadata.Path)
	m.logger.Printf("  Size: %s", metadata.SizeHuman)
	if metadata.VocabSize > 0 {
		m.logger.Printf("  Vocab Size: %d", metadata.VocabSize)
	}
	if metadata.ContextSize > 0 {
		m.logger.Printf("  Context Size: %d", metadata.ContextSize)
	}
	if metadata.EmbeddingSize > 0 {
		m.logger.Printf("  Embedding Size: %d", metadata.EmbeddingSize)
	}
}

// runStartupTest runs a simple inference test to verify the model works.
func (m *ModelLoader) runStartupTest(ctx context.Context, client *Client) error {
	m.logger.Printf("Running startup inference test...")

	// Create context with timeout
	testCtx, cancel := context.WithTimeout(ctx, m.config.StartupTestTimeout)
	defer cancel()

	// Run a simple inference
	params := DefaultInferenceParams()
	params.Prompt = m.config.StartupTestPrompt
	params.MaxTokens = 10 // Just a few tokens for testing

	startTime := time.Now()
	result, err := client.Infer(testCtx, params)
	duration := time.Since(startTime)

	if err != nil {
		m.metadata.StartupTestPassed = false
		return fmt.Errorf("startup test inference failed: %w", err)
	}

	m.metadata.StartupTestPassed = true
	m.metadata.StartupTestDuration = duration

	m.logger.Printf("Startup test passed in %v", duration)
	m.logger.Printf("  Response preview: %s", truncateForLog(result.Text, 50))

	return nil
}

// =============================================================================
// Helper Functions
// =============================================================================

// formatBytes formats bytes to human-readable format.
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}

// truncateForLog truncates a string for logging.
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// =============================================================================
// Convenience Functions
// =============================================================================

// LoadModel is a convenience function that loads a model with default settings.
// Returns a Client ready for inference.
func LoadModel(modelPath string) (*Client, error) {
	config := DefaultModelLoaderConfig()
	config.ModelPath = modelPath

	loader := NewModelLoader(config)
	return loader.Load(context.Background())
}

// LoadModelWithConfig loads a model with custom configuration.
func LoadModelWithConfig(config ModelLoaderConfig) (*Client, error) {
	loader := NewModelLoader(config)
	return loader.Load(context.Background())
}

// MustLoadModel loads a model and panics on error.
// Use only in development or when model loading is critical.
func MustLoadModel(modelPath string) *Client {
	client, err := LoadModel(modelPath)
	if err != nil {
		panic(fmt.Sprintf("failed to load model: %v", err))
	}
	return client
}
