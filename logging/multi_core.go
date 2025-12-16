package logging

import (
	"os"

	"go.uber.org/zap/zapcore"
)

// NewMultiCore creates a zapcore.Core that tees output to both console and file.
// This is a molecule that composes the encoder config atoms from encoder_config.go.
//
// Parameters:
//   - level: The minimum log level for both outputs
//   - filePath: Path to the log file (will be created/appended)
//   - isDev: When true, console uses human-readable format; when false, both use JSON
//
// The file output always uses JSON encoding for structured log processing.
// The console output uses:
//   - Development mode (isDev=true): colored, human-readable format
//   - Production mode (isDev=false): JSON format for consistency
//
// Returns the combined core and any error from file creation.
//
// Example:
//
//	core, err := NewMultiCore(zapcore.InfoLevel, "app.log", true)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	logger := zap.New(core)
func NewMultiCore(level zapcore.Level, filePath string, isDev bool) (zapcore.Core, error) {
	// Create file writer (append mode, create if not exists)
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	// File always uses JSON encoder for structured logging
	fileEncoder := zapcore.NewJSONEncoder(NewEncoderConfig())
	fileCore := zapcore.NewCore(
		fileEncoder,
		zapcore.AddSync(file),
		level,
	)

	// Console encoder depends on mode
	var consoleEncoder zapcore.Encoder
	if isDev {
		// Development: human-readable with colors
		consoleEncoder = zapcore.NewConsoleEncoder(NewConsoleEncoderConfig())
	} else {
		// Production: JSON for consistency
		consoleEncoder = zapcore.NewJSONEncoder(NewEncoderConfig())
	}

	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stdout),
		level,
	)

	// Tee both cores together
	return zapcore.NewTee(consoleCore, fileCore), nil
}

// NewMultiCoreWithWriters creates a zapcore.Core that tees output to provided writers.
// This variant allows for custom writers, useful for testing or special output destinations.
//
// Parameters:
//   - level: The minimum log level for both outputs
//   - consoleWriter: Writer for console output (typically os.Stdout)
//   - fileWriter: Writer for file output
//   - isDev: When true, console uses human-readable format; when false, both use JSON
//
// Example:
//
//	var buf bytes.Buffer
//	core := NewMultiCoreWithWriters(zapcore.DebugLevel, os.Stdout, &buf, true)
//	logger := zap.New(core)
func NewMultiCoreWithWriters(level zapcore.Level, consoleWriter, fileWriter zapcore.WriteSyncer, isDev bool) zapcore.Core {
	// File always uses JSON encoder
	fileEncoder := zapcore.NewJSONEncoder(NewEncoderConfig())
	fileCore := zapcore.NewCore(
		fileEncoder,
		fileWriter,
		level,
	)

	// Console encoder depends on mode
	var consoleEncoder zapcore.Encoder
	if isDev {
		consoleEncoder = zapcore.NewConsoleEncoder(NewConsoleEncoderConfig())
	} else {
		consoleEncoder = zapcore.NewJSONEncoder(NewEncoderConfig())
	}

	consoleCore := zapcore.NewCore(
		consoleEncoder,
		consoleWriter,
		level,
	)

	return zapcore.NewTee(consoleCore, fileCore)
}
