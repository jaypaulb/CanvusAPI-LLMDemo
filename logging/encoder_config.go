package logging

import (
	"time"

	"go.uber.org/zap/zapcore"
)

// Standard field names for structured logging.
// These constants define the JSON keys used in log output.
const (
	// FieldTimestamp is the key for the log entry timestamp
	FieldTimestamp = "timestamp"

	// FieldLevel is the key for the log level (debug, info, warn, error, fatal)
	FieldLevel = "level"

	// FieldSource is the key for the source file and line number
	FieldSource = "source"

	// FieldMessage is the key for the log message
	FieldMessage = "message"

	// FieldStacktrace is the key for stack traces (on error/fatal)
	FieldStacktrace = "stacktrace"

	// FieldCaller is the key for the calling function name
	FieldCaller = "caller"
)

// NewEncoderConfig returns a zapcore.EncoderConfig with standardized field names
// for structured logging output.
//
// This is a pure function that returns a consistent configuration.
// The config uses:
//   - ISO8601 timestamps
//   - Lowercase level names
//   - Full caller path with line numbers
//   - Standard field names defined in this package
//
// Example:
//
//	config := NewEncoderConfig()
//	encoder := zapcore.NewJSONEncoder(config)
func NewEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		// Field keys
		TimeKey:       FieldTimestamp,
		LevelKey:      FieldLevel,
		NameKey:       FieldSource,
		CallerKey:     FieldCaller,
		MessageKey:    FieldMessage,
		StacktraceKey: FieldStacktrace,
		LineEnding:    zapcore.DefaultLineEnding,

		// Encoders
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// NewConsoleEncoderConfig returns a zapcore.EncoderConfig optimized for console output.
// This uses colored level output and human-readable timestamps.
//
// This is a pure function with no side effects.
func NewConsoleEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		// Field keys
		TimeKey:       FieldTimestamp,
		LevelKey:      FieldLevel,
		NameKey:       FieldSource,
		CallerKey:     FieldCaller,
		MessageKey:    FieldMessage,
		StacktraceKey: FieldStacktrace,
		LineEnding:    zapcore.DefaultLineEnding,

		// Encoders - console-friendly
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     shortTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// shortTimeEncoder encodes time in a compact format for console output.
// Format: 15:04:05.000
func shortTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("15:04:05.000"))
}
