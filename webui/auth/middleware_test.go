package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

// testLogger creates a no-op logger for testing.
func testLogger() *zap.Logger {
	return zap.NewNop()
}

// TestNewAuthMiddleware tests the factory function creates all components.
func TestNewAuthMiddleware(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "test-password-123",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mw, err := NewAuthMiddleware(tt.password, testLogger())

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify all components are created
			if mw.passwordHash == "" {
				t.Error("password hash not set")
			}
			if mw.sessions == nil {
				t.Error("session store not created")
			}
			if mw.rateLimiter == nil {
				t.Error("rate limiter not created")
			}
			if mw.logger == nil {
				t.Error("logger not set")
			}
		})
	}
}

// TestNewAuthMiddlewareWithConfig tests custom configuration.
func TestNewAuthMiddlewareWithConfig(t *testing.T) {
	cfg := Config{
		SessionTTL:             1 * time.Hour,
		RateLimitAttempts:      3,
		RateLimitWindowMinutes: 5,
		RateLimitBlockMinutes:  10,
		SecureCookies:          true,
	}

	mw, err := NewAuthMiddlewareWithConfig("password", testLogger(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify cookie config was set
	if !mw.cookieConfig.Secure {
		t.Error("expected secure cookies to be enabled")
	}

	// Verify session TTL is reflected in cookie MaxAge
	expectedMaxAge := int(cfg.SessionTTL.Seconds())
	if mw.cookieConfig.MaxAge != expectedMaxAge {
		t.Errorf("expected MaxAge %d, got %d", expectedMaxAge, mw.cookieConfig.MaxAge)
	}
}

// TestMiddleware_ValidSession tests that a valid session passes through.
func TestMiddleware_ValidSession(t *testing.T) {
	mw, err := NewAuthMiddleware("password", testLogger())
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create a session
	session, cookie, err := mw.CreateSession()
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	if session.ID == "" {
		t.Fatal("session ID is empty")
	}

	// Create a protected handler that records if it was called
	handlerCalled := false
	protected := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// Create request with session cookie
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	protected.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

// TestMiddleware_MissingSession tests that missing session returns 401.
func TestMiddleware_MissingSession(t *testing.T) {
	mw, err := NewAuthMiddleware("password", testLogger())
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create a protected handler that should not be called
	handlerCalled := false
	protected := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	// Create request WITHOUT session cookie
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rr := httptest.NewRecorder()

	protected.ServeHTTP(rr, req)

	if handlerCalled {
		t.Error("handler should not have been called")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

// TestMiddleware_InvalidSession tests that invalid/expired session returns 401.
func TestMiddleware_InvalidSession(t *testing.T) {
	mw, err := NewAuthMiddleware("password", testLogger())
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create a protected handler
	protected := mw.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Create request with fake session cookie (not in store)
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: "fake-session-id-not-in-store",
	})
	rr := httptest.NewRecorder()

	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

// TestRequireAuth tests the convenience wrapper.
func TestRequireAuth(t *testing.T) {
	mw, err := NewAuthMiddleware("password", testLogger())
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create a handler function
	handlerCalled := false
	handler := func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}

	// Wrap it with RequireAuth
	protected := mw.RequireAuth(handler)

	// Test without session - should be 401
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rr := httptest.NewRecorder()
	protected(rr, req)

	if handlerCalled {
		t.Error("handler should not have been called without session")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rr.Code)
	}
}

// TestCheckRateLimit tests rate limiting functionality.
func TestCheckRateLimit(t *testing.T) {
	cfg := Config{
		SessionTTL:             DefaultSessionTTL,
		RateLimitAttempts:      3, // Only allow 3 attempts
		RateLimitWindowMinutes: 1,
		RateLimitBlockMinutes:  5,
		SecureCookies:          false,
	}

	mw, err := NewAuthMiddlewareWithConfig("password", testLogger(), cfg)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	testIP := "192.168.1.100"

	// First 3 attempts should be allowed
	for i := 0; i < 3; i++ {
		rr := httptest.NewRecorder()
		allowed := mw.CheckRateLimit(rr, testIP)
		if !allowed {
			t.Errorf("attempt %d should be allowed", i+1)
		}
		// Record the failed attempt
		mw.RecordFailedAttempt(testIP)
	}

	// 4th attempt should be blocked (after recording 3 failures)
	rr := httptest.NewRecorder()
	allowed := mw.CheckRateLimit(rr, testIP)
	if allowed {
		t.Error("4th attempt should be blocked")
	}
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected status %d, got %d", http.StatusTooManyRequests, rr.Code)
	}
	if rr.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header to be set")
	}
}

// TestMiddlewareVerifyPassword tests password verification via the middleware.
func TestMiddlewareVerifyPassword(t *testing.T) {
	password := "my-secure-password"
	mw, err := NewAuthMiddleware(password, testLogger())
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "correct password",
			password: password,
			wantErr:  false,
		},
		{
			name:     "wrong password",
			password: "wrong-password",
			wantErr:  true,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mw.VerifyPassword(tt.password)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestCreateAndDestroySession tests session lifecycle.
func TestCreateAndDestroySession(t *testing.T) {
	mw, err := NewAuthMiddleware("password", testLogger())
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create a session
	session, cookie, err := mw.CreateSession()
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Verify session exists
	retrieved, err := mw.GetSession(session.ID)
	if err != nil {
		t.Errorf("failed to retrieve session: %v", err)
	}
	if retrieved.ID != session.ID {
		t.Errorf("expected session ID %s, got %s", session.ID, retrieved.ID)
	}

	// Verify cookie properties
	if cookie.Name != SessionCookieName {
		t.Errorf("expected cookie name %s, got %s", SessionCookieName, cookie.Name)
	}
	if cookie.Value != session.ID {
		t.Errorf("cookie value doesn't match session ID")
	}
	if !cookie.HttpOnly {
		t.Error("expected HttpOnly to be true")
	}

	// Destroy the session
	clearCookie := mw.DestroySession(session.ID)

	// Verify session no longer exists
	_, err = mw.GetSession(session.ID)
	if err == nil {
		t.Error("expected error retrieving destroyed session")
	}

	// Verify clear cookie has MaxAge -1
	if clearCookie.MaxAge != -1 {
		t.Errorf("expected clear cookie MaxAge -1, got %d", clearCookie.MaxAge)
	}
}

// TestResetRateLimit tests that rate limit can be reset after successful login.
func TestResetRateLimit(t *testing.T) {
	cfg := Config{
		SessionTTL:             DefaultSessionTTL,
		RateLimitAttempts:      3,
		RateLimitWindowMinutes: 1,
		RateLimitBlockMinutes:  5,
		SecureCookies:          false,
	}

	mw, err := NewAuthMiddlewareWithConfig("password", testLogger(), cfg)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	testIP := "10.0.0.1"

	// Record 2 failed attempts
	mw.RecordFailedAttempt(testIP)
	mw.RecordFailedAttempt(testIP)

	// Should still be allowed (2 < 3)
	rr := httptest.NewRecorder()
	if !mw.CheckRateLimit(rr, testIP) {
		t.Error("should still be allowed after 2 attempts")
	}

	// Reset (simulating successful login)
	mw.ResetRateLimit(testIP)

	// Should be allowed again (counter reset)
	rr = httptest.NewRecorder()
	if !mw.CheckRateLimit(rr, testIP) {
		t.Error("should be allowed after reset")
	}
}

// TestGetClientIP tests IP extraction from requests.
func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xForwarded string
		xRealIP    string
		expected   string
	}{
		{
			name:       "remote addr only",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "x-forwarded-for single",
			remoteAddr: "10.0.0.1:12345",
			xForwarded: "203.0.113.50",
			expected:   "203.0.113.50",
		},
		{
			name:       "x-forwarded-for multiple",
			remoteAddr: "10.0.0.1:12345",
			xForwarded: "203.0.113.50, 70.41.3.18, 150.172.238.178",
			expected:   "203.0.113.50",
		},
		{
			name:       "x-real-ip",
			remoteAddr: "10.0.0.1:12345",
			xRealIP:    "203.0.113.60",
			expected:   "203.0.113.60",
		},
		{
			name:       "x-forwarded-for takes precedence",
			remoteAddr: "10.0.0.1:12345",
			xForwarded: "203.0.113.50",
			xRealIP:    "203.0.113.60",
			expected:   "203.0.113.50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwarded != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwarded)
			}
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, ip)
			}
		})
	}
}

// TestSessionAndRateLimiterAccessors tests the accessor methods.
func TestSessionAndRateLimiterAccessors(t *testing.T) {
	mw, err := NewAuthMiddleware("password", testLogger())
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	if mw.SessionStore() == nil {
		t.Error("SessionStore() returned nil")
	}
	if mw.RateLimiter() == nil {
		t.Error("RateLimiter() returned nil")
	}
}
