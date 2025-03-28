package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go_backend/canvusapi"
	"go_backend/logging"

	"bytes"
	"encoding/base64"

	"github.com/fatih/color"
	"github.com/ledongthuc/pdf"
	"github.com/openai/openai-go"
)

// Add constants at the top
const (
	processingNoteTitle     = "AI Processing"
	processingNoteColor     = "#8B0000" // Dark blood red
	processingNoteTextColor = "#FFFFFF"

	// Google Vision API constants
	visionAPIEndpoint = "https://vision.googleapis.com/v1/images:annotate"
	visionFeatureType = "DOCUMENT_TEXT_DETECTION"
)

// Core types and configuration
type AINoteResponse struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

// Configuration with defaults
var config *Config

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

// logHandler logs messages to both console and file with optional emoji
func logHandler(format string, v ...interface{}) {
	logging.LogHandler(format, v...)
}

// truncateText truncates a text to a specified length
func truncateText(text string, length int) string {
	if len(text) > length {
		return text[:length]
	}
	return text
}

// handleNote processes Note widget updates
func handleNote(update Update, client *canvusapi.Client) {
	if err := validateUpdate(update); err != nil {
		logHandler("❌ Invalid update: %v", err)
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return
	}

	// Check for AI trigger
	noteText, ok := update["text"].(string)
	if !ok {
		return
	}

	// Only process if there's an AI trigger
	if !strings.Contains(noteText, "{{") || !strings.Contains(noteText, "}}") {
		return
	}

	noteID := update["id"].(string)
	start := time.Now()

	// Immediately mark as processing - highest priority
	processingText := strings.ReplaceAll(
		strings.ReplaceAll(noteText, "{{", ""),
		"}}", "",
	)
	baseText := processingText // Store original text without status

	err := updateNoteWithRetry(client, noteID, map[string]interface{}{
		"text": baseText + "\n\n⚙️ Starting AI Processing...",
	})
	if err != nil {
		logHandler("❌ Failed to mark Note as processing: ID=%s, Error=%v", noteID, err)
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return
	}

	// Log the start of processing
	textPreview := truncateText(noteText, 30)
	logHandler("🤖 Processing AI trigger in Note: ID=%s, Text=\"%s\"", noteID, textPreview)

	// Create context with timeout for AI processing
	ctx, cancel := context.WithTimeout(context.Background(), config.AITimeout)
	defer cancel()

	// Update note to show we're analyzing the request
	updateNoteWithRetry(client, noteID, map[string]interface{}{
		"text": baseText + "\n\n🔍 Analyzing request...",
	})

	// Generate the AI response
	systemMessage := "You are an assistant capable of interpreting structured text triggers from a Note widget. " +
		"Evaluate whether the content in the Note is better suited for generating text or creating an image. " +
		"If generating text, respond with a JSON object like: {\"type\": \"text\", \"content\": \"...\"}. " +
		"If creating an image, respond with a JSON object like: {\"type\": \"image\", \"content\": \"...\"}. " +
		"Do not include any additional text or explanations."

	aiPrompt := strings.ReplaceAll(strings.ReplaceAll(noteText, "{{", ""), "}}", "")
	logHandler("🤖 Prompt: \"%.50s...\"", aiPrompt)

	rawResponse, err := generateAIResponse(ctx, aiPrompt, systemMessage, config.OpenAINoteModel, config.NoteResponseTokens)
	if err != nil {
		if err := handleAIError(ctx, client, update, err, baseText); err != nil {
			logHandler("❌ Failed to create error note: %v", err)
		}
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return
	}

	logHandler("✨ Response: \"%.50s...\"", rawResponse)

	// Parse the AI response
	var aiNoteResponse AINoteResponse
	if err := json.Unmarshal([]byte(rawResponse), &aiNoteResponse); err != nil {
		if err := handleAIError(ctx, client, update, err, baseText); err != nil {
			logHandler("❌ Failed to create error note: %v", err)
		}
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return
	}

	// Process the AI response based on type
	var creationErr error
	switch aiNoteResponse.Type {
	case "text":
		updateNoteWithRetry(client, noteID, map[string]interface{}{
			"text": baseText + "\n\n📝 Generating text response...",
		})
		logHandler("📝 Creating text response for Note: ID=%s", noteID)
		creationErr = createNoteFromResponse(aiNoteResponse.Content, noteID, update, false, client)

	case "image":
		// For image generation, provide more detailed progress updates
		updateNoteWithRetry(client, noteID, map[string]interface{}{
			"text": baseText + "\n\n🎨 Generating image...\nThis may take up to 30 seconds.",
		})
		logHandler("🎨 Creating image response for Note: ID=%s", noteID)

		// Update progress after 10 seconds if still processing
		go func() {
			time.Sleep(10 * time.Second)
			updateNoteWithRetry(client, noteID, map[string]interface{}{
				"text": baseText + "\n\n🎨 Still generating image...\n⏳ Almost there!",
			})
		}()

		creationErr = processAIImage(ctx, client, aiNoteResponse.Content, update)

	default:
		logHandler("❌ Unexpected AI response type for Note: ID=%s, Type=%s", noteID, aiNoteResponse.Type)
		creationErr = fmt.Errorf("unexpected response type: %s", aiNoteResponse.Type)
	}

	if creationErr != nil {
		logHandler("❌ Failed to create response widget for Note: ID=%s, Error=%v", noteID, creationErr)
		atomic.AddInt64(&handlerMetrics.errors, 1)
		updateNoteWithRetry(client, noteID, map[string]interface{}{
			"text": baseText + "\n\n❌ Error: Failed to create response",
		})
		return
	}

	// Clear the processing status
	clearProcessingStatus(client, noteID, baseText)

	// Update metrics
	atomic.AddInt64(&handlerMetrics.processedNotes, 1)
	metricsMutex.Lock()
	handlerMetrics.processingDuration += time.Since(start)
	metricsMutex.Unlock()

	logHandler("✅ Completed processing Note: ID=%s (took %v)", noteID, time.Since(start))
}

// validateUpdate checks if an update contains required fields
func validateUpdate(update Update) error {
	id, hasID := update["id"].(string)
	if !hasID || id == "" {
		return fmt.Errorf("missing or empty ID")
	}
	widgetType, hasType := update["widget_type"].(string)
	if !hasType || widgetType == "" {
		return fmt.Errorf("missing Type")
	}
	if _, hasLocation := update["location"].(map[string]interface{}); !hasLocation {
		return fmt.Errorf("missing Location")
	}
	if _, hasSize := update["size"].(map[string]interface{}); !hasSize {
		return fmt.Errorf("missing Size")
	}
	return nil
}

// updateNoteWithRetry attempts to update a note with retries
func updateNoteWithRetry(client *canvusapi.Client, noteID string, payload map[string]interface{}) error {
	for attempt := 1; attempt <= config.MaxRetries; attempt++ {
		_, err := client.UpdateNote(noteID, payload)
		if err == nil {
			logHandler("Note updated successfully: ID=%s on attempt %d", noteID, attempt)
			return nil
		}
		logHandler("Retry %d/%d failed to update Note: ID=%s, Error=%v",
			attempt, config.MaxRetries, noteID, err)
		time.Sleep(config.RetryDelay)
	}
	return fmt.Errorf("failed to update Note: ID=%s after %d attempts", noteID, config.MaxRetries)
}

// generateAIResponse generates an AI response using OpenAI
func generateAIResponse(ctx context.Context, prompt, systemMessage, model string, maxTokens int64) (string, error) {
	client := openai.NewClient()
	completion, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Messages: openai.F([]openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemMessage),
			openai.UserMessage(prompt),
		}),
		Model:            openai.F(model),
		MaxTokens:        openai.Int(maxTokens),
		Temperature:      openai.Float(0.3),
		TopP:             openai.Float(0.1),
		PresencePenalty:  openai.Float(0),
		FrequencyPenalty: openai.Float(0),
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate AI response: %w", err)
	}

	logHandler("Chunk processed")
	return completion.Choices[0].Message.Content, nil
}

// clearProcessingStatus removes processing indicator from note
func clearProcessingStatus(client *canvusapi.Client, noteID, processingText string) {
	clearedText := strings.ReplaceAll(processingText, "\n !! AI Processing !!", "")

	// First get the widget to determine its type
	widget, err := client.GetWidget(noteID, false)
	if err != nil {
		logHandler("Failed to get widget info: ID=%s, Error=%v", noteID, err)
		return
	}

	// Check widget type and use appropriate update method
	widgetType, ok := widget["widget_type"].(string)
	if !ok {
		logHandler("Failed to determine widget type: ID=%s", noteID)
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
		logHandler("Failed to clear processing status: ID=%s, Error=%v", noteID, updateErr)
	}
}

// Helper function to get absolute location
func getAbsoluteLocation(client *canvusapi.Client, widget Update) (map[string]float64, error) {
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
		logHandler("Warning: Parent widget %s has no location", parentID)
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
	logHandler("Location calculation: Parent(%.2f,%.2f) + Relative(%.2f,%.2f) = Absolute(%.2f,%.2f)",
		parentLoc["x"].(float64), parentLoc["y"].(float64),
		widgetLoc["x"].(float64), widgetLoc["y"].(float64),
		absoluteLoc["x"], absoluteLoc["y"])

	return absoluteLoc, nil
}

// createNoteFromResponse creates a new Note widget based on the AI response.
func createNoteFromResponse(content, triggeringNoteID string, triggeringUpdate Update, errorNote bool, client *canvusapi.Client) error {
	// Log content preview
	contentPreview := content
	if len(content) > 30 {
		contentPreview = content[:30]
	}
	logHandler("createNoteFromResponse called with content: %.30s", contentPreview)

	// Get original properties
	originalSize := triggeringUpdate["size"].(map[string]interface{})
	originalWidth := originalSize["width"].(float64)
	originalHeight := originalSize["height"].(float64)
	originalScale := triggeringUpdate["scale"].(float64)

	var size map[string]interface{}
	var scale float64

	if errorNote {
		size = map[string]interface{}{
			"width":  originalWidth,
			"height": originalHeight,
		}
		scale = originalScale
	} else {
		// Calculate content length in tokens (rough approximation: 1 token ≈ 4 characters)
		contentTokens := float64(len(content)) / 4.0

		// For short content (< 150 tokens), use original size and scale
		if contentTokens < 150 {
			size = map[string]interface{}{
				"width":  originalWidth,
				"height": originalHeight,
			}
			scale = originalScale

			logHandler("Short content (%f tokens): using original size and scale", contentTokens)
		} else {
			// Calculate content requirements
			contentLines := float64(strings.Count(content, "\n") + 1)
			maxLineLength := 0.0
			averageLineLength := 0.0
			totalChars := 0.0

			// Calculate max and average line lengths
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				lineLen := float64(len(line))
				if lineLen > maxLineLength {
					maxLineLength = lineLen
				}
				totalChars += lineLen
			}
			averageLineLength = totalChars / contentLines

			// Base measurements for content fitting at scale 1.0:
			const (
				targetWidth    = 830.0 // Target width that worked well
				charsPerLine   = 100.0 // Approximate chars that fit in target width
				linesPerHeight = 40.0  // Lines that fit in 1200 height units
				baseScale      = 0.37  // Base scale for full-size notes
				minWidth       = 300.0 // Minimum width for very short content
			)

			isFormattedText := averageLineLength < (charsPerLine*0.5) && contentLines > 5

			var width float64
			if isFormattedText {
				width = math.Max(minWidth, (maxLineLength/charsPerLine)*targetWidth)
			} else {
				width = targetWidth
			}

			totalLines := contentLines
			if !isFormattedText {
				totalLines += (maxLineLength / charsPerLine)
			}
			height := (totalLines / linesPerHeight) * 1200.0

			contentRatio := math.Min(1.0, math.Max(width/targetWidth, height/1200.0))
			scale = baseScale * (1.0 + (1.0 - contentRatio))

			const maxScale = 1.0
			scale = math.Min(maxScale, scale)

			size = map[string]interface{}{
				"width":  width,
				"height": height,
			}

			logHandler("Content sizing: content_lines=%f, avg_line_len=%f, max_chars=%f, formatted=%v, width=%f, height=%f, scale=%f",
				contentLines, averageLineLength, maxLineLength, isFormattedText, width, height, scale)
		}
	}

	// Get background color with fallback
	backgroundColor := "#FFFFFFBF" // Default white with 75% opacity
	if bgColor, ok := triggeringUpdate["background_color"].(string); ok {
		if len(bgColor) == 9 { // #RRGGBBAA format
			// Extract the alpha value and reduce it by 25%
			baseColor := bgColor[:7]
			alpha, _ := strconv.ParseInt(bgColor[7:], 16, 0)
			newAlpha := fmt.Sprintf("%02X", int(float64(alpha)/1.15)) // Reduces by ~25%
			backgroundColor = baseColor + newAlpha
		} else if len(bgColor) == 7 { // #RRGGBB format
			backgroundColor = bgColor + "BF" // Add 75% opacity
		} else {
			backgroundColor = bgColor // Keep original if format unknown
		}
	}

	// Get depth with fallback
	depth := 0.0
	if d, ok := triggeringUpdate["depth"].(float64); ok {
		depth = d + 200
	}

	// Prepare the creation payload
	payload := map[string]interface{}{
		"title":            fmt.Sprintf("Response to Note %s", triggeringNoteID),
		"text":             content,
		"location":         triggeringUpdate["location"],
		"size":             size,
		"depth":            depth,
		"scale":            scale,
		"background_color": backgroundColor,
		"auto_text_color":  false,
		"text_color":       "#000000ff", // Solid black text
	}

	// Log creation details
	loc := triggeringUpdate["location"].(map[string]interface{})
	logHandler("Creating Note at Loc: %.1f, %.1f - Size: %.1f, %.1f Scale: %.3f",
		loc["x"], loc["y"],
		size["width"], size["height"],
		scale)

	// Create the new Note
	_, err := client.CreateNote(payload)
	if err != nil {
		logHandler("Failed to create Note. Error=%v", err)
		return err
	}

	return nil
}

// processAIImage generates and uploads an image from the AI's response
func processAIImage(ctx context.Context, client *canvusapi.Client, prompt string, update Update) error {
	downloadsMutex.Lock()
	defer downloadsMutex.Unlock()

	logHandler("Generating and uploading AI image for prompt: %q", prompt)

	// Ensure downloads directory exists
	if err := os.MkdirAll(config.DownloadsDir, 0755); err != nil {
		return fmt.Errorf("failed to create downloads directory: %w", err)
	}

	// Generate the image using OpenAI
	aiClient := openai.NewClient()
	image, err := aiClient.Images.Generate(ctx, openai.ImageGenerateParams{
		Prompt:         openai.String(prompt),
		Model:          openai.F(openai.ImageModelDallE3),
		ResponseFormat: openai.F(openai.ImageGenerateParamsResponseFormatURL),
		Style:          openai.F(openai.ImageGenerateParamsStyleVivid),
		N:              openai.Int(1),
	})
	if err != nil {
		return fmt.Errorf("failed to generate AI image: %w", err)
	}

	// Download the image
	req, err := http.NewRequestWithContext(ctx, "GET", image.Data[0].URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
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

	logHandler("Creating image with payload: %+v", payload)
	_, err = client.CreateImage(imagePath, payload)
	if err != nil {
		return fmt.Errorf("failed to create image: %w", err)
	}

	return nil
}

// handleSnapshot processes Snapshot widgets for handwriting recognition
func handleSnapshot(update Update, client *canvusapi.Client) {
	start := time.Now()
	downloadsMutex.Lock()
	defer downloadsMutex.Unlock()

	imageID := update["id"].(string)

	// Log trigger widget details
	triggerLoc := update["location"].(map[string]interface{})
	triggerSize := update["size"].(map[string]interface{})
	logHandler("Snapshot Details:")
	logHandler("- ID: %s", imageID)
	logHandler("- Location: %d,%d", int(triggerLoc["x"].(float64)), int(triggerLoc["y"].(float64)))
	logHandler("- Size: %d,%d", int(triggerSize["width"].(float64)), int(triggerSize["height"].(float64)))

	// Create processing note
	processingNoteID, err := createProcessingNote(client, update)
	if err != nil {
		logHandler("Failed to create processing note: %v", err)
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
		logHandler("Download attempt %d/%d for image ID: %s", attempt, maxRetries, imageID)

		if attempt > 1 {
			// Update note with countdown
			for countdown := 3; countdown > 0; countdown-- {
				updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
					"text": fmt.Sprintf("Download failed. Retrying in %d...", countdown),
				})
				time.Sleep(time.Second)
			}
		}

		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": fmt.Sprintf("Downloading snapshot... (Attempt %d/%d)", attempt, maxRetries),
		})

		// Try to download
		logHandler("Attempting to download image: %s", imageID)
		downloadErr = client.DownloadImage(imageID, downloadPath)
		if downloadErr != nil {
			logHandler("Download attempt %d failed: %v", attempt, downloadErr)
			continue
		}

		// Verify the downloaded file
		fileInfo, err = os.Stat(downloadPath)
		if err != nil {
			logHandler("File verification failed after download attempt %d: %v", attempt, err)
			downloadErr = fmt.Errorf("file verification failed: %w", err)
			continue
		}

		if fileInfo.Size() == 0 {
			logHandler("Downloaded file is empty after attempt %d", attempt)
			downloadErr = fmt.Errorf("downloaded file is empty")
			continue
		}

		logHandler("Download attempt %d successful - File size: %d bytes", attempt, fileInfo.Size())
		downloadErr = nil
		break
	}

	// If all download attempts failed
	if downloadErr != nil {
		logHandler("All download attempts failed for image ID %s: %v", imageID, downloadErr)
		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": "❌ Failed to download image after multiple attempts.\nClick the snapshot again to retry.",
		})
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return // Keep snapshot
	}

	// Read image data
	imageData, err := os.ReadFile(downloadPath)
	if err != nil {
		logHandler("Failed to read image data: %v", err)
		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": "❌ Failed to read image data.\nClick the snapshot again to retry.",
		})
		atomic.AddInt64(&handlerMetrics.errors, 1)
		os.Remove(downloadPath)
		return // Keep snapshot
	}

	logHandler("Successfully read image data - Size: %d bytes", len(imageData))
	updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
		"text": "Processing image through OCR... Please wait.",
	})

	// Perform OCR
	ocrText, err := performGoogleVisionOCR(ctx, imageData)
	if err != nil {
		logHandler("Failed to perform OCR: %v", err)
		errorMessage := "❌ Failed to process image.\n\n"
		if strings.Contains(err.Error(), "no text found") {
			errorMessage += "No readable text was found in the image."
		} else {
			errorMessage += fmt.Sprintf("Error: %v", err)
		}
		errorMessage += "\n\nClick the snapshot again to retry."

		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": errorMessage,
		})
		atomic.AddInt64(&handlerMetrics.errors, 1)
		os.Remove(downloadPath)
		return // Keep snapshot
	}

	updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
		"text": "Creating response note...",
	})

	// Create response note
	if err := createNoteFromResponse(ocrText, imageID, update, false, client); err != nil {
		logHandler("Failed to create response note: %v", err)
		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": "❌ Failed to create response note.\nClick the snapshot again to retry.",
		})
		atomic.AddInt64(&handlerMetrics.errors, 1)
		os.Remove(downloadPath)
		return // Keep snapshot
	}

	// Only cleanup if everything succeeded
	logHandler("OCR process completed successfully, cleaning up resources")

	// Clean up the downloaded file
	if err := os.Remove(downloadPath); err != nil {
		logHandler("Warning: Failed to remove downloaded file: %v", err)
	}

	// Delete the processing note
	if err := deleteTriggeringWidget(client, "note", processingNoteID); err != nil {
		logHandler("Warning: Failed to delete processing note: %v", err)
	}

	// Only delete the snapshot after complete success
	if err := deleteTriggeringWidget(client, "image", imageID); err != nil {
		logHandler("Warning: Failed to delete snapshot: %v", err)
	}

	// Update metrics
	atomic.AddInt64(&handlerMetrics.processedImages, 1)
	metricsMutex.Lock()
	handlerMetrics.processingDuration += time.Since(start)
	metricsMutex.Unlock()

	logHandler("✅ Completed snapshot processing (took %v)", time.Since(start))
}

// handlePDFPrecis generates a summary of a PDF widget
func handlePDFPrecis(update Update, client *canvusapi.Client) {
	start := time.Now()
	parentID, _ := update["parent_id"].(string)

	// Get parent widget (PDF) first to get its properties
	parentWidget, err := client.GetWidget(parentID, false)
	if err != nil {
		logHandler("❌ Failed to get parent PDF widget: %v", err)
		return
	}

	// Log trigger widget (PDF) details
	pdfLoc := parentWidget["location"].(map[string]interface{})
	pdfSize := parentWidget["size"].(map[string]interface{})
	logHandler("Trigger Widget Details (PDF) - Loc: %d,%d - Size: %d,%d - Scale: %d",
		int(pdfLoc["x"].(float64)),
		int(pdfLoc["y"].(float64)),
		int(pdfSize["width"].(float64)),
		int(pdfSize["height"].(float64)),
		int(parentWidget["scale"].(float64)))

	// Verify it's a PDF widget
	widgetType, _ := parentWidget["widget_type"].(string)
	if strings.ToLower(widgetType) != "pdf" {
		logHandler("❌ Invalid widget type for PDF precis: Expected PDF, got %s", widgetType)
		return
	}

	// Use the PDF widget as the trigger widget instead of the AI icon
	triggerWidget := make(Update)
	for k, v := range parentWidget {
		triggerWidget[k] = v
	}

	// Create processing note with properties based on PDF
	processingNote := map[string]interface{}{
		"title": "PDF Analysis",
		"text":  "⚙️ Starting PDF analysis...",
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

	// Create the processing note
	noteResp, err := client.CreateNote(processingNote)
	if err != nil {
		logHandler("❌ Failed to create processing note: %v", err)
		return
	}
	processingNoteID := noteResp["id"].(string)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.ProcessingTimeout)
	defer cancel()

	// Download PDF
	downloadPath := filepath.Join(config.DownloadsDir, fmt.Sprintf("temp_pdf_%s.pdf", parentID))
	updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
		"text": "📥 Downloading PDF...",
	})

	if err := client.DownloadPDF(parentID, downloadPath); err != nil {
		logHandler("❌ Failed to download PDF: %v", err)
		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": "❌ Failed to download PDF",
		})
		return
	}
	defer os.Remove(downloadPath)

	// Extract text
	updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
		"text": "📄 Extracting text from PDF...",
	})

	pdfText, err := extractPDFText(downloadPath)
	if err != nil {
		logHandler("❌ PDF text extraction failed: %v", err)
		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": "❌ Failed to extract text from PDF",
		})
		return
	}

	// Process content in chunks
	chunks := splitIntoChunks(pdfText, int(config.PDFChunkSizeTokens))
	totalChunks := len(chunks)

	updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
		"text": fmt.Sprintf("🔍 Analyzing PDF content...\nProcessing %d sections", totalChunks),
	})

	var summaries []string
	for i, chunk := range chunks {
		select {
		case <-ctx.Done():
			logHandler("❌ PDF processing timeout")
			updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
				"text": "❌ Processing timeout - PDF too complex",
			})
			return
		default:
			updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
				"text": fmt.Sprintf("🔍 Analyzing section %d of %d...", i+1, totalChunks),
			})

			summary, err := generateAIResponse(ctx, chunk, getPDFChunkPrompt(), config.OpenAIPDFModel, config.PDFPrecisTokens)
			if err != nil {
				logHandler("❌ Chunk %d processing failed: %v", i+1, err)
				updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
					"text": fmt.Sprintf("❌ Failed while processing section %d", i+1),
				})
				return
			}
			summaries = append(summaries, summary)
		}
	}

	// Consolidate summaries
	updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
		"text": "📝 Creating final summary...",
	})

	finalSummary, err := consolidateSummaries(ctx, summaries)
	if err != nil {
		logHandler("❌ Failed to consolidate summaries: %v", err)
		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": "❌ Failed to create final summary",
		})
		return
	}

	// Create final note with the summary, using the PDF widget as trigger
	err = createNoteFromResponse(finalSummary, parentID, triggerWidget, false, client)
	if err != nil {
		logHandler("❌ Failed to create summary note: %v", err)
		updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
			"text": "❌ Failed to create summary note",
		})
		return
	}

	// Delete the processing note and icon
	deleteTriggeringWidget(client, "note", processingNoteID)
	deleteTriggeringWidget(client, "image", update["id"].(string))

	// Update metrics
	atomic.AddInt64(&handlerMetrics.processedPDFs, 1)
	metricsMutex.Lock()
	handlerMetrics.processingDuration += time.Since(start)
	metricsMutex.Unlock()

	logHandler("✅ Completed PDF precis (took %v)", time.Since(start))
}

// extractPDFText extracts text content from a PDF file
func extractPDFText(pdfPath string) (string, error) {
	// Open PDF file
	f, r, err := pdf.Open(pdfPath)
	if err != nil {
		return "", fmt.Errorf("failed to open PDF: %w", err)
	}
	defer f.Close()

	var textBuilder strings.Builder
	totalPages := r.NumPage()

	// Extract text from each page
	for pageIndex := 1; pageIndex <= totalPages; pageIndex++ {
		p := r.Page(pageIndex)
		if p.V.IsNull() {
			continue // Skip empty pages
		}

		text, err := p.GetPlainText(nil)
		if err != nil {
			logHandler("Warning: failed to extract text from page %d: %v", pageIndex, err)
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

func getPDFChunkPrompt() string {
	return `Analyze this section of the document and provide a clear summary. Focus on:
1. Main ideas and key points
2. Important details and data
3. Connections to other sections (if apparent)
Maintain academic tone and technical accuracy.`
}

// handleCanvusPrecis processes Canvus widget summaries
func handleCanvusPrecis(update Update, client *canvusapi.Client) {
	start := time.Now()
	canvasID := update["id"].(string)

	// Validate update
	if err := validateUpdate(update); err != nil {
		logHandler("Invalid Canvus precis update: %v", err)
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
		logHandler("Failed to update Canvus precis title: ID=%s, Error=%v", canvasID, err)
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return
	}

	// Fetch all widgets from the canvas
	widgets, err := fetchCanvasWidgets(ctx, client)
	if err != nil {
		logHandler("Failed to fetch widgets for Canvus precis: %v", err)
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return
	}

	// Generate and process the precis
	if err := processCanvusPrecis(ctx, client, update, widgets); err != nil {
		logHandler("Failed to process Canvus precis: ID=%s, Error=%v", canvasID, err)
		atomic.AddInt64(&handlerMetrics.errors, 1)
		return
	}

	// Update metrics
	metricsMutex.Lock()
	handlerMetrics.processingDuration += time.Since(start)
	metricsMutex.Unlock()

	// Cleanup original widget
	if err := deleteTriggeringWidget(client, update["widget_type"].(string), canvasID); err != nil {
		logHandler("Failed to delete triggering Canvus precis widget: ID=%s, Error=%v", canvasID, err)
	}
}

// fetchCanvasWidgets retrieves all widgets with retry logic
func fetchCanvasWidgets(ctx context.Context, client *canvusapi.Client) ([]map[string]interface{}, error) {
	var widgets []map[string]interface{}
	var lastErr error

	for attempt := 1; attempt <= config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			widgets, lastErr = client.GetWidgets(false)
			if lastErr == nil {
				return widgets, nil
			}
			logHandler("Attempt %d/%d failed to fetch widgets: %v",
				attempt, config.MaxRetries, lastErr)
			time.Sleep(config.RetryDelay)
		}
	}
	return nil, fmt.Errorf("failed to fetch widgets after %d attempts: %v",
		config.MaxRetries, lastErr)
}

// processCanvusPrecis generates and creates a summary of the canvas
func processCanvusPrecis(ctx context.Context, client *canvusapi.Client, update Update, widgets []map[string]interface{}) error {
	logHandler("Starting Canvus Precis processing")

	// Log trigger widget details
	triggerLoc := update["location"].(map[string]interface{})
	triggerSize := update["size"].(map[string]interface{})
	logHandler("Trigger Widget Details - Loc: %d,%d - Size: %d,%d - Scale: %d",
		int(triggerLoc["x"].(float64)),
		int(triggerLoc["y"].(float64)),
		int(triggerSize["width"].(float64)),
		int(triggerSize["height"].(float64)),
		int(update["scale"].(float64)))

	// Get icon location and add offset for processing note
	iconLoc := update["location"].(map[string]interface{})
	processingNoteLoc := map[string]interface{}{
		"x": iconLoc["x"].(float64) + 100.0,
		"y": iconLoc["y"].(float64) + 100.0,
	}

	// Create processing note first
	processingNote := map[string]interface{}{
		"title":    processingNoteTitle,
		"text":     "🔍 Analyzing canvas content...",
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
		"text": "🔍 Analyzing canvas content...\nProcessing " + strconv.Itoa(len(filteredWidgets)) + " widgets",
	})

	// Convert filtered widgets to JSON for AI processing
	widgetsJSON, err := json.Marshal(filteredWidgets)
	if err != nil {
		deleteTriggeringWidget(client, "note", processingNoteID)
		return fmt.Errorf("failed to marshal widgets data: %w", err)
	}

	// Configure system message for canvas analysis
	systemMessage := `You are an assistant analyzing a collaborative workspace. 
	Describe the content and relationships between items in a natural, narrative way. 
	Focus on the story the workspace is telling and how items relate to each other. 
	Avoid mentioning technical details like IDs or coordinates. 
	Format your response in markdown with two sections: 
	# Overview
	Describe the main themes and content of the workspace.
	# Insights
	Share observations about relationships between items and suggest next steps.`

	// Update processing status
	updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
		"text": "🤖 Generating canvas analysis...\nThis may take a moment.",
	})

	// Generate AI response
	rawResponse, err := generateAIResponse(ctx, string(widgetsJSON), systemMessage, config.OpenAICanvasModel, config.CanvasPrecisTokens)
	if err != nil {
		deleteTriggeringWidget(client, "note", processingNoteID)
		return handleAIError(ctx, client, update, fmt.Errorf("AI generation failed: %w", err), update["text"].(string))
	}

	// Calculate note size based on content
	contentLength := len(rawResponse)
	baseWidth := 600.0  // Wider base width for canvas analysis
	baseHeight := 400.0 // Taller base height for canvas analysis

	// Assume ~60 chars per line, ~25 lines visible at default scale
	charsPerLine := 60.0
	linesNeeded := float64(contentLength) / charsPerLine
	sizeRatio := math.Sqrt(linesNeeded / 25.0)

	// Limit size increase
	width := baseWidth * math.Max(1.0, math.Min(2.0, sizeRatio))
	height := baseHeight * math.Max(1.0, math.Min(2.0, sizeRatio))

	// Calculate scale - inverse to size increase
	scale := update["scale"].(float64)
	if width > baseWidth {
		scale = scale * (baseWidth / width)
	}

	// Log size calculations for tuning
	logHandler("Response Note size calculation: content_length=%d, size_ratio=%.2f, final_size=%.2f x %.2f, scale=%.2f",
		contentLength, sizeRatio, width, height, scale)

	// Update the location for the response note
	update["location"] = map[string]interface{}{
		"x": iconLoc["x"].(float64) + 100.0,
		"y": iconLoc["y"].(float64) + 100.0,
	}

	// Create response note
	err = createNoteFromResponse(rawResponse, update["id"].(string), update, false, client)
	if err != nil {
		deleteTriggeringWidget(client, "note", processingNoteID)
		return fmt.Errorf("failed to create response note: %w", err)
	}

	// Clean up processing note
	deleteTriggeringWidget(client, "note", processingNoteID)

	return nil
}

// deleteTriggeringWidget safely deletes a widget by type and ID
func deleteTriggeringWidget(client *canvusapi.Client, widgetType, widgetID string) error {
	logHandler("Deleting triggering widget: Type=%s, ID=%s", widgetType, widgetID)

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

// Cleanup performs all necessary cleanup operations for the handlers package
func Cleanup() {
	// Clean up downloads directory
	if err := CleanupDownloads(); err != nil {
		logHandler("Error cleaning up downloads: %v", err)
	}
}

// CleanupDownloads removes temporary files from downloads directory
func CleanupDownloads() error {
	pattern := filepath.Join(config.DownloadsDir, "temp_*")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to list temporary files: %w", err)
	}

	for _, match := range matches {
		if err := os.Remove(match); err != nil {
			logHandler("Failed to remove temporary file %s: %v", match, err)
		}
	}
	return nil
}

// handleAIError creates a friendly error note, clears processing text, and logs the error
func handleAIError(ctx context.Context, client *canvusapi.Client, update Update, err error, baseText string) error {
	logHandler("❌ AI Processing Error: %v", err)

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
	errResp := createNoteFromResponse(errorContent, update["id"].(string), update, true, client)

	// Clear the extra processing text from the original note
	clearProcessingStatus(client, update["id"].(string), baseText)

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

func consolidateSummaries(ctx context.Context, summaries []string) (string, error) {
	systemPrompt := `You are tasked with combining multiple document summaries into a coherent final summary. 
	Maintain key points and relationships between sections while eliminating redundancy.
	Structure the output with clear sections:
	# Overview
	# Key Points
	# Details
	# Conclusions`

	combinedSummaries := strings.Join(summaries, "\n---\n")
	return generateAIResponse(ctx, combinedSummaries, systemPrompt, config.OpenAIPDFModel, config.PDFPrecisTokens*2)
}

// estimateTokenCount provides a rough estimate of tokens in a text
// Using average of 4 characters per token as a rough approximation
func estimateTokenCount(text string) int {
	return len(text) / 4
}

// handleAIIcon processes AI icon updates
func (m *Monitor) handleAIIcon(update Update) error {
	title, _ := update["title"].(string)
	parentID, _ := update["parent_id"].(string)

	switch title {
	case "AI_Icon_PDF_Precis":
		// First check if the icon is placed on the shared canvas - if so, ignore it
		if parentID == sharedCanvas.ID {
			logHandler("Ignoring PDF precis icon on shared canvas")
			return nil
		}

		// Process the PDF once the icon is placed on a PDF widget
		go handlePDFPrecis(update, m.client)

	case "AI_Icon_Canvus_Precis":
		// For canvas precis, we want the parent to be the shared canvas
		if parentID != sharedCanvas.ID {
			return fmt.Errorf("AI_Icon_Canvus_Precis ParentID does not match SharedCanvasID")
		}
		go handleCanvusPrecis(update, m.client)

	default:
		color.Blue("Unrecognized AI_Icon type: %s", title)
	}
	return nil
}

// Helper function to create processing notes (reduces duplication)
func createProcessingNote(client *canvusapi.Client, update Update) (string, error) {
	absoluteLoc, err := getAbsoluteLocation(client, update)
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

// Helper function for cleanup (reduces duplication)
func cleanup(client *canvusapi.Client, processingNoteID, imageID, downloadPath string) {
	// Only try to delete the processing note if we have an ID
	if processingNoteID != "" {
		if err := client.DeleteNote(processingNoteID); err != nil {
			logHandler("Warning: Failed to delete processing note: %v", err)
			// Try to update the note instead of deleting if delete fails
			updateNoteWithRetry(client, processingNoteID, map[string]interface{}{
				"text": "❌ Process completed with errors.\nYou can delete this note.",
			})
		}
	}

	// Only try to delete the image if we have an ID
	if imageID != "" {
		if err := client.DeleteImage(imageID); err != nil {
			logHandler("Warning: Failed to delete snapshot image: %v", err)
		}
	}

	// Only try to delete the download file if path is provided
	if downloadPath != "" {
		if err := os.Remove(downloadPath); err != nil {
			if !os.IsNotExist(err) { // Only log if error is not "file doesn't exist"
				logHandler("Warning: Failed to remove local snapshot file: %v", err)
			}
		}
	}
}

// Add performGoogleVisionOCR function
func performGoogleVisionOCR(ctx context.Context, imageData []byte) (string, error) {
	logHandler("Starting Google Vision OCR process")

	apiKey := os.Getenv("GOOGLE_VISION_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("Google Vision API key not found in environment variables")
	}

	// Validate API key with minimal request
	if err := validateGoogleAPIKey(ctx, apiKey); err != nil {
		logHandler("❌ Google Vision API key validation failed: %v", err)
		return "", fmt.Errorf("invalid API key: %w", err)
	}

	logHandler("✅ Google Vision API key validated successfully")

	// Rest of the existing function...

	// Log image data details
	logHandler("Image data received, size: %d bytes", len(imageData))

	// Create request body as JSON
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
					Content: base64.StdEncoding.EncodeToString(imageData),
				},
				Features: []struct {
					Type       string `json:"type"`
					MaxResults int    `json:"maxResults"`
				}{
					{
						Type:       visionFeatureType,
						MaxResults: 1,
					},
				},
			},
		},
	}

	// Convert request to JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	logHandler("Request JSON created, size: %d bytes", len(jsonData))

	// Create HTTP request
	url := fmt.Sprintf("%s?key=%s", visionAPIEndpoint, apiKey)
	logHandler("Making request to URL: %s", strings.Replace(url, apiKey, "REDACTED", 1))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	logHandler("Sending request to Google Vision API...")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Log response status
	logHandler("Received response from Google Vision API. Status: %d %s",
		resp.StatusCode, resp.Status)

	// Only consider it a failure if the API response is not 200
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Google Vision API error: status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Read response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	logHandler("Response body size: %d bytes", len(bodyBytes))

	var visionResponse GoogleVisionResponse
	if err := json.NewDecoder(bytes.NewReader(bodyBytes)).Decode(&visionResponse); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for API-level errors in the response
	if len(visionResponse.Responses) == 0 {
		return "", fmt.Errorf("empty response from Vision API")
	}
	if visionResponse.Responses[0].Error.Message != "" {
		return "", fmt.Errorf("Vision API error: %s", visionResponse.Responses[0].Error.Message)
	}

	extractedText := visionResponse.Responses[0].FullTextAnnotation.Text
	if extractedText == "" {
		return "", fmt.Errorf("no text found in image")
	}

	// Remove the error phrase checks since we don't want to trigger on words in the actual content
	return extractedText, nil
}

// First, let's add better logging for the Google Vision API authentication
func processImage(ctx context.Context, client *canvusapi.Client, imageID string, processingNoteID string) error {
	logHandler("Starting OCR process for image: %s", imageID)

	// Download the snapshot using the standard client method
	downloadPath := filepath.Join(
		config.DownloadsDir,
		fmt.Sprintf("temp_snapshot_%s.jpg", imageID),
	)

	if err := client.DownloadImage(imageID, downloadPath); err != nil {
		logHandler("❌ Failed to download image %s: %v", imageID, err)
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(downloadPath) // Clean up file after function returns

	// Verify Google Vision API key
	if len(config.GoogleVisionKey) < 30 {
		logHandler("❌ Invalid Google Vision API key length: %d chars", len(config.GoogleVisionKey))
		return fmt.Errorf("invalid Google Vision API key")
	}

	// Perform OCR
	ocrResult, err := performOCR(downloadPath, config.GoogleVisionKey)
	if err != nil {
		logHandler("❌ OCR failed: %v", err)
		// Don't delete widgets on failure, just return the error
		return fmt.Errorf("OCR failed: %w", err)
	}

	// Create response note with OCR results using standard client method
	notePayload := map[string]interface{}{
		"text": ocrResult,
		"location": map[string]interface{}{
			"x": originalNoteX,
			"y": originalNoteY,
		},
		"size": map[string]interface{}{
			"width":  400,
			"height": 300,
		},
		"depth": 0,
		"scale": 1.0,
	}

	_, err = client.CreateNote(notePayload)
	if err != nil {
		logHandler("❌ Failed to create response note: %v", err)
		return fmt.Errorf("failed to create response note: %w", err)
	}

	// Only delete original widgets after successful OCR and response creation
	if err := client.DeleteNote(processingNoteID); err != nil {
		logHandler("⚠️ Failed to delete processing note: %v", err)
		// Continue anyway as this is not critical
	}

	if err := client.DeleteImage(imageID); err != nil {
		logHandler("⚠️ Failed to delete original image: %v", err)
		// Continue anyway as this is not critical
	}

	logHandler("✅ Successfully processed image %s and created response note", imageID)
	return nil
}

// Improve the OCR function with better error handling
func performOCR(imagePath string, apiKey string) (string, error) {
	logHandler("Starting Google Vision OCR for image: %s", filepath.Base(imagePath))

	// Read image file
	imageBytes, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("failed to read image file: %w", err)
	}

	// Create Vision API request
	requestBody := map[string]interface{}{
		"requests": []map[string]interface{}{
			{
				"image": map[string]interface{}{
					"content": base64.StdEncoding.EncodeToString(imageBytes),
				},
				"features": []map[string]interface{}{
					{
						"type": "TEXT_DETECTION",
					},
				},
			},
		},
	}

	// Create HTTP request
	requestURL := fmt.Sprintf("https://vision.googleapis.com/v1/images:annotate?key=%s", apiKey)
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request JSON: %w", err)
	}

	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Make request with detailed error handling
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		logHandler("❌ Google Vision API error response: %s", string(body))
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	// Parse response
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse API response: %w", err)
	}

	// Extract text from response
	responses, ok := result["responses"].([]interface{})
	if !ok || len(responses) == 0 {
		return "", fmt.Errorf("invalid API response format")
	}

	firstResponse := responses[0].(map[string]interface{})
	textAnnotations, ok := firstResponse["textAnnotations"].([]interface{})
	if !ok || len(textAnnotations) == 0 {
		return "", fmt.Errorf("no text found in image")
	}

	text := textAnnotations[0].(map[string]interface{})["description"].(string)
	logHandler("✅ Successfully extracted text from image")
	return text, nil
}

// validateGoogleAPIKey makes a minimal API call to verify the key works
func validateGoogleAPIKey(ctx context.Context, apiKey string) error {
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

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("API key validation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("API key validation failed: status=%d, body=%s", resp.StatusCode, string(body))
}

// Define originalNoteX and originalNoteY
var originalNoteX, originalNoteY float64
