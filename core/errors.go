package core

import (
	"fmt"
)

// ConfigError represents a configuration-related error with actionable instructions.
type ConfigError struct {
	Code    string // Error code for programmatic handling
	Message string // Human-readable error message
	Action  string // Actionable instruction for resolution
}

func (e *ConfigError) Error() string {
	if e.Action != "" {
		return fmt.Sprintf("%s. %s", e.Message, e.Action)
	}
	return e.Message
}

// Error codes for configuration errors
const (
	ErrCodeEnvFileMissing    = "ENV_FILE_MISSING"
	ErrCodeInvalidServerURL  = "INVALID_SERVER_URL"
	ErrCodeMissingAuth       = "MISSING_AUTH"
	ErrCodeServerUnreachable = "SERVER_UNREACHABLE"
	ErrCodeAuthFailed        = "AUTH_FAILED"
	ErrCodeCanvasNotFound    = "CANVAS_NOT_FOUND"
	ErrCodeInvalidCanvasID   = "INVALID_CANVAS_ID"
	ErrCodeMissingConfig     = "MISSING_CONFIG"
)

// ErrEnvFileMissing returns an error for missing .env file
func ErrEnvFileMissing(path string) *ConfigError {
	return &ConfigError{
		Code:    ErrCodeEnvFileMissing,
		Message: fmt.Sprintf("Configuration file not found: %s", path),
		Action:  "Copy example.env to .env and configure the required values",
	}
}

// ErrInvalidServerURL returns an error for invalid server URL format
func ErrInvalidServerURL(url string, reason string) *ConfigError {
	return &ConfigError{
		Code:    ErrCodeInvalidServerURL,
		Message: fmt.Sprintf("Invalid CANVUS_SERVER URL '%s': %s", url, reason),
		Action:  "Set CANVUS_SERVER to a valid URL (e.g., https://canvus.example.com)",
	}
}

// ErrMissingAuth returns an error for missing authentication credentials
func ErrMissingAuth(service string) *ConfigError {
	var action string
	switch service {
	case "canvus":
		action = "Set CANVUS_API_KEY in your .env file"
	case "openai":
		action = "Set OPENAI_API_KEY in your .env file (or configure a local LLM)"
	default:
		action = fmt.Sprintf("Set the required API key for %s in your .env file", service)
	}
	return &ConfigError{
		Code:    ErrCodeMissingAuth,
		Message: fmt.Sprintf("Missing authentication credentials for %s", service),
		Action:  action,
	}
}

// ErrServerUnreachable returns an error when the server cannot be reached
func ErrServerUnreachable(url string, reason string) *ConfigError {
	return &ConfigError{
		Code:    ErrCodeServerUnreachable,
		Message: fmt.Sprintf("Cannot connect to server at %s: %s", url, reason),
		Action:  "Check that CANVUS_SERVER is correct and the server is running. For self-signed certificates, set ALLOW_SELF_SIGNED_CERTS=true",
	}
}

// ErrAuthFailed returns an error when authentication fails
func ErrAuthFailed(service string, reason string) *ConfigError {
	return &ConfigError{
		Code:    ErrCodeAuthFailed,
		Message: fmt.Sprintf("Authentication failed for %s: %s", service, reason),
		Action:  "Verify your API key is correct and has not expired",
	}
}

// ErrCanvasNotFound returns an error when the specified canvas cannot be found
func ErrCanvasNotFound(canvasID string) *ConfigError {
	return &ConfigError{
		Code:    ErrCodeCanvasNotFound,
		Message: fmt.Sprintf("Canvas not found: %s", canvasID),
		Action:  "Verify CANVAS_ID is correct and you have access to the canvas",
	}
}

// ErrInvalidCanvasID returns an error for invalid canvas ID format
func ErrInvalidCanvasID(canvasID string) *ConfigError {
	return &ConfigError{
		Code:    ErrCodeInvalidCanvasID,
		Message: fmt.Sprintf("Invalid CANVAS_ID format: %s", canvasID),
		Action:  "Set CANVAS_ID to a valid canvas identifier",
	}
}

// ErrMissingConfig returns an error for missing required configuration
func ErrMissingConfig(varName string) *ConfigError {
	return &ConfigError{
		Code:    ErrCodeMissingConfig,
		Message: fmt.Sprintf("Missing required configuration: %s", varName),
		Action:  fmt.Sprintf("Set %s in your .env file", varName),
	}
}

// IsConfigError checks if an error is a ConfigError and returns it if so
func IsConfigError(err error) (*ConfigError, bool) {
	if configErr, ok := err.(*ConfigError); ok {
		return configErr, true
	}
	return nil, false
}

// GetErrorCode extracts the error code from an error if it's a ConfigError
func GetErrorCode(err error) string {
	if configErr, ok := IsConfigError(err); ok {
		return configErr.Code
	}
	return ""
}
