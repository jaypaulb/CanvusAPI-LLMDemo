package validation

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestValidationSuite_Creation(t *testing.T) {
	suite := NewValidationSuite()

	if suite == nil {
		t.Fatal("NewValidationSuite() returned nil")
	}
	if suite.output == nil {
		t.Error("output should not be nil")
	}
	if suite.configValidator == nil {
		t.Error("configValidator should not be nil")
	}
	if suite.connectivityChecker == nil {
		t.Error("connectivityChecker should not be nil")
	}
	if suite.authChecker == nil {
		t.Error("authChecker should not be nil")
	}
	if suite.canvasChecker == nil {
		t.Error("canvasChecker should not be nil")
	}
}

func TestValidationSuite_BuilderPattern(t *testing.T) {
	var buf bytes.Buffer

	suite := NewValidationSuite().
		WithOutput(&buf).
		WithAllowSelfSignedCerts(true).
		WithTimeout(5 * time.Second).
		WithShowProgress(false).
		WithFailFast(true).
		WithEnvPath("/custom/path/.env")

	if suite.output != &buf {
		t.Error("WithOutput did not set output correctly")
	}
	if !suite.allowSelfSignedCerts {
		t.Error("WithAllowSelfSignedCerts did not set value correctly")
	}
	if suite.timeout != 5*time.Second {
		t.Error("WithTimeout did not set timeout correctly")
	}
	if suite.showProgress {
		t.Error("WithShowProgress did not set value correctly")
	}
	if !suite.failFast {
		t.Error("WithFailFast did not set value correctly")
	}
}

func TestStepStatus_String(t *testing.T) {
	tests := []struct {
		status   StepStatus
		expected string
	}{
		{StepPending, "pending"},
		{StepRunning, "running"},
		{StepPassed, "passed"},
		{StepFailed, "failed"},
		{StepWarning, "warning"},
		{StepSkipped, "skipped"},
		{StepStatus(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.status.String(); got != tt.expected {
				t.Errorf("StepStatus(%d).String() = %q, want %q", tt.status, got, tt.expected)
			}
		})
	}
}

func TestSuiteResult_GetErrors(t *testing.T) {
	result := SuiteResult{
		Steps: []ValidationStep{
			{Name: "Step1", Status: StepPassed, Error: nil},
			{Name: "Step2", Status: StepFailed, Error: ErrMissingConfig("TEST")},
			{Name: "Step3", Status: StepPassed, Error: nil},
			{Name: "Step4", Status: StepFailed, Error: ErrMissingAuth("test")},
		},
	}

	errors := result.GetErrors()
	if len(errors) != 2 {
		t.Errorf("GetErrors() returned %d errors, expected 2", len(errors))
	}
}

func TestSuiteResult_GetFirstError(t *testing.T) {
	t.Run("has errors", func(t *testing.T) {
		result := SuiteResult{
			Steps: []ValidationStep{
				{Name: "Step1", Status: StepPassed, Error: nil},
				{Name: "Step2", Status: StepFailed, Error: ErrMissingConfig("TEST")},
			},
		}

		err := result.GetFirstError()
		if err == nil {
			t.Error("GetFirstError() should return error when steps have errors")
		}
	})

	t.Run("no errors", func(t *testing.T) {
		result := SuiteResult{
			Steps: []ValidationStep{
				{Name: "Step1", Status: StepPassed, Error: nil},
				{Name: "Step2", Status: StepPassed, Error: nil},
			},
		}

		err := result.GetFirstError()
		if err != nil {
			t.Errorf("GetFirstError() should return nil when no errors, got: %v", err)
		}
	})
}

func TestSuiteResult_Summary(t *testing.T) {
	result := SuiteResult{
		Success:     true,
		TotalSteps:  6,
		PassedSteps: 6,
		FailedSteps: 0,
		Warnings:    0,
		Duration:    1500 * time.Millisecond,
	}

	summary := result.Summary()
	if !strings.Contains(summary, "Passed") {
		t.Error("Summary should contain 'Passed'")
	}
	if !strings.Contains(summary, "6/6") {
		t.Error("Summary should contain '6/6'")
	}
}

func TestSuiteResult_Summary_Failed(t *testing.T) {
	result := SuiteResult{
		Success:     false,
		TotalSteps:  6,
		PassedSteps: 4,
		FailedSteps: 2,
		Warnings:    1,
		Duration:    2000 * time.Millisecond,
	}

	summary := result.Summary()
	if !strings.Contains(summary, "Failed") {
		t.Error("Summary should contain 'Failed'")
	}
	if !strings.Contains(summary, "4/6") {
		t.Error("Summary should contain '4/6'")
	}
	if !strings.Contains(summary, "2 failed") {
		t.Error("Summary should contain '2 failed'")
	}
	if !strings.Contains(summary, "1 warning") {
		t.Error("Summary should contain '1 warning'")
	}
}

func TestValidationSuite_ValidateQuick_NoEnvFile(t *testing.T) {
	// Use a temp directory without .env file
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	var buf bytes.Buffer
	suite := NewValidationSuite().
		WithOutput(&buf).
		WithShowProgress(false)

	result := suite.ValidateQuick()

	if result.Success {
		t.Error("ValidateQuick should fail when .env file is missing")
	}
	if result.FailedSteps == 0 {
		t.Error("Should have at least one failed step")
	}
}

func TestValidationSuite_ValidateQuick_WithEnvFile(t *testing.T) {
	// Create temp directory with .env file
	tempDir := t.TempDir()
	envPath := tempDir + "/.env"
	envContent := `CANVUS_SERVER=https://test.example.com
CANVAS_ID=test-canvas-123
CANVUS_API_KEY=test-api-key-12345678`
	os.WriteFile(envPath, []byte(envContent), 0644)

	// Save current env and restore after
	oldServer := os.Getenv("CANVUS_SERVER")
	oldCanvasID := os.Getenv("CANVAS_ID")
	oldAPIKey := os.Getenv("CANVUS_API_KEY")
	defer func() {
		os.Setenv("CANVUS_SERVER", oldServer)
		os.Setenv("CANVAS_ID", oldCanvasID)
		os.Setenv("CANVUS_API_KEY", oldAPIKey)
	}()

	// Set env vars
	os.Setenv("CANVUS_SERVER", "https://test.example.com")
	os.Setenv("CANVAS_ID", "test-canvas-123")
	os.Setenv("CANVUS_API_KEY", "test-api-key-12345678")

	var buf bytes.Buffer
	suite := NewValidationSuite().
		WithOutput(&buf).
		WithShowProgress(false).
		WithEnvPath(envPath)

	result := suite.ValidateQuick()

	if !result.Success {
		t.Errorf("ValidateQuick should pass with valid config, errors: %v", result.GetErrors())
	}
	if result.PassedSteps != 4 {
		t.Errorf("Should have 4 passed steps, got %d", result.PassedSteps)
	}
}

func TestValidationSuite_FailFast(t *testing.T) {
	// Use a temp directory without .env file
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	os.Chdir(tempDir)
	defer os.Chdir(oldDir)

	var buf bytes.Buffer
	suite := NewValidationSuite().
		WithOutput(&buf).
		WithShowProgress(false).
		WithFailFast(true)

	result := suite.ValidateQuick()

	// With fail fast, should stop after first failure
	if result.TotalSteps != 1 {
		t.Errorf("FailFast should stop after first failure, got %d steps", result.TotalSteps)
	}
}

func TestValidationSuite_ProgressOutput(t *testing.T) {
	// Create temp directory with .env file
	tempDir := t.TempDir()
	envPath := tempDir + "/.env"
	os.WriteFile(envPath, []byte(""), 0644)

	var buf bytes.Buffer
	suite := NewValidationSuite().
		WithOutput(&buf).
		WithShowProgress(true).
		WithEnvPath(envPath)

	// Just run quick validation to check output
	suite.ValidateQuick()

	output := buf.String()
	if !strings.Contains(output, "Configuration") {
		t.Error("Progress output should contain header")
	}
	if !strings.Contains(output, "Environment File") {
		t.Error("Progress output should contain step names")
	}
}

func TestValidationSuite_buildResult(t *testing.T) {
	suite := NewValidationSuite()
	startTime := time.Now().Add(-100 * time.Millisecond)

	steps := []ValidationStep{
		{Name: "Step1", Status: StepPassed},
		{Name: "Step2", Status: StepFailed},
		{Name: "Step3", Status: StepWarning},
		{Name: "Step4", Status: StepSkipped},
	}

	result := suite.buildResult(steps, startTime)

	if result.TotalSteps != 4 {
		t.Errorf("TotalSteps = %d, want 4", result.TotalSteps)
	}
	if result.PassedSteps != 1 {
		t.Errorf("PassedSteps = %d, want 1", result.PassedSteps)
	}
	if result.FailedSteps != 1 {
		t.Errorf("FailedSteps = %d, want 1", result.FailedSteps)
	}
	if result.Warnings != 1 {
		t.Errorf("Warnings = %d, want 1", result.Warnings)
	}
	if result.Success {
		t.Error("Success should be false when there are failures")
	}
	if result.Duration < 100*time.Millisecond {
		t.Errorf("Duration should be at least 100ms, got %v", result.Duration)
	}
}

func TestValidationSuite_hasAllPassed(t *testing.T) {
	suite := NewValidationSuite()

	tests := []struct {
		name     string
		steps    []ValidationStep
		expected bool
	}{
		{
			name: "all passed",
			steps: []ValidationStep{
				{Status: StepPassed},
				{Status: StepPassed},
			},
			expected: true,
		},
		{
			name: "has failure",
			steps: []ValidationStep{
				{Status: StepPassed},
				{Status: StepFailed},
			},
			expected: false,
		},
		{
			name:     "empty",
			steps:    []ValidationStep{},
			expected: true,
		},
		{
			name: "skipped counts as passed",
			steps: []ValidationStep{
				{Status: StepPassed},
				{Status: StepSkipped},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := suite.hasAllPassed(tt.steps); got != tt.expected {
				t.Errorf("hasAllPassed() = %v, want %v", got, tt.expected)
			}
		})
	}
}
