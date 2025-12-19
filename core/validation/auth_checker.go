package validation

import (
	"go_backend/core"
	"context"
	"fmt"
	"time"

	"go_backend/canvusapi"
)

// AuthResult represents the result of an authentication check.
type AuthResult struct {
	Authenticated bool
	Message       string
	Error         error
}

// AuthChecker provides methods to verify API authentication.
// This is a molecule that composes credential validation with actual API authentication.
type AuthChecker struct {
	timeout              time.Duration
	allowSelfSignedCerts bool
}

// NewAuthChecker creates a new AuthChecker with default settings.
// Default timeout is 30 seconds.
func NewAuthChecker() *AuthChecker {
	return &AuthChecker{
		timeout:              30 * time.Second,
		allowSelfSignedCerts: false,
	}
}

// WithTimeout sets the timeout for authentication checks.
func (c *AuthChecker) WithTimeout(timeout time.Duration) *AuthChecker {
	c.timeout = timeout
	return c
}

// WithAllowSelfSignedCerts configures whether to allow self-signed certificates.
func (c *AuthChecker) WithAllowSelfSignedCerts(allow bool) *AuthChecker {
	c.allowSelfSignedCerts = allow
	return c
}

// CheckAPIAuth verifies that the provided credentials can authenticate with the Canvus API.
// It creates a client and attempts to call GetWidgets to verify authentication.
//
// Parameters:
//   - serverURL: The Canvus server URL
//   - canvasID: The canvas ID to authenticate against
//   - apiKey: The API key to verify
//
// Returns an AuthResult with detailed information about the check.
func (c *AuthChecker) CheckAPIAuth(serverURL, canvasID, apiKey string) AuthResult {
	// First validate credentials format using the atom
	creds := core.AuthCredentials{APIKey: apiKey}
	if err := core.ValidateAuthCredentials(creds); err != nil {
		return AuthResult{
			Authenticated: false,
			Message:       "Invalid credentials format",
			Error:         core.ErrMissingAuth("canvus"),
		}
	}

	// Create a client to test authentication
	client := canvusapi.NewClient(serverURL, canvasID, apiKey, c.allowSelfSignedCerts)

	// Try to get widgets - this will fail with 401/403 if auth is invalid
	_, err := client.GetWidgets(false)
	if err != nil {
		// Check if it's an API error with status code
		if apiErr, ok := err.(*canvusapi.APIError); ok {
			switch apiErr.StatusCode {
			case 401:
				return AuthResult{
					Authenticated: false,
					Message:       "Authentication failed: invalid API key",
					Error:         core.ErrAuthFailed("canvus", "invalid or expired API key"),
				}
			case 403:
				return AuthResult{
					Authenticated: false,
					Message:       "Authentication failed: access denied",
					Error:         core.ErrAuthFailed("canvus", "access denied - check permissions"),
				}
			case 404:
				// Canvas not found, but auth might be okay
				return AuthResult{
					Authenticated: true, // Key is valid, just canvas doesn't exist
					Message:       "API key valid but canvas not found",
					Error:         core.ErrCanvasNotFound(canvasID),
				}
			default:
				return AuthResult{
					Authenticated: false,
					Message:       fmt.Sprintf("API error: %d", apiErr.StatusCode),
					Error:         core.ErrAuthFailed("canvus", apiErr.Message),
				}
			}
		}
		// Network or other error
		return AuthResult{
			Authenticated: false,
			Message:       "Connection failed",
			Error:         core.ErrServerUnreachable(serverURL, err.Error()),
		}
	}

	return AuthResult{
		Authenticated: true,
		Message:       "Authentication successful",
	}
}

// CheckAPIAuthWithContext verifies authentication with a custom context for cancellation/timeout.
func (c *AuthChecker) CheckAPIAuthWithContext(ctx context.Context, serverURL, canvasID, apiKey string) AuthResult {
	// Create a channel to receive the result
	resultChan := make(chan AuthResult, 1)

	go func() {
		resultChan <- c.CheckAPIAuth(serverURL, canvasID, apiKey)
	}()

	select {
	case result := <-resultChan:
		return result
	case <-ctx.Done():
		return AuthResult{
			Authenticated: false,
			Message:       "Authentication check cancelled or timed out",
			Error:         core.ErrServerUnreachable(serverURL, ctx.Err().Error()),
		}
	}
}

// CheckCanvusAPIAuth verifies authentication using environment variables.
// Uses CANVUS_SERVER, CANVAS_ID, and CANVUS_API_KEY.
func (c *AuthChecker) CheckCanvusAPIAuth() AuthResult {
	serverURL := core.GetEnvOrDefault("CANVUS_SERVER", "")
	if serverURL == "" {
		return AuthResult{
			Authenticated: false,
			Message:       "CANVUS_SERVER not configured",
			Error:         core.ErrMissingConfig("CANVUS_SERVER"),
		}
	}

	canvasID := core.GetEnvOrDefault("CANVAS_ID", "")
	if canvasID == "" {
		return AuthResult{
			Authenticated: false,
			Message:       "CANVAS_ID not configured",
			Error:         core.ErrMissingConfig("CANVAS_ID"),
		}
	}

	apiKey := core.GetEnvOrDefault("CANVUS_API_KEY", "")
	if apiKey == "" {
		return AuthResult{
			Authenticated: false,
			Message:       "CANVUS_API_KEY not configured",
			Error:         core.ErrMissingAuth("canvus"),
		}
	}

	return c.CheckAPIAuth(serverURL, canvasID, apiKey)
}

// IsAuthenticated is a convenience function to check if credentials are valid.
// Returns true if authentication succeeds, false otherwise.
func (c *AuthChecker) IsAuthenticated(serverURL, canvasID, apiKey string) bool {
	result := c.CheckAPIAuth(serverURL, canvasID, apiKey)
	return result.Authenticated
}
