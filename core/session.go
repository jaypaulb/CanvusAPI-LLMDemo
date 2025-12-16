package core

import (
	"time"
)

// DefaultSessionDuration is the default lifetime for a session (24 hours).
const DefaultSessionDuration = 24 * time.Hour

// Session represents an authenticated user session with expiry tracking.
// Sessions are created after successful authentication and stored server-side.
type Session struct {
	// ID is the unique session identifier (base64 URL-encoded random bytes)
	ID string

	// CreatedAt is when the session was created
	CreatedAt time.Time

	// ExpiresAt is when the session becomes invalid
	ExpiresAt time.Time
}

// NewSession creates a new Session with the given ID and default 24-hour expiration.
// CreatedAt is set to the current time.
func NewSession(id string) Session {
	now := time.Now()
	return Session{
		ID:        id,
		CreatedAt: now,
		ExpiresAt: now.Add(DefaultSessionDuration),
	}
}

// NewSessionWithDuration creates a new Session with a custom expiration duration.
// CreatedAt is set to the current time.
func NewSessionWithDuration(id string, duration time.Duration) Session {
	now := time.Now()
	return Session{
		ID:        id,
		CreatedAt: now,
		ExpiresAt: now.Add(duration),
	}
}

// IsExpired returns true if the session has passed its expiration time.
func (s Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// TimeRemaining returns the duration until the session expires.
// Returns a negative duration if already expired.
func (s Session) TimeRemaining() time.Duration {
	return time.Until(s.ExpiresAt)
}
