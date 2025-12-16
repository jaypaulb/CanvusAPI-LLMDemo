# Phase 1: Zero-Config Installation Infrastructure - Requirements

## Overview

Phase 1 establishes the foundational installation infrastructure for CanvusLocalLLM, enabling users to go from download to working AI integration in under 10 minutes with no technical expertise. This phase implements the "zero-configuration" promise by bundling all dependencies, providing native platform installers, and requiring only Canvus credentials to operate.

## Success Criteria

**Primary Goal**: Enable non-technical users to install CanvusLocalLLM and connect to Canvus in under 10 minutes with zero AI/infrastructure configuration.

**Quantifiable Metrics**:
- Installation completes in <5 minutes on modern hardware
- Configuration requires editing 3-4 environment variables (Canvus credentials only)
- First successful AI operation within 10 minutes of download
- Zero configuration related to AI models, providers, or inference parameters
- 100% of installations work on NVIDIA RTX GPU hardware without additional setup

## User Stories

### Enterprise IT Administrator
**As an** enterprise IT administrator
**I want to** deploy CanvusLocalLLM to team workstations with minimal configuration
**So that** teams can use AI capabilities while maintaining data sovereignty without ongoing infrastructure management

**Acceptance Criteria**:
- Silent installation support for mass deployment
- Service creation for automatic startup
- Configuration validation at startup prevents runtime failures
- Clear error messages for configuration issues

### Product Manager
**As a** product manager using Canvus
**I want to** install AI capabilities with a simple installer
**So that** I can get AI assistance immediately without learning infrastructure concepts

**Acceptance Criteria**:
- Double-click installer with wizard interface
- Only need to know: Canvus server URL, my credentials, canvas ID
- Model downloaded automatically if not bundled
- Clear progress indication during setup

### Independent Consultant
**As an** independent consultant
**I want to** install on Linux without package manager dependencies
**So that** I can deploy to any Linux environment regardless of distribution

**Acceptance Criteria**:
- Tarball distribution works on any Linux with CUDA drivers
- install.sh handles all setup automatically
- No root access required for service-less operation
- Clear documentation for systemd service setup

## Functional Requirements

### 1. NSIS Installer for Windows (Priority: MUST)

**Requirement ID**: FR-1.1
**Description**: Create professional Windows installer using NSIS that bundles all components and guides users through installation

**Detailed Requirements**:
- **FR-1.1.1**: Wizard interface with pages: Welcome, License Agreement, Installation Directory, Components, Installation Progress, Completion
- **FR-1.1.2**: Default installation to `C:\Program Files\CanvusLocalLLM\`
- **FR-1.1.3**: Bundle components:
  - Go application binary (CanvusLocalLLM.exe)
  - Placeholder for llama.cpp libraries (lib/llama.dll) - Phase 2
  - Placeholder for stable-diffusion.cpp libraries (lib/stable-diffusion.dll) - Phase 3
  - Placeholder for Bunny v1.1 model (models/bunny-v1.1-llama-3-8b-v.gguf) - Phase 2
  - Configuration template (.env.example)
  - README.txt with setup instructions
- **FR-1.1.4**: Create directory structure: lib/, models/, downloads/
- **FR-1.1.5**: Copy .env.example to installation directory
- **FR-1.1.6**: Optional component: "Install as Windows Service" checkbox
- **FR-1.1.7**: If service selected, install and start CanvusLocalLLM Windows Service
- **FR-1.1.8**: Uninstaller removes all installed files and service registration
- **FR-1.1.9**: Create Start Menu shortcuts
- **FR-1.1.10**: Registry entries for uninstaller integration

**Validation**:
- Installer executable runs on Windows 10/11 without errors
- All files copied to correct locations
- Service creation works when selected
- Uninstaller removes all components cleanly

### 2. Debian Package for Linux (Priority: MUST)

**Requirement ID**: FR-1.2
**Description**: Create .deb package for Debian/Ubuntu distributions with proper directory structure and post-install configuration

**Detailed Requirements**:
- **FR-1.2.1**: Package metadata (control file):
  - Package name: canvuslocallm
  - Version: 0.1.0 (Phase 1)
  - Architecture: amd64
  - Dependencies: libc6, CUDA runtime (nvidia-cuda-toolkit)
  - Maintainer information
  - Description: Zero-configuration local AI integration for Canvus
- **FR-1.2.2**: Install to `/opt/canvuslocallm/`
- **FR-1.2.3**: Bundle components:
  - Go application binary (canvuslocallm)
  - Placeholder for llama.cpp libraries (lib/llama.so) - Phase 2
  - Placeholder for stable-diffusion.cpp libraries (lib/stable-diffusion.so) - Phase 3
  - Placeholder for Bunny v1.1 model - Phase 2
  - Configuration template (.env.example)
  - README.txt
- **FR-1.2.4**: postinst script:
  - Create downloads/ directory
  - Copy .env.example to .env if not exists
  - Set proper permissions (executable for binary, readable for configs)
  - Display configuration instructions to user
- **FR-1.2.5**: prerm script: Stop service if running
- **FR-1.2.6**: postrm script: Clean up user data (optional, with prompt)

**Validation**:
- dpkg -i installs package successfully
- All files in correct locations with correct permissions
- postinst script creates necessary directories
- dpkg -r removes package cleanly

### 3. Tarball Distribution for Linux (Priority: MUST)

**Requirement ID**: FR-1.3
**Description**: Create portable .tar.gz archive with install.sh script for non-Debian distributions

**Detailed Requirements**:
- **FR-1.3.1**: Archive structure:
  ```
  canvuslocallm-0.1.0/
  ├── install.sh
  ├── bin/canvuslocallm
  ├── lib/ (placeholders for Phase 2/3)
  ├── models/ (placeholders for Phase 2)
  ├── .env.example
  └── README.txt
  ```
- **FR-1.3.2**: install.sh script:
  - Default install to /opt/canvuslocallm/ (requires sudo)
  - Alternative: ~/canvuslocallm/ (no sudo required)
  - Extract all files to install directory
  - Create downloads/ directory
  - Copy .env.example to .env if not exists
  - Set executable permissions on binary
  - Display setup instructions
- **FR-1.3.3**: install.sh supports --prefix flag for custom install location
- **FR-1.3.4**: Includes systemd unit file template in archive

**Validation**:
- Extract and install.sh works on Ubuntu, Fedora, Arch Linux
- Both system-wide and user installations work
- No dependencies on package managers
- Clear error messages for missing CUDA drivers

### 4. Minimal Configuration Template (Priority: MUST)

**Requirement ID**: FR-1.4
**Description**: Create .env.example with minimal configuration focused solely on Canvus connectivity

**Detailed Requirements**:
- **FR-1.4.1**: Required variables:
  ```bash
  # Canvus Server Connection
  CANVUS_SERVER=https://your-canvus-server.com
  CANVAS_ID=your-canvas-id

  # Authentication (use either API key OR username/password)
  CANVUS_API_KEY=your-api-key-here
  # CANVUS_USERNAME=your-username
  # CANVUS_PASSWORD=your-password

  # Canvas Name (optional, for logging)
  CANVAS_NAME=MyCanvas
  ```
- **FR-1.4.2**: NO configuration for:
  - AI models or model paths
  - Inference parameters (temperature, top_p, etc.)
  - Provider selection or API endpoints
  - GPU/CUDA settings
  - Performance tuning
- **FR-1.4.3**: Inline comments explaining each variable
- **FR-1.4.4**: Example values that clearly need replacement
- **FR-1.4.5**: Instructions comment at top of file

**Validation**:
- Configuration contains only Canvus credentials
- All AI parameters are hardcoded in application
- Template is self-documenting
- Clear distinction between required and optional variables

### 5. First-Run Model Download (Priority: MUST for Phase 2, SETUP in Phase 1)

**Requirement ID**: FR-1.5
**Description**: Implement automatic download of Bunny v1.1 model on first run if not bundled with installer

**Detailed Requirements**:
- **FR-1.5.1**: On startup, check if models/bunny-v1.1-llama-3-8b-v.gguf exists
- **FR-1.5.2**: If missing, initiate download from Hugging Face:
  - URL: https://huggingface.co/BAAI/Bunny-v1_1-Llama-3-8B-V/resolve/main/ggml-model-Q4_K_M.gguf
  - Alternative mirrors/CDN if available
- **FR-1.5.3**: Progress indication:
  - Console: ASCII progress bar with percentage and ETA
  - Web UI (if running): Progress updates via WebSocket
  - Log file: Periodic progress messages
- **FR-1.5.4**: Resumable downloads using HTTP Range headers
- **FR-1.5.5**: SHA256 checksum verification after download
- **FR-1.5.6**: Checksum file downloaded from Hugging Face or embedded in application
- **FR-1.5.7**: Download to temporary file, rename on success
- **FR-1.5.8**: Retry logic with exponential backoff (3 attempts)
- **FR-1.5.9**: Clear error messages:
  - Network connectivity issues
  - Insufficient disk space
  - Checksum verification failure
  - Download URL unavailable
- **FR-1.5.10**: Graceful degradation if download fails (display instructions for manual download)

**Implementation Notes (Phase 1)**:
- Create download infrastructure and progress UI
- Use placeholder URL for testing
- Full implementation in Phase 2 with actual model

**Validation**:
- Download completes successfully on clean installation
- Progress indication accurate
- Resume works after interruption
- Checksum verification catches corruption
- Error messages are helpful

### 6. Configuration Validation (Priority: MUST)

**Requirement ID**: FR-1.6
**Description**: Validate configuration at startup and provide helpful error messages before attempting operations

**Detailed Requirements**:
- **FR-1.6.1**: Startup validation checklist:
  - [ ] .env file exists
  - [ ] CANVUS_SERVER is set and valid URL format
  - [ ] CANVAS_ID is set
  - [ ] Either CANVUS_API_KEY OR (CANVUS_USERNAME AND CANVUS_PASSWORD) is set
  - [ ] CANVUS_SERVER is reachable (HTTP HEAD request)
  - [ ] Canvus authentication works (test API call)
  - [ ] CANVAS_ID exists and is accessible
- **FR-1.6.2**: If .env missing:
  ```
  ERROR: Configuration file not found

  Please copy .env.example to .env and configure:
    - CANVUS_SERVER: Your Canvus server URL
    - CANVAS_ID: Your canvas identifier
    - CANVUS_API_KEY: Your API key (or username/password)

  Location: C:\Program Files\CanvusLocalLLM\.env
  ```
- **FR-1.6.3**: If CANVUS_SERVER invalid:
  ```
  ERROR: Invalid Canvus server URL

  Current value: [value]

  Please set CANVUS_SERVER to a valid URL:
    Example: https://canvus.example.com
  ```
- **FR-1.6.4**: If authentication fails:
  ```
  ERROR: Canvus authentication failed

  Could not authenticate to [CANVUS_SERVER]

  Please verify:
    - CANVUS_API_KEY is correct
    OR
    - CANVUS_USERNAME and CANVUS_PASSWORD are correct

  Test your credentials at: [CANVUS_SERVER/api/...]
  ```
- **FR-1.6.5**: If CANVAS_ID not found:
  ```
  ERROR: Canvas not found

  Canvas ID [CANVAS_ID] does not exist or is not accessible.

  Please verify:
    - Canvas ID is correct
    - You have access to this canvas
    - Canvas is not archived/deleted
  ```
- **FR-1.6.6**: If validation passes:
  ```
  ✓ Configuration valid
  ✓ Connected to Canvus server: [CANVUS_SERVER]
  ✓ Canvas accessible: [CANVAS_NAME] (ID: [CANVAS_ID])
  ✓ Starting canvas monitoring...
  ```
- **FR-1.6.7**: All validation errors exit with non-zero status code
- **FR-1.6.8**: Validation happens before any heavy operations (model loading, GPU initialization)

**Validation**:
- Each error condition produces correct message
- Error messages include actionable instructions
- Success messages confirm all prerequisites
- Fast failure (<2 seconds) on configuration errors

### 7. Optional Service Creation (Priority: SHOULD)

**Requirement ID**: FR-1.7
**Description**: Enable CanvusLocalLLM to run as a background service with automatic startup

**Detailed Requirements - Windows**:
- **FR-1.7.1**: Use github.com/kardianos/service for Windows Service API
- **FR-1.7.2**: Service configuration:
  - Name: CanvusLocalLLM
  - Display Name: Canvus Local AI Integration
  - Description: Monitors Canvus workspaces and processes AI requests locally
  - Start Type: Automatic (delayed start)
  - Recovery: Restart on failure (3 attempts)
- **FR-1.7.3**: Service install command: `CanvusLocalLLM.exe install`
- **FR-1.7.4**: Service uninstall command: `CanvusLocalLLM.exe uninstall`
- **FR-1.7.5**: Service runs with configured credentials
- **FR-1.7.6**: Logs to app.log even when running as service
- **FR-1.7.7**: NSIS installer checkbox: "Install as Windows Service"
- **FR-1.7.8**: If checked, installer runs: `CanvusLocalLLM.exe install && sc start CanvusLocalLLM`

**Detailed Requirements - Linux**:
- **FR-1.7.9**: Generate systemd unit file template:
  ```ini
  [Unit]
  Description=Canvus Local AI Integration
  After=network.target

  [Service]
  Type=simple
  User=canvuslocallm
  Group=canvuslocallm
  WorkingDirectory=/opt/canvuslocallm
  ExecStart=/opt/canvuslocallm/bin/canvuslocallm
  Restart=on-failure
  RestartSec=10s

  [Install]
  WantedBy=multi-user.target
  ```
- **FR-1.7.10**: Include systemd unit in tarball/deb
- **FR-1.7.11**: postinst script offers to enable service
- **FR-1.7.12**: install.sh displays systemd installation instructions

**Validation**:
- Windows Service installs and starts successfully
- Service survives reboots
- Service recovery works after crashes
- Systemd unit file works on Ubuntu/Fedora/Arch
- Logs accessible when running as service

## Non-Functional Requirements

### Performance

**NFR-1**: Installation completes in <5 minutes on modern hardware with bundled model
**NFR-2**: Installation completes in <15 minutes including model download (8GB model @ 10MB/s)
**NFR-3**: Configuration validation completes in <2 seconds
**NFR-4**: Application startup (after validation) completes in <5 seconds (placeholder for Phase 2 model loading)

### Usability

**NFR-5**: No technical expertise required to install and configure
**NFR-6**: All error messages include actionable instructions
**NFR-7**: Progress indication for all long-running operations (installation, download)
**NFR-8**: README.txt included with clear setup instructions

### Reliability

**NFR-9**: Installation never leaves system in broken state
**NFR-10**: Download resume works after network interruption
**NFR-11**: Checksum verification catches corrupted downloads
**NFR-12**: Service automatic restart after crashes (configurable)

### Compatibility

**NFR-13**: Windows installer works on Windows 10/11 (64-bit)
**NFR-14**: Debian package works on Ubuntu 20.04+, Debian 11+
**NFR-15**: Tarball works on any Linux distribution with:
  - glibc 2.27+ (or musl)
  - NVIDIA GPU with CUDA 11.8+ drivers
**NFR-16**: NVIDIA RTX GPUs supported (20-series, 30-series, 40-series)

### Security

**NFR-17**: Configuration file permissions: 0600 (user read/write only)
**NFR-18**: No credentials in logs or error messages
**NFR-19**: Model downloads over HTTPS with checksum verification
**NFR-20**: Service runs with minimal privileges (not as SYSTEM/root)

### Maintainability

**NFR-21**: Single NSIS script for Windows installer
**NFR-22**: Automated build process for all distribution formats
**NFR-23**: Version number embedded in installer and packages
**NFR-24**: Uninstall removes all components cleanly

## Technical Constraints

### Hard Requirements
- **TC-1**: NVIDIA GPU with CUDA support required (no CPU fallback)
- **TC-2**: Minimum 8GB GPU VRAM for Bunny v1.1 model
- **TC-3**: Minimum 20GB disk space (10GB for model, 10GB for operations)
- **TC-4**: Go 1.21+ for compilation
- **TC-5**: NSIS 3.0+ for Windows installer compilation

### Platform-Specific
- **TC-6**: Windows: MSVC 2022 build tools for CGo compilation (Phase 2)
- **TC-7**: Linux: GCC 9+ and CUDA toolkit 11.8+ for CGo compilation (Phase 2)
- **TC-8**: Both platforms: CMake 3.20+ for llama.cpp/stable-diffusion.cpp builds (Phase 2)

### Model Requirements
- **TC-9**: Bunny v1.1 Llama-3-8B-V is the fixed model (no alternatives)
- **TC-10**: GGUF quantized format required for llama.cpp compatibility
- **TC-11**: Model file size: ~8GB (Q4_K_M quantization)

## Assumptions

**A-1**: Users have NVIDIA RTX GPUs with up-to-date CUDA drivers installed
**A-2**: Users can obtain Canvus API key or have username/password credentials
**A-3**: Canvus server is accessible from user's network
**A-4**: Users have admin/sudo access for installation (service-less operation possible without)
**A-5**: Internet connection available for model download (if not bundled)

## Dependencies

### External Dependencies
- **D-1**: Canvus server with API access
- **D-2**: Hugging Face for Bunny v1.1 model download
- **D-3**: NVIDIA CUDA drivers on target system

### Internal Dependencies (Future Phases)
- **D-4**: Phase 2 required for actual AI functionality (llama.cpp integration)
- **D-5**: Phase 3 required for image generation (stable-diffusion.cpp integration)

### Build Dependencies
- **D-6**: NSIS for Windows installer creation
- **D-7**: dpkg-deb for Debian package creation
- **D-8**: tar/gzip for tarball creation

## Out of Scope for Phase 1

The following are explicitly OUT OF SCOPE for Phase 1:
- **OOS-1**: Actual AI inference (Phase 2 - llama.cpp integration)
- **OOS-2**: Image generation (Phase 3 - stable-diffusion.cpp integration)
- **OOS-3**: Web UI implementation (Phase 5)
- **OOS-4**: Code signing for installers (Phase 6)
- **OOS-5**: Automated CI/CD for installer builds (Phase 6)
- **OOS-6**: Multi-canvas monitoring (Phase 5)
- **OOS-7**: Advanced error recovery and crash handling (Phase 6)
- **OOS-8**: Structured logging with metrics (Phase 6)

## Risks and Mitigations

### Risk 1: Model Download Failures
**Risk**: Users may experience slow or failed downloads for 8GB model file
**Impact**: High - blocks first-run experience
**Mitigation**:
- Implement resumable downloads
- Provide manual download instructions
- Consider bundling model with installer (increases installer size)
- Use CDN/mirror for downloads

### Risk 2: CUDA Driver Compatibility
**Risk**: Users may have outdated or incompatible CUDA drivers
**Impact**: High - application won't work
**Mitigation**:
- Clear error messages about driver requirements
- Startup check for CUDA availability
- Documentation with driver installation instructions
- Defer actual CUDA usage to Phase 2

### Risk 3: Disk Space Constraints
**Risk**: Users may not have 20GB available disk space
**Impact**: Medium - installation or download fails
**Mitigation**:
- Check disk space before model download
- Clear error message with space requirements
- Option to install to different drive

### Risk 4: Service Permission Issues
**Risk**: Service may not have permissions to access GPU or write logs
**Impact**: Medium - service fails to start or operate
**Mitigation**:
- Document service user requirements
- Test service installation thoroughly
- Provide troubleshooting guide
- Allow non-service operation as fallback

### Risk 5: Configuration Complexity
**Risk**: Even minimal configuration may confuse users
**Impact**: Medium - support burden increases
**Mitigation**:
- Extremely clear error messages
- README with step-by-step instructions
- Configuration validation with helpful feedback
- Example configurations

## Success Metrics

Post-launch metrics to validate Phase 1 success:

**M-1**: 95% of installations complete successfully on first attempt
**M-2**: Average time from download to first AI operation <10 minutes
**M-3**: <5% of support requests related to installation/configuration
**M-4**: 90% of users successfully connect to Canvus on first configuration attempt
**M-5**: Zero configuration questions related to AI models or providers

## Testing Requirements

### Unit Tests
- Configuration parsing and validation
- URL format validation
- File permission handling
- Service installation/uninstallation logic

### Integration Tests
- Full installation process on Windows 10/11
- Full installation process on Ubuntu 20.04/22.04
- Debian package installation
- Tarball extraction and install.sh
- Service creation and startup
- Configuration validation with actual Canvus server

### Manual Testing
- User acceptance testing with non-technical users
- Installation on clean systems
- Uninstallation completeness
- Error message clarity and helpfulness
- README instructions accuracy

### Performance Tests
- Installation time measurement
- Model download performance
- Configuration validation speed
- Service startup time

## Documentation Requirements

**DOC-1**: README.txt with:
- Quick start instructions
- Configuration steps
- Troubleshooting common issues
- Where to get help

**DOC-2**: .env.example with:
- Inline comments for each variable
- Example values
- Clear indication of required vs optional

**DOC-3**: Error messages that:
- Explain what went wrong
- Provide actionable next steps
- Include examples where helpful
- Reference documentation

**DOC-4**: Service documentation:
- How to install/uninstall service
- How to view service logs
- How to troubleshoot service issues

## Acceptance Criteria

Phase 1 is complete when:

1. [ ] Windows NSIS installer installs all components successfully
2. [ ] Windows installer optional service creation works
3. [ ] Debian package installs on Ubuntu 22.04
4. [ ] Tarball install.sh works on Fedora and Arch
5. [ ] .env.example contains only Canvus credentials
6. [ ] Configuration validation catches all error conditions with helpful messages
7. [ ] Model download infrastructure implemented (tested with placeholder)
8. [ ] Service installation works on Windows and Linux
9. [ ] All unit tests pass
10. [ ] All integration tests pass
11. [ ] Non-technical user completes installation successfully in <10 minutes
12. [ ] Uninstallers remove all components cleanly
13. [ ] README.txt is clear and complete
14. [ ] No AI model, provider, or inference configuration exposed to users
