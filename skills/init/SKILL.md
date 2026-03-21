---
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

Once you have collected all answers for the chosen mode, call the `init_project` MCP tool with the collected context as a single JSON object. Map each answer to the corresponding field:

- `project_name`: Derive a concise project name from the "what" answer
- `what`: Direct from the first answer (what are you building)
- `why`: Direct from the second answer (motivation)
- `who`: Direct from the third answer (target users) — use empty string for quick mode
- `done_criteria`: Direct from the "what does done/success look like" answer
- `tech_stack`: Direct from the tech stack answer — use empty string for quick mode
- `constraints`: Direct from the constraints answer — use empty string for quick/pr mode
- `risks`: Direct from the risks answer — use empty string for quick/pr mode
- `mode`: The init mode used ("full", "quick", or "pr")
- `pr_url`: The PR or issue URL if in pr mode — omit or use empty string for other modes

## Post-init

After the `init_project` tool returns successfully, display:

- "Project initialized: {project_name}"
- A brief summary of what was created (files written, project context captured)
- "Ready to proceed? (auto-continuing in 30 seconds...)"

Wait for the developer's response. If they say nothing, auto-proceed after 30 seconds and suggest next steps: running `/gsd-wired:status` to see the project dashboard, then planning the first phase.
