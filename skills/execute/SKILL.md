---
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

Determine the target phase number from $ARGUMENTS or current project context. Call `get_status` if the phase number is unclear.

### Step 2: Get next wave

Call the `execute_wave` MCP tool with `phase_num`. You will receive the current wave number and a `tasks` array. Each task contains: `bead_id`, `plan_id`, `title`, `acceptance_criteria`, `description`, `parent_summary`, and `dep_summaries`.

### Step 3: Check for completion

If the `tasks` array is empty, all tasks are complete for this phase. Display:

```
All tasks complete for Phase {N}.
```

Then display: "Verifying phase... (auto-continuing to /gsd-wired:verify in 30 seconds...)"

Wait 30 seconds, then auto-proceed to run `/gsd-wired:verify {N}`.

### Step 4: Display wave

Display the current wave in GSD table format before spawning agents:

```
## Phase {N}: Wave {wave_number}

| Task    | Title                     | Complexity |
|---------|---------------------------|------------|
| {plan_id} | {title}                 | {S/M/L}    |
```

Then display: "Spawning {count} parallel agents..."

### Step 5: Spawn parallel agents

Spawn one Task() agent per task in the wave, all in parallel. Use this MINIMAL prompt template (keep under 20 lines):

```
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
```

**Critical:** Agents only call `claim_bead` and `close_plan`. Do NOT instruct agents to call `query_by_label`, `get_bead`, or any other graph tools. Commit message format is `feat({plan_id}): {title}` — use plan ID, not bead ID.

### Step 6: Wait for agents

Wait for all Task() agents to complete before proceeding.

### Step 7: Inline validation

For each completed task, verify acceptance criteria (best-effort only — do not do full code review):

- If acceptance mentions a file path: check that the file exists on disk.
- If acceptance mentions "tests" or "go test": run `go test ./...` in the project root.
- Other criteria: note as "manual verification recommended".

Display per-task validation results inline.

### Step 8: Handle validation failures

If any validation fails, surface to the developer:

```
Task {plan_id} did not meet acceptance criteria: {criterion}

Options:
  [r] Retry - re-spawn the task agent
  [s] Skip - continue to next wave (auto-selected in 30 seconds)
  [a] Abort - stop execution
```

Wait 30 seconds then auto-continue with 'skip' if no response.

### Step 9: Next wave

Go to Step 2 to fetch the next wave. The `execute_wave` call returns newly unblocked tasks after the previous wave's tasks are closed.

## Post-execution

When Step 3 is reached (empty tasks array), display a completion summary:

```
Phase {N} execution complete.
  Tasks completed: {total_task_count}
  Waves executed: {wave_count}
```

Then display: "Verifying phase... (auto-continuing to /gsd-wired:verify {N} in 30 seconds...)"

Wait 30 seconds, then auto-proceed to `/gsd-wired:verify {N}`.
