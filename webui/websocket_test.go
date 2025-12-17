package webui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// mockLogger captures log messages for testing
type mockLogger struct {
	mu       sync.Mutex
	messages []string
}

func (m *mockLogger) Printf(format string, v ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, format)
}

func (m *mockLogger) getMessages() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.messages))
	copy(result, m.messages)
	return result
}

func TestNewWebSocketBroadcaster(t *testing.T) {
	b := NewWebSocketBroadcaster()

	if b == nil {
		t.Fatal("Expected non-nil broadcaster")
	}

	if b.clients == nil {
		t.Error("Expected clients map to be initialized")
	}

	if b.broadcast == nil {
		t.Error("Expected broadcast channel to be initialized")
	}

	if b.register == nil {
		t.Error("Expected register channel to be initialized")
	}

	if b.unregister == nil {
		t.Error("Expected unregister channel to be initialized")
	}

	if b.pingInterval != 30*time.Second {
		t.Errorf("Expected pingInterval 30s, got %v", b.pingInterval)
	}

	if b.pongWait != 60*time.Second {
		t.Errorf("Expected pongWait 60s, got %v", b.pongWait)
	}
}

func TestNewWebSocketBroadcasterWithConfig(t *testing.T) {
	logger := &mockLogger{}
	config := BroadcasterConfig{
		PingInterval:         10 * time.Second,
		PongWait:             20 * time.Second,
		WriteWait:            5 * time.Second,
		MaxMessageSize:       1024,
		BroadcastBufferSize:  128,
		ClientSendBufferSize: 64,
		Logger:               logger,
	}

	b := NewWebSocketBroadcasterWithConfig(config)

	if b.pingInterval != 10*time.Second {
		t.Errorf("Expected pingInterval 10s, got %v", b.pingInterval)
	}

	if b.pongWait != 20*time.Second {
		t.Errorf("Expected pongWait 20s, got %v", b.pongWait)
	}

	if b.writeWait != 5*time.Second {
		t.Errorf("Expected writeWait 5s, got %v", b.writeWait)
	}

	if b.maxMessageSize != 1024 {
		t.Errorf("Expected maxMessageSize 1024, got %v", b.maxMessageSize)
	}
}

func TestNewWebSocketBroadcasterWithConfig_NilLogger(t *testing.T) {
	config := BroadcasterConfig{
		PingInterval: 10 * time.Second,
		Logger:       nil, // explicitly nil
	}

	b := NewWebSocketBroadcasterWithConfig(config)

	if b.logger == nil {
		t.Error("Expected default logger to be set when nil provided")
	}
}

func TestDefaultBroadcasterConfig(t *testing.T) {
	config := DefaultBroadcasterConfig()

	if config.PingInterval != 30*time.Second {
		t.Errorf("Expected PingInterval 30s, got %v", config.PingInterval)
	}

	if config.PongWait != 60*time.Second {
		t.Errorf("Expected PongWait 60s, got %v", config.PongWait)
	}

	if config.WriteWait != 10*time.Second {
		t.Errorf("Expected WriteWait 10s, got %v", config.WriteWait)
	}

	if config.MaxMessageSize != 512 {
		t.Errorf("Expected MaxMessageSize 512, got %v", config.MaxMessageSize)
	}

	if config.BroadcastBufferSize != 256 {
		t.Errorf("Expected BroadcastBufferSize 256, got %v", config.BroadcastBufferSize)
	}

	if config.ClientSendBufferSize != 256 {
		t.Errorf("Expected ClientSendBufferSize 256, got %v", config.ClientSendBufferSize)
	}
}

func TestWebSocketBroadcaster_ClientCount_Empty(t *testing.T) {
	b := NewWebSocketBroadcaster()

	if count := b.ClientCount(); count != 0 {
		t.Errorf("Expected 0 clients, got %d", count)
	}
}

func TestWebSocketBroadcaster_Start_ContextCancellation(t *testing.T) {
	logger := &mockLogger{}
	config := DefaultBroadcasterConfig()
	config.Logger = logger

	b := NewWebSocketBroadcasterWithConfig(config)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		b.Start(ctx)
		close(done)
	}()

	// Give broadcaster time to start
	time.Sleep(10 * time.Millisecond)

	// Cancel context
	cancel()

	// Wait for broadcaster to stop
	select {
	case <-done:
		// Success
	case <-time.After(1 * time.Second):
		t.Fatal("Broadcaster did not stop after context cancellation")
	}

	// Check logs
	messages := logger.getMessages()
	foundStarted := false
	foundStopping := false
	for _, msg := range messages {
		if strings.Contains(msg, "started") {
			foundStarted = true
		}
		if strings.Contains(msg, "stopping") {
			foundStopping = true
		}
	}

	if !foundStarted {
		t.Error("Expected 'started' log message")
	}
	if !foundStopping {
		t.Error("Expected 'stopping' log message")
	}
}

func TestWebSocketBroadcaster_BroadcastMessage(t *testing.T) {
	b := NewWebSocketBroadcaster()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Broadcast a message (no clients connected)
	msg := NewTaskUpdateMessage(TaskUpdateData{
		TaskID:   "test-123",
		TaskType: "note_processing",
		Status:   "completed",
	})

	// Should not block even with no clients
	b.BroadcastMessage(msg)

	// Allow time for message to be processed
	time.Sleep(10 * time.Millisecond)

	// No clients, so nothing should happen (but no panic either)
}

func TestWebSocketBroadcaster_BroadcastMessage_BufferFull(t *testing.T) {
	logger := &mockLogger{}
	config := BroadcasterConfig{
		PingInterval:        30 * time.Second,
		PongWait:            60 * time.Second,
		WriteWait:           10 * time.Second,
		MaxMessageSize:      512,
		BroadcastBufferSize: 1, // Very small buffer
		Logger:              logger,
	}

	b := NewWebSocketBroadcasterWithConfig(config)

	// Don't start the broadcaster (so messages accumulate)

	// Fill the buffer
	msg := NewPingMessage()
	b.BroadcastMessage(msg) // Should succeed (buffer has 1 slot)
	b.BroadcastMessage(msg) // Should drop with warning

	time.Sleep(10 * time.Millisecond)

	messages := logger.getMessages()
	foundWarning := false
	for _, m := range messages {
		if strings.Contains(m, "buffer full") {
			foundWarning = true
		}
	}

	if !foundWarning {
		t.Error("Expected warning about buffer full")
	}
}

func TestWebSocketBroadcaster_ConvenienceMethods(t *testing.T) {
	b := NewWebSocketBroadcaster()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Test all convenience methods don't panic
	t.Run("BroadcastTaskUpdate", func(t *testing.T) {
		b.BroadcastTaskUpdate(TaskUpdateData{
			TaskID:   "task-1",
			TaskType: "pdf_analysis",
			Status:   "processing",
		})
	})

	t.Run("BroadcastGPUUpdate", func(t *testing.T) {
		b.BroadcastGPUUpdate(GPUUpdateData{
			Utilization:   75.5,
			Temperature:   68.0,
			MemoryUsed:    4294967296,
			MemoryTotal:   8589934592,
			MemoryPercent: 50.0,
		})
	})

	t.Run("BroadcastCanvasUpdate", func(t *testing.T) {
		b.BroadcastCanvasUpdate(CanvasUpdateData{
			CanvasID:    "canvas-1",
			Name:        "Test Canvas",
			Connected:   true,
			WidgetCount: 42,
		})
	})

	t.Run("BroadcastSystemStatus", func(t *testing.T) {
		b.BroadcastSystemStatus(SystemStatusData{
			Status:         "running",
			Uptime:         time.Hour,
			ActiveTasks:    5,
			TotalProcessed: 1000,
			ErrorRate:      2.5,
		})
	})

	t.Run("BroadcastError", func(t *testing.T) {
		b.BroadcastError("ERR_TIMEOUT", "Request timed out")
	})
}

func TestWebSocketBroadcaster_HandleConnection(t *testing.T) {
	logger := &mockLogger{}
	config := DefaultBroadcasterConfig()
	config.Logger = logger

	b := NewWebSocketBroadcasterWithConfig(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(b.HandleConnection))
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect client
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Give time for registration
	time.Sleep(50 * time.Millisecond)

	// Verify client count
	if count := b.ClientCount(); count != 1 {
		t.Errorf("Expected 1 client, got %d", count)
	}

	// Check logs
	messages := logger.getMessages()
	foundConnected := false
	for _, msg := range messages {
		if strings.Contains(msg, "connected") {
			foundConnected = true
		}
	}

	if !foundConnected {
		t.Error("Expected 'connected' log message")
	}
}

func TestWebSocketBroadcaster_HandleConnection_Disconnect(t *testing.T) {
	logger := &mockLogger{}
	config := DefaultBroadcasterConfig()
	config.Logger = logger

	b := NewWebSocketBroadcasterWithConfig(config)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(b.HandleConnection))
	defer server.Close()

	// Convert HTTP URL to WebSocket URL
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect client
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Give time for registration
	time.Sleep(50 * time.Millisecond)

	if count := b.ClientCount(); count != 1 {
		t.Errorf("Expected 1 client after connect, got %d", count)
	}

	// Close connection
	conn.Close()

	// Give time for unregistration
	time.Sleep(100 * time.Millisecond)

	if count := b.ClientCount(); count != 0 {
		t.Errorf("Expected 0 clients after disconnect, got %d", count)
	}

	// Check logs
	messages := logger.getMessages()
	foundDisconnected := false
	for _, msg := range messages {
		if strings.Contains(msg, "disconnected") {
			foundDisconnected = true
		}
	}

	if !foundDisconnected {
		t.Error("Expected 'disconnected' log message")
	}
}

func TestWebSocketBroadcaster_MultipleClients(t *testing.T) {
	b := NewWebSocketBroadcaster()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(b.HandleConnection))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect multiple clients
	numClients := 5
	conns := make([]*websocket.Conn, numClients)
	dialer := websocket.Dialer{}

	for i := 0; i < numClients; i++ {
		conn, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}
		conns[i] = conn
	}

	// Give time for all registrations
	time.Sleep(100 * time.Millisecond)

	if count := b.ClientCount(); count != numClients {
		t.Errorf("Expected %d clients, got %d", numClients, count)
	}

	// Close all connections
	for _, conn := range conns {
		conn.Close()
	}

	// Give time for all unregistrations
	time.Sleep(200 * time.Millisecond)

	if count := b.ClientCount(); count != 0 {
		t.Errorf("Expected 0 clients after closing all, got %d", count)
	}
}

func TestWebSocketBroadcaster_BroadcastToConnectedClients(t *testing.T) {
	b := NewWebSocketBroadcaster()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(b.HandleConnection))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect client
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Give time for registration
	time.Sleep(100 * time.Millisecond)

	// Broadcast a message
	testMsg := NewTaskUpdateMessage(TaskUpdateData{
		TaskID:   "broadcast-test",
		TaskType: "note_processing",
		Status:   "completed",
	})
	b.BroadcastMessage(testMsg)

	// Set read deadline and read the broadcast message
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("Failed to read broadcast message: %v", err)
	}

	// Verify message contains expected data
	if !strings.Contains(string(message), "broadcast-test") {
		t.Errorf("Expected message to contain 'broadcast-test', got: %s", string(message))
	}

	if !strings.Contains(string(message), "task_update") {
		t.Errorf("Expected message to contain 'task_update', got: %s", string(message))
	}
}

func TestWebSocketBroadcaster_Close(t *testing.T) {
	b := NewWebSocketBroadcaster()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(b.HandleConnection))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect clients
	dialer := websocket.Dialer{}
	conn1, _, _ := dialer.Dial(wsURL, nil)
	conn2, _, _ := dialer.Dial(wsURL, nil)
	defer conn1.Close()
	defer conn2.Close()

	time.Sleep(50 * time.Millisecond)

	if count := b.ClientCount(); count != 2 {
		t.Errorf("Expected 2 clients before close, got %d", count)
	}

	// Call Close
	b.Close()

	if count := b.ClientCount(); count != 0 {
		t.Errorf("Expected 0 clients after close, got %d", count)
	}
}

func TestWebSocketBroadcaster_ThreadSafety(t *testing.T) {
	b := NewWebSocketBroadcaster()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go b.Start(ctx)
	time.Sleep(10 * time.Millisecond)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(b.HandleConnection))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	dialer := websocket.Dialer{}

	// Run concurrent operations
	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent connections
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			conn, _, err := dialer.Dial(wsURL, nil)
			if err == nil {
				time.Sleep(50 * time.Millisecond)
				conn.Close()
			}
		}()
	}

	// Concurrent broadcasts
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				b.BroadcastMessage(NewTaskUpdateMessage(TaskUpdateData{
					TaskID: "concurrent-test",
					Status: "processing",
				}))
			}
		}(i)
	}

	// Concurrent client count reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				_ = b.ClientCount()
			}
		}()
	}

	// Wait for all operations to complete
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success - no race conditions or deadlocks
	case <-time.After(5 * time.Second):
		t.Fatal("Thread safety test timed out - possible deadlock")
	}
}
