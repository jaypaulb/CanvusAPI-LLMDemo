package imagegen

import (
	"context"
	"testing"
	"time"

	"go_backend/core"
)

// TestOpenAIProviderInterface verifies that OpenAIProvider implements Provider.
func TestOpenAIProviderInterface(t *testing.T) {
	// This is a compile-time check via the var _ Provider = (*OpenAIProvider)(nil) line
	// in openai_provider.go, but we include this test for documentation.
	var _ Provider = (*OpenAIProvider)(nil)
}

// TestNewOpenAIProvider_NilConfig tests that NewOpenAIProvider returns error for nil config.
func TestNewOpenAIProvider_NilConfig(t *testing.T) {
	provider, err := NewOpenAIProvider(nil)

	if err == nil {
		t.Error("expected error for nil config, got nil")
	}
	if provider != nil {
		t.Error("expected nil provider for nil config")
	}
	if err != nil && err.Error() != "imagegen: config cannot be nil" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

// TestNewOpenAIProvider_EmptyAPIKey tests that NewOpenAIProvider returns error for empty API key.
func TestNewOpenAIProvider_EmptyAPIKey(t *testing.T) {
	cfg := &core.Config{
		OpenAIAPIKey: "",
		ImageLLMURL:  "https://api.openai.com/v1",
	}

	provider, err := NewOpenAIProvider(cfg)

	if err == nil {
		t.Error("expected error for empty API key, got nil")
	}
	if provider != nil {
		t.Error("expected nil provider for empty API key")
	}
}

// TestNewOpenAIProvider_LocalEndpoint tests that local endpoints are rejected.
func TestNewOpenAIProvider_LocalEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
	}{
		{"localhost", "http://localhost:1234"},
		{"127.0.0.1", "http://127.0.0.1:8080"},
		{"192.168.x.x", "http://192.168.1.100:5000"},
		{"10.x.x.x", "http://10.0.0.1:8000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &core.Config{
				OpenAIAPIKey: "test-key",
				ImageLLMURL:  tt.endpoint,
			}

			provider, err := NewOpenAIProvider(cfg)

			if err == nil {
				t.Errorf("expected error for local endpoint %s, got nil", tt.endpoint)
			}
			if provider != nil {
				t.Errorf("expected nil provider for local endpoint %s", tt.endpoint)
			}
		})
	}
}

// TestNewOpenAIProvider_ValidConfig tests successful provider creation.
func TestNewOpenAIProvider_ValidConfig(t *testing.T) {
	cfg := &core.Config{
		OpenAIAPIKey:     "test-api-key-12345",
		ImageLLMURL:      "https://api.openai.com/v1",
		OpenAIImageModel: "dall-e-3",
		AITimeout:        30 * time.Second,
	}

	provider, err := NewOpenAIProvider(cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
	if provider.Model() != "dall-e-3" {
		t.Errorf("expected model dall-e-3, got %s", provider.Model())
	}
}

// TestNewOpenAIProvider_DefaultModel tests that default model is dall-e-3.
func TestNewOpenAIProvider_DefaultModel(t *testing.T) {
	cfg := &core.Config{
		OpenAIAPIKey:     "test-api-key",
		ImageLLMURL:      "https://api.openai.com/v1",
		OpenAIImageModel: "", // Empty should default to dall-e-3
	}

	provider, err := NewOpenAIProvider(cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.Model() != "dall-e-3" {
		t.Errorf("expected default model dall-e-3, got %s", provider.Model())
	}
}

// TestNewOpenAIProvider_DefaultEndpoint tests that empty ImageLLMURL defaults to OpenAI.
func TestNewOpenAIProvider_DefaultEndpoint(t *testing.T) {
	cfg := &core.Config{
		OpenAIAPIKey: "test-api-key",
		ImageLLMURL:  "", // Empty should default to OpenAI
	}

	provider, err := NewOpenAIProvider(cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
}

// TestNewOpenAIProviderWithConfig_EmptyAPIKey tests explicit config with empty API key.
func TestNewOpenAIProviderWithConfig_EmptyAPIKey(t *testing.T) {
	cfg := OpenAIProviderConfig{
		APIKey: "",
	}

	provider, err := NewOpenAIProviderWithConfig(cfg, nil)

	if err == nil {
		t.Error("expected error for empty API key, got nil")
	}
	if provider != nil {
		t.Error("expected nil provider for empty API key")
	}
}

// TestNewOpenAIProviderWithConfig_LocalEndpoint tests explicit config with local endpoint.
func TestNewOpenAIProviderWithConfig_LocalEndpoint(t *testing.T) {
	cfg := OpenAIProviderConfig{
		APIKey:  "test-key",
		BaseURL: "http://localhost:1234",
	}

	provider, err := NewOpenAIProviderWithConfig(cfg, nil)

	if err == nil {
		t.Error("expected error for local endpoint, got nil")
	}
	if provider != nil {
		t.Error("expected nil provider for local endpoint")
	}
}

// TestNewOpenAIProviderWithConfig_ValidConfig tests explicit config creation.
func TestNewOpenAIProviderWithConfig_ValidConfig(t *testing.T) {
	cfg := OpenAIProviderConfig{
		APIKey:  "test-api-key",
		BaseURL: "https://api.openai.com/v1",
		Model:   "dall-e-2",
	}

	provider, err := NewOpenAIProviderWithConfig(cfg, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
	if provider.Model() != "dall-e-2" {
		t.Errorf("expected model dall-e-2, got %s", provider.Model())
	}
}

// TestOpenAIProvider_Generate_EmptyPrompt tests that empty prompt returns error.
func TestOpenAIProvider_Generate_EmptyPrompt(t *testing.T) {
	cfg := &core.Config{
		OpenAIAPIKey: "test-api-key",
		ImageLLMURL:  "https://api.openai.com/v1",
	}

	provider, err := NewOpenAIProvider(cfg)
	if err != nil {
		t.Fatalf("unexpected error creating provider: %v", err)
	}

	url, err := provider.Generate(context.Background(), "")

	if err == nil {
		t.Error("expected error for empty prompt, got nil")
	}
	if url != "" {
		t.Errorf("expected empty URL for empty prompt, got %s", url)
	}
	if err != nil && err.Error() != "imagegen: prompt cannot be empty" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

// TestDefaultOpenAIProviderConfig tests default configuration values.
func TestDefaultOpenAIProviderConfig(t *testing.T) {
	cfg := DefaultOpenAIProviderConfig()

	if cfg.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("expected default BaseURL https://api.openai.com/v1, got %s", cfg.BaseURL)
	}
	if cfg.Model != "dall-e-3" {
		t.Errorf("expected default Model dall-e-3, got %s", cfg.Model)
	}
}

// TestOpenAIProvider_Model tests the Model() method.
func TestOpenAIProvider_Model(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected string
	}{
		{"dall-e-3", "dall-e-3", "dall-e-3"},
		{"dall-e-2", "dall-e-2", "dall-e-2"},
		{"empty defaults to dall-e-3", "", "dall-e-3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &core.Config{
				OpenAIAPIKey:     "test-api-key",
				ImageLLMURL:      "https://api.openai.com/v1",
				OpenAIImageModel: tt.model,
			}

			provider, err := NewOpenAIProvider(cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if provider.Model() != tt.expected {
				t.Errorf("expected model %s, got %s", tt.expected, provider.Model())
			}
		})
	}
}

// Note: Integration tests that actually call the OpenAI API should be in a
// separate _integration_test.go file and only run with specific build tags
// or environment variables to avoid accidental API charges.
