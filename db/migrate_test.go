package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// setupTestMigrations creates a temporary migrations directory with test migration files.
// Returns the temp directory path (for db), migrations path (with file:// prefix), and cleanup function.
func setupTestMigrations(t *testing.T) (string, string, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	migrationsDir := filepath.Join(tmpDir, "migrations")

	if err := os.MkdirAll(migrationsDir, 0755); err != nil {
		t.Fatalf("failed to create migrations directory: %v", err)
	}

	// Create up migration
	upSQL := `CREATE TABLE IF NOT EXISTS test_table (
		id INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	upPath := filepath.Join(migrationsDir, "000001_create_test_table.up.sql")
	if err := os.WriteFile(upPath, []byte(upSQL), 0644); err != nil {
		t.Fatalf("failed to write up migration: %v", err)
	}

	// Create down migration
	downSQL := `DROP TABLE IF EXISTS test_table;`
	downPath := filepath.Join(migrationsDir, "000001_create_test_table.down.sql")
	if err := os.WriteFile(downPath, []byte(downSQL), 0644); err != nil {
		t.Fatalf("failed to write down migration: %v", err)
	}

	return tmpDir, "file://" + migrationsDir, func() {
		// Cleanup handled by t.TempDir()
	}
}

// TestDefaultMigrationConfig verifies default configuration values.
func TestDefaultMigrationConfig(t *testing.T) {
	path := "file://db/migrations"
	config := DefaultMigrationConfig(path)

	if config.MigrationsPath != path {
		t.Errorf("MigrationsPath = %q, want %q", config.MigrationsPath, path)
	}
	if config.DatabaseName != "main" {
		t.Errorf("DatabaseName = %q, want %q", config.DatabaseName, "main")
	}
}

// TestMigrateUpFromPath_AppliesMigrations verifies that migrations are applied.
func TestMigrateUpFromPath_AppliesMigrations(t *testing.T) {
	tmpDir, migrationsPath, cleanup := setupTestMigrations(t)
	defer cleanup()

	dbPath := filepath.Join(tmpDir, "test.db")

	err := MigrateUpFromPath(dbPath, migrationsPath)
	if err != nil {
		t.Fatalf("MigrateUpFromPath() error = %v", err)
	}

	// Open a new connection to verify the table was created
	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		t.Fatalf("failed to open db for verification: %v", err)
	}
	defer db.Close()

	// Verify table was created
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableName)
	if err != nil {
		t.Errorf("test_table was not created: %v", err)
	}
	if tableName != "test_table" {
		t.Errorf("table name = %q, want %q", tableName, "test_table")
	}
}

// TestMigrateUpFromPath_NoChange verifies ErrNoChange is handled gracefully.
func TestMigrateUpFromPath_NoChange(t *testing.T) {
	tmpDir, migrationsPath, cleanup := setupTestMigrations(t)
	defer cleanup()

	dbPath := filepath.Join(tmpDir, "test.db")

	// Apply migrations first time
	err := MigrateUpFromPath(dbPath, migrationsPath)
	if err != nil {
		t.Fatalf("first MigrateUpFromPath() error = %v", err)
	}

	// Apply migrations second time - should return nil (ErrNoChange handled)
	err = MigrateUpFromPath(dbPath, migrationsPath)
	if err != nil {
		t.Errorf("second MigrateUpFromPath() error = %v, want nil (ErrNoChange handled)", err)
	}
}

// TestMigrateDownFromPath_RollsBackMigrations verifies migrations are rolled back.
func TestMigrateDownFromPath_RollsBackMigrations(t *testing.T) {
	tmpDir, migrationsPath, cleanup := setupTestMigrations(t)
	defer cleanup()

	dbPath := filepath.Join(tmpDir, "test.db")

	// Apply migrations first
	err := MigrateUpFromPath(dbPath, migrationsPath)
	if err != nil {
		t.Fatalf("MigrateUpFromPath() error = %v", err)
	}

	// Verify table exists using separate connection
	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&count)
	db.Close()
	if err != nil || count != 1 {
		t.Fatalf("test_table should exist before rollback")
	}

	// Roll back all migrations
	err = MigrateDownFromPath(dbPath, migrationsPath, -1)
	if err != nil {
		t.Fatalf("MigrateDownFromPath() error = %v", err)
	}

	// Verify table was dropped using separate connection
	db, err = NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		t.Fatalf("failed to open db after rollback: %v", err)
	}
	defer db.Close()

	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&count)
	if err != nil {
		t.Fatalf("query error after rollback: %v", err)
	}
	if count != 0 {
		t.Error("test_table should not exist after rollback")
	}
}

// TestMigrateDownFromPath_NoChange verifies ErrNoChange is handled gracefully.
func TestMigrateDownFromPath_NoChange(t *testing.T) {
	tmpDir, migrationsPath, cleanup := setupTestMigrations(t)
	defer cleanup()

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create database file by opening and closing a connection
	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	db.Close()

	// Try to roll back when no migrations have been applied
	err = MigrateDownFromPath(dbPath, migrationsPath, -1)
	if err != nil {
		t.Errorf("MigrateDownFromPath() on empty db error = %v, want nil (ErrNoChange handled)", err)
	}
}

// TestGetMigrationVersionFromPath_InitialState verifies version 0 when no migrations applied.
func TestGetMigrationVersionFromPath_InitialState(t *testing.T) {
	tmpDir, migrationsPath, cleanup := setupTestMigrations(t)
	defer cleanup()

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create database file
	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	db.Close()

	version, dirty, err := GetMigrationVersionFromPath(dbPath, migrationsPath)
	if err != nil {
		t.Fatalf("GetMigrationVersionFromPath() error = %v", err)
	}
	if version != 0 {
		t.Errorf("version = %d, want 0", version)
	}
	if dirty {
		t.Error("dirty = true, want false")
	}
}

// TestGetMigrationVersionFromPath_AfterMigration verifies version tracking.
func TestGetMigrationVersionFromPath_AfterMigration(t *testing.T) {
	tmpDir, migrationsPath, cleanup := setupTestMigrations(t)
	defer cleanup()

	dbPath := filepath.Join(tmpDir, "test.db")

	// Apply migrations
	err := MigrateUpFromPath(dbPath, migrationsPath)
	if err != nil {
		t.Fatalf("MigrateUpFromPath() error = %v", err)
	}

	version, dirty, err := GetMigrationVersionFromPath(dbPath, migrationsPath)
	if err != nil {
		t.Fatalf("GetMigrationVersionFromPath() error = %v", err)
	}
	if version != 1 {
		t.Errorf("version = %d, want 1", version)
	}
	if dirty {
		t.Error("dirty = true, want false")
	}
}

// TestRunMigrationsFromPath_Success verifies the convenience function works.
func TestRunMigrationsFromPath_Success(t *testing.T) {
	tmpDir, migrationsPath, cleanup := setupTestMigrations(t)
	defer cleanup()

	dbPath := filepath.Join(tmpDir, "test.db")

	err := RunMigrationsFromPath(dbPath, migrationsPath)
	if err != nil {
		t.Fatalf("RunMigrationsFromPath() error = %v", err)
	}

	// Open new connection to verify
	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Verify table was created
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableName)
	if err != nil {
		t.Errorf("test_table was not created by RunMigrationsFromPath: %v", err)
	}
}

// TestMigrateUp_NilDB verifies error on nil database.
func TestMigrateUp_NilDB(t *testing.T) {
	err := MigrateUp(nil, "file://db/migrations")
	if err == nil {
		t.Error("MigrateUp(nil, ...) should return error")
	}
}

// TestMigrateUp_EmptyPath verifies error on empty migrations path.
func TestMigrateUp_EmptyPath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	// Note: db will be closed by MigrateUp's newMigrator or on error

	err = MigrateUp(db, "")
	if err == nil {
		t.Error("MigrateUp(db, \"\") should return error")
	}
}

// TestMigrateUpFromPath_InvalidPath verifies error on invalid migrations path.
func TestMigrateUpFromPath_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	err := MigrateUpFromPath(dbPath, "file:///nonexistent/path/migrations")
	if err == nil {
		t.Error("MigrateUpFromPath with invalid path should return error")
	}
}

// TestMigrateToVersionFromPath verifies migrating to specific version.
func TestMigrateToVersionFromPath(t *testing.T) {
	tmpDir, migrationsPath, cleanup := setupTestMigrations(t)
	defer cleanup()

	dbPath := filepath.Join(tmpDir, "test.db")

	// Migrate to version 1
	err := MigrateToVersionFromPath(dbPath, migrationsPath, 1)
	if err != nil {
		t.Fatalf("MigrateToVersionFromPath() error = %v", err)
	}

	version, _, err := GetMigrationVersionFromPath(dbPath, migrationsPath)
	if err != nil {
		t.Fatalf("GetMigrationVersionFromPath() error = %v", err)
	}
	if version != 1 {
		t.Errorf("version = %d, want 1", version)
	}
}

// TestForceMigrationVersionFromPath verifies forcing version without running migrations.
func TestForceMigrationVersionFromPath(t *testing.T) {
	tmpDir, migrationsPath, cleanup := setupTestMigrations(t)
	defer cleanup()

	dbPath := filepath.Join(tmpDir, "test.db")

	// Create database
	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}
	db.Close()

	// Force version to 1 without running migrations
	err = ForceMigrationVersionFromPath(dbPath, migrationsPath, 1)
	if err != nil {
		t.Fatalf("ForceMigrationVersionFromPath() error = %v", err)
	}

	version, _, err := GetMigrationVersionFromPath(dbPath, migrationsPath)
	if err != nil {
		t.Fatalf("GetMigrationVersionFromPath() error = %v", err)
	}
	if version != 1 {
		t.Errorf("version = %d, want 1", version)
	}

	// Table should NOT exist (migrations weren't actually run)
	db, err = NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&count)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if count != 0 {
		t.Error("test_table should NOT exist when using ForceMigrationVersionFromPath")
	}
}

// TestMigrateDownFromPath_SingleStep verifies rolling back a single migration.
func TestMigrateDownFromPath_SingleStep(t *testing.T) {
	tmpDir, migrationsPath, cleanup := setupTestMigrations(t)
	defer cleanup()

	dbPath := filepath.Join(tmpDir, "test.db")

	// Apply migrations
	err := MigrateUpFromPath(dbPath, migrationsPath)
	if err != nil {
		t.Fatalf("MigrateUpFromPath() error = %v", err)
	}

	// Roll back 1 step
	err = MigrateDownFromPath(dbPath, migrationsPath, 1)
	if err != nil {
		t.Fatalf("MigrateDownFromPath(1) error = %v", err)
	}

	// Version should be 0 after rolling back the only migration
	version, _, err := GetMigrationVersionFromPath(dbPath, migrationsPath)
	if err != nil {
		t.Fatalf("GetMigrationVersionFromPath() error = %v", err)
	}
	if version != 0 {
		t.Errorf("version = %d, want 0 after rollback", version)
	}
}

// TestMigrateUp_ClosesConnection verifies the documented behavior that
// the database connection variant closes the connection.
func TestMigrateUp_ClosesConnection(t *testing.T) {
	tmpDir, migrationsPath, cleanup := setupTestMigrations(t)
	defer cleanup()

	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		t.Fatalf("failed to create db: %v", err)
	}

	// MigrateUp takes ownership and closes the connection
	err = MigrateUp(db, migrationsPath)
	if err != nil {
		t.Fatalf("MigrateUp() error = %v", err)
	}

	// Verify the connection is closed by attempting to ping
	err = db.Ping()
	if err == nil {
		t.Error("db.Ping() should fail after MigrateUp closes connection")
	}
	if err != nil && err != sql.ErrConnDone {
		// Some drivers return different errors for closed connections
		// Just verify we got an error
		t.Logf("Got expected error type: %v", err)
	}
}
