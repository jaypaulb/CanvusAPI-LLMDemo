package logging

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultFileWriterConfig(t *testing.T) {
	config := DefaultFileWriterConfig()

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"MaxSizeMB", config.MaxSizeMB, DefaultMaxSizeMB},
		{"MaxBackups", config.MaxBackups, DefaultMaxBackups},
		{"MaxAgeDays", config.MaxAgeDays, DefaultMaxAgeDays},
		{"Compress", config.Compress, DefaultCompress},
		{"LocalTime", config.LocalTime, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("DefaultFileWriterConfig().%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestNewFileWriter(t *testing.T) {
	// Create temp directory for test log file
	tmpDir, err := os.MkdirTemp("", "filewriter_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "test.log")

	// Create file writer
	writer := NewFileWriter(logPath)

	if writer == nil {
		t.Fatal("NewFileWriter returned nil")
	}

	// Test that we can write to it
	testMessage := []byte("test log message\n")
	n, err := writer.Write(testMessage)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(testMessage) {
		t.Errorf("Write returned %d bytes, expected %d", n, len(testMessage))
	}

	// Sync to ensure data is flushed
	err = writer.Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	// Verify file was created and contains data
	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Errorf("Failed to read log file: %v", err)
	}
	if string(content) != string(testMessage) {
		t.Errorf("File content = %q, want %q", string(content), string(testMessage))
	}
}

func TestNewFileWriterWithConfig(t *testing.T) {
	// Create temp directory for test log file
	tmpDir, err := os.MkdirTemp("", "filewriter_config_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, "custom.log")

	// Custom configuration
	config := FileWriterConfig{
		MaxSizeMB:  50,
		MaxBackups: 3,
		MaxAgeDays: 7,
		Compress:   false,
		LocalTime:  true,
	}

	// Create file writer with custom config
	writer := NewFileWriterWithConfig(logPath, config)

	if writer == nil {
		t.Fatal("NewFileWriterWithConfig returned nil")
	}

	// Test writing
	testMessage := []byte("custom config test\n")
	n, err := writer.Write(testMessage)
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n != len(testMessage) {
		t.Errorf("Write returned %d bytes, expected %d", n, len(testMessage))
	}

	// Sync and verify
	err = writer.Sync()
	if err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	content, err := os.ReadFile(logPath)
	if err != nil {
		t.Errorf("Failed to read log file: %v", err)
	}
	if string(content) != string(testMessage) {
		t.Errorf("File content = %q, want %q", string(content), string(testMessage))
	}
}

func TestApplyFileWriterDefaults(t *testing.T) {
	tests := []struct {
		name     string
		input    FileWriterConfig
		expected FileWriterConfig
	}{
		{
			name:  "all zero values get defaults",
			input: FileWriterConfig{},
			expected: FileWriterConfig{
				MaxSizeMB:  DefaultMaxSizeMB,
				MaxBackups: DefaultMaxBackups,
				MaxAgeDays: DefaultMaxAgeDays,
				Compress:   false, // zero value, not changed
				LocalTime:  false,
			},
		},
		{
			name: "custom values preserved",
			input: FileWriterConfig{
				MaxSizeMB:  50,
				MaxBackups: 3,
				MaxAgeDays: 7,
				Compress:   true,
				LocalTime:  true,
			},
			expected: FileWriterConfig{
				MaxSizeMB:  50,
				MaxBackups: 3,
				MaxAgeDays: 7,
				Compress:   true,
				LocalTime:  true,
			},
		},
		{
			name: "partial custom values",
			input: FileWriterConfig{
				MaxSizeMB: 25,
				Compress:  true,
			},
			expected: FileWriterConfig{
				MaxSizeMB:  25,
				MaxBackups: DefaultMaxBackups,
				MaxAgeDays: DefaultMaxAgeDays,
				Compress:   true,
				LocalTime:  false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyFileWriterDefaults(tt.input)

			if result.MaxSizeMB != tt.expected.MaxSizeMB {
				t.Errorf("MaxSizeMB = %d, want %d", result.MaxSizeMB, tt.expected.MaxSizeMB)
			}
			if result.MaxBackups != tt.expected.MaxBackups {
				t.Errorf("MaxBackups = %d, want %d", result.MaxBackups, tt.expected.MaxBackups)
			}
			if result.MaxAgeDays != tt.expected.MaxAgeDays {
				t.Errorf("MaxAgeDays = %d, want %d", result.MaxAgeDays, tt.expected.MaxAgeDays)
			}
			if result.Compress != tt.expected.Compress {
				t.Errorf("Compress = %v, want %v", result.Compress, tt.expected.Compress)
			}
			if result.LocalTime != tt.expected.LocalTime {
				t.Errorf("LocalTime = %v, want %v", result.LocalTime, tt.expected.LocalTime)
			}
		})
	}
}
