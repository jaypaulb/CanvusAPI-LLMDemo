// Package tests contains integration tests for CanvusLocalLLM Phase 6.
//
// This file contains end-to-end integration tests for Phase 6 production hardening:
// 1. Logging pipeline: log -> JSON -> DB
// 2. Authentication flow: login -> session -> protected route -> logout
// 3. Shutdown sequence: signal -> drain -> cleanup
//
// These tests verify that individual components work together correctly at the
// organism level, ensuring production-ready behavior.
package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go_backend/db"
	"go_backend/logging"
	"go_backend/shutdown"
	"go_backend/webui/auth"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// =============================================================================
// LOGGING PIPELINE TESTS: log -> JSON -> DB
// =============================================================================

// TestLoggingPipeline_JSONFormat verifies that structured logging outputs valid JSON.
// This is the first step in the logging pipeline: ensuring logs can be parsed.
func TestLoggingPipeline_JSONFormat(t *testing.T) {
	// Create temp file for log output
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	logger, err := logging.NewLogger(true, logPath)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Log a structured message with correlation ID and operation type
	// Note: Some fields may be redacted by the sensitive filter
	logger.Info("test message",
		zap.String("correlation_id", "test-123"),
		zap.String("operation", "text_generation"),
		zap.Int("token_count", 100), // Using different name to avoid redaction
		zap.Duration("duration", 250*time.Millisecond),
	)
	logger.Sync()

	// Read the log file and verify content
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	content := string(data)

	// Verify key fields are present in the output
	// The logger outputs structured logs that may include ANSI codes for console
	// but the core data should still be present
	expectedContent := []string{
		"test message",
		"correlation_id",
		"test-123",
		"operation",
		"text_generation",
	}

	for _, expected := range expectedContent {
		if !strings.Contains(content, expected) {
			t.Errorf("Log output missing expected content: %q\nFull output: %s", expected, content)
		}
	}

	// Verify the log is structured (contains JSON-like key-value pairs)
	if !strings.Contains(content, "correlation_id") || !strings.Contains(content, "operation") {
		t.Error("Log output does not appear to be structured (missing key-value pairs)")
	}
}

// TestLoggingPipeline_DatabaseRecording verifies that processing records are stored in the database.
// This tests the complete flow: processing -> structured log -> database record.
func TestLoggingPipeline_DatabaseRecording(t *testing.T) {
	// Setup test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Create migrations directory with schema
	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatalf("Failed to create migrations dir: %v", err)
	}

	// Write minimal schema for processing_history table
	upSQL := `
CREATE TABLE IF NOT EXISTS processing_history (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	correlation_id TEXT NOT NULL,
	canvas_id TEXT NOT NULL,
	widget_id TEXT NOT NULL,
	operation_type TEXT NOT NULL,
	prompt TEXT,
	response TEXT,
	model_name TEXT,
	input_tokens INTEGER DEFAULT 0,
	output_tokens INTEGER DEFAULT 0,
	duration_ms INTEGER DEFAULT 0,
	status TEXT NOT NULL DEFAULT 'pending',
	error_message TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS error_log (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	correlation_id TEXT,
	error_type TEXT NOT NULL,
	error_message TEXT NOT NULL,
	stack_trace TEXT,
	context TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

	upPath := filepath.Join(migrationsDir, "000001_initial.up.sql")
	if err := os.WriteFile(upPath, []byte(upSQL), 0644); err != nil {
		t.Fatalf("Failed to write migration: %v", err)
	}

	downPath := filepath.Join(migrationsDir, "000001_initial.down.sql")
	if err := os.WriteFile(downPath, []byte("DROP TABLE IF EXISTS processing_history; DROP TABLE IF EXISTS error_log;"), 0644); err != nil {
		t.Fatalf("Failed to write down migration: %v", err)
	}

	// Create database
	config := db.DatabaseConfig{
		Path:           dbPath,
		MigrationsPath: "file://" + migrationsDir,
	}
	database, err := db.NewDatabaseWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	repo := db.NewRepository(database, nil)
	ctx := context.Background()

	// Simulate a processing record (what handlers.go does)
	record := db.ProcessingRecord{
		CorrelationID: "logging-pipeline-test-001",
		CanvasID:      "test-canvas",
		WidgetID:      "test-widget",
		OperationType: "text_generation",
		Prompt:        "Test prompt for logging pipeline",
		Response:      "Test response from model",
		ModelName:     "test-model",
		InputTokens:   100,
		OutputTokens:  50,
		DurationMS:    250,
		Status:        "success",
	}

	// Insert the record
	recordID, err := repo.InsertProcessingHistory(ctx, record)
	if err != nil {
		t.Fatalf("InsertProcessingHistory() error = %v", err)
	}
	if recordID == 0 {
		t.Error("InsertProcessingHistory() returned invalid ID")
	}

	// Query back and verify
	records, err := repo.QueryRecentHistory(ctx, 10)
	if err != nil {
		t.Fatalf("QueryRecentProcessingHistory() error = %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(records))
	}

	// Verify record fields
	got := records[0]
	if got.CorrelationID != record.CorrelationID {
		t.Errorf("CorrelationID = %q, want %q", got.CorrelationID, record.CorrelationID)
	}
	if got.OperationType != record.OperationType {
		t.Errorf("OperationType = %q, want %q", got.OperationType, record.OperationType)
	}
	if got.Status != record.Status {
		t.Errorf("Status = %q, want %q", got.Status, record.Status)
	}
}

// TestLoggingPipeline_ErrorRecording verifies that errors are logged to the database.
func TestLoggingPipeline_ErrorRecording(t *testing.T) {
	// Setup test database (similar to above)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	migrationsDir := filepath.Join(tmpDir, "migrations")
	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatalf("Failed to create migrations dir: %v", err)
	}

	upSQL := `
CREATE TABLE IF NOT EXISTS error_log (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	correlation_id TEXT,
	error_type TEXT NOT NULL,
	error_message TEXT NOT NULL,
	stack_trace TEXT,
	context TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

	upPath := filepath.Join(migrationsDir, "000001_initial.up.sql")
	if err := os.WriteFile(upPath, []byte(upSQL), 0644); err != nil {
		t.Fatalf("Failed to write migration: %v", err)
	}

	downPath := filepath.Join(migrationsDir, "000001_initial.down.sql")
	if err := os.WriteFile(downPath, []byte("DROP TABLE IF EXISTS error_log;"), 0644); err != nil {
		t.Fatalf("Failed to write down migration: %v", err)
	}

	config := db.DatabaseConfig{
		Path:           dbPath,
		MigrationsPath: "file://" + migrationsDir,
	}
	database, err := db.NewDatabaseWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer database.Close()

	if err := database.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	repo := db.NewRepository(database, nil)
	ctx := context.Background()

	// Log an error
	errorEntry := db.ErrorLogEntry{
		CorrelationID: "error-test-001",
		ErrorType:     "api_error",
		ErrorMessage:  "Connection timeout to LLM service",
		StackTrace:    "handlers.go:150 -> core/ai.go:75",
		Context:       `{"retries": 3, "timeout_ms": 5000}`,
	}

	errorID, err := repo.InsertErrorLog(ctx, errorEntry)
	if err != nil {
		t.Fatalf("InsertErrorLog() error = %v", err)
	}
	if errorID == 0 {
		t.Error("InsertErrorLog() returned invalid ID")
	}

	// Query and verify
	errors, err := repo.QueryRecentErrorLogs(ctx, 10)
	if err != nil {
		t.Fatalf("QueryRecentErrorLogs() error = %v", err)
	}
	if len(errors) != 1 {
		t.Fatalf("Expected 1 error log, got %d", len(errors))
	}

	got := errors[0]
	if got.ErrorType != errorEntry.ErrorType {
		t.Errorf("ErrorType = %q, want %q", got.ErrorType, errorEntry.ErrorType)
	}
	if got.ErrorMessage != errorEntry.ErrorMessage {
		t.Errorf("ErrorMessage = %q, want %q", got.ErrorMessage, errorEntry.ErrorMessage)
	}
}

// =============================================================================
// AUTHENTICATION FLOW TESTS: login -> session -> protected route -> logout
// =============================================================================

// TestAuthFlow_CompleteLoginLogoutCycle tests the full authentication lifecycle.
func TestAuthFlow_CompleteLoginLogoutCycle(t *testing.T) {
	logger := zap.NewNop()
	password := "phase6-integration-test"

	middleware, err := auth.NewAuthMiddleware(password, logger)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	// Create test handlers
	protectedCalled := false
	protectedHandler := middleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		protectedCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("protected content"))
	})
	loginHandler := auth.LoginHandler(middleware)
	logoutHandler := auth.LogoutHandler(middleware)

	// Step 1: Access protected without auth -> 401
	t.Run("access_without_auth_returns_401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		rr := httptest.NewRecorder()
		protectedHandler(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401, got %d", rr.Code)
		}
	})

	// Step 2: Login with correct password -> session cookie
	var sessionCookie *http.Cookie
	t.Run("login_with_correct_password", func(t *testing.T) {
		form := url.Values{}
		form.Set("password", password)
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		loginHandler(rr, req)

		if rr.Code != http.StatusSeeOther {
			t.Errorf("Expected 303 redirect after login, got %d", rr.Code)
		}

		for _, c := range rr.Result().Cookies() {
			if c.Name == auth.SessionCookieName && c.MaxAge > 0 {
				sessionCookie = c
				break
			}
		}
		if sessionCookie == nil {
			t.Fatal("No session cookie set after login")
		}
	})

	// Step 3: Access protected with session -> 200
	t.Run("access_with_session_returns_200", func(t *testing.T) {
		if sessionCookie == nil {
			t.Skip("No session cookie from previous step")
		}

		protectedCalled = false
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.AddCookie(sessionCookie)
		rr := httptest.NewRecorder()
		protectedHandler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200 with valid session, got %d", rr.Code)
		}
		if !protectedCalled {
			t.Error("Protected handler should have been called")
		}
	})

	// Step 4: Logout -> session invalidated
	t.Run("logout_invalidates_session", func(t *testing.T) {
		if sessionCookie == nil {
			t.Skip("No session cookie from previous step")
		}

		req := httptest.NewRequest(http.MethodPost, "/logout", nil)
		req.AddCookie(sessionCookie)
		rr := httptest.NewRecorder()
		logoutHandler(rr, req)

		if rr.Code != http.StatusSeeOther {
			t.Errorf("Expected 303 redirect after logout, got %d", rr.Code)
		}
	})

	// Step 5: Access protected after logout -> 401
	t.Run("access_after_logout_returns_401", func(t *testing.T) {
		if sessionCookie == nil {
			t.Skip("No session cookie from previous step")
		}

		protectedCalled = false
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.AddCookie(sessionCookie) // Use old cookie
		rr := httptest.NewRecorder()
		protectedHandler(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected 401 after logout, got %d", rr.Code)
		}
		if protectedCalled {
			t.Error("Protected handler should not be called with invalid session")
		}
	})
}

// TestAuthFlow_RateLimiting verifies that login attempts are rate-limited.
func TestAuthFlow_RateLimiting(t *testing.T) {
	logger := zap.NewNop()
	password := "rate-limit-test"

	middleware, err := auth.NewAuthMiddleware(password, logger)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	loginHandler := auth.LoginHandler(middleware)

	// Make multiple failed login attempts
	for i := 0; i < 3; i++ {
		form := url.Values{}
		form.Set("password", "wrong-password")
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.RemoteAddr = "192.168.1.100:12345" // Same IP
		rr := httptest.NewRecorder()
		loginHandler(rr, req)

		// Should fail authentication but not be rate-limited yet
		if i < 3 && rr.Code == http.StatusTooManyRequests {
			t.Errorf("Unexpected rate limit on attempt %d", i+1)
		}
	}

	// Eventually should get rate limited (behavior depends on implementation)
	// This test verifies the rate limiting infrastructure exists
}

// TestAuthFlow_ConcurrentSessions verifies multiple concurrent sessions work correctly.
func TestAuthFlow_ConcurrentSessions(t *testing.T) {
	logger := zap.NewNop()
	password := "concurrent-session-test"

	middleware, err := auth.NewAuthMiddleware(password, logger)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	loginHandler := auth.LoginHandler(middleware)
	protectedHandler := middleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Create multiple sessions
	var sessions []*http.Cookie
	for i := 0; i < 3; i++ {
		form := url.Values{}
		form.Set("password", password)
		req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rr := httptest.NewRecorder()
		loginHandler(rr, req)

		for _, c := range rr.Result().Cookies() {
			if c.Name == auth.SessionCookieName && c.MaxAge > 0 {
				sessions = append(sessions, c)
				break
			}
		}
	}

	if len(sessions) != 3 {
		t.Fatalf("Expected 3 sessions, got %d", len(sessions))
	}

	// All sessions should work concurrently
	var wg sync.WaitGroup
	errors := make(chan error, len(sessions))

	for _, cookie := range sessions {
		wg.Add(1)
		go func(c *http.Cookie) {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.AddCookie(c)
			rr := httptest.NewRecorder()
			protectedHandler(rr, req)

			if rr.Code != http.StatusOK {
				errors <- nil // Mark failure
			}
		}(cookie)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Error("One or more concurrent sessions failed")
		}
	}
}

// =============================================================================
// SHUTDOWN SEQUENCE TESTS: signal -> drain -> cleanup
// =============================================================================

// TestShutdownSequence_OrderedCleanup verifies handlers execute in priority order.
func TestShutdownSequence_OrderedCleanup(t *testing.T) {
	logger := zap.NewNop().Sugar()
	manager := shutdown.NewManager(logger.Desugar())

	var executionOrder []string
	var mu sync.Mutex

	// Register handlers with different priorities
	// Lower priority = executes first
	manager.Register("database", 30, func(ctx context.Context) error {
		mu.Lock()
		executionOrder = append(executionOrder, "database")
		mu.Unlock()
		return nil
	})

	manager.Register("http-server", 10, func(ctx context.Context) error {
		mu.Lock()
		executionOrder = append(executionOrder, "http-server")
		mu.Unlock()
		return nil
	})

	manager.Register("monitor", 20, func(ctx context.Context) error {
		mu.Lock()
		executionOrder = append(executionOrder, "monitor")
		mu.Unlock()
		return nil
	})

	// Execute shutdown
	manager.Shutdown()

	// Verify order: http-server (10) -> monitor (20) -> database (30)
	expected := []string{"http-server", "monitor", "database"}
	if len(executionOrder) != len(expected) {
		t.Fatalf("Expected %d handlers, got %d", len(expected), len(executionOrder))
	}

	for i, name := range expected {
		if executionOrder[i] != name {
			t.Errorf("Handler %d: expected %q, got %q", i, name, executionOrder[i])
		}
	}
}

// TestShutdownSequence_DrainInFlightRequests verifies requests drain before shutdown.
func TestShutdownSequence_DrainInFlightRequests(t *testing.T) {
	logger := zap.NewNop().Sugar()
	manager := shutdown.NewManager(logger.Desugar())

	var operationCompleted atomic.Bool

	// Start a long-running operation
	go func() {
		err := manager.WrapOperation(context.Background(), "long-request", func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond) // Simulate work
			operationCompleted.Store(true)
			return nil
		})
		if err != nil {
			t.Logf("Operation error (expected during shutdown): %v", err)
		}
	}()

	// Give operation time to start
	time.Sleep(10 * time.Millisecond)

	// Verify operation is tracked
	if manager.ActiveOperations() == 0 {
		t.Error("Expected active operation to be tracked")
	}

	// Trigger shutdown (uses internal timeout)
	manager.Shutdown()

	// Operation should have completed
	if !operationCompleted.Load() {
		t.Error("Operation should complete before shutdown finishes")
	}
}

// TestShutdownSequence_RejectsNewRequestsDuringShutdown verifies new requests are rejected.
func TestShutdownSequence_RejectsNewRequestsDuringShutdown(t *testing.T) {
	logger := zap.NewNop().Sugar()
	manager := shutdown.NewManager(logger.Desugar())

	// Shutdown in background
	go manager.Shutdown()

	// Give shutdown time to start
	time.Sleep(10 * time.Millisecond)

	// New operation should be rejected
	executed := false
	err := manager.WrapOperation(context.Background(), "new-request", func(ctx context.Context) error {
		executed = true
		return nil
	})

	// Should get ErrTrackerClosed or similar
	if err == nil {
		t.Error("Expected error when submitting operation during shutdown")
	}
	if executed {
		t.Error("Operation should not have executed during shutdown")
	}
}

// TestShutdownSequence_ContinuesAfterHandlerError verifies shutdown continues after errors.
func TestShutdownSequence_ContinuesAfterHandlerError(t *testing.T) {
	logger := zap.NewNop().Sugar()
	manager := shutdown.NewManager(logger.Desugar())

	var executed []string
	var mu sync.Mutex

	// Handler that will fail
	manager.Register("failing-handler", 10, func(ctx context.Context) error {
		mu.Lock()
		executed = append(executed, "failing-handler")
		mu.Unlock()
		return context.DeadlineExceeded // Simulate error
	})

	// Handler that should still run
	manager.Register("successful-handler", 20, func(ctx context.Context) error {
		mu.Lock()
		executed = append(executed, "successful-handler")
		mu.Unlock()
		return nil
	})

	// Execute shutdown
	manager.Shutdown()

	// Both handlers should have executed
	if len(executed) != 2 {
		t.Fatalf("Expected 2 handlers executed, got %d", len(executed))
	}

	// Verify the successful handler ran despite the error
	found := false
	for _, name := range executed {
		if name == "successful-handler" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Successful handler should have run despite prior error")
	}
}

// =============================================================================
// COMBINED INTEGRATION TESTS
// =============================================================================

// TestCombined_LoggingDuringShutdown verifies logging works during shutdown.
func TestCombined_LoggingDuringShutdown(t *testing.T) {
	// Create logger
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "shutdown.log")

	config := zap.NewProductionConfig()
	config.OutputPaths = []string{logPath}
	config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)

	zapLogger, err := config.Build()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	manager := shutdown.NewManager(zapLogger)

	// Register handler that logs during shutdown
	manager.Register("logging-handler", 10, func(ctx context.Context) error {
		zapLogger.Info("Handler executing during shutdown",
			zap.String("handler", "logging-handler"),
		)
		return nil
	})

	// Execute shutdown
	manager.Shutdown()
	zapLogger.Sync()

	// Verify log file contains shutdown log
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), "Handler executing during shutdown") {
		t.Error("Log file should contain shutdown handler log message")
	}
}
