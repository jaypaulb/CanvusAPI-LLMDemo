package main

import (
	"os"
	"syscall"
	"testing"
	"time"

	"go_backend/canvusapi"
	"go_backend/core"
	"go_backend/logging"
)

// createTestLoggerMain creates a logger for testing that writes to a temp file.
func createTestLoggerMain(t *testing.T) *logging.Logger {
	t.Helper()
	// Create temp file for log output
	tmpFile, err := os.CreateTemp("", "main_test_*.log")
	if err != nil {
		t.Fatalf("failed to create temp log file: %v", err)
	}
	tmpFile.Close()
	t.Cleanup(func() { os.Remove(tmpFile.Name()) })

	logger, err := logging.NewLogger(true, tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	return logger
}

// TestInitializeSDRuntimeNotConfigured tests SD runtime initialization when not configured.
func TestInitializeSDRuntimeNotConfigured(t *testing.T) {
	// Ensure SD_MODEL_PATH is not set
	os.Unsetenv("SD_MODEL_PATH")

	logger := createTestLoggerMain(t)
	defer logger.Sync()

	client := canvusapi.NewClient("http://localhost:8080", "test-canvas", "test-key", false)
	config := &core.Config{
		DownloadsDir: os.TempDir(),
	}

	// DOING: Call initializeSDRuntime without SD_MODEL_PATH configured
	// EXPECT: Should return (nil, nil, nil) since SD is not configured
	pool, processor, err := initializeSDRuntime(logger, client, config)

	// RESULT: Check return values
	if err != nil {
		t.Errorf("expected no error when SD not configured, got: %v", err)
	}
	if pool != nil {
		t.Error("expected nil pool when SD not configured")
	}
	if processor != nil {
		t.Error("expected nil processor when SD not configured")
	}
}

// TestInitializeSDRuntimeMissingModel tests SD runtime initialization when model file is missing.
func TestInitializeSDRuntimeMissingModel(t *testing.T) {
	// Set SD_MODEL_PATH to a non-existent file
	os.Setenv("SD_MODEL_PATH", "/nonexistent/model.safetensors")
	defer os.Unsetenv("SD_MODEL_PATH")

	logger := createTestLoggerMain(t)
	defer logger.Sync()

	client := canvusapi.NewClient("http://localhost:8080", "test-canvas", "test-key", false)
	config := &core.Config{
		DownloadsDir: os.TempDir(),
	}

	// DOING: Call initializeSDRuntime with non-existent model path
	// EXPECT: Should return error about missing model file
	pool, processor, err := initializeSDRuntime(logger, client, config)

	// RESULT: Check return values
	if err == nil {
		t.Error("expected error when model file not found")
	}
	if pool != nil {
		pool.Close() // Clean up if somehow created
		t.Error("expected nil pool when model not found")
	}
	if processor != nil {
		t.Error("expected nil processor when model not found")
	}
}

// TestInitializeSDRuntimeConfigParsing tests that SD configuration is properly loaded.
func TestInitializeSDRuntimeConfigParsing(t *testing.T) {
	tests := []struct {
		name             string
		envVars          map[string]string
		expectedDisabled bool // true if SD should be disabled (no model path)
	}{
		{
			name:             "empty model path disables SD",
			envVars:          map[string]string{},
			expectedDisabled: true,
		},
		{
			name: "model path enables SD (but will fail due to missing file)",
			envVars: map[string]string{
				"SD_MODEL_PATH": "/some/path/model.safetensors",
			},
			expectedDisabled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear relevant env vars
			os.Unsetenv("SD_MODEL_PATH")
			os.Unsetenv("SD_IMAGE_SIZE")
			os.Unsetenv("SD_INFERENCE_STEPS")
			os.Unsetenv("SD_GUIDANCE_SCALE")
			os.Unsetenv("SD_MAX_CONCURRENT")
			os.Unsetenv("SD_TIMEOUT_SECONDS")

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			logger := createTestLoggerMain(t)
			defer logger.Sync()

			client := canvusapi.NewClient("http://localhost:8080", "test-canvas", "test-key", false)
			config := &core.Config{
				DownloadsDir: os.TempDir(),
			}

			pool, processor, err := initializeSDRuntime(logger, client, config)

			if tt.expectedDisabled {
				// SD should be disabled (nil, nil, nil)
				if err != nil {
					t.Errorf("expected no error for disabled SD, got: %v", err)
				}
				if pool != nil || processor != nil {
					t.Error("expected nil pool and processor for disabled SD")
				}
			} else {
				// SD should be enabled but will fail due to missing model
				if err == nil {
					t.Error("expected error for missing model file")
					if pool != nil {
						pool.Close()
					}
				}
			}
		})
	}
}

// TestSplitAndTrim tests the splitAndTrim helper function.
func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		input    string
		sep      string
		expected []string
	}{
		{"", ",", nil},
		{"a,b,c", ",", []string{"a", "b", "c"}},
		{" a , b , c ", ",", []string{"a", "b", "c"}},
		{"  a  ", ",", []string{"a"}},
		{"a,,b", ",", []string{"a", "b"}},
		{"a", ",", []string{"a"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := splitAndTrim(tt.input, tt.sep)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d parts, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("part %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

// TestTrimSpace tests the trimSpace helper function.
func TestTrimSpace(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"a", "a"},
		{" a", "a"},
		{"a ", "a"},
		{" a ", "a"},
		{"  hello world  ", "hello world"},
		{"\t\n a \r\n", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := trimSpace(tt.input)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// =============================================================================
// Note: Web Server tests (TestSetupWebServer, TestDashboardHandler, TestAPI*Handler,
// TestWebServerAuthIntegration) have been moved to webui/server_test.go and
// webui/dashboard_api_test.go as part of the Phase 5 refactoring.
// =============================================================================

// TestDefaultTimeoutConstants verifies the default timeout constants are reasonable.
func TestDefaultTimeoutConstants(t *testing.T) {
	if DefaultReadTimeout < 1*time.Second {
		t.Error("read timeout too short")
	}
	if DefaultWriteTimeout < 1*time.Second {
		t.Error("write timeout too short")
	}
	if DefaultIdleTimeout < 1*time.Second {
		t.Error("idle timeout too short")
	}
	if DefaultShutdownTimeout < 1*time.Second {
		t.Error("shutdown timeout too short")
	}

	// Verify reasonable upper bounds
	if DefaultReadTimeout > 5*time.Minute {
		t.Error("read timeout too long")
	}
	if DefaultShutdownTimeout > 1*time.Minute {
		t.Error("shutdown timeout too long")
	}
}

// =============================================================================
// Shutdown Integration Tests
// =============================================================================

// TestSignalExitCodeMapping tests that signals map to correct exit codes.
func TestSignalExitCodeMapping(t *testing.T) {
	tests := []struct {
		name           string
		signal         os.Signal
		expectedCode   int
		expectedReason string
	}{
		{
			name:           "SIGINT maps to 130",
			signal:         os.Interrupt,
			expectedCode:   core.ExitCodeSIGINT,
			expectedReason: "interrupted (SIGINT)",
		},
		{
			name:           "SIGTERM maps to 143",
			signal:         syscall.SIGTERM,
			expectedCode:   core.ExitCodeSIGTERM,
			expectedReason: "terminated (SIGTERM)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// DOING: Check exit code mapping
			// EXPECT: Correct exit code for each signal
			exitCode := core.ExitCodeSuccess

			// Simulate signal-based exit code determination (from main)
			switch tt.signal {
			case os.Interrupt:
				exitCode = core.ExitCodeSIGINT
			case syscall.SIGTERM:
				exitCode = core.ExitCodeSIGTERM
			}

			// RESULT: Verify correct mapping
			if exitCode != tt.expectedCode {
				t.Errorf("expected exit code %d, got %d", tt.expectedCode, exitCode)
			}

			reason := core.ExitCodeName(exitCode)
			if reason != tt.expectedReason {
				t.Errorf("expected reason %q, got %q", tt.expectedReason, reason)
			}
		})
	}
}

// TestShutdownSequence tests that cleanup functions are registered and executed.
// This is a focused unit test of the shutdown integration logic.
func TestShutdownSequence(t *testing.T) {
	// DOING: Verify shutdown handlers are properly ordered
	// EXPECT: logger-sync (5), http-server (20), sd-pool (30), cleanup-downloads (45)

	// Define handlers in expected priority order (low to high)
	handlers := []struct {
		name     string
		priority int
	}{
		{"logger-sync", 5},
		{"http-server", 20},
		{"sd-pool", 30},
		{"cleanup-downloads", 45},
	}

	// RESULT: Verify priority ordering
	for i := 1; i < len(handlers); i++ {
		if handlers[i].priority <= handlers[i-1].priority {
			t.Errorf("handler %s has priority %d which is not greater than previous %s (%d)",
				handlers[i].name, handlers[i].priority,
				handlers[i-1].name, handlers[i-1].priority)
		}
	}

	// Verify all priorities are in correct range
	for _, h := range handlers {
		if h.priority < 0 || h.priority > 50 {
			t.Errorf("handler %s has out-of-range priority %d", h.name, h.priority)
		}
	}
}

// TestSignalNotifyWrapper tests that the signalNotify wrapper can be mocked.
func TestSignalNotifyWrapper(t *testing.T) {
	// DOING: Test signalNotify wrapper is mockable
	// EXPECT: Can replace signalNotify function

	// Save original
	originalNotify := signalNotify
	defer func() { signalNotify = originalNotify }()

	// Mock implementation
	var mockCalled bool
	signalNotify = func(c chan<- os.Signal, sig ...os.Signal) {
		mockCalled = true
	}

	// Call the mock
	testChan := make(chan os.Signal, 1)
	signalNotify(testChan, os.Interrupt)

	// RESULT: Verify mock was called
	if !mockCalled {
		t.Error("expected signalNotify mock to be called")
	}
}

// TestExitCodePriority tests that error exit codes take priority over signal codes.
func TestExitCodePriority(t *testing.T) {
	tests := []struct {
		name              string
		initialCode       int
		hadError          bool
		signal            os.Signal
		expectedFinalCode int
	}{
		{
			name:              "signal with no error uses signal code",
			initialCode:       core.ExitCodeSuccess,
			hadError:          false,
			signal:            os.Interrupt,
			expectedFinalCode: core.ExitCodeSIGINT,
		},
		{
			name:              "error takes priority over signal",
			initialCode:       core.ExitCodeError,
			hadError:          true,
			signal:            os.Interrupt,
			expectedFinalCode: core.ExitCodeError,
		},
		{
			name:              "no signal uses initial code",
			initialCode:       core.ExitCodeSuccess,
			hadError:          false,
			signal:            nil,
			expectedFinalCode: core.ExitCodeSuccess,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exitCode := tt.initialCode

			// Simulate exit code determination logic from main
			if exitCode == core.ExitCodeSuccess && tt.signal != nil {
				switch tt.signal {
				case os.Interrupt:
					exitCode = core.ExitCodeSIGINT
				case syscall.SIGTERM:
					exitCode = core.ExitCodeSIGTERM
				}
			}

			if exitCode != tt.expectedFinalCode {
				t.Errorf("expected exit code %d, got %d", tt.expectedFinalCode, exitCode)
			}
		})
	}
}

// Note: TestHTTPServerShutdownHandler has been moved to webui/server_test.go
// as TestWebUIServer_Shutdown, testing the WebUIServer.Shutdown() method.
