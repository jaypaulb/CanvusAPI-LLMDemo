# Production Hardening Requirements

## Overview

This specification defines the requirements for hardening CanvusLocalLLM for production deployment. The goal is to transform the current development-grade application into a robust, observable, maintainable production system suitable for enterprise environments.

## Target Users

- **Enterprise IT Administrators**: Deploying and monitoring the application in production environments
- **DevOps Teams**: Managing CI/CD pipelines and automated deployments
- **Support Teams**: Diagnosing issues in production using logs and metrics
- **End Users**: Benefiting from stability, reliability, and security improvements

## Business Requirements

### BR-1: Production Observability
The application must provide comprehensive visibility into its runtime behavior to enable rapid diagnosis and resolution of production issues.

**Success Criteria:**
- Operations team can diagnose 90% of issues from logs alone without code inspection
- All critical operations (AI inference, canvas monitoring, API calls) are logged with timing and metadata
- Logs can be ingested by standard log aggregation tools (ELK, Splunk, CloudWatch)

### BR-2: Operational History
The application must maintain persistent records of AI processing operations to support auditing, debugging, and usage analysis.

**Success Criteria:**
- All AI processing requests and responses are persisted to database
- Historical data can be queried for troubleshooting and usage analysis
- Metrics can be aggregated for operational dashboards

### BR-3: Secure Deployment
The application must provide authentication mechanisms to protect administrative interfaces and sensitive data.

**Success Criteria:**
- Web UI requires authentication before displaying any operational data
- Authentication mechanism is simple and appropriate for local/on-premise deployment
- Failed authentication attempts are logged

### BR-4: Automated Distribution
The application packaging and distribution must be fully automated to ensure consistent, reproducible releases.

**Success Criteria:**
- Installers for Windows and Linux are built automatically on every release
- Build process is reproducible across different environments
- Release artifacts are published automatically with checksums

### BR-5: Reliability and Graceful Degradation
The application must handle shutdown scenarios gracefully without data loss or corruption.

**Success Criteria:**
- In-flight AI operations complete before shutdown when possible
- Application state is persisted before termination
- No orphaned resources (files, database connections, goroutines) after shutdown
- Clean restart after graceful shutdown

## Functional Requirements

### FR-1: Structured JSON Logging

**Description:**
Replace the current simple logging mechanism with structured JSON logging that provides rich metadata and is machine-parseable.

**Requirements:**
1. All log entries must be in JSON format with standardized fields
2. Each log entry must include: timestamp (ISO8601), level (debug/info/warn/error), message, source (file/function), correlation_id
3. Application-specific fields must be included as structured data:
   - AI operations: model, prompt_tokens, completion_tokens, duration_ms, endpoint
   - Canvas operations: canvas_id, widget_id, operation_type
   - HTTP operations: method, url, status_code, duration_ms
   - GPU metrics: vram_used_mb, vram_total_mb, gpu_utilization_percent (when available)
4. Log levels must be configurable via environment variable (default: INFO)
5. Console output should use human-readable format for development, JSON for production
6. Log rotation must be supported (size-based and time-based)

**Technical Constraints:**
- Must use either `logrus` or `zap` (decision to be made in spec)
- Must integrate cleanly with existing `logging.LogHandler` pattern
- Must not introduce significant performance overhead (<5ms per log call)

**Edge Cases:**
- Sensitive data (API keys, passwords) must never appear in logs
- Extremely long prompts/responses must be truncated (max 1000 chars) with indication
- Binary data must be represented as base64 or omitted

### FR-2: llama.cpp Performance Metrics

**Description:**
Capture and log detailed performance metrics from llama.cpp inference operations.

**Requirements:**
1. Track per-inference metrics:
   - Prompt evaluation time (ms)
   - Token generation time (ms)
   - Tokens per second (prompt and generation separately)
   - Total inference duration (ms)
   - Prompt tokens, completion tokens, total tokens
2. Track GPU utilization when available:
   - VRAM usage before/after inference
   - GPU utilization percentage
   - CUDA kernel execution time
3. Include model metadata in logs:
   - Model name/path
   - Quantization level (if applicable)
   - Context size
4. Aggregate metrics for dashboard display
5. Track failed inference attempts with error details

**Technical Constraints:**
- Metrics collection must not slow inference by more than 2%
- GPU metrics require CUDA/NVML integration
- Metrics must be accessible via both logs and database

### FR-3: SQLite Database Persistence

**Description:**
Implement SQLite database to persist processing history, metrics, and operational data.

**Requirements:**
1. Database schema must include tables for:
   - `processing_history`: AI operations (prompt, response, model, tokens, duration, status)
   - `canvas_events`: Canvas monitoring events (widget updates, new widgets, deletions)
   - `performance_metrics`: Aggregated performance data (inference times, token counts, GPU usage)
   - `error_log`: Error events with full context
   - `system_metrics`: Application health metrics (uptime, memory, goroutines)
2. Database must be located in application data directory (not install directory)
   - Windows: `%APPDATA%\CanvusLocalLLM\data.db`
   - Linux: `~/.canvuslocallm/data.db`
3. Schema migrations must be automated and versioned
4. Database writes must be asynchronous to avoid blocking main operations
5. Retention policy must be configurable (default: 30 days for processing_history, 7 days for metrics)
6. Automatic vacuum and optimization on startup
7. Query interface for web UI to access historical data

**Technical Constraints:**
- Must use `github.com/mattn/go-sqlite3` (CGo-based driver)
- Must use `github.com/golang-migrate/migrate` for migrations
- Database file size must be monitored and logged when exceeding thresholds
- Write throughput must support at least 100 operations/second without blocking

**Edge Cases:**
- Database corruption: Detect and recreate from scratch with logged warning
- Disk full: Gracefully degrade to logging-only mode with alert
- Migration failures: Halt startup and log clear error message

### FR-4: Web UI Authentication

**Description:**
Add password authentication to the web UI dashboard.

**Requirements:**
1. Authentication must protect all web UI routes (currently status page)
2. Login page presented on first access to any protected route
3. Session-based authentication with configurable timeout (default: 24 hours)
4. Password configured via `WEBUI_PWD` environment variable (existing)
5. Failed login attempts logged with IP address
6. Rate limiting on login attempts (max 5 per minute per IP)
7. Logout functionality
8. Session persistence across application restarts (optional enhancement)

**Technical Constraints:**
- Must use Go standard library `net/http` with session cookies
- Sessions stored in memory (persistent storage optional for future)
- Password must be hashed in memory, not stored in plain text
- Use secure cookies (httpOnly, sameSite, secure when HTTPS)

**Security Requirements:**
- Timing-safe password comparison to prevent timing attacks
- CSRF protection for logout
- Clear security headers (X-Frame-Options, X-Content-Type-Options)

**Edge Cases:**
- First-time setup with no password set: Display error message to set WEBUI_PWD
- Password change while sessions active: Invalidate all sessions

### FR-5: CI/CD Installer Automation

**Description:**
Automate the building and distribution of Windows and Linux installers using GitHub Actions.

**Requirements:**
1. GitHub Actions workflow triggered on:
   - Git tags matching `v*.*.*` (production releases)
   - Manual workflow dispatch (testing)
2. Build artifacts for each release:
   - Windows: `CanvusLocalLLM-Setup-{version}.exe` (NSIS installer)
   - Linux Debian: `canvuslocallm_{version}_amd64.deb`
   - Linux Generic: `canvuslocallm-{version}-linux-amd64.tar.gz`
3. Each installer must include:
   - Go application binary (compiled with CGo for llama.cpp)
   - llama.cpp shared libraries (with CUDA support)
   - stable-diffusion.cpp shared libraries (with CUDA support)
   - Bunny v1.1 model (or download script if too large)
   - Configuration template (.env.example)
   - README and LICENSE files
4. Automated testing of installers:
   - Install on clean VM (Windows Server, Ubuntu)
   - Verify binary execution
   - Run smoke tests
5. Publish to GitHub Releases with:
   - Release notes (auto-generated from commits)
   - SHA256 checksums for all artifacts
   - Installation instructions
6. Optional: Code signing for Windows executables

**Technical Constraints:**
- Must use `ubuntu-latest` and `windows-latest` GitHub runners
- Build must be reproducible (pinned dependencies, versioned tools)
- Total build time target: <20 minutes
- Artifact storage: GitHub Releases (not Actions artifacts due to size)

**Build Pipeline Steps:**
1. Checkout code
2. Set up Go environment (specific version from .go-version)
3. Build llama.cpp with CUDA (separate job for Windows/Linux)
4. Build stable-diffusion.cpp with CUDA
5. Build Go application with CGo
6. Create platform-specific installer package
7. Run installer smoke tests
8. Generate checksums
9. Upload to GitHub Releases

**Edge Cases:**
- Build failure: Annotate PR/commit with failure reason
- Large model files: Consider download-on-demand vs bundling
- CUDA compatibility: Document required driver versions

### FR-6: Graceful Shutdown

**Description:**
Implement comprehensive graceful shutdown mechanism that ensures clean termination of all application components.

**Requirements:**
1. Shutdown must be triggered by:
   - SIGINT (Ctrl+C)
   - SIGTERM (systemd/service manager)
   - Context cancellation (programmatic)
2. Shutdown sequence must:
   - Stop accepting new AI processing requests immediately
   - Wait for in-flight AI operations to complete (with timeout: 30 seconds)
   - Flush all pending database writes
   - Close database connections cleanly
   - Clean up temporary files in downloads directory
   - Close all HTTP connections
   - Log shutdown completion
3. Shutdown timeout must be configurable (default: 60 seconds total)
4. Force shutdown if graceful shutdown exceeds timeout
5. Exit codes must indicate shutdown reason:
   - 0: Clean shutdown
   - 1: Error during shutdown
   - 130: Interrupted (SIGINT)
   - 143: Terminated (SIGTERM)

**Technical Constraints:**
- Must use context propagation to signal shutdown to all goroutines
- Must track in-flight operations using sync.WaitGroup
- Database flush must use transactions to ensure consistency
- llama.cpp context must be freed before shutdown

**Recovery Requirements:**
- Application must restart cleanly after graceful shutdown
- No database corruption after shutdown
- All temporary files cleaned up
- No orphaned GPU memory allocations

**Edge Cases:**
- Shutdown during model loading: Abort loading and exit
- Multiple shutdown signals: Immediate force shutdown on second signal
- Database locked during shutdown: Log warning and continue shutdown
- In-flight operations exceed timeout: Log which operations timed out

### FR-7: Enhanced Error Reporting

**Description:**
Improve error handling and reporting throughout the application to support production troubleshooting.

**Requirements:**
1. All errors must include:
   - Error message (user-friendly)
   - Error context (operation being performed)
   - Stack trace (in logs, not user-facing)
   - Correlation ID (to trace across operations)
2. Transient errors must be automatically retried with exponential backoff
3. Permanent errors must be logged with full context and surfaced to UI
4. Error metrics tracked in database:
   - Error frequency by type
   - Error rate over time
   - Most common error messages
5. Critical errors trigger alerts (log level: ERROR)

**Error Categories:**
- **Transient**: Network timeouts, temporary API failures → Retry
- **Configuration**: Missing env vars, invalid values → Fail fast with clear message
- **Resource**: Out of memory, disk full → Graceful degradation
- **AI Model**: Model load failure, inference error → Log and report to UI
- **Data**: Invalid canvas data, corrupt PDFs → Skip and log

## Non-Functional Requirements

### NFR-1: Performance
- Logging overhead: <5ms per log call (p99)
- Database write latency: <50ms per operation (p99)
- Graceful shutdown time: <10 seconds under normal load
- CI/CD build time: <20 minutes
- Web UI authentication: <100ms per request

### NFR-2: Reliability
- Database corruption resistance: Auto-detect and recover
- Graceful degradation on disk full: Continue with logging-only
- Crash recovery: Automatic restart via systemd/Windows Service

### NFR-3: Maintainability
- Structured logs machine-parseable (JSON)
- Database schema versioned with migrations
- Build process fully documented and reproducible
- Error messages actionable (what went wrong, what to do)

### NFR-4: Security
- Passwords never logged or exposed in plain text
- Authentication tokens secure (httpOnly cookies)
- Rate limiting on authentication endpoints
- Sensitive data scrubbed from logs

### NFR-5: Compatibility
- Windows 10/11, Windows Server 2019+
- Ubuntu 20.04+, Debian 11+
- Go 1.21+ (for updated standard library features)
- NVIDIA RTX GPUs with CUDA 12.0+ drivers

### NFR-6: Scalability
- Database must handle 10,000+ operations per day
- Log files must not exceed 500MB before rotation
- Web UI must remain responsive with 1000+ database records

## Implementation Dependencies

### Phase Dependencies
- FR-1 (Structured Logging) is a prerequisite for FR-2 (llama.cpp metrics)
- FR-3 (Database) depends on FR-1 for consistent logging of DB operations
- FR-5 (CI/CD) depends on FR-6 (Graceful Shutdown) for proper testing
- FR-4 (Web UI Auth) is independent and can be implemented in parallel

### External Dependencies
- Logging: `github.com/sirupsen/logrus` OR `go.uber.org/zap`
- Database: `github.com/mattn/go-sqlite3`, `github.com/golang-migrate/migrate`
- Authentication: Go standard library (`crypto/bcrypt`, `net/http`)
- CI/CD: GitHub Actions, NSIS (Windows), dpkg (Linux)

## Testing Requirements

### Test Categories

**Unit Tests:**
- Structured logging: JSON serialization, field validation
- Database: CRUD operations, migrations, retention policies
- Authentication: Password hashing, session management, rate limiting
- Graceful shutdown: Context cancellation, timeout handling

**Integration Tests:**
- End-to-end logging pipeline (log → database → query)
- Database persistence across restarts
- Authentication flow (login → session → protected route → logout)
- Shutdown sequence (signal → wait → cleanup → exit)

**Performance Tests:**
- Logging throughput: 1000 log calls in <5 seconds
- Database write throughput: 100 writes/second sustained
- Authentication latency: <100ms per request (p99)

**Platform Tests:**
- Windows: NSIS installer, Windows Service integration
- Linux: .deb package, systemd integration, .tar.gz extraction
- Both: Clean installation, upgrade from previous version, uninstallation

**Smoke Tests:**
- Install on clean system
- Start application
- Verify logs created in correct location
- Verify database created with correct schema
- Access web UI and authenticate
- Trigger AI processing and verify database record
- Graceful shutdown and verify cleanup

## Acceptance Criteria

### Structured Logging (FR-1)
- [ ] All log entries output in JSON format
- [ ] GPU metrics included in AI operation logs (when available)
- [ ] Log level configurable via CANVUSLLM_LOG_LEVEL env var
- [ ] Console uses human-readable format, file uses JSON
- [ ] Sensitive data (API keys) never appears in logs
- [ ] Log rotation works correctly (size: 100MB, keep: 5 files)

### Database Persistence (FR-3)
- [ ] SQLite database created on first run at correct location
- [ ] All AI operations persisted with full metadata
- [ ] Database migrations run automatically on startup
- [ ] Retention policy deletes old records correctly (30 days)
- [ ] Database size monitored and logged when exceeding 500MB
- [ ] Queries from web UI return correct historical data

### Web UI Authentication (FR-4)
- [ ] Unauthenticated access to any protected route redirects to login
- [ ] Successful login creates session and redirects to requested route
- [ ] Session persists across requests for configured timeout (24 hours)
- [ ] Failed login attempts rate-limited (max 5/minute)
- [ ] Logout clears session and redirects to login page
- [ ] Missing WEBUI_PWD displays clear error message

### CI/CD Automation (FR-5)
- [ ] GitHub Actions workflow builds Windows installer on tag push
- [ ] GitHub Actions workflow builds Linux .deb and .tar.gz on tag push
- [ ] Installers include all required components (binary, libraries, model)
- [ ] Smoke tests pass on clean Windows and Linux VMs
- [ ] Release artifacts uploaded to GitHub Releases with checksums
- [ ] Build is reproducible (same tag → identical artifacts)

### Graceful Shutdown (FR-6)
- [ ] SIGINT triggers graceful shutdown sequence
- [ ] SIGTERM triggers graceful shutdown sequence
- [ ] In-flight AI operations allowed to complete (up to 30s timeout)
- [ ] Database connections closed cleanly
- [ ] Temporary files in downloads/ directory removed
- [ ] Shutdown logged with duration and reason
- [ ] Application restarts cleanly after graceful shutdown
- [ ] Second shutdown signal triggers immediate force shutdown

## Out of Scope

The following items are explicitly excluded from this phase:

- **Advanced Monitoring**: Prometheus metrics, distributed tracing, APM integration
- **Multi-Instance Deployment**: Clustering, load balancing, shared state
- **Database Replication**: Backup, restore, replication to external databases
- **Advanced Authentication**: OAuth, SAML, multi-user support, role-based access
- **Configuration UI**: Web-based configuration editor (still uses .env file)
- **Telemetry**: Usage statistics, crash reporting to external services
- **Model Management UI**: Model selection, download, switching via web UI
- **GPU Monitoring UI**: Real-time GPU metrics dashboard (logged only)

These may be considered for future phases after production deployment validation.

## Risk Assessment

### High Risk
- **Database corruption**: Mitigated by automatic detection and recovery
- **Build failures in CI/CD**: Mitigated by reproducible builds and pinned dependencies
- **Installer compatibility**: Mitigated by testing on multiple OS versions

### Medium Risk
- **Logging performance overhead**: Mitigated by async logging and benchmarking
- **Graceful shutdown timeout**: Mitigated by configurable timeouts and force shutdown
- **Authentication security**: Mitigated by standard crypto libraries and rate limiting

### Low Risk
- **Database size growth**: Mitigated by retention policies and monitoring
- **Log file rotation**: Well-established library support
- **Session management**: Standard Go patterns

## Success Metrics

Post-deployment metrics to validate success:

1. **Observability**: Time to diagnose production issues reduced by 50%
2. **Reliability**: Unplanned restarts reduced to <1 per month
3. **Deployment**: Time from code commit to release reduced to <30 minutes
4. **Security**: Zero authentication bypass incidents
5. **Performance**: p99 latency increase <5% from logging/database overhead

## References

- [Go Structured Logging Discussion](https://github.com/golang/go/discussions/54763)
- [logrus Documentation](https://github.com/sirupsen/logrus)
- [zap Documentation](https://github.com/uber-go/zap)
- [SQLite in Go Best Practices](https://github.com/mattn/go-sqlite3)
- [Graceful Shutdown in Go](https://pkg.go.dev/context)
- [GitHub Actions Documentation](https://docs.github.com/en/actions)
