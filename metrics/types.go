// Package metrics provides pure data types for the web UI metrics system.
// This file contains atom-level type definitions with no behavior.
package metrics

import "time"

// TaskRecord represents a single task execution record.
// This is a pure data structure for tracking individual AI processing operations.
type TaskRecord struct {
	// ID is the unique identifier for this task
	ID string `json:"id"`

	// Type identifies the kind of task (e.g., "note", "pdf", "image", "canvas_analysis")
	Type string `json:"type"`

	// CanvasID identifies which canvas this task belongs to
	CanvasID string `json:"canvas_id"`

	// Status indicates the current state: "success", "error", "processing"
	Status string `json:"status"`

	// StartTime is when the task began execution
	StartTime time.Time `json:"start_time"`

	// EndTime is when the task completed (zero value if still processing)
	EndTime time.Time `json:"end_time,omitempty"`

	// Duration is the total execution time
	Duration time.Duration `json:"duration"`

	// ErrorMsg contains error details if Status is "error"
	ErrorMsg string `json:"error_msg,omitempty"`
}

// GPUMetrics represents GPU resource utilization metrics.
// This differs from core.GPUMetrics in that it uses bytes instead of MB
// and includes MemoryFree for the web UI dashboard.
type GPUMetrics struct {
	// Utilization is the GPU utilization percentage (0-100)
	Utilization float64 `json:"utilization"`

	// Temperature is the GPU temperature in Celsius
	Temperature float64 `json:"temperature"`

	// MemoryTotal is the total available GPU memory in bytes
	MemoryTotal int64 `json:"memory_total"`

	// MemoryUsed is the amount of GPU memory currently in use (bytes)
	MemoryUsed int64 `json:"memory_used"`

	// MemoryFree is the amount of available GPU memory (bytes)
	MemoryFree int64 `json:"memory_free"`
}

// CanvasStatus represents the connection and health status of a monitored canvas.
// This is a pure data structure with no behavior.
type CanvasStatus struct {
	// ID is the unique identifier for the canvas
	ID string `json:"id"`

	// Name is the human-readable canvas name
	Name string `json:"name"`

	// ServerURL is the Canvus server URL for this canvas
	ServerURL string `json:"server_url"`

	// Connected indicates if the canvas is currently connected
	Connected bool `json:"connected"`

	// LastUpdate is the timestamp of the last activity on this canvas
	LastUpdate time.Time `json:"last_update"`

	// WidgetCount is the number of widgets currently on the canvas
	WidgetCount int `json:"widget_count"`

	// RequestsToday is the count of AI requests processed today
	RequestsToday int64 `json:"requests_today"`

	// SuccessRate is the percentage of successful operations (0-100)
	SuccessRate float64 `json:"success_rate"`

	// Errors contains recent error messages (limited to last N errors)
	Errors []string `json:"errors,omitempty"`
}

// SystemStatus represents the overall system health and status.
// This is a pure data structure with no behavior.
type SystemStatus struct {
	// Health indicates the system state: "running", "error", "stopped"
	Health string `json:"health"`

	// Version is the application version string
	Version string `json:"version"`

	// Uptime is the duration since the application started
	Uptime time.Duration `json:"uptime"`

	// LastCheck is the timestamp of the last health check
	LastCheck time.Time `json:"last_check"`
}

// TaskMetrics represents aggregated task processing statistics.
// This is a pure data structure with no behavior.
type TaskMetrics struct {
	// TotalProcessed is the total number of tasks processed
	TotalProcessed int64 `json:"total_processed"`

	// TotalSuccess is the count of successfully completed tasks
	TotalSuccess int64 `json:"total_success"`

	// TotalErrors is the count of failed tasks
	TotalErrors int64 `json:"total_errors"`

	// ByType contains per-type statistics
	ByType map[string]*TaskTypeMetrics `json:"by_type"`
}

// TaskTypeMetrics represents statistics for a specific task type.
// This is a pure data structure with no behavior.
type TaskTypeMetrics struct {
	// Count is the total number of tasks of this type
	Count int64 `json:"count"`

	// SuccessRate is the percentage of successful operations (0-100)
	SuccessRate float64 `json:"success_rate"`

	// AvgDuration is the average execution time for this task type
	AvgDuration time.Duration `json:"avg_duration"`
}

// Status constants for TaskRecord
const (
	TaskStatusSuccess    = "success"
	TaskStatusError      = "error"
	TaskStatusProcessing = "processing"
)

// Health constants for SystemStatus
const (
	SystemHealthRunning = "running"
	SystemHealthError   = "error"
	SystemHealthStopped = "stopped"
)

// Task type constants
const (
	TaskTypeNote           = "note"
	TaskTypePDF            = "pdf"
	TaskTypeImage          = "image"
	TaskTypeImageAnalysis  = "image_analysis"
	TaskTypeCanvasAnalysis = "canvas_analysis"
	TaskTypeHandwriting    = "handwriting"
)
