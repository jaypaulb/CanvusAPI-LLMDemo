package validation

import (
	"context"
	"fmt"
	"time"

	"go_backend/canvusapi"
)

// CanvasAccessResult represents the result of a canvas accessibility check.
type CanvasAccessResult struct {
	Accessible  bool
	WidgetCount int
	Message     string
	Error       error
}

// CanvasChecker provides methods to verify canvas accessibility.
// This is a molecule that verifies a canvas exists and is accessible.
type CanvasChecker struct {
	timeout              time.Duration
	allowSelfSignedCerts bool
}

// NewCanvasChecker creates a new CanvasChecker with default settings.
// Default timeout is 30 seconds.
func NewCanvasChecker() *CanvasChecker {
	return &CanvasChecker{
		timeout:              30 * time.Second,
		allowSelfSignedCerts: false,
	}
}

// WithTimeout sets the timeout for canvas checks.
func (c *CanvasChecker) WithTimeout(timeout time.Duration) *CanvasChecker {
	c.timeout = timeout
	return c
}

// WithAllowSelfSignedCerts configures whether to allow self-signed certificates.
func (c *CanvasChecker) WithAllowSelfSignedCerts(allow bool) *CanvasChecker {
	c.allowSelfSignedCerts = allow
	return c
}

// CheckCanvasAccess verifies that a canvas exists and is accessible.
// It attempts to get canvas info and widget count.
//
// Parameters:
//   - serverURL: The Canvus server URL
//   - canvasID: The canvas ID to check
//   - apiKey: The API key for authentication
//
// Returns a CanvasAccessResult with detailed information about the check.
func (c *CanvasChecker) CheckCanvasAccess(serverURL, canvasID, apiKey string) CanvasAccessResult {
	// Validate canvas ID format
	if canvasID == "" {
		return CanvasAccessResult{
			Accessible: false,
			Message:    "Canvas ID is empty",
			Error:      ErrInvalidCanvasID(canvasID),
		}
	}

	// Create a client to test canvas access
	client := canvusapi.NewClient(serverURL, canvasID, apiKey, c.allowSelfSignedCerts)

	// Try to get widgets to verify canvas access
	widgets, err := client.GetWidgets(false)
	if err != nil {
		// Check if it's an API error with status code
		if apiErr, ok := err.(*canvusapi.APIError); ok {
			switch apiErr.StatusCode {
			case 401:
				return CanvasAccessResult{
					Accessible: false,
					Message:    "Authentication failed",
					Error:      ErrAuthFailed("canvus", "invalid or expired API key"),
				}
			case 403:
				return CanvasAccessResult{
					Accessible: false,
					Message:    "Access denied to canvas",
					Error:      ErrAuthFailed("canvus", "access denied - check permissions for this canvas"),
				}
			case 404:
				return CanvasAccessResult{
					Accessible: false,
					Message:    "Canvas not found",
					Error:      ErrCanvasNotFound(canvasID),
				}
			default:
				return CanvasAccessResult{
					Accessible: false,
					Message:    fmt.Sprintf("API error: %d", apiErr.StatusCode),
					Error:      ErrServerUnreachable(serverURL, apiErr.Message),
				}
			}
		}
		// Network or other error
		return CanvasAccessResult{
			Accessible: false,
			Message:    "Connection failed",
			Error:      ErrServerUnreachable(serverURL, err.Error()),
		}
	}

	widgetCount := len(widgets)
	return CanvasAccessResult{
		Accessible:  true,
		WidgetCount: widgetCount,
		Message:     fmt.Sprintf("Canvas accessible (%d widgets)", widgetCount),
	}
}

// CheckCanvasAccessWithContext verifies canvas accessibility with a custom context.
func (c *CanvasChecker) CheckCanvasAccessWithContext(ctx context.Context, serverURL, canvasID, apiKey string) CanvasAccessResult {
	// Create a channel to receive the result
	resultChan := make(chan CanvasAccessResult, 1)

	go func() {
		resultChan <- c.CheckCanvasAccess(serverURL, canvasID, apiKey)
	}()

	select {
	case result := <-resultChan:
		return result
	case <-ctx.Done():
		return CanvasAccessResult{
			Accessible: false,
			Message:    "Canvas check cancelled or timed out",
			Error:      ErrServerUnreachable(serverURL, ctx.Err().Error()),
		}
	}
}

// CheckCanvusCanvas verifies canvas accessibility using environment variables.
// Uses CANVUS_SERVER, CANVAS_ID, and CANVUS_API_KEY.
func (c *CanvasChecker) CheckCanvusCanvas() CanvasAccessResult {
	serverURL := GetEnvOrDefault("CANVUS_SERVER", "")
	if serverURL == "" {
		return CanvasAccessResult{
			Accessible: false,
			Message:    "CANVUS_SERVER not configured",
			Error:      ErrMissingConfig("CANVUS_SERVER"),
		}
	}

	canvasID := GetEnvOrDefault("CANVAS_ID", "")
	if canvasID == "" {
		return CanvasAccessResult{
			Accessible: false,
			Message:    "CANVAS_ID not configured",
			Error:      ErrMissingConfig("CANVAS_ID"),
		}
	}

	apiKey := GetEnvOrDefault("CANVUS_API_KEY", "")
	if apiKey == "" {
		return CanvasAccessResult{
			Accessible: false,
			Message:    "CANVUS_API_KEY not configured",
			Error:      ErrMissingAuth("canvus"),
		}
	}

	return c.CheckCanvasAccess(serverURL, canvasID, apiKey)
}

// IsAccessible is a convenience function to check if a canvas is accessible.
// Returns true if the canvas can be accessed, false otherwise.
func (c *CanvasChecker) IsAccessible(serverURL, canvasID, apiKey string) bool {
	result := c.CheckCanvasAccess(serverURL, canvasID, apiKey)
	return result.Accessible
}

// GetWidgetCount returns the number of widgets in the canvas, or -1 on error.
func (c *CanvasChecker) GetWidgetCount(serverURL, canvasID, apiKey string) int {
	result := c.CheckCanvasAccess(serverURL, canvasID, apiKey)
	if !result.Accessible {
		return -1
	}
	return result.WidgetCount
}
