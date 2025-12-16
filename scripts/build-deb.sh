#!/bin/bash
# build-deb.sh
# Page: Orchestrates the complete Debian package build process
# Purpose: Build CanvusLocalLLM for Linux and package as .deb
#
# Usage:
#   ./scripts/build-deb.sh [OPTIONS]
#
# Options:
#   --version VER   Version string for the package (default: 1.0.0)
#   --output DIR    Output directory for .deb file (default: dist/)
#   --clean         Remove existing build artifacts before building
#   --skip-build    Skip Go build, use existing binary
#   --help          Show this help message
#
# Output:
#   Creates canvuslocallm_VERSION_amd64.deb containing:
#   - /opt/canvuslocallm/canvuslocallm (binary)
#   - /opt/canvuslocallm/canvuslocallm.service (systemd unit)
#   - /opt/canvuslocallm/.env.example (configuration template)
#   - DEBIAN scripts (postinst, prerm, postrm)

set -euo pipefail

# Constants
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
readonly BINARY_NAME="canvuslocallm"
readonly PACKAGE_NAME="canvuslocallm"
readonly INSTALL_DIR="/opt/canvuslocallm"

# Defaults
VERSION="1.0.0"
OUTPUT_DIR="$PROJECT_ROOT/dist"
CLEAN_BUILD=false
SKIP_BUILD=false

# Colors for output (disabled if not a terminal)
if [[ -t 2 ]]; then
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

show_help() {
    sed -n '2,21p' "$0" | sed 's/^# //' | sed 's/^#//'
    exit 0
}

check_dependencies() {
    log_info "Checking build dependencies..."

    local missing=()

    # Check for Go compiler
    if ! command -v go &> /dev/null; then
        missing+=("go")
    else
        local go_version
        go_version=$(go version | sed 's/go version go//' | cut -d' ' -f1)
        log_info "Found Go version: $go_version"
    fi

    # Check for dpkg-deb
    if ! command -v dpkg-deb &> /dev/null; then
        missing+=("dpkg-deb")
    fi

    # Check for fakeroot (optional but recommended)
    if ! command -v fakeroot &> /dev/null; then
        log_warn "fakeroot not found - package may have incorrect permissions"
        log_warn "Install with: sudo apt-get install fakeroot"
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Missing required dependencies: ${missing[*]}"
        log_error "Please install them and try again."
        exit 1
    fi

    log_success "All required dependencies found"
}

validate_source_files() {
    log_info "Validating source files..."

    local missing=()

    # Check for required source files
    if [[ ! -f "$PROJECT_ROOT/main.go" ]]; then
        missing+=("main.go")
    fi

    if [[ ! -d "$PROJECT_ROOT/installer/debian/DEBIAN" ]]; then
        missing+=("installer/debian/DEBIAN")
    fi

    if [[ ! -f "$PROJECT_ROOT/installer/debian/DEBIAN/postinst" ]]; then
        missing+=("installer/debian/DEBIAN/postinst")
    fi

    if [[ ! -f "$PROJECT_ROOT/installer/debian/DEBIAN/prerm" ]]; then
        missing+=("installer/debian/DEBIAN/prerm")
    fi

    if [[ ! -f "$PROJECT_ROOT/installer/debian/DEBIAN/postrm" ]]; then
        missing+=("installer/debian/DEBIAN/postrm")
    fi

    if [[ ! -f "$PROJECT_ROOT/installer/linux/canvuslocallm.service" ]]; then
        missing+=("installer/linux/canvuslocallm.service")
    fi

    if [[ ! -f "$PROJECT_ROOT/example.env" ]]; then
        missing+=("example.env")
    fi

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Missing required source files: ${missing[*]}"
        log_error "Please ensure you're running from the project root."
        exit 1
    fi

    log_success "All required source files found"
}

build_binary() {
    if [[ "$SKIP_BUILD" == "true" ]]; then
        log_info "Skipping Go build (--skip-build specified)"
        if [[ ! -f "$PROJECT_ROOT/$BINARY_NAME" ]]; then
            log_error "Binary not found at $PROJECT_ROOT/$BINARY_NAME"
            log_error "Remove --skip-build or build manually first."
            exit 1
        fi
        return 0
    fi

    log_info "Building $BINARY_NAME for Linux amd64..."

    cd "$PROJECT_ROOT"

    # Set build environment for Linux amd64
    export GOOS=linux
    export GOARCH=amd64
    export CGO_ENABLED=0

    # Build with optimizations
    go build -ldflags="-s -w -X main.Version=$VERSION" -o "$BINARY_NAME" .

    if [[ ! -f "$PROJECT_ROOT/$BINARY_NAME" ]]; then
        log_error "Build failed: binary not created"
        exit 1
    fi

    local binary_size
    binary_size=$(ls -lh "$PROJECT_ROOT/$BINARY_NAME" | awk '{print $5}')
    log_success "Binary built: $BINARY_NAME ($binary_size)"
}

create_package_structure() {
    log_info "Creating Debian package structure..."

    local pkg_dir="$OUTPUT_DIR/deb-staging/${PACKAGE_NAME}_${VERSION}_amd64"

    if [[ "$CLEAN_BUILD" == "true" && -d "$pkg_dir" ]]; then
        log_info "Cleaning existing staging directory..."
        rm -rf "$pkg_dir"
    fi

    # Create directory structure
    mkdir -p "$pkg_dir/DEBIAN"
    mkdir -p "$pkg_dir/opt/canvuslocallm"

    # Return the package directory path via stdout
    echo "$pkg_dir"
}

generate_control_file() {
    local pkg_dir="$1"

    log_info "Generating control file with version $VERSION..."

    # Read the template control file and update version
    cat > "$pkg_dir/DEBIAN/control" << EOF
Package: ${PACKAGE_NAME}
Version: ${VERSION}
Section: misc
Priority: optional
Architecture: amd64
Maintainer: CanvusLocalLLM Team <support@canvuslocallm.local>
Depends: systemd
Description: Local AI integration service for Canvus
 CanvusLocalLLM connects Canvus collaborative workspaces with local AI
 services via llama.cpp ecosystem. It monitors canvas widgets in real-time,
 processes AI prompts enclosed in {{ }}, and handles PDF analysis, canvas
 analysis, and image generation using embedded multimodal models with
 cloud fallback support.
 .
 Features:
  - Real-time canvas monitoring
  - AI prompt processing
  - PDF analysis and summarization
  - Image generation support
  - Handwriting recognition
EOF

    log_success "Generated control file"
}

copy_debian_scripts() {
    local pkg_dir="$1"

    log_info "Copying Debian maintainer scripts..."

    # Copy postinst
    cp "$PROJECT_ROOT/installer/debian/DEBIAN/postinst" "$pkg_dir/DEBIAN/"
    chmod 755 "$pkg_dir/DEBIAN/postinst"
    log_success "Copied postinst"

    # Copy prerm
    cp "$PROJECT_ROOT/installer/debian/DEBIAN/prerm" "$pkg_dir/DEBIAN/"
    chmod 755 "$pkg_dir/DEBIAN/prerm"
    log_success "Copied prerm"

    # Copy postrm
    cp "$PROJECT_ROOT/installer/debian/DEBIAN/postrm" "$pkg_dir/DEBIAN/"
    chmod 755 "$pkg_dir/DEBIAN/postrm"
    log_success "Copied postrm"

    # Copy conffiles if exists
    if [[ -f "$PROJECT_ROOT/installer/debian/DEBIAN/conffiles" ]]; then
        cp "$PROJECT_ROOT/installer/debian/DEBIAN/conffiles" "$pkg_dir/DEBIAN/"
        chmod 644 "$pkg_dir/DEBIAN/conffiles"
        log_success "Copied conffiles"
    fi
}

copy_application_files() {
    local pkg_dir="$1"

    log_info "Copying application files..."

    # Copy binary
    cp "$PROJECT_ROOT/$BINARY_NAME" "$pkg_dir/opt/canvuslocallm/"
    chmod 755 "$pkg_dir/opt/canvuslocallm/$BINARY_NAME"
    log_success "Copied $BINARY_NAME"

    # Copy systemd service file
    cp "$PROJECT_ROOT/installer/linux/canvuslocallm.service" "$pkg_dir/opt/canvuslocallm/"
    chmod 644 "$pkg_dir/opt/canvuslocallm/canvuslocallm.service"
    log_success "Copied canvuslocallm.service"

    # Copy example.env as .env.example
    cp "$PROJECT_ROOT/example.env" "$pkg_dir/opt/canvuslocallm/.env.example"
    chmod 644 "$pkg_dir/opt/canvuslocallm/.env.example"
    log_success "Copied .env.example"
}

build_deb_package() {
    local pkg_dir="$1"
    local deb_name="${PACKAGE_NAME}_${VERSION}_amd64.deb"
    local deb_path="$OUTPUT_DIR/$deb_name"

    log_info "Building Debian package: $deb_name"

    # Create output directory if needed
    mkdir -p "$OUTPUT_DIR"

    # Remove existing deb if present
    if [[ -f "$deb_path" ]]; then
        rm -f "$deb_path"
    fi

    # Build the .deb package
    # Use fakeroot if available for correct file ownership
    if command -v fakeroot &> /dev/null; then
        fakeroot dpkg-deb --build "$pkg_dir" "$deb_path"
    else
        dpkg-deb --build "$pkg_dir" "$deb_path"
    fi

    if [[ ! -f "$deb_path" ]]; then
        log_error "Failed to create .deb package"
        exit 1
    fi

    local deb_size
    deb_size=$(ls -lh "$deb_path" | awk '{print $5}')
    log_success "Created .deb package: $deb_path ($deb_size)"

    # Return the deb path via stdout
    echo "$deb_path"
}

cleanup_staging() {
    local pkg_dir="$1"

    log_info "Cleaning up staging directory..."
    rm -rf "$(dirname "$pkg_dir")"
    log_success "Cleanup complete"
}

verify_package() {
    local deb_path="$1"

    log_info "Verifying package contents..."

    echo "" >&2
    echo "Package information:" >&2
    echo "-------------------" >&2
    dpkg-deb --info "$deb_path" 2>&1 | sed 's/^/  /' >&2
    echo "" >&2

    echo "Package contents:" >&2
    echo "-----------------" >&2
    dpkg-deb --contents "$deb_path" 2>&1 | sed 's/^/  /' >&2
    echo "" >&2

    # Verify essential files are present
    local required_files=(
        "./opt/canvuslocallm/$BINARY_NAME"
        "./opt/canvuslocallm/canvuslocallm.service"
        "./opt/canvuslocallm/.env.example"
    )

    local missing=()
    local contents
    contents=$(dpkg-deb --contents "$deb_path")

    for file in "${required_files[@]}"; do
        if ! echo "$contents" | grep -q "$file"; then
            missing+=("$file")
        fi
    done

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Package missing required files: ${missing[*]}"
        exit 1
    fi

    # Verify DEBIAN scripts exist
    local control_info
    control_info=$(dpkg-deb --info "$deb_path")

    if ! echo "$control_info" | grep -q "Package: ${PACKAGE_NAME}"; then
        log_error "Package control file is invalid"
        exit 1
    fi

    log_success "Package verification passed"
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
            --help|-h)
                show_help
                ;;
            *)
                log_error "Unknown option: $1"
                show_help
                ;;
        esac
    done

    echo "" >&2
    echo "========================================" >&2
    echo "  CanvusLocalLLM Debian Package Builder" >&2
    echo "  Version: $VERSION" >&2
    echo "========================================" >&2
    echo "" >&2

    check_dependencies
    validate_source_files
    build_binary

    local pkg_dir
    pkg_dir=$(create_package_structure)

    generate_control_file "$pkg_dir"
    copy_debian_scripts "$pkg_dir"
    copy_application_files "$pkg_dir"

    local deb_path
    deb_path=$(build_deb_package "$pkg_dir")

    verify_package "$deb_path"

    cleanup_staging "$pkg_dir"

    echo "" >&2
    log_success "Build complete!"
    echo "" >&2
    echo "Output: $deb_path" >&2
    echo "" >&2
    echo "To install the package:" >&2
    echo "  sudo dpkg -i $deb_path" >&2
    echo "" >&2
    echo "To verify the package:" >&2
    echo "  dpkg-deb --info $deb_path" >&2
    echo "  dpkg-deb --contents $deb_path" >&2
    echo "" >&2
    echo "After installation:" >&2
    echo "  sudo nano /opt/canvuslocallm/.env" >&2
    echo "  sudo systemctl start canvuslocallm" >&2
    echo "" >&2
}

main "$@"
