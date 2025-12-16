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

**If product files exist:**

Read existing files:
```bash
cat agent-os/product/mission.md
cat agent-os/product/roadmap.md
cat agent-os/product/tech-stack.md
```

Summarize findings to user:
```
I found existing product planning documentation:

**Mission**: [1-2 sentence summary of mission.md]
**Roadmap**: [List phases found, e.g., "5 phases identified: Phase 1 - Auth, Phase 2 - Dashboard..."]
**Tech Stack**: [Key technologies listed, e.g., "React, Node.js, PostgreSQL, etc."]
```

Use AskUserQuestion tool to confirm completeness and fill gaps:
- Is the mission statement complete and accurate?
- Are all phases in the roadmap clearly defined?
- Is the tech stack documented for all layers (frontend, backend, database, deployment)?
- Are there any additional requirements, constraints, or features to add?

**IMPORTANT**: Use the AskUserQuestion tool for ALL user interactions. Do NOT present questions as plain text.

Based on user responses, delegate to **product-planner** subagent to:
- Update any incomplete sections
- Add missing details
- Refine existing content

**If no existing files - create from scratch:**

Delegate to the **product-planner** subagent with the user's product vision (if provided).

**IMPORTANT**: Instruct product-planner to use AskUserQuestion tool for ALL user interactions when gathering:
- Product idea, features, target users
- Phased roadmap structure
- Tech stack choices

The product-planner will create:
- `agent-os/product/mission.md` with product vision and strategy
- `agent-os/product/roadmap.md` with phased development plan
- `agent-os/product/tech-stack.md` documenting tech stack choices

Once complete (whether updated or created), proceed immediately to PHASE 2.

---

## PHASE 2: Extract Phases from Roadmap

Read the roadmap file to identify all phases:

```bash
cat agent-os/product/roadmap.md
```

Parse the roadmap to extract phase information. Typically organized as:
```
1. [ ] Feature Name — Description `Effort`
2. [ ] Feature Name — Description `Effort`
3. [ ] Feature Name — Description `Effort`
```

For each item, extract:
- **Phase number**: 1, 2, 3, etc.
- **Feature name**: Convert to slug (e.g., "User Authentication" → "user-authentication")
- **Description**: Full description text
- **Effort**: XS, S, M, L, XL

**Display execution plan:**
```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Starting Parallel Spec Creation
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total Phases: [N]
All phases will run IN PARALLEL for maximum speed

Phases to create:
{{FOR each phase}}
  [#]. [Feature Name] ([Effort])
{{ENDFOR}}

Estimated time:
  Sequential: ~[N × 5] minutes
  Parallel: ~5-7 minutes (all at once + consistency check)
  Expected speedup: ~[percentage]%

Launching all spec creation tasks...
```

---

## PHASE 3: Launch All Specs in Parallel

Launch background tasks to create ALL specs simultaneously.

### Step 1: Launch Parallel Spec Creation Tasks

For EACH phase from the roadmap, launch a background task:

Use the Task tool with:
- `subagent_type='general-purpose'`
- `run_in_background=true`
- `description="Create spec for [phase-name]"`
- `prompt`:
  ```
  Create specification for Phase [#]: [Feature Name]

  1. Run shape-spec:
     {{@agent-os/commands/shape-spec/shape-spec.md}}

     Initialize spec with name: "[phase-name-slug]"

  2. Run write-spec:
     {{@agent-os/commands/write-spec/write-spec.md}}

  3. Return the spec path: agent-os/specs/[phase-name-slug]/spec.md

  IMPORTANT: This is part of an autonomous parallel workflow.
  - Do NOT ask user for permission between steps
  - Execute shape-spec and write-spec autonomously
  - This spec is being created in parallel with other phases
  ```

**Store task IDs:**
```
phase_tasks = {
  1: task_id_for_phase_1,
  2: task_id_for_phase_2,
  3: task_id_for_phase_3,
  ...
}
```

**Display progress:**
```
Launched [N] parallel spec creation tasks:
{{FOR each phase}}
  ▸ Task [task_id]: Phase [#] - [Feature Name]
{{ENDFOR}}

Waiting for all tasks to complete...
```

---

### Step 2: Wait for All Tasks to Complete

For each task_id in phase_tasks:

Use `TaskOutput(task_id=task_id, block=true, timeout=600000)` to wait for completion.

**Collect results:**
```
completed_specs = []
failed_specs = []

for phase_num, task_id in phase_tasks:
  result = TaskOutput(task_id)

  if result.success:
    completed_specs.append({
      "phase_num": phase_num,
      "spec_path": result.spec_path
    })
  else:
    failed_specs.append({
      "phase_num": phase_num,
      "error": result.error
    })
```

---

### Step 3: Handle Failures (if any)

**If ALL specs succeeded:**
Proceed to Step 4.

**If ANY spec failed:**

1. **Display partial completion:**
```
⚠️ Spec Creation Partial Completion

✅ Succeeded ([X] specs):
{{FOR each completed_spec}}
  - Phase [#]: [Feature Name] → [spec_path]
{{ENDFOR}}

❌ Failed ([Y] specs):
{{FOR each failed_spec}}
  - Phase [#]: [Feature Name]
    Error: [error message]
{{ENDFOR}}

The autonomous planning loop has STOPPED.
```

2. **STOP autonomous loop** - Do not proceed to consistency check

3. **Ask user for decision:**
```
Options:
1. Fix the issue and rerun autonomous-plan (it will skip completed specs)
2. Continue with only the successful specs (⚠️ roadmap will be incomplete)
3. Abort the autonomous planning loop
```

**Wait for user decision.**

---

### Step 4: Consistency Check and Consolidation

After ALL specs complete successfully, call **product-planner** once to:
- Review all specs for consistency
- Update roadmap with completion status
- Update tech-stack with all decisions
- Note any conflicts or inconsistencies

**Delegate to product-planner:**

Prompt:
```
Review all completed specs and update product planning documents.

**Completed specs:**
{{FOR each completed_spec}}
- Phase [#]: [Feature Name]
  Path: [spec_path]
{{ENDFOR}}

**Tasks:**

1. **Read all spec files** to understand technical decisions made

2. **Check for consistency:**
   - Do specs reference each other correctly?
   - Are there conflicting technical choices? (e.g., one spec uses Redux, another uses Zustand)
   - Are there missing integration points between phases?

3. **Update roadmap.md:**
   - Mark each phase as ✅ complete
   - Add brief technical notes for each phase
   - Note any integration points between phases

4. **Update tech-stack.md:**
   - Consolidate all technical decisions from all specs
   - Group by category (frontend, backend, database, infrastructure)
   - Remove duplicates

5. **Report inconsistencies (if any):**
   - List any conflicts found between specs
   - Suggest resolutions

IMPORTANT:
- This is autonomous - do NOT ask user questions
- If conflicts found, report them but do NOT stop execution
- Provide a summary of all updates made
```

**Display progress:**
```
Running consistency check across all [N] specs...
```

**Process product-planner response:**

If inconsistencies found:
```
⚠️ Consistency Check Complete (with warnings)

✅ Updated:
  - roadmap.md ([N] phases marked complete)
  - tech-stack.md (consolidated decisions)

⚠️ Inconsistencies Found:
[List inconsistencies from product-planner]

Suggested resolutions:
[Suggestions from product-planner]

You may want to review and resolve these before proceeding to task creation.

Continue to create tasks? (yes/no)
```

If no inconsistencies:
```
✅ Consistency Check Complete

All [N] specs reviewed and consolidated:
  ✓ roadmap.md updated
  ✓ tech-stack.md updated
  ✓ No conflicts detected

Proceeding to task creation...
```

---

## PHASE 4: Create Implementation Tasks

After ALL phases have been shaped, written, and plan-product has been updated, create the implementation task breakdown for all specs.

### Step 1: Discover All Specs

First, find all specs that were created:

```bash
# Find all spec folders
SPEC_FOLDERS=$(find agent-os/specs -mindepth 1 -maxdepth 1 -type d)

echo "Found specs:"
for spec_folder in $SPEC_FOLDERS; do
    spec_name=$(basename "$spec_folder")
    echo "  • $spec_name"
done
```

### Step 2: Check and Create Tasks for Each Spec

For each spec, check if tasks/beads already exist, and create them if missing:

**If tracking_mode_beads is enabled:**

Check if `.beads/` exists at project root. If not, create Beads issues for all specs:

```bash
# Should be at project root

if [ ! -d ".beads" ]; then
    echo "Creating Beads issues for all specs..."

    # Initialize Beads at project root
    bd init --stealth

    # For each spec, create Beads issues
    for spec_folder in $SPEC_FOLDERS; do
        spec_name=$(basename "$spec_folder")

        # Extract phase number from roadmap
        PHASE_NUM=$(grep -n "$spec_name" agent-os/product/roadmap.md | cut -d: -f1)

        echo ""
        echo "Creating Beads issues for Phase $PHASE_NUM: $spec_name"

        # Run create-beads-issues workflow for this spec
        # This will create issues with proper phase labels and dependencies
        # Follow: agent-os/workflows/implementation/create-beads-issues.md

    done

    echo ""
    echo "✓ Beads issues created for all $SPEC_COUNT specs"
else
    echo "✓ Beads already initialized at project root"
fi
```

**If tracking_mode_beads is NOT enabled (fallback to tasks.md):**

For each spec, check if `tasks.md` exists, and create it if missing:

```bash
for spec_folder in $SPEC_FOLDERS; do
    spec_name=$(basename "$spec_folder")

    if [ -f "$spec_folder/tasks.md" ]; then
        echo "✓ $spec_name: tasks.md already exists"
    else
        echo "Creating tasks.md for $spec_name..."

        # Delegate to tasks-list-creator subagent
        # Provide it with:
        # - $spec_folder/spec.md
        # - $spec_folder/planning/requirements.md
        # - $spec_folder/planning/visuals/

        echo "✓ Created tasks.md for $spec_name"
    fi
done

echo ""
echo "✓ All specs have tasks.md files"
```

### Step 3: Set Phase Dependencies (Beads mode only)

**If tracking_mode_beads is enabled:**

After all Beads issues are created, set phase dependencies:

```bash
# Already at project root

# For each phase after Phase 1, block on previous phase's integration
for phase_num in $(seq 2 $PHASE_COUNT); do
    prev_phase=$((phase_num - 1))

    # Find this phase's epic
    CURR_EPIC=$(bd list --label "phase-$phase_num" --type epic --json | jq -r '.[0].id')

    # Find previous phase's integration issue
    PREV_INTEGRATION=$(bd list --label "phase-$prev_phase" --json | \
        jq -r '.[] | select(.title | contains("[Integration]")) | .id')

    # Set dependency
    bd dep add "$CURR_EPIC" "$PREV_INTEGRATION" --type blocks

    echo "✓ Phase $phase_num blocks on Phase $prev_phase integration"
done

echo ""
echo "✓ Phase dependencies configured"
```

---

## PHASE 5: Output Summary

Display the following message to the user:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🎉 Autonomous Planning Loop Complete!
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

**Product Planning** (with parallel execution):
  📄 Mission: agent-os/product/mission.md
  📋 Roadmap: agent-os/product/roadmap.md (✓ updated with completion status)
  🔧 Tech Stack: agent-os/product/tech-stack.md (✓ consolidated from all specs)

**Execution Summary**:
  Total Phases: [N]
  Total Duration: [XhYm]
  All specs created in parallel ⚡

**Completed Specs**:
{{FOR each phase}}
  ✅ [#]. [Feature Name] → agent-os/specs/[phase-name-slug]/spec.md
{{ENDFOR}}

**Parallelization Speedup**:
  Sequential estimate: ~[N × 5] minutes
  Actual (parallel): ~[actual duration] minutes
  Time saved: ~[percentage]% ⚡

**Consistency Check**:
  [If inconsistencies were found:]
  ⚠️ Minor inconsistencies found and documented
  Review suggested resolutions in roadmap.md

  [If no inconsistencies:]
  ✅ All specs consistent with each other

```

---

## Autonomous Execution Notes

**This command uses maximum parallelization**:
- No user interaction required after initial product concept gathering
- ALL specs are created in parallel simultaneously
- Single consistency check at the end consolidates all learnings
- Dependencies are defined later during task creation (not during spec writing)

**Parallelization Strategy**:
- **Full parallelization**: All N phases run at the same time
- **Maximum speedup**: Regardless of roadmap structure, all specs run concurrently
- **Single consolidation**: One product-planner call at the end to review all specs
- **Consistency check**: Identifies conflicts or inconsistencies across specs

**Expected Duration**:
- **Sequential baseline**: N phases × ~5 min/phase = N×5 minutes total
- **With full parallelization**: ~5-7 minutes (max phase time + consistency check)
- **Speedup examples**:
  - 4 phases: 20 min → 7 min (~65% reduction)
  - 10 phases: 50 min → 7 min (~86% reduction)
  - 20 phases: 100 min → 7 min (~93% reduction)

**Failure Handling**:
- **Phase failure**: If ANY spec fails, let others complete, then STOP with partial completion report
- **User decision**: On failure, user can fix and rerun (skips completed specs), continue with partial, or abort
- **Consistency issues**: If conflicts found during consistency check, report them but continue to task creation

**Plan-product Updates**:
- Single update at the end (not per-phase)
- Consolidates all technical decisions from all specs
- Marks all phases as complete in roadmap
- Reports any inconsistencies found

**Resuming After Failure**:
1. Fix the issue that caused the failure
2. Rerun `/autonomous-plan`
3. Existing product files and completed specs are detected
4. Only missing/failed specs will be created
5. Consistency check runs again on all specs (including existing ones)
