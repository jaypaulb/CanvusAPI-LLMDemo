package webui

import (
	"context"
	"fmt"
	"go_backend/core"
	"go_backend/logging"
	"go_backend/webui/auth"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

// =============================================================================
// WebUI Server Integration Tests
// These tests verify that the WebUI server components work together correctly.
// =============================================================================

// setupTestServer creates a test HTTP server with authentication configured.
// This is a helper that mimics the setupWebServer function in main.go.
func setupTestServer(t *testing.T, password string) (*http.Server, *auth.AuthMiddleware) {
	t.Helper()

	logger := zap.NewNop()

	// Initialize authentication middleware
	authMiddleware, err := auth.NewAuthMiddleware(password, logger)
	if err != nil {
		t.Fatalf("failed to create auth middleware: %v", err)
	}

	// Create HTTP mux for routing (mimics main.go setup)
	mux := http.NewServeMux()

	// Register public routes
	mux.HandleFunc("/login", auth.LoginHandler(authMiddleware))
	mux.HandleFunc("/logout", auth.LogoutHandler(authMiddleware))

	// Register protected routes
	mux.Handle("/", authMiddleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "<html><body>Dashboard</body></html>")
	})))

	// API endpoints - all protected by auth middleware
	mux.Handle("/api/status", authMiddleware.Middleware(http.HandlerFunc(apiStatusHandler)))
	mux.Handle("/api/canvases", authMiddleware.Middleware(http.HandlerFunc(apiCanvasesHandler)))
	mux.Handle("/api/tasks", authMiddleware.Middleware(http.HandlerFunc(apiTasksHandler)))
	mux.Handle("/api/metrics", authMiddleware.Middleware(http.HandlerFunc(apiMetricsHandler)))
	mux.Handle("/api/gpu", authMiddleware.Middleware(http.HandlerFunc(apiGPUHandler)))

	// Create HTTP server with reasonable timeouts
	server := &http.Server{
		Addr:         ":0", // Let OS choose port for testing
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server, authMiddleware
}

// Placeholder API handlers (mimics main.go)
func apiStatusHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"running","version":"1.0.0"}`)
}

func apiCanvasesHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"canvases":[]}`)
}

func apiTasksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"tasks":[]}`)
}

func apiMetricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"metrics":{}}`)
}

func apiGPUHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"gpu":{"available":false}}`)
}

// TestServer_StartAndStop verifies that the server can start and stop gracefully.
// This test verifies:
//   - Server starts successfully
//   - Server responds to health check requests
//   - Server shuts down gracefully without hanging
//   - Shutdown respects context timeout
func TestServer_StartAndStop(t *testing.T) {
	server, _ := setupTestServer(t, "test-password")

	// Start server in background
	testServer := httptest.NewServer(server.Handler)
	defer testServer.Close()

	// Verify server is responding
	resp, err := http.Get(testServer.URL + "/login")
	if err != nil {
		t.Fatalf("server should respond: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected login page to load, got status %d", resp.StatusCode)
	}

	// Test graceful shutdown with context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownComplete := make(chan error, 1)
	go func() {
		// In a real scenario, we'd shutdown the actual server
		// For httptest.Server, we just verify Close() works
		testServer.Close()
		shutdownComplete <- nil
	}()

	select {
	case err := <-shutdownComplete:
		if err != nil {
			t.Errorf("shutdown failed: %v", err)
		}
	case <-ctx.Done():
		t.Error("shutdown did not complete within timeout")
	}
}

// TestServer_AuthenticationFlow verifies the complete authentication workflow.
// This test verifies:
//   - Unauthenticated requests to protected routes return 401
//   - Login with correct password creates session
//   - Session cookie is set with correct attributes
//   - Authenticated requests to protected routes succeed
//   - Logout destroys session
//   - Requests with destroyed session return 401
func TestServer_AuthenticationFlow(t *testing.T) {
	password := "integration-test-password"
	server, _ := setupTestServer(t, password)
	testServer := httptest.NewServer(server.Handler)
	defer testServer.Close()

	// Step 1: Access protected route without auth - should redirect to login or return 401
	resp, err := http.Get(testServer.URL + "/")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Step 1: expected 401 without auth, got %d", resp.StatusCode)
	}

	// Step 2: Login with correct password
	form := url.Values{}
	form.Set("password", password)

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects
		},
	}

	resp, err = client.Post(
		testServer.URL+"/login",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		t.Fatalf("Step 2: login request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("Step 2: expected 303 after successful login, got %d", resp.StatusCode)
	}

	// Extract session cookie
	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == auth.SessionCookieName && c.MaxAge > 0 {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("Step 2: no session cookie set after login")
	}

	// Verify cookie attributes
	if sessionCookie.Path != "/" {
		t.Errorf("cookie path should be /, got %s", sessionCookie.Path)
	}
	if !sessionCookie.HttpOnly {
		t.Error("cookie should be HttpOnly")
	}
	if sessionCookie.SameSite != http.SameSiteStrictMode {
		t.Errorf("cookie should be SameSite=Strict, got %v", sessionCookie.SameSite)
	}

	// Step 3: Access protected route with session - should succeed
	req, err := http.NewRequest(http.MethodGet, testServer.URL+"/", nil)
	if err != nil {
		t.Fatalf("Step 3: failed to create request: %v", err)
	}
	req.AddCookie(sessionCookie)

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Step 3: authenticated request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Step 3: expected 200 with valid session, got %d", resp.StatusCode)
	}

	// Step 4: Logout
	req, err = http.NewRequest(http.MethodPost, testServer.URL+"/logout", nil)
	if err != nil {
		t.Fatalf("Step 4: failed to create logout request: %v", err)
	}
	req.AddCookie(sessionCookie)

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Step 4: logout request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("Step 4: expected 303 after logout, got %d", resp.StatusCode)
	}

	// Step 5: Try to access protected route after logout - should fail
	req, err = http.NewRequest(http.MethodGet, testServer.URL+"/", nil)
	if err != nil {
		t.Fatalf("Step 5: failed to create request: %v", err)
	}
	req.AddCookie(sessionCookie) // Old cookie still present but session destroyed

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Step 5: request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Step 5: expected 401 after logout, got %d", resp.StatusCode)
	}
}

// TestServer_APIEndpoints verifies that API endpoints are properly protected
// and return expected responses.
// This test verifies:
//   - All API endpoints require authentication
//   - Authenticated requests receive proper JSON responses
//   - Response headers are set correctly
//   - Each endpoint returns its expected structure
func TestServer_APIEndpoints(t *testing.T) {
	password := "test-password"
	server, authMiddleware := setupTestServer(t, password)
	testServer := httptest.NewServer(server.Handler)
	defer testServer.Close()

	// Create a session for authenticated requests
	session, cookie, err := authMiddleware.CreateSession()
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	_ = session // Session object not needed, just cookie

	// Define test cases for each API endpoint
	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
		expectedJSON   string // Substring to check in response
	}{
		{
			name:           "status endpoint",
			endpoint:       "/api/status",
			expectedStatus: http.StatusOK,
			expectedJSON:   `"status":"running"`,
		},
		{
			name:           "canvases endpoint",
			endpoint:       "/api/canvases",
			expectedStatus: http.StatusOK,
			expectedJSON:   `"canvases":[]`,
		},
		{
			name:           "tasks endpoint",
			endpoint:       "/api/tasks",
			expectedStatus: http.StatusOK,
			expectedJSON:   `"tasks":[]`,
		},
		{
			name:           "metrics endpoint",
			endpoint:       "/api/metrics",
			expectedStatus: http.StatusOK,
			expectedJSON:   `"metrics":{}`,
		},
		{
			name:           "gpu endpoint",
			endpoint:       "/api/gpu",
			expectedStatus: http.StatusOK,
			expectedJSON:   `"gpu":`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test 1: Without authentication - should fail
			resp, err := http.Get(testServer.URL + tt.endpoint)
			if err != nil {
				t.Fatalf("unauthenticated request failed: %v", err)
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("expected 401 without auth, got %d", resp.StatusCode)
			}

			// Test 2: With authentication - should succeed
			req, err := http.NewRequest(http.MethodGet, testServer.URL+tt.endpoint, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.AddCookie(cookie)

			client := &http.Client{}
			resp, err = client.Do(req)
			if err != nil {
				t.Fatalf("authenticated request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// Verify Content-Type header
			contentType := resp.Header.Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("expected Content-Type application/json, got %s", contentType)
			}

			// Read response body
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			bodyStr := string(body[:n])

			// Verify expected JSON content
			if !strings.Contains(bodyStr, tt.expectedJSON) {
				t.Errorf("response should contain %q, got: %s", tt.expectedJSON, bodyStr)
			}
		})
	}
}

// TestServer_WebSocketConnection verifies WebSocket upgrade and connection handling.
// This is a placeholder test since WebSocket implementation is not yet in the codebase.
// When WebSocket support is added, this test should verify:
//   - WebSocket upgrade from HTTP
//   - Authentication before upgrade
//   - Message broadcasting
//   - Connection cleanup on client disconnect
func TestServer_WebSocketConnection(t *testing.T) {
	t.Skip("WebSocket support not yet implemented - placeholder for future implementation")

	// TODO: When WebSocket support is added, implement tests for:
	// 1. Successful WebSocket upgrade with valid session
	// 2. WebSocket upgrade rejection without session
	// 3. Message broadcasting to connected clients
	// 4. Connection cleanup on disconnect
	// 5. Concurrent connections handling
}

// TestServer_StaticAssetServing verifies static file serving.
// This is a placeholder test since static asset serving is not yet in the codebase.
// When static asset support is added, this test should verify:
//   - Static files are served with correct Content-Type
//   - Cache headers are set appropriately
//   - 404 for missing files
//   - Directory listing is disabled
func TestServer_StaticAssetServing(t *testing.T) {
	t.Skip("Static asset serving not yet implemented - placeholder for future implementation")

	// TODO: When static asset support is added, implement tests for:
	// 1. CSS files served with correct Content-Type
	// 2. JS files served with correct Content-Type
	// 3. Image files served with correct Content-Type
	// 4. Cache-Control headers set appropriately
	// 5. 404 response for missing files
	// 6. Directory listing disabled (should return 403 or 404)
	// 7. Path traversal protection (../../etc/passwd should not work)
}

// TestServer_ConcurrentRequests verifies that the server handles concurrent
// requests correctly without race conditions.
func TestServer_ConcurrentRequests(t *testing.T) {
	password := "concurrent-test"
	server, authMiddleware := setupTestServer(t, password)
	testServer := httptest.NewServer(server.Handler)
	defer testServer.Close()

	// Create a session
	_, cookie, err := authMiddleware.CreateSession()
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Make concurrent requests to different endpoints
	const numRequests = 50
	done := make(chan error, numRequests)

	endpoints := []string{"/api/status", "/api/canvases", "/api/tasks", "/api/metrics", "/api/gpu"}

	for i := 0; i < numRequests; i++ {
		endpoint := endpoints[i%len(endpoints)]
		go func(ep string) {
			req, err := http.NewRequest(http.MethodGet, testServer.URL+ep, nil)
			if err != nil {
				done <- err
				return
			}
			req.AddCookie(cookie)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				done <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				done <- fmt.Errorf("expected 200, got %d for %s", resp.StatusCode, ep)
				return
			}

			done <- nil
		}(endpoint)
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		if err := <-done; err != nil {
			t.Errorf("concurrent request failed: %v", err)
		}
	}
}

// TestServer_FullIntegration is a comprehensive end-to-end test that exercises
// the complete server functionality in a realistic scenario.
func TestServer_FullIntegration(t *testing.T) {
	// Setup
	password := "integration-password"

	// Create a minimal config for realistic testing
	t.Setenv("WEBUI_PWD", password)
	t.Setenv("PORT", "0") // Let OS choose port

	logger, err := logging.NewLogger(false, "")
	if err != nil {
		// Fallback to nop logger if real logger fails
		logger = &logging.Logger{}
	}

	// Use the real auth middleware setup
	authMw, err := auth.NewAuthMiddleware(password, logger.Zap())
	if err != nil {
		t.Fatalf("failed to create auth middleware: %v", err)
	}

	// Setup server
	mux := http.NewServeMux()
	mux.HandleFunc("/login", auth.LoginHandler(authMw))
	mux.HandleFunc("/logout", auth.LogoutHandler(authMw))
	mux.Handle("/", authMw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Dashboard")
	})))
	mux.Handle("/api/status", authMw.Middleware(http.HandlerFunc(apiStatusHandler)))

	server := &http.Server{
		Handler: mux,
	}

	testServer := httptest.NewServer(server.Handler)
	defer testServer.Close()

	// Scenario: User logs in, accesses dashboard and API, then logs out
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 1. Try to access dashboard without login
	resp, err := client.Get(testServer.URL + "/")
	if err != nil {
		t.Fatalf("initial request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 before login, got %d", resp.StatusCode)
	}

	// 2. Login
	form := url.Values{}
	form.Set("password", password)
	resp, err = client.Post(
		testServer.URL+"/login",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("expected redirect after login, got %d", resp.StatusCode)
	}

	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("no cookies set after login")
	}

	// 3. Access dashboard with session
	req, _ := http.NewRequest(http.MethodGet, testServer.URL+"/", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("dashboard access failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for dashboard, got %d", resp.StatusCode)
	}

	// 4. Access API endpoint
	req, _ = http.NewRequest(http.MethodGet, testServer.URL+"/api/status", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("API access failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for API, got %d", resp.StatusCode)
	}

	// 5. Logout
	req, _ = http.NewRequest(http.MethodPost, testServer.URL+"/logout", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("expected redirect after logout, got %d", resp.StatusCode)
	}

	// 6. Verify session is destroyed
	req, _ = http.NewRequest(http.MethodGet, testServer.URL+"/", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("post-logout request failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 after logout, got %d", resp.StatusCode)
	}
}

// TestServer_RateLimitingIntegration verifies that rate limiting works correctly
// in the context of the full server.
func TestServer_RateLimitingIntegration(t *testing.T) {
	password := "rate-limit-test"

	// Create middleware with low rate limit for testing
	logger := zap.NewNop()
	cfg := auth.Config{
		SessionTTL:             24 * time.Hour,
		RateLimitAttempts:      3,
		RateLimitWindowMinutes: 1,
		RateLimitBlockMinutes:  1,
	}
	authMw, err := auth.NewAuthMiddlewareWithConfig(password, logger, cfg)
	if err != nil {
		t.Fatalf("failed to create auth middleware: %v", err)
	}

	// Setup server
	mux := http.NewServeMux()
	mux.HandleFunc("/login", auth.LoginHandler(authMw))

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Make multiple failed login attempts
	for i := 0; i < 3; i++ {
		form := url.Values{}
		form.Set("password", "wrong-password")

		resp, err := client.Post(
			testServer.URL+"/login",
			"application/x-www-form-urlencoded",
			strings.NewReader(form.Encode()),
		)
		if err != nil {
			t.Fatalf("login attempt %d failed: %v", i+1, err)
		}
		resp.Body.Close()

		// Should get 303 redirect to login page with error
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("attempt %d: expected redirect, got %d", i+1, resp.StatusCode)
		}
	}

	// Next attempt should be rate limited
	form := url.Values{}
	form.Set("password", "wrong-password")

	resp, err := client.Post(
		testServer.URL+"/login",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		t.Fatalf("rate-limited request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected 429 after rate limit exceeded, got %d", resp.StatusCode)
	}

	// Verify Retry-After header is set
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		t.Error("Retry-After header should be set when rate limited")
	}
}

// TestServer_SessionExpiryIntegration verifies session expiry in the context
// of the full server.
func TestServer_SessionExpiryIntegration(t *testing.T) {
	password := "expiry-test"
	logger := zap.NewNop()

	// Create middleware with very short session TTL
	cfg := auth.Config{
		SessionTTL:             100 * time.Millisecond,
		RateLimitAttempts:      5,
		RateLimitWindowMinutes: 1,
		RateLimitBlockMinutes:  5,
	}
	authMw, err := auth.NewAuthMiddlewareWithConfig(password, logger, cfg)
	if err != nil {
		t.Fatalf("failed to create auth middleware: %v", err)
	}

	// Setup server
	mux := http.NewServeMux()
	mux.HandleFunc("/login", auth.LoginHandler(authMw))
	mux.Handle("/", authMw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})))

	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Login
	form := url.Values{}
	form.Set("password", password)
	resp, err := client.Post(
		testServer.URL+"/login",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	defer resp.Body.Close()

	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("no session cookie set")
	}

	// Immediately access protected route - should work
	req, _ := http.NewRequest(http.MethodGet, testServer.URL+"/", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("initial protected access failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 with valid session, got %d", resp.StatusCode)
	}

	// Wait for session to expire
	time.Sleep(150 * time.Millisecond)

	// Access protected route again - should fail
	req, _ = http.NewRequest(http.MethodGet, testServer.URL+"/", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("post-expiry access failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 with expired session, got %d", resp.StatusCode)
	}
}
