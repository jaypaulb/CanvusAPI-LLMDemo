# Image Generation Requirements

## Overview

Phase 3 adds local image generation capabilities to CanvusLocalLLM using stable-diffusion.cpp with CUDA acceleration. This enables users to generate images from text prompts entirely on local hardware without cloud dependencies, maintaining the zero-configuration philosophy.

## Product Context

**Product:** CanvusLocalLLM - zero-configuration, fully local AI integration for Canvus
**Target Users:** Enterprise organizations, collaborative teams, privacy-conscious professionals
**Core Value Proposition:** Complete local privacy with zero cloud dependencies
**Target Hardware:** NVIDIA RTX GPUs with CUDA support (Windows and Linux)

## Business Requirements

### BR-1: Zero-Configuration Image Generation
**Priority:** P0
**Description:** Image generation must work out-of-the-box with no model selection, no provider configuration, and no additional setup beyond Canvus credentials.

**Acceptance Criteria:**
- Users install once with bundled stable-diffusion.cpp model
- No additional configuration required beyond existing .env
- First image generation works immediately after installation
- No separate service or process management needed

### BR-2: Complete Local Privacy
**Priority:** P0
**Description:** All image generation must occur on local hardware with zero external data transmission.

**Acceptance Criteria:**
- No cloud API calls for image generation
- No telemetry or analytics transmission
- All model inference happens via local stable-diffusion.cpp
- No fallback to cloud providers
- Processing completely offline-capable

### BR-3: Native Canvus Integration
**Priority:** P0
**Description:** Generated images must appear directly on the Canvus canvas as native image widgets with intelligent placement.

**Acceptance Criteria:**
- Images automatically uploaded to Canvus after generation
- Positioned near the triggering prompt widget
- Appear as standard canvas image widgets (not external links)
- Support existing `{{ }}` prompt syntax for consistency

### BR-4: Cross-Platform Support
**Priority:** P0
**Description:** Image generation must work identically on Windows and Linux with NVIDIA RTX GPUs.

**Acceptance Criteria:**
- Windows builds with MSVC + CUDA toolchain
- Linux builds with GCC + CUDA toolchain
- Identical functionality across platforms
- Bundled with existing installers (.exe, .deb, .tar.gz)

## Functional Requirements

### FR-1: stable-diffusion.cpp Integration
**Priority:** P0
**Description:** Embed stable-diffusion.cpp C++ library via CGo bindings following the same pattern as llama.cpp integration.

**Acceptance Criteria:**
- Create `sdruntime/` package with Go CGo wrapper
- Load stable-diffusion.cpp as shared library (.dll/.so)
- Support CUDA acceleration on NVIDIA RTX GPUs
- Thread-safe context management for concurrent requests
- Proper memory management and cleanup
- Error handling from C layer translated to Go errors

**Technical Details:**
- Follow llamaruntime/ package structure as template
- Use CMake for cross-platform C++ builds
- Link against CUDA runtime libraries
- Bundle native libraries with Go binary in installer

### FR-2: Text-to-Image Pipeline
**Priority:** P0
**Description:** Process text prompts from canvas widgets and generate images using embedded stable-diffusion.cpp.

**Acceptance Criteria:**
- Detect image generation prompts in `{{ }}` syntax
- Accept prompt text as input
- Generate images via stable-diffusion.cpp CGo calls
- Return generated image data (PNG format)
- Handle generation errors gracefully
- Support configurable generation parameters (size, steps, guidance)

**Technical Details:**
- Default image size: 512x512 or 768x768 (TBD based on model)
- Default inference steps: 20-30
- Default guidance scale: 7.5
- Timeout: 60 seconds per generation
- Format: PNG with reasonable compression

### FR-3: Canvas Image Placement
**Priority:** P0
**Description:** Upload generated images to Canvus and position them intelligently relative to the triggering prompt.

**Acceptance Criteria:**
- Upload image to Canvus via existing API client
- Position image near the original prompt widget
- Use relative coordinate system correctly
- Handle upload failures with retry logic
- Clean up temporary files after upload

**Technical Details:**
- Use existing `canvusapi.Client.UploadImage()` method
- Place image offset from prompt widget (e.g., +300px X, +50px Y)
- Respect canvas coordinate system (relative to parent)
- Store temporary images in downloads/ directory
- Delete temp files after successful upload

### FR-4: Model Selection and Bundling
**Priority:** P0
**Description:** Select an appropriate Stable Diffusion model, bundle it with installers, and configure stable-diffusion.cpp to use it.

**Acceptance Criteria:**
- Choose model optimized for RTX GPU VRAM constraints
- Bundle model with Windows and Linux installers
- Load model automatically on application startup
- No user model selection interface needed
- Model size appropriate for 8-16GB VRAM

**Model Candidates:**
- Stable Diffusion v1.5 (smaller, proven, widely compatible)
- Stable Diffusion v2.1 (higher quality, larger)
- SDXL-Turbo (faster inference, good quality, optimized)
- Recommendation: Start with SD v1.5 for compatibility, evaluate SDXL-Turbo for speed

**Technical Details:**
- Model format: .safetensors or stable-diffusion.cpp native format
- Install location: `models/sd-model.safetensors`
- Quantization: F16 or Q8_0 to balance quality and VRAM usage
- Model size target: <4GB for bundling feasibility

### FR-5: CMake Build Integration
**Priority:** P0
**Description:** Integrate stable-diffusion.cpp CMake build process with CUDA support for Windows and Linux.

**Acceptance Criteria:**
- CMake build produces shared libraries (.dll for Windows, .so for Linux)
- CUDA support enabled and linked correctly
- Libraries bundled in installer packages
- Build process documented for reproducibility
- CI/CD integration possible

**Build Targets:**
- Windows: stable-diffusion.dll with CUDA 11.8+
- Linux: libstable-diffusion.so with CUDA 11.8+
- Output directory: `lib/` in project root
- CMake flags: `-DGGML_CUDA=ON`, optimization flags

**Dependencies:**
- CUDA Toolkit 11.8 or newer
- cuDNN (if required by stable-diffusion.cpp)
- CMake 3.18+
- MSVC 2022 (Windows) or GCC 9+ (Linux)

### FR-6: Prompt Detection and Processing
**Priority:** P1
**Description:** Extend existing `{{ }}` prompt detection to distinguish image generation requests from text generation.

**Acceptance Criteria:**
- Support explicit image generation syntax (e.g., `{{image: prompt}}`)
- OR detect image intent via keywords (e.g., `{{generate/create/draw image of...}}`)
- Route to image generation handler instead of text handler
- Maintain backward compatibility with existing text prompts

**Technical Details:**
- Option A: Prefix-based routing (`{{image: }}` → image gen, `{{ }}` → text gen)
- Option B: Intent detection with keyword matching
- Preference: Prefix-based for clarity and simplicity
- Implementation location: `monitorcanvus.go` in `handleUpdate()`

### FR-7: Configuration Management
**Priority:** P1
**Description:** Add image generation configuration to existing .env with sensible defaults.

**Acceptance Criteria:**
- Optional configuration variables (all with defaults)
- Default behavior works without any config changes
- Configuration loaded via existing `core.Config` struct
- Values validated on startup

**Configuration Variables:**
```env
# Image Generation (all optional with defaults)
SD_IMAGE_SIZE=512              # Default: 512 (512x512)
SD_INFERENCE_STEPS=25          # Default: 25
SD_GUIDANCE_SCALE=7.5          # Default: 7.5
SD_TIMEOUT_SECONDS=60          # Default: 60
SD_MAX_CONCURRENT=2            # Default: 2 (lower than text due to VRAM)
```

### FR-8: Error Handling and Recovery
**Priority:** P1
**Description:** Handle image generation failures gracefully with helpful error messages and automatic recovery.

**Acceptance Criteria:**
- Detect CUDA out-of-memory errors and report clearly
- Handle model loading failures on startup
- Timeout long-running generations
- Log errors to app.log with context
- Display user-friendly error messages on canvas
- Recover from transient failures without restart

**Error Scenarios:**
- Model file missing or corrupted
- CUDA initialization failure
- VRAM exhausted
- Generation timeout
- Canvas upload failure
- Invalid prompt format

## Non-Functional Requirements

### NFR-1: Performance
**Priority:** P0
**Description:** Image generation must complete in reasonable time on target hardware.

**Acceptance Criteria:**
- 512x512 image in <30 seconds on RTX 3060 or better
- 768x768 image in <60 seconds on RTX 3060 or better
- CUDA acceleration functional and utilized
- VRAM usage within 8GB for concurrent operations
- No significant memory leaks over extended usage

### NFR-2: Reliability
**Priority:** P0
**Description:** Image generation must be stable and not crash the application.

**Acceptance Criteria:**
- No crashes from CGo boundary issues
- Proper cleanup of GPU resources
- Memory leaks prevented
- Graceful degradation on errors
- Automatic recovery from transient failures

### NFR-3: Maintainability
**Priority:** P1
**Description:** Code must follow existing atomic design patterns and be well-documented.

**Acceptance Criteria:**
- `sdruntime/` package follows `llamaruntime/` structure
- Atomic design hierarchy respected
- CGo code documented with memory safety notes
- Build process documented
- Tests cover core functionality

### NFR-4: Compatibility
**Priority:** P0
**Description:** Image generation must not interfere with existing text generation and vision capabilities.

**Acceptance Criteria:**
- Text generation (Bunny v1.1) continues working
- No resource contention between llama.cpp and stable-diffusion.cpp
- Concurrent text and image operations supported
- Shared GPU resources managed safely

## Technical Constraints

### TC-1: CUDA Requirement
**Impact:** High
**Description:** stable-diffusion.cpp requires CUDA. CPU-only fallback is not supported.

**Implications:**
- Users without NVIDIA GPUs cannot use image generation
- Documentation must clearly state NVIDIA RTX GPU requirement
- Installation should check for CUDA availability
- Error messages must guide users without CUDA

### TC-2: VRAM Constraints
**Impact:** Medium
**Description:** Stable Diffusion models and inference require significant VRAM.

**Implications:**
- Minimum 6GB VRAM recommended (8GB preferred)
- Concurrent text + image operations may exhaust VRAM
- Limit concurrent image generations (default: 2)
- Consider smaller models or quantization for lower-end GPUs

### TC-3: Model Size and Bundling
**Impact:** Medium
**Description:** Stable Diffusion models are 2-7GB, affecting installer size.

**Implications:**
- Installer size increases significantly
- Consider separate model download vs bundled approach
- Download infrastructure already exists from Bunny v1.1 integration
- Tradeoff: Bundled = zero-config, Download = smaller installer

**Recommendation:** Bundle SD v1.5 (~4GB) directly for zero-config experience.

### TC-4: Cross-Platform Build Complexity
**Impact:** High
**Description:** Building stable-diffusion.cpp with CUDA for Windows and Linux requires significant toolchain setup.

**Implications:**
- Windows: Requires Visual Studio 2022 + CUDA Toolkit
- Linux: Requires GCC + CUDA Toolkit
- CMake build configuration must be cross-platform
- CI/CD pipeline needs GPU-enabled build agents
- Documentation must cover build prerequisites

## Dependencies

### Internal Dependencies
- `core/config.go`: Configuration management
- `canvusapi/canvusapi.go`: Image upload functionality
- `monitorcanvus.go`: Prompt detection and routing
- `handlers.go`: Handler registration (or new `imagegen/` package)
- Existing CMake build patterns from llama.cpp integration

### External Dependencies
- **stable-diffusion.cpp**: C++ library for image generation (llama.cpp ecosystem)
- **CUDA Toolkit**: 11.8+ for GPU acceleration
- **CMake**: 3.18+ for cross-platform builds
- **Model Files**: Stable Diffusion v1.5 or equivalent (~4GB)

### Build Dependencies
- Windows: Visual Studio 2022, CUDA Toolkit, CMake
- Linux: GCC 9+, CUDA Toolkit, CMake
- Go toolchain with CGO_ENABLED=1

## Testing Requirements

### TR-1: Unit Tests
**Priority:** P1
**Description:** Test individual components of the image generation pipeline.

**Test Coverage:**
- `sdruntime/` CGo wrapper functions
- Model loading and initialization
- Image generation with mock prompts
- Error handling for invalid inputs
- Memory cleanup and resource management

### TR-2: Integration Tests
**Priority:** P0
**Description:** Test end-to-end image generation and canvas placement.

**Test Coverage:**
- Generate image from prompt via stable-diffusion.cpp
- Upload image to Canvus mock server
- Verify image placement coordinates
- Test concurrent image generations
- Test CUDA acceleration is utilized

### TR-3: Platform Testing
**Priority:** P0
**Description:** Verify functionality on target platforms.

**Test Coverage:**
- Windows 10/11 with NVIDIA RTX GPU
- Linux (Ubuntu 20.04+, Debian 11+) with NVIDIA RTX GPU
- Installer integration (bundled model, library paths)
- Service mode (Windows Service, systemd)

### TR-4: Performance Testing
**Priority:** P1
**Description:** Validate performance meets requirements.

**Test Coverage:**
- Generation time for 512x512 images
- Generation time for 768x768 images
- VRAM usage during generation
- Concurrent text + image operations
- Memory leak detection over 100+ generations

### TR-5: Error Scenario Testing
**Priority:** P1
**Description:** Validate error handling and recovery.

**Test Coverage:**
- Missing model file
- CUDA not available
- Out of VRAM
- Generation timeout
- Canvas upload failure
- Invalid prompt format

## Success Metrics

### Immediate Success Criteria (Phase 3 Completion)
- [ ] Users can generate images locally without cloud dependencies
- [ ] Images appear on Canvus canvas automatically
- [ ] Works on Windows and Linux with NVIDIA RTX GPUs
- [ ] Bundled with existing installers
- [ ] No additional configuration required beyond .env

### Quality Metrics
- Image generation success rate: >95%
- Average generation time (512x512): <30s on RTX 3060
- Zero crashes from image generation over 100 operations
- VRAM usage stays within 8GB bounds
- User-reported issues: <5 per 100 users

### User Experience Metrics
- Time from installation to first generated image: <5 minutes
- User understands how to trigger image generation: >90%
- Users prefer local generation over cloud: >80% (for privacy-conscious segment)

## Future Considerations

### Potential Enhancements (Not in Scope for Phase 3)
- Support for different aspect ratios (16:9, 4:3, etc.)
- Image-to-image generation (modify existing canvas images)
- Style transfer or artistic filters
- ControlNet integration for guided generation
- Upscaling generated images
- Batch generation from multiple prompts
- Model switching or multiple model support
- LoRA support for style customization

### Known Limitations
- NVIDIA GPU required (no CPU fallback)
- Single model bundled (no user choice)
- Limited to text-to-image (no image-to-image initially)
- Fixed aspect ratios
- No real-time preview during generation
- Generation time varies with GPU capability

## Open Questions

### Q1: Model Selection
**Question:** Should we bundle Stable Diffusion v1.5, v2.1, or SDXL-Turbo?
**Options:**
- SD v1.5: Smaller (~2-3GB), proven, widely compatible, lower VRAM
- SD v2.1: Better quality, larger (~5GB), higher VRAM requirements
- SDXL-Turbo: Fast inference, good quality, optimized, ~4GB

**Recommendation:** Start with SD v1.5 for compatibility and lower VRAM. Evaluate SDXL-Turbo if performance is acceptable.

### Q2: Prompt Syntax
**Question:** How should users differentiate image generation from text generation?
**Options:**
- A) Prefix-based: `{{image: prompt}}` vs `{{prompt}}`
- B) Intent detection: Parse prompt text for image-related keywords
- C) Separate syntax: `[[prompt]]` for images, `{{prompt}}` for text

**Recommendation:** Option A (prefix-based) for clarity and simplicity. Easy to implement, clear to users, no ambiguity.

### Q3: Model Bundling Strategy
**Question:** Should we bundle the SD model with installers or download on first run?
**Options:**
- A) Bundle with installer: True zero-config, larger download (~4GB)
- B) Download on first run: Smaller installer, requires internet once, reuses existing download infrastructure

**Recommendation:** Option A (bundle) to maintain zero-config philosophy and complete offline capability.

### Q4: Concurrent Operations
**Question:** How should we handle concurrent text and image generation?
**Options:**
- A) Shared queue: Single processor, serialize all AI operations
- B) Separate queues: Independent processing, potential VRAM contention
- C) Priority queue: Text generation prioritized over image generation

**Recommendation:** Option B (separate queues) with VRAM monitoring. Set `SD_MAX_CONCURRENT=2` by default, lower than text generation limit.

### Q5: Image Placement Strategy
**Question:** Where should generated images be placed on the canvas?
**Options:**
- A) Fixed offset from prompt widget (e.g., +300px X, +50px Y)
- B) Smart placement based on canvas empty space
- C) Below prompt widget with dynamic spacing
- D) User-configurable offset in .env

**Recommendation:** Start with Option A (fixed offset) for simplicity. Option B could be future enhancement.

## References

- **stable-diffusion.cpp**: https://github.com/leejet/stable-diffusion.cpp
- **llama.cpp ecosystem**: https://github.com/ggerganov/llama.cpp
- **Stable Diffusion models**: https://huggingface.co/runwayml/stable-diffusion-v1-5
- **SDXL-Turbo**: https://huggingface.co/stabilityai/sdxl-turbo
- **Canvus API**: Internal documentation
- **Product Roadmap**: `/home/jaypaulb/Projects/gh/CanvusLocalLLM/agent-os/product/roadmap.md`
