package core

import "fmt"

// Byte size constants for human-readable formatting.
// Using binary units (1024 base) as is standard for file sizes.
const (
	BytesPerKB int64 = 1024
	BytesPerMB int64 = 1024 * BytesPerKB
	BytesPerGB int64 = 1024 * BytesPerMB
	BytesPerTB int64 = 1024 * BytesPerGB
)

// FormatBytes converts a byte count to a human-readable string.
// Uses binary units (KiB = 1024 bytes) but displays as KB/MB/GB/TB for familiarity.
// Examples:
//   - FormatBytes(0) returns "0 B"
//   - FormatBytes(512) returns "512 B"
//   - FormatBytes(1024) returns "1.00 KB"
//   - FormatBytes(1536) returns "1.50 KB"
//   - FormatBytes(1048576) returns "1.00 MB"
//   - FormatBytes(1073741824) returns "1.00 GB"
//   - FormatBytes(1099511627776) returns "1.00 TB"
//
// This is a pure function with no side effects.
func FormatBytes(bytes int64) string {
	// Handle negative values by treating as 0
	if bytes < 0 {
		bytes = 0
	}

	switch {
	case bytes >= BytesPerTB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(BytesPerTB))
	case bytes >= BytesPerGB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(BytesPerGB))
	case bytes >= BytesPerMB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(BytesPerMB))
	case bytes >= BytesPerKB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(BytesPerKB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatBytesCompact returns a more compact representation without decimal places
// when the value is a round number of the unit.
// Examples:
//   - FormatBytesCompact(1024) returns "1 KB"
//   - FormatBytesCompact(1536) returns "1.5 KB"
//   - FormatBytesCompact(2097152) returns "2 MB"
//
// This is a pure function with no side effects.
func FormatBytesCompact(bytes int64) string {
	if bytes < 0 {
		bytes = 0
	}

	switch {
	case bytes >= BytesPerTB:
		val := float64(bytes) / float64(BytesPerTB)
		if val == float64(int64(val)) {
			return fmt.Sprintf("%.0f TB", val)
		}
		return fmt.Sprintf("%.1f TB", val)
	case bytes >= BytesPerGB:
		val := float64(bytes) / float64(BytesPerGB)
		if val == float64(int64(val)) {
			return fmt.Sprintf("%.0f GB", val)
		}
		return fmt.Sprintf("%.1f GB", val)
	case bytes >= BytesPerMB:
		val := float64(bytes) / float64(BytesPerMB)
		if val == float64(int64(val)) {
			return fmt.Sprintf("%.0f MB", val)
		}
		return fmt.Sprintf("%.1f MB", val)
	case bytes >= BytesPerKB:
		val := float64(bytes) / float64(BytesPerKB)
		if val == float64(int64(val)) {
			return fmt.Sprintf("%.0f KB", val)
		}
		return fmt.Sprintf("%.1f KB", val)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// ParseBytes converts a human-readable size string to bytes.
// Supported formats: "100B", "10KB", "5MB", "2GB", "1TB" (case-insensitive).
// Whitespace between number and unit is allowed.
// Returns 0 and error if the format is invalid.
// Examples:
//   - ParseBytes("1KB") returns 1024, nil
//   - ParseBytes("1.5 MB") returns 1572864, nil
//   - ParseBytes("2GB") returns 2147483648, nil
//
// This is a pure function with no side effects.
func ParseBytes(s string) (int64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	// Remove whitespace
	s = trimWhitespace(s)
	if s == "" {
		return 0, fmt.Errorf("empty string after trimming")
	}

	// Find where the number ends and the unit begins
	var numEnd int
	for i, c := range s {
		if (c < '0' || c > '9') && c != '.' && c != '-' {
			numEnd = i
			break
		}
		numEnd = i + 1
	}

	if numEnd == 0 {
		return 0, fmt.Errorf("invalid format: no number found")
	}

	numStr := s[:numEnd]
	unit := trimWhitespace(s[numEnd:])

	var value float64
	_, err := fmt.Sscanf(numStr, "%f", &value)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", numStr)
	}

	if value < 0 {
		return 0, fmt.Errorf("negative values not allowed")
	}

	// Handle unit (case-insensitive)
	var multiplier int64 = 1
	switch toUpper(unit) {
	case "", "B":
		multiplier = 1
	case "KB", "K":
		multiplier = BytesPerKB
	case "MB", "M":
		multiplier = BytesPerMB
	case "GB", "G":
		multiplier = BytesPerGB
	case "TB", "T":
		multiplier = BytesPerTB
	default:
		return 0, fmt.Errorf("unknown unit: %s", unit)
	}

	return int64(value * float64(multiplier)), nil
}

// trimWhitespace removes leading and trailing whitespace without importing strings.
func trimWhitespace(s string) string {
	start := 0
	end := len(s)
	for start < end && isWhitespace(s[start]) {
		start++
	}
	for end > start && isWhitespace(s[end-1]) {
		end--
	}
	return s[start:end]
}

// isWhitespace returns true if the byte is a whitespace character.
func isWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// toUpper converts a string to uppercase without importing strings.
func toUpper(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 32 // Convert to uppercase
		}
		b[i] = c
	}
	return string(b)
}
