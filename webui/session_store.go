// Package webui provides the web-based user interface for CanvusLocalLLM.
// This file contains the session store molecule for managing user sessions.
package webui

import (
	"context"
	"errors"
	"go_backend/core"
	"sync"
	"time"
)

// ErrSessionNotFound is returned when a session ID is not found in the store.
var ErrSessionNotFound = errors.New("session not found")

// ErrSessionExpired is returned when a session exists but has expired.
var ErrSessionExpired = errors.New("session expired")

// SessionStore manages authenticated user sessions with thread-safe operations.
// It composes the Session and GenerateSessionID atoms from the core package.
//
// Molecule composition:
//   - core.Session: Session data structure with expiry tracking
//   - core.GenerateSessionID: Cryptographically secure ID generation
//
// Thread safety is provided via sync.RWMutex for concurrent access.
type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]core.Session
	ttl      time.Duration
}

// NewSessionStore creates a new SessionStore with the given session TTL.
// The TTL determines how long sessions remain valid after creation.
//
// Parameters:
//   - ttl: Duration until sessions expire (use core.DefaultSessionDuration for 24h)
//
// Returns a ready-to-use SessionStore.
func NewSessionStore(ttl time.Duration) *SessionStore {
	return &SessionStore{
		sessions: make(map[string]core.Session),
		ttl:      ttl,
	}
}

// Create generates a new session with a cryptographically secure ID.
// The session is stored internally and returned for cookie setting.
//
// This method composes:
//   - core.GenerateSessionID() for secure random ID generation
//   - core.NewSessionWithDuration() for session creation with custom TTL
//
// Returns the created session or an error if ID generation fails.
func (s *SessionStore) Create() (core.Session, error) {
	// Generate cryptographically secure session ID (atom)
	id, err := core.GenerateSessionID()
	if err != nil {
		return core.Session{}, err
	}

	// Create session with configured TTL (atom)
	session := core.NewSessionWithDuration(id, s.ttl)

	// Store session with thread safety
	s.mu.Lock()
	s.sessions[id] = session
	s.mu.Unlock()

	return session, nil
}

// Get retrieves a session by ID, checking for expiration.
// Returns ErrSessionNotFound if the session doesn't exist.
// Returns ErrSessionExpired if the session exists but has expired.
//
// Expired sessions are automatically removed from the store.
//
// Parameters:
//   - sessionID: The session ID to look up
//
// Returns the session if valid, or an appropriate error.
func (s *SessionStore) Get(sessionID string) (core.Session, error) {
	s.mu.RLock()
	session, exists := s.sessions[sessionID]
	s.mu.RUnlock()

	if !exists {
		return core.Session{}, ErrSessionNotFound
	}

	// Check expiration using Session atom's method
	if session.IsExpired() {
		// Clean up expired session
		s.mu.Lock()
		delete(s.sessions, sessionID)
		s.mu.Unlock()
		return core.Session{}, ErrSessionExpired
	}

	return session, nil
}

// Delete removes a session from the store.
// This is used for explicit logout functionality.
// No error is returned if the session doesn't exist (idempotent operation).
//
// Parameters:
//   - sessionID: The session ID to remove
func (s *SessionStore) Delete(sessionID string) {
	s.mu.Lock()
	delete(s.sessions, sessionID)
	s.mu.Unlock()
}

// Cleanup removes all expired sessions from the store.
// Returns the number of sessions that were removed.
//
// This method should be called periodically to prevent memory growth
// from abandoned sessions. Use StartCleanupTicker for automatic cleanup.
func (s *SessionStore) Cleanup() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	removed := 0
	for id, session := range s.sessions {
		if session.IsExpired() {
			delete(s.sessions, id)
			removed++
		}
	}

	return removed
}

// StartCleanupTicker starts a background goroutine that periodically
// calls Cleanup to remove expired sessions.
//
// The ticker stops when the provided context is cancelled.
// This is typically called during application startup with a context
// that's cancelled on shutdown.
//
// Parameters:
//   - ctx: Context for cancellation (cancel to stop the ticker)
//   - interval: How often to run cleanup (e.g., 5 * time.Minute)
func (s *SessionStore) StartCleanupTicker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.Cleanup()
			}
		}
	}()
}

// Count returns the current number of sessions in the store.
// This is useful for monitoring and debugging.
func (s *SessionStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}
