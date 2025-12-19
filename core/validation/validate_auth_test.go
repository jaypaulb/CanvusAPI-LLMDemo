package validation

import (
	"strings"
	"testing"
)

func TestValidateAuthCredentials(t *testing.T) {
	tests := []struct {
		name    string
		creds   AuthCredentials
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid API key only",
			creds:   AuthCredentials{APIKey: "sk-1234567890"},
			wantErr: false,
		},
		{
			name:    "valid username and password",
			creds:   AuthCredentials{Username: "user", Password: "pass"},
			wantErr: false,
		},
		{
			name:    "API key takes precedence over username/password",
			creds:   AuthCredentials{APIKey: "sk-1234567890", Username: "user", Password: "pass"},
			wantErr: false,
		},
		{
			name:    "API key valid even without username/password",
			creds:   AuthCredentials{APIKey: "sk-1234567890", Username: "", Password: ""},
			wantErr: false,
		},
		{
			name:    "empty credentials",
			creds:   AuthCredentials{},
			wantErr: true,
			errMsg:  "provide either API key or username/password",
		},
		{
			name:    "whitespace-only API key",
			creds:   AuthCredentials{APIKey: "   "},
			wantErr: true,
			errMsg:  "provide either API key or username/password",
		},
		{
			name:    "username without password",
			creds:   AuthCredentials{Username: "user"},
			wantErr: true,
			errMsg:  "password required",
		},
		{
			name:    "password without username",
			creds:   AuthCredentials{Password: "pass"},
			wantErr: true,
			errMsg:  "username required",
		},
		{
			name:    "whitespace username with password",
			creds:   AuthCredentials{Username: "   ", Password: "pass"},
			wantErr: true,
			errMsg:  "username required",
		},
		{
			name:    "username with whitespace password",
			creds:   AuthCredentials{Username: "user", Password: "   "},
			wantErr: true,
			errMsg:  "password required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAuthCredentials(tt.creds)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateAuthCredentials() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateAuthCredentials() error = %q, expected to contain %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAuthCredentials() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateAPIKey(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid API key",
			apiKey:  "sk-1234567890",
			wantErr: false,
		},
		{
			name:    "valid long API key",
			apiKey:  "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz",
			wantErr: false,
		},
		{
			name:    "empty API key",
			apiKey:  "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "whitespace-only API key",
			apiKey:  "   ",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "too short API key",
			apiKey:  "sk-123",
			wantErr: true,
			errMsg:  "too short",
		},
		{
			name:    "minimum valid length (8 chars)",
			apiKey:  "12345678",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIKey(tt.apiKey)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateAPIKey() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateAPIKey() error = %q, expected to contain %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateAPIKey() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateCanvusAPIKey(t *testing.T) {
	// ValidateCanvusAPIKey should behave the same as ValidateAPIKey
	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
	}{
		{"valid key", "canvus-api-key-12345", false},
		{"empty key", "", true},
		{"short key", "abc", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCanvusAPIKey(tt.apiKey)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateCanvusAPIKey() expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateCanvusAPIKey() unexpected error: %v", err)
			}
		})
	}
}

func TestValidateOpenAIAPIKey(t *testing.T) {
	// ValidateOpenAIAPIKey should validate but not require "sk-" prefix
	// because local LLM services may use different formats
	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
	}{
		{"valid OpenAI key", "sk-1234567890abcdef", false},
		{"valid key without sk- prefix", "local-llm-key-123", false},
		{"empty key", "", true},
		{"short key", "sk-123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOpenAIAPIKey(tt.apiKey)
			if tt.wantErr && err == nil {
				t.Errorf("ValidateOpenAIAPIKey() expected error but got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateOpenAIAPIKey() unexpected error: %v", err)
			}
		})
	}
}
