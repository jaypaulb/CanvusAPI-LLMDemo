// Package llamaruntime tests for CGo bindings.
//
// These tests work with both the real CGo bindings and stub implementations.
// Use -tags nocgo to run with stubs.
package llamaruntime

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLlamaInit(t *testing.T) {
	// llamaInit should be safe to call multiple times
	llamaInit()
	llamaInit()
	llamaInit()

	// Verify backend is initialized
	if !llamaBackendInit {
		t.Error("llamaBackendInit should be true after llamaInit()")
	}
}

func TestLoadModelNilPath(t *testing.T) {
	llamaInit()

	// Empty path should fail
	_, err := loadModel("", -1, true, false)
	if err == nil {
		t.Error("loadModel with empty path should fail")
	}

	// Check error type
	var llamaErr *LlamaError
	if errors.As(err, &llamaErr) {
		if llamaErr.Op != "loadModel" {
			t.Errorf("expected Op='loadModel', got '%s'", llamaErr.Op)
		}
	}
}

func TestLoadModelValidPath(t *testing.T) {
	llamaInit()

	// In stub mode, any non-empty path should work
	// In real mode, this will fail if the file doesn't exist
	model, err := loadModel("/tmp/test-model.gguf", -1, true, false)

	// In stub mode, this should succeed
	// In real mode with no actual model, it will fail
	if hasCUDA() {
		// Real mode - expect failure with non-existent file
		if err == nil {
			t.Log("Model loaded (unexpected in test without real model file)")
			model.Close()
		}
	} else {
		// Stub mode - should succeed
		if err != nil {
			t.Errorf("stub loadModel should succeed with non-empty path: %v", err)
		}
		if model != nil {
			model.Close()
		}
	}
}

func TestModelMethods(t *testing.T) {
	llamaInit()

	model, err := loadModel("/tmp/test-model.gguf", -1, true, false)
	if err != nil && !hasCUDA() {
		t.Fatalf("stub loadModel failed: %v", err)
	}
	if model == nil {
		t.Skip("Skipping model method tests - no model loaded")
	}
	defer model.Close()

	// Test model info methods
	vocabSize := model.VocabSize()
	if vocabSize <= 0 {
		t.Errorf("VocabSize should be positive, got %d", vocabSize)
	}

	contextSize := model.ContextTrainSize()
	if contextSize <= 0 {
		t.Errorf("ContextTrainSize should be positive, got %d", contextSize)
	}

	embeddingSize := model.EmbeddingSize()
	if embeddingSize <= 0 {
		t.Errorf("EmbeddingSize should be positive, got %d", embeddingSize)
	}

	bosToken := model.BOSToken()
	t.Logf("BOS token: %d", bosToken)

	eosToken := model.EOSToken()
	t.Logf("EOS token: %d", eosToken)
}

func TestCreateContext(t *testing.T) {
	llamaInit()

	// Test with nil model
	_, err := createContext(nil, 4096, 512, 4)
	if err == nil {
		t.Error("createContext with nil model should fail")
	}

	// Load model for context creation
	model, err := loadModel("/tmp/test-model.gguf", -1, true, false)
	if err != nil && !hasCUDA() {
		t.Fatalf("stub loadModel failed: %v", err)
	}
	if model == nil {
		t.Skip("Skipping context tests - no model loaded")
	}
	defer model.Close()

	// Create context
	ctx, err := createContext(model, 4096, 512, 4)
	if err != nil {
		t.Fatalf("createContext failed: %v", err)
	}
	defer ctx.Close()

	// Test context size
	if ctx.ContextSize() != 4096 {
		t.Errorf("ContextSize expected 4096, got %d", ctx.ContextSize())
	}

	// Test ClearKVCache (should not panic)
	ctx.ClearKVCache()
}

func TestSamplingParams(t *testing.T) {
	params := DefaultSamplingParams()

	if params.Temperature != DefaultTemperature {
		t.Errorf("expected Temperature=%f, got %f", DefaultTemperature, params.Temperature)
	}

	if params.TopK != DefaultTopK {
		t.Errorf("expected TopK=%d, got %d", DefaultTopK, params.TopK)
	}

	if params.TopP != DefaultTopP {
		t.Errorf("expected TopP=%f, got %f", DefaultTopP, params.TopP)
	}

	if params.RepeatPenalty != DefaultRepeatPenalty {
		t.Errorf("expected RepeatPenalty=%f, got %f", DefaultRepeatPenalty, params.RepeatPenalty)
	}
}

func TestInferTextNilContext(t *testing.T) {
	ctx := context.Background()
	params := DefaultSamplingParams()

	_, err := inferText(ctx, nil, "Hello", 10, params)
	if err == nil {
		t.Error("inferText with nil context should fail")
	}

	var llamaErr *LlamaError
	if errors.As(err, &llamaErr) {
		if llamaErr.Op != "inferText" {
			t.Errorf("expected Op='inferText', got '%s'", llamaErr.Op)
		}
	}
}

func TestInferText(t *testing.T) {
	llamaInit()

	model, err := loadModel("/tmp/test-model.gguf", -1, true, false)
	if err != nil && !hasCUDA() {
		t.Fatalf("stub loadModel failed: %v", err)
	}
	if model == nil {
		t.Skip("Skipping inference tests - no model loaded")
	}
	defer model.Close()

	llamaCtx, err := createContext(model, 4096, 512, 4)
	if err != nil {
		t.Fatalf("createContext failed: %v", err)
	}
	defer llamaCtx.Close()

	ctx := context.Background()
	params := DefaultSamplingParams()

	response, err := inferText(ctx, llamaCtx, "Hello, how are you?", 20, params)
	if err != nil {
		t.Fatalf("inferText failed: %v", err)
	}

	if response == "" {
		t.Error("inferText returned empty response")
	}

	t.Logf("Response: %s", response)
}

func TestInferTextWithTimeout(t *testing.T) {
	llamaInit()

	model, err := loadModel("/tmp/test-model.gguf", -1, true, false)
	if err != nil && !hasCUDA() {
		t.Fatalf("stub loadModel failed: %v", err)
	}
	if model == nil {
		t.Skip("Skipping timeout tests - no model loaded")
	}
	defer model.Close()

	llamaCtx, err := createContext(model, 4096, 512, 4)
	if err != nil {
		t.Fatalf("createContext failed: %v", err)
	}
	defer llamaCtx.Close()

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for the timeout to expire
	time.Sleep(10 * time.Millisecond)

	params := DefaultSamplingParams()
	_, err = inferText(ctx, llamaCtx, "Hello", 100, params)

	// In stub mode, this might complete before timeout check
	// In real mode, it should timeout
	if err != nil {
		if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			t.Logf("Got error (expected timeout): %v", err)
		}
	}
}

func TestInferVisionNotImplemented(t *testing.T) {
	llamaInit()

	model, err := loadModel("/tmp/test-model.gguf", -1, true, false)
	if err != nil && !hasCUDA() {
		t.Fatalf("stub loadModel failed: %v", err)
	}
	if model == nil {
		t.Skip("Skipping vision tests - no model loaded")
	}
	defer model.Close()

	llamaCtx, err := createContext(model, 4096, 512, 4)
	if err != nil {
		t.Fatalf("createContext failed: %v", err)
	}
	defer llamaCtx.Close()

	ctx := context.Background()
	params := DefaultSamplingParams()

	// Vision should return an error (not implemented yet)
	_, err = inferVision(ctx, llamaCtx, "Describe this image", []byte{0x89, 0x50, 0x4E, 0x47}, 100, params)
	if err == nil {
		t.Error("inferVision should return error (not implemented)")
	}

	t.Logf("Expected error: %v", err)
}

func TestGetGPUMemory(t *testing.T) {
	llamaInit()

	memInfo, err := getGPUMemory()

	if hasCUDA() {
		// In real mode with CUDA, should return actual memory info
		if err != nil {
			t.Logf("GPU memory query returned error (may be expected): %v", err)
		} else {
			t.Logf("GPU Memory: Used=%d Total=%d Free=%d", memInfo.Used, memInfo.Total, memInfo.Free)
		}
	} else {
		// In stub mode, should return mock data
		if err != nil {
			t.Errorf("stub getGPUMemory should not error: %v", err)
		}
		if memInfo == nil {
			t.Error("stub getGPUMemory should return non-nil")
		} else {
			t.Logf("Stub GPU Memory: Used=%d Total=%d Free=%d", memInfo.Used, memInfo.Total, memInfo.Free)
		}
	}
}

func TestHasCUDA(t *testing.T) {
	result := hasCUDA()
	t.Logf("hasCUDA() = %v", result)

	// In stub mode, should return false
	// In real mode, depends on build
	if !result {
		t.Log("Running in stub mode (no CUDA)")
	} else {
		t.Log("Running with CUDA support")
	}
}

func TestFreeContext(t *testing.T) {
	// freeContext should handle nil gracefully
	freeContext(nil)

	llamaInit()

	model, err := loadModel("/tmp/test-model.gguf", -1, true, false)
	if err != nil && !hasCUDA() {
		t.Fatalf("stub loadModel failed: %v", err)
	}
	if model == nil {
		t.Skip("Skipping free tests - no model loaded")
	}
	defer model.Close()

	llamaCtx, err := createContext(model, 4096, 512, 4)
	if err != nil {
		t.Fatalf("createContext failed: %v", err)
	}

	// freeContext should work
	freeContext(llamaCtx)

	// freeContext should be safe to call again
	freeContext(llamaCtx)
}

func TestFreeModel(t *testing.T) {
	// freeModel should handle nil gracefully
	freeModel(nil)

	llamaInit()

	model, err := loadModel("/tmp/test-model.gguf", -1, true, false)
	if err != nil && !hasCUDA() {
		t.Fatalf("stub loadModel failed: %v", err)
	}
	if model == nil {
		t.Skip("Skipping free tests - no model loaded")
	}

	// freeModel should work
	freeModel(model)

	// freeModel should be safe to call again
	freeModel(model)
}

func TestLlamaBackendFree(t *testing.T) {
	llamaInit()

	// Should be safe to call
	llamaBackendFree()

	// Should be able to reinitialize
	llamaInit()

	if !llamaBackendInit {
		t.Error("llamaBackendInit should be true after reinit")
	}
}

// Benchmark tests
func BenchmarkLoadModel(b *testing.B) {
	llamaInit()

	for i := 0; i < b.N; i++ {
		model, err := loadModel("/tmp/test-model.gguf", -1, true, false)
		if err == nil {
			model.Close()
		}
	}
}

func BenchmarkCreateContext(b *testing.B) {
	llamaInit()

	model, err := loadModel("/tmp/test-model.gguf", -1, true, false)
	if err != nil {
		b.Skip("No model available for benchmark")
	}
	defer model.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, err := createContext(model, 4096, 512, 4)
		if err == nil {
			ctx.Close()
		}
	}
}

func BenchmarkInferText(b *testing.B) {
	llamaInit()

	model, err := loadModel("/tmp/test-model.gguf", -1, true, false)
	if err != nil {
		b.Skip("No model available for benchmark")
	}
	defer model.Close()

	llamaCtx, err := createContext(model, 4096, 512, 4)
	if err != nil {
		b.Skip("Could not create context for benchmark")
	}
	defer llamaCtx.Close()

	ctx := context.Background()
	params := DefaultSamplingParams()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = inferText(ctx, llamaCtx, "Hello", 10, params)
	}
}
