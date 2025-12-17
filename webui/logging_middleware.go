// Package webui provides the web-based user interface for CanvusLocalLLM.
// This file contains the LoggingMiddleware molecule for HTTP request logging.
package webui

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// LoggingMiddleware is a molecule that logs all HTTP requests with
// timestamp, method, path, status code, and duration.
//
// It composes:
//   - HTTP ResponseWriter wrapper (to capture status code)
//   - Time measurement for duration
//   - Logger for output
//
// Thread-safe for concurrent HTTP requests.
type LoggingMiddleware struct {
	// logger for request logging
	logger RequestLogger

	// skipPaths are paths to skip logging (e.g., health checks)
	skipPaths map[string]bool
}

// RequestLogger interface for logging HTTP requests
type RequestLogger interface {
	LogRequest(entry RequestLogEntry)
}

// RequestLogEntry contains all information about a logged HTTP request
type RequestLogEntry struct {
	// Timestamp when the request started
	Timestamp time.Time

	// Method is the HTTP method (GET, POST, etc.)
	Method string

	// Path is the URL path
	Path string

	// StatusCode is the HTTP response status code
	StatusCode int

	// Duration is how long the request took
	Duration time.Duration

	// RemoteAddr is the client's address
	RemoteAddr string

	// UserAgent is the client's user agent string
	UserAgent string

	// ContentLength is the response size in bytes (-1 if unknown)
	ContentLength int64
}

// DefaultRequestLogger logs to the standard log package
type DefaultRequestLogger struct{}

// LogRequest logs a request entry using the default format
func (d *DefaultRequestLogger) LogRequest(entry RequestLogEntry) {
	statusColor := getStatusColor(entry.StatusCode)
	log.Printf("%s %s %s %s%d%s %s %s",
		entry.Timestamp.Format("2006-01-02 15:04:05"),
		entry.Method,
		entry.Path,
		statusColor,
		entry.StatusCode,
		colorReset,
		entry.Duration.Round(time.Millisecond),
		entry.RemoteAddr,
	)
}

// Color codes for terminal output
const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
)

// getStatusColor returns ANSI color code based on status code range
func getStatusColor(status int) string {
	switch {
	case status >= 500:
		return colorRed
	case status >= 400:
		return colorYellow
	case status >= 300:
		return colorCyan
	case status >= 200:
		return colorGreen
	default:
		return colorReset
	}
}

// LoggingMiddlewareConfig holds configuration for the LoggingMiddleware
type LoggingMiddlewareConfig struct {
	// Logger for request logging (default: DefaultRequestLogger)
	Logger RequestLogger

	// SkipPaths are paths to skip logging (default: none)
	SkipPaths []string

	// LogUserAgent whether to include user agent in logs (default: false)
	LogUserAgent bool

	// LogContentLength whether to include response size in logs (default: false)
	LogContentLength bool
}

// DefaultLoggingMiddlewareConfig returns the default configuration
func DefaultLoggingMiddlewareConfig() LoggingMiddlewareConfig {
	return LoggingMiddlewareConfig{
		Logger:           &DefaultRequestLogger{},
		SkipPaths:        nil,
		LogUserAgent:     false,
		LogContentLength: false,
	}
}

// NewLoggingMiddleware creates a new LoggingMiddleware with default configuration.
func NewLoggingMiddleware() *LoggingMiddleware {
	return NewLoggingMiddlewareWithConfig(DefaultLoggingMiddlewareConfig())
}

// NewLoggingMiddlewareWithConfig creates a new LoggingMiddleware with custom configuration.
func NewLoggingMiddlewareWithConfig(config LoggingMiddlewareConfig) *LoggingMiddleware {
	if config.Logger == nil {
		config.Logger = &DefaultRequestLogger{}
	}

	skipPaths := make(map[string]bool)
	for _, path := range config.SkipPaths {
		skipPaths[path] = true
	}

	return &LoggingMiddleware{
		logger:    config.Logger,
		skipPaths: skipPaths,
	}
}

// Handler wraps an http.Handler with request logging.
//
// This is the main middleware function that logs all HTTP requests.
//
// Parameters:
//   - next: The next handler in the chain
//
// Returns:
//   - http.Handler that logs requests before passing to next
func (m *LoggingMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip logging for configured paths
		if m.skipPaths[r.URL.Path] {
			next.ServeHTTP(w, r)
			return
		}

		// Record start time
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriterWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default if not explicitly set
		}

		// Call next handler
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Log the request
		entry := RequestLogEntry{
			Timestamp:     start,
			Method:        r.Method,
			Path:          r.URL.Path,
			StatusCode:    wrapped.statusCode,
			Duration:      duration,
			RemoteAddr:    getClientIP(r),
			UserAgent:     r.UserAgent(),
			ContentLength: wrapped.bytesWritten,
		}

		m.logger.LogRequest(entry)
	})
}

// HandlerFunc wraps an http.HandlerFunc with request logging.
//
// Convenience method for wrapping handler functions.
//
// Parameters:
//   - next: The next handler function
//
// Returns:
//   - http.Handler that logs requests before passing to next
func (m *LoggingMiddleware) HandlerFunc(next http.HandlerFunc) http.Handler {
	return m.Handler(next)
}

// responseWriterWrapper wraps http.ResponseWriter to capture status code
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	wroteHeader  bool
}

// WriteHeader captures the status code
func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	if !w.wroteHeader {
		w.statusCode = statusCode
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write captures the bytes written and ensures header is written
func (w *responseWriterWrapper) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += int64(n)
	return n, err
}

// Flush implements http.Flusher if the underlying writer supports it
func (w *responseWriterWrapper) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// getClientIP extracts the client IP from the request
// Checks X-Forwarded-For and X-Real-IP headers first for proxied requests
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (may contain multiple IPs)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Use the first IP in the list
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// FormattedLogger logs requests with a custom format string
type FormattedLogger struct {
	// Format string using placeholders:
	// %t = timestamp, %m = method, %p = path, %s = status, %d = duration, %a = address
	Format string

	// TimeFormat is the time format string (default: RFC3339)
	TimeFormat string

	// Output function (default: log.Printf)
	Output func(format string, v ...interface{})
}

// LogRequest logs a request using the configured format
func (f *FormattedLogger) LogRequest(entry RequestLogEntry) {
	output := f.Output
	if output == nil {
		output = log.Printf
	}

	timeFormat := f.TimeFormat
	if timeFormat == "" {
		timeFormat = time.RFC3339
	}

	format := f.Format
	if format == "" {
		format = "%t %m %p %s %d %a"
	}

	// Build formatted string
	result := ""
	for i := 0; i < len(format); i++ {
		if format[i] == '%' && i+1 < len(format) {
			switch format[i+1] {
			case 't':
				result += entry.Timestamp.Format(timeFormat)
				i++
			case 'm':
				result += entry.Method
				i++
			case 'p':
				result += entry.Path
				i++
			case 's':
				result += fmt.Sprintf("%d", entry.StatusCode)
				i++
			case 'd':
				result += entry.Duration.Round(time.Millisecond).String()
				i++
			case 'a':
				result += entry.RemoteAddr
				i++
			case 'u':
				result += entry.UserAgent
				i++
			case 'l':
				result += fmt.Sprintf("%d", entry.ContentLength)
				i++
			case '%':
				result += "%"
				i++
			default:
				result += string(format[i])
			}
		} else {
			result += string(format[i])
		}
	}

	output("%s", result)
}

// JSONLogger logs requests as JSON for structured logging
type JSONLogger struct {
	// Output function (default: log.Printf)
	Output func(format string, v ...interface{})
}

// LogRequest logs a request as JSON
func (j *JSONLogger) LogRequest(entry RequestLogEntry) {
	output := j.Output
	if output == nil {
		output = log.Printf
	}

	output(`{"timestamp":"%s","method":"%s","path":"%s","status":%d,"duration_ms":%d,"remote_addr":"%s"}`,
		entry.Timestamp.Format(time.RFC3339),
		entry.Method,
		entry.Path,
		entry.StatusCode,
		entry.Duration.Milliseconds(),
		entry.RemoteAddr,
	)
}

// NoopLogger discards all log entries (for testing or production with external logging)
type NoopLogger struct{}

// LogRequest does nothing
func (n *NoopLogger) LogRequest(entry RequestLogEntry) {}
