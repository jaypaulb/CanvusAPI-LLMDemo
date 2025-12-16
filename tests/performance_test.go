package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"go_backend/sdruntime"
)

// BenchmarkImageGeneration512 benchmarks 512x512 image generation.
// Target: <30s on RTX 3060 with 25 steps.
// Run with: go test -bench=BenchmarkImageGeneration512 -benchtime=1x -timeout=60s
func BenchmarkImageGeneration512(b *testing.B) {
	// Check for CUDA availability and model path
	modelPath := os.Getenv("SD_MODEL_PATH")
	if modelPath == "" {
		b.Skip("SD_MODEL_PATH not set, skipping benchmark")
	}

	// Verify model file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		b.Skipf("Model file not found at %s, skipping benchmark", modelPath)
	}

	// Create context pool with single context (benchmark is sequential)
	pool, err := sdruntime.NewContextPool(1, modelPath)
	if err != nil {
		b.Skipf("Failed to create context pool (CUDA may be unavailable): %v", err)
	}
	defer pool.Close()

	// Define generation parameters
	params := sdruntime.GenerateParams{
		Prompt:         "a beautiful sunset over mountains, high quality, detailed",
		NegativePrompt: "blurry, low quality, artifacts",
		Width:          512,
		Height:         512,
		Steps:          25, // 25 steps as specified
		CFGScale:       7.0,
		Seed:           42, // Fixed seed for reproducibility
	}

	// Reset timer to exclude setup time
	b.ResetTimer()

	// Run benchmark iterations
	for i := 0; i < b.N; i++ {
		// Create generation context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

		// Generate image
		imageData, err := pool.Generate(ctx, params)
		cancel() // Clean up context

		if err != nil {
			// Check if it's a CUDA unavailable error - skip benchmark
			if isSDUnavailableError(err) {
				b.Skipf("CUDA/SD unavailable: %v", err)
			}
			b.Fatalf("Image generation failed: %v", err)
		}

		// Verify image data is not empty
		if len(imageData) == 0 {
			b.Fatal("Generated image data is empty")
		}

		// Verify PNG signature
		if !isPNG(imageData) {
			b.Error("Generated data does not appear to be a valid PNG")
		}

		// Report image size
		b.ReportMetric(float64(len(imageData))/1024.0, "KB/image")
	}

	// Stop timer to report results
	b.StopTimer()

	// Calculate average time per operation
	avgTime := b.Elapsed() / time.Duration(b.N)
	b.Logf("Average generation time for 512x512: %v", avgTime)

	// Verify performance target (30s on RTX 3060)
	// This is informational - benchmark doesn't fail, just reports
	targetTime := 30 * time.Second
	if avgTime > targetTime {
		b.Logf("WARNING: Average time %v exceeds target of %v (may not be RTX 3060)", avgTime, targetTime)
	} else {
		b.Logf("PASS: Average time %v meets target of <%v", avgTime, targetTime)
	}
}

// BenchmarkImageGeneration768 benchmarks 768x768 image generation.
// Target: <60s on RTX 3060 with 25 steps.
// Run with: go test -bench=BenchmarkImageGeneration768 -benchtime=1x -timeout=120s
func BenchmarkImageGeneration768(b *testing.B) {
	// Check for CUDA availability and model path
	modelPath := os.Getenv("SD_MODEL_PATH")
	if modelPath == "" {
		b.Skip("SD_MODEL_PATH not set, skipping benchmark")
	}

	// Verify model file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		b.Skipf("Model file not found at %s, skipping benchmark", modelPath)
	}

	// Create context pool with single context (benchmark is sequential)
	pool, err := sdruntime.NewContextPool(1, modelPath)
	if err != nil {
		b.Skipf("Failed to create context pool (CUDA may be unavailable): %v", err)
	}
	defer pool.Close()

	// Define generation parameters
	params := sdruntime.GenerateParams{
		Prompt:         "a beautiful sunset over mountains, high quality, detailed",
		NegativePrompt: "blurry, low quality, artifacts",
		Width:          768,
		Height:         768,
		Steps:          25, // 25 steps as specified
		CFGScale:       7.0,
		Seed:           42, // Fixed seed for reproducibility
	}

	// Reset timer to exclude setup time
	b.ResetTimer()

	// Run benchmark iterations
	for i := 0; i < b.N; i++ {
		// Create generation context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)

		// Generate image
		imageData, err := pool.Generate(ctx, params)
		cancel() // Clean up context

		if err != nil {
			// Check if it's a CUDA unavailable error - skip benchmark
			if isSDUnavailableError(err) {
				b.Skipf("CUDA/SD unavailable: %v", err)
			}
			b.Fatalf("Image generation failed: %v", err)
		}

		// Verify image data is not empty
		if len(imageData) == 0 {
			b.Fatal("Generated image data is empty")
		}

		// Verify PNG signature
		if !isPNG(imageData) {
			b.Error("Generated data does not appear to be a valid PNG")
		}

		// Report image size
		b.ReportMetric(float64(len(imageData))/1024.0, "KB/image")
	}

	// Stop timer to report results
	b.StopTimer()

	// Calculate average time per operation
	avgTime := b.Elapsed() / time.Duration(b.N)
	b.Logf("Average generation time for 768x768: %v", avgTime)

	// Verify performance target (60s on RTX 3060)
	// This is informational - benchmark doesn't fail, just reports
	targetTime := 60 * time.Second
	if avgTime > targetTime {
		b.Logf("WARNING: Average time %v exceeds target of %v (may not be RTX 3060)", avgTime, targetTime)
	} else {
		b.Logf("PASS: Average time %v meets target of <%v", avgTime, targetTime)
	}
}

// BenchmarkCUDAUtilization provides a benchmark to monitor CUDA utilization patterns.
// This helps verify that the GPU is being properly utilized during generation.
// Run with: go test -bench=BenchmarkCUDAUtilization -benchtime=3x -timeout=180s
func BenchmarkCUDAUtilization(b *testing.B) {
	// Check for CUDA availability and model path
	modelPath := os.Getenv("SD_MODEL_PATH")
	if modelPath == "" {
		b.Skip("SD_MODEL_PATH not set, skipping benchmark")
	}

	// Verify model file exists
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		b.Skipf("Model file not found at %s, skipping benchmark", modelPath)
	}

	// Create context pool with single context
	pool, err := sdruntime.NewContextPool(1, modelPath)
	if err != nil {
		b.Skipf("Failed to create context pool (CUDA may be unavailable): %v", err)
	}
	defer pool.Close()

	// Use smaller resolution for faster iterations to measure CUDA utilization patterns
	params := sdruntime.GenerateParams{
		Prompt:         "test image for CUDA utilization",
		NegativePrompt: "low quality",
		Width:          512,
		Height:         512,
		Steps:          20,
		CFGScale:       7.0,
		Seed:           42,
	}

	b.ResetTimer()

	// Track timing variance to detect CUDA throttling or thermal issues
	var timings []time.Duration

	for i := 0; i < b.N; i++ {
		start := time.Now()

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		imageData, err := pool.Generate(ctx, params)
		cancel()

		elapsed := time.Since(start)
		timings = append(timings, elapsed)

		if err != nil {
			if isSDUnavailableError(err) {
				b.Skipf("CUDA/SD unavailable: %v", err)
			}
			b.Fatalf("Image generation failed: %v", err)
		}

		if len(imageData) == 0 || !isPNG(imageData) {
			b.Fatal("Invalid image generated")
		}
	}

	b.StopTimer()

	// Calculate statistics
	if len(timings) > 0 {
		var total time.Duration
		minTime := timings[0]
		maxTime := timings[0]

		for _, t := range timings {
			total += t
			if t < minTime {
				minTime = t
			}
			if t > maxTime {
				maxTime = t
			}
		}

		avgTime := total / time.Duration(len(timings))
		variance := maxTime - minTime

		b.Logf("CUDA Utilization Stats:")
		b.Logf("  Average: %v", avgTime)
		b.Logf("  Min: %v", minTime)
		b.Logf("  Max: %v", maxTime)
		b.Logf("  Variance: %v", variance)
		b.Logf("  Iterations: %d", len(timings))

		// High variance might indicate thermal throttling or CUDA contention
		variancePercent := float64(variance) / float64(avgTime) * 100
		if variancePercent > 20 {
			b.Logf("WARNING: High timing variance (%.1f%%) - possible thermal throttling or CUDA contention", variancePercent)
		} else {
			b.Logf("PASS: Consistent CUDA utilization (%.1f%% variance)", variancePercent)
		}
	}
}
