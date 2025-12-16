package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDefaultConnectionConfig verifies default configuration values.
func TestDefaultConnectionConfig(t *testing.T) {
	path := "/test/path.db"
	config := DefaultConnectionConfig(path)

	if config.Path != path {
		t.Errorf("Path = %q, want %q", config.Path, path)
	}
	if config.BusyTimeout != 5000 {
		t.Errorf("BusyTimeout = %d, want 5000", config.BusyTimeout)
	}
	if config.MaxOpenConns != 1 {
		t.Errorf("MaxOpenConns = %d, want 1", config.MaxOpenConns)
	}
	if config.MaxIdleConns != 1 {
		t.Errorf("MaxIdleConns = %d, want 1", config.MaxIdleConns)
	}
	if config.ConnMaxLifetime != 0 {
		t.Errorf("ConnMaxLifetime = %v, want 0", config.ConnMaxLifetime)
	}
}

// TestNewSQLiteConnection_EmptyPath verifies error on empty path.
func TestNewSQLiteConnection_EmptyPath(t *testing.T) {
	config := ConnectionConfig{
		Path: "",
	}

	db, err := NewSQLiteConnection(config)
	if err == nil {
		db.Close()
		t.Fatal("expected error for empty path, got nil")
	}
	if db != nil {
		t.Error("expected nil db for empty path")
	}
}

// TestNewSQLiteConnection_CreatesDatabase verifies database file creation.
func TestNewSQLiteConnection_CreatesDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	config := DefaultConnectionConfig(dbPath)
	db, err := NewSQLiteConnection(config)
	if err != nil {
		t.Fatalf("NewSQLiteConnection() error = %v", err)
	}
	defer db.Close()

	// Verify file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

// TestNewSQLiteConnection_WALMode verifies WAL journal mode is enabled.
func TestNewSQLiteConnection_WALMode(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_wal.db")

	config := DefaultConnectionConfig(dbPath)
	db, err := NewSQLiteConnection(config)
	if err != nil {
		t.Fatalf("NewSQLiteConnection() error = %v", err)
	}
	defer db.Close()

	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want %q", journalMode, "wal")
	}

	// WAL mode creates additional files
	walPath := dbPath + "-wal"
	shmPath := dbPath + "-shm"
	// These files may or may not exist immediately depending on write activity
	// Just verify we can query the mode
	_ = walPath
	_ = shmPath
}

// TestNewSQLiteConnection_ForeignKeys verifies foreign keys are enabled.
func TestNewSQLiteConnection_ForeignKeys(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_fk.db")

	config := DefaultConnectionConfig(dbPath)
	db, err := NewSQLiteConnection(config)
	if err != nil {
		t.Fatalf("NewSQLiteConnection() error = %v", err)
	}
	defer db.Close()

	var fkEnabled int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Fatalf("failed to query foreign_keys: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("foreign_keys = %d, want 1", fkEnabled)
	}
}

// TestNewSQLiteConnection_CustomConfig verifies custom configuration is applied.
func TestNewSQLiteConnection_CustomConfig(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_custom.db")

	config := ConnectionConfig{
		Path:            dbPath,
		BusyTimeout:     10000,
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 1 * time.Hour,
	}

	db, err := NewSQLiteConnection(config)
	if err != nil {
		t.Fatalf("NewSQLiteConnection() error = %v", err)
	}
	defer db.Close()

	// Verify busy_timeout pragma
	var busyTimeout int
	err = db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout)
	if err != nil {
		t.Fatalf("failed to query busy_timeout: %v", err)
	}
	if busyTimeout != 10000 {
		t.Errorf("busy_timeout = %d, want 10000", busyTimeout)
	}

	// Connection pool settings can't be queried directly from SQLite
	// but we verify no errors occurred when setting them
}

// TestNewSQLiteConnection_ConcurrentReads verifies concurrent read access works.
func TestNewSQLiteConnection_ConcurrentReads(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_concurrent.db")

	config := ConnectionConfig{
		Path:         dbPath,
		BusyTimeout:  5000,
		MaxOpenConns: 5, // Allow multiple connections for this test
		MaxIdleConns: 5,
	}

	db, err := NewSQLiteConnection(config)
	if err != nil {
		t.Fatalf("NewSQLiteConnection() error = %v", err)
	}
	defer db.Close()

	// Create a test table
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, value TEXT)")
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Insert some data
	_, err = db.Exec("INSERT INTO test (value) VALUES (?)", "test_value")
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	// Perform concurrent reads
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			var value string
			err := db.QueryRow("SELECT value FROM test WHERE id = 1").Scan(&value)
			if err != nil {
				t.Errorf("concurrent read failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all reads
	for i := 0; i < 5; i++ {
		<-done
	}
}

// TestNewSQLiteConnectionWithDefaults verifies the convenience wrapper.
func TestNewSQLiteConnectionWithDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_defaults.db")

	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteConnectionWithDefaults() error = %v", err)
	}
	defer db.Close()

	// Verify WAL mode is enabled
	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("failed to query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want %q", journalMode, "wal")
	}
}

// TestNewSQLiteConnection_InvalidPath verifies error handling for invalid paths.
func TestNewSQLiteConnection_InvalidPath(t *testing.T) {
	// Try to create database in a non-existent directory
	config := DefaultConnectionConfig("/nonexistent/directory/test.db")
	db, err := NewSQLiteConnection(config)
	if err == nil {
		db.Close()
		t.Fatal("expected error for invalid path, got nil")
	}
}

// TestNewSQLiteConnection_Ping verifies database is accessible after creation.
func TestNewSQLiteConnection_Ping(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_ping.db")

	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteConnectionWithDefaults() error = %v", err)
	}
	defer db.Close()

	// Ping should work immediately
	if err := db.Ping(); err != nil {
		t.Errorf("Ping() error = %v", err)
	}
}
