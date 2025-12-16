package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestLogoutHandler_GET_WithValidSession(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	middleware, err := NewAuthMiddleware("testpassword", logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create a session
	session, sessionCookie, err := middleware.CreateSession()
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Verify session exists before logout
	_, err = middleware.GetSession(session.ID)
	if err != nil {
		t.Fatalf("session should exist before logout: %v", err)
	}

	// Create request with session cookie
	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	req.AddCookie(sessionCookie)

	// Execute handler
	rr := httptest.NewRecorder()
	handler := LogoutHandler(middleware)
	handler.ServeHTTP(rr, req)

	// Verify redirect to login
	if rr.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != "/login" {
		t.Errorf("expected redirect to /login, got %s", location)
	}

	// Verify session was destroyed
	_, err = middleware.GetSession(session.ID)
	if err == nil {
		t.Error("session should not exist after logout")
	}

	// Verify clear cookie was set
	cookies := rr.Result().Cookies()
	var clearCookieFound bool
	for _, c := range cookies {
		if c.Name == SessionCookieName && c.MaxAge == -1 {
			clearCookieFound = true
			break
		}
	}
	if !clearCookieFound {
		t.Error("expected clear cookie with MaxAge=-1")
	}
}

func TestLogoutHandler_POST_WithValidSession(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	middleware, err := NewAuthMiddleware("testpassword", logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create a session
	session, sessionCookie, err := middleware.CreateSession()
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Create POST request with session cookie
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(sessionCookie)

	// Execute handler
	rr := httptest.NewRecorder()
	handler := LogoutHandler(middleware)
	handler.ServeHTTP(rr, req)

	// Verify redirect with 303 See Other (correct for POST)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected status %d for POST, got %d", http.StatusSeeOther, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != "/login" {
		t.Errorf("expected redirect to /login, got %s", location)
	}

	// Verify session was destroyed
	_, err = middleware.GetSession(session.ID)
	if err == nil {
		t.Error("session should not exist after logout")
	}
}

func TestLogoutHandler_GET_WithoutSession(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	middleware, err := NewAuthMiddleware("testpassword", logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create request without session cookie
	req := httptest.NewRequest(http.MethodGet, "/logout", nil)

	// Execute handler
	rr := httptest.NewRecorder()
	handler := LogoutHandler(middleware)
	handler.ServeHTTP(rr, req)

	// Should still redirect to login without error
	if rr.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != "/login" {
		t.Errorf("expected redirect to /login, got %s", location)
	}

	// Verify clear cookie was still set
	cookies := rr.Result().Cookies()
	var clearCookieFound bool
	for _, c := range cookies {
		if c.Name == SessionCookieName && c.MaxAge == -1 {
			clearCookieFound = true
			break
		}
	}
	if !clearCookieFound {
		t.Error("expected clear cookie with MaxAge=-1 even without session")
	}
}

func TestLogoutHandler_InvalidMethod(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	middleware, err := NewAuthMiddleware("testpassword", logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	invalidMethods := []string{http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range invalidMethods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/logout", nil)
			rr := httptest.NewRecorder()

			handler := LogoutHandler(middleware)
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d for %s, got %d",
					http.StatusMethodNotAllowed, method, rr.Code)
			}
		})
	}
}

func TestLogoutHandler_WithExpiredSession(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	middleware, err := NewAuthMiddleware("testpassword", logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create a session and then destroy it to simulate expiry
	session, sessionCookie, err := middleware.CreateSession()
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	middleware.DestroySession(session.ID)

	// Create request with the now-invalid session cookie
	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	req.AddCookie(sessionCookie)

	// Execute handler
	rr := httptest.NewRecorder()
	handler := LogoutHandler(middleware)
	handler.ServeHTTP(rr, req)

	// Should still redirect without error (idempotent)
	if rr.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != "/login" {
		t.Errorf("expected redirect to /login, got %s", location)
	}
}

func TestTruncateSessionID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "long session ID",
			input:    "abcdefghijklmnop",
			expected: "abcdefgh...",
		},
		{
			name:     "exactly 8 characters",
			input:    "abcdefgh",
			expected: "abcdefgh...",
		},
		{
			name:     "short session ID",
			input:    "abc",
			expected: "abc...",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateSessionID(tt.input)
			if result != tt.expected {
				t.Errorf("truncateSessionID(%q) = %q, want %q",
					tt.input, result, tt.expected)
			}
		})
	}
}
