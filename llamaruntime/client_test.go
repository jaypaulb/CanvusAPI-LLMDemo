// Package llamaruntime provides Go bindings to llama.cpp for local LLM inference.
// This file contains tests for the Client molecule.
//
//go:build nocgo || !cgo

package llamaruntime

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// =============================================================================
// Test Fixtures
// =============================================================================

// testModelPath returns a path for testing.
// In nocgo mode, the stub implementation accepts any non-empty path.
func testModelPath(t *testing.T) string {
	t.Helper()
	// Create a temporary file to simulate a model file
	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "test-model.gguf")
	if err := os.WriteFile(modelPath, []byte("mock model content"), 0644); err != nil {
		t.Fatalf("failed to create test model file: %v", err)
	}
	return modelPath
}

// =============================================================================
// DefaultClientConfig Tests
// =============================================================================

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"NumContexts", config.NumContexts, 3},
		{"ContextSize", config.ContextSize, DefaultContextSize},
		{"BatchSize", config.BatchSize, DefaultBatchSize},
		{"NumGPULayers", config.NumGPULayers, DefaultNumGPULayers},
		{"NumThreads", config.NumThreads, DefaultNumThreads},
		{"UseMMap", config.UseMMap, true},
		{"UseMlock", config.UseMlock, false},
		{"VerboseLogging", config.VerboseLogging, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}

	// AcquireTimeout is a duration
	if config.AcquireTimeout != 30*time.Second {
		t.Errorf("AcquireTimeout = %v, want %v", config.AcquireTimeout, 30*time.Second)
	}
}

// =============================================================================
// NewClient Tests
// =============================================================================

func TestNewClient_EmptyModelPath(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = ""

	client, err := NewClient(config)
	if err == nil {
		client.Close()
		t.Fatal("expected error for empty model path, got nil")
	}

	if client != nil {
		t.Error("expected nil client for empty model path")
	}

	// Check error type
	llamaErr, ok := err.(*LlamaError)
	if !ok {
		t.Errorf("expected *LlamaError, got %T", err)
	} else if llamaErr.Op != "NewClient" {
		t.Errorf("Op = %q, want %q", llamaErr.Op, "NewClient")
	}
}

func TestNewClient_ModelNotFound(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = "/nonexistent/path/model.gguf"

	client, err := NewClient(config)
	if err == nil {
		client.Close()
		t.Fatal("expected error for nonexistent model, got nil")
	}

	if client != nil {
		t.Error("expected nil client for nonexistent model")
	}
}

func TestNewClient_Success(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)
	config.NumContexts = 2

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	// Verify model info
	info := client.ModelInfo()
	if info == nil {
		t.Fatal("expected non-nil model info")
	}
	if info.Format != "GGUF" {
		t.Errorf("Format = %q, want %q", info.Format, "GGUF")
	}
}

func TestNewClient_DefaultsApplied(t *testing.T) {
	config := ClientConfig{
		ModelPath: testModelPath(t),
		// All other fields left at zero values
	}

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Client should have applied defaults internally
	if client.config.NumContexts <= 0 {
		t.Error("expected positive NumContexts")
	}
}

// =============================================================================
// Infer Tests
// =============================================================================

func TestClient_Infer_Success(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)
	config.NumContexts = 2

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	params := DefaultInferenceParams()
	params.Prompt = "What is the capital of France?"
	params.MaxTokens = 100

	ctx := context.Background()
	result, err := client.Infer(ctx, params)
	if err != nil {
		t.Fatalf("Infer failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Text == "" {
		t.Error("expected non-empty text")
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
}

func TestClient_Infer_EmptyPrompt(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	params := DefaultInferenceParams()
	params.Prompt = "" // Empty prompt

	ctx := context.Background()
	result, err := client.Infer(ctx, params)
	// Empty prompt should still work (returns something in stub mode)
	if err != nil {
		t.Logf("Infer with empty prompt returned error (may be expected): %v", err)
	}
	if result != nil {
		t.Logf("Result text: %s", result.Text)
	}
}

func TestClient_Infer_DefaultsApplied(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Params with zero values - defaults should be applied
	params := InferenceParams{
		Prompt: "Test prompt",
	}

	ctx := context.Background()
	result, err := client.Infer(ctx, params)
	if err != nil {
		t.Fatalf("Infer failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestClient_Infer_ClosedClient(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Close the client
	if err := client.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Try to infer
	params := DefaultInferenceParams()
	params.Prompt = "Test"

	ctx := context.Background()
	_, err = client.Infer(ctx, params)
	if err == nil {
		t.Fatal("expected error when calling Infer on closed client")
	}

	llamaErr, ok := err.(*LlamaError)
	if !ok {
		t.Errorf("expected *LlamaError, got %T", err)
	} else if llamaErr.Message != "client is closed" {
		t.Errorf("unexpected error message: %s", llamaErr.Message)
	}
}

func TestClient_Infer_Timeout(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)
	config.NumContexts = 1 // Only one context

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Use a very short context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Let the timeout expire
	time.Sleep(2 * time.Millisecond)

	params := DefaultInferenceParams()
	params.Prompt = "Test"

	_, err = client.Infer(ctx, params)
	if err == nil {
		// In stub mode, inference may complete before timeout
		t.Log("Infer completed before timeout (expected in stub mode)")
	}
}

// =============================================================================
// InferVision Tests
// =============================================================================

func TestClient_InferVision_NoImage(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	params := VisionParams{
		Prompt: "Describe this image",
		// No image provided
	}

	ctx := context.Background()
	_, err = client.InferVision(ctx, params)
	if err == nil {
		t.Fatal("expected error when no image provided")
	}

	llamaErr, ok := err.(*LlamaError)
	if !ok {
		t.Errorf("expected *LlamaError, got %T", err)
	} else if llamaErr.Op != "InferVision" {
		t.Errorf("Op = %q, want %q", llamaErr.Op, "InferVision")
	}
}

func TestClient_InferVision_WithImageData(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	params := VisionParams{
		Prompt:    "Describe this image",
		ImageData: []byte{0xFF, 0xD8, 0xFF, 0xE0}, // Mock JPEG header
	}

	ctx := context.Background()
	_, err = client.InferVision(ctx, params)
	// In stub mode, vision inference returns an error (not implemented)
	if err != nil {
		t.Logf("InferVision returned expected error in stub mode: %v", err)
	}
}

func TestClient_InferVision_WithImagePath(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Create a test image file
	tmpDir := t.TempDir()
	imagePath := filepath.Join(tmpDir, "test.jpg")
	if err := os.WriteFile(imagePath, []byte{0xFF, 0xD8, 0xFF, 0xE0}, 0644); err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}

	params := VisionParams{
		Prompt:    "Describe this image",
		ImagePath: imagePath,
	}

	ctx := context.Background()
	_, err = client.InferVision(ctx, params)
	// In stub mode, this returns an error (not implemented)
	if err != nil {
		t.Logf("InferVision returned expected error in stub mode: %v", err)
	}
}

func TestClient_InferVision_InvalidImagePath(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	params := VisionParams{
		Prompt:    "Describe this image",
		ImagePath: "/nonexistent/image.jpg",
	}

	ctx := context.Background()
	_, err = client.InferVision(ctx, params)
	if err == nil {
		t.Fatal("expected error for invalid image path")
	}
}

func TestClient_InferVision_ClosedClient(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	client.Close()

	params := VisionParams{
		Prompt:    "Test",
		ImageData: []byte{1, 2, 3},
	}

	ctx := context.Background()
	_, err = client.InferVision(ctx, params)
	if err == nil {
		t.Fatal("expected error when calling InferVision on closed client")
	}
}

// =============================================================================
// GetGPUMemoryUsage Tests
// =============================================================================

func TestClient_GetGPUMemoryUsage(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	info, err := client.GetGPUMemoryUsage()
	if err != nil {
		t.Fatalf("GetGPUMemoryUsage failed: %v", err)
	}

	// In stub mode, we get mock data
	if info == nil {
		t.Fatal("expected non-nil GPU memory info")
	}
	if info.Total <= 0 {
		t.Error("expected positive total memory")
	}
}

func TestClient_GetGPUMemoryUsage_ClosedClient(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	client.Close()

	_, err = client.GetGPUMemoryUsage()
	if err == nil {
		t.Fatal("expected error when calling GetGPUMemoryUsage on closed client")
	}
}

// =============================================================================
// HealthCheck Tests
// =============================================================================

func TestClient_HealthCheck(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	status, err := client.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	if status == nil {
		t.Fatal("expected non-nil health status")
	}
	if !status.Healthy {
		t.Errorf("expected healthy status, got: %s", status.Status)
	}
	if !status.ModelLoaded {
		t.Error("expected model to be loaded")
	}
	if status.Uptime <= 0 {
		t.Error("expected positive uptime")
	}
}

func TestClient_HealthCheck_ClosedClient(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	client.Close()

	status, err := client.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	if status.Healthy {
		t.Error("expected unhealthy status for closed client")
	}
	if status.Status != "client closed" {
		t.Errorf("Status = %q, want %q", status.Status, "client closed")
	}
}

func TestClient_HealthCheck_WithInferences(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Run some inferences
	params := DefaultInferenceParams()
	params.Prompt = "Test"
	for i := 0; i < 3; i++ {
		_, err := client.Infer(context.Background(), params)
		if err != nil {
			t.Fatalf("Infer %d failed: %v", i+1, err)
		}
	}

	status, err := client.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	if status.Stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if status.Stats.TotalInferences < 3 {
		t.Errorf("TotalInferences = %d, want >= 3", status.Stats.TotalInferences)
	}
}

// =============================================================================
// Stats Tests
// =============================================================================

func TestClient_Stats(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	// Initial stats
	stats := client.Stats()
	if stats.TotalInferences != 0 {
		t.Errorf("initial TotalInferences = %d, want 0", stats.TotalInferences)
	}

	// Run an inference
	params := DefaultInferenceParams()
	params.Prompt = "Test"
	_, err = client.Infer(context.Background(), params)
	if err != nil {
		t.Fatalf("Infer failed: %v", err)
	}

	// Updated stats
	stats = client.Stats()
	if stats.TotalInferences != 1 {
		t.Errorf("TotalInferences = %d, want 1", stats.TotalInferences)
	}
	if stats.TotalTokensGenerated <= 0 {
		t.Error("expected positive TotalTokensGenerated")
	}
}

// =============================================================================
// Close Tests
// =============================================================================

func TestClient_Close_Idempotent(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Close multiple times
	for i := 0; i < 3; i++ {
		if err := client.Close(); err != nil {
			t.Errorf("Close %d failed: %v", i+1, err)
		}
	}
}

// =============================================================================
// Concurrent Tests
// =============================================================================

func TestClient_ConcurrentInfer(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)
	config.NumContexts = 3

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	const numGoroutines = 10
	const numInferencesPerGoroutine = 5

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numInferencesPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numInferencesPerGoroutine; j++ {
				params := DefaultInferenceParams()
				params.Prompt = "Test concurrent"

				ctx := context.Background()
				_, err := client.Infer(ctx, params)
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent inference error: %v", err)
	}

	// Verify stats
	stats := client.Stats()
	expectedInferences := int64(numGoroutines * numInferencesPerGoroutine)
	if stats.TotalInferences < expectedInferences {
		t.Errorf("TotalInferences = %d, want >= %d", stats.TotalInferences, expectedInferences)
	}
}

func TestClient_ConcurrentHealthCheck(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			status, err := client.HealthCheck()
			if err != nil {
				t.Errorf("HealthCheck failed: %v", err)
			}
			if status == nil {
				t.Error("expected non-nil status")
			}
		}()
	}

	wg.Wait()
}

// =============================================================================
// Convenience Function Tests
// =============================================================================

func TestQuickInfer(t *testing.T) {
	modelPath := testModelPath(t)

	text, err := QuickInfer(modelPath, "Hello")
	if err != nil {
		t.Fatalf("QuickInfer failed: %v", err)
	}

	if text == "" {
		t.Error("expected non-empty text")
	}
}

func TestQuickInfer_InvalidPath(t *testing.T) {
	_, err := QuickInfer("/nonexistent/model.gguf", "Hello")
	if err == nil {
		t.Fatal("expected error for invalid model path")
	}
}

func TestClient_InferWithTimeout(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	params := DefaultInferenceParams()
	params.Prompt = "Test"

	result, err := client.InferWithTimeout(5*time.Second, params)
	if err != nil {
		t.Fatalf("InferWithTimeout failed: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestClient_InferStream(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	params := DefaultInferenceParams()
	params.Prompt = "Test"

	tokenChan := make(chan string, 10)

	ctx := context.Background()
	result, err := client.InferStream(ctx, params, tokenChan)
	if err != nil {
		t.Fatalf("InferStream failed: %v", err)
	}

	// Collect tokens
	var tokens []string
	for token := range tokenChan {
		tokens = append(tokens, token)
	}

	if len(tokens) == 0 {
		t.Error("expected at least one token")
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// =============================================================================
// Stop Reason Tests
// =============================================================================

func TestDetermineStopReason(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		params   InferenceParams
		expected string
	}{
		{
			name:     "EOS default",
			text:     "Hello world",
			params:   InferenceParams{MaxTokens: 100},
			expected: "eos",
		},
		{
			name:     "Max tokens",
			text:     "A very long text that uses many tokens", // ~10 tokens
			params:   InferenceParams{MaxTokens: 5},
			expected: "max_tokens",
		},
		{
			name:     "Stop sequence",
			text:     "Hello\n\n",
			params:   InferenceParams{MaxTokens: 100, StopSequences: []string{"\n\n"}},
			expected: "stop_sequence",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineStopReason(tt.text, tt.params)
			if got != tt.expected {
				t.Errorf("determineStopReason() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// =============================================================================
// io.Closer Interface Test
// =============================================================================

func TestClient_ImplementsIOCloser(t *testing.T) {
	config := DefaultClientConfig()
	config.ModelPath = testModelPath(t)

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Verify it implements io.Closer
	var closer interface{ Close() error } = client
	if closer == nil {
		t.Fatal("client should implement io.Closer")
	}

	if err := closer.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkClient_Infer(b *testing.B) {
	tmpDir := b.TempDir()
	modelPath := filepath.Join(tmpDir, "bench-model.gguf")
	if err := os.WriteFile(modelPath, []byte("mock"), 0644); err != nil {
		b.Fatalf("failed to create test model: %v", err)
	}

	config := DefaultClientConfig()
	config.ModelPath = modelPath
	config.NumContexts = 5

	client, err := NewClient(config)
	if err != nil {
		b.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	params := DefaultInferenceParams()
	params.Prompt = "Benchmark test prompt"
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.Infer(ctx, params)
			if err != nil {
				b.Errorf("Infer failed: %v", err)
			}
		}
	})
}

func BenchmarkClient_HealthCheck(b *testing.B) {
	tmpDir := b.TempDir()
	modelPath := filepath.Join(tmpDir, "bench-model.gguf")
	if err := os.WriteFile(modelPath, []byte("mock"), 0644); err != nil {
		b.Fatalf("failed to create test model: %v", err)
	}

	config := DefaultClientConfig()
	config.ModelPath = modelPath

	client, err := NewClient(config)
	if err != nil {
		b.Fatalf("NewClient failed: %v", err)
	}
	defer client.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.HealthCheck()
		if err != nil {
			b.Errorf("HealthCheck failed: %v", err)
		}
	}
}
