# llamaruntime Package - API Reference

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Public API Reference](#public-api-reference)
3. [CGo Integration](#cgo-integration)
4. [Performance Tuning](#performance-tuning)
5. [Memory Management](#memory-management)
6. [Error Handling](#error-handling)
7. [Health Monitoring](#health-monitoring)
8. [Best Practices](#best-practices)

---

## Architecture Overview

### Package Purpose

The `llamaruntime` package provides Go bindings to llama.cpp for local LLM inference with CUDA GPU acceleration. It enables CanvusLocalLLM to perform text and vision AI operations entirely locally, eliminating cloud API dependencies.

### Design Philosophy

The package follows atomic design principles:

- **Atoms**: Pure CGo wrappers (`bindings.go`), type definitions (`types.go`), error types (`errors.go`)
- **Molecules**: High-level API client (`client.go`), context pool manager (`context.go`), health monitoring (`health.go`)
- **Organism**: The complete `llamaruntime` package provides full LLM inference capability

### Component Hierarchy

```
llamaruntime (Organism Package)
├── client.go (Molecule)      - High-level Go API
├── context.go (Molecule)     - Context pool management
├── health.go (Molecule)      - GPU monitoring & health checks
├── bindings.go (Atom)        - CGo wrappers to llama.cpp
├── types.go (Atom)           - Go types and constants
└── errors.go (Atom)          - Error types and sentinels
```

### Data Flow

**Text Inference Flow**:
```
Application → Client.Infer() → ContextPool.Acquire() →
CGo Binding → llama.cpp (C++) → CUDA GPU →
CGo Return → ContextPool.Release() → Response
```

**Vision Inference Flow**:
```
Application → Client.InferVision() → Preprocess Image →
ContextPool.Acquire() → CGo Binding → llama.cpp Vision →
CUDA GPU → CGo Return → ContextPool.Release() → Response
```

### Thread Safety

- All public methods are thread-safe
- Context pool uses channels for safe concurrent access
- Client methods protected by `sync.RWMutex`
- No concurrent access to individual llama contexts

---

## Public API Reference

### Client Type

The `Client` is the main entry point for all inference operations.

```go
type Client struct {
    // Contains unexported fields
}
```

#### NewClient

Creates and initializes a llamaruntime client.

```go
func NewClient(config Config) (*Client, error)
```

**Parameters**:
- `config`: Configuration struct with model path, context settings, and inference parameters

**Returns**:
- `*Client`: Initialized client ready for inference
- `error`: Error if initialization fails (GPU detection, model loading, context creation, or health check failure)

**Behavior**:
1. Validates configuration (model path exists, required fields set)
2. Applies default values for unset configuration fields
3. Initializes llama.cpp library
4. Detects and logs GPU information
5. Loads model from disk
6. Creates context pool
7. Runs startup health check
8. Returns initialized client or error

**Example**:
```go
config := llamaruntime.Config{
    ModelPath:     "models/bunny-v1.1-llama-3-8b-v.gguf",
    ContextSize:   4096,
    BatchSize:     512,
    GPULayers:     -1,  // Offload all layers to GPU
    NumContexts:   5,   // Pool size
    Temperature:   0.7,
    TopP:          0.9,
    RepeatPenalty: 1.1,
}

client, err := llamaruntime.NewClient(config)
if err != nil {
    log.Fatalf("Failed to initialize llamaruntime: %v", err)
}
defer client.Close()
```

**Errors**:
- Returns `error` if `ModelPath` is empty
- Returns `ErrModelNotFound` if model file doesn't exist
- Returns wrapped error if GPU detection fails
- Returns wrapped error if model loading fails
- Returns wrapped error if context pool creation fails
- Returns wrapped error if startup health check fails

#### Infer

Generates text from a prompt using the loaded model.

```go
func (c *Client) Infer(ctx context.Context, prompt string, maxTokens int) (string, error)
```

**Parameters**:
- `ctx`: Context for cancellation and timeout control
- `prompt`: Input text prompt for the model
- `maxTokens`: Maximum number of tokens to generate (not including prompt)

**Returns**:
- `string`: Generated text response
- `error`: Error if inference fails or client is closed

**Behavior**:
1. Checks if client is closed (returns error if true)
2. Acquires context from pool (blocks if all contexts busy)
3. Runs text inference via CGo binding
4. Logs inference duration
5. Releases context back to pool (via defer)
6. Returns generated text

**Context Handling**:
- Respects `ctx.Done()` for cancellation
- Inference operation can be cancelled mid-generation
- Context timeout triggers `ErrTimeout`

**Example**:
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := client.Infer(ctx, "Explain quantum computing in simple terms", 200)
if err != nil {
    log.Printf("Inference failed: %v", err)
    return
}
fmt.Println(response)
```

**Thread Safety**: Safe to call from multiple goroutines concurrently.

**Errors**:
- Returns `error` if client is closed
- Returns `ctx.Err()` if context cancelled or timed out
- Returns wrapped error if context acquisition fails
- Returns `ErrInferenceFailed` if inference operation fails
- Returns `ErrTimeout` if inference exceeds configured timeout

#### InferVision

Generates text from a prompt and image using multimodal capabilities.

```go
func (c *Client) InferVision(ctx context.Context, prompt string, imageData []byte, maxTokens int) (string, error)
```

**Parameters**:
- `ctx`: Context for cancellation and timeout control
- `prompt`: Text prompt describing what to analyze in the image
- `imageData`: Raw image bytes (JPEG, PNG, etc.)
- `maxTokens`: Maximum number of tokens to generate

**Returns**:
- `string`: Generated text response describing or analyzing the image
- `error`: Error if inference fails, image is invalid, or client is closed

**Behavior**:
1. Checks if client is closed
2. Preprocesses image (resize, format conversion, normalization)
3. Acquires context from pool
4. Runs multimodal inference via CGo binding (image + text)
5. Logs inference duration
6. Releases context back to pool
7. Returns generated description/analysis

**Image Requirements**:
- Supported formats: JPEG, PNG, BMP, GIF
- Automatically resized to model's expected resolution (typically 336x336 or 448x448)
- Converted to RGB if grayscale
- Pixel values normalized to model's expected range

**Example**:
```go
ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
defer cancel()

imageData, err := os.ReadFile("photo.jpg")
if err != nil {
    log.Fatalf("Failed to read image: %v", err)
}

response, err := client.InferVision(ctx, "Describe this image in detail", imageData, 300)
if err != nil {
    log.Printf("Vision inference failed: %v", err)
    return
}
fmt.Println(response)
```

**Thread Safety**: Safe to call from multiple goroutines concurrently.

**Errors**:
- Returns `error` if client is closed
- Returns `ErrInvalidImage` if image format is unsupported or corrupted
- Returns `ctx.Err()` if context cancelled or timed out
- Returns wrapped error if context acquisition fails
- Returns `ErrInferenceFailed` if inference operation fails

#### GetGPUMemoryUsage

Returns current GPU VRAM usage in bytes.

```go
func (c *Client) GetGPUMemoryUsage() (int64, error)
```

**Returns**:
- `int64`: Current GPU VRAM usage in bytes
- `error`: Error if GPU memory query fails

**Example**:
```go
used, err := client.GetGPUMemoryUsage()
if err != nil {
    log.Printf("Failed to get GPU memory: %v", err)
} else {
    usedGB := float64(used) / (1024 * 1024 * 1024)
    fmt.Printf("GPU VRAM usage: %.2f GB\n", usedGB)
}
```

**Thread Safety**: Safe to call from multiple goroutines concurrently.

#### HealthCheck

Performs a quick inference to verify the model is working.

```go
func (c *Client) HealthCheck(ctx context.Context) error
```

**Parameters**:
- `ctx`: Context for timeout control (recommended: 10-30 seconds)

**Returns**:
- `error`: Error if health check fails, nil if successful

**Behavior**:
1. Runs simple inference with test prompt ("Hello, how are you?")
2. Generates up to 20 tokens
3. Verifies response is non-empty
4. Returns nil on success, error on failure

**Example**:
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := client.HealthCheck(ctx); err != nil {
    log.Printf("Health check failed: %v", err)
    // Trigger recovery or alert
} else {
    log.Println("Health check passed")
}
```

**Use Cases**:
- Startup validation (called automatically by `NewClient`)
- Periodic health monitoring (recommended: every 5-10 minutes)
- Pre-deployment smoke tests

**Thread Safety**: Safe to call from multiple goroutines concurrently.

#### Close

Releases all resources (contexts, model, GPU memory).

```go
func (c *Client) Close() error
```

**Returns**:
- `error`: Always returns nil (provided for interface compatibility)

**Behavior**:
1. Sets closed flag (subsequent operations will fail)
2. Closes context pool (releases all inference contexts)
3. Frees model from memory
4. Logs shutdown completion

**Example**:
```go
client, err := llamaruntime.NewClient(config)
if err != nil {
    log.Fatalf("Failed to create client: %v", err)
}
defer client.Close()

// Use client...
```

**CRITICAL**: Always call `Close()` when done with client to prevent GPU memory leaks. Use `defer` immediately after successful `NewClient()` call.

**Thread Safety**: Safe to call from multiple goroutines (idempotent, only first call has effect).

---

### Config Type

Configuration for llamaruntime client initialization.

```go
type Config struct {
    ModelPath     string  // Required: Path to GGUF model file
    ContextSize   int     // Context window in tokens (default: 4096)
    BatchSize     int     // Batch size for processing (default: 512)
    GPULayers     int     // Layers to offload to GPU: -1=all, 0=CPU only, N=specific count (default: -1)
    NumContexts   int     // Context pool size for concurrency (default: 5)
    Temperature   float32 // Sampling temperature (default: 0.7)
    TopP          float32 // Nucleus sampling parameter (default: 0.9)
    RepeatPenalty float32 // Repetition penalty (default: 1.1)
}
```

**Field Details**:

- `ModelPath` (required): Absolute or relative path to GGUF model file
- `ContextSize`: Number of tokens the model can process at once (prompt + response). Larger = more memory.
- `BatchSize`: Tokens processed in parallel during prompt encoding. Larger = faster but more VRAM.
- `GPULayers`: `-1` offloads all layers to GPU (recommended), `0` runs on CPU only (slow), `N` offloads N layers.
- `NumContexts`: Size of context pool. Should match `MAX_CONCURRENT` setting for optimal throughput.
- `Temperature`: Controls randomness (0.0-2.0). Lower = more deterministic, higher = more creative.
- `TopP`: Nucleus sampling threshold (0.0-1.0). Lower = more focused, higher = more diverse.
- `RepeatPenalty`: Penalty for repeating tokens (1.0-2.0). Higher = less repetition.

**Defaults**: All fields except `ModelPath` have sensible defaults applied by `NewClient()`.

---

### InferenceStats Type

Performance metrics for an inference operation.

```go
type InferenceStats struct {
    PromptTokens     int           // Tokens in input prompt
    CompletionTokens int           // Tokens in generated response
    TotalTokens      int           // PromptTokens + CompletionTokens
    InferenceTime    time.Duration // Total inference duration
    TokensPerSecond  float64       // Completion tokens / inference time
    FirstTokenTime   time.Duration // Time to first generated token (latency)
}
```

**Usage**: Currently for internal tracking. May be exposed in future API versions for performance monitoring.

---

### GPUInfo Type

Information about detected GPU.

```go
type GPUInfo struct {
    Name          string // GPU model name (e.g., "NVIDIA GeForce RTX 4070")
    VRAMTotal     int64  // Total GPU VRAM in bytes
    VRAMFree      int64  // Free GPU VRAM in bytes
    CUDAVersion   string // CUDA runtime version (e.g., "12.3")
    ComputeCap    string // Compute capability (e.g., "8.9")
    DriverVersion string // GPU driver version
}
```

**Usage**: Logged at startup for diagnostics. Available internally to `Client` for decision-making.

---

### Constants

Default configuration values:

```go
const (
    DefaultContextSize   = 4096  // Tokens
    DefaultBatchSize     = 512   // Tokens
    DefaultGPULayers     = -1    // All layers on GPU
    DefaultNumContexts   = 5     // Concurrent contexts
    DefaultTemperature   = 0.7   // Moderate creativity
    DefaultTopP          = 0.9   // Broad nucleus sampling
    DefaultRepeatPenalty = 1.1   // Slight repetition penalty
)
```

---

## CGo Integration

### Memory Management Strategy

#### Principles

1. **C Memory Lifecycle**: Allocate in Go using `C.malloc()` or let C allocate, free in Go using `C.free()`
2. **No Mixed Allocation**: Never mix Go and C memory allocators
3. **Pointer Safety**: Go pointers passed to C must be pinned (use `runtime.Pinner` in Go 1.21+)
4. **C Pointer Storage**: C pointers stored in Go as `unsafe.Pointer` or `uintptr`
5. **No Go Pointers in C**: Never store Go pointers in C-allocated memory

#### C String Handling

```go
// Correct pattern
cPath := C.CString(path)  // Allocates C memory
defer C.free(unsafe.Pointer(cPath))  // Always free

// Incorrect (memory leak)
C.llama_some_function(C.CString(path))  // No way to free!
```

#### Model and Context Wrappers

Use Go finalizers for cleanup safety:

```go
type llamaModel struct {
    ptr *C.llama_model
}

func loadModel(path string) (*llamaModel, error) {
    cPath := C.CString(path)
    defer C.free(unsafe.Pointer(cPath))

    params := C.llama_model_default_params()
    params.n_gpu_layers = -1

    model := C.llama_load_model_from_file(cPath, params)
    if model == nil {
        return nil, fmt.Errorf("failed to load model")
    }

    m := &llamaModel{ptr: model}

    // Finalizer ensures cleanup even if Close() not called
    runtime.SetFinalizer(m, func(m *llamaModel) {
        if m.ptr != nil {
            C.llama_free_model(m.ptr)
        }
    })

    return m, nil
}

// Explicit cleanup (preferred over relying on finalizer)
func freeModel(m *llamaModel) {
    if m.ptr != nil {
        C.llama_free_model(m.ptr)
        m.ptr = nil
    }
}
```

### Error Handling

#### Error Translation Pattern

C functions typically return NULL or error codes. Go wrappers must translate these:

```go
func createContext(model *llamaModel, config Config) (*llamaContext, error) {
    if model == nil || model.ptr == nil {
        return nil, errors.New("invalid model")
    }

    params := C.llama_context_default_params()
    params.n_ctx = C.int(config.ContextSize)
    params.n_batch = C.int(config.BatchSize)

    ctx := C.llama_new_context_with_model(model.ptr, params)
    if ctx == nil {
        return nil, &LlamaError{
            Op:      "createContext",
            Code:    -1,
            Message: "failed to create inference context",
            Err:     ErrContextCreateFailed,
        }
    }

    return &llamaContext{ptr: ctx}, nil
}
```

#### Panic Recovery

Protect application from CGo panics:

```go
func (c *Client) Infer(ctx context.Context, prompt string, maxTokens int) (response string, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("llama.cpp panic: %v", r)
        }
    }()

    // ... inference logic
}
```

### Thread Safety

#### Context Access Control

Individual llama contexts are **not thread-safe**. The context pool ensures exclusive access:

```go
type ContextPool struct {
    contexts chan *llamaContext  // Channel provides mutual exclusion
    // ...
}

func (p *ContextPool) Acquire(ctx context.Context) (*llamaContext, error) {
    select {
    case c := <-p.contexts:
        return c, nil  // Got exclusive access
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

func (p *ContextPool) Release(ctx *llamaContext) {
    p.contexts <- ctx  // Return for others to use
}
```

#### Client Mutex Protection

Client state protected by `sync.RWMutex`:

```go
type Client struct {
    mu     sync.RWMutex
    closed bool
    // ...
}

func (c *Client) Infer(ctx context.Context, prompt string, maxTokens int) (string, error) {
    c.mu.RLock()
    if c.closed {
        c.mu.RUnlock()
        return "", errors.New("client closed")
    }
    c.mu.RUnlock()
    // ... proceed with inference
}

func (c *Client) Close() error {
    c.mu.Lock()
    defer c.mu.Unlock()

    if c.closed {
        return nil  // Already closed
    }
    c.closed = true
    // ... cleanup
}
```

---

## Performance Tuning

### Model Quantization

Quantization reduces model size and VRAM usage at some quality cost:

| Quantization | Size  | VRAM  | Quality | Speed     | Recommended GPU |
|--------------|-------|-------|---------|-----------|-----------------|
| Q4_K_M       | ~5GB  | 6GB   | Good    | Fastest   | RTX 3060 (12GB) |
| Q5_K_M       | ~6GB  | 7GB   | Better  | Fast      | RTX 4070 (12GB) |
| Q6_K         | ~7GB  | 8GB   | Best    | Moderate  | RTX 4080 (16GB) |
| F16 (full)   | ~15GB | 16GB  | Perfect | Slower    | RTX 4090 (24GB) |

**Recommendation**: Start with Q4_K_M for compatibility, upgrade to Q5_K_M if quality insufficient.

### Context Size Tuning

`ContextSize` determines maximum prompt + response tokens:

- **2048**: Short conversations, limited context (less VRAM)
- **4096**: Balanced for most use cases (recommended)
- **8192**: Long documents, complex prompts (more VRAM)
- **16384+**: Extreme use cases, requires high VRAM

**Memory Impact**: Each doubling of context size roughly doubles VRAM usage.

**Tuning**:
```go
// For short notes and quick responses
config.ContextSize = 2048

// For PDF analysis and longer content
config.ContextSize = 8192
```

### Batch Size Tuning

`BatchSize` controls parallel token processing during prompt encoding:

- **256**: Slower prompt processing, less VRAM
- **512**: Balanced (recommended)
- **1024**: Faster prompt processing, more VRAM
- **2048**: Maximum speed, high VRAM

**When to Increase**:
- Long prompts (e.g., full PDF text)
- High VRAM GPU (RTX 4080+)
- Latency-sensitive applications

**When to Decrease**:
- Low VRAM GPU (RTX 3060)
- Short prompts (single sentence)
- Running out of VRAM

### GPU Layer Offloading

`GPULayers` controls how much of model runs on GPU vs CPU:

- **-1**: All layers on GPU (fastest, recommended)
- **0**: All layers on CPU (very slow, not recommended)
- **N** (e.g., 20): First N layers on GPU, rest on CPU (hybrid)

**Use Cases for Hybrid**:
- Insufficient VRAM for full model
- Want to run larger quantization (Q6_K) on smaller GPU
- Testing CPU-only performance

**Example**:
```go
// Full GPU offload (recommended)
config.GPULayers = -1

// Hybrid mode for VRAM-constrained systems
config.GPULayers = 30  // Offload 30/40 layers, keep rest on CPU
```

### Context Pool Size

`NumContexts` determines concurrent inference capacity:

- **1**: Serialized inference only
- **3-5**: Balanced for typical workload (recommended)
- **8-10**: High concurrency, requires more VRAM

**Tuning Rule**: Set `NumContexts = MAX_CONCURRENT` from application config.

**VRAM Impact**: Each context consumes VRAM. Monitor usage with `GetGPUMemoryUsage()`.

### Generation Parameters

Fine-tune response quality and diversity:

```go
// Conservative (factual, deterministic)
config.Temperature = 0.3
config.TopP = 0.85
config.RepeatPenalty = 1.15

// Balanced (recommended)
config.Temperature = 0.7
config.TopP = 0.9
config.RepeatPenalty = 1.1

// Creative (diverse, varied)
config.Temperature = 1.0
config.TopP = 0.95
config.RepeatPenalty = 1.05
```

**When to Adjust**:
- Temperature: Lower for factual tasks (analysis), higher for creative tasks (brainstorming)
- TopP: Lower to reduce randomness, higher for more variety
- RepeatPenalty: Higher if model repeats itself, lower if responses feel constrained

### Performance Expectations

Baseline performance on common GPUs (Q4_K_M quantization, 4096 context):

| GPU Model | VRAM | Tokens/Sec | First Token | 100 Token Response |
|-----------|------|------------|-------------|---------------------|
| RTX 3060  | 12GB | 20-25      | ~500ms      | ~4-5s               |
| RTX 3070  | 8GB  | 30-35      | ~400ms      | ~3s                 |
| RTX 4070  | 12GB | 40-50      | ~300ms      | ~2-2.5s             |
| RTX 4080  | 16GB | 60-70      | ~250ms      | ~1.5-2s             |
| RTX 4090  | 24GB | 80-100     | ~200ms      | ~1-1.2s             |

**Performance Tuning Steps**:
1. Start with defaults
2. Run benchmarks: `go test -bench=. ./tests/`
3. Adjust batch size if prompt processing is slow
4. Adjust context size if VRAM constrained
5. Try different quantization if quality/speed imbalance

---

## Memory Management

### GPU VRAM Budget

Typical VRAM allocation for Q4_K_M model (5GB):

| Component        | VRAM   | Description                          |
|------------------|--------|--------------------------------------|
| Model Weights    | 5GB    | Quantized model parameters           |
| Context KV Cache | 1-2GB  | Cached key-value pairs (per context) |
| Workspace        | 500MB  | Temporary buffers for computation    |
| **Total**        | **6-8GB** | Per inference context             |

**Multi-Context Calculation**: `Total VRAM = Model + (ContextKVCache * NumContexts) + Workspace`

**Example** (RTX 4070 with 12GB):
- Model: 5GB
- KV Cache: 1.5GB × 3 contexts = 4.5GB
- Workspace: 500MB
- Total: 10GB (safe, leaves 2GB headroom)

### Memory Leak Prevention

#### Always Use defer with Close

```go
// Correct
client, err := llamaruntime.NewClient(config)
if err != nil {
    return err
}
defer client.Close()  // Ensures cleanup even if panic

// Incorrect (leaks on panic/early return)
client, err := llamaruntime.NewClient(config)
// ... do work
client.Close()  // May never execute
```

#### Context Pool Resource Management

Context pool automatically cleans up on `Close()`:

```go
func (p *ContextPool) Close() error {
    p.mu.Lock()
    defer p.mu.Unlock()

    if p.closed {
        return nil
    }
    p.closed = true

    close(p.contexts)
    for ctx := range p.contexts {
        freeContext(ctx)  // Release each context
    }

    return nil
}
```

#### Finalizers as Safety Net

Go finalizers provide backup cleanup:

```go
runtime.SetFinalizer(model, func(m *llamaModel) {
    if m.ptr != nil {
        C.llama_free_model(m.ptr)
    }
})
```

**CRITICAL**: Do not rely on finalizers alone. Always call `Close()` explicitly.

### Long-Running Applications

For 24/7 operation, monitor VRAM usage:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// Monitor VRAM every minute
go func() {
    ticker := time.NewTicker(60 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            used, _ := client.GetGPUMemoryUsage()
            usedGB := float64(used) / (1024 * 1024 * 1024)
            log.Printf("[llama] GPU VRAM: %.2f GB", usedGB)

            if usedGB > 10.0 {  // Threshold for 12GB GPU
                log.Printf("[llama] WARNING: High VRAM usage")
            }
        }
    }
}()
```

---

## Error Handling

### Error Types

#### LlamaError

Structured error type for llama.cpp operations:

```go
type LlamaError struct {
    Op      string // Operation that failed (e.g., "loadModel", "infer")
    Code    int    // Error code from C layer (0 = success, non-zero = error)
    Message string // Human-readable error message
    Err     error  // Wrapped underlying error (if any)
}

func (e *LlamaError) Error() string
func (e *LlamaError) Unwrap() error
```

**Usage**:
```go
if err != nil {
    var llamaErr *llamaruntime.LlamaError
    if errors.As(err, &llamaErr) {
        log.Printf("llama.cpp operation '%s' failed: %s (code: %d)",
            llamaErr.Op, llamaErr.Message, llamaErr.Code)
    }
}
```

### Sentinel Errors

Use `errors.Is()` to check for specific error conditions:

```go
var (
    ErrModelNotFound       = errors.New("model file not found")
    ErrModelLoadFailed     = errors.New("failed to load model")
    ErrContextCreateFailed = errors.New("failed to create inference context")
    ErrInferenceFailed     = errors.New("inference failed")
    ErrGPUNotAvailable     = errors.New("CUDA GPU not available")
    ErrInsufficientVRAM    = errors.New("insufficient GPU VRAM")
    ErrInvalidImage        = errors.New("invalid or unsupported image format")
    ErrTimeout             = errors.New("inference timeout")
)
```

**Usage**:
```go
if errors.Is(err, llamaruntime.ErrModelNotFound) {
    log.Fatal("Model file not found. Please download model to models/ directory")
}

if errors.Is(err, llamaruntime.ErrGPUNotAvailable) {
    log.Fatal("CUDA GPU required. Please ensure NVIDIA GPU with CUDA support is available")
}
```

### Error Handling Patterns

#### Startup Errors (Fail Fast)

```go
client, err := llamaruntime.NewClient(config)
if err != nil {
    if errors.Is(err, llamaruntime.ErrGPUNotAvailable) {
        log.Fatalf("ERROR: CUDA GPU not detected. CanvusLocalLLM requires NVIDIA GPU with CUDA support.")
    }
    if errors.Is(err, llamaruntime.ErrModelNotFound) {
        log.Fatalf("ERROR: Model not found at %s. Please download model.", config.ModelPath)
    }
    log.Fatalf("Failed to initialize llamaruntime: %v", err)
}
defer client.Close()
```

#### Inference Errors (Retry with Backoff)

```go
func inferWithRetry(client *llamaruntime.Client, prompt string, maxRetries int) (string, error) {
    var lastErr error
    for attempt := 1; attempt <= maxRetries; attempt++ {
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        response, err := client.Infer(ctx, prompt, 200)
        cancel()

        if err == nil {
            return response, nil
        }

        lastErr = err
        if errors.Is(err, llamaruntime.ErrTimeout) {
            log.Printf("Attempt %d/%d: Inference timed out, retrying...", attempt, maxRetries)
        } else {
            log.Printf("Attempt %d/%d: Inference failed: %v", attempt, maxRetries, err)
        }

        if attempt < maxRetries {
            time.Sleep(time.Duration(attempt) * time.Second)  // Exponential backoff
        }
    }
    return "", fmt.Errorf("inference failed after %d attempts: %w", maxRetries, lastErr)
}
```

---

## Health Monitoring

### GPU Memory Monitoring

Periodic VRAM usage tracking:

```go
func MonitorGPUMemory(ctx context.Context, client *llamaruntime.Client, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            used, err := client.GetGPUMemoryUsage()
            if err != nil {
                log.Printf("[llama] GPU memory query failed: %v", err)
                continue
            }

            usedGB := float64(used) / (1024 * 1024 * 1024)
            log.Printf("[llama] GPU VRAM: %.2f GB", usedGB)

            if usedGB > 10.0 {  // Adjust threshold for your GPU
                log.Printf("[llama] WARNING: High VRAM usage (%.2f GB)", usedGB)
            }
        }
    }
}
```

**Usage**:
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go llamaruntime.MonitorGPUMemory(ctx, client, 60*time.Second)
```

### Periodic Health Checks

Verify model is responsive:

```go
func PeriodicHealthCheck(ctx context.Context, client *llamaruntime.Client, interval time.Duration) {
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

**Usage**:
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

go llamaruntime.PeriodicHealthCheck(ctx, client, 5*time.Minute)
```

### Startup Diagnostics

Log comprehensive startup information:

```go
client, err := llamaruntime.NewClient(config)
if err != nil {
    log.Fatalf("Initialization failed: %v", err)
}
defer client.Close()

// Logs generated by NewClient:
// [llama] Detected GPU: NVIDIA GeForce RTX 4070, VRAM: 12GB, CUDA: 12.3
// [llama] Loading model from models/bunny-v1.1-llama-3-8b-v.gguf...
// [llama] Model loaded: 5.2GB, Q4_K_M quantization
// [llama] Creating context pool with 5 contexts...
// [llama] Context pool created
// [llama] Running startup health check...
// [llama] Health check: 18 tokens in 0.82s (21.95 tok/s)
// [llama] Health check passed
// [llama] llamaruntime initialization complete
```

---

## Best Practices

### Initialization

1. **Always defer Close**: Prevents GPU memory leaks

```go
client, err := llamaruntime.NewClient(config)
if err != nil {
    return err
}
defer client.Close()  // CRITICAL
```

2. **Validate configuration early**: Check model file exists before calling `NewClient()`

```go
if _, err := os.Stat(modelPath); os.IsNotExist(err) {
    return fmt.Errorf("model file not found: %s", modelPath)
}
```

3. **Use reasonable defaults**: Don't override default config unless necessary

```go
config := llamaruntime.Config{
    ModelPath: modelPath,  // Only required field
    // Let NewClient() apply defaults for other fields
}
```

### Context Management

1. **Use context.WithTimeout**: Always set timeouts for inference

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
response, err := client.Infer(ctx, prompt, maxTokens)
```

2. **Respect context cancellation**: Allows graceful shutdown

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// In shutdown handler
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan
cancel()  // Cancels all in-flight inferences
```

### Concurrency

1. **Client is thread-safe**: Safe to call from multiple goroutines

```go
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(i int) {
        defer wg.Done()
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        response, err := client.Infer(ctx, fmt.Sprintf("Prompt %d", i), 100)
        // Handle response
    }(i)
}
wg.Wait()
```

2. **Limit concurrency to NumContexts**: Don't spawn more goroutines than context pool size

```go
// If NumContexts=5, don't spawn 100 concurrent inferences
// Use semaphore to limit concurrency:
sem := make(chan struct{}, 5)  // Match NumContexts
for i := 0; i < 100; i++ {
    sem <- struct{}{}  // Acquire
    go func(i int) {
        defer func() { <-sem }()  // Release
        // ... inference
    }(i)
}
```

### Error Handling

1. **Check for sentinel errors**: Use `errors.Is()` for specific conditions

```go
if errors.Is(err, llamaruntime.ErrTimeout) {
    // Retry or log timeout
}
```

2. **Wrap errors with context**: Add application-specific context

```go
if err != nil {
    return fmt.Errorf("failed to generate response for note %s: %w", noteID, err)
}
```

3. **Log errors comprehensively**: Include all relevant details

```go
if err != nil {
    log.Printf("ERROR: Inference failed for prompt (len=%d, maxTokens=%d): %v",
        len(prompt), maxTokens, err)
}
```

### Performance

1. **Tune for your GPU**: Match quantization and context size to VRAM

```go
// RTX 3060 (12GB)
config.ContextSize = 4096
// Use Q4_K_M model

// RTX 4090 (24GB)
config.ContextSize = 8192
// Use Q5_K_M or Q6_K model
```

2. **Monitor VRAM usage**: Watch for memory pressure

```go
go MonitorGPUMemory(ctx, client, 60*time.Second)
```

3. **Profile in production**: Measure actual performance

```go
start := time.Now()
response, err := client.Infer(ctx, prompt, maxTokens)
elapsed := time.Since(start)
tokensPerSec := float64(len(strings.Split(response, " "))) / elapsed.Seconds()
log.Printf("Inference: %d tokens in %v (%.2f tok/s)", len(response), elapsed, tokensPerSec)
```

### Long-Running Applications

1. **Run periodic health checks**: Detect model degradation

```go
go PeriodicHealthCheck(ctx, client, 5*time.Minute)
```

2. **Monitor for memory leaks**: Track VRAM over hours

```go
go MonitorGPUMemory(ctx, client, 60*time.Second)
```

3. **Implement graceful shutdown**: Clean up on SIGTERM

```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan

log.Println("Shutting down...")
cancel()  // Cancel all contexts
time.Sleep(2 * time.Second)  // Allow in-flight requests to complete
client.Close()  // Cleanup
```

### Testing

1. **Use integration tests with real GPU**: Unit tests can't catch CGo issues

```go
func TestLlamaIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    // ... test with real model and GPU
}
```

2. **Test error conditions**: Simulate failures

```go
// Test with non-existent model
_, err := llamaruntime.NewClient(llamaruntime.Config{
    ModelPath: "does-not-exist.gguf",
})
assert.Error(t, err)
assert.True(t, errors.Is(err, llamaruntime.ErrModelNotFound))
```

3. **Benchmark performance**: Track regressions

```go
go test -bench=. -benchmem -benchtime=30s ./tests/
```

---

## Troubleshooting

### Common Issues

#### "CUDA GPU not available"

**Cause**: No NVIDIA GPU detected or CUDA not installed

**Solutions**:
1. Verify GPU: `nvidia-smi`
2. Install CUDA Toolkit 12.x
3. Update NVIDIA drivers
4. Check llama.cpp built with CUDA support

#### "Model file not found"

**Cause**: `ModelPath` points to non-existent file

**Solutions**:
1. Check file path: `ls -l models/`
2. Download model: See docs/bunny-model.md
3. Use absolute path if relative path not working

#### "Failed to create inference context"

**Cause**: Insufficient VRAM for context

**Solutions**:
1. Reduce `ContextSize` (e.g., 4096 → 2048)
2. Reduce `NumContexts` (e.g., 5 → 3)
3. Use smaller quantization (Q5_K_M → Q4_K_M)
4. Close other GPU-using applications

#### "Inference timeout"

**Cause**: Inference taking longer than context timeout

**Solutions**:
1. Increase context timeout (30s → 60s)
2. Reduce `maxTokens` in `Infer()` call
3. Check GPU not throttling (temperature/power)
4. Reduce `ContextSize` or `BatchSize`

#### "Out of memory"

**Cause**: VRAM exhausted

**Solutions**:
1. Reduce `NumContexts`
2. Reduce `ContextSize`
3. Use smaller quantization
4. Check for memory leaks (call `Close()` properly)
5. Monitor with `GetGPUMemoryUsage()`

### Debugging Tips

1. **Enable verbose logging**: Set log level to debug
2. **Check CUDA version**: `nvcc --version` and `nvidia-smi`
3. **Verify model file**: Check file size and integrity
4. **Monitor VRAM**: Use `nvidia-smi dmon` during operation
5. **Test with simple inference**: Verify basic functionality before complex operations

---

## See Also

- [Build Guide](build-guide.md) - Building llama.cpp and Go application
- [Bunny Model Guide](bunny-model.md) - Model details and prompt engineering
- [Troubleshooting Guide](troubleshooting.md) - Detailed troubleshooting steps
- [Embedded LLM Integration Spec](../agent-os/specs/embedded-llm-integration/spec.md) - Full Phase 2 specification
