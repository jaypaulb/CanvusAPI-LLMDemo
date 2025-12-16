// Package handlers provides the AIClientFactory molecule for creating OpenAI clients.
package handlers

import (
	"net/http"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

// AIClientConfig holds configuration for creating an OpenAI client.
// This struct consolidates the various configuration options used across
// different AI client creation patterns in the codebase.
type AIClientConfig struct {
	// APIKey is the OpenAI or compatible API key
	APIKey string

	// BaseURL is the primary API endpoint URL
	// For text: typically TextLLMURL or BaseLLMURL
	// For images: typically ImageLLMURL
	BaseURL string

	// FallbackURL is used if BaseURL is empty (optional)
	// Used for TextLLMURL → BaseLLMURL fallback pattern
	FallbackURL string

	// HTTPClient is a pre-configured HTTP client
	// Should include TLS settings and timeouts
	HTTPClient *http.Client

	// Timeout is the request timeout (used if HTTPClient is nil)
	Timeout time.Duration
}

// AIClientFactory creates OpenAI-compatible clients with consistent configuration.
// This molecule consolidates the client creation logic that was duplicated 5+ times
// across the codebase, ensuring consistent TLS, timeout, and URL configuration.
//
// This is a molecule that provides factory functions for creating AI clients.
//
// Example:
//
//	factory := handlers.NewAIClientFactory()
//	client := factory.CreateTextClient(config)
//	// Use client for chat completions
type AIClientFactory struct{}

// NewAIClientFactory creates a new AIClientFactory instance.
func NewAIClientFactory() *AIClientFactory {
	return &AIClientFactory{}
}

// CreateClient creates an OpenAI client with the given configuration.
// This is the primary factory method that handles all configuration options.
//
// Example:
//
//	client := factory.CreateClient(handlers.AIClientConfig{
//	    APIKey:      config.OpenAIAPIKey,
//	    BaseURL:     config.TextLLMURL,
//	    FallbackURL: config.BaseLLMURL,
//	    HTTPClient:  core.GetHTTPClient(config, config.AITimeout),
//	})
func (f *AIClientFactory) CreateClient(cfg AIClientConfig) *openai.Client {
	clientConfig := openai.DefaultConfig(cfg.APIKey)

	// Determine the base URL with fallback logic
	baseURL := ResolveBaseURL(cfg.BaseURL, cfg.FallbackURL)
	if baseURL != "" {
		clientConfig.BaseURL = baseURL
	}

	// Set HTTP client if provided
	if cfg.HTTPClient != nil {
		clientConfig.HTTPClient = cfg.HTTPClient
	}

	return openai.NewClientWithConfig(clientConfig)
}

// CreateTextClient creates a client configured for text/chat completions.
// Uses the standard TextLLMURL → BaseLLMURL fallback pattern.
//
// Example:
//
//	client := factory.CreateTextClient(
//	    apiKey,
//	    textLLMURL,
//	    baseLLMURL,
//	    httpClient,
//	)
func (f *AIClientFactory) CreateTextClient(apiKey, textLLMURL, baseLLMURL string, httpClient *http.Client) *openai.Client {
	return f.CreateClient(AIClientConfig{
		APIKey:      apiKey,
		BaseURL:     textLLMURL,
		FallbackURL: baseLLMURL,
		HTTPClient:  httpClient,
	})
}

// CreateImageClient creates a client configured for image generation.
// Uses a specific endpoint without fallback.
//
// Example:
//
//	client := factory.CreateImageClient(
//	    apiKey,
//	    imageLLMURL,
//	    httpClient,
//	)
func (f *AIClientFactory) CreateImageClient(apiKey, imageURL string, httpClient *http.Client) *openai.Client {
	return f.CreateClient(AIClientConfig{
		APIKey:     apiKey,
		BaseURL:    imageURL,
		HTTPClient: httpClient,
	})
}

// ResolveBaseURL returns the primary URL if non-empty, otherwise the fallback.
// This is a pure function (atom) that can be used independently.
//
// Example:
//
//	url := handlers.ResolveBaseURL("", "http://fallback.com")
//	// url == "http://fallback.com"
func ResolveBaseURL(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

// IsLocalEndpoint checks if a URL points to localhost.
// Useful for checking if image generation is supported (local endpoints often don't support it).
//
// Example:
//
//	if handlers.IsLocalEndpoint(endpoint) {
//	    return fmt.Errorf("image generation not supported on local endpoint")
//	}
func IsLocalEndpoint(url string) bool {
	// Check common localhost patterns
	localPatterns := []string{
		"127.0.0.1",
		"localhost",
		"0.0.0.0",
		"[::1]",
	}

	for _, pattern := range localPatterns {
		if containsIgnoreCase(url, pattern) {
			return true
		}
	}
	return false
}

// containsIgnoreCase checks if str contains substr (case-insensitive).
func containsIgnoreCase(str, substr string) bool {
	strLower := toLowerASCII(str)
	substrLower := toLowerASCII(substr)

	for i := 0; i <= len(strLower)-len(substrLower); i++ {
		if strLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

// toLowerASCII converts ASCII characters to lowercase without importing strings.
// This keeps the atom pure with no external dependencies.
func toLowerASCII(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// ClientType represents the type of AI client to create.
type ClientType int

const (
	// TextClient is for chat completions and text generation
	TextClient ClientType = iota
	// ImageClient is for image generation (DALL-E, etc.)
	ImageClient
	// VisionClient is for vision/multimodal models
	VisionClient
)

// String returns a string representation of the ClientType.
func (ct ClientType) String() string {
	switch ct {
	case TextClient:
		return "text"
	case ImageClient:
		return "image"
	case VisionClient:
		return "vision"
	default:
		return "unknown"
	}
}
