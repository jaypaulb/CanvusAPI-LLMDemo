package logging

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the main logging organism that wraps zap.Logger and provides
// structured logging with automatic sensitive data redaction.
//
// This organism composes:
//   - FileWriter molecule (log file rotation via lumberjack)
//   - MultiCore molecule (tee output to console + file)
//   - SensitiveFilter atom (API key redaction)
//
// Example:
//
//	logger, err := NewLogger(true, "app.log")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer logger.Sync()
//
//	logger.Info("server started", zap.String("port", "8080"))
//	logger.Infow("request received", "method", "GET", "path", "/api/health")
type Logger struct {
	// zap is the underlying structured logger
	zap *zap.Logger

	// sugar is the sugared logger for printf-style logging
	sugar *zap.SugaredLogger

	// isDevelopment indicates if running in development mode
	isDevelopment bool

	// logFilePath is the path to the log file
	logFilePath string
}

// NewLogger creates a new Logger instance configured for the given environment.
//
// Parameters:
//   - isDevelopment: When true, uses colored console output with debug level.
//     When false, uses JSON output with info level.
//   - logFilePath: Path to the log file. File will be created if it doesn't exist.
//     Log rotation is automatically configured (100MB max, 5 backups, 30 days).
//
// Returns an error if the log file cannot be created or opened.
//
// The logger automatically:
//   - Outputs to both console and file
//   - Rotates log files when they exceed 100MB
//   - Retains 5 backup log files for up to 30 days
//   - Compresses rotated files
//   - Includes caller information in log entries
//
// Example:
//
//	// Development mode
//	devLogger, err := NewLogger(true, "app.log")
//
//	// Production mode
//	prodLogger, err := NewLogger(false, "/var/log/canvus/app.log")
func NewLogger(isDevelopment bool, logFilePath string) (*Logger, error) {
	// Determine log level based on environment
	var level zapcore.Level
	if isDevelopment {
		level = zapcore.DebugLevel
	} else {
		level = zapcore.InfoLevel
	}

	// Create multi-core that outputs to both console and file
	// Uses FileWriter molecule internally for rotation
	core, err := NewMultiCore(level, logFilePath, isDevelopment)
	if err != nil {
		return nil, fmt.Errorf("failed to create log core: %w", err)
	}

	// Build the zap logger with caller info
	zapLogger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1), // Skip this wrapper layer
	)

	return &Logger{
		zap:           zapLogger,
		sugar:         zapLogger.Sugar(),
		isDevelopment: isDevelopment,
		logFilePath:   logFilePath,
	}, nil
}

// NewLoggerWithConfig creates a Logger with custom file rotation configuration.
//
// This variant allows fine-grained control over log rotation behavior.
// For default configuration, use NewLogger instead.
//
// Example:
//
//	config := FileWriterConfig{
//	    MaxSizeMB:  50,
//	    MaxBackups: 3,
//	    MaxAgeDays: 7,
//	    Compress:   true,
//	}
//	logger, err := NewLoggerWithConfig(true, "app.log", config)
func NewLoggerWithConfig(isDevelopment bool, logFilePath string, fileConfig FileWriterConfig) (*Logger, error) {
	// Determine log level based on environment
	var level zapcore.Level
	if isDevelopment {
		level = zapcore.DebugLevel
	} else {
		level = zapcore.InfoLevel
	}

	// Create file writer with custom rotation config
	fileWriter := NewFileWriterWithConfig(logFilePath, fileConfig)

	// Create console writer
	consoleWriter := zapcore.AddSync(&consoleWriterSync{})

	// Create multi-core with custom writers
	core := NewMultiCoreWithWriters(level, consoleWriter, fileWriter, isDevelopment)

	// Build the zap logger with caller info
	zapLogger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)

	return &Logger{
		zap:           zapLogger,
		sugar:         zapLogger.Sugar(),
		isDevelopment: isDevelopment,
		logFilePath:   logFilePath,
	}, nil
}

// consoleWriterSync wraps os.Stdout to implement zapcore.WriteSyncer
type consoleWriterSync struct{}

func (c *consoleWriterSync) Write(p []byte) (n int, err error) {
	return fmt.Print(string(p))
}

func (c *consoleWriterSync) Sync() error {
	return nil
}

// Sync flushes any buffered log entries.
// Applications should call Sync before exiting to ensure all logs are written.
//
// Example:
//
//	logger, _ := NewLogger(true, "app.log")
//	defer logger.Sync()
func (l *Logger) Sync() error {
	if l == nil || l.zap == nil {
		return nil
	}
	return l.zap.Sync()
}

// Debug logs a message at DebugLevel with optional structured fields.
//
// Example:
//
//	logger.Debug("processing request",
//	    zap.String("request_id", "abc123"),
//	    zap.Int("attempt", 1))
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.zap.Debug(msg, l.redactFields(fields)...)
}

// Info logs a message at InfoLevel with optional structured fields.
//
// Example:
//
//	logger.Info("server started",
//	    zap.String("host", "localhost"),
//	    zap.Int("port", 8080))
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.zap.Info(msg, l.redactFields(fields)...)
}

// Warn logs a message at WarnLevel with optional structured fields.
//
// Example:
//
//	logger.Warn("high memory usage",
//	    zap.Float64("usage_percent", 85.5))
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.zap.Warn(msg, l.redactFields(fields)...)
}

// Error logs a message at ErrorLevel with optional structured fields.
//
// Example:
//
//	logger.Error("failed to connect",
//	    zap.Error(err),
//	    zap.String("host", "api.example.com"))
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.zap.Error(msg, l.redactFields(fields)...)
}

// Fatal logs a message at FatalLevel then calls os.Exit(1).
//
// Example:
//
//	logger.Fatal("configuration error",
//	    zap.String("missing", "CANVUS_API_KEY"))
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.zap.Fatal(msg, l.redactFields(fields)...)
}

// Panic logs a message at PanicLevel then panics.
//
// Example:
//
//	logger.Panic("invariant violated",
//	    zap.String("expected", "non-nil"),
//	    zap.String("got", "nil"))
func (l *Logger) Panic(msg string, fields ...zap.Field) {
	l.zap.Panic(msg, l.redactFields(fields)...)
}

// Debugw logs a message at DebugLevel with loosely-typed key-value pairs.
// Use this for printf-style logging when you don't need type safety.
//
// Example:
//
//	logger.Debugw("processing request",
//	    "request_id", "abc123",
//	    "attempt", 1)
func (l *Logger) Debugw(msg string, keysAndValues ...interface{}) {
	l.sugar.Debugw(msg, l.redactKeysAndValues(keysAndValues)...)
}

// Infow logs a message at InfoLevel with loosely-typed key-value pairs.
//
// Example:
//
//	logger.Infow("user logged in",
//	    "user_id", 12345,
//	    "ip", "192.168.1.1")
func (l *Logger) Infow(msg string, keysAndValues ...interface{}) {
	l.sugar.Infow(msg, l.redactKeysAndValues(keysAndValues)...)
}

// Warnw logs a message at WarnLevel with loosely-typed key-value pairs.
//
// Example:
//
//	logger.Warnw("slow query",
//	    "duration_ms", 1500,
//	    "query", "SELECT * FROM users")
func (l *Logger) Warnw(msg string, keysAndValues ...interface{}) {
	l.sugar.Warnw(msg, l.redactKeysAndValues(keysAndValues)...)
}

// Errorw logs a message at ErrorLevel with loosely-typed key-value pairs.
//
// Example:
//
//	logger.Errorw("request failed",
//	    "error", err.Error(),
//	    "status", 500)
func (l *Logger) Errorw(msg string, keysAndValues ...interface{}) {
	l.sugar.Errorw(msg, l.redactKeysAndValues(keysAndValues)...)
}

// Fatalw logs a message at FatalLevel then calls os.Exit(1).
//
// Example:
//
//	logger.Fatalw("startup failed",
//	    "reason", "database connection timeout")
func (l *Logger) Fatalw(msg string, keysAndValues ...interface{}) {
	l.sugar.Fatalw(msg, l.redactKeysAndValues(keysAndValues)...)
}

// Panicw logs a message at PanicLevel then panics.
//
// Example:
//
//	logger.Panicw("assertion failed",
//	    "expected", 10,
//	    "got", 0)
func (l *Logger) Panicw(msg string, keysAndValues ...interface{}) {
	l.sugar.Panicw(msg, l.redactKeysAndValues(keysAndValues)...)
}

// Debugf logs a formatted message at DebugLevel.
//
// Example:
//
//	logger.Debugf("processing item %d of %d", current, total)
func (l *Logger) Debugf(template string, args ...interface{}) {
	l.sugar.Debugf(template, args...)
}

// Infof logs a formatted message at InfoLevel.
//
// Example:
//
//	logger.Infof("server listening on port %d", port)
func (l *Logger) Infof(template string, args ...interface{}) {
	l.sugar.Infof(template, args...)
}

// Warnf logs a formatted message at WarnLevel.
//
// Example:
//
//	logger.Warnf("rate limit approaching: %d/%d", current, limit)
func (l *Logger) Warnf(template string, args ...interface{}) {
	l.sugar.Warnf(template, args...)
}

// Errorf logs a formatted message at ErrorLevel.
//
// Example:
//
//	logger.Errorf("failed to process: %v", err)
func (l *Logger) Errorf(template string, args ...interface{}) {
	l.sugar.Errorf(template, args...)
}

// Fatalf logs a formatted message at FatalLevel then calls os.Exit(1).
//
// Example:
//
//	logger.Fatalf("fatal error: %v", err)
func (l *Logger) Fatalf(template string, args ...interface{}) {
	l.sugar.Fatalf(template, args...)
}

// Panicf logs a formatted message at PanicLevel then panics.
//
// Example:
//
//	logger.Panicf("unexpected state: %v", state)
func (l *Logger) Panicf(template string, args ...interface{}) {
	l.sugar.Panicf(template, args...)
}

// With creates a child logger with additional fields that will be included
// in all log entries from the child.
//
// This is useful for adding context that applies to a subset of operations,
// such as request IDs or user IDs.
//
// Example:
//
//	requestLogger := logger.With(
//	    zap.String("request_id", "abc123"),
//	    zap.String("user_id", "user456"))
//
//	requestLogger.Info("processing request")
//	requestLogger.Info("request complete")
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{
		zap:           l.zap.With(l.redactFields(fields)...),
		sugar:         l.sugar.With(l.redactFieldsToInterface(fields)...),
		isDevelopment: l.isDevelopment,
		logFilePath:   l.logFilePath,
	}
}

// WithOptions creates a child logger with additional zap options.
//
// Example:
//
//	verboseLogger := logger.WithOptions(zap.AddStacktrace(zapcore.ErrorLevel))
func (l *Logger) WithOptions(opts ...zap.Option) *Logger {
	newZap := l.zap.WithOptions(opts...)
	return &Logger{
		zap:           newZap,
		sugar:         newZap.Sugar(),
		isDevelopment: l.isDevelopment,
		logFilePath:   l.logFilePath,
	}
}

// Named adds a sub-logger name. Logger names appear in log output and
// help identify the source of log entries.
//
// Example:
//
//	httpLogger := logger.Named("http")
//	dbLogger := logger.Named("database")
func (l *Logger) Named(name string) *Logger {
	newZap := l.zap.Named(name)
	return &Logger{
		zap:           newZap,
		sugar:         newZap.Sugar(),
		isDevelopment: l.isDevelopment,
		logFilePath:   l.logFilePath,
	}
}

// Sugar returns the underlying sugared logger for direct access to
// SugaredLogger methods not exposed by this wrapper.
func (l *Logger) Sugar() *zap.SugaredLogger {
	return l.sugar
}

// Zap returns the underlying zap.Logger for direct access to
// Logger methods not exposed by this wrapper.
func (l *Logger) Zap() *zap.Logger {
	return l.zap
}

// IsDevelopment returns true if the logger is configured for development mode.
func (l *Logger) IsDevelopment() bool {
	return l.isDevelopment
}

// LogFilePath returns the path to the log file.
func (l *Logger) LogFilePath() string {
	return l.logFilePath
}

// redactFields filters sensitive data from zap.Field values.
// This is called before every log operation to ensure no sensitive data leaks.
func (l *Logger) redactFields(fields []zap.Field) []zap.Field {
	if len(fields) == 0 {
		return fields
	}

	result := make([]zap.Field, len(fields))
	for i, field := range fields {
		result[i] = l.redactField(field)
	}
	return result
}

// redactField redacts a single zap.Field if it contains sensitive data.
func (l *Logger) redactField(field zap.Field) zap.Field {
	// Check if field name indicates sensitive data
	if IsSensitiveField(field.Key) {
		return zap.String(field.Key, RedactedPlaceholder)
	}

	// For string fields, check and redact the value
	if field.Type == zapcore.StringType {
		redacted := RedactSensitiveData(field.String)
		if redacted != field.String {
			return zap.String(field.Key, redacted)
		}
	}

	return field
}

// redactKeysAndValues filters sensitive data from key-value pairs used in sugared logging.
func (l *Logger) redactKeysAndValues(keysAndValues []interface{}) []interface{} {
	if len(keysAndValues) == 0 {
		return keysAndValues
	}

	result := make([]interface{}, len(keysAndValues))
	copy(result, keysAndValues)

	// Process pairs: even indices are keys, odd indices are values
	for i := 0; i < len(result)-1; i += 2 {
		key, ok := result[i].(string)
		if !ok {
			continue
		}

		// Check if key indicates sensitive field
		if IsSensitiveField(key) {
			result[i+1] = RedactedPlaceholder
			continue
		}

		// Check if value contains sensitive data
		if value, ok := result[i+1].(string); ok {
			result[i+1] = RedactSensitiveData(value)
		}
	}

	return result
}

// redactFieldsToInterface converts zap.Fields to interface slice for sugared logger.
func (l *Logger) redactFieldsToInterface(fields []zap.Field) []interface{} {
	result := make([]interface{}, 0, len(fields)*2)
	for _, field := range fields {
		redacted := l.redactField(field)
		result = append(result, redacted.Key, redacted.String)
	}
	return result
}
