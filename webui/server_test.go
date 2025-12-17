package webui

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go_backend/metrics"

	"go.uber.org/zap"
)

// mockMetricsStore implements metrics.MetricsCollector for testing
type mockMetricsStore struct{}

func (m *mockMetricsStore) GetSystemStatus() metrics.SystemStatus {
	return metrics.SystemStatus{
		Health:    "healthy",
		Uptime:    time.Hour,
		LastCheck: time.Now(),
	}
}

func (m *mockMetricsStore) GetAllCanvasStatuses() []metrics.CanvasStatus {
	return []metrics.CanvasStatus{
		{
			ID:   "test-canvas",
			Name: "Test Canvas",
		},
	}
}

func (m *mockMetricsStore) GetCanvasStatus(canvasID string) (metrics.CanvasStatus, bool) {
	return metrics.CanvasStatus{ID: canvasID, Name: "Test Canvas"}, true
}

func (m *mockMetricsStore) GetRecentTasks(limit int) []metrics.TaskRecord {
	return []metrics.TaskRecord{}
}

func (m *mockMetricsStore) GetTaskMetrics() metrics.TaskMetrics {
	return metrics.TaskMetrics{
		TotalProcessed: 100,
		TotalSuccess:   95,
		TotalErrors:    5,
	}
}

func (m *mockMetricsStore) RecordTask(record metrics.TaskRecord) {}

func (m *mockMetricsStore) UpdateCanvasStatus(status metrics.CanvasStatus) {}

func (m *mockMetricsStore) UpdateGPUMetrics(gpu metrics.GPUMetrics) {}

func (m *mockMetricsStore) GetGPUMetrics() metrics.GPUMetrics {
	return metrics.GPUMetrics{}
}

func TestNewServer(t *testing.T) {
	config := DefaultServerConfig()
	logger := zap.NewNop()
	store := &mockMetricsStore{}

	server, err := NewServer(config, store, nil, nil, logger)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	if server == nil {
		t.Fatal("NewServer() returned nil")
	}

	expectedAddr := "localhost:3000"
	if server.Addr() != expectedAddr {
		t.Errorf("Addr() = %q, want %q", server.Addr(), expectedAddr)
	}

	if server.HasAuth() {
		t.Error("HasAuth() = true, want false (no auth provider given)")
	}
}

func TestWebUIServer_HealthEndpoint(t *testing.T) {
	config := DefaultServerConfig()
	store := &mockMetricsStore{}

	server, err := NewServer(config, store, nil, nil, nil)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	// Create test request
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	body, _ := io.ReadAll(rr.Body)
	if !strings.Contains(string(body), "ok") {
		t.Errorf("body = %q, want to contain 'ok'", string(body))
	}
}

func TestWebUIServer_RootRedirect(t *testing.T) {
	config := DefaultServerConfig()
	store := &mockMetricsStore{}

	// Test without auth - should redirect to dashboard
	server, _ := NewServer(config, store, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusTemporaryRedirect)
	}

	location := rr.Header().Get("Location")
	if location != "/dashboard" {
		t.Errorf("Location = %q, want /dashboard", location)
	}
}

func TestWebUIServer_RootRedirectWithAuth(t *testing.T) {
	config := DefaultServerConfig()
	store := &mockMetricsStore{}

	// Create mock auth provider
	mockAuth := &mockAuthProvider{}

	server, _ := NewServer(config, store, nil, mockAuth, nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusTemporaryRedirect)
	}

	location := rr.Header().Get("Location")
	if location != "/login" {
		t.Errorf("Location = %q, want /login", location)
	}
}

func TestWebUIServer_NotFound(t *testing.T) {
	config := DefaultServerConfig()
	store := &mockMetricsStore{}

	server, _ := NewServer(config, store, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rr := httptest.NewRecorder()

	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestWebUIServer_APIStatus(t *testing.T) {
	config := DefaultServerConfig()
	store := &mockMetricsStore{}

	server, _ := NewServer(config, store, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rr := httptest.NewRecorder()

	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body, _ := io.ReadAll(rr.Body)
	if !strings.Contains(string(body), "healthy") {
		t.Errorf("body should contain 'healthy'")
	}
}

func TestWebUIServer_DashboardPage(t *testing.T) {
	config := DefaultServerConfig()
	store := &mockMetricsStore{}

	server, _ := NewServer(config, store, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rr := httptest.NewRecorder()

	server.mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", contentType)
	}

	body, _ := io.ReadAll(rr.Body)
	if !strings.Contains(string(body), "CanvusLocalLLM Dashboard") {
		t.Errorf("body should contain dashboard title")
	}
}

func TestWebUIServer_Shutdown(t *testing.T) {
	config := DefaultServerConfig()
	config.ShutdownTimeout = 1 * time.Second
	store := &mockMetricsStore{}

	server, _ := NewServer(config, store, nil, nil, nil)

	// Create a context for shutdown
	ctx := context.Background()

	// Shutdown should complete without error
	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}
}

func TestDefaultServerConfig(t *testing.T) {
	config := DefaultServerConfig()

	if config.Port != 3000 {
		t.Errorf("Port = %d, want 3000", config.Port)
	}

	if config.Host != "localhost" {
		t.Errorf("Host = %q, want localhost", config.Host)
	}

	if config.ReadTimeout != 30*time.Second {
		t.Errorf("ReadTimeout = %v, want 30s", config.ReadTimeout)
	}

	if config.WriteTimeout != 30*time.Second {
		t.Errorf("WriteTimeout = %v, want 30s", config.WriteTimeout)
	}

	if config.IdleTimeout != 120*time.Second {
		t.Errorf("IdleTimeout = %v, want 120s", config.IdleTimeout)
	}

	if config.ShutdownTimeout != 30*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 30s", config.ShutdownTimeout)
	}
}

func TestWebUIServer_GetBroadcaster(t *testing.T) {
	config := DefaultServerConfig()
	store := &mockMetricsStore{}

	server, _ := NewServer(config, store, nil, nil, nil)

	broadcaster := server.GetBroadcaster()
	if broadcaster == nil {
		t.Error("GetBroadcaster() returned nil")
	}
}

func TestWebUIServer_GetDashboardAPI(t *testing.T) {
	config := DefaultServerConfig()
	store := &mockMetricsStore{}

	server, _ := NewServer(config, store, nil, nil, nil)

	api := server.GetDashboardAPI()
	if api == nil {
		t.Error("GetDashboardAPI() returned nil")
	}
}

func TestWebUIServer_ProtectHandler(t *testing.T) {
	config := DefaultServerConfig()
	store := &mockMetricsStore{}

	// Without auth provider
	server, _ := NewServer(config, store, nil, nil, nil)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	protected := server.ProtectHandler(testHandler)
	if protected == nil {
		t.Error("ProtectHandler() returned nil")
	}

	// Should be the same handler when no auth
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

// mockAuthProvider implements AuthProvider for testing
type mockAuthProvider struct {
	loginCalled  bool
	logoutCalled bool
}

func (m *mockAuthProvider) Middleware(next http.Handler) http.Handler {
	return next
}

func (m *mockAuthProvider) MiddlewareFunc(next http.HandlerFunc) http.HandlerFunc {
	return next
}

func (m *mockAuthProvider) LoginHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.loginCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("login page"))
	}
}

func (m *mockAuthProvider) LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m.logoutCalled = true
		w.WriteHeader(http.StatusOK)
	}
}

func TestWebUIServer_AuthRoutes(t *testing.T) {
	config := DefaultServerConfig()
	store := &mockMetricsStore{}
	mockAuth := &mockAuthProvider{}

	server, _ := NewServer(config, store, nil, mockAuth, nil)

	// Test login route
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()

	server.mux.ServeHTTP(rr, req)

	if !mockAuth.loginCalled {
		t.Error("LoginHandler was not called")
	}

	// Test logout route
	mockAuth.logoutCalled = false
	req = httptest.NewRequest(http.MethodGet, "/logout", nil)
	rr = httptest.NewRecorder()

	server.mux.ServeHTTP(rr, req)

	if !mockAuth.logoutCalled {
		t.Error("LogoutHandler was not called")
	}
}
