package metrics

import (
	"sync"
	"testing"
	"time"
)

func TestNewMetricsStore(t *testing.T) {
	t.Run("creates store with default config", func(t *testing.T) {
		config := DefaultStoreConfig()
		startTime := time.Now()
		store := NewMetricsStore(config, startTime)

		if store == nil {
			t.Fatal("expected non-nil store")
		}
		if store.taskCap != 100 {
			t.Errorf("expected task capacity 100, got %d", store.taskCap)
		}
		if store.version != "0.0.0" {
			t.Errorf("expected version 0.0.0, got %s", store.version)
		}
	})

	t.Run("creates store with custom config", func(t *testing.T) {
		config := StoreConfig{
			TaskHistoryCapacity: 50,
			Version:             "1.2.3",
		}
		startTime := time.Now()
		store := NewMetricsStore(config, startTime)

		if store.taskCap != 50 {
			t.Errorf("expected task capacity 50, got %d", store.taskCap)
		}
		if store.version != "1.2.3" {
			t.Errorf("expected version 1.2.3, got %s", store.version)
		}
	})

	t.Run("handles zero capacity by defaulting to 100", func(t *testing.T) {
		config := StoreConfig{TaskHistoryCapacity: 0}
		store := NewMetricsStore(config, time.Now())

		if store.taskCap != 100 {
			t.Errorf("expected default capacity 100, got %d", store.taskCap)
		}
	})
}

func TestMetricsStore_RecordTask(t *testing.T) {
	t.Run("records a single task", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		task := TaskRecord{
			ID:        "task-1",
			Type:      TaskTypeNote,
			CanvasID:  "canvas-1",
			Status:    TaskStatusSuccess,
			StartTime: time.Now().Add(-time.Second),
			EndTime:   time.Now(),
			Duration:  time.Second,
		}

		store.RecordTask(task)

		// Verify task was recorded
		tasks := store.GetRecentTasks(10)
		if len(tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(tasks))
		}
		if tasks[0].ID != "task-1" {
			t.Errorf("expected task ID 'task-1', got '%s'", tasks[0].ID)
		}
	})

	t.Run("tracks success count", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		store.RecordTask(TaskRecord{ID: "1", Status: TaskStatusSuccess, Type: TaskTypeNote})
		store.RecordTask(TaskRecord{ID: "2", Status: TaskStatusSuccess, Type: TaskTypeNote})
		store.RecordTask(TaskRecord{ID: "3", Status: TaskStatusError, Type: TaskTypeNote})

		metrics := store.GetTaskMetrics()
		if metrics.TotalProcessed != 3 {
			t.Errorf("expected 3 total, got %d", metrics.TotalProcessed)
		}
		if metrics.TotalSuccess != 2 {
			t.Errorf("expected 2 success, got %d", metrics.TotalSuccess)
		}
		if metrics.TotalErrors != 1 {
			t.Errorf("expected 1 error, got %d", metrics.TotalErrors)
		}
	})

	t.Run("tracks per-type statistics", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		store.RecordTask(TaskRecord{ID: "1", Type: TaskTypeNote, Status: TaskStatusSuccess, Duration: time.Second})
		store.RecordTask(TaskRecord{ID: "2", Type: TaskTypeNote, Status: TaskStatusSuccess, Duration: 2 * time.Second})
		store.RecordTask(TaskRecord{ID: "3", Type: TaskTypePDF, Status: TaskStatusError, Duration: 5 * time.Second})

		metrics := store.GetTaskMetrics()

		noteStats, ok := metrics.ByType[TaskTypeNote]
		if !ok {
			t.Fatal("expected note stats to exist")
		}
		if noteStats.Count != 2 {
			t.Errorf("expected 2 note tasks, got %d", noteStats.Count)
		}
		if noteStats.SuccessRate != 100.0 {
			t.Errorf("expected 100%% note success rate, got %.1f%%", noteStats.SuccessRate)
		}
		expectedAvg := 1500 * time.Millisecond // (1s + 2s) / 2
		if noteStats.AvgDuration != expectedAvg {
			t.Errorf("expected avg duration %v, got %v", expectedAvg, noteStats.AvgDuration)
		}

		pdfStats, ok := metrics.ByType[TaskTypePDF]
		if !ok {
			t.Fatal("expected pdf stats to exist")
		}
		if pdfStats.SuccessRate != 0.0 {
			t.Errorf("expected 0%% pdf success rate, got %.1f%%", pdfStats.SuccessRate)
		}
	})
}

func TestGetRecentTasks(t *testing.T) {
	t.Run("returns empty slice when no tasks", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		tasks := store.GetRecentTasks(10)
		if len(tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(tasks))
		}
	})

	t.Run("returns limited number of tasks", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		for i := 0; i < 10; i++ {
			store.RecordTask(TaskRecord{ID: string(rune('0' + i))})
		}

		tasks := store.GetRecentTasks(5)
		if len(tasks) != 5 {
			t.Errorf("expected 5 tasks, got %d", len(tasks))
		}
	})

	t.Run("returns all tasks when limit exceeds available", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		store.RecordTask(TaskRecord{ID: "1"})
		store.RecordTask(TaskRecord{ID: "2"})
		store.RecordTask(TaskRecord{ID: "3"})

		tasks := store.GetRecentTasks(100)
		if len(tasks) != 3 {
			t.Errorf("expected 3 tasks, got %d", len(tasks))
		}
	})

	t.Run("handles zero and negative limit", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())
		store.RecordTask(TaskRecord{ID: "1"})

		if len(store.GetRecentTasks(0)) != 0 {
			t.Error("expected empty slice for limit 0")
		}
		if len(store.GetRecentTasks(-1)) != 0 {
			t.Error("expected empty slice for negative limit")
		}
	})

	t.Run("handles circular buffer wraparound", func(t *testing.T) {
		config := StoreConfig{TaskHistoryCapacity: 3}
		store := NewMetricsStore(config, time.Now())

		// Add 5 tasks to a buffer of size 3
		store.RecordTask(TaskRecord{ID: "1"})
		store.RecordTask(TaskRecord{ID: "2"})
		store.RecordTask(TaskRecord{ID: "3"})
		store.RecordTask(TaskRecord{ID: "4"})
		store.RecordTask(TaskRecord{ID: "5"})

		// Should only have the last 3
		tasks := store.GetRecentTasks(10)
		if len(tasks) != 3 {
			t.Fatalf("expected 3 tasks, got %d", len(tasks))
		}

		// Should be in order: oldest to newest
		expectedIDs := []string{"3", "4", "5"}
		for i, task := range tasks {
			if task.ID != expectedIDs[i] {
				t.Errorf("task %d: expected ID '%s', got '%s'", i, expectedIDs[i], task.ID)
			}
		}
	})
}

func TestMetricsStore_GPUMetrics(t *testing.T) {
	t.Run("returns zero value when not set", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		gpu := store.GetGPUMetrics()
		if gpu.Utilization != 0 {
			t.Errorf("expected 0 utilization, got %f", gpu.Utilization)
		}
	})

	t.Run("updates and retrieves GPU metrics", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		expected := GPUMetrics{
			Utilization: 75.5,
			Temperature: 65.0,
			MemoryTotal: 8 * 1024 * 1024 * 1024,
			MemoryUsed:  4 * 1024 * 1024 * 1024,
			MemoryFree:  4 * 1024 * 1024 * 1024,
		}

		store.UpdateGPUMetrics(expected)
		actual := store.GetGPUMetrics()

		if actual.Utilization != expected.Utilization {
			t.Errorf("expected utilization %f, got %f", expected.Utilization, actual.Utilization)
		}
		if actual.Temperature != expected.Temperature {
			t.Errorf("expected temperature %f, got %f", expected.Temperature, actual.Temperature)
		}
		if actual.MemoryTotal != expected.MemoryTotal {
			t.Errorf("expected memory total %d, got %d", expected.MemoryTotal, actual.MemoryTotal)
		}
	})

	t.Run("overwrites previous metrics", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		store.UpdateGPUMetrics(GPUMetrics{Utilization: 50.0})
		store.UpdateGPUMetrics(GPUMetrics{Utilization: 75.0})

		gpu := store.GetGPUMetrics()
		if gpu.Utilization != 75.0 {
			t.Errorf("expected utilization 75.0, got %f", gpu.Utilization)
		}
	})
}

func TestMetricsStore_CanvasStatus(t *testing.T) {
	t.Run("returns empty slice when no canvases", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		statuses := store.GetAllCanvasStatuses()
		if len(statuses) != 0 {
			t.Errorf("expected 0 canvases, got %d", len(statuses))
		}
	})

	t.Run("GetCanvasStatus returns false for unknown canvas", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		_, ok := store.GetCanvasStatus("unknown")
		if ok {
			t.Error("expected ok to be false for unknown canvas")
		}
	})

	t.Run("updates and retrieves canvas status", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		status := CanvasStatus{
			ID:            "canvas-1",
			Name:          "Test Canvas",
			ServerURL:     "https://example.com",
			Connected:     true,
			LastUpdate:    time.Now(),
			WidgetCount:   42,
			RequestsToday: 100,
			SuccessRate:   95.5,
		}

		store.UpdateCanvasStatus(status)

		retrieved, ok := store.GetCanvasStatus("canvas-1")
		if !ok {
			t.Fatal("expected to find canvas-1")
		}
		if retrieved.Name != "Test Canvas" {
			t.Errorf("expected name 'Test Canvas', got '%s'", retrieved.Name)
		}
		if retrieved.WidgetCount != 42 {
			t.Errorf("expected widget count 42, got %d", retrieved.WidgetCount)
		}
	})

	t.Run("GetAllCanvasStatuses returns all canvases", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		store.UpdateCanvasStatus(CanvasStatus{ID: "canvas-1", Name: "Canvas 1"})
		store.UpdateCanvasStatus(CanvasStatus{ID: "canvas-2", Name: "Canvas 2"})
		store.UpdateCanvasStatus(CanvasStatus{ID: "canvas-3", Name: "Canvas 3"})

		statuses := store.GetAllCanvasStatuses()
		if len(statuses) != 3 {
			t.Errorf("expected 3 canvases, got %d", len(statuses))
		}

		// Verify all IDs are present (order not guaranteed)
		ids := make(map[string]bool)
		for _, s := range statuses {
			ids[s.ID] = true
		}
		for _, id := range []string{"canvas-1", "canvas-2", "canvas-3"} {
			if !ids[id] {
				t.Errorf("expected canvas %s to be present", id)
			}
		}
	})

	t.Run("updates existing canvas", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		store.UpdateCanvasStatus(CanvasStatus{ID: "canvas-1", Connected: true})
		store.UpdateCanvasStatus(CanvasStatus{ID: "canvas-1", Connected: false})

		status, _ := store.GetCanvasStatus("canvas-1")
		if status.Connected {
			t.Error("expected canvas to be disconnected after update")
		}
	})
}

func TestGetSystemStatus(t *testing.T) {
	t.Run("returns running status with no canvases", func(t *testing.T) {
		config := StoreConfig{Version: "1.0.0"}
		store := NewMetricsStore(config, time.Now())

		status := store.GetSystemStatus()
		if status.Health != SystemHealthRunning {
			t.Errorf("expected health 'running', got '%s'", status.Health)
		}
		if status.Version != "1.0.0" {
			t.Errorf("expected version '1.0.0', got '%s'", status.Version)
		}
	})

	t.Run("returns running when at least one canvas connected", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		store.UpdateCanvasStatus(CanvasStatus{ID: "1", Connected: false})
		store.UpdateCanvasStatus(CanvasStatus{ID: "2", Connected: true})
		store.UpdateCanvasStatus(CanvasStatus{ID: "3", Connected: false})

		status := store.GetSystemStatus()
		if status.Health != SystemHealthRunning {
			t.Errorf("expected health 'running', got '%s'", status.Health)
		}
	})

	t.Run("returns error when all canvases disconnected", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		store.UpdateCanvasStatus(CanvasStatus{ID: "1", Connected: false})
		store.UpdateCanvasStatus(CanvasStatus{ID: "2", Connected: false})

		status := store.GetSystemStatus()
		if status.Health != SystemHealthError {
			t.Errorf("expected health 'error', got '%s'", status.Health)
		}
	})

	t.Run("calculates uptime correctly", func(t *testing.T) {
		startTime := time.Now().Add(-5 * time.Minute)
		store := NewMetricsStore(DefaultStoreConfig(), startTime)

		status := store.GetSystemStatus()

		// Uptime should be approximately 5 minutes
		if status.Uptime < 4*time.Minute || status.Uptime > 6*time.Minute {
			t.Errorf("expected uptime ~5min, got %v", status.Uptime)
		}
	})
}

func TestMetricsStore_ConcurrentAccess(t *testing.T) {
	t.Run("handles concurrent task recording", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		var wg sync.WaitGroup
		numGoroutines := 100
		tasksPerGoroutine := 10

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < tasksPerGoroutine; j++ {
					store.RecordTask(TaskRecord{
						ID:     string(rune(goroutineID*tasksPerGoroutine + j)),
						Type:   TaskTypeNote,
						Status: TaskStatusSuccess,
					})
				}
			}(i)
		}

		wg.Wait()

		metrics := store.GetTaskMetrics()
		expected := int64(numGoroutines * tasksPerGoroutine)
		if metrics.TotalProcessed != expected {
			t.Errorf("expected %d tasks, got %d", expected, metrics.TotalProcessed)
		}
	})

	t.Run("handles concurrent reads and writes", func(t *testing.T) {
		store := NewMetricsStore(DefaultStoreConfig(), time.Now())

		var wg sync.WaitGroup

		// Writers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					store.RecordTask(TaskRecord{ID: string(rune(id*100 + j)), Status: TaskStatusSuccess})
					store.UpdateGPUMetrics(GPUMetrics{Utilization: float64(j)})
					store.UpdateCanvasStatus(CanvasStatus{ID: "canvas", Connected: true})
				}
			}(i)
		}

		// Readers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					_ = store.GetRecentTasks(10)
					_ = store.GetTaskMetrics()
					_ = store.GetGPUMetrics()
					_ = store.GetAllCanvasStatuses()
					_ = store.GetSystemStatus()
				}
			}()
		}

		wg.Wait()
		// If we get here without deadlock or panic, the test passes
	})
}

func TestImplementsMetricsCollector(t *testing.T) {
	// This test verifies at compile time that MetricsStore implements MetricsCollector
	var _ MetricsCollector = (*MetricsStore)(nil)
}
