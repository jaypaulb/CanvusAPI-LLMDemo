package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"go_backend/canvusapi"
	"go_backend/core"
	"go_backend/logging"
)

// sessionCookieName is the name of the session cookie used by the auth package.
const sessionCookieName = "session_id"

// createTestLoggerMain creates a logger for testing that writes to a temp file.
func createTestLoggerMain(t *testing.T) *logging.Logger {
	t.Helper()
	// Create temp file for log output
	tmpFile, err := os.CreateTemp("", "main_test_*.log")
	if err != nil {
		t.Fatalf("failed to create temp log file: %v", err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	logger, err := logging.NewLogger(true, tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return logger
}

// TestInitializeSDRuntimeNotConfigured tests SD runtime initialization when not configured.
func TestInitializeSDRuntimeNotConfigured(t *testing.T) {
	// Ensure SD_MODEL_PATH is not set
	os.Unsetenv("SD_MODEL_PATH")

	logger := createTestLoggerMain(t)
	defer logger.Sync()

	client := canvusapi.NewClient("http://localhost:8080", "test-canvas", "test-key", false)
	config := &core.Config{
		DownloadsDir: os.TempDir(),
	}

	// DOING: Call initializeSDRuntime without SD_MODEL_PATH configured
	// EXPECT: Should return (nil, nil, nil) since SD is not configured
	pool, processor, err := initializeSDRuntime(logger, client, config)

	// RESULT: Check return values
	if err != nil {
		t.Errorf("expected no error when SD not configured, got: %v", err)
	}
	if pool != nil {
		t.Error("expected nil pool when SD not configured")
	}
	if processor != nil {
		t.Error("expected nil processor when SD not configured")
	}
}

// TestInitializeSDRuntimeMissingModel tests SD runtime initialization when model file is missing.
func TestInitializeSDRuntimeMissingModel(t *testing.T) {
	// Set SD_MODEL_PATH to a non-existent file
	os.Setenv("SD_MODEL_PATH", "/nonexistent/model.safetensors")
	defer os.Unsetenv("SD_MODEL_PATH")

	logger := createTestLoggerMain(t)
	defer logger.Sync()

	client := canvusapi.NewClient("http://localhost:8080", "test-canvas", "test-key", false)
	config := &core.Config{
		DownloadsDir: os.TempDir(),
	}

	// DOING: Call initializeSDRuntime with non-existent model path
	// EXPECT: Should return error about missing model file
	pool, processor, err := initializeSDRuntime(logger, client, config)

	// RESULT: Check return values
	if err == nil {
		t.Error("expected error when model file not found")
	}
	if pool != nil {
		pool.Close() // Clean up if somehow created
		t.Error("expected nil pool when model not found")
	}
	if processor != nil {
		t.Error("expected nil processor when model not found")
	}
}

// TestInitializeSDRuntimeConfigParsing tests that SD configuration is properly loaded.
func TestInitializeSDRuntimeConfigParsing(t *testing.T) {
	tests := []struct {
		name             string
		envVars          map[string]string
		expectedDisabled bool // true if SD should be disabled (no model path)
	}{
		{
			name:             "empty model path disables SD",
			envVars:          map[string]string{},
			expectedDisabled: true,
		},
		{
			name: "model path enables SD (but will fail due to missing file)",
			envVars: map[string]string{
				"SD_MODEL_PATH": "/some/path/model.safetensors",
			},
			expectedDisabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear relevant env vars
			os.Unsetenv("SD_MODEL_PATH")
			os.Unsetenv("SD_IMAGE_SIZE")
			os.Unsetenv("SD_INFERENCE_STEPS")
			os.Unsetenv("SD_GUIDANCE_SCALE")
			os.Unsetenv("SD_MAX_CONCURRENT")
			os.Unsetenv("SD_TIMEOUT_SECONDS")

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			logger := createTestLoggerMain(t)
			defer logger.Sync()

			client := canvusapi.NewClient("http://localhost:8080", "test-canvas", "test-key", false)
			config := &core.Config{
				DownloadsDir: os.TempDir(),
			}

			pool, processor, err := initializeSDRuntime(logger, client, config)

			if tt.expectedDisabled {
				// SD should be disabled (nil, nil, nil)
				if err != nil {
					t.Errorf("expected no error for disabled SD, got: %v", err)
				}
				if pool != nil || processor != nil {
					t.Error("expected nil pool and processor for disabled SD")
				}
			} else {
				// SD should be enabled but will fail due to missing model
				if err == nil {
					t.Error("expected error for missing model file")
					if pool != nil {
						pool.Close()
					}
				}
			}
		})
	}
}

// TestSplitAndTrim tests the splitAndTrim helper function.
func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		input    string
		sep      string
		expected []string
	}{
		{"", ",", nil},
		{"a,b,c", ",", []string{"a", "b", "c"}},
		{" a , b , c ", ",", []string{"a", "b", "c"}},
		{"  a  ", ",", []string{"a"}},
		{"a,,b", ",", []string{"a", "b"}},
		{"a", ",", []string{"a"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitAndTrim(tt.input, tt.sep)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d parts, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("part %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

// TestTrimSpace tests the trimSpace helper function.
func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"a", "a"},
		{" a", "a"},
		{"a ", "a"},
		{" a ", "a"},
		{"  hello world  ", "hello world"},
		{"\t\n a \r\n", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := trimSpace(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// =============================================================================
// Web Server Setup Tests
// =============================================================================

// TestSetupWebServer tests that the web server is configured correctly with auth.
func TestSetupWebServer(t *testing.T) {
	logger := createTestLoggerMain(t)
	defer logger.Sync()

	config := &core.Config{
		WebUIPassword: "test-password-123",
		Port:          8080,
	}

	// DOING: Call setupWebServer with valid config
	// EXPECT: Should return configured server and no error
	server, err := setupWebServer(config, logger)

	// RESULT: Check return values
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if server == nil {
		t.Fatal("expected non-nil server")
	}

	// Verify server configuration
	if server.Addr != ":8080" {
		t.Errorf("expected addr :8080, got %s", server.Addr)
	}
	if server.ReadTimeout != DefaultReadTimeout {
		t.Errorf("expected read timeout %v, got %v", DefaultReadTimeout, server.ReadTimeout)
	}
	if server.WriteTimeout != DefaultWriteTimeout {
		t.Errorf("expected write timeout %v, got %v", DefaultWriteTimeout, server.WriteTimeout)
	}
	if server.IdleTimeout != DefaultIdleTimeout {
		t.Errorf("expected idle timeout %v, got %v", DefaultIdleTimeout, server.IdleTimeout)
	}
	if server.Handler == nil {
		t.Error("expected non-nil handler")
	}
}

// TestSetupWebServerEmptyPassword tests that setup fails with empty password.
func TestSetupWebServerEmptyPassword(t *testing.T) {
	logger := createTestLoggerMain(t)
	defer logger.Sync()

	config := &core.Config{
		WebUIPassword: "", // Empty password
		Port:          8080,
	}

	// DOING: Call setupWebServer with empty password
	// EXPECT: Should return error since password is required
	server, err := setupWebServer(config, logger)

	// RESULT: Check return values - bcrypt should reject empty password
	if err == nil {
		t.Error("expected error for empty password")
		if server != nil {
			// Cleanup if server was created
		}
	}
}

// =============================================================================
// Handler Tests
// =============================================================================

// TestDashboardHandler tests the dashboard handler serves the correct content.
func TestDashboardHandler(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectHTML     bool
	}{
		{
			name:           "root path returns dashboard",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectHTML:     true,
		},
		{
			name:           "non-root path returns 404",
			path:           "/other",
			expectedStatus: http.StatusNotFound,
			expectHTML:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()

			dashboardHandler(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectHTML {
				contentType := rr.Header().Get("Content-Type")
				if !strings.Contains(contentType, "text/html") {
					t.Errorf("expected Content-Type text/html, got %s", contentType)
				}
				body := rr.Body.String()
				if !strings.Contains(body, "CanvusLocalLLM Dashboard") {
					t.Error("expected dashboard title in response body")
				}
				if !strings.Contains(body, "System Running") {
					t.Error("expected status indicator in response body")
				}
			}
		})
	}
}

// TestAPIStatusHandler tests the API status endpoint.
func TestAPIStatusHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rr := httptest.NewRecorder()

	apiStatusHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `"status":"running"`) {
		t.Errorf("expected status:running in response, got %s", body)
	}
}

// TestAPICanvasesHandler tests the API canvases endpoint.
func TestAPICanvasesHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/canvases", nil)
	rr := httptest.NewRecorder()

	apiCanvasesHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `"canvases"`) {
		t.Errorf("expected canvases field in response, got %s", body)
	}
}

// TestAPITasksHandler tests the API tasks endpoint.
func TestAPITasksHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rr := httptest.NewRecorder()

	apiTasksHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `"tasks"`) {
		t.Errorf("expected tasks field in response, got %s", body)
	}
}

// TestAPIMetricsHandler tests the API metrics endpoint.
func TestAPIMetricsHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	rr := httptest.NewRecorder()

	apiMetricsHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `"metrics"`) {
		t.Errorf("expected metrics field in response, got %s", body)
	}
}

// TestAPIGPUHandler tests the API GPU endpoint.
func TestAPIGPUHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/gpu", nil)
	rr := httptest.NewRecorder()

	apiGPUHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, `"gpu"`) {
		t.Errorf("expected gpu field in response, got %s", body)
	}
	if !strings.Contains(body, `"available":false`) {
		t.Errorf("expected available:false in response, got %s", body)
	}
}

// =============================================================================
// Integration Tests for Web Server Auth
// =============================================================================

// TestWebServerAuthIntegration tests the full auth flow through the web server.
func TestWebServerAuthIntegration(t *testing.T) {
	logger := createTestLoggerMain(t)
	defer logger.Sync()

	config := &core.Config{
		WebUIPassword: "integration-test-password",
		Port:          0, // Will use test server, not actual port
	}

	server, err := setupWebServer(config, logger)
	if err != nil {
		t.Fatalf("failed to setup web server: %v", err)
	}

	// Create test server
	ts := httptest.NewServer(server.Handler)
	defer ts.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
		Timeout: 5 * time.Second,
	}

	// Step 1: Access protected route without auth - should get 401
	t.Run("access_protected_without_auth", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 without auth, got %d", resp.StatusCode)
		}
	})

	// Step 2: Access API endpoint without auth - should get 401
	t.Run("access_api_without_auth", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/api/status")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 for API without auth, got %d", resp.StatusCode)
		}
	})

	// Step 3: Access login page - should succeed
	t.Run("access_login_page", func(t *testing.T) {
		resp, err := client.Get(ts.URL + "/login")
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for login page, got %d", resp.StatusCode)
		}
	})

	// Step 4: Login with correct password - should get session cookie and redirect
	var sessionCookie *http.Cookie
	t.Run("login_with_correct_password", func(t *testing.T) {
		form := url.Values{}
		form.Set("password", "integration-test-password")

		resp, err := client.PostForm(ts.URL+"/login", form)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("expected 303 after login, got %d", resp.StatusCode)
		}

		// Find session cookie - cookie name is "session_id"
		for _, cookie := range resp.Cookies() {
			if cookie.Name == sessionCookieName && cookie.MaxAge > 0 {
				sessionCookie = cookie
				break
			}
		}
		if sessionCookie == nil {
			t.Error("expected session cookie after login")
		}
	})

	// Step 5: Access protected route with session - should succeed
	if sessionCookie != nil {
		t.Run("access_protected_with_session", func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
			req.AddCookie(sessionCookie)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200 with valid session, got %d", resp.StatusCode)
			}
		})

		// Step 6: Access API with session - should succeed
		t.Run("access_api_with_session", func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/status", nil)
			req.AddCookie(sessionCookie)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200 for API with session, got %d", resp.StatusCode)
			}
		})

		// Step 7: Logout - should clear session
		t.Run("logout", func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPost, ts.URL+"/logout", nil)
			req.AddCookie(sessionCookie)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusSeeOther {
				t.Errorf("expected 303 after logout, got %d", resp.StatusCode)
			}

			// Should have a clear cookie (MaxAge < 0 or MaxAge == 0 means delete)
			for _, cookie := range resp.Cookies() {
				if cookie.Name == sessionCookieName && cookie.MaxAge < 0 {
					// Found clear cookie
					return
				}
			}
			t.Error("expected clear session cookie after logout")
		})

		// Step 8: Access protected route after logout - should fail
		t.Run("access_protected_after_logout", func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
			req.AddCookie(sessionCookie) // Cookie still present but session destroyed

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("expected 401 after logout, got %d", resp.StatusCode)
			}
		})
	}

	// Step 9: Login with wrong password - should fail
	t.Run("login_with_wrong_password", func(t *testing.T) {
		form := url.Values{}
		form.Set("password", "wrong-password")

		resp, err := client.PostForm(ts.URL+"/login", form)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		// Should redirect back to login with error
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("expected 303 redirect on wrong password, got %d", resp.StatusCode)
		}

		// Should NOT have valid session cookie
		for _, cookie := range resp.Cookies() {
			if cookie.Name == sessionCookieName && cookie.MaxAge > 0 {
				t.Error("should not get session cookie with wrong password")
			}
		}
	})
}

// TestDefaultTimeoutConstants verifies the default timeout constants are reasonable.
func TestDefaultTimeoutConstants(t *testing.T) {
	if DefaultReadTimeout < 1*time.Second {
		t.Error("read timeout too short")
	}
	if DefaultWriteTimeout < 1*time.Second {
		t.Error("write timeout too short")
	}
	if DefaultIdleTimeout < 1*time.Second {
		t.Error("idle timeout too short")
	}
	if DefaultShutdownTimeout < 1*time.Second {
		t.Error("shutdown timeout too short")
	}

	// Verify reasonable upper bounds
	if DefaultReadTimeout > 5*time.Minute {
		t.Error("read timeout too long")
	}
	if DefaultShutdownTimeout > 1*time.Minute {
		t.Error("shutdown timeout too long")
	}
}
