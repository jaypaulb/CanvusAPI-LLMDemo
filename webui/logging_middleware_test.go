package webui

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// testLogger captures log entries for testing
type testLogger struct {
	mu      sync.Mutex
	entries []RequestLogEntry
}

func (t *testLogger) LogRequest(entry RequestLogEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.entries = append(t.entries, entry)
}

func (t *testLogger) getEntries() []RequestLogEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	result := make([]RequestLogEntry, len(t.entries))
	copy(result, t.entries)
	return result
}

func TestNewLoggingMiddleware(t *testing.T) {
	m := NewLoggingMiddleware()

	if m == nil {
		t.Fatal("Expected non-nil middleware")
	}

	if m.logger == nil {
		t.Error("Expected logger to be initialized")
	}

	if m.skipPaths == nil {
		t.Error("Expected skipPaths map to be initialized")
	}
}

func TestNewLoggingMiddlewareWithConfig(t *testing.T) {
	logger := &testLogger{}
	config := LoggingMiddlewareConfig{
		Logger:    logger,
		SkipPaths: []string{"/health", "/metrics"},
	}

	m := NewLoggingMiddlewareWithConfig(config)

	if m.logger != logger {
		t.Error("Expected custom logger to be set")
	}

	if !m.skipPaths["/health"] {
		t.Error("Expected /health to be in skipPaths")
	}

	if !m.skipPaths["/metrics"] {
		t.Error("Expected /metrics to be in skipPaths")
	}
}

func TestNewLoggingMiddlewareWithConfig_NilLogger(t *testing.T) {
	config := LoggingMiddlewareConfig{
		Logger: nil,
	}

	m := NewLoggingMiddlewareWithConfig(config)

	if m.logger == nil {
		t.Error("Expected default logger to be set when nil provided")
	}
}

func TestDefaultLoggingMiddlewareConfig(t *testing.T) {
	config := DefaultLoggingMiddlewareConfig()

	if config.Logger == nil {
		t.Error("Expected default logger")
	}

	if config.SkipPaths != nil {
		t.Error("Expected nil SkipPaths by default")
	}

	if config.LogUserAgent {
		t.Error("Expected LogUserAgent to be false by default")
	}

	if config.LogContentLength {
		t.Error("Expected LogContentLength to be false by default")
	}
}

func TestLoggingMiddleware_Handler_BasicRequest(t *testing.T) {
	logger := &testLogger{}
	m := NewLoggingMiddlewareWithConfig(LoggingMiddlewareConfig{
		Logger: logger,
	})

	// Create a simple handler that returns 200 OK
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with middleware
	wrapped := m.Handler(handler)

	// Make a request
	req := httptest.NewRequest("GET", "/test/path", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	// Check response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Check log entry
	entries := logger.getEntries()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(entries))
	}

	entry := entries[0]
	if entry.Method != "GET" {
		t.Errorf("Expected method GET, got %s", entry.Method)
	}

	if entry.Path != "/test/path" {
		t.Errorf("Expected path /test/path, got %s", entry.Path)
	}

	if entry.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", entry.StatusCode)
	}

	if entry.Duration <= 0 {
		t.Error("Expected positive duration")
	}

	if !strings.Contains(entry.RemoteAddr, "192.168.1.1") {
		t.Errorf("Expected RemoteAddr to contain 192.168.1.1, got %s", entry.RemoteAddr)
	}
}

func TestLoggingMiddleware_Handler_DifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"OK", http.StatusOK},
		{"Created", http.StatusCreated},
		{"NoContent", http.StatusNoContent},
		{"BadRequest", http.StatusBadRequest},
		{"Unauthorized", http.StatusUnauthorized},
		{"NotFound", http.StatusNotFound},
		{"InternalServerError", http.StatusInternalServerError},
		{"ServiceUnavailable", http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := &testLogger{}
			m := NewLoggingMiddlewareWithConfig(LoggingMiddlewareConfig{
				Logger: logger,
			})

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			wrapped := m.Handler(handler)
			req := httptest.NewRequest("GET", "/", nil)
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			entries := logger.getEntries()
			if len(entries) != 1 {
				t.Fatalf("Expected 1 log entry, got %d", len(entries))
			}

			if entries[0].StatusCode != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, entries[0].StatusCode)
			}
		})
	}
}

func TestLoggingMiddleware_Handler_DifferentMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS", "HEAD"}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			logger := &testLogger{}
			m := NewLoggingMiddlewareWithConfig(LoggingMiddlewareConfig{
				Logger: logger,
			})

			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			wrapped := m.Handler(handler)
			req := httptest.NewRequest(method, "/", nil)
			rec := httptest.NewRecorder()

			wrapped.ServeHTTP(rec, req)

			entries := logger.getEntries()
			if len(entries) != 1 {
				t.Fatalf("Expected 1 log entry, got %d", len(entries))
			}

			if entries[0].Method != method {
				t.Errorf("Expected method %s, got %s", method, entries[0].Method)
			}
		})
	}
}

func TestLoggingMiddleware_Handler_SkipPaths(t *testing.T) {
	logger := &testLogger{}
	m := NewLoggingMiddlewareWithConfig(LoggingMiddlewareConfig{
		Logger:    logger,
		SkipPaths: []string{"/health", "/metrics"},
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := m.Handler(handler)

	// Request to skipped path
	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Should not log
	entries := logger.getEntries()
	if len(entries) != 0 {
		t.Errorf("Expected 0 log entries for skipped path, got %d", len(entries))
	}

	// Request to normal path
	req = httptest.NewRequest("GET", "/api/status", nil)
	rec = httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	// Should log
	entries = logger.getEntries()
	if len(entries) != 1 {
		t.Errorf("Expected 1 log entry for normal path, got %d", len(entries))
	}
}

func TestLoggingMiddleware_Handler_Duration(t *testing.T) {
	logger := &testLogger{}
	m := NewLoggingMiddlewareWithConfig(LoggingMiddlewareConfig{
		Logger: logger,
	})

	// Handler that takes some time
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	wrapped := m.Handler(handler)
	req := httptest.NewRequest("GET", "/slow", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	entries := logger.getEntries()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(entries))
	}

	// Duration should be at least 50ms
	if entries[0].Duration < 50*time.Millisecond {
		t.Errorf("Expected duration >= 50ms, got %v", entries[0].Duration)
	}
}

func TestLoggingMiddleware_Handler_BytesWritten(t *testing.T) {
	logger := &testLogger{}
	m := NewLoggingMiddlewareWithConfig(LoggingMiddlewareConfig{
		Logger: logger,
	})

	responseBody := "Hello, World! This is a test response."
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseBody))
	})

	wrapped := m.Handler(handler)
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	entries := logger.getEntries()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(entries))
	}

	expectedBytes := int64(len(responseBody))
	if entries[0].ContentLength != expectedBytes {
		t.Errorf("Expected ContentLength %d, got %d", expectedBytes, entries[0].ContentLength)
	}
}

func TestLoggingMiddleware_Handler_XForwardedFor(t *testing.T) {
	logger := &testLogger{}
	m := NewLoggingMiddlewareWithConfig(LoggingMiddlewareConfig{
		Logger: logger,
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := m.Handler(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	entries := logger.getEntries()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(entries))
	}

	// Should use first IP from X-Forwarded-For
	if entries[0].RemoteAddr != "10.0.0.1" {
		t.Errorf("Expected RemoteAddr 10.0.0.1, got %s", entries[0].RemoteAddr)
	}
}

func TestLoggingMiddleware_Handler_XRealIP(t *testing.T) {
	logger := &testLogger{}
	m := NewLoggingMiddlewareWithConfig(LoggingMiddlewareConfig{
		Logger: logger,
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := m.Handler(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Real-IP", "172.16.0.1")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	entries := logger.getEntries()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(entries))
	}

	if entries[0].RemoteAddr != "172.16.0.1" {
		t.Errorf("Expected RemoteAddr 172.16.0.1, got %s", entries[0].RemoteAddr)
	}
}

func TestLoggingMiddleware_Handler_UserAgent(t *testing.T) {
	logger := &testLogger{}
	m := NewLoggingMiddlewareWithConfig(LoggingMiddlewareConfig{
		Logger: logger,
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := m.Handler(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("User-Agent", "TestBrowser/1.0")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	entries := logger.getEntries()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(entries))
	}

	if entries[0].UserAgent != "TestBrowser/1.0" {
		t.Errorf("Expected UserAgent TestBrowser/1.0, got %s", entries[0].UserAgent)
	}
}

func TestLoggingMiddleware_HandlerFunc(t *testing.T) {
	logger := &testLogger{}
	m := NewLoggingMiddlewareWithConfig(LoggingMiddlewareConfig{
		Logger: logger,
	})

	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}

	wrapped := m.HandlerFunc(handler)

	req := httptest.NewRequest("POST", "/create", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	entries := logger.getEntries()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(entries))
	}

	if entries[0].Method != "POST" {
		t.Errorf("Expected method POST, got %s", entries[0].Method)
	}

	if entries[0].StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", entries[0].StatusCode)
	}
}

func TestLoggingMiddleware_Handler_DefaultStatus(t *testing.T) {
	logger := &testLogger{}
	m := NewLoggingMiddlewareWithConfig(LoggingMiddlewareConfig{
		Logger: logger,
	})

	// Handler that writes without calling WriteHeader
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("No explicit status"))
	})

	wrapped := m.Handler(handler)
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	entries := logger.getEntries()
	if len(entries) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(entries))
	}

	// Should default to 200
	if entries[0].StatusCode != http.StatusOK {
		t.Errorf("Expected default status 200, got %d", entries[0].StatusCode)
	}
}

func TestLoggingMiddleware_Handler_Concurrent(t *testing.T) {
	logger := &testLogger{}
	m := NewLoggingMiddlewareWithConfig(LoggingMiddlewareConfig{
		Logger: logger,
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	wrapped := m.Handler(handler)

	numRequests := 50
	var wg sync.WaitGroup
	wg.Add(numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/concurrent", nil)
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)
		}(i)
	}

	wg.Wait()

	entries := logger.getEntries()
	if len(entries) != numRequests {
		t.Errorf("Expected %d log entries, got %d", numRequests, len(entries))
	}
}

func TestGetStatusColor(t *testing.T) {
	tests := []struct {
		status   int
		expected string
	}{
		{200, colorGreen},
		{201, colorGreen},
		{204, colorGreen},
		{301, colorCyan},
		{302, colorCyan},
		{304, colorCyan},
		{400, colorYellow},
		{401, colorYellow},
		{404, colorYellow},
		{500, colorRed},
		{502, colorRed},
		{503, colorRed},
		{100, colorReset},
		{199, colorReset},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			color := getStatusColor(tt.status)
			if color != tt.expected {
				t.Errorf("Expected color %q for status %d, got %q", tt.expected, tt.status, color)
			}
		})
	}
}

func TestFormattedLogger(t *testing.T) {
	var logged string
	logger := &FormattedLogger{
		Format:     "%m %p %s",
		TimeFormat: time.RFC3339,
		Output: func(format string, v ...interface{}) {
			logged = v[0].(string)
		},
	}

	entry := RequestLogEntry{
		Timestamp:  time.Date(2025, 12, 17, 10, 30, 0, 0, time.UTC),
		Method:     "POST",
		Path:       "/api/test",
		StatusCode: 201,
		Duration:   150 * time.Millisecond,
		RemoteAddr: "127.0.0.1",
	}

	logger.LogRequest(entry)

	if logged != "POST /api/test 201" {
		t.Errorf("Expected 'POST /api/test 201', got '%s'", logged)
	}
}

func TestFormattedLogger_AllPlaceholders(t *testing.T) {
	var logged string
	logger := &FormattedLogger{
		Format:     "%t|%m|%p|%s|%d|%a|%u|%l|%%",
		TimeFormat: "15:04:05",
		Output: func(format string, v ...interface{}) {
			logged = v[0].(string)
		},
	}

	entry := RequestLogEntry{
		Timestamp:     time.Date(2025, 12, 17, 10, 30, 0, 0, time.UTC),
		Method:        "GET",
		Path:          "/test",
		StatusCode:    200,
		Duration:      100 * time.Millisecond,
		RemoteAddr:    "10.0.0.1",
		UserAgent:     "TestAgent",
		ContentLength: 1234,
	}

	logger.LogRequest(entry)

	expected := "10:30:00|GET|/test|200|100ms|10.0.0.1|TestAgent|1234|%"
	if logged != expected {
		t.Errorf("Expected '%s', got '%s'", expected, logged)
	}
}

func TestFormattedLogger_DefaultFormat(t *testing.T) {
	var logged string
	logger := &FormattedLogger{
		Output: func(format string, v ...interface{}) {
			logged = v[0].(string)
		},
	}

	entry := RequestLogEntry{
		Timestamp:  time.Now(),
		Method:     "GET",
		Path:       "/",
		StatusCode: 200,
		Duration:   10 * time.Millisecond,
		RemoteAddr: "localhost",
	}

	logger.LogRequest(entry)

	// Default format includes all fields
	if !strings.Contains(logged, "GET") {
		t.Error("Expected logged string to contain method")
	}
	if !strings.Contains(logged, "/") {
		t.Error("Expected logged string to contain path")
	}
}

func TestJSONLogger(t *testing.T) {
	var logged string
	logger := &JSONLogger{
		Output: func(format string, v ...interface{}) {
			logged = fmt.Sprintf(format, v...)
		},
	}

	entry := RequestLogEntry{
		Timestamp:  time.Date(2025, 12, 17, 10, 30, 0, 0, time.UTC),
		Method:     "POST",
		Path:       "/api/users",
		StatusCode: 201,
		Duration:   50 * time.Millisecond,
		RemoteAddr: "192.168.1.100",
	}

	logger.LogRequest(entry)

	// Check JSON structure
	if !strings.Contains(logged, `"method":"POST"`) {
		t.Errorf("Expected JSON to contain method, got: %s", logged)
	}
	if !strings.Contains(logged, `"path":"/api/users"`) {
		t.Errorf("Expected JSON to contain path, got: %s", logged)
	}
	if !strings.Contains(logged, `"status":201`) {
		t.Errorf("Expected JSON to contain status, got: %s", logged)
	}
	if !strings.Contains(logged, `"duration_ms":50`) {
		t.Errorf("Expected JSON to contain duration_ms, got: %s", logged)
	}
}

func TestNoopLogger(t *testing.T) {
	logger := &NoopLogger{}

	// Should not panic
	logger.LogRequest(RequestLogEntry{
		Method:     "GET",
		Path:       "/test",
		StatusCode: 200,
	})
}

func TestResponseWriterWrapper_MultipleWrites(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := &responseWriterWrapper{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	wrapper.Write([]byte("First "))
	wrapper.Write([]byte("Second "))
	wrapper.Write([]byte("Third"))

	// "First " (6) + "Second " (7) + "Third" (5) = 18 bytes
	if wrapper.bytesWritten != 18 {
		t.Errorf("Expected 18 bytes written, got %d", wrapper.bytesWritten)
	}

	if rec.Body.String() != "First Second Third" {
		t.Errorf("Expected 'First Second Third', got '%s'", rec.Body.String())
	}
}

func TestResponseWriterWrapper_WriteHeaderOnce(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := &responseWriterWrapper{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
	}

	// Write header multiple times
	wrapper.WriteHeader(http.StatusCreated)
	wrapper.WriteHeader(http.StatusBadRequest)

	// First call should win
	if wrapper.statusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", wrapper.statusCode)
	}
}
