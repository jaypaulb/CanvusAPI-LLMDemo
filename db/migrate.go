// Package db provides database utilities including connection management,
// async write functionality, and migration support.
package db

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	_ "github.com/golang-migrate/migrate/v4/source/file" // File source driver
)

// MigrationConfig holds configuration for running migrations.
type MigrationConfig struct {
	// MigrationsPath is the path to the migrations directory (e.g., "file://db/migrations")
	MigrationsPath string
	// DatabaseName is used by golang-migrate for internal tracking (default: "main")
	DatabaseName string
}

// DefaultMigrationConfig returns sensible defaults for migration configuration.
func DefaultMigrationConfig(migrationsPath string) MigrationConfig {
	return MigrationConfig{
		MigrationsPath: migrationsPath,
		DatabaseName:   "main",
	}
}

// RunMigrations applies all pending up migrations to the database.
// This is a convenience function that creates a migrator and runs MigrateUp.
//
// IMPORTANT: This function takes ownership of the database connection and will
// close it when complete. Do not use the db connection after calling this function.
// For a path-based approach that manages its own connection, use RunMigrationsFromPath.
//
// It handles migrate.ErrNoChange gracefully (not an error condition).
//
// Example:
//
//	err := RunMigrations(db, "file://db/migrations")
//	// Note: db is now closed, do not use it after this call
func RunMigrations(db *sql.DB, migrationsPath string) error {
	return MigrateUp(db, migrationsPath)
}

// RunMigrationsFromPath applies all pending migrations using a database path.
// This is the recommended approach as it manages its own connection lifecycle.
//
// Example:
//
//	err := RunMigrationsFromPath("/path/to/db.sqlite", "file://db/migrations")
func RunMigrationsFromPath(dbPath, migrationsPath string) error {
	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	// Note: defer not needed here because MigrateUp closes via migrator.Close()

	return MigrateUp(db, migrationsPath)
}

// MigrateUp applies all pending up migrations.
// Returns nil if there are no pending migrations (ErrNoChange is handled gracefully).
//
// IMPORTANT: This function takes ownership of the database connection and will
// close it when complete. Do not use the db connection after calling this function.
//
// Example:
//
//	err := MigrateUp(db, "file://db/migrations")
//	if err != nil {
//	    log.Fatal(err)
//	}
func MigrateUp(db *sql.DB, migrationsPath string) error {
	m, err := newMigrator(db, DefaultMigrationConfig(migrationsPath))
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			// No pending migrations is not an error
			return nil
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	return nil
}

// MigrateUpFromPath applies all pending migrations using a database path.
// This is the recommended approach as it manages its own connection lifecycle.
//
// Example:
//
//	err := MigrateUpFromPath("/path/to/db.sqlite", "file://db/migrations")
func MigrateUpFromPath(dbPath, migrationsPath string) error {
	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	return MigrateUp(db, migrationsPath)
}

// MigrateDown rolls back migrations by the specified number of steps.
// Pass -1 to roll back all migrations.
// Returns nil if there are no migrations to roll back (ErrNoChange is handled gracefully).
//
// IMPORTANT: This function takes ownership of the database connection and will
// close it when complete.
//
// Example:
//
//	// Roll back 1 migration
//	err := MigrateDown(db, "file://db/migrations", 1)
//
//	// Roll back all migrations
//	err := MigrateDown(db, "file://db/migrations", -1)
func MigrateDown(db *sql.DB, migrationsPath string, steps int) error {
	m, err := newMigrator(db, DefaultMigrationConfig(migrationsPath))
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	var migrateErr error
	if steps == -1 {
		// Roll back all migrations
		migrateErr = m.Down()
	} else {
		// Roll back specific number of steps
		migrateErr = m.Steps(-steps)
	}

	if migrateErr != nil {
		if errors.Is(migrateErr, migrate.ErrNoChange) {
			// No migrations to roll back is not an error
			return nil
		}
		return fmt.Errorf("failed to roll back migrations: %w", migrateErr)
	}

	return nil
}

// MigrateDownFromPath rolls back migrations using a database path.
// This is the recommended approach as it manages its own connection lifecycle.
//
// Example:
//
//	err := MigrateDownFromPath("/path/to/db.sqlite", "file://db/migrations", 1)
func MigrateDownFromPath(dbPath, migrationsPath string, steps int) error {
	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	return MigrateDown(db, migrationsPath, steps)
}

// GetMigrationVersion returns the current migration version and dirty state.
// Returns version=0 and dirty=false if no migrations have been applied.
//
// The dirty flag indicates if a migration failed partway through.
// If dirty is true, manual intervention may be required.
//
// IMPORTANT: This function takes ownership of the database connection and will
// close it when complete.
//
// Example:
//
//	version, dirty, err := GetMigrationVersion(db, "file://db/migrations")
//	if dirty {
//	    log.Warn("database is in dirty state")
//	}
func GetMigrationVersion(db *sql.DB, migrationsPath string) (uint, bool, error) {
	m, err := newMigrator(db, DefaultMigrationConfig(migrationsPath))
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	version, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			// No migrations applied yet
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}

// GetMigrationVersionFromPath returns migration version using a database path.
// This is the recommended approach as it manages its own connection lifecycle.
//
// Example:
//
//	version, dirty, err := GetMigrationVersionFromPath("/path/to/db.sqlite", "file://db/migrations")
func GetMigrationVersionFromPath(dbPath, migrationsPath string) (uint, bool, error) {
	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		return 0, false, fmt.Errorf("failed to open database: %w", err)
	}

	return GetMigrationVersion(db, migrationsPath)
}

// MigrateToVersion migrates to a specific version.
// Use this for precise version control.
//
// IMPORTANT: This function takes ownership of the database connection and will
// close it when complete.
//
// Example:
//
//	err := MigrateToVersion(db, "file://db/migrations", 3)
func MigrateToVersion(db *sql.DB, migrationsPath string, version uint) error {
	m, err := newMigrator(db, DefaultMigrationConfig(migrationsPath))
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Migrate(version); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("failed to migrate to version %d: %w", version, err)
	}

	return nil
}

// MigrateToVersionFromPath migrates to a specific version using a database path.
// This is the recommended approach as it manages its own connection lifecycle.
//
// Example:
//
//	err := MigrateToVersionFromPath("/path/to/db.sqlite", "file://db/migrations", 3)
func MigrateToVersionFromPath(dbPath, migrationsPath string, version uint) error {
	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	return MigrateToVersion(db, migrationsPath, version)
}

// ForceMigrationVersion forcibly sets the migration version without running migrations.
// Use this to fix a dirty state after manual database repair.
//
// WARNING: This does not run any migrations. Only use when you know what you're doing.
//
// IMPORTANT: This function takes ownership of the database connection and will
// close it when complete.
//
// Example:
//
//	// After manually fixing the database, set version to last known good
//	err := ForceMigrationVersion(db, "file://db/migrations", 2)
func ForceMigrationVersion(db *sql.DB, migrationsPath string, version int) error {
	m, err := newMigrator(db, DefaultMigrationConfig(migrationsPath))
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version to %d: %w", version, err)
	}

	return nil
}

// ForceMigrationVersionFromPath forcibly sets migration version using a database path.
// This is the recommended approach as it manages its own connection lifecycle.
//
// Example:
//
//	err := ForceMigrationVersionFromPath("/path/to/db.sqlite", "file://db/migrations", 2)
func ForceMigrationVersionFromPath(dbPath, migrationsPath string, version int) error {
	db, err := NewSQLiteConnectionWithDefaults(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	return ForceMigrationVersion(db, migrationsPath, version)
}

// newMigrator creates a new migrate.Migrate instance for the given database.
// This is an internal helper that handles driver setup.
//
// Note: The returned migrator takes ownership of the database connection.
// When migrator.Close() is called, the database connection is also closed.
func newMigrator(db *sql.DB, config MigrationConfig) (*migrate.Migrate, error) {
	if db == nil {
		return nil, errors.New("database connection is required")
	}
	if config.MigrationsPath == "" {
		return nil, errors.New("migrations path is required")
	}

	// Create sqlite driver instance
	driver, err := sqlite.WithInstance(db, &sqlite.Config{
		DatabaseName: config.DatabaseName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create sqlite driver: %w", err)
	}

	// Create migrator with file source
	m, err := migrate.NewWithDatabaseInstance(
		config.MigrationsPath,
		"sqlite",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return m, nil
}
