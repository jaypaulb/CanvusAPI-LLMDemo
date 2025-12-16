package imagegen

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go_backend/canvusapi"
	"go_backend/core"
	"go_backend/logging"
)

// mockProvider implements Provider for testing
type mockProvider struct {
	generateFunc func(ctx context.Context, prompt string) (string, error)
	generateErr  error
	imageURL     string
}

func (m *mockProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, prompt)
	}
	if m.generateErr != nil {
		return "", m.generateErr
	}
	return m.imageURL, nil
}

// newTestLogger creates a logger for testing
func newTestLogger(t *testing.T) *logging.Logger {
	tmpDir := t.TempDir()
	logger, err := logging.NewLogger(true, filepath.Join(tmpDir, "test.log"))
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	t.Cleanup(func() { logger.Sync() })
	return logger
}

// newMockCanvusServer creates a mock Canvus API server for testing
func newMockCanvusServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a simple success response for all requests
		response := map[string]interface{}{
			"id":   "mock-widget-123",
			"type": "image",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
}

// newMockImageServer creates a server that returns a mock image
func newMockImageServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a minimal PNG
		pngData := []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x01, 0x00, 0x00, 0x00, 0x00, 0x37, 0x6E, 0xF9,
			0x24, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
			0x54, 0x78, 0x9C, 0x62, 0x60, 0x00, 0x00, 0x00,
			0x02, 0x00, 0x01, 0xE5, 0x27, 0xDE, 0xFC, 0x00,
			0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
			0x42, 0x60, 0x82, // IEND chunk
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(pngData)
	}))
}

func TestNewGenerator_NilProvider(t *testing.T) {
	logger := newTestLogger(t)
	server := newMockCanvusServer()
	defer server.Close()

	client := canvusapi.NewClient(server.URL, "test-canvas", "test-key", false)
	downloader, _ := NewDownloaderWithConfig(DownloaderConfig{
		DownloadsDir: t.TempDir(),
		Timeout:      10 * time.Second,
	}, nil)

	_, err := NewGenerator(nil, downloader, client, logger, DefaultGeneratorConfig())
	if err == nil {
		t.Error("expected error for nil provider")
	}
}

func TestNewGenerator_NilDownloader(t *testing.T) {
	logger := newTestLogger(t)
	server := newMockCanvusServer()
	defer server.Close()

	client := canvusapi.NewClient(server.URL, "test-canvas", "test-key", false)
	provider := &mockProvider{imageURL: "http://example.com/image.png"}

	_, err := NewGenerator(provider, nil, client, logger, DefaultGeneratorConfig())
	if err == nil {
		t.Error("expected error for nil downloader")
	}
}

func TestNewGenerator_NilClient(t *testing.T) {
	logger := newTestLogger(t)
	downloader, _ := NewDownloaderWithConfig(DownloaderConfig{
		DownloadsDir: t.TempDir(),
		Timeout:      10 * time.Second,
	}, nil)
	provider := &mockProvider{imageURL: "http://example.com/image.png"}

	_, err := NewGenerator(provider, downloader, nil, logger, DefaultGeneratorConfig())
	if err == nil {
		t.Error("expected error for nil client")
	}
}

func TestNewGenerator_NilLogger(t *testing.T) {
	server := newMockCanvusServer()
	defer server.Close()

	client := canvusapi.NewClient(server.URL, "test-canvas", "test-key", false)
	downloader, _ := NewDownloaderWithConfig(DownloaderConfig{
		DownloadsDir: t.TempDir(),
		Timeout:      10 * time.Second,
	}, nil)
	provider := &mockProvider{imageURL: "http://example.com/image.png"}

	_, err := NewGenerator(provider, downloader, client, nil, DefaultGeneratorConfig())
	if err == nil {
		t.Error("expected error for nil logger")
	}
}

func TestNewGenerator_ValidConfig(t *testing.T) {
	logger := newTestLogger(t)
	server := newMockCanvusServer()
	defer server.Close()

	client := canvusapi.NewClient(server.URL, "test-canvas", "test-key", false)

	tempDir := t.TempDir()
	downloader, err := NewDownloaderWithConfig(DownloaderConfig{
		DownloadsDir: tempDir,
		Timeout:      10 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	provider := &mockProvider{imageURL: "http://example.com/image.png"}

	config := DefaultGeneratorConfig()
	config.DownloadsDir = tempDir

	generator, err := NewGenerator(provider, downloader, client, logger, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if generator == nil {
		t.Fatal("expected non-nil generator")
	}

	// Verify accessors
	if generator.Provider() != provider {
		t.Error("Provider() returned wrong provider")
	}
	if generator.Downloader() != downloader {
		t.Error("Downloader() returned wrong downloader")
	}
	if generator.Config().DownloadsDir != tempDir {
		t.Error("Config() returned wrong config")
	}
}

func TestNewGenerator_CreatesDownloadsDir(t *testing.T) {
	logger := newTestLogger(t)
	server := newMockCanvusServer()
	defer server.Close()

	client := canvusapi.NewClient(server.URL, "test-canvas", "test-key", false)

	// Use a subdirectory that doesn't exist yet
	tempDir := t.TempDir()
	newDir := tempDir + "/subdir/downloads"

	downloader, err := NewDownloaderWithConfig(DownloaderConfig{
		DownloadsDir: tempDir, // Use temp dir for downloader
		Timeout:      10 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	provider := &mockProvider{imageURL: "http://example.com/image.png"}

	config := DefaultGeneratorConfig()
	config.DownloadsDir = newDir

	_, err = NewGenerator(provider, downloader, client, logger, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		t.Error("expected downloads directory to be created")
	}
}

func TestDefaultGeneratorConfig(t *testing.T) {
	config := DefaultGeneratorConfig()

	if config.DownloadsDir != "downloads" {
		t.Errorf("expected DownloadsDir 'downloads', got %q", config.DownloadsDir)
	}
	if !config.CleanupTempFiles {
		t.Error("expected CleanupTempFiles to be true by default")
	}
	if config.ProcessingNote.Title == "" {
		t.Error("expected ProcessingNote.Title to be set")
	}
}

func TestGenerator_Generate_EmptyPrompt(t *testing.T) {
	logger := newTestLogger(t)
	canvusServer := newMockCanvusServer()
	defer canvusServer.Close()

	client := canvusapi.NewClient(canvusServer.URL, "test-canvas", "test-key", false)

	tempDir := t.TempDir()
	downloader, _ := NewDownloaderWithConfig(DownloaderConfig{
		DownloadsDir: tempDir,
		Timeout:      10 * time.Second,
	}, nil)

	provider := &mockProvider{imageURL: "http://example.com/image.png"}

	config := DefaultGeneratorConfig()
	config.DownloadsDir = tempDir

	generator, err := NewGenerator(provider, downloader, client, logger, config)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	parent := CanvasWidget{
		ID:       "parent-123",
		Location: WidgetLocation{X: 100, Y: 200},
		Size:     WidgetSize{Width: 300, Height: 200},
		Scale:    1.0,
		Depth:    0,
	}

	ctx := context.Background()
	_, err = generator.Generate(ctx, "", parent)
	if err == nil {
		t.Error("expected error for empty prompt")
	}
}

func TestGenerator_Generate_ProviderError(t *testing.T) {
	logger := newTestLogger(t)
	canvusServer := newMockCanvusServer()
	defer canvusServer.Close()

	client := canvusapi.NewClient(canvusServer.URL, "test-canvas", "test-key", false)

	tempDir := t.TempDir()
	downloader, _ := NewDownloaderWithConfig(DownloaderConfig{
		DownloadsDir: tempDir,
		Timeout:      10 * time.Second,
	}, nil)

	provider := &mockProvider{
		generateErr: context.DeadlineExceeded,
	}

	config := DefaultGeneratorConfig()
	config.DownloadsDir = tempDir

	generator, err := NewGenerator(provider, downloader, client, logger, config)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	parent := CanvasWidget{
		ID:       "parent-123",
		Location: WidgetLocation{X: 100, Y: 200},
		Size:     WidgetSize{Width: 300, Height: 200},
		Scale:    1.0,
		Depth:    0,
	}

	ctx := context.Background()
	_, err = generator.Generate(ctx, "a beautiful sunset", parent)
	if err == nil {
		t.Error("expected error when provider fails")
	}
}

func TestGenerator_Generate_Success(t *testing.T) {
	logger := newTestLogger(t)

	// Create image server first
	imageServer := newMockImageServer()
	defer imageServer.Close()

	// Create Canvus server
	canvusServer := newMockCanvusServer()
	defer canvusServer.Close()

	client := canvusapi.NewClient(canvusServer.URL, "test-canvas", "test-key", false)

	tempDir := t.TempDir()
	downloader, err := NewDownloaderWithConfig(DownloaderConfig{
		DownloadsDir: tempDir,
		Timeout:      10 * time.Second,
	}, nil)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	// Provider returns image server URL
	provider := &mockProvider{imageURL: imageServer.URL + "/image.png"}

	config := DefaultGeneratorConfig()
	config.DownloadsDir = tempDir
	config.CleanupTempFiles = false // Don't cleanup so we can verify

	generator, err := NewGenerator(provider, downloader, client, logger, config)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	parent := CanvasWidget{
		ID:       "parent-123",
		Location: WidgetLocation{X: 100, Y: 200},
		Size:     WidgetSize{Width: 300, Height: 200},
		Scale:    1.0,
		Depth:    0,
	}

	ctx := context.Background()
	result, err := generator.Generate(ctx, "a beautiful sunset", parent)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.WidgetID == "" {
		t.Error("expected non-empty WidgetID")
	}
	if result.ImageURL == "" {
		t.Error("expected non-empty ImageURL")
	}
}

func TestGenerator_Generate_ContextCancellation(t *testing.T) {
	logger := newTestLogger(t)

	canvusServer := newMockCanvusServer()
	defer canvusServer.Close()

	client := canvusapi.NewClient(canvusServer.URL, "test-canvas", "test-key", false)

	tempDir := t.TempDir()
	downloader, _ := NewDownloaderWithConfig(DownloaderConfig{
		DownloadsDir: tempDir,
		Timeout:      10 * time.Second,
	}, nil)

	// Provider that blocks until context is cancelled
	provider := &mockProvider{
		generateFunc: func(ctx context.Context, prompt string) (string, error) {
			<-ctx.Done()
			return "", ctx.Err()
		},
	}

	config := DefaultGeneratorConfig()
	config.DownloadsDir = tempDir

	generator, err := NewGenerator(provider, downloader, client, logger, config)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	parent := CanvasWidget{
		ID:       "parent-123",
		Location: WidgetLocation{X: 100, Y: 200},
		Size:     WidgetSize{Width: 300, Height: 200},
		Scale:    1.0,
		Depth:    0,
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	_, err = generator.Generate(ctx, "a beautiful sunset", parent)
	if err == nil {
		t.Error("expected error when context is cancelled")
	}
}

func TestNewGeneratorFromConfig_NilConfig(t *testing.T) {
	logger := newTestLogger(t)
	canvusServer := newMockCanvusServer()
	defer canvusServer.Close()

	client := canvusapi.NewClient(canvusServer.URL, "test-canvas", "test-key", false)

	_, err := NewGeneratorFromConfig(nil, client, logger)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestNewGeneratorFromConfig_NilClient(t *testing.T) {
	logger := newTestLogger(t)
	cfg := &core.Config{
		OpenAIAPIKey: "test-key",
		DownloadsDir: os.TempDir(),
	}

	_, err := NewGeneratorFromConfig(cfg, nil, logger)
	if err == nil {
		t.Error("expected error for nil client")
	}
}

func TestNewGeneratorFromConfig_NilLogger(t *testing.T) {
	canvusServer := newMockCanvusServer()
	defer canvusServer.Close()

	client := canvusapi.NewClient(canvusServer.URL, "test-canvas", "test-key", false)
	cfg := &core.Config{
		OpenAIAPIKey: "test-key",
		DownloadsDir: os.TempDir(),
	}

	_, err := NewGeneratorFromConfig(cfg, client, nil)
	if err == nil {
		t.Error("expected error for nil logger")
	}
}

func TestNewGeneratorFromConfig_SelectsOpenAIProvider(t *testing.T) {
	logger := newTestLogger(t)
	canvusServer := newMockCanvusServer()
	defer canvusServer.Close()

	client := canvusapi.NewClient(canvusServer.URL, "test-canvas", "test-key", false)
	cfg := &core.Config{
		OpenAIAPIKey: "sk-test-key-12345",
		ImageLLMURL:  "https://api.openai.com/v1",
		DownloadsDir: t.TempDir(),
	}

	generator, err := NewGeneratorFromConfig(cfg, client, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if generator == nil {
		t.Fatal("expected non-nil generator")
	}

	// Provider should be OpenAIProvider
	_, ok := generator.Provider().(*OpenAIProvider)
	if !ok {
		t.Error("expected OpenAIProvider to be selected")
	}
}

func TestNewGeneratorFromConfig_SelectsAzureProvider(t *testing.T) {
	logger := newTestLogger(t)
	canvusServer := newMockCanvusServer()
	defer canvusServer.Close()

	client := canvusapi.NewClient(canvusServer.URL, "test-canvas", "test-key", false)
	cfg := &core.Config{
		OpenAIAPIKey:          "test-key",
		AzureOpenAIEndpoint:   "https://myresource.openai.azure.com/",
		AzureOpenAIDeployment: "dalle3",
		DownloadsDir:          t.TempDir(),
	}

	generator, err := NewGeneratorFromConfig(cfg, client, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if generator == nil {
		t.Fatal("expected non-nil generator")
	}

	// Provider should be AzureProvider
	_, ok := generator.Provider().(*AzureProvider)
	if !ok {
		t.Error("expected AzureProvider to be selected")
	}
}

func TestNewGeneratorFromConfig_NoAPIKey(t *testing.T) {
	logger := newTestLogger(t)
	canvusServer := newMockCanvusServer()
	defer canvusServer.Close()

	client := canvusapi.NewClient(canvusServer.URL, "test-canvas", "test-key", false)
	cfg := &core.Config{
		OpenAIAPIKey: "", // Empty API key
		DownloadsDir: t.TempDir(),
	}

	_, err := NewGeneratorFromConfig(cfg, client, logger)
	if err == nil {
		t.Error("expected error for missing API key")
	}
}
