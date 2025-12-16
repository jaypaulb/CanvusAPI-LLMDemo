// Package auth provides authentication components for the web UI.
// This file contains the auth middleware organism that composes session,
// rate limiting, and password verification molecules.
package auth

import (
	"go_backend/core"
	"go_backend/webui"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// Default configuration for the auth middleware.
const (
	// DefaultRateLimitAttempts is the default number of failed attempts before blocking.
	DefaultRateLimitAttempts = 5

	// DefaultRateLimitWindowMinutes is the default time window for counting attempts.
	DefaultRateLimitWindowMinutes = 1

	// DefaultRateLimitBlockMinutes is the default block duration after max attempts.
	DefaultRateLimitBlockMinutes = 5

	// DefaultSessionTTL is the default session duration.
	DefaultSessionTTL = 24 * time.Hour
)

// AuthMiddleware is an organism that composes authentication molecules to provide
// HTTP middleware for protecting routes that require authentication.
//
// Organism composition:
//   - Password hash (from password.go molecule) for credential verification
//   - SessionStore (from session_store.go molecule) for session management
//   - RateLimiter (from rate_limiter.go molecule) for brute force protection
//   - zap.Logger for structured logging
//
// The middleware checks for valid sessions and rate limits authentication attempts
// to prevent brute force attacks.
type AuthMiddleware struct {
	passwordHash string
	sessions     *webui.SessionStore
	rateLimiter  *webui.RateLimiter
	logger       *zap.Logger
	cookieConfig CookieConfig
}

// Config holds configuration options for the AuthMiddleware.
type Config struct {
	// SessionTTL is how long sessions remain valid (default: 24 hours)
	SessionTTL time.Duration

	// RateLimitAttempts is failed attempts before blocking (default: 5)
	RateLimitAttempts int

	// RateLimitWindowMinutes is the time window for counting attempts (default: 1)
	RateLimitWindowMinutes int

	// RateLimitBlockMinutes is how long to block after max attempts (default: 5)
	RateLimitBlockMinutes int

	// SecureCookies sets the Secure flag on cookies (true for HTTPS)
	SecureCookies bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		SessionTTL:             DefaultSessionTTL,
		RateLimitAttempts:      DefaultRateLimitAttempts,
		RateLimitWindowMinutes: DefaultRateLimitWindowMinutes,
		RateLimitBlockMinutes:  DefaultRateLimitBlockMinutes,
		SecureCookies:          false,
	}
}

// NewAuthMiddleware creates a new AuthMiddleware with the given password and logger.
// This factory composes all the authentication molecules:
//   - Hashes the password using bcrypt (password.go)
//   - Creates a session store (session_store.go)
//   - Creates a rate limiter with default settings (rate_limiter.go)
//
// Parameters:
//   - password: The plaintext password to hash for authentication
//   - logger: Structured logger for authentication events
//
// Returns:
//   - *AuthMiddleware: Configured middleware ready to use
//   - error: If password hashing fails
func NewAuthMiddleware(password string, logger *zap.Logger) (*AuthMiddleware, error) {
	return NewAuthMiddlewareWithConfig(password, logger, DefaultConfig())
}

// NewAuthMiddlewareWithConfig creates a new AuthMiddleware with custom configuration.
// This allows fine-tuning of rate limits, session duration, and cookie security.
//
// Parameters:
//   - password: The plaintext password to hash for authentication
//   - logger: Structured logger for authentication events
//   - cfg: Custom configuration for the middleware
//
// Returns:
//   - *AuthMiddleware: Configured middleware ready to use
//   - error: If password hashing fails
func NewAuthMiddlewareWithConfig(password string, logger *zap.Logger, cfg Config) (*AuthMiddleware, error) {
	// Hash the password using the password molecule
	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Create session store with configured TTL
	sessions := webui.NewSessionStore(cfg.SessionTTL)

	// Create rate limiter with configured limits
	rateLimiter := webui.NewRateLimiter(
		cfg.RateLimitAttempts,
		cfg.RateLimitWindowMinutes,
		cfg.RateLimitBlockMinutes,
	)

	// Configure cookie settings
	cookieConfig := DefaultCookieConfig()
	cookieConfig.Secure = cfg.SecureCookies
	cookieConfig.MaxAge = DurationToSeconds(cfg.SessionTTL)

	return &AuthMiddleware{
		passwordHash: hash,
		sessions:     sessions,
		rateLimiter:  rateLimiter,
		logger:       logger,
		cookieConfig: cookieConfig,
	}, nil
}

// Middleware returns an http.Handler that wraps the given handler with authentication.
// Requests without a valid session receive a 401 Unauthorized response.
//
// The middleware:
//  1. Extracts the session cookie from the request
//  2. Validates the session exists and is not expired
//  3. Allows the request through if valid, otherwise returns 401
//
// Rate limiting is applied per-IP for authentication endpoints (see RequireAuth).
//
// Parameters:
//   - next: The handler to wrap with authentication
//
// Returns:
//   - http.Handler: Handler that enforces authentication
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract session ID from cookie
		sessionID, err := ParseSessionCookieDefault(r)
		if err != nil {
			m.logger.Debug("no session cookie found",
				zap.String("path", r.URL.Path),
				zap.String("ip", getClientIP(r)),
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate the session
		_, err = m.sessions.Get(sessionID)
		if err != nil {
			m.logger.Debug("invalid session",
				zap.String("path", r.URL.Path),
				zap.String("ip", getClientIP(r)),
				zap.Error(err),
			)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Session is valid, allow request through
		m.logger.Debug("session validated",
			zap.String("path", r.URL.Path),
		)
		next.ServeHTTP(w, r)
	})
}

// RequireAuth is a convenience wrapper that converts a HandlerFunc to a
// Handler with authentication middleware applied.
//
// This is useful for protecting individual handler functions:
//
//	mux.HandleFunc("/api/data", authMiddleware.RequireAuth(dataHandler))
//
// Parameters:
//   - next: The handler function to protect
//
// Returns:
//   - http.HandlerFunc: Handler function with authentication enforced
func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return m.Middleware(next).ServeHTTP
}

// CheckRateLimit checks if an IP address is allowed to attempt authentication.
// Returns true if allowed, false if rate limited.
// When rate limited, writes a 429 Too Many Requests response with Retry-After header.
//
// Parameters:
//   - w: Response writer for sending 429 response if rate limited
//   - ip: The IP address to check
//
// Returns:
//   - bool: true if the request is allowed, false if rate limited (response already sent)
func (m *AuthMiddleware) CheckRateLimit(w http.ResponseWriter, ip string) bool {
	allowed, remaining := m.rateLimiter.Allow(ip)
	if !allowed {
		m.logger.Warn("rate limit exceeded",
			zap.String("ip", ip),
			zap.Duration("remaining", remaining),
		)
		w.Header().Set("Retry-After", formatRetryAfter(remaining))
		http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		return false
	}
	return true
}

// RecordFailedAttempt records a failed authentication attempt for rate limiting.
//
// Parameters:
//   - ip: The IP address that made the failed attempt
func (m *AuthMiddleware) RecordFailedAttempt(ip string) {
	m.rateLimiter.RecordAttempt(ip)
	m.logger.Info("failed authentication attempt recorded",
		zap.String("ip", ip),
		zap.Int("attempts", m.rateLimiter.GetAttemptCount(ip)),
	)
}

// ResetRateLimit resets the rate limit counter for an IP after successful login.
//
// Parameters:
//   - ip: The IP address to reset
func (m *AuthMiddleware) ResetRateLimit(ip string) {
	m.rateLimiter.Reset(ip)
}

// VerifyPassword checks if the provided password matches the stored hash.
// Returns nil if the password is correct, error otherwise.
//
// Parameters:
//   - password: The plaintext password to verify
//
// Returns:
//   - error: nil if password matches, ErrPasswordMismatch if not
func (m *AuthMiddleware) VerifyPassword(password string) error {
	return VerifyPassword(password, m.passwordHash)
}

// CreateSession creates a new authenticated session.
// Returns the session and a cookie that should be set on the response.
//
// Returns:
//   - core.Session: The created session
//   - *http.Cookie: Cookie to set on the response
//   - error: If session creation fails
func (m *AuthMiddleware) CreateSession() (core.Session, *http.Cookie, error) {
	session, err := m.sessions.Create()
	if err != nil {
		m.logger.Error("failed to create session", zap.Error(err))
		return core.Session{}, nil, err
	}

	cookie, err := NewSessionCookie(session.ID, m.cookieConfig)
	if err != nil {
		m.logger.Error("failed to create session cookie", zap.Error(err))
		return core.Session{}, nil, err
	}

	m.logger.Info("session created",
		zap.String("session_id", session.ID[:8]+"..."), // Log prefix only for privacy
		zap.Time("expires_at", session.ExpiresAt),
	)

	return session, cookie, nil
}

// DestroySession removes a session and returns a cookie that clears the client cookie.
// This should be called during logout.
//
// Parameters:
//   - sessionID: The session ID to destroy
//
// Returns:
//   - *http.Cookie: Cookie to set on the response that clears the session cookie
func (m *AuthMiddleware) DestroySession(sessionID string) *http.Cookie {
	m.sessions.Delete(sessionID)
	m.logger.Info("session destroyed",
		zap.String("session_id", sessionID[:min(8, len(sessionID))]+"..."),
	)
	return ClearSessionCookieDefault()
}

// GetSession retrieves a session by ID.
// Returns ErrSessionNotFound or ErrSessionExpired if the session is invalid.
//
// Parameters:
//   - sessionID: The session ID to look up
//
// Returns:
//   - core.Session: The session if valid
//   - error: If session not found or expired
func (m *AuthMiddleware) GetSession(sessionID string) (core.Session, error) {
	return m.sessions.Get(sessionID)
}

// SessionStore returns the underlying session store for advanced usage.
// This allows access to cleanup and ticker functionality.
func (m *AuthMiddleware) SessionStore() *webui.SessionStore {
	return m.sessions
}

// RateLimiter returns the underlying rate limiter for advanced usage.
// This allows access to cleanup and ticker functionality.
func (m *AuthMiddleware) RateLimiter() *webui.RateLimiter {
	return m.rateLimiter
}

// getClientIP extracts the client IP address from the request.
// Checks X-Forwarded-For and X-Real-IP headers first for proxy support.
func getClientIP(r *http.Request) string {
	// Check for proxy headers first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs; use the first one
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr (includes port)
	// Strip port if present
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}

// formatRetryAfter formats a duration as seconds for the Retry-After header.
func formatRetryAfter(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return strconv.Itoa(seconds)
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
