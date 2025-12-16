// Package ocrprocessor provides OCR (Optical Character Recognition) functionality
// for CanvusLocalLLM using Google Cloud Vision API.
//
// processor.go implements the Processor organism that orchestrates OCR processing.
// It composes:
//   - client.go: VisionClient for Google Vision API access
//   - atoms.go: API key validation functions
//   - logging.Logger: structured logging
package ocrprocessor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"go_backend/logging"

	"go.uber.org/zap"
)

// ProcessorConfig holds configuration for the OCR processor.
type ProcessorConfig struct {
	// VisionClientConfig for the underlying Vision API client
	VisionClientConfig VisionClientConfig

	// MaxImageSize is the maximum image size in bytes (0 = no limit)
	MaxImageSize int64

	// SupportedFormats lists supported image MIME types
	// If empty, all formats supported by Vision API are accepted
	SupportedFormats []string

	// DownloadTimeout for downloading images from URLs
	DownloadTimeout time.Duration
}

// DefaultProcessorConfig returns sensible default configuration.
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		VisionClientConfig: DefaultVisionClientConfig(),
		MaxImageSize:       20 * 1024 * 1024, // 20MB
		SupportedFormats: []string{
			"image/jpeg",
			"image/png",
			"image/gif",
			"image/bmp",
			"image/webp",
			"image/tiff",
			"application/pdf",
		},
		DownloadTimeout: 30 * time.Second,
	}
}

// Common processor errors.
var (
	// ErrImageTooLarge indicates the image exceeds MaxImageSize.
	ErrImageTooLarge = errors.New("ocrprocessor: image exceeds maximum size")

	// ErrUnsupportedFormat indicates the image format is not supported.
	ErrUnsupportedFormat = errors.New("ocrprocessor: unsupported image format")

	// ErrProcessorNotConfigured indicates the processor is missing required config.
	ErrProcessorNotConfigured = errors.New("ocrprocessor: processor not properly configured")

	// ErrImageLoadFailed indicates the image could not be loaded.
	ErrImageLoadFailed = errors.New("ocrprocessor: failed to load image")
)

// ProcessResult contains the complete result of OCR processing.
type ProcessResult struct {
	// Text is the extracted text from the image
	Text string

	// ProcessingTime is the total time taken to process
	ProcessingTime time.Duration

	// VisionAPITime is the time spent in Vision API call
	VisionAPITime time.Duration

	// ImageSize is the size of the processed image in bytes
	ImageSize int64
}

// ProgressCallback is called to report processing progress.
// stage is the current stage name, progress is 0.0-1.0, message is a human-readable status.
type ProgressCallback func(stage string, progress float64, message string)

// Processor orchestrates OCR processing using Google Vision API.
//
// Thread-Safety:
//   - Processor is safe for concurrent use
//   - Each Process call is independent
type Processor struct {
	config     ProcessorConfig
	client     *VisionClient
	httpClient *http.Client
	logger     *logging.Logger
	progress   ProgressCallback
}

// NewProcessor creates a new OCR Processor.
//
// Parameters:
//   - apiKey: Google Cloud API key with Vision API access
//   - httpClient: HTTP client for API requests (use core.GetHTTPClient())
//   - logger: structured logger for operation tracking
//   - config: processor configuration
//
// Returns an error if the API key is invalid or dependencies are nil.
//
// Example:
//
//	httpClient := core.GetHTTPClient()
//	logger := logging.New()
//	processor, err := NewProcessor("AIza...", httpClient, logger, DefaultProcessorConfig())
func NewProcessor(apiKey string, httpClient *http.Client, logger *logging.Logger, config ProcessorConfig) (*Processor, error) {
	if httpClient == nil {
		return nil, ErrNilClient
	}
	if logger == nil {
		return nil, ErrNilLogger
	}

	// Create the underlying Vision client
	visionClient, err := NewVisionClient(apiKey, httpClient, logger, config.VisionClientConfig)
	if err != nil {
		return nil, fmt.Errorf("ocrprocessor: failed to create vision client: %w", err)
	}

	return &Processor{
		config:     config,
		client:     visionClient,
		httpClient: httpClient,
		logger:     logger.Named("ocr-processor"),
		progress:   nil,
	}, nil
}

// NewProcessorWithProgress creates a Processor with a progress callback.
//
// Example:
//
//	processor, err := NewProcessorWithProgress(apiKey, httpClient, logger, config,
//	    func(stage string, progress float64, msg string) {
//	        fmt.Printf("[%s] %.0f%% - %s\n", stage, progress*100, msg)
//	    })
func NewProcessorWithProgress(apiKey string, httpClient *http.Client, logger *logging.Logger, config ProcessorConfig, progress ProgressCallback) (*Processor, error) {
	p, err := NewProcessor(apiKey, httpClient, logger, config)
	if err != nil {
		return nil, err
	}
	p.progress = progress
	return p, nil
}

// SetProgressCallback sets or updates the progress callback.
func (p *Processor) SetProgressCallback(progress ProgressCallback) {
	p.progress = progress
}

// ProcessImage extracts text from image data.
// This is the main entry point for OCR processing with raw image bytes.
//
// Parameters:
//   - ctx: context for cancellation/timeout
//   - imageData: raw image bytes
//
// Returns the processing result or an error.
//
// Example:
//
//	imageData, _ := os.ReadFile("image.png")
//	result, err := processor.ProcessImage(ctx, imageData)
//	fmt.Println(result.Text)
func (p *Processor) ProcessImage(ctx context.Context, imageData []byte) (*ProcessResult, error) {
	if p.client == nil {
		return nil, ErrProcessorNotConfigured
	}

	start := time.Now()
	log := p.logger.With(
		zap.Int("image_size_bytes", len(imageData)),
	)

	log.Info("starting OCR processing")
	p.reportProgress("processing", 0.0, "Validating image...")

	// Validate image size
	if p.config.MaxImageSize > 0 && int64(len(imageData)) > p.config.MaxImageSize {
		return nil, fmt.Errorf("%w: %d bytes (max: %d)", ErrImageTooLarge, len(imageData), p.config.MaxImageSize)
	}

	p.reportProgress("processing", 0.2, "Sending to Vision API...")

	// Perform OCR using the Vision client
	ocrResult, err := p.client.ExtractText(ctx, imageData)
	if err != nil {
		log.Error("OCR extraction failed", zap.Error(err))
		return nil, err
	}

	p.reportProgress("processing", 1.0, "OCR complete")

	processingTime := time.Since(start)
	log.Info("OCR processing completed",
		zap.Int("text_length", len(ocrResult.Text)),
		zap.Duration("processing_time", processingTime),
		zap.Duration("vision_api_time", ocrResult.ProcessingTime))

	return &ProcessResult{
		Text:           ocrResult.Text,
		ProcessingTime: processingTime,
		VisionAPITime:  ocrResult.ProcessingTime,
		ImageSize:      int64(len(imageData)),
	}, nil
}

// ProcessFile extracts text from an image file.
//
// Parameters:
//   - ctx: context for cancellation/timeout
//   - filePath: path to the image file
//
// Returns the processing result or an error.
//
// Example:
//
//	result, err := processor.ProcessFile(ctx, "/path/to/image.png")
//	fmt.Println(result.Text)
func (p *Processor) ProcessFile(ctx context.Context, filePath string) (*ProcessResult, error) {
	if p.client == nil {
		return nil, ErrProcessorNotConfigured
	}

	log := p.logger.With(zap.String("file_path", filePath))
	log.Info("loading image from file")
	p.reportProgress("loading", 0.0, "Loading image file...")

	// Read the file
	imageData, err := os.ReadFile(filePath)
	if err != nil {
		log.Error("failed to read image file", zap.Error(err))
		return nil, fmt.Errorf("%w: %v", ErrImageLoadFailed, err)
	}

	p.reportProgress("loading", 1.0, "Image loaded")

	return p.ProcessImage(ctx, imageData)
}

// ProcessURL extracts text from an image at a URL.
//
// Parameters:
//   - ctx: context for cancellation/timeout
//   - imageURL: URL of the image to process
//
// Returns the processing result or an error.
//
// Example:
//
//	result, err := processor.ProcessURL(ctx, "https://example.com/image.png")
//	fmt.Println(result.Text)
func (p *Processor) ProcessURL(ctx context.Context, imageURL string) (*ProcessResult, error) {
	if p.client == nil {
		return nil, ErrProcessorNotConfigured
	}

	log := p.logger.With(zap.String("image_url", imageURL))
	log.Info("downloading image from URL")
	p.reportProgress("downloading", 0.0, "Downloading image...")

	// Create download context with timeout
	downloadCtx, cancel := context.WithTimeout(ctx, p.config.DownloadTimeout)
	defer cancel()

	// Create request
	req, err := http.NewRequestWithContext(downloadCtx, "GET", imageURL, nil)
	if err != nil {
		log.Error("failed to create download request", zap.Error(err))
		return nil, fmt.Errorf("%w: failed to create request: %v", ErrImageLoadFailed, err)
	}

	// Download
	resp, err := p.httpClient.Do(req)
	if err != nil {
		log.Error("failed to download image", zap.Error(err))
		return nil, fmt.Errorf("%w: download failed: %v", ErrImageLoadFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP status %d", ErrImageLoadFailed, resp.StatusCode)
	}

	p.reportProgress("downloading", 0.5, "Reading image data...")

	// Check content length if available
	if p.config.MaxImageSize > 0 && resp.ContentLength > p.config.MaxImageSize {
		return nil, fmt.Errorf("%w: %d bytes (max: %d)", ErrImageTooLarge, resp.ContentLength, p.config.MaxImageSize)
	}

	// Read body with size limit
	var imageData []byte
	if p.config.MaxImageSize > 0 {
		limitReader := io.LimitReader(resp.Body, p.config.MaxImageSize+1)
		imageData, err = io.ReadAll(limitReader)
		if int64(len(imageData)) > p.config.MaxImageSize {
			return nil, fmt.Errorf("%w: exceeded %d bytes", ErrImageTooLarge, p.config.MaxImageSize)
		}
	} else {
		imageData, err = io.ReadAll(resp.Body)
	}

	if err != nil {
		log.Error("failed to read image data", zap.Error(err))
		return nil, fmt.Errorf("%w: failed to read response: %v", ErrImageLoadFailed, err)
	}

	p.reportProgress("downloading", 1.0, "Image downloaded")

	log.Debug("image downloaded successfully",
		zap.Int("size_bytes", len(imageData)))

	return p.ProcessImage(ctx, imageData)
}

// reportProgress calls the progress callback if set.
func (p *Processor) reportProgress(stage string, progress float64, message string) {
	if p.progress != nil {
		p.progress(stage, progress, message)
	}
}

// ValidateAPIKey validates the configured API key by making a minimal API call.
//
// Parameters:
//   - ctx: context for cancellation/timeout
//
// Returns nil if the key is valid, or an error describing the issue.
func (p *Processor) ValidateAPIKey(ctx context.Context) error {
	if p.client == nil {
		return ErrProcessorNotConfigured
	}
	return p.client.ValidateAPIKey(ctx)
}

// GetMaskedAPIKey returns a masked version of the API key for safe logging.
func (p *Processor) GetMaskedAPIKey() string {
	if p.client == nil {
		return "[not configured]"
	}
	return p.client.GetMaskedAPIKey()
}

// ProcessImageWithConfig is a convenience function for one-off processing.
// It creates a processor with the given configuration and processes an image.
//
// Example:
//
//	result, err := ProcessImageWithConfig(ctx, apiKey, httpClient, logger, DefaultProcessorConfig(), imageData)
func ProcessImageWithConfig(ctx context.Context, apiKey string, httpClient *http.Client, logger *logging.Logger, config ProcessorConfig, imageData []byte) (*ProcessResult, error) {
	processor, err := NewProcessor(apiKey, httpClient, logger, config)
	if err != nil {
		return nil, err
	}
	return processor.ProcessImage(ctx, imageData)
}

// ExtractText is a convenience function for simple text extraction.
// It uses default configuration and returns just the extracted text.
//
// Parameters:
//   - ctx: context for cancellation/timeout
//   - apiKey: Google Cloud API key
//   - httpClient: HTTP client for API requests (use core.GetHTTPClient())
//   - logger: structured logger for operation tracking
//   - imageData: raw image bytes
//
// Returns the extracted text or an error.
//
// Example:
//
//	httpClient := core.GetHTTPClient(cfg, 30*time.Second)
//	text, err := ExtractText(ctx, "AIza...", httpClient, logger, imageData)
//	fmt.Println(text)
func ExtractText(ctx context.Context, apiKey string, httpClient *http.Client, logger *logging.Logger, imageData []byte) (string, error) {
	processor, err := NewProcessor(apiKey, httpClient, logger, DefaultProcessorConfig())
	if err != nil {
		return "", err
	}

	result, err := processor.ProcessImage(ctx, imageData)
	if err != nil {
		return "", err
	}

	return result.Text, nil
}
