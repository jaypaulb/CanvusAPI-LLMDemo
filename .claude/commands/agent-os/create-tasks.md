# Task List Creation Process

You are creating task breakdowns for all specs that need them.

## PHASE 1: Discover All Specs

First, find all specs in the project:

```bash
# Navigate to project root
cd /path/to/project

# Find all spec folders
SPEC_FOLDERS=$(find agent-os/specs -mindepth 1 -maxdepth 1 -type d)

echo "Found specs:"
for spec_folder in $SPEC_FOLDERS; do
    spec_name=$(basename "$spec_folder")
    echo "  • $spec_name"
done
```

## PHASE 2: Create Task Breakdown

**If tracking_mode_beads is enabled:**

Create Beads issues at PROJECT ROOT for all specs:

```bash
# Already at project root

# Check if .beads/ already exists
if [ -d ".beads" ]; then
    echo "✓ Beads already initialized at project root"
    echo "  Checking for missing issues..."
else
    echo "Initializing Beads at project root..."
    bd init --stealth
fi

# For each spec, create Beads issues if they don't exist
for spec_folder in $SPEC_FOLDERS; do
    spec_name=$(basename "$spec_folder")

    # Check if this spec already has issues
    EXISTING_ISSUES=$(bd list --label "$spec_name" --format json | jq '. | length')

    if [ "$EXISTING_ISSUES" -gt 0 ]; then
        echo "✓ $spec_name: $EXISTING_ISSUES issues already exist"
    else
        echo ""
        echo "Creating Beads issues for $spec_name..."

        # Follow create-beads-issues workflow for this spec
        # This will:
        # 1. Read spec.md and requirements.md
        # 2. Create phase epic with phase-N and spec-name labels
        # 3. Create organisms, molecules, atoms with proper hierarchy
        # 4. Set blocking dependencies

        # Source and run the workflow
        # {{@agent-os/workflows/implementation/create-beads-issues.md}}

        echo "✓ Created Beads issues for $spec_name"
    fi
done

echo ""
echo "✓ All specs have Beads issues at project root (.beads/)"
```

**If tracking_mode_beads is NOT enabled (fallback):**

Create tasks.md files for each spec that doesn't have one:

```bash
# Already at project root

for spec_folder in $SPEC_FOLDERS; do
    spec_name=$(basename "$spec_folder")

    # Check if tasks.md exists
    if [ -f "$spec_folder/tasks.md" ]; then
        echo "✓ $spec_name: tasks.md already exists"
    else
        echo ""
        echo "Creating tasks.md for $spec_name..."

        # Delegate to tasks-list-creator subagent
        # Provide:
        # - $spec_folder/spec.md (if exists)
        # - $spec_folder/planning/requirements.md (if exists)
        # - $spec_folder/planning/visuals/ (if exists)

        # The tasks-list-creator will create tasks.md in the spec folder

        echo "✓ Created tasks.md for $spec_name"
    fi
done

echo ""
echo "✓ All specs have tasks.md files"
```

## PHASE 3: Inform user

**If tracking_mode_beads is enabled:**

Output the following to inform the user:

```bash
# Show summary of created issues
TOTAL_ISSUES=$(bd list --format json | jq '. | length')
EPIC_COUNT=$(bd list --type epic --format json | jq '. | length')

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Beads Issues Created at Project Root"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Location: .beads/issues.jsonl"
echo "Total Issues: $TOTAL_ISSUES"
echo "Phases: $EPIC_COUNT"
echo ""
echo "View all phases:"
bd list --type epic --format json | jq -r '.[] | "  • \(.title) (\(.id))"'
echo ""
echo "View ready work (across all phases):"
bd ready --limit 5
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "NEXT STEPS"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Option 1: Run /orchestrate-tasks"
echo "  - Plan parallel execution across all phases"
echo "  - Assign agents to organisms"
echo "  - Maximize throughput"
echo ""
echo "Option 2: Run /auto-build"
echo "  - Launch autonomous coding harness"
echo "  - Fully autonomous implementation"
echo "  - Works through all phases automatically"
echo ""
```

**If tracking_mode_beads is NOT enabled (fallback):**

Output the following to inform the user:

```bash
# Count specs with tasks.md
TASK_COUNT=$(find agent-os/specs -name "tasks.md" | wc -l)

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Tasks Files Created for All Specs"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Total specs: $TASK_COUNT"
echo ""
echo "Specs with tasks.md:"
find agent-os/specs -name "tasks.md" | while read -r file; do
    spec_name=$(basename "$(dirname "$file")")
    echo "  • $spec_name"
done
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "NEXT STEP"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Run /orchestrate-tasks to plan implementation across all specs"
echo ""
```
