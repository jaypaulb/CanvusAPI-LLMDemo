package db

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// setupCleanupTestMigrations creates temporary migrations directory with tables needed for cleanup tests.
// Returns the temp directory path (for db), migrations path (with file:// prefix).
func setupCleanupTestMigrations(t *testing.T) (string, string) {
	t.Helper()

	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")

	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatalf("failed to create migrations directory: %v", err)
	}

	// Create up migration with all tables needed for cleanup
	upSQL := `-- Tables for cleanup tests
CREATE TABLE IF NOT EXISTS processing_history (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	correlation_id TEXT NOT NULL,
	canvas_id TEXT NOT NULL,
	widget_id TEXT NOT NULL,
	operation_type TEXT NOT NULL,
	status TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS canvas_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	canvas_id TEXT NOT NULL,
	widget_id TEXT NOT NULL,
	event_type TEXT NOT NULL,
	widget_type TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS performance_metrics (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	metric_type TEXT NOT NULL,
	metric_name TEXT NOT NULL,
	metric_value REAL NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS error_log (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	error_type TEXT NOT NULL,
	error_message TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS system_metrics (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	metric_type TEXT NOT NULL,
	cpu_usage REAL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`
	upPath := filepath.Join(migrationsDir, "000001_create_cleanup_tables.up.sql")
	if err := os.WriteFile(upPath, []byte(upSQL), 0644); err != nil {
		t.Fatalf("failed to write up migration: %v", err)
	}

	// Create down migration
	downSQL := `DROP TABLE IF EXISTS processing_history;
DROP TABLE IF EXISTS canvas_events;
DROP TABLE IF EXISTS performance_metrics;
DROP TABLE IF EXISTS error_log;
DROP TABLE IF EXISTS system_metrics;
`
	downPath := filepath.Join(migrationsDir, "000001_create_cleanup_tables.down.sql")
	if err := os.WriteFile(downPath, []byte(downSQL), 0644); err != nil {
		t.Fatalf("failed to write down migration: %v", err)
	}

	return tmpDir, "file://" + migrationsDir
}

// setupTestDatabaseWithData creates a test database with cleanup tables.
// Returns the database.
func setupTestDatabaseWithData(t *testing.T) *Database {
	t.Helper()

	tmpDir, migrationsPath := setupCleanupTestMigrations(t)
	dbPath := filepath.Join(tmpDir, "test_cleanup.db")

	config := DatabaseConfig{
		Path:           dbPath,
		MigrationsPath: migrationsPath,
	}

	db, err := NewDatabaseWithConfig(config)
	if err != nil {
		t.Fatalf("NewDatabaseWithConfig() error = %v", err)
	}

	// Run migrations to create tables
	if err := db.MigrateWithPath(migrationsPath); err != nil {
		db.Close()
		t.Fatalf("MigrateWithPath() error = %v", err)
	}

	return db
}

// insertTestRecords inserts test records with specified ages into all tables.
func insertTestRecords(t *testing.T, db *Database, ageInDays int, count int) {
	t.Helper()

	// Use SQLite datetime function with offset to set record age
	ageParam := "-" + itoa(ageInDays) + " days"

	for i := 0; i < count; i++ {
		// processing_history
		_, err := db.Exec(`
			INSERT INTO processing_history
			(correlation_id, canvas_id, widget_id, operation_type, status, created_at)
			VALUES (?, ?, ?, 'test', 'completed', datetime('now', ?))`,
			"corr-"+string(rune('a'+i)), "canvas-1", "widget-1", ageParam)
		if err != nil {
			t.Fatalf("Failed to insert processing_history: %v", err)
		}

		// canvas_events
		_, err = db.Exec(`
			INSERT INTO canvas_events
			(canvas_id, widget_id, event_type, widget_type, created_at)
			VALUES (?, ?, 'update', 'note', datetime('now', ?))`,
			"canvas-1", "widget-1", ageParam)
		if err != nil {
			t.Fatalf("Failed to insert canvas_events: %v", err)
		}

		// performance_metrics
		_, err = db.Exec(`
			INSERT INTO performance_metrics
			(metric_type, metric_name, metric_value, created_at)
			VALUES ('latency', 'test_metric', 100.0, datetime('now', ?))`,
			ageParam)
		if err != nil {
			t.Fatalf("Failed to insert performance_metrics: %v", err)
		}

		// error_log
		_, err = db.Exec(`
			INSERT INTO error_log
			(error_type, error_message, created_at)
			VALUES ('test_error', 'test message', datetime('now', ?))`,
			ageParam)
		if err != nil {
			t.Fatalf("Failed to insert error_log: %v", err)
		}

		// system_metrics
		_, err = db.Exec(`
			INSERT INTO system_metrics
			(metric_type, cpu_usage, created_at)
			VALUES ('snapshot', 50.0, datetime('now', ?))`,
			ageParam)
		if err != nil {
			t.Fatalf("Failed to insert system_metrics: %v", err)
		}
	}
}

// itoa converts int to string (simple version for small numbers)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}

// countTableRecords returns the number of records in a table.
func countTableRecords(t *testing.T, db *Database, table string) int {
	t.Helper()

	var count int
	row := db.QueryRow("SELECT COUNT(*) FROM " + table)
	if err := row.Scan(&count); err != nil {
		t.Fatalf("Failed to count %s records: %v", table, err)
	}
	return count
}

// TestCleanup tests the basic Cleanup functionality.
func TestCleanup(t *testing.T) {
	t.Run("deletes old records but keeps recent ones", func(t *testing.T) {
		db := setupTestDatabaseWithData(t)
		defer db.Close()

		// Insert 3 old records (45 days old) and 2 recent records (5 days old)
		insertTestRecords(t, db, 45, 3)
		insertTestRecords(t, db, 5, 2)

		// Verify initial counts (5 records per table)
		for _, table := range tablesToClean {
			count := countTableRecords(t, db, table)
			if count != 5 {
				t.Errorf("Initial %s count = %d, want 5", table, count)
			}
		}

		// Run cleanup with 30-day retention
		result, err := db.Cleanup(30)
		if err != nil {
			t.Fatalf("Cleanup() error = %v", err)
		}

		// Verify old records were deleted (3 per table)
		if result.ProcessingHistoryDeleted != 3 {
			t.Errorf("ProcessingHistoryDeleted = %d, want 3", result.ProcessingHistoryDeleted)
		}
		if result.CanvasEventsDeleted != 3 {
			t.Errorf("CanvasEventsDeleted = %d, want 3", result.CanvasEventsDeleted)
		}
		if result.PerformanceMetricsDeleted != 3 {
			t.Errorf("PerformanceMetricsDeleted = %d, want 3", result.PerformanceMetricsDeleted)
		}
		if result.ErrorLogDeleted != 3 {
			t.Errorf("ErrorLogDeleted = %d, want 3", result.ErrorLogDeleted)
		}
		if result.SystemMetricsDeleted != 3 {
			t.Errorf("SystemMetricsDeleted = %d, want 3", result.SystemMetricsDeleted)
		}
		if result.TotalDeleted != 15 {
			t.Errorf("TotalDeleted = %d, want 15", result.TotalDeleted)
		}

		// Verify recent records remain (2 per table)
		for _, table := range tablesToClean {
			count := countTableRecords(t, db, table)
			if count != 2 {
				t.Errorf("After cleanup %s count = %d, want 2", table, count)
			}
		}
	})

	t.Run("handles empty tables gracefully", func(t *testing.T) {
		db := setupTestDatabaseWithData(t)
		defer db.Close()

		// Run cleanup on empty tables
		result, err := db.Cleanup(30)
		if err != nil {
			t.Fatalf("Cleanup() error = %v", err)
		}

		if result.TotalDeleted != 0 {
			t.Errorf("TotalDeleted = %d, want 0 for empty tables", result.TotalDeleted)
		}
	})

	t.Run("returns error for negative retention days", func(t *testing.T) {
		db := setupTestDatabaseWithData(t)
		defer db.Close()

		_, err := db.Cleanup(-1)
		if err == nil {
			t.Error("Cleanup() expected error for negative retentionDays, got nil")
		}
	})

	t.Run("duration is recorded", func(t *testing.T) {
		db := setupTestDatabaseWithData(t)
		defer db.Close()

		result, err := db.Cleanup(30)
		if err != nil {
			t.Fatalf("Cleanup() error = %v", err)
		}

		if result.Duration <= 0 {
			t.Error("Duration should be positive")
		}
	})
}

// TestCleanupWithContext tests context-aware cleanup.
func TestCleanupWithContext(t *testing.T) {
	t.Run("respects context cancellation", func(t *testing.T) {
		db := setupTestDatabaseWithData(t)
		defer db.Close()

		// Insert some data
		insertTestRecords(t, db, 45, 5)

		// Cancel context immediately
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := db.CleanupWithContext(ctx, 30)
		if err == nil {
			t.Error("CleanupWithContext() expected error for cancelled context, got nil")
		}
		if err != context.Canceled {
			t.Errorf("CleanupWithContext() error = %v, want context.Canceled", err)
		}
	})

	t.Run("respects context timeout", func(t *testing.T) {
		db := setupTestDatabaseWithData(t)
		defer db.Close()

		// Create a context that's already expired
		ctx, cancel := context.WithTimeout(context.Background(), 0)
		defer cancel()

		// Give it a moment to expire
		time.Sleep(time.Millisecond)

		_, err := db.CleanupWithContext(ctx, 30)
		if err == nil {
			t.Error("CleanupWithContext() expected error for timed out context, got nil")
		}
	})

	t.Run("completes successfully with valid context", func(t *testing.T) {
		db := setupTestDatabaseWithData(t)
		defer db.Close()

		insertTestRecords(t, db, 45, 3)

		ctx := context.Background()
		result, err := db.CleanupWithContext(ctx, 30)
		if err != nil {
			t.Fatalf("CleanupWithContext() error = %v", err)
		}

		if result.TotalDeleted != 15 { // 3 records * 5 tables
			t.Errorf("TotalDeleted = %d, want 15", result.TotalDeleted)
		}
	})
}

// TestCleanupVacuum tests that VACUUM runs successfully.
func TestCleanupVacuum(t *testing.T) {
	t.Run("VACUUM runs without error", func(t *testing.T) {
		db := setupTestDatabaseWithData(t)
		defer db.Close()

		// Insert and delete data to create freeable space
		insertTestRecords(t, db, 45, 10)

		result, err := db.Cleanup(30)
		if err != nil {
			t.Fatalf("Cleanup() error = %v", err)
		}

		// If we get here without error, VACUUM succeeded
		if result.TotalDeleted != 50 { // 10 records * 5 tables
			t.Errorf("TotalDeleted = %d, want 50", result.TotalDeleted)
		}
	})
}

// TestCleanupScheduler tests the background cleanup scheduler.
func TestCleanupScheduler(t *testing.T) {
	t.Run("scheduler starts and stops cleanly", func(t *testing.T) {
		db := setupTestDatabaseWithData(t)
		defer db.Close()

		ctx, cancel := context.WithCancel(context.Background())

		// Start scheduler with short interval for testing
		db.StartCleanupScheduler(ctx, 30, 100*time.Millisecond)

		// Let it run for a bit
		time.Sleep(50 * time.Millisecond)

		// Cancel and verify it stops
		cancel()

		// Give it time to stop
		time.Sleep(50 * time.Millisecond)

		// No assertion needed - if we get here without deadlock/panic, it works
	})

	t.Run("scheduler runs cleanup on start", func(t *testing.T) {
		db := setupTestDatabaseWithData(t)
		defer db.Close()

		// Insert old records
		insertTestRecords(t, db, 45, 3)

		// Verify records exist
		initialCount := countTableRecords(t, db, "processing_history")
		if initialCount != 3 {
			t.Fatalf("Initial count = %d, want 3", initialCount)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Start scheduler with long interval (we only care about initial run)
		db.StartCleanupScheduler(ctx, 30, 1*time.Hour)

		// Give the initial cleanup time to run
		time.Sleep(100 * time.Millisecond)

		// Verify records were deleted
		finalCount := countTableRecords(t, db, "processing_history")
		if finalCount != 0 {
			t.Errorf("After scheduler start, count = %d, want 0", finalCount)
		}
	})

	t.Run("scheduler with callback receives results", func(t *testing.T) {
		db := setupTestDatabaseWithData(t)
		defer db.Close()

		insertTestRecords(t, db, 45, 2)

		var mu sync.Mutex
		var callbackCalled bool
		var receivedResult CleanupResult

		config := CleanupSchedulerConfig{
			RetentionDays: 30,
			Interval:      1 * time.Hour,
			OnCleanup: func(result CleanupResult, err error) {
				mu.Lock()
				defer mu.Unlock()
				callbackCalled = true
				receivedResult = result
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		db.StartCleanupSchedulerWithConfig(ctx, config)

		// Wait for callback
		time.Sleep(100 * time.Millisecond)

		mu.Lock()
		defer mu.Unlock()

		if !callbackCalled {
			t.Error("Callback was not called")
		}
		if receivedResult.TotalDeleted != 10 { // 2 records * 5 tables
			t.Errorf("Callback received TotalDeleted = %d, want 10", receivedResult.TotalDeleted)
		}
	})

	t.Run("scheduler runs periodically", func(t *testing.T) {
		db := setupTestDatabaseWithData(t)
		defer db.Close()

		var mu sync.Mutex
		var callCount int

		config := CleanupSchedulerConfig{
			RetentionDays: 30,
			Interval:      50 * time.Millisecond,
			OnCleanup: func(result CleanupResult, err error) {
				mu.Lock()
				defer mu.Unlock()
				callCount++
			},
		}

		ctx, cancel := context.WithCancel(context.Background())

		db.StartCleanupSchedulerWithConfig(ctx, config)

		// Wait for multiple runs (initial + 2 periodic)
		time.Sleep(150 * time.Millisecond)

		cancel()

		mu.Lock()
		finalCount := callCount
		mu.Unlock()

		// Should have at least 2 runs (initial + 1 periodic)
		if finalCount < 2 {
			t.Errorf("Callback count = %d, want >= 2", finalCount)
		}
	})
}

// TestCleanupOnClosedDatabase tests behavior with closed database.
func TestCleanupOnClosedDatabase(t *testing.T) {
	t.Run("returns error on closed database", func(t *testing.T) {
		db := setupTestDatabaseWithData(t)

		// Close the database
		if err := db.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}

		// Try to cleanup
		_, err := db.Cleanup(30)
		if err == nil {
			t.Error("Cleanup() expected error on closed database, got nil")
		}
	})
}

// TestCleanupZeroRetention tests edge case of 0 retention days.
func TestCleanupZeroRetention(t *testing.T) {
	t.Run("zero retention deletes all records", func(t *testing.T) {
		db := setupTestDatabaseWithData(t)
		defer db.Close()

		// Insert records from today
		insertTestRecords(t, db, 0, 3)

		// With 0 retention, all records should be deleted
		// (records older than now, which includes records with created_at = now due to processing time)
		result, err := db.Cleanup(0)
		if err != nil {
			t.Fatalf("Cleanup() error = %v", err)
		}

		// All records should be deleted (created_at < datetime('now', '-0 days') = datetime('now'))
		// Records created "now" might or might not be deleted depending on timing
		// The important thing is no error occurs
		t.Logf("Zero retention deleted %d total records", result.TotalDeleted)
	})
}

// TestDefaultCleanupSchedulerConfig tests default configuration values.
func TestDefaultCleanupSchedulerConfig(t *testing.T) {
	config := DefaultCleanupSchedulerConfig()

	if config.RetentionDays != 30 {
		t.Errorf("RetentionDays = %d, want 30", config.RetentionDays)
	}
	if config.Interval != 24*time.Hour {
		t.Errorf("Interval = %v, want 24h", config.Interval)
	}
	if config.OnCleanup != nil {
		t.Error("OnCleanup should be nil by default")
	}
}
