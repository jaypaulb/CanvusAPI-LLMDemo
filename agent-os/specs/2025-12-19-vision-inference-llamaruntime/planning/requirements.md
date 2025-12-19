# Spec Requirements: Vision Inference in llamaruntime

## Initial Description
Implement vision inference capabilities in the llamaruntime package to enable local image understanding using llama.cpp multimodal support with Bunny v1.1 model.

**Strategic Importance**: HIGH - Critical for local-first vision. Replaces Google Vision API (cloud, costs money, requires internet) with offline image understanding.

**Issue**: CanvusLocalLLM-x5pq, Priority: P2

## Requirements Discussion

### Code Analysis Findings

After examining the existing codebase, I identified the following implementation patterns and requirements:

**Q1: What is the current state of vision inference in llamaruntime?**
**Answer**:
- The `inferVision()` function is stubbed at `llamaruntime/bindings.go:762` with a TODO comment
- The high-level API already exists: `Client.InferVision(ctx, VisionParams)` in `client.go:372-505`
- `VisionParams` struct is defined in `types.go:180-216` with fields for `ImageData`, `ImagePath`, `Prompt`, `MaxTokens`, `Temperature`, `Timeout`
- Error handling infrastructure exists with `ErrInvalidImage` sentinel error in `errors.go:60-62`
- The API is designed to accept either raw `ImageData []byte` OR `ImagePath string`
- Current implementation returns "vision inference not yet implemented" error

**Q2: How is the existing OCR processor structured?**
**Answer**:
- Located in `ocrprocessor/` package (2,350 lines, 94.5% coverage)
- Uses Google Vision API for handwriting recognition (cloud-based, requires API key and internet)
- Called from `handlers.go:2399-2435` via `performGoogleVisionOCR()`
- Takes `imageData []byte` as input and returns extracted text
- Uses progress callbacks: `ProgressCallback func(stage string, progress float64, message string)`
- Has configurable timeouts, max image size, and supported formats

**Q3: What llama.cpp vision APIs are available?**
**Answer**:
- Based on `bindings.go` analysis: NO explicit vision/image functions are currently declared in the C API bindings
- Existing C functions include: `llama_tokenize`, `llama_decode`, `llama_get_logits`, etc. (lines 106-140)
- **CRITICAL FINDING**: No `llama_image_*` or `llama_clip_*` or `llama_vision_*` functions found
- **CONCLUSION**: We will need to research and add llama.cpp vision API bindings as part of this implementation

**Q4: What is the Bunny model's vision architecture?**
**Answer** (from `docs/bunny-model.md`):
- Model: Bunny-v1.1-LLaMA-3-8B-V with SigLIP-SO400M/14@384 vision encoder
- Image processing: Input images resized to **384x384 pixels** with aspect ratio preserved via padding
- Image tokens: ~576 visual tokens added to context (reduces available text context)
- Supports formats: JPEG, PNG, BMP, GIF (first frame), WebP
- Vision encoding overhead: ~200-400ms (GPU-dependent) plus ~20ms resize/preprocess
- Good for: OCR (printed > handwritten), diagram analysis, object recognition, scene description
- Moderate for: Fine details (limited by 384x384 resolution)

**Q5: What existing patterns should we follow?**
**Answer**:
- **Context pool pattern**: Use existing `ContextPool` in `context.go` for concurrent vision inference
- **Error handling**: Use `LlamaError` struct with `Op`, `Code`, `Message`, `Err` fields
- **Public API**: Mirror `Client.Infer()` pattern - already done in `Client.InferVision()`
- **Progress reporting**: OCR processor uses `ProgressCallback` - consider adding to `VisionParams`
- **Validation**: Check image size limits, format detection before calling C layer
- **Thread safety**: `llamaContext.mu.Lock()` during C API calls (see `bindings.go:750-752`)
- **Timeout handling**: Use `context.WithTimeout()` pattern from `client.go:449-450`

**Q6: What prompt templates does Bunny use for vision tasks?**
**Answer** (from `docs/bunny-model.md`):
- **Handwriting/OCR**: "Extract all text from this image. If the text is handwritten, do your best to transcribe it accurately. Note any parts that are unclear or illegible."
- **General description**: "Describe this image in detail. Include: Main subjects and objects, Colors and composition, Any text visible in the image, Overall context or setting"
- **Canvas analysis**: "This is a screenshot of a collaborative canvas. Identify and describe: All visible widgets (notes, images, shapes), Any text content on notes, The apparent organization or grouping, Key themes or topics represented"
- **Diagram analysis**: "Analyze this diagram and explain: 1. The main components shown, 2. How they connect or relate, 3. The overall purpose or workflow"
- **Best practice**: Use structured prompts, be specific, set context, limit scope

**Q7: How will this integrate with handlers.go?**
**Answer**:
- Current integration point: `performGoogleVisionOCR()` at `handlers.go:2399-2435`
- Should create parallel function: `performLocalVisionInference()` that uses llamaruntime instead
- Need to handle fallback: If vision inference fails or is unavailable, fall back to Google Vision API
- Environment variable control: Add `USE_LOCAL_VISION=true/false` to `.env` configuration
- Pass through existing `imageData []byte` parameter format
- Return same signature: `(string, error)` for extracted text

**Q8: What are the memory and performance considerations?**
**Answer**:
- Vision encoding adds ~576 tokens to context (reduces available context for response)
- Additional VRAM per context: ~1.5GB for Q5_K_M quantization
- Vision overhead: ~250-450ms before text generation starts (image load + resize + encoding)
- Current `ContextSize` default: 2048 tokens (may need adjustment for vision tasks)
- Vision inference should use same context pool as text inference (shared VRAM budget)
- Risk: Multiple concurrent vision requests could exhaust VRAM - need to monitor

**Q9: What testing approach should we use?**
**Answer**:
- **Unit tests**: Image preprocessing functions (resize, format detection, validation)
- **Integration tests**: Vision inference with test images in `llamaruntime/integration_test.go` pattern
- **Test images needed**:
  - Printed text sample (high OCR accuracy expected)
  - Handwritten text sample (moderate OCR accuracy expected)
  - Simple diagram/chart (object recognition test)
  - Canvas screenshot (real-world use case)
- **Error cases**: Invalid image formats, corrupted data, oversized images, timeout scenarios
- **Performance benchmarks**: Add to `llamaruntime/benchmark_test.go` for tokens/sec tracking

**Q10: What should be explicitly excluded from initial implementation?**
**Answer**:
- **Multi-image batch processing**: Only single image per inference call initially
- **Video/animated GIF processing**: First frame only (as documented for Bunny)
- **Real-time streaming**: Process complete images only, not video frames
- **Image generation**: This is separate (stable-diffusion.cpp integration, not Bunny)
- **Fine-tuning/training**: Use pre-trained Bunny model only
- **Custom vision encoders**: Use Bunny's built-in SigLIP encoder only

### Existing Code Reuse

**Similar Features Identified:**

1. **Text inference pattern** (`llamaruntime/bindings.go:658-756`)
   - Path: `llamaruntime/bindings.go` function `inferText()`
   - Pattern: Context acquisition → tokenization → batch creation → decode loop → sampling → result assembly
   - **Reuse**: Vision inference should follow similar structure but with image encoding step added

2. **Context pool management** (`llamaruntime/context.go`)
   - Path: `llamaruntime/context.go` - `ContextPool` type with `Acquire()` and `Release()`
   - Pattern: Channel-based pool, timeout handling, statistics tracking
   - **Reuse**: Use same pool for vision contexts (no separate pool needed)

3. **OCR processor progress reporting** (`ocrprocessor/processor.go:89-91`)
   - Path: `ocrprocessor/processor.go` - `ProgressCallback` function type
   - Pattern: Report stage, progress (0.0-1.0), and human-readable message
   - **Reuse**: Consider adding optional progress callback to `VisionParams` for long-running vision tasks

4. **Image download and validation** (`imagegen/downloader.go`)
   - Path: `imagegen/downloader.go` (part of 4,881 line package)
   - Pattern: HTTP download, format detection, size validation
   - **Reuse**: Reference for image format validation patterns

5. **Error handling** (`llamaruntime/errors.go`)
   - Path: `llamaruntime/errors.go` - `LlamaError` struct and sentinel errors
   - Pattern: Structured errors with operation, code, message, wrapped error
   - **Reuse**: Use existing `ErrInvalidImage` sentinel, add vision-specific errors if needed

6. **CGo bindings patterns** (`llamaruntime/bindings.go:106-140`)
   - Path: `llamaruntime/bindings.go` - C function declarations and wrappers
   - Pattern: C function extern declarations → Go wrapper with error handling
   - **Reuse**: Follow same pattern for new llama.cpp vision API bindings

### Follow-up Questions

Based on code analysis, I have clarifying questions about llama.cpp vision API implementation:

**Follow-up 1: llama.cpp Vision API Discovery**
The existing `bindings.go` does not include any vision-related C functions (no `llama_image_*`, `llama_clip_*`, or similar). This suggests either:
- (A) llama.cpp's vision API is not yet integrated into this project's bindings
- (B) Vision support in llama.cpp uses a different mechanism than expected

**Question**: Do we need to research llama.cpp's vision API documentation to determine which C functions to bind, or do you have knowledge of the specific API calls required for Bunny vision inference?

**Follow-up 2: Image Preprocessing Location**
The Bunny model requires images to be resized to 384x384 pixels with aspect ratio preservation. This could be done:
- (A) In Go (before calling C layer) using `image/*` standard library packages
- (B) In C/llama.cpp (let llama.cpp handle preprocessing internally)

**Question**: Should we implement image preprocessing (resize to 384x384, format conversion) in Go, or does llama.cpp handle this internally?

**Follow-up 3: Fallback Strategy During Transition**
The OCR processor currently uses Google Vision API. During the vision inference implementation and testing phase:
- (A) Keep Google Vision as primary, add local vision as experimental option
- (B) Make local vision primary, keep Google Vision as fallback for errors
- (C) Completely replace Google Vision API immediately

**Question**: What fallback/transition strategy do you prefer during development and testing?

## Visual Assets

### Files Provided:
No visual files found in `/home/jaypaulb/Projects/gh/CanvusLocalLLM/agent-os/specs/2025-12-19-vision-inference-llamaruntime/planning/visuals/`

### Visual Insights:
User specified "only the existing code and no visuals" - relying on codebase patterns and documentation analysis.

## Requirements Summary

### Functional Requirements

**Core Vision Inference Capability:**
- Implement `inferVision()` function in `llamaruntime/bindings.go` to perform multimodal inference
- Support image input as raw bytes (`ImageData []byte`) or file path (`ImagePath string`)
- Accept text prompt to guide image analysis (e.g., "Extract text from this image")
- Return generated text analysis/description of the image
- Process images through Bunny v1.1's SigLIP vision encoder (384x384 resolution)

**Image Preprocessing:**
- Validate image format (JPEG, PNG, BMP, WebP, GIF)
- Resize images to 384x384 pixels maintaining aspect ratio with padding
- Validate image size limits (configurable max size)
- Convert image data to format expected by llama.cpp vision API

**API Integration:**
- Wire `inferVision()` into existing `Client.InferVision()` high-level API (already stubbed)
- Support sampling parameters: temperature, max tokens, timeout
- Handle vision-specific context size requirements (~576 image tokens + prompt + response)
- Report inference statistics (tokens generated, processing time, tokens/second)

**Handlers Integration:**
- Create `performLocalVisionInference()` function in `handlers.go`
- Use llamaruntime vision inference for handwriting recognition (replace Google Vision API)
- Implement configurable fallback to Google Vision API if local inference fails
- Add environment variable `USE_LOCAL_VISION=true/false` for feature toggle

**Error Handling:**
- Use existing `ErrInvalidImage` for invalid/unsupported image formats
- Add timeout handling for long-running vision inference
- Proper error wrapping with `LlamaError` struct (operation, code, message)
- Handle insufficient VRAM gracefully with helpful error messages

### Reusability Opportunities

**CGo Bindings Pattern:**
- Follow existing pattern in `bindings.go` for C function declarations
- Use `C.CString()` / `C.free()` for string marshaling
- Implement thread-safe C API calls with mutex locking
- Set finalizers for automatic cleanup

**Context Pool Reuse:**
- Use existing `ContextPool` for vision inference contexts
- Leverage `Acquire()` / `Release()` pattern for concurrent operations
- Share VRAM budget with text inference (no separate pool)
- Track vision-specific statistics (`totalVisionInfers` counter already exists)

**Progress Reporting Pattern:**
- Consider adding optional `ProgressCallback` to `VisionParams` (similar to `ocrprocessor`)
- Report stages: "validating" → "preprocessing" → "encoding" → "generating"
- Useful for long-running vision tasks (>1 second)

**Testing Infrastructure:**
- Use existing test patterns from `llamaruntime/integration_test.go`
- Add benchmark tests to `llamaruntime/benchmark_test.go`
- Follow table-driven test pattern from `*_test.go` files

### Scope Boundaries

**In Scope:**
- Implement vision inference for single image + text prompt
- Support JPEG, PNG, BMP, WebP image formats
- Image preprocessing (resize to 384x384, format validation)
- Integration with existing `Client.InferVision()` API
- Replace Google Vision API for handwriting/OCR use cases
- Error handling and timeout support
- Basic testing with sample images
- Documentation of vision inference API

**Out of Scope:**
- Multi-image batch processing (future enhancement)
- Video/animated GIF frame processing (first frame only initially)
- Real-time video streaming analysis
- Image generation (separate stable-diffusion.cpp integration task)
- Custom vision encoder support (use Bunny's SigLIP only)
- Fine-tuning or training custom vision models
- Advanced image preprocessing (filters, enhancement, etc.)

### Technical Considerations

**llama.cpp Vision API Research Required:**
- Need to identify which C functions to bind for vision inference
- Determine if llama.cpp handles image preprocessing internally or if Go must do it
- Understand how to pass image data to llama.cpp (raw bytes, file path, embedded tokens)
- Check if separate vision encoder context is needed or if it's part of main context

**Integration Points:**
- Existing: `Client.InferVision()` in `client.go:372-505` (already implemented, calls stubbed `inferVision()`)
- Existing: `VisionParams` struct in `types.go:180-216`
- New: `inferVision()` implementation in `bindings.go:758-776`
- New: llama.cpp vision API C function bindings in `bindings.go` header section
- Modify: `performGoogleVisionOCR()` in `handlers.go:2399-2435` to support local vision fallback

**VRAM Constraints:**
- Vision encoding adds ~576 tokens to context window
- Each vision context requires ~1.5GB additional VRAM (Q5_K_M quantization)
- Total VRAM for 3 contexts: ~5.5GB (model) + 4.5GB (contexts) + 0.5GB (workspace) = 10.5GB
- Risk: Concurrent vision requests may exhaust VRAM on <16GB GPUs
- Mitigation: Use existing context pool limits, monitor GPU memory

**Performance Expectations:**
- Image preprocessing: ~20ms (resize to 384x384 in Go)
- Vision encoding: ~200-400ms (GPU-dependent, SigLIP encoder)
- Text generation: Same as existing text inference (~18-100 tokens/sec depending on GPU)
- Total first-token latency: ~250-500ms overhead compared to text-only inference

**Fallback Strategy:**
- Phase 1 (Development): Local vision experimental, Google Vision API primary
- Phase 2 (Testing): Local vision primary, Google Vision API fallback on errors
- Phase 3 (Production): Local vision only, remove Google Vision API dependency
- Configuration: `USE_LOCAL_VISION` environment variable controls behavior

**Dependencies:**
- llama.cpp must have vision/multimodal support compiled in (likely already true for Bunny)
- Bunny v1.1 GGUF model must include vision encoder weights
- CUDA libraries must support vision operations (likely transparent)
- Go standard library `image/*` packages for preprocessing (if needed)

**Testing Requirements:**
- Unit tests: Image validation, format detection, preprocessing functions
- Integration tests: Full vision inference with sample images
- Performance benchmarks: Tokens/sec, latency measurements
- Error tests: Invalid formats, oversized images, timeouts, corrupted data
- Real-world tests: Canvas screenshots, handwriting samples, diagrams

**Documentation Needs:**
- API documentation for `inferVision()` function
- Usage examples in `llamaruntime.md`
- Prompt engineering guide for vision tasks (reference `bunny-model.md`)
- Migration guide from Google Vision API to local vision
- Troubleshooting section for vision-specific issues
