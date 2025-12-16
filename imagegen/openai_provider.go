// Package imagegen provides image generation utilities for the Canvus canvas.
//
// openai_provider.go implements the OpenAIProvider molecule that generates
// images using the OpenAI DALL-E API.
//
// This molecule composes:
//   - atoms.go: IsLocalEndpoint for endpoint validation
//   - core.Config: for API configuration
//   - go-openai client: for API calls
package imagegen

import (
	"context"
	"fmt"

	"go_backend/core"

	"github.com/sashabaranov/go-openai"
)

// Provider is the interface for image generation providers.
// Each provider (OpenAI, Azure, local SD) implements this interface
// to allow swappable image generation backends.
//
// The Generate method takes a prompt and returns the URL of the generated image.
// Downloading and uploading the image to Canvus is handled separately by the
// Generator organism.
type Provider interface {
	// Generate creates an image from the given prompt.
	// Returns the URL of the generated image, or an error.
	//
	// The context can be used for cancellation and timeout control.
	Generate(ctx context.Context, prompt string) (string, error)
}

// OpenAIProvider implements Provider for OpenAI DALL-E image generation.
//
// This molecule handles:
//   - OpenAI client configuration with proper HTTP transport
//   - Model selection (DALL-E 2 vs DALL-E 3)
//   - Style parameters for DALL-E 3
//   - Error handling and response validation
//
// Thread Safety: OpenAIProvider is safe for concurrent use.
// The underlying OpenAI client handles connection pooling.
type OpenAIProvider struct {
	client *openai.Client
	config *core.Config
	model  string
}

// OpenAIProviderConfig holds configuration specific to the OpenAI provider.
type OpenAIProviderConfig struct {
	// APIKey is the OpenAI API key (required)
	APIKey string

	// BaseURL is the API endpoint (default: https://api.openai.com/v1)
	BaseURL string

	// Model is the image model to use (default: dall-e-3)
	Model string

	// HTTPClient is the HTTP client for API calls (optional)
	// If nil, a default client will be created
	HTTPClient interface{}
}

// DefaultOpenAIProviderConfig returns sensible defaults for OpenAI image generation.
func DefaultOpenAIProviderConfig() OpenAIProviderConfig {
	return OpenAIProviderConfig{
		BaseURL: "https://api.openai.com/v1",
		Model:   "dall-e-3",
	}
}

// NewOpenAIProvider creates a new OpenAI image generation provider.
//
// Parameters:
//   - cfg: core.Config with API key and endpoint configuration
//
// Returns an error if:
//   - The API key is empty
//   - The endpoint is a local endpoint (localhost, 127.0.0.1)
//     which doesn't support image generation
//
// Example:
//
//	provider, err := NewOpenAIProvider(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	url, err := provider.Generate(ctx, "a sunset over mountains")
func NewOpenAIProvider(cfg *core.Config) (*OpenAIProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("imagegen: config cannot be nil")
	}
	if cfg.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("imagegen: OpenAI API key is required for image generation")
	}

	// Determine the endpoint to use
	endpoint := cfg.ImageLLMURL
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}

	// Validate endpoint - local endpoints don't support image generation
	if IsLocalEndpoint(endpoint) {
		return nil, fmt.Errorf("imagegen: local endpoint (%s) does not support image generation; "+
			"configure IMAGE_LLM_URL to use OpenAI or Azure", endpoint)
	}

	// Configure OpenAI client
	clientConfig := openai.DefaultConfig(cfg.OpenAIAPIKey)
	clientConfig.BaseURL = endpoint
	clientConfig.HTTPClient = core.GetHTTPClient(cfg, cfg.AITimeout)

	// Determine model
	model := cfg.OpenAIImageModel
	if model == "" {
		model = "dall-e-3"
	}

	return &OpenAIProvider{
		client: openai.NewClientWithConfig(clientConfig),
		config: cfg,
		model:  model,
	}, nil
}

// NewOpenAIProviderWithConfig creates an OpenAI provider with explicit configuration.
// This is useful for testing or when you need fine-grained control over settings.
//
// Parameters:
//   - providerCfg: OpenAIProviderConfig with provider-specific settings
//   - coreCfg: core.Config for HTTP client settings (can be nil for defaults)
//
// Returns an error if:
//   - The API key is empty
//   - The endpoint is a local endpoint
func NewOpenAIProviderWithConfig(providerCfg OpenAIProviderConfig, coreCfg *core.Config) (*OpenAIProvider, error) {
	if providerCfg.APIKey == "" {
		return nil, fmt.Errorf("imagegen: OpenAI API key is required")
	}

	endpoint := providerCfg.BaseURL
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1"
	}

	if IsLocalEndpoint(endpoint) {
		return nil, fmt.Errorf("imagegen: local endpoint (%s) does not support image generation", endpoint)
	}

	clientConfig := openai.DefaultConfig(providerCfg.APIKey)
	clientConfig.BaseURL = endpoint

	// Use provided HTTP client or create default
	if coreCfg != nil {
		clientConfig.HTTPClient = core.GetHTTPClient(coreCfg, coreCfg.AITimeout)
	}

	model := providerCfg.Model
	if model == "" {
		model = "dall-e-3"
	}

	return &OpenAIProvider{
		client: openai.NewClientWithConfig(clientConfig),
		config: coreCfg,
		model:  model,
	}, nil
}

// Generate creates an image from the given prompt using OpenAI's DALL-E API.
//
// The method:
//  1. Creates an image request with the configured model
//  2. Adds style parameter for DALL-E 3 (vivid by default)
//  3. Calls the OpenAI API
//  4. Validates the response
//  5. Returns the URL of the generated image
//
// Parameters:
//   - ctx: context for cancellation and timeout
//   - prompt: the text description of the image to generate
//
// Returns:
//   - string: URL of the generated image (temporary, hosted by OpenAI)
//   - error: if generation fails or response is invalid
//
// Note: The returned URL is temporary and should be downloaded promptly.
// URLs typically expire after about 1 hour.
func (p *OpenAIProvider) Generate(ctx context.Context, prompt string) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("imagegen: prompt cannot be empty")
	}

	// Build image request
	req := openai.ImageRequest{
		Prompt:         prompt,
		Model:          p.model,
		ResponseFormat: openai.CreateImageResponseFormatURL,
		N:              1,
	}

	// Add style parameter for DALL-E 3 (not supported by DALL-E 2)
	if p.model == "dall-e-3" {
		req.Style = openai.CreateImageStyleVivid
	}

	// Call OpenAI API
	response, err := p.client.CreateImage(ctx, req)
	if err != nil {
		return "", fmt.Errorf("imagegen: OpenAI image generation failed: %w", err)
	}

	// Validate response
	if response.Data == nil {
		return "", fmt.Errorf("imagegen: OpenAI returned nil Data field")
	}
	if len(response.Data) == 0 {
		return "", fmt.Errorf("imagegen: OpenAI returned empty Data array")
	}
	if response.Data[0].URL == "" {
		return "", fmt.Errorf("imagegen: OpenAI returned empty image URL")
	}

	return response.Data[0].URL, nil
}

// Model returns the configured image model name.
func (p *OpenAIProvider) Model() string {
	return p.model
}

// Ensure OpenAIProvider implements Provider interface at compile time.
var _ Provider = (*OpenAIProvider)(nil)
