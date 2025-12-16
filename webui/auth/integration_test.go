package auth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

// =============================================================================
// Integration Tests for the Authentication System
// These tests verify that auth components work together correctly as an organism.
// =============================================================================

// TestIntegration_FullAuthenticationCycle tests the complete login -> access protected
// resource -> logout cycle. This verifies the auth middleware, session management,
// and handlers work together correctly.
func TestIntegration_FullAuthenticationCycle(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	password := "integration-test-password"
	middleware, err := NewAuthMiddleware(password, logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create protected handler
	protectedCalled := false
	protectedHandler := middleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		protectedCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("protected content"))
	})

	loginHandler := LoginHandler(middleware)
	logoutHandler := LogoutHandler(middleware)

	// Step 1: Try to access protected resource without auth - should fail
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rr := httptest.NewRecorder()
	protectedHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Step 1: expected 401 without auth, got %d", rr.Code)
	}
	if protectedCalled {
		t.Error("Step 1: protected handler should not be called without auth")
	}

	// Step 2: Login
	form := url.Values{}
	form.Set("password", password)
	req = httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr = httptest.NewRecorder()
	loginHandler(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Step 2: expected 303 after login, got %d", rr.Code)
	}

	// Extract session cookie
	var sessionCookie *http.Cookie
	for _, c := range rr.Result().Cookies() {
		if c.Name == SessionCookieName && c.MaxAge > 0 {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("Step 2: no session cookie set after login")
	}

	// Step 3: Access protected resource with session - should succeed
	protectedCalled = false
	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	protectedHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Step 3: expected 200 with valid session, got %d", rr.Code)
	}
	if !protectedCalled {
		t.Error("Step 3: protected handler should be called with valid session")
	}

	// Step 4: Logout
	req = httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(sessionCookie)
	rr = httptest.NewRecorder()
	logoutHandler(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("Step 4: expected 303 after logout, got %d", rr.Code)
	}

	// Step 5: Try to access protected resource after logout - should fail
	protectedCalled = false
	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(sessionCookie) // Cookie still present but session destroyed
	rr = httptest.NewRecorder()
	protectedHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Step 5: expected 401 after logout, got %d", rr.Code)
	}
	if protectedCalled {
		t.Error("Step 5: protected handler should not be called after logout")
	}
}

// TestIntegration_ConcurrentSessionCreation verifies that the session store
// handles concurrent session creation correctly without race conditions.
func TestIntegration_ConcurrentSessionCreation(t *testing.T) {
	logger := zap.NewNop()
	middleware, err := NewAuthMiddleware("password", logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	const numGoroutines = 50
	var wg sync.WaitGroup
	sessionIDs := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)

	// Concurrently create sessions
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			session, _, err := middleware.CreateSession()
			if err != nil {
				errors <- err
				return
			}
			sessionIDs <- session.ID
		}()
	}

	wg.Wait()
	close(sessionIDs)
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("session creation error: %v", err)
	}

	// Verify all sessions are unique
	seen := make(map[string]bool)
	count := 0
	for id := range sessionIDs {
		if seen[id] {
			t.Errorf("duplicate session ID: %s", id)
		}
		seen[id] = true
		count++
	}

	if count != numGoroutines {
		t.Errorf("expected %d sessions, got %d", numGoroutines, count)
	}

	// Verify all sessions are valid
	for id := range seen {
		_, err := middleware.GetSession(id)
		if err != nil {
			t.Errorf("session %s should be valid: %v", id[:8], err)
		}
	}
}

// TestIntegration_ConcurrentSessionValidation verifies that concurrent
// session validation operations work correctly.
func TestIntegration_ConcurrentSessionValidation(t *testing.T) {
	logger := zap.NewNop()
	middleware, err := NewAuthMiddleware("password", logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create a session
	session, _, err := middleware.CreateSession()
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	const numGoroutines = 100
	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines)

	// Concurrently validate the same session
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			retrieved, err := middleware.GetSession(session.ID)
			if err != nil {
				errors <- err
				return
			}
			if retrieved.ID != session.ID {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("session validation error: %v", err)
	}
}

// TestIntegration_RateLimitIsolation verifies that rate limiting for one IP
// does not affect other IPs.
func TestIntegration_RateLimitIsolation(t *testing.T) {
	logger := zap.NewNop()
	cfg := Config{
		SessionTTL:             24 * time.Hour,
		RateLimitAttempts:      2,
		RateLimitWindowMinutes: 1,
		RateLimitBlockMinutes:  5,
	}
	middleware, err := NewAuthMiddlewareWithConfig("password", logger, cfg)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	blockedIP := "10.0.0.1"
	allowedIP := "10.0.0.2"

	// Block the first IP by recording max attempts
	middleware.RecordFailedAttempt(blockedIP)
	middleware.RecordFailedAttempt(blockedIP)

	// Verify blocked IP is blocked
	rr := httptest.NewRecorder()
	allowed := middleware.CheckRateLimit(rr, blockedIP)
	if allowed {
		t.Error("blocked IP should be rate limited")
	}
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 for blocked IP, got %d", rr.Code)
	}

	// Verify other IP is NOT blocked
	rr = httptest.NewRecorder()
	allowed = middleware.CheckRateLimit(rr, allowedIP)
	if !allowed {
		t.Error("unrelated IP should not be rate limited")
	}
	if rr.Code == http.StatusTooManyRequests {
		t.Error("unrelated IP should not get 429")
	}

	// Verify third IP is also not affected
	rr = httptest.NewRecorder()
	allowed = middleware.CheckRateLimit(rr, "192.168.1.1")
	if !allowed {
		t.Error("another unrelated IP should not be rate limited")
	}
}

// TestIntegration_ConcurrentRateLimitRecording verifies that concurrent
// rate limit recording for different IPs works correctly.
func TestIntegration_ConcurrentRateLimitRecording(t *testing.T) {
	logger := zap.NewNop()
	cfg := Config{
		SessionTTL:             24 * time.Hour,
		RateLimitAttempts:      100, // High limit so we don't block during test
		RateLimitWindowMinutes: 1,
		RateLimitBlockMinutes:  5,
	}
	middleware, err := NewAuthMiddlewareWithConfig("password", logger, cfg)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	const numIPs = 20
	const attemptsPerIP = 5
	var wg sync.WaitGroup

	// Concurrently record attempts for different IPs
	for i := 0; i < numIPs; i++ {
		wg.Add(1)
		go func(ipNum int) {
			defer wg.Done()
			ip := "192.168.1." + string(rune('0'+ipNum%10)) + string(rune('0'+ipNum/10))
			for j := 0; j < attemptsPerIP; j++ {
				middleware.RecordFailedAttempt(ip)
			}
		}(i)
	}

	wg.Wait()

	// Verify each IP has the correct attempt count
	for i := 0; i < numIPs; i++ {
		ip := "192.168.1." + string(rune('0'+i%10)) + string(rune('0'+i/10))
		count := middleware.RateLimiter().GetAttemptCount(ip)
		if count != attemptsPerIP {
			t.Errorf("IP %s: expected %d attempts, got %d", ip, attemptsPerIP, count)
		}
	}
}

// TestIntegration_SessionExpiryDuringRequest tests the edge case where a session
// expires between the middleware check and actual use (time-of-check-time-of-use).
func TestIntegration_SessionExpiryDuringRequest(t *testing.T) {
	logger := zap.NewNop()

	// Create middleware with very short session TTL
	cfg := Config{
		SessionTTL:             50 * time.Millisecond, // Very short
		RateLimitAttempts:      5,
		RateLimitWindowMinutes: 1,
		RateLimitBlockMinutes:  5,
	}
	middleware, err := NewAuthMiddlewareWithConfig("password", logger, cfg)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create a session
	session, cookie, err := middleware.CreateSession()
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Verify session is valid initially
	_, err = middleware.GetSession(session.ID)
	if err != nil {
		t.Fatalf("session should be valid initially: %v", err)
	}

	// Wait for session to expire
	time.Sleep(100 * time.Millisecond)

	// Now try to access protected resource - session should be expired
	handlerCalled := false
	protected := middleware.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.AddCookie(cookie)
	rr := httptest.NewRecorder()

	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired session, got %d", rr.Code)
	}
	if handlerCalled {
		t.Error("handler should not be called for expired session")
	}
}

// TestIntegration_MultipleSessionsPerMiddleware verifies that multiple active
// sessions can coexist and be validated independently.
func TestIntegration_MultipleSessionsPerMiddleware(t *testing.T) {
	logger := zap.NewNop()
	middleware, err := NewAuthMiddleware("password", logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	const numSessions = 10
	sessions := make([]string, numSessions)
	cookies := make([]*http.Cookie, numSessions)

	// Create multiple sessions
	for i := 0; i < numSessions; i++ {
		session, cookie, err := middleware.CreateSession()
		if err != nil {
			t.Fatalf("failed to create session %d: %v", i, err)
		}
		sessions[i] = session.ID
		cookies[i] = cookie
	}

	// Verify all sessions are valid
	for i, id := range sessions {
		_, err := middleware.GetSession(id)
		if err != nil {
			t.Errorf("session %d should be valid: %v", i, err)
		}
	}

	// Verify session count
	if count := middleware.SessionStore().Count(); count != numSessions {
		t.Errorf("expected %d sessions in store, got %d", numSessions, count)
	}

	// Delete half the sessions
	for i := 0; i < numSessions/2; i++ {
		middleware.DestroySession(sessions[i])
	}

	// Verify deleted sessions are invalid
	for i := 0; i < numSessions/2; i++ {
		_, err := middleware.GetSession(sessions[i])
		if err == nil {
			t.Errorf("deleted session %d should be invalid", i)
		}
	}

	// Verify remaining sessions are still valid
	for i := numSessions / 2; i < numSessions; i++ {
		_, err := middleware.GetSession(sessions[i])
		if err != nil {
			t.Errorf("remaining session %d should be valid: %v", i, err)
		}
	}

	// Verify session count after deletion
	expectedRemaining := numSessions - numSessions/2
	if count := middleware.SessionStore().Count(); count != expectedRemaining {
		t.Errorf("expected %d sessions remaining, got %d", expectedRemaining, count)
	}
}
