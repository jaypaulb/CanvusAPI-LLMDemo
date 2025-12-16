-- Rollback migration: 001_initial_schema
-- Drops all tables created in the up migration (reverse order)

-- Drop indexes first (SQLite drops indexes automatically with tables, but being explicit)
DROP INDEX IF EXISTS idx_system_metrics_created_at;
DROP INDEX IF EXISTS idx_system_metrics_metric_type;

DROP INDEX IF EXISTS idx_error_log_created_at;
DROP INDEX IF EXISTS idx_error_log_error_type;
DROP INDEX IF EXISTS idx_error_log_correlation_id;

DROP INDEX IF EXISTS idx_performance_metrics_created_at;
DROP INDEX IF EXISTS idx_performance_metrics_metric_type;

DROP INDEX IF EXISTS idx_canvas_events_created_at;
DROP INDEX IF EXISTS idx_canvas_events_widget_id;
DROP INDEX IF EXISTS idx_canvas_events_canvas_id;

DROP INDEX IF EXISTS idx_processing_history_created_at;
DROP INDEX IF EXISTS idx_processing_history_canvas_id;
DROP INDEX IF EXISTS idx_processing_history_correlation_id;

-- Drop tables in reverse order of creation
DROP TABLE IF EXISTS system_metrics;
DROP TABLE IF EXISTS error_log;
DROP TABLE IF EXISTS performance_metrics;
DROP TABLE IF EXISTS canvas_events;
DROP TABLE IF EXISTS processing_history;
