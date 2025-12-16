package sdruntime

import (
	"errors"
	"testing"
)

func TestErrorsAreSentinels(t *testing.T) {
	// Test that all errors are distinct sentinel values
	allErrors := []error{
		ErrModelNotFound,
		ErrModelLoadFailed,
		ErrModelCorrupted,
		ErrGenerationFailed,
		ErrGenerationTimeout,
		ErrInvalidPrompt,
		ErrInvalidParams,
		ErrCUDANotAvailable,
		ErrOutOfVRAM,
		ErrContextPoolClosed,
		ErrAcquireTimeout,
	}

	// Verify each error has a non-empty message
	for _, err := range allErrors {
		if err.Error() == "" {
			t.Errorf("error has empty message: %v", err)
		}
	}

	// Verify errors can be matched with errors.Is
	if !errors.Is(ErrModelNotFound, ErrModelNotFound) {
		t.Error("ErrModelNotFound should match itself with errors.Is")
	}
}

func TestErrorsAreDistinct(t *testing.T) {
	// Verify that different errors don't match each other
	if errors.Is(ErrModelNotFound, ErrModelLoadFailed) {
		t.Error("different errors should not match")
	}
	if errors.Is(ErrGenerationFailed, ErrGenerationTimeout) {
		t.Error("different errors should not match")
	}
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{"ErrModelNotFound", ErrModelNotFound, "model file not found"},
		{"ErrModelLoadFailed", ErrModelLoadFailed, "failed to load model"},
		{"ErrModelCorrupted", ErrModelCorrupted, "corrupted"},
		{"ErrGenerationFailed", ErrGenerationFailed, "generation failed"},
		{"ErrGenerationTimeout", ErrGenerationTimeout, "timed out"},
		{"ErrInvalidPrompt", ErrInvalidPrompt, "invalid prompt"},
		{"ErrInvalidParams", ErrInvalidParams, "invalid generation parameters"},
		{"ErrCUDANotAvailable", ErrCUDANotAvailable, "CUDA not available"},
		{"ErrOutOfVRAM", ErrOutOfVRAM, "VRAM"},
		{"ErrContextPoolClosed", ErrContextPoolClosed, "pool is closed"},
		{"ErrAcquireTimeout", ErrAcquireTimeout, "timeout acquiring"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			if len(msg) == 0 {
				t.Errorf("%s has empty message", tt.name)
			}
		})
	}
}
