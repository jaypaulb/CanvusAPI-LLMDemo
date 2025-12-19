# Spec Initialization: Minimal Configuration Template

## Feature Overview

Create a minimal, user-friendly configuration template (.env.example) and setup documentation that enables easy first-run configuration aligned with the product's zero-configuration vision. This is the critical blocker for all Phase 1 installer work.

## Strategic Context

**Issue:** CanvusLocalLLM-6rw.4 (Priority: P2, Roadmap-4)

**Current Blocker Status:** This issue blocks 5 other issues:
- CanvusLocalLLM-6rw.5: First-Run Model Download
- CanvusLocalLLM-6rw.6: Configuration Validation
- CanvusLocalLLM-6rw.1: NSIS Installer for Windows
- CanvusLocalLLM-6rw.2: Debian Package for Linux
- CanvusLocalLLM-6rw.3: Tarball Distribution for Linux

**User Priority:** "High - Do this immediately"

## Problem Statement

Currently, users must manually configure an extensive 186-line `example.env` file with 40+ environment variables before the application can run. This creates significant friction and contradicts the product mission of "zero-configuration" deployment. The existing configuration includes:

- Complex LLM endpoint URLs and model selection
- Multiple cloud provider configurations (OpenAI, Azure, Google Vision)
- Token limits and processing parameters
- Advanced settings most users don't need to understand

For the zero-config installer vision to succeed, we need to reduce this to the absolute minimum required configuration (Canvus credentials only) while maintaining sensible defaults for everything else.

## Product Mission Alignment

From product mission:
> "Zero-configuration AI integration service... batteries-included, fully embedded local LLM solution that works out of the box with no cloud dependencies"

Current pain points the config template must solve:
- **Enterprise IT Administrator:** "Deploy a simple, secure AI solution that... installs in minutes, and requires only Canvus credentials to configure"
- **Product Manager:** "Install once, provide Canvus credentials, and immediately get AI-powered summaries"
- **Independent Consultant:** "Run powerful AI capabilities entirely on local hardware with zero cloud dependencies"

## Existing Implementation

**Current State:**
- `example.env`: 186 lines with extensive documentation, mixed priorities
- `core/config.go`: Loads all environment variables with fallback defaults
- No validation or helpful error messages for missing/invalid values
- No setup documentation beyond code comments in example.env
- Manual .env creation required for development

**Key Configuration Categories (from example.env):**
1. **Required for zero-config vision:**
   - Canvus credentials (server, API key or username/password, canvas ID)

2. **Should have smart defaults:**
   - Local LLM path/URL (default to embedded Bunny model)
   - Model selection (default to bundled models)
   - Processing parameters (token limits, timeouts, concurrency)

3. **Optional/advanced:**
   - Cloud provider fallback (Azure OpenAI, Google Vision)
   - Advanced tuning (chunk sizes, retry logic, specific model overrides)
   - Developer settings (allow self-signed certs, custom directories)

## Success Criteria

1. **Minimal Required Configuration:** Only Canvus credentials should be required
2. **Clear Documentation:** Each setting should have clear explanation and sensible default
3. **Progressive Disclosure:** Essential settings upfront, advanced settings clearly marked
4. **Validation Support:** Config structure should enable helpful startup validation
5. **Installer-Ready:** Template must work with automated installer population

## Initial Scope Assumptions

- Create enhanced .env.example with reorganized structure
- Add quick-start setup guide (README or docs/)
- Design config validation structure (implementation may be separate issue)
- Focus on supporting Windows/Linux installer workflows
- Maintain backward compatibility with existing config
