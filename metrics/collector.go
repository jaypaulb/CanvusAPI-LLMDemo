// Package metrics provides the MetricsCollector interface for aggregating metrics.
// This is a molecule that composes the atom-level types from types.go.
package metrics

import "time"

// MetricsCollector defines the interface for collecting metrics from various sources.
// This molecule aggregates TaskRecord, GPUMetrics, and CanvasStatus atoms to provide
// a unified interface for metric collection.
//
// Implementation strategy:
// - Implementations should aggregate data from task processing, GPU monitoring, and canvas status
// - Methods should be concurrency-safe
// - Zero values should be returned for unavailable metrics
type MetricsCollector interface {
	// RecordTask logs a completed task execution.
	// Aggregates TaskRecord atoms into the metrics system.
	RecordTask(task TaskRecord)

	// GetTaskMetrics returns aggregated task processing statistics.
	// Composes multiple TaskRecord atoms into TaskMetrics summary.
	GetTaskMetrics() TaskMetrics

	// GetRecentTasks returns the N most recent task records.
	// Provides access to individual TaskRecord atoms.
	GetRecentTasks(limit int) []TaskRecord

	// UpdateGPUMetrics updates the current GPU metrics snapshot.
	// Records the current GPUMetrics atom state.
	UpdateGPUMetrics(gpu GPUMetrics)

	// GetGPUMetrics returns the current GPU metrics.
	// Retrieves the latest GPUMetrics atom.
	GetGPUMetrics() GPUMetrics

	// UpdateCanvasStatus updates the status for a specific canvas.
	// Records the current CanvasStatus atom for a canvas.
	UpdateCanvasStatus(status CanvasStatus)

	// GetCanvasStatus returns the status for a specific canvas by ID.
	// Retrieves the CanvasStatus atom for a given canvas.
	GetCanvasStatus(canvasID string) (CanvasStatus, bool)

	// GetAllCanvasStatuses returns status for all monitored canvases.
	// Provides access to all CanvasStatus atoms.
	GetAllCanvasStatuses() []CanvasStatus

	// GetSystemStatus returns the overall system health status.
	// Composes SystemStatus atom from collected metrics.
	GetSystemStatus() SystemStatus
}

// TaskBroadcaster defines the interface for broadcasting task updates to connected clients.
// This allows the Monitor to send real-time task status updates without depending on webui package.
// The webui.WebSocketBroadcaster implements this interface via BroadcastTaskUpdateFromMetrics.
type TaskBroadcaster interface {
	// BroadcastTaskUpdateFromMetrics sends a task status update to all connected WebSocket clients.
	// The data should contain task ID, type, status, and optional duration/error.
	BroadcastTaskUpdateFromMetrics(data TaskBroadcastData)
}

// TaskBroadcastData contains the information needed for a task broadcast.
// This is a minimal struct that can be converted to webui.TaskUpdateData.
type TaskBroadcastData struct {
	// TaskID is the unique identifier for the task
	TaskID string
	// TaskType identifies the kind of task (note, pdf, image, canvas_analysis, handwriting)
	TaskType string
	// Status is the current state (processing, success, error)
	Status string
	// CanvasID identifies which canvas this task belongs to
	CanvasID string
	// Duration is how long the task took (only set on completion)
	Duration time.Duration
	// Error contains error details if Status is "error"
	Error string
}
