// Package tests contains end-to-end acceptance tests for the llamaruntime integration.
//
// These tests verify the complete local LLM workflow:
// - Note processing with local inference
// - Image analysis with local vision model
// - Canvas analysis with local inference
// - Concurrent request handling
// - Zero external API calls verification
// - Graceful shutdown behavior
//
// To run these tests with a real model:
//
//	LLAMA_TEST_MODEL=/path/to/model.gguf go test -v ./tests/... -run E2E_LlamaRuntime
//
// Build Tags:
// - cgo: Only run when CGo is enabled (real llama.cpp bindings)
// - !nocgo: Exclude when using stub implementations
//
//go:build cgo && !nocgo

package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go_backend/core"
	"go_backend/llamaruntime"
)

// =============================================================================
// Test Setup and Helpers
// =============================================================================

// getE2EModelPath returns a model path for E2E testing.
func getE2EModelPath(t *testing.T) string {
	t.Helper()

	// Check LLAMA_TEST_MODEL first
	if path := os.Getenv("LLAMA_TEST_MODEL"); path != "" {
		if err := llamaruntime.ValidateModelPath(path); err == nil {
			t.Logf("Using test model: %s", path)
			return path
		}
	}

	// Check LLAMA_MODEL_PATH
	if path := os.Getenv("LLAMA_MODEL_PATH"); path != "" {
		if err := llamaruntime.ValidateModelPath(path); err == nil {
			return path
		}
	}

	// Search for models
	cwd, _ := os.Getwd()
	for _, searchPath := range []string{cwd, filepath.Dir(cwd)} {
		modelsDir := filepath.Join(searchPath, "models")
		matches, err := filepath.Glob(filepath.Join(modelsDir, "*.gguf"))
		if err == nil && len(matches) > 0 {
			for _, match := range matches {
				if err := llamaruntime.ValidateModelPath(match); err == nil {
					return match
				}
			}
		}
	}

	t.Skip("No model file available for E2E tests. Set LLAMA_TEST_MODEL")
	return ""
}

// getE2ETestImage returns a test image path.
func getE2ETestImage(t *testing.T) string {
	t.Helper()

	if path := os.Getenv("LLAMA_TEST_IMAGE"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	cwd, _ := os.Getwd()
	for _, searchPath := range []string{cwd, filepath.Dir(cwd)} {
		testFilesDir := filepath.Join(searchPath, "test_files")
		for _, ext := range []string{"*.jpg", "*.jpeg", "*.png"} {
			matches, err := filepath.Glob(filepath.Join(testFilesDir, ext))
			if err == nil && len(matches) > 0 {
				return matches[0]
			}
		}
	}

	t.Skip("No test image available. Set LLAMA_TEST_IMAGE or add images to test_files/")
	return ""
}

// createE2EClient creates a llamaruntime client for E2E testing.
func createE2EClient(t *testing.T) *llamaruntime.Client {
	t.Helper()

	modelPath := getE2EModelPath(t)

	config := llamaruntime.DefaultClientConfig()
	config.ModelPath = modelPath
	config.NumContexts = 3
	config.VerboseLogging = testing.Verbose()

	client, err := llamaruntime.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create llamaruntime client: %v", err)
	}

	return client
}

// =============================================================================
// Note Processing E2E Tests
// =============================================================================

// TestE2E_LlamaRuntime_NoteProcessing tests the complete note processing flow.
func TestE2E_LlamaRuntime_NoteProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	client := createE2EClient(t)
	defer client.Close()

	testCases := []struct {
		name      string
		prompt    string
		maxTokens int
		validate  func(t *testing.T, result *llamaruntime.InferenceResult)
	}{
		{
			name:      "simple_question",
			prompt:    "What is 2+2? Answer with just the number.",
			maxTokens: 20,
			validate: func(t *testing.T, result *llamaruntime.InferenceResult) {
				if result.Text == "" {
					t.Error("Expected non-empty response")
				}
				t.Logf("Response: %s", result.Text)
			},
		},
		{
			name:      "summarization_task",
			prompt:    "Summarize this in one sentence: AI is transforming how we work.",
			maxTokens: 100,
			validate: func(t *testing.T, result *llamaruntime.InferenceResult) {
				if result.Text == "" {
					t.Error("Expected non-empty summary")
				}
				if result.TokensGenerated == 0 {
					t.Error("Expected tokens to be generated")
				}
			},
		},
		{
			name:      "creative_task",
			prompt:    "Write a haiku about coding.",
			maxTokens: 50,
			validate: func(t *testing.T, result *llamaruntime.InferenceResult) {
				if result.Text == "" {
					t.Error("Expected haiku response")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			params := llamaruntime.DefaultInferenceParams()
			params.Prompt = tc.prompt
			params.MaxTokens = tc.maxTokens
			params.Temperature = 0.3

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			result, err := client.Infer(ctx, params)
			if err != nil {
				t.Fatalf("Inference failed: %v", err)
			}

			tc.validate(t, result)
			t.Logf("Duration: %v, Tokens/sec: %.2f", result.Duration, result.TokensPerSecond)
		})
	}
}

// =============================================================================
// Image Analysis E2E Tests
// =============================================================================

// TestE2E_LlamaRuntime_ImageAnalysis tests the complete image analysis flow.
func TestE2E_LlamaRuntime_ImageAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	client := createE2EClient(t)
	defer client.Close()

	imagePath := getE2ETestImage(t)

	params := llamaruntime.DefaultVisionParams()
	params.ImagePath = imagePath
	params.Prompt = "Describe what you see in this image. Be specific about colors, objects, and composition."
	params.MaxTokens = 300

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	result, err := client.InferVision(ctx, params)
	if err != nil {
		// Vision may not be supported by all models
		if strings.Contains(err.Error(), "not implemented") ||
			strings.Contains(err.Error(), "not support") {
			t.Skipf("Model does not support vision inference: %v", err)
		}
		t.Fatalf("Vision inference failed: %v", err)
	}

	t.Logf("Image: %s", imagePath)
	t.Logf("Response: %s", result.Text)
	t.Logf("Duration: %v, Tokens: %d", result.Duration, result.TokensGenerated)

	if result.Text == "" {
		t.Error("Expected non-empty image description")
	}
}

// TestE2E_LlamaRuntime_ImageAnalysis_FromBytes tests vision with raw image data.
func TestE2E_LlamaRuntime_ImageAnalysis_FromBytes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	client := createE2EClient(t)
	defer client.Close()

	imagePath := getE2ETestImage(t)

	// Read image into memory
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		t.Fatalf("Failed to read image: %v", err)
	}

	params := llamaruntime.DefaultVisionParams()
	params.ImageData = imageData
	params.Prompt = "What colors are visible in this image?"
	params.MaxTokens = 100

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	result, err := client.InferVision(ctx, params)
	if err != nil {
		if strings.Contains(err.Error(), "not implemented") ||
			strings.Contains(err.Error(), "not support") {
			t.Skipf("Model does not support vision inference: %v", err)
		}
		t.Fatalf("Vision inference failed: %v", err)
	}

	if result.Text == "" {
		t.Error("Expected non-empty response")
	}
	t.Logf("Color analysis: %s", result.Text)
}

// =============================================================================
// Canvas Analysis E2E Tests
// =============================================================================

// TestE2E_LlamaRuntime_CanvasAnalysis tests summarizing multiple widgets.
func TestE2E_LlamaRuntime_CanvasAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	client := createE2EClient(t)
	defer client.Close()

	// Simulate canvas content from multiple widgets
	canvasContent := `
Canvas Overview:
- Note 1: "Project kickoff meeting scheduled for Monday at 10am"
- Note 2: "Action items: Review designs, Update requirements, Setup dev environment"
- Note 3: "Sprint goals: Complete user auth, Fix performance issues, Deploy to staging"
- Image 1: Architecture diagram showing microservices layout
- Image 2: UI mockup for dashboard

Please provide a brief summary of the canvas content and identify key themes.
`

	params := llamaruntime.DefaultInferenceParams()
	params.Prompt = canvasContent
	params.MaxTokens = 300
	params.Temperature = 0.3

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	result, err := client.Infer(ctx, params)
	if err != nil {
		t.Fatalf("Canvas analysis failed: %v", err)
	}

	t.Logf("Canvas Summary: %s", result.Text)
	t.Logf("Duration: %v, Tokens: %d", result.Duration, result.TokensGenerated)

	if result.Text == "" {
		t.Error("Expected non-empty canvas summary")
	}
}

// =============================================================================
// Concurrency E2E Tests
// =============================================================================

// TestE2E_LlamaRuntime_Concurrency tests handling multiple simultaneous requests.
func TestE2E_LlamaRuntime_Concurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	modelPath := getE2EModelPath(t)

	// Create client with multiple contexts
	config := llamaruntime.DefaultClientConfig()
	config.ModelPath = modelPath
	config.NumContexts = 4

	client, err := llamaruntime.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Run concurrent requests
	numRequests := 10
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64
	results := make(chan string, numRequests)

	prompts := []string{
		"What is the capital of France?",
		"Name a primary color.",
		"What is 5 * 3?",
		"Is the sky blue? Yes or no.",
		"What comes after Tuesday?",
	}

	startTime := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			params := llamaruntime.DefaultInferenceParams()
			params.Prompt = prompts[idx%len(prompts)]
			params.MaxTokens = 30
			params.Temperature = 0.1

			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			result, err := client.Infer(ctx, params)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				results <- fmt.Sprintf("Request %d FAILED: %v", idx, err)
				return
			}

			atomic.AddInt64(&successCount, 1)
			results <- fmt.Sprintf("Request %d: %q -> %q",
				idx, params.Prompt, truncateString(result.Text, 40))
		}(i)
	}

	wg.Wait()
	close(results)
	totalDuration := time.Since(startTime)

	// Log all results
	for res := range results {
		t.Log(res)
	}

	t.Logf("Completed %d requests in %v", numRequests, totalDuration)
	t.Logf("Success: %d, Errors: %d", successCount, errorCount)

	// Verify statistics
	stats := client.Stats()
	t.Logf("Client stats: %d inferences, %.2f avg tok/s",
		stats.TotalInferences, stats.AverageTokensPerSecond)

	if successCount == 0 {
		t.Error("All requests failed - concurrency handling broken")
	}
	if float64(errorCount)/float64(numRequests) > 0.2 {
		t.Errorf("Too many errors: %d/%d (>20%%)", errorCount, numRequests)
	}
}

// =============================================================================
// Zero External Calls Verification Tests
// =============================================================================

// TestE2E_LlamaRuntime_ZeroExternalCalls verifies no external API calls are made.
func TestE2E_LlamaRuntime_ZeroExternalCalls(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create a mock server that will fail if any request reaches it
	externalCallsMade := atomic.Int64{}
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		externalCallsMade.Add(1)
		t.Errorf("UNEXPECTED EXTERNAL CALL: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer mockServer.Close()

	// Set environment to point to mock server (should not be used)
	originalOpenAIURL := os.Getenv("OPENAI_API_BASE")
	os.Setenv("OPENAI_API_BASE", mockServer.URL)
	defer os.Setenv("OPENAI_API_BASE", originalOpenAIURL)

	// Create llamaruntime client
	client := createE2EClient(t)
	defer client.Close()

	// Run multiple inference operations
	params := llamaruntime.DefaultInferenceParams()
	params.Prompt = "Hello, world!"
	params.MaxTokens = 20

	ctx := context.Background()

	for i := 0; i < 5; i++ {
		_, err := client.Infer(ctx, params)
		if err != nil {
			t.Fatalf("Inference %d failed: %v", i, err)
		}
	}

	// Verify no external calls were made
	if calls := externalCallsMade.Load(); calls > 0 {
		t.Errorf("Expected zero external API calls, got %d", calls)
	} else {
		t.Log("SUCCESS: Zero external API calls made - all inference was local")
	}
}

// =============================================================================
// Graceful Shutdown E2E Tests
// =============================================================================

// TestE2E_LlamaRuntime_GracefulShutdown tests shutdown during active inference.
func TestE2E_LlamaRuntime_GracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	client := createE2EClient(t)

	// Start a long-running inference in a goroutine
	var inferenceCompleted atomic.Bool
	var inferenceError atomic.Value

	go func() {
		params := llamaruntime.DefaultInferenceParams()
		params.Prompt = "Write a detailed story about a space adventure."
		params.MaxTokens = 500 // Long output

		ctx := context.Background()
		_, err := client.Infer(ctx, params)
		if err != nil {
			inferenceError.Store(err)
		}
		inferenceCompleted.Store(true)
	}()

	// Wait a bit for inference to start
	time.Sleep(500 * time.Millisecond)

	// Close the client while inference is running
	shutdownStart := time.Now()
	err := client.Close()
	shutdownDuration := time.Since(shutdownStart)

	t.Logf("Shutdown completed in %v", shutdownDuration)

	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	// Shutdown should complete within reasonable time
	if shutdownDuration > 30*time.Second {
		t.Errorf("Shutdown took too long: %v (expected <30s)", shutdownDuration)
	}

	// Wait for goroutine to finish
	time.Sleep(100 * time.Millisecond)

	// The inference may have completed or been cancelled - either is acceptable
	if inferenceCompleted.Load() {
		if storedErr := inferenceError.Load(); storedErr != nil {
			t.Logf("Inference was cancelled during shutdown (expected): %v", storedErr)
		} else {
			t.Log("Inference completed before shutdown")
		}
	} else {
		t.Log("Inference goroutine still running (will be cleaned up)")
	}
}

// TestE2E_LlamaRuntime_MultipleClose tests that Close is idempotent.
func TestE2E_LlamaRuntime_MultipleClose(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	client := createE2EClient(t)

	// Run one inference first
	params := llamaruntime.DefaultInferenceParams()
	params.Prompt = "Hi"
	params.MaxTokens = 5

	ctx := context.Background()
	_, _ = client.Infer(ctx, params)

	// Close multiple times - should not panic or error
	for i := 0; i < 5; i++ {
		err := client.Close()
		if err != nil {
			t.Errorf("Close() call %d returned error: %v", i+1, err)
		}
	}

	t.Log("Multiple Close() calls completed without error")

	// Inference after close should fail gracefully
	_, err := client.Infer(ctx, params)
	if err == nil {
		t.Error("Expected error when calling Infer after Close")
	} else {
		t.Logf("Infer after Close correctly returned error: %v", err)
	}
}

// =============================================================================
// Health Check E2E Tests
// =============================================================================

// TestE2E_LlamaRuntime_HealthCheck tests the health monitoring system.
func TestE2E_LlamaRuntime_HealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	client := createE2EClient(t)
	defer client.Close()

	// Run several inferences to generate statistics
	params := llamaruntime.DefaultInferenceParams()
	params.Prompt = "Test"
	params.MaxTokens = 10

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_, _ = client.Infer(ctx, params)
	}

	// Check health
	health, err := client.HealthCheck()
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	t.Logf("Health Status:")
	t.Logf("  Healthy: %v", health.Healthy)
	t.Logf("  Status: %s", health.Status)
	t.Logf("  Uptime: %v", health.Uptime)

	if health.ModelInfo != nil {
		t.Logf("  Model: %s", health.ModelInfo.Name)
		t.Logf("  Model Size: %d bytes", health.ModelInfo.Size)
	}

	if health.Stats != nil {
		t.Logf("  Total Inferences: %d", health.Stats.TotalInferences)
		t.Logf("  Avg Tokens/sec: %.2f", health.Stats.AverageTokensPerSecond)
		t.Logf("  Error Count: %d", health.Stats.ErrorCount)
	}

	if health.GPUStatus != nil {
		t.Logf("  GPU Available: %v", health.GPUStatus.Available)
		if health.GPUStatus.Available {
			t.Logf("  GPU Memory: %d MB used / %d MB total",
				health.GPUStatus.FreeMemory/(1024*1024),
				health.GPUStatus.TotalMemory/(1024*1024))
		}
	}

	// Validate health
	if !health.Healthy {
		t.Errorf("Expected healthy client, got status: %s", health.Status)
	}
	if !health.ModelLoaded {
		t.Error("Expected model to be loaded")
	}
	if health.Stats == nil || health.Stats.TotalInferences < 5 {
		t.Error("Expected at least 5 inferences in stats")
	}
}

// =============================================================================
// Recovery E2E Tests
// =============================================================================

// TestE2E_LlamaRuntime_Recovery tests the recovery manager functionality.
func TestE2E_LlamaRuntime_Recovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	modelPath := getE2EModelPath(t)

	// Create client with recovery manager
	config := llamaruntime.DefaultClientConfig()
	config.ModelPath = modelPath
	config.NumContexts = 2

	client, err := llamaruntime.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Wrap with recovery manager
	recoveryConfig := llamaruntime.DefaultRecoveryConfig()
	recoveryConfig.MaxRetries = 3

	recoveryManager := llamaruntime.NewRecoveryManager(client, recoveryConfig)
	defer recoveryManager.Close()

	// Run inferences through recovery manager
	params := llamaruntime.DefaultInferenceParams()
	params.Prompt = "What is 1+1?"
	params.MaxTokens = 20

	ctx := context.Background()
	result, err := recoveryManager.Infer(ctx, params)
	if err != nil {
		t.Fatalf("Recovery manager Infer failed: %v", err)
	}

	t.Logf("Response: %s", result.Text)

	// Check recovery stats
	stats := recoveryManager.Stats()
	t.Logf("Recovery Stats:")
	t.Logf("  Total Requests: %d", stats.TotalRequests)
	t.Logf("  Successful: %d", stats.SuccessfulRequests)
	t.Logf("  Retries: %d", stats.RetryAttempts)
	t.Logf("  Context Resets: %d", stats.ContextResets)
	t.Logf("  Model Reloads: %d", stats.ModelReloads)
}

// =============================================================================
// Model Loader E2E Tests
// =============================================================================

// TestE2E_LlamaRuntime_ModelLoader tests the model loading process.
func TestE2E_LlamaRuntime_ModelLoader(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	modelPath := getE2EModelPath(t)

	config := llamaruntime.DefaultModelLoaderConfig()
	config.ModelPath = modelPath
	config.RunStartupTest = true
	config.StartupTestTimeout = 60 * time.Second

	loader := llamaruntime.NewModelLoader(config)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	startTime := time.Now()
	client, err := loader.Load(ctx)
	loadDuration := time.Since(startTime)

	if err != nil {
		t.Fatalf("Model loading failed: %v", err)
	}
	defer client.Close()

	t.Logf("Model loaded in %v", loadDuration)

	metadata := loader.Metadata()
	if metadata != nil {
		t.Logf("Model Metadata:")
		t.Logf("  Name: %s", metadata.Name)
		t.Logf("  Size: %s", metadata.SizeHuman)
		t.Logf("  Startup Test: %v (duration: %v)",
			metadata.StartupTestPassed, metadata.StartupTestDuration)
	}

	// Verify the client works
	params := llamaruntime.DefaultInferenceParams()
	params.Prompt = "Hello"
	params.MaxTokens = 10

	result, err := client.Infer(context.Background(), params)
	if err != nil {
		t.Fatalf("Post-load inference failed: %v", err)
	}

	t.Logf("Post-load inference: %s", result.Text)
}

// =============================================================================
// Configuration Integration Tests
// =============================================================================

// TestE2E_LlamaRuntime_ConfigIntegration tests loading config from environment.
func TestE2E_LlamaRuntime_ConfigIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Load core config (which should include llama settings)
	cfg, err := core.LoadConfig()
	if err != nil {
		t.Logf("Core config load failed (expected if .env not set): %v", err)
		t.Skip("Skipping config integration test - core config not available")
	}

	t.Logf("Loaded config:")
	t.Logf("  Canvus Server: %s", cfg.CanvusServerURL)
	t.Logf("  Base LLM URL: %s", cfg.BaseLLMURL)

	// If we have a model path from config, test it
	modelPath := os.Getenv("LLAMA_MODEL_PATH")
	if modelPath == "" {
		t.Log("LLAMA_MODEL_PATH not set, skipping model test")
		return
	}

	if err := llamaruntime.ValidateModelPath(modelPath); err != nil {
		t.Logf("Model path validation failed: %v", err)
		return
	}

	// Create client using config
	clientConfig := llamaruntime.DefaultClientConfig()
	clientConfig.ModelPath = modelPath

	client, err := llamaruntime.NewClient(clientConfig)
	if err != nil {
		t.Fatalf("Failed to create client from config: %v", err)
	}
	defer client.Close()

	t.Log("Successfully created llamaruntime client from environment config")
}

// =============================================================================
// Helper Functions
// =============================================================================

// truncateString truncates a string to a maximum length.
func truncateString(s string, maxLen int) string {
	// Clean up newlines first
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
