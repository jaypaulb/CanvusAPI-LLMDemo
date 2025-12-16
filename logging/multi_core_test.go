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

func TestNewMultiCore_CreatesFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	core, err := NewMultiCore(zapcore.InfoLevel, logPath, true)
	if err != nil {
		t.Fatalf("NewMultiCore failed: %v", err)
	}

	// Verify core is not nil
	if core == nil {
		t.Fatal("expected non-nil core")
	}

	// Verify file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Fatalf("expected log file to be created at %s", logPath)
	}
}

func TestNewMultiCore_InvalidPath(t *testing.T) {
	// Use an invalid path (directory that doesn't exist and can't be created)
	invalidPath := "/nonexistent/deeply/nested/path/test.log"

	_, err := NewMultiCore(zapcore.InfoLevel, invalidPath, true)
	if err == nil {
		t.Fatal("expected error for invalid path, got nil")
	}
}

func TestNewMultiCoreWithWriters_Development(t *testing.T) {
	var consoleBuf, fileBuf bytes.Buffer

	core := NewMultiCoreWithWriters(
		zapcore.InfoLevel,
		zapcore.AddSync(&consoleBuf),
		zapcore.AddSync(&fileBuf),
		true, // development mode
	)

	logger := zap.New(core)
	logger.Info("test message", zap.String("key", "value"))
	logger.Sync()

	// Console should have human-readable format (not JSON)
	consoleOutput := consoleBuf.String()
	if consoleOutput == "" {
		t.Fatal("expected console output, got empty string")
	}

	// File should have JSON format
	fileOutput := fileBuf.String()
	if fileOutput == "" {
		t.Fatal("expected file output, got empty string")
	}

	// Verify file output is valid JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(fileOutput)), &jsonData); err != nil {
		t.Fatalf("expected file output to be JSON, got: %s, error: %v", fileOutput, err)
	}

	// Verify JSON has expected fields
	if _, ok := jsonData[FieldMessage]; !ok {
		t.Errorf("expected JSON to have %q field", FieldMessage)
	}
	if _, ok := jsonData[FieldLevel]; !ok {
		t.Errorf("expected JSON to have %q field", FieldLevel)
	}
}

func TestNewMultiCoreWithWriters_Production(t *testing.T) {
	var consoleBuf, fileBuf bytes.Buffer

	core := NewMultiCoreWithWriters(
		zapcore.InfoLevel,
		zapcore.AddSync(&consoleBuf),
		zapcore.AddSync(&fileBuf),
		false, // production mode
	)

	logger := zap.New(core)
	logger.Info("test message", zap.String("key", "value"))
	logger.Sync()

	// Both console and file should have JSON format in production
	consoleOutput := consoleBuf.String()
	fileOutput := fileBuf.String()

	// Verify console output is valid JSON
	var consoleJSON map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(consoleOutput)), &consoleJSON); err != nil {
		t.Fatalf("expected console output to be JSON in production mode, got: %s", consoleOutput)
	}

	// Verify file output is valid JSON
	var fileJSON map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(fileOutput)), &fileJSON); err != nil {
		t.Fatalf("expected file output to be JSON, got: %s", fileOutput)
	}
}

func TestNewMultiCoreWithWriters_LevelFiltering(t *testing.T) {
	var consoleBuf, fileBuf bytes.Buffer

	core := NewMultiCoreWithWriters(
		zapcore.WarnLevel, // Only warn and above
		zapcore.AddSync(&consoleBuf),
		zapcore.AddSync(&fileBuf),
		true,
	)

	logger := zap.New(core)

	// Log at Info level - should be filtered out
	logger.Info("info message")
	logger.Sync()

	if consoleBuf.Len() > 0 {
		t.Errorf("expected info message to be filtered, got: %s", consoleBuf.String())
	}
	if fileBuf.Len() > 0 {
		t.Errorf("expected info message to be filtered from file, got: %s", fileBuf.String())
	}

	// Log at Warn level - should appear
	logger.Warn("warn message")
	logger.Sync()

	if consoleBuf.Len() == 0 {
		t.Error("expected warn message in console output")
	}
	if fileBuf.Len() == 0 {
		t.Error("expected warn message in file output")
	}
}

func TestNewMultiCore_WritesBothOutputs(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	core, err := NewMultiCore(zapcore.InfoLevel, logPath, false)
	if err != nil {
		t.Fatalf("NewMultiCore failed: %v", err)
	}

	logger := zap.New(core)
	logger.Info("test entry", zap.Int("count", 42))
	logger.Sync()

	// Read file contents
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if len(content) == 0 {
		t.Fatal("expected log file to have content")
	}

	// Verify it's valid JSON
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(content))), &jsonData); err != nil {
		t.Fatalf("expected valid JSON in log file, got: %s", string(content))
	}

	// Check for our field
	if jsonData["count"] != float64(42) {
		t.Errorf("expected count=42, got %v", jsonData["count"])
	}
}
