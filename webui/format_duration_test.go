package webui

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		// Zero and small values
		{"zero", 0, "0s"},
		{"one second", time.Second, "1s"},
		{"45 seconds", 45 * time.Second, "45s"},

		// Minutes
		{"one minute", time.Minute, "1m 0s"},
		{"one minute 30 seconds", time.Minute + 30*time.Second, "1m 30s"},
		{"two minutes", 2 * time.Minute, "2m 0s"},
		{"59 minutes 59 seconds", 59*time.Minute + 59*time.Second, "59m 59s"},

		// Hours
		{"one hour", time.Hour, "1h 0m"},
		{"one hour 30 minutes", time.Hour + 30*time.Minute, "1h 30m"},
		{"two hours 34 minutes", 2*time.Hour + 34*time.Minute, "2h 34m"},
		{"23 hours 59 minutes", 23*time.Hour + 59*time.Minute, "23h 59m"},

		// Days
		{"one day", 24 * time.Hour, "1d 0h"},
		{"one day 5 hours", 24*time.Hour + 5*time.Hour, "1d 5h"},
		{"three days 12 hours", 3*24*time.Hour + 12*time.Hour, "3d 12h"},
		{"six days 23 hours", 6*24*time.Hour + 23*time.Hour, "6d 23h"},

		// Weeks
		{"one week", 7 * 24 * time.Hour, "1w 0d"},
		{"one week 3 days", 7*24*time.Hour + 3*24*time.Hour, "1w 3d"},
		{"two weeks 5 days", 2*7*24*time.Hour + 5*24*time.Hour, "2w 5d"},

		// Negative durations
		{"negative 5 minutes", -5 * time.Minute, "-5m 0s"},
		{"negative 2 hours 30 minutes", -2*time.Hour - 30*time.Minute, "-2h 30m"},
		{"negative 1 day", -24 * time.Hour, "-1d 0h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestFormatDurationCompact(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		// Zero and small values
		{"zero", 0, "0s"},
		{"one second", time.Second, "1s"},
		{"45 seconds", 45 * time.Second, "45s"},

		// Minutes - shows only minutes
		{"one minute", time.Minute, "1m"},
		{"one minute 30 seconds", time.Minute + 30*time.Second, "1m"},
		{"59 minutes", 59 * time.Minute, "59m"},

		// Hours - shows only hours
		{"one hour", time.Hour, "1h"},
		{"one hour 30 minutes", time.Hour + 30*time.Minute, "1h"},
		{"23 hours", 23 * time.Hour, "23h"},

		// Days - shows only days
		{"one day", 24 * time.Hour, "1d"},
		{"one day 23 hours", 24*time.Hour + 23*time.Hour, "1d"},
		{"6 days", 6 * 24 * time.Hour, "6d"},

		// Weeks - shows only weeks
		{"one week", 7 * 24 * time.Hour, "1w"},
		{"one week 6 days", 7*24*time.Hour + 6*24*time.Hour, "1w"},
		{"3 weeks", 3 * 7 * 24 * time.Hour, "3w"},

		// Negative durations
		{"negative 5 minutes", -5 * time.Minute, "-5m"},
		{"negative 2 hours", -2 * time.Hour, "-2h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDurationCompact(tt.duration)
			if result != tt.expected {
				t.Errorf("FormatDurationCompact(%v) = %q, want %q", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestFormatDuration_SubSecondValuesRoundDown(t *testing.T) {
	// Subsecond values should be treated as zero seconds
	result := FormatDuration(500 * time.Millisecond)
	if result != "0s" {
		t.Errorf("FormatDuration(500ms) = %q, want %q", result, "0s")
	}

	result = FormatDuration(999 * time.Millisecond)
	if result != "0s" {
		t.Errorf("FormatDuration(999ms) = %q, want %q", result, "0s")
	}
}

func TestFormatDurationCompact_SubSecondValuesRoundDown(t *testing.T) {
	// Subsecond values should be treated as zero seconds
	result := FormatDurationCompact(500 * time.Millisecond)
	if result != "0s" {
		t.Errorf("FormatDurationCompact(500ms) = %q, want %q", result, "0s")
	}
}

// Benchmark tests
func BenchmarkFormatDuration(b *testing.B) {
	testCases := []time.Duration{
		0,
		45 * time.Second,
		2*time.Minute + 30*time.Second,
		2*time.Hour + 34*time.Minute,
		3*24*time.Hour + 5*time.Hour,
		2*7*24*time.Hour + 3*24*time.Hour,
	}

	for _, tc := range testCases {
		b.Run(FormatDuration(tc), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				FormatDuration(tc)
			}
		})
	}
}

func BenchmarkFormatDurationCompact(b *testing.B) {
	testCases := []time.Duration{
		0,
		45 * time.Second,
		2 * time.Minute,
		2 * time.Hour,
		3 * 24 * time.Hour,
		2 * 7 * 24 * time.Hour,
	}

	for _, tc := range testCases {
		b.Run(FormatDurationCompact(tc), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				FormatDurationCompact(tc)
			}
		})
	}
}
