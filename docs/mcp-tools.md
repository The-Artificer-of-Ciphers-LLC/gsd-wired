# MCP Tools Reference

gsd-wired exposes 19 MCP tools via `gsdw serve`. These are available to Claude Code when the plugin is installed.

## Graph Operations

### create_phase

Creates a GSD phase as an epic bead in the beads graph.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| phase_num | int | yes | Phase number |
| title | string | yes | Phase title |
| goal | string | yes | Phase goal |
| acceptance | string | yes | Success criteria |
| req_ids | string[] | no | Requirement IDs (e.g., INFRA-01) |

### create_plan

Creates a plan as a task bead under a phase epic.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| plan_id | string | yes | Plan identifier (e.g., "01") |
| phase_num | int | yes | Parent phase number |
| parent_bead_id | string | yes | Epic bead ID |
| title | string | yes | Plan title |
| acceptance | string | yes | Acceptance criteria |
| context | string | no | Additional context |
| req_ids | string[] | no | Requirement IDs |
| dep_bead_ids | string[] | no | Dependency bead IDs |

### create_plan_beads

Batch-creates task beads from a structured plan with topological dependency resolution.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| phase_num | int | yes | Phase number |
| epic_bead_id | string | yes | Phase epic bead ID |
| tasks | object[] | yes | Array of task objects (id, title, acceptance, context, complexity, files, depends_on, req_ids) |

### get_bead

Retrieves a single bead by ID.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | string | yes | Bead ID |

### list_ready

Lists all unblocked (ready) beads. No parameters.

### query_by_label

Queries beads matching a label.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| label | string | yes | Label to match (e.g., gsd:phase, INFRA-02) |

### claim_bead

Atomically claims a bead for the current agent. Fails if already claimed.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | string | yes | Bead ID to claim |

### close_plan

Closes a plan bead and returns newly unblocked beads.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | string | yes | Bead ID to close |
| reason | string | yes | Close reason |

### flush_writes

Flushes accumulated batch writes to Dolt.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| message | string | no | Commit message |

## Lifecycle Operations

### init_project

Initializes a new project: runs bd init, creates project epic + context beads, writes PROJECT.md and .gsdw/config.json.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| project_name | string | yes | Project name |
| what | string | yes | What you're building |
| why | string | yes | Why you're building it |
| done_criteria | string | yes | How you'll know it's done |
| mode | string | yes | full, quick, or pr |
| who | string | no | Target users |
| tech_stack | string | no | Technology choices |
| constraints | string | no | Hard constraints |
| risks | string | no | Known risks |
| pr_url | string | no | PR URL (pr mode only) |

### get_status

Returns current project status. No parameters.

Returns: project name, current phase, ready tasks, phase counts.

### run_research

Creates a research epic with 4 child beads for parallel research agents.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| phase_num | int | yes | Phase to research |
| title | string | yes | Research title |
| req_ids | string[] | no | Requirement IDs |

Returns: epic_bead_id + child_bead_ids (stack, features, architecture, pitfalls).

### synthesize_research

Creates a summary bead combining all research findings.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| phase_num | int | yes | Phase number |
| summary | string | yes | Synthesized summary |

### execute_wave

Returns pre-computed context chains for all ready tasks in a phase.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| phase_num | int | yes | Phase number |

Returns: wave_number + tasks with bead context, parent summary, and dependency summaries.

### verify_phase

Checks phase success criteria against codebase state (file existence, grep patterns, test execution).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| phase_num | int | yes | Phase to verify |
| project_dir | string | no | Override project directory |

Returns: passed (bool) + per-criterion results + failed criteria list.

### create_pr_summary

Generates a bead-sourced PR summary for a phase.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| phase_num | int | yes | Phase number |

Returns: title, markdown body (with requirements checklist), branch_name.

### advance_phase

Closes a phase epic and returns next phase info.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| phase_num | int | yes | Phase to close |
| reason | string | yes | Completion reason |

Returns: closed bead, unblocked beads, next phase.

### get_tiered_context

Returns hot/warm/cold classified beads with budget-fitted context string.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| phase_num | int | yes | Phase number |
| budget_tokens | int | no | Token budget (default 2000) |

Returns: hot/warm/cold bead arrays + context_string + estimated_tokens.

### update_bead_metadata

Merges metadata key-value pairs into an existing bead. Used by research agents to store findings.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | string | yes | Bead ID to update |
| metadata | object | yes | Key-value pairs to merge into bead metadata |

Returns: updated bead.
