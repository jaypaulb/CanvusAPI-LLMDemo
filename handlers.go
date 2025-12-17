package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go_backend/canvasanalyzer"
	"go_backend/canvusapi"
	"go_backend/core"
	"go_backend/db"
	"go_backend/handlers"
	"go_backend/imagegen"
	"go_backend/logging"
	"go_backend/ocrprocessor"
	"go_backend/pdfprocessor"

	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

// Add constants at the top
const (
	processingNoteTitle     = "AI Processing"
	processingNoteColor     = "#8B0000" // Dark blood red
	processingNoteTextColor = "#FFFFFF"

	// Google Vision API constants
	visionAPIEndpoint = "https://vision.googleapis.com/v1/images:annotate"
	visionFeatureType = "DOCUMENT_TEXT_DETECTION"

	// AI Note System Message - instructs the AI on how to respond to note triggers
	noteSystemMessage = `You are an assistant capable of interpreting structured text triggers from a Note widget. ` +
		`Evaluate whether the content in the Note is better suited for generating text or creating an image. ` +
		`If generating text, respond with a JSON object like: {"type": "text", "content": "..."}. ` +
		`If creating an image, respond with a JSON object like: {"type": "image", "content": "..."}. ` +
		`For image requests, you must craft a vivid, imaginative, and highly detailed prompt for an AI image generator. ` +
		`Do NOT simply repeat or rephrase the user's input. Instead, expand it into a unique, creative, and visually rich scene, including style, mood, composition, and any relevant artistic details. ` +
		`The response should be rich and meaningful. Never just a repeat of the user's input. ` +
		`Do not include any additional text or explanations.`
)

// Core types and configuration
type AINoteResponse struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}
// Shared resources and synchronization
var (
	logMutex       sync.Mutex
	downloadsMutex sync.Mutex
	metricsMutex   sync.Mutex
	handlerMetrics struct {
		processedNotes     int64
		processedImages    int64
		processedPDFs      int64
		errors             int64
		processingDuration time.Duration
	}
)

// Add this struct for the Vision API request
type GoogleVisionRequest struct {
	Requests []struct {
		Image struct {
			Content string `json:"content"`
		} `json:"image"`
		Features []struct {
			Type       string `json:"type"`
			MaxResults int    `json:"maxResults"`
		} `json:"features"`
	} `json:"requests"`
}

// Add this struct for the Vision API response
type GoogleVisionResponse struct {
	Responses []struct {
		FullTextAnnotation struct {
			Text string `json:"text"`
		} `json:"fullTextAnnotation"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	} `json:"responses"`
}

// generateCorrelationID creates a unique ID for request tracing
func generateCorrelationID() string {
	return uuid.New().String()[:8]
}

// truncateText truncates a text to a specified length
func truncateText(text string, length int) string {
	if len(text) > length {
		return text[:length]
	}
	return text
}

// recordProcessingHistory records an AI processing operation to the database.
// This is a helper function to avoid code duplication across handlers.
// It performs async database write via the repository.
func recordProcessingHistory(
	ctx context.Context,
	repo *db.Repository,
	correlationID string,
	canvasID string,
	widgetID string,
	operationType string,
	prompt string,
	response string,
	modelName string,
	inputTokens int,
	outputTokens int,
	durationMS int,
	status string,
	errorMessage string,
	log *logging.Logger,
) {
	if repo == nil {
		log.Debug("repository is nil, skipping database recording")
		return
	}

	record := db.ProcessingRecord{
		CorrelationID: correlationID,
		CanvasID:      canvasID,
		WidgetID:      widgetID,
		OperationType: operationType,
		Prompt:        truncateText(prompt, 5000),    // Limit prompt size
		Response:      truncateText(response, 10000), // Limit response size
		ModelName:     modelName,
		InputTokens:   inputTokens,
		OutputTokens:  outputTokens,
		DurationMS:    durationMS,
		Status:        status,
		ErrorMessage:  errorMessage,
	}

	_, err := repo.InsertProcessingHistory(ctx, record)
	if err != nil {
		log.Warn("failed to record processing history to database",
			zap.Error(err),
			zap.String("correlation_id", correlationID))
	} else {
		log.Debug("processing history recorded",
			zap.String("correlation_id", correlationID),
			zap.String("operation_type", operationType))
	}
}

// handleNote processes Note widget updates
func handleNote(update Update, client *canvusapi.Client, config *core.Config, logger *logging.Logger, repo *db.Repository) {
	noteID, _ := update["id"].(string)
	log := logger.With(
		zap.String("correlation_id", generateCorrelationID()),
		zap.String("widget_id", noteID),
		zap.String("widget_type", "Note"),
	)

	// Validate and check for AI trigger
	if err := handlers.ValidateUpdate(update); err != nil {
		log.Error("invalid update", zap.Error(err))
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return
	}
	noteText, ok := update["text"].(string)
	if !ok || !hasAITrigger(noteText) {
		return
	}

	// Initialize processing context
	npc := &noteProcessingContext{
		correlationID: generateCorrelationID(),
		noteID:        noteID,
		baseText:      extractAIPrompt(noteText),
		aiPrompt:      extractAIPrompt(noteText),
		start:         time.Now(),
		client:        client,
		config:        config,
		log:           log,
		repo:          repo,
		update:        update,
	}
	log.Info("processing AI trigger", zap.String("text_preview", truncateText(noteText, 30)))

	// Mark as processing
	if err := updateNoteWithRetry(client, noteID, map[string]interface{}{
		"text": npc.baseText + "\n\n processing...",
	}, config, log); err != nil {
		log.Error("failed to mark note as processing", zap.Error(err))
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return
	}

	// Setup context and update status
	ctx, cancel := context.WithTimeout(context.Background(), config.AITimeout)
	defer cancel()
	npc.ctx = ctx
	updateNoteWithRetry(client, noteID, map[string]interface{}{"text": npc.baseText + "\n\n Analyzing request..."}, config, log)

	// Generate AI response
	metricsLogger := logging.NewMetricsLogger(log)
	inferenceTimer := metricsLogger.StartInference(config.OpenAINoteModel)
	rawResponse, err := generateAIResponse(npc.aiPrompt, config, noteSystemMessage, log)
	if err != nil {
		recordNoteError(npc, err)
		metricsLogger.EndInference(inferenceTimer, 0, 0)
		if err := handleAIError(ctx, client, update, err, npc.baseText, config, log); err != nil {
			log.Error("failed to create error note", zap.Error(err))
		}
		return
	}

	promptTokens, completionTokens := len(npc.aiPrompt)/4, len(rawResponse)/4
	metricsLogger.EndInference(inferenceTimer, promptTokens, completionTokens)
	log.Debug("received AI response", zap.String("response_preview", truncateText(rawResponse, 50)))

	// Parse and process AI response
	var aiNoteResponse AINoteResponse
	if err := json.Unmarshal([]byte(rawResponse), &aiNoteResponse); err != nil {
		if err := handleAIError(ctx, client, update, err, npc.baseText, config, log); err != nil {
			log.Error("failed to create error note", zap.Error(err))
		}
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return
	}

	if err := processAIResponseType(npc, aiNoteResponse); err != nil {
		handleNoteCreationError(npc, err)
		return
	}

	// Success path
	clearProcessingStatus(client, noteID, npc.baseText, config, log)
	recordNoteSuccess(npc, rawResponse, promptTokens, completionTokens)
	log.Info("completed processing note", zap.Duration("duration", time.Since(npc.start)))
}

// noteProcessingContext holds shared state for note AI processing.
// This reduces parameter passing between helper functions.
type noteProcessingContext struct {
	ctx           context.Context
	correlationID string
	noteID        string
	baseText      string
	aiPrompt      string
	start         time.Time
	client        *canvusapi.Client
	config        *core.Config
	log           *logging.Logger
	repo          *db.Repository
	update        Update
}

// extractAIPrompt extracts the prompt text from a note, removing the {{ }} markers.
func extractAIPrompt(noteText string) string {
	return strings.ReplaceAll(strings.ReplaceAll(noteText, "{{", ""), "}}", "")
}

// hasAITrigger checks if the note text contains an AI trigger ({{ }}).
func hasAITrigger(text string) bool {
	return strings.Contains(text, "{{") && strings.Contains(text, "}}")
}

// processAIResponseType handles the AI response based on its type (text or image).
// Returns an error if response processing fails.
func processAIResponseType(npc *noteProcessingContext, response AINoteResponse) error {
	switch response.Type {
	case "text":
		updateNoteWithRetry(npc.client, npc.noteID, map[string]interface{}{
			"text": npc.baseText + "\n\n Generating text response...",
		}, npc.config, npc.log)
		npc.log.Info("creating text response")
		content := strings.ReplaceAll(response.Content, "\\n", "\n")
		return createNoteFromResponse(content, npc.noteID, npc.update, false, npc.client, npc.config, npc.log)

	case "image":
		updateNoteWithRetry(npc.client, npc.noteID, map[string]interface{}{
			"text": npc.baseText + "\n\n Generating image...\nThis may take up to 30 seconds.",
		}, npc.config, npc.log)
		npc.log.Info("creating image response")
		return processAIImage(npc.ctx, npc.client, response.Content, npc.update, npc.config, npc.log)

	default:
		npc.log.Error("unexpected AI response type", zap.String("response_type", response.Type))
		return fmt.Errorf("unexpected response type: %s", response.Type)
	}
}

// recordNoteSuccess records successful note processing to the database and updates metrics.
func recordNoteSuccess(npc *noteProcessingContext, rawResponse string, promptTokens, completionTokens int) {
	recordProcessingHistory(
		npc.ctx, npc.repo, npc.correlationID, npc.config.CanvasID, npc.noteID,
		"text_generation", npc.aiPrompt, rawResponse, npc.config.OpenAINoteModel,
		promptTokens, completionTokens, int(time.Since(npc.start).Milliseconds()),
		"success", "", npc.log,
	)
	atomic.AddInt64(&handlerMetrics.processedNotes, 1)
	metricsMutex.Lock()
	handlerMetrics.processingDuration += time.Since(npc.start)
	metricsMutex.Unlock()
}

// recordNoteError records failed note processing to the database.
func recordNoteError(npc *noteProcessingContext, err error) {
	recordProcessingHistory(
		npc.ctx, npc.repo, npc.correlationID, npc.config.CanvasID, npc.noteID,
		"text_generation", npc.aiPrompt, "", npc.config.OpenAINoteModel,
		len(npc.aiPrompt)/4, 0, int(time.Since(npc.start).Milliseconds()),
		"error", err.Error(), npc.log,
	)
	atomic.AddInt64(&handlerMetrics.errors, 1)
}

// handleNoteCreationError handles errors that occur during note response creation.
func handleNoteCreationError(npc *noteProcessingContext, err error) {
	npc.log.Error("failed to create response widget", zap.Error(err))
	atomic.AddInt64(&handlerMetrics.errors, 1)
	errorContent := fmt.Sprintf("# AI Image Generation Error\n\n Failed to generate image for your request.\n\n**Error Details:** %v", err)
	createNoteFromResponse(errorContent, npc.noteID, npc.update, true, npc.client, npc.config, npc.log)
	clearProcessingStatus(npc.client, npc.noteID, npc.baseText, npc.config, npc.log)
}

// updateNoteWithRetry attempts to update a note with retries
func updateNoteWithRetry(client *canvusapi.Client, noteID string, payload map[string]interface{}, config *core.Config, log *logging.Logger) error {
	for attempt := 1; attempt <= config.MaxRetries; attempt++ {
		log.Debug("attempting note update",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", config.MaxRetries))

		_, err := client.UpdateNote(noteID, payload)
		if err == nil {
			log.Debug("note updated successfully", zap.Int("attempt", attempt))
			return nil
		}

		// Enhanced error logging
		if apiErr, ok := err.(*canvusapi.APIError); ok {
			log.Warn("retry failed to update note",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", config.MaxRetries),
				zap.Int("status_code", apiErr.StatusCode),
				zap.String("error_message", apiErr.Message))
		} else {
			log.Warn("retry failed to update note",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", config.MaxRetries),
				zap.Error(err))
		}

		time.Sleep(config.RetryDelay)
	}
	return fmt.Errorf("failed to update Note: ID=%s after %d attempts", noteID, config.MaxRetries)
}

// generateAIResponse generates an AI response using OpenAI
func generateAIResponse(prompt string, config *core.Config, systemMessage string, log *logging.Logger) (string, error) {
	// Create client with configuration
	clientConfig := openai.DefaultConfig(config.OpenAIAPIKey)

	// Use TextLLMURL if set, otherwise fall back to BaseLLMURL
	if config.TextLLMURL != "" {
		clientConfig.BaseURL = config.TextLLMURL
	} else if config.BaseLLMURL != "" {
		clientConfig.BaseURL = config.BaseLLMURL
	}

	// Configure HTTP client with TLS settings
	clientConfig.HTTPClient = core.GetHTTPClient(config, config.AITimeout)
	client := openai.NewClientWithConfig(clientConfig)

	ctx := context.Background()

	// Make the system message more restrictive and direct
	enhancedSystemMessage := "You are a JSON-only response generator. " +
		"CRITICAL: Respond with ONLY valid JSON. No other text allowed. " +
		"No explanations. No XML tags. No thinking out loud. " +
		systemMessage + "\n" +
		"RESPONSE FORMAT:\n" +
		"For text: {\"type\": \"text\", \"content\": \"your response\"}\n" +
		"For image: {\"type\": \"image\", \"content\": \"your prompt\"}"

	log.Debug("making AI completion request",
		zap.String("model", config.OpenAINoteModel),
		zap.Int("max_tokens", int(config.NoteResponseTokens)))

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: config.OpenAINoteModel,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: enhancedSystemMessage,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens:   int(config.NoteResponseTokens),
			Temperature: 0.3,
		},
	)

	if err != nil {
		log.Error("AI completion request failed", zap.Error(err))
		return "", fmt.Errorf("error generating AI response: %w", err)
	}

	if len(resp.Choices) == 0 {
		log.Error("no response choices returned from AI")
		return "", fmt.Errorf("no response choices returned from AI")
	}

	// Get the raw response
	rawResponse := resp.Choices[0].Message.Content

	// Clean up the response - find the first { and last }
	startIdx := strings.Index(rawResponse, "{")
	endIdx := strings.LastIndex(rawResponse, "}")

	if startIdx == -1 || endIdx == -1 || startIdx > endIdx {
		log.Error("response does not contain valid JSON",
			zap.String("raw_response", truncateText(rawResponse, 100)))
		return "", fmt.Errorf("response does not contain valid JSON: %s", rawResponse)
	}

	// Extract just the JSON part
	jsonResponse := rawResponse[startIdx : endIdx+1]

	// Validate it's parseable JSON
	var testParse map[string]interface{}
	if err := json.Unmarshal([]byte(jsonResponse), &testParse); err != nil {
		log.Error("invalid JSON in response", zap.Error(err))
		return "", fmt.Errorf("invalid JSON in response: %w", err)
	}

	// Validate it has the required fields
	if _, ok := testParse["type"].(string); !ok {
		log.Error("response missing 'type' field")
		return "", fmt.Errorf("response missing 'type' field")
	}
	if _, ok := testParse["content"].(string); !ok {
		log.Error("response missing 'content' field")
		return "", fmt.Errorf("response missing 'content' field")
	}

	return jsonResponse, nil
}

// clearProcessingStatus removes processing indicator from note
func clearProcessingStatus(client *canvusapi.Client, noteID, processingText string, config *core.Config, log *logging.Logger) {
	clearedText := strings.ReplaceAll(processingText, "\n !! AI Processing !!", "")

	// First get the widget to determine its type
	widget, err := client.GetWidget(noteID, false)
	if err != nil {
		log.Warn("failed to get widget info", zap.Error(err))
		return
	}

	// Check widget type and use appropriate update method
	widgetType, ok := widget["widget_type"].(string)
	if !ok {
		log.Warn("failed to determine widget type")
		return
	}

	var updateErr error
	switch widgetType {
	case "Note":
		_, updateErr = client.UpdateNote(noteID, map[string]interface{}{
			"text": clearedText,
		})
	case "Image":
		_, updateErr = client.UpdateImage(noteID, map[string]interface{}{
			"title": clearedText,
		})
	default:
		updateErr = fmt.Errorf("unsupported widget type: %s", widgetType)
	}

	if updateErr != nil {
		log.Warn("failed to clear processing status", zap.Error(updateErr))
	}
}

// Helper function to get absolute location
func getAbsoluteLocation(client *canvusapi.Client, widget Update, config *core.Config, log *logging.Logger) (map[string]float64, error) {
	parentID, ok := widget["parent_id"].(string)
	if !ok {
		return nil, fmt.Errorf("no parent_id found")
	}

	// If parent is shared canvas, return widget location as-is
	if parentID == sharedCanvas.ID {
		widgetLoc := widget["location"].(map[string]interface{})
		return map[string]float64{
			"x": widgetLoc["x"].(float64),
			"y": widgetLoc["y"].(float64),
		}, nil
	}

	// Get parent widget
	parent, err := client.GetWidget(parentID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent widget: %w", err)
	}

	// Get parent location
	parentLoc, ok := parent["location"].(map[string]interface{})
	if !ok {
		// If parent has no location (shouldn't happen), use 0,0
		log.Warn("parent widget has no location", zap.String("parent_id", parentID))
		parentLoc = map[string]interface{}{
			"x": float64(0),
			"y": float64(0),
		}
	}

	// Get widget's relative location
	widgetLoc := widget["location"].(map[string]interface{})

	// Calculate absolute location
	absoluteLoc := map[string]float64{
		"x": parentLoc["x"].(float64) + widgetLoc["x"].(float64),
		"y": parentLoc["y"].(float64) + widgetLoc["y"].(float64),
	}

	// Log the calculation for debugging
	log.Debug("location calculation",
		zap.Float64("parent_x", parentLoc["x"].(float64)),
		zap.Float64("parent_y", parentLoc["y"].(float64)),
		zap.Float64("relative_x", widgetLoc["x"].(float64)),
		zap.Float64("relative_y", widgetLoc["y"].(float64)),
		zap.Float64("absolute_x", absoluteLoc["x"]),
		zap.Float64("absolute_y", absoluteLoc["y"]))

	return absoluteLoc, nil
}

// createNoteFromResponse creates a new Note widget based on the AI response.
func createNoteFromResponse(content, triggeringNoteID string, triggeringUpdate Update, errorNote bool, client *canvusapi.Client, config *core.Config, log *logging.Logger) error {
	log.Debug("creating note from response",
		zap.String("content_preview", truncateText(content, 30)),
		zap.Bool("is_error_note", errorNote))

	// Extract original properties using atoms
	originalSize := handlers.ExtractSize(triggeringUpdate["size"].(map[string]interface{}))
	originalScale := triggeringUpdate["scale"].(float64)

	// Calculate size and scale based on content
	var noteSize handlers.NoteSize
	var scale float64
	if errorNote {
		noteSize = originalSize
		scale = originalScale
	} else {
		noteSize, scale = handlers.CalculateNoteSize(content, originalSize.Width, originalSize.Height, originalScale)
		log.Debug("content sizing calculated",
			zap.Float64("width", noteSize.Width),
			zap.Float64("height", noteSize.Height),
			zap.Float64("scale", scale))
	}

	// Calculate background color and depth using atoms
	bgColor, _ := triggeringUpdate["background_color"].(string)
	backgroundColor := handlers.ReduceBackgroundOpacity(bgColor)
	originalDepth, _ := triggeringUpdate["depth"].(float64)
	depth := handlers.CalculateDepthOffset(originalDepth, 200)

	// Build and send creation payload
	payload := buildResponseNotePayload(content, triggeringNoteID, triggeringUpdate, noteSize, scale, backgroundColor, depth)
	logNoteCreation(log, triggeringUpdate, noteSize, scale)

	if _, err := client.CreateNote(payload); err != nil {
		log.Error("failed to create note", zap.Error(err))
		return err
	}
	return nil
}

// buildResponseNotePayload constructs the payload for creating a response note.
func buildResponseNotePayload(content, triggeringNoteID string, triggeringUpdate Update, size handlers.NoteSize, scale float64, backgroundColor string, depth float64) map[string]interface{} {
	return map[string]interface{}{
		"title":            fmt.Sprintf("Response to Note %s", triggeringNoteID),
		"text":             content,
		"location":         triggeringUpdate["location"],
		"size":             handlers.SizeToMap(size),
		"depth":            depth,
		"scale":            scale,
		"background_color": backgroundColor,
		"auto_text_color":  false,
		"text_color":       "#000000ff",
	}
}

// logNoteCreation logs details about the note being created.
func logNoteCreation(log *logging.Logger, triggeringUpdate Update, size handlers.NoteSize, scale float64) {
	loc := handlers.ExtractLocation(triggeringUpdate["location"].(map[string]interface{}))
	log.Info("creating response note",
		zap.Float64("x", loc.X),
		zap.Float64("y", loc.Y),
		zap.Float64("width", size.Width),
		zap.Float64("height", size.Height),
		zap.Float64("scale", scale))
}

// isAzureOpenAIEndpoint checks if the endpoint is an Azure OpenAI endpoint
func isAzureOpenAIEndpoint(endpoint string) bool {
	return strings.Contains(strings.ToLower(endpoint), "openai.azure.com") ||
		strings.Contains(strings.ToLower(endpoint), "cognitiveservices.azure.com")
}

// processAIImage generates and uploads an image from the AI's response using imagegen package
func processAIImage(ctx context.Context, client *canvusapi.Client, prompt string, update Update, config *core.Config, log *logging.Logger) error {
	downloadsMutex.Lock()
	defer downloadsMutex.Unlock()

	log.Info("generating AI image via imagegen",
		zap.String("prompt_preview", truncateText(prompt, 50)))

	// Check for local endpoint (not supported for cloud image generation)
	if imagegen.IsLocalEndpoint(config.ImageLLMURL) || imagegen.IsLocalEndpoint(config.BaseLLMURL) {
		// For local endpoints, fall back to the original implementation
		// since imagegen.Generator is for cloud providers only
		return processAIImageFallback(ctx, client, prompt, update, config, log)
	}

	// Create the generator using the convenience constructor
	generator, err := imagegen.NewGeneratorFromConfig(config, client, log)
	if err != nil {
		log.Error("failed to create image generator", zap.Error(err))
		return fmt.Errorf("failed to create image generator: %w", err)
	}

	// Convert Update map to ParentWidget interface
	parentWidget := updateToParentWidget(update)

	// Generate the image using the imagegen package
	result, err := generator.Generate(ctx, prompt, parentWidget)
	if err != nil {
		log.Error("image generation failed", zap.Error(err))
		return err
	}

	log.Info("image generation completed",
		zap.String("widget_id", result.WidgetID))

	return nil
}

// updateToParentWidget converts a handler Update map to an imagegen.ParentWidget
func updateToParentWidget(update Update) imagegen.ParentWidget {
	loc := update["location"].(map[string]interface{})
	size := update["size"].(map[string]interface{})

	return imagegen.CanvasWidget{
		ID: update["id"].(string),
		Location: imagegen.WidgetLocation{
			X: loc["x"].(float64),
			Y: loc["y"].(float64),
		},
		Size: imagegen.WidgetSize{
			Width:  size["width"].(float64),
			Height: size["height"].(float64),
		},
		Scale: update["scale"].(float64),
		Depth: update["depth"].(float64),
	}
}

// processAIImageFallback is the original implementation for local endpoints
func processAIImageFallback(ctx context.Context, client *canvusapi.Client, prompt string, update Update, config *core.Config, log *logging.Logger) error {
	// Ensure downloads directory exists
	if err := os.MkdirAll(config.DownloadsDir, 0755); err != nil {
		return fmt.Errorf("failed to create downloads directory: %w", err)
	}

	// Determine the endpoint to use
	var endpoint string
	var isAzure bool

	if config.ImageLLMURL != "" {
		endpoint = config.ImageLLMURL
		log.Debug("using image-specific API URL", zap.String("endpoint", endpoint))
	} else if config.AzureOpenAIEndpoint != "" {
		endpoint = config.AzureOpenAIEndpoint
		isAzure = true
		log.Debug("using Azure OpenAI endpoint", zap.String("endpoint", endpoint))
	} else if config.BaseLLMURL != "" {
		endpoint = config.BaseLLMURL
		log.Debug("using base LLM URL for image generation", zap.String("endpoint", endpoint))
	} else {
		endpoint = "https://api.openai.com/v1"
		log.Debug("using default OpenAI endpoint", zap.String("endpoint", endpoint))
	}

	// Check if this is an Azure endpoint
	if !isAzure {
		isAzure = isAzureOpenAIEndpoint(endpoint)
	}

	// Generate the image using the appropriate API
	if isAzure {
		return processAIImageAzure(ctx, client, prompt, update, config, endpoint, log)
	} else {
		return processAIImageOpenAI(ctx, client, prompt, update, config, endpoint, log)
	}
}

// processAIImageOpenAI generates images using standard OpenAI API
func processAIImageOpenAI(ctx context.Context, client *canvusapi.Client, prompt string, update Update, config *core.Config, endpoint string, log *logging.Logger) error {
	// Generate the image using the configured API endpoint
	imageConfig := openai.DefaultConfig(config.OpenAIAPIKey)
	imageConfig.BaseURL = endpoint

	// Check if the endpoint supports image generation
	if strings.Contains(strings.ToLower(endpoint), "127.0.0.1") ||
		strings.Contains(strings.ToLower(endpoint), "localhost") {
		return fmt.Errorf("image generation is not supported by the local API endpoint (%s). "+
			"Please configure IMAGE_LLM_URL to use a service that supports image generation",
			endpoint)
	}

	// Configure HTTP client with TLS settings
	imageConfig.HTTPClient = core.GetHTTPClient(config, config.AITimeout)
	aiClient := openai.NewClientWithConfig(imageConfig)

	// Use the configured image model
	model := config.OpenAIImageModel
	if model == "" {
		model = "dall-e-3" // Default fallback
	}

	log.Info("creating image with OpenAI",
		zap.String("model", model),
		zap.String("prompt_preview", truncateText(prompt, 50)))

	// Create image request with model-specific parameters
	imageReq := openai.ImageRequest{
		Prompt:         prompt,
		Model:          model,
		ResponseFormat: openai.CreateImageResponseFormatURL,
		N:              1,
	}

	// Only add style parameter for DALL-E 3 (not supported by DALL-E 2)
	if model == "dall-e-3" {
		imageReq.Style = openai.CreateImageStyleVivid
	}

	image, err := aiClient.CreateImage(ctx, imageReq)
	if err != nil {
		log.Error("failed to generate AI image", zap.Error(err))
		return fmt.Errorf("failed to generate AI image: %w", err)
	}

	// Validate image response
	if image.Data == nil {
		log.Error("API returned nil Data field")
		return fmt.Errorf("no image data returned from API")
	}

	if len(image.Data) == 0 {
		log.Error("API returned empty Data array")
		return fmt.Errorf("no image data returned from API")
	}

	if image.Data[0].URL == "" {
		log.Error("API returned empty URL")
		return fmt.Errorf("no image URL returned from API")
	}

	log.Debug("successfully received image URL from API")

	// Download the image
	req, err := http.NewRequestWithContext(ctx, "GET", image.Data[0].URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	httpClient := core.GetDefaultHTTPClient(config)
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download AI image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	// Save the image locally
	imagePath := filepath.Join(config.DownloadsDir, fmt.Sprintf("ai_image_%s.jpg", update["id"].(string)))
	file, err := os.Create(imagePath)
	if err != nil {
		return fmt.Errorf("failed to create image file: %w", err)
	}
	defer func() {
		file.Close()
		os.Remove(imagePath) // Clean up after upload
	}()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("failed to write image data: %w", err)
	}

	// Calculate position for new image
	width := update["size"].(map[string]interface{})["width"].(float64)
	height := update["size"].(map[string]interface{})["height"].(float64)
	x := update["location"].(map[string]interface{})["x"].(float64) + (width * 0.8)
	y := update["location"].(map[string]interface{})["y"].(float64) + (height * 0.8)

	// Create the image widget
	payload := map[string]interface{}{
		"title": fmt.Sprintf("AI Generated Image for %s", update["id"].(string)),
		"location": map[string]float64{
			"x": x,
			"y": y,
		},
		"size":  update["size"],
		"depth": update["depth"].(float64) + 10,
		"scale": update["scale"].(float64) / 3,
	}

	log.Debug("creating image widget",
		zap.Float64("x", x),
		zap.Float64("y", y))

	_, err = client.CreateImage(imagePath, payload)
	if err != nil {
		return fmt.Errorf("failed to create image: %w", err)
	}

	return nil
}

// processAIImageAzure generates images using Azure OpenAI API
func processAIImageAzure(ctx context.Context, client *canvusapi.Client, prompt string, update Update, config *core.Config, endpoint string, log *logging.Logger) error {
	// Validate Azure configuration
	if config.AzureOpenAIDeployment == "" {
		return fmt.Errorf("Azure OpenAI deployment name is required but not configured. Please set AZURE_OPENAI_DEPLOYMENT")
	}

	log.Info("using Azure OpenAI deployment",
		zap.String("deployment", config.AzureOpenAIDeployment))

	// Create Azure-specific client configuration
	imageConfig := openai.DefaultConfig(config.OpenAIAPIKey)
	imageConfig.BaseURL = endpoint

	// Configure HTTP client with TLS settings
	imageConfig.HTTPClient = core.GetHTTPClient(config, config.AITimeout)

	// Azure OpenAI uses different authentication - we'll handle this in the request
	aiClient := openai.NewClientWithConfig(imageConfig)

	// For Azure, we need to use the deployment name as the model
	// Map Azure deployment names to appropriate parameters
	model := config.AzureOpenAIDeployment

	log.Info("creating Azure OpenAI image",
		zap.String("model", model),
		zap.String("prompt_preview", truncateText(prompt, 50)))

	// Create image request with Azure-specific parameters
	imageReq := openai.ImageRequest{
		Prompt:         prompt,
		Model:          model,
		ResponseFormat: openai.CreateImageResponseFormatURL,
		N:              1,
	}

	// Azure OpenAI models may have different parameter support
	// Only add style for dalle3 deployment (not gpt-image-1)
	if strings.Contains(strings.ToLower(model), "dalle3") || strings.Contains(strings.ToLower(model), "dall-e") {
		imageReq.Style = openai.CreateImageStyleVivid
		log.Debug("added style parameter for DALL-E model")
	}

	image, err := aiClient.CreateImage(ctx, imageReq)
	if err != nil {
		log.Error("failed to generate Azure AI image", zap.Error(err))
		return fmt.Errorf("failed to generate Azure AI image: %w", err)
	}

	// Validate image response (same as OpenAI)
	if image.Data == nil {
		log.Error("Azure API returned nil Data field")
		return fmt.Errorf("no image data returned from Azure API")
	}

	if len(image.Data) == 0 {
		log.Error("Azure API returned empty Data array")
		return fmt.Errorf("no image data returned from Azure API")
	}

	if image.Data[0].URL == "" {
		log.Error("Azure API returned empty URL")
		return fmt.Errorf("no image URL returned from Azure API")
	}

	log.Debug("successfully received image URL from Azure API")

	// Download the image (same as OpenAI)
	req, err := http.NewRequestWithContext(ctx, "GET", image.Data[0].URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	httpClient := core.GetDefaultHTTPClient(config)
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download Azure AI image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download Azure image: status %d", resp.StatusCode)
	}

	// Save the image locally (same as OpenAI)
	imagePath := filepath.Join(config.DownloadsDir, fmt.Sprintf("ai_image_%s.jpg", update["id"].(string)))
	file, err := os.Create(imagePath)
	if err != nil {
		return fmt.Errorf("failed to create image file: %w", err)
	}
	defer func() {
		file.Close()
		os.Remove(imagePath) // Clean up after upload
	}()

	if _, err := io.Copy(file, resp.Body); err != nil {
		return fmt.Errorf("failed to write image data: %w", err)
	}

	// Calculate position for new image (same as OpenAI)
	width := update["size"].(map[string]interface{})["width"].(float64)
	height := update["size"].(map[string]interface{})["height"].(float64)
	x := update["location"].(map[string]interface{})["x"].(float64) + (width * 0.8)
	y := update["location"].(map[string]interface{})["y"].(float64) + (height * 0.8)

	// Create the image widget (same as OpenAI)
	payload := map[string]interface{}{
		"title": fmt.Sprintf("AI Generated Image (Azure) for %s", update["id"].(string)),
		"location": map[string]float64{
			"x": x,
			"y": y,
		},
		"size":  update["size"],
		"depth": update["depth"].(float64) + 10,
		"scale": update["scale"].(float64) / 3,
	}

	log.Debug("creating Azure image widget",
		zap.Float64("x", x),
		zap.Float64("y", y))

	_, err = client.CreateImage(imagePath, payload)
	if err != nil {
		return fmt.Errorf("failed to create Azure image: %w", err)
	}

	return nil
}

// handleSnapshot processes Snapshot widgets for handwriting recognition
func handleSnapshot(update Update, client *canvusapi.Client, config *core.Config, logger *logging.Logger, repo *db.Repository) {
	correlationID := generateCorrelationID()
	imageID := update["id"].(string)

	// Create a logger with widget context
	log := logger.With(
		zap.String("correlation_id", correlationID),
		zap.String("widget_id", imageID),
		zap.String("widget_type", "Snapshot"),
	)

	start := time.Now()
	downloadsMutex.Lock()
	defer downloadsMutex.Unlock()

	// Log trigger widget details
	triggerLoc := update["location"].(map[string]interface{})
	triggerSize := update["size"].(map[string]interface{})
	log.Info("processing snapshot",
		zap.Float64("x", triggerLoc["x"].(float64)),
		zap.Float64("y", triggerLoc["y"].(float64)),
		zap.Float64("width", triggerSize["width"].(float64)),
		zap.Float64("height", triggerSize["height"].(float64)))

	// Create processing note
	processingNoteID, err := createProcessingNote(client, update, config, log)
	if err != nil {
		log.Error("failed to create processing note", zap.Error(err))
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return // Keep snapshot
	}

	// Create context for the operation
	ctx, cancel := context.WithTimeout(context.Background(), config.ProcessingTimeout)
	defer cancel()

	// Download the snapshot with retries
	downloadPath := filepath.Join(
		config.DownloadsDir,
		fmt.Sprintf("temp_snapshot_%s.jpg", imageID),
	)

	const maxRetries = 3
	var downloadErr error
	var fileInfo os.FileInfo

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Debug("download attempt",
			zap.Int("attempt", attempt),
			zap.Int("max_retries", maxRetries))

		if attempt > 1 {
			// Update note with countdown
			for countdown := 3; countdown > 0; countdown-- {
				updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
					"text": fmt.Sprintf("Download failed. Retrying in %d...", countdown),
				}, config, log)
				time.Sleep(time.Second)
			}
		}

		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": fmt.Sprintf("Downloading snapshot... (Attempt %d/%d)", attempt, maxRetries),
		}, config, log)

		// Try to download
		log.Debug("attempting to download image")
		downloadErr = client.DownloadImage(imageID, downloadPath)
		if downloadErr != nil {
			log.Warn("download attempt failed",
				zap.Int("attempt", attempt),
				zap.Error(downloadErr))
			continue
		}

		// Verify the downloaded file
		fileInfo, err = os.Stat(downloadPath)
		if err != nil {
			log.Warn("file verification failed after download",
				zap.Int("attempt", attempt),
				zap.Error(err))
			downloadErr = fmt.Errorf("file verification failed: %w", err)
			continue
		}

		if fileInfo.Size() == 0 {
			log.Warn("downloaded file is empty", zap.Int("attempt", attempt))
			downloadErr = fmt.Errorf("downloaded file is empty")
			continue
		}

		log.Debug("download successful",
			zap.Int("attempt", attempt),
			zap.Int64("file_size", fileInfo.Size()))
		downloadErr = nil
		break
	}

	// If all download attempts failed
	if downloadErr != nil {
		log.Error("all download attempts failed", zap.Error(downloadErr))
		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": " Failed to download image after multiple attempts.\nClick the snapshot again to retry.",
		}, config, log)
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return // Keep snapshot
	}

	// Read image data
	imageData, err := os.ReadFile(downloadPath)
	if err != nil {
		log.Error("failed to read image data", zap.Error(err))
		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": " Failed to read image data.\nClick the snapshot again to retry.",
		}, config, log)
		atomic.AddInt64(&handlerMetrics.errors, 1)
		os.Remove(downloadPath)
		return // Keep snapshot
	}

	log.Debug("successfully read image data",
		zap.Int("size_bytes", len(imageData)))

	updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
		"text": "Processing image through OCR... Please wait.",
	}, config, log)

	// Perform OCR
	ocrText, err := performGoogleVisionOCR(ctx, imageData, config, log)
	if err != nil {
		// Record failed OCR processing to database
		recordProcessingHistory(
			ctx,
			repo,
			correlationID,
			config.CanvasID,
			imageID,
			"ocr_processing",
			"Snapshot OCR",
			"",
			"google_vision",
			0,
			0,
			int(time.Since(start).Milliseconds()),
			"error",
			err.Error(),
			log,
		)
		log.Error("failed to perform OCR", zap.Error(err))
		errorMessage := " Failed to process image.\n\n"
		if strings.Contains(err.Error(), "no text found") {
			errorMessage += "No readable text was found in the image."
		} else {
			errorMessage += fmt.Sprintf("Error: %v", err)
		}
		errorMessage += "\n\nClick the snapshot again to retry."

		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": errorMessage,
		}, config, log)
		atomic.AddInt64(&handlerMetrics.errors, 1)
		os.Remove(downloadPath)
		return // Keep snapshot
	}

	updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
		"text": "Creating response note...",
	}, config, log)

	// Create response note
	if err := createNoteFromResponse(ocrText, imageID, update, false, client, config, log); err != nil {
		log.Error("failed to create response note", zap.Error(err))
		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": " Failed to create response note.\nClick the snapshot again to retry.",
		}, config, log)
		atomic.AddInt64(&handlerMetrics.errors, 1)
		os.Remove(downloadPath)
		return // Keep snapshot
	}

	// Only cleanup if everything succeeded
	log.Debug("OCR process completed successfully, cleaning up resources")

	// Clean up the downloaded file
	if err := os.Remove(downloadPath); err != nil {
		log.Warn("failed to remove downloaded file", zap.Error(err))
	}

	// Delete the processing note
	if err := deleteTriggeringWidget(client, "note", processingNoteID, log); err != nil {
		log.Warn("failed to delete processing note", zap.Error(err))
	}

	// Only delete the snapshot after complete success
	if err := deleteTriggeringWidget(client, "image", imageID, log); err != nil {
		log.Warn("failed to delete snapshot", zap.Error(err))
	}


	// Record successful OCR processing to database
	recordProcessingHistory(
		ctx,
		repo,
		correlationID,
		config.CanvasID,
		imageID,
		"ocr_processing",
		"Snapshot OCR",
		ocrText,
		"google_vision",
		0, // OCR doesn't report input tokens
		len(ocrText)/4, // Estimate output tokens
		int(time.Since(start).Milliseconds()),
		"success",
		"",
		log,
	)
	// Update metrics
	atomic.AddInt64(&handlerMetrics.processedImages, 1)
	metricsMutex.Lock()
	handlerMetrics.processingDuration += time.Since(start)
	metricsMutex.Unlock()

	log.Info("completed snapshot processing",
		zap.Duration("duration", time.Since(start)))
}

// getPDFChunkPrompt returns the system message for PDF chunk analysis
func getPDFChunkPrompt() string {
	return `You are analyzing a section of a document. Focus on:
1. Main ideas and key points
2. Important details and evidence
3. Connections to other sections
4. Technical accuracy and academic tone
Format your response as: {"type": "text", "content": "your analysis"}`
}

// handlePDFPrecis generates a summary of a PDF widget using pdfprocessor package
func handlePDFPrecis(update Update, client *canvusapi.Client, config *core.Config, logger *logging.Logger, repo *db.Repository) {
	correlationID := generateCorrelationID()
	parentID, _ := update["parent_id"].(string)

	// Create a logger with widget context
	log := logger.With(
		zap.String("correlation_id", correlationID),
		zap.String("widget_id", update["id"].(string)),
		zap.String("parent_id", parentID),
		zap.String("widget_type", "PDF"),
	)

	start := time.Now()

	parentWidget, err := client.GetWidget(parentID, false)
	if err != nil {
		log.Error("failed to get parent PDF widget", zap.Error(err))
		return
	}

	pdfLoc := parentWidget["location"].(map[string]interface{})
	pdfSize := parentWidget["size"].(map[string]interface{})
	log.Info("processing PDF precis",
		zap.Float64("x", pdfLoc["x"].(float64)),
		zap.Float64("y", pdfLoc["y"].(float64)),
		zap.Float64("width", pdfSize["width"].(float64)),
		zap.Float64("height", pdfSize["height"].(float64)),
		zap.Float64("scale", parentWidget["scale"].(float64)))

	widgetType, _ := parentWidget["widget_type"].(string)
	if strings.ToLower(widgetType) != "pdf" {
		log.Error("invalid widget type for PDF precis",
			zap.String("expected", "PDF"),
			zap.String("actual", widgetType))
		return
	}

	triggerWidget := make(Update)
	for k, v := range parentWidget {
		triggerWidget[k] = v
	}

	// Create processing note for UI feedback
	processingNote := map[string]interface{}{
		"title": "PDF Analysis",
		"text":  " Starting PDF analysis...",
		"location": map[string]interface{}{
			"x": triggerWidget["location"].(map[string]interface{})["x"].(float64) + 100.0,
			"y": triggerWidget["location"].(map[string]interface{})["y"].(float64) + 100.0,
		},
		"size": map[string]interface{}{
			"width":  triggerWidget["size"].(map[string]interface{})["width"].(float64) * 0.5,
			"height": triggerWidget["size"].(map[string]interface{})["height"].(float64) * 0.5,
		},
		"depth":            triggerWidget["depth"].(float64) + 200,
		"scale":            triggerWidget["scale"].(float64) * 1.5,
		"background_color": "#FFFFFF",
		"pinned":           true,
	}

	noteResp, err := client.CreateNote(processingNote)
	if err != nil {
		log.Error("failed to create processing note", zap.Error(err))
		return
	}
	processingNoteID := noteResp["id"].(string)

	ctx, cancel := context.WithTimeout(context.Background(), config.ProcessingTimeout)
	defer cancel()

	// Download PDF from Canvus
	downloadPath := filepath.Join(config.DownloadsDir, fmt.Sprintf("temp_pdf_%s.pdf", parentID))
	updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
		"text": " Downloading PDF...",
	}, config, log)

	if err := client.DownloadPDF(parentID, downloadPath); err != nil {
		log.Error("failed to download PDF", zap.Error(err))
		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": " Failed to download PDF",
		}, config, log)
		return
	}
	defer os.Remove(downloadPath)

	// Configure and create pdfprocessor
	processorConfig := pdfprocessor.DefaultProcessorConfig()
	processorConfig.ChunkerConfig.MaxChunkTokens = int(config.PDFChunkSizeTokens)
	processorConfig.SummarizerConfig.Model = config.OpenAIPDFModel
	processorConfig.SummarizerConfig.MaxTokens = int(config.NoteResponseTokens)

	// Create OpenAI client for the processor
	clientConfig := openai.DefaultConfig(config.OpenAIAPIKey)
	if config.TextLLMURL != "" {
		clientConfig.BaseURL = config.TextLLMURL
	} else if config.BaseLLMURL != "" {
		clientConfig.BaseURL = config.BaseLLMURL
	}
	clientConfig.HTTPClient = core.GetHTTPClient(config, config.AITimeout)
	aiClient := openai.NewClientWithConfig(clientConfig)

	// Create processor with progress callback for UI updates
	processor := pdfprocessor.NewProcessorWithProgress(processorConfig, aiClient, func(stage string, progress float64, message string) {
		// Update the processing note based on stage
		statusText := fmt.Sprintf(" %s\n%.0f%% complete", message, progress*100)
		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": statusText,
		}, config, log)
	})

	// Create metrics logger for inference tracking
	metricsLogger := logging.NewMetricsLogger(log)
	inferenceTimer := metricsLogger.StartInference(config.OpenAIPDFModel)

	// Process the PDF using the pdfprocessor package
	result, err := processor.Process(ctx, downloadPath)

	if err != nil {
		metricsLogger.EndInference(inferenceTimer, 0, 0)
		// Record failed PDF processing to database
		extractedText := ""
		if result != nil && result.ExtractionResult != nil {
			extractedText = result.ExtractionResult.Text
		}
		recordProcessingHistory(
			ctx,
			repo,
			correlationID,
			config.CanvasID,
			update["id"].(string),
			"pdf_analysis",
			extractedText,
			"",
			config.OpenAIPDFModel,
			len(extractedText)/4,
			0,
			int(time.Since(start).Milliseconds()),
			"error",
			err.Error(),
			log,
		)
		log.Error("PDF analysis failed", zap.Error(err))
		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": fmt.Sprintf(" Failed to analyze PDF: %v", err),
		}, config, log)
		return
	}

	// Get token counts from result
	promptTokens := 0
	completionTokens := 0
	if result.SummaryResult != nil {
		promptTokens = result.SummaryResult.PromptTokens
		completionTokens = result.SummaryResult.CompletionTokens
	}
	metricsLogger.EndInference(inferenceTimer, promptTokens, completionTokens)

	// Create summary note with the result
	err = createNoteFromResponse(result.Summary, parentID, triggerWidget, false, client, config, log)
	if err != nil {
		log.Error("failed to create summary note", zap.Error(err))
		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": " Failed to create summary note",
		}, config, log)
		return
	}

	// Clean up trigger widgets
	deleteTriggeringWidget(client, "note", processingNoteID, log)
	deleteTriggeringWidget(client, "image", update["id"].(string), log)

	// Get extracted text for history
	extractedText := ""
	if result.ExtractionResult != nil {
		extractedText = result.ExtractionResult.Text
	}

	// Record successful PDF processing to database
	recordProcessingHistory(
		ctx,
		repo,
		correlationID,
		config.CanvasID,
		update["id"].(string),
		"pdf_analysis",
		extractedText,
		result.Summary,
		config.OpenAIPDFModel,
		promptTokens,
		completionTokens,
		int(time.Since(start).Milliseconds()),
		"success",
		"",
		log,
	)
	atomic.AddInt64(&handlerMetrics.processedPDFs, 1)
	metricsMutex.Lock()
	handlerMetrics.processingDuration += time.Since(start)
	metricsMutex.Unlock()

	log.Info("completed PDF precis",
		zap.Duration("duration", time.Since(start)),
		zap.Duration("processing_time", result.ProcessingTime),
		zap.Int("chunks_processed", result.ChunkerResult.TotalChunks))
}

// extractPDFText extracts text content from a PDF file
func extractPDFText(pdfPath string, log *logging.Logger) (string, error) {
	// Open PDF file
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var textBuilder strings.Builder
	totalPages := r.NumPage()

	log.Debug("extracting text from PDF",
		zap.Int("total_pages", totalPages))

	// Extract text from each page
	for pageIndex := 1; pageIndex <= totalPages; pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue // Skip empty pages
		}

		text, err := p.GetPlainText(nil)
		if err != nil {
			log.Warn("failed to extract text from page",
				zap.Int("page", pageIndex),
				zap.Error(err))
			continue // Skip problematic pages but continue processing
		}
		textBuilder.WriteString(text)
		textBuilder.WriteString("\n\n")
	}

	extractedText := textBuilder.String()
	if extractedText == "" {
		return "", fmt.Errorf("no text content found in PDF")
	}

	return extractedText, nil
}

func splitIntoChunks(text string, maxChunkSize int) []string {
	var chunks []string
	paragraphs := strings.Split(text, "\n\n")

	var currentChunk strings.Builder
	currentSize := 0

	for _, para := range paragraphs {
		paraSize := len(para)

		if currentSize+paraSize > maxChunkSize {
			if currentChunk.Len() > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
				currentSize = 0
			}
		}

		currentChunk.WriteString(para)
		currentChunk.WriteString("\n\n")
		currentSize += paraSize + 2
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// estimateTokenCount provides a rough estimate of tokens in a text
// Using average of 4 characters per token as a rough approximation
func estimateTokenCount(text string) int {
	return len(text) / 4
}

// handleCanvusPrecis processes Canvus widget summaries
func handleCanvusPrecis(update Update, client *canvusapi.Client, config *core.Config, logger *logging.Logger, repo *db.Repository) {
	correlationID := generateCorrelationID()
	canvasID := update["id"].(string)

	// Create a logger with widget context
	log := logger.With(
		zap.String("correlation_id", correlationID),
		zap.String("widget_id", canvasID),
		zap.String("widget_type", "CanvasPrecis"),
	)

	start := time.Now()

	// Create a canvas-specific config that uses the canvas model
	canvasConfig := *config                                       // Create a copy of the config
	canvasConfig.OpenAINoteModel = canvasConfig.OpenAICanvasModel // Use canvas model for all AI calls

	// Validate update
	if err := handlers.ValidateUpdate(update); err != nil {
		log.Error("invalid Canvus precis update", zap.Error(err))
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.ProcessingTimeout)
	defer cancel()

	// Update title to show processing
	processingTitle := "!! AI Processing !! " + update["title"].(string)
	_, err := client.UpdateImage(canvasID, map[string]interface{}{
		"title": processingTitle,
	})
	if err != nil {
		log.Error("failed to update Canvus precis title", zap.Error(err))
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return
	}

	// Fetch all widgets from the canvas
	widgets, err := fetchCanvasWidgets(ctx, client, config, log)
	if err != nil {
		log.Error("failed to fetch widgets for Canvus precis", zap.Error(err))
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return
	}

	log.Info("fetched canvas widgets",
		zap.Int("widget_count", len(widgets)))

	// Generate and process the precis
	if err := processCanvusPrecis(ctx, client, update, widgets, config, log, repo, correlationID, start); err != nil {
		log.Error("failed to process Canvus precis", zap.Error(err))
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return
	}

	// Update metrics
	metricsMutex.Lock()
	handlerMetrics.processingDuration += time.Since(start)
	metricsMutex.Unlock()

	// Cleanup original widget
	if err := deleteTriggeringWidget(client, update["widget_type"].(string), canvasID, log); err != nil {
		log.Warn("failed to delete triggering Canvus precis widget", zap.Error(err))
	}

	log.Info("completed canvas precis",
		zap.Duration("duration", time.Since(start)))
}

// fetchCanvasWidgets retrieves all widgets using canvasanalyzer.Fetcher with retry logic
func fetchCanvasWidgets(ctx context.Context, client *canvusapi.Client, config *core.Config, log *logging.Logger) ([]map[string]interface{}, error) {
	// Create fetcher with configuration
	fetcherConfig := canvasanalyzer.DefaultFetcherConfig()
	fetcherConfig.MaxRetries = config.MaxRetries
	fetcherConfig.RetryDelay = config.RetryDelay

	fetcher := canvasanalyzer.NewFetcher(client, fetcherConfig, log.Zap())

	// Fetch widgets
	result, err := fetcher.Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch widgets: %w", err)
	}

	// Convert canvasanalyzer.Widget slice back to []map[string]interface{}
	widgets := make([]map[string]interface{}, len(result.Widgets))
	for i, w := range result.Widgets {
		widgets[i] = map[string]interface{}(w)
	}

	log.Info("widgets fetched successfully",
		zap.Int("total", result.TotalCount),
		zap.Int("filtered", result.FilteredCount),
		zap.Int("attempts", result.Attempts),
		zap.Duration("duration", result.Duration))

	return widgets, nil
}

// processCanvusPrecis generates and creates a summary of the canvas
func processCanvusPrecis(ctx context.Context, client *canvusapi.Client, update Update, widgets []map[string]interface{}, config *core.Config, log *logging.Logger, repo *db.Repository, correlationID string, start time.Time) error {
	log.Info("starting Canvus Precis processing")

	// Create a canvas-specific config that uses the canvas model
	canvasConfig := *config                                       // Create a copy of the config
	canvasConfig.OpenAINoteModel = canvasConfig.OpenAICanvasModel // Use canvas model for all AI calls

	// Log trigger widget details
	triggerLoc := update["location"].(map[string]interface{})
	triggerSize := update["size"].(map[string]interface{})
	log.Debug("trigger widget details",
		zap.Float64("x", triggerLoc["x"].(float64)),
		zap.Float64("y", triggerLoc["y"].(float64)),
		zap.Float64("width", triggerSize["width"].(float64)),
		zap.Float64("height", triggerSize["height"].(float64)),
		zap.Float64("scale", update["scale"].(float64)))

	// Get icon location and add offset for processing note
	iconLoc := update["location"].(map[string]interface{})
	processingNoteLoc := map[string]interface{}{
		"x": iconLoc["x"].(float64) + 100.0,
		"y": iconLoc["y"].(float64) + 100.0,
	}

	// Create processing note first
	processingNote := map[string]interface{}{
		"title":    processingNoteTitle,
		"text":     " Analyzing canvas content...",
		"location": processingNoteLoc,
		"size": map[string]interface{}{
			"width":  400.0,
			"height": 200.0,
		},
		"depth":            update["depth"].(float64) + 200,
		"scale":            update["scale"].(float64),
		"background_color": processingNoteColor,
		"text_color":       processingNoteTextColor,
		"auto_text_color":  false,
		"pinned":           true,
	}

	noteResp, err := client.CreateNote(processingNote)
	if err != nil {
		return fmt.Errorf("failed to create processing note: %w", err)
	}
	processingNoteID := noteResp["id"].(string)

	// Filter out the triggering icon from the widgets list
	var filteredWidgets []map[string]interface{}
	for _, widget := range widgets {
		if id, ok := widget["id"].(string); ok && id != update["id"].(string) {
			filteredWidgets = append(filteredWidgets, widget)
		}
	}

	// Update processing status
	updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
		"text": " Analyzing canvas content...\nProcessing " + strconv.Itoa(len(filteredWidgets)) + " widgets",
	}, config, log)

	// Convert filtered widgets to JSON for AI processing
	widgetsJSON, err := json.Marshal(filteredWidgets)
	if err != nil {
		deleteTriggeringWidget(client, "note", processingNoteID, log)
		return fmt.Errorf("failed to marshal widgets data: %w", err)
	}

	// Configure system message for canvas analysis
	systemMessage := `You are an assistant analyzing a collaborative workspace.
	Describe the content and relationships between items in a natural, narrative way.
	Focus on the story the workspace is telling and how items relate to each other.
	Avoid mentioning technical details like IDs or coordinates.
	Format your response as text using markdown with three sections:
	# Overview
	Describe the main themes and content of the workspace.
	# Insights
	Share observations about relationships between items and suggest next steps.
	# Recommendations
	Provide actionable recommendations for improving the workspace.`

	// Update processing status
	updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
		"text": " Generating canvas analysis...\nThis may take a moment.",
	}, config, log)

	// Create metrics logger for inference tracking
	metricsLogger := logging.NewMetricsLogger(log)
	inferenceTimer := metricsLogger.StartInference(canvasConfig.OpenAINoteModel)

	// Generate AI response
	rawResponse, err := generateAIResponse(string(widgetsJSON), &canvasConfig, systemMessage, log)
	if err != nil {
		metricsLogger.EndInference(inferenceTimer, 0, 0)
		// Record failed canvas analysis to database
		recordProcessingHistory(
			ctx,
			repo,
			correlationID,
			config.CanvasID,
			update["id"].(string),
			"canvas_analysis",
			string(widgetsJSON),
			"",
			canvasConfig.OpenAINoteModel,
			len(widgetsJSON)/4,
			0,
			int(time.Since(start).Milliseconds()),
			"error",
			err.Error(),
			log,
		)
		deleteTriggeringWidget(client, "note", processingNoteID, log)
		return handleAIError(ctx, client, update, fmt.Errorf("AI generation failed: %w", err), update["text"].(string), config, log)
	}

	// Estimate tokens for logging
	promptTokens := len(widgetsJSON) / 4
	completionTokens := len(rawResponse) / 4
	metricsLogger.EndInference(inferenceTimer, promptTokens, completionTokens)

	// Parse the AI response JSON and extract the content field
	var aiResponse map[string]interface{}
	if err := json.Unmarshal([]byte(rawResponse), &aiResponse); err != nil {
		deleteTriggeringWidget(client, "note", processingNoteID, log)
		return fmt.Errorf("failed to parse AI response JSON: %w", err)
	}
	content, ok := aiResponse["content"].(string)
	if !ok {
		deleteTriggeringWidget(client, "note", processingNoteID, log)
		return fmt.Errorf("AI response missing 'content' field")
	}
	// Convert escaped newlines to actual newlines
	content = strings.ReplaceAll(content, "\\n", "\n")

	// Create response note
	err = createNoteFromResponse(content, update["id"].(string), update, false, client, config, log)
	if err != nil {
		deleteTriggeringWidget(client, "note", processingNoteID, log)
		return fmt.Errorf("failed to create response note: %w", err)
	}

	// Clean up processing note
	deleteTriggeringWidget(client, "note", processingNoteID, log)


	// Record successful canvas analysis to database
	recordProcessingHistory(
		ctx,
		repo,
		correlationID,
		config.CanvasID,
		update["id"].(string),
		"canvas_analysis",
		string(widgetsJSON),
		content,
		config.OpenAINoteModel,
		promptTokens,
		completionTokens,
		int(time.Since(start).Milliseconds()),
		"success",
		"",
		log,
	)
	return nil
}

// deleteTriggeringWidget safely deletes a widget by type and ID
func deleteTriggeringWidget(client *canvusapi.Client, widgetType, widgetID string, log *logging.Logger) error {
	log.Debug("deleting triggering widget",
		zap.String("widget_type", widgetType),
		zap.String("widget_id", widgetID))

	// Normalize the widget type to lowercase for comparison
	var err error
	switch strings.ToLower(widgetType) {
	case "note":
		err = client.DeleteNote(widgetID)
	case "image":
		err = client.DeleteImage(widgetID)
	case "pdf":
		err = client.DeletePDF(widgetID)
	case "widget":
		// For generic widgets, try deleting as a note
		err = client.DeleteNote(widgetID)
	default:
		return fmt.Errorf("unsupported widget type: %s", widgetType)
	}

	if err != nil {
		return fmt.Errorf("failed to delete %s widget %s: %w", widgetType, widgetID, err)
	}
	return nil
}

// handleAIError creates a friendly error note, clears processing text, and logs the error
func handleAIError(ctx context.Context, client *canvusapi.Client, update Update, err error, baseText string, config *core.Config, log *logging.Logger) error {
	log.Error("AI processing error", zap.Error(err))

	errorMessage := `# AI Processing Error

I apologize, but I encountered an error while processing your request.

**What happened**: The AI system returned an invalid or unexpected response.

**What you can do**:
- Try your request again
- If the problem persists, try rephrasing your request
- Contact support if the issue continues

*Technical details: %v*`

	errorContent := fmt.Sprintf(errorMessage, err)

	// Create error note using fixed size and positioning
	errResp := createNoteFromResponse(errorContent, update["id"].(string), update, true, client, config, log)

	// Clear the extra processing text from the original note
	clearProcessingStatus(client, update["id"].(string), baseText, config, log)

	return errResp
}

func chunkPDFContent(content []byte, maxTokens int) []string {
	// Split on paragraph boundaries first
	paragraphs := strings.Split(string(content), "\n\n")

	var chunks []string
	currentChunk := strings.Builder{}
	currentTokens := 0

	for _, para := range paragraphs {
		paraTokens := estimateTokenCount(para)
		if currentTokens+paraTokens > maxTokens {
			// Store current chunk and start new one
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentTokens = 0
		}
		currentChunk.WriteString(para + "\n\n")
		currentTokens += paraTokens
	}

	// Add final chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// handleAIIcon processes AI icon updates

// Helper function to create processing notes (reduces duplication)
func createProcessingNote(client *canvusapi.Client, update Update, config *core.Config, log *logging.Logger) (string, error) {
	absoluteLoc, err := getAbsoluteLocation(client, update, config, log)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute location: %w", err)
	}

	processingNote := map[string]interface{}{
		"title":            processingNoteTitle,
		"text":             "Downloading Snapshot...",
		"location":         absoluteLoc,
		"size":             map[string]interface{}{"width": 300.0, "height": 300.0},
		"depth":            update["depth"].(float64) + 10,
		"scale":            update["scale"].(float64),
		"background_color": processingNoteColor,
		"text_color":       processingNoteTextColor,
		"auto_text_color":  false,
		"pinned":           true,
	}

	noteResp, err := client.CreateNote(processingNote)
	if err != nil {
		return "", fmt.Errorf("failed to create processing note: %w", err)
	}
	return noteResp["id"].(string), nil
}

// performGoogleVisionOCR performs OCR using ocrprocessor package
func performGoogleVisionOCR(ctx context.Context, imageData []byte, config *core.Config, log *logging.Logger) (string, error) {
	log.Info("starting Google Vision OCR process via ocrprocessor")

	apiKey := config.GoogleVisionKey
	if apiKey == "" {
		return "", fmt.Errorf("Google Vision API key not found in configuration")
	}

	// Create HTTP client with TLS settings
	httpClient := core.GetDefaultHTTPClient(config)

	// Create OCR processor
	processor, err := ocrprocessor.NewProcessor(apiKey, httpClient, log, ocrprocessor.DefaultProcessorConfig())
	if err != nil {
		log.Error("failed to create OCR processor", zap.Error(err))
		return "", fmt.Errorf("failed to create OCR processor: %w", err)
	}

	// Process the image using ocrprocessor
	result, err := processor.ProcessImage(ctx, imageData)
	if err != nil {
		log.Error("OCR processing failed", zap.Error(err))
		return "", fmt.Errorf("OCR processing failed: %w", err)
	}

	if result.Text == "" {
		return "", fmt.Errorf("no text found in image")
	}

	log.Info("OCR completed successfully",
		zap.Int("text_length", len(result.Text)),
		zap.Duration("processing_time", result.ProcessingTime),
		zap.Duration("vision_api_time", result.VisionAPITime))

	return result.Text, nil
}

// validateGoogleAPIKey makes a minimal API call to verify the key works
func validateGoogleAPIKey(ctx context.Context, apiKey string, config *core.Config, log *logging.Logger) error {
	// Create minimal request with a 1x1 pixel transparent PNG
	minimalImage := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII="

	requestBody := GoogleVisionRequest{
		Requests: []struct {
			Image struct {
				Content string `json:"content"`
			} `json:"image"`
			Features []struct {
				Type       string `json:"type"`
				MaxResults int    `json:"maxResults"`
			} `json:"features"`
		}{
			{
				Image: struct {
					Content string `json:"content"`
				}{
					Content: minimalImage,
				},
				Features: []struct {
					Type       string `json:"type"`
					MaxResults int    `json:"maxResults"`
				}{
					{
						Type:       "TEXT_DETECTION",
						MaxResults: 1,
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	url := fmt.Sprintf("%s?key=%s", visionAPIEndpoint, apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := core.GetHTTPClient(config, 10*time.Second)
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("API key validation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Debug("API key validation successful")
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("API key validation failed: status=%d, body=%s", resp.StatusCode, string(body))
}
