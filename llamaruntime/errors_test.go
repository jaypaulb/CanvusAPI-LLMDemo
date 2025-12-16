package llamaruntime

import (
	"errors"
	"strings"
	"testing"
)

// TestLlamaError_Error tests the Error() method formatting.
func TestLlamaError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *LlamaError
		contains []string // Strings that should appear in the error message
	}{
		{
			name: "error without wrapped error",
			err: &LlamaError{
				Op:      "loadModel",
				Code:    -1,
				Message: "file not found",
				Err:     nil,
			},
			contains: []string{"llama.cpp", "loadModel", "file not found", "code: -1"},
		},
		{
			name: "error with wrapped error",
			err: &LlamaError{
				Op:      "infer",
				Code:    42,
				Message: "context creation failed",
				Err:     errors.New("out of memory"),
			},
			contains: []string{"llama.cpp", "infer", "context creation failed", "code: 42", "out of memory"},
		},
		{
			name: "error with zero code",
			err: &LlamaError{
				Op:      "healthCheck",
				Code:    0,
				Message: "unexpected success code",
				Err:     nil,
			},
			contains: []string{"llama.cpp", "healthCheck", "unexpected success code", "code: 0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("Error() = %q, should contain %q", got, want)
				}
			}
		})
	}
}

// TestLlamaError_Unwrap tests error unwrapping functionality.
func TestLlamaError_Unwrap(t *testing.T) {
	baseErr := errors.New("base error")
	llamaErr := &LlamaError{
		Op:      "test",
		Code:    1,
		Message: "test error",
		Err:     baseErr,
	}

	unwrapped := llamaErr.Unwrap()
	if unwrapped != baseErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, baseErr)
	}

	// Test with errors.Is
	if !errors.Is(llamaErr, baseErr) {
		t.Errorf("errors.Is() failed to unwrap LlamaError")
	}
}

// TestSentinelErrors verifies all sentinel errors are defined and distinct.
func TestSentinelErrors(t *testing.T) {
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrModelNotFound", ErrModelNotFound},
		{"ErrModelLoadFailed", ErrModelLoadFailed},
		{"ErrContextCreateFailed", ErrContextCreateFailed},
		{"ErrInferenceFailed", ErrInferenceFailed},
		{"ErrGPUNotAvailable", ErrGPUNotAvailable},
		{"ErrInsufficientVRAM", ErrInsufficientVRAM},
		{"ErrInvalidImage", ErrInvalidImage},
		{"ErrTimeout", ErrTimeout},
	}

	// Verify each error is non-nil and has a message
	for _, s := range sentinels {
		t.Run(s.name, func(t *testing.T) {
			if s.err == nil {
				t.Errorf("%s is nil", s.name)
			}
			if s.err.Error() == "" {
				t.Errorf("%s has empty error message", s.name)
			}
		})
	}

	// Verify errors are distinct (different pointers)
	for i, s1 := range sentinels {
		for j, s2 := range sentinels {
			if i != j && s1.err == s2.err {
				t.Errorf("%s and %s are the same error instance", s1.name, s2.name)
			}
		}
	}

	// Verify errors.Is works correctly
	if errors.Is(ErrModelNotFound, ErrModelLoadFailed) {
		t.Errorf("ErrModelNotFound should not be ErrModelLoadFailed")
	}
	if !errors.Is(ErrModelNotFound, ErrModelNotFound) {
		t.Errorf("ErrModelNotFound should be itself")
	}
}
