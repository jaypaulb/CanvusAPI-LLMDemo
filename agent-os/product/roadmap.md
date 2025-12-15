# Product Roadmap

## Phase 1: Codebase Architecture Refactoring

1. [ ] Extract PDF Processing Organism — Create `pdfprocessor/` package by extracting PDF analysis functions from handlers.go into a cohesive organism with atoms (text extraction, chunking) and molecules (summary generation, token management) `M`
2. [ ] Extract Image Generation Organism — Create `imagegen/` package by extracting DALL-E and Azure image generation logic into a dedicated organism with provider abstraction `S`
3. [ ] Extract Canvas Analysis Organism — Create `canvasanalyzer/` package by extracting canvas analysis and synthesis logic into a separate organism that understands spatial relationships `S`
4. [ ] Eliminate Global Config Variable — Refactor handlers.go to accept config via dependency injection (pass through Monitor struct or function parameters) removing shared global state `S`
5. [ ] Create Handwriting Recognition Organism — Extract Google Vision API integration from handlers.go into `handwritingrecog/` package with clean abstraction for vision providers `S`
6. [ ] Split Remaining Handler Logic — Break down remaining handlers.go code (~1000 lines after extractions) into atomic functions and molecules organized by responsibility `M`

## Phase 2: Bundled Local Multimodal LLM

7. [ ] Ollama Runtime Integration — Create `ollamaruntime/` package (organism) with atoms for process lifecycle management, health checking, and model operations using Ollama's Go client library `M`
8. [ ] Cross-Platform Installation Scripts — Build platform-specific installers (Windows .bat, Linux/macOS .sh) that detect architecture, download appropriate Ollama binary, install to application directory, and verify installation `M`
9. [ ] Automatic Model Provisioning — Implement first-run setup that automatically pulls default multimodal model (LLaVA 7B or Llama 3.2-Vision 11B) on first launch with progress tracking and disk space validation `M`
10. [ ] Smart Provider Routing — Create provider abstraction layer that routes requests to bundled Ollama (default), external Ollama servers, OpenAI, or Azure based on configuration with automatic fallback when bundled runtime unavailable `L`
11. [ ] Model Management Interface — Build web UI component for downloading, switching, and deleting models with size/capability comparison, disk space monitoring, and one-click model updates `M`
12. [ ] Bundled Runtime Health Monitoring — Add health checks for Ollama process, automatic restart on failure, graceful degradation to cloud providers when local runtime unavailable, and startup diagnostics `S`

## Phase 3: Enhanced Local LLM Support

13. [ ] Multi-Provider Testing Framework — Build comprehensive test suite that validates OpenAI, Azure OpenAI, bundled Ollama, external Ollama, LLaMA, and LM Studio compatibility with mock servers and real endpoint tests `M`
14. [ ] Provider-Specific Optimizations — Implement adaptive token limits, retry strategies, and error handling tailored to different LLM providers' characteristics and limitations (Ollama vs OpenAI API differences) `M`
15. [ ] Local LLM Health Monitoring — Extend health check endpoints and connection monitoring for all local LLM servers with automatic reconnection and graceful degradation when providers are unavailable `S`
16. [ ] Configuration Profiles — Create preset configuration profiles (Privacy-First/Bundled-Only, Balanced/Mixed, Performance/Cloud-Heavy) for easily switching between provider strategies without editing individual settings `M`

## Phase 4: Web UI Enhancement

17. [ ] Real-time Processing Dashboard — Build web interface showing active canvas monitoring status, processing queue, success/failure metrics, recent AI operations, and which provider handled each request with live updates `L`
18. [ ] Configuration Management Interface — Create web-based settings panel for adjusting token limits, timeouts, provider selection, provider priority/fallback rules, and operational parameters without restarting the service `M`
19. [ ] API Usage Analytics — Add detailed metrics tracking for token consumption, API costs (cloud providers), response times, error rates, and provider usage distribution with exportable reports and visualization `M`
20. [ ] Multi-Canvas Management — Extend web UI to support monitoring multiple Canvus workspaces simultaneously with per-canvas configuration and status `L`

## Phase 5: Production Hardening

21. [ ] Structured Logging and Observability — Replace current logging with structured JSON logging (logrus/zap), add distributed tracing (OpenTelemetry), and integrate with monitoring systems (Prometheus, Grafana) `M`
22. [ ] Database Persistence Layer — Add database (PostgreSQL/SQLite) for storing processing history, audit trails, configuration backups, model download metadata, and operational metrics instead of file-only logs `L`
23. [ ] Authentication and Authorization — Implement role-based access control (RBAC) for web UI, API endpoints, canvas access, and model management with user management and audit logging `L`
24. [ ] Rate Limiting and Quotas — Build token bucket rate limiting for API requests, per-user quotas, provider-specific throttling, and configurable limits to prevent abuse and control costs `M`
25. [ ] Deployment Automation — Create platform-specific installers (MSI for Windows, .deb/.rpm for Linux, .pkg for macOS), Kubernetes manifests, Helm charts, and comprehensive deployment documentation for cloud and on-premises environments `L`
26. [ ] Disaster Recovery — Implement configuration backup/restore, graceful shutdown with in-flight request handling, automatic recovery from crashes with queue persistence, and Ollama runtime state preservation `M`

> Notes
> - Order items by technical dependencies and product architecture
> - Each item should represent an end-to-end (frontend + backend) functional and testable feature
> - Phase 1 (Architecture Refactoring) is foundational and should be completed before major feature additions
> - Phase 2 (Bundled LLM) is the highest priority new capability, providing zero-config deployment
> - Phase 2 items depend on atomic architecture from Phase 1 (especially provider abstraction)
> - Phase 3-5 items can be parallelized once bundled LLM infrastructure is established
> - Installation scripts in Phase 2 must be tested across all target platforms (Windows, Linux amd64/ARM64, macOS Intel/ARM64)
