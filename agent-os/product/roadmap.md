# Product Roadmap

## Phase 1: Zero-Config Installation Infrastructure

1. [ ] NSIS Installer for Windows — Build CanvusLocalLLM-Setup.exe using NSIS with simple wizard (license/location, install binaries with bundled model, create minimal .env template), optional Windows Service creation checkbox `M`
2. [ ] Debian Package for Linux — Create .deb package with proper directory structure (/opt/canvuslocallm/), bundled model files, post-install script to create .env template with Canvus credential placeholders `S`
3. [ ] Tarball Distribution for Linux — Build .tar.gz archive with install.sh script that extracts to /opt/canvuslocallm/, includes bundled model, and displays simple configuration instructions `S`
4. [ ] Minimal Configuration Template — Create .env.example with only Canvus credentials: CANVUS_SERVER, CANVUS_API_KEY (or CANVUS_USERNAME/CANVUS_PASSWORD), and CANVAS_ID - no model selection, no provider options `XS`
5. [ ] First-Run Model Download — Implement Bunny v1.1 Llama-3-8B-V model download on first run (if not bundled) with progress bar, resumable HTTP downloads, SHA256 verification, and clear error messages `M`
6. [ ] Configuration Validation — Build startup validation that checks .env exists, validates Canvus credentials format, tests Canvus server connectivity, and provides helpful error messages `S`
7. [ ] Optional Service Creation — Implement Windows Service installer and systemd unit file generation for Linux background service with automatic startup `M`

## Phase 2: Embedded LLM Integration (llama.cpp + Bunny)

8. [ ] llama.cpp CUDA Integration — Create `llamaruntime/` package with Go CGo bindings to llama.cpp, specifically configured for CUDA acceleration on NVIDIA RTX GPUs `L`
9. [ ] Cross-Platform llama.cpp Build — Integrate llama.cpp CMake build with CUDA support for Windows (Visual Studio 2022) and Linux (GCC + CUDA toolkit), bundling native libraries with installer `L`
10. [ ] Bunny v1.1 Model Integration — Configure llama.cpp runtime specifically for Bunny v1.1 Llama-3-8B-V model with optimized context size, batch settings, and CUDA memory allocation for RTX GPUs `M`
11. [ ] Multimodal Vision Pipeline — Implement image analysis using Bunny's vision capabilities, accepting images from Canvus canvas and returning text descriptions/analysis `M`
12. [ ] Runtime Health Monitoring — Add health checks for llama.cpp inference context, GPU memory usage monitoring, automatic recovery from inference failures, and startup diagnostics `S`

## Phase 3: Image Generation (stable-diffusion.cpp)

13. [ ] stable-diffusion.cpp Integration — Create `sdruntime/` package with Go CGo bindings to stable-diffusion.cpp for local image generation with CUDA acceleration `L`
14. [ ] Local Image Generation Pipeline — Implement text-to-image generation using embedded stable-diffusion.cpp with automatic placement of generated images on canvas `M`
15. [ ] Cross-Platform SD Build — Integrate stable-diffusion.cpp CMake build with CUDA support for Windows and Linux, bundling with installer `M`

## Phase 4: Codebase Architecture Refactoring

16. [ ] Extract PDF Processing Package — Create `pdfprocessor/` package by extracting PDF analysis functions from handlers.go into a cohesive module with text extraction and chunking `M`
17. [ ] Extract Image Generation Package — Create `imagegen/` package for stable-diffusion.cpp integration, cleanly separated from main application `S`
18. [ ] Extract Canvas Analysis Package — Create `canvasanalyzer/` package for canvas analysis and synthesis logic `S`
19. [ ] Eliminate Global Config Variable — Refactor handlers.go to accept config via dependency injection through Monitor struct `S`
20. [ ] Split Handler Logic — Break down handlers.go into atomic functions organized by responsibility `M`

## Phase 5: Web UI Enhancement

21. [ ] Real-time Processing Dashboard — Build web interface showing canvas monitoring status, processing queue, success/failure metrics, recent AI operations, and GPU/memory usage with live updates `L`
22. [ ] Status and Metrics Display — Show inference statistics, GPU utilization, memory usage, and processing history without exposing configuration options `M`
23. [ ] Multi-Canvas Management — Extend web UI to support monitoring multiple Canvus workspaces simultaneously with per-canvas status `L`

## Phase 6: Production Hardening

24. [ ] Structured Logging — Replace current logging with structured JSON logging (logrus/zap) including llama.cpp performance metrics and GPU stats `M`
25. [ ] Database Persistence — Add SQLite for storing processing history, inference logs, and operational metrics `L`
26. [ ] Web UI Authentication — Implement password protection for web UI dashboard `S`
27. [ ] Installer Distribution Automation — Automate building of platform-specific installers with CI/CD pipeline that bundles Go application + llama.cpp + stable-diffusion.cpp libraries + Bunny model, signs binaries, and uploads to releases `L`
28. [ ] Graceful Shutdown — Implement graceful shutdown with in-flight request handling, automatic recovery from crashes, and llama.cpp context cleanup `M`

> Notes
> - Order items by technical dependencies and product architecture
> - Each item represents a functional and testable feature
> - Phase 1 (Installation) establishes zero-config deployment - must complete first
> - Phase 2 (llama.cpp + Bunny) is the core AI functionality
> - Phase 3 (Image Generation) adds stable-diffusion.cpp capabilities
> - Phase 4 (Refactoring) cleans up codebase for maintainability
> - Phases 5-6 can be parallelized once core AI is stable
> - All builds target NVIDIA RTX GPUs with CUDA acceleration
> - No cloud providers, no provider abstraction, no fallback mechanisms - pure local
> - Model is Bunny v1.1 Llama-3-8B-V - no model selection UI needed
