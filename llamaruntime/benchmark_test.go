// Package llamaruntime provides Go bindings to llama.cpp for local LLM inference.
// This file contains performance benchmarks for the llamaruntime package.
//
// To run benchmarks with a real model:
//
//	LLAMA_TEST_MODEL=/path/to/model.gguf go test -bench=. -benchmem ./llamaruntime/...
//
// Performance Targets:
// - RTX 3060 (12GB): 20+ tokens/sec
// - RTX 4070 (12GB): 40+ tokens/sec
// - First token latency: <500ms
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
	"testing"
	"time"
)

// =============================================================================
// Benchmark Helpers
// =============================================================================

// getBenchmarkModelPath returns a model path for benchmarking.
// Similar to getTestModelPath but without t.Skip (benchmarks just skip silently).
func getBenchmarkModelPath(b *testing.B) string {
	b.Helper()

	// Check LLAMA_TEST_MODEL first
	if path := os.Getenv("LLAMA_TEST_MODEL"); path != "" {
		if err := ValidateModelPath(path); err == nil {
			return path
		}
	}

	// Check LLAMA_MODEL_PATH
	if path := os.Getenv("LLAMA_MODEL_PATH"); path != "" {
		if err := ValidateModelPath(path); err == nil {
			return path
		}
	}

	// Search for models in project root
	cwd, err := os.Getwd()
	if err != nil {
		b.Skip("No model file available for benchmarks")
		return ""
	}

	// Walk up to find project root
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			modelsDir := filepath.Join(dir, "models")
			matches, err := filepath.Glob(filepath.Join(modelsDir, "*.gguf"))
			if err == nil && len(matches) > 0 {
				for _, match := range matches {
					if err := ValidateModelPath(match); err == nil {
						return match
					}
				}
			}
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	b.Skip("No model file available for benchmarks. Set LLAMA_TEST_MODEL")
	return ""
}

// createBenchmarkClient creates a Client for benchmarking.
func createBenchmarkClient(b *testing.B, numContexts int) *Client {
	b.Helper()

	modelPath := getBenchmarkModelPath(b)
	if modelPath == "" {
		return nil
	}

	config := DefaultClientConfig()
	config.ModelPath = modelPath
	config.NumContexts = numContexts
	config.VerboseLogging = false

	client, err := NewClient(config)
	if err != nil {
		b.Fatalf("Failed to create client: %v", err)
	}

	return client
}

// =============================================================================
// Basic Inference Benchmarks
// =============================================================================

// BenchmarkInfer measures single-request inference performance.
func BenchmarkInfer(b *testing.B) {
	client := createBenchmarkClient(b, 1)
	if client == nil {
		return
	}
	defer client.Close()

	params := DefaultInferenceParams()
	params.Prompt = "What is the capital of France? Answer briefly."
	params.MaxTokens = 50
	params.Temperature = 0.1

	ctx := context.Background()

	// Warmup
	_, _ = client.Infer(ctx, params)

	b.ResetTimer()
	b.ReportAllocs()

	var totalTokens int64
	var totalDuration time.Duration

	for i := 0; i < b.N; i++ {
		result, err := client.Infer(ctx, params)
		if err != nil {
			b.Fatalf("Infer failed: %v", err)
		}
		totalTokens += int64(result.TokensGenerated)
		totalDuration += result.Duration
	}

	b.StopTimer()

	// Report metrics
	avgTokensPerSec := float64(totalTokens) / totalDuration.Seconds()
	b.ReportMetric(avgTokensPerSec, "tokens/sec")
	b.ReportMetric(float64(totalDuration.Milliseconds())/float64(b.N), "ms/op")
}

// BenchmarkInfer_ShortPrompt measures inference with minimal prompt.
func BenchmarkInfer_ShortPrompt(b *testing.B) {
	client := createBenchmarkClient(b, 1)
	if client == nil {
		return
	}
	defer client.Close()

	params := DefaultInferenceParams()
	params.Prompt = "Hi"
	params.MaxTokens = 20
	params.Temperature = 0.1

	ctx := context.Background()

	// Warmup
	_, _ = client.Infer(ctx, params)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.Infer(ctx, params)
		if err != nil {
			b.Fatalf("Infer failed: %v", err)
		}
	}
}

// BenchmarkInfer_LongPrompt measures inference with longer context.
func BenchmarkInfer_LongPrompt(b *testing.B) {
	client := createBenchmarkClient(b, 1)
	if client == nil {
		return
	}
	defer client.Close()

	// Generate a longer prompt (~500 tokens)
	longPrompt := "Please summarize the following technical document:\n\n"
	longPrompt += "Artificial intelligence (AI) is intelligence demonstrated by machines, as opposed to "
	longPrompt += "natural intelligence displayed by animals including humans. AI research has been "
	longPrompt += "defined as the field of study of intelligent agents, which refers to any system that "
	longPrompt += "perceives its environment and takes actions that maximize its chance of achieving "
	longPrompt += "its goals. The term artificial intelligence had previously been used to describe "
	longPrompt += "machines that mimic and display human cognitive skills that are associated with "
	longPrompt += "the human mind, such as learning and problem-solving. This definition has since "
	longPrompt += "been rejected by major AI researchers who now describe AI in terms of rationality "
	longPrompt += "and acting rationally, which does not limit how intelligence can be articulated.\n\n"
	longPrompt += "Machine learning (ML) is a subset of AI that enables systems to learn and improve "
	longPrompt += "from experience without being explicitly programmed. Deep learning is a subset of "
	longPrompt += "machine learning that uses neural networks with many layers.\n\n"
	longPrompt += "Provide a 2-3 sentence summary:"

	params := DefaultInferenceParams()
	params.Prompt = longPrompt
	params.MaxTokens = 100
	params.Temperature = 0.3

	ctx := context.Background()

	// Warmup
	_, _ = client.Infer(ctx, params)

	b.ResetTimer()
	b.ReportAllocs()

	var totalTokens int64
	var totalDuration time.Duration

	for i := 0; i < b.N; i++ {
		result, err := client.Infer(ctx, params)
		if err != nil {
			b.Fatalf("Infer failed: %v", err)
		}
		totalTokens += int64(result.TokensGenerated)
		totalDuration += result.Duration
	}

	b.StopTimer()

	avgTokensPerSec := float64(totalTokens) / totalDuration.Seconds()
	b.ReportMetric(avgTokensPerSec, "tokens/sec")
}

// BenchmarkInfer_MaxTokenVariations measures how output length affects performance.
func BenchmarkInfer_MaxTokenVariations(b *testing.B) {
	client := createBenchmarkClient(b, 1)
	if client == nil {
		return
	}
	defer client.Close()

	tokenCounts := []int{10, 50, 100, 200}
	ctx := context.Background()

	for _, maxTokens := range tokenCounts {
		b.Run(fmt.Sprintf("MaxTokens_%d", maxTokens), func(b *testing.B) {
			params := DefaultInferenceParams()
			params.Prompt = "Write a story about a robot."
			params.MaxTokens = maxTokens
			params.Temperature = 0.3

			// Warmup
			_, _ = client.Infer(ctx, params)

			b.ResetTimer()

			var totalTokens int64
			var totalDuration time.Duration

			for i := 0; i < b.N; i++ {
				result, err := client.Infer(ctx, params)
				if err != nil {
					b.Fatalf("Infer failed: %v", err)
				}
				totalTokens += int64(result.TokensGenerated)
				totalDuration += result.Duration
			}

			b.StopTimer()

			avgTokensPerSec := float64(totalTokens) / totalDuration.Seconds()
			b.ReportMetric(avgTokensPerSec, "tokens/sec")
		})
	}
}

// =============================================================================
// Concurrent Inference Benchmarks
// =============================================================================

// BenchmarkInferConcurrent measures throughput with concurrent requests.
func BenchmarkInferConcurrent(b *testing.B) {
	concurrencyLevels := []int{1, 2, 3, 4}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency_%d", concurrency), func(b *testing.B) {
			client := createBenchmarkClient(b, concurrency)
			if client == nil {
				return
			}
			defer client.Close()

			params := DefaultInferenceParams()
			params.Prompt = "What is 2+2?"
			params.MaxTokens = 20
			params.Temperature = 0.1

			ctx := context.Background()

			// Warmup
			_, _ = client.Infer(ctx, params)

			b.ResetTimer()
			b.ReportAllocs()

			var wg sync.WaitGroup
			requestCh := make(chan int, b.N)

			// Fill the channel with work items
			for i := 0; i < b.N; i++ {
				requestCh <- i
			}
			close(requestCh)

			// Start workers
			for w := 0; w < concurrency; w++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for range requestCh {
						_, err := client.Infer(ctx, params)
						if err != nil {
							b.Errorf("Infer failed: %v", err)
						}
					}
				}()
			}

			wg.Wait()
		})
	}
}

// BenchmarkInferConcurrent_HighLoad measures throughput under heavy load.
func BenchmarkInferConcurrent_HighLoad(b *testing.B) {
	client := createBenchmarkClient(b, 4)
	if client == nil {
		return
	}
	defer client.Close()

	params := DefaultInferenceParams()
	params.Prompt = "Count to 5."
	params.MaxTokens = 30
	params.Temperature = 0.1

	ctx := context.Background()

	// Warmup
	_, _ = client.Infer(ctx, params)

	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	concurrency := 8 // More workers than contexts to test queueing

	requestsPerWorker := b.N / concurrency
	if requestsPerWorker < 1 {
		requestsPerWorker = 1
	}

	startTime := time.Now()

	for w := 0; w < concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < requestsPerWorker; i++ {
				_, err := client.Infer(ctx, params)
				if err != nil {
					// Don't fail on individual errors in high-load test
					continue
				}
			}
		}()
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	b.StopTimer()

	requestsCompleted := concurrency * requestsPerWorker
	throughput := float64(requestsCompleted) / totalDuration.Seconds()
	b.ReportMetric(throughput, "requests/sec")
}

// =============================================================================
// First Token Latency Benchmarks
// =============================================================================

// BenchmarkFirstTokenLatency measures time to first token.
// Target: <500ms for first token
func BenchmarkFirstTokenLatency(b *testing.B) {
	client := createBenchmarkClient(b, 1)
	if client == nil {
		return
	}
	defer client.Close()

	params := DefaultInferenceParams()
	params.Prompt = "Hello"
	params.MaxTokens = 1 // Just one token to measure latency
	params.Temperature = 0.1

	ctx := context.Background()

	// Warmup (important for accurate latency measurement)
	for i := 0; i < 3; i++ {
		_, _ = client.Infer(ctx, params)
	}

	b.ResetTimer()
	b.ReportAllocs()

	var totalLatency time.Duration

	for i := 0; i < b.N; i++ {
		startTime := time.Now()
		_, err := client.Infer(ctx, params)
		latency := time.Since(startTime)

		if err != nil {
			b.Fatalf("Infer failed: %v", err)
		}

		totalLatency += latency
	}

	b.StopTimer()

	avgLatencyMs := float64(totalLatency.Milliseconds()) / float64(b.N)
	b.ReportMetric(avgLatencyMs, "ms/first_token")

	// Check against target
	if avgLatencyMs > 500 {
		b.Logf("WARNING: First token latency (%.2fms) exceeds 500ms target", avgLatencyMs)
	}
}

// BenchmarkFirstTokenLatency_ColdStart measures latency without warmup.
func BenchmarkFirstTokenLatency_ColdStart(b *testing.B) {
	modelPath := getBenchmarkModelPath(b)
	if modelPath == "" {
		return
	}

	params := DefaultInferenceParams()
	params.Prompt = "Hello"
	params.MaxTokens = 1
	params.Temperature = 0.1

	ctx := context.Background()

	b.ResetTimer()

	var totalLatency time.Duration

	for i := 0; i < b.N; i++ {
		// Create fresh client each iteration (cold start)
		config := DefaultClientConfig()
		config.ModelPath = modelPath
		config.NumContexts = 1

		clientStart := time.Now()
		client, err := NewClient(config)
		if err != nil {
			b.Fatalf("Failed to create client: %v", err)
		}

		_, err = client.Infer(ctx, params)
		latency := time.Since(clientStart)
		client.Close()

		if err != nil {
			b.Fatalf("Infer failed: %v", err)
		}

		totalLatency += latency
	}

	b.StopTimer()

	avgLatencyMs := float64(totalLatency.Milliseconds()) / float64(b.N)
	b.ReportMetric(avgLatencyMs, "ms/cold_start")
}

// =============================================================================
// Memory Benchmarks
// =============================================================================

// BenchmarkMemoryUsage measures memory allocation patterns.
func BenchmarkMemoryUsage(b *testing.B) {
	client := createBenchmarkClient(b, 1)
	if client == nil {
		return
	}
	defer client.Close()

	params := DefaultInferenceParams()
	params.Prompt = "What is AI?"
	params.MaxTokens = 50
	params.Temperature = 0.1

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := client.Infer(ctx, params)
		if err != nil {
			b.Fatalf("Infer failed: %v", err)
		}
	}
}

// =============================================================================
// Context Pool Benchmarks
// =============================================================================

// BenchmarkContextAcquireRelease measures context pool performance.
func BenchmarkContextAcquireRelease(b *testing.B) {
	modelPath := getBenchmarkModelPath(b)
	if modelPath == "" {
		return
	}

	poolConfig := ContextPoolConfig{
		ModelPath:      modelPath,
		NumContexts:    4,
		ContextSize:    DefaultContextSize,
		BatchSize:      DefaultBatchSize,
		NumGPULayers:   DefaultNumGPULayers,
		NumThreads:     DefaultNumThreads,
		UseMMap:        true,
		AcquireTimeout: 30 * time.Second,
	}

	pool, err := NewContextPool(poolConfig)
	if err != nil {
		b.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		llamaCtx, err := pool.Acquire(ctx)
		if err != nil {
			b.Fatalf("Acquire failed: %v", err)
		}
		pool.Release(llamaCtx)
	}
}

// BenchmarkContextPool_Contention measures pool performance under contention.
func BenchmarkContextPool_Contention(b *testing.B) {
	modelPath := getBenchmarkModelPath(b)
	if modelPath == "" {
		return
	}

	poolConfig := ContextPoolConfig{
		ModelPath:      modelPath,
		NumContexts:    2, // Limited contexts to create contention
		ContextSize:    DefaultContextSize,
		BatchSize:      DefaultBatchSize,
		NumGPULayers:   DefaultNumGPULayers,
		NumThreads:     DefaultNumThreads,
		UseMMap:        true,
		AcquireTimeout: 30 * time.Second,
	}

	pool, err := NewContextPool(poolConfig)
	if err != nil {
		b.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	numWorkers := 8 // More workers than contexts

	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	requestsPerWorker := b.N / numWorkers
	if requestsPerWorker < 1 {
		requestsPerWorker = 1
	}

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < requestsPerWorker; i++ {
				llamaCtx, err := pool.Acquire(ctx)
				if err != nil {
					continue
				}
				// Simulate some work
				time.Sleep(time.Microsecond)
				pool.Release(llamaCtx)
			}
		}()
	}

	wg.Wait()
}

// =============================================================================
// Comparison Benchmarks
// =============================================================================

// BenchmarkTemperatureVariations measures impact of temperature on generation.
func BenchmarkTemperatureVariations(b *testing.B) {
	client := createBenchmarkClient(b, 1)
	if client == nil {
		return
	}
	defer client.Close()

	temperatures := []float32{0.0, 0.3, 0.7, 1.0}
	ctx := context.Background()

	for _, temp := range temperatures {
		b.Run(fmt.Sprintf("Temp_%.1f", temp), func(b *testing.B) {
			params := DefaultInferenceParams()
			params.Prompt = "Write a haiku."
			params.MaxTokens = 30
			params.Temperature = temp

			// Warmup
			_, _ = client.Infer(ctx, params)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := client.Infer(ctx, params)
				if err != nil {
					b.Fatalf("Infer failed: %v", err)
				}
			}
		})
	}
}
