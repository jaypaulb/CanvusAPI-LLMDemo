package core

import (
	"fmt"
	"strings"
)

// AuthCredentials represents authentication credentials.
// Either APIKey OR (Username AND Password) must be provided.
type AuthCredentials struct {
	APIKey   string
	Username string
	Password string
}

// ValidateAuthCredentials validates that authentication credentials are properly provided.
// This is a pure function with no network calls or side effects.
//
// Valid configurations:
// - APIKey is set (username/password ignored)
// - Username AND Password are both set (APIKey not required)
//
// Returns nil if credentials are valid, or an error describing the validation failure.
func ValidateAuthCredentials(creds AuthCredentials) error {
	apiKey := strings.TrimSpace(creds.APIKey)
	username := strings.TrimSpace(creds.Username)
	password := strings.TrimSpace(creds.Password)

	// Check if API key is provided
	if apiKey != "" {
		return nil // API key auth is valid
	}

	// Check if username/password pair is provided
	if username != "" && password != "" {
		return nil // Username/password auth is valid
	}

	// Neither valid auth method provided
	if username == "" && password == "" {
		return fmt.Errorf("authentication required: provide either API key or username/password")
	}

	// Partial username/password
	if username == "" {
		return fmt.Errorf("authentication incomplete: username required when password is provided")
	}
	return fmt.Errorf("authentication incomplete: password required when username is provided")
}

// ValidateAPIKey validates that an API key is non-empty and has reasonable format.
// This is a pure function that does NOT verify the key with any service.
func ValidateAPIKey(apiKey string) error {
	apiKey = strings.TrimSpace(apiKey)

	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// Basic sanity check - API keys should have some minimum length
	if len(apiKey) < 8 {
		return fmt.Errorf("API key appears invalid: too short (minimum 8 characters)")
	}

	return nil
}

// ValidateCanvusAPIKey validates the Canvus API key format.
// Canvus API keys should be non-empty.
func ValidateCanvusAPIKey(apiKey string) error {
	return ValidateAPIKey(apiKey)
}

// ValidateOpenAIAPIKey validates the OpenAI API key format.
// OpenAI keys typically start with "sk-" but we allow flexibility for local LLM services.
func ValidateOpenAIAPIKey(apiKey string) error {
	if err := ValidateAPIKey(apiKey); err != nil {
		return err
	}

	// Note: We don't enforce "sk-" prefix because users may use local LLM services
	// that accept different key formats, or even dummy keys for llama.cpp
	return nil
}
