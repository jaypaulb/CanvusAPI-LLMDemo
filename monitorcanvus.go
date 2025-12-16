package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"go_backend/canvusapi"
	"go_backend/core"
	"go_backend/imagegen"
	"go_backend/logging"

	"go.uber.org/zap"
)

// Monitor represents the canvas monitoring service
type Monitor struct {
	client           *canvusapi.Client
	config           *core.Config
	logger           *logging.Logger
	done             chan struct{}
	widgets          map[string]map[string]interface{}
	widgetsMux       sync.RWMutex
	imagegenProc     *imagegen.Processor
	imagegenProcMux  sync.RWMutex
}

// WidgetState tracks widget information
type WidgetState struct {
	LastSeen time.Time
	Text     string
	Title    string
	ParentID string
}

// Update represents a widget update from Canvus
type Update map[string]interface{}

// SharedCanvas information with thread-safe access
type SharedCanvas struct {
	sync.RWMutex
	ID   string
	Data Update
}

var sharedCanvas SharedCanvas

// NewMonitor creates a new Monitor instance
func NewMonitor(client *canvusapi.Client, cfg *core.Config, logger *logging.Logger) *Monitor {
	return &Monitor{
		client:     client,
		config:     cfg,
		logger:     logger,
		done:       make(chan struct{}),
		widgets:    make(map[string]map[string]interface{}),
		widgetsMux: sync.RWMutex{},
	}
}

// SetImagegenProcessor sets the image generation processor for handling {{image:}} prompts.
// This should be called after the SD runtime is initialized. If not set, image prompts
// will fall back to the existing AI classification flow in handleNote.
func (m *Monitor) SetImagegenProcessor(proc *imagegen.Processor) {
	m.imagegenProcMux.Lock()
	defer m.imagegenProcMux.Unlock()
	m.imagegenProc = proc
	m.logger.Info("imagegen processor set for direct image prompt handling")
}

// getImagegenProcessor returns the imagegen processor if available.
func (m *Monitor) getImagegenProcessor() *imagegen.Processor {
	m.imagegenProcMux.RLock()
	defer m.imagegenProcMux.RUnlock()
	return m.imagegenProc
}

// Done returns a channel that's closed when monitoring is complete
func (m *Monitor) Done() <-chan struct{} {
	return m.done
}

// Start begins monitoring the canvas
func (m *Monitor) Start(ctx context.Context) {
	defer close(m.done)

	for {
		select {
		case <-ctx.Done():
			m.logger.Warn("Stopping monitor due to context cancellation")
			return
		default:
			if err := m.connectAndStream(ctx); err != nil {
				m.logger.Error("Stream error, reconnecting in 5 seconds",
					zap.Error(err))
				select {
				case <-ctx.Done():
					return
				case <-time.After(5 * time.Second):
					continue
				}
			}
		}
	}
}

// connectAndStream establishes and maintains the API stream connection
func (m *Monitor) connectAndStream(ctx context.Context) error {
	// Use the existing GetWidgets method with subscribe=true
	widgets, err := m.client.GetWidgets(true)
	if err != nil {
		return fmt.Errorf("failed to connect to widget stream: %w", err)
	}

	// Process initial widget state
	for _, widget := range widgets {
		if widgetJSON, err := json.Marshal(widget); err == nil {
			if err := m.handleUpdate(string(widgetJSON)); err != nil {
				m.logger.Error("Error handling initial widget",
					zap.Error(err))
			}
		}
	}

	return nil
}

// handleUpdate processes a single update from the stream
func (m *Monitor) handleUpdate(line string) error {
	if line == "" {
		return nil // Keep-alive message
	}

	var updates []Update
	if err := m.parseUpdates(line, &updates); err != nil {
		m.logger.Error("Failed to parse updates",
			zap.Error(err))
		return fmt.Errorf("failed to parse updates: %w", err)
	}

	for _, update := range updates {
		if err := m.processUpdate(update); err != nil {
			if id, ok := update["id"].(string); ok {
				m.logger.Error("Error processing update",
					zap.String("widget_id", id),
					zap.Error(err))
			}
		}
	}

	return nil
}

// parseUpdates handles both single and array update formats
func (m *Monitor) parseUpdates(line string, updates *[]Update) error {
	if strings.HasPrefix(line, "[") {
		return json.Unmarshal([]byte(line), updates)
	}

	var single Update
	if err := json.Unmarshal([]byte(line), &single); err != nil {
		return err
	}
	*updates = append(*updates, single)
	return nil
}

// processUpdate handles individual updates
func (m *Monitor) processUpdate(update Update) error {
	state, ok := update["state"].(string)
	if !ok || state != "normal" {
		return nil
	}

	// Handle SharedCanvas updates
	widgetType, _ := update["widget_type"].(string)
	if widgetType == "SharedCanvas" {
		return m.handleSharedCanvasUpdate(update)
	}

	// Process only Note and Image widgets
	if widgetType != "Note" && widgetType != "Image" {
		return nil
	}

	// Round location values
	m.roundLocationValues(&update)

	// Check if update is relevant
	if !m.isRelevantUpdate(update) {
		return nil
	}

	// Route to appropriate handler
	return m.routeUpdate(update)
}

// roundLocationValues rounds location coordinates
func (m *Monitor) roundLocationValues(update *Update) {
	if loc, ok := (*update)["location"].(map[string]interface{}); ok {
		if x, ok := loc["x"].(float64); ok {
			loc["x"] = math.Round(x)
		}
		if y, ok := loc["y"].(float64); ok {
			loc["y"] = math.Round(y)
		}
	}
}

// isRelevantUpdate checks if an update needs processing
func (m *Monitor) isRelevantUpdate(update Update) bool {
	id, ok := update["id"].(string)
	if !ok {
		return false
	}

	m.widgetsMux.RLock()
	state, exists := m.widgets[id]
	m.widgetsMux.RUnlock()

	if !exists {
		m.updateWidgetState(update)
		return true
	}

	text, _ := update["text"].(string)
	title, _ := update["title"].(string)
	parentID, _ := update["parent_id"].(string)

	// Compare with state map values
	stateText, _ := state["text"].(string)
	stateTitle, _ := state["title"].(string)
	stateParentID, _ := state["parent_id"].(string)

	if stateText != text ||
		stateTitle != title ||
		stateParentID != parentID {
		m.updateWidgetState(update)
		return true
	}

	return false
}

// updateWidgetState updates the stored state of a widget
func (m *Monitor) updateWidgetState(update Update) {
	m.widgetsMux.Lock()
	defer m.widgetsMux.Unlock()

	id, _ := update["id"].(string)

	// Create a new state map
	state := map[string]interface{}{
		"last_seen": time.Now(),
		"text":      update["text"],
		"title":     update["title"],
		"parent_id": update["parent_id"],
	}

	m.widgets[id] = state
}

// handleSharedCanvasUpdate processes SharedCanvas updates
func (m *Monitor) handleSharedCanvasUpdate(update Update) error {
	sharedCanvas.Lock()
	defer sharedCanvas.Unlock()

	if sharedCanvas.ID == "" {
		sharedCanvas.ID = update["id"].(string)
		sharedCanvas.Data = update
		return m.saveSharedCanvasData(update)
	}
	return nil
}

// saveSharedCanvasData persists SharedCanvas information
func (m *Monitor) saveSharedCanvasData(data Update) error {
	file, err := os.Create("shared_canvas.json")
	if err != nil {
		return fmt.Errorf("failed to create shared canvas file: %w", err)
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(data)
}

// routeUpdate directs updates to appropriate handlers
func (m *Monitor) routeUpdate(update Update) error {
	switch update["widget_type"].(string) {
	case "Note":
		// Check for direct image prompt {{image:...}}
		if prompt, ok := m.parseImagePrompt(update); ok {
			go m.handleImagePrompt(update, prompt)
			return nil
		}
		// Fall back to existing text/image classification flow
		go handleNote(update, m.client, m.config, m.logger)
	case "Image":
		if title, ok := update["title"].(string); ok {
			if strings.HasPrefix(title, "Snapshot at") {
				go handleSnapshot(update, m.client, m.config, m.logger)
			} else if strings.HasPrefix(title, "AI_Icon_") {
				return m.handleAIIcon(update)
			}
		}
	}
	return nil
}

// parseImagePrompt checks if the note text contains a direct image prompt.
// Returns the extracted prompt and true if found, empty string and false otherwise.
//
// Supported formats:
//   - {{image: prompt text here}}
//   - {{image:prompt text here}}
//   - {{ image: prompt text here }}
//   - {{IMAGE: prompt text here}} (case-insensitive prefix)
func (m *Monitor) parseImagePrompt(update Update) (string, bool) {
	text, ok := update["text"].(string)
	if !ok || text == "" {
		return "", false
	}

	// Find the start of the trigger
	startIdx := strings.Index(text, "{{")
	if startIdx == -1 {
		return "", false
	}

	// Find the end of the trigger
	endIdx := strings.Index(text[startIdx:], "}}")
	if endIdx == -1 {
		return "", false
	}
	endIdx += startIdx // Adjust to absolute position

	// Extract content between {{ and }}
	content := text[startIdx+2 : endIdx]
	content = strings.TrimSpace(content)

	// Check if it starts with "image:" (case-insensitive)
	lower := strings.ToLower(content)
	if !strings.HasPrefix(lower, "image:") {
		return "", false
	}

	// Extract the prompt after "image:"
	prompt := strings.TrimSpace(content[6:]) // len("image:") == 6
	if prompt == "" {
		return "", false
	}

	return prompt, true
}

// handleImagePrompt processes a direct image generation prompt via imagegen.
// If no imagegen processor is available, it falls back to the standard handleNote flow.
func (m *Monitor) handleImagePrompt(update Update, prompt string) {
	noteID, _ := update["id"].(string)
	log := m.logger.With(
		zap.String("widget_id", noteID),
		zap.String("prompt_preview", truncatePrompt(prompt, 50)),
	)

	log.Info("detected direct image prompt")

	// Check if imagegen processor is available
	proc := m.getImagegenProcessor()
	if proc == nil {
		log.Debug("imagegen processor not available, falling back to handleNote")
		handleNote(update, m.client, m.config, m.logger)
		return
	}

	// Create parent widget from update
	parentWidget, err := m.createParentWidget(update)
	if err != nil {
		log.Error("failed to create parent widget for image generation", zap.Error(err))
		// Fall back to handleNote which has error handling
		handleNote(update, m.client, m.config, m.logger)
		return
	}

	// Update the note to show processing
	originalText, _ := update["text"].(string)
	baseText := strings.ReplaceAll(strings.ReplaceAll(originalText, "{{", ""), "}}", "")
	baseText = strings.TrimSpace(baseText)
	// Remove "image:" prefix for display
	if strings.HasPrefix(strings.ToLower(baseText), "image:") {
		baseText = strings.TrimSpace(baseText[6:])
	}

	_, err = m.client.UpdateNote(noteID, map[string]interface{}{
		"text": baseText + "\n\n[SD] Generating image...\nThis may take 10-30 seconds.",
	})
	if err != nil {
		log.Warn("failed to update note with processing status", zap.Error(err))
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), m.config.AITimeout)
	defer cancel()

	// Process the image prompt
	result, err := proc.ProcessImagePrompt(ctx, prompt, parentWidget)
	if err != nil {
		log.Error("image generation failed", zap.Error(err))
		// Update note with error
		_, _ = m.client.UpdateNote(noteID, map[string]interface{}{
			"text": baseText + "\n\n[SD] Image generation failed: " + err.Error(),
		})
		return
	}

	// Clear processing status from note
	_, err = m.client.UpdateNote(noteID, map[string]interface{}{
		"text": baseText,
	})
	if err != nil {
		log.Warn("failed to clear processing status from note", zap.Error(err))
	}

	log.Info("image generation completed successfully",
		zap.String("widget_id", result.WidgetID))
}

// createParentWidget creates an imagegen.ParentWidget from an Update.
func (m *Monitor) createParentWidget(update Update) (imagegen.ParentWidget, error) {
	id, ok := update["id"].(string)
	if !ok {
		return nil, fmt.Errorf("update missing id field")
	}

	// Extract location
	locMap, ok := update["location"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("update missing location field")
	}
	x, _ := locMap["x"].(float64)
	y, _ := locMap["y"].(float64)

	// Extract size
	sizeMap, ok := update["size"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("update missing size field")
	}
	width, _ := sizeMap["width"].(float64)
	height, _ := sizeMap["height"].(float64)

	// Extract scale (default to 1.0 if not present)
	scale, ok := update["scale"].(float64)
	if !ok {
		scale = 1.0
	}

	// Extract depth (default to 0 if not present)
	depth, ok := update["depth"].(float64)
	if !ok {
		depth = 0.0
	}

	return imagegen.CanvasWidget{
		ID: id,
		Location: imagegen.WidgetLocation{
			X: x,
			Y: y,
		},
		Size: imagegen.WidgetSize{
			Width:  width,
			Height: height,
		},
		Scale: scale,
		Depth: depth,
	}, nil
}

// truncatePrompt truncates a prompt string for logging purposes.
func truncatePrompt(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}
