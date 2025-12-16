package auth

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestLoginHandler_GET_RendersLoginPage(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	middleware, err := NewAuthMiddleware("testpassword", logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create GET request
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()

	// Execute handler
	handler := LoginHandler(middleware)
	handler.ServeHTTP(rr, req)

	// Verify response is the login page (200 OK with HTML content)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Check content type is HTML
	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected Content-Type text/html, got %s", contentType)
	}

	// Check that the body contains the login form
	body := rr.Body.String()
	if !strings.Contains(body, "CanvusLocalLLM") {
		t.Error("response should contain 'CanvusLocalLLM' title")
	}
	if !strings.Contains(body, "password") {
		t.Error("response should contain password field")
	}
}

func TestLoginHandler_GET_RedirectsIfAlreadyLoggedIn(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	middleware, err := NewAuthMiddleware("testpassword", logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create a session
	_, sessionCookie, err := middleware.CreateSession()
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	// Create GET request with valid session cookie
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	req.AddCookie(sessionCookie)
	rr := httptest.NewRecorder()

	// Execute handler
	handler := LoginHandler(middleware)
	handler.ServeHTTP(rr, req)

	// Verify redirect to dashboard
	if rr.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("expected redirect to /, got %s", location)
	}
}

func TestLoginHandler_POST_SuccessfulLogin(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	password := "testpassword"
	middleware, err := NewAuthMiddleware(password, logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create POST request with correct password
	form := url.Values{}
	form.Set("password", password)
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	// Execute handler
	handler := LoginHandler(middleware)
	handler.ServeHTTP(rr, req)

	// Verify redirect to dashboard (303 See Other for POST)
	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != "/" {
		t.Errorf("expected redirect to /, got %s", location)
	}

	// Verify session cookie was set
	cookies := rr.Result().Cookies()
	var sessionCookieFound bool
	for _, c := range cookies {
		if c.Name == SessionCookieName && c.Value != "" && c.MaxAge > 0 {
			sessionCookieFound = true
			break
		}
	}
	if !sessionCookieFound {
		t.Error("expected session cookie to be set")
	}
}

func TestLoginHandler_POST_InvalidPassword(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	middleware, err := NewAuthMiddleware("correctpassword", logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create POST request with wrong password
	form := url.Values{}
	form.Set("password", "wrongpassword")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	// Execute handler (note: this will have a 1-second delay)
	start := time.Now()
	handler := LoginHandler(middleware)
	handler.ServeHTTP(rr, req)
	elapsed := time.Since(start)

	// Verify redirect to login with error
	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}

	location := rr.Header().Get("Location")
	if !strings.Contains(location, "/login") {
		t.Errorf("expected redirect to /login, got %s", location)
	}
	if !strings.Contains(location, "error=") {
		t.Errorf("expected error query parameter, got %s", location)
	}

	// Verify delay was applied (at least 900ms to account for timing variance)
	if elapsed < 900*time.Millisecond {
		t.Errorf("expected at least 1 second delay, got %v", elapsed)
	}

	// Verify no session cookie was set
	cookies := rr.Result().Cookies()
	for _, c := range cookies {
		if c.Name == SessionCookieName && c.MaxAge > 0 {
			t.Error("session cookie should not be set on failed login")
		}
	}
}

func TestLoginHandler_POST_EmptyPassword(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	middleware, err := NewAuthMiddleware("testpassword", logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	// Create POST request with empty password
	form := url.Values{}
	form.Set("password", "")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	// Execute handler
	handler := LoginHandler(middleware)
	handler.ServeHTTP(rr, req)

	// Verify redirect to login with error
	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}

	location := rr.Header().Get("Location")
	if !strings.Contains(location, "/login") {
		t.Errorf("expected redirect to /login, got %s", location)
	}
	if !strings.Contains(location, "error=") {
		t.Errorf("expected error query parameter, got %s", location)
	}
}

func TestLoginHandler_POST_RateLimiting(t *testing.T) {
	// Setup with strict rate limiting (2 attempts, quick window)
	logger := zap.NewNop()
	cfg := Config{
		SessionTTL:             24 * time.Hour,
		RateLimitAttempts:      2, // Only 2 attempts allowed
		RateLimitWindowMinutes: 1,
		RateLimitBlockMinutes:  5,
	}
	middleware, err := NewAuthMiddlewareWithConfig("correctpassword", logger, cfg)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	handler := LoginHandler(middleware)

	// Helper to make failed login attempt
	makeFailedAttempt := func() *httptest.ResponseRecorder {
		form := url.Values{}
		form.Set("password", "wrongpassword")
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = "192.168.1.1:12345" // Fixed IP for rate limiting
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
		return rr
	}

	// First attempt - should be allowed (records attempt 1)
	rr1 := makeFailedAttempt()
	if rr1.Code != http.StatusSeeOther {
		t.Errorf("first attempt: expected status %d, got %d", http.StatusSeeOther, rr1.Code)
	}

	// Second attempt - should be allowed (records attempt 2, now blocked)
	rr2 := makeFailedAttempt()
	if rr2.Code != http.StatusSeeOther {
		t.Errorf("second attempt: expected status %d, got %d", http.StatusSeeOther, rr2.Code)
	}

	// Third attempt - should be rate limited (429)
	form := url.Values{}
	form.Set("password", "wrongpassword")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = "192.168.1.1:12345"
	rr3 := httptest.NewRecorder()
	handler.ServeHTTP(rr3, req)

	if rr3.Code != http.StatusTooManyRequests {
		t.Errorf("third attempt: expected status %d, got %d", http.StatusTooManyRequests, rr3.Code)
	}

	// Verify Retry-After header is set
	retryAfter := rr3.Header().Get("Retry-After")
	if retryAfter == "" {
		t.Error("expected Retry-After header to be set")
	}
}

func TestLoginHandler_POST_RateLimitResetOnSuccess(t *testing.T) {
	// Setup with strict rate limiting
	logger := zap.NewNop()
	cfg := Config{
		SessionTTL:             24 * time.Hour,
		RateLimitAttempts:      3,
		RateLimitWindowMinutes: 1,
		RateLimitBlockMinutes:  5,
	}
	middleware, err := NewAuthMiddlewareWithConfig("correctpassword", logger, cfg)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	handler := LoginHandler(middleware)
	clientIP := "192.168.1.2:12345"

	// Make one failed attempt
	form := url.Values{}
	form.Set("password", "wrongpassword")
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = clientIP
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify attempt was recorded
	if middleware.RateLimiter().GetAttemptCount("192.168.1.2") != 1 {
		t.Errorf("expected 1 attempt recorded, got %d", middleware.RateLimiter().GetAttemptCount("192.168.1.2"))
	}

	// Now make a successful login
	form = url.Values{}
	form.Set("password", "correctpassword")
	req = httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.RemoteAddr = clientIP
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify successful login
	if rr.Code != http.StatusSeeOther {
		t.Errorf("successful login: expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}

	// Verify rate limit was reset
	if middleware.RateLimiter().GetAttemptCount("192.168.1.2") != 0 {
		t.Errorf("expected 0 attempts after successful login, got %d", middleware.RateLimiter().GetAttemptCount("192.168.1.2"))
	}
}

func TestLoginHandler_InvalidMethod(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	middleware, err := NewAuthMiddleware("testpassword", logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	invalidMethods := []string{http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range invalidMethods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/login", nil)
			rr := httptest.NewRecorder()

			handler := LoginHandler(middleware)
			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected status %d for %s, got %d",
					http.StatusMethodNotAllowed, method, rr.Code)
			}
		})
	}
}

func TestLoginHandlerWithRedirect_CustomSuccessPath(t *testing.T) {
	// Setup
	logger := zap.NewNop()
	password := "testpassword"
	middleware, err := NewAuthMiddleware(password, logger)
	if err != nil {
		t.Fatalf("failed to create middleware: %v", err)
	}

	customPath := "/dashboard"

	// Create POST request with correct password
	form := url.Values{}
	form.Set("password", password)
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()

	// Execute handler with custom redirect
	handler := LoginHandlerWithRedirect(middleware, customPath)
	handler.ServeHTTP(rr, req)

	// Verify redirect to custom path
	if rr.Code != http.StatusSeeOther {
		t.Errorf("expected status %d, got %d", http.StatusSeeOther, rr.Code)
	}

	location := rr.Header().Get("Location")
	if location != customPath {
		t.Errorf("expected redirect to %s, got %s", customPath, location)
	}
}
