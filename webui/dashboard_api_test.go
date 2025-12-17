package webui

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go_backend/metrics"
)

// mockMetricsCollector is a test implementation of MetricsCollector.
type mockMetricsCollector struct {
	systemStatus   metrics.SystemStatus
	canvasStatuses []metrics.CanvasStatus
	taskRecords    []metrics.TaskRecord
	taskMetrics    metrics.TaskMetrics
	gpuMetrics     metrics.GPUMetrics
}

func newMockMetricsCollector() *mockMetricsCollector {
	return &mockMetricsCollector{
		systemStatus: metrics.SystemStatus{
			Health:    metrics.SystemHealthRunning,
			Version:   "1.0.0",
			Uptime:    time.Hour + 30*time.Minute,
			LastCheck: time.Now(),
		},
		canvasStatuses: []metrics.CanvasStatus{
			{
				ID:        "canvas-1",
				Name:      "Test Canvas 1",
				Connected: true,
			},
			{
				ID:        "canvas-2",
				Name:      "Test Canvas 2",
				Connected: false,
			},
		},
		taskRecords: []metrics.TaskRecord{
			{
				ID:       "task-1",
				Type:     metrics.TaskTypeNote,
				Status:   metrics.TaskStatusSuccess,
				Duration: 100 * time.Millisecond,
			},
			{
				ID:       "task-2",
				Type:     metrics.TaskTypePDF,
				Status:   metrics.TaskStatusSuccess,
				Duration: 500 * time.Millisecond,
			},
		},
		taskMetrics: metrics.TaskMetrics{
			TotalProcessed: 100,
			TotalSuccess:   90,
			TotalErrors:    10,
			ByType: map[string]*metrics.TaskTypeMetrics{
				metrics.TaskTypeNote: {
					Count:       50,
					SuccessRate: 95.0,
					AvgDuration: 100 * time.Millisecond,
				},
			},
		},
	}
}

func (m *mockMetricsCollector) RecordTask(task metrics.TaskRecord) {
	m.taskRecords = append(m.taskRecords, task)
}

func (m *mockMetricsCollector) GetTaskMetrics() metrics.TaskMetrics {
	return m.taskMetrics
}

func (m *mockMetricsCollector) GetRecentTasks(limit int) []metrics.TaskRecord {
	if limit > len(m.taskRecords) {
		limit = len(m.taskRecords)
	}
	return m.taskRecords[:limit]
}

func (m *mockMetricsCollector) UpdateGPUMetrics(gpu metrics.GPUMetrics) {
	m.gpuMetrics = gpu
}

func (m *mockMetricsCollector) GetGPUMetrics() metrics.GPUMetrics {
	return m.gpuMetrics
}

func (m *mockMetricsCollector) UpdateCanvasStatus(status metrics.CanvasStatus) {
	for i, c := range m.canvasStatuses {
		if c.ID == status.ID {
			m.canvasStatuses[i] = status
			return
		}
	}
	m.canvasStatuses = append(m.canvasStatuses, status)
}

func (m *mockMetricsCollector) GetCanvasStatus(canvasID string) (metrics.CanvasStatus, bool) {
	for _, c := range m.canvasStatuses {
		if c.ID == canvasID {
			return c, true
		}
	}
	return metrics.CanvasStatus{}, false
}

func (m *mockMetricsCollector) GetAllCanvasStatuses() []metrics.CanvasStatus {
	return m.canvasStatuses
}

func (m *mockMetricsCollector) GetSystemStatus() metrics.SystemStatus {
	return m.systemStatus
}

func TestNewDashboardAPI(t *testing.T) {
	t.Run("creates API with default config", func(t *testing.T) {
		mock := newMockMetricsCollector()
		config := DefaultDashboardAPIConfig()
		api := NewDashboardAPI(mock, nil, config)

		if api == nil {
			t.Fatal("expected non-nil API")
		}

		if api.defaultLimit != 20 {
			t.Errorf("expected defaultLimit 20, got %d", api.defaultLimit)
		}

		if api.maxLimit != 100 {
			t.Errorf("expected maxLimit 100, got %d", api.maxLimit)
		}
	})

	t.Run("handles invalid config values", func(t *testing.T) {
		mock := newMockMetricsCollector()
		config := DashboardAPIConfig{
			DefaultLimit: 0,
			MaxLimit:     -1,
		}
		api := NewDashboardAPI(mock, nil, config)

		if api.defaultLimit != 20 {
			t.Errorf("expected defaultLimit 20 (corrected), got %d", api.defaultLimit)
		}

		if api.maxLimit != 100 {
			t.Errorf("expected maxLimit 100 (corrected), got %d", api.maxLimit)
		}
	})
}

func TestHandleStatus(t *testing.T) {
	t.Run("returns system status", func(t *testing.T) {
		mock := newMockMetricsCollector()
		config := DefaultDashboardAPIConfig()
		config.VersionInfo = VersionInfo{
			Version:   "1.0.0",
			BuildDate: "2024-01-01",
			GitCommit: "abc123",
		}
		api := NewDashboardAPI(mock, nil, config)

		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		w := httptest.NewRecorder()

		api.HandleStatus(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response StatusResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Health != metrics.SystemHealthRunning {
			t.Errorf("expected health 'running', got '%s'", response.Health)
		}

		if response.Version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got '%s'", response.Version)
		}

		if response.GPUAvail {
			t.Error("expected GPU not available when collector is nil")
		}
	})

	t.Run("rejects non-GET requests", func(t *testing.T) {
		mock := newMockMetricsCollector()
		api := NewDashboardAPI(mock, nil, DefaultDashboardAPIConfig())

		req := httptest.NewRequest(http.MethodPost, "/api/status", nil)
		w := httptest.NewRecorder()

		api.HandleStatus(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", w.Code)
		}
	})

	t.Run("includes GPU availability when collector present", func(t *testing.T) {
		mock := newMockMetricsCollector()
		gpuConfig := metrics.GPUCollectorConfig{
			CollectionInterval: 10 * time.Millisecond,
			HistorySize:        5,
		}
		mockReader := metrics.NewMockGPUReader(metrics.GPUMetrics{Utilization: 50.0})
		gpuCollector := metrics.NewGPUCollectorWithReader(gpuConfig, mockReader, nil)
		gpuCollector.Start()
		defer gpuCollector.Stop()

		// Wait for collection
		time.Sleep(50 * time.Millisecond)

		api := NewDashboardAPI(mock, gpuCollector, DefaultDashboardAPIConfig())

		req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
		w := httptest.NewRecorder()

		api.HandleStatus(w, req)

		var response StatusResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !response.GPUAvail {
			t.Error("expected GPU to be available")
		}
	})
}

func TestHandleCanvases(t *testing.T) {
	t.Run("returns all canvas statuses", func(t *testing.T) {
		mock := newMockMetricsCollector()
		api := NewDashboardAPI(mock, nil, DefaultDashboardAPIConfig())

		req := httptest.NewRequest(http.MethodGet, "/api/canvases", nil)
		w := httptest.NewRecorder()

		api.HandleCanvases(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response CanvasesResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Count != 2 {
			t.Errorf("expected 2 canvases, got %d", response.Count)
		}

		if len(response.Canvases) != 2 {
			t.Errorf("expected 2 canvas items, got %d", len(response.Canvases))
		}
	})

	t.Run("rejects non-GET requests", func(t *testing.T) {
		mock := newMockMetricsCollector()
		api := NewDashboardAPI(mock, nil, DefaultDashboardAPIConfig())

		req := httptest.NewRequest(http.MethodPost, "/api/canvases", nil)
		w := httptest.NewRecorder()

		api.HandleCanvases(w, req)

		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", w.Code)
		}
	})
}

func TestHandleTasks(t *testing.T) {
	t.Run("returns recent tasks with default limit", func(t *testing.T) {
		mock := newMockMetricsCollector()
		api := NewDashboardAPI(mock, nil, DefaultDashboardAPIConfig())

		req := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
		w := httptest.NewRecorder()

		api.HandleTasks(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response TasksResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Mock has 2 tasks, but limit is 20
		if response.Count != 2 {
			t.Errorf("expected 2 tasks, got %d", response.Count)
		}

		if response.Limit != 20 {
			t.Errorf("expected limit 20, got %d", response.Limit)
		}
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		mock := newMockMetricsCollector()
		api := NewDashboardAPI(mock, nil, DefaultDashboardAPIConfig())

		req := httptest.NewRequest(http.MethodGet, "/api/tasks?limit=1", nil)
		w := httptest.NewRecorder()

		api.HandleTasks(w, req)

		var response TasksResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Count != 1 {
			t.Errorf("expected 1 task, got %d", response.Count)
		}

		if response.Limit != 1 {
			t.Errorf("expected limit 1, got %d", response.Limit)
		}
	})

	t.Run("caps limit at max", func(t *testing.T) {
		mock := newMockMetricsCollector()
		config := DashboardAPIConfig{
			DefaultLimit: 10,
			MaxLimit:     50,
		}
		api := NewDashboardAPI(mock, nil, config)

		req := httptest.NewRequest(http.MethodGet, "/api/tasks?limit=1000", nil)
		w := httptest.NewRecorder()

		api.HandleTasks(w, req)

		var response TasksResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Limit != 50 {
			t.Errorf("expected limit capped at 50, got %d", response.Limit)
		}
	})

	t.Run("ignores invalid limit parameter", func(t *testing.T) {
		mock := newMockMetricsCollector()
		api := NewDashboardAPI(mock, nil, DefaultDashboardAPIConfig())

		req := httptest.NewRequest(http.MethodGet, "/api/tasks?limit=invalid", nil)
		w := httptest.NewRecorder()

		api.HandleTasks(w, req)

		var response TasksResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Should use default
		if response.Limit != 20 {
			t.Errorf("expected default limit 20, got %d", response.Limit)
		}
	})
}

func TestHandleMetrics(t *testing.T) {
	t.Run("returns task metrics", func(t *testing.T) {
		mock := newMockMetricsCollector()
		api := NewDashboardAPI(mock, nil, DefaultDashboardAPIConfig())

		req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
		w := httptest.NewRecorder()

		api.HandleMetrics(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response MetricsResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.TotalProcessed != 100 {
			t.Errorf("expected total processed 100, got %d", response.TotalProcessed)
		}

		if response.TotalSuccess != 90 {
			t.Errorf("expected total success 90, got %d", response.TotalSuccess)
		}

		if response.TotalErrors != 10 {
			t.Errorf("expected total errors 10, got %d", response.TotalErrors)
		}

		// Success rate: 90/100 * 100 = 90%
		if response.SuccessRate != 90.0 {
			t.Errorf("expected success rate 90.0, got %f", response.SuccessRate)
		}
	})

	t.Run("handles zero total processed", func(t *testing.T) {
		mock := newMockMetricsCollector()
		mock.taskMetrics = metrics.TaskMetrics{
			TotalProcessed: 0,
			TotalSuccess:   0,
			TotalErrors:    0,
		}
		api := NewDashboardAPI(mock, nil, DefaultDashboardAPIConfig())

		req := httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
		w := httptest.NewRecorder()

		api.HandleMetrics(w, req)

		var response MetricsResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		// Should not panic on division by zero
		if response.SuccessRate != 0 {
			t.Errorf("expected success rate 0 when no tasks, got %f", response.SuccessRate)
		}
	})
}

func TestHandleGPU(t *testing.T) {
	t.Run("returns unavailable when no GPU collector", func(t *testing.T) {
		mock := newMockMetricsCollector()
		api := NewDashboardAPI(mock, nil, DefaultDashboardAPIConfig())

		req := httptest.NewRequest(http.MethodGet, "/api/gpu", nil)
		w := httptest.NewRecorder()

		api.HandleGPU(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var response GPUResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Available {
			t.Error("expected GPU not available")
		}

		if response.Error == "" {
			t.Error("expected error message when GPU collector is nil")
		}
	})

	t.Run("returns current GPU metrics", func(t *testing.T) {
		mock := newMockMetricsCollector()
		gpuConfig := metrics.GPUCollectorConfig{
			CollectionInterval: 10 * time.Millisecond,
			HistorySize:        5,
		}
		mockReader := metrics.NewMockGPUReader(metrics.GPUMetrics{
			Utilization: 75.0,
			Temperature: 65.0,
			MemoryTotal: 8 * 1024 * 1024 * 1024,
			MemoryUsed:  4 * 1024 * 1024 * 1024,
			MemoryFree:  4 * 1024 * 1024 * 1024,
		})
		gpuCollector := metrics.NewGPUCollectorWithReader(gpuConfig, mockReader, nil)
		gpuCollector.Start()
		defer gpuCollector.Stop()

		// Wait for collection
		time.Sleep(50 * time.Millisecond)

		api := NewDashboardAPI(mock, gpuCollector, DefaultDashboardAPIConfig())

		req := httptest.NewRequest(http.MethodGet, "/api/gpu", nil)
		w := httptest.NewRecorder()

		api.HandleGPU(w, req)

		var response GPUResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !response.Available {
			t.Error("expected GPU to be available")
		}

		if response.Current == nil {
			t.Fatal("expected current metrics to be set")
		}

		if response.Current.Utilization != 75.0 {
			t.Errorf("expected utilization 75.0, got %f", response.Current.Utilization)
		}
	})

	t.Run("includes history when requested", func(t *testing.T) {
		mock := newMockMetricsCollector()
		gpuConfig := metrics.GPUCollectorConfig{
			CollectionInterval: 10 * time.Millisecond,
			HistorySize:        100,
		}
		mockReader := metrics.NewMockGPUReader(metrics.GPUMetrics{Utilization: 50.0})
		gpuCollector := metrics.NewGPUCollectorWithReader(gpuConfig, mockReader, nil)
		gpuCollector.Start()
		defer gpuCollector.Stop()

		// Wait for multiple collections
		time.Sleep(100 * time.Millisecond)

		api := NewDashboardAPI(mock, gpuCollector, DefaultDashboardAPIConfig())

		req := httptest.NewRequest(http.MethodGet, "/api/gpu?history=5", nil)
		w := httptest.NewRecorder()

		api.HandleGPU(w, req)

		var response GPUResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.History == nil {
			t.Error("expected history to be included")
		}

		if len(response.History) == 0 {
			t.Error("expected non-empty history")
		}
	})

	t.Run("handles GPU error state", func(t *testing.T) {
		mock := newMockMetricsCollector()
		gpuConfig := metrics.GPUCollectorConfig{
			CollectionInterval: 10 * time.Millisecond,
			HistorySize:        5,
		}
		mockReader := metrics.NewMockGPUReader(metrics.GPUMetrics{})
		mockReader.SetError(errors.New("GPU driver error"))
		gpuCollector := metrics.NewGPUCollectorWithReader(gpuConfig, mockReader, nil)
		gpuCollector.Start()
		defer gpuCollector.Stop()

		// Wait for collection attempt
		time.Sleep(50 * time.Millisecond)

		api := NewDashboardAPI(mock, gpuCollector, DefaultDashboardAPIConfig())

		req := httptest.NewRequest(http.MethodGet, "/api/gpu", nil)
		w := httptest.NewRecorder()

		api.HandleGPU(w, req)

		var response GPUResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if response.Available {
			t.Error("expected GPU not available when error")
		}

		if response.Error == "" {
			t.Error("expected error message")
		}
	})
}

func TestRegisterRoutes(t *testing.T) {
	mock := newMockMetricsCollector()
	api := NewDashboardAPI(mock, nil, DefaultDashboardAPIConfig())

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	routes := []string{
		"/api/status",
		"/api/canvases",
		"/api/tasks",
		"/api/metrics",
		"/api/gpu",
	}

	for _, route := range routes {
		req := httptest.NewRequest(http.MethodGet, route, nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		// Should not be 404
		if w.Code == http.StatusNotFound {
			t.Errorf("route %s should be registered", route)
		}
	}
}

func TestDashboardAPIFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{time.Hour + 30*time.Minute + 15*time.Second, "1h30m15s"},
		{2*time.Hour + 5*time.Minute, "2h5m0s"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("formatDuration(%v) = %s, want %s", tt.duration, result, tt.expected)
			}
		})
	}
}

func TestContentTypeHeader(t *testing.T) {
	mock := newMockMetricsCollector()
	api := NewDashboardAPI(mock, nil, DefaultDashboardAPIConfig())

	endpoints := []string{
		"/api/status",
		"/api/canvases",
		"/api/tasks",
		"/api/metrics",
		"/api/gpu",
	}

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	for _, endpoint := range endpoints {
		req := httptest.NewRequest(http.MethodGet, endpoint, nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		contentType := w.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("%s: expected Content-Type 'application/json', got '%s'", endpoint, contentType)
		}
	}
}
