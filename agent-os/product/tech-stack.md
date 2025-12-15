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

## Embedded LLM Infrastructure (llama.cpp Ecosystem)

**llama.cpp** (Planned - Phase 2)
- C++ LLM inference engine embedded via CGo
- Single-binary deployment with bundled native libraries (.dll/.so/.dylib)
- Cross-platform: Windows (Visual Studio 2022/MSYS), Linux (amd64/ARM64), macOS (Intel/ARM64)
- GGUF model support with quantization (4-bit, 5-bit, 8-bit)
- GPU acceleration: Optional CUDA, Metal, OpenCL support
- Go bindings: go-skynet/go-llama.cpp or tcpipuk/llama-go

**Embedded Multimodal Models** (Planned - Phase 2)
- Default: LLaVA 7B GGUF (~4GB quantized)
- Multimodal projection: llava-mmproj.gguf (vision capabilities)
- Capabilities: Text generation, vision/image analysis, PDF understanding
- Automatic provisioning: Downloaded on first run with progress tracking
- Model management: Download, switch, delete GGUF models via web UI
- Storage: models/ directory in application folder

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
- Configuration: Provider priority, capability-based routing (vision vs text), fallback rules
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
- Planned: Default to embedded llama.cpp when no external provider configured

**Logging**
- File logging: app.log with timestamps and source information
- Console logging: Color-coded output via github.com/fatih/color
- Dual output: Simultaneous file and console via logging.LogHandler()
- Format: Timestamp, log level, source location, message
- Planned: llama.cpp inference metrics, memory usage, context stats

## Architecture Patterns

**Atomic Design**
- Atoms: Pure functions (env parsers, HTTP client factory, llama.cpp context loading, model validation)
- Molecules: Simple compositions (Config struct, canvusapi.Client, provider routing logic, inference request builders)
- Organisms: Feature modules (Monitor, handlers, API client, llamaruntime package, sdruntime package)
- Pages: Composition root (main.go with llama.cpp lifecycle management)

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
- Factory: Create providers based on configuration with dependency injection
- Benefits: Testability, easy provider swapping, graceful degradation, mock providers for testing

**Error Handling**
- Sentinel errors: ErrInvalidInput, ErrProviderUnavailable, ErrModelLoadFailed (planned)
- Error wrapping: fmt.Errorf with %w for context
- Custom types: APIError with status codes, ProviderError with fallback hints, LlamaError for CGo errors
- Retry logic: Exponential backoff (MaxRetries, RetryDelay)
- Provider fallback: Automatic retry with different provider on failure
- CGo error handling: Translate C errors, cleanup on failure, context recovery

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
- Cross-compilation: GOOS/GOARCH for Go + platform-specific CMake builds
- Output: Single Go binary with embedded CGo linkage to bundled .dll/.so/.dylib

**Build Process** (Planned - Phase 2)
1. Clone llama.cpp repository (or use vendored version)
2. Run CMake build for target platform (Windows/Linux/macOS, amd64/ARM64)
3. Compile llama.cpp with appropriate flags (CUDA/Metal/CPU-only)
4. Place compiled libraries in lib/ directory
5. Build Go application with CGo enabled (CGO_ENABLED=1)
6. Link Go binary against llama.cpp shared libraries
7. Bundle native libraries with application binary for distribution

**Target Platforms**
- Windows (amd64): MSVC or MSYS2 build
- Linux (amd64, ARM64): GCC build
- macOS (amd64 Intel, ARM64 Apple Silicon): Clang with Metal support

**Deployment Model**
- Current: Standalone binary, .env configuration
- Planned (Phase 2): Single-executable deployment with embedded llama.cpp
- Bundled artifacts:
  - Application binary (Go executable)
  - Native libraries (llama.dll/libllama.so/libllama.dylib in lib/ subdirectory)
  - Default GGUF model (~4GB, downloaded on first run or bundled with installer)
- Configuration: .env file with smart defaults (embedded llama.cpp if available, cloud fallback)
- Logging: app.log in working directory
- Downloads: downloads/ directory for temporary files
- Models: models/ directory for GGUF files

**Installation Process** (Planned - Phase 2)
1. Platform detection (OS, architecture)
2. Extract application binary + bundled native libraries
3. Verify native library compatibility (check CUDA/Metal availability)
4. Download default LLaVA 7B GGUF model on first run (resumable download)
5. Download multimodal projection weights (llava-mmproj.gguf)
6. Validate models (checksum verification)
7. Initialize llama.cpp context with model
8. Generate .env with embedded runtime as default provider
9. Test inference (simple prompt) to verify installation
10. Start application monitoring

**Platform-Specific Installers** (Planned - Phase 5)
- Windows: MSI installer bundling exe + DLLs + default model
- Linux: .deb/.rpm packages with proper library paths
- macOS: .pkg installer with code signing for Gatekeeper

## Security

**TLS/SSL**
- Certificate validation: Enabled by default
- Development mode: Optional self-signed cert support (not recommended for production)
- Warning logging: When SSL validation disabled

**API Key Management**
- Environment variables: OPENAI_API_KEY (optional with embedded LLM), CANVUS_API_KEY, GOOGLE_VISION_API_KEY
- File security: .env excluded from version control (.gitignore)
- Optional keys: OpenAI/Azure keys only required for cloud fallback or premium features

**Authentication**
- Web UI: Password protection (WEBUI_PWD)
- API: Canvus API key authentication
- Model management: Web UI authentication required for model downloads/deletions

**Data Privacy**
- Default: All AI processing on-premises via embedded llama.cpp (zero cloud data transmission)
- Optional cloud: User explicitly configures OpenAI/Azure for specific use cases
- Audit: Track which provider handled each request for compliance
- No telemetry: Embedded inference fully offline, no phone-home

**CGo Security Considerations**
- Memory safety: Careful pointer handling, bounds checking for C data
- Input validation: Sanitize prompts before passing to C layer
- Resource limits: Max context size, memory limits, inference timeouts
- Crash isolation: Recover from CGo panics, prevent C crashes from killing Go process

## Testing

**Test Framework**
- testing package (standard library): Unit and integration tests
- Table-driven tests: Multiple scenarios per function
- Subtests: t.Run() for organization
- CGo testing: Mock C functions, test memory safety, validate bindings

**Test Organization**
- tests/ directory: All test files
- Fixtures: tests/test_data.go for shared test data, test GGUF models
- Coverage: Canvas API, LLM integration, comprehensive API endpoints
- Planned: Mock llama.cpp provider, CGo binding tests, memory leak tests

**Testing Strategy** (Planned - Phase 2)
- Unit: Provider interface implementations, routing logic, health checks, CGo wrapper functions
- Integration: Embedded llama.cpp lifecycle, model loading, inference requests, context management
- End-to-end: Multi-provider fallback scenarios, installation process on all platforms
- Performance: Inference throughput, memory usage, context switching latency, embedded vs cloud latency
- Memory safety: Leak detection, pointer validation, concurrent access tests
- Platform testing: Windows (MSVC/MSYS), Linux (amd64/ARM64), macOS (Intel/ARM)

## Future Tech Stack Considerations

**Planned Additions** (from roadmap):
- llama.cpp Go bindings: go-skynet/go-llama.cpp or tcpipuk/llama-go
- stable-diffusion.cpp: CGo integration for local image generation
- whisper.cpp: Future audio transcription capabilities
- Structured logging: logrus or zap with provider/request tracing, llama.cpp inference metrics
- Observability: OpenTelemetry for distributed tracing across providers
- Monitoring: Prometheus metrics (provider usage, latency, costs, llama.cpp memory/context stats), Grafana dashboards
- Database: PostgreSQL or SQLite for persistence (processing history, model metadata, inference logs)
- Platform installers: MSI (Windows), .deb/.rpm (Linux), .pkg (macOS) with bundled models
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
- Release artifacts: Platform-specific packages with bundled libraries + models
- Version management: Coordinate llama.cpp version with Go application version
