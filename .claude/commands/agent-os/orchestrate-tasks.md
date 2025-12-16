# Process for Orchestrating Product Implementation

Orchestrate implementation across ALL specs/phases in parallel from project root.

**Run this from PROJECT ROOT, not from a spec folder.**

## Note on Autonomous Build

For autonomous parallel execution, use `/autonomous-build` directly.
It uses `bd` and `bv` exclusively for issue tracking and work selection:
- `bv --robot-plan` for dependency-respecting parallel tracks
- `bd ready`, `bd list`, `bd update` for issue management

This `/orchestrate-tasks` command is for **interactive guided orchestration** with user input.

## Multi-Phase Process

### FIRST: Verify Beads and Get All Phases


### NEXT: Analyze Parallel Execution Opportunities

Use BV to identify which work can run in parallel across ALL phases:


### NEXT: Create Orchestration Plan


### NEXT: Assign atomic design agents to work


### NEXT: Choose Execution Mode (Parallel or Sequential)




### NEXT: Delegate implementations

### NEXT: Delegate implementations to assigned subagents
