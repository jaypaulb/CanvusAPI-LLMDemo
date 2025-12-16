# stable-diffusion.cpp Integration

This directory contains the build configuration for integrating [stable-diffusion.cpp](https://github.com/leejet/stable-diffusion.cpp) into CanvusLocalLLM.

## Overview

stable-diffusion.cpp is a C++ implementation of Stable Diffusion inference, optimized for:
- Minimal dependencies (same philosophy as llama.cpp)
- CUDA acceleration for NVIDIA GPUs
- Cross-platform support (Windows, Linux, macOS)

## Directory Structure

```
deps/stable-diffusion.cpp/
├── CMakeLists.txt       # CMake configuration for CUDA build
├── build-windows.ps1    # Windows build script (PowerShell)
├── build-linux.sh       # Linux build script (Bash)
├── README.md            # This file
├── src/                 # stable-diffusion.cpp source (cloned)
├── build/               # CMake build output (generated)
└── include/             # Header files for CGo bindings
```

## Prerequisites

### Windows

1. **Visual Studio 2022** with C++ desktop development workload
2. **CUDA Toolkit 11.8+**: https://developer.nvidia.com/cuda-downloads
3. **CMake 3.18+**: https://cmake.org/download/
4. **Git**: https://git-scm.com/downloads

### Linux

1. **GCC 9+** or **Clang 10+**
2. **CUDA Toolkit 11.8+**: https://developer.nvidia.com/cuda-downloads
3. **CMake 3.18+**: `sudo apt install cmake`
4. **Git**: `sudo apt install git`

## Build Instructions

### Windows (PowerShell)

```powershell
# Navigate to this directory
cd deps\stable-diffusion.cpp

# Run the build script (clones source and builds)
.\build-windows.ps1

# Or, if source already exists:
.\build-windows.ps1 -SkipClone

# Clean build:
.\build-windows.ps1 -Clean
```

**Output**: `lib/stable-diffusion.dll`

### Linux (Bash)

```bash
# Navigate to this directory
cd deps/stable-diffusion.cpp

# Run the build script (clones source and builds)
./build-linux.sh

# Or, if source already exists:
./build-linux.sh --skip-clone

# Clean build:
./build-linux.sh --clean
```

**Output**: `lib/libstable-diffusion.so`

### Manual Build (Advanced)

If you prefer to build manually:

```bash
# Clone source
git clone --depth 1 https://github.com/leejet/stable-diffusion.cpp.git src

# Create build directory
mkdir build && cd build

# Configure (Linux)
cmake .. \
    -DGGML_CUDA=ON \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_CUDA_ARCHITECTURES="75;86;89" \
    -DBUILD_SHARED_LIBS=ON

# Build
cmake --build . -j$(nproc)
```

## CUDA Architectures

The build targets these NVIDIA GPU architectures:

| Architecture | GPU Generation | Examples |
|--------------|----------------|----------|
| 75 | Turing | RTX 2060, 2070, 2080 |
| 86 | Ampere | RTX 3060, 3070, 3080, 3090 |
| 89 | Ada Lovelace | RTX 4060, 4070, 4080, 4090 |

To target different architectures, modify `CMAKE_CUDA_ARCHITECTURES` in CMakeLists.txt or pass it to cmake:

```bash
cmake .. -DCMAKE_CUDA_ARCHITECTURES="80;86"  # A100 + RTX 30xx
```

## Integration with Go

After building, the shared library is placed in the project's `lib/` directory. The CGo bindings in `sdruntime/` package reference this library:

```go
// sdruntime/cgo_bindings_sd.go
#cgo CFLAGS: -I${SRCDIR}/../deps/stable-diffusion.cpp/include
#cgo LDFLAGS: -L${SRCDIR}/../lib -lstable-diffusion
```

### Building Go with CGo

```bash
# Windows
set CGO_ENABLED=1
go build -tags sd

# Linux
CGO_ENABLED=1 go build -tags sd
export LD_LIBRARY_PATH=$PWD/lib:$LD_LIBRARY_PATH
```

## Troubleshooting

### CMake: CUDA not found

Ensure CUDA is in your PATH:

```bash
# Linux
export PATH=/usr/local/cuda/bin:$PATH
export LD_LIBRARY_PATH=/usr/local/cuda/lib64:$LD_LIBRARY_PATH

# Windows (PowerShell)
$env:PATH = "C:\Program Files\NVIDIA GPU Computing Toolkit\CUDA\v11.8\bin;$env:PATH"
```

### Linker errors: undefined reference

Make sure the library path is set:

```bash
# Linux
export LD_LIBRARY_PATH=$PWD/lib:$LD_LIBRARY_PATH

# Windows
# Add lib\ directory to PATH or copy DLL next to executable
```

### Out of memory during build

CUDA compilation can be memory-intensive. Try:
- Reduce parallel jobs: `cmake --build . -j4` instead of `-j$(nproc)`
- Close other applications
- Use fewer CUDA architectures

### GPU not detected at runtime

1. Check NVIDIA driver: `nvidia-smi`
2. Verify CUDA installation: `nvcc --version`
3. Ensure library can find CUDA runtime:
   - Windows: `cudart64_*.dll` in PATH
   - Linux: `/usr/local/cuda/lib64` in LD_LIBRARY_PATH

## License

stable-diffusion.cpp is licensed under MIT. See the source repository for details.

## References

- [stable-diffusion.cpp GitHub](https://github.com/leejet/stable-diffusion.cpp)
- [CUDA Toolkit Documentation](https://docs.nvidia.com/cuda/)
- [CMake Documentation](https://cmake.org/documentation/)
