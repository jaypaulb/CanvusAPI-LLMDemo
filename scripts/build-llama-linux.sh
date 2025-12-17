#!/bin/bash
# build-llama-linux.sh
# Atom: Wrapper for build-llamacpp-cuda.sh for Linux CUDA builds
# Purpose: Alias script for consistent naming convention
#
# Usage:
#   ./scripts/build-llama-linux.sh [OPTIONS]
#
# This is a thin wrapper around build-llamacpp-cuda.sh.
# All options are passed through. See build-llamacpp-cuda.sh --help for details.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Execute the main build script with all arguments
exec "$SCRIPT_DIR/build-llamacpp-cuda.sh" "$@"
