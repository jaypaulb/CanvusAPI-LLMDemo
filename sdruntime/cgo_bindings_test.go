package sdruntime

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadModel_FileNotFound(t *testing.T) {
	// DOING: Test that LoadModel returns ErrModelNotFound for non-existent path
	// EXPECT: Error wrapping ErrModelNotFound
	// IF YES: Test passes, error handling works correctly
	// IF NO: Error type is wrong or no error returned

	_, err := LoadModel("/nonexistent/path/to/model.safetensors")

	if err == nil {
		t.Fatal("expected error for non-existent model path, got nil")
	}

	if !errors.Is(err, ErrModelNotFound) {
		t.Errorf("expected error to wrap ErrModelNotFound, got: %v", err)
	}
}

func TestLoadModel_ValidPath(t *testing.T) {
	// DOING: Test that LoadModel succeeds with a valid file path
	// EXPECT: In stub mode, returns valid context. In sd mode, returns error (lib not integrated)
	// IF YES: Test passes
	// IF NO: Unexpected behavior

	// Create a temporary file to use as a fake model
	tmpDir := t.TempDir()
	fakeModelPath := filepath.Join(tmpDir, "fake_model.safetensors")

	if err := os.WriteFile(fakeModelPath, []byte("fake model data"), 0644); err != nil {
		t.Fatalf("failed to create fake model file: %v", err)
	}

	ctx, err := LoadModel(fakeModelPath)

	// In stub mode, this should succeed
	// In sd mode without library, this returns an error
	backendInfo := GetBackendInfo()

	if backendInfo == "stub (no stable-diffusion.cpp library linked)" {
		// Stub mode - should succeed
		if err != nil {
			t.Fatalf("stub mode: expected success, got error: %v", err)
		}
		if ctx == nil {
			t.Fatal("stub mode: expected non-nil context")
		}
		if !ctx.IsValid() {
			t.Error("stub mode: expected context to be valid")
		}
		if ctx.ModelPath() != fakeModelPath {
			t.Errorf("stub mode: expected model path %s, got %s", fakeModelPath, ctx.ModelPath())
		}

		// Clean up
		FreeContext(ctx)
		if ctx.IsValid() {
			t.Error("stub mode: expected context to be invalid after FreeContext")
		}
	} else {
		// SD mode - may return error if library not fully integrated
		// This is expected until stable-diffusion.cpp is available
		if err != nil && !errors.Is(err, ErrModelLoadFailed) {
			t.Errorf("sd mode: expected ErrModelLoadFailed or success, got: %v", err)
		}
	}
}

func TestGenerateImage_InvalidContext(t *testing.T) {
	// DOING: Test that GenerateImage fails with nil context
	// EXPECT: Error returned
	// IF YES: Proper error handling for nil context
	// IF NO: Missing nil check

	params := GenerateParams{
		Prompt:   "test prompt",
		Width:    512,
		Height:   512,
		Steps:    20,
		CFGScale: 7.5,
		Seed:     42,
	}

	_, err := GenerateImage(nil, params)
	if err == nil {
		t.Error("expected error for nil context, got nil")
	}
}

func TestGenerateImage_InvalidParams(t *testing.T) {
	// DOING: Test that GenerateImage validates parameters before calling implementation
	// EXPECT: ErrInvalidParams for bad width/height/steps
	// IF YES: Parameter validation working correctly
	// IF NO: Validation not being applied

	// Create a stub context for testing
	tmpDir := t.TempDir()
	fakeModelPath := filepath.Join(tmpDir, "fake_model.safetensors")
	if err := os.WriteFile(fakeModelPath, []byte("fake"), 0644); err != nil {
		t.Fatalf("failed to create fake model file: %v", err)
	}

	ctx, err := LoadModel(fakeModelPath)

	// Only run validation tests if we got a valid context (stub mode)
	if err != nil {
		t.Skipf("skipping parameter validation test - LoadModel returned error: %v", err)
	}
	defer FreeContext(ctx)

	tests := []struct {
		name   string
		params GenerateParams
	}{
		{
			name: "empty prompt",
			params: GenerateParams{
				Prompt:   "",
				Width:    512,
				Height:   512,
				Steps:    20,
				CFGScale: 7.5,
			},
		},
		{
			name: "width too small",
			params: GenerateParams{
				Prompt:   "test",
				Width:    64,
				Height:   512,
				Steps:    20,
				CFGScale: 7.5,
			},
		},
		{
			name: "height not divisible by 8",
			params: GenerateParams{
				Prompt:   "test",
				Width:    512,
				Height:   513,
				Steps:    20,
				CFGScale: 7.5,
			},
		},
		{
			name: "steps too high",
			params: GenerateParams{
				Prompt:   "test",
				Width:    512,
				Height:   512,
				Steps:    200,
				CFGScale: 7.5,
			},
		},
		{
			name: "cfg scale too low",
			params: GenerateParams{
				Prompt:   "test",
				Width:    512,
				Height:   512,
				Steps:    20,
				CFGScale: 0.5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GenerateImage(ctx, tt.params)
			if err == nil {
				t.Error("expected error for invalid params, got nil")
			}
			if !errors.Is(err, ErrInvalidParams) && !errors.Is(err, ErrInvalidPrompt) {
				t.Errorf("expected ErrInvalidParams or ErrInvalidPrompt, got: %v", err)
			}
		})
	}
}

func TestFreeContext_NilSafe(t *testing.T) {
	// DOING: Test that FreeContext handles nil safely
	// EXPECT: No panic
	// IF YES: Nil safety working
	// IF NO: Will panic

	// This should not panic
	FreeContext(nil)

	// Also test double-free safety
	tmpDir := t.TempDir()
	fakeModelPath := filepath.Join(tmpDir, "fake_model.safetensors")
	if err := os.WriteFile(fakeModelPath, []byte("fake"), 0644); err != nil {
		t.Fatalf("failed to create fake model file: %v", err)
	}

	ctx, err := LoadModel(fakeModelPath)
	if err != nil {
		t.Skipf("skipping double-free test - LoadModel returned error: %v", err)
	}

	// First free
	FreeContext(ctx)
	// Second free should not panic
	FreeContext(ctx)
}

func TestGetBackendInfo(t *testing.T) {
	// DOING: Test that GetBackendInfo returns a non-empty string
	// EXPECT: Non-empty string describing backend
	// IF YES: Backend info function working
	// IF NO: Returns empty string

	info := GetBackendInfo()
	if info == "" {
		t.Error("expected non-empty backend info string")
	}
	t.Logf("Backend info: %s", info)
}

func TestSDContext_Methods(t *testing.T) {
	// DOING: Test SDContext helper methods
	// EXPECT: Correct behavior for IsValid and ModelPath
	// IF YES: Methods work correctly
	// IF NO: Broken method implementations

	// Test nil context
	var nilCtx *SDContext
	if nilCtx.IsValid() {
		t.Error("nil context should not be valid")
	}
	if nilCtx.ModelPath() != "" {
		t.Error("nil context should return empty model path")
	}

	// Test zero-value context
	zeroCtx := &SDContext{}
	if zeroCtx.IsValid() {
		t.Error("zero-value context should not be valid")
	}
}
