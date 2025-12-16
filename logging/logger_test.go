package logging

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// syncLogger calls Sync() and logs any non-critical errors as warnings.
// Syncing stdout/stderr returns "invalid argument" on Linux, which is expected.
func syncLogger(t testing.TB, logger *Logger) {
	t.Helper()
	if err := logger.Sync(); err != nil {
		// Sync errors on stdout are expected on Linux - log as info, not error
		if strings.Contains(err.Error(), "invalid argument") {
			// This is expected behavior on Linux when syncing stdout
			return
		}
		t.Logf("Sync() warning: %v", err)
	}
}

func TestNewLogger_Development(t *testing.T) {
	// Create temp directory for log file
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_dev.log")

	logger, err := NewLogger(true, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	// Verify development mode
	if !logger.IsDevelopment() {
		t.Error("IsDevelopment() = false, want true")
	}

	// Verify log file path
	if logger.LogFilePath() != logPath {
		t.Errorf("LogFilePath() = %q, want %q", logger.LogFilePath(), logPath)
	}

	// Write a test log entry
	logger.Info("test message", zap.String("key", "value"))

	// Sync to ensure file is written
	syncLogger(t, logger)

	// Check log file was created and has content
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("log file stat error: %v", err)
	}
	if info.Size() == 0 {
		t.Error("log file is empty, expected content")
	}
}

func TestNewLogger_Production(t *testing.T) {
	// Create temp directory for log file
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_prod.log")

	logger, err := NewLogger(false, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	// Verify production mode
	if logger.IsDevelopment() {
		t.Error("IsDevelopment() = true, want false")
	}

	// Write a test log entry
	logger.Info("production message", zap.Int("count", 42))

	// Sync
	syncLogger(t, logger)

	// Read and verify JSON format in file
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	// Production mode should produce JSON
	var logEntry map[string]interface{}
	if err := json.Unmarshal(content, &logEntry); err != nil {
		t.Errorf("log file content is not valid JSON: %v\nContent: %s", err, content)
	}

	// Verify required fields
	if _, ok := logEntry["message"]; !ok {
		t.Error("log entry missing 'message' field")
	}
	if _, ok := logEntry["level"]; !ok {
		t.Error("log entry missing 'level' field")
	}
}

func TestNewLogger_InvalidPath(t *testing.T) {
	// Try to create logger with invalid path
	_, err := NewLogger(true, "/nonexistent/directory/that/does/not/exist/test.log")
	if err == nil {
		t.Error("NewLogger() with invalid path should return error")
	}
}

func TestNewLoggerWithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_config.log")

	config := FileWriterConfig{
		MaxSizeMB:  50,
		MaxBackups: 3,
		MaxAgeDays: 7,
		Compress:   false,
	}

	logger, err := NewLoggerWithConfig(true, logPath, config)
	if err != nil {
		t.Fatalf("NewLoggerWithConfig() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	// Write a log entry to verify it works
	logger.Info("config test message")

	syncLogger(t, logger)

	// Verify file was created
	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("log file stat error: %v", err)
	}
	if info.Size() == 0 {
		t.Error("log file is empty, expected content")
	}
}

func TestLogger_AllLogLevels(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_levels.log")

	logger, err := NewLogger(true, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	// Test all log levels (except Fatal and Panic which would terminate)
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	syncLogger(t, logger)

	// Verify file has content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	// Should have at least 4 log entries
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 4 {
		t.Errorf("expected at least 4 log entries, got %d", len(lines))
	}
}

func TestLogger_SugaredMethods(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_sugar.log")

	logger, err := NewLogger(true, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	// Test sugared methods
	logger.Debugw("debug with fields", "key1", "value1")
	logger.Infow("info with fields", "key2", "value2", "count", 42)
	logger.Warnw("warn with fields", "warning", "something")
	logger.Errorw("error with fields", "error", "test error")

	// Test formatted methods
	logger.Debugf("debug formatted: %s", "test")
	logger.Infof("info formatted: %d", 123)
	logger.Warnf("warn formatted: %v", true)
	logger.Errorf("error formatted: %s", "oops")

	syncLogger(t, logger)

	// Verify file has content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	// Should have 8 log entries
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) < 8 {
		t.Errorf("expected at least 8 log entries, got %d", len(lines))
	}
}

func TestLogger_SensitiveDataRedaction_StructuredFields(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_redact_struct.log")

	logger, err := NewLogger(false, logPath) // Production mode for JSON
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	// Log with sensitive field name
	logger.Info("api call",
		zap.String("OPENAI_API_KEY", "sk-secret123456789abcdefghij"),
		zap.String("user", "john"))

	// Log with sensitive pattern in value
	logger.Info("config loaded",
		zap.String("config", "key=sk-secret123456789abcdefghij"))

	syncLogger(t, logger)

	// Read and verify content
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	contentStr := string(content)

	// Verify sensitive data is redacted
	if strings.Contains(contentStr, "sk-secret123456789abcdefghij") {
		t.Error("log file contains unredacted API key")
	}

	// Verify redaction placeholder appears
	if !strings.Contains(contentStr, RedactedPlaceholder) {
		t.Error("log file does not contain redaction placeholder")
	}

	// Verify non-sensitive data is preserved
	if !strings.Contains(contentStr, "john") {
		t.Error("log file missing non-sensitive data 'john'")
	}
}

func TestLogger_SensitiveDataRedaction_SugaredFields(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_redact_sugar.log")

	logger, err := NewLogger(false, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	// Log with sensitive field using sugared method
	logger.Infow("api configured",
		"API_KEY", "sk-anothersecretkey12345678901234567890",
		"endpoint", "http://localhost:1234")

	syncLogger(t, logger)

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	contentStr := string(content)

	// Verify sensitive data is redacted
	if strings.Contains(contentStr, "sk-anothersecretkey12345678901234567890") {
		t.Error("log file contains unredacted API key in sugared log")
	}

	// Verify non-sensitive data is preserved
	if !strings.Contains(contentStr, "http://localhost:1234") {
		t.Error("log file missing non-sensitive endpoint URL")
	}
}

func TestLogger_With(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_with.log")

	logger, err := NewLogger(false, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	// Create child logger with context
	requestLogger := logger.With(
		zap.String("request_id", "abc123"),
		zap.String("user_id", "user456"))

	requestLogger.Info("handling request")
	requestLogger.Info("request complete")

	syncLogger(t, logger)

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	contentStr := string(content)

	// Verify context fields are present in both log entries
	if strings.Count(contentStr, "abc123") != 2 {
		t.Error("expected request_id to appear in both log entries")
	}
	if strings.Count(contentStr, "user456") != 2 {
		t.Error("expected user_id to appear in both log entries")
	}
}

func TestLogger_With_SensitiveDataRedaction(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_with_redact.log")

	logger, err := NewLogger(false, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	// Create child logger with sensitive context (should be redacted)
	sensitiveLogger := logger.With(
		zap.String("API_KEY", "sk-supersecret123456789012345678901234"))

	sensitiveLogger.Info("using sensitive context")

	syncLogger(t, logger)

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	contentStr := string(content)

	if strings.Contains(contentStr, "sk-supersecret") {
		t.Error("With() should redact sensitive context fields")
	}
}

func TestLogger_Named(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_named.log")

	logger, err := NewLogger(false, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	// Create named sub-loggers
	httpLogger := logger.Named("http")
	dbLogger := logger.Named("database")

	httpLogger.Info("handling http request")
	dbLogger.Info("executing query")

	syncLogger(t, logger)

	// File should contain the named loggers
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	contentStr := string(content)

	if !strings.Contains(contentStr, "http") {
		t.Error("log file missing 'http' logger name")
	}
	if !strings.Contains(contentStr, "database") {
		t.Error("log file missing 'database' logger name")
	}
}

func TestLogger_Sync_NilLogger(t *testing.T) {
	var nilLogger *Logger
	// Should not panic
	err := nilLogger.Sync()
	if err != nil {
		t.Errorf("Sync() on nil logger should return nil, got: %v", err)
	}
}

func TestLogger_Sugar_Accessor(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_sugar_accessor.log")

	logger, err := NewLogger(true, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	sugar := logger.Sugar()
	if sugar == nil {
		t.Error("Sugar() returned nil")
	}
}

func TestLogger_Zap_Accessor(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_zap_accessor.log")

	logger, err := NewLogger(true, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	zapLogger := logger.Zap()
	if zapLogger == nil {
		t.Error("Zap() returned nil")
	}
}

func TestRedactFields_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_redact_empty.log")

	logger, err := NewLogger(true, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	// Should handle empty fields without issue
	fields := logger.redactFields(nil)
	if fields != nil {
		t.Errorf("redactFields(nil) should return nil, got %v", fields)
	}

	fields = logger.redactFields([]zap.Field{})
	if len(fields) != 0 {
		t.Errorf("redactFields([]) should return empty slice, got %d items", len(fields))
	}
}

func TestRedactKeysAndValues_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_redact_kv_empty.log")

	logger, err := NewLogger(true, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	// Should handle empty keys and values without issue
	kv := logger.redactKeysAndValues(nil)
	if kv != nil {
		t.Errorf("redactKeysAndValues(nil) should return nil, got %v", kv)
	}

	kv = logger.redactKeysAndValues([]interface{}{})
	if len(kv) != 0 {
		t.Errorf("redactKeysAndValues([]) should return empty slice, got %d items", len(kv))
	}
}

func TestRedactKeysAndValues_OddLength(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_redact_kv_odd.log")

	logger, err := NewLogger(true, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	// Should handle odd-length slice (last item has no value)
	kv := logger.redactKeysAndValues([]interface{}{"key1", "value1", "orphan"})
	if len(kv) != 3 {
		t.Errorf("expected 3 items, got %d", len(kv))
	}
}

func TestRedactKeysAndValues_NonStringKey(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_redact_kv_nonstring.log")

	logger, err := NewLogger(true, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	// Should handle non-string keys gracefully
	kv := logger.redactKeysAndValues([]interface{}{123, "value1", "key2", "value2"})
	if len(kv) != 4 {
		t.Errorf("expected 4 items, got %d", len(kv))
	}
	// Value should be unchanged since key is not a string
	if kv[1] != "value1" {
		t.Errorf("expected 'value1', got %v", kv[1])
	}
}

// TestLogger_Integration tests the full logging pipeline
func TestLogger_Integration(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_integration.log")

	// Create production logger
	logger, err := NewLogger(false, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}

	// Create named child with context
	requestLogger := logger.Named("api").With(
		zap.String("request_id", "req-123"),
	)

	// Log various messages
	requestLogger.Info("request started",
		zap.String("method", "GET"),
		zap.String("path", "/api/health"))

	requestLogger.Warn("slow response",
		zap.Int("duration_ms", 1500))

	requestLogger.Error("request failed",
		zap.String("error", "connection timeout"),
		zap.String("OPENAI_API_KEY", "sk-shouldberedacted123456789012345"))

	// Sync and close
	syncLogger(t, logger)

	// Read and parse log entries
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	lines := bytes.Split(bytes.TrimSpace(content), []byte("\n"))
	if len(lines) != 3 {
		t.Errorf("expected 3 log entries, got %d", len(lines))
	}

	// Parse each entry and verify structure
	for i, line := range lines {
		var entry map[string]interface{}
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Errorf("line %d: invalid JSON: %v", i, err)
			continue
		}

		// Verify required fields
		if _, ok := entry["message"]; !ok {
			t.Errorf("line %d: missing 'message' field", i)
		}
		if _, ok := entry["level"]; !ok {
			t.Errorf("line %d: missing 'level' field", i)
		}
		if _, ok := entry["timestamp"]; !ok {
			t.Errorf("line %d: missing 'timestamp' field", i)
		}

		// Verify request_id context is present in all entries
		if _, ok := entry["request_id"]; !ok {
			t.Errorf("line %d: missing 'request_id' context field", i)
		}
	}

	// Verify sensitive data was redacted
	contentStr := string(content)
	if strings.Contains(contentStr, "sk-shouldberedacted") {
		t.Error("log file contains unredacted API key")
	}
	if !strings.Contains(contentStr, RedactedPlaceholder) {
		t.Error("log file missing redaction placeholder")
	}
}

// Benchmark tests

func BenchmarkLogger_Info(b *testing.B) {
	tmpDir := b.TempDir()
	logPath := filepath.Join(tmpDir, "bench.log")

	logger, err := NewLogger(false, logPath)
	if err != nil {
		b.Fatalf("NewLogger() error: %v", err)
	}
	defer logger.Sync()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			zap.String("iteration", "test"),
			zap.Int("count", i))
	}
}

func BenchmarkLogger_Infow(b *testing.B) {
	tmpDir := b.TempDir()
	logPath := filepath.Join(tmpDir, "bench_sugar.log")

	logger, err := NewLogger(false, logPath)
	if err != nil {
		b.Fatalf("NewLogger() error: %v", err)
	}
	defer logger.Sync()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Infow("benchmark message",
			"iteration", "test",
			"count", i)
	}
}

func BenchmarkLogger_WithRedaction(b *testing.B) {
	tmpDir := b.TempDir()
	logPath := filepath.Join(tmpDir, "bench_redact.log")

	logger, err := NewLogger(false, logPath)
	if err != nil {
		b.Fatalf("NewLogger() error: %v", err)
	}
	defer logger.Sync()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark with sensitive data",
			zap.String("API_KEY", "sk-test123456789012345678901234567890"),
			zap.String("normal", "value"))
	}
}

// TestLogger_WithOptions verifies WithOptions creates proper child logger
func TestLogger_WithOptions(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_with_options.log")

	logger, err := NewLogger(true, logPath)
	if err != nil {
		t.Fatalf("NewLogger() returned error: %v", err)
	}
	defer syncLogger(t, logger)

	// Create child with additional stacktrace option
	childLogger := logger.WithOptions(zap.AddStacktrace(zapcore.ErrorLevel))

	// Verify child is a valid logger
	if childLogger == nil {
		t.Error("WithOptions() returned nil")
	}

	// Log an error to trigger stacktrace
	childLogger.Error("error with stack")

	syncLogger(t, logger)

	// Verify log file was written
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}

	if len(content) == 0 {
		t.Error("log file is empty after WithOptions logging")
	}
}
