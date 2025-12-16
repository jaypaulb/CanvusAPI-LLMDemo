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
