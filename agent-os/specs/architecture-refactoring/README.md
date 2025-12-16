# Phase 4: Codebase Architecture Refactoring - Specification

**Status:** Draft  
**Created:** 2025-12-16  
**Owner:** Development Team

## Overview

This specification defines the architectural refactoring of CanvusLocalLLM's Go backend to achieve clean atomic design architecture, eliminate global state, and improve modularity.

## Key Documents

1. **[spec.md](./spec.md)** - Complete technical specification (2,352 lines)
   - Current architecture analysis
   - Target architecture design
   - Package extraction plans
   - Migration strategy
   - Testing approach
   - Acceptance criteria

2. **[planning/requirements.md](./planning/requirements.md)** - Detailed requirements (692 lines)
   - 8 core requirements (R1-R8)
   - Success metrics
   - Constraints and assumptions
   - Risk analysis

## Quick Summary

### Problem
- handlers.go: 2,105 lines of mixed responsibilities
- Global config variable violates dependency injection
- Poor testability and modularity

### Solution
Extract 4 domain packages:
- `pdfprocessor/` - PDF analysis and summarization
- `imagegen/` - AI image generation (OpenAI/Azure)
- `canvasanalyzer/` - Canvas-wide analysis
- `ocrprocessor/` - Google Vision OCR

### Goals
- handlers.go: 2,105 → <500 lines (-76%)
- Test coverage: Current → >80%
- Zero global state
- Clear atomic design hierarchy

### Timeline
**17 days total:**
- Phase 1: Eliminate global config (2 days)
- Phase 2: Extract utilities (2 days)
- Phase 3: Extract pdfprocessor (3 days)
- Phase 4: Extract imagegen (3 days)
- Phase 5: Extract canvasanalyzer (2 days)
- Phase 6: Extract ocrprocessor (3 days)
- Phase 7: Final cleanup (2 days)

## Key Metrics

| Metric | Before | After | Change |
|--------|--------|-------|--------|
| handlers.go lines | 2,105 | <500 | -76% |
| Test coverage | ~15% | >80% | +65pp |
| Packages | 4 | 8 | +4 |
| Global variables | 1 | 0 | -100% |
| Avg function size | ~60 | <30 | -50% |

## Acceptance Criteria

**Functional:**
- ✓ All existing features work identically
- ✓ No user-facing changes
- ✓ API compatibility maintained

**Technical:**
- ✓ handlers.go <500 lines
- ✓ 4 new domain packages
- ✓ Zero global state
- ✓ Clear atomic design

**Quality:**
- ✓ >80% test coverage
- ✓ All packages documented
- ✓ Code review approved
- ✓ No performance regression

## Next Steps

1. Review specification documents
2. Approve migration plan
3. Create implementation tasks
4. Begin Phase 1: Eliminate global config

## Related Documents

- [Product Mission](../../product/mission.md)
- [Tech Stack](../../product/tech-stack.md)
- [Project CLAUDE.md](../../../CLAUDE.md)

---

For detailed information, see [spec.md](./spec.md) and [planning/requirements.md](./planning/requirements.md).
