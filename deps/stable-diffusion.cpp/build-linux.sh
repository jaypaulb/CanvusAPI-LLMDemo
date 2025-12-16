#!/usr/bin/env bash
#
# build-linux.sh
#
# Build script for stable-diffusion.cpp on Linux with CUDA support.
#
# Prerequisites:
#   - GCC 9+ or Clang 10+
#   - CUDA Toolkit 11.8 or newer
#   - CMake 3.18 or newer
#   - Git (for cloning source)
#
# Usage:
#   ./build-linux.sh              # Clone source and build
#   ./build-linux.sh --skip-clone # Build only (if source already exists)
#   ./build-linux.sh --clean      # Clean build directory first
#   ./build-linux.sh --debug      # Build with debug symbols
#
# Output:
#   ../../lib/libstable-diffusion.so

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SRC_DIR="${SCRIPT_DIR}/src"
BUILD_DIR="${SCRIPT_DIR}/build"
LIB_DIR="$(dirname "$(dirname "${SCRIPT_DIR}")")/lib"

# Parse arguments
SKIP_CLONE=false
CLEAN=false
DEBUG=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-clone)
            SKIP_CLONE=true
            shift
            ;;
        --clean)
            CLEAN=true
            shift
            ;;
        --debug)
            DEBUG=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--skip-clone] [--clean] [--debug]"
            exit 1
            ;;
    esac
done

echo ""
echo "=== stable-diffusion.cpp Linux Build ==="
echo ""

# Check prerequisites
echo "Checking prerequisites..."

# Check CMake
if ! command -v cmake &> /dev/null; then
    echo "ERROR: CMake not found. Please install CMake 3.18+"
    echo "  Ubuntu/Debian: sudo apt install cmake"
    echo "  Or download: https://cmake.org/download/"
    exit 1
fi
CMAKE_VERSION=$(cmake --version | head -n1)
echo "  CMake: ${CMAKE_VERSION}"

# Check C++ compiler
if command -v g++ &> /dev/null; then
    GXX_VERSION=$(g++ --version | head -n1)
    echo "  GCC: ${GXX_VERSION}"
elif command -v clang++ &> /dev/null; then
    CLANG_VERSION=$(clang++ --version | head -n1)
    echo "  Clang: ${CLANG_VERSION}"
else
    echo "ERROR: No C++ compiler found. Please install GCC or Clang."
    exit 1
fi

# Check CUDA
if command -v nvcc &> /dev/null; then
    CUDA_VERSION=$(nvcc --version | grep "release" | awk '{print $6}')
    echo "  CUDA: ${CUDA_VERSION}"
else
    echo "WARNING: CUDA nvcc not found in PATH"
    echo "  Make sure CUDA Toolkit is installed and /usr/local/cuda/bin is in PATH"
fi

# Check for CUDA libraries
if [ -d "/usr/local/cuda" ]; then
    echo "  CUDA path: /usr/local/cuda"
else
    echo "WARNING: /usr/local/cuda not found"
fi

# Clean if requested
if [ "$CLEAN" = true ] && [ -d "$BUILD_DIR" ]; then
    echo ""
    echo "Cleaning build directory..."
    rm -rf "$BUILD_DIR"
fi

# Clone source if needed
if [ "$SKIP_CLONE" = false ]; then
    if [ ! -d "$SRC_DIR" ]; then
        echo ""
        echo "Cloning stable-diffusion.cpp..."
        git clone --depth 1 https://github.com/leejet/stable-diffusion.cpp.git "$SRC_DIR"
    else
        echo "Source directory already exists: $SRC_DIR"
    fi
fi

# Create directories
mkdir -p "$BUILD_DIR"
mkdir -p "$LIB_DIR"

# Configure CMake
echo ""
echo "Configuring CMake..."

BUILD_TYPE="Release"
if [ "$DEBUG" = true ]; then
    BUILD_TYPE="Debug"
fi

cd "$BUILD_DIR"

cmake .. \
    -DGGML_CUDA=ON \
    -DCMAKE_BUILD_TYPE="$BUILD_TYPE" \
    -DCMAKE_CUDA_ARCHITECTURES="75;86;89" \
    -DBUILD_SHARED_LIBS=ON \
    -DSD_BUILD_EXAMPLES=OFF

# Build
echo ""
echo "Building with $(nproc) parallel jobs..."

cmake --build . --config "$BUILD_TYPE" -j"$(nproc)"

# Verify output
echo ""
echo "Verifying build output..."

SO_PATH="${LIB_DIR}/libstable-diffusion.so"
if [ -f "$SO_PATH" ]; then
    SO_SIZE=$(du -h "$SO_PATH" | cut -f1)
    echo "SUCCESS: Built libstable-diffusion.so (${SO_SIZE})"
else
    echo "WARNING: libstable-diffusion.so not found at expected location"
    echo "  Expected: $SO_PATH"
    echo "  Check build output for library location"

    # Try to find it
    FOUND_SO=$(find "$BUILD_DIR" -name "*.so" -type f 2>/dev/null | head -1)
    if [ -n "$FOUND_SO" ]; then
        echo "  Found: $FOUND_SO"
        echo "  Copying to lib directory..."
        cp "$FOUND_SO" "$LIB_DIR/"
    fi
fi

echo ""
echo "=== Build Complete ==="
echo ""
echo "Next steps:"
echo "  1. Ensure libstable-diffusion.so is in: $LIB_DIR"
echo "  2. Download SD v1.5 model to: models/sd-v1-5.safetensors"
echo "  3. Set library path: export LD_LIBRARY_PATH=$LIB_DIR:\$LD_LIBRARY_PATH"
echo "  4. Build Go application with: go build -tags sd"
echo ""
