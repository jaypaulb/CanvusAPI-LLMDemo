# Advanced Configuration Guide

This document describes all available configuration options for CanvusLocalLLM. Most users do not need to change these settings - the defaults are optimized for local GPU inference with automatic model management.

## Table of Contents

- [Cloud API Configuration](#cloud-api-configuration)
- [LLM Endpoint Configuration](#llm-endpoint-configuration)
- [Model Selection](#model-selection)
- [Token Limits](#token-limits)
- [Processing Configuration](#processing-configuration)
- [File Handling](#file-handling)
- [Multi-Canvas Mode](#multi-canvas-mode)
- [Azure OpenAI Integration](#azure-openai-integration)
- [Local Model Management](#local-model-management)

---

## Cloud API Configuration

By default, all AI processing uses local GPU inference. Configure these only if you want cloud API fallback.

### OpenAI API

```env
# OpenAI API key for cloud text/image generation fallback
# Only needed if you want to use OpenAI's cloud services instead of local inference
OPENAI_API_KEY=sk-your-openai-api-key-here
```

### Google Vision API

```env
# Google Vision API key for handwriting recognition (OCR)
# Only needed for the handwriting recognition feature
GOOGLE_VISION_API_KEY=your-google-vision-api-key-here
```

---

## LLM Endpoint Configuration

Override the default local LLM server endpoints if using custom infrastructure.

```env
# Base LLM API URL - Default endpoint for all LLM operations
# Default: http://127.0.0.1:1234/v1 (llamaruntime local server)
# Examples:
#   - OpenAI API: https://api.openai.com/v1
#   - Local llama.cpp server: http://localhost:1234/v1
#   - Ollama: http://localhost:11434/v1
#   - Custom server: http://your-llm-server.local:8080/v1
BASE_LLM_URL=http://127.0.0.1:1234/v1

# Optional: Override URL for text generation only
# If not set, BASE_LLM_URL is used
TEXT_LLM_URL=

# Optional: Override URL for image generation
# Default: empty string (uses local Stable Diffusion)
# Examples:
#   - OpenAI DALL-E: https://api.openai.com/v1
#   - Local SD: http://localhost:7860/v1
#   - Azure OpenAI: https://your-resource.openai.azure.com/
IMAGE_LLM_URL=
```

**Why these defaults?**
- `BASE_LLM_URL` defaults to `127.0.0.1:1234` because that's where llamaruntime local server runs
- `IMAGE_LLM_URL` defaults to empty to trigger local Stable Diffusion generation
- This ensures zero cloud dependencies by default

---

## Model Selection

Specify which OpenAI models to use for different operations (only relevant if using OpenAI cloud API).

```env
# Model for note processing
# Default: "" (local models don't need OpenAI identifiers)
# OpenAI examples: gpt-3.5-turbo, gpt-4, gpt-4-turbo
OPENAI_NOTE_MODEL=

# Model for canvas analysis
# Default: "" (local models don't need OpenAI identifiers)
# OpenAI examples: gpt-4, gpt-4-turbo
OPENAI_CANVAS_MODEL=

# Model for PDF analysis
# Default: "" (local models don't need OpenAI identifiers)
# OpenAI examples: gpt-4, gpt-4-turbo
OPENAI_PDF_MODEL=

# Model for image generation
# Default: "" (uses local Stable Diffusion)
# OpenAI examples: dall-e-3, dall-e-2
IMAGE_GEN_MODEL=
```

**Why empty defaults?**
- Local LLM servers (llama.cpp, Ollama) don't require model identifiers in API calls
- The model is loaded once when the server starts
- Only OpenAI cloud API requires specific model names

---

## Token Limits

Control the maximum length of AI responses for different operations.

```env
# Note response token limit
# Default: 400 (optimized for concise answers)
# Range: 100-4000
OPENAI_NOTE_RESPONSE_TOKENS=400

# PDF summarization token limit
# Default: 1000 (allows detailed summaries)
# Range: 500-8000
OPENAI_PDF_PRECIS_TOKENS=1000

# Canvas analysis token limit
# Default: 600 (balanced overview)
# Range: 300-4000
OPENAI_CANVAS_PRECIS_TOKENS=600

# Image analysis token limit
# Default: 16384 (large context for vision models)
# Range: 1000-32000
OPENAI_IMAGE_ANALYSIS_TOKENS=16384

# Error response token limit
# Default: 200 (brief error messages)
# Range: 50-500
OPENAI_ERROR_RESPONSE_TOKENS=200

# PDF chunking configuration
# Chunk size in tokens for processing large PDFs
# Default: 20000 (balance between context and memory)
OPENAI_PDF_CHUNK_SIZE_TOKENS=20000

# Maximum number of chunks to process
# Default: 10 (prevents excessive processing)
OPENAI_PDF_MAX_CHUNKS_TOKENS=10

# Summary ratio - target summary length as fraction of original
# Default: 0.3 (30% of original length)
# Range: 0.1-0.5
OPENAI_PDF_SUMMARY_RATIO=0.3
```

**Why these defaults?**
- Lower token limits for quick operations (notes, errors) improve response time
- Higher limits for complex tasks (PDF analysis, vision) ensure quality
- Chunking settings balance memory usage and processing thoroughness

---

## Processing Configuration

Configure retry behavior, timeouts, and concurrency limits.

```env
# Maximum retry attempts for failed operations
# Default: 3
# Range: 0-10
MAX_RETRIES=3

# Delay between retries (in seconds)
# Default: 1
# Range: 1-60
RETRY_DELAY=1

# AI operation timeout (in seconds)
# Default: 60
# Range: 10-600
AI_TIMEOUT=60

# Overall processing timeout (in seconds)
# Default: 300 (5 minutes)
# Range: 60-3600
PROCESSING_TIMEOUT=300

# Maximum concurrent AI operations
# Default: 5
# Range: 1-20
# Note: Higher values increase throughput but use more GPU memory
MAX_CONCURRENT=5
```

**Why these defaults?**
- 3 retries with 1s delay handles transient network issues without excessive wait
- 60s AI timeout accommodates slower models while preventing hangs
- 5 concurrent operations balances throughput and GPU memory usage
- 300s processing timeout allows complex multi-step operations to complete

---

## File Handling

Configure file size limits and download directory.

```env
# Maximum file size for uploads/processing (in bytes)
# Default: 52428800 (50MB)
# Examples:
#   - 10MB: 10485760
#   - 50MB: 52428800
#   - 100MB: 104857600
MAX_FILE_SIZE=52428800

# Directory for temporary file downloads
# Default: ./downloads
# Path can be absolute or relative to application directory
DOWNLOADS_DIR=./downloads
```

**Why these defaults?**
- 50MB limit handles most PDFs and images while preventing abuse
- `./downloads` keeps temp files local and easy to find/clean

---

## Multi-Canvas Mode

Monitor multiple canvases simultaneously (advanced use case).

```env
# Single canvas mode (default, backward compatible)
CANVAS_ID=your-canvas-id-here

# OR multi-canvas mode (comma-separated list)
# If set, CANVAS_ID is ignored
# All canvases use the same CANVUS_SERVER and CANVUS_API_KEY
CANVAS_IDS=canvas-id-1,canvas-id-2,canvas-id-3

# Optional: Canvas names (comma-separated, same order as CANVAS_IDS)
# If not provided, names will be fetched from API
CANVAS_NAMES=Production Canvas,Staging Canvas,Dev Canvas
```

**When to use multi-canvas mode:**
- Monitoring multiple workspaces with a single installation
- Centralized AI service for multiple teams
- Development/staging/production environment setup

---

## Azure OpenAI Integration

Use Azure OpenAI services instead of standard OpenAI API.

```env
# Azure OpenAI endpoint
# Example: https://your-resource.openai.azure.com/
AZURE_OPENAI_ENDPOINT=

# Azure deployment name for your model
# Example: gpt-4-deployment, dalle3-deployment
AZURE_OPENAI_DEPLOYMENT=

# Azure API version
# Default: 2024-02-15-preview
# Check Azure docs for latest version
AZURE_OPENAI_API_VERSION=2024-02-15-preview
```

**When Azure is selected:**
- Set `IMAGE_LLM_URL` to your Azure endpoint
- Provide `AZURE_OPENAI_DEPLOYMENT` with your deployment name
- The app will use Azure OpenAI instead of standard OpenAI for image generation

---

## Local Model Management

Configure automatic model downloads and local model paths (Phase 1 feature).

```env
# Path to local GGUF model file for text inference
# If not set and LLAMA_AUTO_DOWNLOAD=true, downloads on first run
# Example: ./models/bunny-v1.1-q4_k_m.gguf
LLAMA_MODEL_PATH=

# URL to download model if not found locally
# Used when LLAMA_AUTO_DOWNLOAD=true
# Example: https://huggingface.co/BAAI/Bunny-v1_1-Llama-3-8B-V-gguf/resolve/main/ggml-model-q4_k_m.gguf
LLAMA_MODEL_URL=

# Directory for storing downloaded models
# Default: ./models
LLAMA_MODELS_DIR=./models

# Enable automatic model download on first run
# Default: false (manual download required)
# WARNING: Model files are large (2-8GB). Ensure sufficient disk space.
LLAMA_AUTO_DOWNLOAD=false
```

**Model download behavior:**
1. Check if `LLAMA_MODEL_PATH` file exists
2. If missing and `LLAMA_AUTO_DOWNLOAD=true`, download from `LLAMA_MODEL_URL`
3. Save to `LLAMA_MODELS_DIR`
4. Verify file integrity before use

**Security note:** Only enable auto-download if you control `LLAMA_MODEL_URL` and trust the source.

---

## Common Configuration Scenarios

### Scenario 1: Fully Local (Zero Cloud Dependencies)

```env
# Minimal .env - just Canvus credentials
CANVUS_SERVER=https://your-canvus.com
CANVAS_ID=your-canvas-id
CANVUS_API_KEY=your-api-key
WEBUI_PWD=your-password

# Everything else uses defaults - no cloud APIs needed!
```

**Result:** All AI processing happens locally on your GPU.

---

### Scenario 2: Local LLM + Cloud Image Generation

```env
# Canvus credentials
CANVUS_SERVER=https://your-canvus.com
CANVAS_ID=your-canvas-id
CANVUS_API_KEY=your-api-key
WEBUI_PWD=your-password

# Add OpenAI for image generation only
OPENAI_API_KEY=sk-your-key-here
IMAGE_LLM_URL=https://api.openai.com/v1
```

**Result:** Text processing uses local GPU, image generation uses DALL-E.

---

### Scenario 3: Custom LLM Server

```env
# Canvus credentials
CANVUS_SERVER=https://your-canvus.com
CANVAS_ID=your-canvas-id
CANVUS_API_KEY=your-api-key
WEBUI_PWD=your-password

# Point to custom Ollama instance
BASE_LLM_URL=http://your-server.local:11434/v1
```

**Result:** Uses your custom LLM infrastructure instead of embedded llamaruntime.

---

### Scenario 4: Development with Self-Signed Certs

```env
# Canvus credentials
CANVUS_SERVER=https://dev-canvus.local
CANVAS_ID=your-canvas-id
CANVUS_API_KEY=your-api-key
WEBUI_PWD=your-password

# Allow self-signed certificates (dev only!)
ALLOW_SELF_SIGNED_CERTS=true
```

**Result:** Works with development Canvus instances using self-signed SSL.

---

## Troubleshooting

**Error: "Missing required environment variables"**
- Ensure `.env` file exists in the application directory
- Verify `CANVUS_SERVER`, `CANVAS_ID`, `CANVUS_API_KEY`, and `WEBUI_PWD` are set
- Check for typos in variable names

**Error: "OpenAI API key required for cloud image generation"**
- This means you're trying to use cloud features without configuring credentials
- Either set `OPENAI_API_KEY` or use local generation (default)
- Check handlers are attempting cloud operations when they should use local

**Performance Issues: Slow AI Responses**
- Reduce `OPENAI_*_TOKENS` values to generate shorter responses
- Lower `MAX_CONCURRENT` to reduce GPU memory pressure
- Check GPU utilization with `nvidia-smi`
- Ensure model is loaded with GPU layers (check llamaruntime config)

**High Memory Usage**
- Reduce `OPENAI_PDF_CHUNK_SIZE_TOKENS` for PDF processing
- Lower `MAX_CONCURRENT` to process fewer operations simultaneously
- Reduce token limits across the board
- Consider using smaller quantized models (Q4 instead of Q8)

**Connection Errors to Canvus**
- Verify `CANVUS_SERVER` URL is correct and accessible
- Check `ALLOW_SELF_SIGNED_CERTS=true` if using self-signed certificates
- Ensure `CANVUS_API_KEY` is valid and not expired
- Check firewall settings and network connectivity

---

## Environment Variable Reference

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `CANVUS_SERVER` | Yes | - | Canvus server URL |
| `CANVAS_ID` | Yes* | - | Canvas UUID to monitor |
| `CANVUS_API_KEY` | Yes** | - | API key for authentication |
| `CANVUS_USERNAME` | Yes** | - | Username for authentication |
| `CANVUS_PASSWORD` | Yes** | - | Password for authentication |
| `WEBUI_PWD` | Yes | - | Web UI password |
| `CANVAS_NAME` | No | "" | Human-readable canvas name |
| `PORT` | No | 3000 | Web UI port |
| `ALLOW_SELF_SIGNED_CERTS` | No | false | Allow self-signed SSL |
| `OPENAI_API_KEY` | No | "" | OpenAI cloud API key |
| `GOOGLE_VISION_API_KEY` | No | "" | Google Vision API key |
| `BASE_LLM_URL` | No | http://127.0.0.1:1234/v1 | Default LLM endpoint |
| `TEXT_LLM_URL` | No | "" | Text generation endpoint |
| `IMAGE_LLM_URL` | No | "" | Image generation endpoint |
| `OPENAI_NOTE_MODEL` | No | "" | Note processing model |
| `OPENAI_CANVAS_MODEL` | No | "" | Canvas analysis model |
| `OPENAI_PDF_MODEL` | No | "" | PDF analysis model |
| `IMAGE_GEN_MODEL` | No | "" | Image generation model |
| `OPENAI_NOTE_RESPONSE_TOKENS` | No | 400 | Note response token limit |
| `OPENAI_PDF_PRECIS_TOKENS` | No | 1000 | PDF summary token limit |
| `OPENAI_CANVAS_PRECIS_TOKENS` | No | 600 | Canvas analysis token limit |
| `OPENAI_IMAGE_ANALYSIS_TOKENS` | No | 16384 | Image analysis token limit |
| `OPENAI_ERROR_RESPONSE_TOKENS` | No | 200 | Error response token limit |
| `OPENAI_PDF_CHUNK_SIZE_TOKENS` | No | 20000 | PDF chunk size |
| `OPENAI_PDF_MAX_CHUNKS_TOKENS` | No | 10 | Max PDF chunks |
| `OPENAI_PDF_SUMMARY_RATIO` | No | 0.3 | Summary target ratio |
| `MAX_RETRIES` | No | 3 | Retry attempts |
| `RETRY_DELAY` | No | 1 | Retry delay (seconds) |
| `AI_TIMEOUT` | No | 60 | AI operation timeout (s) |
| `PROCESSING_TIMEOUT` | No | 300 | Overall timeout (seconds) |
| `MAX_CONCURRENT` | No | 5 | Max concurrent operations |
| `MAX_FILE_SIZE` | No | 52428800 | Max file size (bytes) |
| `DOWNLOADS_DIR` | No | ./downloads | Download directory |
| `CANVAS_IDS` | No* | - | Multi-canvas mode IDs |
| `AZURE_OPENAI_ENDPOINT` | No | "" | Azure OpenAI endpoint |
| `AZURE_OPENAI_DEPLOYMENT` | No | "" | Azure deployment name |
| `AZURE_OPENAI_API_VERSION` | No | 2024-02-15-preview | Azure API version |
| `LLAMA_MODEL_PATH` | No | "" | Local model file path |
| `LLAMA_MODEL_URL` | No | "" | Model download URL |
| `LLAMA_MODELS_DIR` | No | ./models | Model storage directory |
| `LLAMA_AUTO_DOWNLOAD` | No | false | Auto-download models |

*Either `CANVAS_ID` or `CANVAS_IDS` is required
**Either `CANVUS_API_KEY` or both `CANVUS_USERNAME` + `CANVUS_PASSWORD` required

---

## Getting Help

- **Documentation**: See README.md for general setup and usage
- **Issues**: Report bugs at https://github.com/jaypaulb/CanvusAPI-LLMDemo/issues
- **Discussions**: Ask questions in GitHub Discussions
- **Configuration Help**: This file covers all environment variables
