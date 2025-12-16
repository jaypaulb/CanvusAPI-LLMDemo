package core

import (
	"testing"
	"time"
)

func TestNewSession_SetsDefaultExpiration(t *testing.T) {
	before := time.Now()
	session := NewSession("test-id")
	after := time.Now()

	// Verify ID is set
	if session.ID != "test-id" {
		t.Errorf("Session.ID = %q, want %q", session.ID, "test-id")
	}

	// Verify CreatedAt is approximately now
	if session.CreatedAt.Before(before) || session.CreatedAt.After(after) {
		t.Errorf("Session.CreatedAt = %v, want between %v and %v", session.CreatedAt, before, after)
	}

	// Verify ExpiresAt is 24 hours from CreatedAt
	expectedExpiry := session.CreatedAt.Add(DefaultSessionDuration)
	if !session.ExpiresAt.Equal(expectedExpiry) {
		t.Errorf("Session.ExpiresAt = %v, want %v", session.ExpiresAt, expectedExpiry)
	}
}

func TestNewSessionWithDuration_SetsCustomExpiration(t *testing.T) {
	customDuration := 1 * time.Hour
	session := NewSessionWithDuration("test-id", customDuration)

	expectedExpiry := session.CreatedAt.Add(customDuration)
	if !session.ExpiresAt.Equal(expectedExpiry) {
		t.Errorf("Session.ExpiresAt = %v, want %v", session.ExpiresAt, expectedExpiry)
	}
}

func TestSession_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     bool
	}{
		{"not expired", 1 * time.Hour, false},
		{"expired", -1 * time.Hour, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := NewSessionWithDuration("test-id", tt.duration)
			if got := session.IsExpired(); got != tt.want {
				t.Errorf("Session.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}
