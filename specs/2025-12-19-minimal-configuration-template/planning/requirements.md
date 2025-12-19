# Spec Requirements: Minimal Configuration Template

## Initial Description
Create a minimal, user-friendly configuration template (.env.example) and setup documentation that enables easy first-run configuration aligned with the product's zero-configuration vision. This is the critical blocker for all Phase 1 installer work (blocks 5 issues: first-run model download, configuration validation, NSIS installer, Debian package, and tarball distribution).

## Requirements Discussion

### Code Analysis Findings

**Current Configuration State (from core/config.go):**

The application currently requires the following environment variables (lines 218-239):
1. `CANVUS_SERVER` - Required
2. `OPENAI_API_KEY` - Currently marked as required (line 220)
3. `CANVUS_API_KEY` - Required
4. `WEBUI_PWD` - Required
5. `CANVAS_ID` or `CANVAS_IDS` - At least one required

**Key Discovery - Misalignment with Zero-Config Vision:**

The code validation (config.go line 220) currently REQUIRES `OPENAI_API_KEY`, which contradicts the zero-config local-first vision. Analysis shows:

- README.md lines 375-434 document migration AWAY from OpenAI API dependency
- Product mission (mission.md) emphasizes "zero cloud dependencies" and "complete local-first privacy"
- Tech stack (tech-stack.md line 106-110) explicitly states OpenAI API is "NOT required for production (local-first architecture)"
- main.go lines 558-562 shows LLAMA_MODEL_PATH is optional (local LLM can be disabled)
- example.env line 5 still shows OPENAI_API_KEY with placeholder value

**Conclusion:** The validation code has not been updated to reflect the local-first architecture. OPENAI_API_KEY should NOT be required in minimal template.

**Configuration Categories Analysis:**

**Category 1: Absolutely Required for Zero-Config (Canvus Connection Only):**
- `CANVUS_SERVER` - Canvus server URL
- `CANVUS_API_KEY` - API key for Canvus authentication
- `CANVAS_ID` - Target canvas identifier (single canvas mode)
- `WEBUI_PWD` - Web UI password (security requirement)

**Category 2: Has Code Defaults (Should NOT be in minimal template):**
- `PORT=3000` (default in config.go line 253)
- `BASE_LLM_URL=http://127.0.0.1:1234/v1` (default line 149, but not used with embedded llama.cpp)
- `IMAGE_LLM_URL=https://api.openai.com/v1` (default line 151, but SD runtime replaces this)
- All token limits (lines 171-178 have defaults)
- Processing config (lines 181-187 have defaults)
- `ALLOW_SELF_SIGNED_CERTS=false` (default line 188)
- `LLAMA_MODELS_DIR=./models` (default line 161)
- `LLAMA_AUTO_DOWNLOAD=false` (default line 162)

**Category 3: Local LLM Configuration (Embedded Runtime - Optional but Important):**
- `LLAMA_MODEL_PATH` - Path to GGUF model (required IF using local LLM)
- `LLAMA_MODEL_URL` - Optional download URL for auto-download
- `LLAMA_AUTO_DOWNLOAD` - Enable auto-download (default: false)

**Category 4: Cloud API Fallbacks (Should be REMOVED from minimal template):**
- `OPENAI_API_KEY` - Contradicts local-first vision
- `GOOGLE_VISION_API_KEY` - Replaced by Bunny vision (roadmap line 124)
- `AZURE_OPENAI_ENDPOINT`, `AZURE_OPENAI_DEPLOYMENT`, `AZURE_OPENAI_API_VERSION` - Cloud fallback only
- `TEXT_LLM_URL`, `IMAGE_LLM_URL` - External LLM endpoints not needed with embedded runtime
- `OPENAI_NOTE_MODEL`, `OPENAI_CANVAS_MODEL`, `OPENAI_PDF_MODEL` - Model selection not needed (Bunny hardcoded)
- `IMAGE_GEN_MODEL` - Not needed (SD model is hardcoded)

**Category 5: Advanced Configuration (Optional Override):**
- Multi-canvas: `CANVAS_IDS` (comma-separated)
- `MAX_CONCURRENT`, `PROCESSING_TIMEOUT`, `MAX_FILE_SIZE`, etc.
- All token limit overrides
- SSL/TLS: `ALLOW_SELF_SIGNED_CERTS` (enterprise use case)

**Category 6: Stable Diffusion Configuration (Phase 3 - Not Yet Implemented):**
- `SD_MODEL_PATH`, `SD_IMAGE_SIZE`, `SD_INFERENCE_STEPS`, etc. (lines 149-186 in example.env)
- Should be deferred to future enhancement as SD integration is still in progress (roadmap line 50-52)

**Authentication Method Analysis:**

From config.go and README.md analysis:
- API key method: `CANVUS_API_KEY` (primary, used in all examples)
- Username/password: `CANVUS_USERNAME`/`CANVUS_PASSWORD` (not currently implemented in code)
- **Decision:** API key is the only supported method currently

**Documentation Structure Analysis:**

From README.md:
- Lines 86-124: "Quick Start" section with step-by-step setup
- Lines 169-259: "Configuration" section with minimal + advanced examples
- Lines 279-309: Platform-specific deployment steps
- Uses clear hierarchical structure with code examples

**Decision:** Maintain similar structure - Quick Start guide embedded in README.md with reference to enhanced .env.example

**First-Run Behavior Analysis:**

From main.go:
- Lines 49-53: .env loading with warning if missing (not fatal error)
- Lines 65-73: Startup validation runs before heavy operations (new feature)
- Lines 655-704: `runStartupValidation()` function checks configuration
- No auto-copy of .env.example or guided setup currently

**Decision:** Minimal template should support future first-run detection (validation issue CanvusLocalLLM-6rw.6)

**Model Path Defaults Analysis:**

From code analysis:
- main.go line 558: `modelPath := os.Getenv("LLAMA_MODEL_PATH")` - empty means disabled
- config.go line 161: `LlamaModelsDir` defaults to `./models`
- Roadmap indicates models will be "bundled with installer" (roadmap line 100)

**Decision:** Default path should assume bundled model at `./models/bunny-v1.1-llama-3-8b-v.gguf`

**LLM URL Configuration Decision:**

From analysis:
- `BASE_LLM_URL` defaults to external server (line 149: `http://127.0.0.1:1234/v1`)
- This is WRONG for embedded llama.cpp runtime (no external server needed)
- These variables are legacy from Phase 1 OpenAI API architecture

**Decision:** Remove BASE_LLM_URL, TEXT_LLM_URL, IMAGE_LLM_URL entirely from minimal template

**Web UI Password Requirement:**

From code:
- config.go line 222: `WEBUI_PWD` is in required variables list
- main.go lines 446-450: Empty password triggers unauthenticated mode warning
- Security concern: Default password is insecure for production

**Decision:** Required in minimal template with strong recommendation to set unique value

**Processing Parameters Decision:**

From config.go analysis:
- All have sensible defaults (lines 171-187)
- Token limits default to reasonable values for Bunny model
- Overriding requires understanding of AI model parameters

**Decision:** Completely omit from minimal template, rely on code defaults, document in advanced configuration section

**SSL Self-Signed Certificates:**

From analysis:
- Used for enterprise Canvus servers with self-signed certs
- Default is false (secure mode)
- README lines 563-580 documents security risks

**Decision:** Include in minimal template with default=false and clear security warning comment

**Multi-Canvas Configuration:**

From config.go:
- Lines 190-215: Supports both single canvas (`CANVAS_ID`) and multi-canvas (`CANVAS_IDS`)
- Multi-canvas is "planned enhancement" (roadmap line 165)
- Single canvas is backward compatible

**Decision:** Minimal template shows single canvas only, document multi-canvas in advanced config

**Validation Error Messages:**

From main.go analysis:
- Lines 655-704: New startup validation system with detailed error reporting
- Uses structured logging with zap for clear messages
- Lines 676-684: Individual failures logged with step name, message, and error

**Decision:** .env.example should have inline format hints, but rely on validation system for detailed runtime errors

### Existing Code to Reference

**Configuration Validation System:**
- File: `core/config.go` lines 218-239 - Current validation logic (needs updating for local-first)
- File: `main.go` lines 649-704 - Startup validation suite with user-friendly error messages
- Pattern: Uses structured logging with zap, provides step-by-step validation feedback

**Model Management:**
- File: `main.go` lines 706-762 - Model availability checking and download logic
- File: `llamaruntime/` package - Model loader with progress indication (referenced in main.go lines 556-646)
- Pattern: Context-based cancellation, progress callbacks, SHA256 verification

**HTTP Client Factory:**
- File: `core/config.go` lines 299-318 - GetHTTPClient with TLS configuration
- Pattern: Respects `ALLOW_SELF_SIGNED_CERTS` flag consistently

**Documentation Style:**
- File: `README.md` - Clear hierarchical structure, code examples, security warnings
- Pattern: Quick Start → Core Config → Advanced Config → Troubleshooting

**Environment Loading:**
- File: `main.go` lines 49-53 - godotenv.Load() with warning on missing .env
- Pattern: Non-fatal warning allows development without .env (uses env vars directly)

### Follow-up Questions

**Follow-up 1: OpenAI API Key Validation Bug**
The code currently requires OPENAI_API_KEY (core/config.go line 220), but this contradicts the zero-config local-first vision documented everywhere. Should the minimal config template work include fixing this validation bug (removing OPENAI_API_KEY from required variables list), or should that be a separate issue?

**Answer (from analysis):** This MUST be fixed as part of this spec. The minimal template cannot be "minimal" if it requires a cloud API key that contradicts the product mission. The validation fix is directly related to defining what's truly required.

**Follow-up 2: Username/Password Authentication**
The code references CANVUS_USERNAME/CANVUS_PASSWORD in comments (example.env lines 23-25, config.go not implemented), but I don't see actual implementation. Is this authentication method actually supported, or should it be removed from the template?

**Answer (from analysis):** Not currently implemented in config.go. Remove from minimal template. API key only.

**Follow-up 3: Stable Diffusion Configuration Deferral**
SD configuration (SD_MODEL_PATH, etc.) is Phase 3 work still in progress. Should these be completely excluded from the minimal .env.example, or included but commented with "Phase 3 - Coming Soon"?

**Answer (from analysis):** Exclude entirely from minimal template. Can be documented separately when SD integration completes. Roadmap shows it's "ready to implement" but not done.

## Visual Assets

### Files Provided:
No visual assets provided (user specified "only the existing code and no visuals").

### Visual Insights:
N/A - Code analysis only

## Requirements Summary

### Functional Requirements

**Minimal Configuration Template (.env.example):**

1. **Essential Canvus Connection (Required Section):**
   - `CANVUS_SERVER` - Canvus server URL with format hint
   - `CANVUS_API_KEY` - API key authentication
   - `CANVAS_ID` - Target canvas UUID
   - `WEBUI_PWD` - Web UI password with security recommendation

2. **Local AI Configuration (Optional Section):**
   - `LLAMA_MODEL_PATH` - Default to `./models/bunny-v1.1-llama-3-8b-v.gguf` (bundled model path)
   - `LLAMA_AUTO_DOWNLOAD` - Default to `false` with explanation
   - `LLAMA_MODEL_URL` - Optional download URL (commented out)

3. **Optional Settings (Clearly Marked as Optional/Advanced):**
   - `ALLOW_SELF_SIGNED_CERTS` - Default `false` with security warning
   - `PORT` - Default `3000`, comment: "Web UI port"
   - Note referencing advanced configuration documentation for other settings

4. **Template Organization:**
   - Clear section headers with visual separation (=== lines)
   - Inline comments explaining each variable's purpose
   - Format hints where applicable (e.g., "Must start with https://")
   - Required vs optional clearly marked
   - Sensible placeholder values that guide users
   - Security warnings where appropriate

5. **Removed from Minimal Template:**
   - All cloud API keys (OPENAI_API_KEY, GOOGLE_VISION_API_KEY, Azure config)
   - All LLM endpoint URLs (BASE_LLM_URL, TEXT_LLM_URL, IMAGE_LLM_URL)
   - Model selection variables (OPENAI_NOTE_MODEL, etc.)
   - Token limit overrides (rely on code defaults)
   - Processing parameters (rely on code defaults)
   - Stable Diffusion configuration (Phase 3, not yet ready)
   - Multi-canvas configuration (advanced feature)
   - Username/password auth (not implemented)

**Setup Documentation:**

1. **Quick Start Guide Enhancement (README.md):**
   - Update lines 86-124 to reflect minimal template
   - Add "First Run" section explaining model auto-download
   - Clear step-by-step with expected outcomes
   - Platform-specific notes (Windows vs Linux paths)

2. **Advanced Configuration Reference:**
   - Create `docs/advanced-configuration.md` documenting all override options
   - Include token limits, processing parameters, multi-canvas setup
   - Document cloud API fallback configuration (for developers)
   - Performance tuning guide reference

3. **Troubleshooting Enhancement:**
   - Add common first-run errors to README troubleshooting section
   - Link to validation error messages
   - GPU detection issues
   - Model download failures

**Configuration Validation Code Updates (Prerequisite):**

1. **Fix Required Variables List (core/config.go lines 218-239):**
   - Remove `OPENAI_API_KEY` from required variables
   - Keep: `CANVUS_SERVER`, `CANVUS_API_KEY`, `WEBUI_PWD`
   - Validate `CANVAS_ID` OR `CANVAS_IDS` (at least one)

2. **Add Helpful Error Messages:**
   - URL format validation for `CANVUS_SERVER` (must start with https://)
   - Canvas ID format validation (UUID format)
   - Model path existence check (if LLAMA_MODEL_PATH set)
   - Port range validation (1-65535)

3. **Support First-Run Detection:**
   - Check if .env exists, suggest copying from .env.example
   - Detect if running with minimal config (no model path) and show guidance
   - Clear messaging about optional vs required settings

### Reusability Opportunities

**Similar Patterns in Codebase:**

1. **Validation Framework:**
   - `main.go` lines 655-704: Startup validation suite pattern can be extended
   - Uses step-by-step validation with clear success/failure reporting
   - Structured logging with zap for user-friendly output

2. **Model Management:**
   - `llamaruntime/` model loader: Progress indication, SHA256 verification, auto-download
   - Pattern for first-run model download (Issue CanvusLocalLLM-6rw.5)

3. **HTTP Client Configuration:**
   - `core/config.go` GetHTTPClient pattern: Consistent TLS handling
   - Should be referenced in advanced configuration docs

4. **Environment Variable Helpers:**
   - `getEnvOrDefault()`, `parseIntEnv()`, etc. (config.go lines 80-138)
   - Show how defaults work in comments

**Code to Modify:**

1. `example.env` - Complete rewrite based on minimal template structure
2. `core/config.go` lines 218-239 - Remove OPENAI_API_KEY from required list
3. `README.md` lines 86-259 - Update Quick Start and Configuration sections
4. Create `docs/advanced-configuration.md` - New file for advanced settings

### Scope Boundaries

**In Scope:**

1. Create minimal .env.example with only essential Canvus connection settings
2. Update core/config.go validation to remove OPENAI_API_KEY requirement
3. Enhance README.md Quick Start section
4. Create advanced configuration documentation
5. Add inline comments and format hints to template
6. Document model path defaults for bundled model scenario
7. Clear required vs optional marking

**Out of Scope:**

1. **First-run guided setup UI** - Deferred to Issue CanvusLocalLLM-6rw.6 (Configuration Validation)
2. **Automated .env generation** - Deferred to installer implementation (Issues 6rw.1, 6rw.2, 6rw.3)
3. **Model download implementation** - Deferred to Issue CanvusLocalLLM-6rw.5 (First-Run Model Download)
4. **Multi-canvas configuration UI** - Advanced feature, deferred to dashboard enhancement
5. **Stable Diffusion configuration** - Phase 3 work, not yet ready for template
6. **Performance tuning guide** - Separate documentation effort (reference existing llamaruntime docs)
7. **Username/password authentication** - Not currently implemented in code
8. **Cloud API fallback configuration** - Document in advanced config only, not minimal template

**Future Enhancements (Documented but Not Implemented):**

1. Interactive configuration wizard for installers
2. Configuration validation web UI in dashboard
3. One-click model download from web interface
4. Configuration profiles (development vs production)
5. Multi-canvas management UI

### Technical Considerations

**Backward Compatibility:**

- Must maintain compatibility with existing .env files
- Code defaults must not break existing deployments
- Migration guide needed for users with current example.env

**Integration Points:**

- Configuration validation (Issue CanvusLocalLLM-6rw.6) depends on this template structure
- First-run model download (Issue CanvusLocalLLM-6rw.5) needs LLAMA_MODEL_PATH defaults
- All installers (Issues 6rw.1, 6rw.2, 6rw.3) will populate this template
- Startup validation in main.go will reference new minimal requirements

**Platform Differences:**

- Windows: Use backslash paths in examples, model path like `.\models\bunny-v1.1-llama-3-8b-v.gguf`
- Linux: Forward slash paths, `/opt/canvuslocallm/models/bunny-v1.1-llama-3-8b-v.gguf`
- Template should use relative paths that work on both platforms

**Security Considerations:**

- WEBUI_PWD must be user-set (no default password)
- ALLOW_SELF_SIGNED_CERTS=false by default with clear warning
- API keys marked as sensitive in comments
- .env must remain in .gitignore

**Dependencies:**

- No new code dependencies
- Documentation only requires markdown
- Validation changes use existing zap logging
- Template is plain text file

**Testing Strategy:**

1. Test with minimal required variables only
2. Test with optional variables included
3. Verify validation rejects missing required variables
4. Verify validation accepts minimal config
5. Test on Windows and Linux paths
6. Verify backward compatibility with existing .env files
