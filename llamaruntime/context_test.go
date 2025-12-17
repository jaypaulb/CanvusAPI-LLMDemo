// Package llamaruntime tests for context pool.
package llamaruntime

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestDefaultContextPoolConfig(t *testing.T) {
	config := DefaultContextPoolConfig()

	if config.NumContexts != 5 {
		t.Errorf("expected NumContexts=5, got %d", config.NumContexts)
	}
	if config.ContextSize != DefaultContextSize {
		t.Errorf("expected ContextSize=%d, got %d", DefaultContextSize, config.ContextSize)
	}
	if config.BatchSize != DefaultBatchSize {
		t.Errorf("expected BatchSize=%d, got %d", DefaultBatchSize, config.BatchSize)
	}
	if config.NumGPULayers != DefaultNumGPULayers {
		t.Errorf("expected NumGPULayers=%d, got %d", DefaultNumGPULayers, config.NumGPULayers)
	}
	if config.NumThreads != DefaultNumThreads {
		t.Errorf("expected NumThreads=%d, got %d", DefaultNumThreads, config.NumThreads)
	}
	if !config.UseMMap {
		t.Error("expected UseMMap=true")
	}
	if config.UseMlock {
		t.Error("expected UseMlock=false")
	}
	if config.AcquireTimeout != 30*time.Second {
		t.Errorf("expected AcquireTimeout=30s, got %v", config.AcquireTimeout)
	}
}

func TestNewContextPoolEmptyPath(t *testing.T) {
	config := DefaultContextPoolConfig()
	config.ModelPath = ""

	_, err := NewContextPool(config)
	if err == nil {
		t.Error("NewContextPool with empty ModelPath should fail")
	}

	var llamaErr *LlamaError
	if errors.As(err, &llamaErr) {
		if llamaErr.Op != "NewContextPool" {
			t.Errorf("expected Op='NewContextPool', got '%s'", llamaErr.Op)
		}
	}
}

func TestNewContextPool(t *testing.T) {
	if hasCUDA() {
		t.Skip("Skipping pool test in CUDA mode (requires real model)")
	}

	config := DefaultContextPoolConfig()
	config.ModelPath = "/tmp/test-model.gguf"
	config.NumContexts = 3

	pool, err := NewContextPool(config)
	if err != nil {
		t.Fatalf("NewContextPool failed: %v", err)
	}
	defer pool.Close()

	// Check stats
	stats := pool.Stats()
	if stats.NumContexts != 3 {
		t.Errorf("expected NumContexts=3, got %d", stats.NumContexts)
	}
	if stats.Available != 3 {
		t.Errorf("expected Available=3, got %d", stats.Available)
	}
	if stats.InUse != 0 {
		t.Errorf("expected InUse=0, got %d", stats.InUse)
	}
	if stats.Closed {
		t.Error("pool should not be closed")
	}
}

func TestContextPoolAcquireRelease(t *testing.T) {
	if hasCUDA() {
		t.Skip("Skipping pool test in CUDA mode (requires real model)")
	}

	config := DefaultContextPoolConfig()
	config.ModelPath = "/tmp/test-model.gguf"
	config.NumContexts = 2

	pool, err := NewContextPool(config)
	if err != nil {
		t.Fatalf("NewContextPool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Acquire first context
	ctx1, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	if ctx1 == nil {
		t.Fatal("Acquire returned nil context")
	}

	stats := pool.Stats()
	if stats.Available != 1 {
		t.Errorf("expected Available=1, got %d", stats.Available)
	}
	if stats.InUse != 1 {
		t.Errorf("expected InUse=1, got %d", stats.InUse)
	}

	// Acquire second context
	ctx2, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	stats = pool.Stats()
	if stats.Available != 0 {
		t.Errorf("expected Available=0, got %d", stats.Available)
	}
	if stats.InUse != 2 {
		t.Errorf("expected InUse=2, got %d", stats.InUse)
	}

	// Release first context
	pool.Release(ctx1)

	stats = pool.Stats()
	if stats.Available != 1 {
		t.Errorf("expected Available=1, got %d", stats.Available)
	}
	if stats.InUse != 1 {
		t.Errorf("expected InUse=1, got %d", stats.InUse)
	}

	// Release second context
	pool.Release(ctx2)

	stats = pool.Stats()
	if stats.Available != 2 {
		t.Errorf("expected Available=2, got %d", stats.Available)
	}
	if stats.InUse != 0 {
		t.Errorf("expected InUse=0, got %d", stats.InUse)
	}

	// Check metrics
	if stats.TotalAcquires != 2 {
		t.Errorf("expected TotalAcquires=2, got %d", stats.TotalAcquires)
	}
	if stats.TotalReleases != 2 {
		t.Errorf("expected TotalReleases=2, got %d", stats.TotalReleases)
	}
}

func TestContextPoolTryAcquire(t *testing.T) {
	if hasCUDA() {
		t.Skip("Skipping pool test in CUDA mode (requires real model)")
	}

	config := DefaultContextPoolConfig()
	config.ModelPath = "/tmp/test-model.gguf"
	config.NumContexts = 1

	pool, err := NewContextPool(config)
	if err != nil {
		t.Fatalf("NewContextPool failed: %v", err)
	}
	defer pool.Close()

	// TryAcquire should succeed
	ctx1, ok := pool.TryAcquire()
	if !ok {
		t.Error("TryAcquire should succeed")
	}
	if ctx1 == nil {
		t.Fatal("TryAcquire returned nil context")
	}

	// TryAcquire should fail (no contexts available)
	ctx2, ok := pool.TryAcquire()
	if ok {
		t.Error("TryAcquire should fail when pool is empty")
	}
	if ctx2 != nil {
		t.Error("TryAcquire should return nil when pool is empty")
	}

	// Release and try again
	pool.Release(ctx1)

	ctx3, ok := pool.TryAcquire()
	if !ok {
		t.Error("TryAcquire should succeed after release")
	}
	if ctx3 == nil {
		t.Fatal("TryAcquire returned nil context after release")
	}
	pool.Release(ctx3)
}

func TestContextPoolAcquireTimeout(t *testing.T) {
	if hasCUDA() {
		t.Skip("Skipping pool test in CUDA mode (requires real model)")
	}

	config := DefaultContextPoolConfig()
	config.ModelPath = "/tmp/test-model.gguf"
	config.NumContexts = 1
	config.AcquireTimeout = 50 * time.Millisecond

	pool, err := NewContextPool(config)
	if err != nil {
		t.Fatalf("NewContextPool failed: %v", err)
	}
	defer pool.Close()

	// Acquire the only context
	ctx1, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	defer pool.Release(ctx1)

	// Try to acquire with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = pool.Acquire(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Acquire should timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}

	// Verify timeout was around expected duration
	if elapsed < 5*time.Millisecond || elapsed > 100*time.Millisecond {
		t.Errorf("timeout duration unexpected: %v", elapsed)
	}

	// Check timeout metric
	stats := pool.Stats()
	if stats.AcquireTimeouts == 0 {
		t.Error("expected AcquireTimeouts > 0")
	}
}

func TestContextPoolClose(t *testing.T) {
	if hasCUDA() {
		t.Skip("Skipping pool test in CUDA mode (requires real model)")
	}

	config := DefaultContextPoolConfig()
	config.ModelPath = "/tmp/test-model.gguf"
	config.NumContexts = 2

	pool, err := NewContextPool(config)
	if err != nil {
		t.Fatalf("NewContextPool failed: %v", err)
	}

	// Acquire one context
	ctx1, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	// Close the pool
	pool.Close()

	if !pool.IsClosed() {
		t.Error("pool should be closed")
	}

	// Acquire should fail after close
	_, err = pool.Acquire(context.Background())
	if err == nil {
		t.Error("Acquire should fail after close")
	}

	// TryAcquire should fail after close
	_, ok := pool.TryAcquire()
	if ok {
		t.Error("TryAcquire should fail after close")
	}

	// Release should not panic (context will be freed)
	pool.Release(ctx1)

	// Close again should be safe (idempotent)
	pool.Close()
}

func TestContextPoolReleaseNil(t *testing.T) {
	if hasCUDA() {
		t.Skip("Skipping pool test in CUDA mode (requires real model)")
	}

	config := DefaultContextPoolConfig()
	config.ModelPath = "/tmp/test-model.gguf"
	config.NumContexts = 1

	pool, err := NewContextPool(config)
	if err != nil {
		t.Fatalf("NewContextPool failed: %v", err)
	}
	defer pool.Close()

	// Release nil should not panic
	pool.Release(nil)
}

func TestContextPoolConcurrent(t *testing.T) {
	if hasCUDA() {
		t.Skip("Skipping pool test in CUDA mode (requires real model)")
	}

	config := DefaultContextPoolConfig()
	config.ModelPath = "/tmp/test-model.gguf"
	config.NumContexts = 3

	pool, err := NewContextPool(config)
	if err != nil {
		t.Fatalf("NewContextPool failed: %v", err)
	}
	defer pool.Close()

	// Run concurrent acquire/release operations
	var wg sync.WaitGroup
	numGoroutines := 10
	numOps := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
				llamaCtx, err := pool.Acquire(ctx)
				cancel()

				if err != nil {
					continue // Timeout or error, that's ok
				}

				// Simulate some work
				time.Sleep(time.Microsecond)

				pool.Release(llamaCtx)
			}
		}()
	}

	wg.Wait()

	// All contexts should be back in pool
	stats := pool.Stats()
	if stats.Available != 3 {
		t.Errorf("expected Available=3 after concurrent test, got %d", stats.Available)
	}
	if stats.InUse != 0 {
		t.Errorf("expected InUse=0 after concurrent test, got %d", stats.InUse)
	}

	t.Logf("Stats after concurrent test: Acquires=%d, Releases=%d, Timeouts=%d",
		stats.TotalAcquires, stats.TotalReleases, stats.AcquireTimeouts)
}

func TestContextPoolWaitForAvailable(t *testing.T) {
	if hasCUDA() {
		t.Skip("Skipping pool test in CUDA mode (requires real model)")
	}

	config := DefaultContextPoolConfig()
	config.ModelPath = "/tmp/test-model.gguf"
	config.NumContexts = 1

	pool, err := NewContextPool(config)
	if err != nil {
		t.Fatalf("NewContextPool failed: %v", err)
	}
	defer pool.Close()

	// WaitForAvailable should return immediately (context available)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	err = pool.WaitForAvailable(ctx)
	cancel()

	if err != nil {
		t.Errorf("WaitForAvailable should succeed: %v", err)
	}

	// Acquire the context
	llamaCtx, _ := pool.Acquire(context.Background())

	// Start a goroutine to release after a delay
	go func() {
		time.Sleep(20 * time.Millisecond)
		pool.Release(llamaCtx)
	}()

	// WaitForAvailable should wait and succeed
	ctx, cancel = context.WithTimeout(context.Background(), 200*time.Millisecond)
	err = pool.WaitForAvailable(ctx)
	cancel()

	if err != nil {
		t.Errorf("WaitForAvailable should succeed after release: %v", err)
	}
}

func TestContextPoolWaitForAvailableTimeout(t *testing.T) {
	if hasCUDA() {
		t.Skip("Skipping pool test in CUDA mode (requires real model)")
	}

	config := DefaultContextPoolConfig()
	config.ModelPath = "/tmp/test-model.gguf"
	config.NumContexts = 1

	pool, err := NewContextPool(config)
	if err != nil {
		t.Fatalf("NewContextPool failed: %v", err)
	}
	defer pool.Close()

	// Acquire the only context
	llamaCtx, _ := pool.Acquire(context.Background())
	defer pool.Release(llamaCtx)

	// WaitForAvailable should timeout
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	err = pool.WaitForAvailable(ctx)
	cancel()

	if err == nil {
		t.Error("WaitForAvailable should timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}

func TestContextPoolConfig(t *testing.T) {
	if hasCUDA() {
		t.Skip("Skipping pool test in CUDA mode (requires real model)")
	}

	config := DefaultContextPoolConfig()
	config.ModelPath = "/tmp/test-model.gguf"
	config.NumContexts = 2
	config.ContextSize = 2048
	config.BatchSize = 256

	pool, err := NewContextPool(config)
	if err != nil {
		t.Fatalf("NewContextPool failed: %v", err)
	}
	defer pool.Close()

	// Check config is preserved
	poolConfig := pool.Config()
	if poolConfig.NumContexts != 2 {
		t.Errorf("expected NumContexts=2, got %d", poolConfig.NumContexts)
	}
	if poolConfig.ContextSize != 2048 {
		t.Errorf("expected ContextSize=2048, got %d", poolConfig.ContextSize)
	}
	if poolConfig.BatchSize != 256 {
		t.Errorf("expected BatchSize=256, got %d", poolConfig.BatchSize)
	}
}

func TestContextPoolModel(t *testing.T) {
	if hasCUDA() {
		t.Skip("Skipping pool test in CUDA mode (requires real model)")
	}

	config := DefaultContextPoolConfig()
	config.ModelPath = "/tmp/test-model.gguf"
	config.NumContexts = 1

	pool, err := NewContextPool(config)
	if err != nil {
		t.Fatalf("NewContextPool failed: %v", err)
	}
	defer pool.Close()

	// Model should be accessible
	model := pool.Model()
	if model == nil {
		t.Error("Model() should return non-nil for pool that owns model")
	}
}

func TestNewContextPoolWithModel(t *testing.T) {
	if hasCUDA() {
		t.Skip("Skipping pool test in CUDA mode (requires real model)")
	}

	// First load a model
	llamaInit()
	model, err := loadModel("/tmp/test-model.gguf", -1, true, false)
	if err != nil {
		t.Fatalf("loadModel failed: %v", err)
	}
	defer model.Close()

	// Create pool with existing model
	config := DefaultContextPoolConfig()
	config.NumContexts = 2

	pool, err := NewContextPoolWithModel(model, config)
	if err != nil {
		t.Fatalf("NewContextPoolWithModel failed: %v", err)
	}
	defer pool.Close()

	// Pool should not own the model
	if pool.Model() != nil {
		t.Error("Pool should not own model when created with NewContextPoolWithModel")
	}

	// Acquire and release should work
	ctx, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	pool.Release(ctx)
}

func TestNewContextPoolWithNilModel(t *testing.T) {
	config := DefaultContextPoolConfig()
	config.NumContexts = 1

	_, err := NewContextPoolWithModel(nil, config)
	if err == nil {
		t.Error("NewContextPoolWithModel with nil model should fail")
	}
}

// Benchmarks

func BenchmarkContextPoolAcquireRelease(b *testing.B) {
	if hasCUDA() {
		b.Skip("Skipping benchmark in CUDA mode")
	}

	config := DefaultContextPoolConfig()
	config.ModelPath = "/tmp/test-model.gguf"
	config.NumContexts = 5

	pool, err := NewContextPool(config)
	if err != nil {
		b.Fatalf("NewContextPool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		llamaCtx, _ := pool.Acquire(ctx)
		pool.Release(llamaCtx)
	}
}

func BenchmarkContextPoolConcurrent(b *testing.B) {
	if hasCUDA() {
		b.Skip("Skipping benchmark in CUDA mode")
	}

	config := DefaultContextPoolConfig()
	config.ModelPath = "/tmp/test-model.gguf"
	config.NumContexts = 5

	pool, err := NewContextPool(config)
	if err != nil {
		b.Fatalf("NewContextPool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			llamaCtx, _ := pool.Acquire(ctx)
			pool.Release(llamaCtx)
		}
	})
}
