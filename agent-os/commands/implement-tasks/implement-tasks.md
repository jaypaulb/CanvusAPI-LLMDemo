Now that we have a spec and tasks list ready for implementation, we will proceed with implementation of this spec by following this multi-phase process:

PHASE 1: Determine which task group(s) from tasks.md should be implemented
PHASE 2: Implement the given task(s)
PHASE 3: After ALL task groups have been implemented, produce the final verification report.

Carefully read and execute the instructions in the following files IN SEQUENCE, following their numbered file names.  Only proceed to the next numbered instruction file once the previous numbered instruction has been executed.

Instructions to follow in sequence:

# PHASE 1: Determine Tasks

First, check if the user has already provided instructions about which task group(s) to implement.

**If the user HAS provided instructions:** Proceed to PHASE 2 to delegate implementation to the appropriate agent(s).

**If the user has NOT provided instructions:**

# PHASE 2: Implement Tasks

Now that you have the task group(s) to be implemented, proceed with implementation by following these instructions:

Implement all tasks assigned to you and ONLY those task(s) that have been assigned to you.

## Implementation process:

1. Analyze the provided spec.md, requirements.md, and visuals (if any)
2. Analyze patterns in the codebase according to its built-in workflow
3. Implement the assigned task group according to requirements and standards
4. Update `agent-os/specs/[this-spec]/tasks.md` to update the tasks you've implemented to mark that as done by updating their checkbox to checked state: `- [x]`

## Context Management (CRITICAL):

**IMPORTANT**: You are running as a subagent with limited context. Monitor your context usage to avoid hitting limits:

### Session Scope Guidelines:
- **Target**: Complete 1-2 issues per session (quality over quantity)
- **Early phase** (<20% of issues closed): May complete 2-3 smaller issues if trivial
- **Mid/late phase** (>20% closed): Focus on 1 issue per session for thoroughness

### When to End Session Early:

End your session cleanly if you observe ANY of these:
1. **Long responses**: Your responses are getting verbose or repetitive
2. **Lost context**: You're uncertain about earlier decisions in this session
3. **Approaching limits**: You sense the conversation is getting lengthy
4. **Multiple issues complete**: You've closed 2+ issues - time for a clean handoff
5. **Complex implementation**: Current issue requires extensive context that would fill remaining space

### How to End Cleanly:

If you need to end early:

1. **Commit current work** (even if incomplete):
   ```bash
   git add [modified-files]
   git commit -m "Work in progress: [what you've done]"
   git push
   ```

2. **Update issue status**:
   ```bash
   # If issue partially done:
   bd update <issue-id> --status in_progress --note "Partial implementation: [what's done]. Next: [what remains]."

   # If issue blocked:
   bd update <issue-id> --status blocked --note "Blocked by: [blocker]. Needs: [what's needed]."
   ```

3. **Write handoff note** to META issue:
   ```bash
   META_ID=$(cat .beads_project.json | jq -r '.meta_issue_id')
   bd comment $META_ID "Session ended early due to context limits.

   Completed:
   - [list what was finished]

   In Progress:
   - Issue: <issue-id>
   - Status: [percentage or description]
   - Remaining: [what's left]

   Next session should:
   - [specific next steps]"
   ```

4. **Return control**: Exit gracefully. Orchestrator will spawn a fresh agent to continue.

### Golden Rule:

**It's better to end with clean handoff notes than to hit context limits mid-task.**

The orchestrator will spawn a new session that can pick up exactly where you left off via Beads state.

## Guide your implementation using:
- **The existing patterns** that you've found and analyzed in the codebase.
- **Specific notes provided in requirements.md, spec.md AND/OR tasks.md**
- **Visuals provided (if any)** which would be located in `agent-os/specs/[this-spec]/planning/visuals/`
- **User Standards & Preferences** which are defined below.

## Self-verify and test your work by:
- Running ONLY the tests you've written (if any) and ensuring those tests pass.
- IF your task involves user-facing UI, and IF you have access to browser testing tools, open a browser and use the feature you've implemented as if you are a user to ensure a user can use the feature in the intended way.
  - Take screenshots of the views and UI elements you've tested and store those in `agent-os/specs/[this-spec]/verification/screenshots/`.  Do not store screenshots anywhere else in the codebase other than this location.
  - Analyze the screenshot(s) you've taken to check them against your current requirements.

## Commit and push your work:

After implementing and testing, commit your changes with a descriptive message. Use automatic fallback for git push operations:

**Git push with automatic fallback (SSH > HTTPS > gh CLI):**

```bash
# Stage your changes
git add [files-you-modified]

# Commit with descriptive message
git commit -m "Your descriptive commit message"

# Push with automatic fallback
PUSH_SUCCESS=false

# Method 1: Try normal git push (uses configured remote)
echo "Attempting git push..."
if git push 2>/dev/null; then
  PUSH_SUCCESS=true
  echo "✓ Pushed successfully"
else
  echo "Push failed, trying alternatives..."

  # Get current remote URL
  REMOTE_URL=$(git config --get remote.origin.url)

  # Method 2: If remote is HTTPS, try SSH
  if [[ "$REMOTE_URL" == https://* ]] && [ "$PUSH_SUCCESS" = false ]; then
    # Extract repo identifier and convert to SSH
    REPO_ID=$(echo "$REMOTE_URL" | sed -E 's#^https://github\.com/(.+)\.git$#\1#')
    SSH_URL="git@github.com:${REPO_ID}.git"

    echo "Attempting SSH push: $SSH_URL"
    if git push "$SSH_URL" $(git branch --show-current) 2>/dev/null; then
      PUSH_SUCCESS=true
      echo "✓ Pushed successfully with SSH"
      # Update remote to use SSH for future pushes
      git remote set-url origin "$SSH_URL"
    fi
  fi

  # Method 3: If remote is SSH, try HTTPS
  if [[ "$REMOTE_URL" == git@* ]] && [ "$PUSH_SUCCESS" = false ]; then
    # Extract repo identifier and convert to HTTPS
    REPO_ID=$(echo "$REMOTE_URL" | sed -E 's#^git@github\.com:(.+)\.git$#\1#')
    HTTPS_URL="https://github.com/${REPO_ID}.git"

    echo "Attempting HTTPS push: $HTTPS_URL"
    if git push "$HTTPS_URL" $(git branch --show-current) 2>/dev/null; then
      PUSH_SUCCESS=true
      echo "✓ Pushed successfully with HTTPS"
      # Update remote to use HTTPS for future pushes
      git remote set-url origin "$HTTPS_URL"
    fi
  fi

  # Method 4: Try gh CLI as last resort
  if [ "$PUSH_SUCCESS" = false ] && command -v gh &> /dev/null; then
    echo "Attempting gh CLI push..."
    if gh repo sync 2>/dev/null; then
      PUSH_SUCCESS=true
      echo "✓ Synced successfully with gh CLI"
    fi
  fi

  # Check if any method succeeded
  if [ "$PUSH_SUCCESS" = false ]; then
    echo "❌ ERROR: Failed to push changes"
    echo "Tried: git push, SSH, HTTPS, gh CLI"
    echo "Please check your git credentials and network connection"
    exit 1
  fi
fi
```

**Commit message guidelines:**
- Use descriptive, concise messages
- Start with a verb (Add, Fix, Update, Implement, etc.)
- Reference the issue/task being implemented
- Example: "Implement user authentication endpoints (Phase 1, Issue #123)"


## Display confirmation and next step

Display a summary of what was implemented.

# PHASE 3: Verify Implementation

Now that we've implemented all tasks in tasks.md, we must run final verifications and produce a verification report using the following MULTI-PHASE workflow:

## Workflow

### Step 1: Ensure tasks.md has been updated

Check `agent-os/specs/[this-spec]/tasks.md` and ensure that all tasks and their sub-tasks are marked as completed with `- [x]`.

If a task is still marked incomplete, then verify that it has in fact been completed by checking the following:
- Run a brief spot check in the code to find evidence that this task's details have been implemented
- Check for existence of an implementation report titled using this task's title in `agent-os/spec/[this-spec]/implementation/` folder.

IF you have concluded that this task has been completed, then mark it's checkbox and its' sub-tasks checkboxes as completed with `- [x]`.

IF you have concluded that this task has NOT been completed, then mark this checkbox with ⚠️ and note it's incompleteness in your verification report.


### Step 2: Update roadmap (if applicable)

Open `agent-os/product/roadmap.md` and check to see whether any item(s) match the description of the current spec that has just been implemented.  If so, then ensure that these item(s) are marked as completed by updating their checkbox(s) to `- [x]`.


### Step 3: Run entire tests suite

Run the entire tests suite for the application so that ALL tests run.  Verify how many tests are passing and how many have failed or produced errors.

Include these counts and the list of failed tests in your final verification report.

DO NOT attempt to fix any failing tests.  Just note their failures in your final verification report.


### Step 4: Create final verification report

Create your final verification report in `agent-os/specs/[this-spec]/verifications/final-verification.html`.

The content of this report should follow this structure:

```markdown
# Verification Report: [Spec Title]

**Spec:** `[spec-name]`
**Date:** [Current Date]
**Verifier:** implementation-verifier
**Status:** ✅ Passed | ⚠️ Passed with Issues | ❌ Failed

---

## Executive Summary

[Brief 2-3 sentence overview of the verification results and overall implementation quality]

---

## 1. Tasks Verification

**Status:** ✅ All Complete | ⚠️ Issues Found

### Completed Tasks
- [x] Task Group 1: [Title]
  - [x] Subtask 1.1
  - [x] Subtask 1.2
- [x] Task Group 2: [Title]
  - [x] Subtask 2.1

### Incomplete or Issues
[List any tasks that were found incomplete or have issues, or note "None" if all complete]

---

## 2. Documentation Verification

**Status:** ✅ Complete | ⚠️ Issues Found

### Implementation Documentation
- [x] Task Group 1 Implementation: `implementations/1-[task-name]-implementation.md`
- [x] Task Group 2 Implementation: `implementations/2-[task-name]-implementation.md`

### Verification Documentation
[List verification documents from area verifiers if applicable]

### Missing Documentation
[List any missing documentation, or note "None"]

---

## 3. Roadmap Updates

**Status:** ✅ Updated | ⚠️ No Updates Needed | ❌ Issues Found

### Updated Roadmap Items
- [x] [Roadmap item that was marked complete]

### Notes
[Any relevant notes about roadmap updates, or note if no updates were needed]

---

## 4. Test Suite Results

**Status:** ✅ All Passing | ⚠️ Some Failures | ❌ Critical Failures

### Test Summary
- **Total Tests:** [count]
- **Passing:** [count]
- **Failing:** [count]
- **Errors:** [count]

### Failed Tests
[List any failing tests with their descriptions, or note "None - all tests passing"]

### Notes
[Any additional context about test results, known issues, or regressions]
```
