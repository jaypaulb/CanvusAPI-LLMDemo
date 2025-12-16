// Package imagegen provides image generation utilities for the Canvus canvas.
//
// atoms.go contains pure utility functions with no dependencies.
package imagegen

import (
	"strings"
)

// IsAzureEndpoint checks if the given endpoint URL is an Azure OpenAI endpoint.
// It performs case-insensitive substring matching against known Azure domain patterns.
//
// This is a pure function with no dependencies - it simply performs string matching.
//
// Azure OpenAI endpoints typically match one of these patterns:
//   - *.openai.azure.com
//   - *.cognitiveservices.azure.com
//
// Example:
//
//	IsAzureEndpoint("https://myresource.openai.azure.com")        // true
//	IsAzureEndpoint("https://myresource.cognitiveservices.azure.com") // true
//	IsAzureEndpoint("https://api.openai.com")                     // false
//	IsAzureEndpoint("")                                            // false
func IsAzureEndpoint(endpoint string) bool {
	if endpoint == "" {
		return false
	}
	lower := strings.ToLower(endpoint)
	return strings.Contains(lower, "openai.azure.com") ||
		strings.Contains(lower, "cognitiveservices.azure.com")
}

// IsOpenAIEndpoint checks if the given endpoint URL is a standard OpenAI API endpoint.
// It performs case-insensitive substring matching against the OpenAI API domain.
//
// This is a pure function with no dependencies - it simply performs string matching.
//
// Standard OpenAI endpoints match:
//   - api.openai.com
//
// Example:
//
//	IsOpenAIEndpoint("https://api.openai.com/v1")  // true
//	IsOpenAIEndpoint("https://openai.azure.com")  // false (Azure endpoint)
//	IsOpenAIEndpoint("http://localhost:1234")     // false (local endpoint)
func IsOpenAIEndpoint(endpoint string) bool {
	if endpoint == "" {
		return false
	}
	lower := strings.ToLower(endpoint)
	return strings.Contains(lower, "api.openai.com")
}

// IsLocalEndpoint checks if the given endpoint URL is a local/self-hosted endpoint.
// It checks for localhost, 127.0.0.1, or common LAN patterns.
//
// This is a pure function with no dependencies - it simply performs string matching.
//
// Local endpoints match:
//   - localhost
//   - 127.0.0.1
//   - 0.0.0.0
//   - 192.168.*.* (common LAN range)
//   - 10.*.*.* (private network range)
//
// Example:
//
//	IsLocalEndpoint("http://localhost:1234")       // true
//	IsLocalEndpoint("http://127.0.0.1:8080")       // true
//	IsLocalEndpoint("http://192.168.1.100:5000")   // true
//	IsLocalEndpoint("https://api.openai.com")      // false
func IsLocalEndpoint(endpoint string) bool {
	if endpoint == "" {
		return false
	}
	lower := strings.ToLower(endpoint)
	return strings.Contains(lower, "localhost") ||
		strings.Contains(lower, "127.0.0.1") ||
		strings.Contains(lower, "0.0.0.0") ||
		strings.Contains(lower, "192.168.") ||
		strings.Contains(lower, "10.")
}
