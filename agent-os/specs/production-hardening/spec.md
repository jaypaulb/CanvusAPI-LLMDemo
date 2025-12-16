# Phase 6: Production Hardening Specification

## Executive Summary

This specification details the implementation of production hardening features for CanvusLocalLLM, transforming it from a development-grade prototype into a robust, observable, maintainable production system. The scope includes structured logging with GPU metrics, SQLite persistence, web UI authentication, CI/CD automation, and graceful shutdown handling.

**Implementation Scope:** 5 major features across 28 tasks
**Estimated Timeline:** 3-4 weeks
**Dependencies:** Phases 1-2 (Installation + llama.cpp integration) must be complete

## Goals and Success Criteria

### Primary Goals
1. **Production Observability**: Enable rapid diagnosis of production issues through structured logging and metrics
2. **Operational History**: Maintain persistent records of AI operations for auditing and analysis
3. **Secure Deployment**: Protect administrative interfaces with authentication
4. **Automated Distribution**: Fully automated, reproducible installer builds
5. **Reliability**: Graceful handling of shutdown scenarios without data loss

### Success Criteria
- Operations team can diagnose 90% of issues from logs alone
- Database contains complete history of AI operations
- Web UI requires authentication before displaying data
- Installers built automatically on every release via CI/CD
- Application shuts down cleanly without orphaned resources

## Architecture Overview

### Component Interaction

```
┌─────────────────────────────────────────────────────────────────┐
│                         Main Application                         │
│  ┌────────────┐  ┌──────────────┐  ┌─────────────┐             │
│  │   Monitor  │→→│ AI Handlers  │→→│  llama.cpp  │             │
│  └────────────┘  └──────────────┘  └─────────────┘             │
│         ↓                ↓                  ↓                    │
│  ┌──────────────────────────────────────────────┐              │
│  │         Structured Logger (zap)              │              │
│  │  • JSON formatting                           │              │
│  │  • GPU metrics integration                   │              │
│  │  • Async writes                              │              │
│  └──────────────────────────────────────────────┘              │
│         ↓                                ↓                       │
│  ┌─────────────┐              ┌─────────────────┐              │
│  │  Log Files  │              │  SQLite DB      │              │
│  │  • Rotation │              │  • History      │              │
│  │  • JSON     │              │  • Metrics      │              │
│  └─────────────┘              │  • Events       │              │
│                                └─────────────────┘              │
│                                        ↑                         │
│                                ┌───────────────┐                │
│                                │   Web UI      │                │
│                                │ • Auth        │                │
│                                │ • Dashboard   │                │
│                                └───────────────┘                │
│                                                                  │
│  Shutdown Manager                                               │
│  • Context propagation                                          │
│  • WaitGroup tracking                                           │
│  • Cleanup sequence                                             │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

**Logging Pipeline:**
```
Operation → Structured Log Call → zap Logger → [Console | File | Database]
                                                   ↓        ↓        ↓
                                              Human   Rotate  Persist
```

**AI Processing Pipeline:**
```
Canvas Event → Handler → llama.cpp → Response
      ↓          ↓           ↓           ↓
    Log        Log      GPU Metrics    Log + DB
```

**Shutdown Sequence:**
```
Signal → Context Cancel → Stop New Requests → Wait for In-Flight
            ↓                                        ↓
    Notify All Goroutines                    Flush DB + Logs
            ↓                                        ↓
    WaitGroup.Wait()                         Close Connections
            ↓                                        ↓
    Cleanup Temp Files                         Exit
```

## Feature 1: Structured Logging with zap

### Decision: zap over logrus

**Rationale:**
- **Performance**: zap is 4-10x faster than logrus in benchmarks, critical for high-throughput logging
- **Allocation**: Near-zero allocations with structured fields, reduces GC pressure
- **JSON-first**: Designed for structured logging, not retrofitted
- **Type-safety**: Compile-time field type checking prevents runtime errors
- **Maturity**: Production-proven at Uber and widely adopted

**Tradeoffs:**
- More verbose API than logrus (explicit field types)
- Slightly steeper learning curve
- Worth it for performance gains in production

### Implementation Design

#### Logger Initialization

**Location:** `logging/logger.go` (replace existing `logging.go`)

```go
package logging

import (
    "os"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
    zap *zap.Logger
    sugar *zap.SugaredLogger
}

func NewLogger(isDevelopment bool, logFilePath string) (*Logger, error) {
    // Core configuration
    encoderConfig := zapcore.EncoderConfig{
        TimeKey:        "timestamp",
        LevelKey:       "level",
        NameKey:        "logger",
        CallerKey:      "source",
        MessageKey:     "message",
        StacktraceKey:  "stacktrace",
        LineEnding:     zapcore.DefaultLineEnding,
        EncodeLevel:    zapcore.LowercaseLevelEncoder,
        EncodeTime:     zapcore.ISO8601TimeEncoder,
        EncodeDuration: zapcore.StringDurationEncoder,
        EncodeCaller:   zapcore.ShortCallerEncoder,
    }

    // Console encoder (human-readable for development)
    consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)

    // JSON encoder (machine-parseable for production)
    jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)

    // Log level from environment
    level := zapcore.InfoLevel
    if levelStr := os.Getenv("CANVUSLLM_LOG_LEVEL"); levelStr != "" {
        level.UnmarshalText([]byte(levelStr))
    }

    // File writer with rotation
    fileWriter := zapcore.AddSync(&lumberjack.Logger{
        Filename:   logFilePath,
        MaxSize:    100, // MB
        MaxBackups: 5,
        MaxAge:     30, // days
        Compress:   true,
    })

    // Console writer
    consoleWriter := zapcore.AddSync(os.Stdout)

    // Core with multiple outputs
    var core zapcore.Core
    if isDevelopment {
        // Development: console (human-readable) + file (JSON)
        core = zapcore.NewTee(
            zapcore.NewCore(consoleEncoder, consoleWriter, level),
            zapcore.NewCore(jsonEncoder, fileWriter, level),
        )
    } else {
        // Production: JSON to both console and file
        core = zapcore.NewTee(
            zapcore.NewCore(jsonEncoder, consoleWriter, level),
            zapcore.NewCore(jsonEncoder, fileWriter, level),
        )
    }

    // Build logger
    zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

    return &Logger{
        zap: zapLogger,
        sugar: zapLogger.Sugar(),
    }, nil
}

func (l *Logger) Sync() error {
    return l.zap.Sync()
}

// Structured logging methods
func (l *Logger) Info(msg string, fields ...zap.Field) {
    l.zap.Info(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
    l.zap.Error(msg, fields...)
}

func (l *Logger) Debug(msg string, fields ...zap.Field) {
    l.zap.Debug(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
    l.zap.Warn(msg, fields...)
}

// Sugar for compatibility (backward compatibility during migration)
func (l *Logger) Infof(template string, args ...interface{}) {
    l.sugar.Infof(template, args...)
}
```

#### GPU Metrics Integration

**Location:** `logging/gpu_metrics.go` (new)

```go
package logging

import (
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

// GPUMetrics represents GPU utilization data
type GPUMetrics struct {
    VRAMUsedMB        int64   `json:"vram_used_mb"`
    VRAMTotalMB       int64   `json:"vram_total_mb"`
    GPUUtilization    int     `json:"gpu_utilization_percent"`
    Temperature       int     `json:"temperature_celsius,omitempty"`
}

// MarshalLogObject implements zapcore.ObjectMarshaler for structured logging
func (g GPUMetrics) MarshalLogObject(enc zapcore.ObjectEncoder) error {
    enc.AddInt64("vram_used_mb", g.VRAMUsedMB)
    enc.AddInt64("vram_total_mb", g.VRAMTotalMB)
    enc.AddInt("gpu_utilization_percent", g.GPUUtilization)
    if g.Temperature > 0 {
        enc.AddInt("temperature_celsius", g.Temperature)
    }
    return nil
}

// InferenceMetrics represents llama.cpp inference performance
type InferenceMetrics struct {
    ModelName           string  `json:"model_name"`
    PromptTokens        int     `json:"prompt_tokens"`
    CompletionTokens    int     `json:"completion_tokens"`
    TotalTokens         int     `json:"total_tokens"`
    PromptEvalTimeMs    int64   `json:"prompt_eval_time_ms"`
    TokenGenTimeMs      int64   `json:"token_gen_time_ms"`
    TotalDurationMs     int64   `json:"total_duration_ms"`
    TokensPerSecond     float64 `json:"tokens_per_second"`
    GPU                 *GPUMetrics `json:"gpu,omitempty"`
}

// MarshalLogObject implements zapcore.ObjectMarshaler
func (i InferenceMetrics) MarshalLogObject(enc zapcore.ObjectEncoder) error {
    enc.AddString("model_name", i.ModelName)
    enc.AddInt("prompt_tokens", i.PromptTokens)
    enc.AddInt("completion_tokens", i.CompletionTokens)
    enc.AddInt("total_tokens", i.TotalTokens)
    enc.AddInt64("prompt_eval_time_ms", i.PromptEvalTimeMs)
    enc.AddInt64("token_gen_time_ms", i.TokenGenTimeMs)
    enc.AddInt64("total_duration_ms", i.TotalDurationMs)
    enc.AddFloat64("tokens_per_second", i.TokensPerSecond)
    if i.GPU != nil {
        enc.AddObject("gpu", i.GPU)
    }
    return nil
}

// Helper to create inference log fields
func InferenceFields(metrics InferenceMetrics) zap.Field {
    return zap.Object("inference", metrics)
}
```

#### Usage Example in Handlers

**Before (handlers.go):**
```go
logging.LogHandler("📝 Processing note prompt: %s", prompt)
```

**After (handlers.go):**
```go
logger.Info("Processing note prompt",
    zap.String("widget_id", widgetID),
    zap.String("canvas_id", canvasID),
    zap.String("prompt", truncatePrompt(prompt, 200)),
    zap.String("correlation_id", correlationID),
)
```

**With GPU Metrics (handlers.go):**
```go
startTime := time.Now()
response, gpuMetrics := invokeInference(prompt) // llama.cpp wrapper returns GPU stats
duration := time.Since(startTime)

logger.Info("Inference completed",
    InferenceFields(InferenceMetrics{
        ModelName:        "bunny-v1.1-llama-3-8b-v",
        PromptTokens:     len(prompt)/4, // Estimate
        CompletionTokens: len(response)/4,
        TotalDurationMs:  duration.Milliseconds(),
        TokensPerSecond:  float64(len(response)/4) / duration.Seconds(),
        GPU:              gpuMetrics,
    }),
)
```

#### Log Output Examples

**Console (Development):**
```
2025-01-15T10:23:45.123Z    INFO    Processing note prompt    {"widget_id": "w123", "canvas_id": "c456", "prompt": "Summarize this document..."}
2025-01-15T10:23:47.456Z    INFO    Inference completed    {"inference": {"model_name": "bunny-v1.1", "prompt_tokens": 150, "completion_tokens": 75, "total_duration_ms": 2333, "tokens_per_second": 32.1, "gpu": {"vram_used_mb": 4096, "gpu_utilization_percent": 87}}}
```

**File (JSON):**
```json
{
  "timestamp": "2025-01-15T10:23:45.123Z",
  "level": "info",
  "message": "Processing note prompt",
  "source": "handlers.go:156",
  "widget_id": "w123",
  "canvas_id": "c456",
  "prompt": "Summarize this document...",
  "correlation_id": "req-abc123"
}
{
  "timestamp": "2025-01-15T10:23:47.456Z",
  "level": "info",
  "message": "Inference completed",
  "source": "handlers.go:178",
  "inference": {
    "model_name": "bunny-v1.1-llama-3-8b-v",
    "prompt_tokens": 150,
    "completion_tokens": 75,
    "total_tokens": 225,
    "total_duration_ms": 2333,
    "tokens_per_second": 32.1,
    "gpu": {
      "vram_used_mb": 4096,
      "vram_total_mb": 8192,
      "gpu_utilization_percent": 87
    }
  }
}
```

### Migration Strategy

**Phase 1: Add zap alongside existing logging**
- Install zap: `go get -u go.uber.org/zap`
- Create `logging/logger.go` with new implementation
- Keep existing `LogHandler` temporarily

**Phase 2: Migrate high-value paths first**
- AI inference handlers (capture GPU metrics)
- Canvas monitoring (capture widget events)
- HTTP handlers (capture request/response)

**Phase 3: Global replacement**
- Replace all `logging.LogHandler` calls
- Remove old `logging.go`
- Update `main.go` to initialize zap logger

## Feature 2: SQLite Database Persistence

### Database Schema

**Location:** `db/migrations/001_initial_schema.sql`

```sql
-- Processing history: All AI operations
CREATE TABLE IF NOT EXISTS processing_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    correlation_id TEXT NOT NULL,
    canvas_id TEXT NOT NULL,
    widget_id TEXT,
    operation_type TEXT NOT NULL, -- 'note', 'pdf', 'canvas_analysis', 'image_gen'
    prompt TEXT NOT NULL,
    response TEXT,
    model_name TEXT NOT NULL,
    prompt_tokens INTEGER,
    completion_tokens INTEGER,
    total_tokens INTEGER,
    duration_ms INTEGER NOT NULL,
    status TEXT NOT NULL, -- 'success', 'error', 'timeout'
    error_message TEXT,

    -- Indexes for common queries
    INDEX idx_created_at (created_at),
    INDEX idx_canvas_id (canvas_id),
    INDEX idx_correlation_id (correlation_id),
    INDEX idx_status (status)
);

-- Canvas events: Widget updates monitored
CREATE TABLE IF NOT EXISTS canvas_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    canvas_id TEXT NOT NULL,
    widget_id TEXT NOT NULL,
    event_type TEXT NOT NULL, -- 'created', 'updated', 'deleted'
    widget_type TEXT NOT NULL, -- 'note', 'image', 'pdf'
    content_preview TEXT, -- First 200 chars

    INDEX idx_created_at (created_at),
    INDEX idx_canvas_id (canvas_id),
    INDEX idx_widget_id (widget_id)
);

-- Performance metrics: Aggregated statistics
CREATE TABLE IF NOT EXISTS performance_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    recorded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metric_type TEXT NOT NULL, -- 'inference', 'gpu', 'system'
    metric_name TEXT NOT NULL,
    metric_value REAL NOT NULL,
    metadata TEXT, -- JSON for additional context

    INDEX idx_recorded_at (recorded_at),
    INDEX idx_metric_type (metric_type)
);

-- Error log: Detailed error tracking
CREATE TABLE IF NOT EXISTS error_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    correlation_id TEXT,
    error_type TEXT NOT NULL, -- 'ai_inference', 'canvas_api', 'database', 'system'
    error_message TEXT NOT NULL,
    stack_trace TEXT,
    context TEXT, -- JSON with additional context

    INDEX idx_created_at (created_at),
    INDEX idx_error_type (error_type)
);

-- System metrics: Application health
CREATE TABLE IF NOT EXISTS system_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    recorded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    uptime_seconds INTEGER NOT NULL,
    memory_used_mb INTEGER NOT NULL,
    goroutines_count INTEGER NOT NULL,
    active_requests INTEGER NOT NULL,

    INDEX idx_recorded_at (recorded_at)
);
```

### Database Package Implementation

**Location:** `db/database.go`

```go
package db

import (
    "database/sql"
    "fmt"
    "os"
    "path/filepath"
    "time"

    _ "github.com/mattn/go-sqlite3"
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/sqlite3"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

type Database struct {
    db *sql.DB
    asyncWriter chan func() // Channel for async writes
}

// NewDatabase creates and initializes the database
func NewDatabase(dataDir string) (*Database, error) {
    // Ensure data directory exists
    if err := os.MkdirAll(dataDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create data directory: %w", err)
    }

    dbPath := filepath.Join(dataDir, "data.db")

    // Open database
    db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }

    // Configure connection pool
    db.SetMaxOpenConns(1) // SQLite doesn't benefit from multiple connections
    db.SetMaxIdleConns(1)
    db.SetConnMaxLifetime(0)

    // Run migrations
    if err := runMigrations(db); err != nil {
        return nil, fmt.Errorf("failed to run migrations: %w", err)
    }

    // Create async writer channel
    asyncWriter := make(chan func(), 100) // Buffer up to 100 writes

    database := &Database{
        db: db,
        asyncWriter: asyncWriter,
    }

    // Start async writer goroutine
    go database.processAsyncWrites()

    return database, nil
}

func runMigrations(db *sql.DB) error {
    driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
    if err != nil {
        return err
    }

    m, err := migrate.NewWithDatabaseInstance(
        "file://db/migrations",
        "sqlite3", driver)
    if err != nil {
        return err
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return err
    }

    return nil
}

func (d *Database) processAsyncWrites() {
    for writeFn := range d.asyncWriter {
        writeFn()
    }
}

// Close gracefully closes the database
func (d *Database) Close() error {
    close(d.asyncWriter) // Signal async writer to stop
    time.Sleep(100 * time.Millisecond) // Allow pending writes to complete
    return d.db.Close()
}

// InsertProcessingHistory records an AI operation
func (d *Database) InsertProcessingHistory(h ProcessingHistory) error {
    query := `
        INSERT INTO processing_history (
            correlation_id, canvas_id, widget_id, operation_type,
            prompt, response, model_name, prompt_tokens, completion_tokens,
            total_tokens, duration_ms, status, error_message
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

    // Async write
    d.asyncWriter <- func() {
        _, err := d.db.Exec(query,
            h.CorrelationID, h.CanvasID, h.WidgetID, h.OperationType,
            h.Prompt, h.Response, h.ModelName, h.PromptTokens, h.CompletionTokens,
            h.TotalTokens, h.DurationMs, h.Status, h.ErrorMessage,
        )
        if err != nil {
            // Log error (can't return from async write)
            fmt.Printf("Failed to insert processing history: %v\n", err)
        }
    }

    return nil
}

// ProcessingHistory represents an AI operation record
type ProcessingHistory struct {
    CorrelationID    string
    CanvasID         string
    WidgetID         string
    OperationType    string
    Prompt           string
    Response         string
    ModelName        string
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
    DurationMs       int64
    Status           string
    ErrorMessage     string
}

// QueryRecentHistory returns recent AI operations
func (d *Database) QueryRecentHistory(limit int) ([]ProcessingHistory, error) {
    query := `
        SELECT correlation_id, canvas_id, widget_id, operation_type,
               prompt, response, model_name, prompt_tokens, completion_tokens,
               total_tokens, duration_ms, status, error_message, created_at
        FROM processing_history
        ORDER BY created_at DESC
        LIMIT ?
    `

    rows, err := d.db.Query(query, limit)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []ProcessingHistory
    for rows.Next() {
        var h ProcessingHistory
        var createdAt string
        err := rows.Scan(
            &h.CorrelationID, &h.CanvasID, &h.WidgetID, &h.OperationType,
            &h.Prompt, &h.Response, &h.ModelName, &h.PromptTokens, &h.CompletionTokens,
            &h.TotalTokens, &h.DurationMs, &h.Status, &h.ErrorMessage, &createdAt,
        )
        if err != nil {
            return nil, err
        }
        results = append(results, h)
    }

    return results, rows.Err()
}

// Cleanup deletes old records based on retention policy
func (d *Database) Cleanup(retentionDays int) error {
    cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

    queries := []string{
        "DELETE FROM processing_history WHERE created_at < ?",
        "DELETE FROM canvas_events WHERE created_at < ?",
        "DELETE FROM performance_metrics WHERE recorded_at < ?",
        "DELETE FROM error_log WHERE created_at < ?",
    }

    for _, query := range queries {
        if _, err := d.db.Exec(query, cutoffDate); err != nil {
            return fmt.Errorf("cleanup failed for query %s: %w", query, err)
        }
    }

    // Vacuum to reclaim space
    if _, err := d.db.Exec("VACUUM"); err != nil {
        return fmt.Errorf("vacuum failed: %w", err)
    }

    return nil
}
```

### Database Integration Points

**main.go:**
```go
// Initialize database
dataDir := getDataDirectory() // Platform-specific
db, err := db.NewDatabase(dataDir)
if err != nil {
    logger.Fatal("Failed to initialize database", zap.Error(err))
}
defer db.Close()

// Schedule cleanup job
go func() {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    for range ticker.C {
        if err := db.Cleanup(30); err != nil {
            logger.Error("Database cleanup failed", zap.Error(err))
        }
    }
}()
```

**handlers.go (after inference):**
```go
// Record operation in database
db.InsertProcessingHistory(db.ProcessingHistory{
    CorrelationID:    correlationID,
    CanvasID:         canvasID,
    WidgetID:         widgetID,
    OperationType:    "note",
    Prompt:           prompt,
    Response:         response,
    ModelName:        "bunny-v1.1-llama-3-8b-v",
    PromptTokens:     promptTokens,
    CompletionTokens: completionTokens,
    TotalTokens:      totalTokens,
    DurationMs:       duration.Milliseconds(),
    Status:           "success",
})
```

### Data Directory Location

**Platform-specific paths:**
- **Windows**: `%APPDATA%\CanvusLocalLLM\` (e.g., `C:\Users\username\AppData\Roaming\CanvusLocalLLM\`)
- **Linux**: `~/.canvuslocallm/` (e.g., `/home/username/.canvuslocallm/`)

**Implementation:**
```go
func getDataDirectory() string {
    if runtime.GOOS == "windows" {
        return filepath.Join(os.Getenv("APPDATA"), "CanvusLocalLLM")
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".canvuslocallm")
}
```

## Feature 3: Web UI Authentication

### Design Overview

**Authentication Method:** Password-based with session cookies
**Storage:** In-memory session store (Redis/persistent storage is future enhancement)
**Security:** bcrypt password hashing, httpOnly cookies, rate limiting

### Implementation

**Location:** `webui/auth.go` (new)

```go
package webui

import (
    "crypto/rand"
    "encoding/base64"
    "net/http"
    "sync"
    "time"

    "golang.org/x/crypto/bcrypt"
    "go.uber.org/zap"
)

type AuthMiddleware struct {
    passwordHash []byte
    sessions     map[string]*Session
    mu           sync.RWMutex
    logger       *zap.Logger
    rateLimiter  *RateLimiter
}

type Session struct {
    ID        string
    CreatedAt time.Time
    ExpiresAt time.Time
}

func NewAuthMiddleware(password string, logger *zap.Logger) (*AuthMiddleware, error) {
    if password == "" {
        return nil, fmt.Errorf("WEBUI_PWD must be set")
    }

    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return nil, fmt.Errorf("failed to hash password: %w", err)
    }

    am := &AuthMiddleware{
        passwordHash: hash,
        sessions:     make(map[string]*Session),
        logger:       logger,
        rateLimiter:  NewRateLimiter(5, time.Minute), // 5 attempts per minute
    }

    // Start session cleanup goroutine
    go am.cleanupExpiredSessions()

    return am, nil
}

func (am *AuthMiddleware) cleanupExpiredSessions() {
    ticker := time.NewTicker(5 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        am.mu.Lock()
        now := time.Now()
        for id, session := range am.sessions {
            if now.After(session.ExpiresAt) {
                delete(am.sessions, id)
            }
        }
        am.mu.Unlock()
    }
}

func (am *AuthMiddleware) LoginHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == "GET" {
        am.renderLoginPage(w)
        return
    }

    // POST - handle login
    password := r.FormValue("password")
    clientIP := r.RemoteAddr

    // Rate limiting
    if !am.rateLimiter.Allow(clientIP) {
        am.logger.Warn("Rate limit exceeded for login",
            zap.String("ip", clientIP),
        )
        http.Error(w, "Too many login attempts. Try again later.", http.StatusTooManyRequests)
        return
    }

    // Verify password (timing-safe comparison via bcrypt)
    err := bcrypt.CompareHashAndPassword(am.passwordHash, []byte(password))
    if err != nil {
        am.logger.Warn("Failed login attempt",
            zap.String("ip", clientIP),
        )
        time.Sleep(1 * time.Second) // Slow down brute force
        http.Error(w, "Invalid password", http.StatusUnauthorized)
        return
    }

    // Create session
    sessionID, err := am.createSession()
    if err != nil {
        http.Error(w, "Internal error", http.StatusInternalServerError)
        return
    }

    // Set cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "session_id",
        Value:    sessionID,
        Path:     "/",
        HttpOnly: true,
        Secure:   r.TLS != nil, // Only set Secure if HTTPS
        SameSite: http.SameSiteStrictMode,
        MaxAge:   int((24 * time.Hour).Seconds()),
    })

    am.logger.Info("Successful login",
        zap.String("ip", clientIP),
        zap.String("session_id", sessionID),
    )

    // Redirect to dashboard
    http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (am *AuthMiddleware) createSession() (string, error) {
    // Generate cryptographically secure session ID
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    sessionID := base64.URLEncoding.EncodeToString(b)

    am.mu.Lock()
    am.sessions[sessionID] = &Session{
        ID:        sessionID,
        CreatedAt: time.Now(),
        ExpiresAt: time.Now().Add(24 * time.Hour),
    }
    am.mu.Unlock()

    return sessionID, nil
}

func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Skip auth for login page
        if r.URL.Path == "/login" {
            next.ServeHTTP(w, r)
            return
        }

        // Check for session cookie
        cookie, err := r.Cookie("session_id")
        if err != nil {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }

        // Validate session
        am.mu.RLock()
        session, exists := am.sessions[cookie.Value]
        am.mu.RUnlock()

        if !exists || time.Now().After(session.ExpiresAt) {
            http.Redirect(w, r, "/login", http.StatusSeeOther)
            return
        }

        // Valid session - proceed
        next.ServeHTTP(w, r)
    })
}

func (am *AuthMiddleware) LogoutHandler(w http.ResponseWriter, r *http.Request) {
    cookie, err := r.Cookie("session_id")
    if err == nil {
        am.mu.Lock()
        delete(am.sessions, cookie.Value)
        am.mu.Unlock()
    }

    // Clear cookie
    http.SetCookie(w, &http.Cookie{
        Name:   "session_id",
        Value:  "",
        Path:   "/",
        MaxAge: -1,
    })

    http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (am *AuthMiddleware) renderLoginPage(w http.ResponseWriter) {
    html := `
<!DOCTYPE html>
<html>
<head>
    <title>CanvusLocalLLM - Login</title>
    <style>
        body {
            font-family: system-ui, -apple-system, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        }
        .login-container {
            background: white;
            padding: 2rem;
            border-radius: 8px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
            width: 100%;
            max-width: 400px;
        }
        h1 { margin-top: 0; color: #333; }
        input[type="password"] {
            width: 100%;
            padding: 0.75rem;
            border: 1px solid #ddd;
            border-radius: 4px;
            font-size: 1rem;
            box-sizing: border-box;
        }
        button {
            width: 100%;
            padding: 0.75rem;
            background: #667eea;
            color: white;
            border: none;
            border-radius: 4px;
            font-size: 1rem;
            cursor: pointer;
            margin-top: 1rem;
        }
        button:hover { background: #5568d3; }
    </style>
</head>
<body>
    <div class="login-container">
        <h1>CanvusLocalLLM</h1>
        <p>Enter password to access dashboard</p>
        <form method="POST">
            <input type="password" name="password" placeholder="Password" required autofocus>
            <button type="submit">Login</button>
        </form>
    </div>
</body>
</html>
    `
    w.Header().Set("Content-Type", "text/html")
    w.Write([]byte(html))
}

// RateLimiter implements simple IP-based rate limiting
type RateLimiter struct {
    attempts map[string]*AttemptRecord
    mu       sync.Mutex
    maxAttempts int
    window   time.Duration
}

type AttemptRecord struct {
    Count     int
    ResetAt   time.Time
}

func NewRateLimiter(maxAttempts int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        attempts: make(map[string]*AttemptRecord),
        maxAttempts: maxAttempts,
        window:   window,
    }
}

func (rl *RateLimiter) Allow(key string) bool {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    now := time.Now()
    record, exists := rl.attempts[key]

    if !exists || now.After(record.ResetAt) {
        rl.attempts[key] = &AttemptRecord{
            Count:   1,
            ResetAt: now.Add(rl.window),
        }
        return true
    }

    if record.Count >= rl.maxAttempts {
        return false
    }

    record.Count++
    return true
}
```

**Integration in main.go:**
```go
// Initialize auth middleware
authMiddleware, err := webui.NewAuthMiddleware(config.WebUIPassword, logger)
if err != nil {
    logger.Fatal("Failed to initialize authentication", zap.Error(err))
}

// Setup HTTP routes with authentication
mux := http.NewServeMux()
mux.HandleFunc("/login", authMiddleware.LoginHandler)
mux.HandleFunc("/logout", authMiddleware.LogoutHandler)

// Protected routes
protected := http.NewServeMux()
protected.HandleFunc("/", dashboardHandler)
protected.HandleFunc("/api/history", historyAPIHandler)

// Wrap protected routes with auth middleware
mux.Handle("/", authMiddleware.Middleware(protected))

// Start server
server := &http.Server{
    Addr:    fmt.Sprintf(":%d", config.Port),
    Handler: mux,
}
```

## Feature 4: CI/CD Installer Automation

### GitHub Actions Workflow

**Location:** `.github/workflows/release.yml`

```yaml
name: Build and Release Installers

on:
  push:
    tags:
      - 'v*.*.*'
  workflow_dispatch:
    inputs:
      version:
        description: 'Version tag (e.g., v1.0.0)'
        required: true

jobs:
  build-windows:
    runs-on: windows-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'

      - name: Install CUDA Toolkit
        uses: Jimver/cuda-toolkit@v0.2.11
        with:
          cuda: '12.2.0'
          method: 'network'

      - name: Build llama.cpp with CUDA
        run: |
          git clone https://github.com/ggerganov/llama.cpp
          cd llama.cpp
          mkdir build
          cd build
          cmake .. -DLLAMA_CUBLAS=ON -DCMAKE_BUILD_TYPE=Release
          cmake --build . --config Release
          # Copy built libraries to project lib/
          cp Release/llama.dll ../../lib/
          cp Release/llama.lib ../../lib/
        shell: powershell

      - name: Build stable-diffusion.cpp with CUDA
        run: |
          git clone https://github.com/leejet/stable-diffusion.cpp
          cd stable-diffusion.cpp
          mkdir build
          cd build
          cmake .. -DSD_CUBLAS=ON -DCMAKE_BUILD_TYPE=Release
          cmake --build . --config Release
          cp Release/stable-diffusion.dll ../../lib/
          cp Release/stable-diffusion.lib ../../lib/
        shell: powershell

      - name: Download Bunny model
        run: |
          mkdir models
          # Download from Hugging Face (placeholder - actual implementation)
          # curl -L -o models/bunny-v1.1.gguf https://huggingface.co/.../bunny-v1.1.gguf
        shell: powershell

      - name: Build Go application
        run: |
          $env:CGO_ENABLED=1
          go build -o CanvusLocalLLM.exe -ldflags "-X main.Version=${{ github.ref_name }}" .
        shell: powershell

      - name: Install NSIS
        run: choco install nsis -y
        shell: powershell

      - name: Build NSIS installer
        run: |
          makensis installer/windows/setup.nsi
        shell: powershell

      - name: Generate checksums
        run: |
          certutil -hashfile CanvusLocalLLM-Setup.exe SHA256 > CanvusLocalLLM-Setup.exe.sha256
        shell: powershell

      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: windows-installer
          path: |
            CanvusLocalLLM-Setup.exe
            CanvusLocalLLM-Setup.exe.sha256

  build-linux:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version-file: 'go.mod'

      - name: Install CUDA Toolkit
        run: |
          wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/cuda-keyring_1.0-1_all.deb
          sudo dpkg -i cuda-keyring_1.0-1_all.deb
          sudo apt-get update
          sudo apt-get -y install cuda-toolkit-12-2

      - name: Build llama.cpp with CUDA
        run: |
          git clone https://github.com/ggerganov/llama.cpp
          cd llama.cpp
          mkdir build && cd build
          cmake .. -DLLAMA_CUBLAS=ON -DCMAKE_BUILD_TYPE=Release
          cmake --build . --config Release
          cp libllama.so ../../lib/

      - name: Build stable-diffusion.cpp with CUDA
        run: |
          git clone https://github.com/leejet/stable-diffusion.cpp
          cd stable-diffusion.cpp
          mkdir build && cd build
          cmake .. -DSD_CUBLAS=ON -DCMAKE_BUILD_TYPE=Release
          cmake --build . --config Release
          cp libstable-diffusion.so ../../lib/

      - name: Download Bunny model
        run: |
          mkdir -p models
          # Download from Hugging Face

      - name: Build Go application
        run: |
          CGO_ENABLED=1 go build -o canvuslocallm -ldflags "-X main.Version=${{ github.ref_name }}" .

      - name: Build Debian package
        run: |
          ./installer/linux/build-deb.sh ${{ github.ref_name }}

      - name: Build tarball
        run: |
          ./installer/linux/build-tarball.sh ${{ github.ref_name }}

      - name: Generate checksums
        run: |
          sha256sum canvuslocallm_*.deb > canvuslocallm.deb.sha256
          sha256sum canvuslocallm-*.tar.gz > canvuslocallm.tar.gz.sha256

      - name: Upload artifacts
        uses: actions/upload-artifact@v4
        with:
          name: linux-packages
          path: |
            canvuslocallm_*.deb
            canvuslocallm-*.tar.gz
            *.sha256

  release:
    needs: [build-windows, build-linux]
    runs-on: ubuntu-latest
    steps:
      - name: Download all artifacts
        uses: actions/download-artifact@v4

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            windows-installer/*
            linux-packages/*
          body: |
            ## CanvusLocalLLM ${{ github.ref_name }}

            ### Installation
            - **Windows**: Download `CanvusLocalLLM-Setup.exe` and run installer
            - **Linux (Debian/Ubuntu)**: Download `.deb` and install with `dpkg -i`
            - **Linux (Other)**: Download `.tar.gz` and extract

            ### Checksums
            Verify downloads using provided `.sha256` files.

            ### Requirements
            - NVIDIA RTX GPU with CUDA 12.0+ drivers
            - Windows 10+ or Ubuntu 20.04+

            Full release notes: [CHANGELOG.md](CHANGELOG.md)
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Build Scripts

**installer/linux/build-deb.sh:**
```bash
#!/bin/bash
set -e

VERSION=$1
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    exit 1
fi

# Strip 'v' prefix
VERSION_NUM=${VERSION#v}

# Create package structure
mkdir -p build/deb/DEBIAN
mkdir -p build/deb/opt/canvuslocallm/{bin,lib,models}

# Copy files
cp canvuslocallm build/deb/opt/canvuslocallm/bin/
cp lib/*.so build/deb/opt/canvuslocallm/lib/
cp models/* build/deb/opt/canvuslocallm/models/
cp .env.example build/deb/opt/canvuslocallm/
cp README.md build/deb/opt/canvuslocallm/

# Create control file
cat > build/deb/DEBIAN/control <<EOF
Package: canvuslocallm
Version: ${VERSION_NUM}
Section: utils
Priority: optional
Architecture: amd64
Depends: libc6 (>= 2.31)
Maintainer: Your Name <your.email@example.com>
Description: Local LLM integration for Canvus workspaces
 CanvusLocalLLM provides AI capabilities for Canvus using embedded llama.cpp
 and Bunny v1.1 model with CUDA acceleration.
EOF

# Create postinst script
cat > build/deb/DEBIAN/postinst <<'EOF'
#!/bin/bash
set -e

# Create data directory
mkdir -p ~/.canvuslocallm

# Set permissions
chmod +x /opt/canvuslocallm/bin/canvuslocallm

echo "CanvusLocalLLM installed successfully!"
echo "Configure .env file in /opt/canvuslocallm/ and run:"
echo "  /opt/canvuslocallm/bin/canvuslocallm"
EOF

chmod +x build/deb/DEBIAN/postinst

# Build package
dpkg-deb --build build/deb canvuslocallm_${VERSION_NUM}_amd64.deb

echo "✓ Debian package built: canvuslocallm_${VERSION_NUM}_amd64.deb"
```

**installer/windows/setup.nsi:**
```nsis
; CanvusLocalLLM Windows Installer Script

!include "MUI2.nsh"

Name "CanvusLocalLLM"
OutFile "CanvusLocalLLM-Setup.exe"
InstallDir "$PROGRAMFILES64\CanvusLocalLLM"
RequestExecutionLevel admin

; Modern UI Configuration
!insertmacro MUI_PAGE_LICENSE "LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

; Installation Section
Section "Install"
    SetOutPath "$INSTDIR"

    ; Copy application files
    File "CanvusLocalLLM.exe"
    File ".env.example"
    File "README.md"
    File "LICENSE"

    ; Copy libraries
    SetOutPath "$INSTDIR\lib"
    File "lib\llama.dll"
    File "lib\stable-diffusion.dll"

    ; Copy models
    SetOutPath "$INSTDIR\models"
    File "models\bunny-v1.1.gguf"

    ; Create uninstaller
    WriteUninstaller "$INSTDIR\Uninstall.exe"

    ; Add to Add/Remove Programs
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CanvusLocalLLM" \
                     "DisplayName" "CanvusLocalLLM"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CanvusLocalLLM" \
                     "UninstallString" "$INSTDIR\Uninstall.exe"

    MessageBox MB_OK "Installation complete! Configure .env file before running."
SectionEnd

; Uninstallation Section
Section "Uninstall"
    Delete "$INSTDIR\CanvusLocalLLM.exe"
    Delete "$INSTDIR\.env.example"
    Delete "$INSTDIR\README.md"
    Delete "$INSTDIR\LICENSE"
    RMDir /r "$INSTDIR\lib"
    RMDir /r "$INSTDIR\models"
    Delete "$INSTDIR\Uninstall.exe"
    RMDir "$INSTDIR"

    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\CanvusLocalLLM"
SectionEnd
```

## Feature 5: Graceful Shutdown

### Design Overview

**Shutdown Triggers:**
- SIGINT (Ctrl+C)
- SIGTERM (systemd, Windows Service)
- Context cancellation (programmatic)

**Shutdown Sequence:**
1. Stop accepting new requests immediately
2. Wait for in-flight operations (with timeout)
3. Flush database writes
4. Cleanup temporary files
5. Close connections
6. Log shutdown completion
7. Exit with appropriate code

### Implementation

**Location:** `shutdown/manager.go` (new package)

```go
package shutdown

import (
    "context"
    "fmt"
    "os"
    "sync"
    "time"

    "go.uber.org/zap"
)

type Manager struct {
    logger         *zap.Logger
    wg             sync.WaitGroup
    shutdownFuncs  []ShutdownFunc
    timeout        time.Duration
    mu             sync.Mutex
}

type ShutdownFunc func(ctx context.Context) error

func NewManager(logger *zap.Logger, timeout time.Duration) *Manager {
    return &Manager{
        logger:  logger,
        timeout: timeout,
        shutdownFuncs: make([]ShutdownFunc, 0),
    }
}

// RegisterShutdownFunc registers a function to be called during shutdown
func (m *Manager) RegisterShutdownFunc(name string, fn ShutdownFunc) {
    m.mu.Lock()
    defer m.mu.Unlock()

    m.shutdownFuncs = append(m.shutdownFuncs, func(ctx context.Context) error {
        m.logger.Info("Running shutdown function", zap.String("name", name))
        if err := fn(ctx); err != nil {
            m.logger.Error("Shutdown function failed",
                zap.String("name", name),
                zap.Error(err),
            )
            return err
        }
        m.logger.Info("Shutdown function completed", zap.String("name", name))
        return nil
    })
}

// TrackOperation increments the wait group for an in-flight operation
func (m *Manager) TrackOperation() {
    m.wg.Add(1)
}

// CompleteOperation decrements the wait group when operation completes
func (m *Manager) CompleteOperation() {
    m.wg.Done()
}

// Shutdown executes graceful shutdown sequence
func (m *Manager) Shutdown(ctx context.Context) error {
    startTime := time.Now()
    m.logger.Info("Initiating graceful shutdown",
        zap.Duration("timeout", m.timeout),
    )

    // Create timeout context for shutdown operations
    shutdownCtx, cancel := context.WithTimeout(ctx, m.timeout)
    defer cancel()

    // Wait for in-flight operations with timeout
    done := make(chan struct{})
    go func() {
        m.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        m.logger.Info("All in-flight operations completed")
    case <-shutdownCtx.Done():
        m.logger.Warn("Timeout waiting for in-flight operations",
            zap.Duration("waited", time.Since(startTime)),
        )
    }

    // Execute registered shutdown functions
    var shutdownErrors []error
    for _, fn := range m.shutdownFuncs {
        if err := fn(shutdownCtx); err != nil {
            shutdownErrors = append(shutdownErrors, err)
        }
    }

    duration := time.Since(startTime)
    if len(shutdownErrors) > 0 {
        m.logger.Error("Shutdown completed with errors",
            zap.Duration("duration", duration),
            zap.Int("error_count", len(shutdownErrors)),
        )
        return fmt.Errorf("shutdown had %d errors", len(shutdownErrors))
    }

    m.logger.Info("Graceful shutdown completed",
        zap.Duration("duration", duration),
    )

    return nil
}
```

**Integration in main.go:**
```go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"

    "go_backend/shutdown"
    "go.uber.org/zap"
)

func main() {
    // ... existing initialization ...

    // Create shutdown manager
    shutdownManager := shutdown.NewManager(logger, 60*time.Second)

    // Register shutdown functions
    shutdownManager.RegisterShutdownFunc("database", func(ctx context.Context) error {
        return db.Close()
    })

    shutdownManager.RegisterShutdownFunc("logger", func(ctx context.Context) error {
        return logger.Sync()
    })

    shutdownManager.RegisterShutdownFunc("cleanup_temp_files", func(ctx context.Context) error {
        return os.RemoveAll(config.DownloadsDir)
    })

    shutdownManager.RegisterShutdownFunc("http_server", func(ctx context.Context) error {
        return server.Shutdown(ctx)
    })

    // Create root context
    ctx, cancel := context.WithCancel(context.Background())

    // Signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    // Start application components
    go func() {
        if err := monitor.Start(ctx, shutdownManager); err != nil {
            logger.Error("Monitor failed", zap.Error(err))
            cancel()
        }
    }()

    go func() {
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            logger.Error("HTTP server failed", zap.Error(err))
            cancel()
        }
    }()

    // Wait for shutdown signal
    sig := <-sigChan
    logger.Info("Received shutdown signal", zap.String("signal", sig.String()))

    // Cancel context to signal all components
    cancel()

    // Execute graceful shutdown
    if err := shutdownManager.Shutdown(context.Background()); err != nil {
        logger.Error("Shutdown failed", zap.Error(err))
        os.Exit(1)
    }

    // Determine exit code
    exitCode := 0
    switch sig {
    case os.Interrupt:
        exitCode = 130 // SIGINT
    case syscall.SIGTERM:
        exitCode = 143 // SIGTERM
    }

    os.Exit(exitCode)
}
```

**Handler wrapper for tracking operations:**
```go
func (m *Monitor) processAIRequest(ctx context.Context, prompt string) error {
    m.shutdownManager.TrackOperation()
    defer m.shutdownManager.CompleteOperation()

    // Check if shutdown is in progress
    select {
    case <-ctx.Done():
        return fmt.Errorf("request cancelled: shutdown in progress")
    default:
    }

    // Process request normally
    return m.handlePrompt(ctx, prompt)
}
```

### Force Shutdown on Second Signal

**Enhancement to signal handling:**
```go
// Signal handling with force shutdown
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

signalCount := 0
go func() {
    for sig := range sigChan {
        signalCount++
        if signalCount == 1 {
            logger.Info("Received shutdown signal, shutting down gracefully...",
                zap.String("signal", sig.String()),
            )
            cancel() // Trigger graceful shutdown
        } else {
            logger.Warn("Received second signal, forcing immediate shutdown",
                zap.String("signal", sig.String()),
            )
            os.Exit(1) // Force exit
        }
    }
}()
```

## Testing Strategy

### Unit Tests

**Logging (logging/logger_test.go):**
```go
func TestJSONLogging(t *testing.T) {
    // Test JSON serialization
    // Test field sanitization (API keys)
    // Test log levels
}

func TestGPUMetricsMarshaling(t *testing.T) {
    // Test GPU metrics object marshaling
}
```

**Database (db/database_test.go):**
```go
func TestDatabaseOperations(t *testing.T) {
    // Test CRUD operations
    // Test migrations
    // Test cleanup/retention
}

func TestAsyncWrites(t *testing.T) {
    // Test async write throughput
}
```

**Authentication (webui/auth_test.go):**
```go
func TestPasswordHashing(t *testing.T) {
    // Test bcrypt hashing
    // Test timing-safe comparison
}

func TestRateLimiting(t *testing.T) {
    // Test rate limiter
}

func TestSessionManagement(t *testing.T) {
    // Test session creation
    // Test session expiry
    // Test session cleanup
}
```

**Shutdown (shutdown/manager_test.go):**
```go
func TestGracefulShutdown(t *testing.T) {
    // Test in-flight operation tracking
    // Test shutdown function execution
    // Test timeout handling
}
```

### Integration Tests

**End-to-end logging pipeline:**
```go
func TestLoggingPipeline(t *testing.T) {
    // Log operation → verify JSON format → verify DB persistence
}
```

**Authentication flow:**
```go
func TestAuthenticationFlow(t *testing.T) {
    // Login → verify session → access protected route → logout
}
```

**Shutdown sequence:**
```go
func TestShutdownSequence(t *testing.T) {
    // Start app → trigger in-flight operations → signal shutdown → verify cleanup
}
```

### Performance Tests

**Logging throughput:**
```bash
go test -bench=BenchmarkLogging -benchmem
# Target: 1000 log calls in <5 seconds
```

**Database write throughput:**
```bash
go test -bench=BenchmarkDatabaseWrites -benchmem
# Target: 100 writes/second sustained
```

### Platform Tests

**Windows:**
```powershell
# Install via NSIS installer
# Verify service registration
# Test startup and shutdown
# Verify logs in %APPDATA%\CanvusLocalLLM\
```

**Linux:**
```bash
# Install via .deb package
dpkg -i canvuslocallm_*.deb
# Verify systemd unit
systemctl status canvuslocallm
# Test startup and shutdown
# Verify logs in ~/.canvuslocallm/
```

## Implementation Plan

### Phase 1: Structured Logging (Week 1)
1. Install zap dependency
2. Implement `logging/logger.go`
3. Implement `logging/gpu_metrics.go`
4. Migrate critical paths (handlers, monitor)
5. Update main.go initialization
6. Write unit tests
7. Performance benchmarks

### Phase 2: Database Persistence (Week 1-2)
1. Install SQLite dependencies
2. Design and create schema migrations
3. Implement `db/database.go`
4. Integrate with handlers (record operations)
5. Implement cleanup/retention
6. Write unit and integration tests
7. Update web UI to query database

### Phase 3: Web UI Authentication (Week 2)
1. Implement `webui/auth.go`
2. Create login page HTML
3. Implement session management
4. Implement rate limiting
5. Update main.go HTTP routes
6. Write unit tests
7. Manual security testing

### Phase 4: CI/CD Automation (Week 3)
1. Create `.github/workflows/release.yml`
2. Implement `installer/linux/build-deb.sh`
3. Implement `installer/linux/build-tarball.sh`
4. Update `installer/windows/setup.nsi`
5. Test workflow on test tag
6. Verify installers on clean VMs
7. Document release process

### Phase 5: Graceful Shutdown (Week 3-4)
1. Implement `shutdown/manager.go`
2. Update main.go signal handling
3. Add operation tracking to handlers
4. Register shutdown functions
5. Test shutdown scenarios
6. Write unit and integration tests
7. Document shutdown behavior

### Phase 6: Integration & Testing (Week 4)
1. End-to-end integration testing
2. Platform testing (Windows + Linux)
3. Performance validation
4. Security audit
5. Documentation updates
6. Release preparation

## Acceptance Criteria Summary

### Structured Logging
- [ ] All logs in JSON format with standardized fields
- [ ] GPU metrics captured and logged
- [ ] Log level configurable via environment variable
- [ ] Log rotation working (100MB, keep 5 files)
- [ ] Sensitive data never logged

### Database Persistence
- [ ] SQLite database created at correct platform location
- [ ] All AI operations persisted with full metadata
- [ ] Migrations run automatically
- [ ] Retention policy enforced (30 days)
- [ ] Web UI queries return historical data

### Web UI Authentication
- [ ] Login required for all protected routes
- [ ] Session management working (24-hour timeout)
- [ ] Rate limiting enforced (5 attempts/minute)
- [ ] Logout functionality working
- [ ] Failed attempts logged

### CI/CD Automation
- [ ] Windows installer built on tag push
- [ ] Linux packages built on tag push
- [ ] Installers include all components
- [ ] Smoke tests pass on clean VMs
- [ ] Release artifacts published with checksums

### Graceful Shutdown
- [ ] SIGINT/SIGTERM trigger graceful shutdown
- [ ] In-flight operations complete (or timeout)
- [ ] Database connections closed cleanly
- [ ] Temporary files removed
- [ ] Shutdown duration logged
- [ ] Clean restart after shutdown

## Dependencies and Prerequisites

### External Dependencies
```go
// go.mod additions
require (
    go.uber.org/zap v1.26.0
    gopkg.in/natefinch/lumberjack.v2 v2.2.1
    github.com/mattn/go-sqlite3 v1.14.18
    github.com/golang-migrate/migrate/v4 v4.17.0
    golang.org/x/crypto v0.17.0
)
```

### Prerequisites
- Phase 1 (Installation) complete
- Phase 2 (llama.cpp integration) complete
- Go 1.21+ for updated standard library features
- GitHub Actions runners with CUDA support (for CI/CD)

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Logging overhead affects performance | High | Async logging, benchmarking, profiling |
| Database corruption in production | High | WAL mode, automatic recovery, backups |
| CI/CD builds fail | Medium | Reproducible builds, pinned dependencies, extensive testing |
| Graceful shutdown timeout | Medium | Configurable timeout, force shutdown on second signal |
| Authentication bypass | High | Standard crypto libraries, security audit, rate limiting |
| Large model files exceed artifact limits | Medium | Download-on-demand strategy, external storage |

## Success Metrics

Post-deployment validation:

1. **Observability**: Time to diagnose issues reduced by 50%
2. **Reliability**: Unplanned restarts <1 per month
3. **Deployment**: Commit to release <30 minutes
4. **Security**: Zero authentication bypass incidents
5. **Performance**: <5% latency increase from overhead

## Out of Scope

Explicitly excluded from this phase:

- Prometheus/Grafana metrics
- Distributed tracing (OpenTelemetry)
- OAuth/SAML authentication
- Multi-instance clustering
- Database replication
- Configuration UI
- Telemetry/crash reporting
- Model management UI
- Real-time GPU dashboard

## References

- [zap Documentation](https://pkg.go.dev/go.uber.org/zap)
- [SQLite Best Practices](https://github.com/mattn/go-sqlite3)
- [Graceful Shutdown in Go](https://pkg.go.dev/context)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
- [NSIS Documentation](https://nsis.sourceforge.io/Docs/)
- [Debian Package Format](https://www.debian.org/doc/debian-policy/)
