# Specification: Minimal Configuration Template

## Goal
Reduce CanvusLocalLLM configuration complexity from 40+ environment variables to 10 essential Canvus credentials, enabling zero-config local AI deployment and unblocking Phase 1 installer infrastructure by making all AI/inference settings internal defaults.

## User Stories
- As an enterprise IT administrator, I want to deploy CanvusLocalLLM with only Canvus credentials so that I can mass-deploy without AI infrastructure expertise
- As a non-technical user, I want to install and configure CanvusLocalLLM in under 5 minutes so that I can start using AI capabilities immediately without learning about models, endpoints, or parameters

## Specific Requirements

**Create Minimal .env.example Template**
- Include only 10 essential variables: CANVUS_SERVER, CANVAS_ID, CANVUS_API_KEY (or CANVUS_USERNAME/CANVUS_PASSWORD), CANVAS_NAME (optional), WEBUI_PWD, ALLOW_SELF_SIGNED_CERTS (optional, dev only)
- Add clear inline comments for each variable explaining purpose and format
- Provide example values that obviously need replacement (e.g., `https://your-canvus-server.com`, `your-canvas-id-here`)
- Include header comment block with quick start instructions and link to full documentation
- Remove all AI configuration: model paths, API endpoints, token limits, inference parameters, provider settings
- Group variables logically: "Canvus Connection", "Authentication", "Optional Settings"
- Use consistent comment style: `# Description` above variable, inline example after equals
- Template should be self-documenting for users who never read external docs

**Fix OPENAI_API_KEY Required Validation Bug**
- Remove "OPENAI_API_KEY" from `core/config.go:218-223` requiredVars array (critical bug)
- Make OpenAIAPIKey field optional in Config struct (change validation logic, not field itself)
- Update `core/config_validator.go:128-155` CheckOpenAICredentials to be optional check (not included in ValidateRequired)
- Add comment in LoadConfig explaining OpenAI key only needed for cloud fallback mode
- Handlers should check `config.OpenAIAPIKey != ""` before creating OpenAI clients
- Return helpful error if cloud operation attempted without key: "OpenAI API key required for cloud image generation. Configure OPENAI_API_KEY or use local generation."
- Test config validation passes with only Canvus credentials set

**Establish Sensible Internal Defaults**
- BaseLLMURL defaults to "http://127.0.0.1:1234/v1" (llamaruntime local server)
- ImageLLMURL defaults to empty string (triggers local SD generation in imagegen package)
- Token limits: NoteResponseTokens=400, PDFPrecisTokens=1000, CanvasPrecisTokens=600, ImageAnalysisTokens=16384
- Processing: MaxRetries=3, RetryDelay=1s, AITimeout=60s, ProcessingTimeout=300s, MaxConcurrent=5
- Files: MaxFileSize=52428800 (50MB), DownloadsDir="./downloads"
- Port=3000 for WebUI
- All model names default to "" (local models don't need OpenAI model identifiers)
- Azure OpenAI fields default to "" (unused in local-first mode)
- Document defaults in code comments explaining why each value chosen

**Remove Optional Cloud Configuration from Template**
- Do NOT include in .env.example: OPENAI_API_KEY, GOOGLE_VISION_API_KEY, BASE_LLM_URL, TEXT_LLM_URL, IMAGE_LLM_URL
- Do NOT include: AZURE_OPENAI_ENDPOINT, AZURE_OPENAI_DEPLOYMENT, AZURE_OPENAI_API_VERSION
- Do NOT include: OPENAI_NOTE_MODEL, OPENAI_CANVAS_MODEL, OPENAI_PDF_MODEL, IMAGE_GEN_MODEL
- Do NOT include: All token limit variables (18 variables removed)
- Do NOT include: LLAMA_MODEL_PATH, LLAMA_MODEL_URL, LLAMA_MODELS_DIR, LLAMA_AUTO_DOWNLOAD (llamaruntime has own config)
- Do NOT include: Processing parameters (MAX_RETRIES, RETRY_DELAY, AI_TIMEOUT, etc.)
- Do NOT include: MAX_FILE_SIZE, DOWNLOADS_DIR, PORT, MAX_CONCURRENT
- Advanced users can override defaults via environment variables, but template shows zero-config path

**Update Documentation and Comments**
- Add "Zero-Config Local AI" section to README explaining minimal configuration philosophy
- Document in .env.example header: "This is the minimal configuration. All AI settings use intelligent defaults for local GPU inference."
- Explain cloud fallback: "To enable cloud API fallback, set OPENAI_API_KEY or GOOGLE_VISION_API_KEY"
- Add troubleshooting section: if config error mentions missing variable not in template, it's a bug to report
- Document all 40+ available environment variables in separate ADVANCED_CONFIG.md for power users
- Include examples of common overrides: changing port, enabling debug logging, adjusting VRAM limits

**Validation Error Message Improvements**
- LoadConfig should NOT fail if OPENAI_API_KEY missing (only fail on Canvus credentials)
- Error message for missing .env: "Configuration file not found. Copy .env.example to .env and configure your Canvus credentials."
- Error message for invalid CANVUS_SERVER: "Invalid Canvus server URL: [value]. Example: https://canvus.example.com"
- Error message for missing CANVAS_ID: "CANVAS_ID required. Find your canvas ID in the canvas URL or settings."
- Error message for missing auth: "Canvus authentication required. Set CANVUS_API_KEY or CANVUS_USERNAME/CANVUS_PASSWORD"
- Add helpful context to each error explaining where to find the value and what format it expects

**Installer Integration Requirements**
- .env.example must be copy-ready for Windows NSIS installer (no manual editing of template)
- Debian postinst script copies .env.example to .env with helpful message: "Edit /opt/canvuslocallm/.env with your Canvus credentials"
- Tarball install.sh displays configuration instructions after extraction: "Configure Canvus connection in .env file (see .env.example)"
- All installers include only minimal .env.example (10 variables), not current 40+ variable version
- First-run validation provides helpful error messages guiding user to configure the 3-4 required variables

## Visual Design
No visual assets (code-only configuration spec).

## Existing Code to Leverage

**core/config.go:79-115 - Environment Variable Parsing Atoms**
- Reuse getEnvOrDefault(), parseIntEnv(), parseInt64Env(), parseFloat64Env() for all default value loading
- Pattern: `getEnvOrDefault("KEY", "default-value")` makes all settings optional with fallbacks
- Apply to all 30+ newly-optional settings: LLM URLs, token limits, processing config, model names
- Keep validation logic in LoadConfig but remove from requiredVars array
- Comment each default explaining why that value chosen for local-first deployment

**core/config_validator.go:128-155 - OpenAI Credentials Check**
- CheckOpenAICredentials() already returns ValidationResult with Valid=false if key missing
- Change ValidateRequired() at lines 171-193 to NOT call CheckOpenAICredentials()
- Keep CheckOpenAICredentials() in ValidateAll() for comprehensive checking (warns but doesn't fail)
- Add new method: ValidateMinimal() that only checks Canvus credentials (server, canvas ID, auth)
- Use ValidateMinimal() in startup validation for local-first mode

**handlers.go Lines 494, 937, 1072, 1810 - OpenAI Client Creation**
- All four locations use pattern: `openai.DefaultConfig(config.OpenAIAPIKey)`
- Wrap in conditional: `if config.OpenAIAPIKey == "" { return ErrCloudAPINotConfigured }`
- Define new sentinel error: `ErrCloudAPINotConfigured = errors.New("cloud API key not configured")`
- Handlers should attempt local inference first, only fall back to cloud if explicitly configured
- Log warning when cloud API used: "Using cloud API (OpenAI) - consider local inference for privacy"

**imagegen/generator.go - Cloud Provider Selection**
- Generator already has Provider interface with OpenAI and Azure implementations
- Add logic: if config.ImageLLMURL == "" && config.OpenAIAPIKey == "", use local SD provider
- Local SD provider (Phase 3) implements same Provider interface
- Selection order: Local SD (if available) → Azure OpenAI (if configured) → OpenAI DALL-E (if configured) → error
- Makes cloud APIs opt-in rather than required

**Current .env.example - Legacy Template Reference**
- Lines 1-22: Canvus credentials section is good model to follow (clear comments, examples)
- Lines 24-76: All AI configuration to be removed from template
- Keep header comment style and grouping pattern
- Current template has good example value pattern: `your-canvus-server.example.com` clearly needs replacement

## Out of Scope
- Implementing local LLM inference (Phase 2 - llamaruntime integration already specced)
- Implementing local image generation (Phase 3 - stable-diffusion.cpp integration already specced)
- Automatic model downloads (Phase 1 - covered in separate First-Run Model Download spec)
- Configuration validation UI or interactive setup wizard (Phase 5)
- Configuration migration tool for existing users (can be added later if needed)
- Environment variable encryption or secrets management (security hardening for future phase)
- Configuration hot-reload or runtime reconfiguration (not needed for first release)
- Multi-canvas configuration in minimal template (advanced feature, not zero-config path)
- Web UI configuration editor (Phase 5 feature)
- Telemetry or analytics about which config options users actually change
