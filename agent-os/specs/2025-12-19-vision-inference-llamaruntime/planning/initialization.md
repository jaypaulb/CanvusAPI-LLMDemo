# Vision Inference in llamaruntime - Initial Idea

**Issue**: CanvusLocalLLM-x5pq
**Priority**: P2
**Date**: 2025-12-19

## Feature Description

Implement vision inference capabilities in the llamaruntime package to enable local image understanding using llama.cpp multimodal support with Bunny v1.1 model.

## Strategic Importance

**HIGH** - Critical for local-first vision capabilities. This feature replaces the cloud-based Google Vision API (ocrprocessor/) with offline image understanding, enabling:
- Zero internet dependency for image analysis
- No API costs for vision operations
- Complete privacy (images never leave local machine)
- Alignment with zero-config end-user deployment strategy

## Current State

- **TODO exists at**: `llamaruntime/bindings.go:762` - vision inference function stubbed but not implemented
- **Existing infrastructure**: llamaruntime package (10,373 lines) with working llama.cpp CGo bindings for text generation
- **Current vision solution**: Google Vision API in ocrprocessor/ (2,350 lines, 94.5% coverage) - cloud-based, requires internet and API key
- **Model in use**: Bunny v1.1 multimodal model already used for text generation

## What Needs to Be Built

1. **Vision inference function** in `llamaruntime/bindings.go` (currently stubbed at line 762)
2. **Image preprocessing utilities** (may already exist in vision/ package)
3. **Integration with canvas monitoring workflow** in handlers.go
4. **Testing infrastructure** for vision inference with Bunny multimodal models

## Dependencies/Blockers

- Requires llama.cpp vision API to be stable and available
- May need specific Bunny model format (GGUF with vision support)
- Understanding of llama.cpp vision API patterns and requirements

## Success Criteria

- Vision inference working locally via llamaruntime
- Able to replace Google Vision API for handwriting recognition and image understanding
- Performance acceptable for real-time canvas monitoring
- Tests passing with multimodal Bunny models
