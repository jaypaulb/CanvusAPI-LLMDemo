// Package db provides database utilities including cleanup and retention functionality.
package db

import (
	"context"
	"fmt"
	"time"
)

// CleanupResult contains statistics about a cleanup operation.
type CleanupResult struct {
	// ProcessingHistoryDeleted is the number of records deleted from processing_history
	ProcessingHistoryDeleted int64
	// CanvasEventsDeleted is the number of records deleted from canvas_events
	CanvasEventsDeleted int64
	// PerformanceMetricsDeleted is the number of records deleted from performance_metrics
	PerformanceMetricsDeleted int64
	// ErrorLogDeleted is the number of records deleted from error_log
	ErrorLogDeleted int64
	// SystemMetricsDeleted is the number of records deleted from system_metrics
	SystemMetricsDeleted int64
	// TotalDeleted is the sum of all deleted records
	TotalDeleted int64
	// Duration is how long the cleanup took
	Duration time.Duration
}

// tablesToClean defines the tables that have retention policies.
// All tables must have a created_at column with DATETIME type.
var tablesToClean = []string{
	"processing_history",
	"canvas_events",
	"performance_metrics",
	"error_log",
	"system_metrics",
}

// Cleanup deletes records older than retentionDays from all retention-managed tables
// and runs VACUUM to reclaim disk space.
//
// This method is thread-safe and uses a transaction to ensure atomicity.
// If any deletion fails, the entire operation is rolled back.
//
// Example:
//
//	result, err := db.Cleanup(30) // Delete records older than 30 days
//	if err != nil {
//	    log.Printf("Cleanup failed: %v", err)
//	}
//	log.Printf("Cleaned up %d total records", result.TotalDeleted)
func (d *Database) Cleanup(retentionDays int) (CleanupResult, error) {
	return d.CleanupWithContext(context.Background(), retentionDays)
}

// CleanupWithContext deletes records older than retentionDays from all retention-managed
// tables, respecting context cancellation.
//
// This is the context-aware version of Cleanup. It will return early if the context
// is cancelled, rolling back any pending changes.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
//	defer cancel()
//
//	result, err := db.CleanupWithContext(ctx, 30)
//	if err != nil {
//	    if ctx.Err() != nil {
//	        log.Printf("Cleanup cancelled: %v", ctx.Err())
//	    } else {
//	        log.Printf("Cleanup failed: %v", err)
//	    }
//	}
func (d *Database) CleanupWithContext(ctx context.Context, retentionDays int) (CleanupResult, error) {
	start := time.Now()
	result := CleanupResult{}

	if retentionDays < 0 {
		return result, fmt.Errorf("retentionDays must be non-negative, got %d", retentionDays)
	}

	// Check context before starting
	select {
	case <-ctx.Done():
		return result, ctx.Err()
	default:
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.db == nil {
		return result, fmt.Errorf("database connection is closed")
	}

	// Begin transaction
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if tx != nil {
			tx.Rollback() // No-op if already committed
		}
	}()

	// Delete from each table within the transaction
	// SQLite datetime comparison: datetime('now', '-N days')
	deletedCounts := make(map[string]int64)

	for _, table := range tablesToClean {
		// Check context before each table
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		query := fmt.Sprintf(
			"DELETE FROM %s WHERE created_at < datetime('now', '-%d days')",
			table, retentionDays,
		)

		res, err := tx.ExecContext(ctx, query)
		if err != nil {
			return result, fmt.Errorf("failed to delete from %s: %w", table, err)
		}

		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return result, fmt.Errorf("failed to get rows affected for %s: %w", table, err)
		}

		deletedCounts[table] = rowsAffected
		result.TotalDeleted += rowsAffected
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("failed to commit transaction: %w", err)
	}
	tx = nil // Prevent rollback in defer

	// Populate result struct
	result.ProcessingHistoryDeleted = deletedCounts["processing_history"]
	result.CanvasEventsDeleted = deletedCounts["canvas_events"]
	result.PerformanceMetricsDeleted = deletedCounts["performance_metrics"]
	result.ErrorLogDeleted = deletedCounts["error_log"]
	result.SystemMetricsDeleted = deletedCounts["system_metrics"]

	// Check context before VACUUM
	select {
	case <-ctx.Done():
		// Transaction committed, but VACUUM not run - acceptable partial success
		result.Duration = time.Since(start)
		return result, ctx.Err()
	default:
	}

	// Run VACUUM to reclaim disk space (must be outside transaction)
	if _, err := d.db.ExecContext(ctx, "VACUUM"); err != nil {
		// VACUUM failure is not critical - data was already deleted
		// Log but don't return error
		result.Duration = time.Since(start)
		return result, fmt.Errorf("cleanup succeeded but VACUUM failed: %w", err)
	}

	result.Duration = time.Since(start)
	return result, nil
}

// CleanupSchedulerConfig holds configuration for the cleanup scheduler.
type CleanupSchedulerConfig struct {
	// RetentionDays is the number of days to retain records
	RetentionDays int
	// Interval is how often to run cleanup
	Interval time.Duration
	// OnCleanup is called after each cleanup run (optional)
	// Useful for logging or metrics
	OnCleanup func(result CleanupResult, err error)
}

// DefaultCleanupSchedulerConfig returns sensible defaults for the cleanup scheduler.
func DefaultCleanupSchedulerConfig() CleanupSchedulerConfig {
	return CleanupSchedulerConfig{
		RetentionDays: 30,
		Interval:      24 * time.Hour,
		OnCleanup:     nil,
	}
}

// StartCleanupScheduler starts a background goroutine that periodically runs cleanup.
//
// The scheduler runs cleanup at the specified interval and stops when the context
// is cancelled. It runs an initial cleanup immediately, then subsequent cleanups
// at each interval.
//
// Example:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	// Run cleanup every 24 hours, keeping 30 days of data
//	db.StartCleanupScheduler(ctx, 30, 24*time.Hour)
//
//	// Later, to stop the scheduler:
//	cancel()
func (d *Database) StartCleanupScheduler(ctx context.Context, retentionDays int, interval time.Duration) {
	config := CleanupSchedulerConfig{
		RetentionDays: retentionDays,
		Interval:      interval,
		OnCleanup:     nil,
	}
	d.StartCleanupSchedulerWithConfig(ctx, config)
}

// StartCleanupSchedulerWithConfig starts a cleanup scheduler with custom configuration.
//
// This version allows specifying a callback for cleanup results, useful for
// logging or monitoring.
//
// Example:
//
//	config := db.CleanupSchedulerConfig{
//	    RetentionDays: 30,
//	    Interval:      24 * time.Hour,
//	    OnCleanup: func(result db.CleanupResult, err error) {
//	        if err != nil {
//	            log.Printf("Cleanup error: %v", err)
//	        } else {
//	            log.Printf("Cleanup deleted %d records in %v", result.TotalDeleted, result.Duration)
//	        }
//	    },
//	}
//	db.StartCleanupSchedulerWithConfig(ctx, config)
func (d *Database) StartCleanupSchedulerWithConfig(ctx context.Context, config CleanupSchedulerConfig) {
	go func() {
		// Run initial cleanup immediately
		result, err := d.CleanupWithContext(ctx, config.RetentionDays)
		if config.OnCleanup != nil {
			config.OnCleanup(result, err)
		}

		// Set up ticker for periodic cleanup
		ticker := time.NewTicker(config.Interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				result, err := d.CleanupWithContext(ctx, config.RetentionDays)
				if config.OnCleanup != nil {
					config.OnCleanup(result, err)
				}
			}
		}
	}()
}
