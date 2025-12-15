You are helping me shape and plan the scope for a new feature.  The following MULTI-PHASE process is aimed at documenting our key decisions regarding scope, design and architecture approach.

Carefully read and execute the instructions in the following files IN SEQUENCE, following their numbered file names.  Only proceed to the next numbered instruction file once the previous numbered instruction has been executed.

Instructions to follow in sequence:

# PHASE 1: Initialize Spec

The FIRST STEP is to initialize the spec by following these instructions:

# Spec Initialization

## Core Responsibilities

1. **Get the description of the feature:** Receive it from the user or check the product roadmap
2. **Initialize Spec Structure**: Create the spec folder with date prefix
3. **Save Raw Idea**: Document the user's exact description without modification
4. **Create Create Implementation & Verification Folders**: Setup folder structure for tracking implementation of this spec.
5. **Prepare for Requirements**: Set up structure for next phase

## Workflow

### Step 1: Get the description of the feature

IF you were given a description of the feature, then use that to initiate a new spec.

OTHERWISE follow these steps to get the description:

1. Check `@agent-os/product/roadmap.md` to find the next feature in the roadmap.
2. OUTPUT the following to user and WAIT for user's response:

```
Which feature would you like to initiate a new spec for?

- The roadmap shows [feature description] is next. Go with that?
- Or provide a description of a feature you'd like to initiate a spec for.
```

**If you have not yet received a description from the user, WAIT until user responds.**

### Step 2: Initialize Spec Structure

Determine a kebab-case spec name from the user's description, then create the spec folder:

```bash
# Get today's date in YYYY-MM-DD format
TODAY=$(date +%Y-%m-%d)

# Determine kebab-case spec name from user's description
SPEC_NAME="[kebab-case-name]"

# Create dated folder name
DATED_SPEC_NAME="${TODAY}-${SPEC_NAME}"

# Store this path for output
SPEC_PATH="agent-os/specs/$DATED_SPEC_NAME"

# Create folder structure following architecture
mkdir -p $SPEC_PATH/planning
mkdir -p $SPEC_PATH/planning/visuals

echo "Created spec folder: $SPEC_PATH"
```

### Step 3: Create Implementation Folder

Create 2 folders:
- `$SPEC_PATH/implementation/`

Leave this folder empty, for now. Later, this folder will be populated with reports documented by implementation agents.

### Step 4: Choose Issue Tracking Mode

{{IF beads_enabled}}
Ask the user which issue tracking mode to use for this spec:

```
Which issue tracking mode would you like to use for this spec?

**Option 1: tasks.md (Recommended for simpler features)**
- Hierarchical task groups in tasks.md
- Checkbox-based progress tracking
- Best for: Single-developer work, straightforward features

**Option 2: beads (Recommended for complex features)**
- Distributed issue tracking with atomic design hierarchy
- Auto-dependency management, cross-session context
- Best for: Multi-session work, complex features, team collaboration
- Requires: beads CLI installed (https://github.com/steveyegge/beads)

Your choice?
```

Wait for user response.

**After receiving choice:**

Create spec configuration file:

```bash
# Create spec-config.yml
cat > $SPEC_PATH/spec-config.yml <<EOF
# Spec Configuration
tracking_mode: [user-choice]  # beads or tasks_md
created: $(date -u +%Y-%m-%dT%H:%M:%SZ)
beads_initialized: false
EOF
```

**If user chose beads:**

Check if beads is installed and initialize:

```bash
# Check if beads is installed
if ! command -v bd &> /dev/null; then
    echo ""
    echo "⚠️  Beads is not installed."
    echo ""

    # Ask user what to do
    read -p "Would you like to (i)nstall beads now or (f)all back to tasks.md? [i/f]: " choice

    case "$choice" in
        i|I|install)
            echo ""
            echo "Installing beads..."
            curl -fsSL https://raw.githubusercontent.com/steveyegge/beads/main/scripts/install.sh | bash

            # Verify installation
            if ! command -v bd &> /dev/null; then
                echo "❌ Beads installation failed. Falling back to tasks.md mode."
                # Update config to use tasks.md instead
                sed -i 's/tracking_mode: beads/tracking_mode: tasks_md/' $SPEC_PATH/spec-config.yml
                echo "✓ Will use tasks.md for issue tracking"
                # Skip beads initialization
                beads_mode=false
            else
                echo "✓ Beads installed successfully"
                beads_mode=true
            fi
            ;;
        f|F|fallback|tasks)
            echo "Falling back to tasks.md mode..."
            # Update config to use tasks.md instead
            sed -i 's/tracking_mode: beads/tracking_mode: tasks_md/' $SPEC_PATH/spec-config.yml
            echo "✓ Will use tasks.md for issue tracking"
            beads_mode=false
            ;;
        *)
            echo "Invalid choice. Falling back to tasks.md mode..."
            sed -i 's/tracking_mode: beads/tracking_mode: tasks_md/' $SPEC_PATH/spec-config.yml
            echo "✓ Will use tasks.md for issue tracking"
            beads_mode=false
            ;;
    esac
else
    # Beads is already installed
    beads_mode=true
fi

# Initialize beads if we're in beads mode
if [[ "$beads_mode" == "true" ]]; then
    cd $SPEC_PATH
    bd init --stealth

    # Update config to mark beads as initialized
    sed -i 's/beads_initialized: false/beads_initialized: true/' $SPEC_PATH/spec-config.yml

    echo "✓ Beads initialized in spec folder"
    cd - > /dev/null

    # Check if bv (beads viewer) is installed
    if ! command -v bv &> /dev/null; then
        echo ""
        echo "⚠️  BV (beads viewer) is not installed."
        echo "   BV provides graph intelligence for better task selection."
        echo ""

        read -p "Would you like to (i)nstall bv now or (s)kip? [i/s]: " bv_choice

        case "$bv_choice" in
            i|I|install)
                echo ""
                echo "Installing bv..."
                # TODO: Replace with actual bv install command when available
                # For now, assume: cargo install bv (or similar)
                cargo install bv || {
                    echo "❌ BV installation failed. Continuing with basic beads (no graph intelligence)."
                    bv_available=false
                }

                # Verify installation
                if command -v bv &> /dev/null; then
                    echo "✓ BV installed successfully"
                    bv_available=true
                else
                    bv_available=false
                fi
                ;;
            s|S|skip)
                echo "Skipping bv installation. Graph intelligence features disabled."
                bv_available=false
                ;;
            *)
                echo "Invalid choice. Skipping bv installation."
                bv_available=false
                ;;
        esac
    else
        # BV already installed
        bv_available=true
        echo "✓ BV detected - graph intelligence enabled"
    fi

    # Update spec-config.yml with bv availability
    if [[ "$bv_available" == "true" ]]; then
        echo "bv_enabled: true" >> $SPEC_PATH/spec-config.yml
    else
        echo "bv_enabled: false" >> $SPEC_PATH/spec-config.yml
    fi
fi
```

**If user chose tasks_md:**

```bash
echo "✓ Will use tasks.md for issue tracking"
```

{{ELSE}}
Create spec configuration file with tasks_md (beads is disabled):

```bash
# Create spec-config.yml
cat > $SPEC_PATH/spec-config.yml <<EOF
# Spec Configuration
tracking_mode: tasks_md
created: $(date -u +%Y-%m-%dT%H:%M:%SZ)
beads_initialized: false
bv_enabled: false
EOF

echo "✓ Will use tasks.md for issue tracking (beads is disabled in config)"
```
{{ENDIF beads_enabled}}

### Step 5: Output Confirmation

Return or output the following:

```
Spec folder initialized: `[spec-path]`

Structure created:
- planning/ - For requirements and specifications
- planning/visuals/ - For mockups and screenshots
- implementation/ - For implementation documentation
- spec-config.yml - Spec configuration (tracking mode: [chosen-mode])

{{IF beads_enabled}}
Issue tracking: [chosen-mode]
{{IF tracking_mode_beads}}
- Beads initialized and ready
- Run `bd list` in spec folder to see issues
{{ELSE}}
- Will use tasks.md for task breakdown
{{ENDIF tracking_mode_beads}}
{{ELSE}}
Issue tracking: tasks.md
{{ENDIF beads_enabled}}

Ready for requirements research phase.
```

## Important Constraints

- Always use dated folder names (YYYY-MM-DD-spec-name)
- Pass the exact spec path back to the orchestrator
- Follow folder structure exactly
- Implementation folder should be empty, for now

# PHASE 2: Shape Spec

Now that you've initialized the folder for this new spec, proceed with the research phase.

Follow these instructions for researching this spec's requirements:

# Spec Research

## Core Responsibilities

1. **Read Initial Idea**: Load the raw idea from initialization.md
2. **Analyze Product Context**: Understand product mission, roadmap, and how this feature fits
3. **Ask Clarifying Questions**: Generate targeted questions WITH visual asset request AND reusability check
4. **Process Answers**: Analyze responses and any provided visuals
5. **Ask Follow-ups**: Based on answers and visual analysis if needed
6. **Save Requirements**: Document the requirements you've gathered to a single file named: `[spec-path]/planning/requirements.md`

## Workflow

### Step 1: Read Initial Idea

Read the raw idea from `[spec-path]/planning/initialization.md` to understand what the user wants to build.

### Step 2: Analyze Product Context

Before generating questions, understand the broader product context:

1. **Read Product Mission**: Load `agent-os/product/mission.md` to understand:
   - The product's overall mission and purpose
   - Target users and their primary use cases
   - Core problems the product aims to solve
   - How users are expected to benefit

2. **Read Product Roadmap**: Load `agent-os/product/roadmap.md` to understand:
   - Features and capabilities already completed
   - The current state of the product
   - Where this new feature fits in the broader roadmap
   - Related features that might inform or constrain this work

3. **Read Product Tech Stack**: Load `agent-os/product/tech-stack.md` to understand:
   - Technologies and frameworks in use
   - Technical constraints and capabilities
   - Libraries and tools available

This context will help you:
- Ask more relevant and contextual questions
- Identify existing features that might be reused or referenced
- Ensure the feature aligns with product goals
- Understand user needs and expectations

### Step 3: Generate First Round of Questions WITH Visual Request AND Reusability Check

Based on the initial idea, generate 4-8 targeted, NUMBERED questions that explore requirements while suggesting reasonable defaults.

**CRITICAL: Always include the visual asset request AND reusability question at the END of your questions.**

**Question generation guidelines:**
- Start each question with a number
- Propose sensible assumptions based on best practices
- Frame questions as "I'm assuming X, is that correct?"
- Make it easy for users to confirm or provide alternatives
- Include specific suggestions they can say yes/no to
- Always end with an open question about exclusions

**Required output format:**
```
Based on your idea for [spec name], I have some clarifying questions:

1. I assume [specific assumption]. Is that correct, or [alternative]?
2. I'm thinking [specific approach]. Should we [alternative]?
3. [Continue with numbered questions...]
[Last numbered question about exclusions]

**Existing Code Reuse:**
Are there existing features in your codebase with similar patterns we should reference? For example:
- Similar interface elements or UI components to re-use
- Comparable page layouts or navigation patterns
- Related backend logic or service objects
- Existing models or controllers with similar functionality

Please provide file/folder paths or names of these features if they exist.

**Visual Assets Request:**
Do you have any design mockups, wireframes, or screenshots that could help guide the development?

If yes, please place them in: `[spec-path]/planning/visuals/`

Use descriptive file names like:
- homepage-mockup.png
- dashboard-wireframe.jpg
- lofi-form-layout.png
- mobile-view.png
- existing-ui-screenshot.png

Please answer the questions above and let me know if you've added any visual files or can point to similar existing features.
```

**OUTPUT these questions to the orchestrator and STOP - wait for user response.**

### Step 4: Process Answers and MANDATORY Visual Check

After receiving user's answers from the orchestrator:

1. Store the user's answers for later documentation

2. **MANDATORY: Check for visual assets regardless of user's response:**

**CRITICAL**: You MUST run the following bash command even if the user says "no visuals" or doesn't mention visuals (Users often add files without mentioning them):

```bash
# List all files in visuals folder - THIS IS MANDATORY
ls -la [spec-path]/planning/visuals/ 2>/dev/null | grep -E '\.(png|jpg|jpeg|gif|svg|pdf)$' || echo "No visual files found"
```

3. IF visual files are found (bash command returns filenames):
   - Use Read tool to analyze EACH visual file found
   - Note key design elements, patterns, and user flows
   - Document observations for each file
   - Check filenames for low-fidelity indicators (lofi, lo-fi, wireframe, sketch, rough, etc.)

4. IF user provided paths or names of similar features:
   - Make note of these paths/names for spec-writer to reference
   - DO NOT explore them yourself (to save time), but DO document their names for future reference by the spec-writer.

### Step 5: Generate Follow-up Questions (if needed)

Determine if follow-up questions are needed based on:

**Visual-triggered follow-ups:**
- If visuals were found but user didn't mention them: "I found [filename(s)] in the visuals folder. Let me analyze these for the specification."
- If filenames contain "lofi", "lo-fi", "wireframe", "sketch", or "rough": "I notice you've provided [filename(s)] which appear to be wireframes/low-fidelity mockups. Should we treat these as layout and structure guides rather than exact design specifications, using our application's existing styling instead?"
- If visuals show features not discussed in answers
- If there are discrepancies between answers and visuals

**Reusability follow-ups:**
   - If user didn't provide similar features but the spec seems common: "This seems like it might share patterns with existing features. Could you point me to any similar forms/pages/logic in your app?"
- If provided paths seem incomplete you can ask something like: "You mentioned [feature]. Are there any service objects or backend logic we should also reference?"

**User's Answers-triggered follow-ups:**
- Vague requirements need clarification
- Missing technical details
- Unclear scope boundaries

**If follow-ups needed, OUTPUT to orchestrator:**
```
Based on your answers [and the visual files I found], I have a few follow-up questions:

1. [Specific follow-up question]
2. [Another follow-up if needed]

Please provide these additional details.
```

**Then STOP and wait for responses.**

### Step 6: Save Complete Requirements

After all questions are answered, record ALL gathered information to ONE FILE at this location with this name: `[spec-path]/planning/requirements.md`

Use the following structure and do not deviate from this structure when writing your gathered information to `requirements.md`.  Include ONLY the items specified in the following structure:

```markdown
# Spec Requirements: [Spec Name]

## Initial Description
[User's original spec description from initialization.md]

## Requirements Discussion

### First Round Questions

**Q1:** [First question asked]
**Answer:** [User's answer]

**Q2:** [Second question asked]
**Answer:** [User's answer]

[Continue for all questions]

### Existing Code to Reference
[Based on user's response about similar features]

**Similar Features Identified:**
- Feature: [Name] - Path: `[path provided by user]`
- Components to potentially reuse: [user's description]
- Backend logic to reference: [user's description]

[If user provided no similar features]
No similar existing features identified for reference.

### Follow-up Questions
[If any were asked]

**Follow-up 1:** [Question]
**Answer:** [User's answer]

## Visual Assets

### Files Provided:
[Based on actual bash check, not user statement]
- `filename.png`: [Description of what it shows from your analysis]
- `filename2.jpg`: [Key elements observed from your analysis]

### Visual Insights:
- [Design patterns identified]
- [User flow implications]
- [UI components shown]
- [Fidelity level: high-fidelity mockup / low-fidelity wireframe]

[If bash check found no files]
No visual assets provided.

## Requirements Summary

### Functional Requirements
- [Core functionality based on answers]
- [User actions enabled]
- [Data to be managed]

### Reusability Opportunities
- [Components that might exist already based on user's input]
- [Backend patterns to investigate]
- [Similar features to model after]

### Scope Boundaries
**In Scope:**
- [What will be built]

**Out of Scope:**
- [What won't be built]
- [Future enhancements mentioned]

### Technical Considerations
- [Integration points mentioned]
- [Existing system constraints]
- [Technology preferences stated]
- [Similar code patterns to follow]
```

### Step 7: Output Completion

Return to orchestrator:

```
Requirements research complete!

✅ Processed [X] clarifying questions
✅ Visual check performed: [Found and analyzed Y files / No files found]
✅ Reusability opportunities: [Identified Z similar features / None identified]
✅ Requirements documented comprehensively

Requirements saved to: `[spec-path]/planning/requirements.md`

Ready for specification creation.
```

## Important Constraints

- **MANDATORY**: Always run bash command to check visuals folder after receiving user answers
- DO NOT write technical specifications for development. Just record your findings from information gathering to this single file: `[spec-path]/planning/requirements.md`.
- Visual check is based on actual file(s) found via bash, NOT user statements
- Check filenames for low-fidelity indicators and clarify design intent if found
- Ask about existing similar features to promote code reuse
- Keep follow-ups minimal (1-3 questions max)
- Save user's exact answers, not interpretations
- Document all visual findings including fidelity level
- Document paths to similar features for spec-writer to reference
- OUTPUT questions and STOP to wait for orchestrator to relay responses


## Display confirmation and next step

Once you've completed your research and documented it, output the following message:

```
✅ I have documented this spec's research and requirements in `agent-os/specs/[this-spec]/planning`.

Next step: Run the command, `1-create-spec.md`.
```

After all steps complete, inform the user:

```
Spec initialized successfully!

✅ Spec folder created: `[spec-path]`
✅ Requirements gathered
✅ Visual assets: [Found X files / No files provided]

👉 Run `/write-spec` to create the spec.md document.
```
