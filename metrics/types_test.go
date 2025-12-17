package metrics

import (
	"encoding/json"
	"testing"
	"time"
)

// TestTaskRecordJSONMarshal verifies TaskRecord can be marshaled to JSON correctly.
func TestTaskRecordJSONMarshal(t *testing.T) {
	startTime := time.Date(2025, 12, 16, 10, 30, 0, 0, time.UTC)
	endTime := startTime.Add(2 * time.Second)

	record := TaskRecord{
		ID:        "task-123",
		Type:      TaskTypeNote,
		CanvasID:  "canvas-456",
		Status:    TaskStatusSuccess,
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  2 * time.Second,
		ErrorMsg:  "",
	}

	data, err := json.Marshal(record)
	if err != nil {
		t.Fatalf("Failed to marshal TaskRecord: %v", err)
	}

	// Verify key fields are present
	jsonStr := string(data)
	if !contains(jsonStr, "task-123") {
		t.Error("Marshaled JSON missing task ID")
	}
	if !contains(jsonStr, TaskTypeNote) {
		t.Error("Marshaled JSON missing task type")
	}
	if !contains(jsonStr, TaskStatusSuccess) {
		t.Error("Marshaled JSON missing status")
	}
}

// TestTaskRecordJSONUnmarshal verifies TaskRecord can be unmarshaled from JSON.
func TestTaskRecordJSONUnmarshal(t *testing.T) {
	jsonData := `{
		"id": "task-789",
		"type": "pdf",
		"canvas_id": "canvas-999",
		"status": "error",
		"start_time": "2025-12-16T10:30:00Z",
		"end_time": "2025-12-16T10:30:05Z",
		"duration": 5000000000,
		"error_msg": "timeout"
	}`

	var record TaskRecord
	err := json.Unmarshal([]byte(jsonData), &record)
	if err != nil {
		t.Fatalf("Failed to unmarshal TaskRecord: %v", err)
	}

	if record.ID != "task-789" {
		t.Errorf("Expected ID 'task-789', got '%s'", record.ID)
	}
	if record.Type != TaskTypePDF {
		t.Errorf("Expected Type 'pdf', got '%s'", record.Type)
	}
	if record.Status != TaskStatusError {
		t.Errorf("Expected Status 'error', got '%s'", record.Status)
	}
	if record.ErrorMsg != "timeout" {
		t.Errorf("Expected ErrorMsg 'timeout', got '%s'", record.ErrorMsg)
	}
}

// TestGPUMetricsJSONMarshal verifies GPUMetrics can be marshaled to JSON.
func TestGPUMetricsJSONMarshal(t *testing.T) {
	metrics := GPUMetrics{
		Utilization: 75.5,
		Temperature: 68.0,
		MemoryTotal: 8589934592, // 8GB
		MemoryUsed:  4294967296, // 4GB
		MemoryFree:  4294967296, // 4GB
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("Failed to marshal GPUMetrics: %v", err)
	}

	// Verify numeric fields are present
	jsonStr := string(data)
	if !contains(jsonStr, "75.5") {
		t.Error("Marshaled JSON missing utilization value")
	}
	if !contains(jsonStr, "68") {
		t.Error("Marshaled JSON missing temperature value")
	}
}

// TestCanvasStatusJSONMarshal verifies CanvasStatus can be marshaled to JSON.
func TestCanvasStatusJSONMarshal(t *testing.T) {
	status := CanvasStatus{
		ID:            "canvas-123",
		Name:          "Test Canvas",
		ServerURL:     "https://canvus.example.com",
		Connected:     true,
		LastUpdate:    time.Now(),
		WidgetCount:   42,
		RequestsToday: 150,
		SuccessRate:   98.5,
		Errors:        []string{"error1", "error2"},
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal CanvasStatus: %v", err)
	}

	jsonStr := string(data)
	if !contains(jsonStr, "canvas-123") {
		t.Error("Marshaled JSON missing canvas ID")
	}
	if !contains(jsonStr, "Test Canvas") {
		t.Error("Marshaled JSON missing canvas name")
	}
	if !contains(jsonStr, "true") {
		t.Error("Marshaled JSON missing connected status")
	}
}

// TestSystemStatusJSONMarshal verifies SystemStatus can be marshaled to JSON.
func TestSystemStatusJSONMarshal(t *testing.T) {
	status := SystemStatus{
		Health:    SystemHealthRunning,
		Version:   "v0.1.0",
		Uptime:    1 * time.Hour,
		LastCheck: time.Now(),
	}

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal SystemStatus: %v", err)
	}

	jsonStr := string(data)
	if !contains(jsonStr, SystemHealthRunning) {
		t.Error("Marshaled JSON missing health status")
	}
	if !contains(jsonStr, "v0.1.0") {
		t.Error("Marshaled JSON missing version")
	}
}

// TestTaskMetricsJSONMarshal verifies TaskMetrics can be marshaled to JSON.
func TestTaskMetricsJSONMarshal(t *testing.T) {
	metrics := TaskMetrics{
		TotalProcessed: 100,
		TotalSuccess:   95,
		TotalErrors:    5,
		ByType: map[string]*TaskTypeMetrics{
			TaskTypeNote: {
				Count:       50,
				SuccessRate: 98.0,
				AvgDuration: 1 * time.Second,
			},
			TaskTypePDF: {
				Count:       30,
				SuccessRate: 90.0,
				AvgDuration: 5 * time.Second,
			},
		},
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("Failed to marshal TaskMetrics: %v", err)
	}

	jsonStr := string(data)
	if !contains(jsonStr, "100") {
		t.Error("Marshaled JSON missing total processed")
	}
	if !contains(jsonStr, TaskTypeNote) {
		t.Error("Marshaled JSON missing task type note")
	}
}

// TestTaskRecordZeroValue verifies zero value TaskRecord behaves correctly.
func TestTaskRecordZeroValue(t *testing.T) {
	var record TaskRecord

	// Zero values should be valid
	if record.ID != "" {
		t.Error("Expected empty ID for zero value")
	}
	if record.Status != "" {
		t.Error("Expected empty Status for zero value")
	}
	if !record.StartTime.IsZero() {
		t.Error("Expected zero time for StartTime")
	}
	if !record.EndTime.IsZero() {
		t.Error("Expected zero time for EndTime")
	}
	if record.Duration != 0 {
		t.Error("Expected zero duration")
	}
}

// TestGPUMetricsZeroValue verifies zero value GPUMetrics behaves correctly.
func TestGPUMetricsZeroValue(t *testing.T) {
	var metrics GPUMetrics

	if metrics.Utilization != 0 {
		t.Error("Expected zero utilization")
	}
	if metrics.Temperature != 0 {
		t.Error("Expected zero temperature")
	}
	if metrics.MemoryTotal != 0 {
		t.Error("Expected zero total memory")
	}
	if metrics.MemoryUsed != 0 {
		t.Error("Expected zero used memory")
	}
	if metrics.MemoryFree != 0 {
		t.Error("Expected zero free memory")
	}
}

// TestTaskStatusConstants verifies task status constants are correct.
func TestTaskStatusConstants(t *testing.T) {
	if TaskStatusSuccess != "success" {
		t.Errorf("Expected TaskStatusSuccess to be 'success', got '%s'", TaskStatusSuccess)
	}
	if TaskStatusError != "error" {
		t.Errorf("Expected TaskStatusError to be 'error', got '%s'", TaskStatusError)
	}
	if TaskStatusProcessing != "processing" {
		t.Errorf("Expected TaskStatusProcessing to be 'processing', got '%s'", TaskStatusProcessing)
	}
}

// TestSystemHealthConstants verifies system health constants are correct.
func TestSystemHealthConstants(t *testing.T) {
	if SystemHealthRunning != "running" {
		t.Errorf("Expected SystemHealthRunning to be 'running', got '%s'", SystemHealthRunning)
	}
	if SystemHealthError != "error" {
		t.Errorf("Expected SystemHealthError to be 'error', got '%s'", SystemHealthError)
	}
	if SystemHealthStopped != "stopped" {
		t.Errorf("Expected SystemHealthStopped to be 'stopped', got '%s'", SystemHealthStopped)
	}
}

// TestTaskTypeConstants verifies task type constants are correct.
func TestTaskTypeConstants(t *testing.T) {
	if TaskTypeNote != "note" {
		t.Errorf("Expected TaskTypeNote to be 'note', got '%s'", TaskTypeNote)
	}
	if TaskTypePDF != "pdf" {
		t.Errorf("Expected TaskTypePDF to be 'pdf', got '%s'", TaskTypePDF)
	}
	if TaskTypeImage != "image" {
		t.Errorf("Expected TaskTypeImage to be 'image', got '%s'", TaskTypeImage)
	}
	if TaskTypeImageAnalysis != "image_analysis" {
		t.Errorf("Expected TaskTypeImageAnalysis to be 'image_analysis', got '%s'", TaskTypeImageAnalysis)
	}
	if TaskTypeCanvasAnalysis != "canvas_analysis" {
		t.Errorf("Expected TaskTypeCanvasAnalysis to be 'canvas_analysis', got '%s'", TaskTypeCanvasAnalysis)
	}
	if TaskTypeHandwriting != "handwriting" {
		t.Errorf("Expected TaskTypeHandwriting to be 'handwriting', got '%s'", TaskTypeHandwriting)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
