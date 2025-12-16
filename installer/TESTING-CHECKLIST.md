# Manual Installation Testing Checklist

This document provides a comprehensive checklist for manually testing the CanvusLocalLLM installers across different platforms.

## Test Environments

### Windows Testing Environments
- [ ] Windows 10 (22H2 or later)
- [ ] Windows 11 (latest)

### Linux Testing Environments
- [ ] Ubuntu 20.04 LTS
- [ ] Ubuntu 22.04 LTS
- [ ] Fedora (latest)
- [ ] Arch Linux (latest)

---

## Windows NSIS Installer Tests

### Prerequisites
- Build the installer: `makensis installer/windows/canvusapi.nsi`
- Build the binary: `GOOS=windows GOARCH=amd64 go build -o bin/CanvusLocalLLM.exe .`
- Ensure LICENSE.txt exists in project root

### Installation Tests

#### WI-001: Fresh Installation
- [ ] Double-click installer to launch
- [ ] Welcome page displays correctly with product name and version
- [ ] License agreement page shows MIT license
- [ ] Accept license proceeds to directory selection
- [ ] Default directory is `C:\Program Files\CanvusLocalLLM`
- [ ] Custom directory selection works
- [ ] Components page shows all sections (Core, SD Support, Service, Desktop, Start Menu)
- [ ] Core Application is read-only (cannot be deselected)
- [ ] Installation completes without errors
- [ ] Progress bar advances smoothly

#### WI-002: File Installation Verification
After installation, verify:
- [ ] `%PROGRAMFILES%\CanvusLocalLLM\CanvusLocalLLM.exe` exists
- [ ] `%PROGRAMFILES%\CanvusLocalLLM\.env.example` exists
- [ ] `%PROGRAMFILES%\CanvusLocalLLM\LICENSE.txt` exists
- [ ] `%PROGRAMFILES%\CanvusLocalLLM\README.txt` exists
- [ ] `%PROGRAMFILES%\CanvusLocalLLM\Uninstall.exe` exists
- [ ] `lib\` directory created
- [ ] `models\` directory created
- [ ] `downloads\` directory created
- [ ] `logs\` directory created

#### WI-003: Registry Entries
Verify in Registry Editor (regedit):
- [ ] Key exists: `HKLM\Software\Microsoft\Windows\CurrentVersion\Uninstall\CanvusLocalLLM`
- [ ] DisplayName = "CanvusLocalLLM"
- [ ] DisplayVersion = "0.1.0"
- [ ] Publisher = "Jaypaul Bridger"
- [ ] UninstallString points to Uninstall.exe
- [ ] InstallLocation is correct

#### WI-004: Add/Remove Programs
- [ ] Open Windows Settings > Apps > Installed Apps
- [ ] CanvusLocalLLM appears in list
- [ ] Correct version displayed
- [ ] Correct publisher displayed
- [ ] Uninstall option available

#### WI-005: Desktop Shortcut (if selected)
- [ ] Shortcut appears on desktop
- [ ] Shortcut icon displays correctly
- [ ] Double-clicking launches application

#### WI-006: Start Menu Shortcuts (if selected)
- [ ] Start Menu folder created: `CanvusLocalLLM`
- [ ] Application shortcut works
- [ ] README shortcut opens README.txt
- [ ] Edit Configuration shortcut opens .env in Notepad
- [ ] View Logs shortcut opens logs folder
- [ ] Uninstall shortcut launches uninstaller

#### WI-007: Windows Service Installation (if selected)
- [ ] Service installs without error
- [ ] Service appears in services.msc as "CanvusLocalLLM"
- [ ] Service is not running (requires configuration first)
- [ ] Service can be started after configuring .env
- [ ] Service can be stopped via services.msc
- [ ] Service can be started via `sc start CanvusLocalLLM`

#### WI-008: Upgrade Installation
- [ ] Install older version first
- [ ] Run newer installer
- [ ] Installer detects existing installation
- [ ] Upgrade prompt appears
- [ ] Clicking OK proceeds with upgrade
- [ ] Previous configuration preserved
- [ ] Application upgraded successfully

### Uninstallation Tests

#### WU-001: Uninstaller Launch
- [ ] Run Uninstall.exe directly
- [ ] Confirmation dialog appears
- [ ] Clicking "No" cancels uninstall
- [ ] Clicking "Yes" proceeds with uninstall

#### WU-002: Service Removal
If service was installed:
- [ ] Service is stopped automatically
- [ ] Service is removed from services.msc

#### WU-003: File Removal
- [ ] Main executable removed
- [ ] .env.example removed
- [ ] Prompt for .env removal appears
- [ ] README.txt removed
- [ ] LICENSE.txt removed
- [ ] Uninstall.exe removed last
- [ ] Empty directories removed

#### WU-004: User Data Prompts
- [ ] Prompt appears for .env configuration file
- [ ] Prompt appears for SD model (if installed)
- [ ] Prompt appears for other models
- [ ] Prompt appears for downloads folder
- [ ] Prompt appears for logs folder
- [ ] User choices are respected

#### WU-005: Shortcut Removal
- [ ] Desktop shortcut removed
- [ ] Start Menu folder removed
- [ ] All Start Menu shortcuts removed

#### WU-006: Registry Cleanup
- [ ] Uninstall registry key removed
- [ ] No orphaned registry entries

#### WU-007: Add/Remove Programs Cleanup
- [ ] Application no longer appears in Installed Apps

---

## Debian Package (.deb) Tests

### Prerequisites
- Build the package with `dpkg-deb --build installer/debian canvuslocallm_1.0.0_amd64.deb`
- Have sudo access on test system

### Installation Tests

#### DI-001: Package Installation
```bash
sudo dpkg -i canvuslocallm_1.0.0_amd64.deb
```
- [ ] Installation completes without errors
- [ ] No missing dependencies

#### DI-002: File Installation Verification
After installation, verify:
- [ ] `/opt/canvuslocallm/canvuslocallm` exists and is executable
- [ ] `/opt/canvuslocallm/.env.example` exists
- [ ] `/opt/canvuslocallm/LICENSE.txt` exists
- [ ] `/opt/canvuslocallm/README.txt` exists
- [ ] `/opt/canvuslocallm/logs/` directory exists
- [ ] `/opt/canvuslocallm/downloads/` directory exists

#### DI-003: User and Group Creation
- [ ] User `canvusllm` exists: `getent passwd canvusllm`
- [ ] Group `canvusllm` exists: `getent group canvusllm`
- [ ] User is a system user (nologin shell)
- [ ] User home directory set to /opt/canvuslocallm

#### DI-004: Permissions Verification
- [ ] Binary owned by canvusllm:canvusllm
- [ ] Binary permissions are 0755
- [ ] .env permissions are 0640 (if created)
- [ ] logs/ owned by canvusllm:canvusllm
- [ ] downloads/ owned by canvusllm:canvusllm

#### DI-005: Systemd Service
- [ ] Service file exists: `/etc/systemd/system/canvuslocallm.service`
- [ ] Service is enabled: `systemctl is-enabled canvuslocallm`
- [ ] Service status shows not running (awaiting config)
- [ ] After configuring .env, service starts: `sudo systemctl start canvuslocallm`
- [ ] Service logs visible: `journalctl -u canvuslocallm`

#### DI-006: Package Info
```bash
dpkg -l | grep canvuslocallm
dpkg -s canvuslocallm
```
- [ ] Package shows as installed
- [ ] Version is correct
- [ ] Architecture is correct

### Uninstallation Tests

#### DU-001: Package Removal
```bash
sudo dpkg -r canvuslocallm
```
- [ ] Removal completes without errors
- [ ] Service stopped before removal
- [ ] Service disabled
- [ ] Binary removed
- [ ] Service file removed

#### DU-002: Configuration Files After Remove
- [ ] `.env` file preserved (if customized)
- [ ] `.env.example` may remain (conffile)
- [ ] logs/ preserved
- [ ] downloads/ preserved

#### DU-003: Purge Package
```bash
sudo dpkg --purge canvuslocallm
```
- [ ] All configuration files removed
- [ ] User canvusllm removed (or preserved based on policy)
- [ ] Group canvusllm removed (or preserved based on policy)
- [ ] /opt/canvuslocallm directory removed (if empty)

#### DU-004: Clean Uninstall Verification
- [ ] No orphaned files in /opt/canvuslocallm
- [ ] No orphaned service files
- [ ] Package no longer shows in dpkg -l

---

## Tarball Installation Tests

### Prerequisites
- Create tarball with required files:
  - canvuslocallm (binary)
  - install.sh
  - .env.example
  - LICENSE.txt
  - README.txt
  - canvuslocallm.service

### Installation Tests

#### TI-001: System Installation (with sudo)
```bash
sudo ./install.sh
```
- [ ] Installation completes without errors
- [ ] Default prefix is `/opt/canvuslocallm`
- [ ] User canvusllm created
- [ ] Group canvusllm created
- [ ] All files copied successfully
- [ ] Systemd service prompt appears
- [ ] Next steps displayed correctly

#### TI-002: User Installation (without sudo)
```bash
./install.sh --user
```
- [ ] Installation completes without errors
- [ ] Default prefix is `~/.local/canvuslocallm`
- [ ] No user/group created (uses current user)
- [ ] All files copied successfully
- [ ] No systemd prompt (user install)
- [ ] Next steps displayed correctly

#### TI-003: Custom Prefix Installation
```bash
sudo ./install.sh --prefix=/usr/local/canvuslocallm
```
- [ ] Installation to custom prefix works
- [ ] All files in custom location
- [ ] Service file adjusted for custom prefix

#### TI-004: Force Reinstall
```bash
sudo ./install.sh --force
```
- [ ] Existing installation detected
- [ ] No confirmation prompt (--force)
- [ ] Files overwritten successfully

#### TI-005: File Verification (System Install)
- [ ] `/opt/canvuslocallm/canvuslocallm` exists and executable
- [ ] `/opt/canvuslocallm/.env.example` exists
- [ ] `/opt/canvuslocallm/.env` created from example
- [ ] `/opt/canvuslocallm/LICENSE.txt` exists
- [ ] `/opt/canvuslocallm/README.txt` exists
- [ ] `/opt/canvuslocallm/logs/` exists
- [ ] `/opt/canvuslocallm/downloads/` exists

#### TI-006: File Verification (User Install)
- [ ] `~/.local/canvuslocallm/canvuslocallm` exists and executable
- [ ] `~/.local/canvuslocallm/.env` created from example
- [ ] Permissions are user-only (700 for dirs, 600 for .env)

#### TI-007: Systemd Service Installation (System Install)
When prompted "Install systemd service?" answer "y":
- [ ] Service file copied to `/etc/systemd/system/`
- [ ] Service enabled
- [ ] Service not started (needs config)
- [ ] `systemctl status canvuslocallm` shows enabled but inactive

### Error Handling Tests

#### TE-001: Missing Binary
```bash
rm canvuslocallm && ./install.sh
```
- [ ] Clear error message about missing binary
- [ ] Installation aborts cleanly

#### TE-002: No Root for System Install
```bash
./install.sh  # without sudo
```
- [ ] Clear error message requiring root
- [ ] Suggests --user option

#### TE-003: Invalid Option
```bash
./install.sh --invalid-option
```
- [ ] Clear error message about unknown option
- [ ] Suggests --help

---

## Cross-Platform Verification Matrix

| Feature | Windows 10 | Windows 11 | Ubuntu 20.04 | Ubuntu 22.04 | Fedora | Arch |
|---------|------------|------------|--------------|--------------|--------|------|
| Binary runs | | | | | | |
| Service installs | | | | | | |
| Service starts | | | | | | |
| Logs created | | | | | | |
| Uninstall clean | | | | | | |

---

## Known Issues and Workarounds

### Windows
1. **UAC Prompt**: Installer requires admin rights, UAC prompt expected
2. **Antivirus**: Some AV may flag the binary; may need to add exception
3. **Path Spaces**: Installation path with spaces should work but test carefully

### Linux
1. **SELinux (Fedora)**: May need to configure SELinux for service
2. **AppArmor (Ubuntu)**: May need AppArmor profile
3. **systemd version**: Older Ubuntu may have different systemd behavior

### Tarball
1. **No systemd**: On systems without systemd, service installation skipped
2. **Permissions**: If extracted with wrong umask, may need to fix permissions

---

## Test Sign-Off

### Windows Installer
- Tested by: _________________
- Date: _________________
- Version tested: _________________
- All tests pass: [ ] Yes [ ] No
- Notes:

### Debian Package
- Tested by: _________________
- Date: _________________
- Version tested: _________________
- All tests pass: [ ] Yes [ ] No
- Notes:

### Tarball Installer
- Tested by: _________________
- Date: _________________
- Version tested: _________________
- All tests pass: [ ] Yes [ ] No
- Notes:

---

## Appendix: Quick Test Commands

### Windows (PowerShell)
```powershell
# Check installation
Test-Path "$env:ProgramFiles\CanvusLocalLLM\CanvusLocalLLM.exe"

# Check service
Get-Service CanvusLocalLLM

# Check registry
Get-ItemProperty "HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\CanvusLocalLLM"
```

### Linux (Bash)
```bash
# Check installation (deb)
dpkg -s canvuslocallm

# Check installation (tarball)
ls -la /opt/canvuslocallm/

# Check service
systemctl status canvuslocallm

# Check user/group
getent passwd canvusllm
getent group canvusllm

# Check permissions
ls -la /opt/canvuslocallm/
```
