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
	"go_backend/llamaruntime"
	"go_backend/logging"
	"go_backend/metrics"
	"go_backend/ocrprocessor"
	"go_backend/pdfprocessor"

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

// HandlerDependencies holds dependencies injected into handler functions.
// This eliminates global state and enables proper dependency injection.
type HandlerDependencies struct {
	// Metrics integration for dashboard
	metricsStore    metrics.MetricsCollector
	taskBroadcaster metrics.TaskBroadcaster
	metricsMux      sync.RWMutex

	// Handler-level metrics tracking
	handlerMetrics struct {
		processedNotes     int64
		processedImages    int64
		processedPDFs      int64
		errors             int64
		processingDuration time.Duration
	}
	metricsMutex sync.Mutex

	// Serialization for image operations (prevents overwhelming downloads dir or API)
	downloadsMutex sync.Mutex
}

// NewHandlerDependencies creates a new HandlerDependencies with optional metrics.
func NewHandlerDependencies(store metrics.MetricsCollector, broadcaster metrics.TaskBroadcaster) *HandlerDependencies {
	return &HandlerDependencies{
		metricsStore:    store,
		taskBroadcaster: broadcaster,
	}
}

// SetMetrics updates the metrics store and broadcaster.
// This is called from main.go when wiring the dashboard.
func (d *HandlerDependencies) SetMetrics(store metrics.MetricsCollector, broadcaster metrics.TaskBroadcaster) {
	d.metricsMux.Lock()
	defer d.metricsMux.Unlock()
	d.metricsStore = store
	d.taskBroadcaster = broadcaster
}

// GetMetrics returns the current metrics store and broadcaster.
func (d *HandlerDependencies) GetMetrics() (metrics.MetricsCollector, metrics.TaskBroadcaster) {
	d.metricsMux.RLock()
	defer d.metricsMux.RUnlock()
	return d.metricsStore, d.taskBroadcaster
}

// recordTaskStart records that a handler task has started processing.
// Returns a TaskRecord that should be passed to recordTaskComplete.
func (d *HandlerDependencies) recordTaskStart(taskID, taskType, canvasID string) metrics.TaskRecord {
	record := metrics.TaskRecord{
		ID:        taskID,
		Type:      taskType,
		CanvasID:  canvasID,
		Status:    metrics.TaskStatusProcessing,
		StartTime: time.Now(),
	}

	// Broadcast the "processing" status
	_, broadcaster := d.GetMetrics()
	if broadcaster != nil {
		broadcaster.BroadcastTaskUpdateFromMetrics(metrics.TaskBroadcastData{
			TaskID:   record.ID,
			TaskType: record.Type,
			Status:   record.Status,
			CanvasID: record.CanvasID,
		})
	}

	return record
}

// recordTaskComplete records that a handler task has completed.
// If errMsg is non-empty, the task is marked as failed; otherwise successful.
func (d *HandlerDependencies) recordTaskComplete(record metrics.TaskRecord, errMsg string) {
	record.EndTime = time.Now()
	record.Duration = record.EndTime.Sub(record.StartTime)

	if errMsg != "" {
		record.Status = metrics.TaskStatusError
		record.ErrorMsg = errMsg
	} else {
		record.Status = metrics.TaskStatusSuccess
	}

	store, broadcaster := d.GetMetrics()

	// Record to metrics store
	if store != nil {
		store.RecordTask(record)
	}

	// Broadcast the completion status
	if broadcaster != nil {
		broadcaster.BroadcastTaskUpdateFromMetrics(metrics.TaskBroadcastData{
			TaskID:   record.ID,
			TaskType: record.Type,
			Status:   record.Status,
			CanvasID: record.CanvasID,
			Duration: record.Duration,
			Error:    record.ErrorMsg,
		})
	}
}

// recordMetrics updates handler-level metrics (processed counts, duration).
func (d *HandlerDependencies) recordMetrics(processType string, duration time.Duration) {
	switch processType {
	case "note":
		atomic.AddInt64(&d.handlerMetrics.processedNotes, 1)
	case "image":
		atomic.AddInt64(&d.handlerMetrics.processedImages, 1)
	case "pdf":
		atomic.AddInt64(&d.handlerMetrics.processedPDFs, 1)
	case "error":
		atomic.AddInt64(&d.handlerMetrics.errors, 1)
	}

	d.metricsMutex.Lock()
	d.handlerMetrics.processingDuration += duration
	d.metricsMutex.Unlock()
}

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

// generateCorrelationID creates a unique ID for request tracing (delegated to handlers package)
func generateCorrelationID() string {
	return handlers.GenerateCorrelationID()
}

// truncateText truncates a text to a specified length (delegated to handlers package)
func truncateText(text string, length int) string {
	return handlers.TruncateText(text, length)
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

// handleNote processes Note widget updates.
// If llamaClient is provided, it uses local inference; otherwise falls back to cloud API.
//
// Atomic design: Organism (orchestrates AI inference, Canvus API, and response creation)
func handleNote(update Update, client *canvusapi.Client, config *core.Config, logger *logging.Logger, repo *db.Repository, llamaClient *llamaruntime.Client, deps *HandlerDependencies) {
	noteID, _ := update["id"].(string)
	log := logger.With(
		zap.String("correlation_id", generateCorrelationID()),
		zap.String("widget_id", noteID),
		zap.String("widget_type", "Note"),
	)

	ctx := context.Background()
	start := time.Now()

	// Record task start for dashboard metrics
	taskRecord := deps.recordTaskStart(noteID, metrics.TaskTypeNote, config.CanvasID)

	// Create noteProcessingContext to reduce parameter passing
	npc := &noteProcessingContext{
		ctx:           ctx,
		client:        client,
		config:        config,
		log:           log,
		repo:          repo,
		llamaClient:   llamaClient,
		deps:          deps,
		update:        update,
		noteID:        noteID,
		correlationID: log.GetCorrelationID(),
		start:         start,
		taskRecord:    taskRecord,
	}

	// Get the note text
	noteText, ok := update["text"].(string)
	if !ok || noteText == "" {
		log.Warn("note text missing or empty")
		deps.recordTaskComplete(taskRecord, "note text missing")
		return
	}

	// Detect AI prompt (supports both {{ }} and {{image:}} formats)
	aiPrompt := handlers.ExtractAIPrompt(noteText)
	if aiPrompt == "" {
		log.Debug("no AI trigger found in note")
		deps.recordTaskComplete(taskRecord, "no AI trigger")
		return
	}

	npc.aiPrompt = aiPrompt
	log.Info("processing AI note",
		zap.String("prompt_preview", truncateText(aiPrompt, 100)))

	// Check for {{image:}} directive first
	if strings.HasPrefix(strings.ToLower(aiPrompt), "image:") {
		// Extract the actual image prompt
		imagePrompt := strings.TrimSpace(strings.TrimPrefix(strings.ToLower(aiPrompt), "image:"))
		imagePrompt = strings.TrimSpace(strings.TrimPrefix(aiPrompt, aiPrompt[:len("image:")]))
		log.Info("direct image generation request detected",
			zap.String("image_prompt", truncateText(imagePrompt, 100)))

		// Process as image directly
		if err := processAIImage(ctx, client, imagePrompt, update, config, log, deps); err != nil {
			log.Error("image generation failed", zap.Error(err))
			recordNoteError(npc, err)
			return
		}

		// Record success
		recordNoteSuccess(npc)
		return
	}

	// If no image directive, use AI to classify and respond
	processNoteWithAI(npc)
}

// noteProcessingContext holds all context needed for note processing.
// This reduces parameter passing and makes the code more maintainable.
type noteProcessingContext struct {
	ctx           context.Context
	client        *canvusapi.Client
	config        *core.Config
	log           *logging.Logger
	repo          *db.Repository
	llamaClient   *llamaruntime.Client
	deps          *HandlerDependencies
	update        Update
	noteID        string
	correlationID string
	aiPrompt      string
	start         time.Time
	taskRecord    metrics.TaskRecord
}

// processNoteWithAI uses AI to classify the prompt and generate appropriate response.
func processNoteWithAI(npc *noteProcessingContext) {
	// Use AI to determine if this is a text or image request
	aiResp, err := classifyNoteIntent(npc)
	if err != nil {
		npc.log.Error("AI classification failed", zap.Error(err))
		recordNoteError(npc, err)
		return
	}

	// Process based on AI response type
	switch aiResp.Type {
	case "text":
		if err := createAITextNote(npc, aiResp.Content); err != nil {
			npc.log.Error("text note creation failed", zap.Error(err))
			recordNoteError(npc, err)
			return
		}
	case "image":
		if err := processAIImage(npc.ctx, npc.client, aiResp.Content, npc.update, npc.config, npc.log, npc.deps); err != nil {
			npc.log.Error("image generation failed", zap.Error(err))
			recordNoteError(npc, err)
			return
		}
	default:
		err := fmt.Errorf("unknown AI response type: %s", aiResp.Type)
		npc.log.Error("invalid AI response", zap.Error(err))
		recordNoteError(npc, err)
		return
	}

	// Record success
	recordNoteSuccess(npc)
}

// classifyNoteIntent uses AI to determine if the prompt is for text or image generation.
func classifyNoteIntent(npc *noteProcessingContext) (*AINoteResponse, error) {
	// Prepare the AI request with the system message
	messages := []openai.ChatCompletionMessage{
		{Role: "system", Content: noteSystemMessage},
		{Role: "user", Content: npc.aiPrompt},
	}

	// Use local inference if available, otherwise cloud API
	var responseText string
	var err error

	if npc.llamaClient != nil {
		npc.log.Info("using local LLM for intent classification")
		responseText, err = npc.llamaClient.Generate(npc.ctx, npc.aiPrompt, llamaruntime.GenerationParams{
			MaxTokens:   500,
			Temperature: 0.7,
			SystemPrompt: &noteSystemMessage,
		})
	} else {
		npc.log.Info("using cloud API for intent classification")
		aiClient := core.CreateOpenAIClient(npc.config)
		resp, apiErr := aiClient.CreateChatCompletion(npc.ctx, openai.ChatCompletionRequest{
			Model:       npc.config.OpenAINoteModel,
			Messages:    messages,
			MaxTokens:   500,
			Temperature: 0.7,
		})
		if apiErr != nil {
			return nil, fmt.Errorf("OpenAI API error: %w", apiErr)
		}
		if len(resp.Choices) == 0 {
			return nil, fmt.Errorf("no response from AI")
		}
		responseText = resp.Choices[0].Message.Content
	}

	if err != nil {
		return nil, fmt.Errorf("AI generation error: %w", err)
	}

	npc.log.Debug("AI classification response",
		zap.String("response", truncateText(responseText, 200)))

	// Parse the JSON response
	var aiResp AINoteResponse
	if err := json.Unmarshal([]byte(responseText), &aiResp); err != nil {
		// If JSON parsing fails, treat it as a text response
		npc.log.Warn("failed to parse AI response as JSON, treating as text",
			zap.Error(err),
			zap.String("response", truncateText(responseText, 200)))
		return &AINoteResponse{
			Type:    "text",
			Content: responseText,
		}, nil
	}

	return &aiResp, nil
}

// createAITextNote creates a note widget with the AI-generated text response.
func createAITextNote(npc *noteProcessingContext, content string) error {
	location := npc.update["location"].(map[string]interface{})
	size := npc.update["size"].(map[string]interface{})

	// Calculate position for the response note (to the right of the trigger)
	newLocation := handlers.CalculateNoteLocation(location, size, npc.config.NoteSpacing)

	note := canvusapi.CreateNoteRequest{
		Location: canvusapi.WidgetLocation{
			X: newLocation["x"].(float64),
			Y: newLocation["y"].(float64),
		},
		Size: canvusapi.WidgetSize{
			Width:  npc.config.NoteWidth,
			Height: npc.config.NoteHeight,
		},
		BackgroundColor: npc.config.NoteColor,
		TextColor:       npc.config.NoteTextColor,
		Text:            content,
	}

	result, err := npc.client.CreateNote(note)
	if err != nil {
		return fmt.Errorf("failed to create note: %w", err)
	}

	npc.log.Info("AI note created",
		zap.String("note_id", result.ID),
		zap.Int("content_length", len(content)))

	return nil
}

// recordNoteSuccess records successful note processing to database and metrics.
func recordNoteSuccess(npc *noteProcessingContext) {
	duration := time.Since(npc.start)
	recordProcessingHistory(
		npc.ctx, npc.repo, npc.correlationID, npc.config.CanvasID, npc.noteID,
		"text_generation", npc.aiPrompt, "", npc.config.OpenAINoteModel,
		0, 0, int(duration.Milliseconds()),
		"success", "", npc.log,
	)
	// Update metrics
	npc.deps.recordMetrics("note", duration)
	npc.deps.recordTaskComplete(npc.taskRecord, "") // Empty string = success
}

// recordNoteError records failed note processing to the database and dashboard metrics.
func recordNoteError(npc *noteProcessingContext, err error) {
	recordProcessingHistory(
		npc.ctx, npc.repo, npc.correlationID, npc.config.CanvasID, npc.noteID,
		"text_generation", npc.aiPrompt, "", npc.config.OpenAINoteModel,
		0, 0, int(time.Since(npc.start).Milliseconds()),
		"error", err.Error(), npc.log,
	)
	npc.deps.recordMetrics("error", time.Since(npc.start))
	npc.deps.recordTaskComplete(npc.taskRecord, err.Error())

	// Try to notify the user via error note
	if notifyErr := handleAIError(npc.ctx, npc.client, npc.update, err, "", npc.config, npc.log); notifyErr != nil {
		npc.log.Error("failed to create error note", zap.Error(notifyErr))
	}
}

// handleNoteCreationError handles errors during note creation by attempting to create an error note.
func handleNoteCreationError(npc *noteProcessingContext, err error) {
	npc.log.Error("note creation failed", zap.Error(err))
	recordNoteError(npc, err)
}

// isAzureOpenAIEndpoint checks if the endpoint is an Azure OpenAI endpoint (delegated to handlers package)
func isAzureOpenAIEndpoint(endpoint string) bool {
	return handlers.IsAzureOpenAIEndpoint(endpoint)
}

// processAIImage generates and uploads an image from the AI's response using imagegen package
func processAIImage(ctx context.Context, client *canvusapi.Client, prompt string, update Update, config *core.Config, log *logging.Logger, deps *HandlerDependencies) error {
	deps.downloadsMutex.Lock()
	defer deps.downloadsMutex.Unlock()

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

	// Create HTTP client with proper TLS configuration
	imageConfig.HTTPClient = core.GetHTTPClient(config.AllowSelfSignedCerts)

	imageClient := openai.NewClientWithConfig(imageConfig)

	resp, err := imageClient.CreateImage(ctx, openai.ImageRequest{
		Prompt:         prompt,
		N:              1,
		Size:           openai.CreateImageSize1024x1024,
		ResponseFormat: openai.CreateImageResponseFormatURL,
	})

	if err != nil {
		log.Error("OpenAI image generation failed", zap.Error(err))
		return fmt.Errorf("failed to generate image: %w", err)
	}

	if len(resp.Data) == 0 {
		return fmt.Errorf("no image data in OpenAI response")
	}

	imageURL := resp.Data[0].URL
	log.Info("image generated via OpenAI",
		zap.String("url", imageURL),
		zap.String("prompt_preview", truncateText(prompt, 50)))

	// Download and upload the image
	return downloadAndUploadImage(ctx, client, imageURL, update, config, log)
}

// processAIImageAzure generates images using Azure OpenAI API
func processAIImageAzure(ctx context.Context, client *canvusapi.Client, prompt string, update Update, config *core.Config, endpoint string, log *logging.Logger) error {
	if config.AzureOpenAIDeployment == "" {
		return fmt.Errorf("Azure OpenAI deployment name not configured")
	}

	// Construct Azure-specific URL
	azureURL := fmt.Sprintf("%s/openai/deployments/%s/images/generations?api-version=%s",
		strings.TrimSuffix(endpoint, "/"),
		config.AzureOpenAIDeployment,
		config.AzureOpenAIAPIVersion)

	// Create the request body
	reqBody := map[string]interface{}{
		"prompt": prompt,
		"n":      1,
		"size":   "1024x1024",
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal Azure request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", azureURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create Azure request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", config.OpenAIAPIKey)

	// Use configured HTTP client with proper TLS settings
	httpClient := core.GetHTTPClient(config.AllowSelfSignedCerts)
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Azure request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Azure API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse Azure response
	var azureResp struct {
		Data []struct {
			URL string `json:"url"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&azureResp); err != nil {
		return fmt.Errorf("failed to decode Azure response: %w", err)
	}

	if len(azureResp.Data) == 0 {
		return fmt.Errorf("no image data in Azure response")
	}

	imageURL := azureResp.Data[0].URL
	log.Info("image generated via Azure OpenAI",
		zap.String("url", imageURL),
		zap.String("prompt_preview", truncateText(prompt, 50)))

	// Download and upload the image
	return downloadAndUploadImage(ctx, client, imageURL, update, config, log)
}

// downloadAndUploadImage downloads an image from a URL and uploads it to the canvas
func downloadAndUploadImage(ctx context.Context, client *canvusapi.Client, imageURL string, update Update, config *core.Config, log *logging.Logger) error {
	// Download the image
	httpClient := core.GetHTTPClient(config.AllowSelfSignedCerts)
	resp, err := httpClient.Get(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("image download failed with status: %d", resp.StatusCode)
	}

	// Save to temporary file
	tempFile := filepath.Join(config.DownloadsDir, fmt.Sprintf("ai_image_%s.png", generateCorrelationID()))
	outFile, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer outFile.Close()
	defer os.Remove(tempFile) // Clean up after upload

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}
	outFile.Close()

	log.Info("image downloaded",
		zap.String("file", tempFile))

	// Calculate position for the image (below and to the right of the trigger note)
	location := update["location"].(map[string]interface{})
	size := update["size"].(map[string]interface{})

	// Use imagegen placement calculation
	placement := imagegen.CalculatePlacement(
		imagegen.WidgetLocation{
			X: location["x"].(float64),
			Y: location["y"].(float64),
		},
		imagegen.WidgetSize{
			Width:  size["width"].(float64),
			Height: size["height"].(float64),
		},
		imagegen.ImageSize{Width: 1024, Height: 1024},
		update["scale"].(float64),
	)

	// Upload the image to the canvas
	uploadReq := canvusapi.UploadImageRequest{
		FilePath: tempFile,
		Location: canvusapi.WidgetLocation{
			X: placement.X,
			Y: placement.Y,
		},
		Size: canvusapi.WidgetSize{
			Width:  placement.Width,
			Height: placement.Height,
		},
	}

	result, err := client.UploadImage(uploadReq)
	if err != nil {
		return fmt.Errorf("failed to upload image to canvas: %w", err)
	}

	log.Info("image uploaded to canvas",
		zap.String("image_id", result.ID),
		zap.Float64("x", placement.X),
		zap.Float64("y", placement.Y))

	return nil
}

// handleSnapshot processes Snapshot (handwriting recognition) widget updates.
// This handler downloads the snapshot image, sends it to Google Vision API for OCR,
// and creates a note with the recognized text.
//
// Atomic design: Organism (orchestrates OCR API, Canvus API, and note creation)
func handleSnapshot(update Update, client *canvusapi.Client, config *core.Config, logger *logging.Logger, repo *db.Repository, deps *HandlerDependencies) {
	snapshotID, _ := update["id"].(string)
	correlationID := generateCorrelationID()
	log := logger.With(
		zap.String("correlation_id", correlationID),
		zap.String("widget_id", snapshotID),
		zap.String("widget_type", "Snapshot"),
	)

	ctx := context.Background()
	start := time.Now()

	// Record task start for dashboard metrics
	taskRecord := deps.recordTaskStart(correlationID, metrics.TaskTypeHandwriting, config.CanvasID)

	deps.downloadsMutex.Lock()
	defer deps.downloadsMutex.Unlock()

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
		deps.recordTaskComplete(taskRecord, "failed to create processing note")
		return
	}

	// Get the snapshot URL
	snapshotURL, ok := update["snapshotUrl"].(string)
	if !ok || snapshotURL == "" {
		log.Error("snapshot URL missing")
		updateProcessingNote(client, processingNoteID, "❌ Error: No snapshot URL", config, log)
		deps.recordTaskComplete(taskRecord, "snapshot URL missing")
		return
	}

	log.Info("snapshot URL retrieved",
		zap.String("url", snapshotURL))

	// Create OCR processor
	ocrProc, err := ocrprocessor.NewProcessor(
		config.GoogleVisionAPIKey,
		core.GetHTTPClient(config.AllowSelfSignedCerts),
		logger,
		ocrprocessor.DefaultProcessorConfig(),
	)
	if err != nil {
		errMsg := fmt.Sprintf("❌ OCR Error: %v", err)
		log.Error("failed to create OCR processor", zap.Error(err))
		updateProcessingNote(client, processingNoteID, errMsg, config, log)
		deps.recordTaskComplete(taskRecord, err.Error())
		return
	}

	// Process the snapshot with OCR
	recognizedText, err := ocrProc.ProcessURL(ctx, snapshotURL)
	if err != nil {
		errMsg := fmt.Sprintf("❌ OCR Error: %v", err)
		log.Error("OCR processing failed", zap.Error(err))
		updateProcessingNote(client, processingNoteID, errMsg, config, log)
		recordProcessingHistory(
			ctx, repo, correlationID, config.CanvasID, snapshotID,
			"handwriting_recognition", snapshotURL, "", "google-vision",
			0, 0, int(time.Since(start).Milliseconds()),
			"error", err.Error(), log,
		)
		deps.recordTaskComplete(taskRecord, err.Error())
		return
	}

	if recognizedText == "" {
		log.Warn("no text recognized in snapshot")
		updateProcessingNote(client, processingNoteID, "⚠️ No text recognized", config, log)
		recordProcessingHistory(
			ctx, repo, correlationID, config.CanvasID, snapshotID,
			"handwriting_recognition", snapshotURL, "", "google-vision",
			0, 0, int(time.Since(start).Milliseconds()),
			"success", "no text detected", log,
		)
		deps.recordTaskComplete(taskRecord, "no text recognized")
		return
	}

	log.Info("text recognized",
		zap.Int("length", len(recognizedText)),
		zap.String("preview", truncateText(recognizedText, 100)))

	// Update the processing note with the recognized text
	updateProcessingNote(client, processingNoteID, recognizedText, config, log)

	// Record success to database
	recordProcessingHistory(
		ctx, repo, correlationID, config.CanvasID, snapshotID,
		"handwriting_recognition", snapshotURL, truncateText(recognizedText, 1000), "google-vision",
		0, len(recognizedText), int(time.Since(start).Milliseconds()),
		"success", "", log,
	)
	// Update metrics
	deps.recordMetrics("image", time.Since(start))
	deps.recordTaskComplete(taskRecord, "") // Empty string = success

	log.Info("completed snapshot processing",
		zap.Duration("duration", time.Since(start)))
}

// handleImageAnalysis analyzes an image widget using llamaruntime.InferVision.
// This handler is triggered when a user places an AI_Icon_Image_Analysis widget on an image.
// It downloads the image, runs vision inference, and creates a note with the description.
//
// Atomic design: Organism (orchestrates vision inference, Canvus API, and note creation)
func handleImageAnalysis(update Update, client *canvusapi.Client, config *core.Config, logger *logging.Logger, repo *db.Repository, llamaClient *llamaruntime.Client, deps *HandlerDependencies) {
	triggerID, _ := update["id"].(string)
	correlationID := generateCorrelationID()
	log := logger.With(
		zap.String("correlation_id", correlationID),
		zap.String("widget_id", triggerID),
		zap.String("widget_type", "AI_Icon_Image_Analysis"),
	)

	ctx := context.Background()
	start := time.Now()

	// Record task start for dashboard metrics
	taskRecord := deps.recordTaskStart(correlationID, metrics.TaskTypeImageAnalysis, config.CanvasID)

	deps.downloadsMutex.Lock()
	defer deps.downloadsMutex.Unlock()

	// Get the parent widget (the image to analyze)
	parentID := update["parentId"].(string)
	if parentID == "" {
		log.Error("no parent image to analyze")
		deps.recordTaskComplete(taskRecord, "no parent image")
		return
	}

	parentWidget, err := client.GetWidget(parentID, false)
	if err != nil {
		log.Error("failed to get parent widget", zap.Error(err))
		deps.recordTaskComplete(taskRecord, fmt.Sprintf("failed to get parent widget: %v", err))
		return
	}

	// Verify parent is an image
	widgetType, _ := parentWidget["type"].(string)
	if widgetType != "Image" {
		log.Error("parent widget is not an image",
			zap.String("parent_type", widgetType))
		deps.recordTaskComplete(taskRecord, "parent is not an image")
		return
	}

	imageURL, ok := parentWidget["url"].(string)
	if !ok || imageURL == "" {
		log.Error("parent image has no URL")
		deps.recordTaskComplete(taskRecord, "parent image has no URL")
		return
	}

	log.Info("analyzing image",
		zap.String("image_url", imageURL),
		zap.String("parent_id", parentID))

	// Create processing note
	processingNoteID, err := createProcessingNote(client, update, config, log)
	if err != nil {
		log.Error("failed to create processing note", zap.Error(err))
		deps.recordTaskComplete(taskRecord, "failed to create processing note")
		return
	}

	// Check if llamaClient is available
	if llamaClient == nil {
		errMsg := "Vision analysis not available (llama runtime not initialized)"
		log.Error(errMsg)
		updateProcessingNote(client, processingNoteID, fmt.Sprintf("❌ %s", errMsg), config, log)
		deps.recordTaskComplete(taskRecord, errMsg)
		return
	}

	// Download the image to a temporary file
	httpClient := core.GetHTTPClient(config.AllowSelfSignedCerts)
	resp, err := httpClient.Get(imageURL)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to download image: %v", err)
		log.Error("image download failed", zap.Error(err))
		updateProcessingNote(client, processingNoteID, fmt.Sprintf("❌ %s", errMsg), config, log)
		deps.recordTaskComplete(taskRecord, errMsg)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("Image download failed with status: %d", resp.StatusCode)
		log.Error("image download failed", zap.Int("status", resp.StatusCode))
		updateProcessingNote(client, processingNoteID, fmt.Sprintf("❌ %s", errMsg), config, log)
		deps.recordTaskComplete(taskRecord, errMsg)
		return
	}

	// Save to temporary file
	tempFile := filepath.Join(config.DownloadsDir, fmt.Sprintf("image_analysis_%s.jpg", correlationID))
	outFile, err := os.Create(tempFile)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create temp file: %v", err)
		log.Error("temp file creation failed", zap.Error(err))
		updateProcessingNote(client, processingNoteID, fmt.Sprintf("❌ %s", errMsg), config, log)
		deps.recordTaskComplete(taskRecord, errMsg)
		return
	}

	_, err = io.Copy(outFile, resp.Body)
	outFile.Close()
	if err != nil {
		os.Remove(tempFile)
		errMsg := fmt.Sprintf("Failed to save image: %v", err)
		log.Error("image save failed", zap.Error(err))
		updateProcessingNote(client, processingNoteID, fmt.Sprintf("❌ %s", errMsg), config, log)
		deps.recordTaskComplete(taskRecord, errMsg)
		return
	}
	defer os.Remove(tempFile) // Clean up after analysis

	log.Info("image downloaded for analysis",
		zap.String("temp_file", tempFile))

	// Run vision inference
	prompt := "Describe this image in detail."
	description, err := llamaClient.InferVision(ctx, tempFile, prompt, llamaruntime.VisionParams{
		MaxTokens:   500,
		Temperature: 0.7,
	})
	if err != nil {
		errMsg := fmt.Sprintf("Vision inference failed: %v", err)
		log.Error("vision inference failed", zap.Error(err))
		updateProcessingNote(client, processingNoteID, fmt.Sprintf("❌ %s", errMsg), config, log)
		recordProcessingHistory(
			ctx, repo, correlationID, config.CanvasID, triggerID,
			"image_analysis", prompt, "", config.VisionModel,
			0, 0, int(time.Since(start).Milliseconds()),
			"error", err.Error(), log,
		)
		deps.recordTaskComplete(taskRecord, errMsg)
		return
	}

	log.Info("image analysis complete",
		zap.Int("description_length", len(description)))

	// Update the processing note with the description
	updateProcessingNote(client, processingNoteID, description, config, log)

	// Record success to database
	recordProcessingHistory(
		ctx, repo, correlationID, config.CanvasID, triggerID,
		"image_analysis", prompt, truncateText(description, 1000), config.VisionModel,
		0, len(description), int(time.Since(start).Milliseconds()),
		"success", "", log,
	)

	// Update metrics
	deps.recordMetrics("image", time.Since(start))
	deps.recordTaskComplete(taskRecord, "") // Empty string = success

	log.Info("completed image analysis",
		zap.Duration("duration", time.Since(start)),
		zap.Int("description_length", len(description)))
}

// getPDFChunkPrompt returns the system message for PDF chunk analysis (delegated to handlers package)
func getPDFChunkPrompt() string {
	return handlers.GetPDFChunkPrompt()
}

// handlePDFPrecis processes PDF analysis requests.
// This handler downloads a PDF, extracts and chunks the text, generates a summary using AI,
// and creates a note with the summary on the canvas.
//
// Atomic design: Organism (orchestrates PDF processing, AI summarization, and note creation)
func handlePDFPrecis(update Update, client *canvusapi.Client, config *core.Config, logger *logging.Logger, repo *db.Repository, deps *HandlerDependencies) {
	triggerID, _ := update["id"].(string)
	correlationID := generateCorrelationID()
	log := logger.With(
		zap.String("correlation_id", correlationID),
		zap.String("widget_id", triggerID),
		zap.String("widget_type", "AI_Icon_PDF_Precis"),
	)

	ctx := context.Background()
	start := time.Now()

	// Record task start for dashboard metrics
	taskRecord := deps.recordTaskStart(correlationID, metrics.TaskTypePDF, config.CanvasID)

	// Create processing note
	processingNoteID, err := createProcessingNote(client, update, config, log)
	if err != nil {
		log.Error("failed to create processing note", zap.Error(err))
		deps.recordTaskComplete(taskRecord, "failed to create processing note")
		return
	}

	// Get the parent widget (the PDF to analyze)
	parentID := update["parentId"].(string)
	if parentID == "" {
		log.Error("no parent PDF to analyze")
		updateProcessingNote(client, processingNoteID, "❌ Error: No parent PDF found", config, log)
		deps.recordTaskComplete(taskRecord, "no parent PDF")
		return
	}

	parentWidget, err := client.GetWidget(parentID, false)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get parent widget: %v", err)
		log.Error("failed to get parent widget", zap.Error(err))
		updateProcessingNote(client, processingNoteID, fmt.Sprintf("❌ %s", errMsg), config, log)
		deps.recordTaskComplete(taskRecord, errMsg)
		return
	}

	// Verify parent is a PDF
	widgetType, _ := parentWidget["type"].(string)
	if widgetType != "Pdf" {
		errMsg := fmt.Sprintf("Parent widget is not a PDF (type: %s)", widgetType)
		log.Error("parent widget is not a PDF", zap.String("parent_type", widgetType))
		updateProcessingNote(client, processingNoteID, fmt.Sprintf("❌ %s", errMsg), config, log)
		deps.recordTaskComplete(taskRecord, errMsg)
		return
	}

	pdfURL, ok := parentWidget["url"].(string)
	if !ok || pdfURL == "" {
		log.Error("parent PDF has no URL")
		updateProcessingNote(client, processingNoteID, "❌ Error: PDF has no URL", config, log)
		deps.recordTaskComplete(taskRecord, "parent PDF has no URL")
		return
	}

	log.Info("analyzing PDF",
		zap.String("pdf_url", pdfURL),
		zap.String("parent_id", parentID))

	// Update processing note to show download in progress
	updateProcessingNote(client, processingNoteID, "⏳ Downloading PDF...", config, log)

	// Download the PDF
	httpClient := core.GetHTTPClient(config.AllowSelfSignedCerts)
	resp, err := httpClient.Get(pdfURL)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to download PDF: %v", err)
		log.Error("PDF download failed", zap.Error(err))
		updateProcessingNote(client, processingNoteID, fmt.Sprintf("❌ %s", errMsg), config, log)
		deps.recordTaskComplete(taskRecord, errMsg)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("PDF download failed with status: %d", resp.StatusCode)
		log.Error("PDF download failed", zap.Int("status", resp.StatusCode))
		updateProcessingNote(client, processingNoteID, fmt.Sprintf("❌ %s", errMsg), config, log)
		deps.recordTaskComplete(taskRecord, errMsg)
		return
	}

	// Save to temporary file
	tempFile := filepath.Join(config.DownloadsDir, fmt.Sprintf("pdf_analysis_%s.pdf", correlationID))
	outFile, err := os.Create(tempFile)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create temp file: %v", err)
		log.Error("temp file creation failed", zap.Error(err))
		updateProcessingNote(client, processingNoteID, fmt.Sprintf("❌ %s", errMsg), config, log)
		deps.recordTaskComplete(taskRecord, errMsg)
		return
	}

	_, err = io.Copy(outFile, resp.Body)
	outFile.Close()
	if err != nil {
		os.Remove(tempFile)
		errMsg := fmt.Sprintf("Failed to save PDF: %v", err)
		log.Error("PDF save failed", zap.Error(err))
		updateProcessingNote(client, processingNoteID, fmt.Sprintf("❌ %s", errMsg), config, log)
		deps.recordTaskComplete(taskRecord, errMsg)
		return
	}
	defer os.Remove(tempFile) // Clean up after processing

	log.Info("PDF downloaded",
		zap.String("temp_file", tempFile))

	// Update processing note to show extraction in progress
	updateProcessingNote(client, processingNoteID, "⏳ Extracting text from PDF...", config, log)

	// Create PDF processor with progress callback
	processorConfig := pdfprocessor.ProcessorConfig{
		ChunkSizeTokens:  config.PDFChunkSizeTokens,
		MaxChunksTokens:  config.PDFMaxChunksTokens,
		SummaryMaxTokens: config.PDFSummaryMaxTokens,
		Model:            config.OpenAIPDFModel,
		Temperature:      0.3,
	}

	// Progress callback to update the processing note
	progressCallback := func(stage string, current, total int) {
		progressMsg := fmt.Sprintf("⏳ %s (%d/%d)", stage, current, total)
		updateProcessingNote(client, processingNoteID, progressMsg, config, log)
	}

	aiClient := core.CreateOpenAIClient(config)
	processor := pdfprocessor.NewProcessorWithProgress(processorConfig, aiClient, progressCallback)

	// Process the PDF
	result, err := processor.Process(ctx, tempFile, "Please provide a comprehensive summary of this document.")
	if err != nil {
		errMsg := fmt.Sprintf("PDF processing failed: %v", err)
		log.Error("PDF processing failed", zap.Error(err))
		updateProcessingNote(client, processingNoteID, fmt.Sprintf("❌ %s", errMsg), config, log)
		recordProcessingHistory(
			ctx, repo, correlationID, config.CanvasID, triggerID,
			"pdf_analysis", pdfURL, "", config.OpenAIPDFModel,
			0, 0, int(time.Since(start).Milliseconds()),
			"error", err.Error(), log,
		)
		deps.recordTaskComplete(taskRecord, errMsg)
		return
	}

	log.Info("PDF summary generated",
		zap.Int("summary_length", len(result.Summary)),
		zap.Int("pages_processed", result.PagesProcessed))

	// Update the processing note with the summary
	updateProcessingNote(client, processingNoteID, result.Summary, config, log)

	// Record success to database
	recordProcessingHistory(
		ctx, repo, correlationID, config.CanvasID, triggerID,
		"pdf_analysis", pdfURL, truncateText(result.Summary, 1000), config.OpenAIPDFModel,
		result.InputTokens, result.OutputTokens, int(time.Since(start).Milliseconds()),
		"success", "", log,
	)
	deps.recordMetrics("pdf", time.Since(start))
	deps.recordTaskComplete(taskRecord, "") // Empty string = success

	log.Info("completed PDF analysis",
		zap.Duration("duration", time.Since(start)))
}

// handleCanvusPrecis processes canvas analysis requests.
// This handler fetches all widgets from the canvas, generates a comprehensive analysis using AI,
// and creates a note with the analysis on the canvas.
//
// Atomic design: Organism (orchestrates canvas fetching, AI analysis, and note creation)
func handleCanvusPrecis(update Update, client *canvusapi.Client, config *core.Config, logger *logging.Logger, repo *db.Repository, llamaClient *llamaruntime.Client, deps *HandlerDependencies) {
	triggerID, _ := update["id"].(string)
	correlationID := generateCorrelationID()
	log := logger.With(
		zap.String("correlation_id", correlationID),
		zap.String("widget_id", triggerID),
		zap.String("widget_type", "AI_Icon_Canvus_Precis"),
	)

	ctx := context.Background()
	start := time.Now()

	// Record task start for dashboard metrics
	taskRecord := deps.recordTaskStart(correlationID, metrics.TaskTypeCanvas, config.CanvasID)

	// Create processing note
	processingNoteID, err := createProcessingNote(client, update, config, log)
	if err != nil {
		log.Error("failed to create processing note", zap.Error(err))
		deps.recordTaskComplete(taskRecord, "failed to create processing note")
		return
	}

	log.Info("analyzing canvas",
		zap.String("canvas_id", config.CanvasID))

	// Update processing note to show fetching in progress
	updateProcessingNote(client, processingNoteID, "⏳ Fetching canvas widgets...", config, log)

	// Create canvas analyzer processor
	analyzerConfig := canvasanalyzer.ProcessorConfig{
		MaxTokens:   config.CanvasAnalysisMaxTokens,
		Model:       config.OpenAICanvasModel,
		Temperature: 0.5,
	}

	var processor *canvasanalyzer.Processor
	if llamaClient != nil {
		log.Info("using local LLM for canvas analysis")
		processor = canvasanalyzer.NewProcessorWithLlama(analyzerConfig, client, llamaClient, logger)
	} else {
		log.Info("using cloud API for canvas analysis")
		aiClient := core.CreateOpenAIClient(config)
		processor = canvasanalyzer.NewProcessor(analyzerConfig, client, aiClient, logger)
	}

	// Process the canvas
	result, err := processor.Process(ctx, "Please provide a comprehensive analysis of this canvas, including the main topics, structure, and key insights.")
	if err != nil {
		errMsg := fmt.Sprintf("Canvas analysis failed: %v", err)
		log.Error("canvas analysis failed", zap.Error(err))
		updateProcessingNote(client, processingNoteID, fmt.Sprintf("❌ %s", errMsg), config, log)
		recordProcessingHistory(
			ctx, repo, correlationID, config.CanvasID, triggerID,
			"canvas_analysis", "", "", config.OpenAICanvasModel,
			0, 0, int(time.Since(start).Milliseconds()),
			"error", err.Error(), log,
		)
		deps.recordTaskComplete(taskRecord, errMsg)
		return
	}

	log.Info("canvas analysis generated",
		zap.Int("analysis_length", len(result.Analysis)),
		zap.Int("widgets_analyzed", result.WidgetsAnalyzed))

	// Update the processing note with the analysis
	updateProcessingNote(client, processingNoteID, result.Analysis, config, log)

	// Record success to database
	recordProcessingHistory(
		ctx, repo, correlationID, config.CanvasID, triggerID,
		"canvas_analysis", "", truncateText(result.Analysis, 1000), config.OpenAICanvasModel,
		result.InputTokens, result.OutputTokens, int(time.Since(start).Milliseconds()),
		"success", "", log,
	)
	deps.recordMetrics("note", time.Since(start))
	deps.recordTaskComplete(taskRecord, "") // Empty string = success

	log.Info("completed canvas analysis",
		zap.Duration("duration", time.Since(start)))
}

// createProcessingNote creates a temporary "AI Processing" note on the canvas.
// This note is updated as processing progresses and eventually contains the final result.
func createProcessingNote(client *canvusapi.Client, triggerWidget Update, config *core.Config, log *logging.Logger) (string, error) {
	location := triggerWidget["location"].(map[string]interface{})
	size := triggerWidget["size"].(map[string]interface{})

	// Calculate position for the processing note (to the right of the trigger)
	newLocation := handlers.CalculateNoteLocation(location, size, config.NoteSpacing)

	note := canvusapi.CreateNoteRequest{
		Location: canvusapi.WidgetLocation{
			X: newLocation["x"].(float64),
			Y: newLocation["y"].(float64),
		},
		Size: canvusapi.WidgetSize{
			Width:  config.NoteWidth,
			Height: config.NoteHeight,
		},
		BackgroundColor: processingNoteColor,
		TextColor:       processingNoteTextColor,
		Text:            "⏳ " + processingNoteTitle,
	}

	result, err := client.CreateNote(note)
	if err != nil {
		return "", fmt.Errorf("failed to create processing note: %w", err)
	}

	log.Debug("processing note created",
		zap.String("note_id", result.ID))

	return result.ID, nil
}

// updateProcessingNote updates the text of an existing note widget.
func updateProcessingNote(client *canvusapi.Client, noteID string, text string, config *core.Config, log *logging.Logger) {
	// Determine the color based on the content
	var bgColor, textColor string
	if strings.HasPrefix(text, "❌") {
		// Error state - red background
		bgColor = "#DC143C" // Crimson
		textColor = "#FFFFFF"
	} else if strings.HasPrefix(text, "⚠️") {
		// Warning state - yellow background
		bgColor = "#FFD700" // Gold
		textColor = "#000000"
	} else if strings.HasPrefix(text, "⏳") {
		// Processing state - dark red
		bgColor = processingNoteColor
		textColor = processingNoteTextColor
	} else {
		// Success state - use configured colors
		bgColor = config.NoteColor
		textColor = config.NoteTextColor
	}

	req := canvusapi.UpdateWidgetRequest{
		Text:            &text,
		BackgroundColor: &bgColor,
		TextColor:       &textColor,
	}

	if err := client.UpdateWidget(noteID, req); err != nil {
		log.Error("failed to update processing note",
			zap.String("note_id", noteID),
			zap.Error(err))
		return
	}

	log.Debug("processing note updated",
		zap.String("note_id", noteID),
		zap.Int("text_length", len(text)))
}

// handleAIError creates an error note on the canvas to inform the user of processing failures.
func handleAIError(ctx context.Context, client *canvusapi.Client, update Update, err error, baseText string, config *core.Config, log *logging.Logger) error {
	location := update["location"].(map[string]interface{})
	size := update["size"].(map[string]interface{})

	// Calculate position for the error note (to the right of the trigger)
	newLocation := handlers.CalculateNoteLocation(location, size, config.NoteSpacing)

	errorText := fmt.Sprintf("❌ Error: %v", err)
	if baseText != "" {
		errorText = fmt.Sprintf("%s\n\n❌ Error: %v", baseText, err)
	}

	note := canvusapi.CreateNoteRequest{
		Location: canvusapi.WidgetLocation{
			X: newLocation["x"].(float64),
			Y: newLocation["y"].(float64),
		},
		Size: canvusapi.WidgetSize{
			Width:  config.NoteWidth,
			Height: config.NoteHeight,
		},
		BackgroundColor: "#DC143C", // Crimson for errors
		TextColor:       "#FFFFFF",
		Text:            errorText,
	}

	result, err := client.CreateNote(note)
	if err != nil {
		return fmt.Errorf("failed to create error note: %w", err)
	}

	log.Debug("error note created",
		zap.String("note_id", result.ID))

	return nil
}
