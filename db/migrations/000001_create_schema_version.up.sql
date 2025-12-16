-- Migration 000001: Create schema_version table
-- This table tracks application-level schema version information
-- (separate from golang-migrate's internal schema_migrations table)

CREATE TABLE IF NOT EXISTS schema_version (
    id INTEGER PRIMARY KEY,
    version TEXT NOT NULL,
    description TEXT,
    applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Insert initial version record
INSERT INTO schema_version (version, description) VALUES ('1.0.0', 'Initial schema setup');
