package imagegen

import (
	"context"
	"testing"
	"time"

	"go_backend/core"
)

// TestAzureProviderInterface verifies that AzureProvider implements Provider.
func TestAzureProviderInterface(t *testing.T) {
	var _ Provider = (*AzureProvider)(nil)
}

// TestNewAzureProvider_NilConfig tests that NewAzureProvider returns error for nil config.
func TestNewAzureProvider_NilConfig(t *testing.T) {
	provider, err := NewAzureProvider(nil)

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

// TestNewAzureProvider_EmptyAPIKey tests that NewAzureProvider returns error for empty API key.
func TestNewAzureProvider_EmptyAPIKey(t *testing.T) {
	cfg := &core.Config{
		OpenAIAPIKey:          "",
		AzureOpenAIEndpoint:   "https://myresource.openai.azure.com/",
		AzureOpenAIDeployment: "dalle3",
	}

	provider, err := NewAzureProvider(cfg)

	if err == nil {
		t.Error("expected error for empty API key, got nil")
	}
	if provider != nil {
		t.Error("expected nil provider for empty API key")
	}
}

// TestNewAzureProvider_EmptyEndpoint tests that NewAzureProvider returns error for empty endpoint.
func TestNewAzureProvider_EmptyEndpoint(t *testing.T) {
	cfg := &core.Config{
		OpenAIAPIKey:          "test-key",
		AzureOpenAIEndpoint:   "",
		ImageLLMURL:           "",
		AzureOpenAIDeployment: "dalle3",
	}

	provider, err := NewAzureProvider(cfg)

	if err == nil {
		t.Error("expected error for empty endpoint, got nil")
	}
	if provider != nil {
		t.Error("expected nil provider for empty endpoint")
	}
}

// TestNewAzureProvider_NonAzureEndpoint tests that non-Azure endpoints are rejected.
func TestNewAzureProvider_NonAzureEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
	}{
		{"OpenAI endpoint", "https://api.openai.com/v1"},
		{"localhost", "http://localhost:1234"},
		{"arbitrary", "https://example.com/api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &core.Config{
				OpenAIAPIKey:          "test-key",
				ImageLLMURL:           tt.endpoint,
				AzureOpenAIDeployment: "dalle3",
			}

			provider, err := NewAzureProvider(cfg)

			if err == nil {
				t.Errorf("expected error for non-Azure endpoint %s, got nil", tt.endpoint)
			}
			if provider != nil {
				t.Errorf("expected nil provider for non-Azure endpoint %s", tt.endpoint)
			}
		})
	}
}

// TestNewAzureProvider_EmptyDeployment tests that NewAzureProvider returns error for empty deployment.
func TestNewAzureProvider_EmptyDeployment(t *testing.T) {
	cfg := &core.Config{
		OpenAIAPIKey:          "test-key",
		AzureOpenAIEndpoint:   "https://myresource.openai.azure.com/",
		AzureOpenAIDeployment: "",
	}

	provider, err := NewAzureProvider(cfg)

	if err == nil {
		t.Error("expected error for empty deployment, got nil")
	}
	if provider != nil {
		t.Error("expected nil provider for empty deployment")
	}
}

// TestNewAzureProvider_ValidConfig tests successful provider creation.
func TestNewAzureProvider_ValidConfig(t *testing.T) {
	cfg := &core.Config{
		OpenAIAPIKey:          "test-api-key-12345",
		AzureOpenAIEndpoint:   "https://myresource.openai.azure.com/",
		AzureOpenAIDeployment: "dalle3",
		AITimeout:             30 * time.Second,
	}

	provider, err := NewAzureProvider(cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
	if provider.Deployment() != "dalle3" {
		t.Errorf("expected deployment dalle3, got %s", provider.Deployment())
	}
}

// TestNewAzureProvider_ImageLLMURLOverride tests that ImageLLMURL takes precedence.
func TestNewAzureProvider_ImageLLMURLOverride(t *testing.T) {
	cfg := &core.Config{
		OpenAIAPIKey:          "test-api-key",
		ImageLLMURL:           "https://override.openai.azure.com/",
		AzureOpenAIEndpoint:   "https://fallback.openai.azure.com/",
		AzureOpenAIDeployment: "dalle3",
	}

	provider, err := NewAzureProvider(cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
	// Provider was created successfully - ImageLLMURL was used
}

// TestNewAzureProvider_CognitiveServicesEndpoint tests cognitiveservices.azure.com endpoint.
func TestNewAzureProvider_CognitiveServicesEndpoint(t *testing.T) {
	cfg := &core.Config{
		OpenAIAPIKey:          "test-api-key",
		AzureOpenAIEndpoint:   "https://myresource.cognitiveservices.azure.com/",
		AzureOpenAIDeployment: "dalle3",
	}

	provider, err := NewAzureProvider(cfg)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider for cognitiveservices endpoint")
	}
}

// TestNewAzureProviderWithConfig_EmptyAPIKey tests explicit config with empty API key.
func TestNewAzureProviderWithConfig_EmptyAPIKey(t *testing.T) {
	cfg := AzureProviderConfig{
		APIKey:     "",
		Endpoint:   "https://myresource.openai.azure.com/",
		Deployment: "dalle3",
	}

	provider, err := NewAzureProviderWithConfig(cfg, nil)

	if err == nil {
		t.Error("expected error for empty API key, got nil")
	}
	if provider != nil {
		t.Error("expected nil provider for empty API key")
	}
}

// TestNewAzureProviderWithConfig_EmptyEndpoint tests explicit config with empty endpoint.
func TestNewAzureProviderWithConfig_EmptyEndpoint(t *testing.T) {
	cfg := AzureProviderConfig{
		APIKey:     "test-key",
		Endpoint:   "",
		Deployment: "dalle3",
	}

	provider, err := NewAzureProviderWithConfig(cfg, nil)

	if err == nil {
		t.Error("expected error for empty endpoint, got nil")
	}
	if provider != nil {
		t.Error("expected nil provider for empty endpoint")
	}
}

// TestNewAzureProviderWithConfig_NonAzureEndpoint tests explicit config with non-Azure endpoint.
func TestNewAzureProviderWithConfig_NonAzureEndpoint(t *testing.T) {
	cfg := AzureProviderConfig{
		APIKey:     "test-key",
		Endpoint:   "https://api.openai.com/v1",
		Deployment: "dalle3",
	}

	provider, err := NewAzureProviderWithConfig(cfg, nil)

	if err == nil {
		t.Error("expected error for non-Azure endpoint, got nil")
	}
	if provider != nil {
		t.Error("expected nil provider for non-Azure endpoint")
	}
}

// TestNewAzureProviderWithConfig_EmptyDeployment tests explicit config with empty deployment.
func TestNewAzureProviderWithConfig_EmptyDeployment(t *testing.T) {
	cfg := AzureProviderConfig{
		APIKey:     "test-key",
		Endpoint:   "https://myresource.openai.azure.com/",
		Deployment: "",
	}

	provider, err := NewAzureProviderWithConfig(cfg, nil)

	if err == nil {
		t.Error("expected error for empty deployment, got nil")
	}
	if provider != nil {
		t.Error("expected nil provider for empty deployment")
	}
}

// TestNewAzureProviderWithConfig_ValidConfig tests explicit config creation.
func TestNewAzureProviderWithConfig_ValidConfig(t *testing.T) {
	cfg := AzureProviderConfig{
		APIKey:     "test-api-key",
		Endpoint:   "https://myresource.openai.azure.com/",
		Deployment: "gpt-image-1",
	}

	provider, err := NewAzureProviderWithConfig(cfg, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider == nil {
		t.Fatal("expected non-nil provider")
	}
	if provider.Deployment() != "gpt-image-1" {
		t.Errorf("expected deployment gpt-image-1, got %s", provider.Deployment())
	}
}

// TestAzureProvider_Generate_EmptyPrompt tests that empty prompt returns error.
func TestAzureProvider_Generate_EmptyPrompt(t *testing.T) {
	cfg := &core.Config{
		OpenAIAPIKey:          "test-api-key",
		AzureOpenAIEndpoint:   "https://myresource.openai.azure.com/",
		AzureOpenAIDeployment: "dalle3",
	}

	provider, err := NewAzureProvider(cfg)
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

// TestAzureProvider_Deployment tests the Deployment() method.
func TestAzureProvider_Deployment(t *testing.T) {
	tests := []struct {
		name       string
		deployment string
	}{
		{"dalle3", "dalle3"},
		{"dall-e-3", "dall-e-3"},
		{"gpt-image-1", "gpt-image-1"},
		{"custom-deployment", "custom-deployment"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &core.Config{
				OpenAIAPIKey:          "test-api-key",
				AzureOpenAIEndpoint:   "https://myresource.openai.azure.com/",
				AzureOpenAIDeployment: tt.deployment,
			}

			provider, err := NewAzureProvider(cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if provider.Deployment() != tt.deployment {
				t.Errorf("expected deployment %s, got %s", tt.deployment, provider.Deployment())
			}
		})
	}
}

// TestIsDalleDeployment tests the isDalleDeployment helper function.
func TestIsDalleDeployment(t *testing.T) {
	tests := []struct {
		deployment string
		expected   bool
	}{
		{"dalle3", true},
		{"DALLE3", true},
		{"dall-e-3", true},
		{"DALL-E-3", true},
		{"dalle-3", true},
		{"my-dalle3-deployment", true},
		{"gpt-image-1", false},
		{"GPT-IMAGE-1", false},
		{"custom-image-model", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.deployment, func(t *testing.T) {
			result := isDalleDeployment(tt.deployment)
			if result != tt.expected {
				t.Errorf("isDalleDeployment(%q) = %v, expected %v", tt.deployment, result, tt.expected)
			}
		})
	}
}

// Note: Integration tests that actually call the Azure OpenAI API should be in a
// separate _integration_test.go file and only run with specific build tags
// or environment variables to avoid accidental API charges.
