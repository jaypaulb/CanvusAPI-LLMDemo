package core

import (
	"testing"
	"time"
)

func TestNewAttemptRecord_InitializesCorrectly(t *testing.T) {
	before := time.Now()
	record := NewAttemptRecord()
	after := time.Now()

	if record.Count != 1 {
		t.Errorf("AttemptRecord.Count = %d, want 1", record.Count)
	}

	expectedResetMin := before.Add(DefaultRateLimitWindow)
	expectedResetMax := after.Add(DefaultRateLimitWindow)
	if record.ResetAt.Before(expectedResetMin) || record.ResetAt.After(expectedResetMax) {
		t.Errorf("AttemptRecord.ResetAt = %v, want between %v and %v", record.ResetAt, expectedResetMin, expectedResetMax)
	}
}

func TestAttemptRecord_IsBlocked(t *testing.T) {
	tests := []struct {
		name        string
		count       int
		maxAttempts int
		want        bool
	}{
		{"below limit", 3, 5, false},
		{"at limit", 5, 5, true},
		{"above limit", 7, 5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := AttemptRecord{Count: tt.count, ResetAt: time.Now().Add(time.Hour)}
			if got := record.IsBlocked(tt.maxAttempts); got != tt.want {
				t.Errorf("AttemptRecord.IsBlocked(%d) = %v, want %v", tt.maxAttempts, got, tt.want)
			}
		})
	}
}

func TestAttemptRecord_Increment(t *testing.T) {
	t.Run("increments count when not expired", func(t *testing.T) {
		record := AttemptRecord{Count: 3, ResetAt: time.Now().Add(time.Hour)}
		newRecord := record.Increment()

		if newRecord.Count != 4 {
			t.Errorf("Incremented count = %d, want 4", newRecord.Count)
		}
		if newRecord.ResetAt != record.ResetAt {
			t.Errorf("ResetAt changed unexpectedly")
		}
	})

	t.Run("resets when expired", func(t *testing.T) {
		record := AttemptRecord{Count: 10, ResetAt: time.Now().Add(-time.Hour)}
		newRecord := record.Increment()

		if newRecord.Count != 1 {
			t.Errorf("Reset count = %d, want 1", newRecord.Count)
		}
	})
}
