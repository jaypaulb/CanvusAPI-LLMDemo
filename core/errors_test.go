package core

import (
	"errors"
	"strings"
	"testing"
)

func TestConfigError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *ConfigError
		contains []string
	}{
		{
			name: "error with action",
			err: &ConfigError{
				Code:    "TEST_CODE",
				Message: "Test message",
				Action:  "Take this action",
			},
			contains: []string{"Test message", "Take this action"},
		},
		{
			name: "error without action",
			err: &ConfigError{
				Code:    "TEST_CODE",
				Message: "Test message only",
				Action:  "",
			},
			contains: []string{"Test message only"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			for _, s := range tt.contains {
				if !strings.Contains(errStr, s) {
					t.Errorf("ConfigError.Error() = %q, expected to contain %q", errStr, s)
				}
			}
		})
	}
}

func TestErrEnvFileMissing(t *testing.T) {
	err := ErrEnvFileMissing(".env")
	if err.Code != ErrCodeEnvFileMissing {
		t.Errorf("Expected code %s, got %s", ErrCodeEnvFileMissing, err.Code)
	}
	if !strings.Contains(err.Message, ".env") {
		t.Errorf("Expected message to contain '.env', got %s", err.Message)
	}
	if !strings.Contains(err.Action, "example.env") {
		t.Errorf("Expected action to mention 'example.env', got %s", err.Action)
	}
}

func TestErrInvalidServerURL(t *testing.T) {
	err := ErrInvalidServerURL("not-a-url", "missing scheme")
	if err.Code != ErrCodeInvalidServerURL {
		t.Errorf("Expected code %s, got %s", ErrCodeInvalidServerURL, err.Code)
	}
	if !strings.Contains(err.Message, "not-a-url") {
		t.Errorf("Expected message to contain URL, got %s", err.Message)
	}
	if !strings.Contains(err.Message, "missing scheme") {
		t.Errorf("Expected message to contain reason, got %s", err.Message)
	}
	if !strings.Contains(err.Action, "CANVUS_SERVER") {
		t.Errorf("Expected action to mention CANVUS_SERVER, got %s", err.Action)
	}
}

func TestErrMissingAuth(t *testing.T) {
	tests := []struct {
		service    string
		expectEnv  string
	}{
		{"canvus", "CANVUS_API_KEY"},
		{"openai", "OPENAI_API_KEY"},
		{"other", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.service, func(t *testing.T) {
			err := ErrMissingAuth(tt.service)
			if err.Code != ErrCodeMissingAuth {
				t.Errorf("Expected code %s, got %s", ErrCodeMissingAuth, err.Code)
			}
			if !strings.Contains(err.Action, tt.expectEnv) {
				t.Errorf("Expected action to mention %s, got %s", tt.expectEnv, err.Action)
			}
		})
	}
}

func TestErrServerUnreachable(t *testing.T) {
	err := ErrServerUnreachable("https://example.com", "connection refused")
	if err.Code != ErrCodeServerUnreachable {
		t.Errorf("Expected code %s, got %s", ErrCodeServerUnreachable, err.Code)
	}
	if !strings.Contains(err.Message, "example.com") {
		t.Errorf("Expected message to contain URL, got %s", err.Message)
	}
	if !strings.Contains(err.Action, "ALLOW_SELF_SIGNED_CERTS") {
		t.Errorf("Expected action to mention ALLOW_SELF_SIGNED_CERTS, got %s", err.Action)
	}
}

func TestErrAuthFailed(t *testing.T) {
	err := ErrAuthFailed("canvus", "invalid token")
	if err.Code != ErrCodeAuthFailed {
		t.Errorf("Expected code %s, got %s", ErrCodeAuthFailed, err.Code)
	}
	if !strings.Contains(err.Message, "canvus") {
		t.Errorf("Expected message to contain service, got %s", err.Message)
	}
	if !strings.Contains(err.Message, "invalid token") {
		t.Errorf("Expected message to contain reason, got %s", err.Message)
	}
}

func TestErrCanvasNotFound(t *testing.T) {
	err := ErrCanvasNotFound("canvas-123")
	if err.Code != ErrCodeCanvasNotFound {
		t.Errorf("Expected code %s, got %s", ErrCodeCanvasNotFound, err.Code)
	}
	if !strings.Contains(err.Message, "canvas-123") {
		t.Errorf("Expected message to contain canvas ID, got %s", err.Message)
	}
	if !strings.Contains(err.Action, "CANVAS_ID") {
		t.Errorf("Expected action to mention CANVAS_ID, got %s", err.Action)
	}
}

func TestErrInvalidCanvasID(t *testing.T) {
	err := ErrInvalidCanvasID("bad-id")
	if err.Code != ErrCodeInvalidCanvasID {
		t.Errorf("Expected code %s, got %s", ErrCodeInvalidCanvasID, err.Code)
	}
	if !strings.Contains(err.Message, "bad-id") {
		t.Errorf("Expected message to contain canvas ID, got %s", err.Message)
	}
}

func TestErrMissingConfig(t *testing.T) {
	err := ErrMissingConfig("CANVAS_NAME")
	if err.Code != ErrCodeMissingConfig {
		t.Errorf("Expected code %s, got %s", ErrCodeMissingConfig, err.Code)
	}
	if !strings.Contains(err.Message, "CANVAS_NAME") {
		t.Errorf("Expected message to contain var name, got %s", err.Message)
	}
	if !strings.Contains(err.Action, "CANVAS_NAME") {
		t.Errorf("Expected action to contain var name, got %s", err.Action)
	}
}

func TestIsConfigError(t *testing.T) {
	t.Run("returns ConfigError when it is one", func(t *testing.T) {
		configErr := ErrEnvFileMissing(".env")
		result, ok := IsConfigError(configErr)
		if !ok {
			t.Error("Expected IsConfigError to return true for ConfigError")
		}
		if result != configErr {
			t.Error("Expected IsConfigError to return the same ConfigError")
		}
	})

	t.Run("returns false for regular error", func(t *testing.T) {
		regularErr := errors.New("regular error")
		result, ok := IsConfigError(regularErr)
		if ok {
			t.Error("Expected IsConfigError to return false for regular error")
		}
		if result != nil {
			t.Error("Expected nil result for non-ConfigError")
		}
	})

	t.Run("returns false for nil", func(t *testing.T) {
		result, ok := IsConfigError(nil)
		if ok {
			t.Error("Expected IsConfigError to return false for nil")
		}
		if result != nil {
			t.Error("Expected nil result for nil input")
		}
	})
}

func TestGetErrorCode(t *testing.T) {
	t.Run("returns code for ConfigError", func(t *testing.T) {
		err := ErrEnvFileMissing(".env")
		code := GetErrorCode(err)
		if code != ErrCodeEnvFileMissing {
			t.Errorf("Expected code %s, got %s", ErrCodeEnvFileMissing, code)
		}
	})

	t.Run("returns empty for regular error", func(t *testing.T) {
		err := errors.New("regular error")
		code := GetErrorCode(err)
		if code != "" {
			t.Errorf("Expected empty code, got %s", code)
		}
	})

	t.Run("returns empty for nil", func(t *testing.T) {
		code := GetErrorCode(nil)
		if code != "" {
			t.Errorf("Expected empty code, got %s", code)
		}
	})
}
