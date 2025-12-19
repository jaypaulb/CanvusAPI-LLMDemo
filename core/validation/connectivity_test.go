package validation

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestConnectivityChecker_CheckServerConnectivity(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("Expected HEAD request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tests := []struct {
		name        string
		serverURL   string
		wantReach   bool
		wantMessage string
	}{
		{
			name:      "valid reachable server",
			serverURL: server.URL,
			wantReach: true,
		},
		{
			name:        "invalid URL format - no scheme",
			serverURL:   "not-a-valid-url",
			wantReach:   false,
			wantMessage: "Invalid URL format",
		},
		{
			name:        "empty URL",
			serverURL:   "",
			wantReach:   false,
			wantMessage: "Invalid URL format",
		},
		{
			name:        "unreachable server",
			serverURL:   "http://localhost:59999", // unlikely to be in use
			wantReach:   false,
			wantMessage: "Connection failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConnectivityChecker().WithTimeout(2 * time.Second)
			result := c.CheckServerConnectivity(tt.serverURL)

			if result.Reachable != tt.wantReach {
				t.Errorf("CheckServerConnectivity() Reachable = %v, want %v", result.Reachable, tt.wantReach)
			}

			if tt.wantMessage != "" && result.Message != tt.wantMessage {
				t.Errorf("CheckServerConnectivity() Message = %q, want %q", result.Message, tt.wantMessage)
			}

			if !tt.wantReach && result.Error == nil {
				t.Error("CheckServerConnectivity() expected error for unreachable server")
			}
		})
	}
}

func TestConnectivityChecker_StatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantReach  bool
	}{
		{
			name:       "200 OK",
			statusCode: http.StatusOK,
			wantReach:  true,
		},
		{
			name:       "301 Redirect",
			statusCode: http.StatusMovedPermanently,
			wantReach:  true,
		},
		{
			name:       "401 Unauthorized",
			statusCode: http.StatusUnauthorized,
			wantReach:  true, // Server is reachable, just auth issue
		},
		{
			name:       "403 Forbidden",
			statusCode: http.StatusForbidden,
			wantReach:  true, // Server is reachable
		},
		{
			name:       "404 Not Found",
			statusCode: http.StatusNotFound,
			wantReach:  true, // Server is reachable
		},
		{
			name:       "500 Internal Server Error",
			statusCode: http.StatusInternalServerError,
			wantReach:  true, // Server is reachable, just erroring
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create server that returns the specific status
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			c := NewConnectivityChecker()
			result := c.CheckServerConnectivity(server.URL)

			if result.Reachable != tt.wantReach {
				t.Errorf("CheckServerConnectivity() Reachable = %v, want %v", result.Reachable, tt.wantReach)
			}

			if result.StatusCode != tt.statusCode {
				t.Errorf("CheckServerConnectivity() StatusCode = %d, want %d", result.StatusCode, tt.statusCode)
			}
		})
	}
}

func TestConnectivityChecker_Timeout(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second) // Long delay
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewConnectivityChecker().WithTimeout(100 * time.Millisecond)
	result := c.CheckServerConnectivity(server.URL)

	if result.Reachable {
		t.Error("Expected timeout to make server appear unreachable")
	}

	if result.Message != "Connection timed out" {
		t.Errorf("Expected timeout message, got: %s", result.Message)
	}
}

func TestConnectivityChecker_WithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a context that cancels quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	c := NewConnectivityChecker().WithTimeout(10 * time.Second)
	result := c.CheckServerConnectivityWithContext(ctx, server.URL)

	if result.Reachable {
		t.Error("Expected cancelled context to make server appear unreachable")
	}
}

func TestConnectivityChecker_Latency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewConnectivityChecker()
	result := c.CheckServerConnectivity(server.URL)

	if !result.Reachable {
		t.Fatalf("Expected server to be reachable")
	}

	if result.Latency < 50*time.Millisecond {
		t.Errorf("Latency should be at least 50ms, got %v", result.Latency)
	}
}

func TestConnectivityChecker_IsReachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := NewConnectivityChecker()

	// Reachable server
	if !c.IsReachable(server.URL) {
		t.Error("IsReachable() should return true for running server")
	}

	// Unreachable server
	if c.IsReachable("http://localhost:59999") {
		t.Error("IsReachable() should return false for unreachable server")
	}
}

func TestConnectivityChecker_CheckCanvusServerConnectivity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tests := []struct {
		name      string
		serverURL string
		wantReach bool
	}{
		{
			name:      "valid server URL",
			serverURL: server.URL,
			wantReach: true,
		},
		{
			name:      "empty server URL",
			serverURL: "",
			wantReach: false,
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

			c := NewConnectivityChecker()
			result := c.CheckCanvusServerConnectivity()

			if result.Reachable != tt.wantReach {
				t.Errorf("CheckCanvusServerConnectivity() Reachable = %v, want %v", result.Reachable, tt.wantReach)
			}
		})
	}
}

func TestConnectivityChecker_WithAllowSelfSignedCerts(t *testing.T) {
	// This test just verifies the builder pattern works
	c := NewConnectivityChecker().
		WithTimeout(5 * time.Second).
		WithAllowSelfSignedCerts(true)

	if c.timeout != 5*time.Second {
		t.Errorf("WithTimeout() did not set timeout correctly")
	}

	if !c.allowSelfSignedCerts {
		t.Errorf("WithAllowSelfSignedCerts() did not set flag correctly")
	}
}

func TestConnectivityChecker_HTTPSServer(t *testing.T) {
	// Create HTTPS test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Without allowing self-signed certs, should fail
	c := NewConnectivityChecker().WithTimeout(2 * time.Second)
	result := c.CheckServerConnectivity(server.URL)

	if result.Reachable {
		t.Error("Expected HTTPS with self-signed cert to fail without allowSelfSignedCerts")
	}

	// With allowing self-signed certs, should succeed
	c2 := NewConnectivityChecker().
		WithTimeout(2 * time.Second).
		WithAllowSelfSignedCerts(true)
	result2 := c2.CheckServerConnectivity(server.URL)

	if !result2.Reachable {
		t.Errorf("Expected HTTPS with self-signed cert to succeed with allowSelfSignedCerts: %v", result2.Error)
	}
}
