// Package handlers provides tests for the ProgressReporter molecule.
package handlers

import (
	"errors"
	"testing"
	"time"

	"go_backend/core"
)

// mockProgressAPIClient is a test implementation of ProgressAPIClient.
type mockProgressAPIClient struct {
	updateCalls       []updateCall
	deleteCalls       []string
	failUpdateUntil   int // Fail updates until this many attempts
	failDeleteUntil   int // Fail deletes until this many attempts
	currentUpdateCall int
	currentDeleteCall int
	updateErr         error
	deleteErr         error
}

type updateCall struct {
	id      string
	payload map[string]interface{}
}

func (m *mockProgressAPIClient) UpdateNote(id string, payload map[string]interface{}) (map[string]interface{}, error) {
	m.currentUpdateCall++
	m.updateCalls = append(m.updateCalls, updateCall{id: id, payload: payload})

	if m.currentUpdateCall <= m.failUpdateUntil {
		if m.updateErr != nil {
			return nil, m.updateErr
		}
		return nil, errors.New("mock update error")
	}
	return payload, nil
}

func (m *mockProgressAPIClient) DeleteNote(id string) error {
	m.currentDeleteCall++
	m.deleteCalls = append(m.deleteCalls, id)

	if m.currentDeleteCall <= m.failDeleteUntil {
		if m.deleteErr != nil {
			return m.deleteErr
		}
		return errors.New("mock delete error")
	}
	return nil
}

// TestProgressReporterInterface verifies that ProgressReporter implements core.ProgressReporter.
func TestProgressReporterInterface(t *testing.T) {
	var _ core.ProgressReporter = (*ProgressReporter)(nil)
	var _ core.ProgressReporter = (*NoOpProgressReporter)(nil)
}

// TestNewProgressReporter tests the constructor.
func TestNewProgressReporter(t *testing.T) {
	client := &mockProgressAPIClient{}
	config := DefaultProgressReporterConfig()

	reporter := NewProgressReporter(client, "note-123", config, nil)

	if reporter == nil {
		t.Fatal("expected non-nil reporter")
	}

	if reporter.NoteID() != "note-123" {
		t.Errorf("NoteID() = %q, want %q", reporter.NoteID(), "note-123")
	}

	if reporter.IsCleanedUp() {
		t.Error("new reporter should not be cleaned up")
	}

	if reporter.IsErrorState() {
		t.Error("new reporter should not be in error state")
	}
}

// TestDefaultProgressReporterConfig tests the default configuration.
func TestDefaultProgressReporterConfig(t *testing.T) {
	config := DefaultProgressReporterConfig()

	if config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", config.MaxRetries)
	}

	if config.RetryDelay != 500*time.Millisecond {
		t.Errorf("RetryDelay = %v, want 500ms", config.RetryDelay)
	}

	if !config.DeleteOnError {
		t.Error("DeleteOnError should be true by default")
	}
}

// TestProgressReporter_ReportProgress tests progress reporting.
func TestProgressReporter_ReportProgress(t *testing.T) {
	tests := []struct {
		name           string
		noteID         string
		client         ProgressAPIClient
		mockClient     *mockProgressAPIClient // For verifying calls
		message        string
		wantErr        bool
		wantErrType    error
		wantUpdateCall bool
	}{
		{
			name:           "successful progress report",
			noteID:         "note-123",
			client:         &mockProgressAPIClient{},
			mockClient:     nil, // Will be set from client
			message:        "Processing...",
			wantErr:        false,
			wantUpdateCall: true,
		},
		{
			name:        "nil client returns error",
			noteID:      "note-123",
			client:      nil, // true nil interface
			mockClient:  nil,
			message:     "Processing...",
			wantErr:     true,
			wantErrType: ErrNilProgressClient,
		},
		{
			name:        "empty note ID returns error",
			noteID:      "",
			client:      &mockProgressAPIClient{},
			mockClient:  nil, // Will be set from client
			message:     "Processing...",
			wantErr:     true,
			wantErrType: ErrEmptyProgressNoteID,
		},
		{
			name:           "empty message is allowed",
			noteID:         "note-123",
			client:         &mockProgressAPIClient{},
			mockClient:     nil, // Will be set from client
			message:        "",
			wantErr:        false,
			wantUpdateCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ProgressReporterConfig{
				MaxRetries: 1,
				RetryDelay: 1 * time.Millisecond,
			}

			// Get the mock client if client is not nil
			var mockClient *mockProgressAPIClient
			if tt.client != nil {
				mockClient = tt.client.(*mockProgressAPIClient)
			}

			reporter := NewProgressReporter(tt.client, tt.noteID, config, nil)

			err := reporter.ReportProgress(tt.message)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.wantErrType != nil && !errors.Is(err, tt.wantErrType) {
					t.Errorf("error = %v, want %v", err, tt.wantErrType)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.wantUpdateCall && mockClient != nil && len(mockClient.updateCalls) == 0 {
				t.Error("expected UpdateNote to be called")
			}

			if tt.wantUpdateCall && mockClient != nil {
				call := mockClient.updateCalls[0]
				if call.id != tt.noteID {
					t.Errorf("UpdateNote id = %q, want %q", call.id, tt.noteID)
				}
				if text, ok := call.payload["text"].(string); !ok || text != tt.message {
					t.Errorf("UpdateNote text = %q, want %q", text, tt.message)
				}
			}
		})
	}
}

// TestProgressReporter_ReportError tests error reporting.
func TestProgressReporter_ReportError(t *testing.T) {
	tests := []struct {
		name           string
		reportErr      error
		wantTextPrefix string
		wantErrorState bool
	}{
		{
			name:           "reports wrapped error",
			reportErr:      errors.New("something went wrong"),
			wantTextPrefix: "Error: something went wrong",
			wantErrorState: true,
		},
		{
			name:           "reports formatted error",
			reportErr:      errors.New("file not found: /path/to/file"),
			wantTextPrefix: "Error: file not found: /path/to/file",
			wantErrorState: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockProgressAPIClient{}
			config := ProgressReporterConfig{
				MaxRetries: 1,
				RetryDelay: 1 * time.Millisecond,
			}

			reporter := NewProgressReporter(client, "note-123", config, nil)

			err := reporter.ReportError(tt.reportErr)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(client.updateCalls) != 1 {
				t.Fatal("expected exactly one update call")
			}

			call := client.updateCalls[0]
			text, ok := call.payload["text"].(string)
			if !ok {
				t.Fatal("expected text in payload")
			}

			if text != tt.wantTextPrefix {
				t.Errorf("text = %q, want %q", text, tt.wantTextPrefix)
			}

			if reporter.IsErrorState() != tt.wantErrorState {
				t.Errorf("IsErrorState() = %v, want %v", reporter.IsErrorState(), tt.wantErrorState)
			}
		})
	}
}

// TestProgressReporter_ReportSuccess tests success reporting.
func TestProgressReporter_ReportSuccess(t *testing.T) {
	client := &mockProgressAPIClient{}
	config := ProgressReporterConfig{
		MaxRetries: 1,
		RetryDelay: 1 * time.Millisecond,
	}

	reporter := NewProgressReporter(client, "note-123", config, nil)

	err := reporter.ReportSuccess("All done!")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(client.updateCalls) != 1 {
		t.Fatal("expected exactly one update call")
	}

	call := client.updateCalls[0]
	text, ok := call.payload["text"].(string)
	if !ok {
		t.Fatal("expected text in payload")
	}

	expectedText := "Success: All done!"
	if text != expectedText {
		t.Errorf("text = %q, want %q", text, expectedText)
	}

	// Success should not set error state
	if reporter.IsErrorState() {
		t.Error("success should not set error state")
	}
}

// TestProgressReporter_Cleanup tests cleanup behavior.
func TestProgressReporter_Cleanup(t *testing.T) {
	tests := []struct {
		name               string
		setupFunc          func(r *ProgressReporter)
		deleteOnError      bool
		wantDeleteCall     bool
		wantErr            bool
		wantCleanedUpAfter bool
	}{
		{
			name:               "normal cleanup deletes note",
			setupFunc:          func(r *ProgressReporter) {},
			deleteOnError:      true,
			wantDeleteCall:     true,
			wantErr:            false,
			wantCleanedUpAfter: true,
		},
		{
			name: "cleanup after error with DeleteOnError=true",
			setupFunc: func(r *ProgressReporter) {
				r.ReportError(errors.New("test error"))
			},
			deleteOnError:      true,
			wantDeleteCall:     true,
			wantErr:            false,
			wantCleanedUpAfter: true,
		},
		{
			name: "cleanup after error with DeleteOnError=false",
			setupFunc: func(r *ProgressReporter) {
				r.ReportError(errors.New("test error"))
			},
			deleteOnError:      false,
			wantDeleteCall:     false,
			wantErr:            false,
			wantCleanedUpAfter: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockProgressAPIClient{}
			config := ProgressReporterConfig{
				MaxRetries:    1,
				RetryDelay:    1 * time.Millisecond,
				DeleteOnError: tt.deleteOnError,
			}

			reporter := NewProgressReporter(client, "note-123", config, nil)
			tt.setupFunc(reporter)

			// Reset client counters after setup
			initialDeleteCalls := len(client.deleteCalls)

			err := reporter.Cleanup()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			newDeleteCalls := len(client.deleteCalls) - initialDeleteCalls
			if tt.wantDeleteCall && newDeleteCalls == 0 {
				t.Error("expected DeleteNote to be called")
			}
			if !tt.wantDeleteCall && newDeleteCalls > 0 {
				t.Error("expected DeleteNote NOT to be called")
			}

			if reporter.IsCleanedUp() != tt.wantCleanedUpAfter {
				t.Errorf("IsCleanedUp() = %v, want %v", reporter.IsCleanedUp(), tt.wantCleanedUpAfter)
			}
		})
	}
}

// TestProgressReporter_CleanupIdempotent tests that cleanup is idempotent.
func TestProgressReporter_CleanupIdempotent(t *testing.T) {
	client := &mockProgressAPIClient{}
	config := ProgressReporterConfig{
		MaxRetries: 1,
		RetryDelay: 1 * time.Millisecond,
	}

	reporter := NewProgressReporter(client, "note-123", config, nil)

	// First cleanup
	err := reporter.Cleanup()
	if err != nil {
		t.Errorf("first cleanup failed: %v", err)
	}

	if len(client.deleteCalls) != 1 {
		t.Errorf("expected 1 delete call after first cleanup, got %d", len(client.deleteCalls))
	}

	// Second cleanup should not call delete again
	err = reporter.Cleanup()
	if err != nil {
		t.Errorf("second cleanup failed: %v", err)
	}

	if len(client.deleteCalls) != 1 {
		t.Errorf("expected still 1 delete call after second cleanup, got %d", len(client.deleteCalls))
	}
}

// TestProgressReporter_RetryLogic tests retry behavior.
func TestProgressReporter_RetryLogic(t *testing.T) {
	tests := []struct {
		name            string
		failUpdateUntil int
		maxRetries      int
		wantErr         bool
		wantAttempts    int
	}{
		{
			name:            "succeeds on first try",
			failUpdateUntil: 0,
			maxRetries:      3,
			wantErr:         false,
			wantAttempts:    1,
		},
		{
			name:            "succeeds after retry",
			failUpdateUntil: 1,
			maxRetries:      3,
			wantErr:         false,
			wantAttempts:    2,
		},
		{
			name:            "fails after max retries",
			failUpdateUntil: 5,
			maxRetries:      3,
			wantErr:         true,
			wantAttempts:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockProgressAPIClient{
				failUpdateUntil: tt.failUpdateUntil,
			}
			config := ProgressReporterConfig{
				MaxRetries: tt.maxRetries,
				RetryDelay: 1 * time.Millisecond,
			}

			reporter := NewProgressReporter(client, "note-123", config, nil)

			err := reporter.ReportProgress("test")

			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(client.updateCalls) != tt.wantAttempts {
				t.Errorf("got %d update attempts, want %d", len(client.updateCalls), tt.wantAttempts)
			}
		})
	}
}

// TestProgressReporter_DeleteRetryLogic tests retry behavior for deletion.
func TestProgressReporter_DeleteRetryLogic(t *testing.T) {
	tests := []struct {
		name            string
		failDeleteUntil int
		maxRetries      int
		wantErr         bool
		wantAttempts    int
	}{
		{
			name:            "delete succeeds on first try",
			failDeleteUntil: 0,
			maxRetries:      3,
			wantErr:         false,
			wantAttempts:    1,
		},
		{
			name:            "delete succeeds after retry",
			failDeleteUntil: 1,
			maxRetries:      3,
			wantErr:         false,
			wantAttempts:    2,
		},
		{
			name:            "delete fails after max retries",
			failDeleteUntil: 5,
			maxRetries:      3,
			wantErr:         true,
			wantAttempts:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockProgressAPIClient{
				failDeleteUntil: tt.failDeleteUntil,
			}
			config := ProgressReporterConfig{
				MaxRetries: tt.maxRetries,
				RetryDelay: 1 * time.Millisecond,
			}

			reporter := NewProgressReporter(client, "note-123", config, nil)

			err := reporter.Cleanup()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if !errors.Is(err, ErrCleanupFailed) {
					t.Errorf("error should wrap ErrCleanupFailed, got: %v", err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if len(client.deleteCalls) != tt.wantAttempts {
				t.Errorf("got %d delete attempts, want %d", len(client.deleteCalls), tt.wantAttempts)
			}
		})
	}
}

// TestNoOpProgressReporter tests the no-op implementation.
func TestNoOpProgressReporter(t *testing.T) {
	reporter := &NoOpProgressReporter{}

	// All methods should succeed silently
	if err := reporter.ReportProgress("message"); err != nil {
		t.Errorf("ReportProgress() error = %v", err)
	}

	if err := reporter.ReportError(errors.New("test")); err != nil {
		t.Errorf("ReportError() error = %v", err)
	}

	if err := reporter.ReportSuccess("done"); err != nil {
		t.Errorf("ReportSuccess() error = %v", err)
	}

	if err := reporter.Cleanup(); err != nil {
		t.Errorf("Cleanup() error = %v", err)
	}
}

// TestProgressReporter_UsagePattern demonstrates the typical usage pattern.
func TestProgressReporter_UsagePattern(t *testing.T) {
	client := &mockProgressAPIClient{}
	config := DefaultProgressReporterConfig()
	config.RetryDelay = 1 * time.Millisecond // Speed up test

	reporter := NewProgressReporter(client, "note-123", config, nil)

	// Typical pattern: defer cleanup
	defer func() {
		if err := reporter.Cleanup(); err != nil {
			t.Errorf("cleanup failed: %v", err)
		}
	}()

	// Report progress through operation
	if err := reporter.ReportProgress("Starting..."); err != nil {
		t.Fatalf("progress report failed: %v", err)
	}

	if err := reporter.ReportProgress("Processing..."); err != nil {
		t.Fatalf("progress report failed: %v", err)
	}

	if err := reporter.ReportSuccess("Done!"); err != nil {
		t.Fatalf("success report failed: %v", err)
	}

	// Verify calls
	if len(client.updateCalls) != 3 {
		t.Errorf("expected 3 update calls, got %d", len(client.updateCalls))
	}

	// Verify cleanup will be called via defer
	if reporter.IsCleanedUp() {
		t.Error("should not be cleaned up before defer runs")
	}
}

// TestProgressReporter_ErrorRecoveryPattern demonstrates error handling pattern.
func TestProgressReporter_ErrorRecoveryPattern(t *testing.T) {
	client := &mockProgressAPIClient{}
	config := ProgressReporterConfig{
		MaxRetries:    1,
		RetryDelay:    1 * time.Millisecond,
		DeleteOnError: false, // Keep note on error for debugging
	}

	reporter := NewProgressReporter(client, "note-123", config, nil)

	// Simulate an operation that fails
	reporter.ReportProgress("Starting...")
	reporter.ReportError(errors.New("operation failed"))
	reporter.Cleanup()

	// Note should not be deleted since DeleteOnError is false
	if len(client.deleteCalls) != 0 {
		t.Error("expected no delete calls when DeleteOnError is false")
	}

	// But cleanup flag should still be set to prevent double cleanup
	if !reporter.IsCleanedUp() {
		t.Error("IsCleanedUp() should be true after Cleanup()")
	}
}
