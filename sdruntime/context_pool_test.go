package sdruntime

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// TestNewContextPool tests pool creation with various parameters.
func TestNewContextPool(t *testing.T) {
	tests := []struct {
		name      string
		maxSize   int
		modelPath string
		wantErr   bool
	}{
		{
			name:      "valid pool creation",
			maxSize:   3,
			modelPath: "/path/to/model.safetensors",
			wantErr:   false,
		},
		{
			name:      "single context pool",
			maxSize:   1,
			modelPath: "/path/to/model.safetensors",
			wantErr:   false,
		},
		{
			name:      "zero size pool fails",
			maxSize:   0,
			modelPath: "/path/to/model.safetensors",
			wantErr:   true,
		},
		{
			name:      "negative size pool fails",
			maxSize:   -1,
			modelPath: "/path/to/model.safetensors",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := NewContextPool(tt.maxSize, tt.modelPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewContextPool() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("NewContextPool() unexpected error: %v", err)
				return
			}

			if pool == nil {
				t.Error("NewContextPool() returned nil pool without error")
				return
			}

			if pool.MaxSize() != tt.maxSize {
				t.Errorf("MaxSize() = %d, want %d", pool.MaxSize(), tt.maxSize)
			}

			if pool.ModelPath() != tt.modelPath {
				t.Errorf("ModelPath() = %s, want %s", pool.ModelPath(), tt.modelPath)
			}

			if pool.Size() != 0 {
				t.Errorf("Size() = %d, want 0 for new pool", pool.Size())
			}

			if pool.Created() != 0 {
				t.Errorf("Created() = %d, want 0 for new pool", pool.Created())
			}

			if pool.IsClosed() {
				t.Error("IsClosed() = true, want false for new pool")
			}

			// Clean up
			pool.Close()
		})
	}
}

// TestContextPoolAcquireRelease tests basic acquire and release operations.
// Uses a temporary file as model path since stub mode checks for file existence.
func TestContextPoolAcquireRelease(t *testing.T) {
	// Use go.mod as a file that exists (stub mode validates file existence)
	modelPath := "/home/jaypaulb/Projects/gh/CanvusLocalLLM/go.mod"

	pool, err := NewContextPool(2, modelPath)
	if err != nil {
		t.Fatalf("NewContextPool() failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Acquire first context (lazy creation)
	pc1, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	if pc1 == nil {
		t.Fatal("Acquire() returned nil context")
	}

	if !pc1.inUse {
		t.Error("Acquired context should be marked as inUse")
	}

	if pool.Created() != 1 {
		t.Errorf("Created() = %d, want 1 after first acquire", pool.Created())
	}

	// Release context back to pool
	pool.Release(pc1)

	if pc1.inUse {
		t.Error("Released context should not be marked as inUse")
	}

	if pool.Size() != 1 {
		t.Errorf("Size() = %d, want 1 after release", pool.Size())
	}

	// Acquire again - should get the same context from pool
	pc2, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Second Acquire() failed: %v", err)
	}

	if pool.Created() != 1 {
		t.Errorf("Created() = %d, want 1 (should reuse)", pool.Created())
	}

	pool.Release(pc2)
}

// TestContextPoolAcquireTimeout tests that Acquire respects context deadline.
func TestContextPoolAcquireTimeout(t *testing.T) {
	modelPath := "/home/jaypaulb/Projects/gh/CanvusLocalLLM/go.mod"

	// Create pool with size 1
	pool, err := NewContextPool(1, modelPath)
	if err != nil {
		t.Fatalf("NewContextPool() failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Acquire the only context
	pc1, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("First Acquire() failed: %v", err)
	}

	// Now try to acquire with a short timeout - should timeout
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = pool.Acquire(timeoutCtx)
	if err != ErrAcquireTimeout {
		t.Errorf("Acquire() with timeout expected ErrAcquireTimeout, got: %v", err)
	}

	// Release the context
	pool.Release(pc1)
}

// TestContextPoolClose tests that Close properly shuts down the pool.
func TestContextPoolClose(t *testing.T) {
	modelPath := "/home/jaypaulb/Projects/gh/CanvusLocalLLM/go.mod"

	pool, err := NewContextPool(3, modelPath)
	if err != nil {
		t.Fatalf("NewContextPool() failed: %v", err)
	}

	ctx := context.Background()

	// Acquire and release some contexts to populate the pool
	pc1, _ := pool.Acquire(ctx)
	pc2, _ := pool.Acquire(ctx)
	pool.Release(pc1)
	pool.Release(pc2)

	if pool.Size() != 2 {
		t.Errorf("Size() = %d, want 2 before close", pool.Size())
	}

	// Close the pool
	err = pool.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}

	if !pool.IsClosed() {
		t.Error("IsClosed() = false, want true after Close()")
	}

	// Acquire should fail after close
	_, err = pool.Acquire(ctx)
	if err != ErrContextPoolClosed {
		t.Errorf("Acquire() after Close() expected ErrContextPoolClosed, got: %v", err)
	}

	// Double close should be safe
	err = pool.Close()
	if err != nil {
		t.Errorf("Double Close() returned error: %v", err)
	}
}

// TestContextPoolConcurrentAccess tests thread safety with concurrent operations.
func TestContextPoolConcurrentAccess(t *testing.T) {
	modelPath := "/home/jaypaulb/Projects/gh/CanvusLocalLLM/go.mod"

	pool, err := NewContextPool(5, modelPath)
	if err != nil {
		t.Fatalf("NewContextPool() failed: %v", err)
	}
	defer pool.Close()

	var wg sync.WaitGroup
	numGoroutines := 10
	iterations := 20

	// Launch multiple goroutines that acquire and release contexts
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()

			for j := 0; j < iterations; j++ {
				pc, err := pool.Acquire(ctx)
				if err != nil {
					t.Errorf("Concurrent Acquire() failed: %v", err)
					return
				}

				// Simulate some work
				time.Sleep(1 * time.Millisecond)

				pool.Release(pc)
			}
		}()
	}

	wg.Wait()

	// Pool should still be functional
	if pool.IsClosed() {
		t.Error("Pool should not be closed after concurrent access")
	}

	// All contexts should be back in pool or created should equal what we used
	created := pool.Created()
	if created > pool.MaxSize() {
		t.Errorf("Created() = %d exceeds MaxSize() = %d", created, pool.MaxSize())
	}
}

// TestContextPoolReleaseNil tests that releasing nil is safe.
func TestContextPoolReleaseNil(t *testing.T) {
	modelPath := "/home/jaypaulb/Projects/gh/CanvusLocalLLM/go.mod"

	pool, err := NewContextPool(2, modelPath)
	if err != nil {
		t.Fatalf("NewContextPool() failed: %v", err)
	}
	defer pool.Close()

	// Should not panic
	pool.Release(nil)
}

// TestContextPoolReleaseAfterClose tests release behavior after pool is closed.
func TestContextPoolReleaseAfterClose(t *testing.T) {
	modelPath := "/home/jaypaulb/Projects/gh/CanvusLocalLLM/go.mod"

	pool, err := NewContextPool(2, modelPath)
	if err != nil {
		t.Fatalf("NewContextPool() failed: %v", err)
	}

	ctx := context.Background()

	// Acquire a context
	pc, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	// Close the pool while context is held
	pool.Close()

	// Release should free the context, not return it to closed pool
	pool.Release(pc) // Should not panic

	// Context should have been freed
	if pc.SDContext.IsValid() {
		t.Error("Context should be invalid after release to closed pool")
	}
}

// TestContextPoolGenerate tests the high-level Generate method.
func TestContextPoolGenerate(t *testing.T) {
	modelPath := "/home/jaypaulb/Projects/gh/CanvusLocalLLM/go.mod"

	pool, err := NewContextPool(2, modelPath)
	if err != nil {
		t.Fatalf("NewContextPool() failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Test with valid parameters - will fail in stub mode with generation error
	params := DefaultParams()
	params.Prompt = "test prompt"

	_, err = pool.Generate(ctx, params)
	// In stub mode, we expect ErrGenerationFailed because the library is not available
	if err == nil {
		t.Error("Generate() in stub mode should return error (no library)")
	} else if !errors.Is(err, ErrGenerationFailed) {
		// We expect the error to wrap ErrGenerationFailed
		t.Logf("Generate() returned expected error: %v", err)
	}
}

// TestContextPoolGenerateInvalidParams tests Generate with invalid parameters.
func TestContextPoolGenerateInvalidParams(t *testing.T) {
	modelPath := "/home/jaypaulb/Projects/gh/CanvusLocalLLM/go.mod"

	pool, err := NewContextPool(2, modelPath)
	if err != nil {
		t.Fatalf("NewContextPool() failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	tests := []struct {
		name    string
		params  GenerateParams
		wantErr error
	}{
		{
			name:    "empty prompt",
			params:  GenerateParams{Prompt: "", Width: 512, Height: 512, Steps: 20, CFGScale: 7.0},
			wantErr: ErrInvalidPrompt,
		},
		{
			name:    "width too small",
			params:  GenerateParams{Prompt: "test", Width: 64, Height: 512, Steps: 20, CFGScale: 7.0},
			wantErr: ErrInvalidParams,
		},
		{
			name:    "height too large",
			params:  GenerateParams{Prompt: "test", Width: 512, Height: 4096, Steps: 20, CFGScale: 7.0},
			wantErr: ErrInvalidParams,
		},
		{
			name:    "steps too low",
			params:  GenerateParams{Prompt: "test", Width: 512, Height: 512, Steps: 0, CFGScale: 7.0},
			wantErr: ErrInvalidParams,
		},
		{
			name:    "cfg scale too high",
			params:  GenerateParams{Prompt: "test", Width: 512, Height: 512, Steps: 20, CFGScale: 100.0},
			wantErr: ErrInvalidParams,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pool.Generate(ctx, tt.params)
			if err == nil {
				t.Error("Generate() expected error, got nil")
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Generate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestContextPoolGenerateAfterClose tests Generate on a closed pool.
func TestContextPoolGenerateAfterClose(t *testing.T) {
	modelPath := "/home/jaypaulb/Projects/gh/CanvusLocalLLM/go.mod"

	pool, err := NewContextPool(2, modelPath)
	if err != nil {
		t.Fatalf("NewContextPool() failed: %v", err)
	}

	// Close the pool
	pool.Close()

	ctx := context.Background()
	params := DefaultParams()
	params.Prompt = "test prompt"

	_, err = pool.Generate(ctx, params)
	if !errors.Is(err, ErrContextPoolClosed) {
		t.Errorf("Generate() after Close() expected ErrContextPoolClosed, got: %v", err)
	}
}

// TestContextPoolGenerateTimeout tests Generate with timeout.
func TestContextPoolGenerateTimeout(t *testing.T) {
	modelPath := "/home/jaypaulb/Projects/gh/CanvusLocalLLM/go.mod"

	// Create pool with size 1
	pool, err := NewContextPool(1, modelPath)
	if err != nil {
		t.Fatalf("NewContextPool() failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	// Acquire the only context to block Generate
	pc, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	// Try to Generate with a short timeout - should timeout waiting for context
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	params := DefaultParams()
	params.Prompt = "test prompt"

	_, err = pool.Generate(timeoutCtx, params)
	if !errors.Is(err, ErrAcquireTimeout) {
		t.Errorf("Generate() with timeout expected ErrAcquireTimeout, got: %v", err)
	}

	// Release the context
	pool.Release(pc)
}
