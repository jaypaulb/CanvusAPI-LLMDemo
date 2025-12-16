// Package handlers provides the ProgressReporter molecule for progress note operations.
package handlers

import (
	"errors"
	"fmt"
	"time"

	"go_backend/core"
	"go_backend/logging"

	"go.uber.org/zap"
)

// ProgressReporter errors
var (
	// ErrNilProgressClient is returned when the API client is nil.
	ErrNilProgressClient = errors.New("nil API client")
	// ErrEmptyProgressNoteID is returned when the progress note ID is empty.
	ErrEmptyProgressNoteID = errors.New("empty progress note ID")
	// ErrCleanupFailed is returned when the progress note deletion fails.
	ErrCleanupFailed = errors.New("failed to cleanup progress note")
)

// ProgressAPIClient is the interface for progress note API operations.
// This allows for dependency injection and testing.
type ProgressAPIClient interface {
	UpdateNote(id string, payload map[string]interface{}) (map[string]interface{}, error)
	DeleteNote(id string) error
}

// ProgressReporterConfig holds configuration for progress reporting behavior.
type ProgressReporterConfig struct {
	MaxRetries    int
	RetryDelay    time.Duration
	DeleteOnError bool // Whether to delete the progress note on error
}

// DefaultProgressReporterConfig returns sensible defaults for progress reporter configuration.
func DefaultProgressReporterConfig() ProgressReporterConfig {
	return ProgressReporterConfig{
		MaxRetries:    3,
		RetryDelay:    500 * time.Millisecond,
		DeleteOnError: true,
	}
}

// ProgressReporter provides progress note update operations with retry logic.
// It implements the core.ProgressReporter interface.
//
// This is a molecule that composes:
// - NoteAPIClient interface for API operations
// - Retry logic for resilient updates
// - Logging for observability
//
// Example:
//
//	reporter := handlers.NewProgressReporter(client, noteID, config, log)
//	defer reporter.Cleanup()
//
//	reporter.ReportProgress("Processing...")
//	// ... do work ...
//	reporter.ReportSuccess("Done!")
type ProgressReporter struct {
	client     ProgressAPIClient
	noteID     string
	config     ProgressReporterConfig
	log        *logging.Logger
	cleanedUp  bool
	errorState bool
}

// Compile-time check that ProgressReporter implements core.ProgressReporter
var _ core.ProgressReporter = (*ProgressReporter)(nil)

// NewProgressReporter creates a new ProgressReporter for managing a progress note.
//
// Parameters:
//   - client: API client for note operations (must not be nil)
//   - noteID: ID of the progress note to update (must not be empty)
//   - config: Configuration for retry behavior
//   - log: Logger for observability (can be nil for no logging)
//
// Returns a configured ProgressReporter ready for use.
func NewProgressReporter(
	client ProgressAPIClient,
	noteID string,
	config ProgressReporterConfig,
	log *logging.Logger,
) *ProgressReporter {
	return &ProgressReporter{
		client:     client,
		noteID:     noteID,
		config:     config,
		log:        log,
		cleanedUp:  false,
		errorState: false,
	}
}

// ReportProgress updates the progress note with a status message.
// Uses retry logic for resilience against transient failures.
//
// Example:
//
//	reporter.ReportProgress("Downloading file... (50%)")
func (r *ProgressReporter) ReportProgress(message string) error {
	if err := r.validate(); err != nil {
		return err
	}

	if r.log != nil {
		r.log.Debug("reporting progress",
			zap.String("note_id", r.noteID),
			zap.String("message", message))
	}

	payload := map[string]interface{}{
		"text": message,
	}

	return r.updateWithRetry(payload)
}

// ReportError updates the progress note with an error message.
// The error is formatted with a visual indicator for clarity.
//
// Example:
//
//	if err := doSomething(); err != nil {
//	    reporter.ReportError(err)
//	}
func (r *ProgressReporter) ReportError(err error) error {
	if validateErr := r.validate(); validateErr != nil {
		return validateErr
	}

	r.errorState = true

	errorMessage := fmt.Sprintf("Error: %v", err)

	if r.log != nil {
		r.log.Warn("reporting error to progress note",
			zap.String("note_id", r.noteID),
			zap.Error(err))
	}

	payload := map[string]interface{}{
		"text": errorMessage,
	}

	return r.updateWithRetry(payload)
}

// ReportSuccess updates the progress note with a success message.
// The message is formatted with a visual indicator for clarity.
//
// Example:
//
//	reporter.ReportSuccess("Processing complete! Created 5 summaries.")
func (r *ProgressReporter) ReportSuccess(message string) error {
	if err := r.validate(); err != nil {
		return err
	}

	successMessage := fmt.Sprintf("Success: %s", message)

	if r.log != nil {
		r.log.Info("reporting success to progress note",
			zap.String("note_id", r.noteID),
			zap.String("message", message))
	}

	payload := map[string]interface{}{
		"text": successMessage,
	}

	return r.updateWithRetry(payload)
}

// Cleanup deletes the progress note from the canvas.
// This is idempotent - calling multiple times has no effect after the first call.
// Typically called via defer immediately after creating the reporter.
//
// Example:
//
//	reporter := NewProgressReporter(client, noteID, config, log)
//	defer reporter.Cleanup()
func (r *ProgressReporter) Cleanup() error {
	// Idempotent - only cleanup once
	if r.cleanedUp {
		if r.log != nil {
			r.log.Debug("progress note already cleaned up", zap.String("note_id", r.noteID))
		}
		return nil
	}

	if err := r.validate(); err != nil {
		return err
	}

	// Skip cleanup if in error state and config says don't delete on error
	if r.errorState && !r.config.DeleteOnError {
		if r.log != nil {
			r.log.Debug("skipping cleanup due to error state",
				zap.String("note_id", r.noteID))
		}
		r.cleanedUp = true
		return nil
	}

	if r.log != nil {
		r.log.Debug("cleaning up progress note", zap.String("note_id", r.noteID))
	}

	if err := r.deleteWithRetry(); err != nil {
		if r.log != nil {
			r.log.Warn("failed to cleanup progress note",
				zap.String("note_id", r.noteID),
				zap.Error(err))
		}
		return fmt.Errorf("%w: %v", ErrCleanupFailed, err)
	}

	r.cleanedUp = true

	if r.log != nil {
		r.log.Debug("progress note cleaned up successfully", zap.String("note_id", r.noteID))
	}

	return nil
}

// IsCleanedUp returns whether the progress note has been deleted.
func (r *ProgressReporter) IsCleanedUp() bool {
	return r.cleanedUp
}

// IsErrorState returns whether the reporter has reported an error.
func (r *ProgressReporter) IsErrorState() bool {
	return r.errorState
}

// NoteID returns the ID of the progress note being managed.
func (r *ProgressReporter) NoteID() string {
	return r.noteID
}

// validate checks that the reporter is properly configured.
func (r *ProgressReporter) validate() error {
	if r.client == nil {
		return ErrNilProgressClient
	}
	if r.noteID == "" {
		return ErrEmptyProgressNoteID
	}
	return nil
}

// updateWithRetry performs a note update with retry logic.
func (r *ProgressReporter) updateWithRetry(payload map[string]interface{}) error {
	var lastErr error

	for attempt := 1; attempt <= r.config.MaxRetries; attempt++ {
		if r.log != nil {
			r.log.Debug("attempting progress note update",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", r.config.MaxRetries))
		}

		_, err := r.client.UpdateNote(r.noteID, payload)
		if err == nil {
			return nil
		}

		lastErr = err

		if r.log != nil {
			r.log.Warn("progress note update failed",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", r.config.MaxRetries),
				zap.Error(err))
		}

		// Don't sleep after the last attempt
		if attempt < r.config.MaxRetries {
			time.Sleep(r.config.RetryDelay)
		}
	}

	return fmt.Errorf("failed to update progress note %s after %d attempts: %w",
		r.noteID, r.config.MaxRetries, lastErr)
}

// deleteWithRetry performs a note deletion with retry logic.
func (r *ProgressReporter) deleteWithRetry() error {
	var lastErr error

	for attempt := 1; attempt <= r.config.MaxRetries; attempt++ {
		if r.log != nil {
			r.log.Debug("attempting progress note deletion",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", r.config.MaxRetries))
		}

		err := r.client.DeleteNote(r.noteID)
		if err == nil {
			return nil
		}

		lastErr = err

		if r.log != nil {
			r.log.Warn("progress note deletion failed",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", r.config.MaxRetries),
				zap.Error(err))
		}

		// Don't sleep after the last attempt
		if attempt < r.config.MaxRetries {
			time.Sleep(r.config.RetryDelay)
		}
	}

	return fmt.Errorf("failed to delete progress note %s after %d attempts: %w",
		r.noteID, r.config.MaxRetries, lastErr)
}

// NoOpProgressReporter is a progress reporter that does nothing.
// Useful for testing or when progress reporting is not needed.
type NoOpProgressReporter struct{}

// Compile-time check that NoOpProgressReporter implements core.ProgressReporter
var _ core.ProgressReporter = (*NoOpProgressReporter)(nil)

// ReportProgress does nothing and returns nil.
func (n *NoOpProgressReporter) ReportProgress(message string) error { return nil }

// ReportError does nothing and returns nil.
func (n *NoOpProgressReporter) ReportError(err error) error { return nil }

// ReportSuccess does nothing and returns nil.
func (n *NoOpProgressReporter) ReportSuccess(message string) error { return nil }

// Cleanup does nothing and returns nil.
func (n *NoOpProgressReporter) Cleanup() error { return nil }
