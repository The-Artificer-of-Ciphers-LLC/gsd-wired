---
name: status
description: Show current project status from the beads graph
disable-model-invocation: true
---

Show the current project status by calling the `get_status` MCP tool.

## Rendering the dashboard

After calling `get_status`, render the result as a dashboard using GSD-familiar terms. Never expose bead IDs, graph structure, or internal terminology to the developer.

Structure the dashboard as:

```
# {project_name}

Current Phase: {current_phase_title} (Phase {N})
Progress: {progress_indicator}

Ready tasks (next wave):
- Plan {plan_id}: {plan_title}
- Plan {plan_id}: {plan_title}

Recent activity:
- {recent_event}
```

Use phases, plans, and waves as the only structural terms. If no phase progress data is available, omit the progress indicator. If no ready tasks exist, display "No tasks ready — all work may be queued or complete."

## No project initialized

If `get_status` returns an error or indicates no project is initialized, tell the developer:

"No gsd-wired project found in this directory. Run `/gsd-wired:init` to initialize a project first."
