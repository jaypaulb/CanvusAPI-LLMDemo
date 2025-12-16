package pdfprocessor

import "testing"

func TestEstimateTokenCount(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected int
	}{
		{
			name:     "empty string returns 0",
			text:     "",
			expected: 0,
		},
		{
			name:     "4 characters returns 1 token",
			text:     "test",
			expected: 1,
		},
		{
			name:     "8 characters returns 2 tokens",
			text:     "testtest",
			expected: 2,
		},
		{
			name:     "13 characters returns 3 tokens (floor division)",
			text:     "Hello, world!",
			expected: 3,
		},
		{
			name:     "1 character returns 0 tokens",
			text:     "a",
			expected: 0,
		},
		{
			name:     "3 characters returns 0 tokens",
			text:     "abc",
			expected: 0,
		},
		{
			name:     "100 characters returns 25 tokens",
			text:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			expected: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateTokenCount(tt.text)
			if result != tt.expected {
				t.Errorf("EstimateTokenCount(%q) = %d, want %d", tt.text, result, tt.expected)
			}
		})
	}
}

func TestTruncateText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxLen   int
		expected string
	}{
		{
			name:     "empty string unchanged",
			text:     "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "short text unchanged",
			text:     "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length unchanged",
			text:     "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long text truncated",
			text:     "Hello, world!",
			maxLen:   5,
			expected: "Hello",
		},
		{
			name:     "zero maxLen returns empty",
			text:     "hello",
			maxLen:   0,
			expected: "",
		},
		{
			name:     "negative maxLen returns empty",
			text:     "hello",
			maxLen:   -5,
			expected: "",
		},
		{
			name:     "truncate to 1 character",
			text:     "hello",
			maxLen:   1,
			expected: "h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateText(tt.text, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateText(%q, %d) = %q, want %q", tt.text, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestTruncateTextWithEllipsis(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxLen   int
		expected string
	}{
		{
			name:     "empty string unchanged",
			text:     "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "short text unchanged",
			text:     "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length unchanged",
			text:     "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "long text gets ellipsis",
			text:     "Hello, world!",
			maxLen:   8,
			expected: "Hello...",
		},
		{
			name:     "maxLen 4 gives single char plus ellipsis",
			text:     "Hello",
			maxLen:   4,
			expected: "H...",
		},
		{
			name:     "maxLen 3 too short for ellipsis",
			text:     "Hello",
			maxLen:   3,
			expected: "Hel",
		},
		{
			name:     "maxLen 2 too short for ellipsis",
			text:     "Hello",
			maxLen:   2,
			expected: "He",
		},
		{
			name:     "zero maxLen returns empty",
			text:     "hello",
			maxLen:   0,
			expected: "",
		},
		{
			name:     "negative maxLen returns empty",
			text:     "hello",
			maxLen:   -5,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateTextWithEllipsis(tt.text, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateTextWithEllipsis(%q, %d) = %q, want %q", tt.text, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// Benchmark tests
func BenchmarkEstimateTokenCount(b *testing.B) {
	text := "This is a sample text for benchmarking the token estimation function."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EstimateTokenCount(text)
	}
}

func BenchmarkTruncateText(b *testing.B) {
	text := "This is a sample text for benchmarking the truncation function with a long string."
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		TruncateText(text, 20)
	}
}
