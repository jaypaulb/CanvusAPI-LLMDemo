#!/bin/bash
# build-all.sh
# Page: Master build orchestrator for CanvusLocalLLM
# Purpose: Build all or selected artifacts (Go binaries, packages, installers)
#
# Usage:
#   ./scripts/build-all.sh [OPTIONS]
#
# Options:
#   --version VER   Version string for all builds (default: 1.0.0)
#   --output DIR    Output directory for all artifacts (default: dist/)
#   --clean         Remove existing build artifacts before building
#   --tarball       Build Linux tarball only
#   --deb           Build Debian package only
#   --windows       Build Windows binary and NSIS installer only
#   --linux         Build all Linux artifacts (tarball + deb)
#   --all           Build all artifacts (default if no target specified)
#   --skip-build    Skip Go build, use existing binaries
#   --help          Show this help message
#
# Examples:
#   ./scripts/build-all.sh --all --version 1.2.0
#   ./scripts/build-all.sh --tarball --deb --clean
#   ./scripts/build-all.sh --windows --version 1.0.0
#
# Output:
#   Creates artifacts in dist/:
#   - canvuslocallm-VERSION-linux-amd64.tar.gz (tarball)
#   - canvuslocallm_VERSION_amd64.deb (Debian package)
#   - canvuslocallm-windows-amd64.exe (Windows binary)
#   - CanvusLocalLLM-VERSION-Setup.exe (Windows installer, if NSIS available)

set -euo pipefail

# Constants
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
readonly BINARY_NAME="canvuslocallm"

# Defaults
VERSION="1.0.0"
OUTPUT_DIR="$PROJECT_ROOT/dist"
CLEAN_BUILD=false
SKIP_BUILD=false

# Build targets (false = not selected)
BUILD_TARBALL=false
BUILD_DEB=false
BUILD_WINDOWS=false
BUILD_ALL=false

# Track build results
declare -A BUILD_RESULTS

# Colors for output (disabled if not a terminal)
if [[ -t 2 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    CYAN=''
    BOLD=''
    NC=''
fi

# All log functions output to stderr to avoid interfering with function return values
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_section() {
    echo "" >&2
    echo -e "${CYAN}${BOLD}=== $1 ===${NC}" >&2
    echo "" >&2
}

show_help() {
    sed -n '2,31p' "$0" | sed 's/^# //' | sed 's/^#//'
    exit 0
}

check_go_compiler() {
    log_info "Checking for Go compiler..."

    if ! command -v go &> /dev/null; then
        log_error "Go compiler not found. Please install Go 1.21 or later."
        exit 1
    fi

    local go_version
    go_version=$(go version | sed 's/go version go//' | cut -d' ' -f1)
    log_info "Found Go version: $go_version"
}

check_nsis() {
    if command -v makensis &> /dev/null; then
        local nsis_version
        nsis_version=$(makensis -VERSION 2>/dev/null || echo "unknown")
        log_info "Found NSIS version: $nsis_version"
        return 0
    else
        log_warn "NSIS (makensis) not found - Windows installer will not be built"
        log_warn "Install NSIS to build Windows installers: apt install nsis"
        return 1
    fi
}

clean_build_dir() {
    if [[ "$CLEAN_BUILD" == "true" ]]; then
        log_info "Cleaning build directory: $OUTPUT_DIR"

        if [[ -d "$OUTPUT_DIR" ]]; then
            rm -rf "$OUTPUT_DIR"
        fi

        # Also clean binaries in project root
        rm -f "$PROJECT_ROOT/$BINARY_NAME"
        rm -f "$PROJECT_ROOT/${BINARY_NAME}.exe"
        rm -f "$PROJECT_ROOT/bin/${BINARY_NAME}.exe"
        rm -f "$PROJECT_ROOT/bin/CanvusLocalLLM.exe"

        log_success "Build directory cleaned"
    fi

    # Ensure output directory exists
    mkdir -p "$OUTPUT_DIR"
}

build_linux_binary() {
    if [[ "$SKIP_BUILD" == "true" ]]; then
        log_info "Skipping Linux binary build (--skip-build specified)"
        if [[ ! -f "$PROJECT_ROOT/$BINARY_NAME" ]]; then
            log_error "Linux binary not found at $PROJECT_ROOT/$BINARY_NAME"
            return 1
        fi
        return 0
    fi

    log_info "Building Linux amd64 binary..."

    cd "$PROJECT_ROOT"

    export GOOS=linux
    export GOARCH=amd64
    export CGO_ENABLED=0

    go build -ldflags="-s -w -X main.Version=$VERSION" -o "$BINARY_NAME" .

    if [[ ! -f "$PROJECT_ROOT/$BINARY_NAME" ]]; then
        log_error "Linux binary build failed"
        return 1
    fi

    local binary_size
    binary_size=$(ls -lh "$PROJECT_ROOT/$BINARY_NAME" | awk '{print $5}')
    log_success "Linux binary built: $BINARY_NAME ($binary_size)"

    return 0
}

build_windows_binary() {
    if [[ "$SKIP_BUILD" == "true" ]]; then
        log_info "Skipping Windows binary build (--skip-build specified)"
        if [[ ! -f "$PROJECT_ROOT/bin/CanvusLocalLLM.exe" ]] && [[ ! -f "$OUTPUT_DIR/${BINARY_NAME}-windows-amd64.exe" ]]; then
            log_error "Windows binary not found"
            return 1
        fi
        return 0
    fi

    log_info "Building Windows amd64 binary..."

    cd "$PROJECT_ROOT"

    export GOOS=windows
    export GOARCH=amd64
    export CGO_ENABLED=0

    # Build to bin/ for NSIS compatibility and dist/ for standalone
    mkdir -p "$PROJECT_ROOT/bin"
    go build -ldflags="-s -w -X main.Version=$VERSION" -o "bin/CanvusLocalLLM.exe" .

    if [[ ! -f "$PROJECT_ROOT/bin/CanvusLocalLLM.exe" ]]; then
        log_error "Windows binary build failed"
        return 1
    fi

    # Copy to dist/ with standardized name
    cp "$PROJECT_ROOT/bin/CanvusLocalLLM.exe" "$OUTPUT_DIR/${BINARY_NAME}-windows-amd64.exe"

    local binary_size
    binary_size=$(ls -lh "$PROJECT_ROOT/bin/CanvusLocalLLM.exe" | awk '{print $5}')
    log_success "Windows binary built: CanvusLocalLLM.exe ($binary_size)"

    return 0
}

build_tarball() {
    log_section "Building Linux Tarball"

    local build_args=("--version" "$VERSION" "--output" "$OUTPUT_DIR")

    if [[ "$CLEAN_BUILD" == "true" ]]; then
        build_args+=("--clean")
    fi

    if [[ "$SKIP_BUILD" == "true" ]]; then
        build_args+=("--skip-build")
    fi

    if "$SCRIPT_DIR/build-tarball.sh" "${build_args[@]}"; then
        BUILD_RESULTS["tarball"]="success"
        return 0
    else
        BUILD_RESULTS["tarball"]="failed"
        return 1
    fi
}

build_deb() {
    log_section "Building Debian Package"

    local build_args=("--version" "$VERSION" "--output" "$OUTPUT_DIR")

    if [[ "$CLEAN_BUILD" == "true" ]]; then
        build_args+=("--clean")
    fi

    if [[ "$SKIP_BUILD" == "true" ]]; then
        build_args+=("--skip-build")
    fi

    if "$SCRIPT_DIR/build-deb.sh" "${build_args[@]}"; then
        BUILD_RESULTS["deb"]="success"
        return 0
    else
        BUILD_RESULTS["deb"]="failed"
        return 1
    fi
}

build_windows() {
    log_section "Building Windows Artifacts"

    # Build Windows binary
    if ! build_windows_binary; then
        BUILD_RESULTS["windows_binary"]="failed"
        return 1
    fi
    BUILD_RESULTS["windows_binary"]="success"

    # Build NSIS installer if available
    if check_nsis; then
        log_info "Building Windows installer..."

        # Ensure LICENSE.txt exists for NSIS
        if [[ ! -f "$PROJECT_ROOT/LICENSE.txt" ]] && [[ -f "$PROJECT_ROOT/LICENSE" ]]; then
            cp "$PROJECT_ROOT/LICENSE" "$PROJECT_ROOT/LICENSE.txt"
        fi

        cd "$PROJECT_ROOT/installer/windows"

        if makensis -DPRODUCT_VERSION="$VERSION" canvusapi.nsi; then
            # Move installer to dist/
            local installer_name="CanvusLocalLLM-${VERSION}-Setup.exe"
            if [[ -f "$PROJECT_ROOT/installer/windows/$installer_name" ]]; then
                mv "$PROJECT_ROOT/installer/windows/$installer_name" "$OUTPUT_DIR/"
                log_success "Windows installer built: $installer_name"
                BUILD_RESULTS["windows_installer"]="success"
            else
                log_warn "Installer file not found after build"
                BUILD_RESULTS["windows_installer"]="not_found"
            fi
        else
            log_error "NSIS build failed"
            BUILD_RESULTS["windows_installer"]="failed"
        fi
    else
        BUILD_RESULTS["windows_installer"]="skipped"
    fi

    return 0
}

print_summary() {
    log_section "Build Summary"

    echo "Version: $VERSION" >&2
    echo "Output:  $OUTPUT_DIR" >&2
    echo "" >&2

    # List results
    local has_failures=false

    for target in "${!BUILD_RESULTS[@]}"; do
        local status="${BUILD_RESULTS[$target]}"
        local status_color=""

        case "$status" in
            "success")
                status_color="${GREEN}SUCCESS${NC}"
                ;;
            "failed")
                status_color="${RED}FAILED${NC}"
                has_failures=true
                ;;
            "skipped")
                status_color="${YELLOW}SKIPPED${NC}"
                ;;
            "not_found")
                status_color="${YELLOW}NOT FOUND${NC}"
                ;;
        esac

        printf "  %-20s %b\n" "$target:" "$status_color" >&2
    done

    echo "" >&2

    # List generated artifacts
    echo "Generated artifacts:" >&2
    if [[ -d "$OUTPUT_DIR" ]]; then
        find "$OUTPUT_DIR" -maxdepth 1 -type f \( -name "*.tar.gz" -o -name "*.deb" -o -name "*.exe" \) -exec ls -lh {} \; 2>/dev/null | while read -r line; do
            echo "  $line" >&2
        done
    fi

    echo "" >&2

    if [[ "$has_failures" == "true" ]]; then
        log_error "Some builds failed. Check the output above for details."
        return 1
    else
        log_success "All requested builds completed successfully!"
        return 0
    fi
}

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --version)
                VERSION="$2"
                shift 2
                ;;
            --output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --clean)
                CLEAN_BUILD=true
                shift
                ;;
            --skip-build)
                SKIP_BUILD=true
                shift
                ;;
            --tarball)
                BUILD_TARBALL=true
                shift
                ;;
            --deb)
                BUILD_DEB=true
                shift
                ;;
            --windows)
                BUILD_WINDOWS=true
                shift
                ;;
            --linux)
                BUILD_TARBALL=true
                BUILD_DEB=true
                shift
                ;;
            --all)
                BUILD_ALL=true
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

    # If no targets specified, build all
    if [[ "$BUILD_TARBALL" == "false" ]] && \
       [[ "$BUILD_DEB" == "false" ]] && \
       [[ "$BUILD_WINDOWS" == "false" ]] && \
       [[ "$BUILD_ALL" == "false" ]]; then
        BUILD_ALL=true
    fi

    # If --all, enable all targets
    if [[ "$BUILD_ALL" == "true" ]]; then
        BUILD_TARBALL=true
        BUILD_DEB=true
        BUILD_WINDOWS=true
    fi

    echo "" >&2
    echo "========================================" >&2
    echo "  CanvusLocalLLM Master Build Script" >&2
    echo "  Version: $VERSION" >&2
    echo "========================================" >&2
    echo "" >&2

    # Show what will be built
    echo "Build targets:" >&2
    [[ "$BUILD_TARBALL" == "true" ]] && echo "  - Linux tarball" >&2
    [[ "$BUILD_DEB" == "true" ]] && echo "  - Debian package" >&2
    [[ "$BUILD_WINDOWS" == "true" ]] && echo "  - Windows binary and installer" >&2
    echo "" >&2

    # Check Go compiler (required for all builds)
    check_go_compiler

    # Clean if requested
    clean_build_dir

    # Build requested targets
    local exit_code=0

    if [[ "$BUILD_TARBALL" == "true" ]]; then
        if ! build_tarball; then
            exit_code=1
        fi
    fi

    if [[ "$BUILD_DEB" == "true" ]]; then
        if ! build_deb; then
            exit_code=1
        fi
    fi

    if [[ "$BUILD_WINDOWS" == "true" ]]; then
        if ! build_windows; then
            exit_code=1
        fi
    fi

    # Print summary
    if ! print_summary; then
        exit_code=1
    fi

    exit $exit_code
}

main "$@"
