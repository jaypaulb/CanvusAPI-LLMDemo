package validation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestCanvasChecker_CheckCanvasAccess_Success(t *testing.T) {
	// Create a mock server that returns widgets
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "widget-1", "type": "note"},
			{"id": "widget-2", "type": "image"},
			{"id": "widget-3", "type": "pdf"},
		})
	}))
	defer server.Close()

	c := NewCanvasChecker()
	result := c.CheckCanvasAccess(server.URL, "test-canvas", "valid-api-key-12345")

	if !result.Accessible {
		t.Errorf("CheckCanvasAccess() Accessible = false, want true: %v", result.Error)
	}

	if result.WidgetCount != 3 {
		t.Errorf("CheckCanvasAccess() WidgetCount = %d, want 3", result.WidgetCount)
	}
}

func TestCanvasChecker_CheckCanvasAccess_EmptyCanvas(t *testing.T) {
	// Create a mock server that returns empty canvas
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	c := NewCanvasChecker()
	result := c.CheckCanvasAccess(server.URL, "empty-canvas", "valid-api-key-12345")

	if !result.Accessible {
		t.Error("CheckCanvasAccess() should succeed for empty canvas")
	}

	if result.WidgetCount != 0 {
		t.Errorf("CheckCanvasAccess() WidgetCount = %d, want 0", result.WidgetCount)
	}
}

func TestCanvasChecker_CheckCanvasAccess_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Canvas not found"))
	}))
	defer server.Close()

	c := NewCanvasChecker()
	result := c.CheckCanvasAccess(server.URL, "nonexistent-canvas", "valid-api-key-12345")

	if result.Accessible {
		t.Error("CheckCanvasAccess() should fail for 404")
	}

	if result.Message != "Canvas not found" {
		t.Errorf("Expected 'Canvas not found' message, got: %s", result.Message)
	}
}

func TestCanvasChecker_CheckCanvasAccess_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Invalid API key"))
	}))
	defer server.Close()

	c := NewCanvasChecker()
	result := c.CheckCanvasAccess(server.URL, "test-canvas", "invalid-key-123456")

	if result.Accessible {
		t.Error("CheckCanvasAccess() should fail for 401")
	}
}

func TestCanvasChecker_CheckCanvasAccess_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Access denied"))
	}))
	defer server.Close()

	c := NewCanvasChecker()
	result := c.CheckCanvasAccess(server.URL, "test-canvas", "valid-key-12345678")

	if result.Accessible {
		t.Error("CheckCanvasAccess() should fail for 403")
	}

	if result.Message != "Access denied to canvas" {
		t.Errorf("Expected 'Access denied to canvas' message, got: %s", result.Message)
	}
}

func TestCanvasChecker_CheckCanvasAccess_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	c := NewCanvasChecker()
	result := c.CheckCanvasAccess(server.URL, "test-canvas", "valid-key-12345678")

	if result.Accessible {
		t.Error("CheckCanvasAccess() should fail for 500")
	}
}

func TestCanvasChecker_CheckCanvasAccess_EmptyCanvasID(t *testing.T) {
	c := NewCanvasChecker()
	result := c.CheckCanvasAccess("http://localhost", "", "valid-api-key-12345")

	if result.Accessible {
		t.Error("CheckCanvasAccess() should fail for empty canvas ID")
	}

	if result.Message != "Canvas ID is empty" {
		t.Errorf("Expected 'Canvas ID is empty' message, got: %s", result.Message)
	}
}

func TestCanvasChecker_CheckCanvasAccess_UnreachableServer(t *testing.T) {
	c := NewCanvasChecker().WithTimeout(1 * time.Second)
	result := c.CheckCanvasAccess("http://localhost:59999", "test-canvas", "valid-key-12345678")

	if result.Accessible {
		t.Error("CheckCanvasAccess() should fail for unreachable server")
	}
}

func TestCanvasChecker_CheckCanvasAccessWithContext(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	// Create a context that times out quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	c := NewCanvasChecker()
	result := c.CheckCanvasAccessWithContext(ctx, server.URL, "test-canvas", "valid-key-12345678")

	if result.Accessible {
		t.Error("CheckCanvasAccessWithContext() should fail with cancelled context")
	}
}

func TestCanvasChecker_CheckCanvusCanvas(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "widget-1"},
		})
	}))
	defer server.Close()

	tests := []struct {
		name       string
		serverURL  string
		canvasID   string
		apiKey     string
		wantAccess bool
	}{
		{
			name:       "all env vars set",
			serverURL:  server.URL,
			canvasID:   "test-canvas",
			apiKey:     "valid-key-12345678",
			wantAccess: true,
		},
		{
			name:       "missing server URL",
			serverURL:  "",
			canvasID:   "test-canvas",
			apiKey:     "valid-key-12345678",
			wantAccess: false,
		},
		{
			name:       "missing canvas ID",
			serverURL:  server.URL,
			canvasID:   "",
			apiKey:     "valid-key-12345678",
			wantAccess: false,
		},
		{
			name:       "missing API key",
			serverURL:  server.URL,
			canvasID:   "test-canvas",
			apiKey:     "",
			wantAccess: false,
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

			c := NewCanvasChecker()
			result := c.CheckCanvusCanvas()

			if result.Accessible != tt.wantAccess {
				t.Errorf("CheckCanvusCanvas() Accessible = %v, want %v: %v", result.Accessible, tt.wantAccess, result.Error)
			}
		})
	}
}

func TestCanvasChecker_IsAccessible(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	c := NewCanvasChecker()

	if !c.IsAccessible(server.URL, "test-canvas", "valid-key-12345678") {
		t.Error("IsAccessible() should return true for accessible canvas")
	}

	if c.IsAccessible("http://localhost:59999", "test-canvas", "valid-key-12345678") {
		t.Error("IsAccessible() should return false for unreachable server")
	}
}

func TestCanvasChecker_GetWidgetCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"id": "widget-1"},
			{"id": "widget-2"},
			{"id": "widget-3"},
			{"id": "widget-4"},
			{"id": "widget-5"},
		})
	}))
	defer server.Close()

	c := NewCanvasChecker()

	count := c.GetWidgetCount(server.URL, "test-canvas", "valid-key-12345678")
	if count != 5 {
		t.Errorf("GetWidgetCount() = %d, want 5", count)
	}

	// Test error case
	count = c.GetWidgetCount("http://localhost:59999", "test-canvas", "valid-key-12345678")
	if count != -1 {
		t.Errorf("GetWidgetCount() should return -1 for error, got %d", count)
	}
}

func TestCanvasChecker_BuilderPattern(t *testing.T) {
	c := NewCanvasChecker().
		WithTimeout(15 * time.Second).
		WithAllowSelfSignedCerts(true)

	if c.timeout != 15*time.Second {
		t.Errorf("WithTimeout() did not set timeout correctly")
	}

	if !c.allowSelfSignedCerts {
		t.Errorf("WithAllowSelfSignedCerts() did not set flag correctly")
	}
}
