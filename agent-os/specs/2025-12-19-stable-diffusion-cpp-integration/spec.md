# Specification: Stable Diffusion Integration via stable-diffusion.cpp

## Goal
Enable local, offline image generation in CanvusLocalLLM by integrating stable-diffusion.cpp with CUDA acceleration, completing the local-first vision pipeline and eliminating dependency on OpenAI DALL-E and Azure OpenAI cloud APIs for image generation.

## User Stories
- As a CanvusLocalLLM user, I want to generate images from text prompts using my local NVIDIA GPU so that I maintain complete privacy without cloud API calls
- As a developer, I want image generation to follow the same Provider interface pattern as OpenAI/Azure providers so that the implementation is swappable and testable

## Specific Requirements

**Complete sdruntime CGo Bindings**
- Implement 5 stubbed functions in `sdruntime/cgo_bindings_sd.go:76-213`: `loadModelImpl()`, `generateImageImpl()`, `freeContextImpl()`, `getBackendInfoImpl()`, `IsCUDAAvailable()`
- Uncomment C header include `#include <stable-diffusion.h>` at line 31 once library is compiled
- Replace placeholder `typedef void* sd_ctx_t;` with actual stable-diffusion.h types
- Map C function signatures: `sd_ctx_create()`, `txt2img()`, `sd_ctx_free()`, `sd_get_backend_info()`, `sd_cuda_available()`
- Handle C string conversions with `C.CString()` and `defer C.free(unsafe.Pointer())` pattern
- Implement `contextMap` using `sync.Map` for thread safety (current implementation at line 74 is not concurrent-safe)
- Return `ErrModelNotFound` for missing model files, `ErrModelLoadFailed` for C library errors, `ErrGenerationFailed` for generation errors

**Image Data Marshaling Between Go and C**
- Convert generated RGBA image from C to Go bytes: `C.GoBytes(unsafe.Pointer(imgPtr), C.int(imgSize))` where size is `width * height * 4`
- Use existing `EncodePNG()` function from `sdruntime/image_utils.go:17-45` to convert RGBA bytes to PNG format
- Validate returned image dimensions match requested dimensions (within stable-diffusion.cpp constraints)
- Handle negative seeds by calling `GenerateSeed()` from `sdruntime/seed.go:9-11` before passing to C
- Free C-allocated image memory with `C.sd_free_image(imgPtr)` immediately after copying to Go slice

**ContextPool Thread Safety and Resource Management**
- `sdruntime/context_pool.go` already implements channel-based context pooling (lines 1-279)
- VRAM budget estimation: SD v1.5 Q8_0 model ~2GB + 512x512 generation ~1.5GB per context = 3.5GB per concurrent generation
- Default `MaxConcurrent=2` from `DefaultClientConfig()` targets 8GB VRAM GPUs (7GB active + 1GB OS/workspace)
- Lazy context creation pattern already implemented at lines 145-167 (create on first Acquire if pool empty)
- Context cleanup on pool close already handles draining channel and freeing C resources (lines 227-247)
- Timeout handling via `context.Context` already implemented: Acquire respects `ctx.Done()` at line 188

**Prompt Validation and Sanitization**
- Use existing `SanitizePrompt()` from `sdruntime/prompt.go:9-13` to strip special characters
- Use existing `ValidatePrompt()` to enforce max prompt length and non-empty constraint
- Apply validation in `imagegen/processor.go:213-219` before passing to `pool.Generate()`
- Negative prompt sanitization follows same pattern (optional parameter, defaults to empty string)
- Return `ErrInvalidPrompt` from `sdruntime/errors.go` if validation fails

**Generation Parameter Validation**
- Validate image dimensions: width and height must be divisible by 8 (stable-diffusion.cpp requirement)
- Enforce dimension range: 128x128 minimum, 1024x1024 maximum (configurable via `SD_MAX_IMAGE_SIZE`)
- Default dimensions: 512x512 (balance quality vs speed on RTX 3060+)
- Validate steps: 1-150 range, default 20 (25 for higher quality, 15-20 for speed)
- Validate CFG scale: 1.0-20.0 range, default 7.0 (classifier-free guidance strength)
- Validate seed: -1 for random (generates via `GenerateSeed()`), or explicit int64 for reproducibility
- Use existing `ValidateParams()` pattern from `sdruntime/context_pool.go:90` for parameter checking

**Provider Interface Implementation for Local SD**
- Create `imagegen/sd/sd_provider.go` implementing `imagegen.Provider` interface from `imagegen/openai_provider.go:21-34`
- Implement `Generate(ctx context.Context, prompt string) (string, error)` method signature
- For local generation, return local file path instead of URL (differs from cloud providers)
- Wrap `sdruntime.ContextPool.Generate()` call with error translation to `GenerationError` types
- Handle context cancellation via `ctx.Done()` during generation loop
- Map sdruntime errors to provider errors: `ErrOutOfVRAM` → `ErrCodeOutOfMemory`, `ErrGenerationFailed` → `ErrCodeGenerationFailed`
- Return file path to saved PNG in downloads directory: `downloads/sd_image_{correlationID}.png`

**Integration with Existing imagegen.Processor**
- `imagegen/processor.go` already implements local SD pipeline using `sdruntime.ContextPool` (lines 1-452)
- `ProcessImagePrompt()` at lines 203-322 orchestrates: validate → create processing note → generate → save → upload → cleanup
- Processor uses `pool.Generate(ctx, params)` at line 251, which calls into sdruntime CGo bindings
- Progress reporting via processing notes at lines 222, 238, 265 ("Generating image...", "Uploading...")
- Error handling creates canvas error notes at line 257 and 273 for user feedback
- Temporary file cleanup with `defer os.Remove(imagePath)` at line 279 prevents downloads directory bloat

**CMake Build System for stable-diffusion.cpp**
- Create `deps/stable-diffusion.cpp/CMakeLists.txt` with CUDA support: `-DGGML_CUDA=ON -DCMAKE_BUILD_TYPE=Release`
- Build script `deps/stable-diffusion.cpp/build-linux.sh`: `cmake -B build && cmake --build build --config Release`
- Build script `deps/stable-diffusion.cpp/build-windows.ps1`: `cmake -B build -G "Visual Studio 17 2022" && cmake --build build --config Release`
- Output shared library to `lib/` directory: `libstable-diffusion.so` (Linux), `stable-diffusion.dll` (Windows)
- Copy CUDA runtime DLLs to `lib/` on Windows: `cudart64_*.dll`, `cublas64_*.dll`, `cublasLt64_*.dll`
- Link against CUDA libraries: `find_package(CUDAToolkit REQUIRED)` and `target_link_libraries(stable-diffusion CUDA::cudart CUDA::cublas)`
- CGo LDFLAGS already configured in `sdruntime/cgo_bindings_sd.go:24-26` to link against `lib/` output

**Model Selection and Bundling**
- Bundle Stable Diffusion v1.5 Q8_0 quantized model (~2.2GB) for balance of quality and size
- Model file path: `models/sd-v1.5-q8_0.gguf` (matches llamaruntime model organization pattern)
- VAE tiling disabled by default (`VAETiling: false` in `imagegen/sd/client.go:98`) - sufficient for 512x512
- Load model once on application startup via `sdruntime.NewContextPool()` in `main.go`
- Model validation: check file exists and is readable before calling `LoadModel()`, return `ErrModelNotFound` if missing
- Alternative model consideration: SDXL-Turbo (~4GB) for faster inference if performance testing shows acceptable quality at lower step counts

**Prompt Detection and Routing in monitorcanvus.go**
- Extend `handleUpdate()` at `monitorcanvus.go:~150` to detect image generation prompts
- Implement prefix-based routing: `{{image: prompt text}}` → image generation, `{{prompt text}}` → text generation
- Extract prompt via regex: `{{image:\s*(.+?)}}` captures text after "image:" prefix
- Route to `imagegen.Processor.ProcessImagePrompt()` for local generation when prefix detected
- Maintain backward compatibility: existing `{{ }}` syntax without prefix continues routing to text generation handlers
- Validation: reject empty prompts after prefix extraction, create error note on canvas

**Configuration Management**
- Add to `core/config.go`: `SDImageSize`, `SDInferenceSteps`, `SDGuidanceScale`, `SDTimeoutSeconds`, `SDMaxConcurrent`, `SDModelPath`
- Default values in `LoadConfig()`: size=512, steps=20, guidance=7.0, timeout=60s, maxConcurrent=2, modelPath="models/sd-v1.5-q8_0.gguf"
- Environment variable mapping in `.env`: `SD_IMAGE_SIZE`, `SD_INFERENCE_STEPS`, `SD_GUIDANCE_SCALE`, `SD_TIMEOUT_SECONDS`, `SD_MAX_CONCURRENT`, `SD_MODEL_PATH`
- Validation: image size divisible by 8, steps 1-150, guidance 1.0-20.0, timeout >=10s, maxConcurrent 1-10
- Wire config into `sdruntime.ContextPool` and `imagegen.Processor` constructors in application bootstrap

**Error Handling and User Feedback**
- CUDA out-of-memory: detect "out of memory" or "VRAM" in C error string, return `ErrOutOfVRAM` with message "Insufficient VRAM. Try reducing image size, concurrent generations, or closing other GPU applications."
- Model loading failure: check file permissions, CUDA availability, library linking errors, return `ErrModelLoadFailed` with specific cause
- Generation timeout: enforce via `context.WithTimeout()` wrapping generation call, return partial result or error
- Canvas error notes: create red-background note with clear message when generation fails (pattern from `processor.go:372-399`)
- Logging: use structured logging with `zap.String("correlation_id", ...)` for request tracing across handlers and CGo boundary

**Cross-Platform Build Considerations**
- Linux: GCC 9+, CUDA Toolkit 11.8+, CMake 3.18+, link with `-Wl,-rpath` for runtime library search
- Windows: MSVC 2022, CUDA Toolkit 11.8+, CMake 3.18+, copy DLLs to executable directory or `lib/`
- macOS: Not supported (CUDA unavailable), build tags exclude SD integration on darwin platform
- Build validation: create `sdruntime/verify.go:9-53` to test CUDA availability and model loading on application startup
- CI/CD: require GPU-enabled build agents for integration tests, skip SD tests on CPU-only runners

## Visual Design
No visual assets provided (code-only implementation spec).

## Existing Code to Leverage

**`imagegen/processor.go:80-132` - Processor Constructor Pattern**
- Constructor validates dependencies (pool, client, logger non-nil), checks pool not closed
- Creates downloads directory with `os.MkdirAll(config.DownloadsDir, 0755)` pattern
- Returns descriptive errors for invalid inputs using `fmt.Errorf("imagegen: %w", err)` wrapping
- Reuse: Apply same validation pattern in `imagegen/sd/sd_provider.go` constructor for consistency

**`imagegen/processor.go:203-322` - End-to-End Image Generation Pipeline**
- Pipeline: validate prompt → create processing note → generate via sdruntime → save PNG → upload to canvas → cleanup temp file → delete processing note
- Progress updates via `updateProcessingNote()` at multiple stages for user feedback
- Error recovery: create error notes on canvas, log with correlation ID, ensure processing note cleanup in defer
- Reuse: This is the main orchestration logic - wire `sdruntime.ContextPool` into line 251 `pool.Generate()` call

**`imagegen/generator.go:120-193` - Factory Pattern for Provider Selection**
- `NewGeneratorFromConfig()` auto-detects provider based on config (Azure vs OpenAI)
- Validates API keys, checks for local vs cloud endpoints
- Creates provider + downloader + assembles Generator organism
- Reuse: Add SDProvider detection path when `SD_MODEL_PATH` is set and model file exists locally

**`llamaruntime/bindings.go:658-756` - CGo Inference Pattern**
- Pattern: tokenize → create batch → decode loop with context check → sample → collect tokens → cleanup
- Thread safety: `llamaCtx.mu.Lock()` during all C API calls
- Context cancellation: `select { case <-ctx.Done(): return partial }` in generation loop
- Error wrapping: `fmt.Errorf("failed to X: %w", err)` for stack tracing
- Reuse: Apply identical pattern in `sdruntime/cgo_bindings_sd.go:110-175` for image generation loop

**`sdruntime/context_pool.go:88-118` - Generate with Context Acquisition**
- High-level API: validate params → acquire context → generate → release context → return
- Timeout enforcement via context: `Acquire(ctx)` respects deadline, returns `ErrAcquireTimeout` on expiration
- Statistics tracking: not yet implemented but structure present for future metrics
- Reuse: This is the main entry point for image generation - implement `GenerateImage()` CGo call at line 107

**`canvusapi/canvusapi.go:26-31` - Widget Coordinate System Documentation**
- CRITICAL: Widget locations are RELATIVE to parent, not absolute canvas coordinates
- Placement calculation: `parentLocation + widgetRelativeLocation = absolute canvas position`
- Reuse: Use existing `CalculatePlacementWithConfig()` from `imagegen/placement.go:35-53` for image widget positioning

**`core/config.go:58-90` - Environment Variable Parsing Pattern**
- Use `getEnvOrDefault()` for string defaults, `parseIntEnv()` for integers with validation
- Validation errors returned during `LoadConfig()` prevent startup with invalid config
- HTTP client factory `GetHTTPClient()` respects TLS configuration for consistency
- Reuse: Add SD config variables following same parsing and validation pattern

## Out of Scope
- Image-to-image generation (only text-to-image supported initially)
- Inpainting, outpainting, or image editing features
- ControlNet integration for guided generation
- LoRA model support for style customization
- Multiple aspect ratios beyond 1:1 (future: 16:9, 4:3, 3:2)
- Real-time preview during generation (complete image only)
- Batch generation from multiple prompts in single call (one prompt per request)
- Model switching UI or multiple model support (single bundled model only)
- Automatic upscaling of generated images
- Style transfer or artistic filters post-generation
- CPU-only fallback (NVIDIA GPU with CUDA required)
- Integration with image generation providers beyond OpenAI/Azure/Local (extensible via Provider interface for future)
