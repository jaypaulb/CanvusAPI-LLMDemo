// Package auth provides authentication molecules for the web UI.
// This file contains the secure cookie builder molecule for session management.
package auth

import (
	"errors"
	"net/http"
	"time"
)

// Cookie configuration defaults
const (
	// DefaultCookieMaxAge is the default session duration (24 hours)
	DefaultCookieMaxAge = 24 * 60 * 60 // seconds

	// DefaultCookiePath is the path for which the cookie is valid
	DefaultCookiePath = "/"

	// SessionCookieName is the default name for session cookies
	SessionCookieName = "session_id"
)

// ErrNoCookie is returned when the requested cookie is not present in the request.
var ErrNoCookie = errors.New("cookie not found")

// ErrEmptyCookieName is returned when attempting to create a cookie with an empty name.
var ErrEmptyCookieName = errors.New("cookie name cannot be empty")

// ErrEmptySessionID is returned when attempting to create a session cookie with an empty ID.
var ErrEmptySessionID = errors.New("session ID cannot be empty")

// CookieConfig holds configuration for secure session cookies.
// All security-related settings have secure defaults.
//
// Molecule composition:
//   - Composes standard http.Cookie attributes with security defaults
//   - Uses time.Duration for MaxAge to integrate with Go's time package
type CookieConfig struct {
	// Name is the cookie name (default: "session_id")
	Name string

	// MaxAge is the cookie lifetime in seconds (default: 24 hours)
	// Set to -1 to delete the cookie, 0 for session cookie (browser close)
	MaxAge int

	// Secure indicates whether the cookie should only be sent over HTTPS
	// Should be true in production, may be false for local development
	Secure bool

	// HTTPOnly prevents JavaScript access to the cookie (always recommended)
	HTTPOnly bool

	// SameSite controls cross-site request behavior
	// Strict prevents CSRF attacks but may affect UX
	// Lax is a good balance for most applications
	SameSite http.SameSite

	// Path restricts the cookie to a specific URL path
	Path string
}

// DefaultCookieConfig returns a CookieConfig with secure defaults.
// The returned config is suitable for production use with HTTPS.
//
// Security defaults:
//   - HTTPOnly: true (prevents XSS attacks from accessing cookies)
//   - SameSite: Strict (prevents CSRF attacks)
//   - Path: "/" (cookie valid for entire site)
//   - MaxAge: 24 hours
//
// Note: Secure is set to false by default; set to true for production HTTPS.
func DefaultCookieConfig() CookieConfig {
	return CookieConfig{
		Name:     SessionCookieName,
		MaxAge:   DefaultCookieMaxAge,
		Secure:   false, // Set true for HTTPS
		HTTPOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     DefaultCookiePath,
	}
}

// NewSessionCookie creates a secure session cookie with the given session ID.
// The cookie is configured with security settings from the provided config.
//
// This molecule composes:
//   - CookieConfig for security settings
//   - http.Cookie for the standard cookie structure
//
// Security features:
//   - HTTPOnly: Prevents JavaScript access (XSS protection)
//   - SameSite: Controls cross-origin behavior (CSRF protection)
//   - Secure: Ensures HTTPS-only transmission when enabled
//
// Parameters:
//   - sessionID: The session identifier to store in the cookie (must not be empty)
//   - cfg: Cookie configuration including security settings
//
// Returns:
//   - *http.Cookie: Ready-to-use cookie for http.SetCookie()
//   - error: ErrEmptySessionID if sessionID is empty, ErrEmptyCookieName if cfg.Name is empty
func NewSessionCookie(sessionID string, cfg CookieConfig) (*http.Cookie, error) {
	if sessionID == "" {
		return nil, ErrEmptySessionID
	}

	name := cfg.Name
	if name == "" {
		return nil, ErrEmptyCookieName
	}

	return &http.Cookie{
		Name:     name,
		Value:    sessionID,
		Path:     cfg.Path,
		MaxAge:   cfg.MaxAge,
		HttpOnly: cfg.HTTPOnly,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
	}, nil
}

// NewSessionCookieWithDefaults creates a session cookie using default secure settings.
// This is a convenience function that uses DefaultCookieConfig().
//
// Parameters:
//   - sessionID: The session identifier to store in the cookie
//   - secure: Whether to set the Secure flag (true for HTTPS)
//
// Returns:
//   - *http.Cookie: Ready-to-use cookie
//   - error: ErrEmptySessionID if sessionID is empty
func NewSessionCookieWithDefaults(sessionID string, secure bool) (*http.Cookie, error) {
	cfg := DefaultCookieConfig()
	cfg.Secure = secure
	return NewSessionCookie(sessionID, cfg)
}

// ParseSessionCookie extracts the session ID from a request cookie.
// Returns the session ID value or an error if the cookie doesn't exist.
//
// This molecule composes:
//   - http.Request cookie parsing
//   - Error handling with descriptive errors
//
// Parameters:
//   - r: The HTTP request containing cookies
//   - name: The cookie name to look for
//
// Returns:
//   - string: The session ID from the cookie
//   - error: ErrNoCookie if the cookie doesn't exist, ErrEmptyCookieName if name is empty
func ParseSessionCookie(r *http.Request, name string) (string, error) {
	if name == "" {
		return "", ErrEmptyCookieName
	}

	cookie, err := r.Cookie(name)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return "", ErrNoCookie
		}
		return "", err
	}

	return cookie.Value, nil
}

// ParseSessionCookieDefault extracts the session ID using the default cookie name.
// This is a convenience function that uses SessionCookieName.
//
// Parameters:
//   - r: The HTTP request containing cookies
//
// Returns:
//   - string: The session ID from the cookie
//   - error: ErrNoCookie if the cookie doesn't exist
func ParseSessionCookieDefault(r *http.Request) (string, error) {
	return ParseSessionCookie(r, SessionCookieName)
}

// ClearSessionCookie creates a cookie that instructs the browser to delete
// the session cookie. The returned cookie has MaxAge=-1 and an empty value.
//
// This should be used during logout to remove the session cookie.
//
// Parameters:
//   - name: The cookie name to clear
//
// Returns:
//   - *http.Cookie: Cookie configured to delete the named cookie
//   - error: ErrEmptyCookieName if name is empty
func ClearSessionCookie(name string) (*http.Cookie, error) {
	if name == "" {
		return nil, ErrEmptyCookieName
	}

	return &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     DefaultCookiePath,
		MaxAge:   -1, // Delete immediately
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}, nil
}

// ClearSessionCookieDefault creates a clear cookie using the default cookie name.
// This is a convenience function that uses SessionCookieName.
//
// Returns:
//   - *http.Cookie: Cookie configured to delete the session cookie
func ClearSessionCookieDefault() *http.Cookie {
	// Can't error with default name, so ignore error
	cookie, _ := ClearSessionCookie(SessionCookieName)
	return cookie
}

// CookieFromConfig creates an http.Cookie from a CookieConfig with a custom value.
// This is useful for creating cookies with arbitrary values using the same config.
//
// Parameters:
//   - value: The cookie value
//   - cfg: Cookie configuration
//
// Returns:
//   - *http.Cookie: Configured cookie
//   - error: ErrEmptyCookieName if cfg.Name is empty
func CookieFromConfig(value string, cfg CookieConfig) (*http.Cookie, error) {
	if cfg.Name == "" {
		return nil, ErrEmptyCookieName
	}

	return &http.Cookie{
		Name:     cfg.Name,
		Value:    value,
		Path:     cfg.Path,
		MaxAge:   cfg.MaxAge,
		HttpOnly: cfg.HTTPOnly,
		Secure:   cfg.Secure,
		SameSite: cfg.SameSite,
	}, nil
}

// SecondsToDuration converts MaxAge seconds to a time.Duration.
// Useful for coordinating cookie expiry with session store TTL.
//
// Parameters:
//   - maxAge: Cookie MaxAge in seconds
//
// Returns:
//   - time.Duration: Equivalent duration
func SecondsToDuration(maxAge int) time.Duration {
	return time.Duration(maxAge) * time.Second
}

// DurationToSeconds converts a time.Duration to MaxAge seconds.
// Useful for setting cookie MaxAge from a time.Duration.
//
// Parameters:
//   - d: Duration to convert
//
// Returns:
//   - int: Equivalent seconds (rounded down)
func DurationToSeconds(d time.Duration) int {
	return int(d.Seconds())
}
