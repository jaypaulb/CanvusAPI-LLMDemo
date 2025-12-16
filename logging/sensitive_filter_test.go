package logging

import (
	"strings"
	"testing"
)

func TestRedactSensitiveData(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // what the output should NOT contain (the sensitive part)
		hasRedacted bool // whether output should contain [REDACTED]
	}{
		{
			name:        "OpenAI API key",
			input:       "key is sk-proj-abc123def456ghi789jkl012mno345pqr678stu901vwx234yz",
			contains:    "sk-proj",
			hasRedacted: true,
		},
		{
			name:        "Google API key",
			input:       "google key: AIzaSyD-abcdefghijklmnopqrstuvwxyz12345",
			contains:    "AIzaSy",
			hasRedacted: true,
		},
		{
			name:        "GitHub token",
			input:       "token ghp_abcdefghijklmnopqrstuvwxyz1234567890",
			contains:    "ghp_",
			hasRedacted: true,
		},
		{
			name:        "Bearer token",
			input:       "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.abc123",
			contains:    "eyJhbGci",
			hasRedacted: true,
		},
		{
			name:        "password assignment",
			input:       "password=mysecretpassword123",
			contains:    "mysecretpassword",
			hasRedacted: true,
		},
		{
			name:        "api_key assignment",
			input:       "api_key: verysecretkey12345",
			contains:    "verysecretkey",
			hasRedacted: true,
		},
		{
			name:        "no sensitive data",
			input:       "Hello, this is a normal message",
			contains:    "",
			hasRedacted: false,
		},
		{
			name:        "empty string",
			input:       "",
			contains:    "",
			hasRedacted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactSensitiveData(tt.input)

			if tt.hasRedacted {
				if !strings.Contains(result, RedactedPlaceholder) {
					t.Errorf("Expected [REDACTED] in output, got: %s", result)
				}
				if tt.contains != "" && strings.Contains(result, tt.contains) {
					t.Errorf("Sensitive data %q should be redacted, got: %s", tt.contains, result)
				}
			} else {
				if strings.Contains(result, RedactedPlaceholder) {
					t.Errorf("Did not expect [REDACTED] in output, got: %s", result)
				}
				if result != tt.input {
					t.Errorf("Non-sensitive input should be unchanged, got: %s", result)
				}
			}
		})
	}
}

func TestRedactField(t *testing.T) {
	tests := []struct {
		name       string
		fieldName  string
		fieldValue string
		expected   string
	}{
		{
			name:       "OPENAI_API_KEY field",
			fieldName:  "OPENAI_API_KEY",
			fieldValue: "sk-secret123",
			expected:   RedactedPlaceholder,
		},
		{
			name:       "password field lowercase",
			fieldName:  "password",
			fieldValue: "secret123",
			expected:   RedactedPlaceholder,
		},
		{
			name:       "api_key in field name",
			fieldName:  "my_api_key_value",
			fieldValue: "something",
			expected:   RedactedPlaceholder,
		},
		{
			name:       "normal field unchanged",
			fieldName:  "username",
			fieldValue: "john",
			expected:   "john",
		},
		{
			name:       "normal field with sensitive value",
			fieldName:  "message",
			fieldValue: "token=abc123verysecrettoken45678901",
			expected:   RedactedPlaceholder,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactField(tt.fieldName, tt.fieldValue)
			if result != tt.expected {
				t.Errorf("RedactField(%q, %q) = %q, want %q",
					tt.fieldName, tt.fieldValue, result, tt.expected)
			}
		})
	}
}

func TestIsSensitiveField(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		expected  bool
	}{
		{"OPENAI_API_KEY", "OPENAI_API_KEY", true},
		{"lowercase api_key", "api_key", true},
		{"contains PASSWORD", "DB_PASSWORD", true},
		{"contains secret", "client_secret", true},
		{"normal field", "username", false},
		{"normal field 2", "message", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSensitiveField(tt.fieldName)
			if result != tt.expected {
				t.Errorf("IsSensitiveField(%q) = %v, want %v",
					tt.fieldName, result, tt.expected)
			}
		})
	}
}

func TestContainsSensitiveData(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		expected bool
	}{
		{"OpenAI key pattern", "sk-proj-abc123def456ghi789jkl012mno345", true},
		{"Bearer token", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", true},
		{"password assignment", "password: mysecretpassword123", true},
		{"normal text", "Hello world", false},
		{"empty string", "", false},
		{"short string", "hi", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ContainsSensitiveData(tt.value)
			if result != tt.expected {
				t.Errorf("ContainsSensitiveData(%q) = %v, want %v",
					tt.value, result, tt.expected)
			}
		})
	}
}
