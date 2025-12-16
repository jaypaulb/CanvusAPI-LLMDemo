# Web UI Enhancement - Requirements

## Overview

Create a real-time monitoring dashboard for CanvusLocalLLM that provides visibility into system status, processing metrics, and multi-canvas monitoring without exposing configuration options. The UI must align with the product's "zero-configuration" philosophy - it's for monitoring and observability, not for configuration.

## Product Context

**What CanvusLocalLLM Is:**
- Zero-configuration local AI integration for Canvus workspaces
- Embeds Bunny v1.1 (text + vision) and stable-diffusion.cpp (image generation)
- Runs entirely on local NVIDIA RTX GPUs with CUDA acceleration
- No cloud dependencies, complete data privacy

**Current State:**
- Console logging only (app.log + terminal output)
- No web interface
- Single canvas monitoring per instance
- Go backend with real-time canvas monitoring via streaming API

## User Personas and Use Cases

### Enterprise IT Administrator
**Needs:**
- Verify the service is running and healthy
- Monitor GPU utilization and memory to ensure efficient resource usage
- View processing metrics to justify hardware investment
- Quick troubleshooting when users report issues

**Use Cases:**
- Check dashboard weekly to ensure system is operating normally
- Investigate when AI responses are slow or failing
- Monitor resource usage to plan for scaling

### Product Manager / Team Lead
**Needs:**
- See which canvases are being monitored
- Understand AI processing activity across team workspaces
- Verify the system is working without technical details

**Use Cases:**
- Quick status check when opening laptop in the morning
- Verify AI integration is working for team members
- Review recent activity to understand usage patterns

### Independent Consultant
**Needs:**
- Confirm system is processing client work correctly
- Monitor resource usage on their workstation
- Quick status dashboard - don't want to dig through logs

**Use Cases:**
- Glance at dashboard when switching between client projects
- Verify AI responses are being generated
- Check that no errors are occurring

## Functional Requirements

### FR1: Real-time Processing Dashboard

**FR1.1: System Status Overview**
- Display system health status (Running, Error, Stopped)
- Show uptime since service start
- Display Go backend version/build info
- Show last successful health check timestamp
- Visual health indicator (green/yellow/red status badge)

**FR1.2: Canvas Monitoring Status**
- List all actively monitored canvases (Phase 5.3 - multi-canvas)
- For each canvas show:
  - Canvas name and ID
  - Connection status (Connected, Disconnected, Error)
  - Last update timestamp
  - Number of widgets being tracked
  - Recent activity indicator (active/idle)
- Ability to see which canvas is "primary" (if applicable)

**FR1.3: Processing Queue and Activity**
- Show current processing queue depth (pending AI requests)
- Display active processing tasks with:
  - Task type (Note Processing, PDF Analysis, Image Generation, Canvas Analysis)
  - Canvas source
  - Start timestamp
  - Estimated progress (if available)
- Recent completed tasks (last 20-50) with:
  - Task type
  - Completion status (Success, Error)
  - Duration
  - Timestamp
- Real-time updates as tasks are processed

**FR1.4: Success/Failure Metrics**
- Total requests processed (since startup)
- Success rate percentage
- Error rate percentage
- Breakdown by task type:
  - Note processing count and success rate
  - PDF analysis count and success rate
  - Image generation count and success rate
  - Canvas analysis count and success rate
- Error count by type (last 24 hours)
- Average processing time by task type

**FR1.5: Recent AI Operations Log**
- Real-time scrolling log of recent AI operations
- For each operation show:
  - Timestamp
  - Operation type
  - Source canvas
  - Status (Success, Error, Processing)
  - Duration (if complete)
  - Brief message/prompt excerpt (first 100 chars)
- Filter by operation type
- Filter by status
- Filter by canvas (in multi-canvas mode)
- Limit to last 100 operations, auto-scroll to newest

### FR2: GPU and Resource Monitoring

**FR2.1: GPU Utilization Display**
- Current GPU utilization percentage (real-time)
- GPU temperature (if available via CUDA/nvml)
- GPU memory usage:
  - Total VRAM
  - Used VRAM
  - Free VRAM
  - Percentage bar visualization
- Historical GPU utilization graph (last 1 hour, 5-minute intervals)
- Alert indicator if GPU temperature exceeds 80C

**FR2.2: System Memory and CPU**
- System RAM usage:
  - Total RAM
  - Used RAM
  - Percentage visualization
- CPU usage by the Go process (percentage)
- Disk space in downloads directory:
  - Used space
  - Available space
  - Warning if <1GB free

**FR2.3: LLM Engine Status**
- llama.cpp runtime status (Loaded, Error, Not Initialized)
- Loaded model information:
  - Model name (Bunny v1.1 Llama-3-8B-V)
  - Model size
  - Quantization format (GGUF)
- stable-diffusion.cpp runtime status
- Current inference context usage (if applicable)
- Average inference time statistics:
  - Text generation (tokens/sec)
  - Image analysis (seconds per request)
  - Image generation (seconds per image)

### FR3: Multi-Canvas Management

**FR3.1: Canvas List View**
- Dedicated section listing all monitored canvases
- For each canvas display:
  - Canvas name
  - Canvas ID
  - Server URL
  - Connection status
  - Widget count
  - Last activity timestamp
  - Processing stats (requests today, success rate)
- Sort by: Name, Last Activity, Request Count
- Visual distinction for active vs idle canvases

**FR3.2: Canvas Selection/Filtering**
- Filter dashboard metrics by specific canvas
- "All Canvases" view (default) shows aggregate metrics
- Single canvas view shows canvas-specific metrics
- Quick toggle between canvases via dropdown or tabs

**FR3.3: Canvas Health Monitoring**
- Per-canvas error tracking
- Connection retry status
- Alert indicator for canvases with recent errors
- Last successful widget fetch timestamp

### FR4: Real-time Updates

**FR4.1: WebSocket or Server-Sent Events**
- Establish persistent connection for real-time updates
- Push updates for:
  - New AI operations
  - Completed tasks
  - GPU metrics (every 5 seconds)
  - Status changes
  - Error events
- Automatic reconnection on disconnect
- Connection status indicator in UI

**FR4.2: Update Frequency**
- GPU/Memory metrics: Every 5 seconds
- Processing queue: Every 2 seconds
- Completed tasks: Immediate push
- Canvas status: Every 10 seconds
- Error events: Immediate push

**FR4.3: Fallback Polling**
- If WebSocket/SSE fails, fall back to HTTP polling
- Polling interval: Every 10 seconds
- Indicate polling mode to user

### FR5: Authentication

**FR5.1: Password Protection**
- Simple password authentication using existing WEBUI_PWD environment variable
- Login page on first access
- Session-based authentication (cookie or token)
- Session expiry after 24 hours of inactivity
- No user management - single password only

**FR5.2: Security**
- HTTPS recommended but not required (local network deployment)
- No password storage in browser (session only)
- Logout functionality
- Session invalidation on logout

## Non-Functional Requirements

### NFR1: Performance
- Dashboard should load in <2 seconds on local network
- Real-time updates should have <500ms latency
- GPU metrics collection should not impact inference performance
- Memory footprint of web server should be <50MB

### NFR2: Simplicity
- Zero configuration required - web server starts automatically with backend
- Single port configuration (default: 3000, from existing PORT env var)
- No database required - all metrics in memory
- No persistent storage of metrics (resets on restart)

### NFR3: Compatibility
- Support modern browsers: Chrome, Firefox, Edge, Safari (last 2 versions)
- Responsive design for desktop (1920x1080, 2560x1440)
- Tablet support optional
- No mobile optimization required (enterprise workstation use case)

### NFR4: Maintainability
- Simple frontend stack (vanilla JS or minimal framework)
- Minimal dependencies
- Code should follow Go best practices and atomic design
- Clear separation between web server and core monitoring logic

### NFR5: Reliability
- Dashboard failure should not affect AI processing
- Graceful degradation if GPU metrics unavailable
- Handle missing data gracefully (show N/A or placeholder)
- Auto-reconnect on connection loss

## Technical Constraints

### TC1: Tech Stack Constraints
- Backend: Go 1.x (existing stack)
- Frontend: Keep it simple - vanilla JS, or lightweight framework (Alpine.js, htmx)
- No React/Vue/Angular - avoid heavy build toolchains
- CSS: Plain CSS or minimal framework (Tailwind CDN acceptable)
- WebSocket or Server-Sent Events for real-time updates

### TC2: Integration Constraints
- Must integrate with existing Go backend without major refactoring
- Should use existing Config struct and logging patterns
- Must not interfere with canvas monitoring logic
- GPU metrics should use existing CUDA bindings (when implemented)

### TC3: Deployment Constraints
- Web server runs in same process as backend (simplicity)
- Single binary deployment (Go's net/http)
- No separate web server process
- Static assets embedded in Go binary (embed package)

### TC4: Data Constraints
- All metrics stored in memory (volatile)
- No persistent database
- Historical data limited to recent window (1 hour max)
- Acceptable to lose metrics on restart

## Out of Scope

### Explicitly NOT Included

**Configuration Management:**
- NO ability to change .env settings via UI
- NO model selection interface
- NO provider configuration
- NO API key management
- Configuration remains file-based only

**Advanced Features:**
- NO user management (single password only)
- NO role-based access control
- NO email/Slack notifications
- NO alerting rules configuration
- NO metric export (Prometheus, etc.)
- NO log file download/viewing
- NO historical data persistence beyond current session

**AI Interaction:**
- NO ability to submit AI requests via UI
- NO prompt testing interface
- NO model interaction beyond monitoring
- NO canvas editing capabilities

**Infrastructure:**
- NO separate web server deployment
- NO reverse proxy setup
- NO SSL/TLS certificate management
- NO Docker/containerization for web UI

## Success Criteria

### Acceptance Criteria

**AC1: Dashboard Functionality**
- Dashboard loads successfully on http://localhost:3000
- All status widgets display correct real-time data
- GPU metrics update every 5 seconds
- Processing queue displays active and completed tasks
- Multi-canvas view shows all monitored canvases

**AC2: Real-time Updates**
- New AI operations appear in log within 2 seconds
- GPU metrics auto-refresh without page reload
- WebSocket connection indicator shows status
- Dashboard reconnects automatically on disconnect

**AC3: Multi-Canvas Support**
- Dashboard displays all monitored canvases
- Can filter metrics by specific canvas
- Per-canvas error tracking visible
- Canvas health status accurate

**AC4: Resource Monitoring**
- GPU utilization displays correctly
- Memory usage accurate
- GPU temperature shown (if available)
- Inference statistics calculated correctly

**AC5: Authentication**
- Login required on first access
- WEBUI_PWD authentication works
- Session persists for 24 hours
- Logout functionality works

**AC6: Error Handling**
- Graceful handling of missing GPU metrics
- Dashboard works when no canvases connected
- Clear error messages for connection issues
- No dashboard crashes on backend errors

### Performance Benchmarks
- Dashboard load time: <2 seconds
- WebSocket latency: <500ms
- Memory overhead: <50MB
- No measurable impact on AI inference performance

### User Validation
- Enterprise IT admin can monitor system health in <30 seconds
- Team lead can verify canvas monitoring status at a glance
- Consultant can check resource usage without reading logs
- All users can identify errors within 1 minute

## Future Considerations (Post-Phase 5)

**Potential Phase 6+ Enhancements:**
- SQLite persistence for historical metrics
- Downloadable processing history (CSV export)
- Advanced GPU metrics (power draw, clock speeds)
- Canvas widget visualization/preview
- Email alerts for critical errors
- Metric retention beyond current session
- API endpoint for external monitoring tools

**Not Planned:**
- Configuration UI (contradicts zero-config philosophy)
- AI interaction features (out of scope)
- User management (single-user tool)
- Cloud integration (local-only product)
