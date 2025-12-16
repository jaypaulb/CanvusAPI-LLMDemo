# Phase 1: Zero-Config Installation Infrastructure - Specification

**Version**: 0.1.0
**Status**: Draft
**Last Updated**: 2025-12-16

## Table of Contents

1. [Overview](#overview)
2. [Goals and Objectives](#goals-and-objectives)
3. [Architecture](#architecture)
4. [Component Specifications](#component-specifications)
5. [File Structure](#file-structure)
6. [Technical Approach](#technical-approach)
7. [Testing Strategy](#testing-strategy)
8. [Acceptance Criteria](#acceptance-criteria)
9. [Implementation Plan](#implementation-plan)

---

## Overview

Phase 1 establishes the foundational installation infrastructure that enables CanvusLocalLLM to deliver on its core promise: zero-configuration, batteries-included local AI integration. This phase creates native platform installers, minimal configuration templates, first-run automation, and optional service integration.

### Scope

**In Scope**:
- NSIS installer for Windows with wizard interface
- Debian package (.deb) for Ubuntu/Debian distributions
- Tarball distribution (.tar.gz) with install.sh for other Linux distributions
- Minimal .env configuration template (Canvus credentials only)
- First-run model download infrastructure with progress indication
- Startup configuration validation with helpful error messages
- Optional Windows Service and Linux systemd service creation

**Out of Scope**:
- Actual AI inference functionality (Phase 2)
- Image generation capabilities (Phase 3)
- Web UI implementation (Phase 5)
- Code signing and CI/CD automation (Phase 6)

### Key Principles

1. **Zero Configuration for AI**: No model selection, no provider configuration, no inference parameters
2. **Batteries Included**: Bundle all dependencies, download model automatically
3. **Fast to Value**: <10 minutes from download to working AI
4. **Clear Error Messages**: Every failure includes actionable instructions
5. **Platform Native**: Use platform-appropriate installation mechanisms

---

## Goals and Objectives

### Primary Goals

**G-1**: Enable installation by non-technical users with no infrastructure expertise
**G-2**: Reduce configuration surface to absolute minimum (Canvus credentials only)
**G-3**: Provide professional, polished installation experience across Windows and Linux
**G-4**: Establish foundation for embedded AI integration in subsequent phases

### Objectives

**O-1**: Create Windows installer that installs in <5 minutes
**O-2**: Create Linux packages that follow distribution best practices
**O-3**: Eliminate all AI-related configuration from user-facing surface
**O-4**: Validate configuration at startup before heavy operations
**O-5**: Provide optional service integration for production deployments

### Success Criteria

- Non-technical user completes installation in <10 minutes
- Zero support questions about AI models or providers
- 95% installation success rate on first attempt
- Clear, actionable error messages for all failure modes

---

## Architecture

### Installation Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Distribution Layer                       │
├─────────────────────┬─────────────────┬────────────────────┤
│   Windows Installer │  Debian Package │ Tarball + Script   │
│   (NSIS .exe)       │  (.deb)         │ (.tar.gz)          │
└─────────────────────┴─────────────────┴────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Installation Components                   │
├─────────────────────────────────────────────────────────────┤
│  • Application Binary (Go)                                   │
│  • Directory Structure (lib/, models/, downloads/)           │
│  • Configuration Template (.env.example)                     │
│  • Documentation (README.txt)                                │
│  • Service Files (Windows Service / systemd unit)           │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    First-Run Experience                      │
├─────────────────────────────────────────────────────────────┤
│  1. Configuration Validation                                 │
│     ├─ Check .env exists                                     │
│     ├─ Validate Canvus credentials format                    │
│     ├─ Test Canvus server connectivity                       │
│     └─ Verify canvas accessibility                           │
│  2. Model Download (if not bundled)                          │
│     ├─ Download Bunny v1.1 from Hugging Face                 │
│     ├─ Progress indication (console + log)                   │
│     ├─ SHA256 checksum verification                          │
│     └─ Resumable downloads                                   │
│  3. Startup                                                  │
│     ├─ Log successful configuration                          │
│     └─ Begin canvas monitoring (Phase 2+)                    │
└─────────────────────────────────────────────────────────────┘
```

### Component Hierarchy (Atomic Design)

**Atoms** (Pure Functions):
- URL format validation
- File existence checks
- Checksum computation
- Environment variable parsing
- HTTP Range header construction

**Molecules** (Simple Compositions):
- Configuration validator (combines env parsing + validation)
- Download manager (HTTP client + progress tracking + checksum)
- Service installer wrapper (service API + configuration)

**Organisms** (Feature Modules):
- Installation wizard flow (NSIS)
- First-run setup coordinator
- Configuration validation suite

**Pages** (Composition Roots):
- Installer entry points (NSIS script, install.sh)
- Application main.go with startup validation

---

## Component Specifications

### 1. NSIS Installer for Windows

**Component**: CanvusLocalLLM-Setup.exe
**Technology**: NSIS 3.x
**Size**: ~10MB (without model bundled), ~8GB (with model bundled)

#### Installer Flow

```
Welcome Page
    │
    ▼
License Agreement (MIT License)
    │
    ▼
Installation Directory
    ├─ Default: C:\Program Files\CanvusLocalLLM
    └─ User can customize
    │
    ▼
Components Selection
    ├─ [x] Core Application (required)
    ├─ [ ] Install as Windows Service (optional)
    └─ [ ] Desktop Shortcut (optional)
    │
    ▼
Installation Progress
    ├─ Extract application binary
    ├─ Create directory structure
    ├─ Copy configuration template
    ├─ Copy documentation
    ├─ Install Windows Service (if selected)
    └─ Create shortcuts
    │
    ▼
Completion Page
    ├─ "Installation Complete"
    ├─ Link to README.txt
    ├─ Next steps: Configure .env file
    └─ [ ] Launch CanvusLocalLLM now
```

#### NSIS Script Structure

```nsis
; CanvusLocalLLM Installer Script
; NSIS 3.x

!define PRODUCT_NAME "CanvusLocalLLM"
!define PRODUCT_VERSION "0.1.0"
!define PRODUCT_PUBLISHER "CanvusLocalLLM Project"
!define PRODUCT_WEB_SITE "https://github.com/yourusername/CanvusLocalLLM"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"

; Include modern UI
!include "MUI2.nsh"

; General settings
Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"
OutFile "CanvusLocalLLM-Setup.exe"
InstallDir "$PROGRAMFILES\CanvusLocalLLM"
ShowInstDetails show
ShowUnInstDetails show

; Interface settings
!define MUI_ABORTWARNING
!define MUI_ICON "installer\icon.ico"
!define MUI_UNICON "installer\icon.ico"

; Pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "LICENSE.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; Languages
!insertmacro MUI_LANGUAGE "English"

; Installation sections
Section "Core Application" SecCore
    SectionIn RO  ; Required

    SetOutPath "$INSTDIR"
    File "bin\CanvusLocalLLM.exe"
    File ".env.example"
    File "README.txt"
    File "LICENSE.txt"

    CreateDirectory "$INSTDIR\lib"
    CreateDirectory "$INSTDIR\models"
    CreateDirectory "$INSTDIR\downloads"

    ; Create uninstaller
    WriteUninstaller "$INSTDIR\Uninstall.exe"

    ; Registry entries
    WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayName" "${PRODUCT_NAME}"
    WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "UninstallString" "$INSTDIR\Uninstall.exe"
    WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
    WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
    WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
SectionEnd

Section "Windows Service" SecService
    ; Install as Windows Service
    ExecWait '"$INSTDIR\CanvusLocalLLM.exe" install'
    ExecWait 'sc start CanvusLocalLLM'
SectionEnd

Section "Desktop Shortcut" SecShortcut
    CreateShortcut "$DESKTOP\CanvusLocalLLM.lnk" "$INSTDIR\CanvusLocalLLM.exe"
SectionEnd

Section "Start Menu Shortcuts" SecStartMenu
    CreateDirectory "$SMPROGRAMS\CanvusLocalLLM"
    CreateShortcut "$SMPROGRAMS\CanvusLocalLLM\CanvusLocalLLM.lnk" "$INSTDIR\CanvusLocalLLM.exe"
    CreateShortcut "$SMPROGRAMS\CanvusLocalLLM\README.lnk" "$INSTDIR\README.txt"
    CreateShortcut "$SMPROGRAMS\CanvusLocalLLM\Uninstall.lnk" "$INSTDIR\Uninstall.exe"
SectionEnd

; Uninstaller
Section "Uninstall"
    ; Stop and remove service if installed
    ExecWait 'sc stop CanvusLocalLLM'
    ExecWait '"$INSTDIR\CanvusLocalLLM.exe" uninstall'

    ; Remove files
    Delete "$INSTDIR\CanvusLocalLLM.exe"
    Delete "$INSTDIR\.env.example"
    Delete "$INSTDIR\.env"
    Delete "$INSTDIR\README.txt"
    Delete "$INSTDIR\LICENSE.txt"
    Delete "$INSTDIR\Uninstall.exe"

    ; Remove directories
    RMDir /r "$INSTDIR\lib"
    RMDir /r "$INSTDIR\models"
    RMDir /r "$INSTDIR\downloads"
    RMDir "$INSTDIR"

    ; Remove shortcuts
    Delete "$DESKTOP\CanvusLocalLLM.lnk"
    RMDir /r "$SMPROGRAMS\CanvusLocalLLM"

    ; Remove registry keys
    DeleteRegKey HKLM "${PRODUCT_UNINST_KEY}"
SectionEnd
```

#### Installation Validation

Post-installation checks:
- All files copied to $INSTDIR
- Directory structure created
- .env.example present
- Service registered (if selected)
- Shortcuts created (if selected)
- Registry entries correct

---

### 2. Debian Package for Linux

**Component**: canvuslocallm_0.1.0_amd64.deb
**Technology**: dpkg-deb
**Size**: ~10MB (without model), ~8GB (with model)

#### Package Structure

```
canvuslocallm_0.1.0_amd64/
├── DEBIAN/
│   ├── control          # Package metadata
│   ├── postinst         # Post-installation script
│   ├── prerm            # Pre-removal script
│   └── postrm           # Post-removal script
├── opt/
│   └── canvuslocallm/
│       ├── bin/
│       │   └── canvuslocallm          # Application binary
│       ├── lib/                       # Placeholder for Phase 2
│       ├── models/                    # Placeholder for Phase 2
│       ├── .env.example              # Configuration template
│       ├── README.txt                # Documentation
│       └── LICENSE.txt
└── etc/
    └── systemd/
        └── system/
            └── canvuslocallm.service  # systemd unit file
```

#### control File

```
Package: canvuslocallm
Version: 0.1.0
Section: utils
Priority: optional
Architecture: amd64
Depends: libc6 (>= 2.27), nvidia-cuda-toolkit
Maintainer: CanvusLocalLLM Project <maintainer@example.com>
Description: Zero-configuration local AI integration for Canvus
 CanvusLocalLLM provides batteries-included, fully embedded local LLM
 capabilities for Canvus collaborative workspaces. Requires NVIDIA RTX GPU
 with CUDA support. All AI processing happens locally with zero cloud
 dependencies.
Homepage: https://github.com/yourusername/CanvusLocalLLM
```

#### postinst Script

```bash
#!/bin/bash
set -e

INSTALL_DIR="/opt/canvuslocallm"

# Create downloads directory
mkdir -p "$INSTALL_DIR/downloads"

# Copy .env.example to .env if not exists
if [ ! -f "$INSTALL_DIR/.env" ]; then
    cp "$INSTALL_DIR/.env.example" "$INSTALL_DIR/.env"
    echo "Created configuration file: $INSTALL_DIR/.env"
fi

# Set permissions
chmod +x "$INSTALL_DIR/bin/canvuslocallm"
chmod 600 "$INSTALL_DIR/.env"

# Display configuration instructions
cat <<EOF

====================================================================
CanvusLocalLLM Installation Complete
====================================================================

Next steps:

1. Edit the configuration file:
   sudo nano $INSTALL_DIR/.env

2. Configure your Canvus credentials:
   - CANVUS_SERVER: Your Canvus server URL
   - CANVAS_ID: Your canvas identifier
   - CANVUS_API_KEY: Your API key (or username/password)

3. (Optional) Enable and start the service:
   sudo systemctl enable canvuslocallm
   sudo systemctl start canvuslocallm

4. View logs:
   journalctl -u canvuslocallm -f

For more information, see: $INSTALL_DIR/README.txt

====================================================================
EOF
```

#### prerm Script

```bash
#!/bin/bash
set -e

# Stop service if running
if systemctl is-active --quiet canvuslocallm; then
    systemctl stop canvuslocallm
fi
```

#### postrm Script

```bash
#!/bin/bash
set -e

INSTALL_DIR="/opt/canvuslocallm"

# Only remove on purge, not upgrade
if [ "$1" = "purge" ]; then
    # Ask user before removing configuration
    echo "Remove configuration and data? (y/N): "
    read -r response
    if [ "$response" = "y" ] || [ "$response" = "Y" ]; then
        rm -f "$INSTALL_DIR/.env"
        rm -rf "$INSTALL_DIR/downloads"
        rm -f "$INSTALL_DIR/app.log"
        echo "Configuration and data removed."
    fi
fi
```

#### systemd Unit File

```ini
[Unit]
Description=Canvus Local AI Integration
Documentation=https://github.com/yourusername/CanvusLocalLLM
After=network.target

[Service]
Type=simple
User=canvuslocallm
Group=canvuslocallm
WorkingDirectory=/opt/canvuslocallm
ExecStart=/opt/canvuslocallm/bin/canvuslocallm
Restart=on-failure
RestartSec=10s
StandardOutput=append:/opt/canvuslocallm/app.log
StandardError=append:/opt/canvuslocallm/app.log

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/canvuslocallm

[Install]
WantedBy=multi-user.target
```

#### Building the Package

```bash
# Build script: build-deb.sh
#!/bin/bash
set -e

VERSION="0.1.0"
ARCH="amd64"
PKG_NAME="canvuslocallm_${VERSION}_${ARCH}"
BUILD_DIR="build/$PKG_NAME"

# Create directory structure
mkdir -p "$BUILD_DIR/DEBIAN"
mkdir -p "$BUILD_DIR/opt/canvuslocallm/bin"
mkdir -p "$BUILD_DIR/opt/canvuslocallm/lib"
mkdir -p "$BUILD_DIR/opt/canvuslocallm/models"
mkdir -p "$BUILD_DIR/etc/systemd/system"

# Copy binary
cp bin/canvuslocallm "$BUILD_DIR/opt/canvuslocallm/bin/"

# Copy configuration and documentation
cp .env.example "$BUILD_DIR/opt/canvuslocallm/"
cp README.txt "$BUILD_DIR/opt/canvuslocallm/"
cp LICENSE.txt "$BUILD_DIR/opt/canvuslocallm/"

# Copy DEBIAN control files
cp installer/debian/control "$BUILD_DIR/DEBIAN/"
cp installer/debian/postinst "$BUILD_DIR/DEBIAN/"
cp installer/debian/prerm "$BUILD_DIR/DEBIAN/"
cp installer/debian/postrm "$BUILD_DIR/DEBIAN/"
chmod +x "$BUILD_DIR/DEBIAN/postinst"
chmod +x "$BUILD_DIR/DEBIAN/prerm"
chmod +x "$BUILD_DIR/DEBIAN/postrm"

# Copy systemd unit
cp installer/debian/canvuslocallm.service "$BUILD_DIR/etc/systemd/system/"

# Build package
dpkg-deb --build "$BUILD_DIR"

echo "Package built: build/$PKG_NAME.deb"
```

---

### 3. Tarball Distribution for Linux

**Component**: canvuslocallm-0.1.0-linux-amd64.tar.gz
**Technology**: tar + gzip
**Size**: ~10MB (without model), ~8GB (with model)

#### Archive Structure

```
canvuslocallm-0.1.0/
├── install.sh                         # Installation script
├── bin/
│   └── canvuslocallm                  # Application binary
├── lib/                               # Placeholder for Phase 2
├── models/                            # Placeholder for Phase 2
├── .env.example                       # Configuration template
├── README.txt                         # Documentation
├── LICENSE.txt
└── systemd/
    └── canvuslocallm.service          # systemd unit template
```

#### install.sh Script

```bash
#!/bin/bash
set -e

VERSION="0.1.0"
DEFAULT_PREFIX="/opt/canvuslocallm"
PREFIX="${1:-$DEFAULT_PREFIX}"

echo "======================================================================"
echo "CanvusLocalLLM Installation Script"
echo "======================================================================"
echo ""

# Check for custom prefix
if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
    cat <<EOF
Usage: ./install.sh [PREFIX]

Install CanvusLocalLLM to specified directory.

Arguments:
  PREFIX    Installation directory (default: /opt/canvuslocallm)
            Use ~/canvuslocallm for user installation (no sudo required)

Examples:
  ./install.sh                       # Install to /opt (requires sudo)
  ./install.sh ~/canvuslocallm       # Install to home directory
  ./install.sh /opt/custom/path      # Install to custom path

EOF
    exit 0
fi

# Check if we need sudo
NEED_SUDO=false
if [ "${PREFIX:0:1}" = "/" ] && [ "$PREFIX" != "$HOME"* ]; then
    if [ "$EUID" -ne 0 ]; then
        NEED_SUDO=true
        echo "Installation to $PREFIX requires sudo privileges."
        echo ""
    fi
fi

# Create installation directory
echo "Installing to: $PREFIX"
if [ "$NEED_SUDO" = true ]; then
    sudo mkdir -p "$PREFIX"
else
    mkdir -p "$PREFIX"
fi

# Copy files
echo "Copying files..."
COPY_CMD="cp -r"
if [ "$NEED_SUDO" = true ]; then
    COPY_CMD="sudo cp -r"
fi

$COPY_CMD bin "$PREFIX/"
$COPY_CMD lib "$PREFIX/"
$COPY_CMD models "$PREFIX/"
$COPY_CMD .env.example "$PREFIX/"
$COPY_CMD README.txt "$PREFIX/"
$COPY_CMD LICENSE.txt "$PREFIX/"

# Create directories
if [ "$NEED_SUDO" = true ]; then
    sudo mkdir -p "$PREFIX/downloads"
else
    mkdir -p "$PREFIX/downloads"
fi

# Copy .env.example to .env if not exists
if [ ! -f "$PREFIX/.env" ]; then
    if [ "$NEED_SUDO" = true ]; then
        sudo cp "$PREFIX/.env.example" "$PREFIX/.env"
        sudo chmod 600 "$PREFIX/.env"
    else
        cp "$PREFIX/.env.example" "$PREFIX/.env"
        chmod 600 "$PREFIX/.env"
    fi
    echo "Created configuration file: $PREFIX/.env"
fi

# Set executable permissions
if [ "$NEED_SUDO" = true ]; then
    sudo chmod +x "$PREFIX/bin/canvuslocallm"
else
    chmod +x "$PREFIX/bin/canvuslocallm"
fi

# Handle systemd service
if [ -d "/etc/systemd/system" ]; then
    echo ""
    echo "systemd detected. Would you like to install as a service? (y/N): "
    read -r response
    if [ "$response" = "y" ] || [ "$response" = "Y" ]; then
        # Update systemd unit with actual prefix
        sed "s|/opt/canvuslocallm|$PREFIX|g" systemd/canvuslocallm.service > /tmp/canvuslocallm.service
        sudo cp /tmp/canvuslocallm.service /etc/systemd/system/
        sudo systemctl daemon-reload
        echo "Service installed. Enable with:"
        echo "  sudo systemctl enable canvuslocallm"
        echo "  sudo systemctl start canvuslocallm"
    fi
fi

# Display next steps
cat <<EOF

====================================================================
Installation Complete!
====================================================================

Installation directory: $PREFIX

Next steps:

1. Edit the configuration file:
   nano $PREFIX/.env
   (or use sudo nano if installed to /opt)

2. Configure your Canvus credentials:
   - CANVUS_SERVER: Your Canvus server URL
   - CANVAS_ID: Your canvas identifier
   - CANVUS_API_KEY: Your API key (or username/password)

3. Run CanvusLocalLLM:
   $PREFIX/bin/canvuslocallm

   (or if installed as service):
   sudo systemctl start canvuslocallm

4. View logs:
   tail -f $PREFIX/app.log

   (or if running as service):
   journalctl -u canvuslocallm -f

For more information, see: $PREFIX/README.txt

====================================================================
EOF
```

#### Building the Tarball

```bash
# Build script: build-tarball.sh
#!/bin/bash
set -e

VERSION="0.1.0"
ARCH="linux-amd64"
ARCHIVE_NAME="canvuslocallm-${VERSION}-${ARCH}"
BUILD_DIR="build/$ARCHIVE_NAME"

# Create directory structure
mkdir -p "$BUILD_DIR/bin"
mkdir -p "$BUILD_DIR/lib"
mkdir -p "$BUILD_DIR/models"
mkdir -p "$BUILD_DIR/systemd"

# Copy binary
cp bin/canvuslocallm "$BUILD_DIR/bin/"

# Copy configuration and documentation
cp .env.example "$BUILD_DIR/"
cp README.txt "$BUILD_DIR/"
cp LICENSE.txt "$BUILD_DIR/"

# Copy install script
cp installer/tarball/install.sh "$BUILD_DIR/"
chmod +x "$BUILD_DIR/install.sh"

# Copy systemd unit
cp installer/debian/canvuslocallm.service "$BUILD_DIR/systemd/"

# Create tarball
cd build
tar -czf "${ARCHIVE_NAME}.tar.gz" "$ARCHIVE_NAME"
cd ..

echo "Tarball built: build/${ARCHIVE_NAME}.tar.gz"
```

---

### 4. Minimal Configuration Template

**Component**: .env.example
**Purpose**: Provide minimal, Canvus-only configuration template

#### Template Content

```bash
# CanvusLocalLLM Configuration
# ============================================================================
# This file contains the minimal configuration needed to connect to Canvus.
# All AI model and inference settings are preconfigured.
#
# Setup Instructions:
# 1. Copy this file to .env
# 2. Replace the placeholder values with your Canvus credentials
# 3. Save and start CanvusLocalLLM
# ============================================================================

# Canvus Server URL
# The URL of your Canvus server (including https://)
CANVUS_SERVER=https://your-canvus-server.com

# Canvas ID
# The unique identifier for the canvas you want to monitor
# Find this in your Canvus canvas URL or settings
CANVAS_ID=your-canvas-id-here

# Canvas Name (Optional)
# A friendly name for logging purposes
CANVAS_NAME=MyCanvas

# ============================================================================
# Authentication - Use EITHER API Key OR Username/Password
# ============================================================================

# Option 1: API Key Authentication (Recommended)
# Get your API key from Canvus account settings
CANVUS_API_KEY=your-api-key-here

# Option 2: Username/Password Authentication
# Uncomment and configure if you don't have an API key
# CANVUS_USERNAME=your-username
# CANVUS_PASSWORD=your-password

# ============================================================================
# Advanced Settings (Optional - defaults are usually fine)
# ============================================================================

# Allow self-signed SSL certificates (for development only)
# Set to true if your Canvus server uses self-signed certificates
# WARNING: This reduces security and should only be used for testing
# ALLOW_SELF_SIGNED_CERTS=false

# Web UI Password (Optional)
# Uncomment to enable password protection for the web dashboard
# WEBUI_PWD=your-secure-password-here

# ============================================================================
# End of Configuration
# ============================================================================
```

#### Configuration Validation Rules

1. **Required Variables**:
   - `CANVUS_SERVER`: Must be valid URL (http:// or https://)
   - `CANVAS_ID`: Must be non-empty string
   - Either `CANVUS_API_KEY` OR both `CANVUS_USERNAME` and `CANVUS_PASSWORD`

2. **Optional Variables**:
   - `CANVAS_NAME`: Defaults to "Canvas"
   - `ALLOW_SELF_SIGNED_CERTS`: Defaults to false
   - `WEBUI_PWD`: No default (web UI accessible without password if not set)

3. **Validation Flow**:
   ```
   Check .env exists
       │
       ▼
   Parse environment variables
       │
       ▼
   Validate CANVUS_SERVER format
       │
       ▼
   Validate authentication credentials present
       │
       ▼
   Test HTTP connectivity to CANVUS_SERVER
       │
       ▼
   Test Canvus API authentication
       │
       ▼
   Test CANVAS_ID accessibility
       │
       ▼
   Success: Log configuration summary
   ```

---

### 5. First-Run Model Download

**Component**: Model download manager in Go
**Purpose**: Automatically download Bunny v1.1 model if not bundled

#### Implementation Design

##### Atoms (Pure Functions)

```go
// core/download.go

// ComputeSHA256 computes SHA256 checksum of file
func ComputeSHA256(filepath string) (string, error) {
    f, err := os.Open(filepath)
    if err != nil {
        return "", fmt.Errorf("open file: %w", err)
    }
    defer f.Close()

    h := sha256.New()
    if _, err := io.Copy(h, f); err != nil {
        return "", fmt.Errorf("compute hash: %w", err)
    }

    return hex.EncodeToString(h.Sum(nil)), nil
}

// FormatBytes formats byte count as human-readable string
func FormatBytes(bytes int64) string {
    const unit = 1024
    if bytes < unit {
        return fmt.Sprintf("%d B", bytes)
    }
    div, exp := int64(unit), 0
    for n := bytes / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
```

##### Molecules (Simple Compositions)

```go
// core/download.go

// ProgressTracker tracks download progress
type ProgressTracker struct {
    Total      int64
    Downloaded int64
    StartTime  time.Time
    mu         sync.Mutex
}

func (pt *ProgressTracker) Update(n int64) {
    pt.mu.Lock()
    defer pt.mu.Unlock()
    pt.Downloaded += n
}

func (pt *ProgressTracker) Progress() (percent float64, speed string, eta string) {
    pt.mu.Lock()
    defer pt.mu.Unlock()

    percent = float64(pt.Downloaded) / float64(pt.Total) * 100
    elapsed := time.Since(pt.StartTime).Seconds()
    if elapsed > 0 {
        bytesPerSec := float64(pt.Downloaded) / elapsed
        speed = FormatBytes(int64(bytesPerSec)) + "/s"

        if bytesPerSec > 0 {
            remaining := float64(pt.Total-pt.Downloaded) / bytesPerSec
            eta = time.Duration(remaining * float64(time.Second)).String()
        }
    }
    return
}

// DownloadWithProgress downloads file with progress tracking and resume support
func DownloadWithProgress(url, destPath string, expectedChecksum string) error {
    // Check if partial download exists
    var resumeFrom int64
    if fi, err := os.Stat(destPath + ".partial"); err == nil {
        resumeFrom = fi.Size()
    }

    // Create HTTP request with Range header if resuming
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }
    if resumeFrom > 0 {
        req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeFrom))
    }

    // Execute request
    client := core.GetDefaultHTTPClient()
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("download request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
        return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
    }

    // Get total size
    totalSize := resp.ContentLength
    if resumeFrom > 0 {
        totalSize += resumeFrom
    }

    // Open destination file
    flags := os.O_CREATE | os.O_WRONLY
    if resumeFrom > 0 {
        flags |= os.O_APPEND
    } else {
        flags |= os.O_TRUNC
    }
    out, err := os.OpenFile(destPath+".partial", flags, 0644)
    if err != nil {
        return fmt.Errorf("create destination: %w", err)
    }
    defer out.Close()

    // Download with progress tracking
    tracker := &ProgressTracker{
        Total:      totalSize,
        Downloaded: resumeFrom,
        StartTime:  time.Now(),
    }

    // Progress printing goroutine
    done := make(chan struct{})
    go func() {
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-done:
                return
            case <-ticker.C:
                percent, speed, eta := tracker.Progress()
                fmt.Printf("\rDownloading: %.1f%% (%s) - %s - ETA: %s",
                    percent, FormatBytes(tracker.Downloaded), speed, eta)
            }
        }
    }()

    // Copy with progress tracking
    buf := make([]byte, 32*1024)
    for {
        n, err := resp.Body.Read(buf)
        if n > 0 {
            if _, writeErr := out.Write(buf[:n]); writeErr != nil {
                close(done)
                return fmt.Errorf("write to file: %w", writeErr)
            }
            tracker.Update(int64(n))
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            close(done)
            return fmt.Errorf("read from response: %w", err)
        }
    }
    close(done)

    // Final progress
    percent, _, _ := tracker.Progress()
    fmt.Printf("\rDownloading: %.1f%% (%s) - Complete\n",
        percent, FormatBytes(tracker.Downloaded))

    // Rename to final destination
    if err := os.Rename(destPath+".partial", destPath); err != nil {
        return fmt.Errorf("rename to final destination: %w", err)
    }

    // Verify checksum
    if expectedChecksum != "" {
        fmt.Println("Verifying checksum...")
        actualChecksum, err := ComputeSHA256(destPath)
        if err != nil {
            return fmt.Errorf("compute checksum: %w", err)
        }
        if actualChecksum != expectedChecksum {
            return fmt.Errorf("checksum mismatch: expected %s, got %s",
                expectedChecksum, actualChecksum)
        }
        fmt.Println("Checksum verified successfully")
    }

    return nil
}
```

##### Organism (Feature Module)

```go
// core/modelmanager.go

const (
    BunnyModelURL      = "https://huggingface.co/BAAI/Bunny-v1_1-Llama-3-8B-V/resolve/main/ggml-model-Q4_K_M.gguf"
    BunnyModelChecksum = "abc123..." // Replace with actual SHA256
    BunnyModelFilename = "bunny-v1.1-llama-3-8b-v.gguf"
)

// ModelManager handles model availability and downloading
type ModelManager struct {
    modelsDir string
}

// NewModelManager creates a new model manager
func NewModelManager(modelsDir string) *ModelManager {
    return &ModelManager{modelsDir: modelsDir}
}

// EnsureModelAvailable checks if model exists, downloads if missing
func (mm *ModelManager) EnsureModelAvailable() error {
    modelPath := filepath.Join(mm.modelsDir, BunnyModelFilename)

    // Check if model already exists
    if _, err := os.Stat(modelPath); err == nil {
        fmt.Println("Model already available:", modelPath)
        return nil
    }

    fmt.Println("Model not found, downloading...")
    fmt.Printf("This is a large file (~8GB) and may take some time.\n\n")

    // Check disk space
    if err := mm.checkDiskSpace(8 * 1024 * 1024 * 1024); err != nil {
        return fmt.Errorf("insufficient disk space: %w", err)
    }

    // Download with retries
    var lastErr error
    for attempt := 1; attempt <= 3; attempt++ {
        if attempt > 1 {
            fmt.Printf("\nRetrying download (attempt %d/3)...\n", attempt)
            time.Sleep(time.Duration(attempt*2) * time.Second)
        }

        err := DownloadWithProgress(BunnyModelURL, modelPath, BunnyModelChecksum)
        if err == nil {
            fmt.Println("\nModel download completed successfully!")
            return nil
        }
        lastErr = err
        fmt.Printf("\nDownload failed: %v\n", err)
    }

    return fmt.Errorf("model download failed after 3 attempts: %w", lastErr)
}

// checkDiskSpace verifies sufficient disk space
func (mm *ModelManager) checkDiskSpace(required int64) error {
    // Implementation depends on platform
    // For now, simplified version
    return nil
}
```

#### Error Messages

```go
// Error message templates

// ErrNoInternet
"ERROR: Cannot reach model download server

Could not connect to Hugging Face to download the AI model.

Please check:
  - Your internet connection is working
  - Firewall is not blocking downloads from huggingface.co

You can also download manually:
  1. Visit: %s
  2. Save to: %s
  3. Restart CanvusLocalLLM
"

// ErrInsufficientSpace
"ERROR: Insufficient disk space

Required: %s
Available: %s

Please free up disk space and try again.
"

// ErrChecksumFailed
"ERROR: Downloaded file is corrupted

The model file failed checksum verification.
This likely means the download was interrupted or corrupted.

The file has been removed. Please try again.
"

// ErrManualDownloadNeeded
"ERROR: Automatic download failed

Could not download model after 3 attempts.

Please download manually:
  1. Visit: %s
  2. Download: %s
  3. Save to: %s
  4. Verify SHA256: %s
  5. Restart CanvusLocalLLM

For help, see: https://github.com/yourusername/CanvusLocalLLM/wiki/manual-model-download
"
```

---

### 6. Configuration Validation

**Component**: Startup validation in main.go
**Purpose**: Validate configuration before operations, provide helpful errors

#### Implementation Design

```go
// core/validation.go

// ConfigValidator validates startup configuration
type ConfigValidator struct {
    config *Config
    client *http.Client
}

// NewConfigValidator creates a new validator
func NewConfigValidator(config *Config) *ConfigValidator {
    return &ConfigValidator{
        config: config,
        client: GetDefaultHTTPClient(),
    }
}

// Validate performs all validation checks
func (cv *ConfigValidator) Validate() error {
    checks := []struct {
        name string
        fn   func() error
    }{
        {"Configuration file", cv.checkEnvFile},
        {"Canvus server URL", cv.checkServerURL},
        {"Canvas ID", cv.checkCanvasID},
        {"Authentication credentials", cv.checkAuthCredentials},
        {"Server connectivity", cv.checkServerConnectivity},
        {"API authentication", cv.checkAPIAuth},
        {"Canvas accessibility", cv.checkCanvasAccess},
    }

    for _, check := range checks {
        fmt.Printf("Checking %s... ", check.name)
        if err := check.fn(); err != nil {
            fmt.Println("✗")
            return fmt.Errorf("%s failed: %w", check.name, err)
        }
        fmt.Println("✓")
    }

    return nil
}

// checkEnvFile verifies .env exists
func (cv *ConfigValidator) checkEnvFile() error {
    if _, err := os.Stat(".env"); os.IsNotExist(err) {
        return fmt.Errorf(envFileMissingTemplate)
    }
    return nil
}

// checkServerURL validates URL format
func (cv *ConfigValidator) checkServerURL() error {
    if cv.config.CanvusServer == "" {
        return fmt.Errorf("CANVUS_SERVER is not set")
    }

    u, err := url.Parse(cv.config.CanvusServer)
    if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
        return fmt.Errorf(invalidServerURLTemplate, cv.config.CanvusServer)
    }

    return nil
}

// checkCanvasID validates canvas ID present
func (cv *ConfigValidator) checkCanvasID() error {
    if cv.config.CanvasID == "" {
        return fmt.Errorf("CANVAS_ID is not set")
    }
    return nil
}

// checkAuthCredentials validates auth config
func (cv *ConfigValidator) checkAuthCredentials() error {
    hasAPIKey := cv.config.CanvusAPIKey != ""
    hasUserPass := cv.config.CanvusUsername != "" && cv.config.CanvusPassword != ""

    if !hasAPIKey && !hasUserPass {
        return fmt.Errorf(missingAuthTemplate)
    }

    return nil
}

// checkServerConnectivity tests HTTP connectivity
func (cv *ConfigValidator) checkServerConnectivity() error {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    req, err := http.NewRequestWithContext(ctx, "HEAD", cv.config.CanvusServer, nil)
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    resp, err := cv.client.Do(req)
    if err != nil {
        return fmt.Errorf(serverUnreachableTemplate, cv.config.CanvusServer, err)
    }
    defer resp.Body.Close()

    return nil
}

// checkAPIAuth tests API authentication
func (cv *ConfigValidator) checkAPIAuth() error {
    // Create canvus API client
    client := canvusapi.NewClient(
        cv.config.CanvusServer,
        cv.config.CanvasID,
        cv.config.CanvusAPIKey,
        cv.config.CanvusUsername,
        cv.config.CanvusPassword,
        cv.client,
    )

    // Try to get canvas info
    _, err := client.GetWidgets()
    if err != nil {
        return fmt.Errorf(authFailedTemplate, cv.config.CanvusServer)
    }

    return nil
}

// checkCanvasAccess verifies canvas exists and is accessible
func (cv *ConfigValidator) checkCanvasAccess() error {
    client := canvusapi.NewClient(
        cv.config.CanvusServer,
        cv.config.CanvasID,
        cv.config.CanvusAPIKey,
        cv.config.CanvusUsername,
        cv.config.CanvusPassword,
        cv.client,
    )

    widgets, err := client.GetWidgets()
    if err != nil {
        return fmt.Errorf(canvasNotFoundTemplate, cv.config.CanvasID)
    }

    // Success - log summary
    fmt.Printf("\n✓ Configuration valid\n")
    fmt.Printf("✓ Connected to Canvus server: %s\n", cv.config.CanvusServer)
    fmt.Printf("✓ Canvas accessible: %s (ID: %s)\n", cv.config.CanvasName, cv.config.CanvasID)
    fmt.Printf("✓ Found %d widgets on canvas\n", len(widgets))

    return nil
}
```

#### Integration in main.go

```go
// main.go

func main() {
    // Load configuration
    config, err := core.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    // Validate configuration
    validator := core.NewConfigValidator(config)
    if err := validator.Validate(); err != nil {
        log.Fatalf("\n%v\n", err)
    }

    // Ensure model available
    modelManager := core.NewModelManager("./models")
    if err := modelManager.EnsureModelAvailable(); err != nil {
        log.Fatalf("Model unavailable: %v", err)
    }

    fmt.Println("\n✓ Starting canvas monitoring...")

    // Continue with application startup...
}
```

---

### 7. Optional Service Creation

**Component**: Windows Service and Linux systemd integration
**Purpose**: Enable background service operation with automatic startup

#### Windows Service Implementation

```go
// service_windows.go
//go:build windows

package main

import (
    "fmt"
    "log"
    "os"
    "time"

    "github.com/kardianos/service"
)

// Program implements service.Interface
type Program struct {
    config  *core.Config
    monitor *Monitor
    logger  service.Logger
}

func (p *Program) Start(s service.Service) error {
    p.logger.Info("Starting CanvusLocalLLM service...")
    go p.run()
    return nil
}

func (p *Program) run() {
    // Initialize logging
    logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        p.logger.Error("Failed to open log file:", err)
        return
    }
    defer logFile.Close()

    log.SetOutput(logFile)

    // Load configuration
    config, err := core.LoadConfig()
    if err != nil {
        p.logger.Error("Failed to load configuration:", err)
        return
    }
    p.config = config

    // Validate configuration
    validator := core.NewConfigValidator(config)
    if err := validator.Validate(); err != nil {
        p.logger.Error("Configuration validation failed:", err)
        return
    }

    // Ensure model available
    modelManager := core.NewModelManager("./models")
    if err := modelManager.EnsureModelAvailable(); err != nil {
        p.logger.Error("Model unavailable:", err)
        return
    }

    // Start monitoring
    p.logger.Info("Starting canvas monitoring...")

    // Create and start monitor
    // (Implementation continues with actual monitor startup)
}

func (p *Program) Stop(s service.Service) error {
    p.logger.Info("Stopping CanvusLocalLLM service...")
    if p.monitor != nil {
        p.monitor.Stop()
    }
    return nil
}

// ServiceMain handles service commands
func ServiceMain() {
    svcConfig := &service.Config{
        Name:        "CanvusLocalLLM",
        DisplayName: "Canvus Local AI Integration",
        Description: "Monitors Canvus workspaces and processes AI requests locally using embedded models.",
    }

    prg := &Program{}
    s, err := service.New(prg, svcConfig)
    if err != nil {
        log.Fatal(err)
    }

    prg.logger, err = s.Logger(nil)
    if err != nil {
        log.Fatal(err)
    }

    // Handle service commands
    if len(os.Args) > 1 {
        cmd := os.Args[1]
        switch cmd {
        case "install":
            err = s.Install()
            if err != nil {
                fmt.Printf("Failed to install service: %v\n", err)
                os.Exit(1)
            }
            fmt.Println("Service installed successfully")
            fmt.Println("Start with: sc start CanvusLocalLLM")
            return
        case "uninstall":
            err = s.Uninstall()
            if err != nil {
                fmt.Printf("Failed to uninstall service: %v\n", err)
                os.Exit(1)
            }
            fmt.Println("Service uninstalled successfully")
            return
        case "start":
            err = s.Start()
            if err != nil {
                fmt.Printf("Failed to start service: %v\n", err)
                os.Exit(1)
            }
            fmt.Println("Service started successfully")
            return
        case "stop":
            err = s.Stop()
            if err != nil {
                fmt.Printf("Failed to stop service: %v\n", err)
                os.Exit(1)
            }
            fmt.Println("Service stopped successfully")
            return
        case "restart":
            err = s.Restart()
            if err != nil {
                fmt.Printf("Failed to restart service: %v\n", err)
                os.Exit(1)
            }
            fmt.Println("Service restarted successfully")
            return
        }
    }

    // Run as service
    err = s.Run()
    if err != nil {
        prg.logger.Error(err)
    }
}
```

#### Linux systemd Service

Systemd unit file already defined in Debian package section. Service management:

```bash
# Install service
sudo systemctl enable canvuslocallm

# Start service
sudo systemctl start canvuslocallm

# Check status
sudo systemctl status canvuslocallm

# View logs
journalctl -u canvuslocallm -f

# Stop service
sudo systemctl stop canvuslocallm

# Disable service
sudo systemctl disable canvuslocallm
```

---

## File Structure

### Development Repository Structure

```
CanvusLocalLLM/
├── go.mod
├── go.sum
├── main.go                          # Application entry point
├── handlers.go                      # AI processing handlers (existing)
├── monitorcanvus.go                 # Canvas monitoring (existing)
├── service_windows.go               # Windows service implementation
├── service_linux.go                 # Linux service stub
├── core/
│   ├── config.go                    # Configuration loading (existing)
│   ├── ai.go                        # AI client creation (existing)
│   ├── validation.go                # Configuration validation (NEW)
│   ├── download.go                  # Download utilities (NEW)
│   └── modelmanager.go              # Model management (NEW)
├── canvusapi/
│   └── canvusapi.go                 # Canvus API client (existing)
├── logging/
│   └── logging.go                   # Logging utilities (existing)
├── tests/
│   ├── config_validation_test.go    # Validation tests (NEW)
│   ├── download_test.go             # Download tests (NEW)
│   └── (existing tests)
├── installer/
│   ├── nsis/
│   │   ├── installer.nsi            # NSIS script (NEW)
│   │   ├── icon.ico                 # Application icon (NEW)
│   │   └── LICENSE.txt              # License for installer (NEW)
│   ├── debian/
│   │   ├── control                  # Package metadata (NEW)
│   │   ├── postinst                 # Post-install script (NEW)
│   │   ├── prerm                    # Pre-removal script (NEW)
│   │   ├── postrm                   # Post-removal script (NEW)
│   │   └── canvuslocallm.service    # systemd unit (NEW)
│   └── tarball/
│       └── install.sh               # Installation script (NEW)
├── .env.example                     # Configuration template (UPDATED)
├── README.txt                       # User documentation (NEW)
├── LICENSE.txt                      # MIT License (NEW)
└── build/                           # Build output directory (generated)
    ├── CanvusLocalLLM-Setup.exe     # Windows installer
    ├── canvuslocallm_0.1.0_amd64.deb
    └── canvuslocallm-0.1.0-linux-amd64.tar.gz
```

### Post-Installation Structure (Windows)

```
C:\Program Files\CanvusLocalLLM\
├── CanvusLocalLLM.exe               # Application binary
├── lib/                             # Native libraries (Phase 2+)
│   ├── llama.dll
│   └── stable-diffusion.dll
├── models/                          # AI models (Phase 2+)
│   └── bunny-v1.1-llama-3-8b-v.gguf
├── downloads/                       # Temporary downloads
├── .env.example                     # Configuration template
├── .env                             # User configuration
├── README.txt                       # Documentation
├── LICENSE.txt                      # License
└── app.log                          # Application log
```

### Post-Installation Structure (Linux)

```
/opt/canvuslocallm/
├── bin/
│   └── canvuslocallm                # Application binary
├── lib/                             # Native libraries (Phase 2+)
│   ├── llama.so
│   └── stable-diffusion.so
├── models/                          # AI models (Phase 2+)
│   └── bunny-v1.1-llama-3-8b-v.gguf
├── downloads/                       # Temporary downloads
├── .env.example                     # Configuration template
├── .env                             # User configuration
├── README.txt                       # Documentation
├── LICENSE.txt                      # License
└── app.log                          # Application log
```

---

## Technical Approach

### Build Process

#### Prerequisites

- Go 1.21+
- NSIS 3.x (Windows installer)
- dpkg-deb (Debian package)
- tar/gzip (tarball)

#### Build Steps

```bash
# 1. Build Go application for Windows
GOOS=windows GOARCH=amd64 go build -o bin/CanvusLocalLLM.exe .

# 2. Build Go application for Linux
GOOS=linux GOARCH=amd64 go build -o bin/canvuslocallm .

# 3. Build Windows installer
makensis installer/nsis/installer.nsi

# 4. Build Debian package
./scripts/build-deb.sh

# 5. Build tarball
./scripts/build-tarball.sh
```

#### Automation Script

```bash
# scripts/build-all.sh
#!/bin/bash
set -e

VERSION="0.1.0"

echo "Building CanvusLocalLLM v${VERSION}"
echo "======================================"

# Clean build directory
rm -rf build/
mkdir -p build/{windows,linux}/bin

# Build Windows binary
echo "Building Windows binary..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o build/windows/bin/CanvusLocalLLM.exe .

# Build Linux binary
echo "Building Linux binary..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/linux/bin/canvuslocallm .

# Build Windows installer
echo "Building Windows installer..."
cp build/windows/bin/CanvusLocalLLM.exe installer/nsis/bin/
makensis installer/nsis/installer.nsi
mv installer/nsis/CanvusLocalLLM-Setup.exe build/

# Build Debian package
echo "Building Debian package..."
./scripts/build-deb.sh

# Build tarball
echo "Building tarball..."
./scripts/build-tarball.sh

echo ""
echo "Build complete!"
echo "================"
echo "Windows: build/CanvusLocalLLM-Setup.exe"
echo "Debian:  build/canvuslocallm_${VERSION}_amd64.deb"
echo "Tarball: build/canvuslocallm-${VERSION}-linux-amd64.tar.gz"
```

### Configuration Validation Strategy

**Fail Fast Principle**: Validate everything before starting heavy operations (model loading, GPU initialization)

**Validation Layers**:
1. **Syntax**: File exists, variables set, correct format
2. **Network**: Server reachable, DNS resolves
3. **Authentication**: Credentials work, API responds
4. **Authorization**: Canvas accessible, permissions sufficient

**Error Message Design**:
- What went wrong (concrete, not abstract)
- Why it matters
- How to fix it (actionable steps)
- Where to get help

### Model Download Strategy

**Design Decisions**:
- Resumable downloads (HTTP Range headers)
- Progress indication (ASCII progress bar)
- Checksum verification (SHA256)
- Retry with exponential backoff
- Graceful degradation (manual download instructions)

**User Experience**:
- Clear indication of file size and time estimate
- Progress updates every second
- Verification step after download
- Success confirmation with next steps

---

## Testing Strategy

### Unit Tests

**Configuration Validation** (`tests/config_validation_test.go`):
```go
func TestValidateServerURL(t *testing.T) {
    tests := []struct {
        name    string
        url     string
        wantErr bool
    }{
        {"valid https", "https://canvus.example.com", false},
        {"valid http", "http://localhost:3000", false},
        {"missing scheme", "canvus.example.com", true},
        {"invalid scheme", "ftp://canvus.example.com", true},
        {"empty", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            config := &core.Config{CanvusServer: tt.url}
            validator := core.NewConfigValidator(config)
            err := validator.checkServerURL()
            if (err != nil) != tt.wantErr {
                t.Errorf("checkServerURL() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

**Download Utilities** (`tests/download_test.go`):
```go
func TestComputeSHA256(t *testing.T) {
    // Create temp file with known content
    content := "test content"
    expectedHash := "6ae8a75555209fd6c44157c0aed8016e763ff435a19cf186f76863140143ff72"

    tmpfile, err := os.CreateTemp("", "test")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpfile.Name())

    if _, err := tmpfile.Write([]byte(content)); err != nil {
        t.Fatal(err)
    }
    tmpfile.Close()

    hash, err := core.ComputeSHA256(tmpfile.Name())
    if err != nil {
        t.Fatalf("ComputeSHA256() error = %v", err)
    }

    if hash != expectedHash {
        t.Errorf("ComputeSHA256() = %v, want %v", hash, expectedHash)
    }
}
```

### Integration Tests

**End-to-End Installation** (Manual):
1. Windows 10/11:
   - Run CanvusLocalLLM-Setup.exe
   - Select default options
   - Verify all files installed
   - Verify service creation (if selected)
   - Verify uninstaller works

2. Ubuntu 22.04:
   - Install .deb: `sudo dpkg -i canvuslocallm_0.1.0_amd64.deb`
   - Verify files in /opt/canvuslocallm/
   - Configure .env
   - Start service: `sudo systemctl start canvuslocallm`
   - Check logs: `journalctl -u canvuslocallm`

3. Fedora (Tarball):
   - Extract tarball
   - Run install.sh
   - Verify installation
   - Test manual startup

**Configuration Validation** (Automated):
```go
func TestConfigValidationEndToEnd(t *testing.T) {
    // Requires Canvus test server
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Test with valid configuration
    config := &core.Config{
        CanvusServer: os.Getenv("TEST_CANVUS_SERVER"),
        CanvasID:     os.Getenv("TEST_CANVAS_ID"),
        CanvusAPIKey: os.Getenv("TEST_API_KEY"),
    }

    validator := core.NewConfigValidator(config)
    err := validator.Validate()
    if err != nil {
        t.Errorf("Validation failed with valid config: %v", err)
    }
}
```

### Performance Tests

**Installation Time**:
- Measure on: Windows 10, Windows 11, Ubuntu 22.04
- Target: <5 minutes with bundled model
- Target: <15 minutes with model download (8GB @ 10MB/s)

**Configuration Validation Time**:
- All checks should complete in <2 seconds
- Network timeout: 10 seconds maximum

**Model Download**:
- Test resume functionality
- Verify progress indication accuracy
- Test on slow connections (throttled)

### User Acceptance Testing

**Scenario 1: Non-Technical User - Windows**
1. Participant: Non-technical user (product manager persona)
2. Task: Install CanvusLocalLLM and configure Canvus connection
3. Measure: Time to completion, errors encountered, confusion points
4. Success: Completes in <10 minutes without assistance

**Scenario 2: IT Administrator - Linux**
1. Participant: IT professional with Linux experience
2. Task: Deploy to production server, enable service, configure
3. Measure: Time to completion, questions asked, documentation gaps
4. Success: Successful deployment in <15 minutes

**Scenario 3: Error Recovery**
1. Participant: Any user
2. Task: Intentionally misconfigure (wrong credentials, invalid URL)
3. Measure: Error message clarity, ability to fix without help
4. Success: User understands error and fixes independently

---

## Acceptance Criteria

Phase 1 is complete when ALL of the following criteria are met:

### Installation Criteria

- [ ] **AC-1.1**: Windows NSIS installer runs on Windows 10/11 without errors
- [ ] **AC-1.2**: All files installed to correct locations with correct permissions
- [ ] **AC-1.3**: Directory structure created (lib/, models/, downloads/)
- [ ] **AC-1.4**: Uninstaller removes all components cleanly
- [ ] **AC-1.5**: Debian package installs successfully on Ubuntu 20.04, 22.04
- [ ] **AC-1.6**: Tarball install.sh works on Ubuntu, Fedora, Arch Linux
- [ ] **AC-1.7**: Start menu / desktop shortcuts created (if selected)

### Configuration Criteria

- [ ] **AC-2.1**: .env.example contains ONLY Canvus credentials
- [ ] **AC-2.2**: No AI model, provider, or inference configuration exposed
- [ ] **AC-2.3**: Inline comments explain each variable clearly
- [ ] **AC-2.4**: Example values clearly need replacement

### Validation Criteria

- [ ] **AC-3.1**: Validation detects missing .env file
- [ ] **AC-3.2**: Validation detects invalid CANVUS_SERVER format
- [ ] **AC-3.3**: Validation detects missing authentication credentials
- [ ] **AC-3.4**: Validation detects unreachable Canvus server
- [ ] **AC-3.5**: Validation detects invalid authentication
- [ ] **AC-3.6**: Validation detects inaccessible canvas
- [ ] **AC-3.7**: All error messages include actionable instructions
- [ ] **AC-3.8**: Validation completes in <2 seconds

### Model Download Criteria

- [ ] **AC-4.1**: Download infrastructure implemented (tested with placeholder)
- [ ] **AC-4.2**: Progress indication shows percentage, speed, ETA
- [ ] **AC-4.3**: Resume works after interruption
- [ ] **AC-4.4**: Checksum verification implemented
- [ ] **AC-4.5**: Retry logic with exponential backoff
- [ ] **AC-4.6**: Clear error messages for all failure modes
- [ ] **AC-4.7**: Manual download instructions provided on failure

### Service Criteria

- [ ] **AC-5.1**: Windows Service installs via installer checkbox
- [ ] **AC-5.2**: Windows Service installs via `CanvusLocalLLM.exe install`
- [ ] **AC-5.3**: Windows Service uninstalls via `CanvusLocalLLM.exe uninstall`
- [ ] **AC-5.4**: Windows Service logs to app.log
- [ ] **AC-5.5**: systemd unit file included in Debian package and tarball
- [ ] **AC-5.6**: Service can be enabled and started on Linux
- [ ] **AC-5.7**: Service logs accessible via journalctl (Linux)

### Testing Criteria

- [ ] **AC-6.1**: All unit tests pass
- [ ] **AC-6.2**: All integration tests pass
- [ ] **AC-6.3**: Installation tested on Windows 10, Windows 11
- [ ] **AC-6.4**: Installation tested on Ubuntu 20.04, 22.04
- [ ] **AC-6.5**: Installation tested on Fedora and Arch (tarball)
- [ ] **AC-6.6**: User acceptance testing with non-technical user succeeds

### Documentation Criteria

- [ ] **AC-7.1**: README.txt includes quick start instructions
- [ ] **AC-7.2**: README.txt includes troubleshooting section
- [ ] **AC-7.3**: README.txt explains service installation (optional)
- [ ] **AC-7.4**: Error messages reference documentation where appropriate

### Performance Criteria

- [ ] **AC-8.1**: Installation completes in <5 minutes (bundled model)
- [ ] **AC-8.2**: Installation completes in <15 minutes (model download)
- [ ] **AC-8.3**: Configuration validation completes in <2 seconds
- [ ] **AC-8.4**: Non-technical user completes setup in <10 minutes

---

## Implementation Plan

### Task Breakdown

**Task 1: Configuration Template and Validation** (XS - 2-4 hours)
1. Update .env.example with minimal Canvus-only configuration
2. Implement core/validation.go with ConfigValidator
3. Add error message templates
4. Write unit tests for validation
5. Integrate validation into main.go

**Task 2: Model Download Infrastructure** (M - 1-2 days)
1. Implement core/download.go (atoms: checksum, format, HTTP Range)
2. Implement core/download.go (molecule: ProgressTracker, DownloadWithProgress)
3. Implement core/modelmanager.go (organism: ModelManager)
4. Add disk space checking
5. Implement retry logic
6. Write unit tests for download utilities
7. Test with placeholder URLs

**Task 3: Windows Service Integration** (M - 1-2 days)
1. Add github.com/kardianos/service dependency
2. Implement service_windows.go (Program struct, Start, Stop, run)
3. Add service commands (install, uninstall, start, stop)
4. Test service installation and operation
5. Test logging when running as service
6. Test service recovery

**Task 4: NSIS Installer** (M - 1-2 days)
1. Create installer/nsis/installer.nsi script
2. Design installer pages (Welcome, License, Directory, Components, Progress, Finish)
3. Implement Core Application section (required)
4. Implement Windows Service section (optional)
5. Implement Desktop Shortcut section (optional)
6. Implement Start Menu Shortcuts section
7. Implement uninstaller
8. Test installer on Windows 10/11
9. Test uninstaller completeness

**Task 5: Debian Package** (S - 4-8 hours)
1. Create installer/debian/control metadata
2. Create installer/debian/postinst script
3. Create installer/debian/prerm script
4. Create installer/debian/postrm script
5. Create installer/debian/canvuslocallm.service systemd unit
6. Create scripts/build-deb.sh build script
7. Test package installation on Ubuntu 20.04, 22.04
8. Test package removal

**Task 6: Tarball Distribution** (S - 4-8 hours)
1. Create installer/tarball/install.sh script
2. Support --prefix flag for custom install location
3. Support system-wide (/opt) and user (~/) installations
4. Add systemd service installation option
5. Create scripts/build-tarball.sh build script
6. Test on Ubuntu, Fedora, Arch Linux
7. Test both system and user installations

**Task 7: Documentation** (S - 4-8 hours)
1. Create README.txt with:
   - Quick start instructions
   - Configuration steps
   - Service installation (optional)
   - Troubleshooting
   - Where to get help
2. Update LICENSE.txt (MIT License)
3. Add inline comments to .env.example
4. Review all error messages for clarity

**Task 8: Build Automation** (S - 4-8 hours)
1. Create scripts/build-all.sh
2. Automate Go binary builds (Windows, Linux)
3. Automate NSIS installer build
4. Automate Debian package build
5. Automate tarball build
6. Test full build process

**Task 9: Testing and Validation** (M - 1-2 days)
1. Write remaining unit tests
2. Write integration tests
3. Perform manual testing on all platforms
4. Conduct user acceptance testing
5. Performance testing (installation time, validation time)
6. Fix bugs and issues
7. Iterate on error messages based on testing feedback

### Estimated Timeline

**Total Effort**: 7-14 days (1 developer)
**Parallel Work Possible**: Tasks 3-6 can be done in parallel after Tasks 1-2 complete

**Milestones**:
- **Week 1**: Configuration validation, model download infrastructure, service integration
- **Week 2**: All installers (NSIS, deb, tarball), documentation, testing

### Dependencies

- **External**: NSIS 3.x installed for Windows builds
- **Internal**: None (Phase 1 is foundation)
- **Deferred**: Actual AI inference (Phase 2), image generation (Phase 3)

### Risks and Mitigations

**Risk**: NSIS learning curve
**Mitigation**: Use provided script as template, extensive testing

**Risk**: Platform-specific bugs in service integration
**Mitigation**: Test on multiple OS versions, clear error handling

**Risk**: Model download reliability
**Mitigation**: Implement resume, retries, manual download fallback

---

## Appendix

### Example README.txt

```
====================================================================
CanvusLocalLLM - Zero-Configuration Local AI Integration for Canvus
====================================================================

Version: 0.1.0
Website: https://github.com/yourusername/CanvusLocalLLM

QUICK START
===========

1. Configure Canvus Connection

   Edit the configuration file with your Canvus credentials:

   Windows: C:\Program Files\CanvusLocalLLM\.env
   Linux:   /opt/canvuslocallm/.env

   Required settings:
   - CANVUS_SERVER: Your Canvus server URL (e.g., https://canvus.example.com)
   - CANVAS_ID: Your canvas identifier
   - CANVUS_API_KEY: Your API key (or username/password)

2. Start CanvusLocalLLM

   Windows (Manual):
     - Double-click CanvusLocalLLM.exe
     - Or run from Start Menu

   Windows (Service):
     - Service starts automatically on boot
     - Control via Services management console

   Linux (Manual):
     /opt/canvuslocallm/bin/canvuslocallm

   Linux (Service):
     sudo systemctl start canvuslocallm

3. Verify Connection

   Check the log file for successful connection:
   - Windows: C:\Program Files\CanvusLocalLLM\app.log
   - Linux:   /opt/canvuslocallm/app.log

   Look for: "✓ Canvas accessible"

USING AI FEATURES
=================

(Phase 2+) Wrap text in {{ }} to trigger AI processing:
  {{ Summarize this document }}

For more information, see:
https://github.com/yourusername/CanvusLocalLLM/wiki

TROUBLESHOOTING
===============

Error: "Configuration file not found"
  → Copy .env.example to .env and configure Canvus credentials

Error: "Canvus authentication failed"
  → Verify CANVUS_API_KEY (or username/password) is correct
  → Test credentials at your Canvus server

Error: "Canvas not found"
  → Verify CANVAS_ID is correct
  → Check you have access to this canvas

Service won't start (Windows):
  → Check app.log for errors
  → Verify configuration is valid
  → Reinstall service: CanvusLocalLLM.exe uninstall && CanvusLocalLLM.exe install

Service won't start (Linux):
  → Check logs: journalctl -u canvuslocallm -f
  → Verify configuration: sudo nano /opt/canvuslocallm/.env
  → Check permissions: ls -la /opt/canvuslocallm/.env

SYSTEM REQUIREMENTS
===================

- NVIDIA RTX GPU (20-series, 30-series, or 40-series)
- 8GB+ GPU VRAM
- 20GB+ disk space
- CUDA 11.8+ drivers
- Windows 10/11 or Linux (Ubuntu, Fedora, Arch, etc.)

GETTING HELP
============

Documentation: https://github.com/yourusername/CanvusLocalLLM/wiki
Issues: https://github.com/yourusername/CanvusLocalLLM/issues
Community: [Discord/Forum link]

LICENSE
=======

MIT License - see LICENSE.txt for details

====================================================================
```

### Example Error Message Templates

```go
// core/errors.go

const (
    envFileMissingTemplate = `
ERROR: Configuration file not found

Please copy .env.example to .env and configure:
  - CANVUS_SERVER: Your Canvus server URL
  - CANVAS_ID: Your canvas identifier
  - CANVUS_API_KEY: Your API key (or username/password)

Location: %s

For setup instructions, see: README.txt
`

    invalidServerURLTemplate = `
ERROR: Invalid Canvus server URL

Current value: %s

Please set CANVUS_SERVER to a valid URL:
  Example: https://canvus.example.com

Edit: %s
`

    missingAuthTemplate = `
ERROR: Missing authentication credentials

You must provide EITHER:
  - CANVUS_API_KEY: Your API key
OR
  - CANVUS_USERNAME and CANVUS_PASSWORD: Your username and password

Edit: %s
`

    serverUnreachableTemplate = `
ERROR: Cannot reach Canvus server

Server: %s
Error: %v

Please check:
  - Server URL is correct
  - Server is running and accessible
  - Firewall is not blocking connections
  - DNS resolves correctly

Test in browser: %s
`

    authFailedTemplate = `
ERROR: Canvus authentication failed

Could not authenticate to %s

Please verify:
  - CANVUS_API_KEY is correct
OR
  - CANVUS_USERNAME and CANVUS_PASSWORD are correct

Get your API key: %s/settings/api

Edit: %s
`

    canvasNotFoundTemplate = `
ERROR: Canvas not found

Canvas ID %s does not exist or is not accessible.

Please verify:
  - Canvas ID is correct (check Canvus URL or settings)
  - You have access to this canvas
  - Canvas is not archived or deleted

Edit: %s
`
)
```

---

**End of Specification**
