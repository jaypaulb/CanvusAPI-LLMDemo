package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestAuthChecker_CheckAPIAuth_Success(t *testing.T) {
	// Create a mock server that returns successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the auth header is present
		if r.Header.Get("Private-Token") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Return empty widget list
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	c := NewAuthChecker()
	result := c.CheckAPIAuth(server.URL, "test-canvas", "valid-api-key-12345")

	if !result.Authenticated {
		t.Errorf("CheckAPIAuth() Authenticated = false, want true: %v", result.Error)
	}
}

func TestAuthChecker_CheckAPIAuth_InvalidKey(t *testing.T) {
	// Create a mock server that returns 401 for invalid keys
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Invalid API key"))
	}))
	defer server.Close()

	c := NewAuthChecker()
	result := c.CheckAPIAuth(server.URL, "test-canvas", "invalid-key-123")

	if result.Authenticated {
		t.Error("CheckAPIAuth() should fail with 401 response")
	}

	if result.Error == nil {
		t.Error("CheckAPIAuth() should return error for 401")
	}
}

func TestAuthChecker_CheckAPIAuth_Forbidden(t *testing.T) {
	// Create a mock server that returns 403
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Access denied"))
	}))
	defer server.Close()

	c := NewAuthChecker()
	result := c.CheckAPIAuth(server.URL, "test-canvas", "valid-key-12345678")

	if result.Authenticated {
		t.Error("CheckAPIAuth() should fail with 403 response")
	}
}

func TestAuthChecker_CheckAPIAuth_CanvasNotFound(t *testing.T) {
	// Create a mock server that returns 404 (canvas not found)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Canvas not found"))
	}))
	defer server.Close()

	c := NewAuthChecker()
	result := c.CheckAPIAuth(server.URL, "nonexistent-canvas", "valid-key-12345678")

	// API key is valid, just canvas doesn't exist
	if !result.Authenticated {
		t.Error("CheckAPIAuth() should report authenticated=true when canvas not found (key is valid)")
	}

	// But there should be an error about the canvas
	if result.Error == nil {
		t.Error("CheckAPIAuth() should return error for canvas not found")
	}
}

func TestAuthChecker_CheckAPIAuth_EmptyCredentials(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
	}{
		{
			name:    "empty API key",
			apiKey:  "",
			wantErr: true,
		},
		{
			name:    "short API key",
			apiKey:  "short",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewAuthChecker()
			result := c.CheckAPIAuth("http://localhost", "canvas", tt.apiKey)

			if result.Authenticated {
				t.Error("CheckAPIAuth() should fail for empty/invalid credentials")
			}
		})
	}
}

func TestAuthChecker_CheckAPIAuth_ServerError(t *testing.T) {
	// Create a mock server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	c := NewAuthChecker()
	result := c.CheckAPIAuth(server.URL, "test-canvas", "valid-key-12345678")

	if result.Authenticated {
		t.Error("CheckAPIAuth() should fail with 500 response")
	}
}

func TestAuthChecker_CheckAPIAuth_UnreachableServer(t *testing.T) {
	c := NewAuthChecker().WithTimeout(1 * time.Second)
	result := c.CheckAPIAuth("http://localhost:59999", "test-canvas", "valid-key-12345678")

	if result.Authenticated {
		t.Error("CheckAPIAuth() should fail for unreachable server")
	}

	if result.Message != "Connection failed" {
		t.Errorf("Expected 'Connection failed' message, got: %s", result.Message)
	}
}

func TestAuthChecker_CheckAPIAuthWithContext(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	// Create a context that times out quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	c := NewAuthChecker()
	result := c.CheckAPIAuthWithContext(ctx, server.URL, "test-canvas", "valid-key-12345678")

	if result.Authenticated {
		t.Error("CheckAPIAuthWithContext() should fail with cancelled context")
	}
}

func TestAuthChecker_CheckCanvusAPIAuth(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	tests := []struct {
		name      string
		serverURL string
		canvasID  string
		apiKey    string
		wantAuth  bool
	}{
		{
			name:      "all env vars set",
			serverURL: server.URL,
			canvasID:  "test-canvas",
			apiKey:    "valid-key-12345678",
			wantAuth:  true,
		},
		{
			name:      "missing server URL",
			serverURL: "",
			canvasID:  "test-canvas",
			apiKey:    "valid-key-12345678",
			wantAuth:  false,
		},
		{
			name:      "missing canvas ID",
			serverURL: server.URL,
			canvasID:  "",
			apiKey:    "valid-key-12345678",
			wantAuth:  false,
		},
		{
			name:      "missing API key",
			serverURL: server.URL,
			canvasID:  "test-canvas",
			apiKey:    "",
			wantAuth:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// Set test values
			if tt.serverURL != "" {
				os.Setenv("CANVUS_SERVER", tt.serverURL)
			} else {
				os.Unsetenv("CANVUS_SERVER")
			}
			if tt.canvasID != "" {
				os.Setenv("CANVAS_ID", tt.canvasID)
			} else {
				os.Unsetenv("CANVAS_ID")
			}
			if tt.apiKey != "" {
				os.Setenv("CANVUS_API_KEY", tt.apiKey)
			} else {
				os.Unsetenv("CANVUS_API_KEY")
			}

			c := NewAuthChecker()
			result := c.CheckCanvusAPIAuth()

			if result.Authenticated != tt.wantAuth {
				t.Errorf("CheckCanvusAPIAuth() Authenticated = %v, want %v: %v", result.Authenticated, tt.wantAuth, result.Error)
			}
		})
	}
}

func TestAuthChecker_IsAuthenticated(t *testing.T) {
	// Create a mock server that succeeds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	c := NewAuthChecker()

	if !c.IsAuthenticated(server.URL, "test-canvas", "valid-key-12345678") {
		t.Error("IsAuthenticated() should return true for valid credentials")
	}

	// Test with unreachable server
	if c.IsAuthenticated("http://localhost:59999", "test-canvas", "valid-key-12345678") {
		t.Error("IsAuthenticated() should return false for unreachable server")
	}
}

func TestAuthChecker_BuilderPattern(t *testing.T) {
	c := NewAuthChecker().
		WithTimeout(15 * time.Second).
		WithAllowSelfSignedCerts(true)

	if c.timeout != 15*time.Second {
		t.Errorf("WithTimeout() did not set timeout correctly")
	}

	if !c.allowSelfSignedCerts {
		t.Errorf("WithAllowSelfSignedCerts() did not set flag correctly")
	}
}
