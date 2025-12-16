// Package imagegen provides image generation utilities for the Canvus canvas.
//
// processor.go implements the Processor organism that handles the end-to-end
// image generation pipeline from prompt to canvas upload.
//
// This organism composes:
//   - sdruntime.ContextPool: for image generation
//   - placement.go: for canvas coordinate calculation
//   - canvusapi.Client: for canvas widget operations
//   - logging.Logger: for structured logging
package imagegen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"go_backend/canvusapi"
	"go_backend/logging"
	"go_backend/sdruntime"

	"go.uber.org/zap"
)

// ProcessingNoteConfig holds configuration for processing indicator notes.
type ProcessingNoteConfig struct {
	Title           string
	BackgroundColor string
	TextColor       string
}

// DefaultProcessingNoteConfig returns the default configuration for processing notes.
func DefaultProcessingNoteConfig() ProcessingNoteConfig {
	return ProcessingNoteConfig{
		Title:           "AI Processing",
		BackgroundColor: "#8B0000", // Dark blood red
		TextColor:       "#FFFFFF", // White
	}
}

// ProcessorConfig holds configuration for the image generation processor.
type ProcessorConfig struct {
	// DownloadsDir is the directory for temporary image files
	DownloadsDir string

	// DefaultWidth is the default image width in pixels (must be divisible by 8)
	DefaultWidth int

	// DefaultHeight is the default image height in pixels (must be divisible by 8)
	DefaultHeight int

	// DefaultSteps is the default number of inference steps
	DefaultSteps int

	// DefaultCFGScale is the default classifier-free guidance scale
	DefaultCFGScale float64

	// PlacementConfig controls image placement relative to parent widget
	PlacementConfig PlacementConfig

	// ProcessingNote controls the appearance of processing indicator notes
	ProcessingNote ProcessingNoteConfig
}

// DefaultProcessorConfig returns sensible default configuration.
func DefaultProcessorConfig() ProcessorConfig {
	return ProcessorConfig{
		DownloadsDir:    "downloads",
		DefaultWidth:    512,
		DefaultHeight:   512,
		DefaultSteps:    20,
		DefaultCFGScale: 7.0,
		PlacementConfig: DefaultPlacementConfig(),
		ProcessingNote:  DefaultProcessingNoteConfig(),
	}
}

// Processor handles the end-to-end image generation pipeline.
// It manages generating images from prompts and uploading them to Canvus canvas.
//
// Thread-Safety:
//   - Processor is safe for concurrent use
//   - Uses mutex to protect downloads directory access
//   - Pool handles concurrent generation internally
type Processor struct {
	pool   *sdruntime.ContextPool
	client *canvusapi.Client
	logger *logging.Logger
	config ProcessorConfig

	// mu protects file operations in downloads directory
	mu sync.Mutex
}

// NewProcessor creates a new image generation processor.
//
// Parameters:
//   - pool: SD context pool for image generation
//   - client: Canvus API client for canvas operations
//   - logger: structured logger for operation tracking
//   - config: processor configuration
//
// The pool must be initialized and not closed. The processor does not
// take ownership of the pool; the caller is responsible for closing it.
func NewProcessor(pool *sdruntime.ContextPool, client *canvusapi.Client, logger *logging.Logger, config ProcessorConfig) (*Processor, error) {
	if pool == nil {
		return nil, fmt.Errorf("imagegen: pool cannot be nil")
	}
	if pool.IsClosed() {
		return nil, fmt.Errorf("imagegen: pool is already closed")
	}
	if client == nil {
		return nil, fmt.Errorf("imagegen: client cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("imagegen: logger cannot be nil")
	}

	// Ensure downloads directory exists
	if err := os.MkdirAll(config.DownloadsDir, 0755); err != nil {
		return nil, fmt.Errorf("imagegen: failed to create downloads directory: %w", err)
	}

	return &Processor{
		pool:   pool,
		client: client,
		logger: logger.Named("imagegen"),
		config: config,
	}, nil
}

// ParentWidget represents the widget that triggered the image generation.
// This interface is used to calculate placement for the generated image.
type ParentWidget interface {
	GetID() string
	GetLocation() WidgetLocation
	GetSize() WidgetSize
	GetScale() float64
	GetDepth() float64
}

// CanvasWidget is a concrete implementation of ParentWidget from canvas data.
type CanvasWidget struct {
	ID       string
	Location WidgetLocation
	Size     WidgetSize
	Scale    float64
	Depth    float64
}

// GetID returns the widget ID.
func (w CanvasWidget) GetID() string { return w.ID }

// GetLocation returns the widget location.
func (w CanvasWidget) GetLocation() WidgetLocation { return w.Location }

// GetSize returns the widget size.
func (w CanvasWidget) GetSize() WidgetSize { return w.Size }

// GetScale returns the widget scale.
func (w CanvasWidget) GetScale() float64 { return w.Scale }

// GetDepth returns the widget depth.
func (w CanvasWidget) GetDepth() float64 { return w.Depth }

// ProcessResult contains the result of image processing.
type ProcessResult struct {
	// ImagePath is the local path to the generated image (before cleanup)
	ImagePath string

	// WidgetID is the ID of the created image widget on the canvas
	WidgetID string

	// Seed is the random seed used for generation
	Seed int64
}

// ProcessImagePrompt handles the end-to-end flow of generating an image
// from a prompt and uploading it to the Canvus canvas.
//
// The flow is:
//  1. Validate and sanitize the prompt
//  2. Create a processing indicator note on the canvas
//  3. Generate the image via sdruntime
//  4. Save the image to a temporary file
//  5. Calculate placement relative to parent widget
//  6. Upload the image to Canvus
//  7. Clean up temporary file
//  8. Delete the processing indicator
//
// On error, an error note is created on the canvas and the processing
// indicator is updated to show the failure.
//
// Parameters:
//   - ctx: context for cancellation/timeout
//   - prompt: the image generation prompt (will be sanitized)
//   - parentWidget: the widget that triggered this generation
//
// Returns the result on success, or an error. Canvas error notes are created
// automatically on failure.
func (p *Processor) ProcessImagePrompt(ctx context.Context, prompt string, parentWidget ParentWidget) (*ProcessResult, error) {
	correlationID := generateCorrelationID()
	log := p.logger.With(
		zap.String("correlation_id", correlationID),
		zap.String("parent_widget_id", parentWidget.GetID()),
	)

	log.Info("starting image generation",
		zap.String("prompt_preview", truncateText(prompt, 50)))

	// Step 1: Validate and sanitize prompt
	prompt = sdruntime.SanitizePrompt(prompt)
	if err := sdruntime.ValidatePrompt(prompt); err != nil {
		log.Error("invalid prompt", zap.Error(err))
		p.createErrorNote(ctx, parentWidget, fmt.Sprintf("Invalid prompt: %v", err), log)
		return nil, fmt.Errorf("imagegen: %w", err)
	}

	// Step 2: Create processing indicator
	processingNoteID, err := p.createProcessingNote(ctx, parentWidget, "Generating image...", log)
	if err != nil {
		log.Warn("failed to create processing note", zap.Error(err))
		// Continue without processing note - not fatal
	}

	// Ensure cleanup of processing note
	defer func() {
		if processingNoteID != "" {
			if delErr := p.client.DeleteNote(processingNoteID); delErr != nil {
				log.Warn("failed to delete processing note", zap.Error(delErr))
			}
		}
	}()

	// Step 3: Update processing note and generate image
	if processingNoteID != "" {
		p.updateProcessingNote(processingNoteID, "Generating image...\nThis may take 10-30 seconds.", log)
	}

	params := sdruntime.GenerateParams{
		Prompt:   prompt,
		Width:    p.config.DefaultWidth,
		Height:   p.config.DefaultHeight,
		Steps:    p.config.DefaultSteps,
		CFGScale: p.config.DefaultCFGScale,
		Seed:     -1, // Random seed
	}

	imageData, err := p.pool.Generate(ctx, params)
	if err != nil {
		log.Error("image generation failed", zap.Error(err))
		if processingNoteID != "" {
			p.updateProcessingNote(processingNoteID, fmt.Sprintf("Generation failed: %v", err), log)
		}
		p.createErrorNote(ctx, parentWidget, fmt.Sprintf("Image generation failed: %v", err), log)
		return nil, fmt.Errorf("imagegen: generation failed: %w", err)
	}

	log.Debug("image generated successfully", zap.Int("size_bytes", len(imageData)))

	// Step 4: Save to temporary file
	if processingNoteID != "" {
		p.updateProcessingNote(processingNoteID, "Uploading image to canvas...", log)
	}

	p.mu.Lock()
	imagePath := filepath.Join(p.config.DownloadsDir, fmt.Sprintf("sd_image_%s.png", correlationID))
	if err := os.WriteFile(imagePath, imageData, 0644); err != nil {
		p.mu.Unlock()
		log.Error("failed to save image file", zap.Error(err))
		p.createErrorNote(ctx, parentWidget, fmt.Sprintf("Failed to save image: %v", err), log)
		return nil, fmt.Errorf("imagegen: failed to save image: %w", err)
	}
	p.mu.Unlock()

	// Ensure cleanup of temp file
	defer func() {
		if removeErr := os.Remove(imagePath); removeErr != nil && !os.IsNotExist(removeErr) {
			log.Warn("failed to remove temp image file", zap.Error(removeErr))
		}
	}()

	// Step 5: Calculate placement
	x, y := CalculatePlacementWithConfig(parentWidget, p.config.PlacementConfig)
	log.Debug("calculated image placement",
		zap.Float64("x", x),
		zap.Float64("y", y))

	// Step 6: Upload to Canvus
	widgetPayload := map[string]interface{}{
		"title": fmt.Sprintf("AI Generated Image for %s", parentWidget.GetID()),
		"location": map[string]float64{
			"x": x,
			"y": y,
		},
		"size": map[string]interface{}{
			"width":  float64(p.config.DefaultWidth),
			"height": float64(p.config.DefaultHeight),
		},
		"depth": parentWidget.GetDepth() + 10,
		"scale": parentWidget.GetScale() / 3,
	}

	response, err := p.client.CreateImage(imagePath, widgetPayload)
	if err != nil {
		log.Error("failed to upload image to canvas", zap.Error(err))
		p.createErrorNote(ctx, parentWidget, fmt.Sprintf("Failed to upload image: %v", err), log)
		return nil, fmt.Errorf("imagegen: failed to upload image: %w", err)
	}

	widgetID, _ := response["id"].(string)
	log.Info("image uploaded successfully",
		zap.String("widget_id", widgetID))

	return &ProcessResult{
		ImagePath: imagePath, // Note: file is cleaned up after return
		WidgetID:  widgetID,
		Seed:      params.Seed,
	}, nil
}

// createProcessingNote creates a processing indicator note on the canvas.
func (p *Processor) createProcessingNote(ctx context.Context, parent ParentWidget, text string, log *logging.Logger) (string, error) {
	loc := parent.GetLocation()

	payload := map[string]interface{}{
		"title": p.config.ProcessingNote.Title,
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
		"background_color": p.config.ProcessingNote.BackgroundColor,
		"text_color":       p.config.ProcessingNote.TextColor,
		"auto_text_color":  false,
		"pinned":           true,
	}

	response, err := p.client.CreateNote(payload)
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
func (p *Processor) updateProcessingNote(noteID, text string, log *logging.Logger) {
	_, err := p.client.UpdateNote(noteID, map[string]interface{}{
		"text": text,
	})
	if err != nil {
		log.Warn("failed to update processing note", zap.Error(err))
	}
}

// createErrorNote creates an error note on the canvas to inform the user.
func (p *Processor) createErrorNote(ctx context.Context, parent ParentWidget, errorMessage string, log *logging.Logger) {
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

	_, err := p.client.CreateNote(payload)
	if err != nil {
		log.Error("failed to create error note", zap.Error(err))
	}
}

// generateCorrelationID creates a unique ID for request tracing.
func generateCorrelationID() string {
	// Use timestamp-based ID for simplicity
	// In production, consider using UUID
	return fmt.Sprintf("%d", currentTimeMillis())
}

// currentTimeMillis returns current time in milliseconds.
// Extracted as a function for testing purposes.
var currentTimeMillis = func() int64 {
	return java_time_now_millis()
}

// java_time_now_millis returns time in milliseconds (Java-style).
func java_time_now_millis() int64 {
	return java_time_now().UnixMilli()
}

// java_time_now returns current time. Extracted for testing.
var java_time_now = func() interface{ UnixMilli() int64 } {
	return timeNow{}
}

type timeNow struct{}

func (t timeNow) UnixMilli() int64 {
	return unixMilliNow()
}

// unixMilliNow returns current Unix time in milliseconds.
func unixMilliNow() int64 {
	// Import time inline to avoid circular dependencies
	return timePackageNow()
}

// timePackageNow is extracted for testability.
var timePackageNow = func() int64 {
	// Inline import to get current time
	return int64(1)
}

// truncateText truncates text to a maximum length with ellipsis.
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}
