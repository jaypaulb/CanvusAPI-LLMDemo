package logging

import (
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Default file writer configuration values
const (
	// DefaultMaxSizeMB is the maximum size in megabytes before rotation
	DefaultMaxSizeMB = 100

	// DefaultMaxBackups is the number of old log files to retain
	DefaultMaxBackups = 5

	// DefaultMaxAgeDays is the maximum number of days to retain old log files
	DefaultMaxAgeDays = 30

	// DefaultCompress enables gzip compression of rotated files
	DefaultCompress = true
)

// FileWriterConfig holds configuration for the file writer with rotation.
// All fields are optional - zero values will use defaults.
type FileWriterConfig struct {
	// MaxSizeMB is the maximum size in megabytes of the log file before rotation.
	// Default: 100 MB
	MaxSizeMB int

	// MaxBackups is the maximum number of old log files to retain.
	// Default: 5 files
	MaxBackups int

	// MaxAgeDays is the maximum number of days to retain old log files.
	// Older files are deleted during rotation.
	// Default: 30 days
	MaxAgeDays int

	// Compress determines if rotated log files should be compressed using gzip.
	// Default: true
	Compress bool

	// LocalTime determines if the timestamps in backup file names use local time.
	// Default: false (uses UTC)
	LocalTime bool
}

// DefaultFileWriterConfig returns a FileWriterConfig with default values.
// This is a pure function with no side effects.
func DefaultFileWriterConfig() FileWriterConfig {
	return FileWriterConfig{
		MaxSizeMB:  DefaultMaxSizeMB,
		MaxBackups: DefaultMaxBackups,
		MaxAgeDays: DefaultMaxAgeDays,
		Compress:   DefaultCompress,
		LocalTime:  false,
	}
}

// NewFileWriter creates a zapcore.WriteSyncer that writes to a file with automatic rotation.
// Uses default configuration: 100MB max size, 5 backups, 30 days retention, compression enabled.
//
// This is a molecule that composes lumberjack.Logger into a zapcore.WriteSyncer.
//
// The returned WriteSyncer implements automatic log rotation based on size and age.
// Rotated files are named with a timestamp suffix and optionally compressed.
//
// Example:
//
//	writer := NewFileWriter("/var/log/app.log")
//	core := zapcore.NewCore(encoder, writer, level)
func NewFileWriter(path string) zapcore.WriteSyncer {
	return NewFileWriterWithConfig(path, DefaultFileWriterConfig())
}

// NewFileWriterWithConfig creates a zapcore.WriteSyncer with custom configuration.
//
// This is a molecule that composes lumberjack.Logger into a zapcore.WriteSyncer.
// It applies defaults for any zero-value fields in the config.
//
// Example:
//
//	config := FileWriterConfig{
//	    MaxSizeMB:  50,
//	    MaxBackups: 3,
//	    MaxAgeDays: 7,
//	    Compress:   true,
//	}
//	writer := NewFileWriterWithConfig("/var/log/app.log", config)
func NewFileWriterWithConfig(path string, config FileWriterConfig) zapcore.WriteSyncer {
	// Apply defaults for zero values
	cfg := applyFileWriterDefaults(config)

	logger := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   cfg.Compress,
		LocalTime:  cfg.LocalTime,
	}

	return zapcore.AddSync(logger)
}

// applyFileWriterDefaults fills in zero values with defaults.
// This is a pure function with no side effects.
func applyFileWriterDefaults(config FileWriterConfig) FileWriterConfig {
	result := config

	if result.MaxSizeMB == 0 {
		result.MaxSizeMB = DefaultMaxSizeMB
	}
	if result.MaxBackups == 0 {
		result.MaxBackups = DefaultMaxBackups
	}
	if result.MaxAgeDays == 0 {
		result.MaxAgeDays = DefaultMaxAgeDays
	}
	// Note: Compress default is true, but Go's zero value for bool is false.
	// We can't distinguish "explicitly set to false" from "not set".
	// The config struct uses explicit defaults via DefaultFileWriterConfig() instead.

	return result
}
