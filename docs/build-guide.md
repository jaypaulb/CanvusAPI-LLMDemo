# CanvusLocalLLM Build Guide

This guide provides comprehensive instructions for building CanvusLocalLLM and its AI backend dependencies (llama.cpp and stable-diffusion.cpp) with CUDA acceleration support.

## Table of Contents

- [Prerequisites](#prerequisites)
  - [Common Requirements](#common-requirements)
  - [Windows-Specific Requirements](#windows-specific-requirements)
  - [Linux-Specific Requirements](#linux-specific-requirements)
- [Building the Go Application](#building-the-go-application)
  - [Initial Setup](#initial-setup)
  - [Build Commands](#build-commands)
  - [Cross-Platform Builds](#cross-platform-builds)
- [Building AI Dependencies with CUDA](#building-ai-dependencies-with-cuda)
  - [Building llama.cpp](#building-llamacpp)
  - [Building stable-diffusion.cpp](#building-stable-diffusioncpp)
- [CMake Configuration Details](#cmake-configuration-details)
  - [llama.cpp CMake Options](#llamacpp-cmake-options)
  - [stable-diffusion.cpp CMake Options](#stable-diffusioncpp-cmake-options)
- [Troubleshooting](#troubleshooting)
  - [Common Build Errors](#common-build-errors)
  - [CUDA-Related Issues](#cuda-related-issues)
  - [Go Build Issues](#go-build-issues)
- [Verification](#verification)

---

## Prerequisites

### Common Requirements

These tools are required on all platforms:

1. **Go Programming Environment**
   - Version: 1.24.0 or higher
   - Download: https://go.dev/dl/
   - Verify installation: `go version`

2. **Git**
   - Version: 2.x or higher
   - Download: https://git-scm.com/downloads
   - Verify installation: `git --version`

3. **CMake**
   - Version: 3.14 or higher (3.20+ recommended)
   - Download: https://cmake.org/download/
   - Verify installation: `cmake --version`

4. **CUDA Toolkit** (Optional, for GPU acceleration)
   - Version: 11.0 or higher (12.0+ recommended)
   - Download: https://developer.nvidia.com/cuda-downloads
   - Verify installation: `nvcc --version`
   - Note: Only required if you want GPU acceleration

### Windows-Specific Requirements

1. **Microsoft Visual Studio**
   - Version: 2019 or 2022 (Community Edition is sufficient)
   - Required components:
     - Desktop development with C++
     - Windows 10 SDK (latest version)
     - C++ CMake tools for Windows
   - Download: https://visualstudio.microsoft.com/downloads/

2. **Build Tools Setup**
   - Add CMake to system PATH
   - Add CUDA bin directory to PATH (e.g., `C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v12.0\bin`)
   - Verify MSVC compiler: Open "Developer Command Prompt for VS 2022" and run `cl`

3. **Git for Windows**
   - Download: https://git-scm.com/download/win
   - Use Git Bash for running build scripts

### Linux-Specific Requirements

1. **C++ Compiler**
   - GCC 7+ or Clang 6+
   - Install on Ubuntu/Debian:
     ```bash
     sudo apt update
     sudo apt install build-essential g++ gcc
     ```
   - Install on Fedora/RHEL:
     ```bash
     sudo dnf install gcc gcc-c++ make
     ```
   - Verify installation: `g++ --version`

2. **Development Headers**
   - Install on Ubuntu/Debian:
     ```bash
     sudo apt install libssl-dev pkg-config
     ```
   - Install on Fedora/RHEL:
     ```bash
     sudo dnf install openssl-devel pkgconf-pkg-config
     ```

3. **CUDA Toolkit** (Optional, for GPU acceleration)
   - Install on Ubuntu:
     ```bash
     # Add NVIDIA package repositories
     wget https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/cuda-keyring_1.1-1_all.deb
     sudo dpkg -i cuda-keyring_1.1-1_all.deb
     sudo apt update
     sudo apt install cuda-toolkit-12-0

     # Add CUDA to PATH
     echo 'export PATH=/usr/local/cuda-12.0/bin:$PATH' >> ~/.bashrc
     echo 'export LD_LIBRARY_PATH=/usr/local/cuda-12.0/lib64:$LD_LIBRARY_PATH' >> ~/.bashrc
     source ~/.bashrc
     ```
   - Verify installation: `nvcc --version`

---

## Building the Go Application

### Initial Setup

1. **Clone the Repository**
   ```bash
   git clone https://github.com/jaypaulb/CanvusLocalLLM.git
   cd CanvusLocalLLM
   ```

2. **Install Go Dependencies**
   ```bash
   go mod download
   go mod verify
   ```

3. **Configure Environment**
   - Copy the example environment file:
     ```bash
     cp example.env .env
     ```
   - Edit `.env` and configure your settings (see README.md for details)

### Build Commands

**Build for Current Platform:**
```bash
# Windows
go build -o CanvusAPI-LLM.exe .

# Linux/macOS
go build -o canvusapi-llm .
```

**Build with Optimizations:**
```bash
# Release build (smaller binary, better performance)
go build -ldflags="-s -w" -o CanvusAPI-LLM.exe .
```

**Build with Race Detector (for development/testing):**
```bash
go build -race -o CanvusAPI-LLM-debug.exe .
```

### Cross-Platform Builds

Build binaries for multiple platforms from a single machine:

**For Linux (amd64):**
```bash
GOOS=linux GOARCH=amd64 go build -o canvusapi-linux-amd64 .
```

**For Linux (ARM64):**
```bash
GOOS=linux GOARCH=arm64 go build -o canvusapi-linux-arm64 .
```

**For Windows (amd64):**
```bash
GOOS=windows GOARCH=amd64 go build -o canvusapi-windows-amd64.exe .
```

**For macOS (Intel):**
```bash
GOOS=darwin GOARCH=amd64 go build -o canvusapi-darwin-amd64 .
```

**For macOS (Apple Silicon):**
```bash
GOOS=darwin GOARCH=arm64 go build -o canvusapi-darwin-arm64 .
```

**Build All Platforms:**
```bash
# Linux script
./scripts/build-all.sh

# Or manually:
for platform in linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64; do
  GOOS=${platform%/*} GOARCH=${platform#*/} go build -o canvusapi-${platform%/*}-${platform#*/} .
done
```

---

## Building AI Dependencies with CUDA

### Building llama.cpp

The project includes an automated build script for llama.cpp with CUDA support.

**Quick Build (Linux/macOS):**
```bash
./scripts/build-llamacpp-cuda.sh
```

**Quick Build (Windows - Git Bash):**
```bash
bash scripts/build-llamacpp-cuda.sh
```

**Advanced Build Options:**
```bash
# Clean build (remove existing build directory)
./scripts/build-llamacpp-cuda.sh --clean

# Specify number of parallel jobs
./scripts/build-llamacpp-cuda.sh --jobs 8

# Custom output directory
./scripts/build-llamacpp-cuda.sh --output /path/to/output

# Combine options
./scripts/build-llamacpp-cuda.sh --clean --jobs 8 --output ./build
```

**Manual Build Process:**

If you prefer to build manually or the script fails:

```bash
# 1. Clone repository
mkdir -p deps
git clone https://github.com/ggerganov/llama.cpp.git deps/llama.cpp
cd deps/llama.cpp

# 2. Create build directory
mkdir build
cd build

# 3. Configure with CMake
cmake .. \
  -DCMAKE_BUILD_TYPE=Release \
  -DLLAMA_CUBLAS=ON \
  -DLLAMA_CUDA=ON \
  -DLLAMA_NATIVE=OFF \
  -DLLAMA_BUILD_TESTS=OFF \
  -DLLAMA_BUILD_EXAMPLES=ON \
  -DLLAMA_BUILD_SERVER=ON

# 4. Build (adjust -j value based on CPU cores)
cmake --build . --config Release -j 8

# 5. Verify build
./bin/llama-server --version
```

**Windows-Specific Manual Build:**

Use Visual Studio Developer Command Prompt:

```cmd
REM 1. Clone repository
mkdir deps
git clone https://github.com/ggerganov/llama.cpp.git deps\llama.cpp
cd deps\llama.cpp

REM 2. Create build directory
mkdir build
cd build

REM 3. Configure with CMake
cmake .. ^
  -DCMAKE_BUILD_TYPE=Release ^
  -DLLAMA_CUBLAS=ON ^
  -DLLAMA_CUDA=ON ^
  -DLLAMA_NATIVE=OFF ^
  -DLLAMA_BUILD_TESTS=OFF ^
  -DLLAMA_BUILD_EXAMPLES=ON ^
  -DLLAMA_BUILD_SERVER=ON ^
  -G "Visual Studio 17 2022"

REM 4. Build
cmake --build . --config Release

REM 5. Verify build
bin\Release\llama-server.exe --version
```

### Building stable-diffusion.cpp

The project includes an automated build script for stable-diffusion.cpp with CUDA support.

**Quick Build (Linux/macOS):**
```bash
./scripts/build-sd-cuda.sh
```

**Quick Build (Windows - Git Bash):**
```bash
bash scripts/build-sd-cuda.sh
```

**Advanced Build Options:**
```bash
# Clean build
./scripts/build-sd-cuda.sh --clean

# Specify number of parallel jobs
./scripts/build-sd-cuda.sh --jobs 8

# Custom output directory
./scripts/build-sd-cuda.sh --output /path/to/output

# Combine options
./scripts/build-sd-cuda.sh --clean --jobs 8 --output ./build
```

**Manual Build Process:**

```bash
# 1. Clone repository with submodules
mkdir -p deps
git clone --recursive https://github.com/leejet/stable-diffusion.cpp.git deps/stable-diffusion.cpp
cd deps/stable-diffusion.cpp

# 2. Create build directory
mkdir build
cd build

# 3. Configure with CMake
cmake .. \
  -DCMAKE_BUILD_TYPE=Release \
  -DSD_CUBLAS=ON \
  -DBUILD_SHARED_LIBS=OFF

# 4. Build
cmake --build . --config Release -j 8

# 5. Verify build
./bin/sd --help
```

**Windows-Specific Manual Build:**

```cmd
REM 1. Clone repository with submodules
mkdir deps
git clone --recursive https://github.com/leejet/stable-diffusion.cpp.git deps\stable-diffusion.cpp
cd deps\stable-diffusion.cpp

REM 2. Create build directory
mkdir build
cd build

REM 3. Configure with CMake
cmake .. ^
  -DCMAKE_BUILD_TYPE=Release ^
  -DSD_CUBLAS=ON ^
  -DBUILD_SHARED_LIBS=OFF ^
  -G "Visual Studio 17 2022"

REM 4. Build
cmake --build . --config Release

REM 5. Verify build
bin\Release\sd.exe --help
```

---

## CMake Configuration Details

### llama.cpp CMake Options

Key CMake flags for llama.cpp:

| Flag | Default | Description |
|------|---------|-------------|
| `LLAMA_CUBLAS` | OFF | Enable CUDA support (legacy flag) |
| `LLAMA_CUDA` | OFF | Enable CUDA support (modern flag) |
| `LLAMA_NATIVE` | OFF | Enable native CPU optimizations (use OFF for portability) |
| `LLAMA_BUILD_TESTS` | ON | Build test executables |
| `LLAMA_BUILD_EXAMPLES` | ON | Build example programs |
| `LLAMA_BUILD_SERVER` | ON | Build llama-server |
| `LLAMA_METAL` | OFF | Enable Metal support (macOS GPU) |
| `LLAMA_BLAS` | OFF | Enable BLAS support |
| `LLAMA_OPENBLAS` | OFF | Enable OpenBLAS support |
| `CMAKE_BUILD_TYPE` | - | Set to Release for optimized builds |

**Recommended Configuration for CUDA:**
```bash
cmake .. \
  -DCMAKE_BUILD_TYPE=Release \
  -DLLAMA_CUBLAS=ON \
  -DLLAMA_CUDA=ON \
  -DLLAMA_NATIVE=OFF \
  -DLLAMA_BUILD_TESTS=OFF \
  -DLLAMA_BUILD_EXAMPLES=ON \
  -DLLAMA_BUILD_SERVER=ON
```

**CPU-Only Build:**
```bash
cmake .. \
  -DCMAKE_BUILD_TYPE=Release \
  -DLLAMA_BUILD_SERVER=ON
```

**macOS with Metal:**
```bash
cmake .. \
  -DCMAKE_BUILD_TYPE=Release \
  -DLLAMA_METAL=ON \
  -DLLAMA_BUILD_SERVER=ON
```

### stable-diffusion.cpp CMake Options

Key CMake flags for stable-diffusion.cpp:

| Flag | Default | Description |
|------|---------|-------------|
| `SD_CUBLAS` | OFF | Enable CUDA support |
| `SD_METAL` | OFF | Enable Metal support (macOS GPU) |
| `BUILD_SHARED_LIBS` | ON | Build shared libraries instead of static |
| `CMAKE_BUILD_TYPE` | - | Set to Release for optimized builds |

**Recommended Configuration for CUDA:**
```bash
cmake .. \
  -DCMAKE_BUILD_TYPE=Release \
  -DSD_CUBLAS=ON \
  -DBUILD_SHARED_LIBS=OFF
```

**CPU-Only Build:**
```bash
cmake .. \
  -DCMAKE_BUILD_TYPE=Release \
  -DBUILD_SHARED_LIBS=OFF
```

**macOS with Metal:**
```bash
cmake .. \
  -DCMAKE_BUILD_TYPE=Release \
  -DSD_METAL=ON \
  -DBUILD_SHARED_LIBS=OFF
```

---

## Troubleshooting

### Common Build Errors

#### Error: "CMake not found"

**Problem:** CMake is not installed or not in PATH.

**Solution:**
- **Linux:** Install via package manager: `sudo apt install cmake`
- **Windows:** Add CMake bin directory to PATH environment variable
- **macOS:** Install via Homebrew: `brew install cmake`

Verify: `cmake --version`

---

#### Error: "CUDA compiler not found"

**Problem:** CUDA Toolkit is not installed or nvcc is not in PATH.

**Solution:**
- Install CUDA Toolkit from https://developer.nvidia.com/cuda-downloads
- Add CUDA bin directory to PATH:
  - **Linux:** `export PATH=/usr/local/cuda/bin:$PATH`
  - **Windows:** Add `C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\vX.X\bin` to PATH
  - **macOS:** CUDA is not supported on recent macOS versions; use Metal instead

Verify: `nvcc --version`

---

#### Error: "No CMAKE_CUDA_COMPILER could be found"

**Problem:** CMake cannot find the CUDA compiler.

**Solution:**
```bash
# Set CUDA compiler explicitly
export CUDACXX=/usr/local/cuda/bin/nvcc

# Or specify in CMake command
cmake .. -DCMAKE_CUDA_COMPILER=/usr/local/cuda/bin/nvcc
```

---

#### Error: "fatal error: cuda_runtime.h: No such file or directory"

**Problem:** CUDA headers not found by compiler.

**Solution:**
- Ensure CUDA Toolkit is properly installed
- Add CUDA include directory to compiler search path:
  ```bash
  export CPLUS_INCLUDE_PATH=/usr/local/cuda/include:$CPLUS_INCLUDE_PATH
  ```
- On Windows, ensure you're using the correct Visual Studio version that matches your CUDA Toolkit

---

#### Error: "undefined reference to `cublasCreate_v2'"

**Problem:** CUDA libraries not linked properly.

**Solution:**
- Ensure CUDA lib directory is in library search path:
  ```bash
  export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH
  ```
- On Windows, add CUDA lib directory to PATH:
  ```cmd
  set PATH=C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v12.0\lib\x64;%PATH%
  ```

---

#### Error: "git submodule update failed"

**Problem:** Submodules failed to initialize (affects stable-diffusion.cpp).

**Solution:**
```bash
# Reinitialize submodules
git submodule update --init --recursive --force

# Or clone with --recursive flag
git clone --recursive https://github.com/leejet/stable-diffusion.cpp.git
```

---

### CUDA-Related Issues

#### Error: "CUDA architecture not supported"

**Problem:** Your GPU architecture is not supported by the CUDA version.

**Solution:**
- Check your GPU compute capability: https://developer.nvidia.com/cuda-gpus
- Update CUDA Toolkit to a version that supports your GPU
- Or specify compute capability manually:
  ```bash
  cmake .. -DCMAKE_CUDA_ARCHITECTURES="75;80;86"
  ```

Common compute capabilities:
- 75: RTX 2000 series, GTX 1600 series
- 80: A100
- 86: RTX 3000 series
- 89: RTX 4000 series

---

#### Error: "out of memory" during build

**Problem:** Insufficient RAM during compilation (common with CUDA builds).

**Solution:**
- Reduce parallel jobs: `cmake --build . -j 2` (instead of -j 8)
- Increase system swap space
- Close other applications
- Build without parallelism: `cmake --build .`

---

#### GPU not detected at runtime

**Problem:** Built successfully but GPU is not used.

**Diagnostic:**
```bash
# Check NVIDIA driver
nvidia-smi

# Check if CUDA libraries are found
ldd ./bin/llama-server | grep cuda
```

**Solution:**
- Update NVIDIA drivers
- Ensure CUDA runtime libraries are in LD_LIBRARY_PATH (Linux) or PATH (Windows)
- Verify GPU compute capability is supported by your build

---

### Go Build Issues

#### Error: "go.mod file not found"

**Problem:** Not in the correct directory.

**Solution:**
```bash
cd /path/to/CanvusLocalLLM
go build -o CanvusAPI-LLM.exe .
```

---

#### Error: "package X is not in GOROOT"

**Problem:** Go dependencies not downloaded.

**Solution:**
```bash
go mod download
go mod verify
go build -o CanvusAPI-LLM.exe .
```

---

#### Error: "cannot find package" during cross-compilation

**Problem:** Some packages have platform-specific dependencies.

**Solution:**
- Ensure CGO is disabled for pure Go cross-compilation:
  ```bash
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o canvusapi-linux-amd64 .
  ```
- Or install cross-compilation toolchain for CGO:
  ```bash
  # For Linux target on Windows, install MinGW-w64
  # For macOS target on Linux, install osxcross
  ```

---

#### Error: Module dependencies out of sync

**Problem:** go.mod or go.sum files are inconsistent.

**Solution:**
```bash
# Resync dependencies
go mod tidy
go mod verify

# Clean module cache if issues persist
go clean -modcache
go mod download
```

---

## Verification

### Verify Go Application Build

```bash
# Check binary was created
ls -lh CanvusAPI-LLM.exe  # or canvusapi-llm on Linux/macOS

# Run version or help command (if implemented)
./CanvusAPI-LLM.exe --version

# Test basic startup (should fail without .env, which is expected)
./CanvusAPI-LLM.exe
```

### Verify llama.cpp Build

```bash
# Check build artifacts
ls -lh deps/llama.cpp/build/bin/

# Run llama-server
./deps/llama.cpp/build/bin/llama-server --version

# Test CUDA detection (if CUDA was enabled)
./deps/llama.cpp/build/bin/llama-server --help | grep -i cuda

# Quick inference test (requires a model)
./deps/llama.cpp/build/bin/llama-server \
  -m /path/to/model.gguf \
  --port 8080 \
  --n-gpu-layers 35
```

Expected output should show CUDA is available if GPU build succeeded.

### Verify stable-diffusion.cpp Build

```bash
# Check build artifacts
ls -lh deps/stable-diffusion.cpp/build/bin/

# Run sd binary
./deps/stable-diffusion.cpp/build/bin/sd --help

# Quick generation test (requires a model)
./deps/stable-diffusion.cpp/build/bin/sd \
  -m /path/to/model.safetensors \
  -p "a beautiful sunset" \
  -o test.png \
  --steps 20
```

Expected output should show CUDA is available if GPU build succeeded.

### Verify CUDA Setup (if applicable)

```bash
# Check NVIDIA driver
nvidia-smi

# Check CUDA version
nvcc --version

# Check CUDA runtime libraries
# Linux:
ldconfig -p | grep cuda

# Windows (PowerShell):
Get-ChildItem -Path "C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA" -Recurse -Filter "*.dll" | Select-Object Name
```

### Full Integration Test

1. **Start llama-server with a model:**
   ```bash
   ./deps/llama.cpp/build/bin/llama-server \
     -m /path/to/text-model.gguf \
     --port 1234 \
     --n-gpu-layers 35
   ```

2. **Configure CanvusLocalLLM:**
   - Edit `.env` and set:
     ```
     BASE_LLM_URL=http://127.0.0.1:1234/v1
     TEXT_LLM_URL=http://127.0.0.1:1234/v1
     ```

3. **Start CanvusLocalLLM:**
   ```bash
   ./CanvusAPI-LLM.exe
   ```

4. **Test AI interaction:**
   - Create a note in your Canvus workspace
   - Type: `{{What is 2+2?}}`
   - Verify AI responds with a new note

---

## Next Steps

After successful build and verification:

1. **Review Configuration:** See [README.md](../README.md) for environment variable details
2. **Download Models:**
   - Text models: https://huggingface.co/models?pipeline_tag=text-generation&library=gguf
   - Image models: https://huggingface.co/models?pipeline_tag=text-to-image
3. **Run Services:** Start llama-server and/or stable-diffusion server
4. **Start Application:** Run CanvusAPI-LLM with proper configuration
5. **Test Features:** Try text generation, PDF analysis, image generation

For detailed usage instructions, see [README.md](../README.md).

---

## Additional Resources

- **llama.cpp Documentation:** https://github.com/ggerganov/llama.cpp
- **stable-diffusion.cpp Documentation:** https://github.com/leejet/stable-diffusion.cpp
- **CUDA Toolkit Documentation:** https://docs.nvidia.com/cuda/
- **Go Documentation:** https://go.dev/doc/
- **CMake Documentation:** https://cmake.org/documentation/

For issues not covered here, please check the project's GitHub Issues page or create a new issue.
