package webui

import (
	"context"
	"go_backend/core"
	"testing"
	"time"
)

// TestRateLimiter_AllowInitial tests that new IPs are always allowed.
func TestRateLimiter_AllowInitial(t *testing.T) {
	limiter := NewRateLimiter(5, 15, 30) // 5 attempts, 15 min window, 30 min block

	allowed, remaining := limiter.Allow("192.168.1.1")
	if !allowed {
		t.Error("Allow() = false for new IP, want true")
	}
	if remaining != 0 {
		t.Errorf("Allow() remaining = %v, want 0", remaining)
	}
}

// TestRateLimiter_BlockAfterMaxAttempts tests blocking after max failed attempts.
func TestRateLimiter_BlockAfterMaxAttempts(t *testing.T) {
	limiter := NewRateLimiter(3, 15, 30) // 3 attempts max
	ip := "192.168.1.100"

	// Record 3 failed attempts
	for i := 0; i < 3; i++ {
		allowed, _ := limiter.Allow(ip)
		if !allowed && i < 2 {
			t.Errorf("Allow() = false after %d attempts, want true", i)
		}
		limiter.RecordAttempt(ip)
	}

	// Should now be blocked
	allowed, remaining := limiter.Allow(ip)
	if allowed {
		t.Error("Allow() = true after max attempts, want false")
	}
	if remaining <= 0 {
		t.Error("Allow() remaining should be positive when blocked")
	}

	// Verify attempt count
	if count := limiter.GetAttemptCount(ip); count != 3 {
		t.Errorf("GetAttemptCount() = %d, want 3", count)
	}
}

// TestRateLimiter_Reset tests that successful login clears attempts.
func TestRateLimiter_Reset(t *testing.T) {
	limiter := NewRateLimiter(5, 15, 30)
	ip := "192.168.1.200"

	// Record some attempts
	limiter.RecordAttempt(ip)
	limiter.RecordAttempt(ip)

	if count := limiter.GetAttemptCount(ip); count != 2 {
		t.Errorf("GetAttemptCount() = %d, want 2", count)
	}

	// Reset (successful login)
	limiter.Reset(ip)

	// Should be allowed again with no attempts
	allowed, _ := limiter.Allow(ip)
	if !allowed {
		t.Error("Allow() = false after Reset, want true")
	}

	if count := limiter.GetAttemptCount(ip); count != 0 {
		t.Errorf("GetAttemptCount() after Reset = %d, want 0", count)
	}
}

// TestRateLimiter_WindowExpiry tests that attempts expire after window.
func TestRateLimiter_WindowExpiry(t *testing.T) {
	// Manually set up the limiter with a very short window
	limiter := &RateLimiter{
		attempts:      make(map[string]core.AttemptRecord),
		maxAttempts:   5,
		windowMinutes: 0, // We'll handle timing manually
		blockMinutes:  0,
	}

	ip := "192.168.1.50"

	// Record attempt with very short window
	limiter.attempts[ip] = core.AttemptRecord{
		Count:   3,
		ResetAt: time.Now().Add(1 * time.Millisecond),
	}

	// Wait for expiry
	time.Sleep(5 * time.Millisecond)

	// Should be allowed now (window expired)
	allowed, _ := limiter.Allow(ip)
	if !allowed {
		t.Error("Allow() = false after window expiry, want true")
	}

	// Attempt count should show 0 (expired)
	if count := limiter.GetAttemptCount(ip); count != 0 {
		t.Errorf("GetAttemptCount() after expiry = %d, want 0", count)
	}
}

// TestRateLimiter_Cleanup tests removal of expired entries.
func TestRateLimiter_Cleanup(t *testing.T) {
	limiter := &RateLimiter{
		attempts:      make(map[string]core.AttemptRecord),
		maxAttempts:   5,
		windowMinutes: 15,
		blockMinutes:  30,
	}

	// Add expired entries
	limiter.attempts["expired1"] = core.AttemptRecord{
		Count:   2,
		ResetAt: time.Now().Add(-1 * time.Minute), // Already expired
	}
	limiter.attempts["expired2"] = core.AttemptRecord{
		Count:   1,
		ResetAt: time.Now().Add(-1 * time.Minute), // Already expired
	}

	// Add valid entry
	limiter.attempts["valid"] = core.AttemptRecord{
		Count:   1,
		ResetAt: time.Now().Add(1 * time.Hour), // Still valid
	}

	removed := limiter.Cleanup()
	if removed != 2 {
		t.Errorf("Cleanup() removed = %d, want 2", removed)
	}

	if count := limiter.Count(); count != 1 {
		t.Errorf("Count() after Cleanup = %d, want 1", count)
	}
}

// TestRateLimiter_CleanupTicker tests the background cleanup ticker.
func TestRateLimiter_CleanupTicker(t *testing.T) {
	limiter := &RateLimiter{
		attempts:      make(map[string]core.AttemptRecord),
		maxAttempts:   5,
		windowMinutes: 15,
		blockMinutes:  30,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add an expired entry
	limiter.attempts["expired"] = core.AttemptRecord{
		Count:   1,
		ResetAt: time.Now().Add(-1 * time.Minute),
	}

	// Start cleanup ticker with short interval
	limiter.StartCleanupTicker(ctx, 10*time.Millisecond)

	// Wait for ticker to run
	time.Sleep(50 * time.Millisecond)

	if count := limiter.Count(); count != 0 {
		t.Errorf("Count() after ticker = %d, want 0", count)
	}

	// Cancel context to stop ticker
	cancel()

	// Give goroutine time to exit cleanly
	time.Sleep(20 * time.Millisecond)
}
