Now that you have the spec.md AND/OR requirements.md, please break those down into an actionable tasks list with strategic grouping and ordering, by following these instructions:

# Task List Creation

## Core Responsibilities

1. **Analyze spec and requirements**: Read and analyze the spec.md and/or requirements.md to inform the tasks list you will create.
2. **Plan task execution order**: Break the requirements into a list of tasks in an order that takes their dependencies into account.
3. **Group tasks by specialization**: Group tasks that require the same skill or stack specialization together (backend, api, ui design, etc.)
4. **Create Tasks list**: Create the markdown tasks list broken into groups with sub-tasks.

## Workflow

### Step 1: Analyze Spec & Requirements

Read each of these files (whichever are available) and analyze them to understand the requirements for this feature implementation:
- `agent-os/specs/[this-spec]/spec.md`
- `agent-os/specs/[this-spec]/planning/requirements.md`

Use your learnings to inform the tasks list and groupings you will create in the next step.


### Step 2: Create Tasks Breakdown

Generate `agent-os/specs/[current-spec]/tasks.md`.

**Important**: The exact tasks, task groups, and organization will vary based on the feature's specific requirements. The following is an example format - adapt the content of the tasks list to match what THIS feature actually needs.

**Atomic Design Structure**: Organize tasks following atomic design hierarchy (atoms → molecules → organisms) and suggest appropriate agents for each task group.

```markdown
# Task Breakdown: [Feature Name]

## Overview
Total Tasks: [count]

## Task List

### Atoms Layer

#### Task Group 1: Core Utilities and Pure Functions
**Dependencies:** None
**Suggested Agent:** `atom-writer`
**Rationale:** Pure functions with zero dependencies - validation, formatting, calculations

- [ ] 1.0 Complete atoms layer
  - [ ] 1.1 Write 2-5 focused atom tests
    - Test pure input/output behaviors only
    - No mocking needed (atoms have no dependencies)
  - [ ] 1.2 Create validation atoms
    - `isValidEmail(email: string): boolean`
    - `isValidPassword(password: string): boolean`
    - `isValidPhone(phone: string): boolean`
  - [ ] 1.3 Create formatting atoms
    - `formatCurrency(amount: number): string`
    - `formatDate(date: Date): string`
  - [ ] 1.4 Create calculation atoms
    - [List specific calculations needed]
  - [ ] 1.5 Ensure atom tests pass
    - Run ONLY the 2-5 tests written in 1.1

**Acceptance Criteria:**
- All atom functions are pure (no side effects)
- Zero dependencies on other code
- All tests pass

### Molecules Layer

#### Task Group 2: Composed Helpers and Services
**Dependencies:** Task Group 1 (atoms)
**Suggested Agent:** `molecule-composer`
**Rationale:** Simple compositions of 2-3 atoms into cohesive helpers

- [ ] 2.0 Complete molecules layer
  - [ ] 2.1 Write 2-5 focused molecule tests
    - Test composition logic, minimal mocking
    - Mock only external dependencies if needed
  - [ ] 2.2 Create validation service molecule
    - Compose email + password + phone validators
    - Returns aggregated validation results
  - [ ] 2.3 Create formatting service molecule
    - Compose currency + date formatters
    - Provides consistent formatting interface
  - [ ] 2.4 Ensure molecule tests pass
    - Run ONLY the 2-5 tests written in 2.1

**Acceptance Criteria:**
- Each molecule composes 2-3 atoms
- No direct external dependencies (only atoms)
- All tests pass

### Database Organism Layer

#### Task Group 3: Data Models and Migrations
**Dependencies:** Task Group 2 (molecules for validation)
**Suggested Agent:** `database-layer-builder`
**Rationale:** Database models, migrations, associations - organism-level backend work

- [ ] 3.0 Complete database organism layer
  - [ ] 3.1 Write 2-8 focused tests for models
    - Test model validations using molecule validators
    - Test associations between models
    - Test critical model methods
  - [ ] 3.2 Create [Model] with validations
    - Fields: [list]
    - Validations: Use molecules from Task Group 2
    - Reuse pattern from: [existing model if applicable]
  - [ ] 3.3 Create migration for [table]
    - Add indexes for: [fields]
    - Foreign keys: [relationships]
  - [ ] 3.4 Set up associations
    - [Model] has_many [related]
    - [Model] belongs_to [parent]
  - [ ] 3.5 Ensure database organism tests pass
    - Run ONLY the 2-8 tests written in 3.1
    - Verify migrations run successfully

**Acceptance Criteria:**
- Models use atoms/molecules for validation
- All tests pass
- Migrations run successfully
- Associations work correctly

### API Organism Layer

#### Task Group 4: API Endpoints and Controllers
**Dependencies:** Task Group 3 (database organisms)
**Suggested Agent:** `api-layer-builder`
**Rationale:** API endpoints, controllers, authentication - organism-level backend API work

- [ ] 4.0 Complete API organism layer
  - [ ] 4.1 Write 2-8 focused tests for API endpoints
    - Test critical CRUD operations
    - Test authentication/authorization
    - Test error cases
  - [ ] 4.2 Create [resource] controller
    - Actions: index, show, create, update, destroy
    - Use database organisms from Task Group 3
    - Follow pattern from: [existing controller]
  - [ ] 4.3 Implement authentication/authorization
    - Use existing auth pattern
    - Add permission checks
  - [ ] 4.4 Add API response formatting
    - JSON responses
    - Error handling using atoms
    - Status codes
  - [ ] 4.5 Ensure API organism tests pass
    - Run ONLY the 2-8 tests written in 4.1
    - Verify critical CRUD operations work

**Acceptance Criteria:**
- All API endpoints functional
- Proper authorization enforced
- Consistent response format
- All tests pass

### UI Organism Layer

#### Task Group 5: UI Components and Pages
**Dependencies:** Task Group 4 (API organisms)
**Suggested Agent:** `ui-component-builder`
**Rationale:** UI components, forms, pages, styles - organism-level frontend work

- [ ] 5.0 Complete UI organism layer
  - [ ] 5.1 Write 2-8 focused tests for UI components
    - Test critical component behaviors
    - Test user interactions
    - Test API integration
  - [ ] 5.2 Create [Component] component atoms
    - Button, Input, Label (if not already in atoms)
  - [ ] 5.3 Create [Component] component molecules
    - FormField (Label + Input + Error)
    - Compose from atoms
  - [ ] 5.4 Create [Feature] organism component
    - Compose molecules + atoms
    - Connect to API endpoints from Task Group 4
  - [ ] 5.5 Build [View] page
    - Layout: [description]
    - Components: [list]
    - Match mockup: `planning/visuals/[file]`
  - [ ] 5.6 Apply styles
    - Follow existing design system
    - Use variables from: [style file]
  - [ ] 5.7 Implement responsive design
    - Mobile: 320px - 768px
    - Tablet: 768px - 1024px
    - Desktop: 1024px+
  - [ ] 5.8 Ensure UI organism tests pass
    - Run ONLY the 2-8 tests written in 5.1
    - Verify critical component behaviors work

**Acceptance Criteria:**
- UI follows atomic design (atoms → molecules → organisms)
- Components render correctly
- Forms validate and submit
- Matches visual design
- All tests pass

### Test Specialists

#### Task Group 6: Molecule-Level Testing
**Dependencies:** Task Group 2 (molecules)
**Suggested Agent:** `test-writer-molecule`
**Rationale:** Specialist in writing unit tests for molecule-level compositions

- [ ] 6.0 Complete molecule testing
  - [ ] 6.1 Write 2-5 unit tests per molecule
    - Test composition logic
    - Minimal mocking (molecules have few dependencies)
  - [ ] 6.2 Ensure all molecule tests pass

**Acceptance Criteria:**
- 2-5 tests per molecule
- All tests pass

#### Task Group 7: Organism-Level Testing
**Dependencies:** Task Groups 3, 4, 5 (organisms)
**Suggested Agent:** `test-writer-organism`
**Rationale:** Specialist in writing integration tests for organism-level layers

- [ ] 7.0 Complete organism testing
  - [ ] 7.1 Write 2-8 integration tests per organism layer
    - Database layer integration tests
    - API layer integration tests
    - UI layer integration tests
  - [ ] 7.2 Ensure all organism tests pass

**Acceptance Criteria:**
- 2-8 tests per organism layer
- Integration points tested
- All tests pass

#### Task Group 8: Test Gap Analysis
**Dependencies:** Task Groups 6, 7
**Suggested Agent:** `test-gap-analyzer`
**Rationale:** Specialist in identifying test coverage gaps and filling critical holes

- [ ] 8.0 Review tests and fill critical gaps
  - [ ] 8.1 Review existing tests from Task Groups 1-7
    - Atom tests (Task 1.1): ~2-5 tests
    - Molecule tests (Tasks 2.1, 6.1): ~4-10 tests
    - Organism tests (Tasks 3.1, 4.1, 5.1, 7.1): ~8-32 tests
    - Total existing tests: ~14-47 tests
  - [ ] 8.2 Analyze test coverage gaps for THIS feature only
    - Identify critical workflows lacking coverage
    - Focus on integration points between layers
    - Look for edge cases in organism interactions
  - [ ] 8.3 Write up to 10 additional strategic tests maximum
    - Focus on end-to-end workflows
    - Test cross-layer integration
    - Critical path scenarios only
  - [ ] 8.4 Run feature-specific tests only
    - Run ONLY tests for this feature
    - Expected total: ~24-57 tests maximum
    - Verify all critical workflows pass

**Acceptance Criteria:**
- All feature tests pass
- Critical workflows covered
- Maximum 10 additional tests added
- Testing focused on this feature only

### Integration

#### Task Group 9: System Integration and E2E Verification
**Dependencies:** Task Groups 3, 4, 5, 8 (all organisms + tests)
**Suggested Agent:** `integration-assembler`
**Rationale:** Specialist in wiring all organisms together and E2E verification

- [ ] 9.0 Complete integration
  - [ ] 9.1 Wire all organisms together
    - Connect database → API → UI flow
    - Verify data flows correctly through all layers
  - [ ] 9.2 Run full feature test suite
    - All tests from Task Groups 1-8
    - Verify E2E workflows
  - [ ] 9.3 Take screenshots/recordings
    - Document working feature
    - Capture key user flows

**Acceptance Criteria:**
- All layers integrated correctly
- All tests pass
- E2E workflows verified
- Documentation complete

## Execution Order

Recommended implementation sequence (follows atomic design dependency graph):
1. Atoms Layer (Task Group 1) - No dependencies
2. Molecules Layer (Task Group 2) - Depends on atoms
3. Database Organism Layer (Task Group 3) - Depends on molecules
4. API Organism Layer (Task Group 4) - Depends on database organisms
5. UI Organism Layer (Task Group 5) - Depends on API organisms
6. Molecule Testing (Task Group 6) - Depends on molecules
7. Organism Testing (Task Group 7) - Depends on organisms
8. Test Gap Analysis (Task Group 8) - Depends on all tests
9. Integration (Task Group 9) - Depends on all organisms + tests

**Agent Assignment:**
- Task Groups suggest specific agents (atom-writer, molecule-composer, etc.)
- Orchestration validates suggestions and can override if needed
- Minimal user input required
```

**Note**: Adapt this structure based on the actual feature requirements. Some features may need:
- Different task groups (e.g., email notifications, payment processing, data migration)
- Different execution order based on dependencies
- More or fewer sub-tasks per group

## Important Constraints

- **Create tasks that are specific and verifiable**
- **Group related tasks:** For example, group back-end engineering tasks together and front-end UI tasks together.
- **Limit test writing during development**:
  - Each task group (1-3) should write 2-8 focused tests maximum
  - Tests should cover only critical behaviors, not exhaustive coverage
  - Test verification should run ONLY the newly written tests, not the entire suite
  - If there is a dedicated test coverage group for filling in gaps in test coverage, this group should add only a maximum of 10 additional tests IF NECESSARY to fill critical gaps
- **Use a focused test-driven approach** where each task group starts with writing 2-8 tests (x.1 sub-task) and ends with running ONLY those tests (final sub-task)
- **Include acceptance criteria** for each task group
- **Reference visual assets** if visuals are available


## Display confirmation and next step

Display the following message to the user:

```
The tasks list has created at `agent-os/specs/[this-spec]/tasks.md`.

Review it closely to make sure it all looks good.

NEXT STEP 👉 Run `/implement-tasks` (simple, effective) or `/orchestrate-tasks` (advanced, powerful) to start building!
```
