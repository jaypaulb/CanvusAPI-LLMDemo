// Package webui provides the web-based user interface for CanvusLocalLLM.
// This file contains the rate limiter molecule for protecting against brute force attacks.
package webui

import (
	"context"
	"go_backend/core"
	"sync"
	"time"
)

// RateLimiter protects against brute force login attacks by tracking
// failed authentication attempts per IP address.
//
// Molecule composition:
//   - core.AttemptRecord: Tracks attempt count and window timing
//
// The limiter uses a sliding window approach where:
//   - Each failed attempt increments the counter
//   - After maxAttempts, the IP is blocked for blockDuration
//   - Successful login resets the counter
//   - Old entries are periodically cleaned up
//
// Thread safety is provided via sync.RWMutex for concurrent access.
type RateLimiter struct {
	mu            sync.RWMutex
	attempts      map[string]core.AttemptRecord
	maxAttempts   int
	windowMinutes int
	blockMinutes  int
}

// NewRateLimiter creates a new RateLimiter with the specified limits.
//
// Parameters:
//   - maxAttempts: Number of failed attempts before blocking (e.g., 5)
//   - windowMinutes: Time window for counting attempts in minutes (e.g., 15)
//   - blockMinutes: How long to block after max attempts in minutes (e.g., 30)
//
// Returns a ready-to-use RateLimiter.
func NewRateLimiter(maxAttempts, windowMinutes, blockMinutes int) *RateLimiter {
	return &RateLimiter{
		attempts:      make(map[string]core.AttemptRecord),
		maxAttempts:   maxAttempts,
		windowMinutes: windowMinutes,
		blockMinutes:  blockMinutes,
	}
}

// Allow checks if an IP address is allowed to attempt authentication.
// Returns (true, 0) if allowed, or (false, remainingBlockTime) if blocked.
//
// This method composes the AttemptRecord atom's IsBlocked and TimeUntilReset methods.
//
// Parameters:
//   - ip: The IP address to check
//
// Returns:
//   - allowed: true if the IP can attempt login, false if blocked
//   - remaining: Duration until the block expires (0 if not blocked)
func (r *RateLimiter) Allow(ip string) (bool, time.Duration) {
	r.mu.RLock()
	record, exists := r.attempts[ip]
	r.mu.RUnlock()

	if !exists {
		return true, 0
	}

	// Check if the window has expired - allow if so
	if record.ShouldReset() {
		return true, 0
	}

	// Check if blocked using AttemptRecord atom
	if record.IsBlocked(r.maxAttempts) {
		return false, record.TimeUntilReset()
	}

	return true, 0
}

// RecordAttempt records a failed authentication attempt for an IP address.
// This should be called after each failed login attempt.
//
// This method composes the AttemptRecord atom's Increment method and creates
// new records using NewAttemptRecordWithWindow.
//
// When max attempts are reached, the block duration is extended.
//
// Parameters:
//   - ip: The IP address that made the failed attempt
func (r *RateLimiter) RecordAttempt(ip string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, exists := r.attempts[ip]
	if !exists {
		// Create new record with configured window (uses AttemptRecord atom)
		window := time.Duration(r.windowMinutes) * time.Minute
		r.attempts[ip] = core.NewAttemptRecordWithWindow(window)
		return
	}

	// Check if window expired - start fresh
	if record.ShouldReset() {
		window := time.Duration(r.windowMinutes) * time.Minute
		r.attempts[ip] = core.NewAttemptRecordWithWindow(window)
		return
	}

	// Increment the counter using AttemptRecord atom
	record = record.Increment()

	// If we just hit the max attempts, extend the reset time to block duration
	if record.Count == r.maxAttempts {
		record = core.AttemptRecord{
			Count:   record.Count,
			ResetAt: time.Now().Add(time.Duration(r.blockMinutes) * time.Minute),
		}
	}

	r.attempts[ip] = record
}

// Reset clears the attempt record for an IP address.
// This should be called after a successful login to clear the slate.
//
// Parameters:
//   - ip: The IP address to reset
func (r *RateLimiter) Reset(ip string) {
	r.mu.Lock()
	delete(r.attempts, ip)
	r.mu.Unlock()
}

// Cleanup removes expired attempt records from the store.
// Returns the number of records that were removed.
//
// This method should be called periodically to prevent memory growth.
// Use StartCleanupTicker for automatic cleanup.
func (r *RateLimiter) Cleanup() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	removed := 0
	for ip, record := range r.attempts {
		if record.ShouldReset() {
			delete(r.attempts, ip)
			removed++
		}
	}

	return removed
}

// StartCleanupTicker starts a background goroutine that periodically
// calls Cleanup to remove expired attempt records.
//
// The ticker stops when the provided context is cancelled.
//
// Parameters:
//   - ctx: Context for cancellation (cancel to stop the ticker)
//   - interval: How often to run cleanup (e.g., 5 * time.Minute)
func (r *RateLimiter) StartCleanupTicker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.Cleanup()
			}
		}
	}()
}

// Count returns the current number of tracked IP addresses.
// This is useful for monitoring and debugging.
func (r *RateLimiter) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.attempts)
}

// GetAttemptCount returns the current attempt count for an IP address.
// Returns 0 if the IP has no recorded attempts or if the window expired.
// This is useful for displaying remaining attempts to users.
//
// Parameters:
//   - ip: The IP address to check
func (r *RateLimiter) GetAttemptCount(ip string) int {
	r.mu.RLock()
	record, exists := r.attempts[ip]
	r.mu.RUnlock()

	if !exists || record.ShouldReset() {
		return 0
	}

	return record.Count
}
