// Package db provides database utilities including connection management,
// async write functionality, and migration support.
package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Database is the main database organism that composes:
// - SQLite connection with WAL mode (molecule)
// - Migration runner (molecule)
// - Async writer for non-blocking writes (molecule)
//
// This is an organism-level component that manages the database lifecycle
// including initialization, migration, and graceful shutdown.
//
// Usage:
//
//	db, err := NewDatabase("/path/to/data.db")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
//
//	// Access underlying connection for repositories
//	conn := db.DB()
type Database struct {
	db             *sql.DB
	path           string
	migrationsPath string
	mu             sync.RWMutex
}

// DatabaseConfig holds configuration for the Database organism.
type DatabaseConfig struct {
	// Path is the database file path
	Path string
	// MigrationsPath is the path to migrations directory (file:// URL format)
	// Default: "file://db/migrations"
	MigrationsPath string
	// ConnectionConfig allows customizing the SQLite connection
	ConnectionConfig *ConnectionConfig
}

// DefaultDatabaseConfig returns sensible defaults for the database.
func DefaultDatabaseConfig(path string) DatabaseConfig {
	return DatabaseConfig{
		Path:             path,
		MigrationsPath:   "file://db/migrations",
		ConnectionConfig: nil, // Use defaults
	}
}

// NewDatabase creates a new Database instance with default configuration.
// It initializes the database connection with WAL mode and foreign keys enabled,
// and runs any pending migrations.
//
// The database file and its parent directories are created if they don't exist.
//
// Example:
//
//	db, err := NewDatabase("/home/user/.canvuslocallm/data.db")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
func NewDatabase(path string) (*Database, error) {
	return NewDatabaseWithConfig(DefaultDatabaseConfig(path))
}

// NewDatabaseWithConfig creates a new Database instance with custom configuration.
//
// Example:
//
//	config := DatabaseConfig{
//	    Path:           "/path/to/db.sqlite",
//	    MigrationsPath: "file://custom/migrations",
//	}
//	db, err := NewDatabaseWithConfig(config)
func NewDatabaseWithConfig(config DatabaseConfig) (*Database, error) {
	if config.Path == "" {
		return nil, fmt.Errorf("database path is required")
	}

	// Ensure parent directory exists
	dir := filepath.Dir(config.Path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory %s: %w", dir, err)
		}
	}

	// Determine connection config
	var connConfig ConnectionConfig
	if config.ConnectionConfig != nil {
		connConfig = *config.ConnectionConfig
	} else {
		connConfig = DefaultConnectionConfig(config.Path)
	}

	// Create SQLite connection with WAL mode and foreign keys
	conn, err := NewSQLiteConnection(connConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
	}

	// Set migrations path default if not provided
	migrationsPath := config.MigrationsPath
	if migrationsPath == "" {
		migrationsPath = "file://db/migrations"
	}

	database := &Database{
		db:             conn,
		path:           config.Path,
		migrationsPath: migrationsPath,
	}

	return database, nil
}

// Migrate runs all pending database migrations.
// This method is safe to call multiple times; it will only apply
// migrations that haven't been applied yet.
//
// Migrations are sourced from the configured migrations path
// (default: file://db/migrations).
//
// Note: This method creates a separate connection for migrations
// to avoid connection ownership issues with golang-migrate.
//
// Example:
//
//	if err := db.Migrate(); err != nil {
//	    log.Fatal("Migration failed:", err)
//	}
func (d *Database) Migrate() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// golang-migrate takes ownership of the connection it's given,
	// so we use the path-based function which manages its own connection
	if err := MigrateUpFromPath(d.path, d.migrationsPath); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// MigrateWithPath runs migrations from a specific path.
// Use this when migrations are located in a non-default location.
//
// Example:
//
//	if err := db.MigrateWithPath("file://custom/migrations"); err != nil {
//	    log.Fatal(err)
//	}
func (d *Database) MigrateWithPath(migrationsPath string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if err := MigrateUpFromPath(d.path, migrationsPath); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// DB returns the underlying sql.DB connection for use by repositories.
// The returned connection should not be closed directly; use Database.Close() instead.
//
// Example:
//
//	conn := db.DB()
//	rows, err := conn.Query("SELECT * FROM processing_history")
func (d *Database) DB() *sql.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.db
}

// Path returns the database file path.
func (d *Database) Path() string {
	return d.path
}

// Close gracefully closes the database connection.
// This should be called when the application shuts down.
//
// After Close is called, the Database instance should not be used.
//
// Example:
//
//	defer db.Close()
func (d *Database) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return nil
	}

	// Close the database connection
	if err := d.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	d.db = nil
	return nil
}

// Ping verifies the database connection is alive.
// This is useful for health checks.
//
// Example:
//
//	if err := db.Ping(); err != nil {
//	    log.Warn("Database connection unhealthy")
//	}
func (d *Database) Ping() error {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return fmt.Errorf("database connection is closed")
	}

	return d.db.Ping()
}

// Stats returns database connection pool statistics.
// Useful for monitoring and debugging.
func (d *Database) Stats() sql.DBStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return sql.DBStats{}
	}

	return d.db.Stats()
}

// Exec executes a query without returning any rows.
// This is a convenience wrapper around sql.DB.Exec.
func (d *Database) Exec(query string, args ...interface{}) (sql.Result, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("database connection is closed")
	}

	return d.db.Exec(query, args...)
}

// Query executes a query that returns rows.
// This is a convenience wrapper around sql.DB.Query.
func (d *Database) Query(query string, args ...interface{}) (*sql.Rows, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("database connection is closed")
	}

	return d.db.Query(query, args...)
}

// QueryRow executes a query that returns at most one row.
// This is a convenience wrapper around sql.DB.QueryRow.
func (d *Database) QueryRow(query string, args ...interface{}) *sql.Row {
	d.mu.RLock()
	defer d.mu.RUnlock()

	// Note: QueryRow never returns an error, it defers error to Scan
	return d.db.QueryRow(query, args...)
}

// Begin starts a new transaction.
// This is a convenience wrapper around sql.DB.Begin.
func (d *Database) Begin() (*sql.Tx, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.db == nil {
		return nil, fmt.Errorf("database connection is closed")
	}

	return d.db.Begin()
}
