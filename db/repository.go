// Package db provides database utilities including repository methods for CRUD operations.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ProcessingRecord represents a record in the processing_history table.
// This tracks AI inference operations performed on canvas widgets.
type ProcessingRecord struct {
	ID            int64     // Auto-incremented primary key
	CorrelationID string    // Unique identifier for tracing related operations
	CanvasID      string    // ID of the canvas containing the widget
	WidgetID      string    // ID of the widget that triggered processing
	OperationType string    // Type of operation (e.g., "text_generation", "image_generation")
	Prompt        string    // Input prompt sent to the model
	Response      string    // Model response text
	ModelName     string    // Name of the model used
	InputTokens   int       // Number of input tokens consumed
	OutputTokens  int       // Number of output tokens generated
	DurationMS    int       // Processing duration in milliseconds
	Status        string    // Status: "pending", "success", "error"
	ErrorMessage  string    // Error message if status is "error"
	CreatedAt     time.Time // Timestamp when record was created
}

// CanvasEvent represents a record in the canvas_events table.
// This tracks widget create, update, and delete events on the canvas.
type CanvasEvent struct {
	ID             int64     // Auto-incremented primary key
	CanvasID       string    // ID of the canvas
	WidgetID       string    // ID of the affected widget
	EventType      string    // Type: "created", "updated", "deleted"
	WidgetType     string    // Type of widget (e.g., "note", "image", "pdf")
	ContentPreview string    // Truncated preview of widget content
	CreatedAt      time.Time // Timestamp when event occurred
}

// ErrorLogEntry represents a record in the error_log table.
// This captures errors with context for debugging.
type ErrorLogEntry struct {
	ID            int64     // Auto-incremented primary key
	CorrelationID string    // Optional correlation ID linking to processing record
	ErrorType     string    // Category of error (e.g., "api_error", "model_error")
	ErrorMessage  string    // Error description
	StackTrace    string    // Stack trace if available
	Context       string    // JSON-encoded additional context
	CreatedAt     time.Time // Timestamp when error was logged
}

// PerformanceMetric represents a record in the performance_metrics table.
// This captures performance measurements over time.
type PerformanceMetric struct {
	ID          int64     // Auto-incremented primary key
	MetricType  string    // Category: "inference", "api", "system"
	MetricName  string    // Name of the metric
	MetricValue float64   // Measured value
	Metadata    string    // JSON-encoded additional data
	CreatedAt   time.Time // Timestamp when metric was recorded
}

// SystemMetric represents a record in the system_metrics table.
// This captures system resource utilization.
type SystemMetric struct {
	ID            int64     // Auto-incremented primary key
	MetricType    string    // Category: "cpu", "memory", "disk", "gpu"
	CPUUsage      float64   // CPU usage percentage
	MemoryUsedMB  float64   // Memory used in megabytes
	MemoryTotalMB float64   // Total memory in megabytes
	DiskUsedMB    float64   // Disk used in megabytes
	DiskTotalMB   float64   // Total disk in megabytes
	CreatedAt     time.Time // Timestamp when metric was recorded
}

// Repository provides CRUD operations for the database tables.
// It wraps a Database instance and provides type-safe methods
// for inserting and querying records.
//
// The Repository is designed to work with both synchronous and
// asynchronous writes via the AsyncWriter.
type Repository struct {
	db          *Database
	asyncWriter *AsyncWriter
}

// NewRepository creates a new Repository instance.
// The asyncWriter parameter is optional; if nil, all writes will be synchronous.
func NewRepository(db *Database, asyncWriter *AsyncWriter) *Repository {
	return &Repository{
		db:          db,
		asyncWriter: asyncWriter,
	}
}

// InsertProcessingHistory inserts a processing history record.
// If an asyncWriter is configured, the write is queued asynchronously.
// Returns the inserted record ID (0 for async writes).
func (r *Repository) InsertProcessingHistory(ctx context.Context, record ProcessingRecord) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	query := `
		INSERT INTO processing_history (
			correlation_id, canvas_id, widget_id, operation_type,
			prompt, response, model_name, input_tokens, output_tokens,
			duration_ms, status, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	args := []interface{}{
		record.CorrelationID,
		record.CanvasID,
		record.WidgetID,
		record.OperationType,
		record.Prompt,
		record.Response,
		record.ModelName,
		record.InputTokens,
		record.OutputTokens,
		record.DurationMS,
		record.Status,
		record.ErrorMessage,
	}

	// Use async writer if available
	if r.asyncWriter != nil && r.asyncWriter.IsStarted() {
		op := asyncInsertOp{
			query: query,
			args:  args,
		}
		if r.asyncWriter.Write(op) {
			return 0, nil // Async write queued successfully
		}
		// Fall through to sync write if channel is full
	}

	// Synchronous write
	result, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to insert processing history: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

// QueryRecentHistory retrieves the most recent processing history records.
// Results are ordered by created_at DESC.
func (r *Repository) QueryRecentHistory(ctx context.Context, limit int) ([]ProcessingRecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	if limit <= 0 {
		limit = 10 // Default limit
	}

	query := `
		SELECT id, correlation_id, canvas_id, widget_id, operation_type,
			   COALESCE(prompt, ''), COALESCE(response, ''), COALESCE(model_name, ''),
			   COALESCE(input_tokens, 0), COALESCE(output_tokens, 0),
			   COALESCE(duration_ms, 0), status, COALESCE(error_message, ''),
			   created_at
		FROM processing_history
		ORDER BY created_at DESC
		LIMIT ?`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query processing history: %w", err)
	}
	defer rows.Close()

	var records []ProcessingRecord
	for rows.Next() {
		var rec ProcessingRecord
		var createdAt string

		err := rows.Scan(
			&rec.ID,
			&rec.CorrelationID,
			&rec.CanvasID,
			&rec.WidgetID,
			&rec.OperationType,
			&rec.Prompt,
			&rec.Response,
			&rec.ModelName,
			&rec.InputTokens,
			&rec.OutputTokens,
			&rec.DurationMS,
			&rec.Status,
			&rec.ErrorMessage,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan processing history row: %w", err)
		}

		// Parse SQLite datetime
		rec.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating processing history rows: %w", err)
	}

	return records, nil
}

// QueryHistoryByCorrelationID retrieves processing history for a specific correlation ID.
func (r *Repository) QueryHistoryByCorrelationID(ctx context.Context, correlationID string) ([]ProcessingRecord, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	query := `
		SELECT id, correlation_id, canvas_id, widget_id, operation_type,
			   COALESCE(prompt, ''), COALESCE(response, ''), COALESCE(model_name, ''),
			   COALESCE(input_tokens, 0), COALESCE(output_tokens, 0),
			   COALESCE(duration_ms, 0), status, COALESCE(error_message, ''),
			   created_at
		FROM processing_history
		WHERE correlation_id = ?
		ORDER BY created_at DESC`

	rows, err := r.db.Query(query, correlationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query processing history: %w", err)
	}
	defer rows.Close()

	var records []ProcessingRecord
	for rows.Next() {
		var rec ProcessingRecord
		var createdAt string

		err := rows.Scan(
			&rec.ID,
			&rec.CorrelationID,
			&rec.CanvasID,
			&rec.WidgetID,
			&rec.OperationType,
			&rec.Prompt,
			&rec.Response,
			&rec.ModelName,
			&rec.InputTokens,
			&rec.OutputTokens,
			&rec.DurationMS,
			&rec.Status,
			&rec.ErrorMessage,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan processing history row: %w", err)
		}

		rec.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		records = append(records, rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating processing history rows: %w", err)
	}

	return records, nil
}

// InsertCanvasEvent inserts a canvas event record.
// If an asyncWriter is configured, the write is queued asynchronously.
func (r *Repository) InsertCanvasEvent(ctx context.Context, event CanvasEvent) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	query := `
		INSERT INTO canvas_events (
			canvas_id, widget_id, event_type, widget_type, content_preview
		) VALUES (?, ?, ?, ?, ?)`

	args := []interface{}{
		event.CanvasID,
		event.WidgetID,
		event.EventType,
		event.WidgetType,
		event.ContentPreview,
	}

	// Use async writer if available
	if r.asyncWriter != nil && r.asyncWriter.IsStarted() {
		op := asyncInsertOp{
			query: query,
			args:  args,
		}
		if r.asyncWriter.Write(op) {
			return 0, nil // Async write queued successfully
		}
		// Fall through to sync write if channel is full
	}

	// Synchronous write
	result, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to insert canvas event: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

// QueryRecentCanvasEvents retrieves the most recent canvas events.
// Results are ordered by created_at DESC.
func (r *Repository) QueryRecentCanvasEvents(ctx context.Context, limit int) ([]CanvasEvent, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, canvas_id, widget_id, event_type, widget_type,
			   COALESCE(content_preview, ''), created_at
		FROM canvas_events
		ORDER BY created_at DESC
		LIMIT ?`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query canvas events: %w", err)
	}
	defer rows.Close()

	var events []CanvasEvent
	for rows.Next() {
		var evt CanvasEvent
		var createdAt string

		err := rows.Scan(
			&evt.ID,
			&evt.CanvasID,
			&evt.WidgetID,
			&evt.EventType,
			&evt.WidgetType,
			&evt.ContentPreview,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan canvas event row: %w", err)
		}

		evt.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		events = append(events, evt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating canvas event rows: %w", err)
	}

	return events, nil
}

// QueryCanvasEventsByWidgetID retrieves events for a specific widget.
func (r *Repository) QueryCanvasEventsByWidgetID(ctx context.Context, widgetID string, limit int) ([]CanvasEvent, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, canvas_id, widget_id, event_type, widget_type,
			   COALESCE(content_preview, ''), created_at
		FROM canvas_events
		WHERE widget_id = ?
		ORDER BY created_at DESC
		LIMIT ?`

	rows, err := r.db.Query(query, widgetID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query canvas events: %w", err)
	}
	defer rows.Close()

	var events []CanvasEvent
	for rows.Next() {
		var evt CanvasEvent
		var createdAt string

		err := rows.Scan(
			&evt.ID,
			&evt.CanvasID,
			&evt.WidgetID,
			&evt.EventType,
			&evt.WidgetType,
			&evt.ContentPreview,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan canvas event row: %w", err)
		}

		evt.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		events = append(events, evt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating canvas event rows: %w", err)
	}

	return events, nil
}

// InsertErrorLog inserts an error log entry.
// If an asyncWriter is configured, the write is queued asynchronously.
func (r *Repository) InsertErrorLog(ctx context.Context, entry ErrorLogEntry) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	query := `
		INSERT INTO error_log (
			correlation_id, error_type, error_message, stack_trace, context
		) VALUES (?, ?, ?, ?, ?)`

	args := []interface{}{
		nullString(entry.CorrelationID),
		entry.ErrorType,
		entry.ErrorMessage,
		nullString(entry.StackTrace),
		nullString(entry.Context),
	}

	// Use async writer if available
	if r.asyncWriter != nil && r.asyncWriter.IsStarted() {
		op := asyncInsertOp{
			query: query,
			args:  args,
		}
		if r.asyncWriter.Write(op) {
			return 0, nil // Async write queued successfully
		}
		// Fall through to sync write if channel is full
	}

	// Synchronous write
	result, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to insert error log: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

// QueryRecentErrorLogs retrieves the most recent error log entries.
// Results are ordered by created_at DESC.
func (r *Repository) QueryRecentErrorLogs(ctx context.Context, limit int) ([]ErrorLogEntry, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, COALESCE(correlation_id, ''), error_type, error_message,
			   COALESCE(stack_trace, ''), COALESCE(context, ''), created_at
		FROM error_log
		ORDER BY created_at DESC
		LIMIT ?`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query error logs: %w", err)
	}
	defer rows.Close()

	var entries []ErrorLogEntry
	for rows.Next() {
		var entry ErrorLogEntry
		var createdAt string

		err := rows.Scan(
			&entry.ID,
			&entry.CorrelationID,
			&entry.ErrorType,
			&entry.ErrorMessage,
			&entry.StackTrace,
			&entry.Context,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan error log row: %w", err)
		}

		entry.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating error log rows: %w", err)
	}

	return entries, nil
}

// QueryErrorLogsByType retrieves error logs filtered by error type.
func (r *Repository) QueryErrorLogsByType(ctx context.Context, errorType string, limit int) ([]ErrorLogEntry, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}

	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT id, COALESCE(correlation_id, ''), error_type, error_message,
			   COALESCE(stack_trace, ''), COALESCE(context, ''), created_at
		FROM error_log
		WHERE error_type = ?
		ORDER BY created_at DESC
		LIMIT ?`

	rows, err := r.db.Query(query, errorType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query error logs: %w", err)
	}
	defer rows.Close()

	var entries []ErrorLogEntry
	for rows.Next() {
		var entry ErrorLogEntry
		var createdAt string

		err := rows.Scan(
			&entry.ID,
			&entry.CorrelationID,
			&entry.ErrorType,
			&entry.ErrorMessage,
			&entry.StackTrace,
			&entry.Context,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan error log row: %w", err)
		}

		entry.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating error log rows: %w", err)
	}

	return entries, nil
}

// InsertPerformanceMetric inserts a performance metric record.
// If an asyncWriter is configured, the write is queued asynchronously.
func (r *Repository) InsertPerformanceMetric(ctx context.Context, metric PerformanceMetric) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	query := `
		INSERT INTO performance_metrics (
			metric_type, metric_name, metric_value, metadata
		) VALUES (?, ?, ?, ?)`

	args := []interface{}{
		metric.MetricType,
		metric.MetricName,
		metric.MetricValue,
		nullString(metric.Metadata),
	}

	// Use async writer if available
	if r.asyncWriter != nil && r.asyncWriter.IsStarted() {
		op := asyncInsertOp{
			query: query,
			args:  args,
		}
		if r.asyncWriter.Write(op) {
			return 0, nil // Async write queued successfully
		}
		// Fall through to sync write if channel is full
	}

	// Synchronous write
	result, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to insert performance metric: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

// InsertSystemMetric inserts a system metric record.
// If an asyncWriter is configured, the write is queued asynchronously.
func (r *Repository) InsertSystemMetric(ctx context.Context, metric SystemMetric) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	query := `
		INSERT INTO system_metrics (
			metric_type, cpu_usage, memory_used_mb, memory_total_mb,
			disk_used_mb, disk_total_mb
		) VALUES (?, ?, ?, ?, ?, ?)`

	args := []interface{}{
		metric.MetricType,
		metric.CPUUsage,
		metric.MemoryUsedMB,
		metric.MemoryTotalMB,
		metric.DiskUsedMB,
		metric.DiskTotalMB,
	}

	// Use async writer if available
	if r.asyncWriter != nil && r.asyncWriter.IsStarted() {
		op := asyncInsertOp{
			query: query,
			args:  args,
		}
		if r.asyncWriter.Write(op) {
			return 0, nil // Async write queued successfully
		}
		// Fall through to sync write if channel is full
	}

	// Synchronous write
	result, err := r.db.Exec(query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to insert system metric: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get last insert id: %w", err)
	}

	return id, nil
}

// asyncInsertOp is an internal type for async insert operations.
type asyncInsertOp struct {
	query string
	args  []interface{}
}

// CreateAsyncWriteHandler creates a WriteHandler for the Repository.
// This handler processes asyncInsertOp operations.
func (r *Repository) CreateAsyncWriteHandler() WriteHandler {
	return func(op WriteOperation) error {
		insertOp, ok := op.Data.(asyncInsertOp)
		if !ok {
			return fmt.Errorf("invalid operation type: expected asyncInsertOp")
		}

		_, err := r.db.Exec(insertOp.query, insertOp.args...)
		return err
	}
}

// nullString converts an empty string to sql.NullString for NULL storage.
func nullString(s string) interface{} {
	if s == "" {
		return sql.NullString{String: "", Valid: false}
	}
	return s
}

// CountProcessingHistory returns the total count of processing history records.
func (r *Repository) CountProcessingHistory(ctx context.Context) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	var count int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM processing_history").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count processing history: %w", err)
	}

	return count, nil
}

// CountCanvasEvents returns the total count of canvas events.
func (r *Repository) CountCanvasEvents(ctx context.Context) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	var count int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM canvas_events").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count canvas events: %w", err)
	}

	return count, nil
}

// CountErrorLogs returns the total count of error log entries.
func (r *Repository) CountErrorLogs(ctx context.Context) (int64, error) {
	if r.db == nil {
		return 0, fmt.Errorf("database connection is nil")
	}

	var count int64
	err := r.db.QueryRow("SELECT COUNT(*) FROM error_log").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count error logs: %w", err)
	}

	return count, nil
}
