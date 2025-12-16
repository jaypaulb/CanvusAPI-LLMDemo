# Phase 4: Codebase Architecture Refactoring - Requirements

## Overview

Refactor the existing CanvusLocalLLM Go codebase to follow atomic design principles, eliminate global state, and improve modularity and testability. The current architecture suffers from a monolithic handlers.go file (~2100 lines) containing mixed responsibilities and a global configuration variable that violates dependency injection principles.

## Problem Statement

### Current Architecture Issues

**1. Monolithic handlers.go File**
- 2100+ lines of mixed concerns
- Multiple distinct feature domains in one file:
  - PDF processing (lines 1070-1383)
  - Image generation (lines 608-897)
  - Canvas analysis (lines 1390-1588)
  - Google Vision OCR (lines 899-1068, 1782-2101)
  - Note processing (lines 102-234)
- Violates Single Responsibility Principle
- Difficult to test in isolation
- Poor code navigation and maintenance

**2. Global Configuration Variable**
- `var config *core.Config` at package level (line 48)
- Breaks dependency injection
- Makes testing difficult (requires global state setup)
- Violates atomic design principles (organisms shouldn't depend on globals)
- Creates hidden dependencies

**3. Lack of Atomic Design Structure**
- Mixed levels of abstraction throughout
- Atoms (pure functions) buried inside complex handlers
- No clear separation between:
  - Atoms: Pure utility functions
  - Molecules: Simple compositions
  - Organisms: Complex feature units
  - Pages: Composition roots

**4. Poor Package Organization**
- All handlers in main package
- Prevents independent testing
- No logical grouping by domain
- Tight coupling between unrelated features

## Requirements

### R1: Extract PDF Processing Package

**Priority:** P1 (High)
**Scope:** Phase 4, Item 16

**Description:** Create a standalone `pdfprocessor` package that encapsulates all PDF-related functionality.

**Current State:**
- PDF functions embedded in handlers.go:
  - `handlePDFPrecis` (lines 1080-1299)
  - `extractPDFText` (lines 1301-1335)
  - `splitIntoChunks` (lines 1337-1365)
  - `consolidateSummaries` (lines 1367-1382)
  - `getPDFChunkPrompt` (lines 1070-1078)
  - `chunkPDFContent` (lines 1668-1694)

**Target State:**
- Package structure:
  ```
  pdfprocessor/
  ├── processor.go       # Main processor organism
  ├── extractor.go       # Text extraction molecule
  ├── chunker.go         # Text chunking molecule
  ├── summarizer.go      # AI summarization molecule
  └── processor_test.go  # Comprehensive tests
  ```

**Functional Requirements:**
- FR1.1: Extract text from PDF files using github.com/ledongthuc/pdf
- FR1.2: Chunk extracted text by token count with paragraph boundaries
- FR1.3: Generate AI summaries using OpenAI client
- FR1.4: Support multi-chunk protocol for large PDFs
- FR1.5: Create response notes with appropriate sizing
- FR1.6: Progress updates via callback interface
- FR1.7: Context-based cancellation support

**Non-Functional Requirements:**
- NFR1.1: No global variables within package
- NFR1.2: All dependencies injected via constructor
- NFR1.3: Thread-safe concurrent processing
- NFR1.4: Comprehensive unit tests (>80% coverage)
- NFR1.5: Integration tests with mock AI client
- NFR1.6: Error handling with wrapped errors

**Dependencies:**
- `core.Config` for configuration
- `canvusapi.Client` for API operations
- OpenAI client for AI generation
- Context for cancellation

**Acceptance Criteria:**
- AC1.1: Package compiles independently
- AC1.2: All existing PDF functionality works identically
- AC1.3: Unit tests pass with mocked dependencies
- AC1.4: Integration tests pass with real PDF files
- AC1.5: No global state in package
- AC1.6: Godoc documentation complete

### R2: Extract Image Generation Package

**Priority:** P1 (High)
**Scope:** Phase 4, Item 17

**Description:** Create a standalone `imagegen` package for AI image generation using OpenAI DALL-E and Azure OpenAI.

**Current State:**
- Image generation functions in handlers.go:
  - `processAIImage` (lines 608-650)
  - `processAIImageOpenAI` (lines 652-774)
  - `processAIImageAzure` (lines 776-897)
  - `isAzureOpenAIEndpoint` (lines 602-606)

**Target State:**
- Package structure:
  ```
  imagegen/
  ├── generator.go        # Main generator organism
  ├── openai.go          # OpenAI provider molecule
  ├── azure.go           # Azure provider molecule
  ├── downloader.go      # Image download molecule
  └── generator_test.go  # Comprehensive tests
  ```

**Functional Requirements:**
- FR2.1: Generate images using OpenAI DALL-E API
- FR2.2: Generate images using Azure OpenAI API
- FR2.3: Auto-detect provider based on endpoint URL
- FR2.4: Download generated images to local filesystem
- FR2.5: Upload images to Canvus with calculated positioning
- FR2.6: Support different model types (dall-e-2, dall-e-3)
- FR2.7: Cleanup temporary files after processing

**Non-Functional Requirements:**
- NFR2.1: Provider abstraction via interface
- NFR2.2: No global variables
- NFR2.3: Injected HTTP client for testability
- NFR2.4: Context-based timeout handling
- NFR2.5: Comprehensive error messages
- NFR2.6: Unit tests with mocked HTTP responses

**Dependencies:**
- `core.Config` for API keys and endpoints
- `canvusapi.Client` for widget creation
- HTTP client for image downloads
- Context for cancellation

**Acceptance Criteria:**
- AC2.1: Package compiles independently
- AC2.2: OpenAI generation works identically
- AC2.3: Azure generation works identically
- AC2.4: Provider auto-detection works correctly
- AC2.5: Unit tests pass with mocked HTTP
- AC2.6: No global state in package

### R3: Extract Canvas Analysis Package

**Priority:** P1 (High)
**Scope:** Phase 4, Item 18

**Description:** Create a standalone `canvasanalyzer` package for generating canvas-wide analysis and insights.

**Current State:**
- Canvas analysis functions in handlers.go:
  - `handleCanvusPrecis` (lines 1390-1445)
  - `fetchCanvasWidgets` (lines 1447-1468)
  - `processCanvusPrecis` (lines 1470-1588)

**Target State:**
- Package structure:
  ```
  canvasanalyzer/
  ├── analyzer.go         # Main analyzer organism
  ├── fetcher.go          # Widget fetcher molecule
  ├── processor.go        # Analysis processor molecule
  └── analyzer_test.go    # Comprehensive tests
  ```

**Functional Requirements:**
- FR3.1: Fetch all widgets from canvas with retries
- FR3.2: Filter out triggering icons from analysis
- FR3.3: Generate AI analysis of canvas content
- FR3.4: Create structured markdown summaries (Overview, Insights, Recommendations)
- FR3.5: Progress updates via callback
- FR3.6: Create response notes with appropriate positioning

**Non-Functional Requirements:**
- NFR3.1: No global variables
- NFR3.2: Retry logic with exponential backoff
- NFR3.3: Context-based cancellation
- NFR3.4: Unit tests with mocked canvas API
- NFR3.5: Integration tests with sample widget data

**Dependencies:**
- `core.Config` for configuration
- `canvusapi.Client` for API operations
- OpenAI client for analysis generation
- Context for cancellation

**Acceptance Criteria:**
- AC3.1: Package compiles independently
- AC3.2: Canvas analysis works identically
- AC3.3: Widget filtering works correctly
- AC3.4: Unit tests pass with mocked dependencies
- AC3.5: No global state in package

### R4: Extract OCR Processing Package

**Priority:** P2 (Medium)
**Scope:** Discovered during analysis

**Description:** Create a standalone `ocrprocessor` package for Google Vision OCR functionality.

**Current State:**
- OCR functions scattered in handlers.go:
  - `handleSnapshot` (lines 899-1068)
  - `performGoogleVisionOCR` (lines 1782-1897)
  - `performOCR` (lines 1965-2040)
  - `validateGoogleAPIKey` (lines 2042-2101)
  - `processImage` (lines 1899-1963)

**Target State:**
- Package structure:
  ```
  ocrprocessor/
  ├── processor.go       # Main processor organism
  ├── vision.go          # Google Vision client molecule
  ├── validator.go       # API key validator atom
  └── processor_test.go  # Comprehensive tests
  ```

**Functional Requirements:**
- FR4.1: OCR processing via Google Vision API
- FR4.2: API key validation
- FR4.3: Image download and cleanup
- FR4.4: Progress notes during processing
- FR4.5: Retry logic for downloads

**Non-Functional Requirements:**
- NFR4.1: No global variables
- NFR4.2: Mocked Google Vision API for tests
- NFR4.3: Context-based cancellation
- NFR4.4: Comprehensive error handling

**Dependencies:**
- `core.Config` for API keys
- `canvusapi.Client` for downloads and note creation
- HTTP client for Google Vision API
- Context for cancellation

**Acceptance Criteria:**
- AC4.1: Package compiles independently
- AC4.2: OCR functionality works identically
- AC4.3: Unit tests pass with mocked API
- AC4.4: No global state in package

### R5: Eliminate Global Config Variable

**Priority:** P0 (Critical)
**Scope:** Phase 4, Item 19

**Description:** Remove the global `var config *core.Config` and refactor all code to use dependency injection.

**Current State:**
- Global variable at line 48: `var config *core.Config`
- Used by cleanup functions at bottom of handlers.go
- Violates dependency injection principles
- Makes testing difficult

**Target State:**
- Config passed as parameter to all functions
- No package-level mutable state
- Config stored in Monitor struct
- Cleanup functions accept config parameter

**Functional Requirements:**
- FR5.1: Pass config to all handler functions
- FR5.2: Store config in Monitor struct
- FR5.3: Update all function signatures
- FR5.4: Pass config to cleanup functions

**Non-Functional Requirements:**
- NFR5.1: Zero global mutable state in main package
- NFR5.2: All tests must work without global setup
- NFR5.3: No behavioral changes
- NFR5.4: Compile-time safety (no nil configs)

**Acceptance Criteria:**
- AC5.1: No global config variable exists
- AC5.2: All functions receive config via parameter
- AC5.3: All existing tests pass
- AC5.4: No nil pointer dereferences possible

### R6: Split Handler Logic into Atomic Functions

**Priority:** P1 (High)
**Scope:** Phase 4, Item 20

**Description:** Decompose large handler functions into atomic design hierarchy.

**Current State:**
- Large handler functions with embedded logic
- Example: `handleNote` (132 lines)
- Example: `createNoteFromResponse` (144 lines)
- Mixed abstraction levels
- Difficult to test individual steps

**Target State:**
- Clear atomic design hierarchy:
  - **Atoms:** Pure utility functions
  - **Molecules:** Simple 2-3 function compositions
  - **Organisms:** Feature-level handlers
  - **Pages:** main.go composition root

**Functional Requirements:**
- FR6.1: Extract validation logic to atoms
- FR6.2: Extract text processing to atoms
- FR6.3: Extract AI client creation to molecules
- FR6.4: Extract response creation to molecules
- FR6.5: Keep high-level coordination in organisms

**Examples of Extraction:**

**Atoms (Pure Functions):**
- `truncateText(text string, length int) string`
- `estimateTokenCount(text string) int`
- `parseFloat64Env(key string, defaultValue float64) float64`
- `validateUpdate(update Update) error`

**Molecules (Simple Compositions):**
- `updateNoteWithRetry(client, noteID, payload, config)`
- `clearProcessingStatus(client, noteID, text, config)`
- `getAbsoluteLocation(client, widget, config)`
- `createProcessingNote(client, update, config)`

**Organisms (Complex Feature Units):**
- `handleNote(update, client, config)` - orchestrates note processing
- `generateAIResponse(prompt, config, systemMessage)` - AI generation
- `createNoteFromResponse(content, id, update, isError, client, config)` - note creation

**Non-Functional Requirements:**
- NFR6.1: Each function has single responsibility
- NFR6.2: Maximum function length: 50 lines
- NFR6.3: Clear separation of concerns
- NFR6.4: Testable in isolation
- NFR6.5: Godoc for all exported functions

**Acceptance Criteria:**
- AC6.1: No function exceeds 50 lines
- AC6.2: Clear atomic design hierarchy
- AC6.3: Unit tests for all atoms
- AC6.4: Integration tests for organisms
- AC6.5: All existing functionality works

### R7: Create Unified Error Handling

**Priority:** P2 (Medium)
**Scope:** Quality improvement

**Description:** Standardize error handling across all packages with consistent patterns.

**Current State:**
- Mixed error handling approaches
- Some errors logged and swallowed
- Some errors returned
- Inconsistent error wrapping

**Target State:**
- Consistent error handling strategy:
  - Domain-specific error types
  - Error wrapping with context
  - Structured logging at error sites
  - Clear error propagation

**Functional Requirements:**
- FR7.1: Define error types per package
- FR7.2: Wrap errors with `fmt.Errorf("%w")`
- FR7.3: Log errors before returning
- FR7.4: Include relevant context in errors

**Error Type Examples:**
```go
// pdfprocessor/errors.go
type ExtractionError struct {
    Path string
    Err  error
}

type ChunkingError struct {
    TokenCount int
    Err        error
}

// imagegen/errors.go
type GenerationError struct {
    Provider string
    Prompt   string
    Err      error
}

type DownloadError struct {
    URL string
    Err error
}
```

**Non-Functional Requirements:**
- NFR7.1: Error types implement error interface
- NFR7.2: Structured error information
- NFR7.3: Easy to test error conditions
- NFR7.4: Clear error messages for users

**Acceptance Criteria:**
- AC7.1: Each package defines error types
- AC7.2: All errors properly wrapped
- AC7.3: Error tests verify error types
- AC7.4: Logs include error context

### R8: Improve Test Coverage

**Priority:** P2 (Medium)
**Scope:** Quality improvement

**Description:** Achieve >80% test coverage across all packages with unit and integration tests.

**Current Test Coverage Analysis:**
- handlers.go: Minimal coverage (complex to test due to globals)
- core/config.go: Partial coverage
- core/ai.go: No tests
- canvusapi/: Minimal coverage
- monitorcanvus.go: No tests

**Target Coverage:**
- pdfprocessor: >80% coverage
- imagegen: >80% coverage
- canvasanalyzer: >80% coverage
- ocrprocessor: >80% coverage
- handlers: >80% coverage (after refactoring)
- core: 90% coverage
- canvusapi: >70% coverage

**Test Strategy:**

**Unit Tests:**
- Test atoms with table-driven tests
- Mock external dependencies
- Test error conditions
- Test edge cases

**Integration Tests:**
- Test molecules with real dependencies where safe
- Test organisms with mocked external APIs
- Test complete workflows end-to-end

**Test Organization:**
```
pdfprocessor/
├── extractor_test.go      # Unit tests for text extraction
├── chunker_test.go        # Unit tests for chunking
├── summarizer_test.go     # Unit tests with mocked AI
└── integration_test.go    # End-to-end PDF processing
```

**Non-Functional Requirements:**
- NFR8.1: Fast unit tests (<100ms each)
- NFR8.2: Table-driven test approach
- NFR8.3: Clear test naming (TestFunctionName_Scenario)
- NFR8.4: Test fixtures in testdata/

**Acceptance Criteria:**
- AC8.1: Overall coverage >80%
- AC8.2: All packages have unit tests
- AC8.3: Integration tests for workflows
- AC8.4: CI runs all tests

## Constraints

### Technical Constraints

**TC1: Backward Compatibility**
- Must maintain exact same external behavior
- API compatibility with Canvus
- Same configuration environment variables
- Same log output format

**TC2: No New Dependencies**
- Use existing dependencies only
- No new third-party packages
- Standard library preferred for new code

**TC3: Go Version**
- Target Go 1.x (current project version)
- No version-specific features

**TC4: Performance**
- No performance degradation
- Maintain concurrent processing limits
- Same memory footprint

### Process Constraints

**PC1: Incremental Migration**
- Refactor one package at a time
- Keep main branch working at all times
- Each package extraction is a separate PR/commit

**PC2: Testing Requirements**
- All tests must pass before merging
- New tests required for new packages
- No decrease in coverage

**PC3: Documentation**
- Godoc for all exported types/functions
- Package-level documentation
- Migration guide for each package

## Success Metrics

### Code Quality Metrics

**CQ1: Code Organization**
- handlers.go reduced from 2100 to <500 lines
- 4 new domain packages created
- Clear atomic design hierarchy

**CQ2: Test Coverage**
- Overall coverage increases to >80%
- All new packages have >80% coverage
- Integration tests for major workflows

**CQ3: Maintainability**
- Average function length <30 lines
- Cyclomatic complexity <10 per function
- Clear separation of concerns

**CQ4: Documentation**
- Godoc coverage 100%
- Package READMEs for new packages
- Architecture documentation updated

### Functional Metrics

**FM1: Feature Parity**
- All existing features work identically
- No user-facing changes
- Same error messages

**FM2: Performance**
- Processing time unchanged (±5%)
- Memory usage unchanged (±10%)
- Concurrent request handling unchanged

**FM3: Reliability**
- All existing tests pass
- No new bugs introduced
- Error handling improved

## Dependencies and Assumptions

### Dependencies

**D1: Existing Packages**
- `core` package provides Config
- `canvusapi` package provides Client
- `logging` package provides LogHandler

**D2: External Libraries**
- github.com/ledongthuc/pdf for PDF extraction
- github.com/sashabaranov/go-openai for AI
- Standard library for HTTP, JSON, context

**D3: Test Infrastructure**
- Go testing framework
- Table-driven test pattern
- Mock generation tools (optional)

### Assumptions

**A1: Configuration Stability**
- Config struct will not change during refactoring
- Environment variables remain the same

**A2: API Stability**
- Canvus API remains stable
- OpenAI API remains stable
- Google Vision API remains stable

**A3: Testing Environment**
- Test environment available for integration tests
- Ability to mock external APIs
- Access to sample PDF/image files

## Risks and Mitigations

### Technical Risks

**TR1: Breaking Changes**
- Risk: Refactoring introduces subtle behavioral changes
- Likelihood: Medium
- Impact: High
- Mitigation:
  - Comprehensive test suite before refactoring
  - Side-by-side comparison testing
  - Incremental rollout with monitoring

**TR2: Performance Regression**
- Risk: New package boundaries add overhead
- Likelihood: Low
- Impact: Medium
- Mitigation:
  - Benchmark critical paths before/after
  - Profile memory allocation
  - Optimize hot paths if needed

**TR3: Circular Dependencies**
- Risk: New packages create import cycles
- Likelihood: Low
- Impact: High
- Mitigation:
  - Clear dependency diagram before coding
  - Interfaces for abstraction
  - Careful package design

### Process Risks

**PR1: Scope Creep**
- Risk: Refactoring expands beyond planned scope
- Likelihood: Medium
- Impact: Medium
- Mitigation:
  - Strict adherence to requirements
  - Separate tickets for improvements
  - Regular scope reviews

**PR2: Testing Gaps**
- Risk: Tests don't cover all edge cases
- Likelihood: Medium
- Impact: High
- Mitigation:
  - Code review focus on tests
  - Coverage metrics enforcement
  - Integration test scenarios

## Out of Scope

The following items are explicitly OUT OF SCOPE for Phase 4:

**OS1: CGo Integration**
- llama.cpp embedding
- stable-diffusion.cpp embedding
- Native library management

**OS2: Installation**
- NSIS installer
- .deb packaging
- Windows Service integration

**OS3: New Features**
- Additional AI capabilities
- New widget types
- UI changes

**OS4: Configuration Changes**
- New environment variables
- Config file format changes
- Default value changes

**OS5: External API Changes**
- Canvus API updates
- OpenAI model changes
- Google Vision API updates

## Glossary

**Atomic Design:** Design methodology organizing code into atoms (pure functions), molecules (simple compositions), organisms (complex units), templates (contracts), and pages (composition roots).

**Dependency Injection:** Pattern where dependencies are passed to components rather than created internally or accessed globally.

**Global State:** Package-level or global variables that hold mutable state.

**Organism:** In atomic design, a complex component that combines molecules and atoms to provide a complete feature.

**Molecule:** In atomic design, a simple component that combines 2-3 atoms for a specific purpose.

**Atom:** In atomic design, the smallest indivisible unit with single responsibility and no dependencies.

**Page:** In atomic design, the composition root that wires together all components.
