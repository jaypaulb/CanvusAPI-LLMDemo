// Package metrics provides the MetricsStore organism for in-memory metrics storage.
// This file contains the MetricsStore which implements the MetricsCollector interface.
package metrics

import (
	"sync"
	"time"
)

// MetricsStore is an in-memory storage organism for all dashboard metrics.
// It implements the MetricsCollector interface and provides thread-safe
// access to task records, GPU metrics, canvas statuses, and system health.
//
// This is an organism-level component that composes:
// - CircularBuffer (from webui package) for task history
// - sync.RWMutex for thread-safety
// - metrics types (TaskRecord, GPUMetrics, CanvasStatus, etc.)
//
// Usage:
//
//	store := NewMetricsStore(100, time.Now()) // 100 task history capacity
//	store.RecordTask(task)
//	metrics := store.GetTaskMetrics()
type MetricsStore struct {
	mu sync.RWMutex

	// Task tracking
	taskHistory []TaskRecord // Circular buffer of recent tasks
	taskCap     int          // Maximum tasks to retain
	taskHead    int          // Write index
	taskSize    int          // Current number of tasks

	// Task aggregation
	totalTasks   int64
	totalSuccess int64
	totalErrors  int64
	taskByType   map[string]*taskTypeStats // Per-type statistics

	// GPU metrics (latest snapshot)
	gpuMetrics GPUMetrics

	// Canvas statuses (keyed by canvas ID)
	canvasStatuses map[string]CanvasStatus

	// System metadata
	startTime time.Time
	version   string
}

// taskTypeStats holds per-type aggregation data
type taskTypeStats struct {
	count         int64
	successCount  int64
	totalDuration time.Duration
}

// StoreConfig configures the MetricsStore behavior.
type StoreConfig struct {
	// TaskHistoryCapacity is the max number of tasks to retain in history
	TaskHistoryCapacity int
	// Version is the application version string
	Version string
}

// DefaultStoreConfig returns a default configuration.
func DefaultStoreConfig() StoreConfig {
	return StoreConfig{
		TaskHistoryCapacity: 100,
		Version:             "0.0.0",
	}
}

// NewMetricsStore creates a new MetricsStore with the specified configuration.
// The startTime is used to calculate uptime.
func NewMetricsStore(config StoreConfig, startTime time.Time) *MetricsStore {
	cap := config.TaskHistoryCapacity
	if cap < 1 {
		cap = 100
	}

	return &MetricsStore{
		taskHistory:    make([]TaskRecord, cap),
		taskCap:        cap,
		taskHead:       0,
		taskSize:       0,
		taskByType:     make(map[string]*taskTypeStats),
		canvasStatuses: make(map[string]CanvasStatus),
		startTime:      startTime,
		version:        config.Version,
	}
}

// RecordTask logs a completed task execution.
// This implements part of the MetricsCollector interface.
func (s *MetricsStore) RecordTask(task TaskRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Add to circular buffer
	s.taskHistory[s.taskHead] = task
	s.taskHead = (s.taskHead + 1) % s.taskCap
	if s.taskSize < s.taskCap {
		s.taskSize++
	}

	// Update aggregations
	s.totalTasks++

	if task.Status == TaskStatusSuccess {
		s.totalSuccess++
	} else if task.Status == TaskStatusError {
		s.totalErrors++
	}

	// Update per-type stats
	stats, ok := s.taskByType[task.Type]
	if !ok {
		stats = &taskTypeStats{}
		s.taskByType[task.Type] = stats
	}
	stats.count++
	if task.Status == TaskStatusSuccess {
		stats.successCount++
	}
	stats.totalDuration += task.Duration
}

// GetTaskMetrics returns aggregated task processing statistics.
// This implements part of the MetricsCollector interface.
func (s *MetricsStore) GetTaskMetrics() TaskMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics := TaskMetrics{
		TotalProcessed: s.totalTasks,
		TotalSuccess:   s.totalSuccess,
		TotalErrors:    s.totalErrors,
		ByType:         make(map[string]*TaskTypeMetrics),
	}

	for taskType, stats := range s.taskByType {
		var successRate float64
		if stats.count > 0 {
			successRate = float64(stats.successCount) / float64(stats.count) * 100
		}

		var avgDuration time.Duration
		if stats.count > 0 {
			avgDuration = stats.totalDuration / time.Duration(stats.count)
		}

		metrics.ByType[taskType] = &TaskTypeMetrics{
			Count:       stats.count,
			SuccessRate: successRate,
			AvgDuration: avgDuration,
		}
	}

	return metrics
}

// GetRecentTasks returns the N most recent task records.
// If limit exceeds available tasks, all available are returned.
// This implements part of the MetricsCollector interface.
func (s *MetricsStore) GetRecentTasks(limit int) []TaskRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || s.taskSize == 0 {
		return []TaskRecord{}
	}

	if limit > s.taskSize {
		limit = s.taskSize
	}

	// Calculate the starting index for the most recent 'limit' items
	result := make([]TaskRecord, limit)
	for i := 0; i < limit; i++ {
		// Work backwards from head to get most recent first
		idx := (s.taskHead - limit + i + s.taskCap) % s.taskCap
		result[i] = s.taskHistory[idx]
	}

	return result
}

// UpdateGPUMetrics updates the current GPU metrics snapshot.
// This implements part of the MetricsCollector interface.
func (s *MetricsStore) UpdateGPUMetrics(gpu GPUMetrics) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gpuMetrics = gpu
}

// GetGPUMetrics returns the current GPU metrics.
// This implements part of the MetricsCollector interface.
func (s *MetricsStore) GetGPUMetrics() GPUMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.gpuMetrics
}

// UpdateCanvasStatus updates the status for a specific canvas.
// This implements part of the MetricsCollector interface.
func (s *MetricsStore) UpdateCanvasStatus(status CanvasStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.canvasStatuses[status.ID] = status
}

// GetCanvasStatus returns the status for a specific canvas by ID.
// This implements part of the MetricsCollector interface.
func (s *MetricsStore) GetCanvasStatus(canvasID string) (CanvasStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.canvasStatuses[canvasID]
	return status, ok
}

// GetAllCanvasStatuses returns status for all monitored canvases.
// This implements part of the MetricsCollector interface.
func (s *MetricsStore) GetAllCanvasStatuses() []CanvasStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]CanvasStatus, 0, len(s.canvasStatuses))
	for _, status := range s.canvasStatuses {
		result = append(result, status)
	}
	return result
}

// GetSystemStatus returns the overall system health status.
// This implements part of the MetricsCollector interface.
func (s *MetricsStore) GetSystemStatus() SystemStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Determine health based on connected canvases
	health := SystemHealthRunning
	hasConnected := false
	for _, canvas := range s.canvasStatuses {
		if canvas.Connected {
			hasConnected = true
			break
		}
	}

	// If we have canvases registered but none are connected, report error
	if len(s.canvasStatuses) > 0 && !hasConnected {
		health = SystemHealthError
	}

	return SystemStatus{
		Health:    health,
		Version:   s.version,
		Uptime:    time.Since(s.startTime),
		LastCheck: time.Now(),
	}
}

// Verify MetricsStore implements MetricsCollector interface
var _ MetricsCollector = (*MetricsStore)(nil)
