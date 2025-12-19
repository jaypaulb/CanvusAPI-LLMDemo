package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigValidator_CheckEnvFile(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() string // returns path to env file or temp dir
		cleanup   func(string)
		wantValid bool
	}{
		{
			name: "env file exists",
			setupFunc: func() string {
				dir := t.TempDir()
				path := filepath.Join(dir, ".env")
				if err := os.WriteFile(path, []byte("TEST=value"), 0644); err != nil {
					t.Fatalf("failed to create test file: %v", err)
				}
				return path
			},
			wantValid: true,
		},
		{
			name: "env file missing",
			setupFunc: func() string {
				return filepath.Join(t.TempDir(), "nonexistent.env")
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setupFunc()
			v := NewConfigValidator().WithEnvPath(path)
			result := v.CheckEnvFile()

			if result.Valid != tt.wantValid {
				t.Errorf("CheckEnvFile() Valid = %v, want %v", result.Valid, tt.wantValid)
			}

			if !tt.wantValid && result.Error == nil {
				t.Error("CheckEnvFile() expected error for invalid case")
			}
		})
	}
}

func TestConfigValidator_CheckServerURL(t *testing.T) {
	tests := []struct {
		name      string
		serverURL string
		wantValid bool
	}{
		{
			name:      "valid https URL",
			serverURL: "https://canvus.example.com",
			wantValid: true,
		},
		{
			name:      "valid http URL",
			serverURL: "http://localhost:8080",
			wantValid: true,
		},
		{
			name:      "empty URL",
			serverURL: "",
			wantValid: false,
		},
		{
			name:      "invalid URL - no scheme",
			serverURL: "canvus.example.com",
			wantValid: false,
		},
		{
			name:      "invalid URL - ftp scheme",
			serverURL: "ftp://canvus.example.com",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original env
			original := os.Getenv("CANVUS_SERVER")
			defer os.Setenv("CANVUS_SERVER", original)

			if tt.serverURL != "" {
				os.Setenv("CANVUS_SERVER", tt.serverURL)
			} else {
				os.Unsetenv("CANVUS_SERVER")
			}

			v := NewConfigValidator()
			result := v.CheckServerURL()

			if result.Valid != tt.wantValid {
				t.Errorf("CheckServerURL() Valid = %v, want %v, message: %s", result.Valid, tt.wantValid, result.Message)
			}

			if !tt.wantValid && result.Error == nil {
				t.Error("CheckServerURL() expected error for invalid case")
			}
		})
	}
}

func TestConfigValidator_CheckCanvasID(t *testing.T) {
	tests := []struct {
		name      string
		canvasID  string
		wantValid bool
	}{
		{
			name:      "valid canvas ID",
			canvasID:  "canvas-123-abc",
			wantValid: true,
		},
		{
			name:      "empty canvas ID",
			canvasID:  "",
			wantValid: false,
		},
		{
			name:      "whitespace-only canvas ID",
			canvasID:  "   ",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original env
			original := os.Getenv("CANVAS_ID")
			defer os.Setenv("CANVAS_ID", original)

			if tt.canvasID != "" {
				os.Setenv("CANVAS_ID", tt.canvasID)
			} else {
				os.Unsetenv("CANVAS_ID")
			}

			v := NewConfigValidator()
			result := v.CheckCanvasID()

			if result.Valid != tt.wantValid {
				t.Errorf("CheckCanvasID() Valid = %v, want %v", result.Valid, tt.wantValid)
			}

			if !tt.wantValid && result.Error == nil {
				t.Error("CheckCanvasID() expected error for invalid case")
			}
		})
	}
}

func TestConfigValidator_CheckAuthCredentials(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		username  string
		password  string
		wantValid bool
	}{
		{
			name:      "valid API key",
			apiKey:    "test-api-key-12345",
			wantValid: true,
		},
		{
			name:      "valid username/password",
			username:  "testuser",
			password:  "testpass",
			wantValid: true,
		},
		{
			name:      "API key takes precedence",
			apiKey:    "test-api-key-12345",
			username:  "testuser",
			password:  "testpass",
			wantValid: true,
		},
		{
			name:      "no credentials",
			wantValid: false,
		},
		{
			name:      "only username",
			username:  "testuser",
			wantValid: false,
		},
		{
			name:      "only password",
			password:  "testpass",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env
			origKey := os.Getenv("CANVUS_API_KEY")
			origUser := os.Getenv("CANVUS_USERNAME")
			origPass := os.Getenv("CANVUS_PASSWORD")

			// Clean up and restore after test
			defer func() {
				os.Setenv("CANVUS_API_KEY", origKey)
				os.Setenv("CANVUS_USERNAME", origUser)
				os.Setenv("CANVUS_PASSWORD", origPass)
			}()

			// Set test values
			if tt.apiKey != "" {
				os.Setenv("CANVUS_API_KEY", tt.apiKey)
			} else {
				os.Unsetenv("CANVUS_API_KEY")
			}
			if tt.username != "" {
				os.Setenv("CANVUS_USERNAME", tt.username)
			} else {
				os.Unsetenv("CANVUS_USERNAME")
			}
			if tt.password != "" {
				os.Setenv("CANVUS_PASSWORD", tt.password)
			} else {
				os.Unsetenv("CANVUS_PASSWORD")
			}

			v := NewConfigValidator()
			result := v.CheckAuthCredentials()

			if result.Valid != tt.wantValid {
				t.Errorf("CheckAuthCredentials() Valid = %v, want %v", result.Valid, tt.wantValid)
			}
		})
	}
}

func TestConfigValidator_CheckOpenAICredentials(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		wantValid bool
	}{
		{
			name:      "valid OpenAI key",
			apiKey:    "sk-test1234567890",
			wantValid: true,
		},
		{
			name:      "valid local LLM key",
			apiKey:    "not-needed-but-set",
			wantValid: true,
		},
		{
			name:      "empty key",
			apiKey:    "",
			wantValid: false,
		},
		{
			name:      "too short key",
			apiKey:    "short",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore original env
			original := os.Getenv("OPENAI_API_KEY")
			defer os.Setenv("OPENAI_API_KEY", original)

			if tt.apiKey != "" {
				os.Setenv("OPENAI_API_KEY", tt.apiKey)
			} else {
				os.Unsetenv("OPENAI_API_KEY")
			}

			v := NewConfigValidator()
			result := v.CheckOpenAICredentials()

			if result.Valid != tt.wantValid {
				t.Errorf("CheckOpenAICredentials() Valid = %v, want %v", result.Valid, tt.wantValid)
			}
		})
	}
}

func TestConfigValidator_ValidateAll(t *testing.T) {
	// Setup complete valid config
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("TEST=value"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Save original env
	origServer := os.Getenv("CANVUS_SERVER")
	origCanvas := os.Getenv("CANVAS_ID")
	origKey := os.Getenv("CANVUS_API_KEY")
	origOpenAI := os.Getenv("OPENAI_API_KEY")

	// Restore after test
	defer func() {
		os.Setenv("CANVUS_SERVER", origServer)
		os.Setenv("CANVAS_ID", origCanvas)
		os.Setenv("CANVUS_API_KEY", origKey)
		os.Setenv("OPENAI_API_KEY", origOpenAI)
	}()

	// Set valid config
	os.Setenv("CANVUS_SERVER", "https://canvus.example.com")
	os.Setenv("CANVAS_ID", "test-canvas-123")
	os.Setenv("CANVUS_API_KEY", "test-api-key-12345")
	os.Setenv("OPENAI_API_KEY", "sk-test1234567890")

	v := NewConfigValidator().WithEnvPath(envPath)
	results := v.ValidateAll()

	if len(results) != 5 {
		t.Errorf("ValidateAll() returned %d results, expected 5", len(results))
	}

	// All should be valid
	for i, r := range results {
		if !r.Valid {
			t.Errorf("ValidateAll()[%d] = invalid (%s), expected valid", i, r.Message)
		}
	}
}

func TestConfigValidator_ValidateRequired(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*ConfigValidator)
		wantError bool
	}{
		{
			name: "all required valid",
			setup: func(v *ConfigValidator) {
				dir := t.TempDir()
				envPath := filepath.Join(dir, ".env")
				os.WriteFile(envPath, []byte("TEST=value"), 0644)
				v.WithEnvPath(envPath)
				os.Setenv("CANVUS_SERVER", "https://example.com")
				os.Setenv("CANVAS_ID", "test-canvas")
				os.Setenv("CANVUS_API_KEY", "test-key-12345")
			},
			wantError: false,
		},
		{
			name: "missing env file",
			setup: func(v *ConfigValidator) {
				v.WithEnvPath(filepath.Join(t.TempDir(), "nonexistent.env"))
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env
			origServer := os.Getenv("CANVUS_SERVER")
			origCanvas := os.Getenv("CANVAS_ID")
			origKey := os.Getenv("CANVUS_API_KEY")

			// Clean and restore
			defer func() {
				os.Setenv("CANVUS_SERVER", origServer)
				os.Setenv("CANVAS_ID", origCanvas)
				os.Setenv("CANVUS_API_KEY", origKey)
			}()

			// Clear env first
			os.Unsetenv("CANVUS_SERVER")
			os.Unsetenv("CANVAS_ID")
			os.Unsetenv("CANVUS_API_KEY")

			v := NewConfigValidator()
			tt.setup(v)

			err := v.ValidateRequired()
			if (err != nil) != tt.wantError {
				t.Errorf("ValidateRequired() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestConfigValidator_IsValid(t *testing.T) {
	// Setup valid config
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	os.WriteFile(envPath, []byte("TEST=value"), 0644)

	// Save original env
	origServer := os.Getenv("CANVUS_SERVER")
	origCanvas := os.Getenv("CANVAS_ID")
	origKey := os.Getenv("CANVUS_API_KEY")

	// Restore after test
	defer func() {
		os.Setenv("CANVUS_SERVER", origServer)
		os.Setenv("CANVAS_ID", origCanvas)
		os.Setenv("CANVUS_API_KEY", origKey)
	}()

	os.Setenv("CANVUS_SERVER", "https://example.com")
	os.Setenv("CANVAS_ID", "test-canvas")
	os.Setenv("CANVUS_API_KEY", "test-key-12345")

	v := NewConfigValidator().WithEnvPath(envPath)
	if !v.IsValid() {
		t.Error("IsValid() = false, expected true for valid config")
	}
}

func TestConfigValidator_CountValidAndInvalid(t *testing.T) {
	// Setup partial config - only env file valid
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	os.WriteFile(envPath, []byte("TEST=value"), 0644)

	// Save original env
	origServer := os.Getenv("CANVUS_SERVER")
	origCanvas := os.Getenv("CANVAS_ID")
	origKey := os.Getenv("CANVUS_API_KEY")
	origOpenAI := os.Getenv("OPENAI_API_KEY")

	// Clear all env and restore after
	defer func() {
		os.Setenv("CANVUS_SERVER", origServer)
		os.Setenv("CANVAS_ID", origCanvas)
		os.Setenv("CANVUS_API_KEY", origKey)
		os.Setenv("OPENAI_API_KEY", origOpenAI)
	}()

	os.Unsetenv("CANVUS_SERVER")
	os.Unsetenv("CANVAS_ID")
	os.Unsetenv("CANVUS_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")

	v := NewConfigValidator().WithEnvPath(envPath)
	valid := v.CountValid()
	invalid := v.CountInvalid()

	// Only env file is valid (1), rest are invalid (4)
	if valid != 1 {
		t.Errorf("CountValid() = %d, expected 1", valid)
	}
	if invalid != 4 {
		t.Errorf("CountInvalid() = %d, expected 4", invalid)
	}
}
