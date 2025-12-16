package core

import (
	"testing"
)

func TestBuildRangeHeader(t *testing.T) {
	tests := []struct {
		name       string
		resumeFrom int64
		expected   string
	}{
		{"zero offset", 0, "bytes=0-"},
		{"1KB offset", 1024, "bytes=1024-"},
		{"1MB offset", 1048576, "bytes=1048576-"},
		{"1GB offset", 1073741824, "bytes=1073741824-"},
		{"negative treated as zero", -100, "bytes=0-"},
		{"negative one treated as zero", -1, "bytes=0-"},
		{"arbitrary offset", 12345, "bytes=12345-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildRangeHeader(tt.resumeFrom)
			if result != tt.expected {
				t.Errorf("BuildRangeHeader(%d) = %q, want %q", tt.resumeFrom, result, tt.expected)
			}
		})
	}
}

func TestBuildRangeHeaderWithEnd(t *testing.T) {
	tests := []struct {
		name     string
		start    int64
		end      int64
		expected string
	}{
		{"first 1000 bytes", 0, 999, "bytes=0-999"},
		{"second 1000 bytes", 1000, 1999, "bytes=1000-1999"},
		{"single byte", 100, 100, "bytes=100-100"},
		{"start greater than end swapped", 999, 0, "bytes=0-999"},
		{"negative start treated as zero", -10, 100, "bytes=0-100"},
		{"negative end treated as zero", 0, -10, "bytes=0-0"},
		{"both negative", -10, -5, "bytes=0-0"},
		{"large range", 0, 1073741823, "bytes=0-1073741823"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildRangeHeaderWithEnd(tt.start, tt.end)
			if result != tt.expected {
				t.Errorf("BuildRangeHeaderWithEnd(%d, %d) = %q, want %q", tt.start, tt.end, result, tt.expected)
			}
		})
	}
}

func TestBuildRangeHeaderSuffix(t *testing.T) {
	tests := []struct {
		name         string
		suffixLength int64
		expected     string
	}{
		{"last 500 bytes", 500, "bytes=-500"},
		{"last 1KB", 1024, "bytes=-1024"},
		{"last 1MB", 1048576, "bytes=-1048576"},
		{"single byte", 1, "bytes=-1"},
		{"zero treated as one", 0, "bytes=-1"},
		{"negative treated as one", -100, "bytes=-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildRangeHeaderSuffix(tt.suffixLength)
			if result != tt.expected {
				t.Errorf("BuildRangeHeaderSuffix(%d) = %q, want %q", tt.suffixLength, result, tt.expected)
			}
		})
	}
}

func TestParseContentRange(t *testing.T) {
	tests := []struct {
		name        string
		header      string
		wantStart   int64
		wantEnd     int64
		wantTotal   int64
		expectError bool
	}{
		{
			name:        "standard format",
			header:      "bytes 0-999/5000",
			wantStart:   0,
			wantEnd:     999,
			wantTotal:   5000,
			expectError: false,
		},
		{
			name:        "partial with unknown total",
			header:      "bytes 1000-1999/*",
			wantStart:   1000,
			wantEnd:     1999,
			wantTotal:   -1,
			expectError: false,
		},
		{
			name:        "large file",
			header:      "bytes 0-1073741823/8589934592",
			wantStart:   0,
			wantEnd:     1073741823,
			wantTotal:   8589934592,
			expectError: false,
		},
		{
			name:        "single byte",
			header:      "bytes 100-100/200",
			wantStart:   100,
			wantEnd:     100,
			wantTotal:   200,
			expectError: false,
		},
		{
			name:        "empty header",
			header:      "",
			expectError: true,
		},
		{
			name:        "invalid format",
			header:      "invalid",
			expectError: true,
		},
		{
			name:        "missing bytes prefix",
			header:      "0-999/5000",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, total, err := ParseContentRange(tt.header)
			if tt.expectError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if start != tt.wantStart {
				t.Errorf("start = %d, want %d", start, tt.wantStart)
			}
			if end != tt.wantEnd {
				t.Errorf("end = %d, want %d", end, tt.wantEnd)
			}
			if total != tt.wantTotal {
				t.Errorf("total = %d, want %d", total, tt.wantTotal)
			}
		})
	}
}

func TestIsPartialContentSupported(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		expected bool
	}{
		{"bytes supported", "bytes", true},
		{"none", "none", false},
		{"empty", "", false},
		{"other value", "custom", false},
		{"BYTES uppercase", "BYTES", false}, // HTTP headers are case-sensitive for values
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPartialContentSupported(tt.header)
			if result != tt.expected {
				t.Errorf("IsPartialContentSupported(%q) = %v, want %v", tt.header, result, tt.expected)
			}
		})
	}
}
