# Phase 7: Execution + Verification - Research

**Researched:** 2026-03-21
**Domain:** Go MCP tool implementation, SKILL.md orchestration, parallel Task() agent patterns, codebase verification
**Confidence:** HIGH

## Summary

Phase 7 is the largest requirement set in the project (12 requirements: EXEC-01 through EXEC-06, VRFY-01 through VRFY-03, CMD-04, CMD-05, CMD-07). The good news: the entire implementation follows patterns that already exist and are proven in Phases 5 and 6. Wave execution is the research orchestration pattern applied to task beads instead of research topics. Verification is the plan checker pattern applied to codebase state. Remediation uses the existing `create_plan_beads` tool unchanged.

The key insight is that Phase 7 adds exactly **two new MCP tools** (`verify_phase` and `execute_wave`) plus **three new SKILL.md files** (`execute`, `verify`, `ready`) and **two new CLI stubs** (`execute`, `verify`). The `ready` command already exists as `gsdw ready` — CMD-07 is a SKILL.md wrapper over the existing CLI command.

**Primary recommendation:** Model `verify_phase` and `execute_wave` exactly after the `run_research` / `synthesize_research` pair. Model the three SKILL.md files exactly after `skills/research/SKILL.md`. Execution agents get bead ID + context chain + 4 instructions — nothing more.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** gsdw is the developer interface. Execution orchestration, wave management, and verification happen behind the scenes. Developer says "execute" and work gets done.
- **D-02:** 30-second auto-proceed pattern continues. After verification passes, auto-proceed to next phase.
- **D-03:** SKILL.md orchestrates execution. Calls `list_ready` for current wave, spawns parallel Task() agents per ready task, each agent claims its bead, works, closes. `list_ready` again for next wave. Repeat until empty. Same pattern as research orchestration.
- **D-04:** Each execution agent receives only its claimed bead's context chain: task description, success criteria, parent epic summary, dependency bead summaries. Minimal prompt per EXEC-02/EXEC-03.
- **D-05:** On task completion, agent closes bead with results. `list_ready` surfaces newly unblocked tasks, triggering next wave automatically per EXEC-04.
- **D-06:** Atomic git commits per completed task. Commit message uses GSD-friendly plan ID (e.g., "feat(07-01): description"), not bd bead ID. Developer never sees bd IDs.
- **D-07:** SKILL.md validates results inline between waves — checks must-haves from task bead's acceptance criteria before proceeding. Best-effort validation, same pattern as plan checker. No separate validation agent.
- **D-08:** Validation errors are surfaced to developer for decision (retry, skip, abort). Same escalation pattern as plan checker.
- **D-09:** New `verify_phase` MCP tool reads phase epic's success criteria, checks against codebase state, returns structured pass/fail per criterion.
- **D-10:** Failed verification criteria automatically produce new remediation task beads via existing `create_plan_beads` tool (VRFY-03).
- **D-11:** `/gsd-wired:verify` SKILL.md presents verification results in GSD-familiar format (pass/fail table, not bead data).
- **D-12:** `/gsd-wired:ready` SKILL.md shows unblocked tasks (CMD-07). Reuses existing `gsdw ready` CLI command + `list_ready` MCP tool.

### Claude's Discretion

- Exact subagent prompt structure for execution agents
- How to extract task context chain (parent epic + dependency summaries) efficiently
- Agent output validation depth (acceptance criteria check vs full code review)
- Verification implementation (code checks, test execution, file existence)
- Remediation task granularity (one per failed criterion vs grouped)

### Deferred Ideas (OUT OF SCOPE)

- PR creation from execution results (Phase 8)
- Token-aware context loading for execution agents (Phase 9)
- Cross-phase regression testing (future — not in v1 requirements)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| EXEC-01 | Wave execution runs unblocked tasks in parallel (tasks from `bd ready`) | `list_ready` + parallel Task() already proven in research phase |
| EXEC-02 | Each execution subagent claims a task bead and receives only that bead's context chain | `claim_bead` exists; context extraction pattern defined below |
| EXEC-03 | Subagent context: task description, success criteria, parent epic summary, dep bead summaries | `get_bead` + `query_by_label` for parent; deps available in bead.Deps |
| EXEC-04 | On task completion, subagent closes bead with results, triggering next wave | `close_plan` returns `unblocked` list; `list_ready` for next wave |
| EXEC-05 | Atomic git commits per completed task with bead ID in commit message | SKILL.md instructs agent to run `git commit -m "feat({plan_id}): ..."` |
| EXEC-06 | Agent output validated at orchestrator before downstream consumption | SKILL.md inline check against acceptance criteria before proceeding |
| VRFY-01 | Verification agent reads success criteria from phase epic's extensible fields | `get_bead` on phase epic; success criteria in bead.Acceptance |
| VRFY-02 | Verification runs checks against codebase and reports pass/fail per criterion | `verify_phase` MCP tool: os.Stat checks, `go test` subprocess, file content grep |
| VRFY-03 | Failed criteria produce new task beads for remediation | `create_plan_beads` (existing tool 13) called from SKILL.md on failure |
| CMD-04 | `/gsd-wired:execute` slash command | `skills/execute/SKILL.md` + `gsdw execute` CLI stub |
| CMD-05 | `/gsd-wired:verify` slash command | `skills/verify/SKILL.md` + `gsdw verify` CLI stub |
| CMD-07 | `/gsd-wired:ready` slash command | `skills/ready/SKILL.md` wrapper over existing `gsdw ready` CLI |
</phase_requirements>

## Standard Stack

### Core (all already in go.mod — no new dependencies)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/modelcontextprotocol/go-sdk/mcp` | v1.4.1 | MCP tool registration and handler | Same as all 13 existing tools |
| `os/exec` stdlib | Go 1.26.1 | Run `go test ./...` in verify_phase | No dep needed; existing bd client uses same pattern |
| `os` stdlib | Go 1.26.1 | File existence checks in verify_phase | Used throughout codebase |
| `strings` stdlib | Go 1.26.1 | Criterion parsing | Used throughout |

**No new dependencies.** Phase 7 adds Go files following the existing `internal/mcp/*.go` pattern.

### Established Patterns (from codebase — HIGH confidence)

| Pattern | File | What it establishes |
|---------|------|---------------------|
| Tool handler signature | `internal/mcp/run_research.go` | `handleFoo(ctx, state, args) (*mcpsdk.CallToolResult, error)` |
| Tool registration | `internal/mcp/tools.go` | `server.AddTool(...)` with JSON schema inline |
| toolError / toolResult | `internal/mcp/tools.go` | Always use these helpers, never construct Content slices directly |
| CLI stub | `internal/cli/plan.go` | `RunE` returns error redirecting to slash command |
| SKILL.md minimal prompts | `skills/research/SKILL.md` | bead ID + topic + 4 instructions — nothing else |
| 30s auto-proceed | `skills/plan/SKILL.md` | Display action, wait 30s, auto-proceed |
| Inline validation loop | `skills/plan/SKILL.md` | Up to 3 iterations, track count explicitly |

## Architecture Patterns

### New Files Required

```
internal/mcp/
  execute_wave.go          # handleExecuteWave: returns context chain for agents
  execute_wave_test.go
  verify_phase.go          # handleVerifyPhase: codebase checks, structured pass/fail
  verify_phase_test.go

skills/
  execute/SKILL.md         # /gsd-wired:execute orchestration
  verify/SKILL.md          # /gsd-wired:verify results display
  ready/SKILL.md           # /gsd-wired:ready (thin wrapper)

internal/cli/
  execute.go               # gsdw execute stub
  execute_test.go
  verify.go                # gsdw verify stub
  verify_test.go
```

**Modify:**
- `internal/mcp/tools.go` — register 2 new tools (total: 15)
- `internal/mcp/server.go` — update debug log count to 15
- `internal/mcp/tools_test.go` — update expected tool count to 15
- `internal/mcp/server_test.go` — update expected tool count to 15
- `internal/cli/root.go` — add NewExecuteCmd and NewVerifyCmd to AddCommand chain

### Pattern 1: execute_wave MCP Tool

**What it does:** Takes a phase number, fetches the phase epic bead, calls `list_ready` scoped to that phase, and for each ready task returns the full context chain (task bead + parent epic summary + dependency summaries). The SKILL.md then uses this context chain to construct minimal execution agent prompts.

**Why a tool instead of SKILL.md doing it:** The context chain requires multiple graph queries (get_bead for each dep). Centralizing in an MCP tool keeps SKILL.md simple (one tool call → ready context chain) and makes the logic testable with fake_bd.

```go
// Source: internal/mcp/execute_wave.go (to be created)
type executeWaveArgs struct {
    PhaseNum int `json:"phase_num"`
}

type taskContext struct {
    BeadID       string            `json:"bead_id"`
    PlanID       string            `json:"plan_id"`
    Title        string            `json:"title"`
    Acceptance   string            `json:"acceptance"`
    Context      string            `json:"context"`
    ParentSummary string           `json:"parent_summary"`    // phase epic acceptance field
    DepSummaries  []string         `json:"dep_summaries"`     // closed dep bead reasons
}

type executeWaveResult struct {
    Wave  int           `json:"wave"`
    Tasks []taskContext `json:"tasks"`
}

func handleExecuteWave(ctx context.Context, state *serverState, args executeWaveArgs) (*mcpsdk.CallToolResult, error) {
    if err := state.init(ctx); err != nil {
        return toolError(err.Error()), nil
    }
    // 1. Find phase epic bead via query_by_label("gsd:phase") + metadata match
    // 2. call ReadyForPhase(ctx, phaseBeadID)
    // 3. For each ready task: get_bead for parent (epic summary), get_bead for each dep (close reason)
    // 4. Return []taskContext — one entry per ready task
}
```

**Test pattern:** Use fake_bd with canned responses for ready, show, and query commands.

### Pattern 2: verify_phase MCP Tool

**What it does:** Reads the phase epic's `Acceptance` field, parses it into individual criteria (newline or semicolon-separated), runs checks, returns structured pass/fail.

**Verification strategy for each criterion type:**
- Criterion contains a file path → `os.Stat(path)` existence check
- Criterion mentions `go test` → run `exec.CommandContext(ctx, "go", "test", "./...")` in project root
- Criterion mentions a function/type name → `grep -r pattern .` equivalent using `os.ReadDir` + file scan
- Default: flag as `manual` (cannot be automated)

```go
// Source: internal/mcp/verify_phase.go (to be created)
type verifyPhaseArgs struct {
    PhaseNum   int    `json:"phase_num"`
    ProjectDir string `json:"project_dir"` // path to check files against
}

type criterionResult struct {
    Criterion string `json:"criterion"`
    Passed    bool   `json:"passed"`
    Method    string `json:"method"`   // "file_exists", "go_test", "grep", "manual"
    Detail    string `json:"detail"`   // what was checked / failure reason
}

type verifyPhaseResult struct {
    PhaseNum int               `json:"phase_num"`
    Passed   bool              `json:"passed"`    // true only if ALL criteria pass
    Results  []criterionResult `json:"results"`
    Failed   []string          `json:"failed"`    // criteria text for failed items (used by SKILL.md for remediation)
}
```

**Key implementation decision:** `verify_phase` does not call `create_plan_beads` internally. The SKILL.md reads `failed` from the result and calls `create_plan_beads` itself (per D-10). This keeps the tool single-responsibility and testable.

### Pattern 3: SKILL.md Execution Orchestration

The execution SKILL.md mirrors `skills/research/SKILL.md` exactly, with task beads replacing research child beads.

```yaml
# Source: skills/execute/SKILL.md (to be created)
---
name: execute
description: Execute the current wave of unblocked tasks in parallel
disable-model-invocation: true
argument-hint: "[phase_number]"
---
```

**Execution agent prompt template** (minimal, per D-04 and established Pitfall 2):

```
You are an execution agent for task {plan_id}: {title}.
Your bead ID is {bead_id}.

Context:
- Task: {context}
- Acceptance: {acceptance}
- Phase goal: {parent_summary}
- Dependencies completed: {dep_summaries}

Instructions:
1. Call claim_bead with id={bead_id}
2. Implement the task: {title}
3. Run: git add -p && git commit -m "feat({plan_id}): {title}"
4. Call close_plan with id={bead_id} and reason={one-line summary of what was done}
```

**Wave loop in SKILL.md:**

```
Step 1: Determine phase number from $ARGUMENTS or project context
Step 2: Call execute_wave with phase_num to get ready tasks + context chains
Step 3: If tasks empty → display "All tasks complete" → auto-proceed to /gsd-wired:verify
Step 4: Display current wave (GSD wave table format)
Step 5: Spawn parallel Task() agents — one per ready task — using context chain from execute_wave
Step 6: Wait for all agents to complete
Step 7: Inline validation — for each completed task, check acceptance criteria against file existence and test output
Step 8: If validation fails → surface to developer (retry / skip / abort per D-08)
Step 9: Go to Step 2 (next wave)
```

### Pattern 4: Minimal Context Chain Extraction

**Problem:** EXEC-02/EXEC-03 require execution agents to receive minimal context — task + parent epic + dep summaries only.

**Solution:** `execute_wave` does ALL the graph queries before spawning agents. Agents receive pre-computed context, not instructions to query the graph themselves. This is the same principle as research: agents get bead ID + prepared context, not a graph client.

**Context chain construction:**
1. Phase epic bead → `Acceptance` field = phase goal (1-2 sentences)
2. Each dependency bead → closed bead's `Reason` field (the one-line summary written when agent called `close_plan`)
3. Task bead → `Title` + `Acceptance` + `Context` (already stored in the bead)

**Key insight:** The `closeResult.Unblocked` response from `close_plan` already surfaces newly unblocked bead IDs. The SKILL.md can use this to know which tasks become available without calling `list_ready` again — though calling `execute_wave` again is cleaner.

### Pattern 5: Verification SKILL.md

```
Step 1: Call verify_phase with phase_num and project_dir="."
Step 2: Display results as pass/fail table (GSD format):
  | Criterion | Status | Method |
  |-----------|--------|--------|
  | All tests pass | PASS | go test |
  | skills/execute/SKILL.md exists | PASS | file check |
  | Failed criterion X | FAIL | manual |
Step 3: If all pass → "Phase {N} verified. (auto-continuing in 30 seconds...)"
Step 4: If failures → call create_plan_beads with remediation tasks (one per failed criterion)
Step 5: Display remediation plan → "Re-running /gsd-wired:execute to close gaps..."
Step 6: Auto-proceed after 30 seconds
```

### Pattern 6: git Commit Automation from Agents (EXEC-05)

**Decision:** Execution agents run `git add` and `git commit` directly using Bash tool. The SKILL.md prompt instructs agents to use the plan ID format `feat({plan_id}): {title}`. The MCP server does NOT shell out to git — git is user-space, not graph infrastructure.

**Commit message format:** `feat({plan_id}): {title}` — e.g., `feat(07-01): execute_wave MCP tool with TDD`

**Why plan ID not bead ID (D-06):** Bead IDs are hashes (`abc123ef`). Plan IDs are human-readable (`07-01`). Developer reads git log and sees phase + sequence, not opaque hashes.

**Agent instruction:**
```
After implementation is complete and tests pass:
  git add {files}
  git commit -m "feat({plan_id}): {title}"
```

### Anti-Patterns to Avoid

- **Fat agent prompts:** Do not include the full phase history, all bead IDs, or research summaries in execution agent prompts. Pass only the 4-item context chain.
- **Agent doing graph queries:** execute_wave pre-computes the context chain. Agents call `claim_bead`, implement, commit, `close_plan` — no other MCP tool calls.
- **verify_phase calling create_plan_beads:** Single-responsibility. verify_phase checks codebase. SKILL.md creates remediation tasks.
- **Tool count drift:** Server.go and both test files must be updated atomically with new tool registrations. Failure causes TestToolsRegistered/TestToolsListed to fail.
- **Skipping fake_bd update:** If execute_wave or verify_phase need `bd ready --parent` or `bd show`, fake_bd must be extended to return canned responses.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Dependency-ordered task creation for remediation | Custom sort in verify_phase | `create_plan_beads` (existing tool 13) | Already handles topological sort, circular dep detection |
| Wave boundary detection | Custom "is this task unblocked?" logic | `list_ready` / `ReadyForPhase` | bd computes this from the dependency graph |
| Parallel agent spawning | goroutines or channels in Go | SKILL.md `Task()` parallel calls | Claude Code's native parallel Task() is the established pattern |
| Git commit construction | MCP tool or hook | SKILL.md instructs agent to use Bash | Git is user-space; tool would add coupling without benefit |
| Criterion text parsing | Regex-heavy parser | Simple string prefix matching | Criteria are short human-readable strings; over-engineering kills velocity |

**Key insight:** Every hard problem in Phase 7 already has a solved implementation in the codebase. The work is wiring them together, not building new infrastructure.

## Common Pitfalls

### Pitfall 1: Tool Count Drift
**What goes wrong:** New MCP tools are registered in `registerTools()` but `server.go` debug log, `tools_test.go` TestToolsRegistered, and `server_test.go` TestToolsListed still expect 13 tools. Tests fail.
**Why it happens:** Four files must be updated together; easy to miss one.
**How to avoid:** Update all four in a single commit. Pattern: grep for "13" in mcp/ files before committing.
**Warning signs:** `TestToolsRegistered` or `TestToolsListed` fails with "expected 13, got 15".

### Pitfall 2: Fat Execution Agent Prompts
**What goes wrong:** Execution agent prompt includes full project history, all task beads, or research summaries. Agent uses 30k tokens on context, leaves little for implementation.
**Why it happens:** Well-intentioned "give agents full context" instinct.
**How to avoid:** The 4-item context chain from execute_wave is all agents get: task + acceptance + parent_summary + dep_summaries. See established pattern in `skills/research/SKILL.md`.
**Warning signs:** SKILL.md prompt template exceeds ~20 lines.

### Pitfall 3: Agents Calling graph Tools Directly
**What goes wrong:** Execution agent prompt says "call query_by_label to find your parent epic". Now agents consume MCP quota doing graph operations instead of implementation work.
**Why it happens:** Trying to make agents self-sufficient.
**How to avoid:** execute_wave pre-fetches everything. Agents only call: claim_bead, close_plan, git commit. No query/get calls needed.
**Warning signs:** SKILL.md execution agent template mentions `query_by_label` or `get_bead`.

### Pitfall 4: verify_phase Returning Vague Criteria
**What goes wrong:** `verify_phase` returns `{"criterion": "All requirements met", "passed": false}` — SKILL.md cannot create a useful remediation task from this.
**Why it happens:** Phase epic's Acceptance field is written for humans, not machines.
**How to avoid:** `verify_phase` returns the raw criterion text AND the detail of what was checked. SKILL.md constructs remediation task titles from detail, not criterion text.
**Warning signs:** Remediation tasks have titles like "Fix: All requirements met".

### Pitfall 5: `go test` in verify_phase Blocking
**What goes wrong:** `verify_phase` runs `go test ./...` synchronously with no timeout. A hanging test blocks the MCP tool indefinitely.
**Why it happens:** `exec.Command` without context timeout.
**How to avoid:** Use `exec.CommandContext(ctx, ...)` — the MCP handler context will be cancelled if the client disconnects. Add an explicit 60-second timeout via `context.WithTimeout`.
**Warning signs:** `/gsd-wired:verify` hangs with no output.

### Pitfall 6: Inline Validation Too Deep
**What goes wrong:** SKILL.md validation between waves attempts full code review of agent output — reads every file touched, checks style, runs linters. Adds 5+ minutes per wave.
**Why it happens:** Confusing "validation" (does it meet acceptance criteria?) with "review" (is it good code?).
**How to avoid:** Inline validation is acceptance criteria only: does the file exist? do tests pass? Per D-07: "best-effort validation". Surface failures to developer; don't block.
**Warning signs:** Validation step consumes more tokens than the implementation step.

### Pitfall 7: fake_bd Not Supporting `bd ready --parent`
**What goes wrong:** execute_wave calls `ReadyForPhase(ctx, phaseBeadID)` which runs `bd ready --parent {id}`. fake_bd doesn't handle `--parent` flag → all tests using execute_wave fail.
**Why it happens:** fake_bd only handles the `--parent` flag if explicitly coded.
**How to avoid:** Extend fake_bd in the same commit that adds execute_wave. Check the pattern: fake_bd already handles `--limit 0`.

## Code Examples

### execute_wave Tool Registration
```go
// Source: internal/mcp/tools.go (to be modified)
server.AddTool(&mcpsdk.Tool{
    Name:        "execute_wave",
    Description: "Returns context chain for all ready tasks in a phase wave. " +
                 "Each task includes: bead_id, plan_id, title, acceptance, context, " +
                 "parent_summary, dep_summaries. Use returned context to spawn execution agents.",
    InputSchema: json.RawMessage(`{"type":"object","properties":{"phase_num":{"type":"integer","description":"Phase number to execute"}},"required":["phase_num"],"additionalProperties":false}`),
}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
    var args executeWaveArgs
    if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
        return toolError("invalid arguments: " + err.Error()), nil
    }
    return handleExecuteWave(ctx, state, args)
})
```

### verify_phase Tool Registration
```go
// Source: internal/mcp/tools.go (to be modified)
server.AddTool(&mcpsdk.Tool{
    Name:        "verify_phase",
    Description: "Checks phase success criteria against codebase state. " +
                 "Returns pass/fail per criterion with method and detail. " +
                 "Failed criteria list used by SKILL.md to create remediation tasks.",
    InputSchema: json.RawMessage(`{"type":"object","properties":{"phase_num":{"type":"integer","description":"Phase number to verify"},"project_dir":{"type":"string","description":"Absolute path to project root for file checks (default: current working directory)"}},"required":["phase_num"],"additionalProperties":false}`),
}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
    var args verifyPhaseArgs
    if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
        return toolError("invalid arguments: " + err.Error()), nil
    }
    return handleVerifyPhase(ctx, state, args)
})
```

### Context Chain Population (execute_wave handler skeleton)
```go
// Source: internal/mcp/execute_wave.go (to be created)
func handleExecuteWave(ctx context.Context, state *serverState, args executeWaveArgs) (*mcpsdk.CallToolResult, error) {
    if err := state.init(ctx); err != nil {
        return toolError(err.Error()), nil
    }

    // Find phase epic.
    epics, err := state.client.QueryByLabel(ctx, "gsd:phase")
    if err != nil {
        return toolError("failed to query phase epic: " + err.Error()), nil
    }
    var phaseBeadID, phaseSummary string
    for _, b := range epics {
        if phaseNumFromMeta(b.Metadata) == args.PhaseNum {
            phaseBeadID = b.ID
            phaseSummary = b.Acceptance // phase goal as one-liner
            break
        }
    }
    if phaseBeadID == "" {
        return toolError(fmt.Sprintf("no phase epic found for phase %d", args.PhaseNum)), nil
    }

    // Get ready tasks scoped to this phase.
    ready, err := state.client.ReadyForPhase(ctx, phaseBeadID)
    if err != nil {
        return toolError("failed to list ready tasks: " + err.Error()), nil
    }

    // Build context chain per task.
    var tasks []taskContext
    for _, t := range ready {
        tc := taskContext{
            BeadID:        t.ID,
            PlanID:        planIDFromMeta(t.Metadata),
            Title:         t.Title,
            Acceptance:    t.Acceptance,
            Context:       t.Context,
            ParentSummary: phaseSummary,
        }
        // Fetch dep summaries (closed beads have Reason set).
        for _, depID := range t.Deps {
            dep, depErr := state.client.GetBead(ctx, depID)
            if depErr == nil && dep.Reason != "" {
                tc.DepSummaries = append(tc.DepSummaries, dep.Reason)
            }
        }
        tasks = append(tasks, tc)
    }

    return toolResult(&executeWaveResult{Wave: waveNum(args.PhaseNum), Tasks: tasks})
}
```

### verify_phase Codebase Check Pattern
```go
// Source: internal/mcp/verify_phase.go (to be created)
func checkCriterion(ctx context.Context, criterion, projectDir string) criterionResult {
    c := criterionResult{Criterion: criterion}

    // File existence check: criterion contains a known file extension or path pattern.
    if path := extractFilePath(criterion); path != "" {
        abs := filepath.Join(projectDir, path)
        if _, err := os.Stat(abs); err == nil {
            c.Passed = true
            c.Method = "file_exists"
            c.Detail = abs + " exists"
        } else {
            c.Method = "file_exists"
            c.Detail = abs + " not found"
        }
        return c
    }

    // Test execution: criterion mentions "test" or "go test".
    if strings.Contains(strings.ToLower(criterion), "test") {
        tCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
        defer cancel()
        cmd := exec.CommandContext(tCtx, "go", "test", "./...")
        cmd.Dir = projectDir
        out, err := cmd.CombinedOutput()
        c.Method = "go_test"
        if err == nil {
            c.Passed = true
            c.Detail = "go test ./... passed"
        } else {
            c.Detail = "go test ./... failed: " + string(out)
        }
        return c
    }

    // Default: manual verification required.
    c.Method = "manual"
    c.Detail = "cannot automate — mark as manual review"
    c.Passed = false // conservative: assume fail until human confirms
    return c
}
```

### Bead Struct Fields (from existing codebase)
```go
// Source: internal/graph/ (existing — check Bead struct for exact fields)
// Key fields used in Phase 7:
// b.ID         — bead hash ID
// b.Title      — task title
// b.Acceptance — acceptance criteria (= success criteria for phase epics)
// b.Context    — task description
// b.Metadata   — map[string]any with gsd_phase, gsd_plan, complexity, files
// b.Labels     — []string with gsd:plan, EXEC-01, etc.
// b.Deps       — []string of dependency bead IDs (check actual field name in Bead struct)
// b.Reason     — close reason (set when close_plan is called)
```

### CLI Stub Pattern (execute and verify — identical to plan stub)
```go
// Source: internal/cli/execute.go (to be created, mirrors internal/cli/plan.go)
func NewExecuteCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "execute",
        Short: "Execute the current wave of unblocked tasks",
        RunE: func(cmd *cobra.Command, args []string) error {
            return fmt.Errorf("use /gsd-wired:execute slash command — wave execution requires Claude Code's Task() tool for parallel agent spawning")
        },
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Per-agent full context load | Minimal context chain (4 items) | Phase 6 design | Drastically reduces agent token cost |
| Verify by reading files manually | verify_phase tool with structured pass/fail | Phase 7 (new) | Machine-readable results enable automatic remediation |
| Single-threaded task execution | Wave-based parallel Task() | Phase 6 pattern | Faster execution, same as research orchestration |

## Open Questions

1. **Bead.Deps field exact name**
   - What we know: `bd show {id} --json` includes dependency IDs; `CreatePlan` uses `--deps` flag
   - What's unclear: The Go struct field is `Deps []string` or `Dependencies []string` or similar — need to check `internal/graph/bead.go`
   - Recommendation: Read `internal/graph/bead.go` in the planning phase before writing execute_wave; use whatever field name exists

2. **Bead.Reason field for closed beads**
   - What we know: `ClosePlan` passes `--reason` to bd, and the closed bead has this populated
   - What's unclear: Whether `bd show` on a closed bead returns the reason field in JSON output
   - Recommendation: Verify with `bd show {closed-bead-id} --json` during implementation; if absent, store reason in metadata instead

3. **phase_epic.Acceptance vs phase_epic.Context for parent summary**
   - What we know: `CreatePhase` stores the goal in `--context` and acceptance criteria in `--acceptance`
   - What's unclear: Which field better serves as the "parent_summary" for execution agents
   - Recommendation: Use `Context` (phase goal) as parent_summary — it's a description, not success criteria

4. **Criterion parsing strategy for verify_phase**
   - What we know: Phase epic Acceptance field is a free-text string written by a human/AI during planning
   - What's unclear: Whether criteria are newline-separated, numbered, or bullet-pointed
   - Recommendation: Try newline split first; strip leading `N. ` or `- ` prefixes; treat each non-empty line as one criterion

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go testing + race detector |
| Config file | none — `go test` is convention |
| Quick run command | `go test ./internal/mcp/... -run TestExecuteWave -race` |
| Full suite command | `go test ./... -race` |

### Phase Requirements to Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| EXEC-01 | execute_wave returns all ready tasks | unit | `go test ./internal/mcp/... -run TestExecuteWave -race` | Wave 0 |
| EXEC-02 | Each task has bead_id for claim_bead | unit | `go test ./internal/mcp/... -run TestExecuteWave -race` | Wave 0 |
| EXEC-03 | Context chain: task+parent+deps present | unit | `go test ./internal/mcp/... -run TestExecuteWaveContextChain -race` | Wave 0 |
| EXEC-04 | close_plan returns unblocked list | existing | `go test ./internal/mcp/... -run TestClosePlan -race` | exists |
| EXEC-05 | Commit message format in SKILL.md | manual | `/gsd-wired:execute` + inspect git log | N/A |
| EXEC-06 | Inline validation in SKILL.md | manual | `/gsd-wired:execute` + watch orchestrator | N/A |
| VRFY-01 | verify_phase reads phase epic Acceptance | unit | `go test ./internal/mcp/... -run TestVerifyPhase -race` | Wave 0 |
| VRFY-02 | File check and go_test check methods | unit | `go test ./internal/mcp/... -run TestVerifyPhaseFileCheck -race` | Wave 0 |
| VRFY-03 | Failed criteria list returned in result | unit | `go test ./internal/mcp/... -run TestVerifyPhaseFailures -race` | Wave 0 |
| CMD-04 | execute CLI stub returns correct error | unit | `go test ./internal/cli/... -run TestExecuteCmd -race` | Wave 0 |
| CMD-05 | verify CLI stub returns correct error | unit | `go test ./internal/cli/... -run TestVerifyCmd -race` | Wave 0 |
| CMD-07 | ready SKILL.md wraps gsdw ready output | manual | `gsdw ready` CLI (already tested) | exists |

### Sampling Rate

- **Per task commit:** `go test ./... -race`
- **Per wave merge:** `go test ./... -race` (same — all tests are fast, < 5s total)
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

- [ ] `internal/mcp/execute_wave_test.go` — TestExecuteWave, TestExecuteWaveContextChain, TestExecuteWaveEmpty
- [ ] `internal/mcp/verify_phase_test.go` — TestVerifyPhase, TestVerifyPhaseFileCheck, TestVerifyPhaseFailures, TestVerifyPhaseGoTest
- [ ] `internal/cli/execute_test.go` — TestRootCmdHasExecute, TestExecuteCmdOutput
- [ ] `internal/cli/verify_test.go` — TestRootCmdHasVerify, TestVerifyCmdOutput
- [ ] fake_bd extension: `bd ready --parent {id}` support (needed by execute_wave)

## Sources

### Primary (HIGH confidence)

- `internal/mcp/run_research.go` — handleRunResearch pattern replicated for handleExecuteWave
- `internal/mcp/create_plan_beads.go` — remediation task creation via existing tool (VRFY-03)
- `internal/mcp/tools.go` — tool registration pattern; all 13 tool examples
- `internal/graph/query.go` — ListReady, ReadyForPhase, QueryByLabel APIs
- `internal/graph/update.go` — ClaimBead, ClosePlan, UpdateBeadMetadata APIs
- `internal/graph/create.go` — CreatePhase, CreatePlanWithMeta metadata field names
- `skills/research/SKILL.md` — minimal agent prompt template (4 instructions pattern)
- `skills/plan/SKILL.md` — inline validation loop pattern, 30s auto-proceed pattern
- `internal/cli/ready.go` — phaseNumFromBead, planIDFromBead helpers (reuse in execute_wave)
- `.planning/phases/07-execution-verification/07-CONTEXT.md` — 12 locked decisions

### Secondary (MEDIUM confidence)

- `.planning/ROADMAP.md` Phase 7 section — 7 success criteria (verify_phase must check these patterns)
- `.planning/STATE.md` accumulated context — confirms patterns and decisions from Phases 1-6

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new dependencies, all patterns from existing code
- Architecture: HIGH — derived directly from working code in Phases 5 and 6
- Pitfalls: HIGH — pitfalls 1-4 and 7 are from accumulated STATE.md decisions; pitfalls 5-6 are from general Go/LLM agent knowledge
- Open questions: MEDIUM — Bead struct field names require a quick read of bead.go to confirm

**Research date:** 2026-03-21
**Valid until:** 2026-04-21 (stable Go patterns; beads API stable for this project)
