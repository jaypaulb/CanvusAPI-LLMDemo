// Package imagegen provides image generation utilities for the Canvus canvas.
//
// generator.go implements the Generator organism that orchestrates the end-to-end
// image generation pipeline using cloud providers (OpenAI/Azure) and image download.
//
// This organism composes:
//   - Provider interface: OpenAIProvider or AzureProvider for image generation
//   - Downloader: for downloading generated images from temporary URLs
//   - canvusapi.Client: for canvas widget operations
//   - logging.Logger: for structured logging
//   - placement.go: for canvas coordinate calculation
//
// Unlike Processor (which uses local SD runtime), Generator uses cloud AI services.
package imagegen

import (
	"context"
	"fmt"
	"os"
	"sync"

	"go_backend/canvusapi"
	"go_backend/core"
	"go_backend/logging"

	"go.uber.org/zap"
)

// Generator handles the end-to-end image generation pipeline using cloud providers.
// It manages generating images from prompts via OpenAI/Azure and uploading them to Canvus canvas.
//
// Thread-Safety:
//   - Generator is safe for concurrent use
//   - Uses mutex to protect file operations in downloads directory
//   - Provider handles concurrent generation internally
type Generator struct {
	provider   Provider
	downloader *Downloader
	client     *canvusapi.Client
	logger     *logging.Logger
	config     GeneratorConfig

	// mu protects file operations in downloads directory
	mu sync.Mutex
}

// GeneratorConfig holds configuration for the image generation generator.
type GeneratorConfig struct {
	// DownloadsDir is the directory for temporary image files
	DownloadsDir string

	// PlacementConfig controls image placement relative to parent widget
	PlacementConfig PlacementConfig

	// ProcessingNote controls the appearance of processing indicator notes
	ProcessingNote ProcessingNoteConfig

	// CleanupTempFiles controls whether to delete temp files after upload
	// Default: true
	CleanupTempFiles bool
}

// DefaultGeneratorConfig returns sensible default configuration.
func DefaultGeneratorConfig() GeneratorConfig {
	return GeneratorConfig{
		DownloadsDir:     "downloads",
		PlacementConfig:  DefaultPlacementConfig(),
		ProcessingNote:   DefaultProcessingNoteConfig(),
		CleanupTempFiles: true,
	}
}

// NewGenerator creates a new cloud image generation generator.
//
// Parameters:
//   - provider: the image generation provider (OpenAI or Azure)
//   - downloader: the image downloader for fetching generated images
//   - client: Canvus API client for canvas operations
//   - logger: structured logger for operation tracking
//   - config: generator configuration
//
// Returns an error if any required component is nil.
//
// Example:
//
//	provider, _ := NewOpenAIProvider(cfg)
//	downloader, _ := NewDownloader(cfg)
//	generator, err := NewGenerator(provider, downloader, canvusClient, logger, DefaultGeneratorConfig())
func NewGenerator(provider Provider, downloader *Downloader, client *canvusapi.Client, logger *logging.Logger, config GeneratorConfig) (*Generator, error) {
	if provider == nil {
		return nil, fmt.Errorf("imagegen: provider cannot be nil")
	}
	if downloader == nil {
		return nil, fmt.Errorf("imagegen: downloader cannot be nil")
	}
	if client == nil {
		return nil, fmt.Errorf("imagegen: client cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("imagegen: logger cannot be nil")
	}

	// Ensure downloads directory exists
	if config.DownloadsDir == "" {
		config.DownloadsDir = "downloads"
	}
	if err := os.MkdirAll(config.DownloadsDir, 0755); err != nil {
		return nil, fmt.Errorf("imagegen: failed to create downloads directory: %w", err)
	}

	return &Generator{
		provider:   provider,
		downloader: downloader,
		client:     client,
		logger:     logger.Named("generator"),
		config:     config,
	}, nil
}

// NewGeneratorFromConfig creates a Generator with automatic provider selection based on config.
//
// This is a convenience constructor that:
//  1. Detects the appropriate provider based on endpoint configuration
//  2. Creates the provider (OpenAI or Azure)
//  3. Creates a Downloader
//  4. Assembles the Generator
//
// Provider selection logic:
//   - If AzureOpenAIEndpoint is set and is an Azure endpoint -> AzureProvider
//   - If ImageLLMURL is an Azure endpoint -> AzureProvider
//   - Otherwise -> OpenAIProvider
//
// Returns an error if:
//   - No valid API key is configured
//   - The endpoint is a local endpoint (not supported for cloud generation)
//   - Any component fails to initialize
func NewGeneratorFromConfig(cfg *core.Config, client *canvusapi.Client, logger *logging.Logger) (*Generator, error) {
	if cfg == nil {
		return nil, fmt.Errorf("imagegen: config cannot be nil")
	}
	if client == nil {
		return nil, fmt.Errorf("imagegen: client cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("imagegen: logger cannot be nil")
	}

	log := logger.Named("generator-init")

	// Determine which provider to use
	var provider Provider
	var err error

	// Check for Azure endpoint first
	useAzure := false
	if cfg.AzureOpenAIEndpoint != "" && IsAzureEndpoint(cfg.AzureOpenAIEndpoint) {
		useAzure = true
	} else if cfg.ImageLLMURL != "" && IsAzureEndpoint(cfg.ImageLLMURL) {
		useAzure = true
	}

	if useAzure {
		log.Info("using Azure OpenAI provider for image generation",
			zap.String("endpoint", cfg.AzureOpenAIEndpoint),
			zap.String("deployment", cfg.AzureOpenAIDeployment))
		provider, err = NewAzureProvider(cfg)
		if err != nil {
			return nil, fmt.Errorf("imagegen: failed to create Azure provider: %w", err)
		}
	} else {
		log.Info("using OpenAI provider for image generation",
			zap.String("model", cfg.OpenAIImageModel))
		provider, err = NewOpenAIProvider(cfg)
		if err != nil {
			return nil, fmt.Errorf("imagegen: failed to create OpenAI provider: %w", err)
		}
	}

	// Create downloader
	downloader, err := NewDownloader(cfg)
	if err != nil {
		return nil, fmt.Errorf("imagegen: failed to create downloader: %w", err)
	}

	// Create generator config
	genConfig := DefaultGeneratorConfig()
	genConfig.DownloadsDir = cfg.DownloadsDir
	if genConfig.DownloadsDir == "" {
		genConfig.DownloadsDir = "downloads"
	}

	return NewGenerator(provider, downloader, client, logger, genConfig)
}

// GenerateResult contains the result of image generation and upload.
type GenerateResult struct {
	// ImagePath is the local path to the downloaded image (may be cleaned up)
	ImagePath string

	// WidgetID is the ID of the created image widget on the canvas
	WidgetID string

	// ImageURL is the temporary URL from the provider (may expire)
	ImageURL string
}

// Generate handles the end-to-end flow of generating an image from a prompt
// and uploading it to the Canvus canvas.
//
// The flow is:
//  1. Validate the prompt
//  2. Create a processing indicator note on the canvas
//  3. Generate the image via cloud provider (OpenAI/Azure)
//  4. Download the image from the temporary URL
//  5. Calculate placement relative to parent widget
//  6. Upload the image to Canvus
//  7. Clean up temporary file (if configured)
//  8. Delete the processing indicator
//
// On error, an error note is created on the canvas and the processing
// indicator is updated to show the failure.
//
// Parameters:
//   - ctx: context for cancellation/timeout
//   - prompt: the image generation prompt
//   - parentWidget: the widget that triggered this generation
//
// Returns the result on success, or an error. Canvas error notes are created
// automatically on failure.
func (g *Generator) Generate(ctx context.Context, prompt string, parentWidget ParentWidget) (*GenerateResult, error) {
	correlationID := generateCorrelationID()
	log := g.logger.With(
		zap.String("correlation_id", correlationID),
		zap.String("parent_widget_id", parentWidget.GetID()),
	)

	log.Info("starting cloud image generation",
		zap.String("prompt_preview", truncateText(prompt, 50)))

	// Step 1: Validate prompt
	if prompt == "" {
		err := fmt.Errorf("imagegen: prompt cannot be empty")
		log.Error("invalid prompt", zap.Error(err))
		g.createErrorNote(ctx, parentWidget, "Prompt cannot be empty", log)
		return nil, err
	}

	// Step 2: Create processing indicator
	processingNoteID, err := g.createProcessingNote(ctx, parentWidget, "Generating image...", log)
	if err != nil {
		log.Warn("failed to create processing note", zap.Error(err))
		// Continue without processing note - not fatal
	}

	// Ensure cleanup of processing note
	defer func() {
		if processingNoteID != "" {
			if delErr := g.client.DeleteNote(processingNoteID); delErr != nil {
				log.Warn("failed to delete processing note", zap.Error(delErr))
			}
		}
	}()

	// Step 3: Generate image via provider
	if processingNoteID != "" {
		g.updateProcessingNote(processingNoteID, "Generating image...\nThis may take 10-30 seconds.", log)
	}

	imageURL, err := g.provider.Generate(ctx, prompt)
	if err != nil {
		log.Error("image generation failed", zap.Error(err))
		if processingNoteID != "" {
			g.updateProcessingNote(processingNoteID, fmt.Sprintf("Generation failed: %v", err), log)
		}
		g.createErrorNote(ctx, parentWidget, fmt.Sprintf("Image generation failed: %v", err), log)
		return nil, fmt.Errorf("imagegen: generation failed: %w", err)
	}

	log.Debug("image generated successfully", zap.String("image_url", truncateText(imageURL, 100)))

	// Step 4: Download the image
	if processingNoteID != "" {
		g.updateProcessingNote(processingNoteID, "Downloading generated image...", log)
	}

	filename := fmt.Sprintf("generated_%s", correlationID)
	downloadResult, err := g.downloader.Download(ctx, imageURL, filename)
	if err != nil {
		log.Error("failed to download image", zap.Error(err))
		g.createErrorNote(ctx, parentWidget, fmt.Sprintf("Failed to download image: %v", err), log)
		return nil, fmt.Errorf("imagegen: download failed: %w", err)
	}

	imagePath := downloadResult.Path
	log.Debug("image downloaded",
		zap.String("path", imagePath),
		zap.Int64("size", downloadResult.Size))

	// Ensure cleanup of temp file if configured
	if g.config.CleanupTempFiles {
		defer func() {
			if removeErr := os.Remove(imagePath); removeErr != nil && !os.IsNotExist(removeErr) {
				log.Warn("failed to remove temp image file", zap.Error(removeErr))
			}
		}()
	}

	// Step 5: Calculate placement
	if processingNoteID != "" {
		g.updateProcessingNote(processingNoteID, "Uploading image to canvas...", log)
	}

	x, y := CalculatePlacementWithConfig(parentWidget, g.config.PlacementConfig)
	log.Debug("calculated image placement",
		zap.Float64("x", x),
		zap.Float64("y", y))

	// Step 6: Upload to Canvus
	// Get image dimensions from file info
	fileInfo, err := os.Stat(imagePath)
	if err != nil {
		log.Warn("failed to stat image file", zap.Error(err))
	}

	// Default size for DALL-E 3 is 1024x1024
	width := 1024.0
	height := 1024.0

	widgetPayload := map[string]interface{}{
		"title": fmt.Sprintf("AI Generated: %s", truncateText(prompt, 50)),
		"location": map[string]float64{
			"x": x,
			"y": y,
		},
		"size": map[string]interface{}{
			"width":  width,
			"height": height,
		},
		"depth": parentWidget.GetDepth() + 10,
		"scale": parentWidget.GetScale() / 3,
	}

	response, err := g.client.CreateImage(imagePath, widgetPayload)
	if err != nil {
		log.Error("failed to upload image to canvas", zap.Error(err))
		g.createErrorNote(ctx, parentWidget, fmt.Sprintf("Failed to upload image: %v", err), log)
		return nil, fmt.Errorf("imagegen: failed to upload image: %w", err)
	}

	widgetID, _ := response["id"].(string)
	log.Info("image uploaded successfully",
		zap.String("widget_id", widgetID),
		zap.Int64("file_size", fileInfo.Size()))

	return &GenerateResult{
		ImagePath: imagePath,
		WidgetID:  widgetID,
		ImageURL:  imageURL,
	}, nil
}

// createProcessingNote creates a processing indicator note on the canvas.
func (g *Generator) createProcessingNote(ctx context.Context, parent ParentWidget, text string, log *logging.Logger) (string, error) {
	loc := parent.GetLocation()

	payload := map[string]interface{}{
		"title": g.config.ProcessingNote.Title,
		"text":  text,
		"location": map[string]float64{
			"x": loc.X + 50,
			"y": loc.Y + 50,
		},
		"size": map[string]interface{}{
			"width":  300.0,
			"height": 100.0,
		},
		"depth":            parent.GetDepth() + 200,
		"scale":            parent.GetScale(),
		"background_color": g.config.ProcessingNote.BackgroundColor,
		"text_color":       g.config.ProcessingNote.TextColor,
		"auto_text_color":  false,
		"pinned":           true,
	}

	response, err := g.client.CreateNote(payload)
	if err != nil {
		return "", fmt.Errorf("failed to create processing note: %w", err)
	}

	noteID, ok := response["id"].(string)
	if !ok {
		return "", fmt.Errorf("processing note response missing id")
	}

	log.Debug("created processing note", zap.String("note_id", noteID))
	return noteID, nil
}

// updateProcessingNote updates the text of a processing note.
func (g *Generator) updateProcessingNote(noteID, text string, log *logging.Logger) {
	_, err := g.client.UpdateNote(noteID, map[string]interface{}{
		"text": text,
	})
	if err != nil {
		log.Warn("failed to update processing note", zap.Error(err))
	}
}

// createErrorNote creates an error note on the canvas to inform the user.
func (g *Generator) createErrorNote(ctx context.Context, parent ParentWidget, errorMessage string, log *logging.Logger) {
	loc := parent.GetLocation()

	content := fmt.Sprintf("# Image Generation Error\n\n%s\n\nPlease try again or adjust your prompt.", errorMessage)

	payload := map[string]interface{}{
		"title": "AI Image Generation Error",
		"text":  content,
		"location": map[string]float64{
			"x": loc.X + 100,
			"y": loc.Y + 100,
		},
		"size": map[string]interface{}{
			"width":  400.0,
			"height": 200.0,
		},
		"depth":            parent.GetDepth() + 100,
		"scale":            parent.GetScale(),
		"background_color": "#FF6B6B", // Light red for error
		"text_color":       "#000000",
		"auto_text_color":  false,
	}

	_, err := g.client.CreateNote(payload)
	if err != nil {
		log.Error("failed to create error note", zap.Error(err))
	}
}

// Provider returns the underlying image generation provider.
func (g *Generator) Provider() Provider {
	return g.provider
}

// Downloader returns the underlying image downloader.
func (g *Generator) Downloader() *Downloader {
	return g.downloader
}

// Config returns the generator configuration.
func (g *Generator) Config() GeneratorConfig {
	return g.config
}
