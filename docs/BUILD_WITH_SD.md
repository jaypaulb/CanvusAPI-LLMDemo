# Building CanvusLocalLLM with Stable Diffusion Support

This guide explains how to build CanvusLocalLLM with integrated stable-diffusion.cpp for local image generation using your GPU.

## Overview

CanvusLocalLLM can be built with Stable Diffusion support for offline, privacy-preserving image generation. This requires:

1. Building stable-diffusion.cpp as a C shared library with CUDA
2. Building the Go application with CGo bindings enabled
3. Bundling the shared libraries with the binary

The build scripts handle all of this automatically.

## Prerequisites

### Linux

- **OS**: Ubuntu 20.04+ or similar (kernel with NVIDIA driver support)
- **Compiler**: GCC 9+ or Clang 10+
- **CUDA**: CUDA Toolkit 11.8 or newer
- **CMake**: 3.18 or newer
- **Go**: 1.21 or newer
- **Git**: For cloning dependencies

Install on Ubuntu/Debian:
```bash
# Build tools
sudo apt update
sudo apt install -y build-essential cmake git

# Go (if not installed)
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# CUDA Toolkit (follow NVIDIA instructions)
# https://developer.nvidia.com/cuda-downloads
```

Verify CUDA installation:
```bash
nvcc --version
nvidia-smi
```

### Windows

- **OS**: Windows 10/11 (64-bit)
- **Visual Studio**: 2022 with "Desktop development with C++" workload
- **CUDA**: CUDA Toolkit 11.8 or newer
- **CMake**: 3.18 or newer
- **Go**: 1.21 or newer
- **MinGW-w64**: For CGo compilation
- **Git**: For cloning dependencies

Install tools:

1. **Visual Studio 2022**: Download from [visualstudio.microsoft.com](https://visualstudio.microsoft.com/)
   - During installation, select "Desktop development with C++"

2. **CUDA Toolkit**: Download from [developer.nvidia.com/cuda-downloads](https://developer.nvidia.com/cuda-downloads)
   - Follow the installer prompts
   - Verify: `nvcc --version`

3. **CMake**: Download from [cmake.org/download](https://cmake.org/download/)
   - Use the Windows installer
   - Add to PATH during installation

4. **Go**: Download from [go.dev/dl](https://go.dev/dl/)
   - Use the Windows installer
   - Verify: `go version`

5. **MinGW-w64**: Download from [mingw-w64.org](https://www.mingw-w64.org/)
   - Recommended: [WinLibs standalone build](https://winlibs.com/)
   - Extract to `C:\mingw64` and add `C:\mingw64\bin` to PATH
   - Verify: `gcc --version`

## Building on Linux

### Quick Build

```bash
# Clone the repository (if not already done)
git clone https://github.com/your-org/CanvusLocalLLM.git
cd CanvusLocalLLM

# Run the SD build script
./scripts/build-sd-linux.sh
```

This will:
1. Build stable-diffusion.cpp with CUDA support
2. Build the Go application with SD support
3. Place the binary at `bin/canvuslocallm-sd-linux-amd64`
4. Place libraries at `lib/libstable-diffusion.so`

### Build with Distribution Package

To create a distributable tarball:

```bash
./scripts/build-sd-linux.sh --tarball
```

Output: `dist/canvuslocallm-sd-VERSION-linux-amd64.tar.gz`

The tarball includes:
- Binary (`canvuslocallm`)
- Shared library (`lib/libstable-diffusion.so`)
- Startup script (`start.sh`) with library path configuration
- Documentation (`README.md`, `LICENSE.txt`, `INSTALL.txt`)
- Model directory (`models/`)
- Example configuration (`example.env`)

### Build Options

```bash
# Specify version
./scripts/build-sd-linux.sh --version 1.2.0

# Clean build (removes all artifacts first)
./scripts/build-sd-linux.sh --clean

# Skip SD library build (use existing library)
./scripts/build-sd-linux.sh --skip-sd

# Combine options
./scripts/build-sd-linux.sh --version 1.2.0 --clean --tarball
```

### Running the Build

After building, run with:

```bash
# Set library path
export LD_LIBRARY_PATH=$PWD/lib:$LD_LIBRARY_PATH

# Run the application
./bin/canvuslocallm-sd-linux-amd64
```

Or use the distribution tarball's startup script:

```bash
tar -xzf dist/canvuslocallm-sd-1.0.0-linux-amd64.tar.gz
cd canvuslocallm-sd
./start.sh
```

## Building on Windows

### Quick Build

```powershell
# Clone the repository (if not already done)
git clone https://github.com/your-org/CanvusLocalLLM.git
cd CanvusLocalLLM

# Run the SD build script (PowerShell)
.\scripts\build-sd-windows.ps1
```

This will:
1. Build stable-diffusion.cpp with CUDA support
2. Build the Go application with SD support
3. Copy CUDA runtime DLLs to `lib/`
4. Place the binary at `bin\canvuslocallm-sd-windows-amd64.exe`
5. Place libraries at `lib\stable-diffusion.dll`

### Build with Distribution Package

To create a distributable ZIP:

```powershell
.\scripts\build-sd-windows.ps1 -Zip
```

Output: `dist\canvuslocallm-sd-VERSION-windows-amd64.zip`

The ZIP includes:
- Binary (`canvuslocallm.exe`)
- DLLs (`lib\stable-diffusion.dll`, CUDA runtime DLLs)
- Startup script (`start.bat`) with PATH configuration
- Documentation (`README.md`, `LICENSE.txt`, `INSTALL.txt`)
- Model directory (`models\`)
- Example configuration (`example.env`)

### Build Options

```powershell
# Specify version
.\scripts\build-sd-windows.ps1 -Version 1.2.0

# Clean build (removes all artifacts first)
.\scripts\build-sd-windows.ps1 -Clean

# Skip SD library build (use existing DLL)
.\scripts\build-sd-windows.ps1 -SkipSD

# Combine options
.\scripts\build-sd-windows.ps1 -Version 1.2.0 -Clean -Zip
```

### Running the Build

After building, run with:

```powershell
# Set library path
$env:PATH = "$PWD\lib;$env:PATH"

# Run the application
.\bin\canvuslocallm-sd-windows-amd64.exe
```

Or use the distribution ZIP's startup script:

```powershell
Expand-Archive dist\canvuslocallm-sd-1.0.0-windows-amd64.zip -DestinationPath .
cd canvuslocallm-sd
.\start.bat
```

## Configuration

After building, you need to:

1. **Download the SD model**:
   ```bash
   # Create models directory
   mkdir -p models

   # Download SD v1.5 model (~4GB)
   wget -O models/sd-v1-5.safetensors \
     https://huggingface.co/runwayml/stable-diffusion-v1-5/resolve/main/v1-5-pruned.safetensors
   ```

2. **Configure the application**:
   ```bash
   # Copy example configuration
   cp example.env .env

   # Edit configuration
   nano .env  # or notepad .env on Windows
   ```

3. **Add SD configuration to `.env`**:
   ```bash
   # Stable Diffusion Configuration
   SD_MODEL_PATH=models/sd-v1-5.safetensors
   SD_IMAGE_SIZE=512
   SD_INFERENCE_STEPS=20
   SD_GUIDANCE_SCALE=7.0
   SD_TIMEOUT_SECONDS=60
   SD_MAX_CONCURRENT=2
   ```

## Build Process Details

### What Happens During Build

1. **stable-diffusion.cpp build**:
   - Clones stable-diffusion.cpp repository to `deps/stable-diffusion.cpp/src/`
   - Runs CMake with CUDA enabled
   - Compiles C++ code with NVIDIA GPU support
   - Outputs shared library to `lib/`

2. **Go application build**:
   - Enables CGo (`CGO_ENABLED=1`)
   - Adds build tag `-tags sd` to include SD code
   - Links against stable-diffusion library
   - Embeds version string via `-ldflags`
   - Sets runtime library path (Linux: `-Wl,-rpath,$ORIGIN/../lib`)

3. **Library bundling** (for distribution packages):
   - Copies shared libraries to `lib/` subdirectory
   - Creates startup scripts that configure library path
   - Packages everything into tarball/ZIP

### Build Tags

The Go code uses build tags to conditionally compile SD support:

- **With SD**: `go build -tags sd`
  - Includes `sdruntime/cgo_bindings_sd.go` (real CGo implementation)
  - Excludes `sdruntime/cgo_bindings_stub.go` (stub implementation)

- **Without SD**: `go build` (default)
  - Includes `sdruntime/cgo_bindings_stub.go` (returns "not available" errors)
  - Excludes `sdruntime/cgo_bindings_sd.go`

This allows building without SD dependencies when CUDA is not available.

## Troubleshooting

### Linux Issues

**Problem**: `libstable-diffusion.so: cannot open shared object file`

**Solution**: Set `LD_LIBRARY_PATH`:
```bash
export LD_LIBRARY_PATH=$PWD/lib:$LD_LIBRARY_PATH
```

Or use the `start.sh` script from the distribution tarball.

---

**Problem**: CUDA out of memory during image generation

**Solution**: Reduce concurrent generation limit:
```bash
# In .env
SD_MAX_CONCURRENT=1
```

Or reduce image size:
```bash
SD_IMAGE_SIZE=256
```

---

**Problem**: `nvcc: command not found`

**Solution**: Add CUDA to PATH:
```bash
export PATH=/usr/local/cuda/bin:$PATH
export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH
```

### Windows Issues

**Problem**: `stable-diffusion.dll not found`

**Solution**: Add `lib/` to PATH:
```powershell
$env:PATH = "$PWD\lib;$env:PATH"
```

Or use the `start.bat` script from the distribution ZIP.

---

**Problem**: `VCRUNTIME140.dll not found`

**Solution**: Install Visual C++ Redistributable:
- Download from [aka.ms/vs/17/release/vc_redist.x64.exe](https://aka.ms/vs/17/release/vc_redist.x64.exe)

---

**Problem**: CGo build fails with "gcc: command not found"

**Solution**: Install MinGW-w64 and add to PATH:
```powershell
# Download from https://winlibs.com/
# Extract to C:\mingw64
$env:PATH = "C:\mingw64\bin;$env:PATH"
gcc --version  # Verify installation
```

---

**Problem**: CUDA DLLs not found at runtime

**Solution**: The build script should copy them automatically. If not:
```powershell
# Manually copy from CUDA installation
copy "C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v11.8\bin\cudart64_*.dll" lib\
copy "C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v11.8\bin\cublas64_*.dll" lib\
copy "C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v11.8\bin\cublasLt64_*.dll" lib\
```

## Performance Notes

- **First image generation**: Takes longer (~10-30 seconds) as the model loads into VRAM
- **Subsequent generations**: Faster (~5-15 seconds) as model stays in VRAM
- **VRAM usage**: ~3.5GB for SD v1.5 Q8_0 + 512x512 generation
- **Concurrent generation**: `SD_MAX_CONCURRENT=2` targets 8GB VRAM GPUs

Recommended settings for different GPUs:

| GPU          | VRAM | SD_MAX_CONCURRENT | SD_IMAGE_SIZE |
|--------------|------|-------------------|---------------|
| RTX 3060     | 12GB | 2-3               | 512           |
| RTX 3070     | 8GB  | 2                 | 512           |
| RTX 4090     | 24GB | 4-6               | 512-768       |
| RTX 3050     | 6GB  | 1                 | 512           |

## Development

### Building for Development

For faster iteration during development:

```bash
# Build SD library once
./scripts/build-sd-linux.sh --skip-sd

# Rebuild Go app only (much faster)
export CGO_ENABLED=1
export CGO_LDFLAGS="-L$PWD/lib -lstable-diffusion -Wl,-rpath,\$ORIGIN/../lib"
go build -tags sd -o bin/canvuslocallm .
```

### Testing SD Integration

After building, verify SD works:

```bash
# Set library path
export LD_LIBRARY_PATH=$PWD/lib:$LD_LIBRARY_PATH

# Run with verbose logging
./bin/canvuslocallm-sd-linux-amd64

# In Canvus canvas, create a text widget with:
{{image: a beautiful sunset over mountains}}

# Check logs for SD generation
tail -f app.log | grep -i "stable\|diffusion\|image"
```

## CI/CD Integration

For automated builds, use the scripts in CI pipelines:

```yaml
# GitHub Actions example
name: Build with SD Support

on: [push, pull_request]

jobs:
  build-linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      # Note: GitHub Actions runners don't have CUDA
      # Skip SD build in CI, or use self-hosted GPU runners
      - name: Build (without SD)
        run: go build -o bin/canvuslocallm .
```

For GPU-enabled builds, use self-hosted runners with CUDA installed.

## See Also

- [ADVANCED_CONFIG.md](../ADVANCED_CONFIG.md) - Full configuration reference
- [README.md](../README.md) - General project documentation
- [stable-diffusion.cpp](https://github.com/leejet/stable-diffusion.cpp) - Upstream C library
- [Stable Diffusion Models](https://huggingface.co/runwayml/stable-diffusion-v1-5) - Model downloads
