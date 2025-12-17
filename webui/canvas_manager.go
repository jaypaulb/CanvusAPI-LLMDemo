// Package webui provides the web-based user interface for CanvusLocalLLM.
// This file contains the CanvasManager organism for multi-canvas client management.
package webui

import (
	"fmt"
	"sync"

	"go_backend/canvusapi"
	"go_backend/metrics"
)

// CanvasManager is an organism that manages multiple Canvus API clients.
// It provides a centralized way to add, remove, and access canvas clients,
// and integrates with CanvasHealthMonitor for health tracking.
//
// This organism composes:
// - canvusapi.Client instances for API interaction
// - CanvasHealthMonitor for health tracking (optional)
// - sync.RWMutex for thread-safe client management
//
// Usage:
//
//	manager := NewCanvasManager(nil, nil) // No health monitoring
//	manager.AddCanvas("canvas-1", client1)
//	client, ok := manager.GetClient("canvas-1")
//	if ok {
//	    info, _ := client.GetCanvasInfo()
//	}
type CanvasManager struct {
	mu            sync.RWMutex
	clients       map[string]*canvusapi.Client
	healthMonitor *CanvasHealthMonitor
	store         metrics.MetricsCollector
	logger        Logger
}

// CanvasManagerConfig configures the CanvasManager behavior.
type CanvasManagerConfig struct {
	// HealthMonitor for tracking canvas connection status (optional)
	HealthMonitor *CanvasHealthMonitor
	// MetricsStore for recording canvas metrics (optional)
	MetricsStore metrics.MetricsCollector
	// Logger for diagnostic output
	Logger Logger
}

// DefaultCanvasManagerConfig returns a default configuration.
func DefaultCanvasManagerConfig() CanvasManagerConfig {
	return CanvasManagerConfig{
		Logger: &defaultLogger{},
	}
}

// NewCanvasManager creates a new CanvasManager with the specified configuration.
// If config is nil, default configuration is used.
func NewCanvasManager(config *CanvasManagerConfig) *CanvasManager {
	if config == nil {
		defaultConfig := DefaultCanvasManagerConfig()
		config = &defaultConfig
	}

	logger := config.Logger
	if logger == nil {
		logger = &defaultLogger{}
	}

	return &CanvasManager{
		clients:       make(map[string]*canvusapi.Client),
		healthMonitor: config.HealthMonitor,
		store:         config.MetricsStore,
		logger:        logger,
	}
}

// AddCanvas registers a canvas client with the manager.
// If a canvas with the same ID already exists, it will be replaced.
// If a health monitor is configured, the canvas will be registered for health checks.
func (m *CanvasManager) AddCanvas(canvasID string, client *canvusapi.Client) error {
	if canvasID == "" {
		return fmt.Errorf("canvas ID cannot be empty")
	}
	if client == nil {
		return fmt.Errorf("client cannot be nil")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if replacing existing canvas
	if _, exists := m.clients[canvasID]; exists {
		m.logger.Printf("CanvasManager: replacing existing canvas %s", canvasID)
		// Unregister from health monitor if present
		if m.healthMonitor != nil {
			m.healthMonitor.UnregisterCanvas(canvasID)
		}
	}

	m.clients[canvasID] = client
	m.logger.Printf("CanvasManager: added canvas %s (%s)", canvasID, client.Server)

	// Register with health monitor if present
	if m.healthMonitor != nil {
		m.healthMonitor.RegisterCanvas(&canvasClientAdapter{client: client})
	}

	return nil
}

// RemoveCanvas unregisters a canvas client from the manager.
// Returns true if the canvas was found and removed, false otherwise.
func (m *CanvasManager) RemoveCanvas(canvasID string) bool {
	if canvasID == "" {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clients[canvasID]; !exists {
		return false
	}

	delete(m.clients, canvasID)
	m.logger.Printf("CanvasManager: removed canvas %s", canvasID)

	// Unregister from health monitor if present
	if m.healthMonitor != nil {
		m.healthMonitor.UnregisterCanvas(canvasID)
	}

	return true
}

// GetClient retrieves a canvas client by ID.
// Returns the client and true if found, nil and false otherwise.
func (m *CanvasManager) GetClient(canvasID string) (*canvusapi.Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[canvasID]
	return client, exists
}

// GetAllCanvases returns the IDs of all registered canvases.
func (m *CanvasManager) GetAllCanvases() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.clients))
	for id := range m.clients {
		ids = append(ids, id)
	}
	return ids
}

// GetClientCount returns the number of registered canvas clients.
func (m *CanvasManager) GetClientCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// MonitorHealth returns the health monitor associated with this manager.
// Returns nil if no health monitor was configured.
func (m *CanvasManager) MonitorHealth() *CanvasHealthMonitor {
	return m.healthMonitor
}

// SetHealthMonitor sets or replaces the health monitor.
// Existing canvases will be registered with the new monitor.
func (m *CanvasManager) SetHealthMonitor(monitor *CanvasHealthMonitor) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Unregister from old monitor
	if m.healthMonitor != nil {
		for canvasID := range m.clients {
			m.healthMonitor.UnregisterCanvas(canvasID)
		}
	}

	m.healthMonitor = monitor

	// Register existing canvases with new monitor
	if monitor != nil {
		for _, client := range m.clients {
			monitor.RegisterCanvas(&canvasClientAdapter{client: client})
		}
	}
}

// GetCanvasStatus returns the health status of a specific canvas.
// Returns the status and true if found, empty status and false otherwise.
func (m *CanvasManager) GetCanvasStatus(canvasID string) (metrics.CanvasStatus, bool) {
	if m.store == nil {
		return metrics.CanvasStatus{}, false
	}
	return m.store.GetCanvasStatus(canvasID)
}

// GetAllCanvasStatuses returns the health status of all registered canvases.
func (m *CanvasManager) GetAllCanvasStatuses() []metrics.CanvasStatus {
	if m.store == nil {
		return nil
	}

	m.mu.RLock()
	ids := make([]string, 0, len(m.clients))
	for id := range m.clients {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	statuses := make([]metrics.CanvasStatus, 0, len(ids))
	for _, id := range ids {
		if status, ok := m.store.GetCanvasStatus(id); ok {
			statuses = append(statuses, status)
		}
	}
	return statuses
}

// canvasClientAdapter adapts canvusapi.Client to the CanvasChecker interface.
type canvasClientAdapter struct {
	client *canvusapi.Client
}

// GetCanvasInfo implements CanvasChecker.GetCanvasInfo.
func (a *canvasClientAdapter) GetCanvasInfo() (map[string]interface{}, error) {
	return a.client.GetCanvasInfo()
}

// GetCanvasID implements CanvasChecker.GetCanvasID.
func (a *canvasClientAdapter) GetCanvasID() string {
	return a.client.CanvasID
}

// GetServerURL implements CanvasChecker.GetServerURL.
func (a *canvasClientAdapter) GetServerURL() string {
	return a.client.Server
}
