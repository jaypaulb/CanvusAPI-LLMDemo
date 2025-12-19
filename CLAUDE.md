# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**CanvusLocalLLM** is a Go-based integration service that connects Canvus collaborative workspaces with local AI services via llama.cpp ecosystem. It monitors canvas widgets in real-time, processes AI prompts enclosed in `{{ }}`, and handles PDF analysis, canvas analysis, and image generation using embedded multimodal models with cloud fallback support.

## Build and Development Commands

### Building the Application
```bash
# Build for current platform
go build -o CanvusAPI-LLM.exe .

# Build for specific platforms
GOOS=linux GOARCH=amd64 go build -o canvusapi-linux-amd64 .
GOOS=darwin GOARCH=arm64 go build -o canvusapi-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o canvusapi-windows-amd64.exe .
```

### Running the Application
```bash
# Ensure .env is configured first
./CanvusAPI-LLM.exe

# Or on Linux/macOS
./canvusapi-linux-amd64
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Run specific test file
go test ./tests/canvas_check_test.go

# Run specific test function
go test -run TestSpecificFunction ./tests/

# Verbose output
go test -v ./...
```

### Code Quality
```bash
# Format code
go fmt ./...

# Run static analysis
go vet ./...

# Run linter (if installed)
golangci-lint run

# Tidy dependencies
go mod tidy

# Update dependencies
go get -u ./...
```

## Agent-OS Integration

This project uses Agent-OS (v2.1.1) for structured development workflows with Claude Code.

### Available Slash Commands

**Product Planning & Specification:**
- `/plan-product` - Create product planning documents (mission, roadmap)
- `/shape-spec` - Shape specification through targeted questions and requirements gathering
- `/write-spec` - Write detailed specification document for development

**Task Management:**
- `/create-tasks` - Create detailed task list from specification
- `/implement-tasks` - Implement tasks following the task list
- `/orchestrate-tasks` - Orchestrate multi-task workflows across parallel workstreams

**Continuous Improvement:**
- `/improve-skills` - Improve Agent-OS skills based on project learnings

### Available Skills (Auto-triggered)

Agent-OS skills are automatically invoked by Claude Code when working with relevant files:

**Backend Skills** (Go, APIs, Databases):
- `backend-api` - API endpoint design and implementation
- `backend-migrations` - Database migration patterns
- `backend-models` - Data model design
- `backend-queries` - Database query optimization

**Frontend Skills** (UI/UX):
- `frontend-accessibility` - Accessibility standards (WCAG)
- `frontend-components` - Component architecture
- `frontend-css` - CSS best practices
- `frontend-responsive` - Responsive design patterns

**Global Skills** (Cross-cutting):
- `global-atomic-design` - Atomic design methodology (Atoms → Molecules → Organisms → Templates → Pages)
- `global-coding-style` - Code style and formatting standards
- `global-commenting` - Documentation and comment standards
- `global-conventions` - Naming and structural conventions
- `global-error-handling` - Error handling patterns
- `global-tech-stack` - Technology selection and architecture
- `global-validation` - Input validation patterns

**Testing Skills**:
- `testing-test-writing` - Test organization and best practices

Skills guide implementation decisions and ensure consistency with established patterns.

## Beads Issue Tracking

This project uses Beads for lightweight, git-native issue tracking. All work should be tracked in Beads, not in TodoWrite or markdown files.

### Core Workflow

**DO NOT use TodoWrite tool** - All task tracking happens in Beads.

**Finding Work:**
```bash
bd ready                          # Show issues ready to work (no blockers)
bd list --status=open             # All open issues
bd list --status=in_progress      # Your active work
bd show <id>                      # Detailed issue view with dependencies
```

**Creating Issues:**
```bash
bd create --title="..." --type=task|bug|feature --priority=2

# Priority: 0-4 or P0-P4
# 0=critical, 1=high, 2=medium (default), 3=low, 4=backlog
# Do NOT use "high"/"medium"/"low" - use numeric values

# Creating multiple issues: use parallel subagents for efficiency
```

**Working on Issues:**
```bash
bd update <id> --status=in_progress    # Claim work
bd update <id> --assignee=username     # Assign to someone
bd close <id>                          # Mark complete
bd close <id1> <id2> ...               # Close multiple at once (more efficient)
bd close <id> --reason="explanation"   # Close with reason
```

**Dependencies & Blocking:**
```bash
bd dep add <issue> <depends-on>   # Issue depends on depends-on
bd blocked                        # Show all blocked issues
bd show <id>                      # See what's blocking/blocked by this issue
```

**Project Health:**
```bash
bd stats                          # Project statistics
bd doctor                         # Check for issues (sync, hooks, etc.)
```

### Git Integration

Beads is git-native and syncs automatically via hooks. However, at the **end of every session**:

```bash
# CRITICAL: Session close checklist (run before saying "done")
git status              # Check what changed
git add <files>         # Stage code changes
bd sync                 # Commit beads changes
git commit -m "..."     # Commit code
bd sync                 # Commit any new beads changes
git push                # Push to remote
```

**Never skip this checklist.** Work is not done until pushed.

To check sync status without syncing:
```bash
bd sync --status
```

To recover context after session break:
```bash
bd prime                # Recover beads workflow context
```

### Beads + Agent-OS Workflow

1. Use `/shape-spec` or `/write-spec` to create specifications
2. Use `/create-tasks` to generate task list from spec
3. Create Beads issues from task list: `bd create --title="Task X.X.X: ..." --type=task`
4. Use `bd dep add` to link dependent tasks
5. Use `/implement-tasks` or `/orchestrate-tasks` to execute
6. Close issues as you complete them: `bd close <id1> <id2> ...`
7. End session with full git sync checklist

### Using bv as an AI Sidecar

`bv` is a fast terminal UI for Beads projects (`.beads/beads.jsonl`). It renders lists/details and precomputes dependency metrics (PageRank, critical path, cycles, etc.) so you instantly see blockers and execution order. For agents, it's a **graph sidecar**: instead of parsing JSONL or risking hallucinated traversal, call the robot flags to get deterministic, dependency-aware outputs.

**⚠️ IMPORTANT: As an agent, you must ONLY use bv with the `--robot-*` flags, otherwise you'll get stuck in the interactive TUI that's intended for human usage only!**

**AI-Facing Commands:**

```bash
# Show all AI-facing commands
bv --robot-help

# JSON graph metrics (PageRank, betweenness, HITS, critical path, cycles)
# Top-N summaries for quick triage
bv --robot-insights

# JSON execution plan: parallel tracks, items per track, unblocks lists
# Shows what each item frees up
bv --robot-plan

# JSON priority recommendations with reasoning and confidence
bv --robot-priority

# List recipes (default, actionable, blocked, etc.)
# Apply via: bv --recipe <name>
bv --robot-recipes

# JSON diff of issue changes since commit/date
# Shows new/closed items and cycles introduced/resolved
bv --robot-diff --diff-since <commit|date>
```

**Usage Pattern:**

Use these commands instead of hand-rolling graph logic; `bv` already computes the hard parts so agents can act safely and quickly.

**Examples:**

```bash
# Get dependency insights before planning work
bv --robot-insights

# Get recommended execution order
bv --robot-plan

# Check what's blocking critical path
bv --recipe blocked --robot-insights

# See what changed since last release
bv --robot-diff --diff-since v1.0.0
```

**Never** run `bv` without `--robot-*` flags as an agent - you'll hang waiting for interactive input!

## Architecture and Code Organization

### High-Level Architecture

**Atomic Design Hierarchy:**

1. **Atoms** (Pure Functions & Primitives)
   - `logging/logging.go`: Simple log handler function
   - `core/config.go`: Environment variable parsers (`getEnvOrDefault`, `parseIntEnv`, etc.)
   - `handlers/text_processing.go`: `GenerateCorrelationID()`, `TruncateText()`, `ExtractAIPrompt()`
   - `handlers/validation.go`: Input validation functions
   - `pdfprocessor/atoms.go`: PDF utility functions
   - `imagegen/atoms.go`: URL validation, format detection
   - `ocrprocessor/atoms.go`: API key validation
   - `canvasanalyzer/atoms.go`: Widget filtering/formatting

2. **Molecules** (Simple Compositions)
   - `core/Config`: Configuration struct composing multiple environment variables
   - `core/GetHTTPClient()`: HTTP client factory composing TLS config + timeout
   - `pdfprocessor/extractor.go`: PDF text extraction composing atoms
   - `pdfprocessor/chunker.go`: Text chunking logic
   - `pdfprocessor/summarizer.go`: AI summarization with OpenAI
   - `imagegen/openai_provider.go`: OpenAI DALL-E image generation
   - `imagegen/azure_provider.go`: Azure OpenAI image generation
   - `imagegen/downloader.go`: Image download from URLs
   - `ocrprocessor/client.go`: Google Vision API client
   - `canvasanalyzer/fetcher.go`: Widget fetching with retry logic
   - `canvasanalyzer/analyzer.go`: Canvas analysis logic

3. **Organisms** (Complex Feature Units)
   - `Monitor`: Canvas monitoring service managing widget state, updates, and processing
   - `pdfprocessor.Processor`: Complete PDF pipeline (extract → chunk → summarize)
   - `imagegen.Generator`: Image generation pipeline (cloud/local → download → upload)
   - `imagegen/sd.Client`: Stable Diffusion client for local image generation
   - `ocrprocessor.Processor`: OCR pipeline (validate → process → return text)
   - `canvasanalyzer.Processor`: Canvas analysis pipeline (fetch → filter → analyze)
   - `canvusapi.Client`: Full API interaction layer with methods for widgets, notes, images
   - `webui.Server`: Web dashboard server with WebSocket real-time updates
   - `webui.CanvasHealthMonitor`: Periodic health checks for canvas connectivity
   - `webui.CanvasManager`: Multi-canvas lifecycle management
   - `llamaruntime.Client`: LLM client for local inference via llama.cpp
   - `db.Repository`: Database repository for persistent storage
   - `metrics.GPUCollector`: GPU metrics collection (NVML/nvidia-smi)
   - Handler functions in `handlers.go`: Wire organisms together for canvas events

4. **Templates** (Structural Contracts)
   - `imagegen.Provider` interface: Abstraction for image generation providers
   - `canvasanalyzer.WidgetClient` interface: Abstraction for widget fetching
   - `core.Config` interface (implicit)
   - Error handling patterns (`APIError` struct, sentinel errors)

5. **Pages** (Composition Roots)
   - `main.go`: Application bootstrap, wires together Monitor + Client + Config
   - `webui/server.go`: Web dashboard server, wires together health monitoring + canvas management + metrics
   - Context-based lifecycle management with signal handling
   - Graceful shutdown coordination via `shutdown/` package

### Package Structure

```
go_backend/
├── main.go                  # Entry point: loads config, creates client & monitor (843 lines)
├── handlers.go              # AI processing handlers (wires organisms together) (2,497 lines)
├── monitorcanvus.go        # Canvas monitoring service with streaming updates (635 lines)
├── core/                    # Core configuration and shared utilities (11,071 lines)
│   ├── config.go           # Configuration management (env parsing, HTTP client factory)
│   ├── ai.go               # OpenAI client creation and basic AI response
│   ├── modelmanager.go     # Model loading and management
│   ├── validation.go       # Configuration validation suite
│   └── types.go            # Shared type definitions
├── canvusapi/              # Canvus API client organism
│   └── canvusapi.go        # Widget CRUD, file uploads, API error handling
├── logging/                # Structured logging infrastructure (4,467 lines)
│   ├── logger.go           # Main logger implementation
│   ├── config.go           # Logging configuration
│   ├── handler.go          # Log handler utilities
│   └── logger_test.go      # Logger test suite
├── handlers/               # Handler utility atoms (1,030 lines)
│   ├── text_processing.go  # GenerateCorrelationID, TruncateText, ExtractAIPrompt
│   ├── validation.go       # Input validation atoms
│   ├── location.go         # Widget location calculation
│   ├── json_parsing.go     # JSON parsing utilities
│   ├── note_updater.go     # Note update helpers
│   ├── progress_reporter.go # Progress reporting utilities
│   └── ai_client_factory.go # AI client creation factory
├── pdfprocessor/           # PDF processing organism (4,070 lines)
│   ├── atoms.go            # Pure PDF utility functions
│   ├── extractor.go        # PDF text extraction molecule
│   ├── chunker.go          # Text chunking molecule
│   ├── summarizer.go       # AI summarization molecule
│   ├── processor.go        # Processor organism (orchestrates all)
│   └── integration_test.go # Comprehensive integration tests (91.9% coverage)
├── imagegen/               # Image generation organism (4,881 lines)
│   ├── atoms.go            # URL validation, format detection
│   ├── placement.go        # Canvas placement calculation
│   ├── openai_provider.go  # OpenAI DALL-E provider molecule
│   ├── azure_provider.go   # Azure OpenAI provider molecule
│   ├── downloader.go       # Image download molecule
│   ├── generator.go        # Generator organism (orchestrates all)
│   └── sd/                 # Stable Diffusion support (CGo bindings)
│       ├── client.go       # SD client interface
│       ├── client_sd.go    # CGo implementation (requires stable-diffusion.cpp)
│       ├── client_stub.go  # Fallback stub
│       └── types.go        # SD data structures
├── ocrprocessor/           # OCR processing organism (2,350 lines)
│   ├── atoms.go            # API key validation atoms
│   ├── client.go           # Google Vision API client molecule
│   ├── processor.go        # Processor organism (orchestrates all)
│   └── processor_test.go   # Integration tests (94.5% coverage)
├── canvasanalyzer/         # Canvas analysis organism (3,249 lines)
│   ├── atoms.go            # Widget filtering/formatting atoms
│   ├── fetcher.go          # Widget fetching molecule with retry
│   ├── analyzer.go         # Analysis logic molecule
│   ├── processor.go        # Processor organism (orchestrates all)
│   └── integration_test.go # Integration tests (94.5% coverage)
├── webui/                  # Real-time web dashboard (8,313 lines)
│   ├── server.go           # HTTP server and API routing
│   ├── dashboard_api.go    # Dashboard API endpoints organism
│   ├── health_monitor.go   # CanvasHealthMonitor organism (periodic health checks)
│   ├── canvas_manager.go   # CanvasManager organism (multi-canvas lifecycle)
│   ├── websocket.go        # WebSocket real-time updates
│   └── static/             # Frontend assets (HTML, CSS, JS)
├── llamaruntime/           # llama.cpp integration (10,373 lines)
│   ├── client.go           # LLM client organism
│   ├── bindings.go         # CGo bindings to llama.cpp
│   ├── types.go            # llama.cpp data structures
│   └── benchmarks/         # Performance benchmarks
├── db/                     # SQLite database layer (5,357 lines)
│   ├── repository.go       # Database repository organism
│   ├── migrations.go       # Schema migrations
│   ├── models.go           # Data models
│   └── queries.go          # SQL query builders
├── metrics/                # Metrics collection for dashboard (2,463 lines)
│   ├── store.go            # Metrics storage
│   ├── collector.go        # Metrics collection
│   ├── gpu_collector.go    # GPUCollector organism (NVML/nvidia-smi)
│   └── broadcaster.go      # Task event broadcasting
├── sdruntime/              # Stable Diffusion runtime (CGo bindings)
│   ├── cgo_bindings_sd.go  # CGo bindings (stubbed, awaiting library)
│   └── types.go            # SD data structures
├── vision/                 # Image preprocessing utilities
│   └── preprocessing.go    # Image transformation atoms
├── shutdown/               # Graceful shutdown management
│   └── handler.go          # Shutdown coordination
└── tests/                  # Integration test suite (139 test files total)
    ├── canvas_check_test.go
    ├── llm_test.go
    ├── testAPI_test.go
    └── test_data.go
```

### Key Components and Responsibilities

**main.go** (Page)
- Loads `.env` configuration
- Initializes logging to `app.log`
- Creates `canvusapi.Client` with server details
- Instantiates `Monitor` with client and config
- Manages application lifecycle with context cancellation and signal handling

**core/config.go** (Molecules)
- Defines `Config` struct with all application settings
- Provides atomic environment variable parsers
- `LoadConfig()`: Validates required variables, returns populated Config
- `GetHTTPClient()`: Creates HTTP client with optional TLS cert validation bypass
- Critical: HTTP clients MUST use `GetHTTPClient()` to respect `ALLOW_SELF_SIGNED_CERTS`

**core/ai.go** (Molecule)
- `TestAIResponse()`: Generates AI responses using OpenAI API
- `createOpenAIClient()`: Configures OpenAI client with proper base URL (TEXT_LLM_URL → BASE_LLM_URL) and HTTP client

**canvusapi/canvusapi.go** (Organism)
- `Client`: Main API client with HTTP transport
- `NewClient()`: Factory with TLS configuration support
- **CRITICAL**: Widget locations are RELATIVE to parent widget (see line 26-31)
- Methods: `GetWidgets()`, `CreateNote()`, `UpdateWidget()`, `UploadImage()`, etc.
- Error handling via `APIError` struct

**monitorcanvus.go** (Organism)
- `Monitor`: Canvas monitoring service
- `Start()`: Main event loop with context cancellation
- `connectAndStream()`: Establishes streaming connection to Canvus API
- `handleUpdate()`: Processes widget updates, detects AI prompts in `{{ }}`
- Thread-safe widget state management with `sync.RWMutex`
- ✅ Dependency injection: Config passed via Monitor struct to all handlers

**handlers.go** (Organism)
- AI processing handlers for different content types (2,497 lines)
- Note processing: Extracts `{{ }}` prompts, calls OpenAI, creates response notes
- PDF analysis: Downloads PDF, extracts text, generates summary
- Canvas analysis: Collects all widgets, generates overview
- Image generation: Supports OpenAI DALL-E and Azure OpenAI
- Handwriting recognition: Google Vision API integration
- Dashboard metrics integration: Records task starts, completions, errors
- ✅ Dependency injection: `HandlerDependencies` struct provides metrics and task broadcasting
- All handlers receive config as parameter (no global config)

**webui/server.go** (Page - Additional Composition Root)
- Real-time web dashboard with WebSocket support
- HTTP server and API routing
- Static file serving for frontend assets
- Health monitoring and multi-canvas management endpoints

**webui/health_monitor.go** (Organism)
- CanvasHealthMonitor: Periodic health checks for canvas connectivity
- Monitors widget counts, processing status, error rates
- Broadcasts health status via WebSocket

**webui/canvas_manager.go** (Organism)
- CanvasManager: Multi-canvas lifecycle management
- Start/stop canvas monitoring dynamically
- Coordinate multiple canvas instances

**llamaruntime/client.go** (Organism)
- LLM client for local inference via llama.cpp
- CGo bindings to llama.cpp C API
- Model loading, inference, context management
- **INCOMPLETE**: Vision inference not yet implemented (llamaruntime/bindings.go:762)

**db/repository.go** (Organism)
- SQLite database layer for persistent storage
- Schema migrations
- Metrics storage and retrieval
- Processing history tracking

**metrics/gpu_collector.go** (Organism)
- GPUCollector: GPU metrics collection
- Uses NVML for NVIDIA GPUs (falls back to nvidia-smi)
- **INCOMPLETE**: Full GPU memory query via NVML not implemented (llamaruntime/bindings.go:803)

### Critical Implementation Notes

**Widget Coordinate System**
- Widget locations in API responses are RELATIVE to parent
- To get absolute canvas coordinates: `parentLocation + widgetRelativeLocation`
- This is essential for correct widget placement (see `canvusapi/canvusapi.go:26-31`)

**TLS Configuration**
- `ALLOW_SELF_SIGNED_CERTS` controls SSL certificate validation
- ALL HTTP clients MUST use `core.GetHTTPClient()` or `core.GetDefaultHTTPClient()`
- This ensures consistent TLS behavior across Canvus API, OpenAI, and Google Vision API
- Never create raw `http.Client{}` without checking TLS settings

**AI Endpoint Configuration**
- `BASE_LLM_URL`: Default for all LLM operations (default: `http://127.0.0.1:1234/v1`)
- `TEXT_LLM_URL`: Optional override for text generation (falls back to BASE_LLM_URL)
- `IMAGE_LLM_URL`: Image generation endpoint (default: OpenAI API)
- Azure OpenAI support via `AZURE_OPENAI_ENDPOINT`, `AZURE_OPENAI_DEPLOYMENT`, `AZURE_OPENAI_API_VERSION`

**Concurrency Patterns**
- Context-based cancellation throughout (`context.Context`)
- Goroutine lifecycle management with signal handling
- Thread-safe shared state using `sync.RWMutex` and `sync.Mutex`
- Semaphore pattern for rate limiting (`MaxConcurrent`)

**Error Handling**
- Sentinel errors for expected conditions (`ErrInvalidInput`)
- Error wrapping with `fmt.Errorf(..., %w, err)` for context
- Custom `APIError` type with status codes
- Retry logic with exponential backoff (configurable via `MaxRetries`, `RetryDelay`)

## Atomic Design Principles for This Codebase

### Building Bottom-Up

When adding features:
1. **Start with atoms**: Pure functions without dependencies
2. **Compose into molecules**: Small helpers combining 2-3 atoms
3. **Build organisms**: Feature modules with clear responsibilities
4. **Keep pages minimal**: `main.go` should only wire components together

### Dependency Rules

- Atoms depend on nothing (or only standard library)
- Molecules depend on atoms
- Organisms depend on molecules and atoms
- Pages depend on all levels but contain no business logic
- **Never** have sideways dependencies at the same level

### Refactoring Guidelines

When refactoring existing code:
1. **Identify buried atoms**: Extract pure functions from complex handlers
2. **Find implicit molecules**: Repeated atom combinations → explicit functions
3. **Split large organisms**: If a file has >500 lines, look for molecules trying to escape
4. **Test by level**:
   - Atoms: Unit tests, pure input/output
   - Molecules: Unit tests, minimal mocking
   - Organisms: Integration tests, may need mocks
   - Pages: End-to-end tests

### Refactoring Status

**Phase 4 (Package Extraction): COMPLETE ✅**
- ✅ Extracted `pdfprocessor/` package (4,070 lines, 91.9% test coverage)
- ✅ Extracted `imagegen/` package (4,881 lines, 73% coverage, includes SD support)
- ✅ Extracted `ocrprocessor/` package (2,350 lines, 94.5% coverage)
- ✅ Extracted `canvasanalyzer/` package (3,249 lines, 94.5% coverage)
- ✅ Extracted `handlers/` package (1,030 lines, shared utilities)
- ✅ Implemented `logging/` package (4,467 lines, structured logging)
- `handlers.go` remains at 2,497 lines but delegates heavy lifting to extracted packages

**Additional Features Beyond Original Roadmap:**
- ✅ `webui/` package (8,313 lines) - Real-time dashboard with WebSocket
- ✅ `llamaruntime/` package (10,373 lines) - llama.cpp CGo bindings
- ✅ `db/` package (5,357 lines) - SQLite persistence layer
- ✅ `metrics/` package (2,463 lines) - Metrics collection and GPU monitoring
- ✅ `sdruntime/` - Stable Diffusion CGo bindings (stubbed, awaiting library)
- ✅ `vision/`, `shutdown/` - Supporting utilities

**Dependency Injection Migration: COMPLETE ✅**
- ✅ Eliminated global `config` variable - now passed via Monitor struct to all handlers
- ✅ Implemented `HandlerDependencies` struct for metrics and task broadcasting
- Remaining: logMutex, downloadsMutex (tracked in separate Beads issues)

**Current Technical Debt (Priority Order):**

1. **Core Package Bloat** (MEDIUM PRIORITY)
   - `core/` package has grown to 11,071 lines
   - Mixes atoms (config parsing), molecules (HTTP clients), and validation suites
   - **Recommended decomposition**:
     - `core/modelmanager/` → separate package for model loading
     - `core/validation/` → separate package for validation suite
     - Keep only config.go, types.go, ai.go in core/

2. **handlers.go Size** (LOW PRIORITY)
   - 2,497 lines is still large but manageable
   - Could split into handler-specific files:
     - `handlers/note_handler.go` (~400 lines)
     - `handlers/image_handler.go` (~400 lines)
     - `handlers/canvas_handler.go` (~300 lines)
     - `handlers/snapshot_handler.go` (~200 lines)
   - **Note**: Orchestration logic is expected to be substantial; current delegation pattern is acceptable

3. **Incomplete Features** (BLOCKED BY EXTERNAL DEPENDENCIES)
   - Vision inference in llamaruntime (TODO at llamaruntime/bindings.go:762)
   - Full GPU memory query via NVML (TODO at llamaruntime/bindings.go:803)
   - Stable Diffusion library integration (TODOs at sdruntime/cgo_bindings_sd.go:89,135,186,196,206)
   - **Status**: Stubbed and ready for implementation when dependencies available

4. **Test Coverage Gaps** (LOW PRIORITY)
   - `imagegen/sd/`: 0% coverage (CGo-dependent, requires SD library)
   - `imagegen`: 73% coverage (cloud API mocking infrastructure needed for 85%+)
   - Most packages have 85-95% coverage; gaps are in external API integration

## Environment Configuration

Required variables (see `example.env`):
- `CANVUS_SERVER`: Canvus server URL
- `CANVAS_NAME`, `CANVAS_ID`: Target canvas
- `OPENAI_API_KEY`, `CANVUS_API_KEY`: API keys
- `WEBUI_PWD`: Web UI password

Key configuration patterns:
- Token limits for different AI operations (PDF, canvas, notes)
- Timeout configuration (AI timeout, processing timeout)
- Concurrency limits (`MAX_CONCURRENT`)
- Downloads directory management

## Testing Strategy

### Test Files
- `tests/canvas_check_test.go`: Canvus API connectivity tests
- `tests/llm_test.go`: OpenAI integration tests
- `tests/testAPI_test.go`: Comprehensive API endpoint tests
- `tests/test_data.go`: Shared test fixtures

### Testing Best Practices
- Table-driven tests for multiple scenarios
- Use `t.Run()` for subtests
- Mock external dependencies (OpenAI, Canvus API) via interfaces
- Test error conditions and edge cases
- Use `-race` flag to detect race conditions in concurrent code

### Test Organization
```go
func TestFeature(t *testing.T) {
    tests := []struct {
        name     string
        input    Type
        expected Type
        wantErr  bool
    }{
        {"valid case", validInput, validOutput, false},
        {"error case", invalidInput, nil, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := Function(tt.input)
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

## Common Development Patterns

### Using the Extracted Packages

**PDF Processing (pdfprocessor)**
```go
import "go_backend/pdfprocessor"

// Create processor with default config
config := pdfprocessor.DefaultProcessorConfig()
processor := pdfprocessor.NewProcessorWithProgress(config, aiClient, progressCallback)

// Process a PDF file
result, err := processor.Process(ctx, filePath, "Summarize this document")
if err != nil {
    return err
}
fmt.Println(result.Summary)
```

**Image Generation (imagegen)**
```go
import "go_backend/imagegen"

// Create generator from config
generator, err := imagegen.NewGeneratorFromConfig(config, canvusClient, logger)
if err != nil {
    return err
}

// Generate image and upload to canvas
err = generator.GenerateAndUpload(ctx, prompt, parentWidget)
```

**OCR Processing (ocrprocessor)**
```go
import "go_backend/ocrprocessor"

// Create processor
processor, err := ocrprocessor.NewProcessor(apiKey, httpClient, logger, ocrprocessor.DefaultProcessorConfig())
if err != nil {
    return err
}

// Process image URL
text, err := processor.ProcessURL(ctx, imageURL)
```

**Canvas Analysis (canvasanalyzer)**
```go
import "go_backend/canvasanalyzer"

// Create fetcher
fetcher := canvasanalyzer.NewFetcher(canvusClient, canvasanalyzer.DefaultFetcherConfig(), logger)

// Fetch widgets with retry
widgets, err := fetcher.FetchWithRetry(ctx)
```

**Local LLM Inference (llamaruntime)**
```go
import "go_backend/llamaruntime"

// Create LLM client
client, err := llamaruntime.NewClient(modelPath, logger, llamaruntime.DefaultClientConfig())
if err != nil {
    return err
}
defer client.Close()

// Generate text
response, err := client.Generate(ctx, prompt, llamaruntime.GenerationParams{
    MaxTokens: 512,
    Temperature: 0.7,
})
```

**Web Dashboard (webui)**
```go
import "go_backend/webui"

// Create and start dashboard server
server := webui.NewServer(config, logger, metricsStore)
go server.Start(ctx)

// Health monitoring
healthMonitor := webui.NewHealthMonitor(canvasClient, metricsStore, logger)
healthMonitor.StartPeriodicChecks(ctx, 30*time.Second)
```

**Database Operations (db)**
```go
import "go_backend/db"

// Open database
repository, err := db.NewRepository(dbPath, logger)
if err != nil {
    return err
}
defer repository.Close()

// Run migrations
if err := repository.RunMigrations(); err != nil {
    return err
}

// Store metrics
err = repository.StoreMetric(ctx, metric)
```

**GPU Metrics (metrics)**
```go
import "go_backend/metrics"

// Create GPU collector
collector := metrics.NewGPUCollector(logger)

// Collect current metrics
gpuMetrics, err := collector.Collect()
if err != nil {
    return err
}

fmt.Printf("GPU Usage: %.1f%%, VRAM: %d MB\n",
    gpuMetrics.Utilization, gpuMetrics.MemoryUsed)
```

### Adding a New AI Feature

1. **Create atom functions** for data extraction/transformation
2. **Create molecule** for AI client interaction
3. **Create organism** handler in handlers.go (or new package if substantial)
4. **Wire into Monitor** in `monitorcanvus.go` handleUpdate logic
5. **Add tests** for each level
6. **Update .env.example** with new configuration variables

### Adding a New API Endpoint

1. **Add method to canvusapi.Client** (organism)
2. **Follow existing patterns**: `Request()` for JSON, `uploadFile()` for multipart
3. **Handle errors** with APIError type
4. **Write integration test** in tests/
5. **Document in code** with godoc comments

### Debugging Streaming Issues

- Check `app.log` for detailed operation logs
- Monitor uses long-polling with `subscribe=true` parameter
- Connection failures auto-retry with 5-second delay
- Widget state tracked in `Monitor.widgets` map (thread-safe)

## Security Considerations

- Never commit `.env` file (included in `.gitignore`)
- `ALLOW_SELF_SIGNED_CERTS=true` is for development only, logs warnings
- All API keys should be environment variables
- Input validation on all AI prompts and file uploads
- File size limits enforced (`MAX_FILE_SIZE`)

## Performance Notes

- Concurrent processing limited by `MAX_CONCURRENT` (default: 5)
- PDF chunking for large files (`PDF_CHUNK_SIZE_TOKENS`, `PDF_MAX_CHUNKS_TOKENS`)
- Downloads cleaned up after processing
- Widget state cached in memory for performance
- Retry logic with exponential backoff prevents API overload

## Logging

- Console output with color coding (`github.com/fatih/color`)
- File logging to `app.log` with timestamps and source info
- Dual output via `logging.LogHandler()`
- Request/response logging for debugging API issues
