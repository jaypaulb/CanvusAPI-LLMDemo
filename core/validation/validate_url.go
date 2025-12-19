package validation

import (
	"fmt"
	"net/url"
	"strings"
)

// ValidateServerURL validates that a URL has a valid format with http or https scheme.
// This is a pure function with no side effects.
//
// Returns nil if the URL is valid, or an error describing the validation failure.
func ValidateServerURL(serverURL string) error {
	// Trim whitespace first
	serverURL = strings.TrimSpace(serverURL)

	if serverURL == "" {
		return fmt.Errorf("server URL cannot be empty")
	}

	// Parse the URL
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Validate scheme
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme, got: %q", parsedURL.Scheme)
	}

	// Validate host is present
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must include a host")
	}

	return nil
}
