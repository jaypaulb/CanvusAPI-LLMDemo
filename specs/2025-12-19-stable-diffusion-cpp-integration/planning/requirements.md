# Spec Requirements: Stable Diffusion.cpp Integration

## Initial Description
Complete the CGo bindings to stable-diffusion.cpp library to enable fully local, offline image generation for CanvusLocalLLM. This will allow users to generate images using Stable Diffusion models running locally on their GPU without requiring internet connectivity or cloud API costs.

The feature is currently stubbed with 5 TODOs in `sdruntime/cgo_bindings_sd.go` (lines 89, 135, 186, 196, 206) awaiting completion. Priority P1 (HIGH), blocks 2 other P1 features.

## Requirements Discussion

### Code Analysis Findings

**Q1: What model formats should be supported?**
**Analysis:** The stable-diffusion.h header (lines 81-82) shows the C API accepts `.safetensors`, `.ckpt`, or `GGUF` formats. Following the pattern from llamaruntime which uses GGUF (memory-mappable, faster loading), all three formats should be supported but GGUF should be prioritized in documentation.

**Q2: How should VAE models be handled?**
**Analysis:** The `sd_ctx_create()` function (lines 94-103 in stable-diffusion.h) accepts optional `vae_path` parameter (NULL for built-in). Based on existing env variable patterns in `core/config.go` and `example.env` (lines 150-186), VAE should be optional with built-in as default. Configuration should follow the llama.cpp pattern with `SD_VAE_PATH` env variable.

**Q3: What should the context pool size be?**
**Analysis:**
- Existing `imagegen.Generator` defaults to `MaxConcurrent=2` for cloud providers (core/config.go line 185)
- Example.env (lines 181-185) recommends 1-2 for RTX 3060 12GB, 2-4 for RTX 4080/4090 16GB+
- The `example.env` already defines `SD_MAX_CONCURRENT=2` as default
- Decision: Use `SD_MAX_CONCURRENT` env variable (already defined), default to 2, with warning that VRAM-constrained systems should use 1

**Q4: What sampling method and steps should be the default?**
**Analysis:**
- stable-diffusion.h (lines 53-62) defines 8 sampling methods
- Header comment (line 59) recommends `DPMPP_2M`
- example.env (lines 161-165) already defines `SD_INFERENCE_STEPS=20` with note "20-50 for quality, 8-15 for speed"
- Decision: Default to `DPMPP_2M` sampling with 20-30 steps (use existing `SD_INFERENCE_STEPS` env variable)

**Q5: How should provider selection work (local SD vs cloud)?**
**Analysis:**
- Current `NewGeneratorFromConfig()` in `imagegen/generator.go` (lines 120-150) auto-detects provider based on endpoint configuration
- Pattern: Azure endpoint → AzureProvider, otherwise → OpenAIProvider
- No explicit mode selection exists, relies on environment variable presence
- Decision: Add local SD detection - if `SD_MODEL_PATH` is set and model exists → SD provider, else fall back to cloud. Automatic fallback, no explicit mode variable needed (follows existing pattern)

**Q6: Should GPU memory be monitored?**
**Analysis:**
- llamaruntime/bindings.go (lines 778-817) has placeholder `getGPUMemory()` with TODO for nvidia-ml-go
- No GPU monitoring implemented yet in either runtime
- Decision: Defer GPU memory monitoring to Phase 2 or separate issue. Initial implementation should handle out-of-memory errors gracefully but not proactively monitor VRAM (matches llama.cpp integration state)

**Q7: How should build scripts be documented?**
**Analysis:**
- Build scripts exist: `deps/stable-diffusion.cpp/build-linux.sh` and `build-windows.ps1`
- CLAUDE.md (lines 14-30) documents build commands for Go application
- Decision: Add SD build section to main README, reference existing build scripts in deps/, document prerequisites (CUDA 11.8+, CMake 3.18+)

**Q8: What testing strategy should be used?**
**Analysis:**
- Existing `imagegen/` package has 73% coverage with cloud providers
- No test models in repo (appropriate - models are large)
- Decision: Add integration tests similar to cloud providers, document where to download test models (SD 1.5 from HuggingFace), do NOT include test models in repo

**Q9: What should be excluded from initial implementation?**
**Analysis of C API features in stable-diffusion.h:**
- LoRA support (lines 84, 98) - optional lora_model_dir parameter
- TAESD fast preview (line 97) - taesd_path parameter
- img2img (not in header, only txt2img at line 138)
- Advanced scheduling (only one sample_method parameter at line 146)

**Decision:** Initial implementation should include:
- ✅ txt2img generation (core feature)
- ✅ All 8 sampling methods (already defined in types.go)
- ✅ VAE support (optional path parameter)
- ✅ Basic error handling

Exclude from initial implementation (defer to future enhancements):
- ❌ LoRA model loading (set lora_model_dir to NULL)
- ❌ TAESD preview (set taesd_path to NULL)
- ❌ img2img (not in current header)
- ❌ GPU memory monitoring (defer to Phase 2)
- ❌ Progress callbacks during generation (defer to Phase 2)

### Existing Code to Reference

**Similar Features Identified:**

**llamaruntime CGo Pattern** (PRIMARY REFERENCE)
- Path: `llamaruntime/bindings.go` (839 lines)
- Key patterns to follow:
  - CGo header declarations with build tags (lines 14-157)
  - Model loading with file validation (lines 212-249)
  - Context creation and pooling (lines 334-388)
  - Error wrapping with sentinel errors (lines 232-238)
  - Finalizers for automatic cleanup (lines 244-246)
  - Mutex protection for C pointers (lines 323-328)
  - Unsafe pointer handling for C arrays (lines 605-638)

**sdruntime Existing Structure** (IMPLEMENT HERE)
- `context_pool.go` (279 lines) - Already implements pool pattern, needs real SD contexts
- `image_utils.go` (94 lines) - PNG validation and encoding ready
- `errors.go` (30 lines) - Sentinel errors defined
- `cgo_bindings_sd.go` (213 lines) - Stubbed functions to implement

**imagegen Provider Pattern** (INTEGRATION PATTERN)
- `openai_provider.go` (lines 1-150) - Provider interface implementation
- `azure_provider.go` (lines 1-150) - Alternative provider pattern
- `generator.go` (lines 120-150) - Auto-detection logic in `NewGeneratorFromConfig()`

**Build Configuration**
- `.env` variables: `SD_MODEL_PATH`, `SD_IMAGE_SIZE`, `SD_INFERENCE_STEPS`, `SD_GUIDANCE_SCALE`, `SD_MAX_CONCURRENT` (already defined in example.env lines 150-186)
- `core/config.go` - Pattern for adding new config fields (lines 40-76)

## Visual Assets

### Files Provided:
No visual assets provided (backend/CLI feature, no UI components).

### Visual Insights:
Not applicable - this is a backend CGo integration with no visual interface.

## Requirements Summary

### Functional Requirements

**Core CGo Bindings:**
1. Implement `loadModelImpl()` to call `sd_ctx_create()` with parameters:
   - model_path (required, from SD_MODEL_PATH)
   - vae_path (optional, from SD_VAE_PATH or NULL)
   - taesd_path (NULL for initial implementation)
   - lora_model_dir (NULL for initial implementation)
   - vae_decode_only (true for txt2img)
   - n_threads (from runtime.NumCPU())
   - vae_tiling (from SD_VAE_TILING env, default false)
   - free_params_immediately (from SD_FREE_PARAMS env, default false)

2. Implement `generateImageImpl()` to call `txt2img()` with parameters:
   - ctx (from context pool)
   - prompt (from GenerateParams.Prompt)
   - negative_prompt (from SD_NEGATIVE_PROMPT or GenerateParams.NegativePrompt)
   - clip_skip (-1 for default)
   - cfg_scale (from SD_GUIDANCE_SCALE, default 7.5)
   - width/height (from SD_IMAGE_SIZE, default 512)
   - sample_method (default DPMPP_2M)
   - sample_steps (from SD_INFERENCE_STEPS, default 20)
   - seed (from GenerateParams.Seed or random)
   - batch_count (1 for initial implementation)

3. Implement `freeContextImpl()` to call `sd_ctx_free()`

4. Implement `getBackendInfoImpl()` to call `sd_get_backend_info()`

5. Implement `IsCUDAAvailable()` to call `sd_cuda_available()`

**Context Management:**
- Context pool in `context_pool.go` works with real SD contexts
- Handle VRAM exhaustion errors gracefully
- Proper cleanup on pool shutdown
- Thread-safe access to C pointers

**Image Data Handling:**
- Convert C RGBA image data (sd_image_t) to PNG using `image_utils.EncodeToPNG()`
- Validate output image data before returning
- Free C image data with `sd_free_image()` after conversion

**Error Handling:**
- Map C library errors to Go sentinel errors (ErrModelLoadFailed, ErrGenerationFailed, etc.)
- Detect out-of-VRAM conditions and return ErrOutOfVRAM
- Provide clear error messages with context

### Reusability Opportunities

**Direct Code Reuse:**
- `llamaruntime/bindings.go` - CGo patterns for model loading, context management, unsafe pointer handling
- `sdruntime/context_pool.go` - Already implements pool logic, just needs real contexts
- `sdruntime/image_utils.go` - PNG encoding ready to use
- `imagegen/sd/client.go` - Interface already defined
- `imagegen/sd/types.go` - Data structures complete

**Backend Patterns to Follow:**
- Provider interface pattern from `openai_provider.go` and `azure_provider.go`
- Auto-detection logic from `generator.go` NewGeneratorFromConfig()
- Environment variable pattern from `core/config.go`
- Error wrapping and sentinel errors from `llamaruntime/bindings.go`

**Integration Points:**
- Add SD-specific config fields to `core.Config` (follow llama.cpp pattern)
- Extend `NewGeneratorFromConfig()` to detect and use SD provider
- Build tags for conditional compilation (`sd`, `cgo`)

### Scope Boundaries

**In Scope:**
- Complete 5 stubbed functions in `cgo_bindings_sd.go`
- txt2img generation with all 8 sampling methods
- Optional VAE model support
- Context pooling for concurrent generation
- CUDA availability detection
- PNG output format
- Error handling with sentinel errors
- Configuration via environment variables
- Integration with existing `imagegen.Generator`
- Build scripts documentation
- Integration tests

**Out of Scope (Future Enhancements):**
- LoRA model loading (requires additional C API integration)
- TAESD fast preview (defer to Phase 2)
- img2img generation (not in current C API)
- Advanced scheduling algorithms (not in current C API)
- GPU memory monitoring (defer to Phase 2 or separate issue)
- Progress callbacks during generation (defer to Phase 2)
- Batch generation (batch_count > 1, defer to Phase 2)
- Image upscaling/enhancement
- Model quantization support
- Dynamic model loading/unloading

### Technical Considerations

**Build System:**
- Build tags: `//go:build sd && cgo` for real implementation
- Build tags: `//go:build !sd || !cgo` for stub fallback
- CGo flags in `cgo_bindings_sd.go` already configured (lines 23-26)
- Library linking: `-lstable-diffusion` with rpath for dynamic loading
- Cross-platform support: Linux (primary), Windows, macOS (CPU fallback)

**Environment Variables to Add to core.Config:**
```go
// Stable Diffusion Configuration (add to Config struct)
SDModelPath           string  // Path to SD model file
SDVAEPath             string  // Optional separate VAE model path
SDImageSize           int     // Output image size (default: 512)
SDInferenceSteps      int     // Denoising steps (default: 20)
SDGuidanceScale       float64 // CFG scale (default: 7.5)
SDNegativePrompt      string  // Default negative prompt
SDTimeoutSeconds      int     // Generation timeout (default: 120)
SDMaxConcurrent       int     // Max concurrent generations (default: 2)
SDVAETiling           bool    // Enable VAE tiling (default: false)
SDFreeParamsImmediately bool  // Free params after load (default: false)
```

**Dependencies:**
- External: stable-diffusion.cpp compiled library in `lib/`
- External: CUDA Toolkit 11.8+ (runtime requirement)
- Go packages: No new dependencies (uses stdlib image/png)
- Build tools: CMake 3.18+, C++17 compiler

**Memory Management:**
- Use `runtime.SetFinalizer()` for automatic cleanup (follow llama.cpp pattern)
- Explicit cleanup with `FreeContext()` and `sd_free_image()`
- Context pool prevents context churn and VRAM fragmentation
- RGBA to PNG conversion happens in Go (no C memory retained after conversion)

**Thread Safety:**
- C contexts are NOT thread-safe (per stable-diffusion.cpp docs)
- Use mutex locks when accessing C pointers (follow llama.cpp pattern)
- Context pool handles concurrent access with channels and locks
- Each pooled context used by one goroutine at a time

**Performance Considerations:**
- Context pool reuse avoids model reload overhead
- Default MaxConcurrent=2 balances throughput and VRAM
- PNG encoding adds ~50-100ms overhead (acceptable tradeoff for compatibility)
- Model loading is one-time cost per pool context (~5-15 seconds)
- Generation time depends on steps, size, GPU: ~5-30 seconds typical

**Integration with Existing Pipeline:**
```go
// In imagegen/generator.go NewGeneratorFromConfig()
// Add SD detection before cloud provider selection:

// Check for local SD availability
if cfg.SDModelPath != "" {
    if _, err := os.Stat(cfg.SDModelPath); err == nil {
        // SD model exists, try to create SD provider
        sdProvider, err := sd.NewClient(sd.DefaultClientConfig().WithModelPath(cfg.SDModelPath))
        if err == nil && sdProvider.IsReady() {
            // SD available, use it
            return newSDGenerator(sdProvider, client, logger, config)
        }
        // SD initialization failed, fall through to cloud
        logger.Warn("SD model found but failed to initialize, falling back to cloud", zap.Error(err))
    }
}

// Fall back to existing cloud provider logic (OpenAI/Azure)
```

**Error Recovery:**
- Out-of-VRAM: Return ErrOutOfVRAM, let caller retry with lower settings or wait
- Model load failure: Return ErrModelLoadFailed with details
- Generation timeout: Return ErrGenerationTimeout after SD_TIMEOUT_SECONDS
- Context pool exhaustion: Wait for available context with timeout (ErrAcquireTimeout)

**Testing Strategy:**
- Unit tests for atoms (image_utils, validation functions) - use small test images
- Integration tests for CGo bindings - require test model download
- Mock tests for error conditions - use stub build
- Document test model download in README (SD 1.5 from HuggingFace, ~4GB)
- CI/CD: Run stub tests by default, CGo tests only when `SD_MODEL_PATH` is set
