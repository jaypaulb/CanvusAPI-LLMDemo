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
	"go_backend/logging"

	"go.uber.org/zap"
)

// Monitor represents the canvas monitoring service
type Monitor struct {
	client     *canvusapi.Client
	config     *core.Config
	logger     *logging.Logger
	done       chan struct{}
	widgets    map[string]map[string]interface{}
	widgetsMux sync.RWMutex
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
