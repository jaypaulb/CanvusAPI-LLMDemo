// Package webui provides the web-based user interface for CanvusLocalLLM.
// This file contains the CanvasHealthMonitor organism for tracking multi-canvas connection status.
package webui

import (
	"context"
	"sync"
	"time"

	"go_backend/metrics"
)

// CanvasChecker defines the interface for checking canvas health.
// This abstraction allows for testing without real network calls.
type CanvasChecker interface {
	// GetCanvasInfo performs a lightweight health check on the canvas.
	// Returns nil error if the canvas is accessible.
	GetCanvasInfo() (map[string]interface{}, error)

	// GetCanvasID returns the canvas identifier.
	GetCanvasID() string

	// GetServerURL returns the Canvus server URL.
	GetServerURL() string
}

// CanvasHealthMonitor is an organism that tracks multi-canvas connection status.
// It periodically performs health checks and updates the MetricsStore with
// per-canvas status information.
//
// This organism composes:
// - metrics.MetricsCollector for storing health status
// - CanvasChecker implementations for performing health checks
// - sync.RWMutex for thread-safe canvas management
//
// Usage:
//
//	monitor := NewCanvasHealthMonitor(metricsStore, 30*time.Second)
//	monitor.RegisterCanvas(canvasChecker)
//	ctx, cancel := context.WithCancel(context.Background())
//	go monitor.Start(ctx)
//	// ... later ...
//	cancel() // Stop the monitor
type CanvasHealthMonitor struct {
	mu              sync.RWMutex
	store           metrics.MetricsCollector
	canvases        map[string]CanvasChecker
	checkInterval   time.Duration
	onStatusChange  func(canvasID string, connected bool) // Optional callback
	logger          Logger
}

// HealthMonitorConfig configures the CanvasHealthMonitor behavior.
type HealthMonitorConfig struct {
	// CheckInterval is how often to perform health checks (default: 30s)
	CheckInterval time.Duration
	// OnStatusChange is called when a canvas connection status changes
	OnStatusChange func(canvasID string, connected bool)
	// Logger for diagnostic output
	Logger Logger
}

// DefaultHealthMonitorConfig returns a default configuration.
func DefaultHealthMonitorConfig() HealthMonitorConfig {
	return HealthMonitorConfig{
		CheckInterval: 30 * time.Second,
		Logger:        &defaultLogger{},
	}
}

// NewCanvasHealthMonitor creates a new CanvasHealthMonitor with the specified
// metrics store and configuration.
func NewCanvasHealthMonitor(store metrics.MetricsCollector, config HealthMonitorConfig) *CanvasHealthMonitor {
	interval := config.CheckInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}

	logger := config.Logger
	if logger == nil {
		logger = &defaultLogger{}
	}

	return &CanvasHealthMonitor{
		store:          store,
		canvases:       make(map[string]CanvasChecker),
		checkInterval:  interval,
		onStatusChange: config.OnStatusChange,
		logger:         logger,
	}
}

// RegisterCanvas adds a canvas to be monitored.
// The canvas will be checked on the next health check cycle.
func (m *CanvasHealthMonitor) RegisterCanvas(canvas CanvasChecker) {
	m.mu.Lock()
	defer m.mu.Unlock()

	canvasID := canvas.GetCanvasID()
	m.canvases[canvasID] = canvas

	// Initialize status as disconnected until first check
	m.store.UpdateCanvasStatus(metrics.CanvasStatus{
		ID:         canvasID,
		ServerURL:  canvas.GetServerURL(),
		Connected:  false,
		LastUpdate: time.Now(),
	})
}

// UnregisterCanvas removes a canvas from monitoring.
func (m *CanvasHealthMonitor) UnregisterCanvas(canvasID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.canvases, canvasID)
}

// GetRegisteredCanvases returns the IDs of all registered canvases.
func (m *CanvasHealthMonitor) GetRegisteredCanvases() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.canvases))
	for id := range m.canvases {
		ids = append(ids, id)
	}
	return ids
}

// Start begins the periodic health check loop.
// It runs until the context is cancelled.
// This method blocks, so it should typically be run in a goroutine.
func (m *CanvasHealthMonitor) Start(ctx context.Context) {
	// Perform initial check immediately
	m.checkAllCanvases()

	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Printf("CanvasHealthMonitor: stopping due to context cancellation")
			return
		case <-ticker.C:
			m.checkAllCanvases()
		}
	}
}

// CheckNow performs an immediate health check on all registered canvases.
// This can be called manually in addition to the periodic checks.
func (m *CanvasHealthMonitor) CheckNow() {
	m.checkAllCanvases()
}

// checkAllCanvases performs health checks on all registered canvases.
func (m *CanvasHealthMonitor) checkAllCanvases() {
	m.mu.RLock()
	canvases := make([]CanvasChecker, 0, len(m.canvases))
	for _, c := range m.canvases {
		canvases = append(canvases, c)
	}
	m.mu.RUnlock()

	for _, canvas := range canvases {
		m.checkCanvas(canvas)
	}
}

// checkCanvas performs a health check on a single canvas.
func (m *CanvasHealthMonitor) checkCanvas(canvas CanvasChecker) {
	canvasID := canvas.GetCanvasID()

	// Get previous status for comparison
	prevStatus, hasPrev := m.store.GetCanvasStatus(canvasID)

	// Perform health check
	info, err := canvas.GetCanvasInfo()
	connected := err == nil

	// Build updated status
	status := metrics.CanvasStatus{
		ID:         canvasID,
		ServerURL:  canvas.GetServerURL(),
		Connected:  connected,
		LastUpdate: time.Now(),
	}

	// Extract canvas name if available
	if connected && info != nil {
		if name, ok := info["name"].(string); ok {
			status.Name = name
		}
	}

	// Preserve some fields from previous status if they exist
	if hasPrev {
		status.WidgetCount = prevStatus.WidgetCount
		status.RequestsToday = prevStatus.RequestsToday
		status.SuccessRate = prevStatus.SuccessRate

		// If disconnected, add error to list
		if !connected && err != nil {
			errMsg := err.Error()
			// Limit error history to last 5 errors
			errors := prevStatus.Errors
			if len(errors) >= 5 {
				errors = errors[1:]
			}
			status.Errors = append(errors, errMsg)
		} else if connected {
			// Clear errors on reconnection
			status.Errors = nil
		}
	}

	// Update the store
	m.store.UpdateCanvasStatus(status)

	// Call status change callback if status changed
	if m.onStatusChange != nil {
		if !hasPrev || prevStatus.Connected != connected {
			m.onStatusChange(canvasID, connected)
		}
	}

	// Log status changes
	if hasPrev && prevStatus.Connected != connected {
		if connected {
			m.logger.Printf("CanvasHealthMonitor: canvas %s reconnected", canvasID)
		} else {
			m.logger.Printf("CanvasHealthMonitor: canvas %s disconnected: %v", canvasID, err)
		}
	}
}

// UpdateCanvasMetrics updates the widget count and request metrics for a canvas.
// This should be called by the canvas monitor when processing events.
func (m *CanvasHealthMonitor) UpdateCanvasMetrics(canvasID string, widgetCount int, requestsToday int64, successRate float64) {
	status, ok := m.store.GetCanvasStatus(canvasID)
	if !ok {
		return
	}

	status.WidgetCount = widgetCount
	status.RequestsToday = requestsToday
	status.SuccessRate = successRate
	status.LastUpdate = time.Now()

	m.store.UpdateCanvasStatus(status)
}
