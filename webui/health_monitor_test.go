package webui

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go_backend/metrics"
)

// mockCanvasChecker implements CanvasChecker for testing.
type mockCanvasChecker struct {
	canvasID    string
	serverURL   string
	healthy     bool
	checkCount  int32
	info        map[string]interface{}
	err         error
	mu          sync.Mutex
}

func newMockCanvasChecker(id, serverURL string, healthy bool) *mockCanvasChecker {
	return &mockCanvasChecker{
		canvasID:  id,
		serverURL: serverURL,
		healthy:   healthy,
		info: map[string]interface{}{
			"name": "Test Canvas " + id,
		},
	}
}

func (m *mockCanvasChecker) GetCanvasInfo() (map[string]interface{}, error) {
	atomic.AddInt32(&m.checkCount, 1)

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.healthy {
		if m.err != nil {
			return nil, m.err
		}
		return nil, errors.New("canvas not reachable")
	}
	return m.info, nil
}

func (m *mockCanvasChecker) GetCanvasID() string {
	return m.canvasID
}

func (m *mockCanvasChecker) GetServerURL() string {
	return m.serverURL
}

func (m *mockCanvasChecker) SetHealthy(healthy bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthy = healthy
}

func (m *mockCanvasChecker) GetCheckCount() int32 {
	return atomic.LoadInt32(&m.checkCount)
}

// healthTestLogger captures log messages for testing.
type healthTestLogger struct {
	messages []string
	mu       sync.Mutex
}

func (l *healthTestLogger) Printf(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// We don't actually format here, just note that logging happened
	l.messages = append(l.messages, format)
}

func TestNewCanvasHealthMonitor(t *testing.T) {
	t.Run("creates monitor with default config", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		config := DefaultHealthMonitorConfig()
		monitor := NewCanvasHealthMonitor(store, config)

		if monitor == nil {
			t.Fatal("expected non-nil monitor")
		}
		if monitor.checkInterval != 30*time.Second {
			t.Errorf("expected 30s check interval, got %v", monitor.checkInterval)
		}
	})

	t.Run("creates monitor with custom config", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		config := HealthMonitorConfig{
			CheckInterval: 10 * time.Second,
		}
		monitor := NewCanvasHealthMonitor(store, config)

		if monitor.checkInterval != 10*time.Second {
			t.Errorf("expected 10s check interval, got %v", monitor.checkInterval)
		}
	})

	t.Run("defaults zero interval to 30s", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		config := HealthMonitorConfig{
			CheckInterval: 0,
		}
		monitor := NewCanvasHealthMonitor(store, config)

		if monitor.checkInterval != 30*time.Second {
			t.Errorf("expected default 30s interval, got %v", monitor.checkInterval)
		}
	})
}

func TestRegisterCanvas(t *testing.T) {
	t.Run("registers canvas and initializes status", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		monitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		canvas := newMockCanvasChecker("canvas-1", "https://example.com", true)
		monitor.RegisterCanvas(canvas)

		status, ok := store.GetCanvasStatus("canvas-1")
		if !ok {
			t.Fatal("expected canvas to be registered")
		}
		if status.Connected {
			t.Error("expected canvas to be initially disconnected")
		}
		if status.ServerURL != "https://example.com" {
			t.Errorf("expected server URL 'https://example.com', got '%s'", status.ServerURL)
		}
	})

	t.Run("GetRegisteredCanvases returns all IDs", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		monitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		monitor.RegisterCanvas(newMockCanvasChecker("canvas-1", "url1", true))
		monitor.RegisterCanvas(newMockCanvasChecker("canvas-2", "url2", true))
		monitor.RegisterCanvas(newMockCanvasChecker("canvas-3", "url3", true))

		ids := monitor.GetRegisteredCanvases()
		if len(ids) != 3 {
			t.Errorf("expected 3 canvases, got %d", len(ids))
		}
	})
}

func TestUnregisterCanvas(t *testing.T) {
	t.Run("removes canvas from monitoring", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		monitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		monitor.RegisterCanvas(newMockCanvasChecker("canvas-1", "url", true))
		monitor.UnregisterCanvas("canvas-1")

		ids := monitor.GetRegisteredCanvases()
		if len(ids) != 0 {
			t.Errorf("expected 0 canvases after unregister, got %d", len(ids))
		}
	})
}

func TestCheckNow(t *testing.T) {
	t.Run("performs immediate health check on all canvases", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		monitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		canvas1 := newMockCanvasChecker("canvas-1", "url1", true)
		canvas2 := newMockCanvasChecker("canvas-2", "url2", true)

		monitor.RegisterCanvas(canvas1)
		monitor.RegisterCanvas(canvas2)
		monitor.CheckNow()

		// Both canvases should have been checked
		if canvas1.GetCheckCount() != 1 {
			t.Errorf("expected canvas1 to be checked once, got %d", canvas1.GetCheckCount())
		}
		if canvas2.GetCheckCount() != 1 {
			t.Errorf("expected canvas2 to be checked once, got %d", canvas2.GetCheckCount())
		}

		// Both should now show as connected
		status1, _ := store.GetCanvasStatus("canvas-1")
		status2, _ := store.GetCanvasStatus("canvas-2")
		if !status1.Connected {
			t.Error("expected canvas-1 to be connected")
		}
		if !status2.Connected {
			t.Error("expected canvas-2 to be connected")
		}
	})

	t.Run("marks unhealthy canvases as disconnected", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		monitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		canvas := newMockCanvasChecker("canvas-1", "url1", false)
		monitor.RegisterCanvas(canvas)
		monitor.CheckNow()

		status, _ := store.GetCanvasStatus("canvas-1")
		if status.Connected {
			t.Error("expected canvas to be disconnected")
		}
	})

	t.Run("extracts canvas name from response", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		monitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		canvas := newMockCanvasChecker("canvas-1", "url1", true)
		canvas.info["name"] = "My Awesome Canvas"
		monitor.RegisterCanvas(canvas)
		monitor.CheckNow()

		status, _ := store.GetCanvasStatus("canvas-1")
		if status.Name != "My Awesome Canvas" {
			t.Errorf("expected name 'My Awesome Canvas', got '%s'", status.Name)
		}
	})
}

func TestStartAndStop(t *testing.T) {
	t.Run("performs periodic health checks", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		config := HealthMonitorConfig{
			CheckInterval: 50 * time.Millisecond, // Short interval for testing
		}
		monitor := NewCanvasHealthMonitor(store, config)

		canvas := newMockCanvasChecker("canvas-1", "url1", true)
		monitor.RegisterCanvas(canvas)

		ctx, cancel := context.WithCancel(context.Background())

		// Start monitor in background
		go monitor.Start(ctx)

		// Wait for multiple check intervals
		time.Sleep(200 * time.Millisecond)
		cancel()

		// Allow time for goroutine to exit
		time.Sleep(20 * time.Millisecond)

		// Should have been checked multiple times (initial + ticker)
		checkCount := canvas.GetCheckCount()
		if checkCount < 2 {
			t.Errorf("expected at least 2 checks, got %d", checkCount)
		}
	})

	t.Run("stops on context cancellation", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		config := HealthMonitorConfig{
			CheckInterval: 1 * time.Hour, // Long interval so we only get initial check
		}
		monitor := NewCanvasHealthMonitor(store, config)

		canvas := newMockCanvasChecker("canvas-1", "url1", true)
		monitor.RegisterCanvas(canvas)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func() {
			monitor.Start(ctx)
			close(done)
		}()

		// Give it time to start
		time.Sleep(20 * time.Millisecond)
		cancel()

		// Should exit promptly
		select {
		case <-done:
			// Good - monitor stopped
		case <-time.After(100 * time.Millisecond):
			t.Fatal("monitor did not stop after context cancellation")
		}
	})
}

func TestStatusChangeCallback(t *testing.T) {
	t.Run("calls callback on status change", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())

		var callbackCalls []struct {
			canvasID  string
			connected bool
		}
		var callbackMu sync.Mutex

		config := HealthMonitorConfig{
			CheckInterval: 50 * time.Millisecond,
			OnStatusChange: func(canvasID string, connected bool) {
				callbackMu.Lock()
				callbackCalls = append(callbackCalls, struct {
					canvasID  string
					connected bool
				}{canvasID, connected})
				callbackMu.Unlock()
			},
		}
		monitor := NewCanvasHealthMonitor(store, config)

		canvas := newMockCanvasChecker("canvas-1", "url1", true)
		monitor.RegisterCanvas(canvas)

		// First check should trigger callback (initial state change)
		monitor.CheckNow()

		callbackMu.Lock()
		if len(callbackCalls) != 1 {
			t.Errorf("expected 1 callback call, got %d", len(callbackCalls))
		}
		if len(callbackCalls) > 0 && !callbackCalls[0].connected {
			t.Error("expected connected=true in callback")
		}
		callbackMu.Unlock()

		// Change to unhealthy
		canvas.SetHealthy(false)
		monitor.CheckNow()

		callbackMu.Lock()
		if len(callbackCalls) != 2 {
			t.Errorf("expected 2 callback calls, got %d", len(callbackCalls))
		}
		if len(callbackCalls) > 1 && callbackCalls[1].connected {
			t.Error("expected connected=false in second callback")
		}
		callbackMu.Unlock()
	})

	t.Run("does not call callback when status unchanged", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())

		callbackCount := 0
		var callbackMu sync.Mutex

		config := HealthMonitorConfig{
			OnStatusChange: func(canvasID string, connected bool) {
				callbackMu.Lock()
				callbackCount++
				callbackMu.Unlock()
			},
		}
		monitor := NewCanvasHealthMonitor(store, config)

		canvas := newMockCanvasChecker("canvas-1", "url1", true)
		monitor.RegisterCanvas(canvas)

		// First check
		monitor.CheckNow()
		// Second check - same status
		monitor.CheckNow()
		// Third check - same status
		monitor.CheckNow()

		callbackMu.Lock()
		if callbackCount != 1 {
			t.Errorf("expected 1 callback call (initial only), got %d", callbackCount)
		}
		callbackMu.Unlock()
	})
}

func TestErrorTracking(t *testing.T) {
	t.Run("tracks errors when disconnected", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		monitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		canvas := newMockCanvasChecker("canvas-1", "url1", false)
		canvas.err = errors.New("connection refused")
		monitor.RegisterCanvas(canvas)

		// First check
		monitor.CheckNow()

		status, _ := store.GetCanvasStatus("canvas-1")
		if len(status.Errors) != 1 {
			t.Errorf("expected 1 error, got %d", len(status.Errors))
		}

		// Second check with different error
		canvas.err = errors.New("timeout")
		monitor.CheckNow()

		status, _ = store.GetCanvasStatus("canvas-1")
		if len(status.Errors) != 2 {
			t.Errorf("expected 2 errors, got %d", len(status.Errors))
		}
	})

	t.Run("limits error history to 5", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		monitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		canvas := newMockCanvasChecker("canvas-1", "url1", false)
		monitor.RegisterCanvas(canvas)

		// Generate 10 errors
		for i := 0; i < 10; i++ {
			canvas.err = errors.New("error " + string(rune('0'+i)))
			monitor.CheckNow()
		}

		status, _ := store.GetCanvasStatus("canvas-1")
		if len(status.Errors) != 5 {
			t.Errorf("expected 5 errors (max), got %d", len(status.Errors))
		}
	})

	t.Run("clears errors on reconnection", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		monitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		canvas := newMockCanvasChecker("canvas-1", "url1", false)
		canvas.err = errors.New("error")
		monitor.RegisterCanvas(canvas)

		// Generate some errors
		monitor.CheckNow()
		monitor.CheckNow()

		status, _ := store.GetCanvasStatus("canvas-1")
		if len(status.Errors) == 0 {
			t.Error("expected errors to be recorded")
		}

		// Reconnect
		canvas.SetHealthy(true)
		monitor.CheckNow()

		status, _ = store.GetCanvasStatus("canvas-1")
		if len(status.Errors) != 0 {
			t.Errorf("expected errors to be cleared, got %d", len(status.Errors))
		}
	})
}

func TestUpdateCanvasMetrics(t *testing.T) {
	t.Run("updates metrics for existing canvas", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		monitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		canvas := newMockCanvasChecker("canvas-1", "url1", true)
		monitor.RegisterCanvas(canvas)
		monitor.CheckNow()

		monitor.UpdateCanvasMetrics("canvas-1", 42, 100, 95.5)

		status, _ := store.GetCanvasStatus("canvas-1")
		if status.WidgetCount != 42 {
			t.Errorf("expected widget count 42, got %d", status.WidgetCount)
		}
		if status.RequestsToday != 100 {
			t.Errorf("expected requests today 100, got %d", status.RequestsToday)
		}
		if status.SuccessRate != 95.5 {
			t.Errorf("expected success rate 95.5, got %f", status.SuccessRate)
		}
	})

	t.Run("ignores update for unknown canvas", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		monitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		// Should not panic
		monitor.UpdateCanvasMetrics("unknown", 42, 100, 95.5)

		_, ok := store.GetCanvasStatus("unknown")
		if ok {
			t.Error("expected unknown canvas to not exist")
		}
	})

	t.Run("preserves metrics through health checks", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		monitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		canvas := newMockCanvasChecker("canvas-1", "url1", true)
		monitor.RegisterCanvas(canvas)
		monitor.CheckNow()

		// Set metrics
		monitor.UpdateCanvasMetrics("canvas-1", 42, 100, 95.5)

		// Perform another health check
		monitor.CheckNow()

		// Metrics should be preserved
		status, _ := store.GetCanvasStatus("canvas-1")
		if status.WidgetCount != 42 {
			t.Errorf("expected widget count to be preserved, got %d", status.WidgetCount)
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	t.Run("handles concurrent operations safely", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		monitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		var wg sync.WaitGroup

		// Register canvases concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				canvas := newMockCanvasChecker(
					string(rune('a'+id)),
					"url",
					true,
				)
				monitor.RegisterCanvas(canvas)
			}(i)
		}

		// Check and unregister concurrently
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				monitor.CheckNow()
				monitor.GetRegisteredCanvases()
				if id%2 == 0 {
					monitor.UnregisterCanvas(string(rune('a' + id)))
				}
			}(i)
		}

		wg.Wait()
		// If we get here without panic/deadlock, test passes
	})
}
