package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"go_backend/metrics"
	"go_backend/webui"
	"go_backend/webui/auth"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// =============================================================================
// End-to-End Dashboard Flow Tests
// These tests verify the complete dashboard flow:
// - Server startup with real components
// - Login via POST
// - Fetch dashboard data via API
// - Connect WebSocket
// - Simulate task updates
// - Verify broadcasts
// - Logout
// =============================================================================

// testAuthProvider adapts auth.AuthMiddleware to webui.AuthProvider interface
type testAuthProvider struct {
	mw *auth.AuthMiddleware
}

func (p *testAuthProvider) Middleware(next http.Handler) http.Handler {
	return p.mw.Middleware(next)
}

func (p *testAuthProvider) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return p.mw.RequireAuth(next)
}

func (p *testAuthProvider) LoginHandler() http.HandlerFunc {
	return auth.LoginHandler(p.mw)
}

func (p *testAuthProvider) LogoutHandler() http.HandlerFunc {
	return auth.LogoutHandler(p.mw)
}

// setupE2ETestServer creates a fully integrated test server with real components.
// Returns the test server, metrics store, broadcaster, auth middleware, and cleanup function.
func setupE2ETestServer(t *testing.T, password string) (*httptest.Server, *metrics.MetricsStore, *webui.WebSocketBroadcaster, *auth.AuthMiddleware, context.CancelFunc) {
	t.Helper()

	logger := zap.NewNop()

	// Create real metrics store
	storeConfig := metrics.StoreConfig{
		TaskHistoryCapacity: 100,
		Version:             "1.0.0-test",
	}
	metricsStore := metrics.NewMetricsStore(storeConfig, time.Now())

	// Create auth middleware
	authMiddleware, err := auth.NewAuthMiddleware(password, logger)
	if err != nil {
		t.Fatalf("failed to create auth middleware: %v", err)
	}

	// Create auth provider adapter
	authProvider := &testAuthProvider{mw: authMiddleware}

	// Create server config
	serverConfig := webui.DefaultServerConfig()
	serverConfig.Port = 0 // Let OS choose port
	serverConfig.VersionInfo = webui.VersionInfo{
		Version:   "1.0.0-test",
		BuildDate: "2024-01-01",
		GitCommit: "abc123",
	}

	// Create WebUI server (no GPU collector for tests)
	server, err := webui.NewServer(serverConfig, metricsStore, nil, authProvider, logger)
	if err != nil {
		t.Fatalf("failed to create WebUI server: %v", err)
	}

	// Create context for broadcaster
	ctx, cancel := context.WithCancel(context.Background())

	// Start broadcaster
	go server.GetBroadcaster().Start(ctx)

	// Create test HTTP server
	// Note: We need to manually create a handler that mimics the full server routing
	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("/login", authProvider.LoginHandler())
	mux.HandleFunc("/logout", authProvider.LogoutHandler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Protected routes
	api := server.GetDashboardAPI()
	mux.Handle("/api/status", authProvider.Middleware(http.HandlerFunc(api.HandleStatus)))
	mux.Handle("/api/canvases", authProvider.Middleware(http.HandlerFunc(api.HandleCanvases)))
	mux.Handle("/api/tasks", authProvider.Middleware(http.HandlerFunc(api.HandleTasks)))
	mux.Handle("/api/metrics", authProvider.Middleware(http.HandlerFunc(api.HandleMetrics)))
	mux.Handle("/api/gpu", authProvider.Middleware(http.HandlerFunc(api.HandleGPU)))

	// WebSocket endpoint (protected)
	mux.Handle("/ws", authProvider.Middleware(http.HandlerFunc(server.GetBroadcaster().HandleConnection)))

	// Dashboard route (protected)
	mux.Handle("/dashboard", authProvider.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body>Dashboard</body></html>"))
	})))

	testServer := httptest.NewServer(mux)

	return testServer, metricsStore, server.GetBroadcaster(), authMiddleware, cancel
}

// TestE2E_CompleteDashboardFlow tests the complete user journey through the dashboard.
// This is a comprehensive end-to-end test that verifies:
//  1. Server is running and health check works
//  2. User can login with correct password
//  3. Authenticated user can fetch dashboard API data
//  4. Authenticated user can connect to WebSocket
//  5. Task updates are broadcast to connected clients
//  6. User can logout
//  7. Logged out user cannot access protected resources
func TestE2E_CompleteDashboardFlow(t *testing.T) {
	password := "e2e-test-password-123"
	testServer, metricsStore, broadcaster, authMiddleware, cancel := setupE2ETestServer(t, password)
	defer testServer.Close()
	defer cancel()

	// Create HTTP client that doesn't follow redirects
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 10 * time.Second,
	}

	// Give broadcaster time to start
	time.Sleep(50 * time.Millisecond)

	// =========================================================================
	// Step 1: Verify health check (public endpoint)
	// =========================================================================
	t.Run("Step1_HealthCheck", func(t *testing.T) {
		resp, err := client.Get(testServer.URL + "/health")
		if err != nil {
			t.Fatalf("health check request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 for health check, got %d", resp.StatusCode)
		}

		var health map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
			t.Fatalf("failed to decode health response: %v", err)
		}

		if health["status"] != "ok" {
			t.Errorf("expected status 'ok', got '%s'", health["status"])
		}
	})

	// =========================================================================
	// Step 2: Verify protected routes require authentication
	// =========================================================================
	t.Run("Step2_ProtectedRoutesRequireAuth", func(t *testing.T) {
		endpoints := []string{"/dashboard", "/api/status", "/api/canvases", "/api/tasks", "/api/metrics"}

		for _, endpoint := range endpoints {
			resp, err := client.Get(testServer.URL + endpoint)
			if err != nil {
				t.Fatalf("request to %s failed: %v", endpoint, err)
			}
			resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("expected 401 for %s without auth, got %d", endpoint, resp.StatusCode)
			}
		}
	})

	// =========================================================================
	// Step 3: Login with correct password
	// =========================================================================
	var sessionCookie *http.Cookie

	t.Run("Step3_LoginWithCorrectPassword", func(t *testing.T) {
		form := url.Values{}
		form.Set("password", password)

		resp, err := client.Post(
			testServer.URL+"/login",
			"application/x-www-form-urlencoded",
			strings.NewReader(form.Encode()),
		)
		if err != nil {
			t.Fatalf("login request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("expected 303 after login, got %d", resp.StatusCode)
		}

		// Extract session cookie
		for _, c := range resp.Cookies() {
			if c.Name == auth.SessionCookieName && c.MaxAge > 0 {
				sessionCookie = c
				break
			}
		}

		if sessionCookie == nil {
			t.Fatal("no session cookie set after login")
		}
	})

	if sessionCookie == nil {
		t.Fatal("cannot continue without session cookie")
	}

	// =========================================================================
	// Step 4: Fetch dashboard data via API
	// =========================================================================
	t.Run("Step4_FetchAPIData", func(t *testing.T) {
		// Add some test data first
		metricsStore.RecordTask(metrics.TaskRecord{
			ID:        "task-001",
			Type:      "note_processing",
			CanvasID:  "canvas-1",
			Status:    metrics.TaskStatusSuccess,
			StartTime: time.Now().Add(-5 * time.Second),
			EndTime:   time.Now(),
			Duration:  5 * time.Second,
		})

		metricsStore.RecordTask(metrics.TaskRecord{
			ID:        "task-002",
			Type:      "pdf_analysis",
			CanvasID:  "canvas-1",
			Status:    metrics.TaskStatusSuccess,
			StartTime: time.Now().Add(-10 * time.Second),
			EndTime:   time.Now().Add(-3 * time.Second),
			Duration:  7 * time.Second,
		})

		metricsStore.UpdateCanvasStatus(metrics.CanvasStatus{
			ID:          "canvas-1",
			Name:        "Test Canvas",
			Connected:   true,
			WidgetCount: 42,
			LastUpdate:  time.Now(),
		})

		// Test /api/status
		t.Run("Status", func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, testServer.URL+"/api/status", nil)
			req.AddCookie(sessionCookie)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("status request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200, got %d", resp.StatusCode)
			}

			var status webui.StatusResponse
			if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
				t.Fatalf("failed to decode status: %v", err)
			}

			if status.Health != metrics.SystemHealthRunning {
				t.Errorf("expected health 'running', got '%s'", status.Health)
			}

			if status.Version != "1.0.0-test" {
				t.Errorf("expected version '1.0.0-test', got '%s'", status.Version)
			}
		})

		// Test /api/canvases
		t.Run("Canvases", func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, testServer.URL+"/api/canvases", nil)
			req.AddCookie(sessionCookie)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("canvases request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200, got %d", resp.StatusCode)
			}

			var canvases webui.CanvasesResponse
			if err := json.NewDecoder(resp.Body).Decode(&canvases); err != nil {
				t.Fatalf("failed to decode canvases: %v", err)
			}

			if canvases.Count != 1 {
				t.Errorf("expected 1 canvas, got %d", canvases.Count)
			}

			if len(canvases.Canvases) > 0 && canvases.Canvases[0].Name != "Test Canvas" {
				t.Errorf("expected canvas name 'Test Canvas', got '%s'", canvases.Canvases[0].Name)
			}
		})

		// Test /api/tasks
		t.Run("Tasks", func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, testServer.URL+"/api/tasks?limit=10", nil)
			req.AddCookie(sessionCookie)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("tasks request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200, got %d", resp.StatusCode)
			}

			var tasks webui.TasksResponse
			if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
				t.Fatalf("failed to decode tasks: %v", err)
			}

			if tasks.Count != 2 {
				t.Errorf("expected 2 tasks, got %d", tasks.Count)
			}
		})

		// Test /api/metrics
		t.Run("Metrics", func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, testServer.URL+"/api/metrics", nil)
			req.AddCookie(sessionCookie)

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("metrics request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected 200, got %d", resp.StatusCode)
			}

			var metricsResp webui.MetricsResponse
			if err := json.NewDecoder(resp.Body).Decode(&metricsResp); err != nil {
				t.Fatalf("failed to decode metrics: %v", err)
			}

			if metricsResp.TotalProcessed != 2 {
				t.Errorf("expected total_processed 2, got %d", metricsResp.TotalProcessed)
			}

			if metricsResp.TotalSuccess != 2 {
				t.Errorf("expected total_success 2, got %d", metricsResp.TotalSuccess)
			}

			if metricsResp.SuccessRate != 100.0 {
				t.Errorf("expected success_rate 100.0, got %f", metricsResp.SuccessRate)
			}
		})
	})

	// =========================================================================
	// Step 5: Connect to WebSocket and receive broadcasts
	// =========================================================================
	t.Run("Step5_WebSocketConnectionAndBroadcast", func(t *testing.T) {
		// Need to create session for WebSocket auth
		_, wsCookie, err := authMiddleware.CreateSession()
		if err != nil {
			t.Fatalf("failed to create session for WebSocket: %v", err)
		}

		// Convert HTTP URL to WebSocket URL
		wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"

		// Connect with session cookie
		dialer := websocket.Dialer{}
		header := http.Header{}
		header.Add("Cookie", fmt.Sprintf("%s=%s", wsCookie.Name, wsCookie.Value))

		conn, _, err := dialer.Dial(wsURL, header)
		if err != nil {
			t.Fatalf("failed to connect to WebSocket: %v", err)
		}
		defer conn.Close()

		// Give time for connection to register
		time.Sleep(100 * time.Millisecond)

		// Verify client is connected
		if broadcaster.ClientCount() < 1 {
			t.Error("expected at least 1 WebSocket client connected")
		}

		// Set up message receiver
		receivedMsg := make(chan []byte, 1)
		go func() {
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			_, message, err := conn.ReadMessage()
			if err == nil {
				receivedMsg <- message
			}
		}()

		// Broadcast a task update
		broadcaster.BroadcastTaskUpdate(webui.TaskUpdateData{
			TaskID:   "broadcast-task-001",
			TaskType: "note_processing",
			Status:   "completed",
			CanvasID: "canvas-1",
			Duration: 2 * time.Second,
		})

		// Wait for message
		select {
		case msg := <-receivedMsg:
			// Verify message content
			if !strings.Contains(string(msg), "broadcast-task-001") {
				t.Errorf("expected message to contain task ID, got: %s", string(msg))
			}
			if !strings.Contains(string(msg), "task_update") {
				t.Errorf("expected message to contain type 'task_update', got: %s", string(msg))
			}
		case <-time.After(3 * time.Second):
			t.Error("timeout waiting for broadcast message")
		}
	})

	// =========================================================================
	// Step 6: Logout
	// =========================================================================
	t.Run("Step6_Logout", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, testServer.URL+"/logout", nil)
		req.AddCookie(sessionCookie)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("logout request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("expected 303 after logout, got %d", resp.StatusCode)
		}
	})

	// =========================================================================
	// Step 7: Verify logged out user cannot access protected resources
	// =========================================================================
	t.Run("Step7_PostLogoutAccess", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, testServer.URL+"/api/status", nil)
		req.AddCookie(sessionCookie) // Old session cookie

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("post-logout request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("expected 401 after logout, got %d", resp.StatusCode)
		}
	})
}

// TestE2E_MultipleWebSocketClients verifies that multiple clients receive broadcasts.
func TestE2E_MultipleWebSocketClients(t *testing.T) {
	password := "multi-client-test"
	testServer, _, broadcaster, authMiddleware, cancel := setupE2ETestServer(t, password)
	defer testServer.Close()
	defer cancel()

	time.Sleep(50 * time.Millisecond)

	// Connect multiple clients
	const numClients = 3
	conns := make([]*websocket.Conn, numClients)
	dialer := websocket.Dialer{}

	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"

	for i := 0; i < numClients; i++ {
		_, wsCookie, err := authMiddleware.CreateSession()
		if err != nil {
			t.Fatalf("failed to create session %d: %v", i, err)
		}

		header := http.Header{}
		header.Add("Cookie", fmt.Sprintf("%s=%s", wsCookie.Name, wsCookie.Value))

		conn, _, err := dialer.Dial(wsURL, header)
		if err != nil {
			t.Fatalf("failed to connect client %d: %v", i, err)
		}
		conns[i] = conn
	}

	defer func() {
		for _, conn := range conns {
			if conn != nil {
				conn.Close()
			}
		}
	}()

	// Give time for all connections to register
	time.Sleep(100 * time.Millisecond)

	if count := broadcaster.ClientCount(); count != numClients {
		t.Errorf("expected %d clients, got %d", numClients, count)
	}

	// Set up receivers for all clients
	var wg sync.WaitGroup
	received := make([]bool, numClients)
	var mu sync.Mutex

	for i, conn := range conns {
		wg.Add(1)
		go func(idx int, c *websocket.Conn) {
			defer wg.Done()
			c.SetReadDeadline(time.Now().Add(3 * time.Second))
			_, message, err := c.ReadMessage()
			if err == nil && strings.Contains(string(message), "multi-client-task") {
				mu.Lock()
				received[idx] = true
				mu.Unlock()
			}
		}(i, conn)
	}

	// Broadcast message
	broadcaster.BroadcastTaskUpdate(webui.TaskUpdateData{
		TaskID:   "multi-client-task",
		TaskType: "test",
		Status:   "completed",
	})

	// Wait for all receivers with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Check all received
		mu.Lock()
		for i, r := range received {
			if !r {
				t.Errorf("client %d did not receive broadcast", i)
			}
		}
		mu.Unlock()
	case <-time.After(5 * time.Second):
		t.Error("timeout waiting for all clients to receive broadcast")
	}
}

// TestE2E_TaskUpdateFlow tests the complete task update flow from recording to broadcast.
func TestE2E_TaskUpdateFlow(t *testing.T) {
	password := "task-flow-test"
	testServer, metricsStore, broadcaster, authMiddleware, cancel := setupE2ETestServer(t, password)
	defer testServer.Close()
	defer cancel()

	time.Sleep(50 * time.Millisecond)

	// Connect WebSocket client
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"
	_, wsCookie, _ := authMiddleware.CreateSession()

	dialer := websocket.Dialer{}
	header := http.Header{}
	header.Add("Cookie", fmt.Sprintf("%s=%s", wsCookie.Name, wsCookie.Value))

	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("failed to connect WebSocket: %v", err)
	}
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	// Create HTTP client for API requests
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Login and get session
	form := url.Values{}
	form.Set("password", password)
	resp, _ := client.Post(testServer.URL+"/login", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == auth.SessionCookieName && c.MaxAge > 0 {
			sessionCookie = c
			break
		}
	}
	resp.Body.Close()

	if sessionCookie == nil {
		t.Fatal("no session cookie")
	}

	// Set up message receiver
	receivedMsgs := make(chan string, 10)
	go func() {
		for {
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			_, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			receivedMsgs <- string(message)
		}
	}()

	// Simulate task lifecycle: start -> progress -> complete
	taskID := "lifecycle-task-001"

	// 1. Broadcast task start
	broadcaster.BroadcastTaskUpdate(webui.TaskUpdateData{
		TaskID:   taskID,
		TaskType: "pdf_analysis",
		Status:   "processing",
		CanvasID: "canvas-1",
	})

	// Wait for start broadcast
	select {
	case msg := <-receivedMsgs:
		if !strings.Contains(msg, "processing") {
			t.Errorf("expected processing status in message: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for start broadcast")
	}

	// 2. Record task completion in metrics store
	metricsStore.RecordTask(metrics.TaskRecord{
		ID:        taskID,
		Type:      "pdf_analysis",
		CanvasID:  "canvas-1",
		Status:    metrics.TaskStatusSuccess,
		StartTime: time.Now().Add(-3 * time.Second),
		EndTime:   time.Now(),
		Duration:  3 * time.Second,
	})

	// 3. Broadcast task completion
	broadcaster.BroadcastTaskUpdate(webui.TaskUpdateData{
		TaskID:   taskID,
		TaskType: "pdf_analysis",
		Status:   "completed",
		CanvasID: "canvas-1",
		Duration: 3 * time.Second,
	})

	// Wait for completion broadcast
	select {
	case msg := <-receivedMsgs:
		if !strings.Contains(msg, "completed") {
			t.Errorf("expected completed status in message: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout waiting for completion broadcast")
	}

	// 4. Verify task is in API response
	req, _ := http.NewRequest(http.MethodGet, testServer.URL+"/api/tasks?limit=10", nil)
	req.AddCookie(sessionCookie)

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("tasks request failed: %v", err)
	}
	defer resp.Body.Close()

	var tasks webui.TasksResponse
	if err := json.NewDecoder(resp.Body).Decode(&tasks); err != nil {
		t.Fatalf("failed to decode tasks: %v", err)
	}

	found := false
	for _, task := range tasks.Tasks {
		if task.ID == taskID {
			found = true
			if task.Status != metrics.TaskStatusSuccess {
				t.Errorf("expected task status 'success', got '%s'", task.Status)
			}
			break
		}
	}

	if !found {
		t.Error("task not found in API response")
	}
}

// TestE2E_ConcurrentAPIAndWebSocket tests concurrent API requests and WebSocket broadcasts.
func TestE2E_ConcurrentAPIAndWebSocket(t *testing.T) {
	password := "concurrent-test"
	testServer, metricsStore, broadcaster, authMiddleware, cancel := setupE2ETestServer(t, password)
	defer testServer.Close()
	defer cancel()

	time.Sleep(50 * time.Millisecond)

	// Create authenticated HTTP client
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Login
	form := url.Values{}
	form.Set("password", password)
	resp, _ := client.Post(testServer.URL+"/login", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == auth.SessionCookieName && c.MaxAge > 0 {
			sessionCookie = c
			break
		}
	}
	resp.Body.Close()

	// Connect WebSocket
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"
	_, wsCookie, _ := authMiddleware.CreateSession()

	dialer := websocket.Dialer{}
	header := http.Header{}
	header.Add("Cookie", fmt.Sprintf("%s=%s", wsCookie.Name, wsCookie.Value))

	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("failed to connect WebSocket: %v", err)
	}
	defer conn.Close()

	time.Sleep(100 * time.Millisecond)

	// Run concurrent operations
	var wg sync.WaitGroup
	errCh := make(chan error, 100)

	// Concurrent API requests
	const numRequests = 20
	wg.Add(numRequests)
	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			defer wg.Done()

			endpoints := []string{"/api/status", "/api/canvases", "/api/tasks", "/api/metrics"}
			endpoint := endpoints[idx%len(endpoints)]

			req, _ := http.NewRequest(http.MethodGet, testServer.URL+endpoint, nil)
			req.AddCookie(sessionCookie)

			resp, err := client.Do(req)
			if err != nil {
				errCh <- fmt.Errorf("request %d failed: %w", idx, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				errCh <- fmt.Errorf("request %d got status %d", idx, resp.StatusCode)
			}
		}(i)
	}

	// Concurrent metrics recording
	const numTasks = 10
	wg.Add(numTasks)
	for i := 0; i < numTasks; i++ {
		go func(idx int) {
			defer wg.Done()

			metricsStore.RecordTask(metrics.TaskRecord{
				ID:        fmt.Sprintf("concurrent-task-%d", idx),
				Type:      "test",
				CanvasID:  "canvas-1",
				Status:    metrics.TaskStatusSuccess,
				StartTime: time.Now().Add(-time.Second),
				EndTime:   time.Now(),
				Duration:  time.Second,
			})
		}(i)
	}

	// Concurrent broadcasts
	const numBroadcasts = 10
	wg.Add(numBroadcasts)
	for i := 0; i < numBroadcasts; i++ {
		go func(idx int) {
			defer wg.Done()

			broadcaster.BroadcastTaskUpdate(webui.TaskUpdateData{
				TaskID:   fmt.Sprintf("broadcast-%d", idx),
				TaskType: "test",
				Status:   "completed",
			})
		}(i)
	}

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Check for errors
		close(errCh)
		for err := range errCh {
			t.Error(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("concurrent test timed out - possible deadlock")
	}
}

// TestE2E_GracefulShutdown tests that the server shuts down gracefully.
func TestE2E_GracefulShutdown(t *testing.T) {
	password := "shutdown-test"
	testServer, _, broadcaster, authMiddleware, cancel := setupE2ETestServer(t, password)

	// Connect a WebSocket client
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws"
	_, wsCookie, _ := authMiddleware.CreateSession()

	dialer := websocket.Dialer{}
	header := http.Header{}
	header.Add("Cookie", fmt.Sprintf("%s=%s", wsCookie.Name, wsCookie.Value))

	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("failed to connect WebSocket: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	initialCount := broadcaster.ClientCount()
	if initialCount < 1 {
		t.Error("expected at least 1 client connected")
	}

	// Close the server
	testServer.Close()

	// Cancel context (stops broadcaster)
	cancel()

	// Give time for cleanup
	time.Sleep(100 * time.Millisecond)

	// Connection should be closed
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Error("expected connection to be closed")
	}
}
