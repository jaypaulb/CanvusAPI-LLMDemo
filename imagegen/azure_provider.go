// Package imagegen provides image generation utilities for the Canvus canvas.
//
// azure_provider.go implements the AzureProvider molecule that generates
// images using Azure OpenAI DALL-E deployments.
//
// This molecule composes:
//   - atoms.go: IsAzureEndpoint for endpoint validation
//   - core.Config: for API configuration
//   - go-openai client: for API calls
package imagegen

import (
	"context"
	"fmt"
	"strings"

	"go_backend/core"

	"github.com/sashabaranov/go-openai"
)

// AzureProvider implements Provider for Azure OpenAI image generation.
//
// Azure OpenAI differs from standard OpenAI in several ways:
//   - Uses deployment names instead of model names
//   - Requires Azure-specific endpoint configuration
//   - May have different parameter support based on deployment
//
// Thread Safety: AzureProvider is safe for concurrent use.
// The underlying OpenAI client handles connection pooling.
type AzureProvider struct {
	client     *openai.Client
	config     *core.Config
	deployment string
}

// AzureProviderConfig holds configuration specific to the Azure provider.
type AzureProviderConfig struct {
	// APIKey is the Azure OpenAI API key (required)
	APIKey string

	// Endpoint is the Azure OpenAI endpoint URL (required)
	// Example: https://your-resource.openai.azure.com/
	Endpoint string

	// Deployment is the Azure deployment name (required)
	// Example: dalle3, gpt-image-1
	Deployment string

	// APIVersion is the Azure API version (optional)
	// Default: 2024-02-15-preview
	APIVersion string
}

// NewAzureProvider creates a new Azure OpenAI image generation provider.
//
// Parameters:
//   - cfg: core.Config with Azure endpoint and deployment configuration
//
// Returns an error if:
//   - The config is nil
//   - The API key is empty
//   - The endpoint is empty or not an Azure endpoint
//   - The deployment name is empty
//
// Example:
//
//	provider, err := NewAzureProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	url, err := provider.Generate(ctx, "a sunset over mountains")
func NewAzureProvider(cfg *core.Config) (*AzureProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("imagegen: config cannot be nil")
	}
	if cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("imagegen: API key is required for Azure image generation")
	}

	// Determine endpoint - use ImageLLMURL or fall back to AzureOpenAIEndpoint
	endpoint := cfg.ImageLLMURL
	if endpoint == "" {
		endpoint = cfg.AzureOpenAIEndpoint
	}
	if endpoint == "" {
		return nil, fmt.Errorf("imagegen: Azure endpoint is required; set IMAGE_LLM_URL or AZURE_OPENAI_ENDPOINT")
	}

	// Validate it's an Azure endpoint
	if !IsAzureEndpoint(endpoint) {
		return nil, fmt.Errorf("imagegen: endpoint (%s) is not an Azure OpenAI endpoint", endpoint)
	}

	// Validate deployment name
	deployment := cfg.AzureOpenAIDeployment
	if deployment == "" {
		return nil, fmt.Errorf("imagegen: Azure deployment name is required; set AZURE_OPENAI_DEPLOYMENT")
	}

	// Configure OpenAI client for Azure
	clientConfig := openai.DefaultConfig(cfg.OpenAIAPIKey)
	clientConfig.BaseURL = endpoint
	clientConfig.HTTPClient = core.GetHTTPClient(cfg, cfg.AITimeout)

	return &AzureProvider{
		client:     openai.NewClientWithConfig(clientConfig),
		config:     cfg,
		deployment: deployment,
	}, nil
}

// NewAzureProviderWithConfig creates an Azure provider with explicit configuration.
// This is useful for testing or when you need fine-grained control over settings.
//
// Parameters:
//   - providerCfg: AzureProviderConfig with provider-specific settings
//   - coreCfg: core.Config for HTTP client settings (can be nil for defaults)
//
// Returns an error if:
//   - The API key is empty
//   - The endpoint is empty or not an Azure endpoint
//   - The deployment name is empty
func NewAzureProviderWithConfig(providerCfg AzureProviderConfig, coreCfg *core.Config) (*AzureProvider, error) {
	if providerCfg.APIKey == "" {
		return nil, fmt.Errorf("imagegen: Azure API key is required")
	}
	if providerCfg.Endpoint == "" {
		return nil, fmt.Errorf("imagegen: Azure endpoint is required")
	}
	if !IsAzureEndpoint(providerCfg.Endpoint) {
		return nil, fmt.Errorf("imagegen: endpoint (%s) is not an Azure OpenAI endpoint", providerCfg.Endpoint)
	}
	if providerCfg.Deployment == "" {
		return nil, fmt.Errorf("imagegen: Azure deployment name is required")
	}

	clientConfig := openai.DefaultConfig(providerCfg.APIKey)
	clientConfig.BaseURL = providerCfg.Endpoint

	// Use provided HTTP client or create default
	if coreCfg != nil {
		clientConfig.HTTPClient = core.GetHTTPClient(coreCfg, coreCfg.AITimeout)
	}

	return &AzureProvider{
		client:     openai.NewClientWithConfig(clientConfig),
		config:     coreCfg,
		deployment: providerCfg.Deployment,
	}, nil
}

// Generate creates an image from the given prompt using Azure OpenAI's DALL-E API.
//
// The method:
//  1. Creates an image request using the deployment name as the model
//  2. Adds style parameter only for DALL-E deployments (not gpt-image-1)
//  3. Calls the Azure OpenAI API
//  4. Validates the response
//  5. Returns the URL of the generated image
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - prompt: the text description of the image to generate
//
// Returns:
//   - string: URL of the generated image (temporary, hosted by Azure)
//   - error: if generation fails or response is invalid
//
// Note: The returned URL is temporary and should be downloaded promptly.
func (p *AzureProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("imagegen: prompt cannot be empty")
	}

	// Build image request - Azure uses deployment name as model
	req := openai.ImageRequest{
		Prompt:         prompt,
		Model:          p.deployment,
		ResponseFormat: openai.CreateImageResponseFormatURL,
		N:              1,
	}

	// Add style parameter only for DALL-E deployments
	// Azure's gpt-image-1 deployment doesn't support style parameter
	if isDalleDeployment(p.deployment) {
		req.Style = openai.CreateImageStyleVivid
	}

	// Call Azure OpenAI API
	response, err := p.client.CreateImage(ctx, req)
	if err != nil {
		return "", fmt.Errorf("imagegen: Azure image generation failed: %w", err)
	}

	// Validate response
	if response.Data == nil {
		return "", fmt.Errorf("imagegen: Azure returned nil Data field")
	}
	if len(response.Data) == 0 {
		return "", fmt.Errorf("imagegen: Azure returned empty Data array")
	}
	if response.Data[0].URL == "" {
		return "", fmt.Errorf("imagegen: Azure returned empty image URL")
	}

	return response.Data[0].URL, nil
}

// Deployment returns the configured Azure deployment name.
func (p *AzureProvider) Deployment() string {
	return p.deployment
}

// isDalleDeployment checks if the deployment name indicates a DALL-E model.
// This is used to determine whether to add style parameters.
//
// Returns true for deployments containing:
//   - "dalle3" or "dall-e" (Azure DALL-E 3 deployments)
//
// Returns false for:
//   - "gpt-image-1" or other non-DALL-E deployments
func isDalleDeployment(deployment string) bool {
	lower := strings.ToLower(deployment)
	return strings.Contains(lower, "dalle3") ||
		strings.Contains(lower, "dall-e") ||
		strings.Contains(lower, "dalle-3")
}

// Ensure AzureProvider implements Provider interface at compile time.
var _ Provider = (*AzureProvider)(nil)
