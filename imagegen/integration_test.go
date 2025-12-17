//go:build integration
// +build integration

// Package imagegen integration tests.
//
// These tests call real cloud APIs (OpenAI, Azure) and are NOT run by default.
// They require:
//   - Build tag: -tags=integration
//   - Environment variables for API keys
//
// Run with:
//
//	go test ./imagegen/... -tags=integration -v
//
// Required environment variables for OpenAI tests:
//   - OPENAI_API_KEY: OpenAI API key
//
// Required environment variables for Azure tests:
//   - AZURE_OPENAI_API_KEY: Azure OpenAI API key (or use OPENAI_API_KEY)
//   - AZURE_OPENAI_ENDPOINT: Azure OpenAI endpoint URL
//   - AZURE_OPENAI_DEPLOYMENT: Azure deployment name (e.g., "dalle3")
//
// Note: These tests incur API costs. Run sparingly.
package imagegen

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"go_backend/core"
)

// Integration test configuration
const (
	integrationTestTimeout = 60 * time.Second
)

// skipIfNoOpenAIKey skips the test if OPENAI_API_KEY is not set.
func skipIfNoOpenAIKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("Skipping: OPENAI_API_KEY not set")
	}
	return key
}

// skipIfNoAzureConfig skips the test if Azure environment variables are not set.
func skipIfNoAzureConfig(t *testing.T) (apiKey, endpoint, deployment string) {
	t.Helper()

	apiKey = os.Getenv("AZURE_OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		t.Skip("Skipping: AZURE_OPENAI_API_KEY or OPENAI_API_KEY not set")
	}

	endpoint = os.Getenv("AZURE_OPENAI_ENDPOINT")
	if endpoint == "" {
		t.Skip("Skipping: AZURE_OPENAI_ENDPOINT not set")
	}

	deployment = os.Getenv("AZURE_OPENAI_DEPLOYMENT")
	if deployment == "" {
		t.Skip("Skipping: AZURE_OPENAI_DEPLOYMENT not set")
	}

	return apiKey, endpoint, deployment
}

// createTestConfig creates a core.Config for integration tests.
func createTestConfig(t *testing.T, apiKey string) *core.Config {
	t.Helper()
	return &core.Config{
		OpenAIAPIKey:         apiKey,
		ImageLLMURL:          "https://api.openai.com/v1",
		OpenAIImageModel:     "dall-e-3",
		AITimeout:            integrationTestTimeout,
		AllowSelfSignedCerts: false,
		DownloadsDir:         t.TempDir(),
	}
}

// createAzureTestConfig creates a core.Config for Azure integration tests.
func createAzureTestConfig(t *testing.T, apiKey, endpoint, deployment string) *core.Config {
	t.Helper()
	return &core.Config{
		OpenAIAPIKey:          apiKey,
		AzureOpenAIEndpoint:   endpoint,
		AzureOpenAIDeployment: deployment,
		AITimeout:             integrationTestTimeout,
		AllowSelfSignedCerts:  false,
		DownloadsDir:          t.TempDir(),
	}
}

// ============================================================================
// OpenAI Provider Integration Tests
// ============================================================================

// TestIntegration_OpenAIProvider_Generate tests real OpenAI DALL-E image generation.
//
// This test:
//  1. Creates an OpenAI provider with real API key
//  2. Generates an image with a simple prompt
//  3. Verifies the returned URL is valid
//  4. Does NOT download the image (to minimize costs)
func TestIntegration_OpenAIProvider_Generate(t *testing.T) {
	apiKey := skipIfNoOpenAIKey(t)

	cfg := createTestConfig(t, apiKey)

	provider, err := NewOpenAIProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), integrationTestTimeout)
	defer cancel()

	// Simple prompt that should work reliably
	prompt := "A simple blue square on white background"

	t.Logf("Generating image with prompt: %q", prompt)
	startTime := time.Now()

	url, err := provider.Generate(ctx, prompt)

	elapsed := time.Since(startTime)
	t.Logf("Generation completed in %v", elapsed)

	if err != nil {
		// Check for common error types
		errStr := err.Error()
		if strings.Contains(errStr, "rate limit") {
			t.Skip("Skipping: Rate limited by OpenAI API")
		}
		if strings.Contains(errStr, "insufficient_quota") {
			t.Skip("Skipping: Insufficient OpenAI quota")
		}
		if strings.Contains(errStr, "content_policy") {
			t.Logf("Warning: Content policy violation (prompt may be too simple)")
		}
		t.Fatalf("Generate() failed: %v", err)
	}

	// Validate URL format
	if url == "" {
		t.Fatal("Generate() returned empty URL")
	}
	if !strings.HasPrefix(url, "https://") {
		t.Errorf("Generate() URL should start with https://, got: %s", url)
	}
	if !strings.Contains(url, "oai") && !strings.Contains(url, "openai") && !strings.Contains(url, "dall-e") {
		t.Logf("Note: URL doesn't contain expected OpenAI identifiers: %s", url)
	}

	t.Logf("Generated image URL: %s", url)
}

// TestIntegration_OpenAIProvider_Generate_DallE2 tests DALL-E 2 generation.
func TestIntegration_OpenAIProvider_Generate_DallE2(t *testing.T) {
	apiKey := skipIfNoOpenAIKey(t)

	cfg := &core.Config{
		OpenAIAPIKey:     apiKey,
		ImageLLMURL:      "https://api.openai.com/v1",
		OpenAIImageModel: "dall-e-2", // Older model, potentially faster/cheaper
		AITimeout:        integrationTestTimeout,
	}

	provider, err := NewOpenAIProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	if provider.Model() != "dall-e-2" {
		t.Errorf("Expected model dall-e-2, got %s", provider.Model())
	}

	ctx, cancel := context.WithTimeout(context.Background(), integrationTestTimeout)
	defer cancel()

	prompt := "A red circle"

	url, err := provider.Generate(ctx, prompt)

	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "insufficient_quota") {
			t.Skip("Skipping: API limit reached")
		}
		t.Fatalf("Generate() failed: %v", err)
	}

	if url == "" {
		t.Fatal("Generate() returned empty URL")
	}

	t.Logf("DALL-E 2 generated image URL: %s", url)
}

// TestIntegration_OpenAIProvider_ContextCancellation tests timeout handling.
func TestIntegration_OpenAIProvider_ContextCancellation(t *testing.T) {
	apiKey := skipIfNoOpenAIKey(t)

	cfg := createTestConfig(t, apiKey)

	provider, err := NewOpenAIProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	// Create a context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before calling Generate

	_, err = provider.Generate(ctx, "A test image")

	if err == nil {
		t.Error("Expected error for cancelled context, got nil")
	}
	if !strings.Contains(err.Error(), "cancel") && !strings.Contains(err.Error(), "context") {
		t.Logf("Note: Error may not explicitly mention cancellation: %v", err)
	}
}

// ============================================================================
// Azure Provider Integration Tests
// ============================================================================

// TestIntegration_AzureProvider_Generate tests real Azure OpenAI image generation.
func TestIntegration_AzureProvider_Generate(t *testing.T) {
	apiKey, endpoint, deployment := skipIfNoAzureConfig(t)

	cfg := createAzureTestConfig(t, apiKey, endpoint, deployment)

	provider, err := NewAzureProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create Azure provider: %v", err)
	}

	if provider.Deployment() != deployment {
		t.Errorf("Expected deployment %s, got %s", deployment, provider.Deployment())
	}

	ctx, cancel := context.WithTimeout(context.Background(), integrationTestTimeout)
	defer cancel()

	prompt := "A simple green triangle on white background"

	t.Logf("Generating Azure image with prompt: %q", prompt)
	startTime := time.Now()

	url, err := provider.Generate(ctx, prompt)

	elapsed := time.Since(startTime)
	t.Logf("Azure generation completed in %v", elapsed)

	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "429") {
			t.Skip("Skipping: Rate limited by Azure API")
		}
		if strings.Contains(errStr, "quota") || strings.Contains(errStr, "insufficient") {
			t.Skip("Skipping: Insufficient Azure quota")
		}
		t.Fatalf("Generate() failed: %v", err)
	}

	if url == "" {
		t.Fatal("Generate() returned empty URL")
	}
	if !strings.HasPrefix(url, "https://") {
		t.Errorf("Generate() URL should start with https://, got: %s", url)
	}

	t.Logf("Azure generated image URL: %s", url)
}

// TestIntegration_AzureProviderWithConfig tests explicit config creation.
func TestIntegration_AzureProviderWithConfig(t *testing.T) {
	apiKey, endpoint, deployment := skipIfNoAzureConfig(t)

	providerCfg := AzureProviderConfig{
		APIKey:     apiKey,
		Endpoint:   endpoint,
		Deployment: deployment,
	}

	provider, err := NewAzureProviderWithConfig(providerCfg, nil)
	if err != nil {
		t.Fatalf("Failed to create Azure provider: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), integrationTestTimeout)
	defer cancel()

	url, err := provider.Generate(ctx, "A yellow star")

	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "quota") {
			t.Skip("Skipping: API limit reached")
		}
		t.Fatalf("Generate() failed: %v", err)
	}

	if url == "" {
		t.Fatal("Generate() returned empty URL")
	}

	t.Logf("Azure explicit config generated URL: %s", url)
}

// ============================================================================
// Downloader Integration Tests
// ============================================================================

// TestIntegration_Downloader_Download tests downloading a real image from URL.
func TestIntegration_Downloader_Download(t *testing.T) {
	// Use a known stable test image URL
	testImageURL := "https://www.google.com/images/branding/googlelogo/2x/googlelogo_color_272x92dp.png"

	cfg := &core.Config{
		AllowSelfSignedCerts: false,
		DownloadsDir:         t.TempDir(),
	}

	downloader, err := NewDownloader(cfg)
	if err != nil {
		t.Fatalf("NewDownloader() failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := downloader.Download(ctx, testImageURL, "test-google-logo")

	if err != nil {
		t.Fatalf("Download() failed: %v", err)
	}

	if result == nil {
		t.Fatal("Download() returned nil result")
	}
	if result.Path == "" {
		t.Error("Download() result Path is empty")
	}
	if result.Size == 0 {
		t.Error("Download() result Size is 0")
	}

	// Verify file exists
	if _, err := os.Stat(result.Path); os.IsNotExist(err) {
		t.Errorf("Downloaded file does not exist: %s", result.Path)
	}

	t.Logf("Downloaded %d bytes to %s (content-type: %s)", result.Size, result.Path, result.ContentType)
}

// TestIntegration_Downloader_DownloadInvalidURL tests handling of invalid URLs.
func TestIntegration_Downloader_DownloadInvalidURL(t *testing.T) {
	cfg := &core.Config{
		AllowSelfSignedCerts: false,
		DownloadsDir:         t.TempDir(),
	}

	downloader, err := NewDownloader(cfg)
	if err != nil {
		t.Fatalf("NewDownloader() failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = downloader.Download(ctx, "https://invalid-domain-that-does-not-exist-12345.com/image.png", "test")

	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

// TestIntegration_Downloader_Download404 tests handling of 404 response.
func TestIntegration_Downloader_Download404(t *testing.T) {
	cfg := &core.Config{
		AllowSelfSignedCerts: false,
		DownloadsDir:         t.TempDir(),
	}

	downloader, err := NewDownloader(cfg)
	if err != nil {
		t.Fatalf("NewDownloader() failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Use a valid domain but non-existent path
	_, err = downloader.Download(ctx, "https://www.google.com/this-path-does-not-exist-12345.png", "test")

	if err == nil {
		t.Error("Expected error for 404 response, got nil")
	}
	if !strings.Contains(err.Error(), "404") && !strings.Contains(err.Error(), "status") {
		t.Logf("Note: Error may not mention 404: %v", err)
	}
}

// ============================================================================
// End-to-End Integration Tests
// ============================================================================

// TestIntegration_EndToEnd_GenerateAndDownload tests complete flow: generate and download.
//
// This is an expensive test - it generates an image via OpenAI and downloads it.
func TestIntegration_EndToEnd_GenerateAndDownload(t *testing.T) {
	apiKey := skipIfNoOpenAIKey(t)

	cfg := createTestConfig(t, apiKey)

	// Create provider
	provider, err := NewOpenAIProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}

	// Create downloader
	downloader, err := NewDownloader(cfg)
	if err != nil {
		t.Fatalf("Failed to create downloader: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Step 1: Generate
	prompt := "A simple purple pentagon on white background"
	t.Logf("Generating image: %q", prompt)

	imageURL, err := provider.Generate(ctx, prompt)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "quota") {
			t.Skip("Skipping: API limit reached")
		}
		t.Fatalf("Generate() failed: %v", err)
	}

	t.Logf("Generated URL: %s", imageURL)

	// Step 2: Download
	t.Log("Downloading generated image...")

	result, err := downloader.Download(ctx, imageURL, "end-to-end-test")
	if err != nil {
		t.Fatalf("Download() failed: %v", err)
	}

	t.Logf("Downloaded to: %s (%d bytes)", result.Path, result.Size)

	// Verify file exists and has content
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if len(data) < 1000 {
		t.Errorf("Downloaded file suspiciously small: %d bytes", len(data))
	}

	// Check for common image signatures
	isJPEG := len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8
	isPNG := len(data) >= 8 && data[0] == 0x89 && data[1] == 0x50
	isWebP := len(data) >= 12 && string(data[8:12]) == "WEBP"

	if isJPEG {
		t.Log("Downloaded image is JPEG format")
	} else if isPNG {
		t.Log("Downloaded image is PNG format")
	} else if isWebP {
		t.Log("Downloaded image is WebP format")
	} else {
		t.Logf("Unknown image format (first 16 bytes: %x)", data[:min(16, len(data))])
	}

	t.Log("End-to-end test completed successfully!")
}

// TestIntegration_DownloaderWithConfig tests NewDownloaderWithConfig.
func TestIntegration_DownloaderWithConfig(t *testing.T) {
	downloaderCfg := DownloaderConfig{
		HTTPClient:   &http.Client{Timeout: 30 * time.Second},
		DownloadsDir: t.TempDir(),
		Timeout:      30 * time.Second,
	}

	downloader, err := NewDownloaderWithConfig(downloaderCfg, nil)
	if err != nil {
		t.Fatalf("NewDownloaderWithConfig() failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testImageURL := "https://www.google.com/images/branding/googlelogo/2x/googlelogo_color_272x92dp.png"

	result, err := downloader.Download(ctx, testImageURL, "config-test")
	if err != nil {
		t.Fatalf("Download() failed: %v", err)
	}

	if result.Size == 0 {
		t.Error("Download() result Size is 0")
	}

	t.Logf("Downloaded %d bytes with custom config", result.Size)
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
