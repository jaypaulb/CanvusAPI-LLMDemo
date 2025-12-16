# Autonomous Planning Loop

You are orchestrating a fully autonomous planning loop that will take a product vision through iterative spec development across multiple phases, culminating in a complete implementation plan.

This command automates the following workflow:
1. **Plan Product**: Create mission, roadmap, and tech stack
2. **Iterative Phase Loop**: For each phase in the roadmap, run shape-spec → write-spec → update plan-product
3. **Create Tasks**: Generate Beads issues with atomic design hierarchy

The loop is designed to be fully autonomous once started, with plan-product updated after each phase to incorporate learnings.

---

## PHASE 1: Initial Product Planning

### Step 1: Check for Existing Product Documentation

First, check if product planning files already exist:

```bash
ls -la agent-os/product/
```

### Step 2: Handle Existing Files (First Run Only)

{{IF product files exist}}

**Read existing files:**
```bash
cat agent-os/product/mission.md
cat agent-os/product/roadmap.md
cat agent-os/product/tech-stack.md
```

**Summarize findings to user:**

Display what you found:
```
I found existing product planning documentation:

**Mission**: [1-2 sentence summary of mission.md]
**Roadmap**: [List phases found, e.g., "5 phases identified: Phase 1 - Auth, Phase 2 - Dashboard..."]
**Tech Stack**: [Key technologies listed, e.g., "React, Node.js, PostgreSQL, etc."]
```

**Use AskUserQuestion tool to confirm completeness and fill gaps:**

Ask the user using the AskUserQuestion tool:
- Is the mission statement complete and accurate?
- Are all phases in the roadmap clearly defined?
- Is the tech stack documented for all layers (frontend, backend, database, deployment)?
- Are there any additional requirements, constraints, or features to add?

**IMPORTANT**: Use the AskUserQuestion tool for ALL user interactions. Do NOT present questions as plain text.

Based on user responses, delegate to **product-planner** subagent to:
- Update any incomplete sections
- Add missing details
- Refine existing content

{{ELSE}}

**No existing files - create from scratch:**

Delegate to the **product-planner** subagent with the user's product vision (if provided).

**IMPORTANT**: Instruct product-planner to use AskUserQuestion tool for ALL user interactions when gathering:
- Product idea, features, target users
- Phased roadmap structure
- Tech stack choices

The product-planner will create:
- `agent-os/product/mission.md` with product vision and strategy
- `agent-os/product/roadmap.md` with phased development plan
- `agent-os/product/tech-stack.md` documenting tech stack choices

{{ENDIF}}

Once complete (whether updated or created), proceed immediately to PHASE 2.

---

## PHASE 2: Extract Phases from Roadmap

Read the roadmap file to identify all phases for iterative spec development:

```bash
cat agent-os/product/roadmap.md
```

Parse the roadmap to extract phase titles. Typically these are organized as:
- Phase 1: [Phase Title]
- Phase 2: [Phase Title]
- Phase 3: [Phase Title]
- etc.

Create a list of phases to iterate through. Store this mentally for the loop.

---

## PHASE 3: Iterative Spec Development Loop

For EACH phase identified in the roadmap, execute the following sub-phases IN SEQUENCE:

### Sub-Phase A: Shape Spec for Current Phase

Run the shape-spec workflow for this specific phase:

{{@agent-os/commands/shape-spec/shape-spec.md}}

When initializing the spec, use the phase title as the spec name (e.g., "user-authentication" for "Phase 1: User Authentication").

### Sub-Phase B: Write Spec for Current Phase

Run the write-spec workflow for this specific phase:

{{@agent-os/commands/write-spec/write-spec.md}}

This creates `agent-os/specs/[phase-spec]/spec.md` with complete implementation details.

### Sub-Phase C: Update Plan-Product with Learnings (Autonomous - No User Questions)

After completing the spec for this phase, update the plan-product documents by calling the **product-planner** subagent with the context from the completed spec.

**IMPORTANT**: This is an autonomous update step. Do NOT ask the user any questions. All user input was already gathered during Sub-Phase A (shape-spec) and Sub-Phase B (write-spec).

1. **Read the newly created spec**:
   ```bash
   cat agent-os/specs/[phase-spec]/spec.md
   ```

2. **Delegate to product-planner** with the following context:

   Provide the product-planner with:
   - The completed spec for this phase (`agent-os/specs/[phase-spec]/spec.md`)
   - The current roadmap (`agent-os/product/roadmap.md`)
   - The current tech stack (`agent-os/product/tech-stack.md`)

   Instruct the product-planner to:
   - **Update roadmap.md**: Add technical notes to this phase's entry, mark it complete, update subsequent phases if dependencies changed
   - **Update tech-stack.md**: Add any new libraries, frameworks, or architectural decisions discovered during spec writing
   - **Preserve mission.md**: No changes needed to mission
   - **Do NOT ask user questions** - all context is in the spec

   The product-planner will identify key insights from the spec:
   - Technical dependencies discovered
   - Architectural decisions made (e.g., "JWT chosen for session management")
   - Constraints or requirements clarified
   - Integration points with other phases

   And incorporate them into the updated documents **autonomously**.

Example of what product-planner will add to roadmap.md:
```markdown
### Phase 1: User Authentication ✓
**Status**: Spec Complete
**Technical Notes**:
- JWT chosen for session management
- Password hashing via bcrypt
- Requires database schema migration before Phase 2
- API endpoints: POST /auth/login, POST /auth/register, POST /auth/logout

**Integration Points**:
- Phase 2 (User Dashboard) depends on auth middleware from this phase
- Phase 3 (Profile Management) uses user context established here
```

### Sub-Phase D: Continue to Next Phase

Repeat Sub-Phases A, B, C for the next phase in the roadmap.

**Autonomous Loop Execution**:
- DO NOT ask for permission between phases
- Automatically proceed to the next phase after updating plan-product
- Only stop if all phases are complete or if an error occurs

---

## PHASE 4: Create Implementation Tasks

After ALL phases have been shaped, written, and plan-product has been updated, create the implementation task breakdown:

{{@agent-os/commands/create-tasks/create-tasks.md}}


---

## PHASE 5: Output Summary

Display the following message to the user:

```
Autonomous planning loop complete!

**Product Planning** (updated iteratively):
- Mission: agent-os/product/mission.md
- Roadmap: agent-os/product/roadmap.md (✓ updated after each phase)
- Tech Stack: agent-os/product/tech-stack.md (✓ updated after each phase)

**Phases Completed**:
[List each phase with its spec location]
- Phase 1: [Phase Title] → agent-os/specs/[phase-spec]/spec.md
- Phase 2: [Phase Title] → agent-os/specs/[phase-spec]/spec.md
- Phase 3: [Phase Title] → agent-os/specs/[phase-spec]/spec.md
[etc.]

**Iterative Updates**:
The product-planner was called after each phase to update roadmap.md and tech-stack.md with:
- Technical decisions made during spec writing
- Dependencies discovered between phases
- Refined requirements for subsequent phases

```

---

## Autonomous Execution Notes

**This command is designed to run fully autonomously**:
- No user interaction required after initial product concept gathering
- Loops through all phases automatically
- Updates plan-product iteratively as context grows
- Handles multi-phase dependencies in Beads issue creation

**Expected Duration**:
- Depends on number of phases and complexity of each
- Typical range: 5-20 phases
- Each phase involves: shape (10-30 mins) + write (15-45 mins) + update (5-10 mins)

**Failure Handling**:
- If a phase fails to shape or write, STOP and report the error
- User can then provide clarification and resume from that phase
- Plan-product updates are cumulative, so no context is lost
