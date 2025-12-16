package logging

import (
	"os"
	"strings"

	"go.uber.org/zap/zapcore"
)

// LogLevel represents valid log levels for the application.
// Using a custom type allows for clear documentation and validation.
type LogLevel = zapcore.Level

// Log level constants for convenience
const (
	DebugLevel = zapcore.DebugLevel
	InfoLevel  = zapcore.InfoLevel
	WarnLevel  = zapcore.WarnLevel
	ErrorLevel = zapcore.ErrorLevel
	FatalLevel = zapcore.FatalLevel
)

// ParseLogLevel parses a log level string from an environment variable.
// It returns the parsed level or the default level if the env var is empty or invalid.
//
// This is a pure function - it reads the env var value but has no other side effects.
// The parsing is case-insensitive.
//
// Valid levels: debug, info, warn, error, fatal (or DEBUG, INFO, WARN, ERROR, FATAL)
//
// Example:
//
//	level := ParseLogLevel("CANVUSLLM_LOG_LEVEL", zapcore.InfoLevel)
func ParseLogLevel(envVarName string, defaultLevel zapcore.Level) zapcore.Level {
	value := os.Getenv(envVarName)
	if value == "" {
		return defaultLevel
	}
	return ParseLogLevelString(value, defaultLevel)
}

// ParseLogLevelString parses a log level string directly.
// This is a pure function with no side effects.
//
// Valid levels: debug, info, warn, warning, error, fatal
// Parsing is case-insensitive.
func ParseLogLevelString(levelStr string, defaultLevel zapcore.Level) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return defaultLevel
	}
}
