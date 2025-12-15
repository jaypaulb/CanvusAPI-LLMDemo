# Product Roadmap

## Phase 0: Installation & Packaging Infrastructure

1. [ ] NSIS Installer for Windows — Build CanvusLocalLLM-Setup.exe using NSIS with 4-step wizard (license/location, install binaries, create .env.example, final page with "Open Configuration" button), optional Windows Service creation checkbox, and optional PATH addition `M`
2. [ ] Debian Package for Linux — Create .deb package with proper directory structure (/opt/canvuslocallm/), post-install script to create .env.example, and post-install message showing configuration file location `S`
3. [ ] Tarball Distribution for Linux — Build .tar.gz archive with install.sh script that extracts to /opt/canvuslocallm/, creates .env.example, and displays configuration instructions for non-Debian distributions `S`
4. [ ] Configuration Template System — Create well-commented .env.example template with sections for required Canvus settings, AI model selection (llava-7b/13b, llama-3.2-vision), privacy modes (local-only/hybrid/cloud-preferred), performance settings, and cloud fallback configuration `S`
5. [ ] First-Run Model Download — Implement model download system that triggers on first application run (not installer), with progress bar showing download speed/ETA, resumable HTTP downloads, SHA256 checksum verification, disk space validation, and retry logic `M`
6. [ ] Configuration Validation System — Build startup validation that checks for .env file existence, validates required values (Canvus server, API key, canvas ID), provides helpful error messages with examples, and guides users to copy .env.example if missing `S`
7. [ ] Optional Service Creation — Implement Windows Service installer (using Windows Service API) and systemd unit file generation for Linux background service with automatic startup, logging, and management via sc.exe or systemctl `M`

## Phase 1: Codebase Architecture Refactoring

8. [ ] Extract PDF Processing Organism — Create `pdfprocessor/` package by extracting PDF analysis functions from handlers.go into a cohesive organism with atoms (text extraction, chunking) and molecules (summary generation, token management) `M`
9. [ ] Extract Image Generation Organism — Create `imagegen/` package by extracting DALL-E and Azure image generation logic into a dedicated organism with provider abstraction `S`
10. [ ] Extract Canvas Analysis Organism — Create `canvasanalyzer/` package by extracting canvas analysis and synthesis logic into a separate organism that understands spatial relationships `S`
11. [ ] Eliminate Global Config Variable — Refactor handlers.go to accept config via dependency injection (pass through Monitor struct or function parameters) removing shared global state `S`
12. [ ] Create Handwriting Recognition Organism — Extract Google Vision API integration from handlers.go into `handwritingrecog/` package with clean abstraction for vision providers `S`
13. [ ] Split Remaining Handler Logic — Break down remaining handlers.go code (~1000 lines after extractions) into atomic functions and molecules organized by responsibility `M`

## Phase 2: Embedded Local Multimodal LLM (llama.cpp)

14. [ ] llama.cpp Integration — Create `llamaruntime/` package (organism) with Go CGo bindings to llama.cpp using go-skynet/go-llama.cpp or tcpipuk/llama-go, including atoms for model loading, inference, and context management `L`
15. [ ] Cross-Platform llama.cpp Build — Integrate llama.cpp CMake build process into Go build pipeline for Windows (Visual Studio 2022/MSYS), Linux (amd64/ARM64), and macOS (Intel/ARM64), bundling native libraries (.dll/.so/.dylib) with application binary `L`
16. [ ] Model Provisioning Integration — Connect first-run model download system to llamaruntime package, supporting multiple models (LLaVA 7B/13B, Llama 3.2-Vision 11B) based on .env configuration, with automatic multimodal projection weight downloads `M`
17. [ ] Smart Provider Routing — Create provider abstraction layer that routes requests to embedded llama.cpp (default), external LLM servers (Ollama, LLaMA, LM Studio), or cloud providers (OpenAI, Azure) based on configuration with automatic fallback when embedded runtime unavailable `L`
18. [ ] Model Management Interface — Build web UI component for downloading, switching, and deleting GGUF models with size/capability comparison (quantization levels: 4-bit, 5-bit, 8-bit), disk space monitoring, and one-click model updates `M`
19. [ ] Embedded Runtime Health Monitoring — Add health checks for llama.cpp inference context, memory usage monitoring, automatic recovery from inference failures, graceful degradation to cloud providers when local runtime unavailable, and startup diagnostics `S`

## Phase 2.5: Image Generation Integration (Future)

20. [ ] stable-diffusion.cpp Integration — Create `sdruntime/` package with Go CGo bindings to stable-diffusion.cpp for local image generation, following same pattern as llama.cpp integration with cross-platform builds `L`
21. [ ] Local Image Generation Pipeline — Implement text-to-image generation using embedded stable-diffusion.cpp with model management, prompt processing, and automatic placement of generated images on canvas `M`
22. [ ] Hybrid Image Generation — Build intelligent routing that uses local stable-diffusion.cpp for privacy-sensitive requests and falls back to OpenAI DALL-E or Azure OpenAI for complex or high-quality image generation based on user configuration `M`

## Phase 3: Enhanced Local LLM Support

23. [ ] Multi-Provider Testing Framework — Build comprehensive test suite that validates OpenAI, Azure OpenAI, embedded llama.cpp, external Ollama, LLaMA, and LM Studio compatibility with mock servers and real endpoint tests `M`
24. [ ] Provider-Specific Optimizations — Implement adaptive token limits, retry strategies, and error handling tailored to different LLM providers' characteristics and limitations (llama.cpp context window vs OpenAI API differences) `M`
25. [ ] Local LLM Health Monitoring — Extend health check endpoints and connection monitoring for all local LLM servers with automatic reconnection and graceful degradation when providers are unavailable `S`
26. [ ] Privacy Mode Configuration — Implement preset privacy modes from .env file (local-only: all processing on-device, hybrid: local first with cloud fallback, cloud-preferred: use OpenAI/Azure when faster) with clear enforcement and audit logging `M`

## Phase 4: Web UI Enhancement

27. [ ] Real-time Processing Dashboard — Build web interface showing active canvas monitoring status, processing queue, success/failure metrics, recent AI operations, which provider handled each request, and embedded runtime status (memory, context usage) with live updates `L`
28. [ ] Configuration Management Interface — Create web-based settings panel for adjusting token limits, timeouts, provider selection, provider priority/fallback rules, llama.cpp context size, and operational parameters without restarting the service `M`
29. [ ] API Usage Analytics — Add detailed metrics tracking for token consumption, API costs (cloud providers), response times, error rates, provider usage distribution, and embedded runtime performance with exportable reports and visualization `M`
30. [ ] Multi-Canvas Management — Extend web UI to support monitoring multiple Canvus workspaces simultaneously with per-canvas configuration and status `L`

## Phase 5: Production Hardening

31. [ ] Structured Logging and Observability — Replace current logging with structured JSON logging (logrus/zap), add distributed tracing (OpenTelemetry), and integrate with monitoring systems (Prometheus, Grafana) including llama.cpp performance metrics `M`
32. [ ] Database Persistence Layer — Add database (PostgreSQL/SQLite) for storing processing history, audit trails, configuration backups, model download metadata, llama.cpp inference logs, and operational metrics instead of file-only logs `L`
33. [ ] Authentication and Authorization — Implement role-based access control (RBAC) for web UI, API endpoints, canvas access, and model management with user management and audit logging `L`
34. [ ] Rate Limiting and Quotas — Build token bucket rate limiting for API requests, per-user quotas, provider-specific throttling, and configurable limits to prevent abuse and control costs `M`
35. [ ] Installer Distribution Automation — Automate building of platform-specific installers (NSIS for Windows, .deb for Debian/Ubuntu, .tar.gz for other Linux) with CI/CD pipeline that bundles Go application + llama.cpp libraries, signs binaries, creates checksums, and uploads to release artifacts `L`
36. [ ] Disaster Recovery — Implement configuration backup/restore, graceful shutdown with in-flight request handling, automatic recovery from crashes with queue persistence, and llama.cpp context state preservation `M`

> Notes
> - Order items by technical dependencies and product architecture
> - Each item should represent an end-to-end (frontend + backend) functional and testable feature
> - Phase 0 (Installation) must be completed FIRST to establish user-friendly deployment before architecture changes
> - Phase 1 (Architecture Refactoring) is foundational and should be completed before Phase 2 LLM work
> - Phase 2 (Embedded llama.cpp) depends on Phase 0 installer infrastructure and Phase 1 provider abstraction
> - Phase 2 items integrate with Phase 0 model download system (item 5) and configuration validation (item 6)
> - Phase 2.5 (Image Generation) is future work, can be deferred until Phase 2 is stable
> - Phase 3-5 items can be parallelized once embedded LLM infrastructure is established
> - Installers in Phase 0 must be tested across all target platforms (Windows 10/11, Debian/Ubuntu, other Linux distributions)
> - llama.cpp builds must be tested across all target platforms (Windows MSVC/MSYS, Linux amd64/ARM64, macOS Intel/ARM64)
> - CGo integration requires proper build toolchain setup (CMake, C++ compiler) on development machines
