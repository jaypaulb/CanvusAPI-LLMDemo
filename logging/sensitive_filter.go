package logging

import (
	"regexp"
	"strings"
)

// RedactedPlaceholder is the string used to replace sensitive data
const RedactedPlaceholder = "[REDACTED]"

// sensitivePatterns contains compiled regex patterns for detecting sensitive data.
// These patterns are compiled once at package initialization for performance.
var sensitivePatterns = []*regexp.Regexp{
	// API keys - common formats
	// OpenAI API keys: sk-... (legacy) or sk-proj-... (project-scoped)
	regexp.MustCompile(`(?i)(sk-[a-zA-Z0-9_-]{20,})`),
	regexp.MustCompile(`(?i)(AIza[a-zA-Z0-9_-]{35})`),        // Google API keys
	regexp.MustCompile(`(?i)([a-f0-9]{32})`),                 // Generic 32-char hex (many API keys)
	regexp.MustCompile(`(?i)(ghp_[a-zA-Z0-9]{36})`),          // GitHub tokens
	regexp.MustCompile(`(?i)(gho_[a-zA-Z0-9]{36})`),          // GitHub OAuth tokens
	regexp.MustCompile(`(?i)(github_pat_[a-zA-Z0-9_]{22,})`), // GitHub fine-grained tokens
	regexp.MustCompile(`(?i)(xox[baprs]-[a-zA-Z0-9-]{10,})`), // Slack tokens
	regexp.MustCompile(`(?i)(bearer\s+[a-zA-Z0-9._-]{20,})`), // Bearer tokens

	// Azure patterns
	regexp.MustCompile(`(?i)(DefaultEndpointsProtocol=[^;]+;[^"'\s]+)`), // Azure connection strings

	// Generic secret patterns
	regexp.MustCompile(`(?i)(password\s*[:=]\s*[^\s,;]{8,})`), // password= or password:
	regexp.MustCompile(`(?i)(secret\s*[:=]\s*[^\s,;]{8,})`),   // secret= or secret:
	regexp.MustCompile(`(?i)(token\s*[:=]\s*[^\s,;]{8,})`),    // token= or token:
	regexp.MustCompile(`(?i)(api_key\s*[:=]\s*[^\s,;]{8,})`),  // api_key= or api_key:
	regexp.MustCompile(`(?i)(apikey\s*[:=]\s*[^\s,;]{8,})`),   // apikey= or apikey:
}

// sensitiveEnvVarPrefixes are environment variable name prefixes that indicate sensitive data
var sensitiveEnvVarPrefixes = []string{
	"OPENAI_API_KEY",
	"CANVUS_API_KEY",
	"GOOGLE_VISION_API_KEY",
	"AZURE_OPENAI_KEY",
	"WEBUI_PWD",
	"PASSWORD",
	"SECRET",
	"TOKEN",
	"API_KEY",
	"APIKEY",
}

// RedactSensitiveData scans a string value and redacts any detected sensitive data.
// This is a pure function - it takes a string and returns a sanitized string.
//
// Patterns detected:
//   - OpenAI API keys (sk-... or sk-proj-...)
//   - Google API keys (AIza...)
//   - GitHub tokens (ghp_, gho_, github_pat_)
//   - Slack tokens (xox...)
//   - Bearer tokens
//   - Azure connection strings
//   - Generic password/secret/token assignments
//
// Example:
//
//	input := "API key is sk-abc123def456..."
//	output := RedactSensitiveData(input)
//	// output: "API key is [REDACTED]"
func RedactSensitiveData(value string) string {
	if value == "" {
		return value
	}

	result := value
	for _, pattern := range sensitivePatterns {
		result = pattern.ReplaceAllString(result, RedactedPlaceholder)
	}
	return result
}

// RedactField redacts a field value if the field name indicates sensitive data.
// This is useful for structured logging where field names are known.
//
// This is a pure function with no side effects.
//
// Example:
//
//	value := RedactField("OPENAI_API_KEY", "sk-secret123")
//	// value: "[REDACTED]"
//
//	value := RedactField("username", "john")
//	// value: "john" (unchanged)
func RedactField(fieldName, fieldValue string) string {
	upperName := strings.ToUpper(fieldName)

	// Check if field name indicates sensitive data
	for _, prefix := range sensitiveEnvVarPrefixes {
		if strings.Contains(upperName, prefix) {
			return RedactedPlaceholder
		}
	}

	// Also scan the value itself for sensitive patterns
	return RedactSensitiveData(fieldValue)
}

// IsSensitiveField returns true if the field name indicates sensitive data.
// This is a pure function that only checks the field name, not the value.
//
// Example:
//
//	IsSensitiveField("OPENAI_API_KEY")  // true
//	IsSensitiveField("username")        // false
func IsSensitiveField(fieldName string) bool {
	upperName := strings.ToUpper(fieldName)

	for _, prefix := range sensitiveEnvVarPrefixes {
		if strings.Contains(upperName, prefix) {
			return true
		}
	}
	return false
}

// ContainsSensitiveData returns true if the value contains any sensitive data patterns.
// This is a pure function that scans the value for known patterns.
//
// Example:
//
//	ContainsSensitiveData("sk-abc123...")     // true
//	ContainsSensitiveData("hello world")     // false
func ContainsSensitiveData(value string) bool {
	if value == "" {
		return false
	}

	for _, pattern := range sensitivePatterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	return false
}
