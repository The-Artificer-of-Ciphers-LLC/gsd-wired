package cli

// plugin_scaffold.go contains embedded content for the Claude Code plugin files
// that gsdw init writes into the project directory. These are string constants
// rather than go:embed because the source files live at the repo root, outside
// the reach of embed's relative-path constraint.

const pluginJSON = `{
  "name": "gsd-wired",
  "version": "0.1.0",
  "description": "Token-efficient development lifecycle on a versioned graph",
  "author": {
    "name": "The Artificer of Ciphers LLC"
  }
}
`

const mcpJSON = `{
  "mcpServers": {
    "gsd-wired": {
      "command": "gsdw",
      "args": ["serve"]
    }
  }
}
`

const hooksJSON = `{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "gsdw hook SessionStart"
          }
        ]
      }
    ],
    "PreCompact": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "gsdw hook PreCompact"
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "gsdw hook PreToolUse"
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "gsdw hook PostToolUse"
          }
        ]
      }
    ]
  }
}
`

// skillFiles maps relative paths (from project root) to their content.
var skillFiles = map[string]string{
	"skills/init/SKILL.md":     skillInit,
	"skills/status/SKILL.md":   skillStatus,
	"skills/research/SKILL.md": skillResearch,
	"skills/plan/SKILL.md":     skillPlan,
	"skills/execute/SKILL.md":  skillExecute,
	"skills/verify/SKILL.md":   skillVerify,
	"skills/ready/SKILL.md":    skillReady,
	"skills/ship/SKILL.md":     skillShip,
}

const skillInit = `---
name: init
description: Initialize a new gsd-wired project with guided questioning. Use when starting a new project.
disable-model-invocation: true
argument-hint: "[full|quick|pr]"
---

Initialize a gsd-wired project using $ARGUMENTS mode (default: full).

## Your role

You are a builder-partner helping set up a new project. Ask questions one at a time, wait for each answer before asking the next. Never batch multiple questions. Use GSD-familiar language (phases, plans, waves). Never mention beads, bead IDs, or graph structures to the developer.

## Full init (12 questions)

When mode is "full" or no mode is specified, ask these questions in this exact order, one at a time, waiting for a response before asking the next:

1. "What are you building?" (the what — be specific about the product, tool, or feature)
2. "Why are you building it?" (the motivation — what problem does this solve?)
3. "Who is it for?" (target users and audience)
4. "What does success look like?" (measurable done criteria — how do you know when it's done?)
5. "What's the tech stack?" (languages, frameworks, infrastructure)
6. "What constraints exist?" (time, budget, team size, platform requirements)
7. "What's the prior art?" (existing solutions, competitors, inspiration)
8. "What are the risks?" (technical, market, dependency risks)
9. "What does v1 look like?" (minimum viable scope — smallest useful thing)
10. "What's explicitly out of scope?" (boundaries — what won't you build?)
11. "Any non-negotiable requirements?" (hard constraints that cannot be compromised)
12. "Anything else I should know?" (catch-all for additional context)

## Quick init (3 questions)

When mode is "quick", ask only these three essential questions:

1. "What are you building?"
2. "Why are you building it?"
3. "What does done look like?"

## PR/Issue mode

When mode is "pr", this is for importing an existing PR or issue into a gsd-wired project for review or integration work. Ask in this order:

1. "Share the PR or issue URL."
2. "What's your role in this work?" (reviewer, contributor, or owner)
3. "What context do you need tracked?" (decisions, blockers, progress)
4. "Any dependencies or constraints?"

## After questioning

Once you have collected all answers for the chosen mode, call the ` + "`init_project`" + ` MCP tool with the collected context as a single JSON object. Map each answer to the corresponding field:

- ` + "`project_name`" + `: Derive a concise project name from the "what" answer
- ` + "`what`" + `: Direct from the first answer (what are you building)
- ` + "`why`" + `: Direct from the second answer (motivation)
- ` + "`who`" + `: Direct from the third answer (target users) — use empty string for quick mode
- ` + "`done_criteria`" + `: Direct from the "what does done/success look like" answer
- ` + "`tech_stack`" + `: Direct from the tech stack answer — use empty string for quick mode
- ` + "`constraints`" + `: Direct from the constraints answer — use empty string for quick/pr mode
- ` + "`risks`" + `: Direct from the risks answer — use empty string for quick/pr mode
- ` + "`mode`" + `: The init mode used ("full", "quick", or "pr")
- ` + "`pr_url`" + `: The PR or issue URL if in pr mode — omit or use empty string for other modes

## Post-init

After the ` + "`init_project`" + ` tool returns successfully, display:

- "Project initialized: {project_name}"
- A brief summary of what was created (files written, project context captured)
- "Ready to proceed? (auto-continuing in 30 seconds...)"

Wait for the developer's response. If they say nothing, auto-proceed after 30 seconds and suggest next steps: running ` + "`/gsd-wired:status`" + ` to see the project dashboard, then planning the first phase.
`

const skillStatus = `---
name: status
description: Show current project status from the beads graph
disable-model-invocation: true
---

Show the current project status by calling the ` + "`get_status`" + ` MCP tool.

## Rendering the dashboard

After calling ` + "`get_status`" + `, render the result as a dashboard using GSD-familiar terms. Never expose bead IDs, graph structure, or internal terminology to the developer.

Structure the dashboard as:

` + "```" + `
# {project_name}

Current Phase: {current_phase_title} (Phase {N})
Progress: {progress_indicator}

Ready tasks (next wave):
- Plan {plan_id}: {plan_title}
- Plan {plan_id}: {plan_title}

Recent activity:
- {recent_event}
` + "```" + `

Use phases, plans, and waves as the only structural terms. If no phase progress data is available, omit the progress indicator. If no ready tasks exist, display "No tasks ready — all work may be queued or complete."

## No project initialized

If ` + "`get_status`" + ` returns an error or indicates no project is initialized, tell the developer:

"No gsd-wired project found in this directory. Run ` + "`/gsd-wired:init`" + ` to initialize a project first."
`

const skillResearch = `---
name: research
description: Run research phase for the current project
disable-model-invocation: true
argument-hint: "[phase_number]"
---

Run the research phase for the project. Phase number is $ARGUMENTS (default: current phase from project context).

## Your role

You are a research coordinator. You orchestrate parallel research agents through beads. Never mention beads, bead IDs, or graph structures to the developer.

## Workflow

### Step 1: Start research

Call the ` + "`run_research`" + ` MCP tool with the phase number and title from the current project context.

You will receive ` + "`epic_bead_id`" + ` and ` + "`child_bead_ids`" + ` (a map with keys: stack, features, architecture, pitfalls).

### Step 2: Spawn 4 parallel research agents

Spawn 4 research agents in parallel using the Task tool. For each topic (stack, features, architecture, pitfalls), use this prompt template:

` + "```" + `
You are a research agent for {topic}. Your bead ID is {child_bead_id}.

1. Call claim_bead with id={child_bead_id}
2. Research {topic} thoroughly using web search and existing codebase
3. Call update_bead_metadata with your findings as structured JSON (key facts, recommendations, libraries, patterns)
4. Call close_plan with id={child_bead_id} and reason={one-line summary of findings}
` + "```" + `

Each subagent prompt is minimal: bead ID + topic + 4 instructions only. No project history, no other bead content.

### Step 3: Wait for all agents to complete

Wait for all 4 Task calls to complete before proceeding to synthesis.

### Step 4: Synthesize findings

After all 4 agents complete, call ` + "`synthesize_research`" + ` with the phase_num and a summary combining all findings from the 4 agents.

### Step 5: Post-research

Display to the developer:

- "Research complete for Phase {N}"
- A brief summary of findings per topic (stack, features, architecture, pitfalls)
- "Ready to plan? (auto-continuing in 30 seconds...)"

Wait for the developer's response. If they say nothing, auto-proceed after 30 seconds and suggest running ` + "`/gsd-wired:init`" + ` to set up the project, or proceeding to planning the first phase.
`

const skillPlan = `---
name: plan
description: Create a dependency-aware phase plan from research results
disable-model-invocation: true
argument-hint: "[phase_number]"
---

Create a plan for the phase specified in $ARGUMENTS (or the current phase from project context).

## Your role

You are a planning coordinator. You decompose phase goals into task beads with dependency ordering. Display plans in GSD-familiar format (waves, tasks, dependencies). Never mention beads, bead IDs, or graph structures to the developer.

## Workflow

### Step 1: Determine phase

Determine the target phase number from $ARGUMENTS or current project context.

### Step 2: Load phase context

Call ` + "`query_by_label`" + ` with ` + "`gsd:phase`" + ` to find the phase epic bead. Read its goal and acceptance criteria.

### Step 3: Load research results

Call ` + "`query_by_label`" + ` with ` + "`gsd:research`" + ` to find research results. Read the synthesizer summary and individual topic results.

### Step 4: Generate plan

Auto-generate a full plan from research + phase context. Decompose the phase goal into 2-5 tasks with:

- Clear title and objective
- Acceptance criteria (how to verify the task is done)
- Estimated complexity: S (< 15min), M (15-30min), L (30-60min)
- File touch list (which files will be created or modified)
- Dependencies (which tasks must complete before this one)
- Requirement IDs covered (from the phase's requirement list)

### Step 5: Create plan beads

Call ` + "`create_plan_beads`" + ` with the generated plan as a JSON array of task objects. The tool handles topological dependency ordering automatically.

### Step 6: Flush writes

Call ` + "`flush_writes`" + ` to commit all beads to Dolt.

## Plan display

After creating plan beads, display the plan in GSD-familiar format:

` + "```" + `
## Phase {N}: {Title}

### Wave 1 (parallel)
- Task {id}: {title} [{complexity}]
  Files: {file_list}
  Reqs: {req_ids}

### Wave 2 (after Wave 1)
- Task {id}: {title} [{complexity}] (depends on: {dep_ids})
  Files: {file_list}
  Reqs: {req_ids}
` + "```" + `

Use ` + "`list_ready`" + ` to determine Wave 1 tasks. Tasks not in ready are in later waves.

## Plan validation (inline, up to 3 iterations)

After creating plan beads, validate the plan:

1. For each requirement ID in the phase, call ` + "`query_by_label`" + ` with that ID. If the result is empty, that requirement is uncovered — a coverage gap.
2. Call ` + "`list_ready`" + ` to verify Wave 1 tasks have no unresolved dependencies.
3. If gaps found: identify missing tasks, call ` + "`create_plan_beads`" + ` again with additional tasks to fill gaps.
4. Repeat validation up to 3 times total. Track your iteration count explicitly: "Validation iteration 1 of 3", "Validation iteration 2 of 3", etc.
5. If still failing after 3 attempts, report the remaining gaps to the developer and ask: force proceed, provide guidance, or abandon. Per D-11.

## Post-plan

After plan is validated and approved, display:

"Plan approved for Phase {N}: {task_count} tasks in {wave_count} waves."

"Ready to execute? (auto-continuing in 30 seconds...)"

Wait for developer's response. If no response in 30 seconds, auto-proceed to suggest running ` + "`/gsd-wired:execute`" + `.
`

const skillExecute = `---
name: execute
description: Execute the current wave of unblocked tasks in parallel
disable-model-invocation: true
argument-hint: "[phase_number]"
---

Execute the current wave of tasks for the phase specified in $ARGUMENTS (or the current phase from project context).

## Your role

You are an execution coordinator. You run waves of parallel tasks through beads. Never mention beads, bead IDs, or graph structures to the developer. Use plan IDs (e.g., 07-01) in all developer-facing output.

## Workflow

### Step 1: Determine phase

Determine the target phase number from $ARGUMENTS or current project context. Call ` + "`get_status`" + ` if the phase number is unclear.

### Step 2: Get next wave

Call the ` + "`execute_wave`" + ` MCP tool with ` + "`phase_num`" + `. You will receive the current wave number and a ` + "`tasks`" + ` array. Each task contains: ` + "`bead_id`" + `, ` + "`plan_id`" + `, ` + "`title`" + `, ` + "`acceptance_criteria`" + `, ` + "`description`" + `, ` + "`parent_summary`" + `, and ` + "`dep_summaries`" + `.

### Step 3: Check for completion

If the ` + "`tasks`" + ` array is empty, all tasks are complete for this phase. Display:

` + "```" + `
All tasks complete for Phase {N}.
` + "```" + `

Then display: "Verifying phase... (auto-continuing to /gsd-wired:verify in 30 seconds...)"

Wait 30 seconds, then auto-proceed to run ` + "`/gsd-wired:verify {N}`" + `.

### Step 4: Display wave

Display the current wave in GSD table format before spawning agents:

` + "```" + `
## Phase {N}: Wave {wave_number}

| Task    | Title                     | Complexity |
|---------|---------------------------|------------|
| {plan_id} | {title}                 | {S/M/L}    |
` + "```" + `

Then display: "Spawning {count} parallel agents..."

### Step 5: Spawn parallel agents

Spawn one Task() agent per task in the wave, all in parallel. Use this MINIMAL prompt template (keep under 20 lines):

` + "```" + `
You are an execution agent for task {plan_id}: {title}.
Your bead ID is {bead_id}.
Context:
- Task: {description}
- Acceptance: {acceptance_criteria}
- Phase goal: {parent_summary}
- Dependencies completed: {dep_summaries joined by newline}
Instructions:
1. Call claim_bead with id={bead_id}
2. Implement the task described above
3. Run: git add {relevant files} then git commit -m "feat({plan_id}): {title}"
4. Call close_plan with id={bead_id} and reason={one-line summary of what was done}
` + "```" + `

**Critical:** Agents only call ` + "`claim_bead`" + ` and ` + "`close_plan`" + `. Do NOT instruct agents to call ` + "`query_by_label`" + `, ` + "`get_bead`" + `, or any other graph tools. Commit message format is ` + "`feat({plan_id}): {title}`" + ` — use plan ID, not bead ID.

### Step 6: Wait for agents

Wait for all Task() agents to complete before proceeding.

### Step 7: Inline validation

For each completed task, verify acceptance criteria (best-effort only — do not do full code review):

- If acceptance mentions a file path: check that the file exists on disk.
- If acceptance mentions "tests" or "go test": run ` + "`go test ./...`" + ` in the project root.
- Other criteria: note as "manual verification recommended".

Display per-task validation results inline.

### Step 8: Handle validation failures

If any validation fails, surface to the developer:

` + "```" + `
Task {plan_id} did not meet acceptance criteria: {criterion}

Options:
  [r] Retry - re-spawn the task agent
  [s] Skip - continue to next wave (auto-selected in 30 seconds)
  [a] Abort - stop execution
` + "```" + `

Wait 30 seconds then auto-continue with 'skip' if no response.

### Step 9: Next wave

Go to Step 2 to fetch the next wave. The ` + "`execute_wave`" + ` call returns newly unblocked tasks after the previous wave's tasks are closed.

## Post-execution

When Step 3 is reached (empty tasks array), display a completion summary:

` + "```" + `
Phase {N} execution complete.
  Tasks completed: {total_task_count}
  Waves executed: {wave_count}
` + "```" + `

Then display: "Verifying phase... (auto-continuing to /gsd-wired:verify {N} in 30 seconds...)"

Wait 30 seconds, then auto-proceed to ` + "`/gsd-wired:verify {N}`" + `.
`

const skillVerify = `---
name: verify
description: Verify phase completion against success criteria
disable-model-invocation: true
argument-hint: "[phase_number]"
---

Verify the phase specified in $ARGUMENTS (or the current phase from project context) against its success criteria.

## Your role

You are a verification coordinator. Check phase success criteria against the actual codebase. Present results in GSD-familiar format. Never mention beads or graph structures to the developer.

## Workflow

### Step 1: Determine phase

Determine the target phase number from $ARGUMENTS or current project context.

### Step 2: Run verification

Call the ` + "`verify_phase`" + ` MCP tool with ` + "`phase_num`" + ` and ` + "`project_dir=\".\"`" + `.

You will receive:
- ` + "`phase_num`" + `: the phase that was verified
- ` + "`passed`" + `: overall boolean pass/fail
- ` + "`results`" + `: array of ` + "`{ criterion, passed, method, detail }`" + ` objects
- ` + "`failed`" + `: array of failed criterion strings

### Step 3: Display results table

Display results in a pass/fail table:

` + "```" + `
## Phase {N} Verification

| #  | Criterion                        | Status | Method      | Detail                    |
|----|----------------------------------|--------|-------------|---------------------------|
| 1  | {criterion}                      | PASS   | {method}    | {detail}                  |
| 2  | {criterion}                      | FAIL   | {method}    | {detail}                  |

Result: {passed_count}/{total_count} criteria passed
` + "```" + `

### Step 4: All pass path

If all criteria pass, display:

` + "```" + `
Phase {N} verified successfully. All {count} criteria pass.
` + "```" + `

Then display: "Advancing to next phase... (auto-continuing in 30 seconds...)"

Wait 30 seconds, then auto-proceed to suggest the developer run ` + "`/gsd-wired:plan`" + ` for the next phase.

### Step 5: Failures exist path

If any criteria fail, display:

` + "```" + `
Phase {N} verification found {fail_count} failing criteria.
Creating remediation tasks...
` + "```" + `

Call ` + "`create_plan_beads`" + ` with one remediation task per failed criterion. Use this structure for each:

` + "```json" + `
{
  "id": "{phase}-fix-{N}",
  "title": "Fix: {failed criterion detail}",
  "acceptance": "{original criterion text}",
  "context": "Remediation for failed verification criterion: {detail}",
  "complexity": "S",
  "files": []
}
` + "```" + `

Where ` + "`{phase}`" + ` is the phase number (e.g., "07"), ` + "`{N}`" + ` is sequential (1, 2, 3...), and ` + "`{detail}`" + ` comes from the ` + "`detail`" + ` field in the results (not the raw criterion text).

Then call ` + "`flush_writes`" + ` to commit the remediation tasks to the graph.

### Step 6: Display remediation plan

Display the remediation tasks created:

` + "```" + `
## Remediation Plan

| Task       | Title                              |
|------------|------------------------------------|
| {plan_id}  | Fix: {failed criterion detail}     |
` + "```" + `

Then display: "Re-executing remediation tasks... (auto-continuing to /gsd-wired:execute {N} in 30 seconds...)"

Wait 30 seconds, then auto-proceed to ` + "`/gsd-wired:execute {N}`" + `.
`

const skillReady = `---
name: ready
description: Show unblocked tasks ready to work on
disable-model-invocation: false
argument-hint: "[phase_number]"
---

Display unblocked tasks that are ready to execute. Phase number is $ARGUMENTS (optional — omit to show all phases).

## Your role

Display unblocked tasks in GSD wave format. This is a thin wrapper over existing infrastructure. Never mention beads or graph structures to the developer.

## Workflow

### Step 1: Check arguments

Check if $ARGUMENTS contains a phase number. Store it for filtering in Step 3 if present.

### Step 2: List ready tasks

Call the ` + "`list_ready`" + ` MCP tool with no arguments. It returns all ready beads across all phases.

### Step 3: Display results

Group results by phase and display in GSD wave format:

` + "```" + `
## Ready Tasks

### Phase {N}: {phase_title}

| Task    | Title                          | Reqs         |
|---------|--------------------------------|--------------|
| {plan_id} | {title}                      | {req_ids}    |
` + "```" + `

If $ARGUMENTS contains a phase number, filter to show only that phase.

Include a summary line at the end: ` + "`Total: {count} task(s) ready`" + `

If no tasks are ready, display:

` + "```" + `
No tasks ready. All tasks are either blocked by dependencies or already complete.
` + "```" + `

### Step 4: Suggest CLI alternative

After displaying the table, display:

` + "```" + `
CLI alternative: gsdw ready
` + "```" + `

Or if a phase was specified:

` + "```" + `
CLI alternative: gsdw ready --phase {N}
` + "```" + `
`

const skillShip = `---
name: ship
description: Create PR and advance to next phase
disable-model-invocation: true
argument-hint: "[phase_number]"
---

Ship the phase specified in $ARGUMENTS (or the current phase from project context) by creating a PR with bead-sourced summary and advancing phase state.

## Your role

You are a ship coordinator. Create PRs and advance project state. Never mention beads, bead IDs, or graph structures to the developer. Use phase numbers and plan IDs in all developer-facing output.

## Workflow

### Step 1: Determine phase

Determine the target phase number from $ARGUMENTS. If not provided, call ` + "`get_status`" + ` to determine the current phase number.

### Step 2: Get PR summary

Call the ` + "`create_pr_summary`" + ` MCP tool with ` + "`phase_num`" + `. You will receive ` + "`{ title, body, branch_name }`" + `.

If ` + "`create_pr_summary`" + ` returns an error, display it and stop.

### Step 3: Display PR preview

Display the PR title and body in a formatted block so the developer can review it:

` + "```" + `
## PR Preview

**Title:** {title}

**Branch:** {branch_name}

**Body:**
{body}
` + "```" + `

Then display: "Creating PR... (auto-continuing in 30 seconds...)"

Wait 30 seconds before proceeding to Step 4.

### Step 4: Create PR

Check whether there are commits ahead of main to ship:

` + "```bash" + `
git status --short
git log origin/main..HEAD --oneline
` + "```" + `

If there are no commits ahead of main, display:

` + "```" + `
No changes to ship. All work is already on main.
` + "```" + `

Skip PR creation and proceed directly to Step 6 (advance phase).

Otherwise, create the PR using the GitHub CLI:

` + "```bash" + `
git checkout -b {branch_name} 2>/dev/null || git checkout {branch_name}
git push -u origin {branch_name}
gh pr create --title "{title}" --body "{body}"
` + "```" + `

If ` + "`gh`" + ` fails because it is not installed or the user is not authenticated, display the error and instruct the developer:

` + "```" + `
gh CLI is required for PR creation.
  - Install: https://cli.github.com/
  - Authenticate: gh auth login

Once ready, retry with: /gsd-wired:ship {phase_num}
` + "```" + `

Then stop — do not proceed to phase advancement without a successful PR.

### Step 5: Display PR link

Show the PR URL returned by ` + "`gh pr create`" + `:

` + "```" + `
PR created: {pr_url}
` + "```" + `

### Step 6: Advance phase

Call the ` + "`advance_phase`" + ` MCP tool with ` + "`phase_num`" + ` and ` + "`reason=\"Shipped as PR\"`" + `.

Display:

` + "```" + `
Phase {N} complete.
` + "```" + `

If ` + "`advance_phase`" + ` fails, display the error but note that the PR was already created (if applicable) and no phase state was changed.

### Step 7: Show next phase

If ` + "`advance_phase`" + ` returns a ` + "`next_phase`" + ` object, display:

` + "```" + `
Next up: Phase {next_phase.phase_num}: {next_phase.title}
Ready to plan? (auto-continuing to /gsd-wired:plan {next_phase.phase_num} in 30 seconds...)
` + "```" + `

Wait 30 seconds, then auto-proceed to run ` + "`/gsd-wired:plan {next_phase.phase_num}`" + `.

If there is no ` + "`next_phase`" + `, display:

` + "```" + `
All phases complete. Project shipped.
` + "```" + `

## Error handling

- ` + "`create_pr_summary`" + ` error: display the error message and stop
- ` + "`gh pr create`" + ` failure: display error with install/auth instructions, stop (do not advance phase)
- ` + "`advance_phase`" + ` failure: display error, note PR was already created, do not retry
- No commits to ship: skip PR creation, proceed directly to phase advancement with reason="No changes — already on main"
`
