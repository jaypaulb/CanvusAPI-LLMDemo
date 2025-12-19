package validation

import (
	"go_backend/core"
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

// ConnectivityResult represents the result of a connectivity check.
type ConnectivityResult struct {
	Reachable  bool
	StatusCode int
	Message    string
	Latency    time.Duration
	Error      error
}

// ConnectivityChecker provides methods to verify network connectivity.
// This is a molecule that composes URL validation with HTTP connectivity tests.
type ConnectivityChecker struct {
	timeout              time.Duration
	allowSelfSignedCerts bool
}

// NewConnectivityChecker creates a new ConnectivityChecker with default settings.
// Default timeout is 10 seconds.
func NewConnectivityChecker() *ConnectivityChecker {
	return &ConnectivityChecker{
		timeout:              10 * time.Second,
		allowSelfSignedCerts: false,
	}
}

// WithTimeout sets the timeout for connectivity checks.
func (c *ConnectivityChecker) WithTimeout(timeout time.Duration) *ConnectivityChecker {
	c.timeout = timeout
	return c
}

// WithAllowSelfSignedCerts configures whether to allow self-signed certificates.
func (c *ConnectivityChecker) WithAllowSelfSignedCerts(allow bool) *ConnectivityChecker {
	c.allowSelfSignedCerts = allow
	return c
}

// CheckServerConnectivity tests if a server is reachable using HTTP HEAD request.
// This validates the URL format first, then attempts a network connection.
//
// Returns a ConnectivityResult with detailed information about the check.
func (c *ConnectivityChecker) CheckServerConnectivity(serverURL string) ConnectivityResult {
	// First validate the URL format using the atom
	if err := core.ValidateServerURL(serverURL); err != nil {
		return ConnectivityResult{
			Reachable: false,
			Message:   "Invalid URL format",
			Error:     core.ErrInvalidServerURL(serverURL, err.Error()),
		}
	}

	// Create HTTP client with TLS configuration
	client := c.createHTTPClient()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// Create HEAD request
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, serverURL, nil)
	if err != nil {
		return ConnectivityResult{
			Reachable: false,
			Message:   "Failed to create request",
			Error:     core.ErrServerUnreachable(serverURL, err.Error()),
		}
	}

	// Record start time for latency measurement
	startTime := time.Now()

	// Perform the request
	resp, err := client.Do(req)
	latency := time.Since(startTime)

	if err != nil {
		// Handle timeout specifically
		if ctx.Err() == context.DeadlineExceeded {
			return ConnectivityResult{
				Reachable: false,
				Message:   "Connection timed out",
				Latency:   latency,
				Error:     core.ErrServerUnreachable(serverURL, fmt.Sprintf("connection timed out after %v", c.timeout)),
			}
		}
		return ConnectivityResult{
			Reachable: false,
			Message:   "Connection failed",
			Latency:   latency,
			Error:     core.ErrServerUnreachable(serverURL, err.Error()),
		}
	}
	defer resp.Body.Close()

	// Check if the server responded (any 2xx or 3xx is considered reachable)
	// 4xx and 5xx indicate server is reachable but may have auth/other issues
	return ConnectivityResult{
		Reachable:  true,
		StatusCode: resp.StatusCode,
		Message:    fmt.Sprintf("Server reachable (status: %d)", resp.StatusCode),
		Latency:    latency,
	}
}

// CheckServerConnectivityWithContext tests server connectivity with custom context.
func (c *ConnectivityChecker) CheckServerConnectivityWithContext(ctx context.Context, serverURL string) ConnectivityResult {
	// First validate the URL format using the atom
	if err := core.ValidateServerURL(serverURL); err != nil {
		return ConnectivityResult{
			Reachable: false,
			Message:   "Invalid URL format",
			Error:     core.ErrInvalidServerURL(serverURL, err.Error()),
		}
	}

	// Create HTTP client with TLS configuration
	client := c.createHTTPClient()

	// Create HEAD request
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, serverURL, nil)
	if err != nil {
		return ConnectivityResult{
			Reachable: false,
			Message:   "Failed to create request",
			Error:     core.ErrServerUnreachable(serverURL, err.Error()),
		}
	}

	// Record start time for latency measurement
	startTime := time.Now()

	// Perform the request
	resp, err := client.Do(req)
	latency := time.Since(startTime)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded || ctx.Err() == context.Canceled {
			return ConnectivityResult{
				Reachable: false,
				Message:   "Request cancelled or timed out",
				Latency:   latency,
				Error:     core.ErrServerUnreachable(serverURL, ctx.Err().Error()),
			}
		}
		return ConnectivityResult{
			Reachable: false,
			Message:   "Connection failed",
			Latency:   latency,
			Error:     core.ErrServerUnreachable(serverURL, err.Error()),
		}
	}
	defer resp.Body.Close()

	return ConnectivityResult{
		Reachable:  true,
		StatusCode: resp.StatusCode,
		Message:    fmt.Sprintf("Server reachable (status: %d)", resp.StatusCode),
		Latency:    latency,
	}
}

// createHTTPClient creates an HTTP client with the configured TLS settings.
func (c *ConnectivityChecker) createHTTPClient() *http.Client {
	client := &http.Client{
		Timeout: c.timeout,
	}

	if c.allowSelfSignedCerts {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	return client
}

// IsReachable is a convenience function to check if a server is reachable.
// Returns true if the server responds, false otherwise.
func (c *ConnectivityChecker) IsReachable(serverURL string) bool {
	result := c.CheckServerConnectivity(serverURL)
	return result.Reachable
}

// CheckCanvusServerConnectivity checks connectivity to the Canvus server
// using the CANVUS_SERVER environment variable.
func (c *ConnectivityChecker) CheckCanvusServerConnectivity() ConnectivityResult {
	serverURL := core.GetEnvOrDefault("CANVUS_SERVER", "")
	if serverURL == "" {
		return ConnectivityResult{
			Reachable: false,
			Message:   "CANVUS_SERVER not configured",
			Error:     core.ErrMissingConfig("CANVUS_SERVER"),
		}
	}
	return c.CheckServerConnectivity(serverURL)
}
