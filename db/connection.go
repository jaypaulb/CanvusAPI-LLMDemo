// Package db provides database utilities including connection management and async write functionality.
package db

import (
	"database/sql"
	"fmt"
	"time"

	// SQLite driver (pure Go, no CGO required)
	_ "modernc.org/sqlite"
)

// ConnectionConfig holds configuration for SQLite connections.
type ConnectionConfig struct {
	// Path is the database file path
	Path string
	// BusyTimeout is how long to wait for locks (milliseconds)
	BusyTimeout int
	// MaxOpenConns limits concurrent connections (SQLite recommends 1 for writes)
	MaxOpenConns int
	// MaxIdleConns limits idle connections in pool
	MaxIdleConns int
	// ConnMaxLifetime limits how long a connection can be reused (0 = no limit)
	ConnMaxLifetime time.Duration
}

// DefaultConnectionConfig returns sensible defaults for SQLite.
// Uses WAL mode with settings optimized for concurrent read, single write.
func DefaultConnectionConfig(path string) ConnectionConfig {
	return ConnectionConfig{
		Path:            path,
		BusyTimeout:     5000, // 5 seconds
		MaxOpenConns:    1,    // SQLite handles concurrency best with single writer
		MaxIdleConns:    1,
		ConnMaxLifetime: 0, // No limit
	}
}

// NewSQLiteConnection creates a new SQLite database connection with WAL mode enabled.
//
// This molecule composes:
// - Database path validation (atom)
// - sql.Open (stdlib)
// - Pragma configuration via SQL statements (atoms)
// - Connection pool settings (atoms)
//
// WAL (Write-Ahead Logging) mode enables:
// - Concurrent readers with a single writer
// - Better write performance
// - Crash recovery
//
// Example:
//
//	db, err := NewSQLiteConnection(DefaultConnectionConfig("/path/to/db.sqlite"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer db.Close()
func NewSQLiteConnection(config ConnectionConfig) (*sql.DB, error) {
	if config.Path == "" {
		return nil, fmt.Errorf("database path is required")
	}

	// Open database connection
	// modernc.org/sqlite uses simple path as DSN
	db, err := sql.Open("sqlite", config.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify connection works
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Configure pragmas via SQL statements
	// These must be executed on each connection, so we do it right after opening
	pragmas := []struct {
		name  string
		query string
	}{
		{"journal_mode", "PRAGMA journal_mode=WAL"},
		{"busy_timeout", fmt.Sprintf("PRAGMA busy_timeout=%d", config.BusyTimeout)},
		{"foreign_keys", "PRAGMA foreign_keys=ON"},
	}

	for _, p := range pragmas {
		if _, err := db.Exec(p.query); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set %s pragma: %w", p.name, err)
		}
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Verify WAL mode was enabled (some configurations may prevent it)
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to verify journal mode: %w", err)
	}
	if journalMode != "wal" {
		db.Close()
		return nil, fmt.Errorf("WAL mode not enabled, got: %s", journalMode)
	}

	return db, nil
}

// NewSQLiteConnectionWithDefaults creates a connection using default configuration.
// This is a convenience wrapper around NewSQLiteConnection.
func NewSQLiteConnectionWithDefaults(path string) (*sql.DB, error) {
	return NewSQLiteConnection(DefaultConnectionConfig(path))
}
