# Specification: Vision Inference in llamaruntime

## Goal
Enable local multimodal image understanding in the llamaruntime package using llama.cpp vision support and Bunny v1.1 model, replacing cloud-based Google Vision API with offline inference for handwriting recognition, OCR, and canvas analysis.

## User Stories
- As a CanvusLocalLLM user, I want to perform OCR on handwritten notes using my local GPU so that I don't need internet connectivity or cloud API costs
- As a developer, I want vision inference to follow the same patterns as text inference so that the API is consistent and predictable

## Specific Requirements

**llama.cpp Vision API Bindings**
- Research and identify llama.cpp C functions for vision/multimodal inference (no `llama_image_*` or `llama_clip_*` functions currently bound)
- Add C function declarations to `bindings.go` header section following existing CGo binding patterns
- Determine if llama.cpp requires separate vision encoder context or uses main context
- Verify Bunny model GGUF includes vision encoder weights (SigLIP-SO400M/14@384)
- Handle image data marshaling between Go and C (raw bytes vs file path vs embedded tokens)

**Image Preprocessing and Validation**
- Accept image input as raw bytes (`ImageData []byte`) or file path (`ImagePath string`)
- Validate image formats: JPEG, PNG, BMP, WebP (GIF first frame only)
- Resize images to 384x384 pixels maintaining aspect ratio with padding (Bunny requirement)
- Implement format detection and size validation before C layer (use `MaxImageSize` config)
- Use Go `image/*` standard library packages for preprocessing if llama.cpp doesn't handle internally
- Return `ErrInvalidImage` sentinel error for unsupported formats or invalid data

**Vision Inference Implementation**
- Implement `inferVision()` function at `bindings.go:758-776` (currently stubbed with TODO)
- Follow `inferText()` pattern: tokenize prompt → encode image → create batch → decode loop → sample tokens → assemble result
- Insert ~576 vision tokens from SigLIP encoder into context before text prompt
- Verify prompt + vision tokens + max response tokens fit within context size (default 2048 tokens)
- Apply sampling parameters: temperature, top-k, top-p, repeat penalty
- Use thread-safe C API calls with `llamaCtx.mu.Lock()` during decode operations
- Return generated text analysis/description of image

**Context Pool Integration**
- Use existing `ContextPool` from `context.go` for vision inference contexts
- Share VRAM budget with text inference (no separate pool needed)
- Vision contexts require ~1.5GB additional VRAM per context for Q5_K_M quantization
- Track vision inference statistics using existing `totalVisionInfers` counter (already defined in client)
- Leverage `Acquire()` / `Release()` pattern for concurrent vision operations

**High-Level API Wiring**
- Wire `inferVision()` into existing `Client.InferVision()` at `client.go:372-505` (currently calls stubbed function)
- `VisionParams` struct already defined in `types.go:180-216` with all required fields
- Handle timeout via `context.WithTimeout()` pattern (vision overhead ~250-450ms before generation)
- Update statistics: `totalInferences`, `totalVisionInfers`, `totalTokensGen`, `totalTokensPrompt`, `totalDuration`
- Return `InferenceResult` with text, token counts, duration, tokens/second, stop reason

**Handlers Integration for OCR**
- Create `performLocalVisionInference()` function in `handlers.go` parallel to `performGoogleVisionOCR()` at line 2399
- Accept `imageData []byte`, `config *core.Config`, `log *logging.Logger` parameters
- Use Bunny OCR prompt template: "Extract all text from this image. If the text is handwritten, do your best to transcribe it accurately. Note any parts that are unclear or illegible."
- Return `(string, error)` signature matching Google Vision OCR function
- Implement fallback strategy: if local vision fails or unavailable, fall back to Google Vision API
- Add `USE_LOCAL_VISION` environment variable to `.env` for feature toggle (default: true for local-first approach)

**Error Handling and Timeouts**
- Use existing `LlamaError` struct with `Op`, `Code`, `Message`, `Err` fields
- Handle insufficient VRAM errors with helpful message about context pool limits
- Wrap errors with context: `fmt.Errorf("failed to encode image: %w", err)`
- Apply timeout from `VisionParams.Timeout` (defaults to `DefaultTimeout`)
- Cancel inference on context cancellation via `select` on `ctx.Done()` channel
- Return partial results on timeout during generation phase (same as text inference)

**Performance and Resource Management**
- Vision encoding overhead: ~200-400ms for SigLIP encoder (GPU-dependent)
- Image preprocessing overhead: ~20ms for resize to 384x384 in Go
- Total first-token latency: ~250-500ms overhead vs text-only inference
- VRAM budget for 3 concurrent contexts: ~5.5GB model + 4.5GB contexts + 0.5GB workspace = 10.5GB total
- Risk mitigation: context pool limits prevent VRAM exhaustion on <16GB GPUs
- Context size consideration: ~576 image tokens reduce available response tokens (2048 - 576 - prompt = response budget)

## Visual Design
No visual assets provided (code-only implementation spec).

## Existing Code to Leverage

**`llamaruntime/bindings.go:658-756` - Text Inference Pattern**
- Pattern: tokenize prompt → setup batch → decode prompt → sampling loop → decode token → assemble result
- Reuse: Vision inference follows identical structure but adds image encoding step before prompt tokenization
- Thread safety: Use `llamaCtx.mu.Lock()` during all C API calls to `llama_decode()` and `llama_sampler_sample()`
- EOS detection: Check for `llama_token(llamaCtx.model.EOSToken())` to stop generation
- Context cancellation: Use `select` on `ctx.Done()` in generation loop

**`llamaruntime/client.go:372-505` - InferVision High-Level API**
- Already implemented: parameter validation, context acquisition, timeout handling, statistics tracking
- Calls stubbed `inferVision()` at line 462 - only need to implement the underlying C layer
- Use `DefaultVisionParams()` for sensible defaults (MaxTokens, Temperature, Timeout)
- Statistics already wired: `atomic.AddInt64(&c.totalVisionInfers, 1)` at line 489

**`llamaruntime/context.go` - Context Pool Pattern**
- Channel-based pool with `Acquire()` / `Release()` pattern for lock-free operation
- Timeout handling via context: `select { case ctx := <-p.contexts: ... case <-ctx.Done(): ... }`
- Statistics tracking: `acquiredAt` timestamps for debugging stuck contexts
- Reuse: Vision contexts use same pool as text contexts (shared VRAM budget)

**`ocrprocessor/processor.go:89-91` - Progress Callback Pattern**
- `ProgressCallback func(stage string, progress float64, message string)` for long-running operations
- Consider adding to `VisionParams` for vision tasks: "validating" → "preprocessing" → "encoding" → "generating"
- Report progress during image preprocessing and vision encoding phases (useful for large images)

**`handlers.go:2399-2435` - Google Vision OCR Integration**
- Current: `performGoogleVisionOCR()` creates `ocrprocessor.Processor`, calls `ProcessImage()`, returns text
- New parallel: `performLocalVisionInference()` creates llamaruntime client, calls `InferVision()`, returns text
- Fallback logic: try local first, fall back to Google Vision on error if `USE_LOCAL_VISION=fallback` mode
- Error handling: wrap errors with context, log processing times and text length

## Out of Scope
- Multi-image batch processing (single image per inference call only)
- Video or animated GIF frame-by-frame processing (first frame only as per Bunny docs)
- Real-time video streaming analysis (complete static images only)
- Image generation capabilities (separate stable-diffusion.cpp integration task)
- Custom vision encoder support (use Bunny's SigLIP-SO400M/14@384 only)
- Fine-tuning or training vision models (pre-trained Bunny model only)
- Advanced image preprocessing beyond resize/pad (no filters, enhancement, augmentation)
- Progress callbacks in initial implementation (optional future enhancement)
- Automatic image quality optimization or resolution selection
- Multi-modal fusion beyond single image + text prompt
