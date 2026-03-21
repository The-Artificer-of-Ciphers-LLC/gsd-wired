---
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

Call the `list_ready` MCP tool with no arguments. It returns all ready beads across all phases.

### Step 3: Display results

Group results by phase and display in GSD wave format:

```
## Ready Tasks

### Phase {N}: {phase_title}

| Task    | Title                          | Reqs         |
|---------|--------------------------------|--------------|
| {plan_id} | {title}                      | {req_ids}    |
```

If $ARGUMENTS contains a phase number, filter to show only that phase.

Include a summary line at the end: `Total: {count} task(s) ready`

If no tasks are ready, display:

```
No tasks ready. All tasks are either blocked by dependencies or already complete.
```

### Step 4: Suggest CLI alternative

After displaying the table, display:

```
CLI alternative: gsdw ready
```

Or if a phase was specified:

```
CLI alternative: gsdw ready --phase {N}
```
