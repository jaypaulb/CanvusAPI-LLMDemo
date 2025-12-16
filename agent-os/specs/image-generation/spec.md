# Image Generation Specification
## Phase 3: stable-diffusion.cpp Integration

**Version:** 1.0
**Status:** Draft
**Author:** Agent-OS
**Date:** 2025-12-16

---

## Table of Contents

1. [Overview](#overview)
2. [Goals and Non-Goals](#goals-and-non-goals)
3. [Architecture](#architecture)
4. [stable-diffusion.cpp Integration](#stable-diffusioncpp-integration)
5. [CGo Binding Architecture](#cgo-binding-architecture)
6. [Image Generation Pipeline](#image-generation-pipeline)
7. [Canvas Integration](#canvas-integration)
8. [Model Selection and Management](#model-selection-and-management)
9. [Configuration](#configuration)
10. [Error Handling](#error-handling)
11. [Testing Strategy](#testing-strategy)
12. [Build and Deployment](#build-and-deployment)
13. [Performance Considerations](#performance-considerations)
14. [Security Considerations](#security-considerations)
15. [Acceptance Criteria](#acceptance-criteria)
16. [Implementation Plan](#implementation-plan)

---

## Overview

### Purpose

Phase 3 adds local image generation capabilities to CanvusLocalLLM by integrating stable-diffusion.cpp, a C++ library from the llama.cpp ecosystem. This enables users to generate images from text prompts entirely on local NVIDIA RTX GPUs with CUDA acceleration, maintaining complete data privacy and the zero-configuration installation philosophy.

### Background

CanvusLocalLLM currently provides text generation and vision analysis via embedded Bunny v1.1 model using llama.cpp. Phase 3 extends multimodal capabilities by adding text-to-image generation through stable-diffusion.cpp, following the same embedded CGo pattern. This completes the local AI toolkit with no cloud dependencies.

### Scope

**In Scope:**
- stable-diffusion.cpp CGo bindings via `sdruntime/` package
- Text-to-image generation pipeline
- Automatic canvas placement of generated images
- Cross-platform builds (Windows + Linux) with CUDA
- Model bundling with installers (Stable Diffusion v1.5)
- Configuration management via existing .env system
- Error handling and recovery
- Integration testing

**Out of Scope:**
- Image-to-image generation
- Inpainting or outpainting
- ControlNet integration
- Multiple model support
- LoRA fine-tuning
- Upscaling or super-resolution
- CPU fallback (CUDA required)
- Real-time generation preview

### Key Principles

1. **Zero-Configuration:** Works out-of-the-box with bundled model, no setup required
2. **Complete Local Privacy:** All processing on local hardware, zero cloud dependencies
3. **Atomic Design:** Follow established architecture patterns from llama.cpp integration
4. **CUDA-First:** Optimized for NVIDIA RTX GPUs, no CPU fallback
5. **Canvus-Native:** Generated images appear directly on canvas as widgets

---

## Goals and Non-Goals

### Goals

1. **Enable Local Image Generation:** Users can generate images from text prompts using local hardware
2. **Maintain Zero-Config Philosophy:** No model selection, no provider configuration, just Canvus credentials
3. **Seamless Canvas Integration:** Images automatically appear on canvas near triggering prompts
4. **Cross-Platform Support:** Identical functionality on Windows and Linux with NVIDIA RTX GPUs
5. **Follow Established Patterns:** Reuse CGo integration patterns from llama.cpp, maintain atomic design
6. **Production-Ready:** Stable, performant, well-tested image generation

### Non-Goals

1. **Support Multiple Models:** Single bundled model (SD v1.5), no user choice
2. **CPU Fallback:** CUDA required, no CPU-only image generation
3. **Advanced Features:** No ControlNet, LoRA, or image-to-image initially
4. **Cloud Integration:** No cloud provider fallback or hybrid approach
5. **Interactive Generation:** No real-time preview or iterative refinement
6. **Custom Training:** No model fine-tuning or training capabilities

---

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      CanvusLocalLLM                         │
│                                                             │
│  ┌────────────────────────────────────────────────────┐   │
│  │              main.go (Page)                        │   │
│  │  - Initializes Config                              │   │
│  │  - Creates Monitor with sdruntime context pool     │   │
│  │  │  - Creates Canvus API Client                     │   │
│  │  - Manages application lifecycle                   │   │
│  └─────────────────┬──────────────────────────────────┘   │
│                    │                                        │
│  ┌─────────────────▼──────────────────────────────────┐   │
│  │        monitorcanvus.go (Organism)                 │   │
│  │  - Detects {{ }} and {{image: }} prompts          │   │
│  │  - Routes to text vs image handlers                │   │
│  │  - Manages widget state                            │   │
│  └─────────────┬──────────────┬───────────────────────┘   │
│                │              │                            │
│  ┌─────────────▼──────────┐  ┌▼────────────────────────┐  │
│  │  handlers.go (Organism)│  │ imagegen/ (Organism)    │  │
│  │  - Text generation     │  │  - Image generation     │  │
│  │  - PDF analysis        │  │  - Prompt processing    │  │
│  │  - Vision analysis     │  │  - Canvas placement     │  │
│  └────────────────────────┘  └──┬──────────────────────┘  │
│                                  │                          │
│  ┌───────────────────────────────▼─────────────────────┐  │
│  │            sdruntime/ (Molecule)                     │  │
│  │  - CGo wrapper for stable-diffusion.cpp             │  │
│  │  - Context management and thread safety             │  │
│  │  - Image generation API                             │  │
│  └─────────────────────┬────────────────────────────────┘  │
│                        │ CGo Boundary                       │
│  ┌─────────────────────▼────────────────────────────────┐  │
│  │      stable-diffusion.cpp (Native Library)           │  │
│  │  - C++ inference engine with CUDA                    │  │
│  │  - Model loading and context management             │  │
│  │  - Text-to-image generation                          │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │            canvusapi/ (Organism)                     │  │
│  │  - UploadImage() for generated images                │  │
│  │  - Coordinate management                             │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Component Layers (Atomic Design)

**Atoms (Pure Functions):**
- `sdruntime/load_model.go`: Load SD model from disk
- `sdruntime/image_utils.go`: PNG encoding, memory management
- `core/config.go`: Parse SD configuration variables (SD_IMAGE_SIZE, etc.)

**Molecules (Simple Compositions):**
- `sdruntime/context.go`: SD context pool management
- `sdruntime/generate.go`: Single image generation call
- `imagegen/placement.go`: Calculate image placement coordinates

**Organisms (Feature Modules):**
- `sdruntime/` package: Complete stable-diffusion.cpp CGo wrapper
- `imagegen/` package: Image generation pipeline (prompt → image → canvas)
- `monitorcanvus.go`: Extended with image prompt detection

**Pages (Composition Roots):**
- `main.go`: Initialize SD runtime alongside llama runtime, wire into Monitor

### Data Flow

```
1. User writes prompt on canvas: {{image: a sunset over mountains}}
2. Monitor detects {{image: ...}} syntax via handleUpdate()
3. Routes to imagegen.ProcessImagePrompt()
4. imagegen extracts prompt text: "a sunset over mountains"
5. Calls sdruntime.Generate(prompt, params)
6. sdruntime acquires context from pool
7. CGo call to stable-diffusion.cpp: sd_txt2img()
8. CUDA accelerated inference generates image bytes
9. Return PNG image data to Go layer
10. imagegen saves to downloads/temp-{uuid}.png
11. Calculate placement coordinates relative to prompt widget
12. canvusapi.UploadImage(imagePath, x, y)
13. Canvus displays image widget on canvas
14. Clean up temporary file
```

---

## stable-diffusion.cpp Integration

### Library Overview

**stable-diffusion.cpp** is a C++ implementation of Stable Diffusion inference optimized for CPU and CUDA. It follows the same architecture philosophy as llama.cpp:
- Minimal dependencies
- GGML backend for tensor operations
- CUDA acceleration support
- Cross-platform (Windows, Linux, macOS)
- Permissive license (MIT)

**Repository:** https://github.com/leejet/stable-diffusion.cpp

### Integration Approach

Follow the same pattern as llama.cpp integration in Phase 2:

1. **Build native library** with CMake + CUDA
2. **Create Go CGo wrapper** in `sdruntime/` package
3. **Manage contexts in pool** for thread safety and concurrency
4. **Bundle shared library** (.dll/.so) with installer
5. **Load at startup** and maintain for application lifetime

### Key C API Functions

```c
// stable-diffusion.cpp C API (simplified)
typedef struct sd_ctx sd_ctx_t;

// Initialize context with model
sd_ctx_t* sd_load_from_file(
    const char* model_path,
    const char* vae_path,
    const char* taesd_path,
    const char* lora_model_dir,
    bool vae_decode_only,
    bool free_params_immediately
);

// Generate image from text
sd_image_t* txt2img(
    sd_ctx_t* ctx,
    const char* prompt,
    const char* negative_prompt,
    int clip_skip,
    float cfg_scale,
    int width,
    int height,
    int sample_method,
    int sample_steps,
    int64_t seed
);

// Clean up
void sd_free(sd_ctx_t* ctx);
```

### Model Format

Stable Diffusion models come in several formats:
- **SafeTensors (.safetensors):** PyTorch safe serialization format
- **Checkpoint (.ckpt):** Original SD checkpoint format
- **Diffusers:** HuggingFace directory structure

stable-diffusion.cpp supports SafeTensors natively, which is the recommended format for safety and compatibility.

**Selected Model:** Stable Diffusion v1.5
- **Source:** https://huggingface.co/runwayml/stable-diffusion-v1-5
- **Format:** SafeTensors
- **Size:** ~4GB (FP16)
- **VRAM:** 4-6GB during inference
- **Quality:** Proven, widely compatible, good balance

---

## CGo Binding Architecture

### Package Structure

```
sdruntime/
├── sdruntime.go          // Main Go API, context pool management
├── cgo_bindings.go       // CGo wrapper functions (import "C")
├── types.go              // Go types for SD parameters and results
├── errors.go             // Error handling and translation
├── context.go            // Context pool implementation
├── generate.go           // High-level generation API
├── image_utils.go        // PNG encoding, memory management
└── load_model.go         // Model loading and validation

lib/
├── stable-diffusion.dll  // Windows CUDA build
└── stable-diffusion.so   // Linux CUDA build

models/
└── sd-v1-5.safetensors   // Bundled Stable Diffusion v1.5 model
```

### CGo Bindings Pattern

Following the llama.cpp pattern established in Phase 2:

```go
// sdruntime/cgo_bindings.go
package sdruntime

/*
#cgo CFLAGS: -I${SRCDIR}/../lib/stable-diffusion
#cgo LDFLAGS: -L${SRCDIR}/../lib -lstable-diffusion
#cgo windows LDFLAGS: -lcuda -lcudart
#cgo linux LDFLAGS: -lcuda -lcudart

#include <stable_diffusion.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

// loadModel wraps sd_load_from_file
func loadModel(modelPath string) (*C.sd_ctx_t, error) {
    cModelPath := C.CString(modelPath)
    defer C.free(unsafe.Pointer(cModelPath))

    ctx := C.sd_load_from_file(
        cModelPath,
        nil, // vae_path (use default)
        nil, // taesd_path (not used)
        nil, // lora_model_dir (not used)
        C.bool(false), // vae_decode_only
        C.bool(false), // free_params_immediately
    )

    if ctx == nil {
        return nil, ErrModelLoadFailed
    }

    return ctx, nil
}

// generateImage wraps txt2img
func generateImage(ctx *C.sd_ctx_t, params GenerateParams) ([]byte, error) {
    cPrompt := C.CString(params.Prompt)
    cNegative := C.CString(params.NegativePrompt)
    defer C.free(unsafe.Pointer(cPrompt))
    defer C.free(unsafe.Pointer(cNegative))

    image := C.txt2img(
        ctx,
        cPrompt,
        cNegative,
        C.int(params.ClipSkip),
        C.float(params.CFGScale),
        C.int(params.Width),
        C.int(params.Height),
        C.int(params.SampleMethod),
        C.int(params.Steps),
        C.int64_t(params.Seed),
    )

    if image == nil {
        return nil, ErrGenerationFailed
    }
    defer C.sd_free_image(image)

    // Convert C image to Go bytes
    return imageToBytes(image), nil
}

// freeContext wraps sd_free
func freeContext(ctx *C.sd_ctx_t) {
    if ctx != nil {
        C.sd_free(ctx)
    }
}
```

### Context Pool Management

Thread-safe context pool for concurrent image generation:

```go
// sdruntime/context.go
package sdruntime

import (
    "context"
    "errors"
    "sync"
    "time"
)

var (
    ErrContextPoolClosed = errors.New("context pool is closed")
    ErrAcquireTimeout    = errors.New("timeout acquiring SD context")
)

// ContextPool manages stable-diffusion.cpp contexts for concurrent use
type ContextPool struct {
    contexts chan *SDContext
    maxSize  int
    mu       sync.Mutex
    closed   bool
}

// SDContext wraps a stable-diffusion.cpp context with metadata
type SDContext struct {
    ctx      *C.sd_ctx_t
    acquired time.Time
}

// NewContextPool creates a pool with the specified size
func NewContextPool(modelPath string, poolSize int) (*ContextPool, error) {
    pool := &ContextPool{
        contexts: make(chan *SDContext, poolSize),
        maxSize:  poolSize,
    }

    // Pre-load contexts
    for i := 0; i < poolSize; i++ {
        ctx, err := loadModel(modelPath)
        if err != nil {
            pool.Close() // Clean up any already loaded
            return nil, err
        }
        pool.contexts <- &SDContext{ctx: ctx}
    }

    return pool, nil
}

// Acquire gets a context from the pool with timeout
func (p *ContextPool) Acquire(ctx context.Context) (*SDContext, error) {
    p.mu.Lock()
    if p.closed {
        p.mu.Unlock()
        return nil, ErrContextPoolClosed
    }
    p.mu.Unlock()

    select {
    case sdCtx := <-p.contexts:
        sdCtx.acquired = time.Now()
        return sdCtx, nil
    case <-ctx.Done():
        return nil, ErrAcquireTimeout
    }
}

// Release returns a context to the pool
func (p *ContextPool) Release(sdCtx *SDContext) {
    p.mu.Lock()
    defer p.mu.Unlock()

    if !p.closed {
        p.contexts <- sdCtx
    } else {
        freeContext(sdCtx.ctx)
    }
}

// Close releases all contexts and closes the pool
func (p *ContextPool) Close() error {
    p.mu.Lock()
    defer p.mu.Unlock()

    if p.closed {
        return nil
    }
    p.closed = true

    close(p.contexts)
    for sdCtx := range p.contexts {
        freeContext(sdCtx.ctx)
    }

    return nil
}
```

### High-Level API

Clean Go API for image generation:

```go
// sdruntime/generate.go
package sdruntime

import (
    "context"
    "crypto/rand"
    "encoding/binary"
    "fmt"
)

// GenerateParams holds parameters for image generation
type GenerateParams struct {
    Prompt         string
    NegativePrompt string
    Width          int
    Height         int
    Steps          int
    CFGScale       float32
    Seed           int64
    ClipSkip       int
    SampleMethod   int
}

// DefaultParams returns sensible defaults for SD v1.5
func DefaultParams() GenerateParams {
    return GenerateParams{
        Prompt:         "",
        NegativePrompt: "ugly, blurry, low quality",
        Width:          512,
        Height:         512,
        Steps:          25,
        CFGScale:       7.5,
        Seed:           randomSeed(),
        ClipSkip:       -1,
        SampleMethod:   1, // Euler A
    }
}

// Generate creates an image from a text prompt
func (p *ContextPool) Generate(ctx context.Context, params GenerateParams) ([]byte, error) {
    // Acquire context from pool
    sdCtx, err := p.Acquire(ctx)
    if err != nil {
        return nil, fmt.Errorf("acquire context: %w", err)
    }
    defer p.Release(sdCtx)

    // Validate parameters
    if err := validateParams(params); err != nil {
        return nil, fmt.Errorf("invalid params: %w", err)
    }

    // Generate image via CGo
    imageData, err := generateImage(sdCtx.ctx, params)
    if err != nil {
        return nil, fmt.Errorf("generate image: %w", err)
    }

    return imageData, nil
}

// validateParams checks parameter ranges
func validateParams(p GenerateParams) error {
    if p.Width < 128 || p.Width > 2048 {
        return fmt.Errorf("width must be 128-2048")
    }
    if p.Height < 128 || p.Height > 2048 {
        return fmt.Errorf("height must be 128-2048")
    }
    if p.Steps < 1 || p.Steps > 100 {
        return fmt.Errorf("steps must be 1-100")
    }
    if p.CFGScale < 1.0 || p.CFGScale > 30.0 {
        return fmt.Errorf("cfg_scale must be 1.0-30.0")
    }
    return nil
}

// randomSeed generates a random seed
func randomSeed() int64 {
    var seed int64
    binary.Read(rand.Reader, binary.LittleEndian, &seed)
    if seed < 0 {
        seed = -seed
    }
    return seed
}
```

### Memory Safety Considerations

Critical CGo safety practices:

1. **Always free C strings:** Use `defer C.free(unsafe.Pointer(cStr))` immediately after `C.CString()`
2. **Validate pointers:** Check for `nil` before dereferencing C pointers
3. **Copy C data to Go:** Don't retain pointers to C memory beyond function scope
4. **Use defer for cleanup:** Ensure C resources freed even on error paths
5. **Context pool:** Manage GPU contexts carefully, prevent leaks
6. **Panic recovery:** Wrap CGo calls in recover() to prevent crashes

---

## Image Generation Pipeline

### End-to-End Flow

```
┌─────────────────────────────────────────────────────────────┐
│ 1. User writes prompt: {{image: sunset over mountains}}    │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│ 2. Monitor detects {{image: }} prefix in handleUpdate()    │
│    - Extract prompt: "sunset over mountains"                │
│    - Get parent widget coordinates                          │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│ 3. Route to imagegen.ProcessImagePrompt()                  │
│    - Create processing indicator note: "Generating..."      │
│    - Build GenerateParams from prompt + config              │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│ 4. Call sdruntime.Generate(ctx, params)                    │
│    - Acquire SD context from pool                           │
│    - Pass to stable-diffusion.cpp via CGo                   │
│    - CUDA generates image (20-30 seconds)                   │
│    - Return PNG bytes                                       │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│ 5. Save to temporary file                                   │
│    - downloads/sd-{uuid}.png                                │
│    - Validate image data                                    │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│ 6. Calculate placement coordinates                          │
│    - Parent X + offset (e.g., +300)                         │
│    - Parent Y + offset (e.g., +50)                          │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│ 7. Upload to Canvus via canvusapi.UploadImage()            │
│    - Multipart form upload                                  │
│    - Set position, parent, zIndex                           │
│    - Retry on transient failures                            │
└─────────────────────────┬───────────────────────────────────┘
                          │
┌─────────────────────────▼───────────────────────────────────┐
│ 8. Clean up                                                 │
│    - Delete processing indicator note                       │
│    - Remove temporary file                                  │
│    - Release SD context to pool                             │
└─────────────────────────────────────────────────────────────┘
```

### Implementation: imagegen Package

Create new `imagegen/` organism for image generation pipeline:

```go
// imagegen/processor.go
package imagegen

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "time"

    "github.com/google/uuid"
    "canvuslocallm/canvusapi"
    "canvuslocallm/core"
    "canvuslocallm/sdruntime"
)

// Processor handles image generation requests
type Processor struct {
    client   *canvusapi.Client
    sdPool   *sdruntime.ContextPool
    config   *core.Config
    tempDir  string
}

// NewProcessor creates an image generation processor
func NewProcessor(client *canvusapi.Client, sdPool *sdruntime.ContextPool, config *core.Config) *Processor {
    return &Processor{
        client:  client,
        sdPool:  sdPool,
        config:  config,
        tempDir: config.DownloadsDir,
    }
}

// ProcessImagePrompt generates an image and places it on canvas
func (p *Processor) ProcessImagePrompt(ctx context.Context, prompt string, parentWidget Widget) error {
    // Create processing indicator
    processingNote := p.createProcessingNote(parentWidget)
    defer p.client.DeleteWidget(processingNote.ID)

    // Build generation parameters
    params := p.buildParams(prompt)

    // Generate image via SD runtime
    ctx, cancel := context.WithTimeout(ctx, time.Duration(p.config.SDTimeoutSeconds)*time.Second)
    defer cancel()

    imageData, err := p.sdPool.Generate(ctx, params)
    if err != nil {
        return fmt.Errorf("generate image: %w", err)
    }

    // Save to temporary file
    tempPath, err := p.saveTempImage(imageData)
    if err != nil {
        return fmt.Errorf("save temp image: %w", err)
    }
    defer os.Remove(tempPath)

    // Calculate placement
    x, y := p.calculatePlacement(parentWidget)

    // Upload to canvas
    if err := p.client.UploadImage(tempPath, x, y, parentWidget.ParentID); err != nil {
        return fmt.Errorf("upload image: %w", err)
    }

    return nil
}

// buildParams constructs GenerateParams from prompt and config
func (p *Processor) buildParams(prompt string) sdruntime.GenerateParams {
    params := sdruntime.DefaultParams()
    params.Prompt = prompt

    // Apply config overrides
    if p.config.SDImageSize > 0 {
        params.Width = p.config.SDImageSize
        params.Height = p.config.SDImageSize
    }
    if p.config.SDInferenceSteps > 0 {
        params.Steps = p.config.SDInferenceSteps
    }
    if p.config.SDGuidanceScale > 0 {
        params.CFGScale = p.config.SDGuidanceScale
    }

    return params
}

// saveTempImage writes image data to temporary file
func (p *Processor) saveTempImage(imageData []byte) (string, error) {
    filename := fmt.Sprintf("sd-%s.png", uuid.New().String())
    tempPath := filepath.Join(p.tempDir, filename)

    if err := os.WriteFile(tempPath, imageData, 0644); err != nil {
        return "", fmt.Errorf("write file: %w", err)
    }

    return tempPath, nil
}

// calculatePlacement determines image position relative to parent widget
func (p *Processor) calculatePlacement(parentWidget Widget) (x, y float64) {
    // Place to the right and slightly below the prompt widget
    x = parentWidget.X + 300.0
    y = parentWidget.Y + 50.0
    return
}

// createProcessingNote shows "Generating image..." indicator
func (p *Processor) createProcessingNote(parentWidget Widget) *canvusapi.Widget {
    x, y := p.calculatePlacement(parentWidget)
    note, _ := p.client.CreateNote("Generating image...", x-50, y-20, parentWidget.ParentID)
    return note
}
```

### Prompt Detection Enhancement

Extend `monitorcanvus.go` to detect image generation prompts:

```go
// monitorcanvus.go
func (m *Monitor) handleUpdate(update Update) {
    // ... existing code ...

    // Check for AI prompts
    promptPattern := regexp.MustCompile(`\{\{(.+?)\}\}`)
    matches := promptPattern.FindAllStringSubmatch(update.Widget.Content, -1)

    for _, match := range matches {
        fullPrompt := match[1] // Content inside {{ }}

        // Check for image generation prefix
        if strings.HasPrefix(strings.TrimSpace(fullPrompt), "image:") {
            // Extract prompt after "image:"
            imagePrompt := strings.TrimSpace(strings.TrimPrefix(fullPrompt, "image:"))

            // Route to image generation
            go m.imageProcessor.ProcessImagePrompt(context.Background(), imagePrompt, update.Widget)
        } else {
            // Route to text generation (existing behavior)
            go processAINote(context.Background(), m.client, fullPrompt, update, m.config)
        }
    }
}
```

---

## Canvas Integration

### Image Upload

Use existing `canvusapi.UploadImage()` method:

```go
// canvusapi/canvusapi.go (existing)
func (c *Client) UploadImage(imagePath string, x, y float64, parentID string) (*Widget, error) {
    file, err := os.Open(imagePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    // Add file
    part, err := writer.CreateFormFile("file", filepath.Base(imagePath))
    if err != nil {
        return nil, err
    }
    io.Copy(part, file)

    // Add metadata
    writer.WriteField("x", fmt.Sprintf("%f", x))
    writer.WriteField("y", fmt.Sprintf("%f", y))
    writer.WriteField("parentId", parentID)
    writer.WriteField("type", "image")

    writer.Close()

    // POST to /api/widgets/image
    req, _ := http.NewRequest("POST", c.baseURL+"/api/widgets/image", body)
    req.Header.Set("Content-Type", writer.FormDataContentType())
    req.Header.Set("X-Api-Key", c.apiKey)

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Parse response
    var widget Widget
    json.NewDecoder(resp.Body).Decode(&widget)
    return &widget, nil
}
```

### Coordinate System

**Critical:** Canvus widget positions are **relative to parent widget**, not absolute canvas coordinates.

```
Canvas (root)
├── Parent Widget (x=100, y=200)
│   ├── Prompt Widget (relative x=50, y=30)  → Absolute: (150, 230)
│   └── Image Widget (relative x=350, y=80)  → Absolute: (450, 280)
```

**Placement Strategy:**
- Get prompt widget's relative coordinates (x, y)
- Add fixed offset: (x + 300, y + 50)
- This places image to the right and slightly below prompt
- Both widgets share same parent, so coordinates are relative to that parent

**Alternative Strategies (future):**
- Smart placement: Find empty space on canvas
- Configurable offset: Allow user to set X/Y offset in .env
- Directional placement: Left, right, above, below based on available space

---

## Model Selection and Management

### Selected Model: Stable Diffusion v1.5

**Source:** https://huggingface.co/runwayml/stable-diffusion-v1-5

**Rationale:**
1. **Proven Quality:** Widely used, well-documented, reliable results
2. **Reasonable Size:** ~4GB (FP16) fits in installer and VRAM
3. **VRAM Efficient:** Works well on 6-8GB GPUs
4. **Compatibility:** Excellent stable-diffusion.cpp support
5. **License:** CreativeML Open RAIL-M (permissive for end-user generation)

**Specifications:**
- **Architecture:** Latent Diffusion Model (LDM)
- **Resolution:** 512x512 native (can do 768x768 with more VRAM)
- **Format:** SafeTensors (FP16)
- **File Size:** ~4GB
- **VRAM Usage:** 4-6GB during inference
- **Inference Speed:** 20-30 seconds on RTX 3060 (25 steps)

**Files:**
```
models/
└── sd-v1-5.safetensors  (4.27 GB)
```

**Download:**
```bash
# From HuggingFace
wget https://huggingface.co/runwayml/stable-diffusion-v1-5/resolve/main/v1-5-pruned-emaonly.safetensors
mv v1-5-pruned-emaonly.safetensors models/sd-v1-5.safetensors
```

### Model Bundling Strategy

**Decision: Bundle with Installer**

**Pros:**
- True zero-configuration experience
- Works completely offline
- No first-run download wait
- Aligns with product philosophy

**Cons:**
- Larger installer (~4.5GB total with Go binary + llama.cpp + SD model)
- Longer download time for installer

**Implementation:**
- Include `models/sd-v1-5.safetensors` in NSIS installer and .deb package
- Verify SHA256 checksum on application startup
- If missing or corrupted, display helpful error message

### Model Loading

```go
// main.go
func main() {
    // ... existing setup ...

    // Initialize Stable Diffusion runtime
    sdModelPath := filepath.Join(installDir, "models", "sd-v1-5.safetensors")

    // Verify model exists
    if _, err := os.Stat(sdModelPath); os.IsNotExist(err) {
        log.Fatal("SD model not found. Please reinstall application.")
    }

    // Create SD context pool
    poolSize := config.SDMaxConcurrent // Default: 2
    sdPool, err := sdruntime.NewContextPool(sdModelPath, poolSize)
    if err != nil {
        log.Fatalf("Failed to initialize SD runtime: %v", err)
    }
    defer sdPool.Close()

    logging.LogHandler(fmt.Sprintf("Initialized SD runtime with %d contexts", poolSize))

    // Create image processor
    imageProcessor := imagegen.NewProcessor(client, sdPool, config)

    // Pass to Monitor
    monitor := NewMonitor(client, config, sdPool, imageProcessor)

    // ... rest of main ...
}
```

### Future Model Considerations

**Potential Upgrades (Out of Scope for Phase 3):**
- **SDXL-Turbo:** Faster inference, good quality, ~4GB
- **SD v2.1:** Better quality, larger (~5GB), higher VRAM
- **Custom fine-tunes:** Artistic styles, specific domains
- **LoRA support:** Lightweight style customization

**Model Switching (Not in Scope):**
- Phase 3 uses single hardcoded model
- No user-facing model selection UI
- Future phases could add model management

---

## Configuration

### Environment Variables

Add to existing `.env` file (all optional with defaults):

```env
# Image Generation Configuration (all optional)
SD_IMAGE_SIZE=512              # Image dimensions (512x512). Options: 512, 768
SD_INFERENCE_STEPS=25          # Number of denoising steps (1-100). Default: 25
SD_GUIDANCE_SCALE=7.5          # CFG scale (1.0-30.0). Default: 7.5
SD_TIMEOUT_SECONDS=60          # Generation timeout. Default: 60
SD_MAX_CONCURRENT=2            # Max concurrent generations. Default: 2
SD_NEGATIVE_PROMPT=            # Default negative prompt. Default: "ugly, blurry, low quality"
```

### Config Struct Extension

```go
// core/config.go
type Config struct {
    // ... existing fields ...

    // Image Generation
    SDImageSize       int     `env:"SD_IMAGE_SIZE" default:"512"`
    SDInferenceSteps  int     `env:"SD_INFERENCE_STEPS" default:"25"`
    SDGuidanceScale   float32 `env:"SD_GUIDANCE_SCALE" default:"7.5"`
    SDTimeoutSeconds  int     `env:"SD_TIMEOUT_SECONDS" default:"60"`
    SDMaxConcurrent   int     `env:"SD_MAX_CONCURRENT" default:"2"`
    SDNegativePrompt  string  `env:"SD_NEGATIVE_PROMPT" default:"ugly, blurry, low quality"`
}

func LoadConfig() (*Config, error) {
    // ... existing code ...

    // Validate SD config
    if config.SDImageSize != 512 && config.SDImageSize != 768 {
        return nil, fmt.Errorf("SD_IMAGE_SIZE must be 512 or 768")
    }
    if config.SDInferenceSteps < 1 || config.SDInferenceSteps > 100 {
        return nil, fmt.Errorf("SD_INFERENCE_STEPS must be 1-100")
    }
    if config.SDGuidanceScale < 1.0 || config.SDGuidanceScale > 30.0 {
        return nil, fmt.Errorf("SD_GUIDANCE_SCALE must be 1.0-30.0")
    }
    if config.SDMaxConcurrent < 1 || config.SDMaxConcurrent > 5 {
        return nil, fmt.Errorf("SD_MAX_CONCURRENT must be 1-5")
    }

    return config, nil
}
```

### Default Behavior

**Zero-Config Principle:** Application works perfectly without any SD configuration variables. Defaults are optimized for RTX 3060-class GPUs:

- **512x512 images:** Fast generation, low VRAM, good quality
- **25 steps:** Balance of speed and quality
- **CFG 7.5:** Standard guidance scale
- **2 concurrent:** Prevents VRAM exhaustion
- **60s timeout:** Reasonable for most generations

---

## Error Handling

### Error Types

```go
// sdruntime/errors.go
package sdruntime

import "errors"

var (
    // Model errors
    ErrModelNotFound    = errors.New("SD model file not found")
    ErrModelLoadFailed  = errors.New("failed to load SD model")
    ErrModelCorrupted   = errors.New("SD model file corrupted (checksum mismatch)")

    // Generation errors
    ErrGenerationFailed = errors.New("image generation failed")
    ErrGenerationTimeout = errors.New("image generation timed out")
    ErrInvalidPrompt    = errors.New("invalid prompt (empty or too long)")
    ErrInvalidParams    = errors.New("invalid generation parameters")

    // Resource errors
    ErrCUDANotAvailable = errors.New("CUDA not available (NVIDIA GPU required)")
    ErrOutOfVRAM        = errors.New("out of VRAM (reduce image size or concurrent generations)")
    ErrContextPoolClosed = errors.New("SD context pool is closed")
    ErrAcquireTimeout    = errors.New("timeout acquiring SD context from pool")
)
```

### Error Handling Strategy

**1. Startup Errors (Fatal):**
```go
// main.go
func initSDRuntime(config *core.Config) (*sdruntime.ContextPool, error) {
    modelPath := filepath.Join(installDir, "models", "sd-v1-5.safetensors")

    // Check model exists
    if _, err := os.Stat(modelPath); os.IsNotExist(err) {
        return nil, fmt.Errorf("SD model not found at %s. Please reinstall application.", modelPath)
    }

    // Check CUDA availability
    if !sdruntime.IsCUDAAvailable() {
        return nil, fmt.Errorf("CUDA not available. NVIDIA GPU with CUDA support required.")
    }

    // Create context pool
    pool, err := sdruntime.NewContextPool(modelPath, config.SDMaxConcurrent)
    if err != nil {
        if errors.Is(err, sdruntime.ErrOutOfVRAM) {
            return nil, fmt.Errorf("insufficient VRAM. Try reducing SD_MAX_CONCURRENT to 1.")
        }
        return nil, fmt.Errorf("failed to initialize SD runtime: %w", err)
    }

    return pool, nil
}
```

**2. Runtime Errors (Recoverable):**
```go
// imagegen/processor.go
func (p *Processor) ProcessImagePrompt(ctx context.Context, prompt string, widget Widget) error {
    defer func() {
        if r := recover(); r != nil {
            logging.LogHandler(fmt.Sprintf("Panic in image generation: %v", r))
            p.createErrorNote("Image generation failed. Please try again.", widget)
        }
    }()

    // Generate image
    imageData, err := p.sdPool.Generate(ctx, params)
    if err != nil {
        switch {
        case errors.Is(err, sdruntime.ErrGenerationTimeout):
            p.createErrorNote("Image generation timed out. Try reducing SD_INFERENCE_STEPS.", widget)
        case errors.Is(err, sdruntime.ErrOutOfVRAM):
            p.createErrorNote("Out of VRAM. Try reducing SD_IMAGE_SIZE or close other applications.", widget)
        case errors.Is(err, sdruntime.ErrAcquireTimeout):
            p.createErrorNote("Image generation queue full. Please wait and try again.", widget)
        default:
            p.createErrorNote(fmt.Sprintf("Image generation failed: %v", err), widget)
        }
        return err
    }

    // ... rest of processing ...
}
```

**3. User-Facing Error Messages:**

Display errors as notes on canvas near the prompt:

```go
func (p *Processor) createErrorNote(message string, widget Widget) {
    x, y := p.calculatePlacement(widget)
    p.client.CreateNote(fmt.Sprintf("❌ %s", message), x, y, widget.ParentID)
}
```

### Logging

Log all errors to `app.log` with context:

```go
logging.LogHandler(fmt.Sprintf("[SD] Generation failed: prompt='%s', error=%v", prompt, err))
logging.LogHandler(fmt.Sprintf("[SD] VRAM exhausted: current_concurrent=%d, max=%d", active, max))
logging.LogHandler(fmt.Sprintf("[SD] Model loaded: path=%s, vram=%dMB", path, vram))
```

---

## Testing Strategy

### Unit Tests

**Package: sdruntime/**

```go
// sdruntime/sdruntime_test.go
package sdruntime_test

import (
    "context"
    "testing"
    "time"

    "canvuslocallm/sdruntime"
)

func TestContextPool_CreateAndClose(t *testing.T) {
    // Test context pool creation and cleanup
    pool, err := sdruntime.NewContextPool("testdata/mock-model.safetensors", 2)
    if err != nil {
        t.Fatalf("Failed to create pool: %v", err)
    }
    defer pool.Close()

    // Verify pool size
    // ...
}

func TestContextPool_AcquireRelease(t *testing.T) {
    pool := setupTestPool(t)
    defer pool.Close()

    ctx := context.Background()

    // Acquire context
    sdCtx, err := pool.Acquire(ctx)
    if err != nil {
        t.Fatalf("Failed to acquire: %v", err)
    }

    // Release context
    pool.Release(sdCtx)

    // Verify can acquire again
    // ...
}

func TestContextPool_AcquireTimeout(t *testing.T) {
    pool := setupTestPool(t)
    defer pool.Close()

    // Acquire all contexts
    ctx1, _ := pool.Acquire(context.Background())
    ctx2, _ := pool.Acquire(context.Background())

    // Next acquire should timeout
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    _, err := pool.Acquire(ctx)
    if err != sdruntime.ErrAcquireTimeout {
        t.Errorf("Expected timeout error, got: %v", err)
    }

    pool.Release(ctx1)
    pool.Release(ctx2)
}

func TestGenerate_ValidPrompt(t *testing.T) {
    pool := setupTestPool(t)
    defer pool.Close()

    params := sdruntime.DefaultParams()
    params.Prompt = "a sunset over mountains"

    ctx := context.Background()
    imageData, err := pool.Generate(ctx, params)
    if err != nil {
        t.Fatalf("Generation failed: %v", err)
    }

    // Verify image data
    if len(imageData) == 0 {
        t.Error("Expected image data, got empty")
    }

    // Verify PNG signature
    if !isPNG(imageData) {
        t.Error("Generated data is not valid PNG")
    }
}

func TestGenerate_InvalidParams(t *testing.T) {
    pool := setupTestPool(t)
    defer pool.Close()

    tests := []struct {
        name   string
        params sdruntime.GenerateParams
        wantErr error
    }{
        {"empty prompt", sdruntime.GenerateParams{Prompt: ""}, sdruntime.ErrInvalidPrompt},
        {"invalid width", sdruntime.GenerateParams{Width: 10000}, sdruntime.ErrInvalidParams},
        {"invalid steps", sdruntime.GenerateParams{Steps: 0}, sdruntime.ErrInvalidParams},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := pool.Generate(context.Background(), tt.params)
            if !errors.Is(err, tt.wantErr) {
                t.Errorf("Expected %v, got %v", tt.wantErr, err)
            }
        })
    }
}
```

**Package: imagegen/**

```go
// imagegen/processor_test.go
package imagegen_test

import (
    "context"
    "testing"

    "canvuslocallm/imagegen"
)

func TestProcessor_ProcessImagePrompt(t *testing.T) {
    // Use mock Canvus client and SD pool
    mockClient := &MockCanvasClient{}
    mockSDPool := &MockSDPool{}
    config := &core.Config{SDImageSize: 512}

    processor := imagegen.NewProcessor(mockClient, mockSDPool, config)

    widget := Widget{X: 100, Y: 200, ParentID: "parent-123"}
    err := processor.ProcessImagePrompt(context.Background(), "test prompt", widget)

    if err != nil {
        t.Errorf("ProcessImagePrompt failed: %v", err)
    }

    // Verify image was uploaded
    if !mockClient.UploadImageCalled {
        t.Error("Expected UploadImage to be called")
    }
}

func TestCalculatePlacement(t *testing.T) {
    processor := setupTestProcessor(t)

    widget := Widget{X: 100, Y: 200}
    x, y := processor.calculatePlacement(widget)

    // Verify offset applied
    expectedX := 400.0 // 100 + 300
    expectedY := 250.0 // 200 + 50

    if x != expectedX || y != expectedY {
        t.Errorf("Expected (%f, %f), got (%f, %f)", expectedX, expectedY, x, y)
    }
}
```

### Integration Tests

```go
// tests/image_generation_test.go
package tests

import (
    "context"
    "os"
    "testing"
    "time"

    "canvuslocallm/sdruntime"
)

func TestImageGeneration_EndToEnd(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Verify CUDA available
    if !sdruntime.IsCUDAAvailable() {
        t.Skip("CUDA not available, skipping")
    }

    // Load actual model
    modelPath := "../models/sd-v1-5.safetensors"
    if _, err := os.Stat(modelPath); os.IsNotExist(err) {
        t.Skip("Model not found, skipping")
    }

    pool, err := sdruntime.NewContextPool(modelPath, 1)
    if err != nil {
        t.Fatalf("Failed to create pool: %v", err)
    }
    defer pool.Close()

    // Generate image
    params := sdruntime.DefaultParams()
    params.Prompt = "a simple red circle"
    params.Steps = 10 // Faster for testing

    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    start := time.Now()
    imageData, err := pool.Generate(ctx, params)
    duration := time.Since(start)

    if err != nil {
        t.Fatalf("Generation failed: %v", err)
    }

    t.Logf("Generation took %v", duration)

    // Verify image
    if len(imageData) < 1000 {
        t.Error("Generated image suspiciously small")
    }

    // Save for manual inspection
    os.WriteFile("test-output.png", imageData, 0644)
}

func TestConcurrentGeneration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    pool := setupTestPoolWithModel(t, 2)
    defer pool.Close()

    // Generate 5 images concurrently (pool size 2)
    const numImages = 5
    results := make(chan error, numImages)

    for i := 0; i < numImages; i++ {
        go func(n int) {
            params := sdruntime.DefaultParams()
            params.Prompt = fmt.Sprintf("test image %d", n)
            params.Steps = 10

            ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
            defer cancel()

            _, err := pool.Generate(ctx, params)
            results <- err
        }(i)
    }

    // Collect results
    for i := 0; i < numImages; i++ {
        if err := <-results; err != nil {
            t.Errorf("Concurrent generation %d failed: %v", i, err)
        }
    }
}
```

### Performance Tests

```go
// tests/performance_test.go
package tests

func BenchmarkImageGeneration512(b *testing.B) {
    pool := setupTestPoolWithModel(b, 1)
    defer pool.Close()

    params := sdruntime.DefaultParams()
    params.Prompt = "benchmark test image"
    params.Width = 512
    params.Height = 512
    params.Steps = 25

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := pool.Generate(context.Background(), params)
        if err != nil {
            b.Fatalf("Generation failed: %v", err)
        }
    }
}

func BenchmarkImageGeneration768(b *testing.B) {
    // Similar but with 768x768
}
```

### Platform Testing

**Manual Testing Checklist:**

Windows 10/11 with NVIDIA RTX GPU:
- [ ] Install via NSIS installer
- [ ] Verify model bundled correctly
- [ ] Generate 512x512 image successfully
- [ ] Generate 768x768 image successfully
- [ ] Verify CUDA acceleration used (check GPU utilization)
- [ ] Test concurrent generations
- [ ] Verify error handling (invalid prompt, timeout)
- [ ] Check VRAM usage stays within bounds

Linux (Ubuntu 22.04) with NVIDIA RTX GPU:
- [ ] Install via .deb package
- [ ] Same test suite as Windows

### Test Coverage Goals

- **Unit Tests:** >80% coverage for sdruntime/ and imagegen/
- **Integration Tests:** End-to-end generation pipeline
- **Performance Tests:** Verify <30s for 512x512 on RTX 3060
- **Error Scenario Tests:** All error paths covered
- **Platform Tests:** Windows and Linux both validated

---

## Build and Deployment

### Build Process

**Step 1: Build stable-diffusion.cpp Native Library**

Windows (PowerShell):
```powershell
# Prerequisites: Visual Studio 2022, CUDA Toolkit 11.8+, CMake 3.18+

cd stable-diffusion.cpp
mkdir build
cd build

cmake .. `
  -G "Visual Studio 17 2022" `
  -A x64 `
  -DGGML_CUDA=ON `
  -DCMAKE_BUILD_TYPE=Release `
  -DCMAKE_CUDA_ARCHITECTURES="75;86;89"

cmake --build . --config Release

# Output: Release/stable-diffusion.dll
copy Release\stable-diffusion.dll ..\..\lib\
```

Linux (Bash):
```bash
# Prerequisites: GCC 9+, CUDA Toolkit 11.8+, CMake 3.18+

cd stable-diffusion.cpp
mkdir build
cd build

cmake .. \
  -DGGML_CUDA=ON \
  -DCMAKE_BUILD_TYPE=Release \
  -DCMAKE_CUDA_ARCHITECTURES="75;86;89"

make -j$(nproc)

# Output: libstable-diffusion.so
cp libstable-diffusion.so ../../lib/
```

CUDA Architectures:
- 75: RTX 20xx (Turing)
- 86: RTX 30xx (Ampere)
- 89: RTX 40xx (Ada Lovelace)

**Step 2: Download Stable Diffusion Model**

```bash
# Download SD v1.5 model
mkdir -p models
cd models
wget https://huggingface.co/runwayml/stable-diffusion-v1-5/resolve/main/v1-5-pruned-emaonly.safetensors
mv v1-5-pruned-emaonly.safetensors sd-v1-5.safetensors

# Verify checksum (SHA256)
sha256sum sd-v1-5.safetensors
# Expected: [insert checksum]
```

**Step 3: Build Go Application**

```bash
# Enable CGo
export CGO_ENABLED=1

# Windows
GOOS=windows GOARCH=amd64 go build -o CanvusLocalLLM.exe .

# Linux
GOOS=linux GOARCH=amd64 go build -o canvuslocallm .
```

**Step 4: Create Installer Packages**

Windows (NSIS):
```nsis
; installer.nsi
!include "MUI2.nsh"

Name "CanvusLocalLLM"
OutFile "CanvusLocalLLM-Setup.exe"
InstallDir "$PROGRAMFILES64\CanvusLocalLLM"

Section "Main" SecMain
  SetOutPath "$INSTDIR"

  ; Application binary
  File "CanvusLocalLLM.exe"

  ; Native libraries
  File "lib\llama.dll"
  File "lib\stable-diffusion.dll"

  ; CUDA runtime (if not system-installed)
  File "lib\cudart64_11.dll"

  ; Models
  SetOutPath "$INSTDIR\models"
  File "models\bunny-v1.1-llama-3-8b-v.gguf"
  File "models\sd-v1-5.safetensors"

  ; Config template
  SetOutPath "$INSTDIR"
  File ".env.example"

  ; Create uninstaller
  WriteUninstaller "$INSTDIR\Uninstall.exe"
SectionEnd
```

Linux (.deb):
```bash
# Create package structure
mkdir -p canvuslocallm_1.0.0_amd64/DEBIAN
mkdir -p canvuslocallm_1.0.0_amd64/opt/canvuslocallm/{lib,models}

# Control file
cat > canvuslocallm_1.0.0_amd64/DEBIAN/control << EOF
Package: canvuslocallm
Version: 1.0.0
Architecture: amd64
Maintainer: CanvusLocalLLM Team
Depends: libc6, nvidia-cuda-toolkit (>= 11.8)
Description: Local AI integration for Canvus workspaces
EOF

# Copy files
cp canvuslocallm canvuslocallm_1.0.0_amd64/opt/canvuslocallm/
cp lib/*.so canvuslocallm_1.0.0_amd64/opt/canvuslocallm/lib/
cp models/*.{gguf,safetensors} canvuslocallm_1.0.0_amd64/opt/canvuslocallm/models/
cp .env.example canvuslocallm_1.0.0_amd64/opt/canvuslocallm/

# Build package
dpkg-deb --build canvuslocallm_1.0.0_amd64
```

### Directory Structure (Post-Install)

```
Windows: C:\Program Files\CanvusLocalLLM\
Linux:   /opt/canvuslocallm/

├── CanvusLocalLLM.exe (or canvuslocallm)
├── lib/
│   ├── llama.dll (or .so)
│   ├── stable-diffusion.dll (or .so)
│   └── cudart64_11.dll (Windows only, if bundled)
├── models/
│   ├── bunny-v1.1-llama-3-8b-v.gguf (~8GB)
│   └── sd-v1-5.safetensors (~4GB)
├── .env.example
├── .env (created by user)
├── downloads/ (created at runtime)
├── app.log (created at runtime)
└── README.txt
```

### Installation Steps (User Perspective)

**Windows:**
1. Download `CanvusLocalLLM-Setup.exe` (~13GB)
2. Run installer, accept license, choose install location
3. Installer extracts all files (binary, libraries, models)
4. Copy `.env.example` to `.env` and add Canvus credentials
5. Run `CanvusLocalLLM.exe`
6. Application loads models and starts monitoring

**Linux:**
1. Download `canvuslocallm_1.0.0_amd64.deb` (~13GB)
2. Install: `sudo dpkg -i canvuslocallm_1.0.0_amd64.deb`
3. Configure: `cd /opt/canvuslocallm && cp .env.example .env`
4. Edit `.env` with Canvus credentials
5. Run: `/opt/canvuslocallm/canvuslocallm`

Total installation time: 5-10 minutes (including download)

### CI/CD Integration

**GitHub Actions Workflow:**

```yaml
# .github/workflows/build.yml
name: Build Installers

on:
  push:
    tags:
      - 'v*'

jobs:
  build-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup CUDA
        uses: Jimver/cuda-toolkit@v0.2.11
        with:
          cuda: '11.8.0'

      - name: Build stable-diffusion.cpp
        run: |
          cd stable-diffusion.cpp
          mkdir build && cd build
          cmake .. -G "Visual Studio 17 2022" -DGGML_CUDA=ON
          cmake --build . --config Release
          copy Release\stable-diffusion.dll ..\..\lib\

      - name: Build Go Application
        run: |
          $env:CGO_ENABLED = "1"
          go build -o CanvusLocalLLM.exe .

      - name: Download Models
        run: |
          # Download SD v1.5 and Bunny v1.1

      - name: Build NSIS Installer
        run: |
          makensis installer.nsi

      - name: Upload Installer
        uses: actions/upload-artifact@v3
        with:
          name: CanvusLocalLLM-Setup.exe
          path: CanvusLocalLLM-Setup.exe

  build-linux:
    runs-on: ubuntu-latest
    steps:
      # Similar steps for Linux .deb build
```

---

## Performance Considerations

### Target Performance Metrics

**Hardware Baseline:** NVIDIA RTX 3060 (12GB VRAM)

| Image Size | Steps | Expected Time | VRAM Usage |
|------------|-------|---------------|------------|
| 512x512    | 20    | 15-20s        | 4GB        |
| 512x512    | 25    | 20-25s        | 4GB        |
| 512x512    | 30    | 25-30s        | 4GB        |
| 768x768    | 20    | 35-40s        | 6GB        |
| 768x768    | 25    | 45-50s        | 6GB        |
| 768x768    | 30    | 55-60s        | 6GB        |

**Concurrent Operations:**
- Text generation (llama.cpp): ~2-4GB VRAM
- Image generation (SD): ~4-6GB VRAM
- Total: 6-10GB VRAM for concurrent text + image
- Recommendation: RTX 3060 (12GB) or better

### Optimization Strategies

**1. Context Pooling:**
- Pre-load SD contexts at startup
- Reuse contexts across generations
- Avoid reload overhead (~5-10s per load)

**2. VRAM Management:**
- Limit concurrent SD generations (default: 2)
- Monitor VRAM usage, reject requests if low
- Consider offloading inactive contexts to system RAM (future)

**3. Model Quantization:**
- Use FP16 model (half precision) vs FP32
- Reduces VRAM by ~50% with minimal quality loss
- Consider Q8_0 quantization for even lower VRAM (future)

**4. Batching (Future):**
- Batch multiple prompts into single generation
- Amortize context loading overhead
- Requires stable-diffusion.cpp batch API support

**5. Async Processing:**
- Non-blocking image generation
- Monitor shows "Generating..." indicator
- User can continue working on canvas

### Performance Monitoring

Log generation metrics:

```go
// After generation completes
logging.LogHandler(fmt.Sprintf(
    "[SD] Generated %dx%d image in %v (steps=%d, vram=%dMB)",
    width, height, duration, steps, vramUsage,
))
```

Planned metrics (Phase 6):
- Average generation time by size
- VRAM usage over time
- Success/failure rate
- Queue depth and wait times

---

## Security Considerations

### Input Validation

**Prompt Sanitization:**
```go
func validatePrompt(prompt string) error {
    // Max length (SD models have token limits)
    if len(prompt) > 1000 {
        return fmt.Errorf("prompt too long (max 1000 chars)")
    }

    // Prevent empty prompts
    if strings.TrimSpace(prompt) == "" {
        return fmt.Errorf("prompt cannot be empty")
    }

    // No null bytes (C string safety)
    if strings.Contains(prompt, "\x00") {
        return fmt.Errorf("prompt contains invalid characters")
    }

    return nil
}
```

**Parameter Validation:**
- Enforce ranges for image size, steps, CFG scale
- Prevent resource exhaustion attacks (e.g., 10000 steps)
- Limit concurrent generations per user (if multi-user in future)

### CGo Memory Safety

**Critical Practices:**
1. **Always free C strings:** Use `defer C.free()` immediately
2. **Copy C data to Go:** Don't retain pointers beyond function scope
3. **Validate pointers:** Check for `nil` before dereferencing
4. **Bounds checking:** Validate array access in C layer
5. **Panic recovery:** Wrap CGo calls to prevent crashes

```go
func (p *ContextPool) Generate(ctx context.Context, params GenerateParams) (imageData []byte, err error) {
    // Recover from CGo panics
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("CGo panic: %v", r)
            logging.LogHandler(fmt.Sprintf("[SD] Panic recovered: %v", r))
        }
    }()

    // ... rest of function ...
}
```

### Resource Limits

**Prevent Abuse:**
- Timeout all generations (default: 60s)
- Limit queue depth (reject if queue full)
- Monitor VRAM, reject if insufficient
- Rate limiting per user (future, if multi-user)

### Data Privacy

**Guarantee Local Processing:**
- No network calls during image generation
- All inference via local stable-diffusion.cpp
- Prompts never transmitted externally
- Generated images stored temporarily, deleted after upload

**Logging Privacy:**
- Don't log full prompts (only length/hash)
- Redact sensitive info in error messages
- User can disable logging (future)

### Model Integrity

**Verify Model Files:**
```go
func verifyModelChecksum(modelPath string) error {
    const expectedChecksum = "abc123..." // SHA256 of sd-v1-5.safetensors

    f, err := os.Open(modelPath)
    if err != nil {
        return err
    }
    defer f.Close()

    h := sha256.New()
    if _, err := io.Copy(h, f); err != nil {
        return err
    }

    checksum := hex.EncodeToString(h.Sum(nil))
    if checksum != expectedChecksum {
        return ErrModelCorrupted
    }

    return nil
}
```

**Bundled Model Security:**
- Download from official HuggingFace repository
- Verify checksum before bundling
- Sign installer packages (Windows code signing)
- Publish checksums with releases

---

## Acceptance Criteria

### Phase 3 Complete When:

**Core Functionality:**
- [ ] `sdruntime/` package implements stable-diffusion.cpp CGo bindings
- [ ] Context pool manages SD contexts with thread safety
- [ ] `Generate()` API produces valid PNG images from text prompts
- [ ] `imagegen/` package processes image prompts end-to-end
- [ ] Generated images upload to Canvus and appear on canvas
- [ ] Prompt detection distinguishes `{{image: }}` from `{{ }}`

**Cross-Platform:**
- [ ] Windows build with MSVC + CUDA produces stable-diffusion.dll
- [ ] Linux build with GCC + CUDA produces libstable-diffusion.so
- [ ] NSIS installer bundles SD model and libraries
- [ ] .deb package bundles SD model and libraries
- [ ] Identical functionality on Windows and Linux

**Configuration:**
- [ ] `.env` supports SD_IMAGE_SIZE, SD_INFERENCE_STEPS, SD_GUIDANCE_SCALE, etc.
- [ ] Defaults work without any configuration (zero-config)
- [ ] Invalid config values detected and reported at startup

**Error Handling:**
- [ ] Missing model file shows helpful error message
- [ ] CUDA unavailable shows clear error message
- [ ] Out of VRAM creates error note on canvas
- [ ] Generation timeout creates error note on canvas
- [ ] All errors logged to app.log with context

**Testing:**
- [ ] Unit tests cover sdruntime/ (>80% coverage)
- [ ] Unit tests cover imagegen/ (>80% coverage)
- [ ] Integration test generates 512x512 image successfully
- [ ] Integration test generates 768x768 image successfully
- [ ] Platform tests pass on Windows 10/11 + NVIDIA GPU
- [ ] Platform tests pass on Linux (Ubuntu 22.04) + NVIDIA GPU

**Performance:**
- [ ] 512x512 image generates in <30s on RTX 3060
- [ ] 768x768 image generates in <60s on RTX 3060
- [ ] CUDA acceleration is functional (verify GPU utilization)
- [ ] Concurrent text + image operations work without crashes
- [ ] VRAM usage stays within 10GB for concurrent operations

**Documentation:**
- [ ] Build process documented (CMake, CGo, dependencies)
- [ ] Configuration variables documented in .env.example
- [ ] Error messages are user-friendly
- [ ] README includes image generation instructions

### User Acceptance Criteria:

**Zero-Configuration:**
- [ ] User installs via NSIS/.deb with bundled model
- [ ] User adds only Canvus credentials to .env
- [ ] First image generates successfully with no additional setup
- [ ] Process takes <10 minutes from download to first image

**Canvus Integration:**
- [ ] User types `{{image: sunset}}` on canvas
- [ ] Generated image appears automatically near prompt
- [ ] Image is native Canvus widget (not external link)
- [ ] Existing text generation `{{ }}` still works

**Privacy:**
- [ ] No network calls during image generation (verified)
- [ ] All processing on local hardware (verified)
- [ ] User can disconnect internet and still generate images

**Quality:**
- [ ] Generated images match prompt intent (subjective, manual review)
- [ ] Images are clear and usable (not corrupted or glitched)
- [ ] 512x512 images suitable for canvas use

---

## Implementation Plan

### Phase 3 Breakdown

**Task 3.1: stable-diffusion.cpp Build Integration (2-3 days)**
- Set up CMake build for Windows (MSVC + CUDA)
- Set up CMake build for Linux (GCC + CUDA)
- Verify CUDA linking and runtime dependencies
- Bundle shared libraries (.dll/.so) in `lib/`
- Document build process and prerequisites

**Task 3.2: CGo Bindings (`sdruntime/` package) (3-4 days)**
- Create `cgo_bindings.go` with C wrapper functions
- Implement `loadModel()`, `generateImage()`, `freeContext()`
- Create `types.go` with Go parameter structs
- Implement `errors.go` with SD-specific errors
- Write unit tests for CGo layer (mocked C functions)

**Task 3.3: Context Pool Management (2 days)**
- Implement `ContextPool` struct with channel-based pooling
- Add `Acquire()` and `Release()` with timeout support
- Implement `Close()` for graceful shutdown
- Add thread-safety with mutexes
- Write unit tests for pool lifecycle

**Task 3.4: High-Level Generation API (2 days)**
- Implement `Generate()` API in `generate.go`
- Add `DefaultParams()` and parameter validation
- Create `image_utils.go` for PNG handling
- Write unit tests for parameter validation
- Write integration tests with actual model

**Task 3.5: Image Generation Pipeline (`imagegen/` package) (3 days)**
- Create `imagegen/processor.go` with `Processor` struct
- Implement `ProcessImagePrompt()` end-to-end flow
- Add placement calculation logic
- Add processing indicator note creation
- Add error note creation for failures
- Write unit tests with mocked dependencies

**Task 3.6: Prompt Detection Enhancement (1 day)**
- Update `monitorcanvus.go` to detect `{{image: }}` prefix
- Add routing logic: `{{image: }}` → imagegen, `{{ }}` → text
- Test prompt detection with various formats
- Ensure backward compatibility with existing prompts

**Task 3.7: Configuration Management (1 day)**
- Add SD configuration variables to `core.Config`
- Implement validation for SD_IMAGE_SIZE, SD_INFERENCE_STEPS, etc.
- Update `.env.example` with SD configuration
- Add defaults for zero-config operation

**Task 3.8: Model Selection and Bundling (1-2 days)**
- Download SD v1.5 model from HuggingFace
- Verify checksum and test with stable-diffusion.cpp
- Add model to installer packages (NSIS, .deb)
- Implement startup model verification
- Document model licensing

**Task 3.9: Error Handling and Logging (1-2 days)**
- Implement all error types in `sdruntime/errors.go`
- Add error handling in generation pipeline
- Create user-friendly error notes for canvas
- Add logging for SD operations in app.log
- Test all error scenarios

**Task 3.10: Integration Testing (2-3 days)**
- Write end-to-end integration test (prompt → image → canvas)
- Test concurrent text + image generation
- Test error scenarios (missing model, CUDA unavailable, out of VRAM)
- Performance testing on RTX 3060
- Manual testing on Windows and Linux

**Task 3.11: Installer Updates (1-2 days)**
- Update NSIS installer to include stable-diffusion.dll and SD model
- Update .deb package to include libstable-diffusion.so and SD model
- Test installation process end-to-end
- Verify file sizes and bundling

**Task 3.12: Documentation (1 day)**
- Document build process (CMake, CGo, CUDA dependencies)
- Update README with image generation instructions
- Document configuration variables
- Add troubleshooting guide for common issues

### Total Estimated Time: 20-26 days

### Dependencies and Risks

**Critical Path:**
1. stable-diffusion.cpp build → CGo bindings → Context pool → Generation API → Pipeline
2. Model selection and bundling required before installer updates

**Risks:**
- **stable-diffusion.cpp build complexity:** CMake + CUDA can be finicky
  - Mitigation: Start with reference builds, document thoroughly
- **VRAM constraints:** Concurrent text + image may exhaust VRAM
  - Mitigation: Conservative defaults (SD_MAX_CONCURRENT=2), monitoring
- **CGo memory safety:** Pointer bugs can cause crashes
  - Mitigation: Careful coding, defer cleanup, panic recovery, thorough testing
- **Installer size:** ~13GB with both models may be too large
  - Mitigation: Consider download-on-first-run alternative (though less zero-config)
- **Cross-platform build:** Windows and Linux builds may diverge
  - Mitigation: CI/CD with both platforms, maintain parity

---

## Appendices

### A. Stable Diffusion Primer

**What is Stable Diffusion?**
A latent diffusion model that generates images from text descriptions by:
1. Encoding text prompt via CLIP text encoder
2. Generating latent representation through iterative denoising (U-Net)
3. Decoding latent to pixel space via VAE decoder

**Key Parameters:**
- **Prompt:** Text description of desired image
- **Negative Prompt:** Text describing what to avoid
- **Steps:** Number of denoising iterations (more = better quality, slower)
- **CFG Scale:** Classifier-free guidance scale (how closely to follow prompt)
- **Seed:** Random seed for reproducibility
- **Width/Height:** Output image dimensions

**Sample Methods:**
- Euler A (default): Fast, good quality
- DPM++ 2M: Higher quality, slower
- DDIM: Deterministic, fewer steps

### B. CUDA Architecture Compatibility

**NVIDIA GPU Generations:**
- **Turing (RTX 20xx):** CUDA 75
- **Ampere (RTX 30xx):** CUDA 86
- **Ada Lovelace (RTX 40xx):** CUDA 89

**Minimum Requirements:**
- GPU: NVIDIA RTX 2060 or newer
- VRAM: 6GB minimum (8GB+ recommended)
- CUDA: 11.8 or newer
- Driver: 520.xx or newer

### C. stable-diffusion.cpp vs Alternatives

**Why stable-diffusion.cpp?**
- Minimal dependencies (same as llama.cpp)
- CUDA acceleration built-in
- Cross-platform (Windows, Linux, macOS)
- Single-file distribution
- Active development and community support
- MIT license (permissive)

**Alternatives Considered:**
- **Diffusers (Python):** Requires Python runtime, PyTorch, complex dependencies
- **Automatic1111 WebUI:** Heavyweight, web-based, not embeddable
- **InvokeAI:** Similar issues, Python-based
- **ComfyUI:** Node-based, not suitable for embedding

stable-diffusion.cpp wins for embeddability and alignment with llama.cpp architecture.

### D. Future Enhancement Ideas

**Phase 4+ Potential Features:**
- Image-to-image: Modify existing canvas images
- Inpainting: Edit portions of images
- ControlNet: Guided generation with edge maps, poses, depth
- LoRA support: Artistic style customization
- Upscaling: Enhance generated images to higher resolution
- Batch generation: Multiple images from single prompt
- Style presets: Predefined artistic styles
- Model switching: Support multiple SD models
- Negative prompt UI: Easier negative prompt configuration

---

## Conclusion

Phase 3 integrates stable-diffusion.cpp to enable local image generation in CanvusLocalLLM. By following the established llama.cpp CGo pattern, this spec maintains architectural consistency while adding powerful multimodal capabilities. The zero-configuration philosophy is preserved through model bundling and sensible defaults, ensuring users can generate images immediately after installation.

**Key Success Factors:**
1. Follow atomic design patterns established in Phase 2
2. Reuse CGo integration expertise from llama.cpp
3. Conservative VRAM defaults prevent resource exhaustion
4. Comprehensive testing catches cross-platform issues early
5. User-friendly error messages guide troubleshooting

**Next Steps:**
1. Review and approve this specification
2. Create Beads issues for each task (3.1 - 3.12)
3. Set up dependencies between tasks
4. Begin implementation with Task 3.1 (stable-diffusion.cpp build)

---

**End of Specification**
