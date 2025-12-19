# Spec Initialization: Stable Diffusion.cpp Integration

## Feature Name
Stable Diffusion.cpp CGo Integration for Local Image Generation

## Feature Description

Complete the CGo bindings to stable-diffusion.cpp library to enable fully local, offline image generation for CanvusLocalLLM. This will allow users to generate images using Stable Diffusion models running locally on their GPU without requiring internet connectivity or cloud API costs.

## Context and Strategic Importance

**Current State:**
- Cloud image generation fully functional via OpenAI DALL-E and Azure OpenAI providers
- `imagegen/` package (4,881 lines, 73% coverage) handles cloud generation and canvas integration
- `sdruntime/` package exists with stubbed CGo bindings awaiting stable-diffusion.cpp library integration
- `imagegen/sd/` subdirectory contains client interface and types ready for local SD support
- stable-diffusion.cpp library scaffolding exists in `deps/stable-diffusion.cpp/` with header file and build scripts

**Priority Level:** P1 (HIGH)
- Enables offline image generation (critical for local-first vision)
- Removes dependency on cloud providers (reduces costs, improves privacy)
- Blocks 2 P1 features:
  - CanvusLocalLLM-149u: Cross-Platform SD Build
  - CanvusLocalLLM-s0gk: Local Image Generation Pipeline

**User Value:**
- No API costs for image generation
- Full privacy (no data sent to cloud)
- Offline capability (works without internet)
- GPU acceleration via CUDA (faster than cloud in many cases)

## Technical Foundation Already in Place

**1. sdruntime Package (CGo Layer)**
- `cgo_bindings_sd.go`: CGo wrapper with placeholder implementations (5 TODOs at lines 89, 135, 186, 196, 206)
- `context_pool.go`: Thread-safe context pool for managing SD contexts
- `image_utils.go`: PNG validation and encoding utilities
- `errors.go`: Error types and handling
- Build tags configured (`sd`, `cgo`) for conditional compilation

**2. imagegen/sd Package (Go Interface Layer)**
- `client.go`: Client interface matching Provider pattern
- `client_sd.go`: Real CGo implementation (uses sdruntime)
- `client_stub.go`: Fallback stub for non-CUDA builds
- `types.go`: Complete data structures (GenerationRequest, GenerationResponse, etc.)
- `config.go`: Client configuration with validation

**3. stable-diffusion.cpp Library Scaffolding**
- Header file: `deps/stable-diffusion.cpp/include/stable-diffusion.h`
- Build scripts: `build-linux.sh`, `build-windows.ps1`
- CMakeLists.txt configured for compilation

**4. Integration Pattern Reference**
- `llamaruntime/bindings.go` provides proven CGo pattern for similar integration (llama.cpp)
- Same build approach, same CUDA handling, same context pooling

## What Needs to Be Built

**Phase 1: Core CGo Bindings**
1. Uncomment and implement C function calls in `cgo_bindings_sd.go`:
   - `sd_ctx_create()` for model loading
   - `txt2img()` for image generation
   - `sd_ctx_free()` for cleanup
   - `sd_get_backend_info()` for backend detection
   - `sd_cuda_available()` for CUDA availability check

2. Implement the 5 stubbed functions:
   - `loadModelImpl()` - Load SD model from file
   - `generateImageImpl()` - Generate image from prompt
   - `freeContextImpl()` - Free SD context
   - `getBackendInfoImpl()` - Query backend info
   - `IsCUDAAvailable()` - Check CUDA availability

**Phase 2: Context Management**
3. Context pool integration:
   - Ensure `ContextPool` works with real SD contexts
   - Handle GPU memory pressure (VRAM monitoring)
   - Implement proper cleanup on pool shutdown

**Phase 3: Integration with Existing Pipeline**
4. Wire SD client into `imagegen.Generator`:
   - Add SD provider option to `NewGeneratorFromConfig()`
   - Implement provider selection (local SD vs cloud)
   - Fallback logic (SD → Cloud if SD unavailable)

5. Configuration management:
   - Add environment variables for SD model path
   - SD-specific settings (VAE path, tiling, etc.)
   - Document in `example.env`

**Phase 4: Build System**
6. Cross-platform build support:
   - Linux CUDA build
   - Windows CUDA build
   - macOS build (CPU fallback)
   - Build documentation

## Dependencies and Prerequisites

**External Library:**
- stable-diffusion.cpp compiled as shared library (libstable-diffusion.so/dll)
- Placed in `lib/` directory with headers in `deps/stable-diffusion.cpp/include/`

**Build Requirements:**
- CUDA Toolkit 11.8+ (for GPU acceleration)
- CMake 3.18+ (for building stable-diffusion.cpp)
- C++ compiler with C++17 support
- CGo enabled (CGO_ENABLED=1)

**Runtime Requirements:**
- NVIDIA GPU with CUDA support (recommended)
- Stable Diffusion model in GGUF or safetensors format
- Sufficient VRAM (4GB+ depending on model)

## Success Criteria

1. **Functional Completeness:**
   - Load SD model from file
   - Generate images from text prompts
   - Return PNG image data
   - Handle errors gracefully

2. **Performance:**
   - Context pool reuse works correctly
   - No memory leaks in CGo boundary
   - CUDA acceleration functional

3. **Integration:**
   - Works seamlessly with existing `imagegen.Generator`
   - Falls back to cloud if SD unavailable
   - Configuration via environment variables

4. **Build System:**
   - Cross-platform build scripts functional
   - Clear build documentation
   - CI/CD integration possible

## Related Issues
- CanvusLocalLLM-149u: Cross-Platform SD Build (BLOCKED by this)
- CanvusLocalLLM-s0gk: Local Image Generation Pipeline (BLOCKED by this)

## Initial Questions to User
Will be gathered through systematic requirements research process.
