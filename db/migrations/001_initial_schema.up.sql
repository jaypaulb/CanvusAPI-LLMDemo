-- Initial database schema for CanvusLocalLLM
-- Migration: 001_initial_schema
-- Created: 2025-01-16

-- processing_history: Records of AI processing operations
-- Used for tracking and auditing AI inference requests
CREATE TABLE processing_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    correlation_id TEXT NOT NULL,
    canvas_id TEXT NOT NULL,
    widget_id TEXT NOT NULL,
    operation_type TEXT NOT NULL,
    prompt TEXT,
    response TEXT,
    model_name TEXT,
    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    duration_ms INTEGER DEFAULT 0,
    status TEXT NOT NULL,
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for processing_history
CREATE INDEX idx_processing_history_correlation_id ON processing_history(correlation_id);
CREATE INDEX idx_processing_history_canvas_id ON processing_history(canvas_id);
CREATE INDEX idx_processing_history_created_at ON processing_history(created_at);

-- canvas_events: Events that occurred on canvas widgets
-- Used for tracking canvas activity and changes
CREATE TABLE canvas_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    canvas_id TEXT NOT NULL,
    widget_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    widget_type TEXT NOT NULL,
    content_preview TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for canvas_events
CREATE INDEX idx_canvas_events_canvas_id ON canvas_events(canvas_id);
CREATE INDEX idx_canvas_events_widget_id ON canvas_events(widget_id);
CREATE INDEX idx_canvas_events_created_at ON canvas_events(created_at);

-- performance_metrics: Performance measurements
-- Used for tracking system and inference performance over time
CREATE TABLE performance_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_type TEXT NOT NULL,
    metric_name TEXT NOT NULL,
    metric_value REAL NOT NULL,
    metadata TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance_metrics
CREATE INDEX idx_performance_metrics_metric_type ON performance_metrics(metric_type);
CREATE INDEX idx_performance_metrics_created_at ON performance_metrics(created_at);

-- error_log: Logged errors with context
-- Used for error tracking and debugging
CREATE TABLE error_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    correlation_id TEXT,
    error_type TEXT NOT NULL,
    error_message TEXT NOT NULL,
    stack_trace TEXT,
    context TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for error_log
CREATE INDEX idx_error_log_correlation_id ON error_log(correlation_id);
CREATE INDEX idx_error_log_error_type ON error_log(error_type);
CREATE INDEX idx_error_log_created_at ON error_log(created_at);

-- system_metrics: System resource utilization metrics
-- Used for tracking CPU, memory, disk, and GPU usage
CREATE TABLE system_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_type TEXT NOT NULL,
    cpu_usage REAL,
    memory_used_mb REAL,
    memory_total_mb REAL,
    disk_used_mb REAL,
    disk_total_mb REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for system_metrics
CREATE INDEX idx_system_metrics_metric_type ON system_metrics(metric_type);
CREATE INDEX idx_system_metrics_created_at ON system_metrics(created_at);
