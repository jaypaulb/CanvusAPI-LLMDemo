# Auto-Build: Autonomous Implementation Harness

You are launching the autonomous coding harness to implement Beads issues created from specs.

This command:
1. Verifies Beads issues are ready
2. Prepares the project for autonomous execution
3. Launches the autonomous coding harness (converted from Linear to Beads)
4. Runs with unlimited iterations until all issues are closed

**Prerequisites**:
- Beads issues must be created (run `/autonomous-plan` or `/create-tasks` first)
- Autonomous coding harness must be installed at `/home/jaypaulb/Projects/gh/Linear-Coding-Agent-Harness/`
- `CLAUDE_CODE_OAUTH_TOKEN` environment variable must be set

---

## PHASE 1: Verify Beads Issues Exist

First, determine which spec we're building. If not already known, ask the user:

```
Which spec should I build?

Please provide the spec name (e.g., "user-authentication") or the full path to the spec folder.

Available specs:
[list directories in agent-os/specs/]
```

Once you know the spec, verify that Beads is initialized:

```bash
cd agent-os/specs/[this-spec]/

# Check for .beads/ directory
if [ -d ".beads" ]; then
  echo "✓ Beads initialized"
else
  echo "✗ Beads not initialized. Run /create-tasks first."
  exit 1
fi

# Check for issues
ISSUE_COUNT=$(bd list --json | jq '. | length')
echo "Found $ISSUE_COUNT Beads issues"

if [ "$ISSUE_COUNT" -eq 0 ]; then
  echo "✗ No Beads issues found. Run /create-tasks first."
  exit 1
fi

# Show ready work
echo ""
echo "Ready to implement:"
bd ready --limit 5
```

If no issues exist or `.beads/` directory is missing, STOP and output:

```
Error: Beads issues not found.

Please run one of the following first:
- /autonomous-plan (full planning loop + task creation)
- /create-tasks (task creation only, requires existing spec)

Then run /auto-build again.
```

---

## PHASE 1.5: Check for Changes and Cycles (BV)

Before starting the build, check what changed since the last build and verify no circular dependencies exist:

```bash
cd agent-os/specs/[this-spec]/

# Source BV helpers
source ../../../workflows/implementation/bv-helpers.md

if bv_available; then
    echo ""
    echo "=== Build Change Tracking ==="

    # Track changes since last build
    LAST_BUILD_COMMIT=$(cat .beads/last-build-commit 2>/dev/null || echo "HEAD~1")

    echo "Checking changes since last build ($LAST_BUILD_COMMIT)..."
    DIFF=$(bv --robot-diff --diff-since "$LAST_BUILD_COMMIT" --format json)

    # Log summary
    NEW_ISSUES=$(echo "$DIFF" | jq -r '.changes.new_issues | length')
    CLOSED_ISSUES=$(echo "$DIFF" | jq -r '.changes.closed_issues | length')
    MODIFIED_ISSUES=$(echo "$DIFF" | jq -r '.changes.modified_issues | length')
    NEW_CYCLES=$(echo "$DIFF" | jq -r '.graph_changes.new_cycles | length')

    echo "Changes detected:"
    echo "  New issues: $NEW_ISSUES"
    echo "  Closed issues: $CLOSED_ISSUES"
    echo "  Modified issues: $MODIFIED_ISSUES"
    echo "  New cycles: $NEW_CYCLES"

    # Warn if cycles introduced since last build
    if [[ "$NEW_CYCLES" -gt 0 ]]; then
        echo ""
        echo "❌ WARNING: New dependency cycles detected since last build!"
        echo "$DIFF" | jq -r '.graph_changes.new_cycles[] | "  Cycle: " + (. | join(" → "))'
        echo ""

        # Optionally block build
        read -p "Continue build despite cycles? [y/N]: " continue_build
        if [[ ! "$continue_build" =~ ^[Yy]$ ]]; then
            echo "Build aborted due to circular dependencies."
            echo "Fix cycles before running auto-build."
            exit 1
        fi
    fi

    # Check for existing cycles (even if not new)
    echo ""
    echo "=== Cycle Detection ==="
    if ! check_cycles; then
        echo ""
        echo "❌ ERROR: Circular dependencies exist in the graph!"
        echo "These must be resolved before autonomous build can proceed."
        echo ""
        read -p "Attempt build anyway (not recommended)? [y/N]: " force_build
        if [[ ! "$force_build" =~ ^[Yy]$ ]]; then
            exit 1
        fi
    else
        echo "✓ No circular dependencies detected"
    fi

    # Store current commit for next build comparison
    git rev-parse HEAD > .beads/last-build-commit
    echo ""
else
    echo ""
    echo "BV unavailable - skipping change tracking and cycle detection"
    echo ""
fi
```

---

## PHASE 2: Install/Update Harness

The autonomous harness is managed as a dependency. Check if it's installed, and install/update if needed:

```bash
# Read harness config from agent-os/config.yml
HARNESS_REPO=$(grep "autonomous_harness_repo:" ~/agent-os/config.yml | cut -d' ' -f2)
HARNESS_BRANCH=$(grep "autonomous_harness_branch:" ~/agent-os/config.yml | cut -d' ' -f2)
HARNESS_PATH="$HOME/.agent-os-harness"

echo "Harness repository: $HARNESS_REPO"
echo "Harness branch: $HARNESS_BRANCH"

# Install or update harness
if [ -d "$HARNESS_PATH" ]; then
  echo "✓ Harness found at $HARNESS_PATH"
  echo "Updating harness..."
  cd "$HARNESS_PATH"
  git fetch origin
  git checkout "$HARNESS_BRANCH"
  git pull origin "$HARNESS_BRANCH"
  cd -
else
  echo "Installing harness..."
  git clone "$HARNESS_REPO" "$HARNESS_PATH"
  cd "$HARNESS_PATH"
  git checkout "$HARNESS_BRANCH"
  cd -
fi

# Verify key files exist
if [ -f "$HARNESS_PATH/autonomous_agent_demo.py" ]; then
  echo "✓ Main entry point found"
else
  echo "✗ autonomous_agent_demo.py not found"
  exit 1
fi

if [ -f "$HARNESS_PATH/beads_config.py" ]; then
  echo "✓ Beads integration confirmed"
else
  echo "✗ beads_config.py not found. Harness may not be converted to Beads."
  exit 1
fi
```

---

## PHASE 3: Prepare Project for Autonomous Execution

The harness expects to work in a project directory with `.beads_project.json` marker file.

Determine the project root (typically the spec folder for single-spec builds, or workspace root for multi-spec):

```bash
# For single-spec build, use spec folder as project root
PROJECT_ROOT="$(pwd)"  # Should be agent-os/specs/[this-spec]/

# Create .beads_project.json marker if it doesn't exist
if [ ! -f ".beads_project.json" ]; then
  echo "Creating .beads_project.json marker..."

  # Get project metadata from Beads
  EPIC_ID=$(bd list --json | jq -r '.[] | select(.type=="epic") | .id' | head -1)
  EPIC_TITLE=$(bd show "$EPIC_ID" | grep "title:" | cut -d: -f2- | xargs)

  cat > .beads_project.json <<EOF
{
  "project_name": "$EPIC_TITLE",
  "epic_id": "$EPIC_ID",
  "initialized_at": "$(date -Iseconds)",
  "tracking_mode": "beads"
}
EOF

  echo "✓ Created .beads_project.json"
fi
```

---

## PHASE 4: Configure Harness for Pre-Populated Beads

The harness has two modes:
1. **Initializer mode**: Creates 50 issues from scratch (for new projects)
2. **Continuation mode**: Works on existing issues (for pre-populated Beads)

Since we've already created Beads issues via `/autonomous-plan` or `/create-tasks`, we want **continuation mode**.

The harness detects this automatically via `is_beads_initialized()` which checks for `.beads_project.json`.

Verify this will work:

```bash
# The harness checks: is_beads_initialized(project_dir)
# which reads .beads_project.json
# If it exists → continuation mode (skips initializer)
# If missing → initializer mode (runs initializer agent)

if [ -f ".beads_project.json" ]; then
  echo "✓ Harness will run in continuation mode (skip initializer)"
else
  echo "✗ .beads_project.json missing. Harness will create new issues instead of using existing ones."
  exit 1
fi
```

---

## PHASE 5: Launch Autonomous Harness

Launch the harness with unlimited iterations:

```bash
cd "$PROJECT_ROOT"

# Set environment variables
export CLAUDE_CODE_OAUTH_TOKEN="${CLAUDE_CODE_OAUTH_TOKEN}"

# Verify OAuth token is set
if [ -z "$CLAUDE_CODE_OAUTH_TOKEN" ]; then
  echo "Error: CLAUDE_CODE_OAUTH_TOKEN environment variable not set"
  echo "Please set it in your shell: export CLAUDE_CODE_OAUTH_TOKEN='your-token'"
  exit 1
fi

# Launch harness
echo ""
echo "========================================="
echo "Launching Autonomous Coding Harness"
echo "========================================="
echo "Project: $EPIC_TITLE"
echo "Issues: $ISSUE_COUNT total"
echo "Mode: Continuation (pre-populated Beads)"
echo "Max iterations: Unlimited"
echo ""
echo "The harness will:"
echo "1. Query ready work: bd ready"
echo "2. Claim highest priority issue: bd update <id> --status in_progress"
echo "3. Implement the issue following atomic design workflow"
echo "4. Run tests and close issue: bd close <id>"
echo "5. Repeat until all issues are closed"
echo ""
echo "Progress will be tracked in .beads/issues.jsonl"
echo "========================================="
echo ""

# Run the harness
python3 "$HARNESS_PATH/autonomous_agent_demo.py" \
  --project-dir "$PROJECT_ROOT" \
  --max-iterations 0  # 0 = unlimited

# Note: The harness will:
# - Detect .beads_project.json exists
# - Skip initializer agent
# - Start with coding agent in continuation mode
# - Use bd ready to find work
# - Use bd update/close to track progress
# - Run until all issues are closed or user stops it
```

**Expected Output**:

The harness will print session summaries after each iteration:

```
Session 1 Summary:
- Issues worked: bd-a1b2 (Email validator atom)
- Status: Closed (tests passing)
- Next ready: bd-a1b3 (Password validator atom)

Session 2 Summary:
- Issues worked: bd-a1b3 (Password validator atom)
- Status: Closed (tests passing)
- Next ready: bd-a1b4 (Phone formatter atom)

[continues until all issues closed]
```

---

## PHASE 6: Monitor Progress

While the harness runs, you can monitor progress in another terminal:

```bash
cd agent-os/specs/[this-spec]/

# View all issues
bd list --json | jq -r '.[] | "\(.status) | \(.id): \(.title)"' | column -t -s '|'

# View completed work
bd list --status closed --json | jq -r '.[] | "\(.id): \(.title)"'

# View in-progress work
bd list --status in_progress --json | jq -r '.[] | "\(.id): \(.title)"'

# View ready work (unblocked)
bd ready

# View dependency tree to see progress through atomic hierarchy
EPIC_ID=$(bd list --json | jq -r '.[] | select(.type=="epic") | .id' | head -1)
bd dep tree "$EPIC_ID"
```

**Atomic Design Progress Pattern**:

You should see issues close in this order:
1. **Atoms close first** (pure functions, no dependencies)
2. **Molecules unblock** as their atom dependencies close
3. **Molecules close** (composition of atoms)
4. **Organisms unblock** as their molecule dependencies close
5. **Organisms close** (database layer → API layer → UI layer)
6. **Tests unblock** as organisms close
7. **Tests close**
8. **Integration unblocks** as everything closes
9. **Integration closes** (final issue)

This bottom-up flow is enforced by the blocking dependencies set during issue creation.

---

## PHASE 7: Handle Completion or Interruption

### If Harness Completes All Issues:

The harness will exit when `bd ready` returns empty (all issues closed or blocked with no unblocked work).

Output to user:

```
Auto-build complete!

**Final Status**:
Total issues: [count]
Closed: [count]
Blocked: [count]
Open: [count]

**Dependency Tree**:
[show bd dep tree output to visualize what was built]

**Next Steps**:
1. Review implementation:
   - Check git log for commits
   - Run tests: [test command from tech-stack.md]
   - Start app: [start command from tech-stack.md]

2. If issues are blocked:
   - Investigate blockers: bd list --status blocked
   - Resolve manually or create new issues for discovered work

3. If integration is complete:
   - Perform manual QA
   - Take screenshots (Puppeteer integration available)
   - Deploy to staging
```

### If User Interrupts (Ctrl+C):

The harness saves state automatically in Beads. Output to user:

```
Auto-build interrupted.

**Progress Saved**:
All work is tracked in .beads/issues.jsonl (git-backed).

**Resume anytime**:
cd agent-os/specs/[this-spec]/
/auto-build

The harness will pick up where it left off using bd ready to find the next unblocked issue.

**Current State**:
[show bd list summary]
```

---

## Autonomous Execution Notes

**Fully autonomous after launch**:
- No user interaction required during execution
- Harness runs until all issues are closed or fatal error occurs
- Cross-session context recovery built-in (uses beads-context-recovery.md workflow)

**Session handoff**:
- Each coding iteration updates Beads with progress
- If context limit reached, harness spawns new session
- New session uses `bd ready` to find next work
- No context loss due to Beads' queryable state

**Discovered work**:
- If agent discovers new work during implementation, it creates new Beads issue
- Uses `discovered-from` link to track origin
- Sets blocking dependency if new work blocks current issue
- Automatically surfaces in `bd ready` when dependencies clear

**Error handling**:
- Test failures prevent issue closure (stays `in_progress`)
- Blocking issues automatically detected by `bd ready`
- Fatal errors stop harness, but state is preserved in Beads

**Expected Duration**:
- Depends on total issue count and complexity
- Atoms: 5-15 mins each
- Molecules: 10-30 mins each
- Organisms: 30-90 mins each
- Integration: 30-60 mins
- Typical full build: 4-12 hours for medium-sized feature

**Resource management**:
- Each session uses CLAUDE_CODE_OAUTH_TOKEN (count tokens)
- Long builds may require multiple days if paused/resumed
- Git commits created automatically by harness after each issue
