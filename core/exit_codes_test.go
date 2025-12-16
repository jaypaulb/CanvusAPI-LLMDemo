package core

import (
	"testing"
)

func TestExitCodeConstants(t *testing.T) {
	// Verify expected values match Unix conventions
	tests := []struct {
		name  string
		code  int
		value int
	}{
		{"ExitCodeSuccess", ExitCodeSuccess, 0},
		{"ExitCodeError", ExitCodeError, 1},
		{"ExitCodeSIGINT", ExitCodeSIGINT, 130},
		{"ExitCodeSIGTERM", ExitCodeSIGTERM, 143},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.value {
				t.Errorf("%s = %d, want %d", tt.name, tt.code, tt.value)
			}
		})
	}
}

func TestExitCodeName(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{ExitCodeSuccess, "success"},
		{ExitCodeError, "error"},
		{ExitCodeSIGINT, "interrupted (SIGINT)"},
		{ExitCodeSIGTERM, "terminated (SIGTERM)"},
		{99, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := ExitCodeName(tt.code); got != tt.want {
				t.Errorf("ExitCodeName(%d) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

func TestIsSignalExit(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{ExitCodeSuccess, false},
		{ExitCodeError, false},
		{ExitCodeSIGINT, true},
		{ExitCodeSIGTERM, true},
		{99, false},
	}

	for _, tt := range tests {
		t.Run(ExitCodeName(tt.code), func(t *testing.T) {
			if got := IsSignalExit(tt.code); got != tt.want {
				t.Errorf("IsSignalExit(%d) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}
