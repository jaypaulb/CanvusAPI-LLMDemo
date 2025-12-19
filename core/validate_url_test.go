package core

import (
	"testing"
)

func TestValidateServerURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid https URL",
			url:     "https://example.com",
			wantErr: false,
		},
		{
			name:    "valid http URL",
			url:     "http://example.com",
			wantErr: false,
		},
		{
			name:    "valid URL with port",
			url:     "https://example.com:8080",
			wantErr: false,
		},
		{
			name:    "valid URL with path",
			url:     "https://example.com/api/v1",
			wantErr: false,
		},
		{
			name:    "valid localhost URL",
			url:     "http://localhost:1234",
			wantErr: false,
		},
		{
			name:    "valid IP address URL",
			url:     "http://192.168.1.1:8080",
			wantErr: false,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "whitespace only",
			url:     "   ",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "missing scheme",
			url:     "example.com",
			wantErr: true,
			errMsg:  "must use http or https",
		},
		{
			name:    "invalid scheme ftp",
			url:     "ftp://example.com",
			wantErr: true,
			errMsg:  "must use http or https",
		},
		{
			name:    "invalid scheme file",
			url:     "file:///path/to/file",
			wantErr: true,
			errMsg:  "must use http or https",
		},
		{
			name:    "scheme only",
			url:     "https://",
			wantErr: true,
			errMsg:  "must include a host",
		},
		{
			name:    "URL with whitespace gets trimmed",
			url:     "  https://example.com  ",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateServerURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateServerURL(%q) expected error but got nil", tt.url)
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateServerURL(%q) error = %q, expected to contain %q", tt.url, err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateServerURL(%q) unexpected error: %v", tt.url, err)
				}
			}
		})
	}
}

// contains checks if s contains substr (case-insensitive would be better but this is simple)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
