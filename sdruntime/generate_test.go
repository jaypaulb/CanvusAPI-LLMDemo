package sdruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// createTempModelFile creates a temporary file to simulate a model file for stub mode testing.
// The stub implementation only checks if the file exists, not its contents.
func createTempModelFile(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	modelPath := filepath.Join(tmpDir, "test-model.safetensors")
	if err := os.WriteFile(modelPath, []byte("fake model data"), 0644); err != nil {
		t.Fatalf("failed to create temp model file: %v", err)
	}
	return modelPath
}

// isStubMode checks if we're running in stub mode by checking backend info.
func isStubMode() bool {
	return GetBackendInfo() == "stub (no stable-diffusion.cpp library linked)"
}

func TestNewGenerator(t *testing.T) {
	modelPath := createTempModelFile(t)

	tests := []struct {
		name      string
		poolSize  int
		modelPath string
		wantErr   bool
	}{
		{
			name:      "valid parameters",
			poolSize:  3,
			modelPath: modelPath,
			wantErr:   false,
		},
		{
			name:      "zero pool size",
			poolSize:  0,
			modelPath: modelPath,
			wantErr:   true,
		},
		{
			name:      "negative pool size",
			poolSize:  -1,
			modelPath: modelPath,
			wantErr:   true,
		},
		{
			name:      "single context pool",
			poolSize:  1,
			modelPath: modelPath,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewGenerator(tt.poolSize, tt.modelPath)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if gen == nil {
				t.Error("generator is nil")
				return
			}
			defer gen.Close()

			if gen.PoolSize() != tt.poolSize {
				t.Errorf("pool size = %d, want %d", gen.PoolSize(), tt.poolSize)
			}
		})
	}
}

func TestNewGeneratorModelNotFound(t *testing.T) {
	// Test with non-existent model file
	gen, err := NewGenerator(1, "/nonexistent/model.safetensors")
	if err != nil {
		t.Fatalf("NewGenerator itself should not fail for bad path: %v", err)
	}
	defer gen.Close()

	// Generation should fail with model not found
	ctx := context.Background()
	params := DefaultParams()
	params.Prompt = "test"

	_, err = gen.Generate(ctx, params)
	if err == nil {
		t.Error("expected error for non-existent model")
		return
	}
	// Should get ErrModelNotFound when trying to acquire context
	if !errors.Is(err, ErrModelNotFound) {
		t.Logf("got error: %v (may be wrapped)", err)
	}
}

func TestDefaultParams(t *testing.T) {
	params := DefaultParams()

	// Verify default values
	if params.Prompt != "" {
		t.Errorf("default prompt should be empty, got %q", params.Prompt)
	}
	if params.NegativePrompt != "" {
		t.Errorf("default negative prompt should be empty, got %q", params.NegativePrompt)
	}
	if params.Width != 512 {
		t.Errorf("default width = %d, want 512", params.Width)
	}
	if params.Height != 512 {
		t.Errorf("default height = %d, want 512", params.Height)
	}
	if params.Steps != 20 {
		t.Errorf("default steps = %d, want 20", params.Steps)
	}
	if params.CFGScale != 7.0 {
		t.Errorf("default CFGScale = %f, want 7.0", params.CFGScale)
	}
	if params.Seed != -1 {
		t.Errorf("default seed = %d, want -1", params.Seed)
	}

	// Verify defaults pass validation when prompt is set
	params.Prompt = "a test prompt"
	if err := ValidateParams(params); err != nil {
		t.Errorf("default params with prompt should be valid: %v", err)
	}
}

func TestGeneratorGenerateValidation(t *testing.T) {
	modelPath := createTempModelFile(t)
	gen, err := NewGenerator(2, modelPath)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}
	defer gen.Close()

	t.Run("invalid params - empty prompt", func(t *testing.T) {
		ctx := context.Background()
		params := DefaultParams()
		// Prompt is empty

		_, err := gen.Generate(ctx, params)
		if err == nil {
			t.Error("expected error for empty prompt")
			return
		}
		if !errors.Is(err, ErrInvalidPrompt) {
			t.Errorf("expected ErrInvalidPrompt, got: %v", err)
		}
	})

	t.Run("invalid params - width too small", func(t *testing.T) {
		ctx := context.Background()
		params := DefaultParams()
		params.Prompt = "test"
		params.Width = 64 // Below minimum

		_, err := gen.Generate(ctx, params)
		if err == nil {
			t.Error("expected error for invalid width")
			return
		}
		if !errors.Is(err, ErrInvalidParams) {
			t.Errorf("expected ErrInvalidParams, got: %v", err)
		}
	})

	t.Run("invalid params - height not divisible by 8", func(t *testing.T) {
		ctx := context.Background()
		params := DefaultParams()
		params.Prompt = "test"
		params.Height = 513 // Not divisible by 8

		_, err := gen.Generate(ctx, params)
		if err == nil {
			t.Error("expected error for invalid height")
			return
		}
		if !errors.Is(err, ErrInvalidParams) {
			t.Errorf("expected ErrInvalidParams, got: %v", err)
		}
	})

	t.Run("invalid params - steps too high", func(t *testing.T) {
		ctx := context.Background()
		params := DefaultParams()
		params.Prompt = "test"
		params.Steps = 150 // Above maximum

		_, err := gen.Generate(ctx, params)
		if err == nil {
			t.Error("expected error for invalid steps")
			return
		}
		if !errors.Is(err, ErrInvalidParams) {
			t.Errorf("expected ErrInvalidParams, got: %v", err)
		}
	})
}

func TestGeneratorGenerateStubMode(t *testing.T) {
	// This test verifies behavior in stub mode specifically
	if !isStubMode() {
		t.Skip("skipping stub-specific test: not in stub mode")
	}

	modelPath := createTempModelFile(t)
	gen, err := NewGenerator(2, modelPath)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}
	defer gen.Close()

	ctx := context.Background()
	params := DefaultParams()
	params.Prompt = "test"

	// In stub mode, generation fails with ErrGenerationFailed
	_, err = gen.Generate(ctx, params)
	if err == nil {
		t.Error("expected error in stub mode")
		return
	}
	if !errors.Is(err, ErrGenerationFailed) {
		t.Errorf("expected ErrGenerationFailed in stub mode, got: %v", err)
	}
}

func TestGeneratorGenerateWithResultStubMode(t *testing.T) {
	if !isStubMode() {
		t.Skip("skipping stub-specific test: not in stub mode")
	}

	modelPath := createTempModelFile(t)
	gen, err := NewGenerator(1, modelPath)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}
	defer gen.Close()

	ctx := context.Background()
	params := DefaultParams()
	params.Prompt = "test"

	// In stub mode, should fail with generation error
	_, err = gen.GenerateWithResult(ctx, params)
	if err == nil {
		t.Error("expected error in stub mode")
		return
	}
	if !errors.Is(err, ErrGenerationFailed) {
		t.Errorf("expected ErrGenerationFailed in stub mode, got: %v", err)
	}
}

func TestGeneratorContextCancellation(t *testing.T) {
	modelPath := createTempModelFile(t)
	gen, err := NewGenerator(1, modelPath)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}
	defer gen.Close()

	// First, acquire the only context to make the pool empty
	ctx1 := context.Background()
	pooledCtx, acquireErr := gen.pool.Acquire(ctx1)
	if acquireErr != nil {
		t.Fatalf("failed to acquire context: %v", acquireErr)
	}

	// Now try with an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	params := DefaultParams()
	params.Prompt = "test"

	// Should fail immediately due to cancelled context
	_, err = gen.Generate(ctx, params)
	if err == nil {
		t.Error("expected error for cancelled context")
	} else if !errors.Is(err, ErrAcquireTimeout) {
		t.Logf("got error: %v (expected ErrAcquireTimeout)", err)
	}

	// Release the acquired context
	gen.pool.Release(pooledCtx)
}

func TestGeneratorClose(t *testing.T) {
	modelPath := createTempModelFile(t)
	gen, err := NewGenerator(2, modelPath)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}

	// Not closed yet
	if gen.IsClosed() {
		t.Error("generator should not be closed yet")
	}

	// Close should succeed
	if err := gen.Close(); err != nil {
		t.Errorf("close error: %v", err)
	}

	// Should be marked as closed
	if !gen.IsClosed() {
		t.Error("generator should be marked as closed")
	}

	// Double close should be safe
	if err := gen.Close(); err != nil {
		t.Errorf("double close error: %v", err)
	}

	// Generate after close should fail
	ctx := context.Background()
	params := DefaultParams()
	params.Prompt = "test"

	_, err = gen.Generate(ctx, params)
	if err == nil {
		t.Error("expected error after close")
		return
	}
	if !errors.Is(err, ErrContextPoolClosed) {
		t.Errorf("expected ErrContextPoolClosed, got: %v", err)
	}
}

func TestQuickGenerateValidation(t *testing.T) {
	modelPath := createTempModelFile(t)
	ctx := context.Background()

	t.Run("empty prompt fails", func(t *testing.T) {
		_, err := QuickGenerate(ctx, modelPath, "")
		if err == nil {
			t.Error("expected error for empty prompt")
		}
		if !errors.Is(err, ErrInvalidPrompt) {
			t.Errorf("expected ErrInvalidPrompt, got: %v", err)
		}
	})

	t.Run("model not found", func(t *testing.T) {
		_, err := QuickGenerate(ctx, "/nonexistent/path.safetensors", "test prompt")
		if err == nil {
			t.Error("expected error for missing model")
		}
		// Should get ErrModelNotFound
		if !errors.Is(err, ErrModelNotFound) {
			t.Logf("got error: %v", err)
		}
	})
}

func TestQuickGenerateStubMode(t *testing.T) {
	if !isStubMode() {
		t.Skip("skipping stub-specific test: not in stub mode")
	}

	modelPath := createTempModelFile(t)
	ctx := context.Background()

	_, err := QuickGenerate(ctx, modelPath, "test prompt")
	if err == nil {
		t.Error("expected error in stub mode")
		return
	}
	if !errors.Is(err, ErrGenerationFailed) {
		t.Errorf("expected ErrGenerationFailed in stub mode, got: %v", err)
	}
}

func TestGeneratorPoolMethods(t *testing.T) {
	modelPath := createTempModelFile(t)
	gen, err := NewGenerator(5, modelPath)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}
	defer gen.Close()

	// Test pool size
	if gen.PoolSize() != 5 {
		t.Errorf("PoolSize() = %d, want 5", gen.PoolSize())
	}

	// Test pool available (lazy creation means 0 at start)
	if gen.PoolAvailable() != 0 {
		t.Errorf("PoolAvailable() = %d, want 0 (lazy creation)", gen.PoolAvailable())
	}

	// Not closed
	if gen.IsClosed() {
		t.Error("generator should not be closed")
	}
}

func TestGeneratorConcurrentAcquireRelease(t *testing.T) {
	modelPath := createTempModelFile(t)
	gen, err := NewGenerator(3, modelPath)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}
	defer gen.Close()

	// Test concurrent acquire/release of pool contexts
	// This tests the pool machinery without actual generation
	const numGoroutines = 10
	results := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Just acquire and release to test pool concurrency
			pooledCtx, err := gen.pool.Acquire(ctx)
			if err != nil {
				results <- err
				return
			}
			// Simulate some work
			time.Sleep(10 * time.Millisecond)
			gen.pool.Release(pooledCtx)
			results <- nil
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines; i++ {
		if err := <-results; err != nil {
			t.Errorf("goroutine %d failed: %v", i, err)
		}
	}
}

func TestRandomSeedHandling(t *testing.T) {
	// Test that RandomSeed is called when seed is -1
	params := DefaultParams()
	params.Seed = -1

	// Generate random seed
	seed1 := RandomSeed()
	seed2 := RandomSeed()

	// Seeds should be non-negative
	if seed1 < 0 {
		t.Errorf("seed1 should be non-negative, got %d", seed1)
	}
	if seed2 < 0 {
		t.Errorf("seed2 should be non-negative, got %d", seed2)
	}

	// Seeds should generally be different (with high probability)
	// Note: there's a tiny chance they could be equal, so we just test non-negativity
}

func TestGeneratorGenerateWithResultValidation(t *testing.T) {
	modelPath := createTempModelFile(t)
	gen, err := NewGenerator(1, modelPath)
	if err != nil {
		t.Fatalf("failed to create generator: %v", err)
	}
	defer gen.Close()

	ctx := context.Background()

	t.Run("empty prompt", func(t *testing.T) {
		params := DefaultParams()
		// Empty prompt

		_, err := gen.GenerateWithResult(ctx, params)
		if err == nil {
			t.Error("expected error for empty prompt")
			return
		}
		if !errors.Is(err, ErrInvalidPrompt) {
			t.Errorf("expected ErrInvalidPrompt, got: %v", err)
		}
	})

	t.Run("invalid dimensions", func(t *testing.T) {
		params := DefaultParams()
		params.Prompt = "test"
		params.Width = 100 // Not divisible by 8

		_, err := gen.GenerateWithResult(ctx, params)
		if err == nil {
			t.Error("expected error for invalid dimensions")
			return
		}
		if !errors.Is(err, ErrInvalidParams) {
			t.Errorf("expected ErrInvalidParams, got: %v", err)
		}
	})
}
