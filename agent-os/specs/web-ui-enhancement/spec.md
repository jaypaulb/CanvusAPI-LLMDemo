# Web UI Enhancement Specification

## Executive Summary

This specification defines a real-time monitoring dashboard for CanvusLocalLLM that provides visibility into system status, AI processing metrics, GPU utilization, and multi-canvas monitoring. The dashboard is strictly for observability - no configuration management - aligning with the product's zero-configuration philosophy. Users install the application once, provide Canvus credentials in .env, and access a web dashboard to monitor system health and activity.

**Key Principles:**
- **Monitoring, Not Configuration** - Dashboard displays status only, no settings UI
- **Zero Additional Setup** - Web server starts automatically with backend
- **Real-time Visibility** - Live updates via WebSocket for immediate feedback
- **Simple Tech Stack** - Minimal dependencies, embedded static assets, single binary

**Target Users:**
- Enterprise IT administrators monitoring deployment health
- Team leads verifying AI integration for multiple canvases
- Consultants checking resource usage on their workstations

## Goals and Non-Goals

### Goals

1. **Real-time Processing Visibility** - Show current AI operations, queue depth, and recent activity with <2 second latency
2. **Resource Monitoring** - Display GPU utilization, memory usage, and inference performance metrics
3. **Multi-Canvas Management** - Support monitoring multiple Canvus workspaces with per-canvas health tracking
4. **Operational Troubleshooting** - Provide error tracking and system status to quickly identify issues
5. **Simple Deployment** - Embedded web server in Go binary, no separate deployment or configuration

### Non-Goals

1. **Configuration Management** - Will NOT provide UI for changing .env settings, model selection, or API keys
2. **AI Interaction** - Will NOT allow submitting prompts, testing models, or interacting with AI via UI
3. **Data Persistence** - Will NOT persist metrics beyond current session (acceptable to reset on restart)
4. **Advanced Monitoring** - Will NOT implement alerting rules, metric export, or integration with external monitoring tools
5. **Mobile Optimization** - Will NOT optimize for mobile devices (desktop/workstation use case only)

## Architecture Overview

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        User Browser                              │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │              Dashboard UI (HTML/CSS/JS)                    │  │
│  │  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐  │  │
│  │  │   Status    │  │ GPU Metrics  │  │  Canvas List    │  │  │
│  │  │   Widget    │  │   Widget     │  │     Widget      │  │  │
│  │  └─────────────┘  └──────────────┘  └─────────────────┘  │  │
│  │  ┌─────────────────────────────────────────────────────┐  │  │
│  │  │         Processing Queue & Activity Log            │  │  │
│  │  └─────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────┘  │
│                            │                                     │
│                     WebSocket / SSE                              │
└────────────────────────────┼────────────────────────────────────┘
                             │
┌────────────────────────────▼────────────────────────────────────┐
│                      Go Backend Process                          │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │                    webui/ Package (New)                    │  │
│  │  ┌────────────┐  ┌─────────────┐  ┌──────────────────┐   │  │
│  │  │  HTTP      │  │  WebSocket  │  │  Metrics         │   │  │
│  │  │  Server    │  │  Handler    │  │  Collector       │   │  │
│  │  └────────────┘  └─────────────┘  └──────────────────┘   │  │
│  │  ┌──────────────────────────────────────────────────────┐ │  │
│  │  │           Static Asset Embedder (embed.FS)           │ │  │
│  │  └──────────────────────────────────────────────────────┘ │  │
│  └───────────────────────────────────────────────────────────┘  │
│                             │                                    │
│  ┌───────────────────────────┼────────────────────────────────┐ │
│  │     metrics/ Package (New) - In-Memory Metrics Store       │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐ │ │
│  │  │   Task       │  │    GPU       │  │    Canvas       │ │ │
│  │  │   Metrics    │  │   Metrics    │  │    Status       │ │ │
│  │  └──────────────┘  └──────────────┘  └─────────────────┘ │ │
│  └──────────────────────────────────────────────────────────┘  │
│                             │                                    │
│  ┌───────────────────────────┼────────────────────────────────┐ │
│  │         Existing Application Components                    │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐ │ │
│  │  │   Monitor    │  │  Handlers    │  │   Core/Config   │ │ │
│  │  │   (Canvas)   │  │  (AI Proc)   │  │                 │ │ │
│  │  └──────────────┘  └──────────────┘  └─────────────────┘ │ │
│  │           │                 │                              │ │
│  │           └────────┬────────┘                              │ │
│  │                    │                                        │ │
│  │  ┌─────────────────▼──────────────────────────────────┐   │ │
│  │  │     llamaruntime/ (Future)  GPU/llama.cpp          │   │ │
│  │  └────────────────────────────────────────────────────┘   │ │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Atomic Design Component Hierarchy

Following atomic design principles, the web UI components are organized:

**Atoms** (Pure Functions & Primitives):
- `metrics.RecordTaskStart(type, canvasID)`
- `metrics.RecordTaskComplete(type, duration, success)`
- `metrics.GetGPUStats()` - Query GPU utilization
- `webui.FormatDuration(d time.Duration)`
- `webui.FormatBytes(bytes int64)`

**Molecules** (Simple Compositions):
- `MetricsCollector`: Aggregates task and GPU metrics
- `SessionManager`: Handles authentication sessions
- `WebSocketBroadcaster`: Manages client connections and broadcasts
- `StaticAssetHandler`: Serves embedded HTML/CSS/JS

**Organisms** (Complex Feature Units):
- `WebUIServer`: Main HTTP server with routes and middleware
- `DashboardAPI`: REST API for dashboard data
- `MetricsStore`: In-memory storage for all dashboard metrics
- `CanvasHealthMonitor`: Tracks multi-canvas connection status

**Pages** (Composition Roots):
- `main.go`: Wires WebUIServer into application lifecycle
- Dashboard HTML page: Composes all UI widgets

### Package Structure

New packages to be created:

```
go_backend/
├── webui/                         # Web UI server package (Organism)
│   ├── server.go                  # HTTP server, routes, middleware
│   ├── websocket.go               # WebSocket handler and broadcaster
│   ├── auth.go                    # Simple password authentication
│   ├── api.go                     # REST API handlers for dashboard data
│   ├── static/                    # Static assets (embedded via embed.FS)
│   │   ├── index.html             # Main dashboard page
│   │   ├── login.html             # Login page
│   │   ├── dashboard.js           # Dashboard JavaScript
│   │   ├── websocket.js           # WebSocket client logic
│   │   └── styles.css             # Dashboard styles
│   └── embed.go                   # embed.FS directives
├── metrics/                       # Metrics collection package (Organism)
│   ├── store.go                   # In-memory metrics storage
│   ├── collector.go               # Metrics collector interface
│   ├── task_metrics.go            # Task processing metrics
│   ├── gpu_metrics.go             # GPU utilization metrics (placeholder)
│   └── canvas_metrics.go          # Canvas health metrics
└── [existing packages...]
```

Integration points with existing code:
- `Monitor.handleUpdate()` in monitorcanvus.go calls `metrics.RecordTaskStart()`
- Handler functions call `metrics.RecordTaskComplete()` on success/failure
- `WebUIServer` reads from `MetricsStore` to serve dashboard data
- `main.go` initializes and starts WebUIServer alongside Monitor

## Detailed Component Design

### 1. Web UI Server (webui/server.go)

**Responsibilities:**
- Start HTTP server on configured port (default 3000)
- Serve static assets from embedded filesystem
- Handle authentication middleware
- Route API requests to appropriate handlers
- Manage WebSocket connections

**Interface:**

```go
package webui

type Server struct {
    config      *core.Config
    metricsStore *metrics.Store
    broadcaster  *WebSocketBroadcaster
    sessions     *SessionManager
}

func NewServer(cfg *core.Config, store *metrics.Store) *Server

func (s *Server) Start(ctx context.Context) error

func (s *Server) setupRoutes() *http.ServeMux
```

**Routes:**

```
GET  /                     -> Redirect to /dashboard (if authenticated) or /login
GET  /login                -> Login page (static HTML)
POST /login                -> Authenticate with WEBUI_PWD
GET  /logout               -> Invalidate session
GET  /dashboard            -> Dashboard page (requires auth)
GET  /api/status           -> System status JSON
GET  /api/canvases         -> Canvas list JSON
GET  /api/tasks            -> Recent tasks JSON
GET  /api/metrics          -> Aggregated metrics JSON
GET  /api/gpu              -> GPU stats JSON
GET  /ws                   -> WebSocket connection (requires auth)
```

**Middleware:**

```go
// AuthMiddleware checks for valid session cookie
func (s *Server) AuthMiddleware(next http.Handler) http.Handler

// LoggingMiddleware logs all HTTP requests
func (s *Server) LoggingMiddleware(next http.Handler) http.Handler
```

**Implementation Notes:**
- Use Go's net/http standard library (no external web framework)
- Embed static assets using `embed.FS` (Go 1.16+)
- Session management with secure HTTP-only cookies
- CORS not required (same-origin deployment)

### 2. Metrics Store (metrics/store.go)

**Responsibilities:**
- Store all dashboard metrics in memory
- Provide thread-safe access to metrics
- Maintain circular buffers for historical data
- Calculate aggregate statistics

**Interface:**

```go
package metrics

type Store struct {
    mu sync.RWMutex

    // System status
    startTime    time.Time
    systemStatus SystemStatus

    // Task metrics
    tasks        *TaskMetrics
    taskHistory  *CircularBuffer[TaskRecord]

    // GPU metrics
    gpuMetrics   *GPUMetrics
    gpuHistory   *CircularBuffer[GPUSnapshot]

    // Canvas metrics
    canvases     map[string]*CanvasStatus
}

type SystemStatus struct {
    Health      string    // "running", "error", "stopped"
    Version     string
    Uptime      time.Duration
    LastCheck   time.Time
}

type TaskMetrics struct {
    TotalProcessed  int64
    TotalSuccess    int64
    TotalErrors     int64

    ByType map[string]*TaskTypeMetrics
}

type TaskTypeMetrics struct {
    Count       int64
    SuccessRate float64
    AvgDuration time.Duration
}

type TaskRecord struct {
    ID          string
    Type        string
    CanvasID    string
    Status      string // "success", "error", "processing"
    StartTime   time.Time
    EndTime     time.Time
    Duration    time.Duration
    ErrorMsg    string
}

type GPUMetrics struct {
    Utilization  float64 // Percentage
    Temperature  float64 // Celsius
    MemoryTotal  int64   // Bytes
    MemoryUsed   int64   // Bytes
    MemoryFree   int64   // Bytes
}

type CanvasStatus struct {
    ID              string
    Name            string
    ServerURL       string
    Connected       bool
    LastUpdate      time.Time
    WidgetCount     int
    RequestsToday   int64
    SuccessRate     float64
    Errors          []string // Recent errors
}

func NewStore() *Store

// Task recording
func (s *Store) RecordTaskStart(taskType, canvasID string) string // Returns taskID
func (s *Store) RecordTaskComplete(taskID string, success bool, errorMsg string)
func (s *Store) GetRecentTasks(limit int) []TaskRecord
func (s *Store) GetTaskMetrics() *TaskMetrics

// GPU metrics
func (s *Store) UpdateGPUMetrics(gpu *GPUMetrics)
func (s *Store) GetGPUMetrics() *GPUMetrics
func (s *Store) GetGPUHistory() []GPUSnapshot

// Canvas status
func (s *Store) UpdateCanvasStatus(canvasID string, status *CanvasStatus)
func (s *Store) GetCanvasStatus(canvasID string) *CanvasStatus
func (s *Store) GetAllCanvases() map[string]*CanvasStatus

// System status
func (s *Store) UpdateSystemStatus(status *SystemStatus)
func (s *Store) GetSystemStatus() *SystemStatus
```

**Implementation Notes:**
- Use `sync.RWMutex` for thread-safe access
- Circular buffers limit memory usage (last 100 tasks, 1 hour of GPU data)
- All data volatile - acceptable to lose on restart
- No persistence layer in Phase 5

### 3. WebSocket Handler (webui/websocket.go)

**Responsibilities:**
- Manage WebSocket connections from browsers
- Broadcast real-time updates to all connected clients
- Handle client disconnections gracefully
- Provide reconnection support

**Interface:**

```go
package webui

type WebSocketBroadcaster struct {
    clients   map[*websocket.Conn]bool
    clientsMu sync.RWMutex
    broadcast chan Message
}

type Message struct {
    Type    string      `json:"type"` // "task_update", "gpu_update", "canvas_update"
    Payload interface{} `json:"payload"`
}

func NewWebSocketBroadcaster() *WebSocketBroadcaster

func (b *WebSocketBroadcaster) Start(ctx context.Context)

func (b *WebSocketBroadcaster) HandleConnection(w http.ResponseWriter, r *http.Request)

func (b *WebSocketBroadcaster) BroadcastMessage(msg Message)

func (b *WebSocketBroadcaster) removeClient(conn *websocket.Conn)
```

**Message Types:**

```json
// Task Update
{
  "type": "task_update",
  "payload": {
    "task_id": "task-123",
    "type": "pdf_analysis",
    "canvas_id": "canvas-456",
    "status": "completed",
    "duration": 3500,
    "timestamp": "2025-12-16T10:30:00Z"
  }
}

// GPU Update
{
  "type": "gpu_update",
  "payload": {
    "utilization": 75.5,
    "temperature": 68.0,
    "memory_used": 4294967296,
    "memory_total": 8589934592,
    "timestamp": "2025-12-16T10:30:05Z"
  }
}

// Canvas Update
{
  "type": "canvas_update",
  "payload": {
    "canvas_id": "canvas-456",
    "connected": true,
    "widget_count": 42,
    "last_update": "2025-12-16T10:30:00Z"
  }
}

// System Status
{
  "type": "system_status",
  "payload": {
    "health": "running",
    "uptime": 3600,
    "version": "v0.1.0"
  }
}
```

**Implementation Notes:**
- Use `gorilla/websocket` or `nhooyr.io/websocket` library
- Each client connection runs in its own goroutine
- Broadcast channel with select for non-blocking sends
- Ping/pong for connection health checks
- Graceful shutdown on context cancellation

### 4. Dashboard API (webui/api.go)

**Responsibilities:**
- Provide REST API endpoints for dashboard data
- Return JSON responses for initial page load
- Handle API request validation and errors

**Endpoints:**

```go
// GET /api/status - System status overview
type StatusResponse struct {
    Health      string    `json:"health"`
    Version     string    `json:"version"`
    Uptime      int64     `json:"uptime_seconds"`
    LastCheck   time.Time `json:"last_check"`
}

// GET /api/canvases - All monitored canvases
type CanvasesResponse struct {
    Canvases []CanvasInfo `json:"canvases"`
}

type CanvasInfo struct {
    ID            string    `json:"id"`
    Name          string    `json:"name"`
    ServerURL     string    `json:"server_url"`
    Connected     bool      `json:"connected"`
    LastUpdate    time.Time `json:"last_update"`
    WidgetCount   int       `json:"widget_count"`
    RequestsToday int64     `json:"requests_today"`
    SuccessRate   float64   `json:"success_rate"`
}

// GET /api/tasks?limit=50 - Recent task history
type TasksResponse struct {
    Tasks []TaskInfo `json:"tasks"`
}

type TaskInfo struct {
    ID        string    `json:"id"`
    Type      string    `json:"type"`
    CanvasID  string    `json:"canvas_id"`
    Status    string    `json:"status"`
    StartTime time.Time `json:"start_time"`
    EndTime   time.Time `json:"end_time,omitempty"`
    Duration  int64     `json:"duration_ms"`
    ErrorMsg  string    `json:"error_msg,omitempty"`
}

// GET /api/metrics - Aggregated processing metrics
type MetricsResponse struct {
    TotalProcessed int64              `json:"total_processed"`
    SuccessRate    float64            `json:"success_rate"`
    ByType         map[string]TypeMetrics `json:"by_type"`
}

type TypeMetrics struct {
    Count          int64   `json:"count"`
    SuccessRate    float64 `json:"success_rate"`
    AvgDurationMs  int64   `json:"avg_duration_ms"`
}

// GET /api/gpu - GPU metrics
type GPUResponse struct {
    Utilization    float64 `json:"utilization_percent"`
    Temperature    float64 `json:"temperature_celsius"`
    MemoryUsed     int64   `json:"memory_used_bytes"`
    MemoryTotal    int64   `json:"memory_total_bytes"`
    MemoryFree     int64   `json:"memory_free_bytes"`
    Available      bool    `json:"available"`
}
```

**Implementation Notes:**
- Standard JSON encoding with proper error handling
- Return 500 on internal errors with generic message
- Return 401 on authentication failures
- Use HTTP status codes appropriately

### 5. Session Manager (webui/auth.go)

**Responsibilities:**
- Authenticate users with WEBUI_PWD
- Generate and validate session tokens
- Manage session expiry (24 hours)
- Provide logout functionality

**Interface:**

```go
package webui

type SessionManager struct {
    password string
    sessions map[string]*Session
    mu       sync.RWMutex
}

type Session struct {
    Token     string
    CreatedAt time.Time
    ExpiresAt time.Time
}

func NewSessionManager(password string) *SessionManager

func (sm *SessionManager) Authenticate(password string) (string, error)

func (sm *SessionManager) ValidateSession(token string) bool

func (sm *SessionManager) InvalidateSession(token string)

func (sm *SessionManager) CleanupExpiredSessions()
```

**Implementation Notes:**
- Use `crypto/rand` for secure token generation
- Store session token in HTTP-only cookie
- Periodic cleanup goroutine for expired sessions (every 1 hour)
- No user database - single password from config

### 6. GPU Metrics Collector (metrics/gpu_metrics.go)

**Responsibilities:**
- Collect GPU utilization, temperature, and memory stats
- Interface with NVIDIA Management Library (NVML) or fallback to mock data
- Update metrics store periodically

**Interface:**

```go
package metrics

type GPUCollector struct {
    store     *Store
    available bool
    nvml      NVMLInterface // Interface for testing
}

type NVMLInterface interface {
    GetUtilization() (float64, error)
    GetTemperature() (float64, error)
    GetMemoryInfo() (total, used, free int64, err error)
}

func NewGPUCollector(store *Store) *GPUCollector

func (gc *GPUCollector) Start(ctx context.Context)

func (gc *GPUCollector) Collect() (*GPUMetrics, error)
```

**Implementation Notes:**
- Phase 5 may use mock data if NVML bindings not ready
- Use `github.com/NVIDIA/go-nvml` or similar library
- Fallback gracefully if GPU not available (show N/A in UI)
- Collection interval: 5 seconds
- Store last 720 samples (1 hour at 5-second intervals)

## Frontend Design

### Technology Stack

**Core Technologies:**
- **HTML5** - Semantic markup
- **CSS3** - Styling with CSS Grid and Flexbox
- **Vanilla JavaScript (ES6+)** - No framework for simplicity

**Optional Libraries (via CDN):**
- **Alpine.js** (16KB) - Lightweight reactivity for dynamic UI updates
- **Chart.js** (via CDN) - GPU utilization graph
- **DayJS** (via CDN) - Date/time formatting

**Design System:**
- **Colors**: Dark theme for reduced eye strain (enterprise workstation use)
- **Typography**: System fonts (San Francisco, Segoe UI, Roboto)
- **Layout**: CSS Grid for dashboard widget layout
- **Responsive**: Desktop-first (1920x1080, 2560x1440)

### Dashboard Layout

```
┌─────────────────────────────────────────────────────────────────┐
│  CanvusLocalLLM Dashboard           [Connected ●]    [Logout]   │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────┐  │
│  │  System Status   │  │   GPU Metrics    │  │   Canvas     │  │
│  │  ────────────    │  │   ──────────     │  │   Status     │  │
│  │  ✓ Running       │  │   Util: 75.5%    │  │              │  │
│  │  Uptime: 2h 34m  │  │   Temp: 68°C     │  │  Canvas A    │  │
│  │  Version: v0.1   │  │   VRAM: 4/8 GB   │  │  ✓ Connected │  │
│  │                  │  │                  │  │              │  │
│  │                  │  │  [GPU Graph]     │  │  Canvas B    │  │
│  │                  │  │                  │  │  ⚠ Errors    │  │
│  └──────────────────┘  └──────────────────┘  └──────────────┘  │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Processing Metrics                                         │ │
│  │  ─────────────────                                          │ │
│  │  Total: 1,234  |  Success: 98.5%  |  Errors: 19  |  Avg: 3s│ │
│  │                                                              │ │
│  │  Note Processing: 890 (99% success, 1.2s avg)               │ │
│  │  PDF Analysis: 200 (95% success, 8.5s avg)                  │ │
│  │  Image Generation: 100 (98% success, 12s avg)               │ │
│  │  Canvas Analysis: 44 (100% success, 5s avg)                 │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Active Processing Queue (3)              Filter: [All ▾]   │ │
│  │  ───────────────────────────────────────────────────────   │ │
│  │  • PDF Analysis - Canvas A - Started 00:12 ago              │ │
│  │  • Note Processing - Canvas B - Started 00:03 ago           │ │
│  │  • Image Generation - Canvas A - Started 00:45 ago          │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Recent Activity (Last 50)                Filter: [All ▾]   │ │
│  │  ─────────────────────────                                  │ │
│  │  10:34:22 ✓ Note Processing - Canvas A - 1.2s              │ │
│  │  10:34:15 ✓ PDF Analysis - Canvas B - 8.5s                 │ │
│  │  10:33:58 ✗ Image Generation - Canvas A - Error: timeout   │ │
│  │  10:33:45 ✓ Canvas Analysis - Canvas B - 5.1s              │ │
│  │  10:33:12 ✓ Note Processing - Canvas A - 0.9s              │ │
│  │  [Auto-scroll: On ▾]                              [Load more]│
│  └────────────────────────────────────────────────────────────┘ │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Key UI Components

**1. System Status Widget**
- Health indicator (colored badge: green/yellow/red)
- Uptime display (formatted as "Xh Ym")
- Version number
- Last health check timestamp

**2. GPU Metrics Widget**
- Current utilization (percentage with progress bar)
- Temperature (Celsius, color-coded: green <70, yellow 70-80, red >80)
- VRAM usage (used/total with progress bar)
- Historical utilization graph (last 1 hour, Chart.js line chart)
- "N/A" placeholders if GPU unavailable

**3. Canvas Status Widget**
- List of all monitored canvases
- Per-canvas indicators:
  - Connection status icon (✓ green, ⚠ yellow, ✗ red)
  - Canvas name and ID
  - Last update timestamp (relative, e.g., "2m ago")
  - Widget count
- Click to filter dashboard by canvas

**4. Processing Metrics Panel**
- Aggregate statistics (total, success rate, error count, avg duration)
- Breakdown by task type with metrics
- Color-coded success rates (green >95%, yellow 90-95%, red <90%)

**5. Active Queue Panel**
- List of currently processing tasks
- For each task: type, canvas, elapsed time
- Empty state message when queue is empty
- Auto-updates via WebSocket

**6. Recent Activity Log**
- Scrollable list of recent tasks (last 50)
- Timestamp, status icon, task type, canvas, duration/error
- Filter dropdown (All, Success, Error, By Type)
- Auto-scroll toggle (on by default)
- Load more button for pagination

**7. Connection Status Indicator**
- Shows WebSocket connection status
- "Connected" (green), "Connecting" (yellow), "Disconnected" (red)
- Auto-reconnect on disconnect

### JavaScript Architecture

**Module Structure:**

```javascript
// dashboard.js - Main application logic

class DashboardApp {
    constructor() {
        this.ws = null;
        this.isConnected = false;
        this.autoScroll = true;
        this.currentFilter = 'all';
    }

    async init() {
        await this.loadInitialData();
        this.connectWebSocket();
        this.setupEventListeners();
    }

    async loadInitialData() {
        // Fetch initial data from REST API
        const [status, canvases, tasks, metrics, gpu] = await Promise.all([
            fetch('/api/status').then(r => r.json()),
            fetch('/api/canvases').then(r => r.json()),
            fetch('/api/tasks?limit=50').then(r => r.json()),
            fetch('/api/metrics').then(r => r.json()),
            fetch('/api/gpu').then(r => r.json())
        ]);

        this.renderStatus(status);
        this.renderCanvases(canvases);
        this.renderTasks(tasks);
        this.renderMetrics(metrics);
        this.renderGPU(gpu);
    }

    connectWebSocket() {
        this.ws = new WebSocket(`ws://${window.location.host}/ws`);

        this.ws.onopen = () => {
            this.isConnected = true;
            this.updateConnectionStatus('connected');
        };

        this.ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.handleWebSocketMessage(message);
        };

        this.ws.onclose = () => {
            this.isConnected = false;
            this.updateConnectionStatus('disconnected');
            // Attempt reconnection after 5 seconds
            setTimeout(() => this.connectWebSocket(), 5000);
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    handleWebSocketMessage(message) {
        switch (message.type) {
            case 'task_update':
                this.handleTaskUpdate(message.payload);
                break;
            case 'gpu_update':
                this.handleGPUUpdate(message.payload);
                break;
            case 'canvas_update':
                this.handleCanvasUpdate(message.payload);
                break;
            case 'system_status':
                this.handleSystemStatus(message.payload);
                break;
        }
    }

    handleTaskUpdate(task) {
        // Update active queue if task is processing
        // Add to recent activity if task completed
        // Update metrics
    }

    renderStatus(status) {
        // Render system status widget
    }

    renderGPU(gpu) {
        // Render GPU metrics widget
        // Initialize Chart.js graph
    }

    // ... more rendering methods
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', () => {
    const app = new DashboardApp();
    app.init();
});
```

**WebSocket Client (websocket.js):**

```javascript
class WebSocketClient {
    constructor(url, onMessage, onConnect, onDisconnect) {
        this.url = url;
        this.onMessage = onMessage;
        this.onConnect = onConnect;
        this.onDisconnect = onDisconnect;
        this.ws = null;
        this.reconnectInterval = 5000;
        this.shouldReconnect = true;
    }

    connect() {
        this.ws = new WebSocket(this.url);

        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.onConnect?.();
        };

        this.ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.onMessage?.(message);
        };

        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            this.onDisconnect?.();
            if (this.shouldReconnect) {
                setTimeout(() => this.connect(), this.reconnectInterval);
            }
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };
    }

    disconnect() {
        this.shouldReconnect = false;
        this.ws?.close();
    }

    send(message) {
        if (this.ws?.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        }
    }
}
```

## Integration with Existing Codebase

### Monitor Integration (monitorcanvus.go)

**Current Implementation:**
- `Monitor.handleUpdate()` processes widget updates and calls handlers
- Handler functions (in handlers.go) perform AI processing

**Required Changes:**

```go
// In monitorcanvus.go

func (m *Monitor) handleUpdate(ctx context.Context, update WidgetUpdate) {
    // Existing logic...

    // NEW: Record task start before processing
    taskID := m.metricsStore.RecordTaskStart(determineTaskType(update), m.client.CanvasID)

    // Existing handler call
    err := processUpdate(ctx, update)

    // NEW: Record task completion
    success := err == nil
    errMsg := ""
    if err != nil {
        errMsg = err.Error()
    }
    m.metricsStore.RecordTaskComplete(taskID, success, errMsg)

    // NEW: Broadcast to WebSocket clients
    m.broadcaster.BroadcastMessage(webui.Message{
        Type: "task_update",
        Payload: map[string]interface{}{
            "task_id": taskID,
            "status": getStatus(success),
            "duration": time.Since(startTime).Milliseconds(),
        },
    })
}
```

**Task Type Determination:**

```go
func determineTaskType(update WidgetUpdate) string {
    // Check widget content for {{ }} - "note_processing"
    // Check for PDF attachment - "pdf_analysis"
    // Check for image generation trigger - "image_generation"
    // Check for canvas analysis trigger - "canvas_analysis"
    return taskType
}
```

### Main.go Integration

**Required Changes:**

```go
// In main.go

func main() {
    // Existing initialization...
    config, err := core.LoadConfig()
    // ...

    // NEW: Initialize metrics store
    metricsStore := metrics.NewStore()

    // NEW: Initialize GPU collector
    gpuCollector := metrics.NewGPUCollector(metricsStore)
    go gpuCollector.Start(ctx)

    // NEW: Initialize web UI server
    webUIServer := webui.NewServer(config, metricsStore)
    go func() {
        if err := webUIServer.Start(ctx); err != nil {
            log.Printf("Web UI server error: %v", err)
        }
    }()

    // Existing monitor initialization
    monitor := NewMonitor(client, config, metricsStore) // Pass metricsStore
    go monitor.Start(ctx)

    // Existing signal handling...
}
```

### Handler Integration (handlers.go)

**Required Changes:**

All handler functions should report metrics:

```go
// Example: PDF analysis handler

func handlePDFAnalysis(ctx context.Context, pdfPath string, canvasID string, metricsStore *metrics.Store) error {
    taskID := metricsStore.RecordTaskStart("pdf_analysis", canvasID)
    defer func() {
        success := err == nil
        errMsg := ""
        if err != nil {
            errMsg = err.Error()
        }
        metricsStore.RecordTaskComplete(taskID, success, errMsg)
    }()

    // Existing PDF analysis logic...
    return nil
}
```

**Pattern for All Handlers:**
1. Record task start at function entry
2. Defer task completion recording
3. Capture success/failure and error messages
4. Broadcast updates via WebSocket (if available)

## Multi-Canvas Architecture

### Canvas Manager

**New Component: canvusapi/manager.go**

```go
package canvusapi

type CanvasManager struct {
    clients      map[string]*Client // canvasID -> Client
    clientsMu    sync.RWMutex
    metricsStore *metrics.Store
}

func NewCanvasManager(metricsStore *metrics.Store) *CanvasManager

func (cm *CanvasManager) AddCanvas(canvasID, serverURL, apiKey string, allowSelfSigned bool) error

func (cm *CanvasManager) RemoveCanvas(canvasID string)

func (cm *CanvasManager) GetClient(canvasID string) (*Client, error)

func (cm *CanvasManager) GetAllCanvases() []*CanvasStatus

func (cm *CanvasManager) MonitorHealth(ctx context.Context)
```

**Implementation Notes:**
- Phase 5.3 (Multi-Canvas Management) introduces this component
- Each canvas gets its own Client instance
- Health monitoring checks connectivity for each canvas every 30 seconds
- Metrics store tracks per-canvas statistics
- Configuration likely from .env with comma-separated canvas IDs or config file

**Configuration Approach:**

Option 1: Environment variables (for Phase 5)
```
CANVAS_IDS=canvas1,canvas2,canvas3
CANVAS_1_NAME=Project Alpha
CANVAS_1_ID=abc123
CANVAS_2_NAME=Project Beta
CANVAS_2_ID=def456
```

Option 2: JSON config file (future enhancement)
```json
{
  "canvases": [
    {"id": "abc123", "name": "Project Alpha", "server": "..."},
    {"id": "def456", "name": "Project Beta", "server": "..."}
  ]
}
```

**UI Updates:**
- Canvas Status Widget shows all canvases
- Filter dropdown allows viewing metrics per canvas
- Recent Activity Log can filter by canvas
- Metrics aggregated across all canvases by default

## Testing Strategy

### Backend Testing

**Unit Tests:**

```go
// metrics/store_test.go
func TestStore_RecordTaskStart(t *testing.T)
func TestStore_RecordTaskComplete(t *testing.T)
func TestStore_GetTaskMetrics(t *testing.T)
func TestStore_UpdateGPUMetrics(t *testing.T)
func TestStore_GetRecentTasks(t *testing.T)

// webui/auth_test.go
func TestSessionManager_Authenticate(t *testing.T)
func TestSessionManager_ValidateSession(t *testing.T)
func TestSessionManager_ExpireOldSessions(t *testing.T)

// webui/websocket_test.go
func TestWebSocketBroadcaster_AddClient(t *testing.T)
func TestWebSocketBroadcaster_BroadcastMessage(t *testing.T)
func TestWebSocketBroadcaster_RemoveClient(t *testing.T)
```

**Integration Tests:**

```go
// webui/server_test.go
func TestServer_StartAndStop(t *testing.T)
func TestServer_AuthenticationFlow(t *testing.T)
func TestServer_APIEndpoints(t *testing.T)
func TestServer_WebSocketConnection(t *testing.T)
func TestServer_StaticAssetServing(t *testing.T)

// End-to-end test
func TestDashboard_FullFlow(t *testing.T) {
    // 1. Start server
    // 2. Login via POST /login
    // 3. Fetch dashboard data via API
    // 4. Connect WebSocket
    // 5. Simulate task updates
    // 6. Verify WebSocket broadcasts
    // 7. Logout
}
```

**Testing Approach:**
- Use `httptest` package for HTTP server tests
- Mock MetricsStore for API handler tests
- Use gorilla/websocket test helpers for WebSocket tests
- Test authentication with valid/invalid passwords
- Test session expiry logic
- Test concurrent WebSocket clients

### Frontend Testing

**Manual Testing Checklist:**

- [ ] Dashboard loads successfully on localhost:3000
- [ ] Login page displays and accepts correct password
- [ ] Login rejects incorrect password
- [ ] All widgets render with initial data
- [ ] GPU metrics update every 5 seconds
- [ ] Recent activity log updates in real-time
- [ ] Active queue updates when tasks start/complete
- [ ] WebSocket reconnects automatically on disconnect
- [ ] Filter dropdowns work correctly
- [ ] Canvas list displays all canvases
- [ ] Filtering by canvas updates all metrics
- [ ] Logout clears session
- [ ] Session persists across page reloads
- [ ] Session expires after 24 hours
- [ ] Dashboard works in Chrome, Firefox, Edge
- [ ] Dashboard works on 1920x1080 and 2560x1440 screens

**Browser Testing:**
- Chrome (latest)
- Firefox (latest)
- Edge (latest)
- Safari (latest) on macOS

**Performance Testing:**
- Page load time <2 seconds
- WebSocket message latency <500ms
- Memory usage <50MB after 1 hour
- No memory leaks over 24 hours
- Dashboard remains responsive with 100+ tasks in log

### Load Testing

**Scenarios:**

1. **High Task Volume** - Simulate 100 concurrent AI operations
   - Verify metrics update correctly
   - Verify WebSocket broadcasts don't lag
   - Check memory usage remains stable

2. **Multiple Clients** - Open dashboard in 10 browser tabs
   - Verify all clients receive updates
   - Check server memory usage
   - Verify no connection drops

3. **Long Running Session** - Leave dashboard open for 24 hours
   - Check for memory leaks
   - Verify auto-reconnect works
   - Check session expiry handling

## Deployment and Operations

### Deployment Steps

**Phase 5 Deployment (with existing application):**

1. Build application with embedded web UI:
   ```bash
   go build -o CanvusLocalLLM.exe .
   ```

2. Ensure .env has WEBUI_PWD configured:
   ```
   WEBUI_PWD=your_secure_password
   PORT=3000
   ```

3. Start application:
   ```bash
   ./CanvusLocalLLM.exe
   ```

4. Access dashboard:
   - Open browser to http://localhost:3000
   - Login with WEBUI_PWD

**No additional deployment steps required** - web server is part of the main application.

### Configuration

**Environment Variables:**

```bash
# Existing variables (from CLAUDE.md)
CANVUS_SERVER=https://canvus.example.com
CANVAS_ID=abc123
CANVUS_API_KEY=your_api_key
# ... other existing vars

# NEW: Web UI variables
WEBUI_PWD=your_secure_password   # Required for dashboard login
PORT=3000                         # Optional, default 3000

# Future: Multi-canvas configuration
CANVAS_IDS=canvas1,canvas2,canvas3
```

**No configuration UI** - all settings remain in .env file.

### Monitoring and Troubleshooting

**Log Messages:**

```
[INFO] Starting web UI server on port 3000
[INFO] WebSocket client connected (total: 3)
[INFO] WebSocket client disconnected (total: 2)
[WARN] Failed to collect GPU metrics: nvml unavailable
[ERROR] API request failed: /api/status - internal error
```

**Common Issues:**

| Issue | Symptom | Resolution |
|-------|---------|------------|
| Login fails | "Invalid password" | Check WEBUI_PWD in .env |
| Dashboard shows N/A | GPU metrics unavailable | Normal if NVML not initialized yet |
| WebSocket disconnects | Red "Disconnected" indicator | Check network, should auto-reconnect |
| Port already in use | Server won't start | Change PORT in .env |
| Session expired | Redirected to login | Normal after 24 hours, login again |

**Health Checks:**

```bash
# Check if web UI server is running
curl http://localhost:3000/api/status

# Expected response:
{
  "health": "running",
  "version": "v0.1.0",
  "uptime_seconds": 3600,
  "last_check": "2025-12-16T10:30:00Z"
}
```

### Security Considerations

**Authentication:**
- Password required for all dashboard access
- Session cookie HTTP-only (prevents XSS theft)
- Session expires after 24 hours
- No persistent "remember me" option

**Network Security:**
- Dashboard binds to localhost:3000 by default
- HTTPS recommended but not required (local deployment)
- No CORS headers (same-origin only)
- No API keys exposed in frontend

**Data Privacy:**
- No metrics sent to external services
- All data in memory (volatile)
- No persistent logs of AI operations
- Prompts not stored (only task type and duration)

**Hardening Recommendations:**
- Use strong WEBUI_PWD (12+ characters)
- If exposing outside localhost, use reverse proxy with HTTPS
- Consider IP whitelist if deployed on network

## Implementation Plan

### Task Breakdown

**Phase 5.1: Real-time Processing Dashboard (Large - 10-15 days)**

1. **Create metrics package** (3 days)
   - Implement Store with thread-safe operations
   - Create TaskMetrics, GPUMetrics, CanvasStatus types
   - Implement circular buffers for historical data
   - Write unit tests

2. **Create webui package** (5 days)
   - Implement HTTP server with routes
   - Create authentication middleware and SessionManager
   - Implement REST API endpoints
   - Embed static assets (HTML/CSS/JS)
   - Write integration tests

3. **Implement WebSocket handler** (3 days)
   - Create WebSocketBroadcaster
   - Implement message types and broadcasting
   - Add reconnection logic
   - Test with multiple clients

4. **Build frontend dashboard** (4 days)
   - Create HTML layout with widgets
   - Implement JavaScript dashboard logic
   - Add WebSocket client
   - Style with CSS (dark theme)
   - Test in multiple browsers

**Phase 5.2: Status and Metrics Display (Medium - 5-7 days)**

1. **Implement GPU metrics collector** (3 days)
   - Create GPUCollector with NVML interface
   - Implement fallback for unavailable GPU
   - Add periodic collection goroutine
   - Test with real GPU and mock

2. **Integrate metrics with handlers** (2 days)
   - Add RecordTaskStart/Complete to all handlers
   - Update Monitor.handleUpdate() to record metrics
   - Test metrics accuracy

3. **Add GPU visualization** (2 days)
   - Integrate Chart.js for historical graph
   - Display real-time GPU stats
   - Handle unavailable GPU gracefully

**Phase 5.3: Multi-Canvas Management (Large - 8-10 days)**

1. **Create CanvasManager** (3 days)
   - Implement multi-canvas client management
   - Add health monitoring for each canvas
   - Store per-canvas metrics
   - Write tests

2. **Update configuration** (2 days)
   - Add multi-canvas config parsing
   - Update main.go to initialize multiple canvases
   - Test with 3+ canvases

3. **Update frontend for multi-canvas** (3 days)
   - Add canvas list widget
   - Implement canvas filtering
   - Update all widgets to support per-canvas view
   - Test with multiple canvases

### Development Workflow

**Incremental Development:**

1. **Week 1-2: Core Infrastructure**
   - Create metrics and webui packages
   - Basic HTTP server with static assets
   - REST API endpoints
   - Authentication

2. **Week 2-3: Real-time Updates**
   - WebSocket implementation
   - Frontend dashboard with live updates
   - Integration with existing Monitor

3. **Week 3-4: GPU Monitoring**
   - GPU metrics collector
   - Frontend visualization
   - Testing with real GPU

4. **Week 4-5: Multi-Canvas Support**
   - CanvasManager implementation
   - Multi-canvas configuration
   - Frontend updates

**Testing Throughout:**
- Unit tests for each component
- Integration tests for API and WebSocket
- Manual testing on real deployment
- Performance testing with load scenarios

### Dependencies

**External Dependencies:**
- `gorilla/websocket` or `nhooyr.io/websocket` - WebSocket support
- `github.com/NVIDIA/go-nvml` - GPU metrics (or mock for Phase 5)

**Internal Dependencies:**
- Requires existing Monitor, handlers, Config
- Builds on atomic design patterns from CLAUDE.md
- Uses existing logging and error handling patterns

**Dependency Ordering:**
- Phase 5.1 can start immediately (no blockers)
- Phase 5.2 depends on 5.1 (metrics store)
- Phase 5.3 depends on 5.1 and 5.2 (full dashboard operational)

## Acceptance Criteria

### Phase 5.1: Real-time Processing Dashboard

- [ ] Web server starts automatically with backend on port 3000
- [ ] Dashboard loads at http://localhost:3000 after login
- [ ] Login page accepts WEBUI_PWD and creates session
- [ ] System status widget displays health, uptime, version
- [ ] Processing metrics show total requests, success rate, by-type breakdown
- [ ] Recent activity log displays last 50 tasks with real-time updates
- [ ] Active queue displays currently processing tasks
- [ ] WebSocket connection status indicator works
- [ ] Auto-reconnect works after disconnect
- [ ] Logout functionality clears session
- [ ] Dashboard works in Chrome, Firefox, Edge
- [ ] Page load time <2 seconds
- [ ] No errors in browser console

### Phase 5.2: Status and Metrics Display

- [ ] GPU utilization displays percentage in real-time (or N/A if unavailable)
- [ ] GPU temperature displays in Celsius (or N/A)
- [ ] VRAM usage displays used/total with progress bar
- [ ] GPU metrics update every 5 seconds
- [ ] Historical GPU graph displays last 1 hour
- [ ] Inference statistics display avg times per task type
- [ ] System memory and CPU usage display
- [ ] Downloads directory space displays
- [ ] Graceful fallback when GPU metrics unavailable
- [ ] No measurable impact on AI inference performance

### Phase 5.3: Multi-Canvas Management

- [ ] Dashboard displays all monitored canvases
- [ ] Each canvas shows connection status, widget count, last update
- [ ] Can filter dashboard metrics by specific canvas
- [ ] Canvas health monitoring tracks errors per canvas
- [ ] Per-canvas success rates display correctly
- [ ] Configuration supports multiple canvas IDs
- [ ] CanvasManager health monitoring runs periodically
- [ ] Dashboard handles 5+ canvases without performance issues
- [ ] All WebSocket updates include canvas ID for filtering

### Cross-Phase Acceptance Criteria

- [ ] Dashboard runs for 24 hours without memory leaks
- [ ] Multiple browser clients (10+) connect without issues
- [ ] High task volume (100+ concurrent) handled correctly
- [ ] WebSocket message latency <500ms
- [ ] Memory overhead <50MB
- [ ] No crashes or errors in 24-hour soak test
- [ ] Documentation updated (README, CLAUDE.md)
- [ ] All tests passing (unit, integration, manual)

## Future Enhancements (Post-Phase 5)

### Potential Phase 6 Features

**Metric Persistence:**
- SQLite database for historical metrics
- Retain data across restarts
- Query historical trends (daily, weekly, monthly)

**Advanced Visualizations:**
- Heatmap of processing activity by hour
- Success rate trends over time
- GPU utilization correlation with task volume

**Alerting:**
- Email notifications for critical errors
- Threshold-based alerts (GPU temp >85C, error rate >5%)
- Webhook integrations (Slack, Discord)

**Export and Reporting:**
- CSV export of processing history
- PDF reports of system metrics
- API endpoint for external monitoring tools (Prometheus, Grafana)

**Enhanced GPU Metrics:**
- GPU power draw monitoring
- GPU clock speeds
- Multiple GPU support
- GPU memory per-process breakdown

**Canvas Visualizations:**
- Preview of canvas widgets in dashboard
- Thumbnail images of recent AI-generated content
- Interactive canvas map showing processing activity

### Explicitly Out of Scope

**Will NOT Be Added:**
- Configuration UI for .env settings
- Model selection/switching interface
- Prompt testing/playground
- Canvas editing capabilities
- User management/RBAC
- Cloud integrations
- Mobile app

These features contradict the "zero-configuration" philosophy or expand beyond monitoring scope.

## Appendix

### API Endpoint Reference

Full REST API documentation:

```
Authentication Required: All endpoints except /login require valid session

GET /
  → Redirects to /dashboard or /login

GET /login
  → Returns: login.html

POST /login
  Body: {"password": "string"}
  → Returns: {"success": true, "token": "string"}
  Sets cookie: session_token

GET /logout
  → Invalidates session, redirects to /login

GET /dashboard
  → Returns: index.html (requires auth)

GET /api/status
  → Returns: SystemStatus JSON

GET /api/canvases
  → Returns: CanvasesResponse JSON

GET /api/tasks?limit=50
  Query params: limit (default 50)
  → Returns: TasksResponse JSON

GET /api/metrics
  → Returns: MetricsResponse JSON

GET /api/gpu
  → Returns: GPUResponse JSON

GET /ws
  → Upgrades to WebSocket (requires auth)
```

### WebSocket Message Reference

Full WebSocket message format:

```json
// Client → Server (none currently, future expansion)

// Server → Client

{
  "type": "task_update",
  "payload": {
    "task_id": "string",
    "type": "note_processing" | "pdf_analysis" | "image_generation" | "canvas_analysis",
    "canvas_id": "string",
    "status": "processing" | "completed" | "error",
    "start_time": "ISO 8601",
    "end_time": "ISO 8601" | null,
    "duration": number (milliseconds),
    "error_msg": "string" | null
  }
}

{
  "type": "gpu_update",
  "payload": {
    "utilization": number (percent),
    "temperature": number (celsius),
    "memory_used": number (bytes),
    "memory_total": number (bytes),
    "memory_free": number (bytes),
    "timestamp": "ISO 8601"
  }
}

{
  "type": "canvas_update",
  "payload": {
    "canvas_id": "string",
    "name": "string",
    "connected": boolean,
    "widget_count": number,
    "last_update": "ISO 8601",
    "requests_today": number,
    "success_rate": number (percent)
  }
}

{
  "type": "system_status",
  "payload": {
    "health": "running" | "error" | "stopped",
    "uptime": number (seconds),
    "version": "string",
    "last_check": "ISO 8601"
  }
}
```

### Code Organization Summary

```
go_backend/
├── main.go                        # Wire WebUIServer, start goroutines
├── monitorcanvus.go              # Add metrics recording to handleUpdate
├── handlers.go                   # Add metrics recording to all handlers
├── metrics/                       # NEW: Metrics package
│   ├── store.go                   # In-memory metrics storage
│   ├── collector.go               # Metrics collector interface
│   ├── task_metrics.go            # Task tracking
│   ├── gpu_metrics.go             # GPU stats (NVML or mock)
│   ├── canvas_metrics.go          # Canvas health
│   └── circular_buffer.go         # Helper for historical data
├── webui/                         # NEW: Web UI package
│   ├── server.go                  # HTTP server and routes
│   ├── api.go                     # REST API handlers
│   ├── websocket.go               # WebSocket handler
│   ├── auth.go                    # Session management
│   ├── embed.go                   # Static asset embedding
│   └── static/
│       ├── index.html             # Dashboard page
│       ├── login.html             # Login page
│       ├── dashboard.js           # Dashboard logic
│       ├── websocket.js           # WebSocket client
│       └── styles.css             # Dashboard styles
├── canvusapi/
│   ├── canvusapi.go              # Existing API client
│   └── manager.go                 # NEW: Multi-canvas manager (Phase 5.3)
└── [existing packages...]
```

### Glossary

**Terms:**

- **Canvas** - A Canvus workspace being monitored for AI processing
- **Task** - A single AI operation (note processing, PDF analysis, etc.)
- **Task Type** - Category of AI operation (note_processing, pdf_analysis, image_generation, canvas_analysis)
- **Metrics Store** - In-memory storage for all dashboard metrics
- **WebSocket Broadcaster** - Component managing real-time updates to browser clients
- **Session** - Authenticated user session lasting 24 hours
- **Widget** - UI component in dashboard (status widget, GPU widget, etc.)
- **Circular Buffer** - Fixed-size FIFO buffer for historical data
- **NVML** - NVIDIA Management Library for GPU metrics

**Abbreviations:**

- **GPU** - Graphics Processing Unit
- **VRAM** - Video RAM (GPU memory)
- **SSE** - Server-Sent Events (alternative to WebSocket)
- **REST** - Representational State Transfer (API style)
- **JSON** - JavaScript Object Notation
- **CDN** - Content Delivery Network
- **CRUD** - Create, Read, Update, Delete

---

**Document Version:** 1.0
**Last Updated:** 2025-12-16
**Author:** Claude (Sonnet 4.5)
**Status:** Ready for Implementation
