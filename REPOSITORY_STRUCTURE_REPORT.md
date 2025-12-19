# Repository Structure Analysis Report

**Date:** 2025-12-19  
**Repository:** CanvusLocalLLM

## Executive Summary

The repository structure has several organizational issues that deviate from industry best practices. The main concerns are:

1. **Dev support tools mixed with application code** - Development tooling directories (`.beads/`, `.claude/`, `.cursor/`, `.harness/`) are in the root alongside application code
2. **Root directory clutter** - Multiple Go files, JSON config files, and documentation files all in root
3. **Orphaned JSON files** - Several JSON files scattered throughout without clear organization
4. **Mixed concerns** - Application packages, dev tools, build artifacts, and runtime data all at the same level

## Current Structure

### Root Directory Contents

#### Application Code (Go files)
```
main.go                    # Entry point (844 lines)
handlers.go                # Main handlers (2,497 lines) 
handlers_test.go           # Handler tests (283 lines)
monitorcanvus.go           # Canvas monitor (635 lines)
monitorcanvus_test.go      # Monitor tests (453 lines)
service_other.go           # Service implementation (non-Windows)
service_windows.go         # Windows service implementation
service_test.go            # Service tests
main_test.go               # Main tests
```

**Issue:** While having `main.go` in root is standard Go practice, having 8 Go files in root is excessive. The handlers, monitor, and service files should ideally be in packages.

#### Configuration Files
```
.env                       # Local environment (gitignored)
example.env                # Example configuration
.gitignore                 # Git ignore rules
```

#### Documentation
```
README.md                  # Main documentation (620 lines)
README.txt                 # Installer documentation (279 lines)
CLAUDE.md                  # Claude AI documentation
CONTRIBUTING.md            # Contribution guidelines
LICENSE.txt                # License file
```

#### Development Tool Files (JSON)
```
.beads_project.json        # Beads project configuration
.claude_settings.json      # Claude IDE settings
shared_canvas.json         # Test/development canvas data
```

**Issue:** Dev tool configuration files are mixed with application code in root.

### Directory Structure

#### Application Packages (Go code)
```
canvasanalyzer/            # Canvas analysis package
canvusapi/                 # Canvus API client
core/                      # Core utilities and config
db/                        # Database layer
handlers/                  # Handler utilities (atoms/molecules)
imagegen/                  # Image generation
llamaruntime/              # LLM runtime integration
logging/                   # Logging infrastructure
metrics/                   # Metrics collection
ocrprocessor/              # OCR processing
pdfprocessor/              # PDF processing
sdruntime/                 # Stable Diffusion runtime
shutdown/                  # Graceful shutdown
vision/                    # Vision processing
webui/                     # Web UI server
```

**Status:** ✅ Well-organized, follows Go package conventions

#### Development Support Tools
```
.beads/                    # Beads issue tracking database & state
  ├── beads.db             # SQLite database (1.2MB)
  ├── autonomous-state/    # Agent state files
  └── config.yaml          # Beads configuration

.claude/                   # Claude IDE settings
  └── settings.local.json

.cursor/                   # Cursor IDE rules
  └── rules/

.harness/                  # Harness tooling
  └── .beads/              # Duplicate beads config?

agent-os/                  # Agent OS specifications
  ├── commands/
  ├── product/
  ├── specs/
  └── standards/
```

**Status:** ✅ Acceptable - Dev tool directories in root is standard practice (like `.git/`, `.github/`). No action needed.

#### Build & Distribution
```
installer/                 # Installer scripts and configs
  ├── windows/             # NSIS installer
  ├── debian/              # Debian package
  └── tarball/             # Tarball distribution

scripts/                   # Build scripts
  ├── build-*.sh           # Build scripts
  └── build-*.ps1          # PowerShell scripts

dist/                      # Build output (gitignored)
```

**Status:** ✅ Reasonably organized

#### Resources & Assets
```
icons-for-custom-menu/     # UI icons
test_files/                # Test data files
models/                    # AI model storage (gitignored)
```

**Status:** ✅ Acceptable

#### Runtime Data Directories
```
logs/                      # Application logs (gitignored)
generations/               # Generated content (gitignored)
lib/                       # Native libraries (gitignored)
deps/                      # Build dependencies (gitignored)
```

**Status:** ✅ Properly gitignored

#### Documentation
```
docs/                      # Project documentation
  ├── api-reference/       # API documentation (moved from Docs/)
  ├── build-guide.md
  ├── bunny-model.md
  ├── llamaruntime.md
  └── ...
```

**Status:** ✅ Well-organized after recent cleanup

### JSON Files Analysis

#### Root Level JSON Files
```
.beads_project.json        # Beads project config (dev tool)
.claude_settings.json      # Claude IDE settings (dev tool)
shared_canvas.json         # Test/development data
```

**Status:** ✅ Acceptable - Dev tool configs must be in root for tools to discover them. No action needed.

#### Beads Database Files
```
.beads/beads.db            # SQLite database (1.2MB)
.beads/beads.db-shm        # Shared memory file
.beads/beads.db-wal        # Write-ahead log
.beads/metadata.json       # Beads metadata
.beads/autonomous-state/   # Agent state JSON files
```

**Status:** ✅ Acceptable - Beads database in `.beads/` is correct location. No action needed.

#### Other JSON Files
```
.claude/settings.local.json
.generations/.claude_settings.json  # Duplicate?
.harness/.beads_project.json        # Duplicate?
logs/*.json                         # Audit logs
```

**Issue:** Multiple duplicate config files in different locations.

## Industry Standard Comparison

### Standard Go Project Structure

**Recommended:**
```
project/
├── cmd/                    # Application entry points
│   └── app/
│       └── main.go
├── internal/               # Private application code
│   ├── handlers/
│   ├── monitor/
│   └── service/
├── pkg/                    # Public library code
│   ├── canvasanalyzer/
│   ├── canvusapi/
│   └── ...
├── api/                    # API definitions
├── web/                    # Web assets
├── configs/                # Configuration files
├── scripts/                # Build scripts
├── docs/                   # Documentation
├── .dev/                   # Development tools
│   ├── beads/
│   ├── claude/
│   └── cursor/
└── testdata/               # Test data
```

**Current Structure Issues:**
1. ❌ No `cmd/` directory - `main.go` in root (acceptable for single binary)
2. ❌ No `internal/` vs `pkg/` separation
3. ❌ Dev tools in root instead of `.dev/`
4. ❌ Test data mixed with application code
5. ✅ Good package organization within packages
6. ✅ Documentation well-organized

### Standard Development Tool Organization

**Industry Standard:**
```
project-root/
├── .git/                   # Git repository (standard location)
├── .github/                # GitHub configs (standard location)
├── .vscode/                # VS Code settings (standard location)
├── .beads/                 # Beads issue tracking (standard location)
├── .claude/                # Claude IDE settings (standard location)
└── .cursor/                # Cursor IDE rules (standard location)
```

**Current:** ✅ Follows industry standard - dev tools in root as hidden directories.

## Tool Discovery Mechanism

### How Development Tools Find Their Configs

**Beads (`bd`):**
- Looks for `.beads/` directory in current working directory or parent directories
- Uses `--db` flag or `BEADS_DB` env var to override, but defaults to `.beads/beads.db`
- `.beads_project.json` must be in root for project initialization

**Claude IDE:**
- Looks for `.claude/` and `.claude_settings.json` in root
- No documented way to change these paths

**Cursor IDE:**
- Looks for `.cursor/` directory in root
- Rules are loaded from `.cursor/rules/`

**Conclusion:** These tools have hardcoded expectations. Moving configs would break them unless using symlinks.

## Recommendations

### Priority 1: Organize Development Tools

**⚠️ IMPORTANT CONSTRAINT:** Development tools (beads, claude, cursor) expect their configuration files in standard locations (`.beads/`, `.claude/`, `.cursor/`). Moving them would break tool functionality.

**Recommended Solution: Keep Tools in Root**

**Action:** **Accept that dev tools stay in root** - this is standard practice and there's no real benefit to moving them:

- Many projects keep `.git/`, `.github/`, `.vscode/`, `.idea/`, etc. in root
- Hidden directories (starting with `.`) are already visually separated
- Tools work out-of-the-box without configuration
- **Symlinks don't help** - seeing `.beads` vs `symlink -> .dev/beads` is the same visual clutter
- Symlinks add complexity with no benefit (Windows compatibility, Git issues, maintenance)

**Conclusion:** Dev tool directories (`.beads/`, `.claude/`, `.cursor/`, `.harness/`) should remain in root. This is industry standard and acceptable.

### Priority 2: Organize Root-Level Files

**⚠️ IMPORTANT:** Dev tool JSON files (`.beads_project.json`, `.claude_settings.json`) must stay in root - tools expect them there.

**Action:** Only move non-tool files:

```bash
# Move test data to appropriate location
shared_canvas.json → testdata/shared_canvas.json
```

**Note:** Keep `.beads_project.json` and `.claude_settings.json` in root - these are tool configuration files that must be discoverable by the tools.

### Priority 3: Consider Package Reorganization

**Option A: Keep Current Structure (Simpler)**
- Accept that `main.go`, `handlers.go`, `monitorcanvus.go` are in root
- This is acceptable for a single-binary Go application
- Document the structure clearly

**Option B: Standard Go Layout (More Organized)**
```
cmd/
└── canvuslocallm/
    ├── main.go
    ├── handlers.go
    ├── monitorcanvus.go
    └── service_*.go

internal/                  # Private packages
├── handlers/              # Move handlers/ here
└── monitor/               # Extract monitor logic

pkg/                       # Public packages (current packages)
├── canvasanalyzer/
├── canvusapi/
└── ...
```

**Recommendation:** Option A for now (less disruptive), consider Option B for future refactoring.

### Priority 4: Clean Up Duplicate Config Files

**Action:** Remove duplicates:
- `.harness/.beads_project.json` (duplicate)
- `generations/.claude_settings.json` (duplicate)

### Priority 5: Update .gitignore

**Action:** Ensure `.dev/` is properly handled:
```gitignore
# Development tools (optional - can be committed)
.dev/beads/beads.db*
.dev/beads/autonomous-state/
```

## Implementation Plan

### Phase 1: Dev Tools Organization (No Action Needed)

**Decision: Keep dev tools in root**
- `.beads/`, `.claude/`, `.cursor/`, `.harness/` stay in root
- This is standard practice (similar to `.git/`, `.github/`, `.vscode/`)
- No risk of breaking tools
- Symlinks don't improve tidiness (same visual clutter)
- Focus cleanup efforts on actual issues (test data, duplicates, application code)

### Phase 2: Organize Root Files (Low Risk)
1. **Keep** `.beads_project.json` and `.claude_settings.json` in root (tools need them there)
2. Move `shared_canvas.json` → `testdata/shared_canvas.json`
3. Update references in code/docs

### Phase 3: Clean Duplicates (Low Risk)
1. Remove duplicate config files
2. Update any references

### Phase 4: Documentation (Low Risk)
1. Update README.md with new structure
2. Document `.dev/` directory purpose
3. Update any build scripts that reference old paths

## Metrics

### Current State
- **Root-level Go files:** 9 files
- **Root-level JSON files:** 3 files
- **Dev tool directories in root:** 4 directories
- **Package directories:** 15 packages
- **Total directories:** 30+ directories

### Target State
- **Root-level Go files:** 1-2 files (main.go, optionally main_test.go)
- **Root-level JSON files:** 0 files (all in .dev/ or testdata/)
- **Dev tool directories in root:** 0 (all in .dev/)
- **Package directories:** 15 packages (unchanged)
- **Total directories:** 30+ directories (better organized)

## Conclusion

The repository has good package-level organization but suffers from root-level clutter. However, **dev tool directories (`.beads/`, `.claude/`, `.cursor/`) should remain in root** because:

1. Tools expect them there (hardcoded discovery)
2. This is standard practice (similar to `.git/`, `.github/`, `.vscode/`)
3. Hidden directories (`.`) are already visually separated
4. Moving them risks breaking tool functionality

**Revised Recommendations:**
- ✅ **Keep dev tools in root** - This is acceptable and standard
- ✅ **Move test data** (`shared_canvas.json` → `testdata/`)
- ✅ **Remove duplicate configs** (`.harness/.beads_project.json`, etc.)
- ✅ **Focus on organizing application code** (consider `cmd/` or `internal/` structure)

**Estimated Effort:** 1-2 hours (much less without moving dev tools)  
**Risk Level:** Very Low (only moving non-critical files)  
**Impact:** Medium (cleaner, but dev tools stay in root as expected)

