package webui

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewWSMessage(t *testing.T) {
	before := time.Now()
	msg := NewWSMessage(MessageTypeTaskUpdate, "test-data")
	after := time.Now()

	if msg.Type != MessageTypeTaskUpdate {
		t.Errorf("Type = %q, want %q", msg.Type, MessageTypeTaskUpdate)
	}
	if msg.Timestamp.Before(before) || msg.Timestamp.After(after) {
		t.Error("Timestamp should be between before and after test")
	}
	if msg.Data != "test-data" {
		t.Errorf("Data = %v, want 'test-data'", msg.Data)
	}
}

func TestWSMessage_MarshalJSON(t *testing.T) {
	msg := WSMessage{
		Type:      MessageTypeTaskUpdate,
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Data:      map[string]string{"key": "value"},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	if parsed["type"] != MessageTypeTaskUpdate {
		t.Errorf("Parsed type = %v, want %q", parsed["type"], MessageTypeTaskUpdate)
	}
}

func TestTaskUpdateData_JSON(t *testing.T) {
	data := TaskUpdateData{
		TaskID:   "task-123",
		TaskType: "pdf",
		Status:   "success",
		CanvasID: "canvas-456",
		Duration: 2*time.Second + 500*time.Millisecond,
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var parsed TaskUpdateData
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if parsed.TaskID != data.TaskID {
		t.Errorf("TaskID = %q, want %q", parsed.TaskID, data.TaskID)
	}
	if parsed.TaskType != data.TaskType {
		t.Errorf("TaskType = %q, want %q", parsed.TaskType, data.TaskType)
	}
	if parsed.Status != data.Status {
		t.Errorf("Status = %q, want %q", parsed.Status, data.Status)
	}
}

func TestGPUUpdateData_JSON(t *testing.T) {
	data := GPUUpdateData{
		Utilization:   75.5,
		Temperature:   65.0,
		MemoryUsed:    4 * 1024 * 1024 * 1024, // 4GB
		MemoryTotal:   8 * 1024 * 1024 * 1024, // 8GB
		MemoryPercent: 50.0,
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var parsed GPUUpdateData
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if parsed.Utilization != data.Utilization {
		t.Errorf("Utilization = %v, want %v", parsed.Utilization, data.Utilization)
	}
	if parsed.MemoryUsed != data.MemoryUsed {
		t.Errorf("MemoryUsed = %v, want %v", parsed.MemoryUsed, data.MemoryUsed)
	}
}

func TestCanvasUpdateData_JSON(t *testing.T) {
	now := time.Now()
	data := CanvasUpdateData{
		CanvasID:     "canvas-123",
		Name:         "My Canvas",
		Connected:    true,
		WidgetCount:  42,
		LastActivity: now,
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var parsed CanvasUpdateData
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if parsed.CanvasID != data.CanvasID {
		t.Errorf("CanvasID = %q, want %q", parsed.CanvasID, data.CanvasID)
	}
	if parsed.Connected != data.Connected {
		t.Errorf("Connected = %v, want %v", parsed.Connected, data.Connected)
	}
}

func TestSystemStatusData_JSON(t *testing.T) {
	data := SystemStatusData{
		Status:         "running",
		Uptime:         24 * time.Hour,
		ActiveTasks:    3,
		TotalProcessed: 1000,
		ErrorRate:      2.5,
		Version:        "1.0.0",
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var parsed SystemStatusData
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if parsed.Status != data.Status {
		t.Errorf("Status = %q, want %q", parsed.Status, data.Status)
	}
	if parsed.TotalProcessed != data.TotalProcessed {
		t.Errorf("TotalProcessed = %v, want %v", parsed.TotalProcessed, data.TotalProcessed)
	}
}

func TestMessageTypeConstants(t *testing.T) {
	// Verify constants are distinct and non-empty
	types := []string{
		MessageTypeTaskUpdate,
		MessageTypeGPUUpdate,
		MessageTypeCanvasUpdate,
		MessageTypeSystemStatus,
		MessageTypeError,
		MessageTypePing,
		MessageTypePong,
		MessageTypeInitial,
	}

	seen := make(map[string]bool)
	for _, msgType := range types {
		if msgType == "" {
			t.Error("Message type constant is empty")
		}
		if seen[msgType] {
			t.Errorf("Duplicate message type: %q", msgType)
		}
		seen[msgType] = true
	}
}

func TestNewTaskUpdateMessage(t *testing.T) {
	data := TaskUpdateData{TaskID: "test-123", Status: "success"}
	msg := NewTaskUpdateMessage(data)

	if msg.Type != MessageTypeTaskUpdate {
		t.Errorf("Type = %q, want %q", msg.Type, MessageTypeTaskUpdate)
	}
	if msg.Data.(TaskUpdateData).TaskID != "test-123" {
		t.Error("Data not correctly set")
	}
}

func TestNewGPUUpdateMessage(t *testing.T) {
	data := GPUUpdateData{Utilization: 75.0}
	msg := NewGPUUpdateMessage(data)

	if msg.Type != MessageTypeGPUUpdate {
		t.Errorf("Type = %q, want %q", msg.Type, MessageTypeGPUUpdate)
	}
}

func TestNewCanvasUpdateMessage(t *testing.T) {
	data := CanvasUpdateData{CanvasID: "canvas-1", Connected: true}
	msg := NewCanvasUpdateMessage(data)

	if msg.Type != MessageTypeCanvasUpdate {
		t.Errorf("Type = %q, want %q", msg.Type, MessageTypeCanvasUpdate)
	}
}

func TestNewSystemStatusMessage(t *testing.T) {
	data := SystemStatusData{Status: "running"}
	msg := NewSystemStatusMessage(data)

	if msg.Type != MessageTypeSystemStatus {
		t.Errorf("Type = %q, want %q", msg.Type, MessageTypeSystemStatus)
	}
}

func TestNewErrorMessage(t *testing.T) {
	msg := NewErrorMessage("ERR_001", "Something went wrong")

	if msg.Type != MessageTypeError {
		t.Errorf("Type = %q, want %q", msg.Type, MessageTypeError)
	}
	errData, ok := msg.Data.(ErrorData)
	if !ok {
		t.Fatal("Data is not ErrorData")
	}
	if errData.Code != "ERR_001" {
		t.Errorf("Code = %q, want 'ERR_001'", errData.Code)
	}
	if errData.Message != "Something went wrong" {
		t.Errorf("Message = %q, want 'Something went wrong'", errData.Message)
	}
}

func TestNewPingMessage(t *testing.T) {
	msg := NewPingMessage()

	if msg.Type != MessageTypePing {
		t.Errorf("Type = %q, want %q", msg.Type, MessageTypePing)
	}
	if msg.Data != nil {
		t.Errorf("Data = %v, want nil", msg.Data)
	}
}

func TestNewInitialMessage(t *testing.T) {
	data := InitialData{
		System: SystemStatusData{Status: "running"},
		Canvases: []CanvasUpdateData{
			{CanvasID: "canvas-1", Connected: true},
		},
	}
	msg := NewInitialMessage(data)

	if msg.Type != MessageTypeInitial {
		t.Errorf("Type = %q, want %q", msg.Type, MessageTypeInitial)
	}
}

func TestInitialData_JSON(t *testing.T) {
	data := InitialData{
		System: SystemStatusData{
			Status: "running",
			Uptime: 24 * time.Hour,
		},
		GPU: &GPUUpdateData{
			Utilization: 50.0,
			MemoryUsed:  4 * 1024 * 1024 * 1024,
		},
		Canvases: []CanvasUpdateData{
			{CanvasID: "c1", Name: "Canvas One", Connected: true},
			{CanvasID: "c2", Name: "Canvas Two", Connected: false},
		},
		RecentTasks: []TaskUpdateData{
			{TaskID: "t1", Status: "success"},
		},
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	var parsed InitialData
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if parsed.System.Status != "running" {
		t.Errorf("System.Status = %q, want 'running'", parsed.System.Status)
	}
	if parsed.GPU == nil {
		t.Error("GPU should not be nil")
	}
	if len(parsed.Canvases) != 2 {
		t.Errorf("len(Canvases) = %d, want 2", len(parsed.Canvases))
	}
	if len(parsed.RecentTasks) != 1 {
		t.Errorf("len(RecentTasks) = %d, want 1", len(parsed.RecentTasks))
	}
}

func TestInitialData_NilGPU(t *testing.T) {
	data := InitialData{
		System:   SystemStatusData{Status: "running"},
		GPU:      nil, // GPU not available
		Canvases: []CanvasUpdateData{},
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal error = %v", err)
	}

	// GPU field should be omitted from JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("Unmarshal error = %v", err)
	}

	if _, exists := parsed["gpu"]; exists {
		t.Error("gpu field should be omitted when nil")
	}
}
