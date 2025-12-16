# Tech Stack

## Programming Language & Runtime

**Go 1.x**
- Primary language for backend service
- Chosen for: performance, concurrency primitives (goroutines, channels), single-binary deployment, strong standard library
- Used in: all application code (main.go, handlers.go, monitorcanvus.go, core/, canvusapi/, logging/)

**CGo**
- C/Go interoperability for native library integration
- Used for: Embedding llama.cpp and stable-diffusion.cpp
- Build requirements: CMake, C++ compiler (MSVC/GCC), CUDA toolkit

## Installation & Packaging

**NSIS (Nullsoft Scriptable Install System)** - Windows
- Windows installer creation framework
- Creates CanvusLocalLLM-Setup.exe with simple wizard
- Features: License page, install location, file installation, uninstaller generation
- Bundles: Application binary, llama.cpp libraries, stable-diffusion.cpp libraries, Bunny model (or download on first run)
- Optional: Windows Service creation checkbox

**Debian Package Tools** - Linux
- dpkg-deb: Create .deb packages for Debian/Ubuntu
- Package structure: DEBIAN/ directory with control, postinst scripts
- Install location: /opt/canvuslocallm/
- Bundles all components including model files

**Tarball Distribution** - Linux
- .tar.gz archive for non-Debian distributions
- Includes install.sh script for extraction and setup
- Portable distribution method

**Minimal Configuration**
- .env file with only Canvus credentials:
  - CANVUS_SERVER: Canvus server URL
  - CANVUS_API_KEY: API key authentication
  - CANVUS_USERNAME/CANVUS_PASSWORD: Alternative username/password auth
  - CANVAS_ID: Target canvas identifier
- No model selection, no provider options, no complexity
- All AI parameters preconfigured and optimized

## Embedded LLM Infrastructure

**llama.cpp**
- C++ LLM inference engine embedded via CGo
- Single-binary deployment with bundled native libraries (.dll/.so)
- Cross-platform: Windows (Visual Studio 2022 + CUDA), Linux (GCC + CUDA)
- GGUF model format support
- **CUDA acceleration required** - optimized for NVIDIA RTX GPUs
- Go bindings: go-skynet/go-llama.cpp or custom CGo wrapper

**Bunny v1.1 Llama-3-8B-V**
- Source: https://huggingface.co/BAAI/Bunny-v1_1-Llama-3-8B-V
- Multimodal model: text generation + vision/image analysis
- GGUF quantized format for efficient inference
- Capabilities: Text generation, image understanding, PDF analysis support
- Bundled with installer or downloaded on first run
- No user model selection - single hardcoded model

**stable-diffusion.cpp**
- C++ image generation engine from llama.cpp ecosystem
- CGo integration following same pattern as llama.cpp
- Cross-platform native builds for Windows/Linux with CUDA
- Local text-to-image generation
- Model bundled with installer

**Runtime Architecture**
- Embedded integration: llama.cpp and stable-diffusion.cpp loaded as shared libraries within Go process
- No separate service: All inference in-process via CGo calls
- Context management: Thread-safe llama context pool for concurrent requests
- Memory management: Optimized for RTX GPU VRAM
- Health monitoring: Inference failure detection, GPU memory tracking, automatic recovery

## Canvas Integration

**Canvus REST API**
- Real-time workspace monitoring via streaming endpoint (subscribe=true)
- Widget CRUD operations (notes, images, PDFs)
- File upload (multipart/form-data)
- Coordinate system: relative to parent widget
- Authentication: API key via X-Api-Key header OR username/password

## Core Dependencies

**HTTP Client & Networking**
- net/http (standard library): HTTP client with configurable TLS
- TLS Configuration: Optional self-signed certificate support (ALLOW_SELF_SIGNED_CERTS)
- Timeouts: Configurable per operation
- Model downloads: HTTP client with Range request support for resumable downloads

**PDF Processing**
- github.com/ledongthuc/pdf: PDF text extraction
- Custom chunking logic for feeding to LLM

**Concurrency & Synchronization**
- context.Context: Cancellation and timeout propagation
- sync.RWMutex: Thread-safe widget state management
- sync.Mutex: Shared resource protection (logs, llama context pool)
- Goroutines: Asynchronous processing with lifecycle management
- Channels: Inter-goroutine communication

**Configuration Management**
- godotenv: .env file parsing
- Environment variables: Minimal configuration via env vars
- Validation: Canvus credentials checked at startup

**Logging**
- File logging: app.log with timestamps and source information
- Console logging: Color-coded output via github.com/fatih/color
- Dual output: Simultaneous file and console via logging.LogHandler()
- Planned: GPU metrics, inference performance stats

**Service Integration**
- Windows: github.com/kardianos/service for Windows Service API
- Linux: systemd unit file generation

## Architecture Patterns

**Atomic Design**
- Atoms: Pure functions (env parsers, HTTP client factory, llama.cpp context loading)
- Molecules: Simple compositions (Config struct, canvusapi.Client, inference request builders)
- Organisms: Feature modules (Monitor, handlers, API client, llamaruntime package, sdruntime package)
- Pages: Composition root (main.go with llama.cpp lifecycle management)

**CGo Integration Pattern**
- C++ library compilation: CMake-based build of llama.cpp/stable-diffusion.cpp with CUDA
- Shared library bundling: .dll (Windows), .so (Linux) packaged with Go binary
- Go bindings: Wrapper packages (llamaruntime/, sdruntime/) providing Go-friendly API
- Memory safety: Careful handling of pointers, manual memory management, defer cleanup
- Thread safety: Context pool pattern for concurrent inference requests
- Error handling: Translate C errors to Go error types

**Dependency Injection**
- Config passed to constructors (NewClient, NewMonitor)
- Target: Eliminate global config variable in handlers.go
- llama.cpp integration: Inject model context pool into inference handlers

**Error Handling**
- Sentinel errors: ErrInvalidInput, ErrModelLoadFailed, ErrConfigMissing
- Error wrapping: fmt.Errorf with %w for context
- Custom types: APIError with status codes, LlamaError for CGo errors
- Retry logic: Exponential backoff for transient failures
- User-friendly errors: Helpful messages for configuration issues

**Concurrency Model**
- Context-based cancellation: Propagated through call chains including CGo inference
- Signal handling: Graceful shutdown on SIGINT/SIGTERM (includes llama.cpp context cleanup)
- Thread-safe state: RWMutex for widget cache, Mutex for llama context pool
- CGo safety: Proper locking for C library calls

## Build & Deployment

**Build Tools**
- Go toolchain: go build, go test, go mod
- CMake: Build llama.cpp, stable-diffusion.cpp native libraries with CUDA
- C++ compiler: MSVC (Windows), GCC (Linux)
- CUDA toolkit: Required for GPU acceleration
- NSIS: Windows installer compilation (makensis.exe)
- dpkg-deb: Debian package creation
- tar/gzip: Tarball creation

**Build Process**
1. Build llama.cpp with CUDA support for target platform
2. Build stable-diffusion.cpp with CUDA support for target platform
3. Place compiled libraries in lib/ directory
4. Build Go application with CGo enabled (CGO_ENABLED=1)
5. Link Go binary against native libraries
6. Download/prepare Bunny v1.1 model files
7. Create installer packages with bundled binaries, libraries, and model

**Target Platforms**
- Windows (amd64): MSVC + CUDA build, NVIDIA RTX GPU required
- Linux (amd64): GCC + CUDA build, NVIDIA RTX GPU required

**Deployment Model**
- **Installation Method**: Native installers (CanvusLocalLLM-Setup.exe, .deb, .tar.gz)
- **Install Locations**:
  - Windows: C:\Program Files\CanvusLocalLLM\
  - Linux: /opt/canvuslocallm/
- **Installed Components**:
  - Application binary (CanvusLocalLLM.exe or canvuslocallm)
  - Native libraries (llama + CUDA, stable-diffusion + CUDA)
  - Bunny v1.1 model files (bundled or downloaded on first run)
  - Stable diffusion model files
  - Configuration template (.env.example)
- **Configuration**: .env file with Canvus credentials only
- **Logging**: app.log in install directory

**Directory Structure (Post-Installation)**:
```
C:\Program Files\CanvusLocalLLM\  (or /opt/canvuslocallm/)
├── CanvusLocalLLM.exe            (main application)
├── lib/
│   ├── llama.dll                 (llama.cpp + CUDA)
│   └── stable-diffusion.dll      (stable-diffusion.cpp + CUDA)
├── models/
│   ├── bunny-v1.1-llama-3-8b-v.gguf  (text + vision model)
│   └── sd-model.safetensors      (image generation model)
├── .env.example                  (configuration template)
├── .env                          (user config - Canvus creds only)
├── downloads/                    (temporary files)
├── app.log                       (application log)
└── README.txt
```

**First-Run Experience**
1. Check for .env file - if missing, display instructions to copy from .env.example
2. Validate Canvus credentials - test connection to server
3. If model not bundled, download Bunny v1.1 with progress bar
4. Initialize llama.cpp runtime with CUDA
5. Run quick inference test to verify GPU acceleration
6. Begin canvas monitoring

## Security

**TLS/SSL**
- Certificate validation: Enabled by default
- Development mode: Optional self-signed cert support

**Credential Management**
- Environment variables: CANVUS_API_KEY or CANVUS_USERNAME/CANVUS_PASSWORD
- File security: .env excluded from version control (.gitignore)
- No cloud API keys needed - pure local processing

**Authentication**
- Web UI: Password protection (WEBUI_PWD)
- Canvus: API key or username/password authentication

**Data Privacy**
- All AI processing on local hardware - zero external data transmission
- No cloud providers, no telemetry, no phone-home
- Complete data sovereignty guaranteed

**CGo Security**
- Memory safety: Careful pointer handling, bounds checking
- Input validation: Sanitize prompts before passing to C layer
- Resource limits: Context size limits, memory limits, inference timeouts
- Crash isolation: Recover from CGo panics

**Installer Security**
- Code signing: Windows executables signed (release builds)
- Checksum verification: SHA256 checksums published with releases
- HTTPS downloads: Models downloaded over HTTPS

## Testing

**Test Framework**
- testing package (standard library): Unit and integration tests
- Table-driven tests: Multiple scenarios per function
- Subtests: t.Run() for organization
- CGo testing: Mock C functions, test memory safety

**Test Organization**
- tests/ directory: All test files
- Fixtures: tests/test_data.go for shared test data
- Coverage: Canvas API, LLM integration, configuration validation

**Testing Strategy**
- **Unit**: CGo wrapper functions, configuration validation, canvas API
- **Integration**: llama.cpp lifecycle, model loading, inference requests
- **End-to-end**: Full installation, first-run, canvas monitoring
- **Performance**: Inference throughput, GPU utilization, memory usage
- **Platform testing**:
  - Windows: MSVC + CUDA builds, installer on Windows 10/11
  - Linux: GCC + CUDA builds, .deb and tarball installation

## Future Considerations

**Planned Additions**:
- Structured logging: logrus or zap with GPU metrics
- SQLite: Processing history, inference logs
- Observability: GPU monitoring, inference statistics dashboard

**Refactoring Targets**:
- Package extraction: pdfprocessor/, imagegen/, canvasanalyzer/, llamaruntime/, sdruntime/
- Dependency injection: Remove global config variable
- Atomic architecture: Decompose handlers.go

**Model Considerations**:
- Primary: GGUF format for llama.cpp
- Bunny v1.1 is the fixed model - no model switching
- Future: Evaluate newer multimodal models as they become available
- Quantization: Optimized for RTX GPU VRAM constraints
