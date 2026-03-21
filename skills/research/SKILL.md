---
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

Call the `run_research` MCP tool with the phase number and title from the current project context.

You will receive `epic_bead_id` and `child_bead_ids` (a map with keys: stack, features, architecture, pitfalls).

### Step 2: Spawn 4 parallel research agents

Spawn 4 research agents in parallel using the Task tool. For each topic (stack, features, architecture, pitfalls), use this prompt template:

```
You are a research agent for {topic}. Your bead ID is {child_bead_id}.

1. Call claim_bead with id={child_bead_id}
2. Research {topic} thoroughly using web search and existing codebase
3. Call update_bead_metadata with your findings as structured JSON (key facts, recommendations, libraries, patterns)
4. Call close_plan with id={child_bead_id} and reason={one-line summary of findings}
```

Each subagent prompt is minimal: bead ID + topic + 4 instructions only. No project history, no other bead content.

### Step 3: Wait for all agents to complete

Wait for all 4 Task calls to complete before proceeding to synthesis.

### Step 4: Synthesize findings

After all 4 agents complete, call `synthesize_research` with the phase_num and a summary combining all findings from the 4 agents.

### Step 5: Post-research

Display to the developer:

- "Research complete for Phase {N}"
- A brief summary of findings per topic (stack, features, architecture, pitfalls)
- "Ready to plan? (auto-continuing in 30 seconds...)"

Wait for the developer's response. If they say nothing, auto-proceed after 30 seconds and suggest running `/gsd-wired:init` to set up the project, or proceeding to planning the first phase.
