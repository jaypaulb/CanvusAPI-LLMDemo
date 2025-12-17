package llamaruntime

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// =============================================================================
// ModelLoaderConfig Tests
// =============================================================================

func TestDefaultModelLoaderConfig(t *testing.T) {
	cfg := DefaultModelLoaderConfig()

	if cfg.ModelsDir != "models" {
		t.Errorf("ModelsDir = %q, want %q", cfg.ModelsDir, "models")
	}
	if cfg.AllowDownload != false {
		t.Error("AllowDownload should be false by default")
	}
	if cfg.RunStartupTest != true {
		t.Error("RunStartupTest should be true by default")
	}
	if cfg.StartupTestPrompt != "Hello" {
		t.Errorf("StartupTestPrompt = %q, want %q", cfg.StartupTestPrompt, "Hello")
	}
	if cfg.StartupTestTimeout != 30*time.Second {
		t.Errorf("StartupTestTimeout = %v, want %v", cfg.StartupTestTimeout, 30*time.Second)
	}
}

// =============================================================================
// NewModelLoader Tests
// =============================================================================

func TestNewModelLoader_DefaultLogger(t *testing.T) {
	config := DefaultModelLoaderConfig()
	loader := NewModelLoader(config)

	if loader == nil {
		t.Fatal("NewModelLoader returned nil")
	}
	if loader.logger == nil {
		t.Error("logger should not be nil")
	}
}

func TestNewModelLoader_CustomLogger(t *testing.T) {
	config := DefaultModelLoaderConfig()
	customLogger := log.New(os.Stderr, "[Test] ", log.LstdFlags)
	config.Logger = customLogger

	loader := NewModelLoader(config)

	if loader.logger != customLogger {
		t.Error("custom logger was not used")
	}
}

// =============================================================================
// ModelLoader.Load Tests
// =============================================================================

func TestModelLoader_Load_EmptyPath(t *testing.T) {
	config := DefaultModelLoaderConfig()
	config.ModelPath = ""
	config.RunStartupTest = false

	loader := NewModelLoader(config)
	_, err := loader.Load(context.Background())

	if err == nil {
		t.Error("expected error for empty model path")
	}
}

func TestModelLoader_Load_NotFound(t *testing.T) {
	config := DefaultModelLoaderConfig()
	config.ModelPath = "/nonexistent/model.gguf"
	config.RunStartupTest = false

	loader := NewModelLoader(config)
	_, err := loader.Load(context.Background())

	if err == nil {
		t.Error("expected error for nonexistent model")
	}
}

func TestModelLoader_Load_ValidModel(t *testing.T) {
	// Create a valid GGUF file
	tmpDir, err := os.MkdirTemp("", "modelloader-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	modelPath := filepath.Join(tmpDir, "test-model.gguf")
	// Write GGUF magic header
	if err := os.WriteFile(modelPath, []byte("GGUF"+string(make([]byte, 100))), 0644); err != nil {
		t.Fatalf("failed to write test model: %v", err)
	}

	config := DefaultModelLoaderConfig()
	config.ModelPath = modelPath
	config.RunStartupTest = false // Skip startup test for this test

	loader := NewModelLoader(config)
	client, err := loader.Load(context.Background())

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if client == nil {
		t.Error("client should not be nil")
	}

	// Check metadata was populated
	metadata := loader.Metadata()
	if metadata == nil {
		t.Error("metadata should not be nil")
	} else {
		if metadata.Path != modelPath {
			t.Errorf("metadata.Path = %q, want %q", metadata.Path, modelPath)
		}
		if metadata.Name != "test-model" {
			t.Errorf("metadata.Name = %q, want %q", metadata.Name, "test-model")
		}
		if metadata.Size != 104 { // "GGUF" + 100 bytes
			t.Errorf("metadata.Size = %d, want 104", metadata.Size)
		}
	}

	if client != nil {
		client.Close()
	}
}

func TestModelLoader_Load_WithStartupTest(t *testing.T) {
	// Create a valid GGUF file
	tmpDir, err := os.MkdirTemp("", "modelloader-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	modelPath := filepath.Join(tmpDir, "bunny-v1.1.gguf")
	if err := os.WriteFile(modelPath, []byte("GGUF"+string(make([]byte, 100))), 0644); err != nil {
		t.Fatalf("failed to write test model: %v", err)
	}

	config := DefaultModelLoaderConfig()
	config.ModelPath = modelPath
	config.RunStartupTest = true
	config.StartupTestTimeout = 5 * time.Second

	loader := NewModelLoader(config)
	client, err := loader.Load(context.Background())

	// In stub mode, this should work (stub returns mock responses)
	if err != nil {
		t.Errorf("unexpected error with startup test: %v", err)
	}

	metadata := loader.Metadata()
	if metadata != nil {
		if !metadata.StartupTestPassed {
			t.Error("startup test should have passed")
		}
		if metadata.StartupTestDuration == 0 {
			t.Error("startup test duration should be recorded")
		}
	}

	if client != nil {
		client.Close()
	}
}

// =============================================================================
// ModelLoader.Metadata Tests
// =============================================================================

func TestModelLoader_Metadata_BeforeLoad(t *testing.T) {
	config := DefaultModelLoaderConfig()
	loader := NewModelLoader(config)

	metadata := loader.Metadata()
	if metadata != nil {
		t.Error("metadata should be nil before Load is called")
	}
}

// =============================================================================
// ModelMetadata Tests
// =============================================================================

func TestModelMetadata_Fields(t *testing.T) {
	metadata := ModelMetadata{
		Path:                "/path/to/model.gguf",
		Name:                "model",
		Size:                1024 * 1024 * 1024, // 1 GB
		SizeHuman:           "1.00 GB",
		VocabSize:           32000,
		ContextSize:         4096,
		EmbeddingSize:       4096,
		LoadedAt:            time.Now(),
		StartupTestPassed:   true,
		StartupTestDuration: 500 * time.Millisecond,
	}

	if metadata.Path != "/path/to/model.gguf" {
		t.Error("Path field mismatch")
	}
	if metadata.Name != "model" {
		t.Error("Name field mismatch")
	}
	if metadata.VocabSize != 32000 {
		t.Error("VocabSize field mismatch")
	}
}

// =============================================================================
// formatBytes Tests
// =============================================================================

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 bytes"},
		{100, "100 bytes"},
		{1024, "1.00 KB"},
		{1536, "1.50 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{4 * 1024 * 1024 * 1024, "4.00 GB"},
		{int64(7.5 * 1024 * 1024 * 1024), "7.50 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// truncateForLog Tests
// =============================================================================

func TestTruncateForLog(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is a ..."},
		{"", 10, ""},
		{"a", 0, "..."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateForLog(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateForLog(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// Convenience Function Tests
// =============================================================================

func TestLoadModel_NotFound(t *testing.T) {
	_, err := LoadModel("/nonexistent/model.gguf")
	if err == nil {
		t.Error("expected error for nonexistent model")
	}
}

func TestLoadModelWithConfig_Valid(t *testing.T) {
	// Create a valid GGUF file
	tmpDir, err := os.MkdirTemp("", "loadmodel-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	modelPath := filepath.Join(tmpDir, "test.gguf")
	if err := os.WriteFile(modelPath, []byte("GGUF"+string(make([]byte, 100))), 0644); err != nil {
		t.Fatalf("failed to write test model: %v", err)
	}

	config := DefaultModelLoaderConfig()
	config.ModelPath = modelPath
	config.RunStartupTest = false

	client, err := LoadModelWithConfig(config)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if client != nil {
		client.Close()
	}
}

func TestMustLoadModel_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nonexistent model")
		}
	}()

	MustLoadModel("/nonexistent/model.gguf")
}
