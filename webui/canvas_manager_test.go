package webui

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"go_backend/canvusapi"
	"go_backend/metrics"
)

// mockCanvasClient creates a mock canvusapi.Client for testing
func mockCanvasClient(server, canvasID string) *canvusapi.Client {
	return &canvusapi.Client{
		Server:   server,
		CanvasID: canvasID,
		ApiKey:   "test-api-key",
		HTTP:     nil, // Not used in unit tests
	}
}

func TestNewCanvasManager(t *testing.T) {
	t.Run("creates manager with default config", func(t *testing.T) {
		manager := NewCanvasManager(nil)

		if manager == nil {
			t.Fatal("expected non-nil manager")
		}
		if manager.clients == nil {
			t.Error("expected initialized clients map")
		}
		if manager.GetClientCount() != 0 {
			t.Error("expected empty client map")
		}
		if manager.MonitorHealth() != nil {
			t.Error("expected nil health monitor with default config")
		}
	})

	t.Run("creates manager with custom config", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		healthMonitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		config := &CanvasManagerConfig{
			HealthMonitor: healthMonitor,
			MetricsStore:  store,
			Logger:        &managerTestLogger{},
		}
		manager := NewCanvasManager(config)

		if manager.MonitorHealth() != healthMonitor {
			t.Error("expected health monitor to be set")
		}
		if manager.store != store {
			t.Error("expected store to be set")
		}
	})
}

func TestCanvasManager_AddCanvas(t *testing.T) {
	t.Run("adds canvas successfully", func(t *testing.T) {
		manager := NewCanvasManager(nil)
		client := mockCanvasClient("https://server1.example.com", "canvas-1")

		err := manager.AddCanvas("canvas-1", client)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if manager.GetClientCount() != 1 {
			t.Error("expected 1 client")
		}

		retrieved, ok := manager.GetClient("canvas-1")
		if !ok {
			t.Error("expected to find client")
		}
		if retrieved != client {
			t.Error("expected same client instance")
		}
	})

	t.Run("replaces existing canvas", func(t *testing.T) {
		manager := NewCanvasManager(nil)
		client1 := mockCanvasClient("https://server1.example.com", "canvas-1")
		client2 := mockCanvasClient("https://server2.example.com", "canvas-1")

		manager.AddCanvas("canvas-1", client1)
		manager.AddCanvas("canvas-1", client2)

		if manager.GetClientCount() != 1 {
			t.Error("expected 1 client after replacement")
		}

		retrieved, _ := manager.GetClient("canvas-1")
		if retrieved != client2 {
			t.Error("expected replaced client")
		}
	})

	t.Run("rejects empty canvas ID", func(t *testing.T) {
		manager := NewCanvasManager(nil)
		client := mockCanvasClient("https://server1.example.com", "canvas-1")

		err := manager.AddCanvas("", client)

		if err == nil {
			t.Error("expected error for empty canvas ID")
		}
	})

	t.Run("rejects nil client", func(t *testing.T) {
		manager := NewCanvasManager(nil)

		err := manager.AddCanvas("canvas-1", nil)

		if err == nil {
			t.Error("expected error for nil client")
		}
	})

	t.Run("registers with health monitor", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		healthMonitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		config := &CanvasManagerConfig{
			HealthMonitor: healthMonitor,
			MetricsStore:  store,
		}
		manager := NewCanvasManager(config)
		client := mockCanvasClient("https://server1.example.com", "canvas-1")

		manager.AddCanvas("canvas-1", client)

		// Check that canvas was registered with health monitor
		registered := healthMonitor.GetRegisteredCanvases()
		if len(registered) != 1 || registered[0] != "canvas-1" {
			t.Errorf("expected canvas-1 to be registered with health monitor, got %v", registered)
		}
	})
}

func TestCanvasManager_RemoveCanvas(t *testing.T) {
	t.Run("removes existing canvas", func(t *testing.T) {
		manager := NewCanvasManager(nil)
		client := mockCanvasClient("https://server1.example.com", "canvas-1")
		manager.AddCanvas("canvas-1", client)

		removed := manager.RemoveCanvas("canvas-1")

		if !removed {
			t.Error("expected true for successful removal")
		}
		if manager.GetClientCount() != 0 {
			t.Error("expected 0 clients after removal")
		}
		if _, ok := manager.GetClient("canvas-1"); ok {
			t.Error("expected canvas to be removed")
		}
	})

	t.Run("returns false for non-existent canvas", func(t *testing.T) {
		manager := NewCanvasManager(nil)

		removed := manager.RemoveCanvas("non-existent")

		if removed {
			t.Error("expected false for non-existent canvas")
		}
	})

	t.Run("returns false for empty canvas ID", func(t *testing.T) {
		manager := NewCanvasManager(nil)

		removed := manager.RemoveCanvas("")

		if removed {
			t.Error("expected false for empty canvas ID")
		}
	})

	t.Run("unregisters from health monitor", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		healthMonitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		config := &CanvasManagerConfig{
			HealthMonitor: healthMonitor,
			MetricsStore:  store,
		}
		manager := NewCanvasManager(config)
		client := mockCanvasClient("https://server1.example.com", "canvas-1")

		manager.AddCanvas("canvas-1", client)
		manager.RemoveCanvas("canvas-1")

		// Check that canvas was unregistered from health monitor
		registered := healthMonitor.GetRegisteredCanvases()
		if len(registered) != 0 {
			t.Errorf("expected no canvases registered with health monitor, got %v", registered)
		}
	})
}

func TestCanvasManager_GetClient(t *testing.T) {
	t.Run("returns client when exists", func(t *testing.T) {
		manager := NewCanvasManager(nil)
		client := mockCanvasClient("https://server1.example.com", "canvas-1")
		manager.AddCanvas("canvas-1", client)

		retrieved, ok := manager.GetClient("canvas-1")

		if !ok {
			t.Error("expected ok to be true")
		}
		if retrieved != client {
			t.Error("expected same client instance")
		}
	})

	t.Run("returns false when not exists", func(t *testing.T) {
		manager := NewCanvasManager(nil)

		_, ok := manager.GetClient("non-existent")

		if ok {
			t.Error("expected ok to be false")
		}
	})
}

func TestCanvasManager_GetAllCanvases(t *testing.T) {
	t.Run("returns empty slice for no canvases", func(t *testing.T) {
		manager := NewCanvasManager(nil)

		canvases := manager.GetAllCanvases()

		if len(canvases) != 0 {
			t.Error("expected empty slice")
		}
	})

	t.Run("returns all canvas IDs", func(t *testing.T) {
		manager := NewCanvasManager(nil)
		manager.AddCanvas("canvas-1", mockCanvasClient("https://s1.example.com", "canvas-1"))
		manager.AddCanvas("canvas-2", mockCanvasClient("https://s2.example.com", "canvas-2"))
		manager.AddCanvas("canvas-3", mockCanvasClient("https://s3.example.com", "canvas-3"))

		canvases := manager.GetAllCanvases()

		if len(canvases) != 3 {
			t.Errorf("expected 3 canvases, got %d", len(canvases))
		}

		// Check all IDs are present (order not guaranteed)
		found := make(map[string]bool)
		for _, id := range canvases {
			found[id] = true
		}
		for _, expected := range []string{"canvas-1", "canvas-2", "canvas-3"} {
			if !found[expected] {
				t.Errorf("expected to find %s", expected)
			}
		}
	})
}

func TestCanvasManager_GetClientCount(t *testing.T) {
	manager := NewCanvasManager(nil)

	if manager.GetClientCount() != 0 {
		t.Error("expected 0 initially")
	}

	manager.AddCanvas("canvas-1", mockCanvasClient("https://s1.example.com", "canvas-1"))
	if manager.GetClientCount() != 1 {
		t.Error("expected 1 after adding")
	}

	manager.AddCanvas("canvas-2", mockCanvasClient("https://s2.example.com", "canvas-2"))
	if manager.GetClientCount() != 2 {
		t.Error("expected 2 after adding second")
	}

	manager.RemoveCanvas("canvas-1")
	if manager.GetClientCount() != 1 {
		t.Error("expected 1 after removing")
	}
}

func TestCanvasManager_SetHealthMonitor(t *testing.T) {
	t.Run("sets new health monitor and registers existing canvases", func(t *testing.T) {
		manager := NewCanvasManager(nil)
		manager.AddCanvas("canvas-1", mockCanvasClient("https://s1.example.com", "canvas-1"))
		manager.AddCanvas("canvas-2", mockCanvasClient("https://s2.example.com", "canvas-2"))

		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		healthMonitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		manager.SetHealthMonitor(healthMonitor)

		// Check that existing canvases were registered
		registered := healthMonitor.GetRegisteredCanvases()
		if len(registered) != 2 {
			t.Errorf("expected 2 canvases registered, got %d", len(registered))
		}
	})

	t.Run("replaces existing health monitor", func(t *testing.T) {
		store1 := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		healthMonitor1 := NewCanvasHealthMonitor(store1, DefaultHealthMonitorConfig())

		config := &CanvasManagerConfig{
			HealthMonitor: healthMonitor1,
			MetricsStore:  store1,
		}
		manager := NewCanvasManager(config)
		manager.AddCanvas("canvas-1", mockCanvasClient("https://s1.example.com", "canvas-1"))

		// Verify registered with first monitor
		if len(healthMonitor1.GetRegisteredCanvases()) != 1 {
			t.Error("expected 1 canvas in first monitor")
		}

		// Replace with new monitor
		store2 := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		healthMonitor2 := NewCanvasHealthMonitor(store2, DefaultHealthMonitorConfig())
		manager.SetHealthMonitor(healthMonitor2)

		// Check old monitor has no canvases
		if len(healthMonitor1.GetRegisteredCanvases()) != 0 {
			t.Error("expected 0 canvases in old monitor after replacement")
		}

		// Check new monitor has the canvas
		if len(healthMonitor2.GetRegisteredCanvases()) != 1 {
			t.Error("expected 1 canvas in new monitor")
		}
	})

	t.Run("handles nil monitor", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		healthMonitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

		config := &CanvasManagerConfig{
			HealthMonitor: healthMonitor,
		}
		manager := NewCanvasManager(config)
		manager.AddCanvas("canvas-1", mockCanvasClient("https://s1.example.com", "canvas-1"))

		// Set to nil
		manager.SetHealthMonitor(nil)

		if manager.MonitorHealth() != nil {
			t.Error("expected nil monitor after setting to nil")
		}

		// Old monitor should have no canvases
		if len(healthMonitor.GetRegisteredCanvases()) != 0 {
			t.Error("expected 0 canvases in old monitor")
		}
	})
}

func TestCanvasManager_GetCanvasStatus(t *testing.T) {
	t.Run("returns false when no store", func(t *testing.T) {
		manager := NewCanvasManager(nil)

		_, ok := manager.GetCanvasStatus("canvas-1")

		if ok {
			t.Error("expected false when no store")
		}
	})

	t.Run("returns status from store", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
		store.UpdateCanvasStatus(metrics.CanvasStatus{
			ID:        "canvas-1",
			Name:      "Test Canvas",
			Connected: true,
		})

		config := &CanvasManagerConfig{
			MetricsStore: store,
		}
		manager := NewCanvasManager(config)

		status, ok := manager.GetCanvasStatus("canvas-1")

		if !ok {
			t.Error("expected status to be found")
		}
		if status.Name != "Test Canvas" {
			t.Errorf("expected 'Test Canvas', got %s", status.Name)
		}
		if !status.Connected {
			t.Error("expected connected to be true")
		}
	})
}

func TestCanvasManager_GetAllCanvasStatuses(t *testing.T) {
	t.Run("returns nil when no store", func(t *testing.T) {
		manager := NewCanvasManager(nil)

		statuses := manager.GetAllCanvasStatuses()

		if statuses != nil {
			t.Error("expected nil when no store")
		}
	})

	t.Run("returns statuses for all registered canvases", func(t *testing.T) {
		store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())

		config := &CanvasManagerConfig{
			MetricsStore: store,
		}
		manager := NewCanvasManager(config)

		// Add canvases
		manager.AddCanvas("canvas-1", mockCanvasClient("https://s1.example.com", "canvas-1"))
		manager.AddCanvas("canvas-2", mockCanvasClient("https://s2.example.com", "canvas-2"))

		// Add statuses to store
		store.UpdateCanvasStatus(metrics.CanvasStatus{ID: "canvas-1", Connected: true})
		store.UpdateCanvasStatus(metrics.CanvasStatus{ID: "canvas-2", Connected: false})
		store.UpdateCanvasStatus(metrics.CanvasStatus{ID: "canvas-3", Connected: true}) // Not in manager

		statuses := manager.GetAllCanvasStatuses()

		// Should only return statuses for registered canvases
		if len(statuses) != 2 {
			t.Errorf("expected 2 statuses, got %d", len(statuses))
		}
	})
}

func TestCanvasManager_ThreadSafety(t *testing.T) {
	manager := NewCanvasManager(nil)

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	// Concurrent adds
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("canvas-%d", n)
			client := mockCanvasClient(fmt.Sprintf("https://s%d.example.com", n), id)
			if err := manager.AddCanvas(id, client); err != nil {
				errors <- err
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("canvas-%d", n)
			manager.GetClient(id)
			manager.GetAllCanvases()
			manager.GetClientCount()
		}(i)
	}

	// Concurrent removes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			id := fmt.Sprintf("canvas-%d", n)
			manager.RemoveCanvas(id)
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent operation error: %v", err)
	}
}

func TestCanvasClientAdapter(t *testing.T) {
	t.Run("implements CanvasChecker interface", func(t *testing.T) {
		client := mockCanvasClient("https://server.example.com", "test-canvas")
		adapter := &canvasClientAdapter{client: client}

		// Test GetCanvasID
		if adapter.GetCanvasID() != "test-canvas" {
			t.Errorf("expected 'test-canvas', got %s", adapter.GetCanvasID())
		}

		// Test GetServerURL
		if adapter.GetServerURL() != "https://server.example.com" {
			t.Errorf("expected 'https://server.example.com', got %s", adapter.GetServerURL())
		}

		// Verify it implements CanvasChecker
		var _ CanvasChecker = adapter
	})
}

// managerTestLogger is a simple logger for testing CanvasManager
type managerTestLogger struct {
	messages []string
	mu       sync.Mutex
}

func (l *managerTestLogger) Printf(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, fmt.Sprintf(format, args...))
}

// Helper to create a test setup with manager, store, and health monitor
func createTestSetup() (*CanvasManager, *metrics.MetricsStore, *CanvasHealthMonitor) {
	store := metrics.NewMetricsStore(metrics.DefaultStoreConfig(), time.Now())
	healthMonitor := NewCanvasHealthMonitor(store, DefaultHealthMonitorConfig())

	config := &CanvasManagerConfig{
		HealthMonitor: healthMonitor,
		MetricsStore:  store,
		Logger:        &managerTestLogger{},
	}
	manager := NewCanvasManager(config)

	return manager, store, healthMonitor
}

func TestCanvasManager_IntegrationWithHealthMonitor(t *testing.T) {
	t.Run("full lifecycle with health monitoring", func(t *testing.T) {
		manager, store, healthMonitor := createTestSetup()

		// Add canvases
		client1 := mockCanvasClient("https://server1.example.com", "canvas-1")
		client2 := mockCanvasClient("https://server2.example.com", "canvas-2")

		if err := manager.AddCanvas("canvas-1", client1); err != nil {
			t.Fatalf("failed to add canvas-1: %v", err)
		}
		if err := manager.AddCanvas("canvas-2", client2); err != nil {
			t.Fatalf("failed to add canvas-2: %v", err)
		}

		// Verify health monitor registered both
		registered := healthMonitor.GetRegisteredCanvases()
		if len(registered) != 2 {
			t.Errorf("expected 2 registered, got %d", len(registered))
		}

		// Simulate status update through store
		store.UpdateCanvasStatus(metrics.CanvasStatus{
			ID:         "canvas-1",
			Connected:  true,
			LastUpdate: time.Now(),
		})

		// Check status through manager
		status, ok := manager.GetCanvasStatus("canvas-1")
		if !ok {
			t.Error("expected to find status")
		}
		if !status.Connected {
			t.Error("expected connected status")
		}

		// Remove canvas
		manager.RemoveCanvas("canvas-1")

		// Verify health monitor only has canvas-2
		registered = healthMonitor.GetRegisteredCanvases()
		if len(registered) != 1 || registered[0] != "canvas-2" {
			t.Errorf("expected only canvas-2 registered, got %v", registered)
		}

		// Verify manager state
		if manager.GetClientCount() != 1 {
			t.Error("expected 1 client after removal")
		}
	})
}
