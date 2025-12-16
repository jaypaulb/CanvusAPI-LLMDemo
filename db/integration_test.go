package db

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestDatabaseOrganismIntegration tests the full database organism working together.
// This is an organism-level integration test covering:
// - Database lifecycle (create, migrate, close)
// - Repository CRUD operations
// - Cleanup/retention policies
// - Async write throughput
// - End-to-end data flow
func TestDatabaseOrganismIntegration(t *testing.T) {
	t.Run("full lifecycle with migrations and CRUD", func(t *testing.T) {
		// Setup: Create database with migrations
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "integration_test.db")

		// Create migrations directory
		migrationsDir := filepath.Join(tmpDir, "migrations")
		if err := os.MkdirAll(migrationsDir, 0755); err != nil {
			t.Fatalf("Failed to create migrations dir: %v", err)
		}

		// Write migration files (same as production schema)
		upSQL := testSchemaUp // Reuse from repository_test.go
		upPath := filepath.Join(migrationsDir, "000001_initial.up.sql")
		if err := os.WriteFile(upPath, []byte(upSQL), 0644); err != nil {
			t.Fatalf("Failed to write up migration: %v", err)
		}

		downSQL := testSchemaDown
		downPath := filepath.Join(migrationsDir, "000001_initial.down.sql")
		if err := os.WriteFile(downPath, []byte(downSQL), 0644); err != nil {
			t.Fatalf("Failed to write down migration: %v", err)
		}

		// Create database organism
		config := DatabaseConfig{
			Path:           dbPath,
			MigrationsPath: "file://" + migrationsDir,
		}
		db, err := NewDatabaseWithConfig(config)
		if err != nil {
			t.Fatalf("NewDatabaseWithConfig() error = %v", err)
		}
		defer db.Close()

		// Verify database is healthy
		if err := db.Ping(); err != nil {
			t.Fatalf("Ping() after creation error = %v", err)
		}

		// Run migrations
		if err := db.Migrate(); err != nil {
			t.Fatalf("Migrate() error = %v", err)
		}

		// Verify WAL mode and foreign keys are enabled
		var walMode string
		if err := db.DB().QueryRow("PRAGMA journal_mode").Scan(&walMode); err != nil {
			t.Fatalf("Failed to check journal_mode: %v", err)
		}
		if walMode != "wal" {
			t.Errorf("journal_mode = %v, want 'wal'", walMode)
		}

		var foreignKeys int
		if err := db.DB().QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
			t.Fatalf("Failed to check foreign_keys: %v", err)
		}
		if foreignKeys != 1 {
			t.Errorf("foreign_keys = %v, want 1", foreignKeys)
		}

		// Create repository for CRUD operations
		repo := NewRepository(db, nil)
		ctx := context.Background()

		// Test CRUD: Insert processing history
		historyRecord := ProcessingRecord{
			CorrelationID: "integration-test-001",
			CanvasID:      "canvas-integration",
			WidgetID:      "widget-integration",
			OperationType: "text_generation",
			Prompt:        "Test prompt for integration",
			Response:      "Test response from integration",
			ModelName:     "llama-3-8b",
			InputTokens:   15,
			OutputTokens:  20,
			DurationMS:    250,
			Status:        "success",
		}
		historyID, err := repo.InsertProcessingHistory(ctx, historyRecord)
		if err != nil {
			t.Fatalf("InsertProcessingHistory() error = %v", err)
		}
		if historyID <= 0 {
			t.Errorf("InsertProcessingHistory() returned invalid ID = %d", historyID)
		}

		// Test CRUD: Insert canvas event
		canvasEvent := CanvasEvent{
			CanvasID:       "canvas-integration",
			WidgetID:       "widget-integration",
			EventType:      "created",
			WidgetType:     "note",
			ContentPreview: "Integration test note preview",
		}
		eventID, err := repo.InsertCanvasEvent(ctx, canvasEvent)
		if err != nil {
			t.Fatalf("InsertCanvasEvent() error = %v", err)
		}
		if eventID <= 0 {
			t.Errorf("InsertCanvasEvent() returned invalid ID = %d", eventID)
		}

		// Test CRUD: Query data back
		historyRecords, err := repo.QueryRecentHistory(ctx, 10)
		if err != nil {
			t.Fatalf("QueryRecentHistory() error = %v", err)
		}
		if len(historyRecords) != 1 {
			t.Fatalf("QueryRecentHistory() returned %d records, want 1", len(historyRecords))
		}
		if historyRecords[0].CorrelationID != historyRecord.CorrelationID {
			t.Errorf("Retrieved CorrelationID = %v, want %v", historyRecords[0].CorrelationID, historyRecord.CorrelationID)
		}

		events, err := repo.QueryRecentCanvasEvents(ctx, 10)
		if err != nil {
			t.Fatalf("QueryRecentCanvasEvents() error = %v", err)
		}
		if len(events) != 1 {
			t.Fatalf("QueryRecentCanvasEvents() returned %d events, want 1", len(events))
		}

		// Verify database stats show activity
		stats := db.Stats()
		if stats.OpenConnections <= 0 {
			t.Error("Expected at least one open connection")
		}

		// Test graceful shutdown
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}

		// Verify database file was created
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Error("Database file should exist after operations")
		}
	})
}

// TestAsyncWriteThroughput tests the database organism with async writes
// under concurrent load to verify throughput.
func TestAsyncWriteThroughput(t *testing.T) {
	// Setup database with async writer
	tmpDir, migrationsPath := setupTestMigrationsForRepo(t)
	dbPath := filepath.Join(tmpDir, "async_throughput.db")

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

	// Create repository without async writer first
	repo := NewRepository(db, nil)

	// Create and start async writer
	asyncWriter := NewAsyncWriter(repo.CreateAsyncWriteHandler())
	asyncWriter.Start()
	defer asyncWriter.Close()

	// Attach async writer to repository
	repo.asyncWriter = asyncWriter

	ctx := context.Background()

	// Test: High-throughput concurrent writes
	const numGoroutines = 20
	const writesPerGoroutine = 50
	const totalExpected = numGoroutines * writesPerGoroutine

	var wg sync.WaitGroup
	errChan := make(chan error, totalExpected)

	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < writesPerGoroutine; j++ {
				record := ProcessingRecord{
					CorrelationID: "throughput-test",
					CanvasID:      "canvas-throughput",
					WidgetID:      "widget-throughput",
					OperationType: "text_generation",
					Status:        "success",
				}
				_, err := repo.InsertProcessingHistory(ctx, record)
				if err != nil {
					errChan <- err
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)
	close(errChan)

	// Check for errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}
	if len(errors) > 0 {
		t.Fatalf("Async writes produced %d errors: %v", len(errors), errors[0])
	}

	// Stop async writer and wait for drain
	asyncWriter.Stop()

	// Verify all writes completed
	count, err := repo.CountProcessingHistory(ctx)
	if err != nil {
		t.Fatalf("CountProcessingHistory() error = %v", err)
	}

	if count != totalExpected {
		t.Errorf("Processing history count = %d, want %d", count, totalExpected)
	}

	// Log throughput metrics
	throughput := float64(totalExpected) / elapsed.Seconds()
	t.Logf("Async write throughput: %.2f writes/sec (%d writes in %v)", throughput, totalExpected, elapsed)

	// Sanity check: throughput should be reasonable (at least 100 writes/sec)
	if throughput < 100 {
		t.Logf("Warning: Low throughput (%.2f writes/sec), expected > 100 writes/sec", throughput)
	}
}

// TestCleanupRetentionPolicy tests the cleanup organism with retention policies.
func TestCleanupRetentionPolicy(t *testing.T) {
	// Setup database with cleanup tables
	tmpDir, migrationsPath := setupCleanupTestMigrations(t)
	dbPath := filepath.Join(tmpDir, "cleanup_retention.db")

	config := DatabaseConfig{
		Path:           dbPath,
		MigrationsPath: migrationsPath,
	}

	db, err := NewDatabaseWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if err := db.MigrateWithPath(migrationsPath); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Insert test data with different ages
	// Old data: 60 days old (should be deleted with 30-day retention)
	insertTestRecords(t, db, 60, 5)
	// Middle data: 20 days old (should be kept)
	insertTestRecords(t, db, 20, 3)
	// Recent data: 5 days old (should be kept)
	insertTestRecords(t, db, 5, 2)

	// Verify initial counts
	initialCount := countTableRecords(t, db, "processing_history")
	if initialCount != 10 {
		t.Fatalf("Initial processing_history count = %d, want 10", initialCount)
	}

	// Run cleanup with 30-day retention
	result, err := db.Cleanup(30)
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	// Verify cleanup results
	if result.ProcessingHistoryDeleted != 5 {
		t.Errorf("ProcessingHistoryDeleted = %d, want 5", result.ProcessingHistoryDeleted)
	}
	if result.TotalDeleted != 25 { // 5 records * 5 tables
		t.Errorf("TotalDeleted = %d, want 25", result.TotalDeleted)
	}
	if result.Duration <= 0 {
		t.Error("Cleanup duration should be positive")
	}

	// Verify remaining data
	finalCount := countTableRecords(t, db, "processing_history")
	if finalCount != 5 { // 3 middle + 2 recent
		t.Errorf("Final processing_history count = %d, want 5", finalCount)
	}

	// Run cleanup again - should delete nothing
	result2, err := db.Cleanup(30)
	if err != nil {
		t.Fatalf("Second Cleanup() error = %v", err)
	}
	if result2.TotalDeleted != 0 {
		t.Errorf("Second cleanup TotalDeleted = %d, want 0", result2.TotalDeleted)
	}

	// Test VACUUM was executed (no way to directly verify, but it shouldn't error)
	t.Logf("Cleanup successfully executed VACUUM in %v", result.Duration)
}

// TestMigrationIdempotency tests that migrations can be run multiple times safely.
func TestMigrationIdempotency(t *testing.T) {
	tmpDir, migrationsPath := setupTestMigrationsForRepo(t)
	dbPath := filepath.Join(tmpDir, "migration_test.db")

	config := DatabaseConfig{
		Path:           dbPath,
		MigrationsPath: migrationsPath,
	}

	db, err := NewDatabaseWithConfig(config)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Run migrations first time
	if err := db.Migrate(); err != nil {
		t.Fatalf("First Migrate() error = %v", err)
	}

	// Insert test data
	repo := NewRepository(db, nil)
	ctx := context.Background()
	_, err = repo.InsertProcessingHistory(ctx, ProcessingRecord{
		CorrelationID: "migration-test",
		CanvasID:      "canvas-1",
		WidgetID:      "widget-1",
		OperationType: "test",
		Status:        "success",
	})
	if err != nil {
		t.Fatalf("InsertProcessingHistory() error = %v", err)
	}

	// Run migrations again - should be no-op
	if err := db.Migrate(); err != nil {
		t.Fatalf("Second Migrate() error = %v", err)
	}

	// Verify data still exists
	count, err := repo.CountProcessingHistory(ctx)
	if err != nil {
		t.Fatalf("CountProcessingHistory() error = %v", err)
	}
	if count != 1 {
		t.Errorf("After second migration, count = %d, want 1 (data should be preserved)", count)
	}

	// Run migrations third time with explicit path
	if err := db.MigrateWithPath(migrationsPath); err != nil {
		t.Fatalf("Third Migrate() error = %v", err)
	}

	// Verify data still exists
	count, err = repo.CountProcessingHistory(ctx)
	if err != nil {
		t.Fatalf("CountProcessingHistory() error = %v", err)
	}
	if count != 1 {
		t.Errorf("After third migration, count = %d, want 1 (data should be preserved)", count)
	}
}

// TestRepositoryCRUDComprehensive tests all CRUD operations across all tables.
func TestRepositoryCRUDComprehensive(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Test: Insert and query all table types
	t.Run("all table types CRUD", func(t *testing.T) {
		// Processing history
		histRecord := ProcessingRecord{
			CorrelationID: "crud-test-001",
			CanvasID:      "canvas-crud",
			WidgetID:      "widget-crud",
			OperationType: "image_generation",
			Prompt:        "Generate a sunset",
			Response:      "Image generated successfully",
			ModelName:     "stable-diffusion",
			InputTokens:   8,
			OutputTokens:  0,
			DurationMS:    3500,
			Status:        "success",
		}
		histID, err := repo.InsertProcessingHistory(ctx, histRecord)
		if err != nil {
			t.Fatalf("InsertProcessingHistory() error = %v", err)
		}
		if histID <= 0 {
			t.Errorf("InsertProcessingHistory() returned invalid ID")
		}

		// Canvas event
		canvasEvt := CanvasEvent{
			CanvasID:       "canvas-crud",
			WidgetID:       "widget-crud",
			EventType:      "updated",
			WidgetType:     "image",
			ContentPreview: "Sunset image preview",
		}
		evtID, err := repo.InsertCanvasEvent(ctx, canvasEvt)
		if err != nil {
			t.Fatalf("InsertCanvasEvent() error = %v", err)
		}
		if evtID <= 0 {
			t.Errorf("InsertCanvasEvent() returned invalid ID")
		}

		// Error log
		errLog := ErrorLogEntry{
			CorrelationID: "crud-test-001",
			ErrorType:     "warning",
			ErrorMessage:  "Low memory warning",
			StackTrace:    "stacktrace here",
			Context:       `{"memory_mb": 512}`,
		}
		errID, err := repo.InsertErrorLog(ctx, errLog)
		if err != nil {
			t.Fatalf("InsertErrorLog() error = %v", err)
		}
		if errID <= 0 {
			t.Errorf("InsertErrorLog() returned invalid ID")
		}

		// Performance metric
		perfMetric := PerformanceMetric{
			MetricType:  "inference",
			MetricName:  "latency_ms",
			MetricValue: 3500.0,
			Metadata:    `{"model": "stable-diffusion"}`,
		}
		perfID, err := repo.InsertPerformanceMetric(ctx, perfMetric)
		if err != nil {
			t.Fatalf("InsertPerformanceMetric() error = %v", err)
		}
		if perfID <= 0 {
			t.Errorf("InsertPerformanceMetric() returned invalid ID")
		}

		// System metric
		sysMetric := SystemMetric{
			MetricType:    "snapshot",
			CPUUsage:      75.5,
			MemoryUsedMB:  8192.0,
			MemoryTotalMB: 16384.0,
			DiskUsedMB:    204800.0,
			DiskTotalMB:   1024000.0,
		}
		sysID, err := repo.InsertSystemMetric(ctx, sysMetric)
		if err != nil {
			t.Fatalf("InsertSystemMetric() error = %v", err)
		}
		if sysID <= 0 {
			t.Errorf("InsertSystemMetric() returned invalid ID")
		}

		// Query all data back
		histRecords, err := repo.QueryRecentHistory(ctx, 10)
		if err != nil {
			t.Fatalf("QueryRecentHistory() error = %v", err)
		}
		if len(histRecords) != 1 {
			t.Errorf("QueryRecentHistory() returned %d records, want 1", len(histRecords))
		}

		events, err := repo.QueryRecentCanvasEvents(ctx, 10)
		if err != nil {
			t.Fatalf("QueryRecentCanvasEvents() error = %v", err)
		}
		if len(events) != 1 {
			t.Errorf("QueryRecentCanvasEvents() returned %d events, want 1", len(events))
		}

		errorLogs, err := repo.QueryRecentErrorLogs(ctx, 10)
		if err != nil {
			t.Fatalf("QueryRecentErrorLogs() error = %v", err)
		}
		if len(errorLogs) != 1 {
			t.Errorf("QueryRecentErrorLogs() returned %d logs, want 1", len(errorLogs))
		}

		// Verify correlation ID queries work
		corrRecords, err := repo.QueryHistoryByCorrelationID(ctx, "crud-test-001")
		if err != nil {
			t.Fatalf("QueryHistoryByCorrelationID() error = %v", err)
		}
		if len(corrRecords) != 1 {
			t.Errorf("QueryHistoryByCorrelationID() returned %d records, want 1", len(corrRecords))
		}

		// Verify count methods
		histCount, _ := repo.CountProcessingHistory(ctx)
		if histCount != 1 {
			t.Errorf("CountProcessingHistory() = %d, want 1", histCount)
		}

		evtCount, _ := repo.CountCanvasEvents(ctx)
		if evtCount != 1 {
			t.Errorf("CountCanvasEvents() = %d, want 1", evtCount)
		}

		errCount, _ := repo.CountErrorLogs(ctx)
		if errCount != 1 {
			t.Errorf("CountErrorLogs() = %d, want 1", errCount)
		}
	})

	// Test: Multiple inserts and query limits
	t.Run("query limits and ordering", func(t *testing.T) {
		// Insert 20 more records
		for i := 0; i < 20; i++ {
			_, _ = repo.InsertProcessingHistory(ctx, ProcessingRecord{
				CorrelationID: "limit-test",
				CanvasID:      "canvas",
				WidgetID:      "widget",
				OperationType: "test",
				Status:        "success",
			})
		}

		// Query with limit
		records, err := repo.QueryRecentHistory(ctx, 5)
		if err != nil {
			t.Fatalf("QueryRecentHistory(5) error = %v", err)
		}
		if len(records) != 5 {
			t.Errorf("QueryRecentHistory(5) returned %d records, want 5", len(records))
		}

		// Verify ordering (most recent first)
		if len(records) >= 2 {
			first := records[0].CreatedAt
			second := records[1].CreatedAt
			if !first.After(second) && !first.Equal(second) {
				t.Error("Records should be ordered by created_at DESC (most recent first)")
			}
		}
	})
}

// TestDatabaseTransactionRollback tests transaction behavior on errors.
func TestDatabaseTransactionRollback(t *testing.T) {
	repo, db, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx := context.Background()

	// Insert initial record
	_, err := repo.InsertProcessingHistory(ctx, ProcessingRecord{
		CorrelationID: "txn-test-001",
		CanvasID:      "canvas",
		WidgetID:      "widget",
		OperationType: "test",
		Status:        "success",
	})
	if err != nil {
		t.Fatalf("Initial insert error = %v", err)
	}

	initialCount, _ := repo.CountProcessingHistory(ctx)

	// Start a transaction
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	// Insert in transaction
	_, err = tx.Exec(`
		INSERT INTO processing_history
		(correlation_id, canvas_id, widget_id, operation_type, status)
		VALUES (?, ?, ?, ?, ?)`,
		"txn-test-002", "canvas", "widget", "test", "success")
	if err != nil {
		t.Fatalf("Transaction insert error = %v", err)
	}

	// Rollback
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	// Verify count didn't change
	finalCount, _ := repo.CountProcessingHistory(ctx)
	if finalCount != initialCount {
		t.Errorf("After rollback, count = %d, want %d (rollback should undo insert)", finalCount, initialCount)
	}

	// Now test commit
	tx2, err := db.Begin()
	if err != nil {
		t.Fatalf("Second Begin() error = %v", err)
	}

	_, err = tx2.Exec(`
		INSERT INTO processing_history
		(correlation_id, canvas_id, widget_id, operation_type, status)
		VALUES (?, ?, ?, ?, ?)`,
		"txn-test-003", "canvas", "widget", "test", "success")
	if err != nil {
		t.Fatalf("Second transaction insert error = %v", err)
	}

	// Commit
	if err := tx2.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	// Verify count increased
	finalCount2, _ := repo.CountProcessingHistory(ctx)
	if finalCount2 != initialCount+1 {
		t.Errorf("After commit, count = %d, want %d", finalCount2, initialCount+1)
	}
}

// TestCleanupSchedulerIntegration tests the cleanup scheduler in a realistic scenario.
func TestCleanupSchedulerIntegration(t *testing.T) {
	db := setupTestDatabaseWithData(t)
	defer db.Close()

	// Insert data that will be cleaned up
	insertTestRecords(t, db, 60, 5) // Old data
	insertTestRecords(t, db, 10, 3) // Recent data

	initialCount := countTableRecords(t, db, "processing_history")
	if initialCount != 8 {
		t.Fatalf("Initial count = %d, want 8", initialCount)
	}

	// Track cleanup results
	var mu sync.Mutex
	var cleanupResults []CleanupResult
	var cleanupErrors []error

	config := CleanupSchedulerConfig{
		RetentionDays: 30,
		Interval:      100 * time.Millisecond,
		OnCleanup: func(result CleanupResult, err error) {
			mu.Lock()
			defer mu.Unlock()
			cleanupResults = append(cleanupResults, result)
			if err != nil {
				cleanupErrors = append(cleanupErrors, err)
			}
		},
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Start scheduler
	db.StartCleanupSchedulerWithConfig(ctx, config)

	// Let it run for at least 2 cleanup cycles
	time.Sleep(250 * time.Millisecond)

	// Stop scheduler
	cancel()
	time.Sleep(50 * time.Millisecond) // Let it clean up

	mu.Lock()
	resultsCount := len(cleanupResults)
	errorsCount := len(cleanupErrors)
	firstResult := CleanupResult{}
	if len(cleanupResults) > 0 {
		firstResult = cleanupResults[0]
	}
	mu.Unlock()

	// Verify scheduler ran at least once (initial run)
	if resultsCount < 1 {
		t.Fatalf("Scheduler should have run at least once, got %d runs", resultsCount)
	}

	// Verify no errors
	if errorsCount > 0 {
		t.Errorf("Scheduler produced %d errors: %v", errorsCount, cleanupErrors[0])
	}

	// Verify first cleanup deleted old records
	if firstResult.TotalDeleted != 25 { // 5 old records * 5 tables
		t.Errorf("First cleanup TotalDeleted = %d, want 25", firstResult.TotalDeleted)
	}

	// Verify final state
	finalCount := countTableRecords(t, db, "processing_history")
	if finalCount != 3 {
		t.Errorf("Final count = %d, want 3 (old data should be deleted)", finalCount)
	}

	t.Logf("Cleanup scheduler ran %d times successfully", resultsCount)
}
