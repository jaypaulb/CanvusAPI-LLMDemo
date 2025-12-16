# Platform Testing Guide - Windows 10/11 and Ubuntu 22.04

## Overview

This document provides a comprehensive manual testing checklist for validating CanvusLocalLLM installers on Windows 10/11 and Ubuntu 22.04. The focus is on verifying stable-diffusion.cpp integration, CUDA acceleration, and image generation functionality.

## Prerequisites

### Windows 10/11 Testing
- Windows 10 (Build 19041+) or Windows 11
- NVIDIA GPU with CUDA support (RTX 20xx/30xx/40xx series)
- Minimum 8GB GPU memory
- Administrator privileges
- NVIDIA drivers installed (525+ recommended)
- Clean test environment (VM or fresh install recommended)

### Ubuntu 22.04 Testing
- Ubuntu 22.04 LTS
- NVIDIA GPU with CUDA support (RTX 20xx/30xx/40xx series)
- Minimum 8GB GPU memory
- sudo privileges
- NVIDIA drivers 525+ installed
- Clean test environment (VM or fresh install recommended)

### Verification Tools
- NVIDIA GPU monitoring: `nvidia-smi` (Linux/Windows)
- Task Manager (Windows) or `top`/`htop` (Linux) for system monitoring
- Image viewer (PNG support)
- Text editor for configuration

---

## Windows 10/11 Testing Checklist

### 1. Pre-Installation Verification

- [ ] Verify NVIDIA driver version: `nvidia-smi` in PowerShell
  - Expected: Driver version 525.x or higher
  - Note GPU model and memory

- [ ] Check available disk space
  - Required: ~13GB for full installation (includes bundled models)
  - Location: `C:\Program Files\CanvusLocalLLM\`

- [ ] Verify no previous installation exists
  - Check Add/Remove Programs for "CanvusLocalLLM"
  - Check `C:\Program Files\CanvusLocalLLM\` does not exist

### 2. Installer Execution

- [ ] Locate installer: `CanvusLocalLLM-{version}-Setup.exe`
- [ ] Run installer as Administrator (right-click → "Run as administrator")
- [ ] Verify installer UI appears with Modern UI 2 interface
- [ ] Accept license agreement
- [ ] Choose installation directory (default: `C:\Program Files\CanvusLocalLLM`)
- [ ] Select components:
  - [x] Core Application (required, cannot deselect)
  - [ ] Install as Windows Service (optional, test both scenarios)
  - [ ] Desktop Shortcut (optional)
  - [ ] Start Menu Shortcuts (optional)

- [ ] Monitor installation progress
  - Expected duration: 5-15 minutes (depending on disk speed)
  - Progress bar should show file extraction

- [ ] Verify no errors during installation
- [ ] Installation completes successfully with finish page

### 3. Post-Installation Verification

#### 3.1 File Structure
Navigate to installation directory and verify:

- [ ] `CanvusLocalLLM.exe` exists
- [ ] `.env.example` exists
- [ ] `README.txt` and `LICENSE.txt` exist
- [ ] `lib\stable-diffusion.dll` exists (~100-200MB)
- [ ] `models\sd-v1-5.safetensors` exists (~4GB)
- [ ] `Uninstall.exe` exists
- [ ] Directories created: `lib\`, `models\`, `downloads\`, `logs\`

#### 3.2 Configuration
- [ ] Copy `.env.example` to `.env`
- [ ] Edit `.env` with required settings:
  ```
  CANVUS_SERVER=https://your-canvus-server.com
  CANVAS_NAME=TestCanvas
  CANVAS_ID=your-canvas-id
  OPENAI_API_KEY=your-key-or-local-endpoint
  CANVUS_API_KEY=your-canvus-api-key

  # Stable Diffusion Configuration
  SD_MODEL_PATH=models/sd-v1-5.safetensors
  SD_POOL_SIZE=2
  SD_DEFAULT_STEPS=20
  SD_DEFAULT_CFG_SCALE=7.0
  ```

- [ ] Verify configuration syntax (no missing quotes, valid paths)

#### 3.3 Windows Service (if selected during install)
- [ ] Open Services (services.msc)
- [ ] Locate "CanvusLocalLLM" service
- [ ] Verify service status: "Running" or "Stopped" (depending on .env config)
- [ ] Check service properties:
  - Startup type: Automatic
  - Log on as: Local System
- [ ] Test service controls:
  - Stop service → Verify stops
  - Start service → Verify starts
  - Check `logs\app.log` for service startup messages

#### 3.4 Registry Entries
- [ ] Open Registry Editor (regedit.exe)
- [ ] Navigate to: `HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\CanvusLocalLLM`
- [ ] Verify keys exist:
  - DisplayName
  - DisplayVersion
  - Publisher
  - UninstallString
  - InstallLocation

### 4. Image Generation Testing

#### 4.1 CUDA Verification
- [ ] Open PowerShell or CMD in installation directory
- [ ] Run: `.\CanvusLocalLLM.exe` (if not running as service)
- [ ] Open second PowerShell window
- [ ] Run: `nvidia-smi` continuously: `while ($true) { cls; nvidia-smi; sleep 1 }`
- [ ] Watch for GPU utilization during image generation

#### 4.2 512x512 Image Generation
- [ ] In Canvus canvas, create a note widget
- [ ] Enter prompt: `{{image:512x512:a beautiful mountain landscape at sunset}}`
- [ ] Save note widget
- [ ] Monitor application logs: `logs\app.log`
- [ ] Expected log entries:
  - "Image generation request detected"
  - "Acquiring SD context from pool"
  - "Generating image with parameters: 512x512"
  - "Image generation completed in X seconds"
  - "Releasing SD context back to pool"

- [ ] Verify GPU utilization:
  - GPU memory usage increases (expect 4-6GB for SD v1.5)
  - GPU utilization spikes (70-100% during generation)
  - Generation time: 10-30 seconds (depends on GPU)

- [ ] Verify image appears in Canvus:
  - New image widget created near prompt widget
  - Image displays correctly (512x512 pixels)
  - Image matches prompt content
  - PNG format

- [ ] Download generated image and verify:
  - File size reasonable (50-200KB for 512x512)
  - Opens in image viewer
  - No corruption or artifacts

#### 4.3 768x768 Image Generation
- [ ] Create new note widget in Canvus
- [ ] Enter prompt: `{{image:768x768:a futuristic city with flying cars}}`
- [ ] Save note widget
- [ ] Monitor logs and GPU utilization (same as 4.2)
- [ ] Expected generation time: 20-60 seconds (larger resolution)
- [ ] Verify image appears in Canvus (768x768 pixels)
- [ ] Verify GPU memory usage (may be higher than 512x512)
- [ ] Download and verify image quality

#### 4.4 Concurrent Generation Testing
- [ ] Create 5 note widgets with different image prompts:
  1. `{{image:512x512:a red sports car}}`
  2. `{{image:512x512:a blue ocean wave}}`
  3. `{{image:512x512:a green forest}}`
  4. `{{image:512x512:a yellow sunset}}`
  5. `{{image:512x512:a purple mountain}}`

- [ ] Save all 5 widgets quickly (within 10 seconds)
- [ ] Monitor logs for queuing behavior:
  - "Acquiring SD context from pool" messages
  - Pool size limit (default: 2 contexts)
  - Requests should queue if pool is full
  - "Waiting for available context" messages

- [ ] Verify GPU utilization pattern:
  - Should maintain high utilization (70-100%)
  - No GPU crashes or out-of-memory errors
  - All 5 images eventually complete

- [ ] Verify all 5 images appear in Canvus
- [ ] Verify generation times are reasonable
- [ ] Check for any timeout or error messages in logs

#### 4.5 Error Handling Verification

##### 4.5.1 Invalid Prompt Test
- [ ] Create note widget with invalid prompt: `{{image:invalid:test}}`
- [ ] Verify error handling:
  - Application logs error message
  - No crash or hang
  - User receives error notification in Canvus (if implemented)

##### 4.5.2 Invalid Resolution Test
- [ ] Create note widget: `{{image:9999x9999:test}}`
- [ ] Verify error handling:
  - Application rejects invalid resolution
  - Logs error message
  - No CUDA errors or crashes

##### 4.5.3 Missing Model Test
- [ ] Stop application
- [ ] Rename `models\sd-v1-5.safetensors` to `models\sd-v1-5.safetensors.backup`
- [ ] Start application
- [ ] Verify error message in logs:
  - "Model file not found"
  - Application starts but SD features disabled
- [ ] Restore model file

##### 4.5.4 CUDA Error Simulation
- [ ] Monitor behavior if GPU is busy with other tasks
- [ ] Run GPU-intensive application (game, 3D rendering)
- [ ] Attempt image generation
- [ ] Verify graceful handling:
  - Queuing behavior works correctly
  - No crashes
  - May be slower but completes

### 5. Performance Benchmarks

#### 5.1 Generation Timing
Record generation times for different configurations:

| Resolution | Steps | CFG Scale | GPU Model | Generation Time | GPU Memory |
|------------|-------|-----------|-----------|-----------------|------------|
| 512x512    | 20    | 7.0       | RTX 3060  | _____ sec       | _____ GB   |
| 768x768    | 20    | 7.0       | RTX 3060  | _____ sec       | _____ GB   |
| 512x512    | 30    | 7.0       | RTX 3060  | _____ sec       | _____ GB   |

#### 5.2 Pool Size Impact
Test with different `SD_POOL_SIZE` values:

- [ ] Pool size 1: Concurrent requests → sequential processing
- [ ] Pool size 2: Concurrent requests → parallel processing (up to 2)
- [ ] Pool size 3+: Verify GPU memory can handle (may OOM on 8GB cards)

### 6. Uninstallation Testing

- [ ] Open Add/Remove Programs
- [ ] Locate "CanvusLocalLLM"
- [ ] Click "Uninstall"
- [ ] Uninstaller UI appears
- [ ] Confirm uninstallation
- [ ] Verify prompts for:
  - Remove configuration file (.env)? → Test both YES and NO
  - Remove downloaded models? → Test both YES and NO
  - Remove downloads folder? → Test both YES and NO
  - Remove logs? → Test both YES and NO

- [ ] Verify uninstallation completes
- [ ] Check installation directory:
  - If "YES" to all: Directory should be empty or removed
  - If "NO" to some: Verify selected files remain

- [ ] Verify registry entries removed
- [ ] Verify Start Menu shortcuts removed
- [ ] Verify Desktop shortcut removed (if created)

---

## Ubuntu 22.04 Testing Checklist

### 1. Pre-Installation Verification

- [ ] Verify NVIDIA driver version: `nvidia-smi`
  - Expected: Driver version 525.x or higher
  - Note GPU model and memory

- [ ] Check CUDA toolkit: `nvcc --version`
  - Expected: CUDA 11.0 or later
  - If not installed, installer should handle dependencies

- [ ] Check available disk space: `df -h /opt`
  - Required: ~4.5GB for installation
  - Location: `/opt/canvuslocallm/`

- [ ] Verify no previous installation:
  ```bash
  dpkg -l | grep canvuslocallm
  ls -la /opt/canvuslocallm/
  ```

### 2. Package Installation

- [ ] Locate package: `canvuslocallm_1.0.0_amd64.deb`
- [ ] Install package:
  ```bash
  sudo dpkg -i canvuslocallm_1.0.0_amd64.deb
  ```

- [ ] Check for dependency errors:
  - If errors occur, run: `sudo apt-get install -f`
  - Verify CUDA toolkit dependency installed

- [ ] Verify installation success:
  ```bash
  dpkg -l | grep canvuslocallm
  ```
  - Status should show "ii" (installed)

### 3. Post-Installation Verification

#### 3.1 File Structure
Verify installation files:

```bash
ls -la /opt/canvuslocallm/
```

- [ ] `canvuslocallm` binary exists
- [ ] `.env.example` exists
- [ ] `README.md` and `LICENSE` exist
- [ ] `lib/libstable-diffusion.so` exists (~100-200MB)
- [ ] `models/sd-v1-5.safetensors` exists (~4GB)
- [ ] Directories exist: `lib/`, `models/`, `downloads/`, `logs/`

#### 3.2 File Permissions
- [ ] Binary is executable: `ls -l /opt/canvuslocallm/canvuslocallm`
  - Should show: `-rwxr-xr-x`
- [ ] Check ownership: `ls -l /opt/canvuslocallm/`
  - Should be owned by root or canvuslocallm user

#### 3.3 Configuration
- [ ] Copy configuration template:
  ```bash
  sudo cp /opt/canvuslocallm/.env.example /opt/canvuslocallm/.env
  ```

- [ ] Edit configuration:
  ```bash
  sudo nano /opt/canvuslocallm/.env
  ```

- [ ] Set required variables:
  ```
  CANVUS_SERVER=https://your-canvus-server.com
  CANVAS_NAME=TestCanvas
  CANVAS_ID=your-canvas-id
  OPENAI_API_KEY=your-key-or-local-endpoint
  CANVUS_API_KEY=your-canvus-api-key

  # Stable Diffusion Configuration
  SD_MODEL_PATH=/opt/canvuslocallm/models/sd-v1-5.safetensors
  SD_POOL_SIZE=2
  SD_DEFAULT_STEPS=20
  SD_DEFAULT_CFG_SCALE=7.0
  ```

#### 3.4 Systemd Service (if included)
- [ ] Check service file: `sudo systemctl status canvuslocallm`
- [ ] Enable service: `sudo systemctl enable canvuslocallm`
- [ ] Start service: `sudo systemctl start canvuslocallm`
- [ ] Verify running: `sudo systemctl status canvuslocallm`
  - Status should show "active (running)"

- [ ] Check logs: `sudo journalctl -u canvuslocallm -f`
  - Should show startup messages
  - Look for SD initialization messages

### 4. Library Dependencies Verification

- [ ] Check library loading:
  ```bash
  ldd /opt/canvuslocallm/canvuslocallm
  ```
  - All dependencies should resolve (no "not found" errors)

- [ ] Verify CUDA library linkage:
  ```bash
  ldd /opt/canvuslocallm/lib/libstable-diffusion.so | grep cuda
  ```
  - Should show CUDA runtime libraries

- [ ] Test binary execution:
  ```bash
  cd /opt/canvuslocallm
  sudo ./canvuslocallm
  ```
  - Should start without immediate errors
  - Check logs for CUDA initialization

### 5. Image Generation Testing

#### 5.1 CUDA Verification
- [ ] Terminal 1: Start application (if not running as service)
  ```bash
  cd /opt/canvuslocallm
  sudo ./canvuslocallm
  ```

- [ ] Terminal 2: Monitor GPU:
  ```bash
  watch -n 1 nvidia-smi
  ```
  - Watch for GPU utilization during generation

#### 5.2 512x512 Image Generation
- [ ] In Canvus canvas, create note widget
- [ ] Enter prompt: `{{image:512x512:a beautiful mountain landscape at sunset}}`
- [ ] Save note widget
- [ ] Monitor logs:
  ```bash
  sudo tail -f /opt/canvuslocallm/logs/app.log
  ```

- [ ] Expected log entries:
  - "Image generation request detected"
  - "Acquiring SD context from pool"
  - "Generating image with parameters: 512x512"
  - "Image generation completed in X seconds"

- [ ] Monitor GPU utilization (nvidia-smi):
  - GPU memory increases (4-6GB for SD v1.5)
  - GPU utilization: 70-100%
  - Generation time: 10-30 seconds

- [ ] Verify image appears in Canvus:
  - New image widget created
  - 512x512 resolution
  - PNG format
  - Matches prompt

#### 5.3 768x768 Image Generation
- [ ] Create note widget: `{{image:768x768:a futuristic city with flying cars}}`
- [ ] Monitor logs and GPU (same as 5.2)
- [ ] Expected generation time: 20-60 seconds
- [ ] Verify 768x768 image in Canvus

#### 5.4 Concurrent Generation Testing
- [ ] Create 5 note widgets with different prompts:
  1. `{{image:512x512:a red sports car}}`
  2. `{{image:512x512:a blue ocean wave}}`
  3. `{{image:512x512:a green forest}}`
  4. `{{image:512x512:a yellow sunset}}`
  5. `{{image:512x512:a purple mountain}}`

- [ ] Save all 5 quickly
- [ ] Monitor logs for queuing:
  - Pool size limit enforcement
  - Request queuing behavior
  - Context acquisition/release

- [ ] Monitor GPU:
  - Sustained high utilization
  - No OOM errors
  - All generations complete

- [ ] Verify all 5 images appear in Canvus

#### 5.5 Error Handling Verification

##### 5.5.1 Invalid Prompt Test
- [ ] Test: `{{image:invalid:test}}`
- [ ] Verify error logged, no crash

##### 5.5.2 Invalid Resolution Test
- [ ] Test: `{{image:9999x9999:test}}`
- [ ] Verify rejection, error logged

##### 5.5.3 Missing Model Test
- [ ] Stop service: `sudo systemctl stop canvuslocallm`
- [ ] Rename model:
  ```bash
  sudo mv /opt/canvuslocallm/models/sd-v1-5.safetensors \
          /opt/canvuslocallm/models/sd-v1-5.safetensors.backup
  ```
- [ ] Start service: `sudo systemctl start canvuslocallm`
- [ ] Verify error in logs: "Model file not found"
- [ ] Restore model:
  ```bash
  sudo mv /opt/canvuslocallm/models/sd-v1-5.safetensors.backup \
          /opt/canvuslocallm/models/sd-v1-5.safetensors
  ```

##### 5.5.4 Permission Test
- [ ] Remove write permission from downloads:
  ```bash
  sudo chmod 555 /opt/canvuslocallm/downloads
  ```
- [ ] Attempt image generation
- [ ] Verify error handling (should log permission error)
- [ ] Restore permissions:
  ```bash
  sudo chmod 755 /opt/canvuslocallm/downloads
  ```

### 6. Performance Benchmarks

#### 6.1 Generation Timing
Record generation times:

| Resolution | Steps | CFG Scale | GPU Model | Generation Time | GPU Memory |
|------------|-------|-----------|-----------|-----------------|------------|
| 512x512    | 20    | 7.0       | RTX 3060  | _____ sec       | _____ GB   |
| 768x768    | 20    | 7.0       | RTX 3060  | _____ sec       | _____ GB   |
| 512x512    | 30    | 7.0       | RTX 3060  | _____ sec       | _____ GB   |

#### 6.2 Pool Size Impact
Test different pool sizes:

- [ ] Edit `.env`: `SD_POOL_SIZE=1`
- [ ] Restart: `sudo systemctl restart canvuslocallm`
- [ ] Test concurrent requests → Sequential processing

- [ ] Edit `.env`: `SD_POOL_SIZE=2`
- [ ] Restart and test → Parallel processing (up to 2)

- [ ] Edit `.env`: `SD_POOL_SIZE=3`
- [ ] Restart and test → May OOM on 8GB cards

#### 6.3 CPU vs GPU Performance
- [ ] Force CPU mode (if supported): Add to `.env`:
  ```
  CUDA_VISIBLE_DEVICES=-1
  ```
- [ ] Restart and test generation
- [ ] Record timing (should be MUCH slower, 5-10x)
- [ ] Remove line to re-enable GPU

### 7. System Integration Testing

#### 7.1 Service Lifecycle
- [ ] Stop service: `sudo systemctl stop canvuslocallm`
- [ ] Verify stopped: `sudo systemctl status canvuslocallm`
- [ ] Start service: `sudo systemctl start canvuslocallm`
- [ ] Verify running: `sudo systemctl status canvuslocallm`
- [ ] Restart service: `sudo systemctl restart canvuslocallm`
- [ ] Verify restart: Check logs for restart message

#### 7.2 Boot Persistence
- [ ] Enable service: `sudo systemctl enable canvuslocallm`
- [ ] Reboot system: `sudo reboot`
- [ ] After boot, verify service running:
  ```bash
  sudo systemctl status canvuslocallm
  ```

#### 7.3 Log Rotation (if implemented)
- [ ] Check log files: `ls -lh /opt/canvuslocallm/logs/`
- [ ] Generate logs (run multiple generations)
- [ ] Verify log rotation configuration (if exists)

### 8. Uninstallation Testing

- [ ] Stop service:
  ```bash
  sudo systemctl stop canvuslocallm
  sudo systemctl disable canvuslocallm
  ```

- [ ] Remove package:
  ```bash
  sudo dpkg -r canvuslocallm
  ```

- [ ] Verify removal:
  ```bash
  dpkg -l | grep canvuslocallm
  ```
  - Should show "rc" (removed, config remains) or not appear

- [ ] Check filesystem:
  ```bash
  ls /opt/canvuslocallm/
  ```
  - May still contain config and models (by design)

- [ ] Purge completely (remove configs):
  ```bash
  sudo dpkg -P canvuslocallm
  sudo rm -rf /opt/canvuslocallm/
  ```

- [ ] Verify complete removal:
  ```bash
  ls /opt/canvuslocallm/
  dpkg -l | grep canvuslocallm
  ```

---

## Common Issues and Troubleshooting

### Windows

#### Issue: Installer fails to extract files
- **Cause**: Antivirus blocking, insufficient disk space
- **Solution**: Temporarily disable antivirus, check disk space

#### Issue: DLL not found errors
- **Cause**: stable-diffusion.dll not bundled or extracted
- **Solution**: Verify `lib\stable-diffusion.dll` exists, re-install

#### Issue: CUDA initialization fails
- **Cause**: NVIDIA drivers outdated or not installed
- **Solution**: Update to driver 525+, reboot

#### Issue: Out of memory during generation
- **Cause**: GPU has insufficient VRAM
- **Solution**: Reduce `SD_POOL_SIZE` to 1, close other GPU applications

#### Issue: Service won't start
- **Cause**: Invalid `.env` configuration
- **Solution**: Check logs, verify .env syntax, check API keys

### Ubuntu

#### Issue: dpkg dependency errors
- **Cause**: Missing CUDA toolkit or system libraries
- **Solution**: Run `sudo apt-get install -f`

#### Issue: .so library not found
- **Cause**: libstable-diffusion.so not bundled or incorrect path
- **Solution**: Verify `lib/libstable-diffusion.so` exists, check `ldd` output

#### Issue: Permission denied errors
- **Cause**: Incorrect file permissions
- **Solution**: Check ownership, run with sudo, or create dedicated user

#### Issue: CUDA not available
- **Cause**: NVIDIA drivers not installed or outdated
- **Solution**: Install nvidia-driver-525+, reboot

#### Issue: Service fails to start
- **Cause**: Configuration errors, missing dependencies
- **Solution**: Check `journalctl -u canvuslocallm`, verify .env

---

## Test Results Template

### Windows 10/11 Test Results

**Test Environment:**
- OS Version: Windows 10/11 Build _____
- GPU: NVIDIA RTX _____
- GPU Memory: _____ GB
- Driver Version: _____
- CUDA Version: _____
- Tester: _____
- Date: _____

**Installation:**
- Installer executed successfully: YES / NO
- All files bundled correctly: YES / NO
- Windows Service installed: YES / NO / N/A
- Registry entries created: YES / NO

**Image Generation:**
- 512x512 generation: PASS / FAIL (Time: _____ sec)
- 768x768 generation: PASS / FAIL (Time: _____ sec)
- Concurrent generation (5 images): PASS / FAIL
- CUDA acceleration verified: YES / NO
- GPU utilization observed: YES / NO (Peak: _____%)
- GPU memory usage: _____ GB

**Error Handling:**
- Invalid prompt handling: PASS / FAIL
- Invalid resolution handling: PASS / FAIL
- Missing model handling: PASS / FAIL
- CUDA error handling: PASS / FAIL

**Performance:**
| Resolution | Steps | Time | GPU Memory |
|------------|-------|------|------------|
| 512x512    | 20    |      |            |
| 768x768    | 20    |      |            |

**Uninstallation:**
- Uninstaller executed: YES / NO
- Files removed correctly: YES / NO
- Registry cleaned: YES / NO

**Issues Found:**
1. _____
2. _____

**Overall Status:** PASS / FAIL

---

### Ubuntu 22.04 Test Results

**Test Environment:**
- OS Version: Ubuntu 22.04.___
- GPU: NVIDIA RTX _____
- GPU Memory: _____ GB
- Driver Version: _____
- CUDA Version: _____
- Tester: _____
- Date: _____

**Installation:**
- Package installed successfully: YES / NO
- Dependencies resolved: YES / NO
- All files present: YES / NO
- Service installed: YES / NO / N/A

**Image Generation:**
- 512x512 generation: PASS / FAIL (Time: _____ sec)
- 768x768 generation: PASS / FAIL (Time: _____ sec)
- Concurrent generation (5 images): PASS / FAIL
- CUDA acceleration verified: YES / NO
- GPU utilization observed: YES / NO (Peak: _____%)
- GPU memory usage: _____ GB

**Error Handling:**
- Invalid prompt handling: PASS / FAIL
- Invalid resolution handling: PASS / FAIL
- Missing model handling: PASS / FAIL
- Permission error handling: PASS / FAIL

**Performance:**
| Resolution | Steps | Time | GPU Memory |
|------------|-------|------|------------|
| 512x512    | 20    |      |            |
| 768x768    | 20    |      |            |

**System Integration:**
- Service lifecycle: PASS / FAIL
- Boot persistence: PASS / FAIL
- Log rotation: PASS / FAIL / N/A

**Uninstallation:**
- Package removed: YES / NO
- Files cleaned: YES / NO

**Issues Found:**
1. _____
2. _____

**Overall Status:** PASS / FAIL

---

## Sign-off

**Windows Testing:**
- Tester Name: _____________________
- Signature: _____________________
- Date: _____________________

**Ubuntu Testing:**
- Tester Name: _____________________
- Signature: _____________________
- Date: _____________________

**Approved By:**
- Name: _____________________
- Role: _____________________
- Signature: _____________________
- Date: _____________________
