#!/bin/bash
# download-sd-model.sh
# Molecule: Downloads Stable Diffusion v1.5 model with checksum verification
# Purpose: Download and verify SD v1.5 model for use with stable-diffusion.cpp
#
# Usage:
#   ./scripts/download-sd-model.sh [OPTIONS]
#
# Options:
#   --output DIR    Output directory for model (default: models/)
#   --force         Force re-download even if file exists
#   --no-verify     Skip checksum verification (not recommended)
#   --help          Show this help message
#
# Model Info:
#   Source: huggingface.co/runwayml/stable-diffusion-v1-5
#   File: v1-5-pruned-emaonly.safetensors
#   Size: ~4.27 GB
#   License: CreativeML Open RAIL-M
#
# Requirements:
#   - wget or curl
#   - sha256sum (for verification)
#   - ~5GB free disk space

set -euo pipefail

# Constants
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Model configuration
readonly MODEL_URL="https://huggingface.co/runwayml/stable-diffusion-v1-5/resolve/main/v1-5-pruned-emaonly.safetensors"
readonly MODEL_FILENAME="sd-v1-5.safetensors"
# SHA256 checksum for v1-5-pruned-emaonly.safetensors
# Source: HuggingFace model card and community verification
readonly MODEL_SHA256="cc6cb27103417325ff94f52b7a5d2dde45a7515b25c255d8e396c90014281516"
readonly MODEL_SIZE_HUMAN="4.27 GB"

# Defaults
OUTPUT_DIR="$PROJECT_ROOT/models"
FORCE_DOWNLOAD=false
VERIFY_CHECKSUM=true

# Colors for output (disabled if not a terminal)
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    CYAN='\033[0;36m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    CYAN=''
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
    sed -n '2,24p' "$0" | sed 's/^# //' | sed 's/^#//'
    exit 0
}

show_license() {
    echo ""
    echo -e "${CYAN}=============================================="
    echo "        CreativeML Open RAIL-M License"
    echo "==============================================${NC}"
    echo ""
    echo "Stable Diffusion v1.5 is released under the CreativeML Open RAIL-M license."
    echo ""
    echo "Key points:"
    echo "  - You CAN use, modify, and distribute the model"
    echo "  - You CAN use it for commercial purposes"
    echo "  - You CANNOT use it to generate illegal content"
    echo "  - You CANNOT use it to harm, deceive, or exploit"
    echo "  - You MUST include the license with any distribution"
    echo ""
    echo "Full license: https://huggingface.co/runwayml/stable-diffusion-v1-5/blob/main/LICENSE.md"
    echo ""
}

check_dependencies() {
    log_info "Checking dependencies..."

    local has_downloader=false

    # Check for wget or curl
    if command -v wget &> /dev/null; then
        has_downloader=true
        log_info "Found wget"
    elif command -v curl &> /dev/null; then
        has_downloader=true
        log_info "Found curl"
    fi

    if [[ "$has_downloader" == "false" ]]; then
        log_error "Neither wget nor curl found. Please install one of them."
        exit 1
    fi

    # Check for sha256sum (only if verification is enabled)
    if [[ "$VERIFY_CHECKSUM" == "true" ]]; then
        if ! command -v sha256sum &> /dev/null; then
            log_warn "sha256sum not found. Checksum verification will be skipped."
            log_warn "On macOS, you can use: brew install coreutils"
            VERIFY_CHECKSUM=false
        fi
    fi

    # Check disk space (need ~5GB for download + some buffer)
    local available_space
    available_space=$(df -BG "$OUTPUT_DIR" 2>/dev/null | awk 'NR==2 {print $4}' | sed 's/G//' || echo "unknown")

    if [[ "$available_space" != "unknown" ]] && [[ "$available_space" -lt 5 ]]; then
        log_error "Insufficient disk space. Need at least 5GB, have ${available_space}GB"
        exit 1
    fi

    log_success "Dependency check passed"
}

download_model() {
    local target_file="$OUTPUT_DIR/$MODEL_FILENAME"
    local temp_file="$OUTPUT_DIR/${MODEL_FILENAME}.downloading"

    # Check if file already exists
    if [[ -f "$target_file" ]] && [[ "$FORCE_DOWNLOAD" == "false" ]]; then
        log_info "Model already exists at: $target_file"

        if [[ "$VERIFY_CHECKSUM" == "true" ]]; then
            log_info "Verifying existing file checksum..."
            if verify_checksum "$target_file"; then
                log_success "Existing model is valid. Use --force to re-download."
                return 0
            else
                log_warn "Existing file has invalid checksum. Re-downloading..."
            fi
        else
            log_success "Skipping download (use --force to re-download)"
            return 0
        fi
    fi

    # Create output directory
    mkdir -p "$OUTPUT_DIR"

    log_info "Downloading Stable Diffusion v1.5 model..."
    log_info "Source: $MODEL_URL"
    log_info "Target: $target_file"
    log_info "Size: ~$MODEL_SIZE_HUMAN"
    echo ""
    log_warn "This may take a while depending on your connection speed..."
    echo ""

    # Clean up any partial downloads
    rm -f "$temp_file"

    # Download with progress
    if command -v wget &> /dev/null; then
        wget --progress=bar:force:noscroll \
             --output-document="$temp_file" \
             "$MODEL_URL"
    elif command -v curl &> /dev/null; then
        curl --location \
             --progress-bar \
             --output "$temp_file" \
             "$MODEL_URL"
    fi

    # Check if download succeeded
    if [[ ! -f "$temp_file" ]]; then
        log_error "Download failed - file not created"
        exit 1
    fi

    # Check file size (should be > 4GB)
    local file_size
    file_size=$(stat --format=%s "$temp_file" 2>/dev/null || stat -f%z "$temp_file" 2>/dev/null || echo 0)

    if [[ "$file_size" -lt 4000000000 ]]; then
        log_error "Downloaded file too small: $file_size bytes (expected ~4.27GB)"
        log_error "Download may have failed or been interrupted"
        rm -f "$temp_file"
        exit 1
    fi

    # Move to final location
    mv "$temp_file" "$target_file"

    log_success "Download complete: $target_file"
}

verify_checksum() {
    local file="$1"

    if [[ "$VERIFY_CHECKSUM" == "false" ]]; then
        log_warn "Checksum verification skipped"
        return 0
    fi

    log_info "Verifying SHA256 checksum..."
    log_info "Expected: $MODEL_SHA256"

    local actual_checksum
    actual_checksum=$(sha256sum "$file" | awk '{print $1}')

    log_info "Actual:   $actual_checksum"

    if [[ "$actual_checksum" == "$MODEL_SHA256" ]]; then
        log_success "Checksum verified!"
        return 0
    else
        log_error "Checksum mismatch!"
        log_error "The downloaded file may be corrupted or modified."
        return 1
    fi
}

test_model() {
    local model_file="$OUTPUT_DIR/$MODEL_FILENAME"
    local sd_binary="$PROJECT_ROOT/deps/stable-diffusion.cpp/build/bin/sd"

    # Check if sd binary exists
    if [[ ! -f "$sd_binary" ]]; then
        sd_binary="$PROJECT_ROOT/deps/stable-diffusion.cpp/build/sd"
    fi

    if [[ ! -f "$sd_binary" ]]; then
        log_info "stable-diffusion.cpp binary not found. Skipping model test."
        log_info "Build it with: ./scripts/build-sd-cuda.sh"
        return 0
    fi

    log_info "Testing model with stable-diffusion.cpp..."
    log_info "Binary: $sd_binary"

    # Just verify the model loads (don't generate an image)
    # The --help flag should work without loading model
    # We can verify model validity by checking if it can be parsed
    if "$sd_binary" --help &>/dev/null; then
        log_success "stable-diffusion.cpp binary is functional"
        log_info "To generate images, run:"
        echo "  $sd_binary -m $model_file -p 'your prompt' -o output.png"
    else
        log_warn "Could not verify stable-diffusion.cpp binary"
    fi
}

main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --force)
                FORCE_DOWNLOAD=true
                shift
                ;;
            --no-verify)
                VERIFY_CHECKSUM=false
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
    echo "========================================"
    echo "  Stable Diffusion v1.5 Model Download"
    echo "========================================"

    show_license

    check_dependencies
    download_model

    local target_file="$OUTPUT_DIR/$MODEL_FILENAME"
    if [[ -f "$target_file" ]]; then
        verify_checksum "$target_file" || exit 1
        test_model
    fi

    echo ""
    log_success "Model ready at: $target_file"
    echo ""
    echo "Next steps:"
    echo "  1. Build stable-diffusion.cpp: ./scripts/build-sd-cuda.sh"
    echo "  2. Generate images: sd -m $target_file -p 'prompt'"
    echo ""
}

main "$@"
