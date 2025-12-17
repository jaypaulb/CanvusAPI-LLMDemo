package metrics

import (
	"sync"
	"testing"
	"time"
)

// MockCollector is a simple in-memory implementation of MetricsCollector for testing.
// This validates that the interface can be implemented and used correctly.
type MockCollector struct {
	mu             sync.RWMutex
	tasks          []TaskRecord
	taskMetrics    TaskMetrics
	gpuMetrics     GPUMetrics
	canvasStatuses map[string]CanvasStatus
	systemStatus   SystemStatus
}

// NewMockCollector creates a new mock collector for testing.
func NewMockCollector() *MockCollector {
	return &MockCollector{
		tasks:          make([]TaskRecord, 0),
		canvasStatuses: make(map[string]CanvasStatus),
		taskMetrics: TaskMetrics{
			ByType: make(map[string]*TaskTypeMetrics),
		},
	}
}

func (m *MockCollector) RecordTask(task TaskRecord) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks = append(m.tasks, task)
}

func (m *MockCollector) GetTaskMetrics() TaskMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.taskMetrics
}

func (m *MockCollector) GetRecentTasks(limit int) []TaskRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.tasks) <= limit {
		result := make([]TaskRecord, len(m.tasks))
		copy(result, m.tasks)
		return result
	}

	start := len(m.tasks) - limit
	result := make([]TaskRecord, limit)
	copy(result, m.tasks[start:])
	return result
}

func (m *MockCollector) UpdateGPUMetrics(gpu GPUMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gpuMetrics = gpu
}

func (m *MockCollector) GetGPUMetrics() GPUMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gpuMetrics
}

func (m *MockCollector) UpdateCanvasStatus(status CanvasStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.canvasStatuses[status.ID] = status
}

func (m *MockCollector) GetCanvasStatus(canvasID string) (CanvasStatus, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status, ok := m.canvasStatuses[canvasID]
	return status, ok
}

func (m *MockCollector) GetAllCanvasStatuses() []CanvasStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	statuses := make([]CanvasStatus, 0, len(m.canvasStatuses))
	for _, status := range m.canvasStatuses {
		statuses = append(statuses, status)
	}
	return statuses
}

func (m *MockCollector) GetSystemStatus() SystemStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.systemStatus
}

// TestMetricsCollectorInterface verifies that MockCollector implements MetricsCollector.
func TestMetricsCollectorInterface(t *testing.T) {
	var _ MetricsCollector = (*MockCollector)(nil)
}

// TestRecordTask verifies task recording functionality.
func TestRecordTask(t *testing.T) {
	collector := NewMockCollector()

	task := TaskRecord{
		ID:        "task-1",
		Type:      TaskTypeNote,
		CanvasID:  "canvas-1",
		Status:    TaskStatusSuccess,
		StartTime: time.Now(),
		EndTime:   time.Now().Add(time.Second),
		Duration:  time.Second,
	}

	collector.RecordTask(task)

	tasks := collector.GetRecentTasks(10)
	if len(tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(tasks))
	}

	if tasks[0].ID != "task-1" {
		t.Errorf("Expected task ID 'task-1', got '%s'", tasks[0].ID)
	}
}

// TestGetRecentTasksLimit verifies that GetRecentTasks respects the limit.
func TestGetRecentTasksLimit(t *testing.T) {
	collector := NewMockCollector()

	// Add 10 tasks
	for i := 0; i < 10; i++ {
		task := TaskRecord{
			ID:       string(rune('0' + i)),
			Type:     TaskTypeNote,
			CanvasID: "canvas-1",
			Status:   TaskStatusSuccess,
		}
		collector.RecordTask(task)
	}

	// Request only 5 most recent
	tasks := collector.GetRecentTasks(5)

	if len(tasks) != 5 {
		t.Errorf("Expected 5 tasks, got %d", len(tasks))
	}
}

// TestGetRecentTasksLimitExceedsTotal verifies behavior when limit exceeds total tasks.
func TestGetRecentTasksLimitExceedsTotal(t *testing.T) {
	collector := NewMockCollector()

	// Add 3 tasks
	for i := 0; i < 3; i++ {
		task := TaskRecord{
			ID:       string(rune('0' + i)),
			Type:     TaskTypeNote,
			CanvasID: "canvas-1",
			Status:   TaskStatusSuccess,
		}
		collector.RecordTask(task)
	}

	// Request 10 (more than available)
	tasks := collector.GetRecentTasks(10)

	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasks))
	}
}

// TestGPUMetrics verifies GPU metrics update and retrieval.
func TestGPUMetrics(t *testing.T) {
	collector := NewMockCollector()

	gpu := GPUMetrics{
		Utilization: 75.5,
		Temperature: 65.0,
		MemoryTotal: 8589934592, // 8GB in bytes
		MemoryUsed:  4294967296, // 4GB in bytes
		MemoryFree:  4294967296, // 4GB in bytes
	}

	collector.UpdateGPUMetrics(gpu)

	retrieved := collector.GetGPUMetrics()

	if retrieved.Utilization != 75.5 {
		t.Errorf("Expected utilization 75.5, got %f", retrieved.Utilization)
	}

	if retrieved.MemoryUsed != 4294967296 {
		t.Errorf("Expected memory used 4294967296, got %d", retrieved.MemoryUsed)
	}
}

// TestCanvasStatus verifies canvas status update and retrieval.
func TestCanvasStatus(t *testing.T) {
	collector := NewMockCollector()

	status := CanvasStatus{
		ID:            "canvas-1",
		Name:          "Test Canvas",
		ServerURL:     "https://test.canvus.io",
		Connected:     true,
		LastUpdate:    time.Now(),
		WidgetCount:   42,
		RequestsToday: 100,
		SuccessRate:   95.5,
	}

	collector.UpdateCanvasStatus(status)

	retrieved, ok := collector.GetCanvasStatus("canvas-1")

	if !ok {
		t.Fatal("Expected to find canvas status")
	}

	if retrieved.ID != "canvas-1" {
		t.Errorf("Expected canvas ID 'canvas-1', got '%s'", retrieved.ID)
	}

	if retrieved.WidgetCount != 42 {
		t.Errorf("Expected widget count 42, got %d", retrieved.WidgetCount)
	}
}

// TestCanvasStatusNotFound verifies behavior when canvas is not found.
func TestCanvasStatusNotFound(t *testing.T) {
	collector := NewMockCollector()

	_, ok := collector.GetCanvasStatus("nonexistent")

	if ok {
		t.Error("Expected not to find canvas status")
	}
}

// TestGetAllCanvasStatuses verifies retrieval of all canvas statuses.
func TestGetAllCanvasStatuses(t *testing.T) {
	collector := NewMockCollector()

	status1 := CanvasStatus{ID: "canvas-1", Name: "Canvas 1"}
	status2 := CanvasStatus{ID: "canvas-2", Name: "Canvas 2"}

	collector.UpdateCanvasStatus(status1)
	collector.UpdateCanvasStatus(status2)

	statuses := collector.GetAllCanvasStatuses()

	if len(statuses) != 2 {
		t.Errorf("Expected 2 canvas statuses, got %d", len(statuses))
	}
}

// TestSystemStatus verifies system status retrieval.
func TestSystemStatus(t *testing.T) {
	collector := NewMockCollector()

	// Default system status should have zero values
	status := collector.GetSystemStatus()

	if status.Health != "" {
		t.Errorf("Expected empty health, got '%s'", status.Health)
	}
}

// TestConcurrentAccess verifies thread-safety of the collector.
func TestConcurrentAccess(t *testing.T) {
	collector := NewMockCollector()

	// Launch multiple goroutines to record tasks concurrently
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			task := TaskRecord{
				ID:       string(rune('0' + (id % 10))),
				Type:     TaskTypeNote,
				CanvasID: "canvas-1",
				Status:   TaskStatusSuccess,
			}
			collector.RecordTask(task)
		}(i)
	}

	wg.Wait()

	tasks := collector.GetRecentTasks(1000)
	if len(tasks) != 100 {
		t.Errorf("Expected 100 tasks, got %d", len(tasks))
	}
}
