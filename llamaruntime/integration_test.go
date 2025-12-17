// Package llamaruntime provides Go bindings to llama.cpp for local LLM inference.
// This file contains integration tests that require a real GPU and model file.
//
// These tests are only run when CGo is enabled and a model file is available.
// To run these tests:
//
//	LLAMA_TEST_MODEL=/path/to/model.gguf go test -v ./llamaruntime/... -run Integration
//
// Or set LLAMA_MODEL_PATH in your .env file.
//
// Build Tags:
// - cgo: Only run when CGo is enabled (real llama.cpp bindings)
// - !nocgo: Exclude when using stub implementations
//
//go:build cgo && !nocgo

package llamaruntime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// =============================================================================
// Test Setup and Helpers
// =============================================================================

// getTestModelPath returns the path to a test model, or skips the test if unavailable.
// Checks (in order):
// 1. LLAMA_TEST_MODEL environment variable
// 2. LLAMA_MODEL_PATH environment variable
// 3. models/*.gguf in the project root
func getTestModelPath(t *testing.T) string {
	t.Helper()

	// Check LLAMA_TEST_MODEL first (explicit test model)
	if path := os.Getenv("LLAMA_TEST_MODEL"); path != "" {
		if err := ValidateModelPath(path); err == nil {
			t.Logf("Using test model: %s", path)
			return path
		}
		t.Logf("LLAMA_TEST_MODEL set but invalid: %s", path)
	}

	// Check LLAMA_MODEL_PATH (production model)
	if path := os.Getenv("LLAMA_MODEL_PATH"); path != "" {
		if err := ValidateModelPath(path); err == nil {
			t.Logf("Using model from LLAMA_MODEL_PATH: %s", path)
			return path
		}
		t.Logf("LLAMA_MODEL_PATH set but invalid: %s", path)
	}

	// Search for models in project root
	projectRoot := findProjectRoot()
	if projectRoot != "" {
		modelsDir := filepath.Join(projectRoot, "models")
		matches, err := filepath.Glob(filepath.Join(modelsDir, "*.gguf"))
		if err == nil && len(matches) > 0 {
			// Use the first valid model found
			for _, match := range matches {
				if err := ValidateModelPath(match); err == nil {
					t.Logf("Found model in models/: %s", match)
					return match
				}
			}
		}
	}

	t.Skip("No model file available for integration tests. Set LLAMA_TEST_MODEL or LLAMA_MODEL_PATH")
	return ""
}

// findProjectRoot finds the project root directory by looking for go.mod.
func findProjectRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Walk up to find go.mod
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return cwd
}

// getTestImagePath returns the path to a test image, or skips the test if unavailable.
func getTestImagePath(t *testing.T) string {
	t.Helper()

	// Check LLAMA_TEST_IMAGE environment variable
	if path := os.Getenv("LLAMA_TEST_IMAGE"); path != "" {
		if _, err := os.Stat(path); err == nil {
			t.Logf("Using test image: %s", path)
			return path
		}
	}

	// Search for test images in test_files directory
	projectRoot := findProjectRoot()
	if projectRoot != "" {
		testFilesDir := filepath.Join(projectRoot, "test_files")
		extensions := []string{"*.jpg", "*.jpeg", "*.png"}
		for _, ext := range extensions {
			matches, err := filepath.Glob(filepath.Join(testFilesDir, ext))
			if err == nil && len(matches) > 0 {
				t.Logf("Found test image: %s", matches[0])
				return matches[0]
			}
		}
	}

	t.Skip("No test image available for vision tests. Set LLAMA_TEST_IMAGE or add images to test_files/")
	return ""
}

// createIntegrationClient creates a Client for integration testing.
func createIntegrationClient(t *testing.T) *Client {
	t.Helper()

	modelPath := getTestModelPath(t)

	config := DefaultClientConfig()
	config.ModelPath = modelPath
	config.NumContexts = 2 // Enough for concurrent tests
	config.VerboseLogging = testing.Verbose()

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	return client
}

// =============================================================================
// Text Inference Integration Tests
// =============================================================================

func TestIntegration_TextInference_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := createIntegrationClient(t)
	defer client.Close()

	params := DefaultInferenceParams()
	params.Prompt = "What is 2 + 2? Answer with just the number."
	params.MaxTokens = 50
	params.Temperature = 0.1 // Low temperature for deterministic output

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := client.Infer(ctx, params)
	if err != nil {
		t.Fatalf("Infer failed: %v", err)
	}

	t.Logf("Prompt: %s", params.Prompt)
	t.Logf("Response: %s", result.Text)
	t.Logf("Tokens generated: %d", result.TokensGenerated)
	t.Logf("Duration: %v", result.Duration)
	t.Logf("Tokens/sec: %.2f", result.TokensPerSecond)

	// Validate result structure
	if result.Text == "" {
		t.Error("Expected non-empty response")
	}
	if result.Duration <= 0 {
		t.Error("Expected positive duration")
	}
	if result.TokensGenerated <= 0 {
		t.Error("Expected positive token count")
	}
}

func TestIntegration_TextInference_LongPrompt(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := createIntegrationClient(t)
	defer client.Close()

	// Create a longer prompt to test context handling
	longPrompt := "Please summarize the following text:\n\n"
	longPrompt += "Artificial intelligence (AI) is intelligence demonstrated by machines, "
	longPrompt += "as opposed to natural intelligence displayed by animals including humans. "
	longPrompt += "AI research has been defined as the field of study of intelligent agents, "
	longPrompt += "which refers to any system that perceives its environment and takes actions "
	longPrompt += "that maximize its chance of achieving its goals. The term 'artificial intelligence' "
	longPrompt += "had previously been used to describe machines that mimic and display human cognitive "
	longPrompt += "skills that are associated with the human mind, such as learning and problem-solving.\n\n"
	longPrompt += "Provide a brief 2-3 sentence summary."

	params := DefaultInferenceParams()
	params.Prompt = longPrompt
	params.MaxTokens = 200
	params.Temperature = 0.3

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	result, err := client.Infer(ctx, params)
	if err != nil {
		t.Fatalf("Infer failed: %v", err)
	}

	t.Logf("Response: %s", result.Text)
	t.Logf("Tokens prompt: %d, generated: %d", result.TokensPrompt, result.TokensGenerated)
	t.Logf("Duration: %v, Tokens/sec: %.2f", result.Duration, result.TokensPerSecond)

	if result.Text == "" {
		t.Error("Expected non-empty response")
	}
}

func TestIntegration_TextInference_StopSequence(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := createIntegrationClient(t)
	defer client.Close()

	params := DefaultInferenceParams()
	params.Prompt = "Count from 1 to 10: 1, 2, 3,"
	params.MaxTokens = 100
	params.StopSequences = []string{"7", "8"}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := client.Infer(ctx, params)
	if err != nil {
		t.Fatalf("Infer failed: %v", err)
	}

	t.Logf("Response: %s", result.Text)
	t.Logf("Stop reason: %s", result.StopReason)

	// The response should stop before reaching 10
	if result.Text == "" {
		t.Error("Expected non-empty response")
	}
}

func TestIntegration_TextInference_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := createIntegrationClient(t)
	defer client.Close()

	params := DefaultInferenceParams()
	params.Prompt = "Write a very long essay about the history of computing."
	params.MaxTokens = 10000                // Very long
	params.Timeout = 100 * time.Millisecond // Very short timeout

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Infer(ctx, params)
	// We expect this to either timeout or complete (depending on model speed)
	// The important thing is that it doesn't hang
	if err != nil {
		t.Logf("Inference returned error (expected for timeout): %v", err)
	} else {
		t.Log("Inference completed before timeout (fast model)")
	}
}

// =============================================================================
// Vision Inference Integration Tests
// =============================================================================

func TestIntegration_VisionInference_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := createIntegrationClient(t)
	defer client.Close()

	imagePath := getTestImagePath(t)

	params := DefaultVisionParams()
	params.Prompt = "Describe this image in detail. What do you see?"
	params.ImagePath = imagePath
	params.MaxTokens = 200

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	result, err := client.InferVision(ctx, params)
	if err != nil {
		// Vision inference may not be supported by all models
		if IsModelCapabilityError(err) {
			t.Skipf("Model does not support vision inference: %v", err)
		}
		t.Fatalf("InferVision failed: %v", err)
	}

	t.Logf("Image: %s", imagePath)
	t.Logf("Prompt: %s", params.Prompt)
	t.Logf("Response: %s", result.Text)
	t.Logf("Duration: %v", result.Duration)

	if result.Text == "" {
		t.Error("Expected non-empty response")
	}
}

func TestIntegration_VisionInference_WithImageData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := createIntegrationClient(t)
	defer client.Close()

	imagePath := getTestImagePath(t)

	// Read image data into memory
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("Failed to read image file: %v", err)
	}

	params := DefaultVisionParams()
	params.Prompt = "What colors are prominent in this image?"
	params.ImageData = imageData // Use raw bytes instead of path
	params.MaxTokens = 100

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	result, err := client.InferVision(ctx, params)
	if err != nil {
		if IsModelCapabilityError(err) {
			t.Skipf("Model does not support vision inference: %v", err)
		}
		t.Fatalf("InferVision failed: %v", err)
	}

	t.Logf("Response: %s", result.Text)

	if result.Text == "" {
		t.Error("Expected non-empty response")
	}
}

// =============================================================================
// Concurrent Request Integration Tests
// =============================================================================

func TestIntegration_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	modelPath := getTestModelPath(t)

	// Create client with multiple contexts for concurrent requests
	config := DefaultClientConfig()
	config.ModelPath = modelPath
	config.NumContexts = 3 // Allow 3 concurrent requests

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Run concurrent inference requests
	numRequests := 5
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	prompts := []string{
		"What is the capital of France?",
		"What is 10 * 5?",
		"Name a color.",
		"What day comes after Monday?",
		"Is water wet? Answer yes or no.",
	}

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			params := DefaultInferenceParams()
			params.Prompt = prompts[idx%len(prompts)]
			params.MaxTokens = 50
			params.Temperature = 0.1

			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			result, err := client.Infer(ctx, params)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				t.Logf("Request %d failed: %v", idx, err)
				return
			}

			atomic.AddInt64(&successCount, 1)
			t.Logf("Request %d: %q -> %q (%.2f tok/s)",
				idx, params.Prompt, truncateForLog(result.Text, 50), result.TokensPerSecond)
		}(i)
	}

	wg.Wait()

	t.Logf("Concurrent results: %d success, %d errors", successCount, errorCount)

	if successCount == 0 {
		t.Error("All concurrent requests failed")
	}

	// Verify client statistics
	stats := client.Stats()
	t.Logf("Client stats: %d total inferences, %d errors", stats.TotalInferences, stats.ErrorCount)
}

func TestIntegration_ConcurrentRequests_HighLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	modelPath := getTestModelPath(t)

	// Create client with limited contexts to test queueing
	config := DefaultClientConfig()
	config.ModelPath = modelPath
	config.NumContexts = 2                   // Only 2 contexts for 10 requests
	config.AcquireTimeout = 60 * time.Second // Allow waiting for contexts

	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Run more requests than contexts
	numRequests := 10
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64
	var timeoutCount int64

	startTime := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			params := DefaultInferenceParams()
			params.Prompt = fmt.Sprintf("Count to %d.", (idx%5)+1)
			params.MaxTokens = 30
			params.Temperature = 0.1

			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			result, err := client.Infer(ctx, params)
			if err != nil {
				if IsTimeoutError(err) {
					atomic.AddInt64(&timeoutCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
				t.Logf("Request %d failed: %v", idx, err)
				return
			}

			atomic.AddInt64(&successCount, 1)
			t.Logf("Request %d completed in %v", idx, result.Duration)
		}(i)
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	t.Logf("High-load test completed in %v", totalDuration)
	t.Logf("Results: %d success, %d errors, %d timeouts", successCount, errorCount, timeoutCount)

	// At least some requests should succeed
	if successCount < int64(numRequests/2) {
		t.Errorf("Too many failures: only %d/%d succeeded", successCount, numRequests)
	}
}

// =============================================================================
// Health Check Integration Tests
// =============================================================================

func TestIntegration_HealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := createIntegrationClient(t)
	defer client.Close()

	// Perform a few inferences first to generate stats
	params := DefaultInferenceParams()
	params.Prompt = "Hello"
	params.MaxTokens = 10

	for i := 0; i < 3; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		_, _ = client.Infer(ctx, params)
		cancel()
	}

	// Now check health
	health, err := client.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	t.Logf("Health status: %s (healthy: %v)", health.Status, health.Healthy)
	t.Logf("Uptime: %v", health.Uptime)
	t.Logf("Model loaded: %v", health.ModelLoaded)

	if health.ModelInfo != nil {
		t.Logf("Model: %s (size: %d bytes)", health.ModelInfo.Name, health.ModelInfo.Size)
	}

	if health.Stats != nil {
		t.Logf("Stats: %d inferences, %.2f avg tok/s",
			health.Stats.TotalInferences, health.Stats.AverageTokensPerSecond)
	}

	if health.GPUStatus != nil {
		t.Logf("GPU: available=%v, free=%d bytes", health.GPUStatus.Available, health.GPUStatus.FreeMemory)
	}

	if !health.Healthy {
		t.Errorf("Expected healthy client, got status: %s", health.Status)
	}
}

func TestIntegration_GPUMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	client := createIntegrationClient(t)
	defer client.Close()

	gpuMem, err := client.GetGPUMemoryUsage()
	if err != nil {
		// GPU memory may not be available on all systems
		t.Logf("GPU memory query failed (may be expected on CPU-only systems): %v", err)
		return
	}

	if gpuMem == nil {
		t.Log("No GPU memory info available (CPU-only mode)")
		return
	}

	t.Logf("GPU Memory: used=%d MB, free=%d MB, total=%d MB (%.1f%% used)",
		gpuMem.Used/(1024*1024),
		gpuMem.Free/(1024*1024),
		gpuMem.Total/(1024*1024),
		gpuMem.UsedPct)

	// Sanity checks
	if gpuMem.Total > 0 && gpuMem.Used+gpuMem.Free > gpuMem.Total*2 {
		t.Errorf("Invalid GPU memory values: used=%d + free=%d > total=%d",
			gpuMem.Used, gpuMem.Free, gpuMem.Total)
	}
}

// =============================================================================
// Model Loader Integration Tests
// =============================================================================

func TestIntegration_ModelLoader(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	modelPath := getTestModelPath(t)

	config := DefaultModelLoaderConfig()
	config.ModelPath = modelPath
	config.RunStartupTest = true
	config.StartupTestPrompt = "Say hello."
	config.StartupTestTimeout = 60 * time.Second

	loader := NewModelLoader(config)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	client, err := loader.Load(ctx)
	if err != nil {
		t.Fatalf("ModelLoader.Load failed: %v", err)
	}
	defer client.Close()

	// Check metadata
	metadata := loader.Metadata()
	if metadata == nil {
		t.Fatal("Expected non-nil metadata")
	}

	t.Logf("Loaded model: %s", metadata.Name)
	t.Logf("Size: %s", metadata.SizeHuman)
	t.Logf("Startup test passed: %v (duration: %v)", metadata.StartupTestPassed, metadata.StartupTestDuration)

	if !metadata.StartupTestPassed {
		t.Error("Expected startup test to pass")
	}
}

// =============================================================================
// Error Handling Helpers
// =============================================================================

// IsTimeoutError checks if an error is a timeout error.
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if llamaErr, ok := err.(*LlamaError); ok {
		return llamaErr.Err == ErrTimeout
	}
	return false
}

// IsModelCapabilityError checks if an error is due to model capability limitations.
func IsModelCapabilityError(err error) bool {
	if err == nil {
		return false
	}
	if llamaErr, ok := err.(*LlamaError); ok {
		// Check for common vision-not-supported errors
		return llamaErr.Message == "vision inference not implemented in stub mode" ||
			llamaErr.Message == "model does not support vision inference"
	}
	return false
}
