package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultCookieConfig(t *testing.T) {
	cfg := DefaultCookieConfig()

	// Verify secure defaults
	if cfg.Name != SessionCookieName {
		t.Errorf("expected name %q, got %q", SessionCookieName, cfg.Name)
	}
	if cfg.MaxAge != DefaultCookieMaxAge {
		t.Errorf("expected MaxAge %d, got %d", DefaultCookieMaxAge, cfg.MaxAge)
	}
	if !cfg.HTTPOnly {
		t.Error("expected HTTPOnly to be true for security")
	}
	if cfg.SameSite != http.SameSiteStrictMode {
		t.Errorf("expected SameSiteStrictMode, got %v", cfg.SameSite)
	}
	if cfg.Path != DefaultCookiePath {
		t.Errorf("expected path %q, got %q", DefaultCookiePath, cfg.Path)
	}
	// Secure defaults to false (must be explicitly set for HTTPS)
	if cfg.Secure {
		t.Error("Secure should default to false to allow local development")
	}
}

func TestNewSessionCookie(t *testing.T) {
	tests := []struct {
		name        string
		sessionID   string
		cfg         CookieConfig
		wantErr     error
		checkCookie func(*testing.T, *http.Cookie)
	}{
		{
			name:      "valid session cookie",
			sessionID: "test-session-id-123",
			cfg:       DefaultCookieConfig(),
			wantErr:   nil,
			checkCookie: func(t *testing.T, c *http.Cookie) {
				if c.Name != SessionCookieName {
					t.Errorf("expected name %q, got %q", SessionCookieName, c.Name)
				}
				if c.Value != "test-session-id-123" {
					t.Errorf("expected value %q, got %q", "test-session-id-123", c.Value)
				}
				if !c.HttpOnly {
					t.Error("expected HTTPOnly to be true")
				}
				if c.SameSite != http.SameSiteStrictMode {
					t.Error("expected SameSiteStrictMode")
				}
			},
		},
		{
			name:      "empty session ID",
			sessionID: "",
			cfg:       DefaultCookieConfig(),
			wantErr:   ErrEmptySessionID,
		},
		{
			name:      "empty cookie name",
			sessionID: "valid-id",
			cfg:       CookieConfig{Name: ""},
			wantErr:   ErrEmptyCookieName,
		},
		{
			name:      "custom config with Secure flag",
			sessionID: "secure-session",
			cfg: CookieConfig{
				Name:     "custom_session",
				MaxAge:   3600,
				Secure:   true,
				HTTPOnly: true,
				SameSite: http.SameSiteLaxMode,
				Path:     "/api",
			},
			wantErr: nil,
			checkCookie: func(t *testing.T, c *http.Cookie) {
				if c.Name != "custom_session" {
					t.Errorf("expected name %q, got %q", "custom_session", c.Name)
				}
				if !c.Secure {
					t.Error("expected Secure to be true")
				}
				if c.MaxAge != 3600 {
					t.Errorf("expected MaxAge 3600, got %d", c.MaxAge)
				}
				if c.Path != "/api" {
					t.Errorf("expected path /api, got %q", c.Path)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cookie, err := NewSessionCookie(tt.sessionID, tt.cfg)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if cookie == nil {
				t.Fatal("expected cookie, got nil")
			}

			if tt.checkCookie != nil {
				tt.checkCookie(t, cookie)
			}
		})
	}
}

func TestNewSessionCookieWithDefaults(t *testing.T) {
	// Test with secure=false (development)
	cookie, err := NewSessionCookieWithDefaults("dev-session", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cookie.Secure {
		t.Error("expected Secure=false for development")
	}

	// Test with secure=true (production)
	cookie, err = NewSessionCookieWithDefaults("prod-session", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cookie.Secure {
		t.Error("expected Secure=true for production")
	}

	// Test with empty session ID
	_, err = NewSessionCookieWithDefaults("", false)
	if err != ErrEmptySessionID {
		t.Errorf("expected ErrEmptySessionID, got %v", err)
	}
}

func TestParseSessionCookie(t *testing.T) {
	tests := []struct {
		name       string
		cookieName string
		cookies    []*http.Cookie
		wantValue  string
		wantErr    error
	}{
		{
			name:       "valid session cookie",
			cookieName: "session_id",
			cookies: []*http.Cookie{
				{Name: "session_id", Value: "abc123"},
			},
			wantValue: "abc123",
			wantErr:   nil,
		},
		{
			name:       "cookie not found",
			cookieName: "session_id",
			cookies:    []*http.Cookie{},
			wantErr:    ErrNoCookie,
		},
		{
			name:       "empty cookie name",
			cookieName: "",
			wantErr:    ErrEmptyCookieName,
		},
		{
			name:       "multiple cookies find correct one",
			cookieName: "session_id",
			cookies: []*http.Cookie{
				{Name: "other_cookie", Value: "other_value"},
				{Name: "session_id", Value: "correct_value"},
			},
			wantValue: "correct_value",
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			for _, c := range tt.cookies {
				req.AddCookie(c)
			}

			value, err := ParseSessionCookie(req, tt.cookieName)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if value != tt.wantValue {
				t.Errorf("expected value %q, got %q", tt.wantValue, value)
			}
		})
	}
}

func TestParseSessionCookieDefault(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "default-test"})

	value, err := ParseSessionCookieDefault(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if value != "default-test" {
		t.Errorf("expected %q, got %q", "default-test", value)
	}
}

func TestClearSessionCookie(t *testing.T) {
	cookie, err := ClearSessionCookie("session_id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify deletion settings
	if cookie.MaxAge != -1 {
		t.Errorf("expected MaxAge=-1 for deletion, got %d", cookie.MaxAge)
	}
	if cookie.Value != "" {
		t.Errorf("expected empty value for deletion, got %q", cookie.Value)
	}
	if !cookie.HttpOnly {
		t.Error("clear cookie should still have HTTPOnly for security")
	}

	// Test with empty name
	_, err = ClearSessionCookie("")
	if err != ErrEmptyCookieName {
		t.Errorf("expected ErrEmptyCookieName, got %v", err)
	}
}

func TestClearSessionCookieDefault(t *testing.T) {
	cookie := ClearSessionCookieDefault()
	if cookie == nil {
		t.Fatal("expected cookie, got nil")
	}
	if cookie.Name != SessionCookieName {
		t.Errorf("expected name %q, got %q", SessionCookieName, cookie.Name)
	}
	if cookie.MaxAge != -1 {
		t.Errorf("expected MaxAge=-1, got %d", cookie.MaxAge)
	}
}

func TestSecondsToDuration(t *testing.T) {
	tests := []struct {
		seconds  int
		expected int64 // nanoseconds
	}{
		{0, 0},
		{1, 1e9},
		{60, 60e9},
		{3600, 3600e9},
		{86400, 86400e9}, // 24 hours
	}

	for _, tt := range tests {
		d := SecondsToDuration(tt.seconds)
		if d.Nanoseconds() != tt.expected {
			t.Errorf("SecondsToDuration(%d) = %v, want %d nanoseconds", tt.seconds, d, tt.expected)
		}
	}
}

func TestDurationToSeconds(t *testing.T) {
	tests := []struct {
		nanoseconds int64
		expected    int
	}{
		{0, 0},
		{1e9, 1},
		{60e9, 60},
		{3600e9, 3600},
		{86400e9, 86400},
	}

	for _, tt := range tests {
		d := SecondsToDuration(tt.expected)
		result := DurationToSeconds(d)
		if result != tt.expected {
			t.Errorf("DurationToSeconds(%v) = %d, want %d", d, result, tt.expected)
		}
	}
}

func TestCookieFromConfig(t *testing.T) {
	cfg := CookieConfig{
		Name:     "test_cookie",
		MaxAge:   1800,
		Secure:   true,
		HTTPOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/admin",
	}

	cookie, err := CookieFromConfig("custom-value", cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cookie.Value != "custom-value" {
		t.Errorf("expected value %q, got %q", "custom-value", cookie.Value)
	}
	if cookie.Name != "test_cookie" {
		t.Errorf("expected name %q, got %q", "test_cookie", cookie.Name)
	}

	// Test with empty name
	cfg.Name = ""
	_, err = CookieFromConfig("value", cfg)
	if err != ErrEmptyCookieName {
		t.Errorf("expected ErrEmptyCookieName, got %v", err)
	}
}
