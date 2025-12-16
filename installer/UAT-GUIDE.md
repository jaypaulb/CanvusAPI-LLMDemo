# User Acceptance Testing (UAT) Guide

## Overview

This guide is designed for **non-technical users** to verify that CanvusLocalLLM can be installed, configured, and used successfully in real-world scenarios. The goal is to ensure the installation process is smooth, intuitive, and recoverable from common errors.

## Time Requirements

- **Windows Installation**: Should complete in **< 10 minutes**
- **Linux Installation**: Should complete in **< 15 minutes**
- **Recovery from Errors**: Should be resolvable in **< 5 minutes** with clear error messages

## UAT Test Scenarios

### Scenario 1: Fresh Windows Installation (Non-Technical User)

**Persona**: Marketing manager with basic Windows knowledge, no programming experience

**Prerequisites**:
- Windows 10 or 11 computer
- Downloaded installer file: `CanvusLocalLLM-Setup.exe`
- No previous installation
- User has admin rights (can install software)

**Steps**:
1. **Download and Launch** (Target: 1 min)
   - [ ] Double-click installer file
   - [ ] Windows SmartScreen warning appears (expected)
   - [ ] Click "More info" → "Run anyway"
   - [ ] Installer window opens without errors

2. **Installation Wizard** (Target: 3 min)
   - [ ] Welcome screen is clear and professional
   - [ ] License agreement is readable (MIT license)
   - [ ] Click "I Agree" button works
   - [ ] Installation location shows: `C:\Program Files\CanvusLocalLLM`
   - [ ] "Browse" button works if user wants custom location
   - [ ] Component selection screen shows:
     - Core Application (required, cannot uncheck)
     - Stable Diffusion Support (optional)
     - Install as Windows Service (recommended)
     - Create Desktop Shortcut (optional)
     - Create Start Menu Shortcuts (optional)
   - [ ] Default selections are reasonable for typical user

3. **Installation Progress** (Target: 2 min)
   - [ ] Progress bar advances smoothly
   - [ ] No error popups appear
   - [ ] Completion screen shows success message
   - [ ] "Show README" checkbox is checked by default
   - [ ] Click "Finish" button

4. **Post-Installation Verification** (Target: 3 min)
   - [ ] Desktop shortcut appears (if selected)
   - [ ] Start Menu folder "CanvusLocalLLM" exists
   - [ ] README file opens automatically (if checkbox was checked)
   - [ ] README explains next steps clearly:
     - How to configure API keys
     - Where to find the .env file
     - How to start the service
   - [ ] "Edit Configuration" shortcut opens .env file in Notepad

5. **First Run Configuration** (Target: 1 min)
   - [ ] User can find and edit .env file easily
   - [ ] Instructions explain what each setting does
   - [ ] Example values help user understand format

**Success Criteria**:
- [ ] Total time: **< 10 minutes**
- [ ] User felt confident during installation
- [ ] User knows what to do next after installation
- [ ] No confusing error messages appeared
- [ ] User can locate application after installation

**Common User Questions to Test**:
- [ ] "Where did it install?"
- [ ] "How do I configure it?"
- [ ] "How do I start it?"
- [ ] "How do I know it's running?"
- [ ] "How do I uninstall it?"

---

### Scenario 2: Windows Service Management (Non-Technical User)

**Persona**: Same marketing manager, trying to start the service for first time

**Prerequisites**:
- Fresh installation from Scenario 1
- User has configured .env file with API keys

**Steps**:
1. **Finding the Service** (Target: 1 min)
   - [ ] User opens Services app via Start Menu search "Services"
   - [ ] "CanvusLocalLLM" appears in alphabetical list
   - [ ] Service description is clear and helpful

2. **Starting the Service** (Target: 2 min)
   - [ ] Right-click service → "Start"
   - [ ] Service starts without errors
   - [ ] Status changes to "Running"
   - [ ] OR: Clear error message if configuration is invalid
     - [ ] Error message explains what's wrong
     - [ ] Error message tells user how to fix it

3. **Verifying Service is Working** (Target: 1 min)
   - [ ] "View Logs" shortcut from Start Menu opens logs folder
   - [ ] Recent log file exists with today's date
   - [ ] Log shows "Server started" or similar success message
   - [ ] No scary-looking error messages

**Success Criteria**:
- [ ] User successfully started service in **< 5 minutes**
- [ ] User can verify service is running
- [ ] Error messages (if any) were helpful, not cryptic

---

### Scenario 3: Fresh Linux Installation - Ubuntu Desktop User

**Persona**: Graphic designer comfortable with Ubuntu desktop, minimal command-line experience

**Prerequisites**:
- Ubuntu 20.04 or 22.04 Desktop
- Downloaded: `canvuslocallm_1.0.0_amd64.deb`
- User knows how to open Terminal

**Steps**:
1. **Installation via GUI** (Target: 3 min)
   - [ ] Double-click .deb file
   - [ ] Software Center or GDebi opens
   - [ ] "Install" button is visible
   - [ ] Click "Install" button
   - [ ] Password prompt appears (expected for sudo)
   - [ ] Enter password
   - [ ] Installation completes with success message

2. **Alternative: Command-Line Installation** (Target: 2 min)
   ```bash
   sudo dpkg -i canvuslocallm_1.0.0_amd64.deb
   ```
   - [ ] User can copy-paste command from README
   - [ ] Installation shows progress
   - [ ] No errors appear
   - [ ] Success message is clear

3. **Configuration** (Target: 5 min)
   - [ ] User can find installation location: `/opt/canvuslocallm/`
   - [ ] README or install output explains how to configure
   - [ ] User copies .env.example to .env
   - [ ] User edits .env with text editor (gedit, nano, etc.)
   - [ ] File permissions allow editing (may need sudo)

4. **Starting the Service** (Target: 3 min)
   ```bash
   sudo systemctl start canvuslocallm
   sudo systemctl status canvuslocallm
   ```
   - [ ] User can copy-paste commands from README
   - [ ] Service starts successfully
   - [ ] Status shows "active (running)" in green
   - [ ] OR: Clear error message with fix instructions

5. **Verification** (Target: 2 min)
   - [ ] Logs exist: `/opt/canvuslocallm/logs/app.log`
   - [ ] User can view logs with `cat` or text editor
   - [ ] Logs show successful startup

**Success Criteria**:
- [ ] Total time: **< 15 minutes**
- [ ] User completed installation without getting stuck
- [ ] User knows how to start/stop service
- [ ] User can find logs if something goes wrong

---

### Scenario 4: Linux Installation - Server/Tarball (Technical User)

**Persona**: Sysadmin deploying to Fedora server without GUI

**Prerequisites**:
- Fedora server with SSH access
- Downloaded: `canvuslocallm-linux-amd64.tar.gz`
- Root or sudo access

**Steps**:
1. **Extract and Install** (Target: 5 min)
   ```bash
   tar -xzf canvuslocallm-linux-amd64.tar.gz
   cd canvuslocallm-linux-amd64
   sudo ./install.sh
   ```
   - [ ] Extraction works without errors
   - [ ] install.sh is executable
   - [ ] Install script shows clear prompts
   - [ ] Default prefix `/opt/canvuslocallm` is reasonable
   - [ ] Script asks about systemd service installation
   - [ ] Script creates user/group automatically
   - [ ] Script sets correct permissions

2. **Configuration** (Target: 5 min)
   - [ ] .env file created from .env.example
   - [ ] User can edit with vi/nano
   - [ ] Configuration instructions are clear

3. **Service Setup** (Target: 3 min)
   - [ ] Service file installed to `/etc/systemd/system/`
   - [ ] Service enabled for auto-start
   - [ ] Service starts without errors
   - [ ] journalctl shows logs correctly

**Success Criteria**:
- [ ] Total time: **< 15 minutes**
- [ ] Installation can be scripted/automated
- [ ] Follows Linux best practices (dedicated user, proper permissions)

---

### Scenario 5: Error Recovery - Missing Configuration

**Persona**: Any user who tries to run without configuring .env

**Steps**:
1. **Start Without Configuration** (Target: 1 min)
   - [ ] User tries to start service/application
   - [ ] Clear error message appears immediately
   - [ ] Error explains: "Configuration file not found or invalid"

2. **Error Message Quality**
   - [ ] Error message is in plain English (not technical jargon)
   - [ ] Error tells user WHAT went wrong
   - [ ] Error tells user HOW to fix it
   - [ ] Error points to specific file location

3. **Recovery** (Target: 3 min)
   - [ ] User follows instructions from error message
   - [ ] User creates/edits .env file
   - [ ] User tries again
   - [ ] Application starts successfully

**Success Criteria**:
- [ ] Error recovery in **< 5 minutes**
- [ ] User didn't need to search online for solution
- [ ] Error message was helpful, not scary

---

### Scenario 6: Error Recovery - Invalid API Key

**Persona**: User who entered wrong API key in .env

**Steps**:
1. **Start With Invalid Key** (Target: 1 min)
   - [ ] User starts service with typo in OPENAI_API_KEY
   - [ ] Service starts but first API call fails
   - [ ] Error logged clearly in log file

2. **Error Message Quality**
   - [ ] Log message identifies: "Authentication failed: Invalid API key"
   - [ ] Log points to which API key failed (OpenAI vs Canvus)
   - [ ] Log suggests checking .env configuration

3. **Recovery** (Target: 3 min)
   - [ ] User opens .env file
   - [ ] User corrects API key
   - [ ] User restarts service
   - [ ] Service works correctly

**Success Criteria**:
- [ ] Error recovery in **< 5 minutes**
- [ ] User could identify which API key was wrong
- [ ] User didn't need external help

---

### Scenario 7: Error Recovery - Port Already in Use

**Persona**: User with another service on port 1234

**Steps**:
1. **Start With Port Conflict** (Target: 1 min)
   - [ ] Another service is using port 1234
   - [ ] User tries to start CanvusLocalLLM
   - [ ] Clear error: "Cannot bind to port 1234: address already in use"

2. **Error Message Quality**
   - [ ] Error explains port is in use
   - [ ] Error suggests checking .env PORT setting
   - [ ] Error suggests checking for other services

3. **Recovery** (Target: 3 min)
   - [ ] User changes PORT in .env to 1235
   - [ ] User restarts service
   - [ ] Service binds to new port successfully

**Success Criteria**:
- [ ] Error recovery in **< 5 minutes**
- [ ] User understood port conflict concept
- [ ] User successfully resolved without technical support

---

### Scenario 8: Uninstallation - Windows

**Persona**: User who needs to uninstall completely

**Steps**:
1. **Find Uninstaller** (Target: 1 min)
   - [ ] User opens Settings → Apps → Installed Apps
   - [ ] "CanvusLocalLLM" appears in list
   - [ ] Click "..." → "Uninstall"
   - [ ] OR: Use Start Menu → CanvusLocalLLM → Uninstall

2. **Uninstall Process** (Target: 2 min)
   - [ ] Confirmation dialog appears
   - [ ] Dialog asks about keeping user data (logs, config, models)
   - [ ] Checkboxes are clearly labeled
   - [ ] User chooses what to keep/remove
   - [ ] Uninstall completes without errors

3. **Verification** (Target: 1 min)
   - [ ] Application no longer in Installed Apps
   - [ ] Desktop shortcut removed (if existed)
   - [ ] Start Menu folder removed
   - [ ] User data kept/removed per user choice

**Success Criteria**:
- [ ] Total time: **< 5 minutes**
- [ ] User had control over data removal
- [ ] Clean uninstall with no orphaned files (unless user chose to keep)

---

### Scenario 9: Uninstallation - Linux (Debian)

**Persona**: Ubuntu user removing package

**Steps**:
1. **Remove Package** (Target: 2 min)
   ```bash
   sudo dpkg -r canvuslocallm
   ```
   - [ ] Package removal shows progress
   - [ ] Service stopped automatically
   - [ ] Binary and service files removed
   - [ ] Configuration preserved (expected behavior)

2. **Purge (Complete Removal)** (Target: 1 min)
   ```bash
   sudo dpkg --purge canvuslocallm
   ```
   - [ ] All configuration files removed
   - [ ] User/group removed (if policy allows)
   - [ ] No orphaned files in /opt/canvuslocallm

**Success Criteria**:
- [ ] Total time: **< 5 minutes**
- [ ] Clear distinction between remove and purge
- [ ] User data handling matches user expectations

---

## UAT Acceptance Criteria

### Installation Experience
- [ ] **Windows**: ≤ 10 minutes from download to running service
- [ ] **Linux**: ≤ 15 minutes from download to running service
- [ ] **Clarity**: User knows what to do at each step without external help
- [ ] **Safety**: No unexpected prompts or scary warnings
- [ ] **Flexibility**: User can customize installation location

### Error Handling
- [ ] **Error Recovery**: Any error resolvable in < 5 minutes
- [ ] **Error Messages**: Plain English, not technical jargon
- [ ] **Error Guidance**: Each error tells user how to fix it
- [ ] **Error Context**: User can identify which setting/file is wrong

### Documentation
- [ ] **README Quality**: Non-technical user can follow without confusion
- [ ] **Configuration Help**: Each .env variable has clear explanation
- [ ] **Troubleshooting**: Common issues documented with solutions
- [ ] **Next Steps**: User knows how to verify installation succeeded

### Usability
- [ ] **Shortcuts Work**: Desktop/Start Menu shortcuts launch correctly
- [ ] **Logs Accessible**: User can find and read log files
- [ ] **Service Management**: User can start/stop/restart service
- [ ] **Uninstall Clean**: Complete removal when desired

### Platform-Specific
#### Windows
- [ ] No console window flashing when running as service
- [ ] Service appears in Services app with clear description
- [ ] Registry entries are appropriate and clean
- [ ] Add/Remove Programs entry is complete

#### Linux
- [ ] Follows FHS (Filesystem Hierarchy Standard)
- [ ] systemd service works on Ubuntu, Fedora, Arch
- [ ] Dedicated user/group for security
- [ ] Logs integrate with journalctl

---

## UAT Test Session Template

**Tester Name**: _________________
**Date**: _________________
**Platform**: [ ] Windows 10  [ ] Windows 11  [ ] Ubuntu 20.04  [ ] Ubuntu 22.04  [ ] Fedora  [ ] Arch
**Technical Level**: [ ] Non-technical  [ ] Basic  [ ] Intermediate  [ ] Advanced
**Version Tested**: _________________

### Session Goals
- [ ] Fresh installation
- [ ] Error recovery testing
- [ ] Uninstallation testing

### Scenario Results

| Scenario | Completed? | Time Taken | Issues Found | Severity (Low/Med/High) |
|----------|------------|------------|--------------|-------------------------|
| Fresh Install | | | | |
| Service Management | | | | |
| Configuration | | | | |
| Error Recovery | | | | |
| Uninstallation | | | | |

### Overall Impressions

**What worked well:**
-
-
-

**What was confusing:**
-
-
-

**What should be improved:**
-
-
-

**Would you recommend this installation process?** [ ] Yes  [ ] No  [ ] With changes

**Additional Comments:**




---

## UAT Sign-Off

### Windows UAT
- [ ] Non-technical user test completed successfully
- [ ] All time targets met (< 10 min installation)
- [ ] Error recovery scenarios tested
- [ ] Tester Name: _________________
- [ ] Date: _________________
- [ ] Blockers found: _________________

### Linux UAT
- [ ] Non-technical user test completed successfully
- [ ] All time targets met (< 15 min installation)
- [ ] Error recovery scenarios tested
- [ ] Tester Name: _________________
- [ ] Date: _________________
- [ ] Blockers found: _________________

### Ready for Release?
- [ ] All critical issues resolved
- [ ] All UAT scenarios passed
- [ ] Documentation is clear and complete
- [ ] Error messages are user-friendly

**Final Approval**: _________________
**Date**: _________________

---

## Appendix: Common UAT Issues and Resolutions

### Installation Issues
**Issue**: Windows Defender/SmartScreen blocks installer
**Expected**: User sees "More info" option and can proceed
**Resolution**: Document in README as expected behavior

**Issue**: Ubuntu Software Center says "Package has unmet dependencies"
**Expected**: User can install via `apt install -f`
**Resolution**: Pre-check dependencies or improve error message

**Issue**: User doesn't know where application installed
**Expected**: README shows installation path clearly
**Resolution**: Installation success message shows path

### Configuration Issues
**Issue**: User can't find .env file
**Expected**: "Edit Configuration" shortcut opens it directly
**Resolution**: Add shortcut, document path in README

**Issue**: User doesn't know what to put in .env
**Expected**: .env.example has clear comments for each variable
**Resolution**: Improve comments, add example values

### Service Issues
**Issue**: Service won't start but no error shown
**Expected**: Log file contains clear error message
**Resolution**: Improve error logging, add startup validation

**Issue**: User can't tell if service is running
**Expected**: Status command or GUI shows running state
**Resolution**: Add health check endpoint or status command

### Error Recovery Issues
**Issue**: Error message is cryptic (e.g., "Error 500")
**Expected**: Message explains what went wrong in plain English
**Resolution**: Improve error message formatting and context

**Issue**: User doesn't know how to view logs
**Expected**: "View Logs" shortcut or command documented
**Resolution**: Add shortcut, document log location

---

**End of UAT Guide**
