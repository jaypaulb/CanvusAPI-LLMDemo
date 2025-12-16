#!/bin/bash
# build-tarball.sh
# Page: Orchestrates the complete tarball build process
# Purpose: Build CanvusLocalLLM for Linux and package as tarball
#
# Usage:
#   ./scripts/build-tarball.sh [OPTIONS]
#
# Options:
#   --version VER   Version string for the tarball (default: 1.0.0)
#   --output DIR    Output directory for tarball (default: dist/)
#   --clean         Remove existing build artifacts before building
#   --skip-build    Skip Go build, use existing binary
#   --help          Show this help message
#
# Output:
#   Creates canvuslocallm-VERSION-linux-amd64.tar.gz containing:
#   - canvuslocallm (binary)
#   - install.sh (installer script)
#   - canvuslocallm.service (systemd unit)
#   - .env.example (configuration template)
#   - README.txt (quick start guide)
#   - LICENSE.txt (license file, if exists)

set -euo pipefail

# Constants
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
readonly BINARY_NAME="canvuslocallm"
readonly SERVICE_NAME="canvuslocallm"

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
    sed -n '2,22p' "$0" | sed 's/^# //' | sed 's/^#//'
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

    # Check for tar
    if ! command -v tar &> /dev/null; then
        missing+=("tar")
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

    if [[ ! -f "$PROJECT_ROOT/installer/tarball/install.sh" ]]; then
        missing+=("installer/tarball/install.sh")
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

create_staging_dir() {
    log_info "Creating staging directory..."

    local staging_dir="$OUTPUT_DIR/staging"

    if [[ "$CLEAN_BUILD" == "true" && -d "$staging_dir" ]]; then
        log_info "Cleaning existing staging directory..."
        rm -rf "$staging_dir"
    fi

    mkdir -p "$staging_dir"

    # Return the staging directory path via stdout
    echo "$staging_dir"
}

copy_files_to_staging() {
    local staging_dir="$1"

    log_info "Copying files to staging directory..."

    # Copy binary
    cp "$PROJECT_ROOT/$BINARY_NAME" "$staging_dir/"
    chmod 755 "$staging_dir/$BINARY_NAME"
    log_success "Copied $BINARY_NAME"

    # Copy install script
    cp "$PROJECT_ROOT/installer/tarball/install.sh" "$staging_dir/"
    chmod 755 "$staging_dir/install.sh"
    log_success "Copied install.sh"

    # Copy systemd service file
    cp "$PROJECT_ROOT/installer/linux/canvuslocallm.service" "$staging_dir/"
    chmod 644 "$staging_dir/canvuslocallm.service"
    log_success "Copied canvuslocallm.service"

    # Copy example.env as .env.example
    cp "$PROJECT_ROOT/example.env" "$staging_dir/.env.example"
    chmod 644 "$staging_dir/.env.example"
    log_success "Copied .env.example"

    # Copy LICENSE if exists
    if [[ -f "$PROJECT_ROOT/LICENSE" ]]; then
        cp "$PROJECT_ROOT/LICENSE" "$staging_dir/LICENSE.txt"
        chmod 644 "$staging_dir/LICENSE.txt"
        log_success "Copied LICENSE.txt"
    elif [[ -f "$PROJECT_ROOT/LICENSE.txt" ]]; then
        cp "$PROJECT_ROOT/LICENSE.txt" "$staging_dir/"
        chmod 644 "$staging_dir/LICENSE.txt"
        log_success "Copied LICENSE.txt"
    else
        log_warn "No LICENSE file found, skipping"
    fi

    # Create README.txt for tarball
    create_readme "$staging_dir"
}

create_readme() {
    local staging_dir="$1"

    log_info "Creating README.txt..."

    cat > "$staging_dir/README.txt" << 'EOF'
CanvusLocalLLM - Canvus AI Integration Service
===============================================

Quick Start
-----------

1. Install the application:

   # System-wide installation (recommended)
   sudo ./install.sh

   # User-local installation (no root required)
   ./install.sh --user

2. Configure the application:

   # Edit the configuration file
   sudo nano /opt/canvuslocallm/.env

   At minimum, set these values:
   - CANVUS_SERVER: Your Canvus server URL
   - CANVUS_API_KEY: Your Canvus API key
   - CANVAS_ID: The canvas ID to monitor
   - OPENAI_API_KEY: Your OpenAI API key (or configure local LLM)

3. Start the service:

   sudo systemctl start canvuslocallm
   sudo systemctl status canvuslocallm

4. View logs:

   sudo journalctl -u canvuslocallm -f

For more information:
- Documentation: https://github.com/jaypaulb/CanvusLocalLLM
- Issues: https://github.com/jaypaulb/CanvusLocalLLM/issues

EOF

    chmod 644 "$staging_dir/README.txt"
    log_success "Created README.txt"
}

create_tarball() {
    local staging_dir="$1"
    local tarball_name="canvuslocallm-${VERSION}-linux-amd64.tar.gz"
    local tarball_path="$OUTPUT_DIR/$tarball_name"

    log_info "Creating tarball: $tarball_name"

    # Create output directory if needed
    mkdir -p "$OUTPUT_DIR"

    # Remove existing tarball if present
    if [[ -f "$tarball_path" ]]; then
        rm -f "$tarball_path"
    fi

    # Create tarball with staging contents
    # Use --transform to set the root directory name
    tar -czf "$tarball_path" \
        --transform "s,^staging,canvuslocallm-${VERSION}," \
        -C "$OUTPUT_DIR" \
        staging

    if [[ ! -f "$tarball_path" ]]; then
        log_error "Failed to create tarball"
        exit 1
    fi

    local tarball_size
    tarball_size=$(ls -lh "$tarball_path" | awk '{print $5}')
    log_success "Created tarball: $tarball_path ($tarball_size)"

    # Return the tarball path via stdout
    echo "$tarball_path"
}

cleanup_staging() {
    local staging_dir="$1"

    log_info "Cleaning up staging directory..."
    rm -rf "$staging_dir"
    log_success "Cleanup complete"
}

verify_tarball() {
    local tarball_path="$1"

    log_info "Verifying tarball contents..."

    echo "" >&2
    echo "Tarball contents:" >&2
    echo "-----------------" >&2
    tar -tzf "$tarball_path" | sed 's/^/  /' >&2
    echo "" >&2

    # Verify essential files are present
    local required_files=(
        "canvuslocallm-${VERSION}/$BINARY_NAME"
        "canvuslocallm-${VERSION}/install.sh"
        "canvuslocallm-${VERSION}/canvuslocallm.service"
        "canvuslocallm-${VERSION}/.env.example"
        "canvuslocallm-${VERSION}/README.txt"
    )

    local missing=()
    for file in "${required_files[@]}"; do
        if ! tar -tzf "$tarball_path" | grep -q "^$file$"; then
            missing+=("$file")
        fi
    done

    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "Tarball missing required files: ${missing[*]}"
        exit 1
    fi

    log_success "Tarball verification passed"
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
    echo "  CanvusLocalLLM Tarball Builder" >&2
    echo "  Version: $VERSION" >&2
    echo "========================================" >&2
    echo "" >&2

    check_dependencies
    validate_source_files
    build_binary

    local staging_dir
    staging_dir=$(create_staging_dir)

    copy_files_to_staging "$staging_dir"

    local tarball_path
    tarball_path=$(create_tarball "$staging_dir")

    verify_tarball "$tarball_path"

    cleanup_staging "$staging_dir"

    echo "" >&2
    log_success "Build complete!"
    echo "" >&2
    echo "Output: $tarball_path" >&2
    echo "" >&2
    echo "To test the tarball:" >&2
    echo "  mkdir /tmp/test-install" >&2
    echo "  tar -xzf $tarball_path -C /tmp/test-install" >&2
    echo "  cd /tmp/test-install/canvuslocallm-${VERSION}" >&2
    echo "  ./install.sh --help" >&2
    echo "" >&2
}

main "$@"
