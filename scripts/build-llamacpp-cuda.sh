#!/bin/bash
# build-llamacpp-cuda.sh
# Molecule: Composes git clone + cmake configuration + build execution
# Purpose: Build llama.cpp with CUDA support for CanvusLocalLLM
#
# Usage:
#   ./scripts/build-llamacpp-cuda.sh [OPTIONS]
#
# Options:
#   --clean       Remove existing build directory before building
#   --jobs N      Number of parallel build jobs (default: nproc)
#   --output DIR  Output directory for built libraries (default: deps/llama.cpp/build)
#   --help        Show this help message
#
# Requirements:
#   - CUDA Toolkit (nvcc in PATH)
#   - CMake >= 3.14
#   - Git
#   - C++ compiler (gcc/g++ or clang)

set -euo pipefail

# Constants
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
readonly LLAMACPP_REPO="https://github.com/ggerganov/llama.cpp.git"
readonly LLAMACPP_DIR="$PROJECT_ROOT/deps/llama.cpp"

# Defaults
CLEAN_BUILD=false
BUILD_JOBS=$(nproc 2>/dev/null || echo 4)
OUTPUT_DIR="$LLAMACPP_DIR/build"

# Colors for output (disabled if not a terminal)
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

show_help() {
    sed -n '2,18p' "$0" | sed 's/^# //' | sed 's/^#//'
    exit 0
}

check_dependencies() {
    log_info "Checking dependencies..."

    local missing=()

    # Check for git
    if ! command -v git &> /dev/null; then
        missing+=("git")
    fi

    # Check for cmake
    if ! command -v cmake &> /dev/null; then
        missing+=("cmake")
    fi

    # Check for CUDA compiler
    if ! command -v nvcc &> /dev/null; then
        log_warn "nvcc not found in PATH - CUDA support may not be available"
        log_warn "Ensure CUDA Toolkit is installed and nvcc is in your PATH"
        log_warn "Continuing build (will fail if CUDA headers not found)..."
    else
        local cuda_version
        cuda_version=$(nvcc --version | grep "release" | sed 's/.*release //' | sed 's/,.*//')
        log_info "Found CUDA version: $cuda_version"
    fi

    # Check for C++ compiler
    if ! command -v g++ &> /dev/null && ! command -v clang++ &> /dev/null; then
        missing+=("g++ or clang++")
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Missing required dependencies: ${missing[*]}"
        log_error "Please install them and try again."
        exit 1
    fi

    log_success "All required dependencies found"
}

clone_or_update_repo() {
    log_info "Setting up llama.cpp repository..."

    if [[ -d "$LLAMACPP_DIR/.git" ]]; then
        log_info "Repository exists, pulling latest changes..."
        cd "$LLAMACPP_DIR"
        git fetch origin
        git pull origin master || git pull origin main
    else
        log_info "Cloning llama.cpp repository..."
        mkdir -p "$(dirname "$LLAMACPP_DIR")"
        git clone --depth 1 "$LLAMACPP_REPO" "$LLAMACPP_DIR"
    fi

    log_success "Repository ready at: $LLAMACPP_DIR"
}

configure_cmake() {
    log_info "Configuring CMake with CUDA support..."

    local build_dir="$LLAMACPP_DIR/build"

    if [[ "$CLEAN_BUILD" == "true" && -d "$build_dir" ]]; then
        log_info "Cleaning existing build directory..."
        rm -rf "$build_dir"
    fi

    mkdir -p "$build_dir"
    cd "$build_dir"

    # Configure with CUDA support
    # LLAMA_CUBLAS is the legacy flag, newer versions use LLAMA_CUDA
    # We set both for compatibility
    cmake .. \
        -DCMAKE_BUILD_TYPE=Release \
        -DLLAMA_CUBLAS=ON \
        -DLLAMA_CUDA=ON \
        -DLLAMA_NATIVE=OFF \
        -DLLAMA_BUILD_TESTS=OFF \
        -DLLAMA_BUILD_EXAMPLES=ON \
        -DLLAMA_BUILD_SERVER=ON

    log_success "CMake configuration complete"
}

build_project() {
    log_info "Building llama.cpp with $BUILD_JOBS parallel jobs..."

    cd "$LLAMACPP_DIR/build"

    cmake --build . --config Release -j "$BUILD_JOBS"

    log_success "Build complete"
}

copy_artifacts() {
    log_info "Copying build artifacts to: $OUTPUT_DIR"

    # The build happens in-place, so artifacts are already in build dir
    # If a different output dir is specified, copy the relevant files
    if [[ "$OUTPUT_DIR" != "$LLAMACPP_DIR/build" ]]; then
        mkdir -p "$OUTPUT_DIR"

        # Copy shared libraries
        find "$LLAMACPP_DIR/build" -name "*.so" -exec cp {} "$OUTPUT_DIR/" \; 2>/dev/null || true
        find "$LLAMACPP_DIR/build" -name "*.dll" -exec cp {} "$OUTPUT_DIR/" \; 2>/dev/null || true
        find "$LLAMACPP_DIR/build" -name "*.dylib" -exec cp {} "$OUTPUT_DIR/" \; 2>/dev/null || true

        # Copy server binary
        if [[ -f "$LLAMACPP_DIR/build/bin/llama-server" ]]; then
            cp "$LLAMACPP_DIR/build/bin/llama-server" "$OUTPUT_DIR/"
        elif [[ -f "$LLAMACPP_DIR/build/bin/server" ]]; then
            cp "$LLAMACPP_DIR/build/bin/server" "$OUTPUT_DIR/"
        fi

        log_success "Artifacts copied to: $OUTPUT_DIR"
    else
        log_info "Artifacts available in: $OUTPUT_DIR"
    fi

    # List what was built
    echo ""
    log_info "Built artifacts:"
    find "$OUTPUT_DIR" -maxdepth 1 \( -name "*.so" -o -name "*.dll" -o -name "*.dylib" -o -name "llama-server" -o -name "server" \) -exec ls -lh {} \; 2>/dev/null || true
}

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --clean)
                CLEAN_BUILD=true
                shift
                ;;
            --jobs)
                BUILD_JOBS="$2"
                shift 2
                ;;
            --output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --help|-h)
                show_help
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                ;;
        esac
    done

    echo ""
    echo "========================================"
    echo "  llama.cpp CUDA Build Script"
    echo "========================================"
    echo ""

    check_dependencies
    clone_or_update_repo
    configure_cmake
    build_project
    copy_artifacts

    echo ""
    log_success "llama.cpp build complete!"
    echo ""
    echo "Next steps:"
    echo "  - Server binary: $OUTPUT_DIR/llama-server (or server)"
    echo "  - Start server:  $OUTPUT_DIR/llama-server -m /path/to/model.gguf"
    echo ""
}

main "$@"
