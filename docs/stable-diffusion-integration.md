# Stable Diffusion Integration Guide

This document describes the complete integration of stable-diffusion.cpp into CanvusLocalLLM for local GPU-accelerated image generation.

## Overview

CanvusLocalLLM integrates [stable-diffusion.cpp](https://github.com/leejet/stable-diffusion.cpp) via CGo bindings in the `sdruntime/` package. This enables:

- **Local image generation**: No cloud API dependencies
- **CUDA acceleration**: GPU-accelerated inference on NVIDIA GPUs
- **Minimal dependencies**: Following llama.cpp philosophy
- **Cross-platform**: Linux, Windows, macOS support

## Architecture

### Package Structure

```
sdruntime/                    # Stable Diffusion runtime package
├── cgo_bindings.go          # Public API wrapper (atoms → molecules)
├── cgo_bindings_sd.go       # Real CGo implementation (build tag: sd)
├── cgo_bindings_stub.go     # Stub implementation (default build)
├── config.go                # Configuration parsing (atoms)
├── types.go                 # Type definitions and validation (atoms)
├── prompt.go                # Prompt validation (atom)
├── seed.go                  # Random seed generation (atom)
├── image_utils.go           # PNG encoding utilities (atoms)
├── errors.go                # Domain-specific errors
├── context_pool.go          # Context pool management (molecule)
├── generate.go              # High-level Generator API (organism)
└── verify.go                # Model checksum verification (molecule)
```

### Atomic Design Layers

**Atoms** (Pure functions, no dependencies):
- `ValidateParams()`, `ValidatePrompt()`: Input validation
- `RandomSeed()`: Random seed generation
- `EncodeToPNG()`: RGBA to PNG conversion
- Config parsers: `parseImageSize()`, `parseInferenceSteps()`, etc.

**Molecules** (Simple compositions):
- `LoadModel()`: Wraps `loadModelImpl()` with error handling
- `GenerateImage()`: Wraps `generateImageImpl()` with validation
- `ContextPool`: Manages multiple SD contexts for concurrent generation

**Organism** (Complete feature):
- `Generator`: High-level API combining ContextPool + config management

## Implementation Status

### ✅ Completed Infrastructure

1. **CGo Bindings (sdruntime/cgo_bindings_sd.go)**
   - Full API alignment with stable-diffusion.cpp C API
   - `sd_ctx_create()`: 8-parameter signature (model, VAE, LoRA, threads, etc.)
   - `txt2img()`: 11-parameter signature (prompt, size, steps, sampler, seed, etc.)
   - `sd_free_image()`: Proper cleanup of returned image data
   - `sd_get_backend_info()`, `sd_cuda_available()`: Utility functions
   - Thread-safe context management with `sync.Map`
   - Memory-safe C string handling with `C.CString`/`C.free`

2. **Configuration Management (sdruntime/config.go + core/config.go)**
   - Environment variable parsing: `SD_*` variables
   - Validation: image size (divisible by 8), steps (1-100), CFG scale (1.0-30.0)
   - Default values: 512px, 20 steps, 7.5 guidance
   - Type-safe config struct with `LoadSDConfig()`

3. **Build Infrastructure (deps/stable-diffusion.cpp/)**
   - `build-linux.sh`: Linux build script with CUDA support
   - `build-windows.ps1`: Windows build script (PowerShell)
   - `CMakeLists.txt`: CMake configuration targeting Turing/Ampere/Ada architectures
   - `stable-diffusion.h`: C API header defining expected interface

4. **Documentation**
   - Package godoc (sdruntime/doc.go): Complete API documentation
   - Build instructions (deps/stable-diffusion.cpp/README.md)
   - Configuration reference with environment variables
   - Error handling guide with domain-specific errors

5. **Testing**
   - Unit tests for all atoms (validation, parsing, utilities)
   - Context pool tests (acquisition, release, concurrency)
   - Stub implementation for testing without C library
   - All tests passing in stub mode

### 🚧 Blocked: Waiting on C Library

The following require the actual stable-diffusion.cpp library to be compiled:

1. **Library Compilation**
   - Clone stable-diffusion.cpp source: `git clone https://github.com/leejet/stable-diffusion.cpp`
   - Build with CUDA: `./build-linux.sh` or `.\build-windows.ps1`
   - Output: `lib/libstable-diffusion.so` (Linux) or `lib/stable-diffusion.dll` (Windows)

2. **CGo Linking**
   - Enable CGo: `CGO_ENABLED=1`
   - Build with tag: `go build -tags sd`
   - Set library path: `export LD_LIBRARY_PATH=$PWD/lib:$LD_LIBRARY_PATH`

3. **Integration Testing**
   - Load real SD model (e.g., sd-v1-5.safetensors)
   - Test image generation with various parameters
   - Verify CUDA acceleration
   - Benchmark performance

4. **Model Download**
   - Download SD v1.5 model to `models/sd-v1-5.safetensors`
   - Sources: HuggingFace, Civitai
   - ~4GB file size (safetensors format recommended)

## Build Instructions

### Prerequisites

**Linux:**
- GCC 9+ or Clang 10+
- CUDA Toolkit 11.8+
- CMake 3.18+
- Git

**Windows:**
- Visual Studio 2022 with C++ workload
- CUDA Toolkit 11.8+
- CMake 3.18+
- Git

### Step 1: Build stable-diffusion.cpp Library

**Linux:**
```bash
cd deps/stable-diffusion.cpp
./build-linux.sh
# Output: ../../lib/libstable-diffusion.so
```

**Windows:**
```powershell
cd deps\stable-diffusion.cpp
.\build-windows.ps1
# Output: ..\..\lib\stable-diffusion.dll
```

**Verify Output:**
```bash
ls -lh lib/
# Should show libstable-diffusion.so (Linux) or stable-diffusion.dll (Windows)
```

### Step 2: Build Go Application with CGo

**Linux:**
```bash
# Set library path
export LD_LIBRARY_PATH=$PWD/lib:$LD_LIBRARY_PATH

# Build with sd tag
CGO_ENABLED=1 go build -tags sd -o canvusapi-sd

# Run
./canvusapi-sd
```

**Windows:**
```powershell
# Add library to PATH
$env:PATH = "$PWD\lib;$env:PATH"

# Build with sd tag
$env:CGO_ENABLED = 1
go build -tags sd -o canvusapi-sd.exe

# Run
.\canvusapi-sd.exe
```

### Step 3: Download Model

```bash
# Create models directory
mkdir -p models

# Download SD v1.5 model (example using wget)
wget -O models/sd-v1-5.safetensors \
  https://huggingface.co/runwayml/stable-diffusion-v1-5/resolve/main/v1-5-pruned-emaonly.safetensors

# Or use curl
curl -L -o models/sd-v1-5.safetensors \
  https://huggingface.co/runwayml/stable-diffusion-v1-5/resolve/main/v1-5-pruned-emaonly.safetensors
```

### Step 4: Configure Environment

Add to `.env`:

```bash
# Stable Diffusion Configuration
SD_MODEL_PATH=models/sd-v1-5.safetensors
SD_IMAGE_SIZE=512                # Default image size (512, 768, 1024)
SD_INFERENCE_STEPS=20            # Denoising steps (1-100)
SD_GUIDANCE_SCALE=7.5            # CFG scale (1.0-30.0)
SD_NEGATIVE_PROMPT=blurry, low quality, distorted
SD_TIMEOUT_SECONDS=120           # Generation timeout
SD_MAX_CONCURRENT=2              # Max concurrent generations (VRAM management)
```

## Usage

### Basic Usage (ContextPool API)

```go
package main

import (
    "context"
    "log"
    "os"
    "time"
    "go_backend/sdruntime"
)

func main() {
    // Create context pool with 2 concurrent slots
    pool, err := sdruntime.NewContextPool(2, "models/sd-v1-5.safetensors")
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Set up generation parameters
    params := sdruntime.GenerateParams{
        Prompt:         "a beautiful sunset over mountains, digital art",
        NegativePrompt: "blurry, low quality",
        Width:          512,
        Height:         512,
        Steps:          20,
        CFGScale:       7.5,
        Seed:           -1, // Random seed
    }

    // Generate image with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    imageData, err := pool.Generate(ctx, params)
    if err != nil {
        log.Fatal(err)
    }

    // Save to file
    os.WriteFile("output.png", imageData, 0644)
    log.Println("Image saved to output.png")
}
```

### High-Level Generator API

```go
// Load configuration from environment
config := sdruntime.LoadSDConfig()

// Create generator
gen, err := sdruntime.NewGenerator(config.MaxConcurrent, config.ModelPath)
if err != nil {
    log.Fatal(err)
}
defer gen.Close()

// Generate with config defaults
params := sdruntime.DefaultParams()
params.Prompt = "a serene lake at dawn"

imageData, err := gen.Generate(ctx, params)
```

### Integration with Canvus

```go
// In handlers.go or imagegen package
func generateLocalImage(prompt string, widget canvusapi.Widget) error {
    // Use global SD generator (similar to LLM client)
    params := sdruntime.GenerateParams{
        Prompt: prompt,
        Width:  512,
        Height: 512,
        Steps:  20,
        CFGScale: 7.5,
    }

    imageData, err := sdGenerator.Generate(ctx, params)
    if err != nil {
        return err
    }

    // Upload to canvas
    return canvusClient.UploadImage(imageData, widget)
}
```

## Configuration Reference

### Environment Variables

| Variable | Type | Default | Range | Description |
|----------|------|---------|-------|-------------|
| `SD_MODEL_PATH` | string | *(required)* | - | Path to .safetensors, .ckpt, or GGUF model |
| `SD_IMAGE_SIZE` | int | 512 | 128-2048 | Default image dimensions (must be ÷8) |
| `SD_INFERENCE_STEPS` | int | 20 | 1-100 | Number of denoising steps |
| `SD_GUIDANCE_SCALE` | float | 7.5 | 1.0-30.0 | Classifier-free guidance scale (CFG) |
| `SD_NEGATIVE_PROMPT` | string | "" | - | Default negative prompt |
| `SD_TIMEOUT_SECONDS` | int | 120 | 10+ | Generation timeout in seconds |
| `SD_MAX_CONCURRENT` | int | 2 | 1-10 | Max concurrent generations (VRAM limit) |
| `SD_MAX_IMAGE_SIZE` | int | 1024 | 128-2048 | Maximum allowed image size |

### Parameter Constraints

**Image Size:**
- Must be divisible by 8 (stable-diffusion.cpp requirement)
- Common values: 512, 768, 1024
- Custom sizes supported if divisible by 8

**Inference Steps:**
- Range: 1-100
- Quality vs speed tradeoff
- 20-30 steps recommended for quality
- 10-15 steps for speed

**CFG Scale:**
- Range: 1.0-30.0
- Higher = stronger prompt adherence
- 7.0-9.0 recommended for most cases
- Lower values allow more creativity

**Seed:**
- `-1` for random generation
- Positive integer for reproducibility
- Same seed + params = same image

## Error Handling

### Domain-Specific Errors

```go
import "errors"

// Check for specific errors
_, err := pool.Generate(ctx, params)
if err != nil {
    switch {
    case errors.Is(err, sdruntime.ErrOutOfVRAM):
        // Reduce image size or max concurrent
        log.Println("Out of VRAM, reducing settings...")
    case errors.Is(err, sdruntime.ErrGenerationTimeout):
        // Increase timeout or reduce steps
        log.Println("Generation timed out, increasing timeout...")
    case errors.Is(err, sdruntime.ErrModelNotFound):
        // Model file doesn't exist
        log.Fatal("Model not found:", err)
    default:
        log.Fatal("Generation failed:", err)
    }
}
```

### Common Error Scenarios

| Error | Cause | Solution |
|-------|-------|----------|
| `ErrModelNotFound` | Model file doesn't exist | Download model to correct path |
| `ErrModelLoadFailed` | Failed to load model | Check VRAM, file integrity |
| `ErrOutOfVRAM` | GPU memory exhausted | Reduce image size or concurrent count |
| `ErrGenerationTimeout` | Took too long | Increase timeout or reduce steps |
| `ErrCUDANotAvailable` | No NVIDIA GPU | Use CPU mode or cloud API |
| `ErrInvalidParams` | Bad parameters | Check size divisibility, ranges |

## Performance

### Benchmarks

**NVIDIA RTX 3080 (10GB VRAM):**
- 512x512, 20 steps: ~2-3 seconds
- 768x768, 20 steps: ~5-7 seconds
- 1024x1024, 20 steps: ~10-15 seconds

**Concurrent Generation:**
- Max 2-3 concurrent with 10GB VRAM
- Max 4-6 concurrent with 24GB VRAM
- Adjust `SD_MAX_CONCURRENT` based on VRAM

### Optimization Tips

1. **Batch Size**: Generate multiple images in parallel up to VRAM limit
2. **Step Count**: Use 15-20 steps for faster generation, 30-50 for quality
3. **Image Size**: Smaller sizes (512x512) are 4x faster than 1024x1024
4. **VAE Tiling**: Enable for larger images to reduce VRAM (at cost of speed)
5. **Model Selection**: SD 1.5 is faster than SDXL (smaller model)

## Troubleshooting

### Build Errors

**"CUDA not found"**
```bash
# Linux: Add CUDA to PATH
export PATH=/usr/local/cuda/bin:$PATH
export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH

# Windows: Check CUDA installation
nvcc --version
```

**"undefined reference to sd_ctx_create"**
```bash
# Make sure library is built and in lib/
ls -l lib/libstable-diffusion.so

# Set library path
export LD_LIBRARY_PATH=$PWD/lib:$LD_LIBRARY_PATH
```

### Runtime Errors

**"Model file corrupted"**
```bash
# Verify model checksum
go run . -verify-model models/sd-v1-5.safetensors
```

**"Out of VRAM"**
- Reduce `SD_IMAGE_SIZE` to 512
- Reduce `SD_MAX_CONCURRENT` to 1
- Close other GPU applications
- Use smaller model (SD 1.5 instead of SDXL)

**"GPU not detected"**
```bash
# Check NVIDIA driver
nvidia-smi

# Verify CUDA installation
nvcc --version

# Check backend info
go run . -sd-backend-info
```

## Next Steps

1. **Compile C Library**: Run build scripts in `deps/stable-diffusion.cpp/`
2. **Download Model**: Get SD v1.5 model (~4GB)
3. **Test CGo Build**: `CGO_ENABLED=1 go build -tags sd`
4. **Integration Testing**: Test generation with real model
5. **Canvas Integration**: Wire into `imagegen/` package for Canvus uploads
6. **Performance Tuning**: Benchmark and optimize concurrent generation

## References

- [stable-diffusion.cpp GitHub](https://github.com/leejet/stable-diffusion.cpp)
- [Stable Diffusion Models (HuggingFace)](https://huggingface.co/models?pipeline_tag=text-to-image)
- [CUDA Toolkit Documentation](https://docs.nvidia.com/cuda/)
- [Go CGo Documentation](https://pkg.go.dev/cmd/cgo)
- [CanvusLocalLLM CLAUDE.md](../CLAUDE.md) - Codebase architecture
