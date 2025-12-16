package core

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		// Zero and small values
		{"zero bytes", 0, "0 B"},
		{"one byte", 1, "1 B"},
		{"512 bytes", 512, "512 B"},
		{"1023 bytes", 1023, "1023 B"},

		// Kilobytes
		{"exactly 1 KB", 1024, "1.00 KB"},
		{"1.5 KB", 1536, "1.50 KB"},
		{"10 KB", 10 * 1024, "10.00 KB"},
		{"999 KB", 999 * 1024, "999.00 KB"},

		// Megabytes
		{"exactly 1 MB", 1024 * 1024, "1.00 MB"},
		{"1.5 MB", 1536 * 1024, "1.50 MB"},
		{"100 MB", 100 * 1024 * 1024, "100.00 MB"},
		{"999 MB", 999 * 1024 * 1024, "999.00 MB"},

		// Gigabytes
		{"exactly 1 GB", 1024 * 1024 * 1024, "1.00 GB"},
		{"1.5 GB", 1536 * 1024 * 1024, "1.50 GB"},
		{"8 GB (typical model size)", 8 * 1024 * 1024 * 1024, "8.00 GB"},

		// Terabytes
		{"exactly 1 TB", 1024 * 1024 * 1024 * 1024, "1.00 TB"},
		{"2.5 TB", int64(2.5 * 1024 * 1024 * 1024 * 1024), "2.50 TB"},

		// Negative values (should be treated as 0)
		{"negative value", -100, "0 B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatBytesCompact(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		// Zero and small values
		{"zero bytes", 0, "0 B"},
		{"one byte", 1, "1 B"},
		{"512 bytes", 512, "512 B"},

		// Round kilobytes (no decimal)
		{"exactly 1 KB", 1024, "1 KB"},
		{"exactly 10 KB", 10 * 1024, "10 KB"},

		// Non-round kilobytes (with decimal)
		{"1.5 KB", 1536, "1.5 KB"},

		// Round megabytes
		{"exactly 1 MB", 1024 * 1024, "1 MB"},
		{"exactly 100 MB", 100 * 1024 * 1024, "100 MB"},

		// Non-round megabytes
		{"1.5 MB", 1536 * 1024, "1.5 MB"},

		// Round gigabytes
		{"exactly 1 GB", 1024 * 1024 * 1024, "1 GB"},
		{"exactly 8 GB", 8 * 1024 * 1024 * 1024, "8 GB"},

		// Non-round gigabytes
		{"7.5 GB", int64(7.5 * 1024 * 1024 * 1024), "7.5 GB"},

		// Negative values
		{"negative value", -100, "0 B"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytesCompact(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytesCompact(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestParseBytes(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int64
		expectError bool
	}{
		// Basic values
		{"zero bytes", "0", 0, false},
		{"zero bytes explicit", "0B", 0, false},
		{"100 bytes", "100B", 100, false},
		{"100 bytes with space", "100 B", 100, false},

		// Kilobytes
		{"1 KB", "1KB", 1024, false},
		{"1 KB with space", "1 KB", 1024, false},
		{"1.5 KB", "1.5KB", 1536, false},
		{"1K shorthand", "1K", 1024, false},

		// Megabytes
		{"1 MB", "1MB", 1024 * 1024, false},
		{"1.5 MB", "1.5MB", 1536 * 1024, false},
		{"1M shorthand", "1M", 1024 * 1024, false},

		// Gigabytes
		{"1 GB", "1GB", 1024 * 1024 * 1024, false},
		{"8 GB", "8GB", 8 * 1024 * 1024 * 1024, false},
		{"1G shorthand", "1G", 1024 * 1024 * 1024, false},

		// Terabytes
		{"1 TB", "1TB", 1024 * 1024 * 1024 * 1024, false},
		{"1T shorthand", "1T", 1024 * 1024 * 1024 * 1024, false},

		// Case insensitivity
		{"lowercase kb", "1kb", 1024, false},
		{"lowercase mb", "1mb", 1024 * 1024, false},
		{"mixed case Kb", "1Kb", 1024, false},

		// Error cases
		{"empty string", "", 0, true},
		{"only whitespace", "   ", 0, true},
		{"no number", "KB", 0, true},
		{"invalid unit", "100XB", 0, true},
		{"negative value", "-100KB", 0, true},
		{"invalid format", "abc", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseBytes(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseBytes(%q) expected error, got %d", tt.input, result)
				}
			} else {
				if err != nil {
					t.Errorf("ParseBytes(%q) unexpected error: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("ParseBytes(%q) = %d, want %d", tt.input, result, tt.expected)
				}
			}
		})
	}
}

func TestBytesConstants(t *testing.T) {
	// Verify constants are correctly defined
	if BytesPerKB != 1024 {
		t.Errorf("BytesPerKB = %d, want 1024", BytesPerKB)
	}
	if BytesPerMB != 1024*1024 {
		t.Errorf("BytesPerMB = %d, want %d", BytesPerMB, 1024*1024)
	}
	if BytesPerGB != 1024*1024*1024 {
		t.Errorf("BytesPerGB = %d, want %d", BytesPerGB, 1024*1024*1024)
	}
	if BytesPerTB != 1024*1024*1024*1024 {
		t.Errorf("BytesPerTB = %d, want %d", BytesPerTB, int64(1024*1024*1024*1024))
	}
}

func TestFormatBytesRoundTrip(t *testing.T) {
	// Test that parsing a formatted value gives back approximately the original
	testValues := []int64{
		0,
		1024,
		1024 * 1024,
		1024 * 1024 * 1024,
		8 * 1024 * 1024 * 1024,
	}

	for _, original := range testValues {
		formatted := FormatBytes(original)
		// Extract just the number and unit from formatted string
		var value float64
		var unit string
		_, err := parseFormattedBytes(formatted, &value, &unit)
		if err != nil {
			t.Errorf("Failed to parse formatted value %q: %v", formatted, err)
			continue
		}

		// Parse back
		parsed, err := ParseBytes(formatted)
		if err != nil {
			t.Errorf("ParseBytes(%q) error: %v", formatted, err)
			continue
		}

		// Allow small rounding differences (within 0.01 of the unit)
		diff := original - parsed
		if diff < 0 {
			diff = -diff
		}

		// For exact values, they should match exactly
		if original == parsed {
			continue
		}

		// For non-exact, difference should be less than 1% of original
		if original > 0 && float64(diff)/float64(original) > 0.01 {
			t.Errorf("Round trip %d -> %q -> %d, diff too large", original, formatted, parsed)
		}
	}
}

// parseFormattedBytes is a helper to parse the formatted output
func parseFormattedBytes(s string, value *float64, unit *string) (int, error) {
	return 0, nil // Simplified - actual parsing done in ParseBytes
}

// Benchmarks
func BenchmarkFormatBytes(b *testing.B) {
	testCases := []int64{0, 1024, 1024 * 1024, 1024 * 1024 * 1024}
	for _, tc := range testCases {
		b.Run(FormatBytes(tc), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				FormatBytes(tc)
			}
		})
	}
}

func BenchmarkParseBytes(b *testing.B) {
	testCases := []string{"0B", "1KB", "1MB", "1GB", "8GB"}
	for _, tc := range testCases {
		b.Run(tc, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ParseBytes(tc)
			}
		})
	}
}
