# Requirements: Embedded LLM Integration (Phase 2)

## Overview

Phase 2 integrates llama.cpp inference engine with Bunny v1.1 Llama-3-8B-V multimodal model to enable fully local, CUDA-accelerated AI processing. This replaces the current OpenAI API dependency with an embedded solution running entirely on NVIDIA RTX GPUs.

## Functional Requirements

### FR1: llama.cpp CGo Integration

**FR1.1: Native Library Bindings**
- Create `llamaruntime/` package in Go codebase
- Implement CGo wrapper for llama.cpp C++ API
- Provide Go-friendly API that abstracts C memory management
- Support model loading from filesystem path
- Support context creation with configurable parameters
- Support text generation with streaming and non-streaming modes
- Support multimodal inference (text + image input)

**FR1.2: Memory Management**
- Implement safe pointer handling between Go and C
- Provide automatic cleanup via Go finalizers and defer statements
- Prevent memory leaks in long-running service
- Handle C library panics gracefully without crashing Go process
- Implement context pool for reusing loaded models across requests

**FR1.3: Error Handling**
- Translate C error codes to Go error types
- Provide descriptive error messages for common failure modes
- Distinguish between recoverable and fatal errors
- Log detailed error information for debugging

### FR2: Cross-Platform llama.cpp Build

**FR2.1: Windows Build (MSVC + CUDA)**
- CMake configuration for Windows x64 build
- CUDA 12.x support for NVIDIA RTX GPUs
- Build outputs: `llama.dll`, `llama.lib` for linking
- Visual Studio 2022 compatibility
- Static linking of runtime dependencies where possible

**FR2.2: Linux Build (GCC + CUDA)**
- CMake configuration for Linux x64 build
- CUDA 12.x support for NVIDIA RTX GPUs
- Build outputs: `libllama.so` for dynamic linking
- GCC 11+ compatibility
- Handle different Linux distributions (Ubuntu, Fedora, etc.)

**FR2.3: Build Automation**
- Build scripts for each platform (PowerShell for Windows, Bash for Linux)
- Place compiled libraries in `lib/` directory in project root
- Verify CUDA support in compiled binaries
- Document build prerequisites (CUDA toolkit version, compiler versions)

**FR2.4: Installer Integration**
- Bundle compiled llama.cpp libraries with installers
- Place libraries in install directory (`C:\Program Files\CanvusLocalLLM\lib\` or `/opt/canvuslocallm/lib/`)
- Ensure Go binary can find and load shared libraries at runtime
- Windows: Set PATH or use DLL search path
- Linux: Set LD_LIBRARY_PATH or use rpath

### FR3: Bunny v1.1 Model Integration

**FR3.1: Model Configuration**
- Configure llama.cpp specifically for Bunny v1.1 Llama-3-8B-V
- Model source: https://huggingface.co/BAAI/Bunny-v1_1-Llama-3-8B-V
- Use GGUF quantized format for efficient inference
- Recommended quantization: Q4_K_M or Q5_K_M for RTX GPUs
- Context size: 4096 tokens (or model maximum)
- Batch size: Optimize for RTX VRAM (e.g., 512)

**FR3.2: Model Loading**
- Load model from filesystem path: `models/bunny-v1.1-llama-3-8b-v.gguf`
- Support loading during application startup
- Provide startup diagnostics: model size, quantization type, context size
- Fail gracefully with clear error message if model not found
- Log model loading time and memory usage

**FR3.3: Inference Parameters**
- Temperature: 0.7 (default, configurable via environment)
- Top-p: 0.9 (default, configurable)
- Repeat penalty: 1.1 (default)
- Max tokens: Configurable per operation type (notes, PDFs, canvas analysis)
- GPU layers: Automatic detection, offload all layers to GPU if VRAM sufficient

**FR3.4: Model Validation**
- Run test inference on startup to verify model works
- Simple prompt: "Hello, how are you?"
- Verify response is coherent and non-empty
- Log inference time and tokens per second
- Fail startup if test inference fails

### FR4: Multimodal Vision Pipeline

**FR4.1: Image Input Support**
- Accept image data from Canvus API (JPEG, PNG)
- Convert image format to llama.cpp-compatible format
- Support images up to 2048x2048 pixels
- Resize or downsample if necessary for model constraints

**FR4.2: Vision Inference**
- Combine image input with text prompt
- Use Bunny's vision capabilities for image understanding
- Support prompts like "Describe this image", "What is in this diagram?", "Analyze this chart"
- Return text description/analysis

**FR4.3: Canvas Image Analysis**
- Detect when user places image widget on canvas with analysis trigger
- Download image from Canvus API
- Send to Bunny model with appropriate analysis prompt
- Create response note on canvas with analysis results
- Position note intelligently relative to image widget

**FR4.4: PDF Image Extraction**
- Extract images from PDF documents during PDF analysis
- Optionally analyze key images (charts, diagrams) using vision model
- Include image descriptions in PDF summary

### FR5: Runtime Health Monitoring

**FR5.1: Startup Diagnostics**
- Check CUDA availability and version
- Detect NVIDIA GPU(s) and report model, VRAM capacity
- Verify llama.cpp library loads successfully
- Test model loading and inference
- Log all diagnostic information to console and app.log
- Provide clear error messages if requirements not met

**FR5.2: GPU Memory Monitoring**
- Track GPU VRAM usage during inference
- Warn if VRAM usage exceeds 90% capacity
- Log GPU memory statistics periodically
- Provide memory usage in health check endpoint (future web UI)

**FR5.3: Inference Health Checks**
- Periodic inference health check (e.g., every 5 minutes)
- Simple test prompt to verify model still responsive
- Track inference success/failure rate
- Track inference latency (p50, p95, p99)
- Expose metrics via internal API for web UI (Phase 5)

**FR5.4: Automatic Recovery**
- Detect inference failures (timeout, GPU error, memory error)
- Implement retry logic with exponential backoff
- If persistent failure, attempt to reload model/context
- If still failing, log critical error and enter degraded mode
- Provide clear status in health check endpoint

**FR5.5: Graceful Shutdown**
- Clean up llama.cpp contexts on application shutdown
- Free GPU memory properly
- Close model files
- Log shutdown process

## Non-Functional Requirements

### NFR1: Performance

**NFR1.1: Inference Speed**
- Target: 20+ tokens/second on RTX 3060 or better
- Target: 40+ tokens/second on RTX 4070 or better
- First token latency: <500ms for text-only, <1s for vision
- Model loading time: <10 seconds on modern SSD

**NFR1.2: Concurrency**
- Support multiple concurrent inference requests (up to MAX_CONCURRENT)
- Use context pool to avoid reloading model for each request
- Queue requests if all contexts busy
- Implement request timeout (configurable, default 60s)

**NFR1.3: Memory Efficiency**
- Keep GPU VRAM usage under 6GB for Q4_K_M quantization
- Minimize CPU memory usage (<2GB for application overhead)
- Release memory promptly after inference completes

### NFR2: Reliability

**NFR2.1: Error Handling**
- No crashes from CGo errors
- All C errors caught and converted to Go errors
- Panics recovered at CGo boundary
- Detailed error logging for debugging

**NFR2.2: Resource Management**
- No memory leaks in CGo layer
- Proper cleanup even on error paths
- Context pool prevents resource exhaustion
- File handles closed properly

**NFR2.3: Stability**
- Run continuously for days/weeks without degradation
- Handle edge cases: empty prompts, malformed images, OOM scenarios
- Graceful degradation if GPU becomes unavailable

### NFR3: Maintainability

**NFR3.1: Code Organization**
- llamaruntime/ package follows atomic design
- Clear separation between CGo bindings (atoms) and Go API (molecules)
- Well-documented public API with examples
- Internal complexity hidden from callers

**NFR3.2: Testing**
- Unit tests for Go API functions
- Integration tests for full inference pipeline
- Mock CGo layer for testing without GPU
- Performance benchmarks for tracking regressions

**NFR3.3: Documentation**
- README in llamaruntime/ explaining architecture
- Godoc comments on all exported functions/types
- Build documentation for each platform
- Troubleshooting guide for common issues

### NFR4: Platform Support

**NFR4.1: Windows**
- Windows 10 (1909+) and Windows 11
- NVIDIA RTX 2000 series or newer
- CUDA 12.x support
- Visual Studio 2022 runtime redistributable

**NFR4.2: Linux**
- Ubuntu 20.04+, Fedora 35+, or equivalent
- NVIDIA RTX 2000 series or newer
- CUDA 12.x support
- GCC 11+ runtime libraries

**NFR4.3: GPU Requirements**
- NVIDIA RTX GPU required (no CPU fallback)
- Minimum 6GB VRAM
- CUDA compute capability 7.5+ (Turing architecture or newer)

### NFR5: Security

**NFR5.1: Memory Safety**
- Validate all inputs before passing to C layer
- Bounds checking on buffers
- No buffer overflows in CGo layer
- Sanitize user prompts (length limits, special character handling)

**NFR5.2: Resource Limits**
- Max prompt length: 4096 tokens
- Max output length: Configurable per operation
- Max concurrent requests: Configurable (default 5)
- Inference timeout: Configurable (default 60s)

**NFR5.3: Data Privacy**
- All inference happens locally - no external calls
- No telemetry or usage tracking
- Prompt data never leaves local machine
- GPU memory cleared after inference

## Integration Requirements

### IR1: Replace OpenAI API Calls

**IR1.1: Text Generation**
- Replace `core.TestAIResponse()` to use llamaruntime instead of OpenAI
- Maintain same function signature for compatibility
- Support same prompt patterns and response format

**IR1.2: Note Processing**
- Update `handlers.go` note processing to use llamaruntime
- Maintain `{{ }}` syntax for triggering AI
- Keep same note creation and positioning logic

**IR1.3: PDF Analysis**
- Replace OpenAI PDF summarization with llamaruntime
- Maintain chunking logic for large PDFs
- Keep same summary formatting

**IR1.4: Canvas Analysis**
- Replace OpenAI canvas overview generation with llamaruntime
- Maintain spatial relationship understanding
- Keep same output format

### IR2: Configuration Changes

**IR2.1: Remove OpenAI Config**
- Remove OPENAI_API_KEY requirement from .env
- Remove model selection environment variables (OPENAI_NOTE_MODEL, etc.)
- Remove BASE_LLM_URL, TEXT_LLM_URL (no longer needed)

**IR2.2: Add llama.cpp Config**
- MODEL_PATH: Path to Bunny GGUF model (default: `models/bunny-v1.1-llama-3-8b-v.gguf`)
- GPU_LAYERS: Number of layers to offload to GPU (-1 for auto)
- CONTEXT_SIZE: Context window size (default: 4096)
- BATCH_SIZE: Batch size for inference (default: 512)
- Keep existing token limits (PDF_PRECIS_TOKENS, etc.)

**IR2.3: Minimal Configuration**
- Only Canvus credentials required for operation
- All AI parameters have sensible defaults
- No model selection needed (Bunny is hardcoded)

### IR3: Backward Compatibility

**IR3.1: Phase 1 Compatibility**
- Do not break existing Canvus integration
- Canvas monitoring continues to work unchanged
- Widget CRUD operations unchanged
- PDF upload and download unchanged

**IR3.2: Graceful Migration**
- Provide migration guide from OpenAI to embedded LLM
- Document configuration changes needed
- Provide example .env for Phase 2

## Testing Requirements

### TR1: Unit Tests

- Test model loading with valid and invalid paths
- Test inference with various prompt types
- Test vision inference with different image formats
- Test error handling and recovery
- Test memory cleanup
- Mock CGo layer to test without GPU

### TR2: Integration Tests

- Test full pipeline: load model → inference → cleanup
- Test concurrent requests with context pool
- Test long-running inference (large PDF processing)
- Test GPU memory usage stays within bounds
- Test recovery after simulated GPU error

### TR3: Performance Tests

- Benchmark inference speed on different RTX GPUs
- Measure first token latency
- Measure tokens per second throughput
- Measure memory usage over time (check for leaks)
- Compare performance vs. OpenAI API baseline

### TR4: Platform Tests

- Test Windows build with Visual Studio 2022 + CUDA 12
- Test Linux build with GCC 11 + CUDA 12
- Test on different RTX GPU models (3060, 3070, 4070, 4090)
- Test on different VRAM capacities (6GB, 8GB, 12GB, 24GB)

### TR5: Acceptance Tests

- End-to-end: User adds note with `{{ prompt }}` → AI response appears
- End-to-end: User uploads PDF → analysis completes → summary note appears
- End-to-end: User uploads image → vision analysis completes → description appears
- End-to-end: User requests canvas analysis → overview generates correctly
- Verify no OpenAI API calls made
- Verify all processing happens locally

## Dependencies

### Code Dependencies

- llama.cpp (latest from https://github.com/ggerganov/llama.cpp)
- CMake 3.20+ (build-time)
- CUDA Toolkit 12.x (build-time and runtime)
- C++ compiler: MSVC 2022 (Windows), GCC 11+ (Linux)
- Existing Go dependencies (core, canvusapi, logging packages)

### Model Dependencies

- Bunny v1.1 Llama-3-8B-V GGUF model
- Source: https://huggingface.co/BAAI/Bunny-v1_1-Llama-3-8B-V
- Quantization: Q4_K_M or Q5_K_M recommended
- File size: ~5GB (Q4_K_M), ~6GB (Q5_K_M)
- Download on first run if not bundled with installer

### Hardware Dependencies

- NVIDIA RTX GPU (2000 series or newer)
- Minimum 6GB VRAM
- CUDA compute capability 7.5+
- Modern multi-core CPU (4+ cores recommended)
- 16GB+ system RAM recommended
- SSD storage for model files

## Constraints

- **No cloud dependencies**: All inference must run locally
- **CUDA required**: No CPU fallback, RTX GPU is mandatory
- **Single model**: Bunny v1.1 is hardcoded, no model switching UI
- **GGUF format**: Only GGUF quantized models supported
- **No training**: Inference only, no fine-tuning or model modification
- **English primary**: Bunny model optimized for English, other languages best-effort
- **Platform limited**: Windows and Linux only, macOS not supported (no CUDA on macOS)

## Success Criteria

### SC1: Functional Success

- llama.cpp loads successfully on both Windows and Linux
- Bunny model loads and runs test inference successfully
- Text generation works for notes, PDFs, canvas analysis
- Vision inference works for image analysis
- All OpenAI API calls removed from codebase
- Application runs without OpenAI API key

### SC2: Performance Success

- Inference speed ≥20 tokens/second on RTX 3060
- First token latency ≤500ms for text-only
- Model loading time ≤10 seconds
- Memory usage ≤6GB VRAM for Q4_K_M model
- Concurrent requests (≤5) work without degradation

### SC3: Reliability Success

- Application runs for 24+ hours without crash or degradation
- Automatic recovery works after simulated GPU error
- No memory leaks detected in 8-hour stress test
- Error handling prevents crashes from invalid inputs

### SC4: User Experience Success

- Configuration requires only Canvus credentials (OPENAI_API_KEY removed)
- AI responses arrive in <5 seconds for typical prompts
- Vision analysis completes in <10 seconds for typical images
- Startup completes in <30 seconds with diagnostics
- Error messages clearly explain what went wrong and how to fix

## Out of Scope for Phase 2

- Image generation (stable-diffusion.cpp) - deferred to Phase 3
- Model switching/selection - Bunny is hardcoded
- CPU fallback - CUDA required
- Quantization selection - Q4_K_M/Q5_K_M hardcoded
- Fine-tuning or model customization
- Web UI for monitoring (deferred to Phase 5)
- Multiple model loading (e.g., separate text and vision models)
- Distributed inference (multiple GPUs, multiple machines)

## Risks and Mitigations

### Risk 1: CGo Complexity and Stability

**Risk**: CGo integration is complex and error-prone. Pointer errors, memory leaks, and crashes can be difficult to debug.

**Mitigation**:
- Use proven patterns from go-llama.cpp or similar projects
- Extensive unit tests with memory leak detection
- Mock CGo layer for testing without GPU
- Conservative error handling with panics recovered at boundary
- Detailed logging of all CGo calls during development

### Risk 2: CUDA Build Complexity

**Risk**: Building llama.cpp with CUDA on multiple platforms is complex. CMake configurations, library paths, and runtime dependencies can cause issues.

**Mitigation**:
- Document build process step-by-step for each platform
- Provide build scripts that automate common steps
- Test builds on clean VMs for each platform
- Bundle all runtime dependencies with installer
- Provide troubleshooting guide for common build errors

### Risk 3: GPU Compatibility

**Risk**: Different RTX GPU models and driver versions may behave differently. VRAM constraints vary widely.

**Mitigation**:
- Test on multiple RTX GPU models (3060, 4070, 4090)
- Test on different driver versions
- Implement GPU memory monitoring with warnings
- Provide guidance on VRAM requirements in documentation
- Fail gracefully if VRAM insufficient

### Risk 4: Model Performance

**Risk**: Bunny v1.1 8B model may be slower or lower quality than OpenAI GPT-4 for some tasks.

**Mitigation**:
- Set realistic user expectations: local = private, but different quality
- Optimize inference parameters (batch size, GPU layers, quantization)
- Benchmark against OpenAI and document trade-offs
- Keep temperature/top-p tunable for users who want to experiment
- Consider future upgrade to larger model if user demand exists

### Risk 5: Model Download Size

**Risk**: 5-6GB model download is large. Users may have slow connections or bandwidth caps.

**Mitigation**:
- Provide option to bundle model with installer (larger installer)
- Implement resumable downloads for first-run download
- Show clear progress bar during download
- Provide torrent/mirror options for faster downloads (future)
- Document model size clearly in system requirements

### Risk 6: Integration Breakage

**Risk**: Replacing OpenAI integration may break existing functionality or introduce bugs.

**Mitigation**:
- Phase 2 is a major change - thorough testing required
- Maintain same function signatures initially for minimal disruption
- Comprehensive integration tests covering all AI operations
- Acceptance tests verify end-to-end workflows still work
- Consider feature flag to toggle OpenAI vs. llama.cpp during development

## Timeline Estimate

Based on roadmap sizing:
- Item 8 (llama.cpp CUDA Integration): Large = 6-10 days
- Item 9 (Cross-Platform Build): Large = 6-10 days
- Item 10 (Bunny Integration): Medium = 3-5 days
- Item 11 (Vision Pipeline): Medium = 3-5 days
- Item 12 (Health Monitoring): Small = 1-2 days

**Total Estimate**: 19-32 days for Phase 2

**Critical Path**: Item 8 → Item 9 → Item 10 → Item 11 → Item 12 (sequential dependencies)

## Acceptance Criteria

Phase 2 is complete when:

1. ✓ llamaruntime/ package created with CGo bindings to llama.cpp
2. ✓ llama.cpp builds successfully on Windows (MSVC + CUDA) and Linux (GCC + CUDA)
3. ✓ Bunny v1.1 model loads and runs test inference on startup
4. ✓ Text generation works for notes ({{ prompt }} syntax)
5. ✓ Text generation works for PDF analysis
6. ✓ Text generation works for canvas analysis
7. ✓ Vision inference works for image analysis
8. ✓ All OpenAI API calls removed from codebase
9. ✓ OPENAI_API_KEY no longer required in .env
10. ✓ GPU memory monitoring implemented and logged
11. ✓ Startup diagnostics verify CUDA, GPU, model load successfully
12. ✓ Automatic recovery works after simulated inference failure
13. ✓ Unit tests pass on both platforms
14. ✓ Integration tests pass on both platforms
15. ✓ Performance benchmarks meet targets (20+ tokens/sec on RTX 3060)
16. ✓ Application runs for 24+ hours without crash
17. ✓ Documentation updated (README, build guides, troubleshooting)
18. ✓ Example .env updated with new configuration
