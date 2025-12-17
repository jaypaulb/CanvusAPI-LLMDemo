package ocrprocessor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go_backend/logging"
)

// testLogger creates a logger for testing
func testLogger(t *testing.T) *logging.Logger {
	t.Helper()
	// Create a temp file for the test log
	tmpDir := os.TempDir()
	logFile := filepath.Join(tmpDir, "ocrprocessor_test.log")
	logger, err := logging.NewLogger(true, logFile)
	if err != nil {
		t.Fatalf("failed to create test logger: %v", err)
	}
	return logger
}

func TestNewVisionClient(t *testing.T) {
	logger := testLogger(t)
	httpClient := &http.Client{}

	tests := []struct {
		name        string
		apiKey      string
		httpClient  *http.Client
		logger      *logging.Logger
		wantErr     bool
		errContains string
	}{
		{
			name:       "valid inputs",
			apiKey:     "AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY", // Valid format
			httpClient: httpClient,
			logger:     logger,
			wantErr:    false,
		},
		{
			name:        "nil HTTP client",
			apiKey:      "AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY",
			httpClient:  nil,
			logger:      logger,
			wantErr:     true,
			errContains: "HTTP client cannot be nil",
		},
		{
			name:        "nil logger",
			apiKey:      "AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY",
			httpClient:  httpClient,
			logger:      nil,
			wantErr:     true,
			errContains: "logger cannot be nil",
		},
		{
			name:        "empty API key",
			apiKey:      "",
			httpClient:  httpClient,
			logger:      logger,
			wantErr:     true,
			errContains: "empty",
		},
		{
			name:        "too short API key",
			apiKey:      "short",
			httpClient:  httpClient,
			logger:      logger,
			wantErr:     true,
			errContains: "too short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewVisionClient(tt.apiKey, tt.httpClient, tt.logger, DefaultVisionClientConfig())
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewVisionClient() expected error, got nil")
				} else if tt.errContains != "" && !containsStr(err.Error(), tt.errContains) {
					t.Errorf("NewVisionClient() error = %q, want to contain %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewVisionClient() unexpected error: %v", err)
				}
				if client == nil {
					t.Error("NewVisionClient() returned nil client without error")
				}
			}
		})
	}
}

func TestDefaultVisionClientConfig(t *testing.T) {
	config := DefaultVisionClientConfig()

	if config.Endpoint == "" {
		t.Error("DefaultVisionClientConfig().Endpoint should not be empty")
	}
	if config.Endpoint != "https://vision.googleapis.com/v1/images:annotate" {
		t.Errorf("DefaultVisionClientConfig().Endpoint = %q, want Google Vision endpoint", config.Endpoint)
	}
	if config.FeatureType == "" {
		t.Error("DefaultVisionClientConfig().FeatureType should not be empty")
	}
	if config.Timeout <= 0 {
		t.Error("DefaultVisionClientConfig().Timeout should be positive")
	}
	if config.MaxResults <= 0 {
		t.Error("DefaultVisionClientConfig().MaxResults should be positive")
	}
}

func TestVisionClient_ExtractText_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Return mock response
		response := visionResponse{
			Responses: []visionResponseItem{
				{
					FullTextAnnotation: struct {
						Text string `json:"text"`
					}{
						Text: "Hello, World!\nThis is extracted text.",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultVisionClientConfig()
	config.Endpoint = server.URL

	client, err := NewVisionClient("AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY", server.Client(), logger, config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	// Test with minimal image data
	imageData := []byte{0x89, 0x50, 0x4E, 0x47} // PNG header

	result, err := client.ExtractText(context.Background(), imageData)
	if err != nil {
		t.Fatalf("ExtractText() unexpected error: %v", err)
	}

	if result.Text != "Hello, World!\nThis is extracted text." {
		t.Errorf("ExtractText().Text = %q, want extracted text", result.Text)
	}
	if result.ProcessingTime <= 0 {
		t.Error("ExtractText().ProcessingTime should be positive")
	}
}

func TestVisionClient_ExtractText_EmptyImage(t *testing.T) {
	logger := testLogger(t)
	client, err := NewVisionClient("AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY", &http.Client{}, logger, DefaultVisionClientConfig())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.ExtractText(context.Background(), []byte{})
	if err == nil {
		t.Error("ExtractText() with empty image should return error")
	}
	if !containsStr(err.Error(), "empty") {
		t.Errorf("ExtractText() error = %q, want to contain 'empty'", err.Error())
	}
}

func TestVisionClient_ExtractText_NoTextFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := visionResponse{
			Responses: []visionResponseItem{
				{
					FullTextAnnotation: struct {
						Text string `json:"text"`
					}{
						Text: "", // No text found
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultVisionClientConfig()
	config.Endpoint = server.URL

	client, err := NewVisionClient("AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY", server.Client(), logger, config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.ExtractText(context.Background(), []byte{0x89, 0x50, 0x4E, 0x47})
	if err != ErrNoTextFound {
		t.Errorf("ExtractText() error = %v, want ErrNoTextFound", err)
	}
}

func TestVisionClient_ExtractText_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := visionResponse{
			Responses: []visionResponseItem{
				{
					Error: struct {
						Code    int    `json:"code"`
						Message string `json:"message"`
					}{
						Code:    400,
						Message: "Invalid image data",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultVisionClientConfig()
	config.Endpoint = server.URL

	client, err := NewVisionClient("AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY", server.Client(), logger, config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.ExtractText(context.Background(), []byte{0x89, 0x50, 0x4E, 0x47})
	if err == nil {
		t.Error("ExtractText() with API error should return error")
	}
	if !containsStr(err.Error(), "Invalid image data") {
		t.Errorf("ExtractText() error = %q, want to contain API error message", err.Error())
	}
}

func TestVisionClient_ExtractText_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultVisionClientConfig()
	config.Endpoint = server.URL

	client, err := NewVisionClient("AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY", server.Client(), logger, config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.ExtractText(context.Background(), []byte{0x89, 0x50, 0x4E, 0x47})
	if err == nil {
		t.Error("ExtractText() with HTTP error should return error")
	}
	if !containsStr(err.Error(), "500") {
		t.Errorf("ExtractText() error = %q, want to contain status code", err.Error())
	}
}

func TestVisionClient_ExtractText_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := visionResponse{
			Responses: []visionResponseItem{}, // Empty responses
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultVisionClientConfig()
	config.Endpoint = server.URL

	client, err := NewVisionClient("AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY", server.Client(), logger, config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	_, err = client.ExtractText(context.Background(), []byte{0x89, 0x50, 0x4E, 0x47})
	if err != ErrEmptyResponse {
		t.Errorf("ExtractText() error = %v, want ErrEmptyResponse", err)
	}
}

func TestVisionClient_ExtractText_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate slow response
		json.NewEncoder(w).Encode(visionResponse{})
	}))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultVisionClientConfig()
	config.Endpoint = server.URL

	client, err := NewVisionClient("AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY", server.Client(), logger, config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = client.ExtractText(ctx, []byte{0x89, 0x50, 0x4E, 0x47})
	if err == nil {
		t.Error("ExtractText() with cancelled context should return error")
	}
}

func TestVisionClient_ValidateAPIKey_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := visionResponse{
			Responses: []visionResponseItem{
				{}, // Empty but valid response
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultVisionClientConfig()
	config.Endpoint = server.URL

	client, err := NewVisionClient("AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY", server.Client(), logger, config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.ValidateAPIKey(context.Background())
	if err != nil {
		t.Errorf("ValidateAPIKey() unexpected error: %v", err)
	}
}

func TestVisionClient_ValidateAPIKey_InvalidKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("API key not valid"))
	}))
	defer server.Close()

	logger := testLogger(t)
	config := DefaultVisionClientConfig()
	config.Endpoint = server.URL

	client, err := NewVisionClient("AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY", server.Client(), logger, config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.ValidateAPIKey(context.Background())
	if err == nil {
		t.Error("ValidateAPIKey() with invalid key should return error")
	}
	if !containsStr(err.Error(), "invalid API key") {
		t.Errorf("ValidateAPIKey() error = %q, want to contain 'invalid API key'", err.Error())
	}
}

func TestVisionClient_GetMaskedAPIKey(t *testing.T) {
	logger := testLogger(t)
	client, err := NewVisionClient("AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY", &http.Client{}, logger, DefaultVisionClientConfig())
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	masked := client.GetMaskedAPIKey()
	if masked == "AIzaSyD-9tSrke72PouQMnMX-a7eZSW0jkFMBWY" {
		t.Error("GetMaskedAPIKey() should not return the full key")
	}
	if !containsStr(masked, "****") {
		t.Errorf("GetMaskedAPIKey() = %q, want to contain masked characters", masked)
	}
}

// containsStr is a helper to check if a string contains a substring
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
