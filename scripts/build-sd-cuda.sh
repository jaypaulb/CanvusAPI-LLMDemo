#!/bin/bash
# build-sd-cuda.sh
# Molecule: Composes git clone + cmake configuration + build execution
# Purpose: Build stable-diffusion.cpp with CUDA support for CanvusLocalLLM
#
# Usage:
#   ./scripts/build-sd-cuda.sh [OPTIONS]
#
# Options:
#   --clean       Remove existing build directory before building
#   --jobs N      Number of parallel build jobs (default: nproc)
#   --output DIR  Output directory for built libraries (default: deps/stable-diffusion.cpp/build)
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
readonly SD_REPO="https://github.com/leejet/stable-diffusion.cpp.git"
readonly SD_DIR="$PROJECT_ROOT/deps/stable-diffusion.cpp"

# Defaults
CLEAN_BUILD=false
BUILD_JOBS=$(nproc 2>/dev/null || echo 4)
OUTPUT_DIR="$SD_DIR/build"

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
    log_info "Setting up stable-diffusion.cpp repository..."

    if [[ -d "$SD_DIR/.git" ]]; then
        log_info "Repository exists, pulling latest changes..."
        cd "$SD_DIR"
        git fetch origin
        git pull origin master || git pull origin main
        # Update submodules (stable-diffusion.cpp uses them)
        git submodule update --init --recursive
    else
        log_info "Cloning stable-diffusion.cpp repository..."
        mkdir -p "$(dirname "$SD_DIR")"
        git clone --recursive "$SD_REPO" "$SD_DIR"
    fi

    log_success "Repository ready at: $SD_DIR"
}

configure_cmake() {
    log_info "Configuring CMake with CUDA support..."

    local build_dir="$SD_DIR/build"

    if [[ "$CLEAN_BUILD" == "true" && -d "$build_dir" ]]; then
        log_info "Cleaning existing build directory..."
        rm -rf "$build_dir"
    fi

    mkdir -p "$build_dir"
    cd "$build_dir"

    # Configure with CUDA support
    # SD_CUBLAS is the flag for CUDA support in stable-diffusion.cpp
    cmake .. \
        -DCMAKE_BUILD_TYPE=Release \
        -DSD_CUBLAS=ON \
        -DBUILD_SHARED_LIBS=OFF

    log_success "CMake configuration complete"
}

build_project() {
    log_info "Building stable-diffusion.cpp with $BUILD_JOBS parallel jobs..."

    cd "$SD_DIR/build"

    cmake --build . --config Release -j "$BUILD_JOBS"

    log_success "Build complete"
}

copy_artifacts() {
    log_info "Copying build artifacts to: $OUTPUT_DIR"

    # The build happens in-place, so artifacts are already in build dir
    # If a different output dir is specified, copy the relevant files
    if [[ "$OUTPUT_DIR" != "$SD_DIR/build" ]]; then
        mkdir -p "$OUTPUT_DIR"

        # Copy shared libraries
        find "$SD_DIR/build" -name "*.so" -exec cp {} "$OUTPUT_DIR/" \; 2>/dev/null || true
        find "$SD_DIR/build" -name "*.dll" -exec cp {} "$OUTPUT_DIR/" \; 2>/dev/null || true
        find "$SD_DIR/build" -name "*.dylib" -exec cp {} "$OUTPUT_DIR/" \; 2>/dev/null || true

        # Copy main binary (sd)
        if [[ -f "$SD_DIR/build/bin/sd" ]]; then
            cp "$SD_DIR/build/bin/sd" "$OUTPUT_DIR/"
        elif [[ -f "$SD_DIR/build/sd" ]]; then
            cp "$SD_DIR/build/sd" "$OUTPUT_DIR/"
        fi

        log_success "Artifacts copied to: $OUTPUT_DIR"
    else
        log_info "Artifacts available in: $OUTPUT_DIR"
    fi

    # List what was built
    echo ""
    log_info "Built artifacts:"
    find "$OUTPUT_DIR" -maxdepth 2 \( -name "*.so" -o -name "*.dll" -o -name "*.dylib" -o -name "sd" \) -exec ls -lh {} \; 2>/dev/null || true
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
    echo "  stable-diffusion.cpp CUDA Build Script"
    echo "========================================"
    echo ""

    check_dependencies
    clone_or_update_repo
    configure_cmake
    build_project
    copy_artifacts

    echo ""
    log_success "stable-diffusion.cpp build complete!"
    echo ""
    echo "Next steps:"
    echo "  - Binary: $OUTPUT_DIR/sd (or bin/sd)"
    echo "  - Generate image: $OUTPUT_DIR/sd -m /path/to/model.safetensors -p 'prompt'"
    echo ""
}

main "$@"
