# Slash Commands (Skills)

gsd-wired registers 8 slash commands as Claude Code skills. These orchestrate multi-agent workflows on the beads graph.

## Lifecycle Commands

### /gsd-wired:init

Initialize a new project with guided questioning.

```
/gsd-wired:init [full|quick|pr]
```

| Mode | Questions | Use case |
|------|-----------|----------|
| `full` | 12 questions | New greenfield project |
| `quick` | 3 questions | Quick setup |
| `pr` | 4 questions | Scope to a single PR |

Creates project epic + context child beads, writes PROJECT.md and .gsdw/config.json.

### /gsd-wired:research

Run parallel research for a phase.

```
/gsd-wired:research [phase_number]
```

Spawns 4 parallel research agents:
- **Stack** — technology choices and versions
- **Features** — feature implementation patterns
- **Architecture** — system design and integration
- **Pitfalls** — common mistakes and edge cases

Results are stored as child beads under a research epic and synthesized into a summary.

### /gsd-wired:plan

Create a dependency-aware phase plan.

```
/gsd-wired:plan [phase_number]
```

Generates 2-5 tasks with:
- Acceptance criteria
- File touch lists
- Complexity estimates
- Dependency graph (for wave computation)
- Requirement ID traceability

Validates requirement coverage across up to 3 iterations.

### /gsd-wired:execute

Execute the current wave of unblocked tasks.

```
/gsd-wired:execute [phase_number]
```

Runs tasks in parallel waves computed from the dependency graph. Each wave:
1. Queries ready (unblocked) tasks
2. Spawns parallel agents — one per task
3. Each agent claims its bead, executes, commits
4. Validates acceptance criteria
5. Advances to next wave

### /gsd-wired:verify

Verify phase completion against success criteria.

```
/gsd-wired:verify [phase_number]
```

Checks each criterion via file existence, grep patterns, or test execution. On failure, creates remediation task beads and re-executes.

### /gsd-wired:ship

Create PR and advance to next phase.

```
/gsd-wired:ship [phase_number]
```

1. Generates PR summary from bead metadata (title, body, requirements checklist)
2. Creates PR via `gh` CLI
3. Advances phase (closes epic, unblocks next phase)

## Informational Commands

### /gsd-wired:status

Show current project status.

```
/gsd-wired:status
```

Displays: project name, current phase, plan progress, ready task count.

### /gsd-wired:ready

Show unblocked tasks ready to work on.

```
/gsd-wired:ready [phase_number]
```

Lists tasks grouped by phase in tree format. Lightweight — model invocation enabled (no agent spawning).
