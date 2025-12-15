## Atomic Design Principles

Build software from the smallest composable units upward. This hierarchy applies regardless of language or domain.

### The Compositional Hierarchy

| Level | Description | Examples |
|-------|-------------|----------|
| **Atom** | Smallest indivisible unit. No dependencies on peers. Single responsibility. Pure where possible. | Pure function, primitive type, single utility, constant |
| **Molecule** | Simple composition of 2-3 atoms. One clear purpose. | Function composing atoms, simple data structure, small helper |
| **Organism** | Complex composition of molecules/atoms. Cohesive feature unit. | Module, class, service, component with internal state |
| **Template** | Structural contract. Defines shape without implementation. | Interface, abstract class, schema, protocol |
| **Page** | Composition root. Entry point. Wires everything together. | Main function, app bootstrap, API endpoint handler |

### Core Principles

- **Build bottom-up**: Atoms first, then molecules, then organisms
- **Dependencies flow downward**: Each level only depends on levels below it, never above or sideways at the same level
- **Atoms are pure**: Keep atoms dependency-free where possible
- **Name by what things ARE**: Not what they're used for—this enables reuse
- **Composition over inheritance**: Prefer composition at every level
- **Resist premature complexity**: Don't create organisms when a molecule suffices

### The Abstraction Rule

**You must have ≥3 real examples before abstracting.**

- Second time you write similar code, write it again
- Third time, *consider* abstracting
- Imagined future cases don't count—only real, concrete examples

Three similar lines of code is better than a premature abstraction.

### Refactoring Guidance

When refactoring existing code:

1. **Identify buried atoms** — Extract pure, single-purpose functions first
2. **Find implicit molecules** — Repeated atom combinations should become explicit
3. **Decompose bloated organisms** — Organisms that do too much likely contain molecules trying to escape

### Test Implications

| Level | Testing Approach | Tests Per Unit |
|-------|------------------|----------------|
| Atoms | Unit tests, pure input/output | 1-3 focused tests |
| Molecules | Unit tests, minimal mocking | 2-5 focused tests |
| Organisms | Integration tests, may need mocks | 2-8 focused tests |
| Pages | End-to-end tests | Critical path only |

**Testing Philosophy:** Write the minimum tests needed to verify critical behavior. Avoid comprehensive test suites—they create maintenance burden. Focus on edge cases and integration points.

### Agent Responsibility Mappings

When using atomic design agents in agent-os, work is distributed by compositional level:

| Atomic Level | Agent | Responsibility |
|--------------|-------|----------------|
| **Atoms** | `atom-writer` | Pure functions, constants, utilities with zero peer dependencies |
| **Molecules** | `molecule-composer` | Compose 2-3 atoms into helpers, simple data structures, small services |
| **Organisms (Database)** | `database-layer-builder` | Models, migrations, associations, database schema |
| **Organisms (API)** | `api-layer-builder` | Controllers, endpoints, auth, response formatting |
| **Organisms (UI)** | `ui-component-builder` | Components, forms, pages, styles, responsive design |
| **Templates** | `template-designer` | Interfaces, schemas, protocols, type definitions |
| **Pages** | `integration-assembler` | Wire organisms together, E2E verification, composition roots |
| **Tests (Molecules)** | `test-writer-molecule` | Unit tests for molecules (minimal mocking) |
| **Tests (Organisms)** | `test-writer-organism` | Integration tests for organisms (2-8 per layer) |
| **Tests (Gap Analysis)** | `test-gap-analyzer` | Review all tests, identify gaps, write up to 10 additional tests |

### Dependency Flow Enforcement

**The Cardinal Rule:** Dependencies ONLY flow downward. Violations break composability.

**Valid Dependencies:**
- ✅ Molecule → Atom
- ✅ Organism → Molecule
- ✅ Organism → Atom
- ✅ Page → Organism
- ✅ Page → Molecule (sparingly)
- ✅ Page → Atom (rarely)

**Invalid Dependencies:**
- ❌ Atom → Molecule
- ❌ Atom → Organism
- ❌ Molecule → Organism
- ❌ Any → Page (pages are terminal, nothing should depend on them)
- ❌ Peer dependencies at same level (molecules shouldn't depend on other molecules directly)

**Agent Coordination:** Agents must complete work in sequence:
1. `atom-writer` completes first (no dependencies)
2. `molecule-composer` waits for atoms
3. Organism builders (`database-layer-builder`, `api-layer-builder`, `ui-component-builder`) wait for molecules
4. `integration-assembler` waits for all organisms

### Beads Integration Patterns

When using **beads** for issue tracking with atomic design:

#### Issue Hierarchy

Issues should mirror the atomic design hierarchy using beads parent-child relationships:

```
bd-a1b2e9         [epic] User Authentication Feature
bd-a1b2e9.1       [organism] Database Layer
bd-a1b2e9.1.1     [molecule] User validation helpers
bd-a1b2e9.1.1.1   [atom] Email validator function
bd-a1b2e9.1.1.2   [atom] Password strength checker
bd-a1b2e9.1.2     [molecule] Password hashing service
bd-a1b2e9.2       [organism] API Layer
bd-a1b2e9.2.1     [molecule] Auth middleware composer
bd-a1b2e9.3       [organism] UI Layer
bd-a1b2e9.3.1     [molecule] Login form component
```

Parent hash provides namespace for children—no coordination needed across agents.

#### Dependency Types

Use beads dependency types to enforce atomic design principles:

| Beads Dependency | Atomic Design Use Case |
|------------------|------------------------|
| **parent-child** | Hierarchical decomposition (epic → organism → molecule → atom) |
| **blocks** | Atoms must complete before molecules, molecules before organisms |
| **related** | Molecules that use similar atoms, organisms that share molecules |
| **discovered-from** | New atoms/molecules found while implementing organisms |

#### Agent Workflow with Beads

**Before starting work:**
```bash
# Find unblocked work at your atomic level
bd ready

# Recover cross-session context
bd list --json | jq '.[] | select(.status=="in_progress")'
```

**While working:**
```bash
# Mark issue in progress
bd update [issue-id] --status in_progress

# Discover new atomic work
bd create "New atom: validate phone number" -p 2
bd dep add [new-atom-id] [current-molecule-id] --type discovered-from

# Block dependent work until done
bd dep add [molecule-id] [atom-id] --type blocks
```

**After completing work:**
```bash
# Close completed issue
bd close [issue-id] --reason "Implemented with tests passing"
```

**Agent Memory:** Agents query beads on startup to understand:
- What work was previously completed (`status:closed`)
- What was in progress (`status:in_progress`)
- What was discovered (`discovered-from` links)
- What's unblocked and ready (`bd ready`)

#### Auto-Generation from Spec

When creating beads issues from spec.md:

1. **Identify organisms** in the spec (database layer, API layer, UI layer)
2. **Create parent epic** for the feature
3. **Create organism issues** as children of epic
4. **Analyze organisms** to identify required molecules
5. **Create molecule issues** as children of organisms
6. **Analyze molecules** to identify required atoms
7. **Create atom issues** as children of molecules
8. **Set blocking dependencies:** Atoms block molecules, molecules block organisms
9. **Result:** `bd ready` shows only atoms initially (nothing blocks them)

This ensures work flows bottom-up naturally through the dependency graph.
