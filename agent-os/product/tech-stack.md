# Tech Stack

## Programming Language & Runtime

**Go 1.x**
- Primary language for backend service
- Chosen for: performance, concurrency primitives (goroutines, channels), single-binary deployment, strong standard library
- Used in: all application code (main.go, handlers.go, monitorcanvus.go, core/, canvusapi/, logging/)

**CGo**
- C/Go interoperability for native library integration
- Used for: Embedding llama.cpp, stable-diffusion.cpp (future), whisper.cpp (future)
- Build requirements: CMake, C++ compiler (MSVC/GCC/Clang)

## Installation & Packaging

**NSIS (Nullsoft Scriptable Install System)** (Phase 0 - Windows)
- Windows installer creation framework
- Creates CanvusLocalLLM-Setup.exe with 4-step wizard
- Features: Custom pages, registry manipulation, file installation, uninstaller generation
- Integration: Optional Windows Service creation, PATH modification

**Debian Package Tools** (Phase 0 - Linux)
- dpkg-deb: Create .deb packages for Debian/Ubuntu
- Package structure: DEBIAN/ directory with control, postinst, prerm scripts
- Install location: /opt/canvuslocallm/
- Integration: systemd service file creation, post-install configuration messages

**Tarball Distribution** (Phase 0 - Linux)
- .tar.gz archive for non-Debian distributions
- Includes install.sh script for extraction and setup
- Manual installation to /opt/canvuslocallm/ or user-specified location
- Portable distribution method

**Configuration Templating** (Phase 0)
- .env.example template with well-commented sections
- Template structure:
  - Required Canvus configuration (server, API key, canvas ID)
  - AI model selection (llava-7b, llava-13b, llama-3.2-vision-11b)
  - Privacy modes (local-only, hybrid, cloud-preferred)
  - Optional cloud provider settings (OpenAI, Azure)
  - Performance tuning (context size, concurrency)
  - Advanced settings (logging, SSL, timeouts)
- Generated during installation, opened automatically for user editing

**Installer Wizard Flow** (Phase 0)
1. **Welcome & License Page**: Accept license agreement, choose install location
2. **Installation Page**: Copy binaries (CanvusLocalLLM.exe, llama-server.exe, libraries)
3. **Configuration Page**: Create .env.example with template
4. **Completion Page**: "Open Configuration File" button → opens .env.example in Notepad (Windows) or default editor (Linux)

**Optional Components** (Phase 0)
- Windows Service: Checkbox to install as background service with automatic startup
- PATH Addition: Checkbox to add install directory to system PATH
- systemd Service: Linux post-install option to create systemd unit file

## First-Run Setup Infrastructure

**Model Download System** (Phase 0)
- HTTP downloader with progress reporting:
  - Download speed tracking (MB/s)
  - ETA calculation
  - Progress bar (percentage, current/total size)
  - Example: `[████████████░░░░░░░░] 2.8GB/4.1GB (68%) - 5.2 MB/s - ETA 4m 12s`
- Resumable downloads: HTTP Range requests for interrupted transfers
- Multi-model support: Download based on AI_MODEL setting in .env
- Model variants:
  - llava-7b: ~4GB (faster, lower accuracy)
  - llava-13b: ~8GB (better quality, slower)
  - llama-3.2-vision-11b: ~7GB (balanced)
- Multimodal projection weights: Automatic llava-mmproj.gguf download for vision capabilities
- Storage: models/ directory in install location

**Configuration Validation** (Phase 0)
- Startup validation checks:
  - .env file existence (if not found, helpful error with instructions to copy from .env.example)
  - Required variables present: CANVUS_SERVER, CANVUS_API_KEY, CANVAS_ID
  - Variable format validation: URL format for servers, non-empty API keys
  - Model selection validation: Ensure AI_MODEL matches available/supported models
- Error messages with examples:
  ```
  ERROR: .env file not found
  → Please copy .env.example to .env and configure with your settings
  → Example: cp .env.example .env (Linux) or copy .env.example .env (Windows)
  ```
- Privacy mode validation: Ensure cloud API keys present if hybrid/cloud-preferred mode selected

**SHA256 Verification** (Phase 0)
- Model integrity checking: Verify downloaded models against known checksums
- Checksum sources: Bundled checksums.txt or fetched from release metadata
- Validation on download completion and before model loading
- Automatic re-download if verification fails

**Service Creation** (Phase 0)
- **Windows Service**:
  - Uses Windows Service API via syscall or github.com/kardianos/service
  - Service name: CanvusLocalLLM
  - Display name: Canvus Local LLM Service
  - Automatic startup, runs in background
  - Logging to Windows Event Log + app.log
  - Management via sc.exe or Services MMC snap-in
- **systemd Service**:
  - Unit file: /etc/systemd/system/canvuslocallm.service
  - Type=simple, user=canvuslocalllm (dedicated user)
  - Automatic restart on failure
  - Logging to journald + app.log
  - Management via systemctl (start, stop, status, enable)

## Embedded LLM Infrastructure (llama.cpp Ecosystem)

**llama.cpp** (Planned - Phase 2)
- C++ LLM inference engine embedded via CGo
- Single-binary deployment with bundled native libraries (.dll/.so/.dylib)
- Cross-platform: Windows (Visual Studio 2022/MSYS), Linux (amd64/ARM64), macOS (Intel/ARM64)
- GGUF model support with quantization (4-bit, 5-bit, 8-bit)
- GPU acceleration: Optional CUDA, Metal, OpenCL support
- Go bindings: go-skynet/go-llama.cpp or tcpipuk/llama-go

**Embedded Multimodal Models** (Planned - Phase 2)
- Options configured via AI_MODEL in .env:
  - llava-7b: ~4GB quantized (fast, suitable for most tasks)
  - llava-13b: ~8GB quantized (higher quality, more capable)
  - llama-3.2-vision-11b: ~7GB quantized (balanced performance/quality)
- Multimodal projection: llava-mmproj.gguf (vision capabilities)
- Capabilities: Text generation, vision/image analysis, PDF understanding
- Automatic provisioning: Downloaded on first application run (not during installation)
- Model management: Download, switch, delete GGUF models via web UI
- Storage: models/ directory in install location

**stable-diffusion.cpp** (Planned - Phase 2.5)
- C++ image generation engine from llama.cpp ecosystem
- CGo integration following same pattern as llama.cpp
- Cross-platform native builds for Windows/Linux/macOS
- Local text-to-image generation with GGUF/Safetensors model support
- Hybrid routing: Local generation (privacy) + cloud fallback (quality)

**whisper.cpp** (Future)
- C++ audio transcription engine from llama.cpp ecosystem
- For future voice/speech capabilities
- Proven Windows support, same cross-platform approach

**Provider Abstraction Layer** (Planned - Phase 2)
- Smart routing: Embedded llama.cpp (default) → External LLM servers → Cloud providers (fallback)
- Configuration: Provider priority via PRIVACY_MODE (.env), capability-based routing (vision vs text), fallback rules
- Health-aware: Automatic failover when embedded runtime unavailable or inference fails
- Atoms: Health check functions, provider detection, model loading
- Molecules: Routing decision logic, fallback strategies, context management
- Organism: Unified provider interface with dependency injection

**Runtime Lifecycle Management** (Planned - Phase 2)
- Embedded integration: llama.cpp loaded as shared library within Go process
- No separate service: All inference in-process via CGo calls
- Context management: Thread-safe llama context pool for concurrent requests
- Memory management: Automatic model loading/unloading based on usage patterns
- Health monitoring: Inference failure detection, memory usage tracking, automatic recovery
- Graceful degradation: Fall back to OpenAI/Azure when embedded runtime unavailable

## AI & Machine Learning Services

**OpenAI API**
- GPT models: gpt-3.5-turbo (notes), gpt-4 (PDF/canvas analysis)
- DALL-E: dall-e-3 and dall-e-2 for image generation
- Integration: github.com/sashabaranov/go-openai client library
- Configuration: BASE_LLM_URL, TEXT_LLM_URL, model selection per operation type
- Role: Fallback provider when embedded LLM unavailable, premium models for complex tasks

**Azure OpenAI**
- Alternative deployment for OpenAI models
- Enterprise-focused with private endpoints and compliance features
- Configuration: AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_DEPLOYMENT, AZURE_OPENAI_API_VERSION
- Role: Enterprise cloud fallback, compliance-focused deployments

**Local LLM Servers** (Optional External Providers)
- Supported: External Ollama, LLaMA, LM Studio
- OpenAI-compatible API endpoints
- Common endpoints: http://localhost:8000/v1 (LLaMA), http://localhost:11434/v1 (Ollama), http://localhost:1234/v1 (LM Studio)
- Enables: Custom model selection, advanced configurations, separate infrastructure
- Role: Power user customization, specialized models beyond embedded runtime

**Google Vision API**
- Handwriting recognition and OCR
- Integration: google.golang.org/api/vision/v1
- Configuration: GOOGLE_VISION_API_KEY
- Role: Specialized OCR when embedded vision models insufficient

## Canvas Integration

**Canvus REST API**
- Real-time workspace monitoring via streaming endpoint (subscribe=true)
- Widget CRUD operations (notes, images, PDFs)
- File upload (multipart/form-data)
- Coordinate system: relative to parent widget
- Authentication: API key via X-Api-Key header

## Core Dependencies

**HTTP Client & Networking**
- net/http (standard library): HTTP client with configurable TLS
- TLS Configuration: Optional self-signed certificate support (ALLOW_SELF_SIGNED_CERTS)
- Timeouts: Configurable per operation (AI_TIMEOUT, PROCESSING_TIMEOUT)
- Model downloads: HTTP client with Range request support for resumable downloads

**PDF Processing**
- github.com/ledongthuc/pdf: PDF text extraction
- Custom chunking logic: Configurable token-based chunking (PDF_CHUNK_SIZE_TOKENS, PDF_MAX_CHUNKS_TOKENS)

**Concurrency & Synchronization**
- context.Context: Cancellation and timeout propagation
- sync.RWMutex: Thread-safe widget state management
- sync.Mutex: Shared resource protection (logs, downloads, metrics, llama context pool)
- Goroutines: Asynchronous processing with lifecycle management
- Channels: Inter-goroutine communication
- Semaphore pattern: Rate limiting (MAX_CONCURRENT)

**Configuration Management**
- godotenv: .env file parsing
- Environment variables: All configuration via env vars
- Validation: Required variables checked at startup (LoadConfig)
- Template: .env.example with comprehensive documentation
- Privacy modes: local-only, hybrid, cloud-preferred

**Logging**
- File logging: app.log with timestamps and source information
- Console logging: Color-coded output via github.com/fatih/color
- Dual output: Simultaneous file and console via logging.LogHandler()
- Format: Timestamp, log level, source location, message
- Planned: llama.cpp inference metrics, memory usage, context stats

**Service Integration** (Phase 0)
- Windows: github.com/kardianos/service or syscall for Windows Service API
- Linux: systemd unit file generation, no external Go dependencies

## Architecture Patterns

**Atomic Design**
- Atoms: Pure functions (env parsers, HTTP client factory, llama.cpp context loading, model validation, checksum verification)
- Molecules: Simple compositions (Config struct, canvusapi.Client, provider routing logic, inference request builders, download progress tracker)
- Organisms: Feature modules (Monitor, handlers, API client, llamaruntime package, sdruntime package, installer wizard)
- Pages: Composition root (main.go with llama.cpp lifecycle management, installer main)

**CGo Integration Pattern**
- C++ library compilation: CMake-based build of llama.cpp/stable-diffusion.cpp for target platform
- Shared library bundling: .dll (Windows), .so (Linux), .dylib (macOS) packaged with Go binary
- Go bindings: Wrapper package (llamaruntime/) providing Go-friendly API over C functions
- Memory safety: Careful handling of pointers, manual memory management, defer cleanup
- Thread safety: Context pool pattern for concurrent inference requests
- Error handling: Translate C errors to Go error types with proper context

**Dependency Injection**
- Current: Config passed to constructors (NewClient, NewMonitor)
- Target: Eliminate global config variable in handlers.go (refactoring priority)
- Planned: Provider interface injected into handlers for testing and flexibility
- llama.cpp integration: Inject model context pool into inference handlers

**Provider Pattern** (Planned - Phase 2)
- Interface: Unified LLM provider interface (GenerateText, AnalyzeImage, GetHealth)
- Implementations: LlamaProvider, OpenAIProvider, AzureProvider, ExternalLLMProvider
- Factory: Create providers based on PRIVACY_MODE and configuration with dependency injection
- Benefits: Testability, easy provider swapping, graceful degradation, mock providers for testing

**Error Handling**
- Sentinel errors: ErrInvalidInput, ErrProviderUnavailable, ErrModelLoadFailed, ErrConfigMissing (planned)
- Error wrapping: fmt.Errorf with %w for context
- Custom types: APIError with status codes, ProviderError with fallback hints, LlamaError for CGo errors
- Retry logic: Exponential backoff (MaxRetries, RetryDelay)
- Provider fallback: Automatic retry with different provider on failure
- CGo error handling: Translate C errors, cleanup on failure, context recovery
- User-friendly errors: Helpful messages with examples for configuration issues

**Concurrency Model**
- Context-based cancellation: Propagated through call chains including CGo inference
- Signal handling: Graceful shutdown on SIGINT/SIGTERM (includes llama.cpp context cleanup)
- Thread-safe state: RWMutex for widget cache, Mutex for llama context pool
- Lifecycle management: Goroutine cleanup via defer and context
- CGo safety: Proper locking for C library calls, context pool to avoid race conditions

## Build & Deployment

**Build Tools**
- Go toolchain: go build, go test, go mod
- CMake: Build llama.cpp, stable-diffusion.cpp native libraries
- C++ compiler: MSVC (Windows), GCC (Linux), Clang (macOS)
- NSIS: Windows installer compilation (makensis.exe)
- dpkg-deb: Debian package creation
- tar/gzip: Tarball creation for portable Linux distribution
- Cross-compilation: GOOS/GOARCH for Go + platform-specific CMake builds
- Output: Installer packages (.exe, .deb, .tar.gz) containing Go binary + native libraries + example config

**Build Process** (Planned - Phase 2)
1. Clone llama.cpp repository (or use vendored version)
2. Run CMake build for target platform (Windows/Linux/macOS, amd64/ARM64)
3. Compile llama.cpp with appropriate flags (CUDA/Metal/CPU-only)
4. Place compiled libraries in lib/ directory
5. Build Go application with CGo enabled (CGO_ENABLED=1)
6. Link Go binary against llama.cpp shared libraries
7. Create installer packages with bundled binaries and libraries

**Installer Build Process** (Phase 0)
**Windows (NSIS):**
1. Compile Go application (CanvusLocalLLM.exe)
2. Create NSIS script (.nsi) with wizard pages, file installation, registry entries
3. Bundle binaries, libraries, .env.example template
4. Compile installer: `makensis installer.nsi` → CanvusLocalLLM-Setup.exe
5. Sign executable (optional, for release builds)

**Linux (Debian):**
1. Compile Go application (canvuslocallm)
2. Create package directory structure: opt/canvuslocallm/, DEBIAN/
3. Copy binaries, libraries, .env.example to package directory
4. Create control file with package metadata
5. Create postinst script to generate .env.example, show configuration message
6. Build package: `dpkg-deb --build package/ canvuslocallm_1.0.0_amd64.deb`

**Linux (Tarball):**
1. Compile Go application
2. Create archive structure with install.sh script
3. Package: `tar czf canvuslocallm-1.0.0-linux-amd64.tar.gz canvuslocallm/`

**Target Platforms**
- Windows (amd64): MSVC or MSYS2 build
- Linux (amd64, ARM64): GCC build
- macOS (amd64 Intel, ARM64 Apple Silicon): Clang with Metal support

**Deployment Model** (Phase 0 + Phase 2)
- **Installation Method**: Native installers (CanvusLocalLLM-Setup.exe, .deb, .tar.gz)
- **Install Locations**:
  - Windows: C:\Program Files\CanvusLocalLLM\
  - Linux: /opt/canvuslocallm/
- **Installed Components**:
  - Application binary (CanvusLocalLLM.exe or canvuslocallm)
  - Native libraries (llama.dll/libllama.so/libllama.dylib in lib/ subdirectory)
  - Configuration template (.env.example)
  - Documentation (README.txt, LICENSE.txt)
- **Models**: Downloaded on first run to models/ directory (~4-8GB depending on selection)
- **Configuration**: .env file (user creates from .env.example)
- **Logging**: app.log in install directory
- **Downloads**: downloads/ directory for temporary files

**Directory Structure (Post-Installation)**:
```
C:\Program Files\CanvusLocalLLM\  (or /opt/canvuslocallm/)
├── CanvusLocalLLM.exe            (main application)
├── lib/
│   └── llama.dll                 (native libraries)
├── .env.example                  (configuration template)
├── .env                          (user-created config)
├── models/                       (created on first run)
│   ├── llava-7b-q4.gguf         (downloaded model)
│   └── llava-mmproj.gguf        (multimodal projection)
├── downloads/                    (temporary files)
├── app.log                       (application log)
├── README.txt
└── LICENSE.txt
```

**First-Run Experience** (Phase 0 + Phase 2)
When user runs CanvusLocalLLM.exe for the first time:

1. Check for .env file
   - If not found: Display error with instructions to copy from .env.example
   - Example: `ERROR: .env file not found. Please copy .env.example to .env and configure.`
2. Validate configuration
   - Check required values: CANVUS_SERVER, CANVUS_API_KEY, CANVAS_ID
   - Check model selection: AI_MODEL is valid (llava-7b, llava-13b, llama-3.2-vision-11b)
   - Check privacy mode: PRIVACY_MODE is valid and has required cloud keys if hybrid/cloud-preferred
3. Download selected model
   - Display progress: `📦 Downloading LLaVA 7B model... [████████████░░░░░░░░] 2.8GB/4.1GB (68%) - 5.2 MB/s - ETA 4m 12s`
   - Download multimodal projection weights if vision model
   - Resumable downloads if interrupted
4. Verify model integrity
   - SHA256 checksum validation
   - Re-download if verification fails
5. Initialize llama.cpp runtime
   - Load model into llama context
   - Allocate memory pools
6. Test inference
   - Quick "Hello World" prompt to verify functionality
   - Display success or error message
7. Begin monitoring
   - Connect to Canvus server
   - Start canvas monitoring loop

**CI/CD & Release Automation** (Phase 5)
- Automated builds for all platforms (Windows/Linux/macOS, amd64/ARM64)
- Installer generation in CI pipeline
- Binary signing (Windows Authenticode, macOS codesign)
- Checksum generation (SHA256) for all release artifacts
- Upload to GitHub Releases with auto-generated release notes
- Version coordination: llama.cpp version + Go application version

## Security

**TLS/SSL**
- Certificate validation: Enabled by default
- Development mode: Optional self-signed cert support (not recommended for production)
- Warning logging: When SSL validation disabled

**API Key Management**
- Environment variables: OPENAI_API_KEY (optional with embedded LLM), CANVUS_API_KEY, GOOGLE_VISION_API_KEY
- File security: .env excluded from version control (.gitignore)
- Optional keys: OpenAI/Azure keys only required for cloud fallback or premium features (hybrid/cloud-preferred modes)
- Installer security: .env.example does not contain actual keys, only placeholders and instructions

**Authentication**
- Web UI: Password protection (WEBUI_PWD)
- API: Canvus API key authentication
- Model management: Web UI authentication required for model downloads/deletions

**Data Privacy**
- Default: All AI processing on-premises via embedded llama.cpp (zero cloud data transmission)
- Privacy modes enforce data handling:
  - local-only: Never send data to cloud providers
  - hybrid: Local first, cloud only on failure
  - cloud-preferred: Use cloud when faster, but user explicitly opted in
- Optional cloud: User explicitly configures OpenAI/Azure for specific use cases
- Audit: Track which provider handled each request for compliance
- No telemetry: Embedded inference fully offline, no phone-home

**CGo Security Considerations**
- Memory safety: Careful pointer handling, bounds checking for C data
- Input validation: Sanitize prompts before passing to C layer
- Resource limits: Max context size, memory limits, inference timeouts
- Crash isolation: Recover from CGo panics, prevent C crashes from killing Go process

**Installer Security**
- Code signing: Windows executables signed with Authenticode certificate (release builds)
- Checksum verification: SHA256 checksums published with releases
- HTTPS downloads: Models downloaded over HTTPS with cert validation
- No bundled secrets: .env.example contains no real API keys or credentials

## Testing

**Test Framework**
- testing package (standard library): Unit and integration tests
- Table-driven tests: Multiple scenarios per function
- Subtests: t.Run() for organization
- CGo testing: Mock C functions, test memory safety, validate bindings
- Installer testing: Virtual machines for clean installation testing

**Test Organization**
- tests/ directory: All test files
- Fixtures: tests/test_data.go for shared test data, test GGUF models
- Coverage: Canvas API, LLM integration, comprehensive API endpoints, configuration validation
- Planned: Mock llama.cpp provider, CGo binding tests, memory leak tests, installer integration tests

**Testing Strategy** (Planned - Phase 0 + Phase 2)
- **Unit**: Provider interface implementations, routing logic, health checks, CGo wrapper functions, configuration validation
- **Integration**: Embedded llama.cpp lifecycle, model loading, inference requests, context management, installer execution
- **End-to-end**: Multi-provider fallback scenarios, installation process on all platforms, first-run model download
- **Performance**: Inference throughput, memory usage, context switching latency, embedded vs cloud latency
- **Memory safety**: Leak detection, pointer validation, concurrent access tests
- **Platform testing**:
  - Windows: MSVC/MSYS builds, installer on Windows 10/11, Service creation
  - Linux: amd64/ARM64, .deb installation on Ubuntu/Debian, .tar.gz extraction, systemd service
  - macOS: Intel/ARM builds, installer testing
- **Installation testing**:
  - Clean install on fresh systems (VMs)
  - Upgrade from previous version
  - Uninstall completeness (no orphaned files/registry entries)
  - Configuration file opening (correct editor, proper permissions)
  - First-run model download (progress, resume, verification)

## Future Tech Stack Considerations

**Planned Additions** (from roadmap):
- llama.cpp Go bindings: go-skynet/go-llama.cpp or tcpipuk/llama-go
- stable-diffusion.cpp: CGo integration for local image generation
- whisper.cpp: Future audio transcription capabilities
- Structured logging: logrus or zap with provider/request tracing, llama.cpp inference metrics
- Observability: OpenTelemetry for distributed tracing across providers
- Monitoring: Prometheus metrics (provider usage, latency, costs, llama.cpp memory/context stats), Grafana dashboards
- Database: PostgreSQL or SQLite for persistence (processing history, model metadata, inference logs)
- Additional AI providers: Potential support for Claude, Gemini (via provider abstraction)

**Refactoring Targets**:
- Package extraction: pdfprocessor/, imagegen/, canvasanalyzer/, handwritingrecog/, llamaruntime/, sdruntime/
- Dependency injection: Remove global config variable, inject provider interface
- Provider abstraction: Unified interface for embedded llama.cpp + cloud providers
- Atomic architecture: Further decomposition of large files (handlers.go ~2000 lines)
- CGo safety: Comprehensive memory management, error handling patterns

**Model Format Considerations**:
- Primary: GGUF (llama.cpp native, compressed, quantized)
- Quantization: 4-bit (Q4_K_M), 5-bit (Q5_K_M), 8-bit (Q8_0) models for size/performance tradeoffs
- Vision models: Separate projection weights (llava-mmproj.gguf) for multimodal capabilities
- Download: Progressive download with resume support for large models, checksum validation
- Storage: models/ directory, efficient file organization, automatic cleanup of old models
- Conversion: Potential support for converting PyTorch/Safetensors to GGUF

**Build Infrastructure**:
- CI/CD: Automated cross-platform builds (Windows/Linux/macOS, amd64/ARM64)
- CMake automation: Scripted llama.cpp builds with proper flags per platform
- Installer automation: NSIS, dpkg-deb, tarball creation in CI pipeline
- Release artifacts: Platform-specific installers with bundled libraries
- Version management: Coordinate llama.cpp version with Go application version
- Binary signing: Authenticode (Windows), codesign (macOS)
- Checksum generation: SHA256 for all release artifacts
