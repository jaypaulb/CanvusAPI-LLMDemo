# CanvusLocalLLM

An intelligent integration between Canvus collaborative workspaces and fully local AI services that enables real-time AI-powered interactions running entirely on your hardware with complete data privacy.

## Features

- **Real-time AI Processing**: Monitors Canvus workspaces and processes content enclosed in double curly braces `{{ }}` using embedded local LLM
- **Fully Local Inference**: All AI processing happens on your NVIDIA RTX GPU - zero cloud dependencies, complete data privacy
- **Multiple AI Capabilities**:
  - Text Analysis and Response
  - PDF Document Summarization
  - Canvas Content Analysis
  - Image Analysis and Description (vision capabilities)
  - Handwriting Recognition (optional Google Vision API integration)
- **Embedded Multimodal Model**: Built-in Bunny v1.1 Llama-3-8B-V model with vision capabilities
- **CUDA GPU Acceleration**: Leverages llama.cpp with CUDA for high-performance inference (20+ tokens/second on RTX 3060)

## System Requirements

### Hardware Requirements

**GPU (Required)**:
- NVIDIA RTX GPU with CUDA Compute Capability 7.5 or higher
- Minimum 8GB VRAM (12GB recommended)
- Examples of compatible GPUs:
  - RTX 3060 (12GB) or better
  - RTX 3070 (8GB) or better
  - RTX 4060 Ti (16GB) or better
  - RTX 4070 (12GB) or better
  - RTX 4080/4090 (16GB+)

**System Memory**:
- Minimum 16GB RAM (32GB recommended)

**Disk Space**:
- ~10GB for application, model, and dependencies
  - Application: ~500MB
  - Bunny v1.1 model (Q4_K_M): ~5GB
  - llama.cpp + CUDA libraries: ~2GB
  - Workspace: ~2GB

### Software Requirements

**Operating System**:
- Windows 10/11 (64-bit) with CUDA support
- Linux (Ubuntu 20.04+, Debian 11+, or equivalent) with CUDA support
- macOS is **not supported** (no CUDA support on macOS)

**NVIDIA Software** (Installed automatically by NVIDIA drivers):
- CUDA Toolkit 12.x (CUDA runtime bundled with application)
- Latest NVIDIA GPU drivers (version 525+ for CUDA 12)

**Other**:
- A Canvus Server instance
- Go 1.21+ (only if building from source)

### Verifying Your GPU

To check if your GPU is compatible:

**Windows**:
```bash
nvidia-smi
```

**Linux**:
```bash
nvidia-smi
```

Look for:
- GPU model name (should be RTX 3060 or better)
- CUDA Version (should be 12.0 or higher)
- Memory (should show total VRAM)

## Prerequisites

- A Canvus Server instance (URL and credentials)
- NVIDIA RTX GPU with CUDA support (see System Requirements above)
- NVIDIA GPU drivers installed (version 525+ recommended)

**Optional**:
- Google Vision API key (only for handwriting recognition feature)

## Setup

### Quick Start

1. **Download the Application**
   - Get the latest release for your platform from [GitHub Releases](https://github.com/jaypaulb/CanvusAPI-LLMDemo/releases/latest)
   - Download `example.env` from the repository

2. **Configure Environment**
   Copy `example.env` to `.env` and configure:
   ```
   # Canvus Server Configuration (Required)
   CANVUS_SERVER=https://your-canvus-server.com
   CANVAS_NAME=YOUR_CANVAS_NAME
   CANVAS_ID=your-canvas-id
   CANVUS_API_KEY=your-canvus-api-key

   # Web UI Password (Required)
   WEBUI_PWD=your-password

   # SSL Configuration (Optional)
   ALLOW_SELF_SIGNED_CERTS=false  # Set to true only for development/testing

   # Optional: Google Vision API for handwriting recognition
   GOOGLE_VISION_API_KEY=your-google-vision-key
   ```

3. **Run the Application**

   **Windows**:
   ```bash
   canvusapi-windows-amd64.exe
   ```

   **Linux**:
   ```bash
   chmod +x canvusapi-linux-amd64
   ./canvusapi-linux-amd64
   ```

4. **Verify GPU Detection**
   On startup, you should see:
   ```
   [llama] Detected GPU: NVIDIA GeForce RTX 4070, VRAM: 12GB, CUDA: 12.3
   [llama] Model loaded: 5.2GB, Q4_K_M quantization
   [llama] Health check passed
   ```

### Building from Source

If you prefer to build from source:

1. **Clone the repository**
   ```bash
   git clone https://github.com/jaypaulb/CanvusLocalLLM.git
   cd CanvusLocalLLM
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Build llama.cpp with CUDA** (required once)
   ```bash
   # Windows (requires MSVC and CUDA Toolkit)
   ./scripts/build-llamacpp-cuda.bat

   # Linux (requires GCC and CUDA Toolkit)
   ./scripts/build-llamacpp-cuda.sh
   ```

4. **Build the application**
   ```bash
   # Current platform
   go build -o CanvusLocalLLM.exe .

   # Cross-platform builds
   GOOS=linux GOARCH=amd64 go build -o canvusapi-linux-amd64 .
   GOOS=windows GOARCH=amd64 go build -o canvusapi-windows-amd64.exe .
   ```

5. **Configure and run** (see Quick Start step 2-4)

## Configuration

### Core Configuration

Minimal required configuration in `.env`:

```
# Canvus Server
CANVUS_SERVER=https://your-canvus-server.com
CANVAS_NAME=YOUR_CANVAS_NAME
CANVAS_ID=your-canvas-id
CANVUS_API_KEY=your-canvus-api-key

# Web UI
WEBUI_PWD=your-password
```

### Advanced LLM Configuration

The application uses sensible defaults for the embedded LLM. You can override these if needed:

```
# Model Configuration
MODEL_PATH=models/bunny-v1.1-llama-3-8b-v.gguf  # Path to GGUF model file

# Context Settings
CONTEXT_SIZE=4096         # Context window in tokens (default: 4096)
BATCH_SIZE=512           # Batch size for prompt processing (default: 512)
GPU_LAYERS=-1            # GPU layer offload: -1=all, 0=CPU only (default: -1)
NUM_CONTEXTS=5           # Context pool size for concurrency (default: 5)

# Generation Parameters
TEMPERATURE=0.7          # Sampling temperature: 0.0-2.0 (default: 0.7)
TOP_P=0.9               # Nucleus sampling: 0.0-1.0 (default: 0.9)
REPEAT_PENALTY=1.1      # Repetition penalty: 1.0-2.0 (default: 1.1)
```

**Configuration Guidelines**:

| Setting | Low VRAM (8GB) | Balanced (12GB) | High VRAM (16GB+) |
|---------|----------------|-----------------|-------------------|
| CONTEXT_SIZE | 2048 | 4096 (default) | 8192 |
| BATCH_SIZE | 256 | 512 (default) | 1024 |
| NUM_CONTEXTS | 3 | 5 (default) | 8 |

**Generation Parameters**:
- **TEMPERATURE**: Lower (0.3-0.5) for factual/analytical tasks, higher (0.8-1.0) for creative tasks
- **TOP_P**: Lower (0.85) for focused responses, higher (0.95) for diverse responses
- **REPEAT_PENALTY**: Higher (1.2-1.5) if model repeats itself, lower (1.0-1.1) for natural flow

### Token Limits

Configure response lengths for different operations:

```
# Token Limits
OPENAI_PDF_PRECIS_TOKENS=4000        # PDF document analysis
OPENAI_CANVAS_PRECIS_TOKENS=4000     # Canvas overview analysis
OPENAI_NOTE_RESPONSE_TOKENS=2000     # Note responses
OPENAI_IMAGE_ANALYSIS_TOKENS=1000    # Image descriptions
OPENAI_ERROR_RESPONSE_TOKENS=500     # Error messages
```

Note: Variable names retain "OPENAI_" prefix for backward compatibility, but they now control the local LLM.

### Processing Configuration

```
# Concurrency and Timeouts
MAX_CONCURRENT=5           # Max concurrent AI operations (should match NUM_CONTEXTS)
PROCESSING_TIMEOUT=300     # Processing timeout in seconds
AI_TIMEOUT=60s            # AI inference timeout

# File Processing
MAX_FILE_SIZE=52428800    # Max file size in bytes (default: 50MB)
DOWNLOADS_DIR=./downloads # Temporary downloads directory

# Retry Behavior
MAX_RETRIES=3             # Max retry attempts
RETRY_DELAY=1s            # Delay between retries
```

### SSL/TLS Configuration

```
# SSL Configuration
ALLOW_SELF_SIGNED_CERTS=false  # Set to true only for development/testing
```

**Security Warning**: Setting `ALLOW_SELF_SIGNED_CERTS=true` disables SSL certificate validation and is not recommended for production use.

## Pre-built Releases

Download pre-built binaries from the [GitHub Releases](https://github.com/jaypaulb/CanvusAPI-LLMDemo/releases/latest) page.

### Available Platforms

#### Windows (amd64)
- **Binary**: [canvusapi-windows-amd64.exe](https://github.com/jaypaulb/CanvusAPI-LLMDemo/releases/latest/download/canvusapi-windows-amd64.exe)
- **example.env**: [Download example.env](https://github.com/jaypaulb/CanvusAPI-LLMDemo/raw/main/example.env)

#### Linux (amd64)
- **Binary**: [canvusapi-linux-amd64](https://github.com/jaypaulb/CanvusAPI-LLMDemo/releases/latest/download/canvusapi-linux-amd64)
- **example.env**: [Download example.env](https://github.com/jaypaulb/CanvusAPI-LLMDemo/raw/main/example.env)

#### Linux (ARM64)
- **Binary**: [canvusapi-linux-arm64](https://github.com/jaypaulb/CanvusAPI-LLMDemo/releases/latest/download/canvusapi-linux-arm64)
- **example.env**: [Download example.env](https://github.com/jaypaulb/CanvusAPI-LLMDemo/raw/main/example.env)
- **Note**: ARM64 builds require CUDA-compatible ARM processors (e.g., NVIDIA Jetson)

### Deployment Steps

1. **Download the binary**: Visit the [GitHub Releases](https://github.com/jaypaulb/CanvusAPI-LLMDemo/releases/latest) page and download the appropriate binary for your platform
2. **Download the `example.env` file**: [Download example.env](https://github.com/jaypaulb/CanvusAPI-LLMDemo/raw/main/example.env)
3. **Place both files in the same directory**
4. **Rename `example.env` to `.env`**
5. **Update the details in the `.env` file** with your configuration (see Configuration section)
6. **If connecting to a server with a self-signed certificate**:
   - Set `ALLOW_SELF_SIGNED_CERTS=true` in your `.env` file
   - Note: This is not recommended for production environments

#### Linux-specific Steps
1. Make the binary executable:
   ```bash
   chmod +x canvusapi-linux-amd64
   # or for ARM64:
   chmod +x canvusapi-linux-arm64
   ```
2. Run the binary:
   ```bash
   ./canvusapi-linux-amd64
   # or for ARM64:
   ./canvusapi-linux-arm64
   ```

#### Windows-specific Steps
1. Run the executable:
   ```bash
   canvusapi-windows-amd64.exe
   ```

## Usage

1. **Basic AI Interaction**:
   - Create a note in your Canvus workspace
   - Type your prompt inside double curly braces: `{{What is the capital of France?}}`
   - The system will process the prompt using the local LLM and create a new note with the AI response

2. **PDF Analysis**:
   - Upload a PDF to your canvas
   - Place the AI_Icon_PDF_Precis on the PDF
   - The system will analyze and summarize the PDF content using local AI

3. **Canvas Analysis**:
   - Place the AI_Icon_Canvus_Precis on your canvas
   - The system will analyze all content and relationships between items
   - Receive an overview and insights about your workspace

4. **Image Analysis**:
   - Place the AI_Icon_Image_Analysis on an image
   - The system will analyze and describe the image using local vision capabilities
   - Supports handwriting recognition if Google Vision API is configured

5. **Custom Menu Integration**:
   The application provides special icons for your Canvus custom menu:
   - `AI_Icon_PDF_Precis`: Creates a PDF analysis trigger
   - `AI_Icon_Canvus_Precis`: Creates a canvas analysis trigger
   - `AI_Icon_Image_Analysis`: Creates an image analysis trigger

   To set up the custom menu:
   1. Navigate to your Canvus custom menu settings
   2. Add the icons from the `icons-for-custom-menu` directory:
      - Use the icons in the root directory for the menu entries
      - Use the icons in the `Content` subdirectory for the content triggers
   3. When users click these icons in the custom menu, they can:
      - Place the PDF analysis trigger on any PDF to generate a summary
      - Place the canvas analysis trigger on the background to analyze the entire workspace
      - Place the image analysis trigger on images for descriptions

   **Important Notes**:
   - The canvas analysis trigger must be placed on the background to work
   - You can temporarily store the triggers on notes until you're ready to use them
   - The icons are scaled to 33% of their original size when placed on the canvas

   Example `menu.yml` configuration:
   ```yaml
   items:
     - tooltip: 'AI PDF Precis Helper'
       icon: 'icons/AI_Icon_PDF_Precis.png'
       actions:
         - name: 'create'
           parameters:
             type: 'image'
             source: 'content/AI_Icon_PDF_Precis.png'
             scale: 0.33

     - tooltip: 'AI Canvus Precis Helper'
       icon: 'icons/AI_Icon_Canvus_Precis.png'
       actions:
         - name: 'create'
           parameters:
             type: 'image'
             source: 'content/AI_Icon_Canvus_Precis.png'
             scale: 0.33
   ```

## Migration from Phase 1 (OpenAI API Version)

If you're upgrading from the OpenAI API-based version to the local LLM version:

### What's Changed

**Removed**:
- OpenAI API dependency and API key requirement
- `OPENAI_API_KEY` environment variable
- `BASE_LLM_URL`, `TEXT_LLM_URL` configuration
- `OPENAI_NOTE_MODEL`, `OPENAI_CANVAS_MODEL`, `OPENAI_PDF_MODEL` model selection
- Cloud API costs and data privacy concerns
- Image generation capability (temporarily removed, will return in Phase 3 with local Stable Diffusion)

**Added**:
- Embedded llama.cpp inference engine with CUDA acceleration
- Bunny v1.1 Llama-3-8B-V multimodal model
- GPU memory monitoring and health checks
- Local vision capabilities for image analysis
- Complete data privacy (no external API calls)
- New configuration options for LLM tuning (see Configuration section)

**System Requirements**:
- Now requires NVIDIA RTX GPU with CUDA support
- Minimum 8GB GPU VRAM (12GB recommended)
- NVIDIA GPU drivers (version 525+ for CUDA 12)

### Migration Steps

1. **Verify GPU Compatibility**
   ```bash
   nvidia-smi
   ```
   Ensure you have an RTX 3060 or better with 8GB+ VRAM.

2. **Update Your `.env` File**

   **Remove these lines** (no longer needed):
   ```
   OPENAI_API_KEY=...
   BASE_LLM_URL=...
   TEXT_LLM_URL=...
   IMAGE_LLM_URL=...
   OPENAI_NOTE_MODEL=...
   OPENAI_CANVAS_MODEL=...
   OPENAI_PDF_MODEL=...
   IMAGE_GEN_MODEL=...
   AZURE_OPENAI_ENDPOINT=...
   AZURE_OPENAI_DEPLOYMENT=...
   AZURE_OPENAI_API_VERSION=...
   ```

   **Keep these lines** (still required):
   ```
   CANVUS_SERVER=...
   CANVAS_NAME=...
   CANVAS_ID=...
   CANVUS_API_KEY=...
   WEBUI_PWD=...
   ```

   **Optionally add** (for advanced tuning):
   ```
   MODEL_PATH=models/bunny-v1.1-llama-3-8b-v.gguf
   CONTEXT_SIZE=4096
   BATCH_SIZE=512
   GPU_LAYERS=-1
   NUM_CONTEXTS=5
   TEMPERATURE=0.7
   TOP_P=0.9
   REPEAT_PENALTY=1.1
   ```

3. **Download the New Release**
   - Get the Phase 2 binary from [GitHub Releases](https://github.com/jaypaulb/CanvusAPI-LLMDemo/releases/latest)
   - The model file will be bundled or downloaded automatically on first run

4. **Test the Migration**
   - Run the application
   - Verify GPU detection in startup logs
   - Test a simple note prompt: `{{Hello, are you working?}}`
   - Verify response is generated locally (no network calls to OpenAI)

5. **Performance Expectations**
   - First response may be slower due to model loading (~30 seconds)
   - Subsequent responses should be fast (20+ tokens/second on RTX 3060)
   - GPU VRAM usage should be ~6GB for Q4_K_M model

### Feature Compatibility

| Feature | Phase 1 (OpenAI) | Phase 2 (Local LLM) |
|---------|------------------|---------------------|
| Note AI Processing | ✅ Cloud | ✅ Local |
| PDF Analysis | ✅ Cloud | ✅ Local |
| Canvas Analysis | ✅ Cloud | ✅ Local |
| Image Analysis | ✅ Cloud | ✅ Local (Vision Model) |
| Handwriting Recognition | ✅ Google Vision | ✅ Google Vision (Optional) |
| Image Generation | ✅ DALL-E | ❌ Removed (Coming in Phase 3) |
| Data Privacy | ❌ Sent to OpenAI | ✅ 100% Local |
| API Costs | ❌ Pay per use | ✅ Free after hardware |
| Internet Required | ✅ Required | ✅ Optional (only for Canvus server) |

### Troubleshooting Migration

**"CUDA GPU not available"**:
- Verify GPU compatibility: `nvidia-smi`
- Update NVIDIA drivers to version 525+
- Ensure CUDA 12 support

**"Model file not found"**:
- Check `MODEL_PATH` in `.env`
- Ensure model file downloaded to `models/` directory
- Model file should be ~5GB

**"Out of memory"** or **VRAM errors**:
- Reduce `CONTEXT_SIZE` (e.g., 4096 → 2048)
- Reduce `NUM_CONTEXTS` (e.g., 5 → 3)
- Close other GPU-using applications

**Slower than expected**:
- Verify GPU is being used: Check logs for "GPU layers: -1" or "GPU layers: 40"
- Check GPU isn't thermal throttling: `nvidia-smi` (look at temperature and power)
- Increase `BATCH_SIZE` for faster prompt processing (if VRAM allows)

**Need image generation**:
- Image generation was temporarily removed in Phase 2
- It will return in Phase 3 with local Stable Diffusion support
- Alternative: Continue using Phase 1 for image generation needs

## Performance

### Expected Performance

Baseline performance with default settings (Q4_K_M quantization, 4096 context):

| GPU Model | VRAM | Tokens/Sec | First Token | 100 Token Response |
|-----------|------|------------|-------------|---------------------|
| RTX 3060  | 12GB | 20-25      | ~500ms      | ~4-5s               |
| RTX 3070  | 8GB  | 30-35      | ~400ms      | ~3s                 |
| RTX 4070  | 12GB | 40-50      | ~300ms      | ~2-2.5s             |
| RTX 4080  | 16GB | 60-70      | ~250ms      | ~1.5-2s             |
| RTX 4090  | 24GB | 80-100     | ~200ms      | ~1-1.2s             |

### Performance Tuning

If performance is below expectations:

1. **Check GPU Utilization**: Verify GPU is being used (look for "GPU layers: -1" in logs)
2. **Increase Batch Size**: Higher batch size = faster prompt processing (more VRAM)
3. **Adjust Context Size**: Smaller context = less VRAM, faster inference
4. **Monitor VRAM**: Use `nvidia-smi dmon` to watch VRAM usage
5. **Check Thermal Throttling**: Ensure GPU isn't overheating

See [llamaruntime documentation](docs/llamaruntime.md) for detailed performance tuning guide.

## Error Handling

- The system includes robust error handling and retry mechanisms
- Processing status is displayed through color-coded notes
- Failed operations are logged with detailed error messages
- SSL/TLS connection errors are clearly reported in logs
- GPU errors and VRAM issues are detected and reported

## Logging

Logs are stored in `app.log` with detailed information about:
- System operations
- API interactions
- Error messages
- Processing status
- SSL/TLS connection status and warnings
- GPU memory usage
- Inference performance metrics
- Model loading and health checks

## Security

- API keys (Canvus) are stored securely in the `.env` file
- All AI processing happens locally on your hardware
- No data sent to external AI services (complete privacy)
- The system supports secure connections to the Canvus server
- Web interface is protected by authentication
- SSL/TLS certificate validation is enabled by default
- Self-signed certificate support is available but not recommended for production
- Warning messages are logged when SSL verification is disabled

### SSL/TLS Configuration

The application supports two SSL/TLS modes:

1. **Secure Mode (Default)**
   - SSL certificate validation is enabled
   - Recommended for production environments
   - Ensures secure communication with the server
   - Validates server certificates against trusted CAs

2. **Development Mode (Self-signed Certificates)**
   - Enabled by setting `ALLOW_SELF_SIGNED_CERTS=true`
   - Disables SSL certificate validation
   - Useful for development/testing environments
   - **Security Risks**:
     - Vulnerable to man-in-the-middle attacks
     - Cannot verify server identity
     - Not recommended for production use
     - Warning messages are logged when enabled

## Troubleshooting

### Common Issues

**GPU Not Detected**:
- Run `nvidia-smi` to verify GPU is recognized
- Update NVIDIA drivers to version 525+
- Ensure GPU has CUDA Compute Capability 7.5+

**Model Loading Fails**:
- Check `MODEL_PATH` points to correct location
- Verify model file is complete (~5GB for Q4_K_M)
- Ensure sufficient disk space

**Out of Memory**:
- Reduce `CONTEXT_SIZE` (4096 → 2048)
- Reduce `NUM_CONTEXTS` (5 → 3)
- Close other GPU applications
- Use smaller quantization model

**Slow Performance**:
- Verify GPU layers offloaded: Check logs for "GPU layers: -1"
- Increase `BATCH_SIZE` if VRAM available
- Check GPU thermal throttling: `nvidia-smi`
- Monitor VRAM usage: `nvidia-smi dmon`

**Inference Timeout**:
- Increase `AI_TIMEOUT` in `.env`
- Reduce `maxTokens` for responses
- Check GPU not throttling

For detailed troubleshooting, see [docs/llamaruntime.md](docs/llamaruntime.md).

## Contributing

Contributions are welcome! Please feel free to submit pull requests or create issues for bugs and feature requests.

## License

This project is proprietary software. All rights reserved.
