package db

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// testSchemaUp is the SQL schema for creating test tables.
// This mirrors the production schema from 001_initial_schema.up.sql.
const testSchemaUp = `
CREATE TABLE processing_history (
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
    status TEXT NOT NULL,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_processing_history_correlation_id ON processing_history(correlation_id);
CREATE INDEX idx_processing_history_canvas_id ON processing_history(canvas_id);
CREATE INDEX idx_processing_history_created_at ON processing_history(created_at);

CREATE TABLE canvas_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    canvas_id TEXT NOT NULL,
    widget_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    widget_type TEXT NOT NULL,
    content_preview TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_canvas_events_canvas_id ON canvas_events(canvas_id);
CREATE INDEX idx_canvas_events_widget_id ON canvas_events(widget_id);
CREATE INDEX idx_canvas_events_created_at ON canvas_events(created_at);

CREATE TABLE performance_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_type TEXT NOT NULL,
    metric_name TEXT NOT NULL,
    metric_value REAL NOT NULL,
    metadata TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_performance_metrics_metric_type ON performance_metrics(metric_type);
CREATE INDEX idx_performance_metrics_created_at ON performance_metrics(created_at);

CREATE TABLE error_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    correlation_id TEXT,
    error_type TEXT NOT NULL,
    error_message TEXT NOT NULL,
    stack_trace TEXT,
    context TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_error_log_correlation_id ON error_log(correlation_id);
CREATE INDEX idx_error_log_error_type ON error_log(error_type);
CREATE INDEX idx_error_log_created_at ON error_log(created_at);

CREATE TABLE system_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_type TEXT NOT NULL,
    cpu_usage REAL,
    memory_used_mb REAL,
    memory_total_mb REAL,
    disk_used_mb REAL,
    disk_total_mb REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_system_metrics_metric_type ON system_metrics(metric_type);
CREATE INDEX idx_system_metrics_created_at ON system_metrics(created_at);
`

const testSchemaDown = `
DROP INDEX IF EXISTS idx_system_metrics_created_at;
DROP INDEX IF EXISTS idx_system_metrics_metric_type;
DROP INDEX IF EXISTS idx_error_log_created_at;
DROP INDEX IF EXISTS idx_error_log_error_type;
DROP INDEX IF EXISTS idx_error_log_correlation_id;
DROP INDEX IF EXISTS idx_performance_metrics_created_at;
DROP INDEX IF EXISTS idx_performance_metrics_metric_type;
DROP INDEX IF EXISTS idx_canvas_events_created_at;
DROP INDEX IF EXISTS idx_canvas_events_widget_id;
DROP INDEX IF EXISTS idx_canvas_events_canvas_id;
DROP INDEX IF EXISTS idx_processing_history_created_at;
DROP INDEX IF EXISTS idx_processing_history_canvas_id;
DROP INDEX IF EXISTS idx_processing_history_correlation_id;
DROP TABLE IF EXISTS system_metrics;
DROP TABLE IF EXISTS error_log;
DROP TABLE IF EXISTS performance_metrics;
DROP TABLE IF EXISTS canvas_events;
DROP TABLE IF EXISTS processing_history;
`

// setupTestMigrationsForRepo creates a temporary migrations directory with test migration files.
// Returns the temp directory path (for db) and migrations path (with file:// prefix).
func setupTestMigrationsForRepo(t *testing.T) (string, string) {
	t.Helper()

	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")

	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatalf("failed to create migrations directory: %v", err)
	}

	// Create up migration
	upPath := filepath.Join(migrationsDir, "000001_initial_schema.up.sql")
	if err := os.WriteFile(upPath, []byte(testSchemaUp), 0644); err != nil {
		t.Fatalf("failed to write up migration: %v", err)
	}

	// Create down migration
	downPath := filepath.Join(migrationsDir, "000001_initial_schema.down.sql")
	if err := os.WriteFile(downPath, []byte(testSchemaDown), 0644); err != nil {
		t.Fatalf("failed to write down migration: %v", err)
	}

	return tmpDir, "file://" + migrationsDir
}

// setupTestRepository creates a test database with migrations and returns a Repository.
func setupTestRepository(t *testing.T) (*Repository, *Database, func()) {
	t.Helper()

	tmpDir, migrationsPath := setupTestMigrationsForRepo(t)
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DatabaseConfig{
		Path:           dbPath,
		MigrationsPath: migrationsPath,
	}

	db, err := NewDatabaseWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Run migrations
	if err := db.Migrate(); err != nil {
		db.Close()
		t.Fatalf("Failed to run migrations: %v", err)
	}

	repo := NewRepository(db, nil)

	cleanup := func() {
		db.Close()
	}

	return repo, db, cleanup
}

// TestInsertProcessingHistory tests inserting and querying processing history.
func TestInsertProcessingHistory(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("insert and query single record", func(t *testing.T) {
		record := ProcessingRecord{
			CorrelationID: "test-corr-001",
			CanvasID:      "canvas-123",
			WidgetID:      "widget-456",
			OperationType: "text_generation",
			Prompt:        "Hello, world!",
			Response:      "Hi there!",
			ModelName:     "llama-2-7b",
			InputTokens:   10,
			OutputTokens:  5,
			DurationMS:    150,
			Status:        "success",
			ErrorMessage:  "",
		}

		id, err := repo.InsertProcessingHistory(ctx, record)
		if err != nil {
			t.Fatalf("InsertProcessingHistory() error = %v", err)
		}
		if id <= 0 {
			t.Errorf("InsertProcessingHistory() returned invalid id = %d", id)
		}

		// Query back
		records, err := repo.QueryRecentHistory(ctx, 10)
		if err != nil {
			t.Fatalf("QueryRecentHistory() error = %v", err)
		}
		if len(records) != 1 {
			t.Fatalf("QueryRecentHistory() returned %d records, want 1", len(records))
		}

		got := records[0]
		if got.CorrelationID != record.CorrelationID {
			t.Errorf("CorrelationID = %v, want %v", got.CorrelationID, record.CorrelationID)
		}
		if got.CanvasID != record.CanvasID {
			t.Errorf("CanvasID = %v, want %v", got.CanvasID, record.CanvasID)
		}
		if got.WidgetID != record.WidgetID {
			t.Errorf("WidgetID = %v, want %v", got.WidgetID, record.WidgetID)
		}
		if got.OperationType != record.OperationType {
			t.Errorf("OperationType = %v, want %v", got.OperationType, record.OperationType)
		}
		if got.Prompt != record.Prompt {
			t.Errorf("Prompt = %v, want %v", got.Prompt, record.Prompt)
		}
		if got.Response != record.Response {
			t.Errorf("Response = %v, want %v", got.Response, record.Response)
		}
		if got.ModelName != record.ModelName {
			t.Errorf("ModelName = %v, want %v", got.ModelName, record.ModelName)
		}
		if got.InputTokens != record.InputTokens {
			t.Errorf("InputTokens = %v, want %v", got.InputTokens, record.InputTokens)
		}
		if got.OutputTokens != record.OutputTokens {
			t.Errorf("OutputTokens = %v, want %v", got.OutputTokens, record.OutputTokens)
		}
		if got.DurationMS != record.DurationMS {
			t.Errorf("DurationMS = %v, want %v", got.DurationMS, record.DurationMS)
		}
		if got.Status != record.Status {
			t.Errorf("Status = %v, want %v", got.Status, record.Status)
		}
	})

	t.Run("query by correlation ID", func(t *testing.T) {
		// Insert another record with different correlation ID
		record := ProcessingRecord{
			CorrelationID: "test-corr-002",
			CanvasID:      "canvas-789",
			WidgetID:      "widget-012",
			OperationType: "image_generation",
			Prompt:        "Generate an image",
			Status:        "success",
		}
		_, err := repo.InsertProcessingHistory(ctx, record)
		if err != nil {
			t.Fatalf("InsertProcessingHistory() error = %v", err)
		}

		// Query by specific correlation ID
		records, err := repo.QueryHistoryByCorrelationID(ctx, "test-corr-002")
		if err != nil {
			t.Fatalf("QueryHistoryByCorrelationID() error = %v", err)
		}
		if len(records) != 1 {
			t.Fatalf("QueryHistoryByCorrelationID() returned %d records, want 1", len(records))
		}
		if records[0].CorrelationID != "test-corr-002" {
			t.Errorf("CorrelationID = %v, want test-corr-002", records[0].CorrelationID)
		}
	})

	t.Run("query ordering is DESC by created_at", func(t *testing.T) {
		// Clear and insert multiple records
		records, err := repo.QueryRecentHistory(ctx, 100)
		if err != nil {
			t.Fatalf("QueryRecentHistory() error = %v", err)
		}

		// Insert a new record
		newRecord := ProcessingRecord{
			CorrelationID: "test-corr-003",
			CanvasID:      "canvas-new",
			WidgetID:      "widget-new",
			OperationType: "text_generation",
			Status:        "pending",
		}
		_, err = repo.InsertProcessingHistory(ctx, newRecord)
		if err != nil {
			t.Fatalf("InsertProcessingHistory() error = %v", err)
		}

		// Query again - newest should be first
		newRecords, err := repo.QueryRecentHistory(ctx, 100)
		if err != nil {
			t.Fatalf("QueryRecentHistory() error = %v", err)
		}

		if len(newRecords) != len(records)+1 {
			t.Fatalf("Expected %d records, got %d", len(records)+1, len(newRecords))
		}

		// Newest record should be first
		if newRecords[0].CorrelationID != "test-corr-003" {
			t.Errorf("First record should be newest, got CorrelationID = %v", newRecords[0].CorrelationID)
		}
	})
}

// TestInsertCanvasEvent tests inserting and querying canvas events.
func TestInsertCanvasEvent(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("insert and query single event", func(t *testing.T) {
		event := CanvasEvent{
			CanvasID:       "canvas-123",
			WidgetID:       "widget-456",
			EventType:      "created",
			WidgetType:     "note",
			ContentPreview: "This is a test note...",
		}

		id, err := repo.InsertCanvasEvent(ctx, event)
		if err != nil {
			t.Fatalf("InsertCanvasEvent() error = %v", err)
		}
		if id <= 0 {
			t.Errorf("InsertCanvasEvent() returned invalid id = %d", id)
		}

		// Query back
		events, err := repo.QueryRecentCanvasEvents(ctx, 10)
		if err != nil {
			t.Fatalf("QueryRecentCanvasEvents() error = %v", err)
		}
		if len(events) != 1 {
			t.Fatalf("QueryRecentCanvasEvents() returned %d events, want 1", len(events))
		}

		got := events[0]
		if got.CanvasID != event.CanvasID {
			t.Errorf("CanvasID = %v, want %v", got.CanvasID, event.CanvasID)
		}
		if got.WidgetID != event.WidgetID {
			t.Errorf("WidgetID = %v, want %v", got.WidgetID, event.WidgetID)
		}
		if got.EventType != event.EventType {
			t.Errorf("EventType = %v, want %v", got.EventType, event.EventType)
		}
		if got.WidgetType != event.WidgetType {
			t.Errorf("WidgetType = %v, want %v", got.WidgetType, event.WidgetType)
		}
		if got.ContentPreview != event.ContentPreview {
			t.Errorf("ContentPreview = %v, want %v", got.ContentPreview, event.ContentPreview)
		}
	})

	t.Run("query by widget ID", func(t *testing.T) {
		// Insert events for different widgets
		event1 := CanvasEvent{
			CanvasID:   "canvas-123",
			WidgetID:   "widget-specific",
			EventType:  "updated",
			WidgetType: "note",
		}
		_, err := repo.InsertCanvasEvent(ctx, event1)
		if err != nil {
			t.Fatalf("InsertCanvasEvent() error = %v", err)
		}

		event2 := CanvasEvent{
			CanvasID:   "canvas-123",
			WidgetID:   "widget-other",
			EventType:  "created",
			WidgetType: "image",
		}
		_, err = repo.InsertCanvasEvent(ctx, event2)
		if err != nil {
			t.Fatalf("InsertCanvasEvent() error = %v", err)
		}

		// Query by specific widget
		events, err := repo.QueryCanvasEventsByWidgetID(ctx, "widget-specific", 10)
		if err != nil {
			t.Fatalf("QueryCanvasEventsByWidgetID() error = %v", err)
		}
		if len(events) != 1 {
			t.Fatalf("QueryCanvasEventsByWidgetID() returned %d events, want 1", len(events))
		}
		if events[0].WidgetID != "widget-specific" {
			t.Errorf("WidgetID = %v, want widget-specific", events[0].WidgetID)
		}
	})
}

// TestInsertErrorLog tests inserting and querying error logs.
func TestInsertErrorLog(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	t.Run("insert and query single entry", func(t *testing.T) {
		entry := ErrorLogEntry{
			CorrelationID: "corr-err-001",
			ErrorType:     "api_error",
			ErrorMessage:  "Connection refused to AI endpoint",
			StackTrace:    "at main.go:123\nat handler.go:456",
			Context:       `{"endpoint": "http://localhost:1234"}`,
		}

		id, err := repo.InsertErrorLog(ctx, entry)
		if err != nil {
			t.Fatalf("InsertErrorLog() error = %v", err)
		}
		if id <= 0 {
			t.Errorf("InsertErrorLog() returned invalid id = %d", id)
		}

		// Query back
		entries, err := repo.QueryRecentErrorLogs(ctx, 10)
		if err != nil {
			t.Fatalf("QueryRecentErrorLogs() error = %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("QueryRecentErrorLogs() returned %d entries, want 1", len(entries))
		}

		got := entries[0]
		if got.CorrelationID != entry.CorrelationID {
			t.Errorf("CorrelationID = %v, want %v", got.CorrelationID, entry.CorrelationID)
		}
		if got.ErrorType != entry.ErrorType {
			t.Errorf("ErrorType = %v, want %v", got.ErrorType, entry.ErrorType)
		}
		if got.ErrorMessage != entry.ErrorMessage {
			t.Errorf("ErrorMessage = %v, want %v", got.ErrorMessage, entry.ErrorMessage)
		}
		if got.StackTrace != entry.StackTrace {
			t.Errorf("StackTrace = %v, want %v", got.StackTrace, entry.StackTrace)
		}
		if got.Context != entry.Context {
			t.Errorf("Context = %v, want %v", got.Context, entry.Context)
		}
	})

	t.Run("insert with empty optional fields", func(t *testing.T) {
		entry := ErrorLogEntry{
			ErrorType:    "validation_error",
			ErrorMessage: "Invalid input",
			// CorrelationID, StackTrace, Context are empty
		}

		id, err := repo.InsertErrorLog(ctx, entry)
		if err != nil {
			t.Fatalf("InsertErrorLog() error = %v", err)
		}
		if id <= 0 {
			t.Errorf("InsertErrorLog() returned invalid id = %d", id)
		}
	})

	t.Run("query by error type", func(t *testing.T) {
		// Insert multiple error types
		entry := ErrorLogEntry{
			ErrorType:    "model_error",
			ErrorMessage: "Model load failed",
		}
		_, err := repo.InsertErrorLog(ctx, entry)
		if err != nil {
			t.Fatalf("InsertErrorLog() error = %v", err)
		}

		// Query by specific type
		entries, err := repo.QueryErrorLogsByType(ctx, "model_error", 10)
		if err != nil {
			t.Fatalf("QueryErrorLogsByType() error = %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("QueryErrorLogsByType() returned %d entries, want 1", len(entries))
		}
		if entries[0].ErrorType != "model_error" {
			t.Errorf("ErrorType = %v, want model_error", entries[0].ErrorType)
		}
	})
}

// TestInsertPerformanceMetric tests inserting performance metrics.
func TestInsertPerformanceMetric(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	metric := PerformanceMetric{
		MetricType:  "inference",
		MetricName:  "tokens_per_second",
		MetricValue: 45.7,
		Metadata:    `{"model": "llama-2-7b"}`,
	}

	id, err := repo.InsertPerformanceMetric(ctx, metric)
	if err != nil {
		t.Fatalf("InsertPerformanceMetric() error = %v", err)
	}
	if id <= 0 {
		t.Errorf("InsertPerformanceMetric() returned invalid id = %d", id)
	}
}

// TestInsertSystemMetric tests inserting system metrics.
func TestInsertSystemMetric(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	metric := SystemMetric{
		MetricType:    "snapshot",
		CPUUsage:      45.5,
		MemoryUsedMB:  4096.0,
		MemoryTotalMB: 16384.0,
		DiskUsedMB:    102400.0,
		DiskTotalMB:   512000.0,
	}

	id, err := repo.InsertSystemMetric(ctx, metric)
	if err != nil {
		t.Fatalf("InsertSystemMetric() error = %v", err)
	}
	if id <= 0 {
		t.Errorf("InsertSystemMetric() returned invalid id = %d", id)
	}
}

// TestRepositoryConcurrentAccess tests thread safety of repository methods.
func TestRepositoryConcurrentAccess(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()
	const numGoroutines = 10
	const opsPerGoroutine = 5

	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines*opsPerGoroutine*3)

	// Concurrent inserts
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			for j := 0; j < opsPerGoroutine; j++ {
				// Insert processing history
				_, err := repo.InsertProcessingHistory(ctx, ProcessingRecord{
					CorrelationID: "concurrent-test",
					CanvasID:      "canvas-concurrent",
					WidgetID:      "widget-concurrent",
					OperationType: "text_generation",
					Status:        "success",
				})
				if err != nil {
					errChan <- err
				}

				// Insert canvas event
				_, err = repo.InsertCanvasEvent(ctx, CanvasEvent{
					CanvasID:   "canvas-concurrent",
					WidgetID:   "widget-concurrent",
					EventType:  "updated",
					WidgetType: "note",
				})
				if err != nil {
					errChan <- err
				}

				// Insert error log
				_, err = repo.InsertErrorLog(ctx, ErrorLogEntry{
					ErrorType:    "test_error",
					ErrorMessage: "Concurrent test error",
				})
				if err != nil {
					errChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		t.Errorf("Concurrent access produced %d errors: %v", len(errors), errors[0])
	}

	// Verify counts
	historyCount, err := repo.CountProcessingHistory(ctx)
	if err != nil {
		t.Fatalf("CountProcessingHistory() error = %v", err)
	}
	expectedHistory := int64(numGoroutines * opsPerGoroutine)
	if historyCount != expectedHistory {
		t.Errorf("Processing history count = %d, want %d", historyCount, expectedHistory)
	}

	eventCount, err := repo.CountCanvasEvents(ctx)
	if err != nil {
		t.Fatalf("CountCanvasEvents() error = %v", err)
	}
	if eventCount != expectedHistory {
		t.Errorf("Canvas events count = %d, want %d", eventCount, expectedHistory)
	}

	errorCount, err := repo.CountErrorLogs(ctx)
	if err != nil {
		t.Fatalf("CountErrorLogs() error = %v", err)
	}
	if errorCount != expectedHistory {
		t.Errorf("Error logs count = %d, want %d", errorCount, expectedHistory)
	}
}

// TestRepositoryClosedDatabase tests behavior with closed database.
func TestRepositoryClosedDatabase(t *testing.T) {
	repo, db, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Close the database
	if err := db.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// All operations should fail gracefully
	t.Run("InsertProcessingHistory on closed db", func(t *testing.T) {
		_, err := repo.InsertProcessingHistory(ctx, ProcessingRecord{
			CorrelationID: "test",
			CanvasID:      "test",
			WidgetID:      "test",
			OperationType: "test",
			Status:        "test",
		})
		if err == nil {
			t.Error("InsertProcessingHistory() should fail on closed database")
		}
	})

	t.Run("QueryRecentHistory on closed db", func(t *testing.T) {
		_, err := repo.QueryRecentHistory(ctx, 10)
		if err == nil {
			t.Error("QueryRecentHistory() should fail on closed database")
		}
	})

	t.Run("InsertCanvasEvent on closed db", func(t *testing.T) {
		_, err := repo.InsertCanvasEvent(ctx, CanvasEvent{
			CanvasID:   "test",
			WidgetID:   "test",
			EventType:  "test",
			WidgetType: "test",
		})
		if err == nil {
			t.Error("InsertCanvasEvent() should fail on closed database")
		}
	})

	t.Run("InsertErrorLog on closed db", func(t *testing.T) {
		_, err := repo.InsertErrorLog(ctx, ErrorLogEntry{
			ErrorType:    "test",
			ErrorMessage: "test",
		})
		if err == nil {
			t.Error("InsertErrorLog() should fail on closed database")
		}
	})
}

// TestRepositoryWithAsyncWriter tests async write functionality.
func TestRepositoryWithAsyncWriter(t *testing.T) {
	tmpDir, migrationsPath := setupTestMigrationsForRepo(t)
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DatabaseConfig{
		Path:           dbPath,
		MigrationsPath: migrationsPath,
	}

	db, err := NewDatabaseWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create repository without async writer first to create the handler
	repo := NewRepository(db, nil)

	// Create async writer with repository's handler
	asyncWriter := NewAsyncWriter(repo.CreateAsyncWriteHandler())
	asyncWriter.Start()
	defer asyncWriter.Close()

	// Update repository with async writer
	repo.asyncWriter = asyncWriter

	ctx := context.Background()

	t.Run("async insert processing history", func(t *testing.T) {
		record := ProcessingRecord{
			CorrelationID: "async-test-001",
			CanvasID:      "canvas-async",
			WidgetID:      "widget-async",
			OperationType: "text_generation",
			Status:        "success",
		}

		// This should be queued asynchronously
		id, err := repo.InsertProcessingHistory(ctx, record)
		if err != nil {
			t.Fatalf("InsertProcessingHistory() error = %v", err)
		}
		// ID should be 0 for async writes
		if id != 0 {
			t.Logf("Note: Got synchronous write (id=%d), async channel may have been full", id)
		}

		// Wait a bit for async processing
		time.Sleep(100 * time.Millisecond)

		// Verify it was written
		records, err := repo.QueryRecentHistory(ctx, 10)
		if err != nil {
			t.Fatalf("QueryRecentHistory() error = %v", err)
		}

		found := false
		for _, r := range records {
			if r.CorrelationID == "async-test-001" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Async write did not complete - record not found")
		}
	})
}

// TestRepositoryNilDatabase tests behavior with nil database.
func TestRepositoryNilDatabase(t *testing.T) {
	repo := NewRepository(nil, nil)
	ctx := context.Background()

	t.Run("InsertProcessingHistory with nil db", func(t *testing.T) {
		_, err := repo.InsertProcessingHistory(ctx, ProcessingRecord{})
		if err == nil {
			t.Error("Expected error for nil database")
		}
	})

	t.Run("QueryRecentHistory with nil db", func(t *testing.T) {
		_, err := repo.QueryRecentHistory(ctx, 10)
		if err == nil {
			t.Error("Expected error for nil database")
		}
	})

	t.Run("InsertCanvasEvent with nil db", func(t *testing.T) {
		_, err := repo.InsertCanvasEvent(ctx, CanvasEvent{})
		if err == nil {
			t.Error("Expected error for nil database")
		}
	})

	t.Run("InsertErrorLog with nil db", func(t *testing.T) {
		_, err := repo.InsertErrorLog(ctx, ErrorLogEntry{})
		if err == nil {
			t.Error("Expected error for nil database")
		}
	})
}

// TestQueryLimitDefault tests that default limit is applied.
func TestQueryLimitDefault(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Insert 15 records
	for i := 0; i < 15; i++ {
		_, err := repo.InsertProcessingHistory(ctx, ProcessingRecord{
			CorrelationID: "limit-test",
			CanvasID:      "canvas-limit",
			WidgetID:      "widget-limit",
			OperationType: "text_generation",
			Status:        "success",
		})
		if err != nil {
			t.Fatalf("InsertProcessingHistory() error = %v", err)
		}
	}

	// Query with zero limit (should use default of 10)
	records, err := repo.QueryRecentHistory(ctx, 0)
	if err != nil {
		t.Fatalf("QueryRecentHistory() error = %v", err)
	}

	if len(records) != 10 {
		t.Errorf("QueryRecentHistory(0) returned %d records, want 10 (default)", len(records))
	}

	// Query with negative limit (should use default of 10)
	records, err = repo.QueryRecentHistory(ctx, -5)
	if err != nil {
		t.Fatalf("QueryRecentHistory() error = %v", err)
	}

	if len(records) != 10 {
		t.Errorf("QueryRecentHistory(-5) returned %d records, want 10 (default)", len(records))
	}
}

// TestCountMethods tests the count helper methods.
func TestCountMethods(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Initial counts should be zero
	historyCount, err := repo.CountProcessingHistory(ctx)
	if err != nil {
		t.Fatalf("CountProcessingHistory() error = %v", err)
	}
	if historyCount != 0 {
		t.Errorf("Initial processing history count = %d, want 0", historyCount)
	}

	eventCount, err := repo.CountCanvasEvents(ctx)
	if err != nil {
		t.Fatalf("CountCanvasEvents() error = %v", err)
	}
	if eventCount != 0 {
		t.Errorf("Initial canvas events count = %d, want 0", eventCount)
	}

	errorCount, err := repo.CountErrorLogs(ctx)
	if err != nil {
		t.Fatalf("CountErrorLogs() error = %v", err)
	}
	if errorCount != 0 {
		t.Errorf("Initial error logs count = %d, want 0", errorCount)
	}

	// Insert some records
	_, _ = repo.InsertProcessingHistory(ctx, ProcessingRecord{
		CorrelationID: "count-test",
		CanvasID:      "canvas",
		WidgetID:      "widget",
		OperationType: "test",
		Status:        "success",
	})
	_, _ = repo.InsertProcessingHistory(ctx, ProcessingRecord{
		CorrelationID: "count-test-2",
		CanvasID:      "canvas",
		WidgetID:      "widget",
		OperationType: "test",
		Status:        "success",
	})

	_, _ = repo.InsertCanvasEvent(ctx, CanvasEvent{
		CanvasID:   "canvas",
		WidgetID:   "widget",
		EventType:  "created",
		WidgetType: "note",
	})

	// Verify counts
	historyCount, _ = repo.CountProcessingHistory(ctx)
	if historyCount != 2 {
		t.Errorf("Processing history count = %d, want 2", historyCount)
	}

	eventCount, _ = repo.CountCanvasEvents(ctx)
	if eventCount != 1 {
		t.Errorf("Canvas events count = %d, want 1", eventCount)
	}
}
