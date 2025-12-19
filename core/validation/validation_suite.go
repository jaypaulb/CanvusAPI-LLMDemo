package validation

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

// ValidationStep represents a single validation step with its status.
type ValidationStep struct {
	Name    string
	Status  StepStatus
	Message string
	Error   error
	Latency time.Duration
}

// StepStatus represents the status of a validation step.
type StepStatus int

const (
	StepPending StepStatus = iota
	StepRunning
	StepPassed
	StepFailed
	StepWarning
	StepSkipped
)

// String returns the string representation of a step status.
func (s StepStatus) String() string {
	switch s {
	case StepPending:
		return "pending"
	case StepRunning:
		return "running"
	case StepPassed:
		return "passed"
	case StepFailed:
		return "failed"
	case StepWarning:
		return "warning"
	case StepSkipped:
		return "skipped"
	default:
		return "unknown"
	}
}

// SuiteResult represents the complete result of validation suite execution.
type SuiteResult struct {
	Steps       []ValidationStep
	TotalSteps  int
	PassedSteps int
	FailedSteps int
	Warnings    int
	Duration    time.Duration
	Success     bool
}

// ValidationSuite orchestrates all validation molecules for complete startup validation.
// This is an organism that composes ConfigValidator, ConnectivityChecker, AuthChecker,
// and CanvasChecker to provide comprehensive validation with progress output.
type ValidationSuite struct {
	output               io.Writer
	configValidator      *ConfigValidator
	connectivityChecker  *ConnectivityChecker
	authChecker          *AuthChecker
	canvasChecker        *CanvasChecker
	allowSelfSignedCerts bool
	timeout              time.Duration
	showProgress         bool
	failFast             bool
}

// NewValidationSuite creates a new ValidationSuite with default settings.
func NewValidationSuite() *ValidationSuite {
	return &ValidationSuite{
		output:               os.Stdout,
		configValidator:      NewConfigValidator(),
		connectivityChecker:  NewConnectivityChecker(),
		authChecker:          NewAuthChecker(),
		canvasChecker:        NewCanvasChecker(),
		allowSelfSignedCerts: false,
		timeout:              30 * time.Second,
		showProgress:         true,
		failFast:             false,
	}
}

// WithOutput sets the output writer for progress messages.
func (s *ValidationSuite) WithOutput(w io.Writer) *ValidationSuite {
	s.output = w
	return s
}

// WithAllowSelfSignedCerts configures whether to allow self-signed certificates.
func (s *ValidationSuite) WithAllowSelfSignedCerts(allow bool) *ValidationSuite {
	s.allowSelfSignedCerts = allow
	s.connectivityChecker.WithAllowSelfSignedCerts(allow)
	s.authChecker.WithAllowSelfSignedCerts(allow)
	s.canvasChecker.WithAllowSelfSignedCerts(allow)
	return s
}

// WithTimeout sets the timeout for network operations.
func (s *ValidationSuite) WithTimeout(timeout time.Duration) *ValidationSuite {
	s.timeout = timeout
	s.connectivityChecker.WithTimeout(timeout)
	s.authChecker.WithTimeout(timeout)
	s.canvasChecker.WithTimeout(timeout)
	return s
}

// WithShowProgress enables or disables progress output.
func (s *ValidationSuite) WithShowProgress(show bool) *ValidationSuite {
	s.showProgress = show
	return s
}

// WithFailFast stops validation on first failure if enabled.
func (s *ValidationSuite) WithFailFast(failFast bool) *ValidationSuite {
	s.failFast = failFast
	return s
}

// WithEnvPath sets a custom path for the .env file.
func (s *ValidationSuite) WithEnvPath(path string) *ValidationSuite {
	s.configValidator.WithEnvPath(path)
	return s
}

// Validate runs all validation checks in sequence with progress output.
// Returns a SuiteResult with complete validation results.
func (s *ValidationSuite) Validate() SuiteResult {
	startTime := time.Now()
	steps := make([]ValidationStep, 0, 6)

	// Header
	if s.showProgress {
		s.printHeader("CanvusLocalLLM Configuration Validation")
	}

	// Step 1: Check .env file exists
	step := s.runStep("Environment File", func() (bool, string, error) {
		result := s.configValidator.CheckEnvFile()
		return result.Valid, result.Message, result.Error
	})
	steps = append(steps, step)
	if s.failFast && step.Status == StepFailed {
		return s.buildResult(steps, startTime)
	}

	// Step 2: Check server URL configuration
	step = s.runStep("Server URL Configuration", func() (bool, string, error) {
		result := s.configValidator.CheckServerURL()
		return result.Valid, result.Message, result.Error
	})
	steps = append(steps, step)
	if s.failFast && step.Status == StepFailed {
		return s.buildResult(steps, startTime)
	}

	// Step 3: Check canvas ID configuration
	step = s.runStep("Canvas ID Configuration", func() (bool, string, error) {
		result := s.configValidator.CheckCanvasID()
		return result.Valid, result.Message, result.Error
	})
	steps = append(steps, step)
	if s.failFast && step.Status == StepFailed {
		return s.buildResult(steps, startTime)
	}

	// Step 4: Check authentication credentials
	step = s.runStep("Authentication Credentials", func() (bool, string, error) {
		result := s.configValidator.CheckAuthCredentials()
		return result.Valid, result.Message, result.Error
	})
	steps = append(steps, step)
	if s.failFast && step.Status == StepFailed {
		return s.buildResult(steps, startTime)
	}

	// Step 5: Check server connectivity (only if config is valid)
	if s.hasAllPassed(steps) {
		step = s.runStep("Server Connectivity", func() (bool, string, error) {
			result := s.connectivityChecker.CheckCanvusServerConnectivity()
			msg := result.Message
			if result.Latency > 0 {
				msg = fmt.Sprintf("%s (latency: %v)", msg, result.Latency.Round(time.Millisecond))
			}
			return result.Reachable, msg, result.Error
		})
	} else {
		step = ValidationStep{
			Name:    "Server Connectivity",
			Status:  StepSkipped,
			Message: "Skipped due to configuration errors",
		}
		if s.showProgress {
			s.printStep(step)
		}
	}
	steps = append(steps, step)
	if s.failFast && step.Status == StepFailed {
		return s.buildResult(steps, startTime)
	}

	// Step 6: Check canvas accessibility (only if connectivity is good)
	if step.Status == StepPassed {
		step = s.runStep("Canvas Accessibility", func() (bool, string, error) {
			result := s.canvasChecker.CheckCanvusCanvas()
			return result.Accessible, result.Message, result.Error
		})
	} else {
		step = ValidationStep{
			Name:    "Canvas Accessibility",
			Status:  StepSkipped,
			Message: "Skipped due to connectivity issues",
		}
		if s.showProgress {
			s.printStep(step)
		}
	}
	steps = append(steps, step)

	result := s.buildResult(steps, startTime)

	// Summary
	if s.showProgress {
		s.printSummary(result)
	}

	return result
}

// ValidateQuick runs only essential configuration checks (no network calls).
// Useful for quick startup validation.
func (s *ValidationSuite) ValidateQuick() SuiteResult {
	startTime := time.Now()
	steps := make([]ValidationStep, 0, 4)

	if s.showProgress {
		s.printHeader("Quick Configuration Check")
	}

	// Only run configuration checks (no network)
	checks := []struct {
		name string
		fn   func() ValidationResult
	}{
		{"Environment File", s.configValidator.CheckEnvFile},
		{"Server URL Configuration", s.configValidator.CheckServerURL},
		{"Canvas ID Configuration", s.configValidator.CheckCanvasID},
		{"Authentication Credentials", s.configValidator.CheckAuthCredentials},
	}

	for _, check := range checks {
		step := s.runStep(check.name, func() (bool, string, error) {
			result := check.fn()
			return result.Valid, result.Message, result.Error
		})
		steps = append(steps, step)
		if s.failFast && step.Status == StepFailed {
			break
		}
	}

	result := s.buildResult(steps, startTime)

	if s.showProgress {
		s.printSummary(result)
	}

	return result
}

// runStep executes a validation step with timing and progress output.
func (s *ValidationSuite) runStep(name string, fn func() (bool, string, error)) ValidationStep {
	step := ValidationStep{Name: name, Status: StepRunning}

	if s.showProgress {
		s.printStepStart(name)
	}

	startTime := time.Now()
	passed, message, err := fn()
	step.Latency = time.Since(startTime)
	step.Message = message
	step.Error = err

	if passed {
		step.Status = StepPassed
	} else {
		step.Status = StepFailed
	}

	if s.showProgress {
		s.printStep(step)
	}

	return step
}

// hasAllPassed checks if all steps have passed.
func (s *ValidationSuite) hasAllPassed(steps []ValidationStep) bool {
	for _, step := range steps {
		if step.Status == StepFailed {
			return false
		}
	}
	return true
}

// buildResult creates a SuiteResult from completed steps.
func (s *ValidationSuite) buildResult(steps []ValidationStep, startTime time.Time) SuiteResult {
	result := SuiteResult{
		Steps:      steps,
		TotalSteps: len(steps),
		Duration:   time.Since(startTime),
		Success:    true,
	}

	for _, step := range steps {
		switch step.Status {
		case StepPassed:
			result.PassedSteps++
		case StepFailed:
			result.FailedSteps++
			result.Success = false
		case StepWarning:
			result.Warnings++
		}
	}

	return result
}

// printHeader prints a validation header.
func (s *ValidationSuite) printHeader(title string) {
	fmt.Fprintln(s.output)
	headerColor := color.New(color.FgCyan, color.Bold)
	headerColor.Fprintf(s.output, "━━━ %s ━━━\n", title)
	fmt.Fprintln(s.output)
}

// printStepStart prints the step name before execution (for real-time feedback).
func (s *ValidationSuite) printStepStart(name string) {
	fmt.Fprintf(s.output, "  ◌ %s...", name)
}

// printStep prints a completed validation step with status indicator.
func (s *ValidationSuite) printStep(step ValidationStep) {
	var icon string
	var clr *color.Color

	switch step.Status {
	case StepPassed:
		icon = "✓"
		clr = color.New(color.FgGreen)
	case StepFailed:
		icon = "✗"
		clr = color.New(color.FgRed)
	case StepWarning:
		icon = "!"
		clr = color.New(color.FgYellow)
	case StepSkipped:
		icon = "○"
		clr = color.New(color.FgHiBlack)
	default:
		icon = "?"
		clr = color.New(color.FgWhite)
	}

	// Clear the "running" line and print result
	fmt.Fprintf(s.output, "\r")
	clr.Fprintf(s.output, "  %s %s", icon, step.Name)

	// Add message if present
	if step.Message != "" {
		dim := color.New(color.FgHiBlack)
		dim.Fprintf(s.output, " - %s", step.Message)
	}

	fmt.Fprintln(s.output)

	// Print error details for failed steps
	if step.Status == StepFailed && step.Error != nil {
		errColor := color.New(color.FgRed)
		errColor.Fprintf(s.output, "    └─ %s\n", step.Error.Error())
	}
}

// printSummary prints the validation summary.
func (s *ValidationSuite) printSummary(result SuiteResult) {
	fmt.Fprintln(s.output)

	if result.Success {
		successColor := color.New(color.FgGreen, color.Bold)
		successColor.Fprintf(s.output, "━━━ Validation Passed ")
		color.New(color.FgHiBlack).Fprintf(s.output, "(%d/%d checks passed in %v)",
			result.PassedSteps, result.TotalSteps, result.Duration.Round(time.Millisecond))
		successColor.Fprintln(s.output, " ━━━")
	} else {
		failColor := color.New(color.FgRed, color.Bold)
		failColor.Fprintf(s.output, "━━━ Validation Failed ")
		color.New(color.FgHiBlack).Fprintf(s.output, "(%d passed, %d failed)",
			result.PassedSteps, result.FailedSteps)
		failColor.Fprintln(s.output, " ━━━")
	}

	fmt.Fprintln(s.output)
}

// GetErrors returns all errors from failed steps.
func (r SuiteResult) GetErrors() []error {
	errors := make([]error, 0)
	for _, step := range r.Steps {
		if step.Error != nil {
			errors = append(errors, step.Error)
		}
	}
	return errors
}

// GetFirstError returns the first error from failed steps, or nil if all passed.
func (r SuiteResult) GetFirstError() error {
	for _, step := range r.Steps {
		if step.Error != nil {
			return step.Error
		}
	}
	return nil
}

// Summary returns a human-readable summary string.
func (r SuiteResult) Summary() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Validation %s: ", map[bool]string{true: "Passed", false: "Failed"}[r.Success]))
	sb.WriteString(fmt.Sprintf("%d/%d checks passed", r.PassedSteps, r.TotalSteps))
	if r.FailedSteps > 0 {
		sb.WriteString(fmt.Sprintf(", %d failed", r.FailedSteps))
	}
	if r.Warnings > 0 {
		sb.WriteString(fmt.Sprintf(", %d warnings", r.Warnings))
	}
	sb.WriteString(fmt.Sprintf(" (took %v)", r.Duration.Round(time.Millisecond)))
	return sb.String()
}
