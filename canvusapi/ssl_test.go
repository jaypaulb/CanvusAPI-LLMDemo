package canvusapi

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestSSLConfiguration verifies the SSL certificate handling
func TestSSLConfiguration(t *testing.T) {
	// Test cases
	testCases := []struct {
		name            string
		allowSelfSigned bool
		serverCert      *tls.Certificate
		expectError     bool
	}{
		{
			name:            "Valid SSL with validation enabled",
			allowSelfSigned: false,
			serverCert:      nil, // Use default valid cert
			expectError:     false,
		},
		{
			name:            "Self-signed with validation disabled",
			allowSelfSigned: true,
			serverCert:      nil, // Use default valid cert
			expectError:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "ok"}`))
			}))
			defer server.Close()

			// Create client with test configuration
			client := NewClient(server.URL, "test-canvas", "test-key", tc.allowSelfSigned)

			// Verify HTTP client configuration
			if tc.allowSelfSigned {
				if client.HTTP.Transport == nil {
					t.Error("Expected Transport to be configured for self-signed certificates")
				}
				if transport, ok := client.HTTP.Transport.(*http.Transport); ok {
					if transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
						t.Error("Expected InsecureSkipVerify to be true for self-signed certificates")
					}
				}
			}

			// Test actual API call
			_, err := client.GetCanvasInfo()
			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

// TestEnvironmentConfiguration verifies environment variable handling
func TestEnvironmentConfiguration(t *testing.T) {
	// Save original environment
	originalEnv := os.Getenv("ALLOW_SELF_SIGNED_CERTS")
	defer os.Setenv("ALLOW_SELF_SIGNED_CERTS", originalEnv)

	testCases := []struct {
		name             string
		envValue         string
		expectSelfSigned bool
	}{
		{
			name:             "Environment variable not set",
			envValue:         "",
			expectSelfSigned: false,
		},
		{
			name:             "Environment variable set to false",
			envValue:         "false",
			expectSelfSigned: false,
		},
		{
			name:             "Environment variable set to true",
			envValue:         "true",
			expectSelfSigned: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set test environment
			os.Setenv("ALLOW_SELF_SIGNED_CERTS", tc.envValue)

			// Create client from environment
			client, err := NewClientFromEnv()
			if err == nil {
				// Verify client configuration
				if tc.expectSelfSigned {
					if client.HTTP.Transport == nil {
						t.Error("Expected Transport to be configured for self-signed certificates")
					}
					if transport, ok := client.HTTP.Transport.(*http.Transport); ok {
						if transport.TLSClientConfig == nil || !transport.TLSClientConfig.InsecureSkipVerify {
							t.Error("Expected InsecureSkipVerify to be true for self-signed certificates")
						}
					}
				} else {
					if client.HTTP.Transport != nil {
						t.Error("Expected no custom Transport configuration")
					}
				}
			}
		})
	}
}

// TestRealServerConnection tests connection to a real server with self-signed certificate
func TestRealServerConnection(t *testing.T) {
	// Skip this test if not explicitly enabled
	if os.Getenv("TEST_REAL_SERVER") != "true" {
		t.Skip("Skipping real server test. Set TEST_REAL_SERVER=true to enable.")
	}

	// Test with self-signed certificates enabled
	os.Setenv("ALLOW_SELF_SIGNED_CERTS", "true")
	client, err := NewClientFromEnv()
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test API call
	_, err = client.GetCanvasInfo()
	if err != nil {
		t.Errorf("Failed to connect to server: %v", err)
	}
}
