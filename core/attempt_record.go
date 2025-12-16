package core

import (
	"time"
)

// DefaultRateLimitWindow is the default time window for rate limiting (15 minutes).
const DefaultRateLimitWindow = 15 * time.Minute

// DefaultMaxAttempts is the default maximum attempts before rate limiting kicks in.
const DefaultMaxAttempts = 5

// AttemptRecord tracks authentication attempts for rate limiting purposes.
// Each record is associated with an identifier (typically IP address).
type AttemptRecord struct {
	// Count is the number of attempts within the current window
	Count int

	// ResetAt is when the attempt count should reset
	ResetAt time.Time
}

// NewAttemptRecord creates a new AttemptRecord with count=1 and default window duration.
func NewAttemptRecord() AttemptRecord {
	return AttemptRecord{
		Count:   1,
		ResetAt: time.Now().Add(DefaultRateLimitWindow),
	}
}

// NewAttemptRecordWithWindow creates a new AttemptRecord with count=1 and custom window duration.
func NewAttemptRecordWithWindow(window time.Duration) AttemptRecord {
	return AttemptRecord{
		Count:   1,
		ResetAt: time.Now().Add(window),
	}
}

// ShouldReset returns true if the current time is past the ResetAt time.
func (a AttemptRecord) ShouldReset() bool {
	return time.Now().After(a.ResetAt)
}

// IsBlocked returns true if the attempt count has reached or exceeded the given limit.
func (a AttemptRecord) IsBlocked(maxAttempts int) bool {
	return a.Count >= maxAttempts
}

// TimeUntilReset returns the duration until the attempt record resets.
// Returns zero if already past reset time.
func (a AttemptRecord) TimeUntilReset() time.Duration {
	remaining := time.Until(a.ResetAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Increment returns a new AttemptRecord with count incremented by 1.
// If the record should reset, returns a fresh record with count=1 instead.
func (a AttemptRecord) Increment() AttemptRecord {
	if a.ShouldReset() {
		return NewAttemptRecord()
	}
	return AttemptRecord{
		Count:   a.Count + 1,
		ResetAt: a.ResetAt,
	}
}
