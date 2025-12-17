package llamaruntime

import (
	"os"
	"path/filepath"
	"testing"
)

// =============================================================================
// ValidateModelPath Tests
// =============================================================================

func TestValidateModelPath_EmptyPath(t *testing.T) {
	err := ValidateModelPath("")
	if err == nil {
		t.Error("expected error for empty path, got nil")
	}
	if err.Error() != "model path is empty" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestValidateModelPath_NotFound(t *testing.T) {
	err := ValidateModelPath("/nonexistent/path/model.gguf")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestValidateModelPath_Directory(t *testing.T) {
	// Create a temp directory
	tmpDir, err := os.MkdirTemp("", "model-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Try to validate a directory as a model
	err = ValidateModelPath(tmpDir)
	if err == nil {
		t.Error("expected error for directory, got nil")
	}
}

func TestValidateModelPath_WrongExtension(t *testing.T) {
	// Create a temp file with wrong extension
	tmpFile, err := os.CreateTemp("", "model-test-*.bin")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	err = ValidateModelPath(tmpFile.Name())
	if err == nil {
		t.Error("expected error for wrong extension, got nil")
	}
}

func TestValidateModelPath_ValidGGUF(t *testing.T) {
	// Create a temp file with GGUF extension and magic header
	tmpDir, err := os.MkdirTemp("", "model-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	modelPath := filepath.Join(tmpDir, "test-model.gguf")
	// Write GGUF magic header
	if err := os.WriteFile(modelPath, []byte("GGUF"+string(make([]byte, 100))), 0644); err != nil {
		t.Fatalf("failed to write test model: %v", err)
	}

	err = ValidateModelPath(modelPath)
	if err != nil {
		t.Errorf("expected no error for valid GGUF file, got: %v", err)
	}
}

func TestValidateModelPath_InvalidMagic(t *testing.T) {
	// Create a temp file with GGUF extension but wrong magic
	tmpDir, err := os.MkdirTemp("", "model-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	modelPath := filepath.Join(tmpDir, "test-model.gguf")
	// Write wrong magic header
	if err := os.WriteFile(modelPath, []byte("XXXX"+string(make([]byte, 100))), 0644); err != nil {
		t.Fatalf("failed to write test model: %v", err)
	}

	err = ValidateModelPath(modelPath)
	if err == nil {
		t.Error("expected error for invalid magic, got nil")
	}
}

// =============================================================================
// IsGGUFFile Tests
// =============================================================================

func TestIsGGUFFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"lowercase .gguf", "model.gguf", true},
		{"uppercase .GGUF", "model.GGUF", true},
		{"mixed case .GguF", "model.GguF", true},
		{"no extension", "model", false},
		{"wrong extension .bin", "model.bin", false},
		{"wrong extension .ggml", "model.ggml", false},
		{"gguf in name but wrong ext", "gguf-model.bin", false},
		{"empty string", "", false},
		{"path with gguf", "/path/to/model.gguf", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGGUFFile(tt.path)
			if result != tt.expected {
				t.Errorf("IsGGUFFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// ResolveModelPath Tests
// =============================================================================

func TestResolveModelPath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		modelsDir string
		expected  string
	}{
		{"empty path", "", "models", ""},
		{"absolute path", "/abs/path/model.gguf", "models", "/abs/path/model.gguf"},
		{"relative path", "model.gguf", "models", "models/model.gguf"},
		{"relative path with subdir", "subdir/model.gguf", "models", "models/subdir/model.gguf"},
		{"empty modelsDir", "model.gguf", "", "./model.gguf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ResolveModelPath(tt.path, tt.modelsDir)
			// Use filepath.Clean for comparison to handle OS differences
			if tt.expected != "" && result != filepath.Clean(tt.expected) {
				t.Errorf("ResolveModelPath(%q, %q) = %q, want %q", tt.path, tt.modelsDir, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// ModelExists Tests
// =============================================================================

func TestModelExists(t *testing.T) {
	// Test empty path
	if ModelExists("") {
		t.Error("ModelExists(\"\") should return false")
	}

	// Test nonexistent path
	if ModelExists("/nonexistent/path/model.gguf") {
		t.Error("ModelExists should return false for nonexistent file")
	}

	// Test existing file
	tmpFile, err := os.CreateTemp("", "model-test-*.gguf")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	if !ModelExists(tmpFile.Name()) {
		t.Error("ModelExists should return true for existing file")
	}

	// Test directory (should return false)
	tmpDir, err := os.MkdirTemp("", "model-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	if ModelExists(tmpDir) {
		t.Error("ModelExists should return false for directory")
	}
}

// =============================================================================
// GetModelSize Tests
// =============================================================================

func TestGetModelSize(t *testing.T) {
	// Test empty path
	if GetModelSize("") != 0 {
		t.Error("GetModelSize(\"\") should return 0")
	}

	// Test nonexistent path
	if GetModelSize("/nonexistent/path/model.gguf") != 0 {
		t.Error("GetModelSize should return 0 for nonexistent file")
	}

	// Test existing file with known size
	tmpFile, err := os.CreateTemp("", "model-test-*.gguf")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testData := []byte("test data content")
	if _, err := tmpFile.Write(testData); err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}
	tmpFile.Close()

	size := GetModelSize(tmpFile.Name())
	if size != int64(len(testData)) {
		t.Errorf("GetModelSize returned %d, want %d", size, len(testData))
	}
}

// =============================================================================
// ExtractModelName Tests
// =============================================================================

func TestExtractModelName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"", ""},
		{"model.gguf", "model"},
		{"bunny-v1.1-q4_k_m.gguf", "bunny-v1.1-q4_k_m"},
		{"/path/to/llama-7b.gguf", "llama-7b"},
		{"models/mistral-7b-instruct.gguf", "mistral-7b-instruct"},
		{"model", "model"},     // No extension
		{"model.bin", "model"}, // Different extension
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := ExtractModelName(tt.path)
			if result != tt.expected {
				t.Errorf("ExtractModelName(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

// =============================================================================
// DefaultModelPathConfig Tests
// =============================================================================

func TestDefaultModelPathConfig(t *testing.T) {
	cfg := DefaultModelPathConfig()

	if cfg.ModelsDir != "models" {
		t.Errorf("ModelsDir = %q, want %q", cfg.ModelsDir, "models")
	}
	if cfg.AllowDownload != false {
		t.Error("AllowDownload should be false by default")
	}
	if cfg.ModelPath != "" {
		t.Errorf("ModelPath should be empty, got %q", cfg.ModelPath)
	}
	if cfg.ModelURL != "" {
		t.Errorf("ModelURL should be empty, got %q", cfg.ModelURL)
	}
}

// =============================================================================
// ResolveModelPathConfig Tests
// =============================================================================

func TestResolveModelPathConfig_EmptyPath(t *testing.T) {
	cfg := ModelPathConfig{}
	_, err := ResolveModelPathConfig(cfg, nil)
	if err == nil {
		t.Error("expected error for empty model path")
	}
}

func TestResolveModelPathConfig_NotFound_NoDownload(t *testing.T) {
	cfg := ModelPathConfig{
		ModelPath:     "nonexistent.gguf",
		ModelsDir:     "/tmp/nonexistent-models",
		AllowDownload: false,
	}
	_, err := ResolveModelPathConfig(cfg, nil)
	if err == nil {
		t.Error("expected error when model not found and download disabled")
	}
}

func TestResolveModelPathConfig_NotFound_NoURL(t *testing.T) {
	cfg := ModelPathConfig{
		ModelPath:     "nonexistent.gguf",
		ModelsDir:     "/tmp/nonexistent-models",
		AllowDownload: true,
		ModelURL:      "", // No URL provided
	}
	_, err := ResolveModelPathConfig(cfg, nil)
	if err == nil {
		t.Error("expected error when model not found and no download URL")
	}
}

func TestResolveModelPathConfig_ExistingValid(t *testing.T) {
	// Create a valid GGUF file
	tmpDir, err := os.MkdirTemp("", "model-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	modelPath := filepath.Join(tmpDir, "test-model.gguf")
	// Write GGUF magic header
	if err := os.WriteFile(modelPath, []byte("GGUF"+string(make([]byte, 100))), 0644); err != nil {
		t.Fatalf("failed to write test model: %v", err)
	}

	cfg := ModelPathConfig{
		ModelPath: modelPath,
	}

	resolved, err := ResolveModelPathConfig(cfg, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resolved != modelPath {
		t.Errorf("resolved path = %q, want %q", resolved, modelPath)
	}
}
