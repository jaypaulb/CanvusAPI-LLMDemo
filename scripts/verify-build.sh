#!/bin/bash
# verify-build.sh
# Molecule: Build verification and CUDA compatibility check
# Purpose: Verify that llama.cpp is properly built and CUDA is available
#
# Usage:
#   ./scripts/verify-build.sh [OPTIONS]
#
# Options:
#   --check-cuda    Check CUDA availability and GPU information
#   --check-libs    Verify llama.cpp libraries are installed
#   --check-all     Run all checks (default)
#   --verbose       Show detailed output
#   --json          Output results as JSON
#   --help          Show this help message
#
# Exit codes:
#   0 - All checks passed
#   1 - One or more checks failed
#   2 - Invalid arguments

set -euo pipefail

# Script configuration
readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
readonly LIB_DIR="$PROJECT_ROOT/lib"
readonly DEPS_DIR="$PROJECT_ROOT/deps"
readonly LLAMACPP_DIR="$DEPS_DIR/llama.cpp"

# Options
CHECK_CUDA=false
CHECK_LIBS=false
CHECK_ALL=true
VERBOSE=false
JSON_OUTPUT=false

# Results tracking
declare -A RESULTS
OVERALL_SUCCESS=true

# Colors (disabled if not a terminal)
if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[1;33m'
    BLUE='\033[0;34m'
    NC='\033[0m'
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

log_info() {
    if [[ "$JSON_OUTPUT" != "true" ]]; then
        echo -e "${BLUE}[INFO]${NC} $1"
    fi
}

log_success() {
    if [[ "$JSON_OUTPUT" != "true" ]]; then
        echo -e "${GREEN}[PASS]${NC} $1"
    fi
}

log_warn() {
    if [[ "$JSON_OUTPUT" != "true" ]]; then
        echo -e "${YELLOW}[WARN]${NC} $1"
    fi
}

log_error() {
    if [[ "$JSON_OUTPUT" != "true" ]]; then
        echo -e "${RED}[FAIL]${NC} $1"
    fi
}

log_detail() {
    if [[ "$VERBOSE" == "true" && "$JSON_OUTPUT" != "true" ]]; then
        echo -e "       $1"
    fi
}

show_help() {
    sed -n '2,20p' "$0" | sed 's/^# //' | sed 's/^#//'
    exit 0
}

record_result() {
    local check_name="$1"
    local passed="$2"
    local message="${3:-}"

    RESULTS["$check_name"]="$passed"

    if [[ "$passed" == "false" ]]; then
        OVERALL_SUCCESS=false
    fi
}

# CUDA Checks
check_nvidia_driver() {
    log_info "Checking NVIDIA driver..."

    if command -v nvidia-smi &> /dev/null; then
        local driver_version
        driver_version=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader 2>/dev/null | head -1)

        if [[ -n "$driver_version" ]]; then
            log_success "NVIDIA driver version: $driver_version"
            record_result "nvidia_driver" "true"

            # Get GPU info
            if [[ "$VERBOSE" == "true" ]]; then
                local gpu_name gpu_memory
                gpu_name=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1)
                gpu_memory=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader 2>/dev/null | head -1)
                log_detail "GPU: $gpu_name"
                log_detail "VRAM: $gpu_memory"
            fi
            return 0
        fi
    fi

    log_error "NVIDIA driver not found or not working"
    record_result "nvidia_driver" "false"
    return 1
}

check_cuda_toolkit() {
    log_info "Checking CUDA toolkit..."

    if command -v nvcc &> /dev/null; then
        local cuda_version
        cuda_version=$(nvcc --version 2>/dev/null | grep "release" | sed 's/.*release //' | sed 's/,.*//')

        if [[ -n "$cuda_version" ]]; then
            log_success "CUDA toolkit version: $cuda_version"
            record_result "cuda_toolkit" "true"

            if [[ "$VERBOSE" == "true" ]]; then
                log_detail "nvcc path: $(command -v nvcc)"
            fi
            return 0
        fi
    fi

    log_warn "CUDA toolkit (nvcc) not found - GPU acceleration may not be available"
    record_result "cuda_toolkit" "false"
    return 1
}

check_cuda_compute_capability() {
    log_info "Checking CUDA compute capability..."

    if command -v nvidia-smi &> /dev/null; then
        # Get compute capability from nvidia-smi
        local compute_cap
        compute_cap=$(nvidia-smi --query-gpu=compute_cap --format=csv,noheader 2>/dev/null | head -1)

        if [[ -n "$compute_cap" ]]; then
            log_success "Compute capability: $compute_cap"
            record_result "compute_capability" "true"

            # Check minimum requirement (6.1 for Pascal)
            local major minor
            major=$(echo "$compute_cap" | cut -d. -f1)
            minor=$(echo "$compute_cap" | cut -d. -f2)

            if [[ "$major" -lt 6 ]] || [[ "$major" -eq 6 && "$minor" -lt 1 ]]; then
                log_warn "GPU compute capability $compute_cap may have limited support"
                log_warn "Recommended: 6.1 (Pascal) or higher for best performance"
            fi
            return 0
        fi
    fi

    log_warn "Could not determine GPU compute capability"
    record_result "compute_capability" "false"
    return 1
}

check_gpu_info() {
    log_info "GPU Information:"

    if command -v nvidia-smi &> /dev/null; then
        local gpu_name gpu_memory gpu_util mem_util

        gpu_name=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1)
        gpu_memory=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader 2>/dev/null | head -1)
        gpu_util=$(nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader 2>/dev/null | head -1)
        mem_util=$(nvidia-smi --query-gpu=memory.used --format=csv,noheader 2>/dev/null | head -1)

        echo ""
        echo "  GPU Name:        $gpu_name"
        echo "  Total VRAM:      $gpu_memory"
        echo "  GPU Utilization: $gpu_util"
        echo "  Memory Used:     $mem_util"
        echo ""

        record_result "gpu_info" "true"
        return 0
    fi

    log_warn "Could not retrieve GPU information"
    record_result "gpu_info" "false"
    return 1
}

# Library Checks
check_libllama() {
    log_info "Checking libllama library..."

    local found=false

    # Check in project lib/ directory
    if [[ -f "$LIB_DIR/libllama.so" ]]; then
        log_success "Found: $LIB_DIR/libllama.so"
        found=true

        if [[ "$VERBOSE" == "true" ]]; then
            local size
            size=$(ls -lh "$LIB_DIR/libllama.so" | awk '{print $5}')
            log_detail "Size: $size"
        fi
    elif [[ -f "$LIB_DIR/llama.dll" ]]; then
        log_success "Found: $LIB_DIR/llama.dll"
        found=true
    elif [[ -f "$LIB_DIR/libllama.dylib" ]]; then
        log_success "Found: $LIB_DIR/libllama.dylib"
        found=true
    fi

    # Check in build directory
    if [[ "$found" != "true" ]]; then
        local build_lib
        build_lib=$(find "$LLAMACPP_DIR/build" -name "libllama.so*" -o -name "llama.dll" -o -name "libllama.dylib*" 2>/dev/null | head -1)

        if [[ -n "$build_lib" ]]; then
            log_warn "Found in build dir but not installed: $build_lib"
            log_warn "Run: ./scripts/build-llamacpp-cuda.sh to install to lib/"
            found=true
        fi
    fi

    if [[ "$found" == "true" ]]; then
        record_result "libllama" "true"
        return 0
    fi

    log_error "libllama not found"
    log_error "Run: ./scripts/build-llamacpp-cuda.sh to build and install"
    record_result "libllama" "false"
    return 1
}

check_ggml_libs() {
    log_info "Checking GGML libraries..."

    local found_count=0

    # Check for GGML libraries (ggml, ggml-base, ggml-cuda, etc.)
    for lib in "$LIB_DIR"/libggml*.so* "$LIB_DIR"/ggml*.dll "$LIB_DIR"/libggml*.dylib*; do
        if [[ -f "$lib" ]]; then
            ((found_count++)) || true
            if [[ "$VERBOSE" == "true" ]]; then
                log_detail "Found: $(basename "$lib")"
            fi
        fi
    done

    if [[ $found_count -gt 0 ]]; then
        log_success "Found $found_count GGML library file(s)"
        record_result "ggml_libs" "true"
        return 0
    fi

    log_warn "GGML libraries not found (may be statically linked)"
    record_result "ggml_libs" "unknown"
    return 0  # Not a failure, might be static
}

check_llamacpp_headers() {
    log_info "Checking llama.cpp headers..."

    local header_path="$LLAMACPP_DIR/include/llama.h"
    local alt_header_path="$LLAMACPP_DIR/llama.h"

    if [[ -f "$header_path" ]]; then
        log_success "Found headers at: $LLAMACPP_DIR/include/"
        record_result "llamacpp_headers" "true"
        return 0
    elif [[ -f "$alt_header_path" ]]; then
        log_success "Found headers at: $LLAMACPP_DIR/"
        record_result "llamacpp_headers" "true"
        return 0
    fi

    log_error "llama.cpp headers not found"
    log_error "Run: ./scripts/build-llamacpp-cuda.sh to clone repository"
    record_result "llamacpp_headers" "false"
    return 1
}

check_llama_server() {
    log_info "Checking llama-server binary..."

    local server_paths=(
        "$LLAMACPP_DIR/build/bin/llama-server"
        "$LLAMACPP_DIR/build/bin/server"
        "$LLAMACPP_DIR/build/Release/llama-server.exe"
        "$LLAMACPP_DIR/build/bin/Release/llama-server.exe"
    )

    for server_path in "${server_paths[@]}"; do
        if [[ -f "$server_path" ]]; then
            log_success "Found: $server_path"
            record_result "llama_server" "true"

            if [[ "$VERBOSE" == "true" ]]; then
                local version
                version=$("$server_path" --version 2>&1 | head -1) || true
                if [[ -n "$version" ]]; then
                    log_detail "Version: $version"
                fi
            fi
            return 0
        fi
    done

    log_warn "llama-server binary not found"
    log_warn "This is optional if using CGo bindings directly"
    record_result "llama_server" "unknown"
    return 0
}

# Go Build Check
check_go_build() {
    log_info "Checking Go build capability..."

    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        record_result "go_build" "false"
        return 1
    fi

    local go_version
    go_version=$(go version | sed 's/go version //')
    log_detail "Go version: $go_version"

    # Try to build with CGo
    cd "$PROJECT_ROOT"

    if go build -o /dev/null ./llamaruntime 2>/dev/null; then
        log_success "Go build with CGo successful"
        record_result "go_build" "true"
        return 0
    else
        log_error "Go build failed - check library paths"
        log_error "Ensure llama.cpp is built and libraries are in lib/"
        record_result "go_build" "false"
        return 1
    fi
}

# Output JSON results
output_json() {
    echo "{"
    echo "  \"success\": $([[ "$OVERALL_SUCCESS" == "true" ]] && echo "true" || echo "false"),"
    echo "  \"checks\": {"

    local first=true
    for key in "${!RESULTS[@]}"; do
        if [[ "$first" != "true" ]]; then
            echo ","
        fi
        first=false
        local value="${RESULTS[$key]}"
        echo -n "    \"$key\": \"$value\""
    done
    echo ""

    echo "  },"

    # Add system info
    echo "  \"system\": {"

    # GPU info
    if command -v nvidia-smi &> /dev/null; then
        local gpu_name driver_ver cuda_ver compute_cap vram
        gpu_name=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1 | tr -d '\n')
        driver_ver=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader 2>/dev/null | head -1 | tr -d '\n')
        compute_cap=$(nvidia-smi --query-gpu=compute_cap --format=csv,noheader 2>/dev/null | head -1 | tr -d '\n')
        vram=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader 2>/dev/null | head -1 | tr -d '\n')

        echo "    \"gpu_name\": \"$gpu_name\","
        echo "    \"driver_version\": \"$driver_ver\","
        echo "    \"compute_capability\": \"$compute_cap\","
        echo "    \"vram\": \"$vram\","
    fi

    # CUDA version
    if command -v nvcc &> /dev/null; then
        cuda_ver=$(nvcc --version 2>/dev/null | grep "release" | sed 's/.*release //' | sed 's/,.*//' | tr -d '\n')
        echo "    \"cuda_version\": \"$cuda_ver\","
    fi

    echo "    \"platform\": \"$(uname -s)\","
    echo "    \"arch\": \"$(uname -m)\""
    echo "  }"
    echo "}"
}

# Main execution
main() {
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --check-cuda)
                CHECK_CUDA=true
                CHECK_ALL=false
                shift
                ;;
            --check-libs)
                CHECK_LIBS=true
                CHECK_ALL=false
                shift
                ;;
            --check-all)
                CHECK_ALL=true
                shift
                ;;
            --verbose|-v)
                VERBOSE=true
                shift
                ;;
            --json)
                JSON_OUTPUT=true
                shift
                ;;
            --help|-h)
                show_help
                ;;
            *)
                echo "Unknown option: $1"
                show_help
                exit 2
                ;;
        esac
    done

    if [[ "$JSON_OUTPUT" != "true" ]]; then
        echo ""
        echo "========================================"
        echo "  Build Verification"
        echo "========================================"
        echo ""
    fi

    # Run CUDA checks
    if [[ "$CHECK_ALL" == "true" || "$CHECK_CUDA" == "true" ]]; then
        check_nvidia_driver || true
        check_cuda_toolkit || true
        check_cuda_compute_capability || true

        if [[ "$VERBOSE" == "true" ]]; then
            check_gpu_info || true
        fi
    fi

    # Run library checks
    if [[ "$CHECK_ALL" == "true" || "$CHECK_LIBS" == "true" ]]; then
        check_llamacpp_headers || true
        check_libllama || true
        check_ggml_libs || true
        check_llama_server || true
    fi

    # Run Go build check
    if [[ "$CHECK_ALL" == "true" ]]; then
        check_go_build || true
    fi

    # Output results
    if [[ "$JSON_OUTPUT" == "true" ]]; then
        output_json
    else
        echo ""
        echo "========================================"
        if [[ "$OVERALL_SUCCESS" == "true" ]]; then
            log_success "All checks passed!"
        else
            log_error "Some checks failed. Review output above."
        fi
        echo "========================================"
        echo ""
    fi

    # Exit with appropriate code
    if [[ "$OVERALL_SUCCESS" == "true" ]]; then
        exit 0
    else
        exit 1
    fi
}

main "$@"
