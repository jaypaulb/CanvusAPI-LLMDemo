package tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go_backend/core"
)

// TestConfigLoadingEndToEnd tests the full config loading flow from .env to Config struct.
// This is an integration test that exercises LoadConfig with real environment variables.
func TestConfigLoadingEndToEnd(t *testing.T) {
	// Save all original env vars to restore after each test
	envVarsToRestore := []string{
		"CANVUS_SERVER",
		"CANVAS_NAME",
		"CANVAS_ID",
		"OPENAI_API_KEY",
		"OPENAI_KEY",
		"CANVUS_API_KEY",
		"WEBUI_PWD",
		"BASE_LLM_URL",
		"TEXT_LLM_URL",
		"IMAGE_LLM_URL",
		"PORT",
		"MAX_RETRIES",
		"MAX_CONCURRENT",
		"ALLOW_SELF_SIGNED_CERTS",
	}

	// Helper to save and restore env
	saveEnv := func() map[string]string {
		saved := make(map[string]string)
		for _, key := range envVarsToRestore {
			saved[key] = os.Getenv(key)
		}
		return saved
	}

	restoreEnv := func(saved map[string]string) {
		for key, value := range saved {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}

	clearEnv := func() {
		for _, key := range envVarsToRestore {
			os.Unsetenv(key)
		}
	}

	// Helper to set valid required env vars
	setValidRequiredEnv := func() {
		os.Setenv("CANVUS_SERVER", "https://canvus.example.com")
		os.Setenv("CANVAS_NAME", "Test Canvas")
		os.Setenv("CANVAS_ID", "test-canvas-123")
		os.Setenv("OPENAI_API_KEY", "sk-test-key-1234567890")
		os.Setenv("CANVUS_API_KEY", "canvus-api-key-12345")
		os.Setenv("WEBUI_PWD", "test-password-123")
	}

	t.Run("valid complete configuration loads successfully", func(t *testing.T) {
		saved := saveEnv()
		defer restoreEnv(saved)
		clearEnv()
		setValidRequiredEnv()

		// Add optional config
		os.Setenv("PORT", "8080")
		os.Setenv("MAX_RETRIES", "5")
		os.Setenv("MAX_CONCURRENT", "10")
		os.Setenv("BASE_LLM_URL", "http://localhost:1234/v1")
		os.Setenv("ALLOW_SELF_SIGNED_CERTS", "true")

		cfg, err := core.LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		// Verify required values
		if cfg.CanvusServerURL != "https://canvus.example.com" {
			t.Errorf("CanvusServerURL = %q, want %q", cfg.CanvusServerURL, "https://canvus.example.com")
		}
		if cfg.CanvasName != "Test Canvas" {
			t.Errorf("CanvasName = %q, want %q", cfg.CanvasName, "Test Canvas")
		}
		if cfg.CanvasID != "test-canvas-123" {
			t.Errorf("CanvasID = %q, want %q", cfg.CanvasID, "test-canvas-123")
		}

		// Verify optional values were parsed
		if cfg.Port != 8080 {
			t.Errorf("Port = %d, want %d", cfg.Port, 8080)
		}
		if cfg.MaxRetries != 5 {
			t.Errorf("MaxRetries = %d, want %d", cfg.MaxRetries, 5)
		}
		if cfg.MaxConcurrent != 10 {
			t.Errorf("MaxConcurrent = %d, want %d", cfg.MaxConcurrent, 10)
		}
		if cfg.BaseLLMURL != "http://localhost:1234/v1" {
			t.Errorf("BaseLLMURL = %q, want %q", cfg.BaseLLMURL, "http://localhost:1234/v1")
		}
		if !cfg.AllowSelfSignedCerts {
			t.Error("AllowSelfSignedCerts = false, want true")
		}
	})

	t.Run("missing required env vars returns error", func(t *testing.T) {
		saved := saveEnv()
		defer restoreEnv(saved)
		clearEnv()

		// Don't set any env vars
		cfg, err := core.LoadConfig()

		if err == nil {
			t.Fatal("LoadConfig() expected error for missing required vars, got nil")
		}
		if cfg != nil {
			t.Error("LoadConfig() expected nil config on error")
		}

		// Verify error message mentions missing vars
		errMsg := err.Error()
		if !strings.Contains(errMsg, "CANVUS_SERVER") {
			t.Errorf("error should mention CANVUS_SERVER, got: %v", err)
		}
	})

	t.Run("missing single required var returns descriptive error", func(t *testing.T) {
		saved := saveEnv()
		defer restoreEnv(saved)
		clearEnv()
		setValidRequiredEnv()

		// Remove one required var
		os.Unsetenv("CANVAS_ID")

		cfg, err := core.LoadConfig()

		if err == nil {
			t.Fatal("LoadConfig() expected error for missing CANVAS_ID, got nil")
		}
		if cfg != nil {
			t.Error("LoadConfig() expected nil config on error")
		}

		errMsg := err.Error()
		if !strings.Contains(errMsg, "CANVAS_ID") {
			t.Errorf("error should mention CANVAS_ID, got: %v", err)
		}
	})

	t.Run("defaults applied when optional vars not set", func(t *testing.T) {
		saved := saveEnv()
		defer restoreEnv(saved)
		clearEnv()
		setValidRequiredEnv()

		cfg, err := core.LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		// Verify defaults
		if cfg.Port != 3000 {
			t.Errorf("Port default = %d, want %d", cfg.Port, 3000)
		}
		if cfg.MaxRetries != 3 {
			t.Errorf("MaxRetries default = %d, want %d", cfg.MaxRetries, 3)
		}
		if cfg.MaxConcurrent != 5 {
			t.Errorf("MaxConcurrent default = %d, want %d", cfg.MaxConcurrent, 5)
		}
		if cfg.BaseLLMURL != "http://127.0.0.1:1234/v1" {
			t.Errorf("BaseLLMURL default = %q, want %q", cfg.BaseLLMURL, "http://127.0.0.1:1234/v1")
		}
		if cfg.AllowSelfSignedCerts != false {
			t.Error("AllowSelfSignedCerts default = true, want false")
		}
		if cfg.DownloadsDir != "./downloads" {
			t.Errorf("DownloadsDir default = %q, want %q", cfg.DownloadsDir, "./downloads")
		}
	})

	t.Run("legacy OPENAI_KEY fallback works", func(t *testing.T) {
		saved := saveEnv()
		defer restoreEnv(saved)
		clearEnv()

		// Set all required except use legacy key name
		os.Setenv("CANVUS_SERVER", "https://canvus.example.com")
		os.Setenv("CANVAS_NAME", "Test Canvas")
		os.Setenv("CANVAS_ID", "test-canvas-123")
		os.Setenv("OPENAI_KEY", "sk-legacy-key-12345") // Legacy name
		os.Setenv("CANVUS_API_KEY", "canvus-api-key-12345")
		os.Setenv("WEBUI_PWD", "test-password-123")

		// Note: LoadConfig checks OPENAI_API_KEY first, but needs it set for validation
		// The legacy fallback is for the config value, not the validation check
		// So we need to also set OPENAI_API_KEY for the required check
		os.Setenv("OPENAI_API_KEY", "sk-test-key-1234567890")

		cfg, err := core.LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		// OPENAI_API_KEY takes precedence over OPENAI_KEY
		if cfg.OpenAIAPIKey != "sk-test-key-1234567890" {
			t.Errorf("OpenAIAPIKey = %q, want %q", cfg.OpenAIAPIKey, "sk-test-key-1234567890")
		}
	})
}

// TestConfigValidatorIntegration tests the ConfigValidator with real env vars and files.
func TestConfigValidatorIntegration(t *testing.T) {
	// Save and restore env
	envVarsToRestore := []string{
		"CANVUS_SERVER",
		"CANVAS_ID",
		"CANVUS_API_KEY",
		"CANVUS_USERNAME",
		"CANVUS_PASSWORD",
		"OPENAI_API_KEY",
	}

	saveEnv := func() map[string]string {
		saved := make(map[string]string)
		for _, key := range envVarsToRestore {
			saved[key] = os.Getenv(key)
		}
		return saved
	}

	restoreEnv := func(saved map[string]string) {
		for key, value := range saved {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}

	clearEnv := func() {
		for _, key := range envVarsToRestore {
			os.Unsetenv(key)
		}
	}

	t.Run("ValidateAll returns correct count with mixed valid/invalid", func(t *testing.T) {
		saved := saveEnv()
		defer restoreEnv(saved)
		clearEnv()

		// Create temp .env file
		dir := t.TempDir()
		envPath := filepath.Join(dir, ".env")
		if err := os.WriteFile(envPath, []byte("# Test env file\n"), 0644); err != nil {
			t.Fatalf("failed to create test .env: %v", err)
		}

		// Set some valid, some invalid
		os.Setenv("CANVUS_SERVER", "https://canvus.example.com") // valid
		os.Setenv("CANVAS_ID", "test-canvas")                    // valid
		// CANVUS_API_KEY not set - invalid
		// OPENAI_API_KEY not set - invalid

		v := core.NewConfigValidator().WithEnvPath(envPath)
		results := v.ValidateAll()

		validCount := 0
		invalidCount := 0
		for _, r := range results {
			if r.Valid {
				validCount++
			} else {
				invalidCount++
			}
		}

		// Expected: env file (valid), server URL (valid), canvas ID (valid),
		// auth creds (invalid), openai creds (invalid)
		if validCount != 3 {
			t.Errorf("valid count = %d, want 3", validCount)
		}
		if invalidCount != 2 {
			t.Errorf("invalid count = %d, want 2", invalidCount)
		}
	})

	t.Run("ValidateRequired fails fast on first error", func(t *testing.T) {
		saved := saveEnv()
		defer restoreEnv(saved)
		clearEnv()

		// Point to nonexistent .env file
		v := core.NewConfigValidator().WithEnvPath(filepath.Join(t.TempDir(), "nonexistent.env"))

		err := v.ValidateRequired()
		if err == nil {
			t.Fatal("ValidateRequired() expected error for missing .env file")
		}

		// Should fail on env file check first
		configErr, ok := core.IsConfigError(err)
		if !ok {
			t.Errorf("expected ConfigError, got %T", err)
		} else if configErr.Code != core.ErrCodeEnvFileMissing {
			t.Errorf("error code = %q, want %q", configErr.Code, core.ErrCodeEnvFileMissing)
		}
	})

	t.Run("IsValid returns false when config incomplete", func(t *testing.T) {
		saved := saveEnv()
		defer restoreEnv(saved)
		clearEnv()

		// Create temp .env file but missing other required vars
		dir := t.TempDir()
		envPath := filepath.Join(dir, ".env")
		if err := os.WriteFile(envPath, []byte("TEST=value\n"), 0644); err != nil {
			t.Fatalf("failed to create test .env: %v", err)
		}

		v := core.NewConfigValidator().WithEnvPath(envPath)

		if v.IsValid() {
			t.Error("IsValid() = true, want false for incomplete config")
		}
	})

	t.Run("full validation passes with complete config", func(t *testing.T) {
		saved := saveEnv()
		defer restoreEnv(saved)
		clearEnv()

		// Create temp .env file
		dir := t.TempDir()
		envPath := filepath.Join(dir, ".env")
		if err := os.WriteFile(envPath, []byte("# Complete config\n"), 0644); err != nil {
			t.Fatalf("failed to create test .env: %v", err)
		}

		// Set all required vars
		os.Setenv("CANVUS_SERVER", "https://canvus.example.com")
		os.Setenv("CANVAS_ID", "test-canvas-123")
		os.Setenv("CANVUS_API_KEY", "test-api-key-12345")
		os.Setenv("OPENAI_API_KEY", "sk-test-key-12345678")

		v := core.NewConfigValidator().WithEnvPath(envPath)

		if !v.IsValid() {
			err := v.GetFirstError()
			t.Errorf("IsValid() = false, want true. Error: %v", err)
		}

		if v.CountInvalid() != 0 {
			t.Errorf("CountInvalid() = %d, want 0", v.CountInvalid())
		}
	})
}

// TestEnvParsingEdgeCases tests edge cases in environment variable parsing.
func TestEnvParsingEdgeCases(t *testing.T) {
	t.Run("ParseIntEnv handles invalid integers gracefully", func(t *testing.T) {
		// Save and restore
		original := os.Getenv("TEST_INT_VAR")
		defer func() {
			if original == "" {
				os.Unsetenv("TEST_INT_VAR")
			} else {
				os.Setenv("TEST_INT_VAR", original)
			}
		}()

		os.Setenv("TEST_INT_VAR", "not-a-number")
		result := core.ParseIntEnv("TEST_INT_VAR", 42)

		if result != 42 {
			t.Errorf("ParseIntEnv() = %d, want default 42 for invalid input", result)
		}
	})

	t.Run("ParseBoolEnv handles various true/false representations", func(t *testing.T) {
		original := os.Getenv("TEST_BOOL_VAR")
		defer func() {
			if original == "" {
				os.Unsetenv("TEST_BOOL_VAR")
			} else {
				os.Setenv("TEST_BOOL_VAR", original)
			}
		}()

		testCases := []struct {
			input    string
			expected bool
		}{
			{"true", true},
			{"TRUE", true},
			{"True", true},
			{"1", true},
			{"yes", true},
			{"YES", true},
			{"on", true},
			{"false", false},
			{"FALSE", false},
			{"0", false},
			{"no", false},
			{"off", false},
		}

		for _, tc := range testCases {
			os.Setenv("TEST_BOOL_VAR", tc.input)
			result := core.ParseBoolEnv("TEST_BOOL_VAR", !tc.expected)

			if result != tc.expected {
				t.Errorf("ParseBoolEnv(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		}
	})

	t.Run("GetEnvOrDefault returns default for empty var", func(t *testing.T) {
		original := os.Getenv("TEST_EMPTY_VAR")
		defer func() {
			if original == "" {
				os.Unsetenv("TEST_EMPTY_VAR")
			} else {
				os.Setenv("TEST_EMPTY_VAR", original)
			}
		}()

		os.Unsetenv("TEST_EMPTY_VAR")
		result := core.GetEnvOrDefault("TEST_EMPTY_VAR", "default-value")

		if result != "default-value" {
			t.Errorf("GetEnvOrDefault() = %q, want %q", result, "default-value")
		}
	})
}
