package tests

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"go_backend/sdruntime"
)

// TestImageGeneration_EndToEnd tests end-to-end image generation at different resolutions.
// This test requires CUDA and an actual SD model to be available.
// It will skip if CUDA is not available or the model path is not set.
func TestImageGeneration_EndToEnd(t *testing.T) {
	// Check for CUDA availability and model path
	modelPath := os.Getenv("SD_MODEL_PATH")
	if modelPath == "" {
		t.Skip("SD_MODEL_PATH not set, skipping end-to-end image generation test")
	}

	// Verify model file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("Model file not found at %s, skipping test", modelPath)
	}

	// Try to create a context pool to verify CUDA is available
	// If this fails, we skip the test
	pool, err := sdruntime.NewContextPool(1, modelPath)
	if err != nil {
		t.Skipf("Failed to create context pool (CUDA may be unavailable): %v", err)
	}
	defer pool.Close()

	// Create temporary directory for output images
	tempDir := t.TempDir()

	// Test cases for different resolutions
	testCases := []struct {
		name   string
		width  int
		height int
	}{
		{
			name:   "512x512 generation",
			width:  512,
			height: 512,
		},
		{
			name:   "768x768 generation",
			width:  768,
			height: 768,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create generation context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			// Generate image
			params := sdruntime.GenerateParams{
				Prompt:         "a beautiful sunset over mountains, high quality, detailed",
				NegativePrompt: "blurry, low quality, artifacts",
				Width:          tc.width,
				Height:         tc.height,
				Steps:          20,
				CFGScale:       7.0,
				Seed:           42, // Fixed seed for reproducibility
			}

			imageData, err := pool.Generate(ctx, params)
			if err != nil {
				// Check if it's a CUDA unavailable error - skip test
				if isSDUnavailableError(err) {
					t.Skipf("CUDA/SD unavailable: %v", err)
				}
				t.Fatalf("Image generation failed: %v", err)
			}

			// Verify image data is not empty
			if len(imageData) == 0 {
				t.Fatal("Generated image data is empty")
			}

			// Verify PNG signature (first 8 bytes)
			if !isPNG(imageData) {
				t.Errorf("Generated data does not appear to be a valid PNG")
			}

			// Save image to temp directory for manual inspection if needed
			outputPath := filepath.Join(tempDir, tc.name+".png")
			if err := os.WriteFile(outputPath, imageData, 0644); err != nil {
				t.Logf("Warning: Failed to save output image: %v", err)
			} else {
				t.Logf("Generated image saved to: %s", outputPath)
			}

			// Verify image size is reasonable (should be at least 1KB)
			if len(imageData) < 1024 {
				t.Errorf("Generated image size is suspiciously small: %d bytes", len(imageData))
			}

			t.Logf("Successfully generated %dx%d image (%d bytes)", tc.width, tc.height, len(imageData))
		})
	}
}

// TestConcurrentGeneration tests concurrent image generation with a limited pool.
// This verifies that the context pool correctly handles concurrent requests
// and that generation is thread-safe.
func TestConcurrentGeneration(t *testing.T) {
	// Check for CUDA availability and model path
	modelPath := os.Getenv("SD_MODEL_PATH")
	if modelPath == "" {
		t.Skip("SD_MODEL_PATH not set, skipping concurrent generation test")
	}

	// Verify model file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		t.Skipf("Model file not found at %s, skipping test", modelPath)
	}

	// Create context pool with size 2 (bottleneck for 5 concurrent requests)
	poolSize := 2
	pool, err := sdruntime.NewContextPool(poolSize, modelPath)
	if err != nil {
		t.Skipf("Failed to create context pool (CUDA may be unavailable): %v", err)
	}
	defer pool.Close()

	// Create temporary directory for output images
	tempDir := t.TempDir()

	// Number of concurrent generation requests
	numGenerations := 5

	// Use WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(numGenerations)

	// Track results
	results := make([]error, numGenerations)
	imageSizes := make([]int, numGenerations)

	// Launch concurrent generation requests
	for i := 0; i < numGenerations; i++ {
		go func(index int) {
			defer wg.Done()

			// Create generation context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			// Generate image with unique prompt/seed
			params := sdruntime.GenerateParams{
				Prompt:         "a beautiful landscape",
				NegativePrompt: "blurry, low quality",
				Width:          512,
				Height:         512,
				Steps:          15, // Fewer steps for faster testing
				CFGScale:       7.0,
				Seed:           int64(index + 1), // Unique seed per request
			}

			imageData, err := pool.Generate(ctx, params)
			if err != nil {
				results[index] = err
				return
			}

			// Verify PNG
			if !isPNG(imageData) {
				results[index] = err
				return
			}

			// Save image
			outputPath := filepath.Join(tempDir, "concurrent_"+string(rune('0'+index))+".png")
			if err := os.WriteFile(outputPath, imageData, 0644); err != nil {
				t.Logf("Warning: Failed to save concurrent image %d: %v", index, err)
			}

			imageSizes[index] = len(imageData)
		}(i)
	}

	// Wait for all generations to complete
	wg.Wait()

	// Verify results
	successCount := 0
	for i, err := range results {
		if err != nil {
			// Check if it's a CUDA unavailable error - skip entire test
			if isSDUnavailableError(err) {
				t.Skipf("CUDA/SD unavailable during concurrent test: %v", err)
			}
			t.Errorf("Generation %d failed: %v", i, err)
		} else {
			successCount++
			if imageSizes[i] < 1024 {
				t.Errorf("Generation %d produced suspiciously small image: %d bytes", i, imageSizes[i])
			}
		}
	}

	if successCount != numGenerations {
		t.Errorf("Only %d/%d generations succeeded", successCount, numGenerations)
	}

	t.Logf("Successfully completed %d concurrent generations with pool size %d", successCount, poolSize)
}

// isPNG checks if the byte slice has a valid PNG signature.
// PNG files start with: 0x89 0x50 0x4E 0x47 0x0D 0x0A 0x1A 0x0A
func isPNG(data []byte) bool {
	if len(data) < 8 {
		return false
	}
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	for i := 0; i < 8; i++ {
		if data[i] != pngSignature[i] {
			return false
		}
	}
	return true
}

// isSDUnavailableError checks if an error indicates SD/CUDA is unavailable.
// This allows tests to skip gracefully on systems without CUDA.
func isSDUnavailableError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	// Check for common CUDA/SD unavailable error messages
	return contains(errMsg, "stub mode") ||
		contains(errMsg, "CUDA") ||
		contains(errMsg, "not available") ||
		contains(errMsg, "library not available")
}

// contains is a simple string contains check (case-sensitive).
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOfSubstring(s, substr) >= 0)
}

// indexOfSubstring returns the index of the first instance of substr in s, or -1.
func indexOfSubstring(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return 0
	}
	if n > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-n; i++ {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}
