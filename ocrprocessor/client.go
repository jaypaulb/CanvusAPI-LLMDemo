// Package ocrprocessor provides OCR (Optical Character Recognition) functionality
// for CanvusLocalLLM using Google Cloud Vision API.
//
// client.go implements the VisionClient molecule that wraps Google Vision API
// for OCR operations. It composes:
//   - atoms.go: API key validation functions
//   - core.GetHTTPClient: HTTP client factory
//   - logging.Logger: structured logging
package ocrprocessor

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go_backend/logging"

	"go.uber.org/zap"
)

// VisionClient wraps Google Cloud Vision API for OCR operations.
//
// Thread-Safety:
//   - VisionClient is safe for concurrent use
//   - HTTP client handles concurrency internally
type VisionClient struct {
	apiKey     string
	httpClient *http.Client
	logger     *logging.Logger
	config     VisionClientConfig
}

// VisionClientConfig holds configuration for the Vision API client.
type VisionClientConfig struct {
	// Endpoint is the Google Vision API endpoint
	Endpoint string

	// FeatureType is the Vision API feature to use
	// Common values: "TEXT_DETECTION", "DOCUMENT_TEXT_DETECTION"
	FeatureType string

	// Timeout for API requests
	Timeout time.Duration

	// MaxResults limits the number of results returned
	MaxResults int
}

// DefaultVisionClientConfig returns sensible default configuration.
func DefaultVisionClientConfig() VisionClientConfig {
	return VisionClientConfig{
		Endpoint:    "https://vision.googleapis.com/v1/images:annotate",
		FeatureType: "DOCUMENT_TEXT_DETECTION",
		Timeout:     30 * time.Second,
		MaxResults:  1,
	}
}

// Common errors for OCR operations.
var (
	// ErrNoTextFound indicates the image contains no recognizable text.
	ErrNoTextFound = errors.New("ocrprocessor: no text found in image")

	// ErrEmptyResponse indicates the API returned an empty response.
	ErrEmptyResponse = errors.New("ocrprocessor: empty response from Vision API")

	// ErrNilClient indicates the HTTP client is nil.
	ErrNilClient = errors.New("ocrprocessor: HTTP client cannot be nil")

	// ErrNilLogger indicates the logger is nil.
	ErrNilLogger = errors.New("ocrprocessor: logger cannot be nil")
)

// visionRequest is the Google Vision API request structure.
type visionRequest struct {
	Requests []visionRequestItem `json:"requests"`
}

// visionRequestItem represents a single request in the batch.
type visionRequestItem struct {
	Image    visionImage    `json:"image"`
	Features []visionFeature `json:"features"`
}

// visionImage holds the image data.
type visionImage struct {
	Content string `json:"content"`
}

// visionFeature specifies the detection feature.
type visionFeature struct {
	Type       string `json:"type"`
	MaxResults int    `json:"maxResults"`
}

// visionResponse is the Google Vision API response structure.
type visionResponse struct {
	Responses []visionResponseItem `json:"responses"`
}

// visionResponseItem represents a single response in the batch.
type visionResponseItem struct {
	FullTextAnnotation struct {
		Text string `json:"text"`
	} `json:"fullTextAnnotation"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewVisionClient creates a new Vision API client.
//
// Parameters:
//   - apiKey: Google Cloud API key with Vision API access
//   - httpClient: HTTP client for API requests (use core.GetHTTPClient)
//   - logger: structured logger for operation tracking
//   - config: client configuration
//
// Returns an error if the API key is invalid or dependencies are nil.
func NewVisionClient(apiKey string, httpClient *http.Client, logger *logging.Logger, config VisionClientConfig) (*VisionClient, error) {
	if httpClient == nil {
		return nil, ErrNilClient
	}
	if logger == nil {
		return nil, ErrNilLogger
	}

	// Validate API key using atom function
	if err := ValidateGoogleAPIKey(apiKey); err != nil {
		return nil, fmt.Errorf("ocrprocessor: %w", err)
	}

	return &VisionClient{
		apiKey:     apiKey,
		httpClient: httpClient,
		logger:     logger.Named("vision-client"),
		config:     config,
	}, nil
}

// OCRResult contains the result of OCR processing.
type OCRResult struct {
	// Text is the extracted text from the image
	Text string

	// ProcessingTime is how long the API call took
	ProcessingTime time.Duration
}

// ExtractText performs OCR on the provided image data.
//
// The image data should be in a format supported by Google Vision API:
// JPEG, PNG, GIF, BMP, WEBP, RAW, ICO, PDF, or TIFF.
//
// Parameters:
//   - ctx: context for cancellation/timeout
//   - imageData: raw image bytes
//
// Returns the extracted text and processing metadata, or an error.
func (c *VisionClient) ExtractText(ctx context.Context, imageData []byte) (*OCRResult, error) {
	startTime := time.Now()

	log := c.logger.With(
		zap.Int("image_size_bytes", len(imageData)),
	)

	log.Info("starting OCR extraction")

	// Validate input
	if len(imageData) == 0 {
		return nil, fmt.Errorf("ocrprocessor: image data is empty")
	}

	// Build request
	reqBody := c.buildRequest(imageData)

	// Marshal to JSON
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("ocrprocessor: failed to marshal request: %w", err)
	}
	log.Debug("request JSON created", zap.Int("size_bytes", len(jsonData)))

	// Create HTTP request
	url := fmt.Sprintf("%s?key=%s", c.config.Endpoint, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ocrprocessor: failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	log.Debug("sending request to Vision API")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ocrprocessor: failed to send request: %w", err)
	}
	defer resp.Body.Close()

	log.Debug("received response from Vision API",
		zap.Int("status_code", resp.StatusCode),
		zap.String("status", resp.Status))

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ocrprocessor: failed to read response body: %w", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ocrprocessor: Vision API error: status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var visionResp visionResponse
	if err := json.Unmarshal(bodyBytes, &visionResp); err != nil {
		return nil, fmt.Errorf("ocrprocessor: failed to decode response: %w", err)
	}

	// Extract text from response
	text, err := c.extractTextFromResponse(&visionResp)
	if err != nil {
		return nil, err
	}

	processingTime := time.Since(startTime)
	log.Info("OCR extraction completed",
		zap.Int("text_length", len(text)),
		zap.Duration("processing_time", processingTime))

	return &OCRResult{
		Text:           text,
		ProcessingTime: processingTime,
	}, nil
}

// buildRequest creates the Vision API request structure.
func (c *VisionClient) buildRequest(imageData []byte) *visionRequest {
	return &visionRequest{
		Requests: []visionRequestItem{
			{
				Image: visionImage{
					Content: base64.StdEncoding.EncodeToString(imageData),
				},
				Features: []visionFeature{
					{
						Type:       c.config.FeatureType,
						MaxResults: c.config.MaxResults,
					},
				},
			},
		},
	}
}

// extractTextFromResponse extracts the text from the Vision API response.
func (c *VisionClient) extractTextFromResponse(resp *visionResponse) (string, error) {
	if len(resp.Responses) == 0 {
		return "", ErrEmptyResponse
	}

	item := resp.Responses[0]

	// Check for API-level error
	if item.Error.Message != "" {
		return "", fmt.Errorf("ocrprocessor: Vision API error: %s (code: %d)", item.Error.Message, item.Error.Code)
	}

	text := item.FullTextAnnotation.Text
	if text == "" {
		return "", ErrNoTextFound
	}

	return text, nil
}

// ValidateAPIKey makes a minimal API call to verify the key works.
// This can be used to validate the key before processing real images.
//
// Parameters:
//   - ctx: context for cancellation/timeout
//
// Returns nil if the key is valid, or an error describing the issue.
func (c *VisionClient) ValidateAPIKey(ctx context.Context) error {
	log := c.logger

	log.Debug("validating API key with minimal request")

	// 1x1 pixel transparent PNG
	minimalImage := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00,
		0x0A, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49,
		0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	// Use TEXT_DETECTION for validation (lighter weight)
	reqBody := &visionRequest{
		Requests: []visionRequestItem{
			{
				Image: visionImage{
					Content: base64.StdEncoding.EncodeToString(minimalImage),
				},
				Features: []visionFeature{
					{
						Type:       "TEXT_DETECTION",
						MaxResults: 1,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("ocrprocessor: failed to marshal validation request: %w", err)
	}

	url := fmt.Sprintf("%s?key=%s", c.config.Endpoint, c.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("ocrprocessor: failed to create validation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ocrprocessor: API key validation failed: %w", err)
	}
	defer resp.Body.Close()

	// Read body for error details
	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("ocrprocessor: invalid API key: %s", string(bodyBytes))
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ocrprocessor: API validation returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response to check for API-level errors
	var visionResp visionResponse
	if err := json.Unmarshal(bodyBytes, &visionResp); err == nil {
		if len(visionResp.Responses) > 0 && visionResp.Responses[0].Error.Message != "" {
			errMsg := visionResp.Responses[0].Error.Message
			// "no text found" is not an error for validation
			if errMsg != "Unable to find text in image" {
				return fmt.Errorf("ocrprocessor: API key validation error: %s", errMsg)
			}
		}
	}

	log.Debug("API key validated successfully")
	return nil
}

// GetMaskedAPIKey returns a masked version of the API key for safe logging.
func (c *VisionClient) GetMaskedAPIKey() string {
	return MaskAPIKey(c.apiKey)
}
