package logging

import (
	"os"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestParseLogLevelString(t *testing.T) {
	tests := []struct {
		name         string
		levelStr     string
		defaultLevel zapcore.Level
		expected     zapcore.Level
	}{
		{
			name:         "debug level lowercase",
			levelStr:     "debug",
			defaultLevel: zapcore.InfoLevel,
			expected:     zapcore.DebugLevel,
		},
		{
			name:         "info level uppercase",
			levelStr:     "INFO",
			defaultLevel: zapcore.DebugLevel,
			expected:     zapcore.InfoLevel,
		},
		{
			name:         "warn level mixed case",
			levelStr:     "Warn",
			defaultLevel: zapcore.InfoLevel,
			expected:     zapcore.WarnLevel,
		},
		{
			name:         "warning alternative",
			levelStr:     "warning",
			defaultLevel: zapcore.InfoLevel,
			expected:     zapcore.WarnLevel,
		},
		{
			name:         "error level",
			levelStr:     "error",
			defaultLevel: zapcore.InfoLevel,
			expected:     zapcore.ErrorLevel,
		},
		{
			name:         "fatal level",
			levelStr:     "FATAL",
			defaultLevel: zapcore.InfoLevel,
			expected:     zapcore.FatalLevel,
		},
		{
			name:         "invalid level returns default",
			levelStr:     "invalid",
			defaultLevel: zapcore.WarnLevel,
			expected:     zapcore.WarnLevel,
		},
		{
			name:         "empty string returns default",
			levelStr:     "",
			defaultLevel: zapcore.ErrorLevel,
			expected:     zapcore.ErrorLevel,
		},
		{
			name:         "whitespace trimmed",
			levelStr:     "  debug  ",
			defaultLevel: zapcore.InfoLevel,
			expected:     zapcore.DebugLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseLogLevelString(tt.levelStr, tt.defaultLevel)
			if result != tt.expected {
				t.Errorf("ParseLogLevelString(%q, %v) = %v, want %v",
					tt.levelStr, tt.defaultLevel, result, tt.expected)
			}
		})
	}
}

func TestParseLogLevel(t *testing.T) {
	const testEnvVar = "TEST_LOG_LEVEL_PARSE"

	// Clean up after test
	defer os.Unsetenv(testEnvVar)

	t.Run("reads from environment variable", func(t *testing.T) {
		os.Setenv(testEnvVar, "debug")
		result := ParseLogLevel(testEnvVar, zapcore.InfoLevel)
		if result != zapcore.DebugLevel {
			t.Errorf("ParseLogLevel with env=debug got %v, want %v", result, zapcore.DebugLevel)
		}
	})

	t.Run("returns default when env var empty", func(t *testing.T) {
		os.Unsetenv(testEnvVar)
		result := ParseLogLevel(testEnvVar, zapcore.WarnLevel)
		if result != zapcore.WarnLevel {
			t.Errorf("ParseLogLevel with empty env got %v, want %v", result, zapcore.WarnLevel)
		}
	})

	t.Run("returns default when env var invalid", func(t *testing.T) {
		os.Setenv(testEnvVar, "not-a-level")
		result := ParseLogLevel(testEnvVar, zapcore.ErrorLevel)
		if result != zapcore.ErrorLevel {
			t.Errorf("ParseLogLevel with invalid env got %v, want %v", result, zapcore.ErrorLevel)
		}
	})
}
