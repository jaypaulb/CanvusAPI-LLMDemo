package db

import (
	"os"
	"path/filepath"
	"testing"
)

// TestNewDatabase tests the Database factory function.
func TestNewDatabase(t *testing.T) {
	t.Run("creates database with valid path", func(t *testing.T) {
		// Create temp directory for test
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("NewDatabase() error = %v", err)
		}
		defer db.Close()

		// Verify connection works
		if err := db.Ping(); err != nil {
			t.Errorf("Ping() error = %v", err)
		}

		// Verify path is set
		if db.Path() != dbPath {
			t.Errorf("Path() = %v, want %v", db.Path(), dbPath)
		}

		// Verify file was created
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Errorf("Database file was not created at %s", dbPath)
		}
	})

	t.Run("creates parent directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "nested", "dir", "test.db")

		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("NewDatabase() error = %v", err)
		}
		defer db.Close()

		// Verify nested directory was created
		parentDir := filepath.Dir(dbPath)
		if _, err := os.Stat(parentDir); os.IsNotExist(err) {
			t.Errorf("Parent directory was not created at %s", parentDir)
		}
	})

	t.Run("returns error for empty path", func(t *testing.T) {
		_, err := NewDatabase("")
		if err == nil {
			t.Error("NewDatabase() expected error for empty path, got nil")
		}
	})
}

// TestDatabaseClose tests the Close method.
func TestDatabaseClose(t *testing.T) {
	t.Run("closes database connection", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("NewDatabase() error = %v", err)
		}

		// Close should succeed
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}

		// Ping should fail after close
		if err := db.Ping(); err == nil {
			t.Error("Ping() should fail after Close()")
		}
	})

	t.Run("close is idempotent", func(t *testing.T) {
		tmpDir := t.TempDir()
		dbPath := filepath.Join(tmpDir, "test.db")

		db, err := NewDatabase(dbPath)
		if err != nil {
			t.Fatalf("NewDatabase() error = %v", err)
		}

		// First close
		if err := db.Close(); err != nil {
			t.Errorf("First Close() error = %v", err)
		}

		// Second close should not error
		if err := db.Close(); err != nil {
			t.Errorf("Second Close() error = %v", err)
		}
	})
}

// TestDatabaseDB tests the DB accessor.
func TestDatabaseDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	conn := db.DB()
	if conn == nil {
		t.Error("DB() returned nil")
	}

	// Verify the connection is usable
	var result int
	err = conn.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		t.Errorf("QueryRow() error = %v", err)
	}
	if result != 1 {
		t.Errorf("Query result = %v, want 1", result)
	}
}

// TestDatabaseWALMode tests that WAL mode is enabled.
func TestDatabaseWALMode(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	var journalMode string
	err = db.DB().QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Failed to query journal_mode: %v", err)
	}

	if journalMode != "wal" {
		t.Errorf("journal_mode = %v, want 'wal'", journalMode)
	}
}

// TestDatabaseForeignKeys tests that foreign keys are enabled.
func TestDatabaseForeignKeys(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	var foreignKeys int
	err = db.DB().QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys)
	if err != nil {
		t.Fatalf("Failed to query foreign_keys: %v", err)
	}

	if foreignKeys != 1 {
		t.Errorf("foreign_keys = %v, want 1 (enabled)", foreignKeys)
	}
}

// TestDatabaseExec tests the Exec convenience method.
func TestDatabaseExec(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	// Create a test table
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("Exec() CREATE TABLE error = %v", err)
	}

	// Insert a row
	result, err := db.Exec("INSERT INTO test (name) VALUES (?)", "test_name")
	if err != nil {
		t.Fatalf("Exec() INSERT error = %v", err)
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId() error = %v", err)
	}
	if lastID != 1 {
		t.Errorf("LastInsertId() = %v, want 1", lastID)
	}
}

// TestDatabaseQuery tests the Query convenience method.
func TestDatabaseQuery(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	// Setup test data
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("Setup CREATE TABLE error = %v", err)
	}
	_, err = db.Exec("INSERT INTO test (name) VALUES (?), (?)", "Alice", "Bob")
	if err != nil {
		t.Fatalf("Setup INSERT error = %v", err)
	}

	// Query the data
	rows, err := db.Query("SELECT name FROM test ORDER BY name")
	if err != nil {
		t.Fatalf("Query() error = %v", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("Scan() error = %v", err)
		}
		names = append(names, name)
	}

	if len(names) != 2 {
		t.Errorf("Query returned %d rows, want 2", len(names))
	}
	if names[0] != "Alice" || names[1] != "Bob" {
		t.Errorf("Query results = %v, want [Alice, Bob]", names)
	}
}

// TestDatabaseStats tests the Stats method.
func TestDatabaseStats(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("NewDatabase() error = %v", err)
	}
	defer db.Close()

	stats := db.Stats()

	// MaxOpenConnections should be 1 for SQLite (as configured in DefaultConnectionConfig)
	if stats.MaxOpenConnections != 1 {
		t.Errorf("Stats().MaxOpenConnections = %v, want 1", stats.MaxOpenConnections)
	}
}
