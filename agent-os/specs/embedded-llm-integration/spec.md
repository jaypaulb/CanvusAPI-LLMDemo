# Specification: Embedded LLM Integration (Phase 2)

## Document Metadata

- **Phase**: 2 - Embedded LLM Integration
- **Status**: Draft
- **Version**: 1.0
- **Last Updated**: 2025-12-16
- **Owner**: Development Team
- **Related Roadmap Items**: 8, 9, 10, 11, 12

## Overview

### Purpose

Phase 2 replaces the current OpenAI API dependency with an embedded, fully local LLM inference solution using llama.cpp and the Bunny v1.1 Llama-3-8B-V multimodal model. This transformation enables CanvusLocalLLM to deliver on its core value proposition: zero-configuration, batteries-included AI processing that runs entirely on local NVIDIA RTX hardware with complete data privacy.

### Goals

1. **Eliminate Cloud Dependencies**: Remove all OpenAI API calls and requirements, enabling pure local operation
2. **Embed Inference Engine**: Integrate llama.cpp via CGo for in-process, CUDA-accelerated inference
3. **Enable Multimodal AI**: Leverage Bunny v1.1's vision capabilities for text and image understanding
4. **Simplify Configuration**: Remove need for OpenAI API keys and model selection, reducing .env to only Canvus credentials
5. **Maintain Performance**: Achieve 20+ tokens/second inference on RTX 3060 or better
6. **Ensure Reliability**: Implement health monitoring, automatic recovery, and graceful error handling

### Scope

**In Scope for Phase 2**:
- llama.cpp CGo integration with CUDA support
- Cross-platform builds (Windows MSVC + CUDA, Linux GCC + CUDA)
- Bunny v1.1 model loading and configuration
- Text generation for notes, PDFs, canvas analysis
- Vision inference for image analysis
- GPU memory monitoring and health checks
- Automatic recovery from inference failures
- Replace all OpenAI API calls in existing handlers

**Out of Scope for Phase 2**:
- Image generation with stable-diffusion.cpp (deferred to Phase 3)
- Web UI monitoring dashboard (deferred to Phase 5)
- Model switching or selection UI (Bunny is hardcoded)
- CPU fallback (CUDA required)
- macOS support (no CUDA on macOS)

### Success Metrics

- **Configuration Simplification**: OPENAI_API_KEY removed from .env, only Canvus credentials required
- **Performance**: ≥20 tokens/second on RTX 3060, ≥40 tokens/second on RTX 4070
- **Reliability**: 24-hour continuous operation without crashes or degradation
- **Latency**: First token <500ms for text, <1s for vision
- **Memory**: GPU VRAM usage ≤6GB for Q4_K_M quantization
- **Zero External Calls**: All inference happens locally, verified via network monitoring

## Architecture

### System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         CanvusLocalLLM                          │
│                           (Go Process)                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────┐      ┌──────────────┐      ┌──────────────┐ │
│  │   main.go    │─────▶│   Monitor    │─────▶│  handlers.go │ │
│  │ (Page/Root)  │      │  (Organism)  │      │  (Organism)  │ │
│  └──────────────┘      └──────────────┘      └──────┬───────┘ │
│                                                      │         │
│                                ┌─────────────────────┘         │
│                                │                               │
│                                ▼                               │
│                    ┌───────────────────────┐                   │
│                    │  llamaruntime/        │                   │
│                    │  (Organism Package)   │                   │
│                    ├───────────────────────┤                   │
│                    │ • Client (Molecule)   │                   │
│                    │ • Context Pool        │                   │
│                    │ • CGo Bindings (Atom) │                   │
│                    └──────────┬────────────┘                   │
│                               │ CGo Boundary                   │
└───────────────────────────────┼────────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                   llama.cpp (C++ Library)                       │
│                     Loaded as .dll/.so                          │
├─────────────────────────────────────────────────────────────────┤
│  • Model Loading (GGUF format)                                  │
│  • Context Management                                           │
│  • Inference Engine                                             │
│  • Vision Processing (Bunny multimodal)                         │
│  • CUDA Acceleration                                            │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                      NVIDIA RTX GPU                             │
│                    (CUDA Compute 7.5+)                          │
├─────────────────────────────────────────────────────────────────┤
│  • Tensor Operations                                            │
│  • VRAM Management                                              │
│  • Parallel Processing                                          │
└─────────────────────────────────────────────────────────────────┘
```

### Package Structure

Following atomic design principles:

```
CanvusLocalLLM/
├── main.go                      # Page: Application entry point, lifecycle management
├── handlers.go                  # Organism: AI request handlers (notes, PDFs, images)
├── monitorcanvus.go            # Organism: Canvas monitoring service
├── core/                        # Molecules: Configuration and utilities
│   ├── config.go               # Configuration management
│   └── ai.go                   # Will be refactored to use llamaruntime
├── canvusapi/                   # Organism: Canvus API client
│   └── canvusapi.go
├── logging/                     # Atom: Logging utilities
│   └── logging.go
├── llamaruntime/                # NEW: Organism: llama.cpp integration
│   ├── client.go               # Molecule: High-level Go API
│   ├── context.go              # Molecule: Context pool management
│   ├── bindings.go             # Atom: CGo bindings to llama.cpp
│   ├── types.go                # Atom: Go types and constants
│   ├── errors.go               # Atom: Error types and handling
│   └── health.go               # Molecule: Health monitoring
├── lib/                         # Native libraries (bundled with installer)
│   ├── llama.dll               # Windows llama.cpp + CUDA
│   ├── libllama.so             # Linux llama.cpp + CUDA
│   └── ...                     # CUDA runtime dependencies
└── models/                      # AI models (bundled or downloaded)
    └── bunny-v1.1-llama-3-8b-v.gguf  # Bunny model (5-6GB)
```

### llamaruntime Package Design

**Atomic Hierarchy**:

1. **Atoms** (Pure, no dependencies):
   - `bindings.go`: CGo wrapper functions calling C API
   - `types.go`: Go structs, constants, enums
   - `errors.go`: Error types (`LlamaError`, `ErrModelNotFound`, etc.)

2. **Molecules** (Compose atoms):
   - `client.go`: `Client` type with methods `LoadModel()`, `Infer()`, `InferVision()`
   - `context.go`: `ContextPool` managing multiple inference contexts
   - `health.go`: Health check functions, GPU monitoring

3. **Organism** (Full feature module):
   - `llamaruntime/` package as a whole provides complete LLM inference capability

**Public API Example**:

```go
package llamaruntime

// Client provides high-level access to llama.cpp inference
type Client struct {
    modelPath   string
    contextPool *ContextPool
    config      Config
}

// Config holds llamaruntime configuration
type Config struct {
    ModelPath    string
    ContextSize  int
    BatchSize    int
    GPULayers    int // -1 for auto
    NumContexts  int // Pool size
    Temperature  float32
    TopP         float32
    RepeatPenalty float32
}

// NewClient creates a new llamaruntime client and loads the model
func NewClient(config Config) (*Client, error)

// Infer generates text from a prompt
func (c *Client) Infer(ctx context.Context, prompt string, maxTokens int) (string, error)

// InferVision generates text from prompt + image
func (c *Client) InferVision(ctx context.Context, prompt string, imageData []byte, maxTokens int) (string, error)

// GetGPUMemoryUsage returns current GPU VRAM usage in bytes
func (c *Client) GetGPUMemoryUsage() (int64, error)

// HealthCheck performs a quick inference to verify model is working
func (c *Client) HealthCheck(ctx context.Context) error

// Close cleans up resources (model, contexts, GPU memory)
func (c *Client) Close() error
```

### CGo Integration Pattern

**Memory Management Strategy**:

1. **C Memory Lifecycle**:
   - Allocate in Go using `C.malloc()` or let C allocate
   - Free in Go using `C.free()` via defer or finalizers
   - Never mix Go and C memory allocators

2. **Pointer Safety**:
   - Go pointers passed to C must be pinned (use `runtime.Pinner` in Go 1.21+)
   - C pointers stored in Go as `unsafe.Pointer` or `uintptr`
   - No Go pointers stored in C-allocated memory

3. **Error Handling**:
   - C functions return error codes or NULL pointers
   - Go wrapper checks return values and translates to Go errors
   - Panics in CGo recovered at package boundary

4. **Thread Safety**:
   - C library calls protected by Go mutexes
   - Context pool ensures thread-safe access to llama contexts
   - No concurrent calls to same context handle

**Example CGo Binding**:

```go
// bindings.go
package llamaruntime

/*
#cgo CFLAGS: -I${SRCDIR}/../lib/llama.cpp
#cgo LDFLAGS: -L${SRCDIR}/../lib -lllama
#include <llama.h>
#include <stdlib.h>
*/
import "C"
import (
    "fmt"
    "runtime"
    "unsafe"
)

// llamaModel wraps C.llama_model pointer with automatic cleanup
type llamaModel struct {
    ptr C.llama_model
}

// loadModel loads a GGUF model from disk
func loadModel(path string) (*llamaModel, error) {
    cPath := C.CString(path)
    defer C.free(unsafe.Pointer(cPath))

    params := C.llama_model_default_params()
    params.n_gpu_layers = -1 // Offload all layers to GPU

    model := C.llama_load_model_from_file(cPath, params)
    if model == nil {
        return nil, fmt.Errorf("failed to load model from %s", path)
    }

    m := &llamaModel{ptr: model}
    runtime.SetFinalizer(m, func(m *llamaModel) {
        C.llama_free_model(m.ptr)
    })

    return m, nil
}

// ... more bindings
```

### Data Flow

**Text Generation Flow** (Notes, PDFs, Canvas):

```
1. User Action (Canvus)
   ↓
2. Monitor detects change (monitorcanvus.go)
   ↓
3. Handler extracts prompt (handlers.go)
   ↓
4. Handler calls llamaruntime.Client.Infer()
   ↓
5. Client acquires context from pool
   ↓
6. Client calls CGo bindings
   ↓
7. llama.cpp runs inference on GPU
   ↓
8. Response text returned through CGo
   ↓
9. Client releases context to pool
   ↓
10. Handler creates note widget with response
    ↓
11. Monitor pushes note to Canvus API
```

**Vision Inference Flow** (Image Analysis):

```
1. User uploads image to canvas
   ↓
2. Monitor detects new image widget
   ↓
3. Handler downloads image via Canvus API
   ↓
4. Handler calls llamaruntime.Client.InferVision(prompt, imageData)
   ↓
5. Client acquires context from pool
   ↓
6. Client converts image to llama.cpp format
   ↓
7. Client calls CGo bindings with image + prompt
   ↓
8. llama.cpp runs multimodal inference (Bunny vision)
   ↓
9. Description text returned through CGo
   ↓
10. Client releases context to pool
    ↓
11. Handler creates note widget with description
    ↓
12. Monitor pushes note to Canvus API
```

## Detailed Design

### Component 1: llamaruntime/bindings.go (Atom)

**Responsibility**: Thin CGo wrapper around llama.cpp C API

**Functions**:
- `llamaInit()`: Initialize llama.cpp library
- `loadModel(path string) (*llamaModel, error)`: Load GGUF model
- `createContext(model *llamaModel, params ContextParams) (*llamaContext, error)`: Create inference context
- `inferText(ctx *llamaContext, prompt string, maxTokens int) (string, error)`: Run text inference
- `inferVision(ctx *llamaContext, prompt string, image []byte, maxTokens int) (string, error)`: Run multimodal inference
- `freeContext(ctx *llamaContext)`: Release inference context
- `freeModel(model *llamaModel)`: Release model
- `getGPUMemory() (used int64, total int64, error)`: Query GPU VRAM usage

**Error Handling**:
- All C functions checked for NULL returns or error codes
- Translated to Go errors with descriptive messages
- Panics recovered via `defer recover()` at package boundary

**Memory Management**:
- Go finalizers on `llamaModel` and `llamaContext` to ensure cleanup
- Explicit `Close()` methods for deterministic cleanup
- No Go pointers passed to C that outlive the C call
- C strings created with `C.CString()` and freed with `C.free()`

### Component 2: llamaruntime/types.go (Atom)

**Types**:

```go
// Config holds llamaruntime configuration
type Config struct {
    ModelPath     string
    ContextSize   int
    BatchSize     int
    GPULayers     int     // -1 = auto, 0 = CPU only, N = offload N layers
    NumContexts   int     // Context pool size
    Temperature   float32
    TopP          float32
    RepeatPenalty float32
}

// InferenceStats tracks performance metrics
type InferenceStats struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
    InferenceTime    time.Duration
    TokensPerSecond  float64
    FirstTokenTime   time.Duration
}

// GPUInfo describes detected GPU
type GPUInfo struct {
    Name         string
    VRAMTotal    int64 // Bytes
    VRAMFree     int64
    CUDAVersion  string
    ComputeCap   string // e.g., "8.6" for RTX 4070
    DriverVersion string
}
```

**Constants**:

```go
const (
    DefaultContextSize   = 4096
    DefaultBatchSize     = 512
    DefaultGPULayers     = -1   // Auto
    DefaultNumContexts   = 5
    DefaultTemperature   = 0.7
    DefaultTopP          = 0.9
    DefaultRepeatPenalty = 1.1
)
```

### Component 3: llamaruntime/errors.go (Atom)

**Error Types**:

```go
// LlamaError represents an error from llama.cpp
type LlamaError struct {
    Op      string // Operation that failed
    Code    int    // Error code from C
    Message string
    Err     error  // Wrapped error
}

func (e *LlamaError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("llama.cpp %s: %s: %w", e.Op, e.Message, e.Err)
    }
    return fmt.Sprintf("llama.cpp %s: %s", e.Op, e.Message)
}

func (e *LlamaError) Unwrap() error {
    return e.Err
}

// Sentinel errors
var (
    ErrModelNotFound     = errors.New("model file not found")
    ErrModelLoadFailed   = errors.New("failed to load model")
    ErrContextCreateFailed = errors.New("failed to create inference context")
    ErrInferenceFailed   = errors.New("inference failed")
    ErrGPUNotAvailable   = errors.New("CUDA GPU not available")
    ErrInsufficientVRAM  = errors.New("insufficient GPU VRAM")
    ErrInvalidImage      = errors.New("invalid or unsupported image format")
    ErrTimeout           = errors.New("inference timeout")
)
```

### Component 4: llamaruntime/context.go (Molecule)

**Responsibility**: Manage pool of inference contexts for concurrency

**ContextPool Design**:

```go
// ContextPool manages a pool of reusable inference contexts
type ContextPool struct {
    model      *llamaModel
    config     Config
    contexts   chan *llamaContext
    numContexts int
    mu         sync.Mutex
    closed     bool
}

// NewContextPool creates a pool of inference contexts
func NewContextPool(model *llamaModel, config Config) (*ContextPool, error) {
    pool := &ContextPool{
        model:      model,
        config:     config,
        contexts:   make(chan *llamaContext, config.NumContexts),
        numContexts: config.NumContexts,
    }

    // Pre-create all contexts
    for i := 0; i < config.NumContexts; i++ {
        ctx, err := createContext(model, config)
        if err != nil {
            pool.Close()
            return nil, fmt.Errorf("failed to create context %d: %w", i, err)
        }
        pool.contexts <- ctx
    }

    return pool, nil
}

// Acquire gets a context from the pool (blocks if all busy)
func (p *ContextPool) Acquire(ctx context.Context) (*llamaContext, error) {
    select {
    case c := <-p.contexts:
        return c, nil
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

// Release returns a context to the pool
func (p *ContextPool) Release(ctx *llamaContext) {
    select {
    case p.contexts <- ctx:
    default:
        // Pool closed or full, free context
        freeContext(ctx)
    }
}

// Close closes all contexts in pool
func (p *ContextPool) Close() error {
    p.mu.Lock()
    defer p.mu.Unlock()

    if p.closed {
        return nil
    }
    p.closed = true

    close(p.contexts)
    for ctx := range p.contexts {
        freeContext(ctx)
    }

    return nil
}
```

**Rationale**: Reusing contexts avoids expensive context creation/destruction for each request. Pool size matches MAX_CONCURRENT config for optimal throughput.

### Component 5: llamaruntime/client.go (Molecule)

**Responsibility**: High-level Go API for llama.cpp inference

**Implementation**:

```go
// Client provides access to llama.cpp inference
type Client struct {
    model       *llamaModel
    contextPool *ContextPool
    config      Config
    gpuInfo     GPUInfo
    mu          sync.RWMutex
    closed      bool
}

// NewClient creates and initializes a llamaruntime client
func NewClient(config Config) (*Client, error) {
    // Validate config
    if config.ModelPath == "" {
        return nil, fmt.Errorf("ModelPath is required")
    }
    if _, err := os.Stat(config.ModelPath); os.IsNotExist(err) {
        return nil, ErrModelNotFound
    }

    // Set defaults
    if config.ContextSize == 0 {
        config.ContextSize = DefaultContextSize
    }
    if config.BatchSize == 0 {
        config.BatchSize = DefaultBatchSize
    }
    if config.NumContexts == 0 {
        config.NumContexts = DefaultNumContexts
    }
    if config.Temperature == 0 {
        config.Temperature = DefaultTemperature
    }
    if config.TopP == 0 {
        config.TopP = DefaultTopP
    }
    if config.RepeatPenalty == 0 {
        config.RepeatPenalty = DefaultRepeatPenalty
    }

    // Initialize llama.cpp
    llamaInit()

    // Detect GPU
    gpuInfo, err := detectGPU()
    if err != nil {
        return nil, fmt.Errorf("GPU detection failed: %w", err)
    }
    log.Printf("[llama] Detected GPU: %s, VRAM: %dGB", gpuInfo.Name, gpuInfo.VRAMTotal/(1024*1024*1024))

    // Load model
    log.Printf("[llama] Loading model from %s...", config.ModelPath)
    model, err := loadModel(config.ModelPath)
    if err != nil {
        return nil, fmt.Errorf("failed to load model: %w", err)
    }
    log.Printf("[llama] Model loaded successfully")

    // Create context pool
    log.Printf("[llama] Creating context pool with %d contexts...", config.NumContexts)
    pool, err := NewContextPool(model, config)
    if err != nil {
        freeModel(model)
        return nil, fmt.Errorf("failed to create context pool: %w", err)
    }
    log.Printf("[llama] Context pool created")

    client := &Client{
        model:       model,
        contextPool: pool,
        config:      config,
        gpuInfo:     gpuInfo,
    }

    // Run startup health check
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := client.HealthCheck(ctx); err != nil {
        client.Close()
        return nil, fmt.Errorf("startup health check failed: %w", err)
    }
    log.Printf("[llama] Health check passed")

    return client, nil
}

// Infer generates text from a prompt
func (c *Client) Infer(ctx context.Context, prompt string, maxTokens int) (string, error) {
    c.mu.RLock()
    if c.closed {
        c.mu.RUnlock()
        return "", errors.New("client closed")
    }
    c.mu.RUnlock()

    // Acquire context from pool
    llamaCtx, err := c.contextPool.Acquire(ctx)
    if err != nil {
        return "", fmt.Errorf("failed to acquire context: %w", err)
    }
    defer c.contextPool.Release(llamaCtx)

    // Run inference
    start := time.Now()
    response, err := inferText(llamaCtx, prompt, maxTokens)
    if err != nil {
        return "", err
    }
    elapsed := time.Since(start)

    log.Printf("[llama] Inference completed in %v", elapsed)
    return response, nil
}

// InferVision generates text from prompt + image
func (c *Client) InferVision(ctx context.Context, prompt string, imageData []byte, maxTokens int) (string, error) {
    // Similar to Infer but calls inferVision()
    // ...
}

// GetGPUMemoryUsage returns current GPU VRAM usage
func (c *Client) GetGPUMemoryUsage() (int64, error) {
    used, total, err := getGPUMemory()
    if err != nil {
        return 0, err
    }
    return used, nil
}

// HealthCheck runs a quick inference to verify model is working
func (c *Client) HealthCheck(ctx context.Context) error {
    response, err := c.Infer(ctx, "Hello, how are you?", 20)
    if err != nil {
        return fmt.Errorf("health check inference failed: %w", err)
    }
    if len(response) == 0 {
        return errors.New("health check returned empty response")
    }
    return nil
}

// Close releases all resources
func (c *Client) Close() error {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.closed {
        return nil
    }
    c.closed = true

    if c.contextPool != nil {
        c.contextPool.Close()
    }
    if c.model != nil {
        freeModel(c.model)
    }

    log.Printf("[llama] Client closed")
    return nil
}
```

### Component 6: llamaruntime/health.go (Molecule)

**Responsibility**: GPU monitoring and health checks

**Functions**:

```go
// detectGPU detects NVIDIA GPU and returns info
func detectGPU() (GPUInfo, error) {
    // Call llama.cpp GPU detection functions via CGo
    // Return GPU name, VRAM, CUDA version, compute capability
}

// MonitorGPUMemory periodically logs GPU VRAM usage
func MonitorGPUMemory(ctx context.Context, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            used, total, err := getGPUMemory()
            if err != nil {
                log.Printf("[llama] GPU memory query failed: %v", err)
                continue
            }

            usedGB := float64(used) / (1024 * 1024 * 1024)
            totalGB := float64(total) / (1024 * 1024 * 1024)
            usedPercent := float64(used) / float64(total) * 100

            log.Printf("[llama] GPU VRAM: %.2fGB / %.2fGB (%.1f%%)", usedGB, totalGB, usedPercent)

            if usedPercent > 90 {
                log.Printf("[llama] WARNING: GPU VRAM usage high (%.1f%%)", usedPercent)
            }
        }
    }
}

// PeriodicHealthCheck runs health checks at intervals
func PeriodicHealthCheck(ctx context.Context, client *Client, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
            err := client.HealthCheck(checkCtx)
            cancel()

            if err != nil {
                log.Printf("[llama] Health check FAILED: %v", err)
                // Could trigger automatic recovery here
            } else {
                log.Printf("[llama] Health check OK")
            }
        }
    }
}
```

### Integration: Replacing OpenAI Calls

**Current (Phase 1)**: `core/ai.go` uses OpenAI client

**Phase 2**: Replace with llamaruntime

**Before**:
```go
// core/ai.go
func TestAIResponse(ctx context.Context, cfg *Config, prompt string) (string, error) {
    client := createOpenAIClient(cfg)
    resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model: cfg.OpenAINoteModel,
        Messages: []openai.ChatCompletionMessage{
            {Role: openai.ChatMessageRoleUser, Content: prompt},
        },
        MaxTokens: int(cfg.NoteResponseTokens),
    })
    // ... handle response
}
```

**After**:
```go
// core/ai.go (refactored)
func TestAIResponse(ctx context.Context, cfg *Config, llamaClient *llamaruntime.Client, prompt string) (string, error) {
    response, err := llamaClient.Infer(ctx, prompt, int(cfg.NoteResponseTokens))
    if err != nil {
        return "", fmt.Errorf("failed to generate AI response: %w", err)
    }
    return response, nil
}
```

**Changes in main.go**:

```go
// main.go (Phase 2)
func main() {
    // Load config
    cfg, err := core.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Initialize llamaruntime
    llamaConfig := llamaruntime.Config{
        ModelPath:     getEnvOrDefault("MODEL_PATH", "models/bunny-v1.1-llama-3-8b-v.gguf"),
        ContextSize:   parseInt64Env("CONTEXT_SIZE", 4096),
        BatchSize:     parseInt64Env("BATCH_SIZE", 512),
        GPULayers:     parseInt64Env("GPU_LAYERS", -1),
        NumContexts:   parseInt64Env("NUM_CONTEXTS", 5),
        Temperature:   parseFloat32Env("TEMPERATURE", 0.7),
        TopP:          parseFloat32Env("TOP_P", 0.9),
        RepeatPenalty: parseFloat32Env("REPEAT_PENALTY", 1.1),
    }

    llamaClient, err := llamaruntime.NewClient(llamaConfig)
    if err != nil {
        log.Fatalf("Failed to initialize llamaruntime: %v", err)
    }
    defer llamaClient.Close()

    // Start GPU monitoring
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go llamaruntime.MonitorGPUMemory(ctx, 60*time.Second)
    go llamaruntime.PeriodicHealthCheck(ctx, llamaClient, 5*time.Minute)

    // Create Canvus client
    canvusClient := canvusapi.NewClient(cfg.CanvusServerURL, cfg.CanvasID, cfg.CanvusAPIKey)

    // Create monitor with llama client
    monitor := NewMonitor(canvusClient, cfg, llamaClient)

    // Start monitoring
    if err := monitor.Start(ctx); err != nil {
        log.Fatalf("Failed to start monitor: %v", err)
    }

    // Wait for interrupt
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    log.Println("Shutting down...")
    cancel()
    time.Sleep(2 * time.Second) // Allow graceful shutdown
}
```

**Changes in handlers.go**:

All AI calls in handlers now use `llamaClient.Infer()` or `llamaClient.InferVision()` instead of OpenAI API.

Example for note processing:

```go
// handlers.go (Phase 2)
func (m *Monitor) processNote(widget Widget, prompt string) error {
    ctx, cancel := context.WithTimeout(context.Background(), m.config.AITimeout)
    defer cancel()

    // Call llama instead of OpenAI
    response, err := m.llamaClient.Infer(ctx, prompt, int(m.config.NoteResponseTokens))
    if err != nil {
        log.Printf("Inference failed: %v", err)
        // Create error note
        errorNote := fmt.Sprintf("AI inference failed: %v", err)
        m.createResponseNote(widget, errorNote)
        return err
    }

    // Create response note
    m.createResponseNote(widget, response)
    return nil
}
```

## Cross-Platform Build Process

### Build Requirements

**Windows**:
- Visual Studio 2022 (Build Tools or Community Edition)
- CUDA Toolkit 12.x (12.3+ recommended)
- CMake 3.20+
- Go 1.21+
- PowerShell 5.1+

**Linux**:
- GCC 11+ or Clang 14+
- CUDA Toolkit 12.x (12.3+ recommended)
- CMake 3.20+
- Go 1.21+
- make, git

### Build Steps

**Step 1: Clone llama.cpp**

```bash
# Run once during initial setup
git clone https://github.com/ggerganov/llama.cpp.git
cd llama.cpp
git checkout <stable-tag>  # Use stable release, not master
```

**Step 2: Build llama.cpp with CUDA (Windows)**

```powershell
# build-llama-windows.ps1
cd llama.cpp
mkdir build -ErrorAction SilentlyContinue
cd build

# Configure with CMake
cmake .. -G "Visual Studio 17 2022" -A x64 `
    -DLLAMA_CUBLAS=ON `
    -DCMAKE_CUDA_COMPILER="C:/Program Files/NVIDIA GPU Computing Toolkit/CUDA/v12.3/bin/nvcc.exe" `
    -DCMAKE_BUILD_TYPE=Release

# Build
cmake --build . --config Release

# Copy outputs to project lib/ directory
Copy-Item Release/llama.dll ../../lib/
Copy-Item Release/llama.lib ../../lib/
```

**Step 3: Build llama.cpp with CUDA (Linux)**

```bash
# build-llama-linux.sh
cd llama.cpp
mkdir -p build
cd build

# Configure with CMake
cmake .. \
    -DLLAMA_CUBLAS=ON \
    -DCMAKE_CUDA_COMPILER=/usr/local/cuda/bin/nvcc \
    -DCMAKE_BUILD_TYPE=Release

# Build
make -j$(nproc)

# Copy outputs to project lib/ directory
cp libllama.so ../../lib/
```

**Step 4: Build Go Application**

```bash
# Set CGo flags to find llama.cpp headers and libraries
export CGO_CFLAGS="-I${PWD}/llama.cpp"
export CGO_LDFLAGS="-L${PWD}/lib -lllama"

# Enable CGo
export CGO_ENABLED=1

# Build (Windows)
go build -o CanvusLocalLLM.exe .

# Build (Linux)
go build -o canvuslocallm .
```

**Step 5: Verify CUDA Support**

```bash
# Windows
.\CanvusLocalLLM.exe --check-cuda

# Linux
./canvuslocallm --check-cuda

# Should output:
# [llama] CUDA available: Yes
# [llama] GPU: NVIDIA GeForce RTX 4070
# [llama] VRAM: 12GB
# [llama] CUDA version: 12.3
# [llama] Compute capability: 8.9
```

### Build Automation

Create build scripts for each platform that automate the above steps:

- `scripts/build-windows.ps1`: Full Windows build
- `scripts/build-linux.sh`: Full Linux build
- `scripts/verify-build.sh`: Verify libraries and dependencies

### Installer Integration

**Windows Installer (NSIS)**:

```nsis
# installer-windows.nsi
Section "Main Application"
    SetOutPath "$INSTDIR"
    File "CanvusLocalLLM.exe"
    File "lib\llama.dll"
    File "lib\cublas64_12.dll"  # CUDA runtime dependencies
    File "lib\cublasLt64_12.dll"
    File "models\bunny-v1.1-llama-3-8b-v.gguf"
    File ".env.example"
    File "README.txt"

    # Set PATH for DLL loading
    EnVar::AddValue "PATH" "$INSTDIR\lib"
SectionEnd
```

**Linux Debian Package**:

```
# debian package structure
canvuslocallm_2.0.0_amd64/
├── DEBIAN/
│   ├── control
│   └── postinst
└── opt/
    └── canvuslocallm/
        ├── canvuslocallm (binary)
        ├── lib/
        │   ├── libllama.so
        │   └── (CUDA runtime .so files)
        ├── models/
        │   └── bunny-v1.1-llama-3-8b-v.gguf
        ├── .env.example
        └── README.txt
```

**postinst script**:
```bash
#!/bin/bash
# Set LD_LIBRARY_PATH for library loading
echo "export LD_LIBRARY_PATH=/opt/canvuslocallm/lib:\$LD_LIBRARY_PATH" >> /etc/profile.d/canvuslocallm.sh
ldconfig /opt/canvuslocallm/lib
```

## Bunny v1.1 Model Integration

### Model Acquisition

**Source**: https://huggingface.co/BAAI/Bunny-v1_1-Llama-3-8B-V

**Quantization**: Use pre-quantized GGUF models or quantize yourself using llama.cpp

**Recommended Quantization**:
- Q4_K_M: 5GB, good quality/performance balance, fits 6GB VRAM
- Q5_K_M: 6GB, better quality, fits 8GB VRAM
- Q6_K: 7GB, near-original quality, requires 10GB+ VRAM

**Download Command**:
```bash
# Using huggingface-cli
huggingface-cli download BAAI/Bunny-v1_1-Llama-3-8B-V bunny-v1.1-llama-3-8b-v-q4_k_m.gguf --local-dir models/
```

**Installer Bundling**:
- Option A: Bundle model with installer (adds 5-6GB to installer size)
- Option B: Download on first run with progress indicator (recommended for distribution)

### Model Configuration

**llama.cpp Context Parameters** (optimized for Bunny + RTX):

```go
llamaParams := C.llama_context_default_params()
llamaParams.n_ctx = C.int(4096)        // Context window
llamaParams.n_batch = C.int(512)       // Batch size for prompt processing
llamaParams.n_threads = C.int(4)       // CPU threads (minimal, GPU does heavy lifting)
llamaParams.n_gpu_layers = C.int(-1)   // -1 = offload all layers to GPU

// Inference parameters
llamaParams.temperature = C.float(0.7)
llamaParams.top_p = C.float(0.9)
llamaParams.repeat_penalty = C.float(1.1)
llamaParams.mirostat = C.int(0)        // 0 = disabled, 1/2 = enabled
```

**Tuning for Different RTX GPUs**:

| GPU Model | VRAM | Recommended Quantization | Batch Size | Expected Speed |
|-----------|------|--------------------------|------------|----------------|
| RTX 3060  | 12GB | Q4_K_M                   | 512        | 20-25 tok/s    |
| RTX 3070  | 8GB  | Q4_K_M                   | 512        | 30-35 tok/s    |
| RTX 4070  | 12GB | Q5_K_M                   | 512        | 40-50 tok/s    |
| RTX 4080  | 16GB | Q6_K                     | 1024       | 60-70 tok/s    |
| RTX 4090  | 24GB | Q6_K                     | 1024       | 80-100 tok/s   |

### Multimodal (Vision) Usage

**Image Preprocessing**:
- Resize to model's expected resolution (likely 336x336 or 448x448 for Bunny)
- Convert to RGB if grayscale
- Normalize pixel values (typically [0, 1] or [-1, 1])
- Encode as raw bytes or base64 depending on llama.cpp API

**Vision Inference API**:

```go
// InferVision combines image and text prompt
func (c *Client) InferVision(ctx context.Context, prompt string, imageData []byte, maxTokens int) (string, error) {
    // Preprocess image
    processed, err := preprocessImage(imageData)
    if err != nil {
        return "", fmt.Errorf("image preprocessing failed: %w", err)
    }

    // Acquire context
    llamaCtx, err := c.contextPool.Acquire(ctx)
    if err != nil {
        return "", err
    }
    defer c.contextPool.Release(llamaCtx)

    // Call CGo binding with image + prompt
    response, err := inferVision(llamaCtx, prompt, processed, maxTokens)
    if err != nil {
        return "", err
    }

    return response, nil
}
```

**Example Prompts for Vision**:
- "Describe this image in detail."
- "What objects are visible in this image?"
- "Analyze the chart and explain the trends."
- "Extract text from this screenshot." (OCR capability)

### Startup Validation

**On Application Start**:

1. Check model file exists at `models/bunny-v1.1-llama-3-8b-v.gguf`
2. Load model with llama.cpp
3. Log model metadata:
   - Model size (GB)
   - Quantization type (Q4_K_M, Q5_K_M, etc.)
   - Context size
   - Vocabulary size
4. Run test inference: "Hello, how are you?" (expect coherent response)
5. Log inference stats:
   - Tokens generated
   - Inference time
   - Tokens per second
6. If any step fails, log detailed error and exit with clear message

**Example Startup Log**:
```
[llama] Detected GPU: NVIDIA GeForce RTX 4070, VRAM: 12GB, CUDA: 12.3, Driver: 537.42
[llama] Loading model from models/bunny-v1.1-llama-3-8b-v.gguf...
[llama] Model loaded: 5.2GB, Q4_K_M quantization, 4096 context size
[llama] Creating context pool with 5 contexts...
[llama] Context pool created
[llama] Running startup health check...
[llama] Health check prompt: "Hello, how are you?"
[llama] Health check response: "Hello! I'm doing well, thank you for asking. How can I assist you today?"
[llama] Health check inference: 18 tokens in 0.82s (21.95 tok/s)
[llama] Health check passed
[llama] llamaruntime initialization complete
```

## Health Monitoring and Recovery

### Startup Diagnostics

**On Startup, Log**:
1. CUDA availability and version
2. GPU detection (name, VRAM, compute capability, driver version)
3. Model loading (file path, size, quantization type)
4. Context pool creation
5. Test inference (prompt, response, speed)

**Fail Fast If**:
- CUDA not available
- No compatible GPU found
- Model file not found
- Model loading fails
- Context creation fails
- Test inference fails

### Runtime Monitoring

**GPU Memory Monitoring**:
- Check VRAM usage every 60 seconds
- Log: `GPU VRAM: 4.2GB / 12GB (35%)`
- Warn if >90%: `WARNING: GPU VRAM usage high (92%)`

**Periodic Health Checks**:
- Run simple inference every 5 minutes
- Prompt: "Test" or "Hello"
- Expected: Non-empty response in <5 seconds
- Log success/failure

**Inference Metrics**:
- Track per-request: prompt tokens, completion tokens, inference time, tok/s
- Aggregate: p50/p95/p99 latency, success rate, error rate
- Expose via internal API for future web UI (Phase 5)

### Automatic Recovery

**On Inference Failure**:

1. **Retry**: Attempt inference up to 3 times with exponential backoff
2. **Context Reset**: If retries fail, try releasing and re-acquiring context
3. **Model Reload**: If persistent failure, try reloading model and context pool
4. **Degraded Mode**: If reload fails, enter degraded mode (refuse new requests, return errors)
5. **Alert**: Log critical error, expose status via health check endpoint

**Recovery Scenarios**:

| Error Type | Recovery Action | Retry Count |
|------------|-----------------|-------------|
| Timeout | Retry with longer timeout | 3 |
| CUDA OOM | Release context, reduce batch size, retry | 2 |
| Corrupted Context | Release and re-acquire context | 1 |
| Model Unresponsive | Reload model and context pool | 1 |
| Unrecoverable | Enter degraded mode, require restart | 0 |

### Graceful Shutdown

**On SIGINT/SIGTERM**:

1. Cancel context (stop accepting new requests)
2. Wait for in-flight inferences to complete (max 30 seconds)
3. Close context pool (frees all contexts)
4. Free model
5. Log shutdown complete

**Example Shutdown Log**:
```
[main] Received interrupt signal, shutting down...
[monitor] Stopping canvas monitoring...
[monitor] Waiting for in-flight requests to complete...
[monitor] All requests completed
[llama] Closing context pool...
[llama] Freeing model...
[llama] Client closed
[main] Shutdown complete
```

## Configuration Changes

### Removed Environment Variables

Phase 1 variables no longer needed in Phase 2:

- `OPENAI_API_KEY` - No longer required
- `OPENAI_NOTE_MODEL` - Bunny is hardcoded
- `OPENAI_CANVAS_MODEL` - Bunny is hardcoded
- `OPENAI_PDF_MODEL` - Bunny is hardcoded
- `BASE_LLM_URL` - No external API
- `TEXT_LLM_URL` - No external API
- `IMAGE_LLM_URL` - Deferred to Phase 3 (stable-diffusion.cpp)
- `AZURE_OPENAI_ENDPOINT` - No cloud provider
- `AZURE_OPENAI_DEPLOYMENT` - No cloud provider
- `AZURE_OPENAI_API_VERSION` - No cloud provider

### New Environment Variables

Phase 2 llamaruntime configuration:

```bash
# Model Configuration
MODEL_PATH=models/bunny-v1.1-llama-3-8b-v.gguf  # Path to GGUF model

# Inference Parameters
CONTEXT_SIZE=4096           # Context window (tokens)
BATCH_SIZE=512              # Batch size for processing
GPU_LAYERS=-1               # Number of layers to offload to GPU (-1 = all)
NUM_CONTEXTS=5              # Context pool size (concurrent requests)

# Generation Parameters (optional, defaults shown)
TEMPERATURE=0.7             # Sampling temperature
TOP_P=0.9                   # Nucleus sampling
REPEAT_PENALTY=1.1          # Repetition penalty

# Monitoring (optional)
GPU_MONITOR_INTERVAL=60     # GPU memory check interval (seconds)
HEALTH_CHECK_INTERVAL=300   # Health check interval (seconds)
```

### Retained Environment Variables

Still required from Phase 1:

```bash
# Canvus Integration (required)
CANVUS_SERVER=https://canvus.example.com
CANVAS_ID=your-canvas-id
CANVUS_API_KEY=your-api-key

# Optional Canvus Config
CANVAS_NAME=My Canvas
ALLOW_SELF_SIGNED_CERTS=false

# Token Limits (still used for max_tokens in inference)
PDF_PRECIS_TOKENS=1000
CANVAS_PRECIS_TOKENS=600
NOTE_RESPONSE_TOKENS=400
IMAGE_ANALYSIS_TOKENS=300

# Processing Config
MAX_CONCURRENT=5
AI_TIMEOUT=60
PROCESSING_TIMEOUT=300
MAX_RETRIES=3
RETRY_DELAY=1
MAX_FILE_SIZE=52428800
DOWNLOADS_DIR=./downloads

# Web UI
WEBUI_PWD=your-password
PORT=3000
```

### Updated .env.example

```bash
# CanvusLocalLLM Configuration (Phase 2: Embedded LLM)

# ============================================================
# CANVUS INTEGRATION (Required)
# ============================================================
CANVUS_SERVER=https://canvus.example.com
CANVAS_ID=your-canvas-id-here
CANVUS_API_KEY=your-api-key-here

# ============================================================
# EMBEDDED LLM CONFIGURATION
# ============================================================
# Path to Bunny v1.1 GGUF model (default: models/bunny-v1.1-llama-3-8b-v.gguf)
MODEL_PATH=models/bunny-v1.1-llama-3-8b-v.gguf

# Context size (default: 4096)
CONTEXT_SIZE=4096

# Batch size for processing (default: 512)
BATCH_SIZE=512

# GPU layers to offload (-1 = all, 0 = CPU only, N = specific count)
GPU_LAYERS=-1

# Number of concurrent inference contexts (default: 5)
NUM_CONTEXTS=5

# ============================================================
# GENERATION PARAMETERS (Optional)
# ============================================================
TEMPERATURE=0.7
TOP_P=0.9
REPEAT_PENALTY=1.1

# ============================================================
# TOKEN LIMITS
# ============================================================
PDF_PRECIS_TOKENS=1000
CANVAS_PRECIS_TOKENS=600
NOTE_RESPONSE_TOKENS=400
IMAGE_ANALYSIS_TOKENS=300

# ============================================================
# PROCESSING CONFIGURATION
# ============================================================
MAX_CONCURRENT=5
AI_TIMEOUT=60
PROCESSING_TIMEOUT=300
MAX_RETRIES=3
MAX_FILE_SIZE=52428800

# ============================================================
# WEB UI (Optional)
# ============================================================
WEBUI_PWD=change-this-password
PORT=3000
```

## Testing Strategy

### Unit Tests

**llamaruntime/bindings_test.go**:
- Test CGo bindings with mocked C functions
- Test error handling from C layer
- Test memory management (allocations and frees)
- No GPU required (mocked)

**llamaruntime/context_test.go**:
- Test context pool acquire/release
- Test concurrent access to pool
- Test pool closure and cleanup
- Mock actual llama contexts

**llamaruntime/client_test.go**:
- Test Client creation and initialization
- Test Infer() with various prompts
- Test InferVision() with test images
- Test error handling and recovery
- Mock llamaruntime backend

### Integration Tests

**tests/llama_integration_test.go**:

```go
func TestLlamaIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Requires actual model file and GPU
    config := llamaruntime.Config{
        ModelPath:   "models/bunny-v1.1-llama-3-8b-v.gguf",
        ContextSize: 4096,
        BatchSize:   512,
        GPULayers:   -1,
        NumContexts: 2,
    }

    client, err := llamaruntime.NewClient(config)
    if err != nil {
        t.Fatalf("Failed to create client: %v", err)
    }
    defer client.Close()

    tests := []struct {
        name      string
        prompt    string
        maxTokens int
        wantLen   int // Minimum response length
    }{
        {"simple", "Hello", 20, 5},
        {"question", "What is 2+2?", 50, 3},
        {"complex", "Explain quantum computing in simple terms", 200, 50},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
            defer cancel()

            response, err := client.Infer(ctx, tt.prompt, tt.maxTokens)
            if err != nil {
                t.Errorf("Infer() error = %v", err)
                return
            }

            if len(response) < tt.wantLen {
                t.Errorf("Infer() response too short, got %d chars, want >= %d", len(response), tt.wantLen)
            }

            t.Logf("Prompt: %s", tt.prompt)
            t.Logf("Response: %s", response)
        })
    }
}

func TestLlamaVisionIntegration(t *testing.T) {
    // Test vision inference with sample images
    // ...
}

func TestLlamaConcurrency(t *testing.T) {
    // Test multiple concurrent inferences
    // ...
}
```

### Performance Benchmarks

**tests/llama_bench_test.go**:

```go
func BenchmarkInfer(b *testing.B) {
    client := setupClient(b)
    defer client.Close()

    ctx := context.Background()
    prompt := "Explain artificial intelligence"
    maxTokens := 100

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := client.Infer(ctx, prompt, maxTokens)
        if err != nil {
            b.Fatalf("Infer failed: %v", err)
        }
    }
}

func BenchmarkInferConcurrent(b *testing.B) {
    client := setupClient(b)
    defer client.Close()

    ctx := context.Background()
    prompt := "Explain artificial intelligence"
    maxTokens := 100

    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, err := client.Infer(ctx, prompt, maxTokens)
            if err != nil {
                b.Fatalf("Infer failed: %v", err)
            }
        }
    })
}
```

**Run Benchmarks**:
```bash
go test -bench=. -benchmem -benchtime=30s ./tests/
```

### Platform Testing

**Test Matrix**:

| Platform | OS Version | GPU | CUDA | Compiler | Status |
|----------|------------|-----|------|----------|--------|
| Windows | 11 | RTX 4070 | 12.3 | MSVC 2022 | ✓ |
| Windows | 10 | RTX 3060 | 12.3 | MSVC 2022 | ✓ |
| Linux | Ubuntu 22.04 | RTX 4070 | 12.3 | GCC 11 | ✓ |
| Linux | Ubuntu 20.04 | RTX 3060 | 12.1 | GCC 11 | ✓ |
| Linux | Fedora 38 | RTX 4080 | 12.3 | GCC 13 | ✓ |

**Test on Each Platform**:
1. Build llama.cpp from source
2. Build Go application with CGo
3. Run unit tests
4. Run integration tests
5. Run performance benchmarks
6. Test installer package
7. Test first-run experience
8. Test 24-hour continuous operation

### Acceptance Tests

**End-to-End Scenarios**:

1. **Note Processing**:
   - Create note with `{{ Explain quantum computing }}`
   - Verify AI response note appears within 5 seconds
   - Verify response is coherent and relevant

2. **PDF Analysis**:
   - Upload PDF to canvas
   - Trigger analysis via custom menu
   - Verify summary note appears within 30 seconds
   - Verify summary captures key points

3. **Image Analysis**:
   - Upload image to canvas
   - Trigger vision analysis
   - Verify description note appears within 10 seconds
   - Verify description is accurate

4. **Canvas Analysis**:
   - Create canvas with multiple widgets (notes, images, PDFs)
   - Request canvas overview
   - Verify overview captures spatial layout and content

5. **Concurrency**:
   - Trigger 5 AI operations simultaneously
   - Verify all complete successfully
   - Verify no crashes or errors

6. **Long-Running**:
   - Run application for 24 hours
   - Perform periodic AI operations (every 5 minutes)
   - Verify no memory leaks
   - Verify no degradation in performance

7. **Recovery**:
   - Simulate GPU error (disconnect/reconnect)
   - Verify automatic recovery
   - Verify subsequent operations succeed

8. **Zero External Calls**:
   - Monitor network traffic during operation
   - Verify no connections to OpenAI or other cloud services
   - All inference happens locally

## Documentation Requirements

### README Updates

Update main README.md:
- Remove OpenAI API key requirement
- Add system requirements (NVIDIA RTX GPU, CUDA 12.x)
- Update configuration section (new .env variables)
- Add troubleshooting for common issues (GPU not detected, model not found)

### New Documentation Files

**docs/build-guide.md**:
- Prerequisites for each platform
- Step-by-step build instructions
- CMake configuration details
- Troubleshooting build errors

**docs/llamaruntime.md**:
- Architecture overview
- Public API reference
- CGo integration details
- Performance tuning guide
- Memory management best practices

**docs/bunny-model.md**:
- Model description and capabilities
- Quantization options and trade-offs
- Prompt engineering tips
- Vision inference examples
- Performance expectations per GPU

**docs/troubleshooting.md**:
- CUDA not found
- GPU not detected
- Model loading fails
- Inference timeout
- Memory errors
- Build errors (Windows, Linux)

### Code Documentation

- Godoc comments on all exported types and functions
- Examples in doc comments
- Package-level documentation explaining purpose and usage
- Inline comments for complex CGo logic

## Risk Mitigation Strategies

### Risk: CGo Integration Complexity

**Mitigation**:
1. Use proven patterns from existing projects (go-llama.cpp, whisper.cpp Go bindings)
2. Extensive unit testing with mocked C layer
3. Integration tests with real GPU to catch runtime issues
4. Code review focused on memory safety and error handling
5. Valgrind/AddressSanitizer to detect memory errors during development

### Risk: Platform Build Complexity

**Mitigation**:
1. Document build process in extreme detail
2. Provide automated build scripts
3. Test on clean VMs for each platform
4. Maintain Docker images with build environments
5. CI/CD pipeline to catch build breakage early

### Risk: GPU Compatibility Issues

**Mitigation**:
1. Test on multiple RTX GPU models (3060, 4070, 4090)
2. Test on different CUDA versions (12.1, 12.3)
3. Detect GPU at runtime and warn if incompatible
4. Provide clear system requirements in docs
5. Fail gracefully with helpful error message if GPU insufficient

### Risk: Model Performance

**Mitigation**:
1. Set realistic user expectations (local ≠ OpenAI quality)
2. Benchmark against OpenAI API (Phase 1) for comparison
3. Optimize inference parameters (batch size, quantization)
4. Provide tuning guide for users who want to experiment
5. Consider larger model in future if user demand exists

### Risk: Model Download Size

**Mitigation**:
1. Offer bundled installer (larger) and download-on-first-run (smaller)
2. Implement resumable HTTP downloads
3. Show clear progress bar during download
4. Provide torrent/mirror options for faster downloads
5. Document model size in system requirements

### Risk: Integration Breakage

**Mitigation**:
1. Maintain same function signatures initially
2. Comprehensive integration tests for all AI operations
3. Acceptance tests verify end-to-end workflows
4. Consider feature flag to toggle OpenAI vs. llama during development
5. Thorough manual testing before merging Phase 2

## Timeline and Milestones

### Estimated Timeline

Based on roadmap sizing:
- **Item 8** (llama.cpp CUDA Integration): 6-10 days
- **Item 9** (Cross-Platform Build): 6-10 days
- **Item 10** (Bunny Integration): 3-5 days
- **Item 11** (Vision Pipeline): 3-5 days
- **Item 12** (Health Monitoring): 1-2 days

**Total**: 19-32 days (4-6 weeks)

**Critical Path**: 8 → 9 → 10 → 11 → 12 (sequential dependencies)

### Milestones

**Milestone 1: CGo Bindings (Week 1-2)**
- llamaruntime/ package created
- Basic CGo wrappers for llama.cpp API
- Unit tests with mocked C layer
- **Deliverable**: llamaruntime can load model and run inference (mocked)

**Milestone 2: Cross-Platform Builds (Week 2-3)**
- CMake build scripts for Windows and Linux
- Compiled llama.dll and libllama.so with CUDA
- Go application links against native libraries
- **Deliverable**: Application builds and runs on both platforms

**Milestone 3: Bunny Integration (Week 3-4)**
- Bunny v1.1 model loaded successfully
- Text inference works for notes, PDFs, canvas
- OpenAI API calls replaced in handlers.go
- **Deliverable**: End-to-end text generation works locally

**Milestone 4: Vision Pipeline (Week 4-5)**
- Image preprocessing implemented
- InferVision() API functional
- Image analysis works on canvas
- **Deliverable**: End-to-end image analysis works

**Milestone 5: Health Monitoring (Week 5-6)**
- Startup diagnostics implemented
- GPU memory monitoring active
- Periodic health checks running
- Automatic recovery tested
- **Deliverable**: Production-ready reliability

**Final: Phase 2 Complete (Week 6)**
- All tests passing (unit, integration, acceptance)
- Documentation complete
- Performance benchmarks meet targets
- 24-hour stability test passed
- **Deliverable**: Phase 2 ready to merge to main

## Acceptance Criteria

Phase 2 is **DONE** when all of the following are true:

### Functional Criteria

- [ ] llamaruntime/ package created with complete Go API
- [ ] CGo bindings to llama.cpp functional and tested
- [ ] llama.cpp builds successfully on Windows (MSVC + CUDA)
- [ ] llama.cpp builds successfully on Linux (GCC + CUDA)
- [ ] Bunny v1.1 model loads on application startup
- [ ] Startup health check passes with test inference
- [ ] Text generation works for notes (`{{ prompt }}` syntax)
- [ ] Text generation works for PDF analysis
- [ ] Text generation works for canvas analysis
- [ ] Vision inference works for image analysis
- [ ] All OpenAI API calls removed from codebase
- [ ] OPENAI_API_KEY no longer required in .env
- [ ] GPU memory monitoring logs VRAM usage
- [ ] Periodic health checks run successfully
- [ ] Automatic recovery works after simulated failure

### Performance Criteria

- [ ] Inference speed ≥20 tokens/second on RTX 3060
- [ ] Inference speed ≥40 tokens/second on RTX 4070
- [ ] First token latency ≤500ms for text-only
- [ ] First token latency ≤1s for vision
- [ ] Model loading time ≤10 seconds
- [ ] GPU VRAM usage ≤6GB for Q4_K_M model
- [ ] Concurrent requests (≤5) work without degradation

### Reliability Criteria

- [ ] Application runs for 24+ hours without crash
- [ ] No memory leaks detected in 8-hour stress test
- [ ] Automatic recovery successful after simulated GPU error
- [ ] Graceful shutdown cleans up all resources

### Testing Criteria

- [ ] All unit tests pass on Windows
- [ ] All unit tests pass on Linux
- [ ] All integration tests pass on Windows
- [ ] All integration tests pass on Linux
- [ ] All acceptance tests pass (end-to-end scenarios)
- [ ] Performance benchmarks meet targets

### Documentation Criteria

- [ ] README.md updated with Phase 2 changes
- [ ] docs/build-guide.md created
- [ ] docs/llamaruntime.md created
- [ ] docs/bunny-model.md created
- [ ] docs/troubleshooting.md created
- [ ] .env.example updated
- [ ] Godoc comments complete
- [ ] Migration guide from Phase 1 to Phase 2 written

### User Experience Criteria

- [ ] Configuration requires only Canvus credentials
- [ ] AI responses arrive in <5 seconds for typical prompts
- [ ] Vision analysis completes in <10 seconds for typical images
- [ ] Startup completes in <30 seconds with clear diagnostics
- [ ] Error messages clearly explain what went wrong
- [ ] No network connections to external AI services (verified)

## Appendix

### Glossary

- **CGo**: Mechanism for calling C code from Go
- **CUDA**: NVIDIA's parallel computing platform for GPU acceleration
- **GGUF**: GPT-Generated Unified Format, model file format for llama.cpp
- **Quantization**: Reducing model precision (e.g., 16-bit → 4-bit) to save memory
- **Context Window**: Number of tokens the model can process at once
- **Batch Size**: Number of tokens processed in parallel during inference
- **GPU Layers**: Number of model layers offloaded to GPU (vs. CPU)
- **RTX**: NVIDIA's GPU architecture with ray tracing and tensor cores
- **VRAM**: Video RAM, GPU memory for model weights and activations
- **Tokens**: Subword units used by LLMs (roughly ¾ of a word)
- **Tok/s**: Tokens per second, measure of inference speed

### References

- llama.cpp: https://github.com/ggerganov/llama.cpp
- Bunny v1.1: https://huggingface.co/BAAI/Bunny-v1_1-Llama-3-8B-V
- GGUF Specification: https://github.com/ggerganov/ggml/blob/master/docs/gguf.md
- Go CGo Documentation: https://pkg.go.dev/cmd/cgo
- CUDA Toolkit: https://developer.nvidia.com/cuda-toolkit
- CMake Documentation: https://cmake.org/documentation/

### Related Documents

- [Product Mission](../../product/mission.md)
- [Product Roadmap](../../product/roadmap.md)
- [Tech Stack](../../product/tech-stack.md)
- [Phase 1 Spec](../zero-config-installation/spec.md) (if exists)
- [Requirements Document](./planning/requirements.md)
