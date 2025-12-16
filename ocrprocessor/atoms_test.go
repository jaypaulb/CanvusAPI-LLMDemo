package ocrprocessor

import (
	"errors"
	"testing"
)

func TestValidateGoogleAPIKey(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		wantErr   error
		wantNilErr bool
	}{
		{
			name:    "empty string returns ErrEmptyAPIKey",
			apiKey:  "",
			wantErr: ErrEmptyAPIKey,
		},
		{
			name:    "whitespace only returns ErrEmptyAPIKey",
			apiKey:  "   ",
			wantErr: ErrEmptyAPIKey,
		},
		{
			name:    "too short returns ErrAPIKeyTooShort",
			apiKey:  "AIza123",
			wantErr: ErrAPIKeyTooShort,
		},
		{
			name:    "valid Google API key format",
			apiKey:  "AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY",
			wantNilErr: true,
		},
		{
			name:    "valid key with whitespace",
			apiKey:  "  AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY  ",
			wantNilErr: true,
		},
		{
			name:    "non-Google key format but valid length",
			apiKey:  "sk-1234567890abcdefghijklmnopqrstuvwxyz",
			wantNilErr: true, // We accept non-Google keys of valid length
		},
		{
			name:    "AIza prefix with wrong length",
			apiKey:  "AIzaSyD-9tSrke72Pou", // Too short for AIza pattern
			wantErr: ErrAPIKeyTooShort,
		},
		{
			name:    "AIza prefix with correct length but invalid chars",
			apiKey:  "AIza!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!", // Invalid chars
			wantErr: ErrInvalidAPIKeyFormat,
		},
		{
			name:    "way too long key",
			apiKey:  "AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWYxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
			wantErr: ErrAPIKeyTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateGoogleAPIKey(tt.apiKey)
			if tt.wantNilErr {
				if err != nil {
					t.Errorf("ValidateGoogleAPIKey(%q) = %v, want nil", tt.apiKey, err)
				}
			} else if !errors.Is(err, tt.wantErr) {
				t.Errorf("ValidateGoogleAPIKey(%q) = %v, want %v", tt.apiKey, err, tt.wantErr)
			}
		})
	}
}

func TestIsGoogleAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "empty string returns false",
			input:    "",
			expected: false,
		},
		{
			name:     "valid Google API key",
			input:    "AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY",
			expected: true,
		},
		{
			name:     "valid key with whitespace",
			input:    "  AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY  ",
			expected: true,
		},
		{
			name:     "non-Google key format",
			input:    "sk-1234567890abcdef",
			expected: false,
		},
		{
			name:     "wrong prefix",
			input:    "BIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY",
			expected: false,
		},
		{
			name:     "too short",
			input:    "AIzaSyD-9tSrke72Pou",
			expected: false,
		},
		{
			name:     "too long",
			input:    "AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWYextra",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGoogleAPIKey(tt.input)
			if result != tt.expected {
				t.Errorf("IsGoogleAPIKey(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeAPIKey(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantKey     string
		wantErr     bool
	}{
		{
			name:    "empty returns error",
			input:   "",
			wantKey: "",
			wantErr: true,
		},
		{
			name:    "valid key returned trimmed",
			input:   "  AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY  ",
			wantKey: "AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY",
			wantErr: false,
		},
		{
			name:    "too short returns error",
			input:   "short",
			wantKey: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, err := SanitizeAPIKey(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("SanitizeAPIKey(%q) expected error, got nil", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("SanitizeAPIKey(%q) unexpected error: %v", tt.input, err)
			}
			if key != tt.wantKey {
				t.Errorf("SanitizeAPIKey(%q) = %q, want %q", tt.input, key, tt.wantKey)
			}
		})
	}
}

func TestMaskAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty returns [empty]",
			input:    "",
			expected: "[empty]",
		},
		{
			name:     "short key (4 chars) fully masked",
			input:    "abcd",
			expected: "****",
		},
		{
			name:     "medium key (8 chars) partially masked",
			input:    "abcdefgh",
			expected: "abcd****",
		},
		{
			name:     "standard Google key masked properly",
			input:    "AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY",
			expected: "AIzaSyD-****MBWY",
		},
		{
			name:     "12 char boundary case",
			input:    "123456789012",
			expected: "1234********",
		},
		{
			name:     "13 char gets full masking",
			input:    "1234567890123",
			expected: "12345678****0123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskAPIKey(tt.input)
			if result != tt.expected {
				t.Errorf("MaskAPIKey(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkValidateGoogleAPIKey(b *testing.B) {
	apiKey := "AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateGoogleAPIKey(apiKey)
	}
}

func BenchmarkIsGoogleAPIKey(b *testing.B) {
	apiKey := "AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		IsGoogleAPIKey(apiKey)
	}
}

func BenchmarkMaskAPIKey(b *testing.B) {
	apiKey := "AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MaskAPIKey(apiKey)
	}
}
