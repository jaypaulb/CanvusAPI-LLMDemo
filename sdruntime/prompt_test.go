package sdruntime

import (
	"errors"
	"strings"
	"testing"
)

func TestValidatePrompt_Valid(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"simple prompt", "a cat sitting on a chair"},
		{"prompt with numbers", "a 3d render of a robot"},
		{"prompt with punctuation", "beautiful sunset, orange sky, peaceful scene"},
		{"single character", "x"},
		{"max length", strings.Repeat("a", MaxPromptLength)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrompt(tt.prompt)
			if err != nil {
				t.Errorf("expected no error for valid prompt, got: %v", err)
			}
		})
	}
}

func TestValidatePrompt_Empty(t *testing.T) {
	tests := []struct {
		name   string
		prompt string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"tabs only", "\t\t"},
		{"newlines only", "\n\n"},
		{"mixed whitespace", "  \t\n  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrompt(tt.prompt)
			if err == nil {
				t.Error("expected error for empty/whitespace prompt")
			}
			if !errors.Is(err, ErrInvalidPrompt) {
				t.Errorf("expected ErrInvalidPrompt, got: %v", err)
			}
		})
	}
}

func TestValidatePrompt_NullBytes(t *testing.T) {
	promptWithNull := "hello\x00world"

	err := ValidatePrompt(promptWithNull)
	if err == nil {
		t.Error("expected error for prompt with null byte")
	}
	if !errors.Is(err, ErrInvalidPrompt) {
		t.Errorf("expected ErrInvalidPrompt, got: %v", err)
	}
}

func TestValidatePrompt_TooLong(t *testing.T) {
	longPrompt := strings.Repeat("a", MaxPromptLength+1)

	err := ValidatePrompt(longPrompt)
	if err == nil {
		t.Error("expected error for prompt exceeding max length")
	}
	if !errors.Is(err, ErrInvalidPrompt) {
		t.Errorf("expected ErrInvalidPrompt, got: %v", err)
	}
}

func TestSanitizePrompt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no change needed", "hello world", "hello world"},
		{"leading whitespace", "  hello world", "hello world"},
		{"trailing whitespace", "hello world  ", "hello world"},
		{"both ends", "  hello world  ", "hello world"},
		{"tabs and newlines", "\t\nhello\t\n", "hello"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizePrompt(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
