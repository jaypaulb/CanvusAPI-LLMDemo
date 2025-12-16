# CanvusLocalLLM Installer Documentation

This directory contains all installer-related files and testing documentation for CanvusLocalLLM.

## Directory Structure

```
installer/
├── windows/           # Windows NSIS installer scripts
├── debian/            # Debian package structure
├── tarball/           # Generic Linux tarball installer
├── TESTING-CHECKLIST.md   # Comprehensive manual testing checklist
├── UAT-GUIDE.md          # User Acceptance Testing guide
├── UAT-RESULTS.md        # UAT session results tracker
├── UAT-QUICKREF.md       # UAT quick reference card (printable)
└── README.md             # This file
```

## Documentation Overview

### For Developers

#### TESTING-CHECKLIST.md
**Purpose**: Comprehensive manual testing checklist for all installer types
**Audience**: QA engineers, developers
**Scope**:
- Windows NSIS installer (10 installation tests, 7 uninstall tests)
- Debian package (6 installation tests, 4 uninstall tests)
- Tarball installer (7 installation tests, 3 error handling tests)
- Cross-platform verification matrix

**When to use**: During development and QA cycles to verify installer functionality

### For UAT Testers

#### UAT-GUIDE.md
**Purpose**: User Acceptance Testing scenarios for non-technical users
**Audience**: Non-technical users, beta testers, stakeholders
**Scope**:
- 9 detailed UAT scenarios with time targets
- Windows installation (< 10 min target)
- Linux installation (< 15 min target)
- Error recovery scenarios (< 5 min recovery target)
- Acceptance criteria and sign-off procedures

**When to use**: Before release, with real users testing in real-world conditions

#### UAT-RESULTS.md
**Purpose**: Track actual UAT sessions and outcomes
**Audience**: QA leads, product managers, release managers
**Scope**:
- Session templates for recording UAT results
- Issue tracking (critical, non-critical, enhancements)
- Summary statistics and time performance metrics
- User satisfaction scoring
- Release sign-off checklist

**When to use**: During UAT sessions to record results systematically

#### UAT-QUICKREF.md
**Purpose**: One-page quick reference for UAT testers
**Audience**: UAT testers (print this out!)
**Scope**:
- Time targets and checklists
- Platform-specific quick commands
- Error scenario procedures
- Issue logging format
- Post-session questions

**When to use**: Print and keep handy during UAT sessions

## Testing Workflow

### Phase 1: Development Testing
1. Build installer for target platform
2. Follow **TESTING-CHECKLIST.md** for that platform
3. Fix any issues found
4. Repeat until all checklist items pass

### Phase 2: User Acceptance Testing
1. Recruit non-technical users (2-3 per platform)
2. Give them **UAT-QUICKREF.md** (printed)
3. Have them follow scenarios in **UAT-GUIDE.md**
4. Record results in **UAT-RESULTS.md**
5. Address critical issues immediately
6. Fix non-critical issues before release

### Phase 3: Release Sign-Off
1. Review all UAT session results in **UAT-RESULTS.md**
2. Verify acceptance criteria met:
   - Time targets achieved
   - No critical issues open
   - User satisfaction ≥ 4.0
   - Recommendation rate ≥ 80%
3. Get sign-off from stakeholders
4. Release approved version

## Quick Links

### Building Installers

**Windows NSIS**:
```bash
# Build binary
GOOS=windows GOARCH=amd64 go build -o bin/CanvusLocalLLM.exe .

# Build installer (requires NSIS)
makensis installer/windows/canvusapi.nsi
```

**Debian Package**:
```bash
# Build binary
GOOS=linux GOARCH=amd64 go build -o installer/debian/opt/canvuslocallm/canvuslocallm .

# Build package
dpkg-deb --build installer/debian canvuslocallm_1.0.0_amd64.deb
```

**Tarball**:
```bash
# Build binary
GOOS=linux GOARCH=amd64 go build -o bin/canvuslocallm .

# Create tarball
tar -czf canvuslocallm-linux-amd64.tar.gz -C bin canvuslocallm install.sh .env.example LICENSE.txt README.txt canvuslocallm.service
```

### Testing Installers

**Run Development Tests**:
```bash
# Follow platform-specific section in TESTING-CHECKLIST.md
# Example for Windows:
# - Install on clean Windows 10/11 VM
# - Work through WI-001 through WI-008
# - Test uninstall with WU-001 through WU-007
```

**Run UAT**:
```bash
# 1. Give tester UAT-QUICKREF.md (printed)
# 2. Have them follow UAT-GUIDE.md scenarios
# 3. Record results in UAT-RESULTS.md
# 4. Track time for each scenario
# 5. Note all issues and suggestions
```

## Platform-Specific Notes

### Windows
- Requires NSIS 3.x for building installer
- UAC prompt expected during installation (needs admin)
- Service installation is optional component
- Supports Windows 10 and 11

### Linux (Debian/Ubuntu)
- Requires dpkg-deb for building package
- Supports Ubuntu 20.04, 22.04 LTS
- Creates dedicated user/group for service
- Integrates with systemd

### Linux (Generic/Tarball)
- Works on any Linux with systemd (Fedora, Arch, etc.)
- Manual installation via install.sh script
- Supports both system-wide and user installations
- No package manager dependencies

## UAT Acceptance Criteria Summary

For release approval, UAT must achieve:

| Criteria | Target | Critical? |
|----------|--------|-----------|
| Windows install time | < 10 min | YES |
| Linux install time | < 15 min | YES |
| Error recovery time | < 5 min | YES |
| User satisfaction score | ≥ 4.0/5.0 | YES |
| Recommendation rate | ≥ 80% | YES |
| Critical issues | 0 open | YES |
| UAT sessions completed | ≥ 2 per platform | YES |

## Issue Severity Definitions

### Critical (Release Blocker)
- Can't complete installation
- Service won't start with unclear errors
- Data loss or corruption
- Security vulnerabilities
- Can't recover from errors
- Misleading errors causing wrong actions

### Non-Critical (Should Fix)
- Confusing but not blocking
- Extra steps needed but works
- Typos or unclear documentation
- UI/UX annoyances

### Enhancement (Future)
- Feature requests
- Convenience improvements
- Better defaults
- Additional documentation
- Cosmetic improvements

## Contact & Support

**Issue Tracking**: Use Beads (`bd create`) or GitHub issues
**UAT Questions**: See UAT-GUIDE.md appendix
**Developer Questions**: See TESTING-CHECKLIST.md appendix

---

**Last Updated**: 2025-12-16
**Version**: 1.0.0
