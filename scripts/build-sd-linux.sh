#!/usr/bin/env bash
#
# build-sd-linux.sh
#
# Cross-platform build script for CanvusLocalLLM with Stable Diffusion support on Linux.
#
# This script orchestrates:
#   1. Building stable-diffusion.cpp C library with CUDA
#   2. Building Go application with CGo bindings enabled
#   3. Bundling shared libraries with the binary
#   4. Creating a distributable tarball
#
# Prerequisites:
#   - GCC 9+ or Clang 10+
#   - CUDA Toolkit 11.8 or newer
#   - CMake 3.18 or newer
#   - Go 1.21 or newer
#   - Git (for cloning stable-diffusion.cpp if needed)
#
# Usage:
#   ./build-sd-linux.sh              # Build with defaults
#   ./build-sd-linux.sh --version 1.2.0
#   ./build-sd-linux.sh --clean      # Clean build from scratch
#   ./build-sd-linux.sh --skip-sd    # Skip SD library build (use existing)
#   ./build-sd-linux.sh --tarball    # Create distribution tarball
#
# Output:
#   - Binary: bin/canvuslocallm-sd-linux-amd64
#   - Libraries: lib/libstable-diffusion.so
#   - Tarball: dist/canvuslocallm-sd-VERSION-linux-amd64.tar.gz (if --tarball)
#

set -euo pipefail

# Constants
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
readonly SD_DIR="$PROJECT_ROOT/deps/stable-diffusion.cpp"
readonly LIB_DIR="$PROJECT_ROOT/lib"
readonly BIN_DIR="$PROJECT_ROOT/bin"
readonly DIST_DIR="$PROJECT_ROOT/dist"
readonly BINARY_NAME="canvuslocallm-sd-linux-amd64"

# Defaults
VERSION="1.0.0"
CLEAN_BUILD=false
SKIP_SD_BUILD=false
CREATE_TARBALL=false
VERBOSE=false

# Colors
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    NC='\033[0m'
else
    RED='' GREEN='' YELLOW='' BLUE='' CYAN='' BOLD='' NC=''
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

log_section() {
    echo ""
    echo -e "${CYAN}${BOLD}=== $1 ===${NC}"
    echo ""
}

show_help() {
    sed -n '2,25p' "$0" | sed 's/^# //' | sed 's/^#//'
    exit 0
}

check_prerequisites() {
    log_section "Checking Prerequisites"

    local missing=()

    # Check Go
    if ! command -v go &> /dev/null; then
        missing+=("go (1.21+)")
    else
        local go_version
        go_version=$(go version | sed 's/go version go//' | cut -d' ' -f1)
        log_info "Go: $go_version"
    fi

    # Check CMake
    if ! command -v cmake &> /dev/null; then
        missing+=("cmake (3.18+)")
    else
        local cmake_version
        cmake_version=$(cmake --version | head -n1 | cut -d' ' -f3)
        log_info "CMake: $cmake_version"
    fi

    # Check C++ compiler
    if command -v g++ &> /dev/null; then
        local gcc_version
        gcc_version=$(g++ --version | head -n1)
        log_info "GCC: $gcc_version"
    elif command -v clang++ &> /dev/null; then
        local clang_version
        clang_version=$(clang++ --version | head -n1)
        log_info "Clang: $clang_version"
    else
        missing+=("g++ or clang++")
    fi

    # Check CUDA (warning only, not fatal)
    if command -v nvcc &> /dev/null; then
        local cuda_version
        cuda_version=$(nvcc --version | grep "release" | awk '{print $6}')
        log_info "CUDA: $cuda_version"
    else
        log_warn "CUDA nvcc not found in PATH - SD will build without GPU acceleration"
    fi

    # Check Git
    if ! command -v git &> /dev/null; then
        missing+=("git")
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Missing required tools: ${missing[*]}"
        log_error "Please install them and try again."
        exit 1
    fi

    log_success "All prerequisites found"
}

clean_build_artifacts() {
    if [[ "$CLEAN_BUILD" != "true" ]]; then
        return 0
    fi

    log_section "Cleaning Build Artifacts"

    # Clean SD build
    if [[ -d "$SD_DIR/build" ]]; then
        log_info "Removing $SD_DIR/build"
        rm -rf "$SD_DIR/build"
    fi

    # Clean libraries
    if [[ -f "$LIB_DIR/libstable-diffusion.so" ]]; then
        log_info "Removing $LIB_DIR/libstable-diffusion.so"
        rm -f "$LIB_DIR/libstable-diffusion.so"
    fi

    # Clean Go binary
    if [[ -f "$BIN_DIR/$BINARY_NAME" ]]; then
        log_info "Removing $BIN_DIR/$BINARY_NAME"
        rm -f "$BIN_DIR/$BINARY_NAME"
    fi

    log_success "Clean complete"
}

build_sd_library() {
    if [[ "$SKIP_SD_BUILD" == "true" ]]; then
        log_section "Skipping SD Library Build"

        # Verify library exists
        if [[ ! -f "$LIB_DIR/libstable-diffusion.so" ]]; then
            log_error "libstable-diffusion.so not found at $LIB_DIR/libstable-diffusion.so"
            log_error "Cannot skip SD build when library doesn't exist"
            exit 1
        fi

        log_info "Using existing library at $LIB_DIR/libstable-diffusion.so"
        return 0
    fi

    log_section "Building stable-diffusion.cpp Library"

    # Check if build script exists
    if [[ ! -f "$SD_DIR/build-linux.sh" ]]; then
        log_error "Build script not found: $SD_DIR/build-linux.sh"
        exit 1
    fi

    # Run SD build script
    cd "$SD_DIR"

    local build_args=()
    [[ "$CLEAN_BUILD" == "true" ]] && build_args+=("--clean")

    if bash "$SD_DIR/build-linux.sh" "${build_args[@]}"; then
        log_success "stable-diffusion.cpp library built successfully"
    else
        log_error "Failed to build stable-diffusion.cpp library"
        exit 1
    fi

    # Verify library was created
    if [[ ! -f "$LIB_DIR/libstable-diffusion.so" ]]; then
        log_error "Library not found after build: $LIB_DIR/libstable-diffusion.so"
        exit 1
    fi

    local lib_size
    lib_size=$(ls -lh "$LIB_DIR/libstable-diffusion.so" | awk '{print $5}')
    log_info "Library size: $lib_size"
}

build_go_application() {
    log_section "Building Go Application with SD Support"

    cd "$PROJECT_ROOT"

    # Ensure bin directory exists
    mkdir -p "$BIN_DIR"

    log_info "Building with CGO_ENABLED=1 and -tags sd"

    # Set up CGo environment
    export CGO_ENABLED=1
    export GOOS=linux
    export GOARCH=amd64

    # Build flags
    local build_tags="sd"
    local ldflags="-s -w -X main.Version=$VERSION"

    # Add library path to runtime linker
    # Use $ORIGIN to make binary relocatable
    export CGO_LDFLAGS="-L${LIB_DIR} -lstable-diffusion -Wl,-rpath,\$ORIGIN/../lib"

    log_info "Build command: go build -tags $build_tags -ldflags=\"$ldflags\" -o \"$BIN_DIR/$BINARY_NAME\""

    if go build -tags "$build_tags" -ldflags="$ldflags" -o "$BIN_DIR/$BINARY_NAME" .; then
        log_success "Go application built successfully"
    else
        log_error "Failed to build Go application"
        exit 1
    fi

    # Verify binary was created
    if [[ ! -f "$BIN_DIR/$BINARY_NAME" ]]; then
        log_error "Binary not found after build: $BIN_DIR/$BINARY_NAME"
        exit 1
    fi

    # Make binary executable
    chmod +x "$BIN_DIR/$BINARY_NAME"

    local bin_size
    bin_size=$(ls -lh "$BIN_DIR/$BINARY_NAME" | awk '{print $5}')
    log_info "Binary size: $bin_size"

    # Verify binary links correctly
    log_info "Checking library dependencies..."
    if command -v ldd &> /dev/null; then
        ldd "$BIN_DIR/$BINARY_NAME" | grep -E "(libstable-diffusion|cuda|cublas)" || true
    fi
}

create_tarball_distribution() {
    if [[ "$CREATE_TARBALL" != "true" ]]; then
        return 0
    fi

    log_section "Creating Distribution Tarball"

    mkdir -p "$DIST_DIR"

    local tarball_name="canvuslocallm-sd-${VERSION}-linux-amd64.tar.gz"
    local tarball_path="$DIST_DIR/$tarball_name"
    local staging_dir="$DIST_DIR/staging"
    local app_dir="$staging_dir/canvuslocallm-sd"

    # Clean staging directory
    rm -rf "$staging_dir"
    mkdir -p "$app_dir"

    log_info "Preparing distribution files..."

    # Copy binary
    cp "$BIN_DIR/$BINARY_NAME" "$app_dir/canvuslocallm"
    chmod +x "$app_dir/canvuslocallm"

    # Copy libraries
    mkdir -p "$app_dir/lib"
    cp "$LIB_DIR/libstable-diffusion.so" "$app_dir/lib/"

    # Copy documentation
    [[ -f "$PROJECT_ROOT/README.md" ]] && cp "$PROJECT_ROOT/README.md" "$app_dir/"
    [[ -f "$PROJECT_ROOT/LICENSE.txt" ]] && cp "$PROJECT_ROOT/LICENSE.txt" "$app_dir/"
    [[ -f "$PROJECT_ROOT/example.env" ]] && cp "$PROJECT_ROOT/example.env" "$app_dir/"

    # Create models directory
    mkdir -p "$app_dir/models"
    echo "Place your SD model (sd-v1-5.safetensors) here" > "$app_dir/models/README.txt"

    # Create startup script
    cat > "$app_dir/start.sh" << 'EOF'
#!/bin/bash
# CanvusLocalLLM with Stable Diffusion Support
# This script sets up the library path and starts the application

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Add lib directory to LD_LIBRARY_PATH
export LD_LIBRARY_PATH="$SCRIPT_DIR/lib:$LD_LIBRARY_PATH"

# Run the application
exec "$SCRIPT_DIR/canvuslocallm" "$@"
EOF
    chmod +x "$app_dir/start.sh"

    # Create installation instructions
    cat > "$app_dir/INSTALL.txt" << EOF
CanvusLocalLLM with Stable Diffusion Support - Linux Installation
==================================================================

Prerequisites:
--------------
1. CUDA Toolkit 11.8+ (for GPU acceleration)
2. NVIDIA GPU with CUDA support
3. Linux kernel with NVIDIA driver installed

Installation:
-------------
1. Extract this archive to your desired location
2. Download SD v1.5 model to models/ directory:
   wget -O models/sd-v1-5.safetensors https://huggingface.co/runwayml/stable-diffusion-v1-5/resolve/main/v1-5-pruned.safetensors

3. Copy example.env to .env and configure:
   cp example.env .env
   nano .env

4. Set SD configuration in .env:
   SD_MODEL_PATH=models/sd-v1-5.safetensors
   SD_IMAGE_SIZE=512
   SD_INFERENCE_STEPS=20

Running:
--------
./start.sh

Or manually:
export LD_LIBRARY_PATH=\$(pwd)/lib:\$LD_LIBRARY_PATH
./canvuslocallm

Troubleshooting:
----------------
If you see "libstable-diffusion.so not found":
  export LD_LIBRARY_PATH=\$(pwd)/lib:\$LD_LIBRARY_PATH

If you see CUDA errors:
  - Verify CUDA is installed: nvcc --version
  - Verify NVIDIA driver: nvidia-smi
  - Check GPU compatibility with CUDA 11.8+

For more information, see README.md
EOF

    # Create tarball
    log_info "Creating tarball: $tarball_name"
    cd "$staging_dir"
    tar -czf "$tarball_path" "canvuslocallm-sd"

    # Clean staging
    cd "$PROJECT_ROOT"
    rm -rf "$staging_dir"

    if [[ -f "$tarball_path" ]]; then
        local tarball_size
        tarball_size=$(ls -lh "$tarball_path" | awk '{print $5}')
        log_success "Tarball created: $tarball_name ($tarball_size)"
        log_info "Location: $tarball_path"
    else
        log_error "Failed to create tarball"
        exit 1
    fi
}

verify_build() {
    log_section "Verifying Build"

    local errors=0

    # Check library
    if [[ -f "$LIB_DIR/libstable-diffusion.so" ]]; then
        log_success "Library present: libstable-diffusion.so"
    else
        log_error "Library missing: libstable-diffusion.so"
        errors=$((errors + 1))
    fi

    # Check binary
    if [[ -f "$BIN_DIR/$BINARY_NAME" ]]; then
        log_success "Binary present: $BINARY_NAME"

        # Check if executable
        if [[ -x "$BIN_DIR/$BINARY_NAME" ]]; then
            log_success "Binary is executable"
        else
            log_error "Binary is not executable"
            errors=$((errors + 1))
        fi
    else
        log_error "Binary missing: $BINARY_NAME"
        errors=$((errors + 1))
    fi

    # Check tarball if created
    if [[ "$CREATE_TARBALL" == "true" ]]; then
        local tarball_name="canvuslocallm-sd-${VERSION}-linux-amd64.tar.gz"
        if [[ -f "$DIST_DIR/$tarball_name" ]]; then
            log_success "Tarball present: $tarball_name"
        else
            log_error "Tarball missing: $tarball_name"
            errors=$((errors + 1))
        fi
    fi

    if [[ $errors -gt 0 ]]; then
        log_error "Build verification failed with $errors error(s)"
        return 1
    fi

    log_success "Build verification passed"
    return 0
}

print_summary() {
    log_section "Build Summary"

    echo "Version:  $VERSION"
    echo "Platform: Linux amd64"
    echo ""
    echo "Artifacts:"
    echo "  Binary:  $BIN_DIR/$BINARY_NAME"
    echo "  Library: $LIB_DIR/libstable-diffusion.so"

    if [[ "$CREATE_TARBALL" == "true" ]]; then
        echo "  Tarball: $DIST_DIR/canvuslocallm-sd-${VERSION}-linux-amd64.tar.gz"
    fi

    echo ""
    echo "Next steps:"
    echo "  1. Download SD model: wget -O models/sd-v1-5.safetensors <model-url>"
    echo "  2. Configure .env: cp example.env .env && nano .env"
    echo "  3. Run: export LD_LIBRARY_PATH=$LIB_DIR:\$LD_LIBRARY_PATH && $BIN_DIR/$BINARY_NAME"
    echo ""
}

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --version)
                VERSION="$2"
                shift 2
                ;;
            --clean)
                CLEAN_BUILD=true
                shift
                ;;
            --skip-sd)
                SKIP_SD_BUILD=true
                shift
                ;;
            --tarball)
                CREATE_TARBALL=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                set -x
                shift
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
    echo "========================================================"
    echo "  CanvusLocalLLM with Stable Diffusion - Linux Build"
    echo "  Version: $VERSION"
    echo "========================================================"
    echo ""

    check_prerequisites
    clean_build_artifacts
    build_sd_library
    build_go_application
    create_tarball_distribution
    verify_build
    print_summary

    log_success "Build complete!"
}

main "$@"
