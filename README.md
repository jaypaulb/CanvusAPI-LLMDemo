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

### Quick Start (Zero Configuration)

**Get up and running in under 5 minutes with only 4 configuration values!**

1. **Download the Application**
   - Get the latest release for your platform from [GitHub Releases](https://github.com/jaypaulb/CanvusAPI-LLMDemo/releases/latest)
   - Download `.env.example` from the repository

2. **Configure Canvus Credentials** (Only 4 Required Values)

   Copy `.env.example` to `.env` and configure only these essentials:
   ```env
   # Canvus Connection (Required)
   CANVUS_SERVER=https://your-canvus-server.com
   CANVAS_ID=your-canvas-id

   # Canvus Authentication (Required)
   CANVUS_API_KEY=your-canvus-api-key

   # Web UI Security (Required)
   WEBUI_PWD=your-secure-password
   ```

   **That's it!** All AI settings use intelligent defaults for local GPU inference.

   **Optional Settings** (already configured with sensible defaults):
   - Model selection, token limits, and inference parameters are pre-configured
   - Cloud API fallback (OpenAI, Azure) is available but disabled by default
   - For advanced configuration, see [ADVANCED_CONFIG.md](ADVANCED_CONFIG.md)

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

5. **Start Using AI Features**
   - Models download automatically on first run if needed
   - All AI processing runs locally on your GPU
   - Zero cloud dependencies by default

### Zero-Config Local AI Philosophy

CanvusLocalLLM is designed for **local-first AI** with **zero configuration complexity**:

- **No model selection required**: Embedded Bunny v1.1 multimodal model is pre-configured
- **No endpoint configuration**: Local llamaruntime server starts automatically
- **No token limit tuning**: Sensible defaults optimized for GPU inference
- **No cloud API keys**: All processing is local by default
- **Automatic model downloads**: First-run setup handles model acquisition (Phase 1)

**Cloud APIs are opt-in, not required:**
- Set `OPENAI_API_KEY` only if you want cloud fallback for specific features
- Set `GOOGLE_VISION_API_KEY` only if you need handwriting recognition (OCR)
- See [ADVANCED_CONFIG.md](ADVANCED_CONFIG.md) for all optional configuration

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


   **Optional: Build with Stable Diffusion Support**
   
   To enable local image generation capabilities, use the dedicated SD build scripts:
   ```bash
   # Linux
   ./scripts/build-sd-linux.sh --tarball
   
   # Windows (PowerShell)
   .\scripts\build-sd-windows.ps1 -Zip
   ```
   
   See [docs/BUILD_WITH_SD.md](docs/BUILD_WITH_SD.md) for complete instructions.
5. **Configure and run** (see Quick Start step 2-4)

## Configuration

### Minimal Configuration (Recommended)

For zero-config local AI deployment, configure only these 4 required variables in `.env`:

```env
# Canvus Server
CANVUS_SERVER=https://your-canvus-server.com
CANVAS_ID=your-canvas-id
CANVUS_API_KEY=your-canvus-api-key

# Web UI
WEBUI_PWD=your-password
```

**All other settings have sensible defaults:**
- **LLM Endpoint**: `http://127.0.0.1:1234/v1` (local llamaruntime server)
- **Token Limits**: 400 (notes), 1000 (PDFs), 600 (canvas), 16384 (vision)
- **Processing**: 3 retries, 60s AI timeout, 5 concurrent operations
- **Model Selection**: Embedded Bunny v1.1 multimodal model
- **Cloud APIs**: Disabled (local-first by default)

### Optional Configuration

**Development with Self-Signed Certificates**:
```env
ALLOW_SELF_SIGNED_CERTS=true  # Only for development/testing
```

**Cloud API Fallback** (advanced users):
```env
OPENAI_API_KEY=sk-...          # For cloud text/image generation
GOOGLE_VISION_API_KEY=...      # For handwriting recognition
```

**Custom Port**:
```env
PORT=3000  # Default web UI port
```

### Advanced Configuration

For power users who want fine-grained control, see [ADVANCED_CONFIG.md](ADVANCED_CONFIG.md) for:
- Custom LLM endpoint configuration
- Token limit tuning for different operations
- Processing configuration (retries, timeouts, concurrency)
- Azure OpenAI integration
- Local model management
- Multi-canvas monitoring
- All 40+ available environment variables

**Note**: The minimal configuration template in `.env.example` shows only required settings. Advanced users can override any default by setting the corresponding environment variable.

## Pre-built Releases

Download pre-built binaries from the [GitHub Releases](https://github.com/jaypaulb/CanvusAPI-LLMDemo/releases/latest) page.

### Available Platforms

#### Windows (amd64)
- **Binary**: [canvusapi-windows-amd64.exe](https://github.com/jaypaulb/CanvusAPI-LLMDemo/releases/latest/download/canvusapi-windows-amd64.exe)
- **.env.example**: [Download .env.example](https://github.com/jaypaulb/CanvusAPI-LLMDemo/raw/main/.env.example)

#### Linux (amd64)
- **Binary**: [canvusapi-linux-amd64](https://github.com/jaypaulb/CanvusAPI-LLMDemo/releases/latest/download/canvusapi-linux-amd64)
- **.env.example**: [Download .env.example](https://github.com/jaypaulb/CanvusAPI-LLMDemo/raw/main/.env.example)

#### Linux (ARM64)
- **Binary**: [canvusapi-linux-arm64](https://github.com/jaypaulb/CanvusAPI-LLMDemo/releases/latest/download/canvusapi-linux-arm64)
- **.env.example**: [Download .env.example](https://github.com/jaypaulb/CanvusAPI-LLMDemo/raw/main/.env.example)
- **Note**: ARM64 builds require CUDA-compatible ARM processors (e.g., NVIDIA Jetson)

### Deployment Steps

1. **Download the binary**: Visit the [GitHub Releases](https://github.com/jaypaulb/CanvusAPI-LLMDemo/releases/latest) page and download the appropriate binary for your platform
2. **Download the `.env.example` file**: [Download .env.example](https://github.com/jaypaulb/CanvusAPI-LLMDemo/raw/main/.env.example)
3. **Place both files in the same directory**
4. **Rename `.env.example` to `.env`**
5. **Update the 4 required values in the `.env` file** (see Minimal Configuration above)
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
   - Add a note with prompt: `{{Summarize this PDF}}`
   - The system will extract text, chunk it intelligently, and generate a comprehensive summary

3. **Canvas Analysis**:
   - Add a note with prompt: `{{Analyze this canvas}}`
   - The system will collect all widgets, analyze relationships, and provide insights

4. **Image Analysis**:
   - Upload an image to your canvas
   - Add a note with prompt: `{{Describe this image}}`
   - The multimodal model will analyze the image and provide a detailed description

5. **Handwriting Recognition** (requires Google Vision API key):
   - Upload an image of handwritten text
   - Add a note with prompt: `{{Extract text from this image}}`
   - The OCR system will convert handwriting to editable text

## Troubleshooting

### Configuration Errors

**"Missing required environment variables"**
- Ensure `.env` file exists in the application directory
- Verify `CANVUS_SERVER`, `CANVAS_ID`, `CANVUS_API_KEY`, and `WEBUI_PWD` are set
- Check for typos in variable names
- See `.env.example` for the correct format

**"OpenAI API key required for cloud image generation"**
- This error only occurs if you're trying to use cloud features without configuring them
- Either set `OPENAI_API_KEY` in your `.env` file
- Or use local generation (default) - this error indicates a bug if you haven't requested cloud features

**"Invalid Canvus server URL"**
- Check that `CANVUS_SERVER` starts with `https://` or `http://`
- Remove any trailing slashes from the URL
- Example: `https://canvus.example.com` (not `https://canvus.example.com/`)

### GPU Issues

**GPU not detected**
- Run `nvidia-smi` to verify your GPU and drivers are working
- Check that CUDA version is 12.0 or higher
- Ensure NVIDIA drivers are up to date (version 525+)
- Restart the application after driver updates

**Out of memory errors**
- Reduce concurrent operations: set `MAX_CONCURRENT=3` in advanced config
- Lower token limits in [ADVANCED_CONFIG.md](ADVANCED_CONFIG.md)
- Use a smaller model quantization (Q4_K_M is recommended balance)
- Close other GPU-intensive applications

### Performance Issues

**Slow AI responses**
- Check GPU utilization with `nvidia-smi` - should show near 100% during inference
- Verify all GPU layers are loaded (check startup logs)
- Reduce token limits for faster (but shorter) responses
- See [ADVANCED_CONFIG.md](ADVANCED_CONFIG.md) for performance tuning

**High memory usage**
- Reduce `MAX_CONCURRENT` to process fewer operations simultaneously
- Lower token limits across the board
- Check for memory leaks (restart application periodically if needed)

### Connection Errors

**Cannot connect to Canvus server**
- Verify `CANVUS_SERVER` URL is correct and accessible
- Set `ALLOW_SELF_SIGNED_CERTS=true` if using self-signed certificates (development only)
- Check firewall settings and network connectivity
- Ensure `CANVUS_API_KEY` is valid and not expired

**First-run model download fails** (Phase 1 feature)
- Check internet connectivity
- Verify disk space (models are 2-8GB)
- Manual download option available in error message
- See [ADVANCED_CONFIG.md](ADVANCED_CONFIG.md) for manual model setup

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes with clear commit messages
4. Add tests for new functionality
5. Submit a pull request

## License

[Your License Here]

## Acknowledgments

- Built with [llama.cpp](https://github.com/ggerganov/llama.cpp) for CUDA-accelerated inference
- Uses [Bunny v1.1](https://huggingface.co/BAAI/Bunny-v1_1-Llama-3-8B-V) multimodal model
- Integrates with [Canvus](https://canvus.ai/) collaborative workspace platform
