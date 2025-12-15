# Tech Stack

## Programming Language & Runtime

**Go 1.x**
- Primary language for backend service
- Chosen for: performance, concurrency primitives (goroutines, channels), single-binary deployment, strong standard library
- Used in: all application code (main.go, handlers.go, monitorcanvus.go, core/, canvusapi/, logging/)

## Bundled LLM Infrastructure

**Ollama Runtime** (Planned - Phase 2)
- Local LLM inference engine bundled with application
- Single binary, cross-platform (Windows, Linux, macOS)
- Built on llama.cpp for efficient CPU/GPU inference
- OpenAI-compatible API endpoint (http://localhost:11434/v1)
- No Docker required, standalone installation
- Integration: Ollama Go client library (github.com/ollama/ollama/api)

**Bundled Multimodal Models** (Planned - Phase 2)
- Default: LLaVA 7B or Llama 3.2-Vision 11B
- Model format: GGUF (compressed, quantized)
- Size: ~4-7GB quantized models
- Capabilities: Text generation, vision/image analysis, PDF understanding
- Automatic provisioning: Downloaded on first run via installation script
- Model management: Download, switch, delete models via web UI

**Provider Abstraction Layer** (Planned - Phase 2)
- Smart routing: Bundled Ollama (default) → External LLM servers → Cloud providers (fallback)
- Configuration: Provider priority, capability-based routing (vision vs text), fallback rules
- Health-aware: Automatic failover when bundled runtime unavailable
- Atoms: Health check functions, provider detection
- Molecules: Routing decision logic, fallback strategies
- Organism: Unified provider interface with dependency injection

**Installation & Lifecycle Management** (Planned - Phase 2)
- Platform-specific installers: Download Ollama binary for target platform during setup
- Architecture detection: Auto-detect amd64/ARM64 on Linux/macOS, amd64 on Windows
- Process management: Start/stop Ollama server as child process or system service
- Health monitoring: Periodic health checks, automatic restart on failure
- Graceful degradation: Fall back to OpenAI/Azure when local runtime unavailable

## AI & Machine Learning Services

**OpenAI API**
- GPT models: gpt-3.5-turbo (notes), gpt-4 (PDF/canvas analysis)
- DALL-E: dall-e-3 and dall-e-2 for image generation
- Integration: github.com/sashabaranov/go-openai client library
- Configuration: BASE_LLM_URL, TEXT_LLM_URL, model selection per operation type
- Role: Fallback provider when bundled LLM unavailable, premium models for complex tasks

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
- Role: Power user customization, specialized models

**Google Vision API**
- Handwriting recognition and OCR
- Integration: google.golang.org/api/vision/v1
- Configuration: GOOGLE_VISION_API_KEY
- Role: Specialized OCR when bundled vision models insufficient

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
- sync.Mutex: Shared resource protection (logs, downloads, metrics)
- Goroutines: Asynchronous processing with lifecycle management
- Channels: Inter-goroutine communication
- Semaphore pattern: Rate limiting (MAX_CONCURRENT)

**Configuration Management**
- godotenv: .env file parsing
- Environment variables: All configuration via env vars
- Validation: Required variables checked at startup (LoadConfig)
- Planned: Default to bundled Ollama when no external provider configured

**Logging**
- File logging: app.log with timestamps and source information
- Console logging: Color-coded output via github.com/fatih/color
- Dual output: Simultaneous file and console via logging.LogHandler()
- Format: Timestamp, log level, source location, message

## Architecture Patterns

**Atomic Design**
- Atoms: Pure functions (env parsers, HTTP client factory, Ollama health checks)
- Molecules: Simple compositions (Config struct, canvusapi.Client, provider routing logic)
- Organisms: Feature modules (Monitor, handlers, API client, ollamaruntime package)
- Pages: Composition root (main.go with Ollama lifecycle management)

**Dependency Injection**
- Current: Config passed to constructors (NewClient, NewMonitor)
- Target: Eliminate global config variable in handlers.go (refactoring priority)
- Planned: Provider interface injected into handlers for testing and flexibility

**Provider Pattern** (Planned - Phase 2)
- Interface: Unified LLM provider interface (GenerateText, AnalyzeImage, GetHealth)
- Implementations: OllamaProvider, OpenAIProvider, AzureProvider
- Factory: Create providers based on configuration with dependency injection
- Benefits: Testability, easy provider swapping, graceful degradation

**Error Handling**
- Sentinel errors: ErrInvalidInput, ErrProviderUnavailable (planned)
- Error wrapping: fmt.Errorf with %w for context
- Custom types: APIError with status codes, ProviderError with fallback hints
- Retry logic: Exponential backoff (MaxRetries, RetryDelay)
- Provider fallback: Automatic retry with different provider on failure

**Concurrency Model**
- Context-based cancellation: Propagated through call chains
- Signal handling: Graceful shutdown on SIGINT/SIGTERM (includes Ollama process cleanup)
- Thread-safe state: RWMutex for widget cache
- Lifecycle management: Goroutine cleanup via defer and context
- Process management: Ollama subprocess lifecycle tied to main process

## Build & Deployment

**Build Tools**
- Go toolchain: go build, go test, go mod
- Cross-compilation: GOOS/GOARCH for Linux, macOS (Intel/ARM), Windows
- Output: Single statically-linked binary (application) + Ollama binary (bundled runtime)

**Target Platforms**
- Linux (amd64, ARM64)
- macOS (amd64 Intel, ARM64 Apple Silicon)
- Windows (amd64)

**Deployment Model**
- Current: Standalone binary, .env configuration
- Planned (Phase 2): Installation script downloads both application and Ollama, provisions default model
- Bundled artifacts: Application binary + Ollama binary + default GGUF model (~4-7GB)
- Configuration: .env file with smart defaults (bundled Ollama if available, cloud fallback)
- Logging: app.log in working directory
- Downloads: downloads/ directory for temporary files
- Models: ~/.ollama/models (Ollama default) or custom models directory

**Installation Process** (Planned - Phase 2)
1. Platform detection (OS, architecture)
2. Download application binary
3. Download Ollama binary for target platform
4. Install Ollama to application directory or system location
5. Start Ollama server (background process or service)
6. Pull default multimodal model (LLaVA 7B or Llama 3.2-Vision)
7. Generate .env with bundled Ollama as default provider
8. Verify installation: health check, test inference
9. Start application monitoring

## Security

**TLS/SSL**
- Certificate validation: Enabled by default
- Development mode: Optional self-signed cert support (not recommended for production)
- Warning logging: When SSL validation disabled

**API Key Management**
- Environment variables: OPENAI_API_KEY (optional with bundled LLM), CANVUS_API_KEY, GOOGLE_VISION_API_KEY
- File security: .env excluded from version control (.gitignore)
- Optional keys: OpenAI/Azure keys only required for cloud fallback or premium features

**Authentication**
- Web UI: Password protection (WEBUI_PWD)
- API: Canvus API key authentication
- Model management: Web UI authentication required for model downloads/deletions

**Data Privacy**
- Default: All AI processing on-premises via bundled Ollama (zero cloud data transmission)
- Optional cloud: User explicitly configures OpenAI/Azure for specific use cases
- Audit: Track which provider handled each request for compliance

## Testing

**Test Framework**
- testing package (standard library): Unit and integration tests
- Table-driven tests: Multiple scenarios per function
- Subtests: t.Run() for organization

**Test Organization**
- tests/ directory: All test files
- Fixtures: tests/test_data.go for shared test data
- Coverage: Canvas API, LLM integration, comprehensive API endpoints
- Planned: Mock Ollama server, provider abstraction tests, failover scenarios

**Testing Strategy** (Planned - Phase 2)
- Unit: Provider interface implementations, routing logic, health checks
- Integration: Bundled Ollama lifecycle, model provisioning, request handling
- End-to-end: Multi-provider fallback scenarios, installation scripts on all platforms
- Performance: Token throughput, response latency for bundled vs cloud providers

## Future Tech Stack Considerations

**Planned Additions** (from roadmap):
- Ollama Go client: github.com/ollama/ollama/api for model management
- Structured logging: logrus or zap with provider/request tracing
- Observability: OpenTelemetry for distributed tracing across providers
- Monitoring: Prometheus metrics (provider usage, latency, costs), Grafana dashboards
- Database: PostgreSQL or SQLite for persistence (processing history, model metadata)
- Platform installers: MSI (Windows), .deb/.rpm (Linux), .pkg (macOS)
- Additional AI providers: Potential support for Claude, Gemini (via provider abstraction)

**Refactoring Targets**:
- Package extraction: pdfprocessor/, imagegen/, canvasanalyzer/, handwritingrecog/, ollamaruntime/
- Dependency injection: Remove global config variable
- Provider abstraction: Unified interface for all LLM providers
- Atomic architecture: Further decomposition of large files (handlers.go ~2000 lines)

**Model Format Considerations**:
- Primary: GGUF (Ollama native, compressed, quantized)
- Quantization: 4-bit, 5-bit, 8-bit models for size/performance tradeoffs
- Download: Progressive download with resume support for large models
- Storage: Efficient deduplication (Ollama handles shared layers)
