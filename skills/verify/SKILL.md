---
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

Call the `verify_phase` MCP tool with `phase_num` and `project_dir="."`.

You will receive:
- `phase_num`: the phase that was verified
- `passed`: overall boolean pass/fail
- `results`: array of `{ criterion, passed, method, detail }` objects
- `failed`: array of failed criterion strings

### Step 3: Display results table

Display results in a pass/fail table:

```
## Phase {N} Verification

| #  | Criterion                        | Status | Method      | Detail                    |
|----|----------------------------------|--------|-------------|---------------------------|
| 1  | {criterion}                      | PASS   | {method}    | {detail}                  |
| 2  | {criterion}                      | FAIL   | {method}    | {detail}                  |

Result: {passed_count}/{total_count} criteria passed
```

### Step 4: All pass path

If all criteria pass, display:

```
Phase {N} verified successfully. All {count} criteria pass.
```

Then display: "Advancing to next phase... (auto-continuing in 30 seconds...)"

Wait 30 seconds, then auto-proceed to suggest the developer run `/gsd-wired:plan` for the next phase.

### Step 5: Failures exist path

If any criteria fail, display:

```
Phase {N} verification found {fail_count} failing criteria.
Creating remediation tasks...
```

Call `create_plan_beads` with one remediation task per failed criterion. Use this structure for each:

```json
{
  "id": "{phase}-fix-{N}",
  "title": "Fix: {failed criterion detail}",
  "acceptance": "{original criterion text}",
  "context": "Remediation for failed verification criterion: {detail}",
  "complexity": "S",
  "files": []
}
```

Where `{phase}` is the phase number (e.g., "07"), `{N}` is sequential (1, 2, 3...), and `{detail}` comes from the `detail` field in the results (not the raw criterion text).

Then call `flush_writes` to commit the remediation tasks to the graph.

### Step 6: Display remediation plan

Display the remediation tasks created:

```
## Remediation Plan

| Task       | Title                              |
|------------|------------------------------------|
| {plan_id}  | Fix: {failed criterion detail}     |
```

Then display: "Re-executing remediation tasks... (auto-continuing to /gsd-wired:execute {N} in 30 seconds...)"

Wait 30 seconds, then auto-proceed to `/gsd-wired:execute {N}`.
