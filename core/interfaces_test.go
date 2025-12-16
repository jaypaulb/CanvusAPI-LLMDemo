// Package core provides shared interfaces for CanvusLocalLLM components.
package core

import (
	"errors"
	"testing"
)

// TestProgressReporterInterface verifies that types implementing ProgressReporter
// can be used interchangeably.
func TestProgressReporterInterface(t *testing.T) {
	// This is a compile-time check that mockProgressReporter implements ProgressReporter
	var _ ProgressReporter = (*mockProgressReporter)(nil)
}

// mockProgressReporter is a test implementation of ProgressReporter.
type mockProgressReporter struct {
	progressMessages []string
	errorMessages    []string
	successMessages  []string
	cleanupCalled    bool
	failOnProgress   bool
	failOnError      bool
	failOnSuccess    bool
	failOnCleanup    bool
}

func (m *mockProgressReporter) ReportProgress(message string) error {
	if m.failOnProgress {
		return errors.New("mock progress error")
	}
	m.progressMessages = append(m.progressMessages, message)
	return nil
}

func (m *mockProgressReporter) ReportError(err error) error {
	if m.failOnError {
		return errors.New("mock error reporting error")
	}
	m.errorMessages = append(m.errorMessages, err.Error())
	return nil
}

func (m *mockProgressReporter) ReportSuccess(message string) error {
	if m.failOnSuccess {
		return errors.New("mock success error")
	}
	m.successMessages = append(m.successMessages, message)
	return nil
}

func (m *mockProgressReporter) Cleanup() error {
	if m.failOnCleanup {
		return errors.New("mock cleanup error")
	}
	m.cleanupCalled = true
	return nil
}

// TestMockProgressReporter_ReportProgress tests progress reporting.
func TestMockProgressReporter_ReportProgress(t *testing.T) {
	tests := []struct {
		name           string
		messages       []string
		failOnProgress bool
		wantErr        bool
		wantMessages   []string
	}{
		{
			name:           "single message",
			messages:       []string{"Processing..."},
			failOnProgress: false,
			wantErr:        false,
			wantMessages:   []string{"Processing..."},
		},
		{
			name:           "multiple messages",
			messages:       []string{"Step 1", "Step 2", "Step 3"},
			failOnProgress: false,
			wantErr:        false,
			wantMessages:   []string{"Step 1", "Step 2", "Step 3"},
		},
		{
			name:           "fails on progress",
			messages:       []string{"Will fail"},
			failOnProgress: true,
			wantErr:        true,
			wantMessages:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockProgressReporter{failOnProgress: tt.failOnProgress}

			var gotErr bool
			for _, msg := range tt.messages {
				if err := m.ReportProgress(msg); err != nil {
					gotErr = true
					break
				}
			}

			if gotErr != tt.wantErr {
				t.Errorf("ReportProgress() error = %v, wantErr %v", gotErr, tt.wantErr)
			}

			if !tt.wantErr {
				if len(m.progressMessages) != len(tt.wantMessages) {
					t.Errorf("got %d messages, want %d", len(m.progressMessages), len(tt.wantMessages))
				}
				for i, msg := range tt.wantMessages {
					if m.progressMessages[i] != msg {
						t.Errorf("message[%d] = %q, want %q", i, m.progressMessages[i], msg)
					}
				}
			}
		})
	}
}

// TestMockProgressReporter_ReportError tests error reporting.
func TestMockProgressReporter_ReportError(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		failOnError bool
		wantErr     bool
	}{
		{
			name:        "reports error successfully",
			err:         errors.New("something went wrong"),
			failOnError: false,
			wantErr:     false,
		},
		{
			name:        "fails to report error",
			err:         errors.New("original error"),
			failOnError: true,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockProgressReporter{failOnError: tt.failOnError}

			err := m.ReportError(tt.err)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReportError() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && len(m.errorMessages) != 1 {
				t.Error("expected error message to be recorded")
			}
		})
	}
}

// TestMockProgressReporter_ReportSuccess tests success reporting.
func TestMockProgressReporter_ReportSuccess(t *testing.T) {
	m := &mockProgressReporter{}

	if err := m.ReportSuccess("Operation complete!"); err != nil {
		t.Errorf("ReportSuccess() unexpected error: %v", err)
	}

	if len(m.successMessages) != 1 || m.successMessages[0] != "Operation complete!" {
		t.Error("success message not recorded correctly")
	}
}

// TestMockProgressReporter_Cleanup tests cleanup behavior.
func TestMockProgressReporter_Cleanup(t *testing.T) {
	tests := []struct {
		name          string
		failOnCleanup bool
		wantErr       bool
	}{
		{
			name:          "cleanup succeeds",
			failOnCleanup: false,
			wantErr:       false,
		},
		{
			name:          "cleanup fails",
			failOnCleanup: true,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockProgressReporter{failOnCleanup: tt.failOnCleanup}

			err := m.Cleanup()
			if (err != nil) != tt.wantErr {
				t.Errorf("Cleanup() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.failOnCleanup && !m.cleanupCalled {
				t.Error("cleanup should have been marked as called")
			}
		})
	}
}

// TestProgressReporterUsagePattern demonstrates typical usage pattern.
func TestProgressReporterUsagePattern(t *testing.T) {
	reporter := &mockProgressReporter{}

	// Typical usage pattern with defer for cleanup
	defer func() {
		if err := reporter.Cleanup(); err != nil {
			t.Errorf("cleanup failed: %v", err)
		}
	}()

	// Report progress
	if err := reporter.ReportProgress("Starting operation..."); err != nil {
		t.Fatalf("progress report failed: %v", err)
	}

	if err := reporter.ReportProgress("Step 1 complete"); err != nil {
		t.Fatalf("progress report failed: %v", err)
	}

	// Simulate success
	if err := reporter.ReportSuccess("All done!"); err != nil {
		t.Fatalf("success report failed: %v", err)
	}

	// Verify state
	if len(reporter.progressMessages) != 2 {
		t.Errorf("expected 2 progress messages, got %d", len(reporter.progressMessages))
	}

	if len(reporter.successMessages) != 1 {
		t.Errorf("expected 1 success message, got %d", len(reporter.successMessages))
	}
}
