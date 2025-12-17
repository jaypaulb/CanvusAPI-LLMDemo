// Package webui provides the web-based user interface for CanvusLocalLLM.
// This file contains WebSocket message types and constants.
package webui

import (
	"encoding/json"
	"time"
)

// Message type constants for WebSocket communication.
// These define the types of real-time updates sent to connected clients.
const (
	// MessageTypeTaskUpdate indicates a task status change (started, completed, error).
	MessageTypeTaskUpdate = "task_update"

	// MessageTypeGPUUpdate indicates GPU metrics have been updated.
	MessageTypeGPUUpdate = "gpu_update"

	// MessageTypeCanvasUpdate indicates a canvas status change.
	MessageTypeCanvasUpdate = "canvas_update"

	// MessageTypeSystemStatus indicates overall system health status change.
	MessageTypeSystemStatus = "system_status"

	// MessageTypeError indicates a server-side error message.
	MessageTypeError = "error"

	// MessageTypePing is a keep-alive message from the server.
	MessageTypePing = "ping"

	// MessageTypePong is a keep-alive response from the client.
	MessageTypePong = "pong"

	// MessageTypeInitial contains the initial state snapshot on connection.
	MessageTypeInitial = "initial"
)

// WSMessage is the base structure for all WebSocket messages.
// It uses a common envelope format with type-specific data in the Data field.
//
// This is a pure data structure atom with no behavior beyond JSON marshaling.
type WSMessage struct {
	// Type identifies the message kind (use MessageType* constants)
	Type string `json:"type"`

	// Timestamp is when the message was created
	Timestamp time.Time `json:"timestamp"`

	// Data contains the type-specific payload (decoded based on Type)
	Data interface{} `json:"data,omitempty"`
}

// NewWSMessage creates a new WebSocket message with the current timestamp.
//
// Parameters:
//   - msgType: The message type (use MessageType* constants)
//   - data: The type-specific payload
//
// Returns:
//   - WSMessage: Ready-to-send message
func NewWSMessage(msgType string, data interface{}) WSMessage {
	return WSMessage{
		Type:      msgType,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// MarshalJSON serializes the message to JSON bytes.
// This is a convenience method for sending messages over WebSocket.
func (m WSMessage) MarshalJSON() ([]byte, error) {
	type Alias WSMessage
	return json.Marshal(Alias(m))
}

// TaskUpdateData contains details for a task status update.
type TaskUpdateData struct {
	// TaskID is the unique identifier for the task
	TaskID string `json:"task_id"`

	// TaskType identifies the kind of task (note, pdf, image, etc.)
	TaskType string `json:"task_type"`

	// Status is the current state (processing, success, error)
	Status string `json:"status"`

	// CanvasID identifies which canvas this task belongs to
	CanvasID string `json:"canvas_id,omitempty"`

	// Duration is how long the task took (only set on completion)
	Duration time.Duration `json:"duration,omitempty"`

	// Error contains error details if Status is "error"
	Error string `json:"error,omitempty"`
}

// GPUUpdateData contains current GPU metrics.
type GPUUpdateData struct {
	// Utilization is the GPU usage percentage (0-100)
	Utilization float64 `json:"utilization"`

	// Temperature is the GPU temperature in Celsius
	Temperature float64 `json:"temperature"`

	// MemoryUsed is the amount of GPU memory in use (bytes)
	MemoryUsed int64 `json:"memory_used"`

	// MemoryTotal is the total GPU memory (bytes)
	MemoryTotal int64 `json:"memory_total"`

	// MemoryPercent is the memory usage percentage (0-100)
	MemoryPercent float64 `json:"memory_percent"`
}

// CanvasUpdateData contains canvas connection status.
type CanvasUpdateData struct {
	// CanvasID is the unique identifier for the canvas
	CanvasID string `json:"canvas_id"`

	// Name is the human-readable canvas name
	Name string `json:"name"`

	// Connected indicates if the canvas is currently connected
	Connected bool `json:"connected"`

	// WidgetCount is the number of widgets on the canvas
	WidgetCount int `json:"widget_count"`

	// LastActivity is when the last AI request was processed
	LastActivity time.Time `json:"last_activity,omitempty"`
}

// SystemStatusData contains overall system health information.
type SystemStatusData struct {
	// Status indicates system state: "running", "error", "stopped"
	Status string `json:"status"`

	// Uptime is how long the system has been running
	Uptime time.Duration `json:"uptime"`

	// ActiveTasks is the count of currently processing tasks
	ActiveTasks int `json:"active_tasks"`

	// TotalProcessed is the total count of tasks processed since start
	TotalProcessed int64 `json:"total_processed"`

	// ErrorRate is the percentage of failed tasks (0-100)
	ErrorRate float64 `json:"error_rate"`

	// Version is the application version string
	Version string `json:"version,omitempty"`
}

// ErrorData contains error information sent to clients.
type ErrorData struct {
	// Code is an application-specific error code
	Code string `json:"code"`

	// Message is a human-readable error description
	Message string `json:"message"`
}

// InitialData contains the complete state snapshot sent on connection.
type InitialData struct {
	// System contains current system status
	System SystemStatusData `json:"system"`

	// GPU contains current GPU metrics (nil if GPU unavailable)
	GPU *GPUUpdateData `json:"gpu,omitempty"`

	// Canvases contains status for all monitored canvases
	Canvases []CanvasUpdateData `json:"canvases"`

	// RecentTasks contains the last N task records
	RecentTasks []TaskUpdateData `json:"recent_tasks"`
}

// Helper functions for creating common messages

// NewTaskUpdateMessage creates a task update message.
func NewTaskUpdateMessage(data TaskUpdateData) WSMessage {
	return NewWSMessage(MessageTypeTaskUpdate, data)
}

// NewGPUUpdateMessage creates a GPU metrics update message.
func NewGPUUpdateMessage(data GPUUpdateData) WSMessage {
	return NewWSMessage(MessageTypeGPUUpdate, data)
}

// NewCanvasUpdateMessage creates a canvas status update message.
func NewCanvasUpdateMessage(data CanvasUpdateData) WSMessage {
	return NewWSMessage(MessageTypeCanvasUpdate, data)
}

// NewSystemStatusMessage creates a system status message.
func NewSystemStatusMessage(data SystemStatusData) WSMessage {
	return NewWSMessage(MessageTypeSystemStatus, data)
}

// NewErrorMessage creates an error message.
func NewErrorMessage(code, message string) WSMessage {
	return NewWSMessage(MessageTypeError, ErrorData{Code: code, Message: message})
}

// NewPingMessage creates a ping keep-alive message.
func NewPingMessage() WSMessage {
	return NewWSMessage(MessageTypePing, nil)
}

// NewInitialMessage creates the initial state snapshot message.
func NewInitialMessage(data InitialData) WSMessage {
	return NewWSMessage(MessageTypeInitial, data)
}
