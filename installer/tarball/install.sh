#!/bin/bash
# CanvusLocalLLM Tarball Installer
# Installs CanvusLocalLLM from extracted tarball
set -e

# ==============================================================================
# Configuration
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION="1.0.0"
SERVICE_NAME="canvuslocallm"
SERVICE_USER="canvusllm"
SERVICE_GROUP="canvusllm"
BINARY_NAME="canvuslocallm"

# Defaults (will be set based on install type)
PREFIX=""
USER_INSTALL=false
INSTALL_SERVICE=false
FORCE=false

# ==============================================================================
# Color Output Functions
# ==============================================================================

# Check if terminal supports colors
if [[ -t 1 ]] && [[ -n "$TERM" ]] && command -v tput &>/dev/null; then
    COLORS_SUPPORTED=true
else
    COLORS_SUPPORTED=false
fi

print_green() {
    if $COLORS_SUPPORTED; then
        echo -e "\033[0;32m$1\033[0m"
    else
        echo "$1"
    fi
}

print_yellow() {
    if $COLORS_SUPPORTED; then
        echo -e "\033[0;33m$1\033[0m"
    else
        echo "$1"
    fi
}

print_red() {
    if $COLORS_SUPPORTED; then
        echo -e "\033[0;31m$1\033[0m"
    else
        echo "$1"
    fi
}

print_blue() {
    if $COLORS_SUPPORTED; then
        echo -e "\033[0;34m$1\033[0m"
    else
        echo "$1"
    fi
}

# ==============================================================================
# Helper Functions
# ==============================================================================

show_help() {
    cat << EOF
CanvusLocalLLM Installer v${VERSION}

Usage: $(basename "$0") [OPTIONS]

Options:
  --prefix=PATH    Install to PATH (default: /opt/canvuslocallm for system,
                   ~/.local/canvuslocallm for user install)
  --user           Install for current user only (no sudo required)
  --force          Overwrite existing installation without prompting
  --help           Show this help message

Examples:
  # System-wide installation (requires root/sudo)
  sudo ./install.sh

  # User-local installation
  ./install.sh --user

  # Custom prefix
  sudo ./install.sh --prefix=/usr/local/canvuslocallm

  # User install with custom prefix
  ./install.sh --user --prefix=~/apps/canvuslocallm

EOF
}

error_exit() {
    print_red "ERROR: $1"
    exit 1
}

warn() {
    print_yellow "WARNING: $1"
}

info() {
    print_blue "INFO: $1"
}

success() {
    print_green "OK: $1"
}

# Check if running as root or with sudo
is_root() {
    [[ $EUID -eq 0 ]]
}

# Check if sudo is available
has_sudo() {
    command -v sudo &>/dev/null
}

# Check if systemd is available
has_systemd() {
    command -v systemctl &>/dev/null && [[ -d /run/systemd/system ]]
}

# Prompt user for yes/no
prompt_yes_no() {
    local prompt="$1"
    local default="${2:-n}"
    local response

    if [[ "$default" == "y" ]]; then
        prompt="$prompt [Y/n]: "
    else
        prompt="$prompt [y/N]: "
    fi

    read -r -p "$prompt" response
    response="${response:-$default}"

    [[ "$response" =~ ^[Yy]$ ]]
}

# ==============================================================================
# Validation Functions
# ==============================================================================

validate_prerequisites() {
    info "Validating prerequisites..."

    # Check for required source files
    local required_files=("$BINARY_NAME")
    local missing_files=()

    for file in "${required_files[@]}"; do
        if [[ ! -f "$SCRIPT_DIR/$file" ]]; then
            missing_files+=("$file")
        fi
    done

    if [[ ${#missing_files[@]} -gt 0 ]]; then
        error_exit "Missing required files: ${missing_files[*]}
Please ensure you're running the installer from the extracted tarball directory."
    fi

    # Check binary is executable or can be made executable
    if [[ ! -x "$SCRIPT_DIR/$BINARY_NAME" ]]; then
        if [[ -f "$SCRIPT_DIR/$BINARY_NAME" ]]; then
            chmod +x "$SCRIPT_DIR/$BINARY_NAME" 2>/dev/null || \
                error_exit "Cannot make $BINARY_NAME executable. Check file permissions."
        fi
    fi

    # For system install, verify we have root access
    if ! $USER_INSTALL && ! is_root; then
        error_exit "System installation requires root privileges.
Run with 'sudo $0' or use '--user' for user-local installation."
    fi

    # For user install, ensure we're NOT root (unless explicitly using --prefix)
    if $USER_INSTALL && is_root && [[ -z "$PREFIX" ]]; then
        warn "Running user install as root. Files will be owned by root."
    fi

    success "Prerequisites validated"
}

validate_prefix() {
    # Expand ~ if present
    PREFIX="${PREFIX/#\~/$HOME}"

    # Create parent directory if needed
    local parent_dir
    parent_dir="$(dirname "$PREFIX")"

    if [[ ! -d "$parent_dir" ]]; then
        if ! mkdir -p "$parent_dir" 2>/dev/null; then
            error_exit "Cannot create parent directory: $parent_dir"
        fi
    fi

    # Check if prefix already exists
    if [[ -d "$PREFIX" ]]; then
        if [[ -f "$PREFIX/$BINARY_NAME" ]] && ! $FORCE; then
            print_yellow "Existing installation found at $PREFIX"
            if ! prompt_yes_no "Overwrite existing installation?" "n"; then
                error_exit "Installation cancelled by user"
            fi
        fi
    fi

    success "Installation prefix validated: $PREFIX"
}

# ==============================================================================
# Installation Functions
# ==============================================================================

create_directories() {
    info "Creating installation directories..."

    if [[ ! -d "$PREFIX" ]]; then
        mkdir -p "$PREFIX" || error_exit "Failed to create directory: $PREFIX"
    fi

    # Create subdirectories for logs and downloads
    mkdir -p "$PREFIX/logs" 2>/dev/null || true
    mkdir -p "$PREFIX/downloads" 2>/dev/null || true

    success "Directories created"
}

copy_files() {
    info "Copying files to $PREFIX..."

    # Copy binary
    cp "$SCRIPT_DIR/$BINARY_NAME" "$PREFIX/" || \
        error_exit "Failed to copy binary"
    success "Copied $BINARY_NAME"

    # Copy .env.example if exists
    if [[ -f "$SCRIPT_DIR/.env.example" ]]; then
        cp "$SCRIPT_DIR/.env.example" "$PREFIX/" || \
            error_exit "Failed to copy .env.example"
        success "Copied .env.example"

        # Create .env from example if it doesn't exist
        if [[ ! -f "$PREFIX/.env" ]]; then
            cp "$PREFIX/.env.example" "$PREFIX/.env"
            success "Created .env configuration file"
        else
            info ".env already exists, not overwriting"
        fi
    else
        warn ".env.example not found in tarball"
    fi

    # Copy LICENSE.txt if exists
    if [[ -f "$SCRIPT_DIR/LICENSE.txt" ]]; then
        cp "$SCRIPT_DIR/LICENSE.txt" "$PREFIX/" || \
            warn "Failed to copy LICENSE.txt"
        success "Copied LICENSE.txt"
    fi

    # Copy README.txt if exists
    if [[ -f "$SCRIPT_DIR/README.txt" ]]; then
        cp "$SCRIPT_DIR/README.txt" "$PREFIX/" || \
            warn "Failed to copy README.txt"
        success "Copied README.txt"
    fi

    # Copy systemd service file if exists (for system installs)
    if [[ -f "$SCRIPT_DIR/$SERVICE_NAME.service" ]] && ! $USER_INSTALL; then
        cp "$SCRIPT_DIR/$SERVICE_NAME.service" "$PREFIX/" || \
            warn "Failed to copy systemd service file"
        success "Copied systemd service file"
    fi
}

set_permissions() {
    info "Setting file permissions..."

    # Make binary executable
    chmod 755 "$PREFIX/$BINARY_NAME" || error_exit "Failed to set binary permissions"

    if $USER_INSTALL; then
        # User install: standard permissions
        chmod 700 "$PREFIX"
        [[ -f "$PREFIX/.env" ]] && chmod 600 "$PREFIX/.env"
        [[ -f "$PREFIX/.env.example" ]] && chmod 644 "$PREFIX/.env.example"
        [[ -d "$PREFIX/logs" ]] && chmod 700 "$PREFIX/logs"
        [[ -d "$PREFIX/downloads" ]] && chmod 700 "$PREFIX/downloads"
    else
        # System install: create user/group and set ownership
        create_service_user

        chown -R "$SERVICE_USER:$SERVICE_GROUP" "$PREFIX"
        chmod 750 "$PREFIX"
        [[ -f "$PREFIX/.env" ]] && chmod 640 "$PREFIX/.env"
        [[ -f "$PREFIX/.env.example" ]] && chmod 644 "$PREFIX/.env.example"
        [[ -d "$PREFIX/logs" ]] && chmod 750 "$PREFIX/logs"
        [[ -d "$PREFIX/downloads" ]] && chmod 750 "$PREFIX/downloads"
    fi

    success "Permissions set"
}

create_service_user() {
    # Only for system installs
    if $USER_INSTALL; then
        return 0
    fi

    info "Setting up service user..."

    # Create service group if not exists
    if ! getent group "$SERVICE_GROUP" > /dev/null 2>&1; then
        groupadd --system "$SERVICE_GROUP" || \
            error_exit "Failed to create group: $SERVICE_GROUP"
        success "Created system group: $SERVICE_GROUP"
    else
        info "Group $SERVICE_GROUP already exists"
    fi

    # Create service user if not exists
    if ! getent passwd "$SERVICE_USER" > /dev/null 2>&1; then
        useradd --system \
            --gid "$SERVICE_GROUP" \
            --home-dir "$PREFIX" \
            --shell /usr/sbin/nologin \
            --comment "CanvusLocalLLM Service Account" \
            "$SERVICE_USER" || error_exit "Failed to create user: $SERVICE_USER"
        success "Created system user: $SERVICE_USER"
    else
        info "User $SERVICE_USER already exists"
    fi
}

install_systemd_service() {
    # Only for system installs with systemd
    if $USER_INSTALL; then
        return 0
    fi

    if ! has_systemd; then
        warn "systemd not detected, skipping service installation"
        return 0
    fi

    local service_file="$PREFIX/$SERVICE_NAME.service"

    if [[ ! -f "$service_file" ]]; then
        # Try to find it in script directory
        if [[ -f "$SCRIPT_DIR/$SERVICE_NAME.service" ]]; then
            service_file="$SCRIPT_DIR/$SERVICE_NAME.service"
        else
            warn "Systemd service file not found, skipping service installation"
            return 0
        fi
    fi

    echo ""
    print_blue "=========================================="
    print_blue "Systemd Service Installation"
    print_blue "=========================================="
    echo ""
    echo "Would you like to install and enable the systemd service?"
    echo "This will:"
    echo "  - Copy the service file to /etc/systemd/system/"
    echo "  - Enable the service to start on boot"
    echo "  - NOT start the service (you need to configure .env first)"
    echo ""

    if ! prompt_yes_no "Install systemd service?" "y"; then
        info "Skipping systemd service installation"
        return 0
    fi

    INSTALL_SERVICE=true

    info "Installing systemd service..."

    # Update service file paths if prefix is not default
    local temp_service="/tmp/$SERVICE_NAME.service.$$"
    if [[ "$PREFIX" != "/opt/canvuslocallm" ]]; then
        # Adjust paths in service file for custom prefix
        sed -e "s|/opt/canvuslocallm|$PREFIX|g" "$service_file" > "$temp_service"
        service_file="$temp_service"
    fi

    # Copy service file to systemd directory
    cp "$service_file" "/etc/systemd/system/$SERVICE_NAME.service" || \
        error_exit "Failed to install systemd service file"

    # Clean up temp file if created
    [[ -f "$temp_service" ]] && rm -f "$temp_service"

    # Set correct permissions on service file
    chmod 644 "/etc/systemd/system/$SERVICE_NAME.service"

    # Reload systemd daemon
    systemctl daemon-reload || error_exit "Failed to reload systemd daemon"

    # Enable the service
    systemctl enable "$SERVICE_NAME.service" || \
        error_exit "Failed to enable systemd service"

    success "Systemd service installed and enabled"
}

# ==============================================================================
# Main Script
# ==============================================================================

main() {
    echo ""
    print_green "=========================================="
    print_green "  CanvusLocalLLM Installer v${VERSION}"
    print_green "=========================================="
    echo ""

    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            --prefix=*)
                PREFIX="${1#*=}"
                ;;
            --prefix)
                shift
                PREFIX="$1"
                ;;
            --user)
                USER_INSTALL=true
                ;;
            --force)
                FORCE=true
                ;;
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                error_exit "Unknown option: $1
Use --help for usage information."
                ;;
        esac
        shift
    done

    # Detect installation type and set defaults
    if [[ -z "$PREFIX" ]]; then
        if $USER_INSTALL; then
            PREFIX="$HOME/.local/canvuslocallm"
            info "User installation mode"
        else
            PREFIX="/opt/canvuslocallm"
            info "System installation mode"
        fi
    fi

    info "Installation prefix: $PREFIX"
    echo ""

    # Run installation steps
    validate_prerequisites
    validate_prefix
    create_directories
    copy_files
    set_permissions
    install_systemd_service

    # Print success message and next steps
    echo ""
    print_green "=========================================="
    print_green "  Installation Complete!"
    print_green "=========================================="
    echo ""
    print_blue "Installation location: $PREFIX"
    echo ""
    print_yellow "Next steps:"
    echo ""
    echo "  1. Configure the application:"
    if $USER_INSTALL; then
        echo "     nano $PREFIX/.env"
    else
        echo "     sudo nano $PREFIX/.env"
    fi
    echo ""
    echo "  2. Set your Canvus server URL, API key, and canvas ID"
    echo ""

    if $USER_INSTALL; then
        echo "  3. Run the application manually:"
        echo "     cd $PREFIX && ./$BINARY_NAME"
        echo ""
        echo "  Tip: Add $PREFIX to your PATH for easier access:"
        echo "     export PATH=\"\$PATH:$PREFIX\""
    else
        if $INSTALL_SERVICE; then
            echo "  3. Start the service:"
            echo "     sudo systemctl start $SERVICE_NAME"
            echo ""
            echo "  4. Check service status:"
            echo "     sudo systemctl status $SERVICE_NAME"
            echo ""
            echo "  5. View logs:"
            echo "     sudo journalctl -u $SERVICE_NAME -f"
        else
            echo "  3. Run the application:"
            echo "     cd $PREFIX && sudo -u $SERVICE_USER ./$BINARY_NAME"
            echo ""
            echo "  Or install the systemd service manually:"
            if [[ -f "$PREFIX/$SERVICE_NAME.service" ]]; then
                echo "     sudo cp $PREFIX/$SERVICE_NAME.service /etc/systemd/system/"
            else
                echo "     (service file not found in installation)"
            fi
            echo "     sudo systemctl daemon-reload"
            echo "     sudo systemctl enable $SERVICE_NAME"
            echo "     sudo systemctl start $SERVICE_NAME"
        fi
    fi

    echo ""
    if [[ -f "$PREFIX/README.txt" ]]; then
        echo "  For more information, see: $PREFIX/README.txt"
    fi
    echo ""
    print_green "Thank you for installing CanvusLocalLLM!"
    echo ""
}

# Run main function with all arguments
main "$@"
