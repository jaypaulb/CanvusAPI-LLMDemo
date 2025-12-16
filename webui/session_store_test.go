package webui

import (
	"context"
	"testing"
	"time"
)

// TestSessionStore_CreateAndGet tests the basic create and get workflow.
func TestSessionStore_CreateAndGet(t *testing.T) {
	store := NewSessionStore(1 * time.Hour)

	// Create a session
	session, err := store.Create()
	if err != nil {
		t.Fatalf("Create() error = %v, want nil", err)
	}

	// Verify session has valid ID
	if session.ID == "" {
		t.Error("Create() returned session with empty ID")
	}

	// Verify session can be retrieved
	retrieved, err := store.Get(session.ID)
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("Get() returned session.ID = %v, want %v", retrieved.ID, session.ID)
	}

	// Verify count
	if count := store.Count(); count != 1 {
		t.Errorf("Count() = %d, want 1", count)
	}
}

// TestSessionStore_GetNotFound tests retrieving a non-existent session.
func TestSessionStore_GetNotFound(t *testing.T) {
	store := NewSessionStore(1 * time.Hour)

	_, err := store.Get("nonexistent-session-id")
	if err != ErrSessionNotFound {
		t.Errorf("Get() error = %v, want ErrSessionNotFound", err)
	}
}

// TestSessionStore_Delete tests session deletion.
func TestSessionStore_Delete(t *testing.T) {
	store := NewSessionStore(1 * time.Hour)

	// Create and then delete
	session, _ := store.Create()
	store.Delete(session.ID)

	// Should not be found
	_, err := store.Get(session.ID)
	if err != ErrSessionNotFound {
		t.Errorf("Get() after Delete() error = %v, want ErrSessionNotFound", err)
	}

	// Delete non-existent should not panic (idempotent)
	store.Delete("nonexistent") // Should not panic
}

// TestSessionStore_Expiry tests that expired sessions are rejected.
func TestSessionStore_Expiry(t *testing.T) {
	// Create store with very short TTL
	store := NewSessionStore(1 * time.Millisecond)

	session, _ := store.Create()

	// Wait for expiration
	time.Sleep(5 * time.Millisecond)

	_, err := store.Get(session.ID)
	if err != ErrSessionExpired {
		t.Errorf("Get() on expired session error = %v, want ErrSessionExpired", err)
	}

	// Expired session should be auto-removed
	if count := store.Count(); count != 0 {
		t.Errorf("Count() after expired Get() = %d, want 0 (auto-cleanup)", count)
	}
}

// TestSessionStore_Cleanup tests the cleanup functionality.
func TestSessionStore_Cleanup(t *testing.T) {
	store := NewSessionStore(1 * time.Millisecond)

	// Create multiple sessions
	for i := 0; i < 5; i++ {
		_, err := store.Create()
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	if count := store.Count(); count != 5 {
		t.Fatalf("Count() = %d, want 5", count)
	}

	// Wait for expiration
	time.Sleep(5 * time.Millisecond)

	// Run cleanup
	removed := store.Cleanup()
	if removed != 5 {
		t.Errorf("Cleanup() removed = %d, want 5", removed)
	}

	if count := store.Count(); count != 0 {
		t.Errorf("Count() after Cleanup() = %d, want 0", count)
	}
}

// TestSessionStore_CleanupTicker tests the background cleanup ticker.
func TestSessionStore_CleanupTicker(t *testing.T) {
	store := NewSessionStore(1 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start cleanup ticker with short interval
	store.StartCleanupTicker(ctx, 10*time.Millisecond)

	// Create sessions
	for i := 0; i < 3; i++ {
		store.Create()
	}

	// Wait for sessions to expire and ticker to run
	time.Sleep(50 * time.Millisecond)

	// Sessions should be cleaned up
	if count := store.Count(); count != 0 {
		t.Errorf("Count() after ticker = %d, want 0", count)
	}

	// Cancel context to stop ticker
	cancel()

	// Give goroutine time to exit cleanly
	time.Sleep(20 * time.Millisecond)
}
