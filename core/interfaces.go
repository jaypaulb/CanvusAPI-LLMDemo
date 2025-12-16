// Package core provides shared interfaces for CanvusLocalLLM components.
package core

// ProgressReporter is the interface for reporting progress during AI operations.
// Implementations should handle updating a UI element (typically a canvas note)
// with status updates during long-running operations.
//
// This interface enables dependency injection and testing of progress-reporting
// components without requiring actual API clients.
//
// Example usage:
//
//	reporter := handlers.NewProgressReporter(client, processingNoteID, config, log)
//	defer reporter.Cleanup() // Always cleanup to delete processing note
//
//	reporter.ReportProgress("Downloading file...")
//	// ... download file ...
//	reporter.ReportProgress("Processing with AI...")
//	// ... process ...
//	reporter.ReportSuccess("Processing complete!")
type ProgressReporter interface {
	// ReportProgress updates the progress note with an informational message.
	// This is used for status updates during normal operation.
	// Returns an error if the update fails.
	ReportProgress(message string) error

	// ReportError updates the progress note with an error message.
	// The error is formatted and displayed to indicate failure.
	// Returns an error if the update fails.
	ReportError(err error) error

	// ReportSuccess updates the progress note with a success message.
	// This is typically called when the operation completes successfully.
	// Returns an error if the update fails.
	ReportSuccess(message string) error

	// Cleanup removes the progress note from the canvas.
	// This should be called when the operation is complete (success or failure)
	// to clean up the UI. Typically called via defer.
	// Returns an error if deletion fails.
	Cleanup() error
}
