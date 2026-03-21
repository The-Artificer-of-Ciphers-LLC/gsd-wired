# Phase 6: Research + Planning - Research

**Researched:** 2026-03-21
**Domain:** Claude Code SKILL.md multi-agent orchestration + Go MCP tool patterns for research/planning workflows
**Confidence:** HIGH

## Summary

Phase 6 adds two user-facing slash commands — `/gsd-wired:research` and `/gsd-wired:plan` — that orchestrate multi-agent workflows through the beads graph. The research workflow spawns 4 parallel Claude subagents via SKILL.md Task() instructions, each claiming a child bead (stack, features, architecture, pitfalls), writing results, then closing. A synthesizer agent reads all 4 closed beads and writes a summary. The planning workflow reads research results and the phase context, auto-generates a full dependency-aware plan as task beads, displays it in GSD-familiar wave format, and auto-proceeds to execution after 30 seconds.

The implementation follows the established Phase 5 pattern precisely: SKILL.md files orchestrate via natural language, MCP tools execute graph operations. New MCP tools needed: `run_research` (epic + 4 child beads + synthesizer coordination) and `create_plan_beads` (task beads with deps from a plan description). The plan checker is an inline SKILL.md instruction loop (up to 3 iterations) — no separate agent needed.

The key architectural insight for this phase: Claude Code's Task() tool is called from SKILL.md natural language instructions. The orchestrating Claude invokes Task() for each subagent, receives their results, and orchestrates sequentially or tracks parallel completion. Beads serve as the rendezvous point — subagents write to beads, synthesizer reads from beads — so no direct agent-to-agent communication is needed.

**Primary recommendation:** Follow the init_project.go handleInitProject() pattern for `run_research` (epic + children in one tool call), add `create_plan_beads` for batch task bead creation with dependency resolution, and implement plan checker as a SKILL.md inline loop using `query_by_label` + manual coverage verification.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Developer speaks plainly — say "research this" or "plan this" and gsdw handles all orchestration. Agent coordination, bead claiming, synthesizer triggering — all invisible.
- **D-02:** 30-second auto-proceed is the standard pattern. After plan approval, pause 30s for developer reaction, then flow into execution.
- **D-03:** SKILL.md orchestrates research. Natural language instructions tell Claude to spawn 4 agents, each claims a bead, writes results. No programmatic Go orchestration.
- **D-04:** Synthesizer coordination at Claude's discretion — optimize for performance and reliability. Developer doesn't care how it knows all 4 are done.
- **D-05:** Result storage at Claude's discretion — beads, files, or both. Developer just wants the research result available for planning.
- **D-06:** Fixed 4 research topics matching GSD: stack, features, architecture, pitfalls. Not configurable.
- **D-07:** Auto-generate full plan from research + context. No interactive task-by-task questioning. Same as GSD.
- **D-08:** Developer sees familiar GSD-style plan output — wave structure, task list with objectives, dependency graph. Not raw bead data.
- **D-09:** 30-second auto-proceed after plan display. Developer can interrupt to review or modify.
- **D-10:** Plan checker implementation at Claude's discretion — SKILL.md instruction, separate agent, or MCP tool. Optimize for performance/reliability.
- **D-11:** Same escalation as GSD: iterate up to 3 times. After 3 failures, ask developer (force proceed, provide guidance, abandon).
- **D-12:** Requirement coverage gate required — every phase requirement ID must appear in at least one task bead. Same as GSD's coverage check.
- **D-13:** After plan approved, 30-second auto-proceed then auto-flow into execution.

### Claude's Discretion
- Research agent spawning mechanics (Task tool calls from SKILL.md)
- Synthesizer trigger mechanism (poll, event, or instruction-based)
- Result storage format (bead content, bead metadata, files, or combination)
- Plan checker implementation approach
- Plan display format (exact layout of wave/task/dependency output)
- How to show GSD-familiar output while data lives in beads

### Deferred Ideas (OUT OF SCOPE)
- Execution of plans (Phase 7)
- PR creation from plan results (Phase 8)
- Token-aware research context loading (Phase 9)
- Direct Go import of beads library for performance (Phase 6 optimization path — deferred to v2)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| RSRCH-01 | Research phase creates epic bead with 4 child beads (stack, features, architecture, pitfalls) | `run_research` MCP tool mirrors `handleInitProject` pattern — epic + children in one call |
| RSRCH-02 | Each research agent claims its child bead via `bd update --claim` | `claim_bead` MCP tool already implemented; SKILL.md instructs subagent to call it |
| RSRCH-03 | Research results stored as bead content/metadata, not separate markdown files | `update_bead_metadata` + bead description field store results; `close_plan` with reason stores summary |
| RSRCH-04 | Synthesizer agent queries all 4 child beads when they close, produces summary bead | Synthesizer calls `query_by_label` with `gsd:research` + phase label; creates summary as child bead |
| PLAN-01 | User can create a phase plan via `/gsd-wired:plan` slash command | `skills/plan/SKILL.md` + `gsdw plan` CLI subcommand (mirrors init/status pattern) |
| PLAN-02 | Plan decomposes phase epic into task beads with dependencies | `create_plan_beads` MCP tool batch-creates task beads with `--deps` wiring |
| PLAN-03 | Each task bead has success criteria, estimated complexity, and file touch list | Stored as bead acceptance_criteria + metadata fields `{"complexity": "M", "files": [...]}` |
| PLAN-04 | Plan checker agent validates plan achieves phase goal before execution | Inline SKILL.md loop: query beads, check coverage, iterate up to 3× before escalating |
| CMD-03 | `/gsd-wired:plan` — Create phase plan (task beads with dependencies) | `skills/plan/SKILL.md` registered at skills/ root per Phase 5 pattern |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/modelcontextprotocol/go-sdk/mcp | v1.4.1 | MCP server + tool registration | Already in use — authoritative Google co-maintained SDK |
| github.com/spf13/cobra | existing | CLI subcommand for `gsdw plan` | Already in use — matches `gsdw init`, `gsdw status` pattern |
| encoding/json (stdlib) | Go 1.26.1 | JSON marshaling for tool args/results | Zero-dependency, already used everywhere |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| log/slog (stdlib) | Go 1.26.1 | Structured logging in MCP tools | All error paths that don't return toolError |
| os/exec (stdlib) | Go 1.26.1 | bd CLI invocation via graph.Client | Always — never call bd directly, use Client methods |
| testing (stdlib) | Go 1.26.1 | TDD with fake_bd pattern | All new MCP tool tests |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| SKILL.md Task() orchestration | Go goroutines in MCP tool | Go approach requires complex polling; SKILL.md is locked by D-03 |
| Inline plan checker in SKILL.md | Separate checker subagent | Separate agent adds latency and complexity; inline loop is simpler and faster |
| Bead metadata for research results | Separate markdown files | Files create .planning/ coupling we're trying to avoid; metadata keeps it in graph |

**Installation:** No new dependencies — all needed packages already in go.mod.

## Architecture Patterns

### Recommended Project Structure
```
internal/mcp/
├── run_research.go       # handleRunResearch() — epic + 4 child beads
├── create_plan_beads.go  # handleCreatePlanBeads() — batch task bead creation
├── tools.go              # +2 new tool registrations (was 10, becomes 12)
skills/
├── research/
│   └── SKILL.md          # /gsd-wired:research slash command
├── plan/
│   └── SKILL.md          # /gsd-wired:plan slash command
internal/cli/
├── research.go           # gsdw research CLI subcommand
├── plan.go               # gsdw plan CLI subcommand
```

### Pattern 1: SKILL.md Multi-Agent Orchestration via Task()

**What:** SKILL.md instructs the orchestrating Claude to spawn subagents using natural language that Claude Code resolves into Task() tool calls. Each subagent receives a focused prompt and the bead ID it owns.

**When to use:** Any time 4+ parallel workstreams need to run independently and rendezvous through shared state (beads).

**Example (in SKILL.md):**
```markdown
## Spawning research agents

After calling `run_research`, you will receive a JSON object with:
- `epic_bead_id`: the research phase epic
- `child_bead_ids`: {"stack": "...", "features": "...", "architecture": "...", "pitfalls": "..."}

Spawn 4 research agents in parallel. For each child bead, use the Task tool with this
prompt template:

"You are a research agent for [topic]. Your bead ID is [child_bead_id].
1. Call claim_bead with id=[child_bead_id]
2. Research [topic] thoroughly using web search and existing codebase
3. Call update_bead_metadata with your findings as structured JSON
4. Call close_plan with id=[child_bead_id] and reason=[one-line summary]"

Wait for all 4 Task() calls to complete before proceeding to synthesis.
```

**Confidence:** HIGH — confirmed by Phase 5 SKILL.md pattern (skills/init/SKILL.md spawns no subagents but demonstrates the natural language → MCP tool call chain that subagents would follow).

### Pattern 2: Research Epic + Children via Single MCP Tool

**What:** `run_research` creates the epic bead and all 4 child beads atomically (mirroring `handleInitProject`). Returns structured JSON with all bead IDs so SKILL.md can immediately dispatch subagents without additional graph queries.

**When to use:** Any time you need a parent + fixed set of children (same as init_project creating project + context beads).

**Example (Go, mirroring init_project.go):**
```go
// Source: internal/mcp/init_project.go (established pattern)
type runResearchResult struct {
    EpicBeadID   string            `json:"epic_bead_id"`
    ChildBeadIDs map[string]string `json:"child_bead_ids"` // {"stack":"...", "features":"...", ...}
}

func handleRunResearch(ctx context.Context, state *serverState, args runResearchArgs) (*mcpsdk.CallToolResult, error) {
    if err := state.init(ctx); err != nil {
        return toolError(err.Error()), nil
    }
    // Create research epic (gsd:research label)
    epic, err := state.client.CreatePhase(ctx, args.PhaseNum, "Research: "+args.PhaseTitle,
        "Research phase for "+args.PhaseTitle, "All 4 topics researched and synthesized",
        append([]string{"gsd:research"}, args.ReqIDs...))
    if err != nil {
        return toolError("create research epic: " + err.Error()), nil
    }
    // Create 4 child beads
    topics := []struct{ id, title string }{
        {"stack", "Stack Research"},
        {"features", "Features Research"},
        {"architecture", "Architecture Research"},
        {"pitfalls", "Pitfalls Research"},
    }
    childIDs := make(map[string]string, 4)
    for _, t := range topics {
        child, err := state.client.CreatePlan(ctx, t.id, args.PhaseNum, epic.ID,
            t.title, "Research complete and results stored in bead",
            "Research topic: "+t.title, []string{"gsd:research-child"}, nil)
        if err != nil {
            return toolError("create child bead " + t.id + ": " + err.Error()), nil
        }
        childIDs[t.id] = child.ID
    }
    return toolResult(&runResearchResult{EpicBeadID: epic.ID, ChildBeadIDs: childIDs})
}
```

### Pattern 3: Plan Beads with Dependency Wiring

**What:** `create_plan_beads` accepts a structured plan (array of task objects with dep references) and creates all task beads in dependency order, resolving local references to actual bead IDs.

**When to use:** SKILL.md generates the plan structure as JSON and passes it to this tool in one call.

**Example (Go):**
```go
// Source: internal/graph/create.go CreatePlan() — dependency wiring via --deps flag
type planTask struct {
    ID         string   `json:"id"`           // local ref like "06-01"
    Title      string   `json:"title"`
    Acceptance string   `json:"acceptance"`
    Context    string   `json:"context"`
    ReqIDs     []string `json:"req_ids"`
    DependsOn  []string `json:"depends_on"`   // local IDs resolved to bead IDs
    Complexity string   `json:"complexity"`   // "S", "M", "L"
    Files      []string `json:"files"`        // estimated file touch list
}

// Create in topological order: tasks with no deps first, then those whose
// deps are already created. Map local ID -> bead ID as we go.
localToBead := make(map[string]string)
for _, task := range ordered {
    depBeadIDs := resolveDepIDs(task.DependsOn, localToBead)
    meta := map[string]any{"complexity": task.Complexity, "files": task.Files, "gsd_plan": task.ID}
    bead, err := state.client.CreatePlan(ctx, task.ID, args.PhaseNum, epicBeadID,
        task.Title, task.Acceptance, task.Context, task.ReqIDs, depBeadIDs)
    localToBead[task.ID] = bead.ID
}
```

### Pattern 4: Plan Validation as Inline SKILL.md Loop

**What:** After `create_plan_beads` returns, SKILL.md instructs Claude to validate inline: check requirement coverage by querying `query_by_label` for each req ID, check wave ordering by examining bead dependencies, iterate up to 3 times.

**When to use:** Plan validation. Avoids spawning a separate validator subagent.

**Example (in SKILL.md):**
```markdown
## Plan validation (inline, up to 3 iterations)

After creating plan beads, validate the plan:

1. For each requirement ID in the phase (e.g., RSRCH-01), call query_by_label with that ID.
   If the result is empty, that requirement is uncovered — a coverage gap.
2. Call list_ready to verify Wave 1 tasks have no unresolved dependencies.
3. If gaps found: identify missing tasks, call create_plan_beads again with fixes.
4. Repeat up to 3 times total. If still failing after 3 attempts, report to the developer.
```

### Anti-Patterns to Avoid

- **Direct bd CLI calls in SKILL.md:** Never embed `bd` commands in SKILL.md. Always use MCP tools. SKILL.md → MCP tool → graph.Client → bd.
- **Polling loop in Go for agent completion:** Don't implement a Go-level polling loop waiting for subagents. SKILL.md Task() calls are synchronous from the orchestrator's perspective — they complete before the next SKILL.md instruction executes.
- **Fat prompts to subagents:** Research subagents receive only: their bead ID, their topic, and instructions to claim+research+close. They do NOT receive the full project context. This is the token-efficiency core of the design.
- **Creating beads one at a time from SKILL.md:** Use `run_research` (1 MCP call) not 5 sequential `create_phase`/`create_plan` calls from SKILL.md. Reduces round-trips.
- **Storing research results only in bead metadata (large JSON):** Metadata is for structured fields (complexity, files, req_ids). Narrative research content goes in the bead's `description` field via `close_plan` reason or a dedicated `update_bead_description` operation.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Parallel agent coordination | Go channels/goroutines in MCP tool | SKILL.md Task() calls | Task() is the Claude Code API for subagents; Go goroutines can't spawn Claude agents |
| Dependency graph topology sort | Custom topological sort in Go | bd's dependency graph + `bd ready` | bd already computes waves; let it do the work |
| Bead ID tracking across plan | Custom map in MCP tool session | localToBead map within single `create_plan_beads` call | Stateless MCP tools; map lives within one handler invocation |
| Research synthesizer logic | LLM reasoning in Go | Claude synthesizer subagent reading beads | Go can't synthesize research; Claude can |
| Wave visualization | Custom ASCII renderer | Reuse `renderReadyTree` from `internal/cli/ready.go` | Already implemented and tested |
| Requirement coverage check | Complex regex scanner | `query_by_label` for each req ID | bd label index handles this in O(1) per label |

**Key insight:** The split is clean — Go handles graph state mutation (MCP tools), Claude handles reasoning and coordination (SKILL.md). Never conflate the two.

## Common Pitfalls

### Pitfall 1: Race condition on bead claim
**What goes wrong:** Two subagents try to claim the same child bead simultaneously.
**Why it happens:** SKILL.md spawns all 4 Task() calls before any complete; bd's `--claim` is atomic but if SKILL.md assigns bead IDs correctly (each agent gets its own ID), this cannot happen.
**How to avoid:** `run_research` returns a map `{"stack": id1, "features": id2, ...}`. SKILL.md passes each subagent its specific bead ID. Each agent only claims its own bead. No ambiguity.
**Warning signs:** `claim_bead` returning an error saying bead already claimed — check that SKILL.md isn't sharing bead IDs across agents.

### Pitfall 2: Subagent receives too much context
**What goes wrong:** Research quality degrades or token budget exceeded when subagent gets full project state.
**Why it happens:** SKILL.md prompt to subagent accidentally includes SESSION context (full graph, all beads).
**How to avoid:** Subagent prompt in SKILL.md is minimal: "You are a [topic] researcher. Bead ID: [id]. Claim it, research [topic], store findings in bead, close it." No project history, no other bead content.
**Warning signs:** Subagent taking >60s or producing generic non-project-specific research.

### Pitfall 3: `create_plan_beads` called with unresolved deps
**What goes wrong:** Dependency bead IDs passed as local strings (e.g., "06-01") instead of actual bd bead IDs (hash strings).
**Why it happens:** SKILL.md plan JSON uses human-readable task IDs in `depends_on`, but CreatePlan expects bd bead IDs.
**How to avoid:** `handleCreatePlanBeads` maintains localToBead map, resolves deps before each CreatePlan call. Process tasks in topological order (tasks with no deps first).
**Warning signs:** `bd create --deps` returning "bead not found" error — the dep ID is a local reference not yet resolved.

### Pitfall 4: Plan checker attempts infinite loop
**What goes wrong:** Plan checker keeps iterating beyond 3 times because SKILL.md instruction lacks a counter.
**Why it happens:** Natural language "repeat up to 3 times" is ambiguous without explicit state tracking.
**How to avoid:** SKILL.md explicitly tracks iteration count with numbered headings or asks Claude to state the iteration number each time. After 3rd failure, SKILL.md instruction explicitly says "stop and report to developer."
**Warning signs:** `/gsd-wired:plan` running for >5 minutes without producing output.

### Pitfall 5: `flush_writes` omitted after batch bead creation
**What goes wrong:** Research epic and child beads exist in bd's batch buffer but not committed to Dolt; next session doesn't see them.
**Why it happens:** `run_research` creates multiple beads in batch mode; `flush_writes` must be called before returning.
**How to avoid:** `handleRunResearch` and `handleCreatePlanBeads` call `state.client.FlushWrites(ctx)` as the last step before returning the result. Same as established pattern in all write operations.
**Warning signs:** `query_by_label "gsd:research"` returning empty after `run_research` succeeds.

### Pitfall 6: SKILL.md placement outside skills/ root
**What goes wrong:** `/gsd-wired:research` and `/gsd-wired:plan` commands not auto-discovered by Claude Code.
**Why it happens:** SKILL.md placed inside `.claude-plugin/` instead of `skills/` root.
**How to avoid:** Follow Phase 5 Pitfall 1 established fix: place at `skills/research/SKILL.md` and `skills/plan/SKILL.md`. Confirmed working pattern from Phase 5.
**Warning signs:** Slash command not appearing in Claude Code command palette.

## Code Examples

Verified patterns from project codebase:

### Creating an epic with child beads (from init_project.go)
```go
// Source: internal/mcp/init_project.go handleInitProject()
// Pattern: epic + fixed set of children, non-fatal child failures
epic, err := state.client.CreatePhase(ctx, phaseNum, title, goal, acceptance, labels)
if err != nil {
    return toolError("failed to create epic: " + err.Error()), nil
}
for _, child := range children {
    bead, err := state.client.CreatePlan(ctx, child.id, phaseNum, epic.ID,
        child.title, child.acceptance, child.context, child.reqIDs, nil)
    if err != nil {
        continue // non-fatal: partial is acceptable
    }
    resultIDs = append(resultIDs, bead.ID)
}
```

### Claiming a bead (from graph/update.go)
```go
// Source: internal/graph/update.go ClaimBead()
// bd update [beadID] --claim — atomic, fails if already claimed
bead, err := c.ClaimBead(ctx, beadID)
// Returns updated bead with assignee set; IsError on failure
```

### Querying beads by label (from graph/query.go)
```go
// Source: internal/graph/query.go QueryByLabel()
// Used by synthesizer to find all research child beads
beads, err := state.client.QueryByLabel(ctx, "gsd:research-child")
// Returns all beads with that label, no limit (--limit 0)
```

### Creating plan beads with dependencies (from graph/create.go)
```go
// Source: internal/graph/create.go CreatePlan()
// dep_bead_ids passed as --deps flag to bd, establishing dependency graph
bead, err := state.client.CreatePlan(ctx, planID, phaseNum, parentID,
    title, acceptance, context, reqIDs, depBeadIDs)
// bd computes waves from this dependency graph via bd ready
```

### Tool registration pattern (from tools.go)
```go
// Source: internal/mcp/tools.go registerTools()
// New tools follow the exact same registration pattern
server.AddTool(&mcpsdk.Tool{
    Name:        "run_research",
    Description: "Creates research epic + 4 child beads for parallel research agents.",
    InputSchema: json.RawMessage(`{"type":"object","properties":{...},...}`),
}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
    var args runResearchArgs
    if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
        return toolError("invalid arguments: " + err.Error()), nil
    }
    return handleRunResearch(ctx, state, args)
})
```

### SKILL.md tool call instruction pattern (from skills/init/SKILL.md)
```markdown
# Established pattern from skills/init/SKILL.md

Once you have collected all answers, call the `init_project` MCP tool with the
collected context as a single JSON object. Map each answer to the corresponding field.

# Research SKILL.md will follow same pattern:
After receiving the phase context, call the `run_research` MCP tool. You will
receive a JSON object with epic_bead_id and child_bead_ids. Then spawn 4 research
agents using the Task tool...
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| GSD research via markdown RESEARCH.md files | Research results stored in bead description + metadata | Phase 6 | No file proliferation; beads queryable |
| Sequential planning with human per-task review | Auto-generate full plan from research context | Phase 6 (D-07) | Same as current GSD — no regression |
| Flat task lists | Wave-based dependency graph (bd ready) | Phase 2 | Already in place |

**Deprecated/outdated:**
- Writing RESEARCH.md to .planning/ during this phase: results go to beads only (D-05 allows both, but bead-only is the correct gsd-wired approach)

## Open Questions

1. **Does bd's `--claim` semantics prevent double-claim across parallel subagents?**
   - What we know: `ClaimBead` runs `bd update [id] --claim` which bd documents as atomic
   - What's unclear: Whether bd uses row-level locking in Dolt or just checks assignee field
   - Recommendation: Trust bd's claim semantics per Phase 2 research (existing `ClaimBead` implementation); design SKILL.md to give each agent a unique bead ID to eliminate the race entirely

2. **Does SKILL.md Task() wait for all parallel tasks before proceeding?**
   - What we know: Claude Code's Task() is a tool call; multiple simultaneous Task() calls are possible
   - What's unclear: Whether the orchestrator must explicitly await all Tasks or if they're guaranteed synchronous
   - Recommendation: Write SKILL.md to explicitly state "wait for all 4 Task calls to complete before proceeding to synthesis" — Claude Code's tool execution semantics ensure this

3. **Is `update_bead_description` a supported bd operation?**
   - What we know: `bd update` supports `--claim`, `--add-label`, `--metadata`; `close_plan` has a `--reason` field
   - What's unclear: Whether bd has a `--description` flag on update
   - Recommendation: Store narrative research content as the `close_plan` reason (long text allowed) + key facts in `--metadata`. Avoids needing a new bd operation. Alternatively, call `query_by_label` to get bead and the description may be stored there if `bd show` returns it — but safest is `close_plan` reason for the summary.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + fake_bd pattern |
| Config file | none — go test ./... discovers tests automatically |
| Quick run command | `go test ./internal/mcp/... -race` |
| Full suite command | `go test ./... -race` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| RSRCH-01 | `run_research` creates epic + 4 child beads | unit | `go test ./internal/mcp/... -run TestRunResearch -race` | ❌ Wave 0 |
| RSRCH-02 | `claim_bead` MCP tool atomically claims a bead | unit | `go test ./internal/mcp/... -run TestClaimBead -race` | ✅ (existing claim_bead tool tested) |
| RSRCH-03 | Research results stored in bead metadata/description | unit | `go test ./internal/mcp/... -run TestRunResearch -race` | ❌ Wave 0 |
| RSRCH-04 | Synthesizer queries label, produces summary bead | unit | `go test ./internal/mcp/... -run TestSynthesizer -race` | ❌ Wave 0 |
| PLAN-01 | `/gsd-wired:plan` slash command file exists and is valid SKILL.md | manual | Inspect `skills/plan/SKILL.md` | ❌ Wave 0 |
| PLAN-02 | `create_plan_beads` creates task beads with deps | unit | `go test ./internal/mcp/... -run TestCreatePlanBeads -race` | ❌ Wave 0 |
| PLAN-03 | Task bead has acceptance, complexity, files in metadata | unit | `go test ./internal/mcp/... -run TestCreatePlanBeads -race` | ❌ Wave 0 |
| PLAN-04 | Plan checker validates requirement coverage | manual | Run `/gsd-wired:plan` and observe validation output | manual-only (SKILL.md logic) |
| CMD-03 | `gsdw plan` CLI subcommand exists | unit | `go test ./internal/cli/... -run TestRootCmdHasPlan -race` | ❌ Wave 0 |

**PLAN-04 is manual-only:** Plan checker logic lives in SKILL.md natural language instructions — no automated test can exercise Claude's plan validation reasoning. Validate manually by running `/gsd-wired:plan` and confirming it catches a missing requirement.

### Sampling Rate
- **Per task commit:** `go test ./internal/mcp/... -race`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/mcp/run_research_test.go` — covers RSRCH-01, RSRCH-03 (TestRunResearch with fake_bd)
- [ ] `internal/mcp/create_plan_beads_test.go` — covers PLAN-02, PLAN-03 (TestCreatePlanBeads with dep resolution)
- [ ] `internal/cli/plan_test.go` — covers CMD-03 (TestRootCmdHasPlan, TestPlanCmdOutput)
- [ ] `skills/research/SKILL.md` — PLAN-01 analog for research command
- [ ] `skills/plan/SKILL.md` — PLAN-01: slash command must exist before testing

## Sources

### Primary (HIGH confidence)
- `internal/mcp/init_project.go` — handleInitProject() is the canonical pattern for epic + children; direct source inspection
- `internal/mcp/tools.go` — registerTools() pattern for new tool registration; direct source inspection
- `internal/graph/create.go` — CreatePlan() with --deps wiring; direct source inspection
- `internal/graph/update.go` — ClaimBead() atomic claim semantics; direct source inspection
- `internal/graph/query.go` — QueryByLabel() for synthesizer use; direct source inspection
- `skills/init/SKILL.md` — Canonical SKILL.md pattern: natural language → MCP tool → result → auto-proceed; direct source inspection
- `skills/status/SKILL.md` — Display-oriented SKILL.md pattern; direct source inspection
- `.planning/phases/05-project-init/05-02-SUMMARY.md` — SKILL.md placement Pitfall 1 confirmed fix

### Secondary (MEDIUM confidence)
- Phase 5 SUMMARY files — established patterns for cli subcommand + SKILL.md co-registration
- CONTEXT.md decisions D-03/D-04/D-10 — SKILL.md orchestration is locked; plan checker is discretionary

### Tertiary (LOW confidence)
- Claude Code Task() tool behavior for parallel subagents — inferred from SKILL.md design patterns and Claude Code plugin documentation; exact synchronization semantics not directly verified in codebase

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries already in use; no new dependencies
- Architecture: HIGH — directly mirrors init_project.go pattern confirmed working in Phase 5
- Pitfalls: HIGH (Pitfalls 1-3, 5-6 from codebase) / MEDIUM (Pitfall 4 from SKILL.md pattern reasoning)
- Task() parallelism: MEDIUM — behavioral inference from Claude Code design; not directly testable in unit tests

**Research date:** 2026-03-21
**Valid until:** 2026-04-21 (stable stack, no external dependencies changing)
