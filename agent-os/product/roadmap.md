# Product Roadmap

## Phase 1: Codebase Architecture Refactoring

1. [ ] Extract PDF Processing Organism — Create `pdfprocessor/` package by extracting PDF analysis functions from handlers.go into a cohesive organism with atoms (text extraction, chunking) and molecules (summary generation, token management) `M`
2. [ ] Extract Image Generation Organism — Create `imagegen/` package by extracting DALL-E and Azure image generation logic into a dedicated organism with provider abstraction `S`
3. [ ] Extract Canvas Analysis Organism — Create `canvasanalyzer/` package by extracting canvas analysis and synthesis logic into a separate organism that understands spatial relationships `S`
4. [ ] Eliminate Global Config Variable — Refactor handlers.go to accept config via dependency injection (pass through Monitor struct or function parameters) removing shared global state `S`
5. [ ] Create Handwriting Recognition Organism — Extract Google Vision API integration from handlers.go into `handwritingrecog/` package with clean abstraction for vision providers `S`
6. [ ] Split Remaining Handler Logic — Break down remaining handlers.go code (~1000 lines after extractions) into atomic functions and molecules organized by responsibility `M`

## Phase 2: Embedded Local Multimodal LLM (llama.cpp)

7. [ ] llama.cpp Integration — Create `llamaruntime/` package (organism) with Go CGo bindings to llama.cpp using go-skynet/go-llama.cpp or tcpipuk/llama-go, including atoms for model loading, inference, and context management `L`
8. [ ] Cross-Platform llama.cpp Build — Integrate llama.cpp CMake build process into Go build pipeline for Windows (Visual Studio 2022/MSYS), Linux (amd64/ARM64), and macOS (Intel/ARM64), bundling native libraries (.dll/.so/.dylib) with application binary `L`
9. [ ] Automatic Model Provisioning — Implement first-run setup that automatically downloads default LLaVA 7B GGUF model (~4GB) and multimodal projection weights (llava-mmproj.gguf) on first launch with progress tracking, resumable downloads, and disk space validation `M`
10. [ ] Smart Provider Routing — Create provider abstraction layer that routes requests to embedded llama.cpp (default), external LLM servers (Ollama, LLaMA, LM Studio), or cloud providers (OpenAI, Azure) based on configuration with automatic fallback when embedded runtime unavailable `L`
11. [ ] Model Management Interface — Build web UI component for downloading, switching, and deleting GGUF models with size/capability comparison (quantization levels: 4-bit, 5-bit, 8-bit), disk space monitoring, and one-click model updates `M`
12. [ ] Embedded Runtime Health Monitoring — Add health checks for llama.cpp inference context, memory usage monitoring, automatic recovery from inference failures, graceful degradation to cloud providers when local runtime unavailable, and startup diagnostics `S`

## Phase 2.5: Image Generation Integration (Future)

13. [ ] stable-diffusion.cpp Integration — Create `sdruntime/` package with Go CGo bindings to stable-diffusion.cpp for local image generation, following same pattern as llama.cpp integration with cross-platform builds `L`
14. [ ] Local Image Generation Pipeline — Implement text-to-image generation using embedded stable-diffusion.cpp with model management, prompt processing, and automatic placement of generated images on canvas `M`
15. [ ] Hybrid Image Generation — Build intelligent routing that uses local stable-diffusion.cpp for privacy-sensitive requests and falls back to OpenAI DALL-E or Azure OpenAI for complex or high-quality image generation based on user configuration `M`

## Phase 3: Enhanced Local LLM Support

16. [ ] Multi-Provider Testing Framework — Build comprehensive test suite that validates OpenAI, Azure OpenAI, embedded llama.cpp, external Ollama, LLaMA, and LM Studio compatibility with mock servers and real endpoint tests `M`
17. [ ] Provider-Specific Optimizations — Implement adaptive token limits, retry strategies, and error handling tailored to different LLM providers' characteristics and limitations (llama.cpp context window vs OpenAI API differences) `M`
18. [ ] Local LLM Health Monitoring — Extend health check endpoints and connection monitoring for all local LLM servers with automatic reconnection and graceful degradation when providers are unavailable `S`
19. [ ] Configuration Profiles — Create preset configuration profiles (Privacy-First/Embedded-Only, Balanced/Mixed, Performance/Cloud-Heavy) for easily switching between provider strategies without editing individual settings `M`

## Phase 4: Web UI Enhancement

20. [ ] Real-time Processing Dashboard — Build web interface showing active canvas monitoring status, processing queue, success/failure metrics, recent AI operations, which provider handled each request, and embedded runtime status (memory, context usage) with live updates `L`
21. [ ] Configuration Management Interface — Create web-based settings panel for adjusting token limits, timeouts, provider selection, provider priority/fallback rules, llama.cpp context size, and operational parameters without restarting the service `M`
22. [ ] API Usage Analytics — Add detailed metrics tracking for token consumption, API costs (cloud providers), response times, error rates, provider usage distribution, and embedded runtime performance with exportable reports and visualization `M`
23. [ ] Multi-Canvas Management — Extend web UI to support monitoring multiple Canvus workspaces simultaneously with per-canvas configuration and status `L`

## Phase 5: Production Hardening

24. [ ] Structured Logging and Observability — Replace current logging with structured JSON logging (logrus/zap), add distributed tracing (OpenTelemetry), and integrate with monitoring systems (Prometheus, Grafana) including llama.cpp performance metrics `M`
25. [ ] Database Persistence Layer — Add database (PostgreSQL/SQLite) for storing processing history, audit trails, configuration backups, model download metadata, llama.cpp inference logs, and operational metrics instead of file-only logs `L`
26. [ ] Authentication and Authorization — Implement role-based access control (RBAC) for web UI, API endpoints, canvas access, and model management with user management and audit logging `L`
27. [ ] Rate Limiting and Quotas — Build token bucket rate limiting for API requests, per-user quotas, provider-specific throttling, and configurable limits to prevent abuse and control costs `M`
28. [ ] Deployment Automation — Create platform-specific installers (MSI for Windows, .deb/.rpm for Linux, .pkg for macOS) that bundle Go application + llama.cpp libraries + default GGUF model, with Kubernetes manifests, Helm charts, and comprehensive deployment documentation `XL`
29. [ ] Disaster Recovery — Implement configuration backup/restore, graceful shutdown with in-flight request handling, automatic recovery from crashes with queue persistence, and llama.cpp context state preservation `M`

> Notes
> - Order items by technical dependencies and product architecture
> - Each item should represent an end-to-end (frontend + backend) functional and testable feature
> - Phase 1 (Architecture Refactoring) is foundational and should be completed before major feature additions
> - Phase 2 (Embedded llama.cpp) is the highest priority new capability, providing zero-config deployment with native Windows support
> - Phase 2 items depend on atomic architecture from Phase 1 (especially provider abstraction)
> - Phase 2.5 (Image Generation) is future work, can be deferred until Phase 2 is stable
> - Phase 3-5 items can be parallelized once embedded LLM infrastructure is established
> - llama.cpp builds must be tested across all target platforms (Windows MSVC/MSYS, Linux amd64/ARM64, macOS Intel/ARM64)
> - CGo integration requires proper build toolchain setup (CMake, C++ compiler) on development machines
