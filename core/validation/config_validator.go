package validation

import (
	"os"
	"strings"
)

// ValidationResult represents the result of a configuration validation check.
type ValidationResult struct {
	Valid   bool
	Message string
	Error   error
}

// ConfigValidator composes validation atoms to provide comprehensive configuration checking.
// This is a molecule that orchestrates URL validation, file existence, and auth credential checks.
type ConfigValidator struct {
	envPath string // Path to .env file (default: ".env")
}

// NewConfigValidator creates a new ConfigValidator with default settings.
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		envPath: ".env",
	}
}

// WithEnvPath sets a custom path for the .env file.
func (v *ConfigValidator) WithEnvPath(path string) *ConfigValidator {
	v.envPath = path
	return v
}

// CheckEnvFile validates that the .env file exists.
// Returns a ValidationResult with error details if the file is missing.
func (v *ConfigValidator) CheckEnvFile() ValidationResult {
	if err := CheckFileExists(v.envPath); err != nil {
		return ValidationResult{
			Valid:   false,
			Message: "Configuration file not found. Copy .env.example to .env and configure your Canvus credentials.",
			Error:   ErrEnvFileMissing(v.envPath),
		}
	}
	return ValidationResult{
		Valid:   true,
		Message: "Environment file found",
	}
}

// CheckServerURL validates the CANVUS_SERVER environment variable.
// Returns a ValidationResult with error details if the URL is invalid.
func (v *ConfigValidator) CheckServerURL() ValidationResult {
	serverURL := GetEnvOrDefault("CANVUS_SERVER", "")

	if serverURL == "" {
		return ValidationResult{
			Valid:   false,
			Message: "CANVUS_SERVER required. Set your Canvus instance URL (e.g., https://canvus.example.com)",
			Error:   ErrMissingConfig("CANVUS_SERVER"),
		}
	}

	if err := ValidateServerURL(serverURL); err != nil {
		return ValidationResult{
			Valid:   false,
			Message: "Invalid Canvus server URL: " + serverURL + ". Example: https://canvus.example.com",
			Error:   ErrInvalidServerURL(serverURL, err.Error()),
		}
	}

	return ValidationResult{
		Valid:   true,
		Message: "Server URL valid",
	}
}

// CheckCanvasID validates the CANVAS_ID environment variable.
// Returns a ValidationResult with error details if the canvas ID is missing or invalid.
func (v *ConfigValidator) CheckCanvasID() ValidationResult {
	canvasID := GetEnvOrDefault("CANVAS_ID", "")

	if canvasID == "" {
		return ValidationResult{
			Valid:   false,
			Message: "CANVAS_ID required. Find your canvas ID in the canvas URL or settings.",
			Error:   ErrMissingConfig("CANVAS_ID"),
		}
	}

	// Basic validation: canvas ID should not be whitespace-only
	if strings.TrimSpace(canvasID) == "" {
		return ValidationResult{
			Valid:   false,
			Message: "Canvas ID invalid",
			Error:   ErrInvalidCanvasID(canvasID),
		}
	}

	return ValidationResult{
		Valid:   true,
		Message: "Canvas ID valid",
	}
}

// CheckAuthCredentials validates that Canvus authentication credentials are configured.
// Returns a ValidationResult with error details if authentication is missing.
func (v *ConfigValidator) CheckAuthCredentials() ValidationResult {
	creds := AuthCredentials{
		APIKey:   os.Getenv("CANVUS_API_KEY"),
		Username: os.Getenv("CANVUS_USERNAME"),
		Password: os.Getenv("CANVUS_PASSWORD"),
	}

	if err := ValidateAuthCredentials(creds); err != nil {
		return ValidationResult{
			Valid:   false,
			Message: "Canvus authentication required. Set CANVUS_API_KEY or CANVUS_USERNAME/CANVUS_PASSWORD",
			Error:   ErrMissingAuth("canvus"),
		}
	}

	return ValidationResult{
		Valid:   true,
		Message: "Canvus authentication configured",
	}
}

// CheckOpenAICredentials validates that OpenAI/LLM credentials are configured.
// Returns a ValidationResult with error details if the API key is missing.
// NOTE: This is OPTIONAL for local-first deployment. OpenAI key is only needed for cloud fallback.
func (v *ConfigValidator) CheckOpenAICredentials() ValidationResult {
	apiKey := os.Getenv("OPENAI_API_KEY")

	// Empty API key is acceptable for local LLM services
	if apiKey == "" {
		return ValidationResult{
			Valid:   false,
			Message: "OpenAI API key not configured (optional - only needed for cloud fallback)",
			Error:   ErrMissingAuth("openai"),
		}
	}

	if err := ValidateOpenAIAPIKey(apiKey); err != nil {
		return ValidationResult{
			Valid:   false,
			Message: "OpenAI API key invalid",
			Error:   ErrMissingAuth("openai"),
		}
	}

	return ValidationResult{
		Valid:   true,
		Message: "OpenAI API key configured",
	}
}

// ValidateAll runs all configuration checks and returns all results.
// This provides a comprehensive view of the configuration state, including optional settings.
func (v *ConfigValidator) ValidateAll() []ValidationResult {
	return []ValidationResult{
		v.CheckEnvFile(),
		v.CheckServerURL(),
		v.CheckCanvasID(),
		v.CheckAuthCredentials(),
		v.CheckOpenAICredentials(), // Included but not required
	}
}

// ValidateRequired runs only the required configuration checks for local-first deployment.
// This does NOT check OpenAI credentials - only Canvus connection settings.
// Returns the first validation failure, or nil if all required checks pass.
func (v *ConfigValidator) ValidateRequired() error {
	// Check env file first
	if result := v.CheckEnvFile(); !result.Valid {
		return result.Error
	}

	// Check server URL
	if result := v.CheckServerURL(); !result.Valid {
		return result.Error
	}

	// Check canvas ID
	if result := v.CheckCanvasID(); !result.Valid {
		return result.Error
	}

	// Check Canvus auth
	if result := v.CheckAuthCredentials(); !result.Valid {
		return result.Error
	}

	// NOTE: OpenAI credentials are NOT checked here - only needed for cloud fallback
	// Handlers should check config.OpenAIAPIKey != "" before attempting cloud operations

	return nil
}

// ValidateMinimal validates only the absolute minimum required configuration.
// This is the same as ValidateRequired for zero-config local deployment.
func (v *ConfigValidator) ValidateMinimal() error {
	return v.ValidateRequired()
}

// IsValid returns true if all required configuration is valid.
// This checks ONLY required settings (Canvus credentials), not optional cloud APIs.
func (v *ConfigValidator) IsValid() bool {
	return v.ValidateRequired() == nil
}

// GetFirstError returns the first validation error, or nil if all required checks pass.
func (v *ConfigValidator) GetFirstError() error {
	return v.ValidateRequired()
}

// CountValid returns the number of valid configuration items.
func (v *ConfigValidator) CountValid() int {
	results := v.ValidateAll()
	count := 0
	for _, r := range results {
		if r.Valid {
			count++
		}
	}
	return count
}

// CountInvalid returns the number of invalid configuration items.
func (v *ConfigValidator) CountInvalid() int {
	results := v.ValidateAll()
	return len(results) - v.CountValid()
}
