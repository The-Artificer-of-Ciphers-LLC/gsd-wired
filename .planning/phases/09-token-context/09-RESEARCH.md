# Phase 9: Token-Aware Context - Research

**Researched:** 2026-03-21
**Domain:** Go token estimation, context tiering, MCP tool authoring, hook budget injection
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** gsdw is the developer interface. Token optimization is invisible — developer gets the right context without knowing about budgets or tiers.
- **D-02:** This is the core innovation: graph queries replace full file reads, context is O(relevant) not O(total).
- **D-03:** Active beads (open, in_progress) = hot → full context (description, acceptance, notes, metadata).
- **D-04:** Recently closed beads = warm → summary only (title + close reason, ~50 tokens).
- **D-05:** Old closed beads = cold → ID + title only (~10 tokens).
- **D-06:** "Recently closed" threshold at Claude's discretion (e.g., last 24 hours, last N beads).
- **D-07:** Simple byte-count heuristic: 1 token ≈ 4 bytes. No external tokenizer dependency.
- **D-08:** Budget-aware context fitting: measure total tiered content, trim warm→cold or omit cold beads to fit within budget.
- **D-09:** SessionStart's `additionalContext` becomes budget-aware. Check available budget, inject tiered content that fits.
- **D-10:** New `get_tiered_context` MCP tool returns context at requested tier level with budget constraint. Used by hooks and skills.
- **D-11:** `execute_wave` context chains use compacted summaries for closed dependency beads (warm tier) instead of full content.
- **D-12:** Closed beads are automatically compacted (summary replaces full content in query results). This is a graph-layer optimization, not a tool-layer one.

### Claude's Discretion
- Exact tiering thresholds (hot/warm/cold boundaries)
- Budget estimation accuracy (byte heuristic vs more sophisticated)
- How to handle budget overflow (progressive degradation strategy)
- Compaction trigger (on close, on query, on timer)
- Whether to modify existing tools or create new ones

### Deferred Ideas (OUT OF SCOPE)
- PreToolUse file-aware context injection (TOKEN-A01, v2)
- Automatic dependency detection suggestions (TOKEN-A02, v2)
- TUI visualization of token usage (PLAT-01, v2)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| TOKEN-01 | Graph queries replace full markdown file reads (O(relevant) not O(total)) | Tiered query layer on graph.Client; `get_tiered_context` tool returns only what callers need |
| TOKEN-02 | Subagent prompts contain only claimed bead context, not full project state | `execute_wave` compaction: dep beads use warm-tier CloseReason only, not full Description |
| TOKEN-03 | Closed beads automatically compacted (summary replaces full content) | Compaction via `UpdateBeadMetadata` on close, storing `gsd:compact` label + summary field; query layer returns compact form |
| TOKEN-04 | Token-aware context routing: hot/warm/cold tiers | `TieredBead` response type; `QueryTiered` graph method classifies on ClosedAt timestamp |
| TOKEN-05 | Context budget tracking estimates tokens per bead and fits within remaining window | `estimateTokens(s string) int` via `len(s)/4`; budget loop trims warm→cold→omit |
| TOKEN-06 | Tiered context injection in SessionStart based on available token budget | `buildSessionContext` refactored to accept budget int, returns tiered content that fits |
</phase_requirements>

## Summary

Phase 9 adds a token-awareness layer across three surfaces: the `graph.Client` query layer, the `internal/hook/session_start.go` injection path, and a new `get_tiered_context` MCP tool. The core innovation (D-02) is already partially in place — `execute_wave` pre-computes context chains using only `CloseReason` for dependency beads. Phase 9 formalizes this into a system-wide tiering contract.

The byte-count heuristic (1 token ≈ 4 bytes, D-07) is both sufficient and correct for this use case. The goal is budget guidance, not tokenizer precision. Claude's actual tokenizer is BPE-based and varies by content type, but 4 bytes/token is a reliable conservative estimate for English prose + code. No external dependency is needed or wanted.

The critical insight for implementation: compaction (D-12) is a **write-time operation** triggered when a bead closes. It stores a compact summary in `bead.Metadata["gsd:compact"]`. The query layer (`QueryTiered`) reads this field to decide which tier to return. This avoids re-computing summaries at query time and keeps graph.Client stateless between calls.

**Primary recommendation:** Implement in three tasks: (1) graph-layer tiering with compaction on close, (2) budget-aware `buildSessionContext` refactor + new `get_tiered_context` MCP tool, (3) `execute_wave` warm-tier dep summaries using compacted field.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib only | Go 1.26.1 | All implementation | Project constraint — no external deps (confirmed by existing go.mod pattern) |
| `encoding/json` | stdlib | Bead serialization/deserialization | Already the project standard |
| `time` | stdlib | ClosedAt timestamp for tiering thresholds | Already on Bead struct |
| `strings` | stdlib | String building for context output | Already used in formatSessionContext |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `log/slog` | stdlib | Warn on budget overflow or compaction errors | Same pattern as existing hooks — never fatal |
| `context` | stdlib | Timeout propagation into graph queries | Already required by all graph methods |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `len(s)/4` byte heuristic | tiktoken-go or anthropic tokenizer | External dep, CGO complexity, ~5% accuracy improvement not worth it |
| `UpdateBeadMetadata` for compaction | New `bd compact` CLI command | bd doesn't have compact subcommand; metadata field is the only available persistence path |
| Metadata field for compact summary | New bead label `gsd:compact` | Labels are boolean flags; metadata supports string values — use metadata for the summary text, label for filtering |

**Installation:** No new packages. All existing.

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── graph/
│   ├── bead.go          # Add TieredBead type, Tier constants
│   ├── query.go         # Add QueryTiered, classifyTier (pure functions)
│   └── update.go        # Add CompactBead (writes summary to metadata + label)
├── hook/
│   └── session_start.go # Refactor buildSessionContext → buildBudgetContext(budget int)
└── mcp/
    ├── tools.go         # Register get_tiered_context (tool 18), update execute_wave
    └── get_tiered_context.go  # New file: handleGetTieredContext
```

### Pattern 1: Tier Constants and TieredBead Type (graph layer)

**What:** Define `Tier` as a string constant (`TierHot`, `TierWarm`, `TierCold`). Define `TieredBead` as a response type that carries both the bead and its tier assignment, with a `Compact` string field for warm/cold rendering.

**When to use:** Any call site that needs to classify beads for context inclusion.

**Example:**
```go
// internal/graph/bead.go additions

// Tier represents a context tier assignment for a bead.
type Tier string

const (
    TierHot  Tier = "hot"
    TierWarm Tier = "warm"
    TierCold Tier = "cold"
)

// TieredBead wraps a Bead with its computed tier and a pre-rendered compact summary.
// Hot beads: Compact == "" (caller uses full fields).
// Warm beads: Compact == "<title>: <close_reason>" (50 tokens).
// Cold beads: Compact == "<id> <title>" (10 tokens).
type TieredBead struct {
    Bead
    Tier    Tier   `json:"tier"`
    Compact string `json:"compact,omitempty"`
}
```

### Pattern 2: classifyTier — Pure Function, No I/O

**What:** `classifyTier` takes a `Bead` and a `time.Time` threshold, returns a `Tier`. This is a pure function — testable without fake_bd, no graph I/O.

**When to use:** Inside `QueryTiered` and inside `buildBudgetContext` when walking bead lists.

**Example:**
```go
// internal/graph/query.go

// classifyTier assigns a tier based on bead status and close time.
// Threshold is the cutoff: beads closed after threshold are warm, before are cold.
// hot:  status == "open" or status == "in_progress"
// warm: closed, ClosedAt >= threshold (e.g., last 48 hours)
// cold: closed, ClosedAt < threshold (or ClosedAt nil on closed bead — treat as cold)
func classifyTier(b Bead, threshold time.Time) Tier {
    if b.Status == "open" || b.Status == "in_progress" {
        return TierHot
    }
    if b.ClosedAt != nil && b.ClosedAt.After(threshold) {
        return TierWarm
    }
    return TierCold
}
```

### Pattern 3: estimateTokens — Single Source of Truth for Budget

**What:** One package-level function in `internal/graph/` or a new `internal/budget/` package. Everything that estimates tokens calls this.

**When to use:** Everywhere. Never inline `len(s)/4`.

**Example:**
```go
// estimateTokens returns a conservative token estimate for a string.
// Uses the 1 token ≈ 4 bytes heuristic (D-07).
// Result is always >= 1 for non-empty strings to prevent undercount.
func estimateTokens(s string) int {
    n := len(s) / 4
    if n == 0 && len(s) > 0 {
        return 1
    }
    return n
}
```

### Pattern 4: Budget Loop with Progressive Degradation (D-08)

**What:** Given a list of tiered beads and a token budget, fill context greedily. When budget is exceeded: first degrade warm→cold, then omit cold entirely.

**When to use:** Inside `buildBudgetContext` (hook) and `handleGetTieredContext` (MCP tool).

**Example:**
```go
// buildBudgetContext fills a context string from tiered beads, degrading to fit budget.
// budget is in tokens (estimated via estimateTokens).
// Degradation: hot always included, warm degraded to cold if over budget, cold omitted if still over.
func buildBudgetContext(hot, warm, cold []TieredBead, budget int) string {
    var sb strings.Builder
    used := 0

    // Always include hot beads (active work — never omit).
    for _, b := range hot {
        chunk := formatHot(b)
        sb.WriteString(chunk)
        used += estimateTokens(chunk)
    }

    // Include warm beads if budget allows; degrade to cold summary if tight.
    for _, b := range warm {
        warmChunk := formatWarm(b)
        if used+estimateTokens(warmChunk) <= budget {
            sb.WriteString(warmChunk)
            used += estimateTokens(warmChunk)
        } else {
            // Degrade to cold: ID + title only.
            coldChunk := formatCold(b)
            if used+estimateTokens(coldChunk) <= budget {
                sb.WriteString(coldChunk)
                used += estimateTokens(coldChunk)
            }
            // If even cold doesn't fit, omit.
        }
    }

    // Include cold beads if budget allows.
    for _, b := range cold {
        coldChunk := formatCold(b)
        if used+estimateTokens(coldChunk) <= budget {
            sb.WriteString(coldChunk)
            used += estimateTokens(coldChunk)
        }
    }

    return sb.String()
}
```

### Pattern 5: Compaction on Close (graph-layer, write path)

**What:** `CompactBead` writes a compact summary to `bead.Metadata["gsd:compact"]` and adds label `gsd:compact` for filterability. Called by `ClosePlan` automatically after closing.

**When to use:** Inside `ClosePlan` in `update.go`, after the close operation succeeds, before returning. Best-effort: compaction failure never blocks close.

**Example:**
```go
// In update.go, after ClosePlan succeeds:

// compactSummary returns the summary text for compaction.
// Format: "<title>: <close_reason>" (warm tier rendering).
func compactSummary(b *Bead) string {
    if b.CloseReason != "" {
        return b.Title + ": " + b.CloseReason
    }
    return b.Title
}

// CompactBead writes the compact summary to bead metadata and adds gsd:compact label.
// Best-effort: errors are returned but callers (ClosePlan) should log and continue.
func (c *Client) CompactBead(ctx context.Context, beadID string, summary string) (*Bead, error) {
    meta := map[string]any{"gsd:compact": summary}
    return c.UpdateBeadMetadata(ctx, beadID, meta)
}
```

### Pattern 6: get_tiered_context MCP Tool (tool 18)

**What:** New MCP tool. Takes `phase_num` (int) and `budget_tokens` (int, default 2000). Returns `TieredContextResult` with hot/warm/cold arrays and a pre-rendered `context_string` that fits the budget.

**When to use:** SKILL.md files that need to build subagent prompts. Hooks call `buildBudgetContext` directly (they don't go through MCP).

**Example:**
```go
// internal/mcp/get_tiered_context.go

type tieredContextArgs struct {
    PhaseNum     int `json:"phase_num"`
    BudgetTokens int `json:"budget_tokens"` // 0 means use default (2000)
}

type tieredContextResult struct {
    Hot           []graph.TieredBead `json:"hot"`
    Warm          []graph.TieredBead `json:"warm"`
    Cold          []graph.TieredBead `json:"cold"`
    ContextString string             `json:"context_string"`
    EstimatedTokens int              `json:"estimated_tokens"`
}
```

### Pattern 7: execute_wave Warm-Tier Compaction (D-11)

**What:** In `execute_wave.go`, when resolving dep summaries, prefer `bead.Metadata["gsd:compact"]` over `bead.CloseReason` if the compact field exists. This is already structurally available — `GetBead` returns the full metadata map.

**When to use:** Replace the existing dep resolution loop in `handleExecuteWave`.

**Example:**
```go
// In execute_wave.go, dep resolution loop:
depBead, err := state.client.GetBead(ctx, dep.DependsOnID)
if err != nil {
    continue
}
// Prefer compacted summary if available (D-11).
summary := ""
if compact, ok := depBead.Metadata["gsd:compact"]; ok {
    if s, ok := compact.(string); ok && s != "" {
        summary = s
    }
}
if summary == "" {
    summary = depBead.CloseReason // Fallback to raw close reason.
}
if summary != "" {
    tc.DepSummaries = append(tc.DepSummaries, summary)
}
```

### Anti-Patterns to Avoid
- **Storing compact summary in a new bead field:** The Bead struct is owned by bd. Adding a `Compact string` to `graph.Bead` would require matching bd's JSON schema. Use `Metadata["gsd:compact"]` instead — metadata is a `map[string]any` and safe for arbitrary extension.
- **Tiering at query time with bd CLI calls:** Fetching individual beads to classify them defeats the purpose. Classify after a single `QueryByLabel` bulk fetch.
- **Using a per-call threshold:** The warm/cold threshold (e.g., 48 hours ago) must be computed once per `buildBudgetContext` call, not per bead. Compute `threshold := time.Now().Add(-48 * time.Hour)` at the top of the function.
- **Making compaction synchronous and blocking in ClosePlan:** Compaction is best-effort. If `CompactBead` fails, log slog.Warn and return the close result anyway. The uncompacted bead is still usable.
- **Injecting budget in SessionStart via a hardcoded constant:** Make the budget a configurable parameter (with a sensible default of 2000 tokens) so it's testable. Pass it into `buildBudgetContext` rather than accessing a global.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Token counting | BPE tokenizer, tiktoken port | `len(s)/4` heuristic | External dep, CGO risk, <5% accuracy delta not worth it for a budget guide |
| Bead status classification | Complex state machine | `classifyTier()` pure function on Status + ClosedAt | Status field is already "open"/"closed"/"in_progress" — string comparison is sufficient |
| Context truncation | Smart sentence splitter, semantic chunker | Budget loop with progressive warm→cold degradation | Simple and deterministic; semantic chunking adds non-determinism with no benefit |
| Compact summary generation | LLM summarization call | `title + ": " + close_reason` string concat | CloseReason is already a human-written summary written at bead-close time; re-summarizing adds latency and cost |

**Key insight:** CloseReason is the compaction artifact. The work of writing a good summary was already done at bead-close time (SKILL.md writes it). Phase 9 doesn't generate new summaries — it surfaces what's already there.

## Common Pitfalls

### Pitfall 1: Mutating Bead Struct for Compact Field
**What goes wrong:** Adding `Compact string` to `graph.Bead` causes JSON unmarshal failures if bd's actual response doesn't include that field, or causes silent zero-values that mask missing compaction.
**Why it happens:** Natural instinct to add fields to the canonical type.
**How to avoid:** Store compact summary in `Metadata["gsd:compact"]`. `TieredBead` carries the rendered `Compact` string as a derived field computed from metadata at classification time, not stored on `Bead`.
**Warning signs:** Any PR that adds a new exported field to `graph.Bead` without a corresponding bd schema change.

### Pitfall 2: Budget Overflow Silently Truncating Hot Beads
**What goes wrong:** If the budget loop applies budget limits to hot beads, active tasks get dropped from context — the worst possible outcome.
**Why it happens:** Generic budget loop treats all tiers symmetrically.
**How to avoid:** Hot beads bypass the budget check entirely. Only warm and cold beads are subject to budget trimming. Document this explicitly in the function comment.
**Warning signs:** Test where hot bead count > budget still returns all hot beads.

### Pitfall 3: Compaction Running on Open Beads
**What goes wrong:** `CompactBead` called on an open or in-progress bead produces a misleading "compact" summary that suppresses full context for an active task.
**Why it happens:** Compaction logic placed in wrong trigger point.
**How to avoid:** `CompactBead` is only called inside `ClosePlan`, after the close succeeds. Add a guard: if `bead.Status != "closed"`, skip compaction and log slog.Warn.
**Warning signs:** `gsd:compact` label appears on a bead with status "open".

### Pitfall 4: 48-Hour Threshold Makes Tests Non-Deterministic
**What goes wrong:** Tests that create real `time.Time` values and check `TierWarm` vs `TierCold` fail depending on when they run.
**Why it happens:** `classifyTier` uses `time.Now()` internally.
**How to avoid:** `classifyTier` takes `threshold time.Time` as a parameter — the caller controls the threshold. Tests pass an explicit `time.Now().Add(-48 * time.Hour)` (or override). The function is pure and deterministic given inputs.
**Warning signs:** Time-dependent test failures on slow CI or at midnight.

### Pitfall 5: get_tiered_context Tool Count Not Updated
**What goes wrong:** `server.go` debug log and `tools_test.go` + `server_test.go` count checks fail because the count wasn't updated from 17 to 18.
**Why it happens:** All previous tool additions required updating 4 files atomically: `tools.go` (registration), `server.go` (debug count), `tools_test.go` (wantNames), `server_test.go` (wantNames).
**How to avoid:** The task that registers `get_tiered_context` MUST update all 4 files in the same commit. Established pattern from Phase 7 Plan 01 key-decisions.
**Warning signs:** `go test ./internal/mcp/...` fails with count mismatch.

### Pitfall 6: QueryByLabel Returns All Phase Beads Including Closed
**What goes wrong:** `buildBudgetContext` queries all `gsd:phase` beads and includes closed phases in the warm/cold tiers, adding significant token overhead for long-running projects.
**Why it happens:** `QueryByLabel("gsd:phase")` has no status filter.
**How to avoid:** Classify the full result set through `classifyTier` and explicitly decide how many cold phase beads to include. For SessionStart, cap cold phase count (e.g., last 3 closed phases) since the active phase is all that matters for warm-start.
**Warning signs:** SessionStart additionalContext grows proportionally to number of completed phases.

## Code Examples

Verified patterns from existing codebase (all file paths confirmed present):

### Token Estimation (stdlib only)
```go
// estimateTokens: 1 token ≈ 4 bytes (D-07).
// Consistent with industry rule of thumb for English prose + code.
func estimateTokens(s string) int {
    if len(s) == 0 {
        return 0
    }
    n := len(s) / 4
    if n == 0 {
        return 1
    }
    return n
}
```

### Tiered Format Functions (pure, testable)
```go
// formatHot returns full bead context for hot-tier beads.
func formatHot(b TieredBead) string {
    var sb strings.Builder
    fmt.Fprintf(&sb, "## [HOT] %s (%s)\n", b.Title, b.ID)
    if b.Description != "" {
        fmt.Fprintf(&sb, "**Goal:** %s\n", b.Description)
    }
    if b.AcceptanceCriteria != "" {
        fmt.Fprintf(&sb, "**Done when:** %s\n", b.AcceptanceCriteria)
    }
    return sb.String()
}

// formatWarm returns summary context for warm-tier beads (~50 tokens).
func formatWarm(b TieredBead) string {
    return fmt.Sprintf("- [WARM] %s: %s\n", b.Title, b.CloseReason)
}

// formatCold returns ID + title only for cold-tier beads (~10 tokens).
func formatCold(b TieredBead) string {
    return fmt.Sprintf("- [%s] %s\n", b.ID, b.Title)
}
```

### Existing ClosePlan Pattern to Extend (source: internal/graph/update.go)
```go
// Current ClosePlan returns (closed *Bead, unblocked []Bead, error).
// Phase 9: after close succeeds, call CompactBead best-effort:
func (c *Client) ClosePlan(ctx context.Context, beadID, reason string) (*Bead, []Bead, error) {
    // ... existing close logic ...

    // Phase 9 addition: compact the closed bead (best-effort).
    if len(closed) > 0 {
        summary := compactSummary(&closed[0])
        if _, err := c.CompactBead(ctx, beadID, summary); err != nil {
            slog.Warn("ClosePlan: compaction failed (best-effort)", "bead_id", beadID, "err", err)
        }
    }

    // ... existing unblocked detection logic ...
}
```

### SessionStart Budget Integration (source: internal/hook/session_start.go)
```go
// buildSessionContext refactored to accept a token budget.
// Default budget for SessionStart = 2000 tokens (~8KB of context).
// Signature change: buildBudgetContext(ctx, client, budget int) string
const sessionStartDefaultBudget = 2000

func buildSessionContext(ctx context.Context, c *graph.Client) string {
    return buildBudgetContext(ctx, c, sessionStartDefaultBudget)
}

// buildBudgetContext is the new implementation (testable with explicit budget).
func buildBudgetContext(ctx context.Context, c *graph.Client, budget int) string {
    // ... query all phase beads + ready beads ...
    // ... classify into hot/warm/cold ...
    // ... call budget loop ...
}
```

### execute_wave Compaction Integration (source: internal/mcp/execute_wave.go)
```go
// In handleExecuteWave dep resolution loop (lines 118-127):
for _, dep := range bead.Dependencies {
    depBead, err := state.client.GetBead(ctx, dep.DependsOnID)
    if err != nil {
        continue
    }
    // Prefer compacted summary if available (D-11, TOKEN-03).
    summary := extractCompact(depBead)
    if summary != "" {
        tc.DepSummaries = append(tc.DepSummaries, summary)
    }
}

// extractCompact returns the compact summary from metadata, falling back to CloseReason.
func extractCompact(b *graph.Bead) string {
    if b.Metadata != nil {
        if v, ok := b.Metadata["gsd:compact"]; ok {
            if s, ok := v.(string); ok && s != "" {
                return s
            }
        }
    }
    return b.CloseReason
}
```

### get_tiered_context MCP Tool Schema
```go
// tools.go registration (tool 18):
server.AddTool(&mcpsdk.Tool{
    Name:        "get_tiered_context",
    Description: "Returns hot/warm/cold tiered context for a phase, fitting within a token budget. Hot=full context, Warm=title+summary, Cold=ID+title only.",
    InputSchema: json.RawMessage(`{"type":"object","properties":{"phase_num":{"type":"integer","description":"Phase number to get context for"},"budget_tokens":{"type":"integer","description":"Token budget (default 2000 if omitted)"}},"required":["phase_num"],"additionalProperties":false}`),
}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
    var args tieredContextArgs
    if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
        return toolError("invalid arguments: " + err.Error()), nil
    }
    return handleGetTieredContext(ctx, state, args)
})
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `buildSessionContext` returns full phase + ready list unconditionally | `buildBudgetContext` trims warm/cold to fit token budget | Phase 9 | SessionStart additionalContext stays bounded as project grows |
| `execute_wave` dep summaries use raw `CloseReason` | Prefer `Metadata["gsd:compact"]`, fall back to `CloseReason` | Phase 9 | Subagent prompts use pre-structured compact summaries |
| No compaction on bead close | `ClosePlan` calls `CompactBead` best-effort after close | Phase 9 | Closed beads carry forward their compact form automatically |
| No tiered query | `classifyTier` + `QueryTiered` graph method | Phase 9 | Callers get tier-classified beads in one call |
| Token estimation: none | `estimateTokens(s string) int` via `len(s)/4` | Phase 9 | Single source of truth for budget math |

**No deprecated features in this phase.** All changes are additive extensions to existing functions.

## Open Questions

1. **Warm-tier threshold: time-based (48h) vs count-based (last N closed)**
   - What we know: D-06 leaves this to Claude's discretion
   - What's unclear: Time-based is simpler but fails for inactive projects (all beads go cold); count-based (last 5 closed) is more robust for project lifecycle
   - Recommendation: Use count-based (last 5 closed beads = warm). Sort by `ClosedAt` desc, take first 5. This is deterministic and project-lifecycle independent. Implement `classifyTier` with a `warmSet map[string]bool` parameter (set of IDs to treat as warm).

2. **Where to put `estimateTokens` and `classifyTier`**
   - What we know: They have no graph.Client dependency — they're pure functions operating on strings and Bead structs
   - What's unclear: Put in `graph` package (co-located with Bead type) or a new `budget` package?
   - Recommendation: Put in `graph` package. Avoids a new package for just 2 functions. Co-location with `Bead` and `TieredBead` is natural.

3. **What token budget to expose in SessionStart**
   - What we know: Claude Code's context window is 200K tokens (claude-sonnet-4-6). SessionStart `additionalContext` competes with conversation history.
   - What's unclear: How many tokens are "available" at SessionStart? Claude Code doesn't expose remaining context to hooks.
   - Recommendation: Default to 2000 tokens (~8KB). This is conservative and leaves ample room. Make it a `const sessionStartDefaultBudget = 2000` so it's easy to tune.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib), `go test -race` |
| Config file | none (standard go test) |
| Quick run command | `go test ./internal/graph/... ./internal/hook/... ./internal/mcp/...` |
| Full suite command | `go test -race ./...` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| TOKEN-01 | `QueryTiered` returns hot/warm/cold classified beads | unit | `go test ./internal/graph/... -run TestQueryTiered -v` | ❌ Wave 0 |
| TOKEN-01 | `classifyTier` pure function correctness | unit | `go test ./internal/graph/... -run TestClassifyTier -v` | ❌ Wave 0 |
| TOKEN-02 | `execute_wave` uses compact summary over raw CloseReason | unit | `go test ./internal/mcp/... -run TestExecuteWaveCompaction -v` | ❌ Wave 0 |
| TOKEN-03 | `ClosePlan` calls `CompactBead` after close | unit | `go test ./internal/graph/... -run TestClosePlanCompacts -v` | ❌ Wave 0 |
| TOKEN-03 | `CompactBead` writes gsd:compact to metadata | unit | `go test ./internal/graph/... -run TestCompactBead -v` | ❌ Wave 0 |
| TOKEN-04 | `TieredBead` type carries correct tier + compact string | unit | `go test ./internal/graph/... -run TestTieredBead -v` | ❌ Wave 0 |
| TOKEN-05 | `estimateTokens` returns len/4 with min 1 | unit | `go test ./internal/graph/... -run TestEstimateTokens -v` | ❌ Wave 0 |
| TOKEN-05 | Budget loop excludes warm beads when over budget | unit | `go test ./internal/hook/... -run TestBuildBudgetContext -v` | ❌ Wave 0 |
| TOKEN-06 | `buildSessionContext` emits reduced context when budget tight | unit | `go test ./internal/hook/... -run TestSessionStartBudget -v` | ❌ Wave 0 |
| TOKEN-06 | `get_tiered_context` MCP tool registered as tool 18 | unit | `go test ./internal/mcp/... -run TestToolCount -v` | ❌ Wave 0 |
| TOKEN-06 | Hot beads always included regardless of budget | unit | `go test ./internal/hook/... -run TestBuildBudgetContextHotAlways -v` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -race ./internal/graph/... ./internal/hook/... ./internal/mcp/...`
- **Per wave merge:** `go test -race ./...` (all 7 packages, currently 164 tests)
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps

All test files are new — they follow the existing TDD pattern established in Phases 6 and 7:

- [ ] `internal/graph/tier_test.go` — covers TOKEN-01, TOKEN-03, TOKEN-04, TOKEN-05 (`TestClassifyTier`, `TestEstimateTokens`, `TestTieredBead`, `TestCompactBead`, `TestClosePlanCompacts`)
- [ ] `internal/graph/tier.go` — new file for `TieredBead`, `Tier`, `classifyTier`, `estimateTokens`, `CompactBead`, `QueryTiered`
- [ ] `internal/mcp/get_tiered_context.go` — new file for `handleGetTieredContext`
- [ ] `internal/mcp/get_tiered_context_test.go` — covers TOKEN-06 tool registration
- [ ] `internal/hook/session_start_test.go` — extend existing file with `TestBuildBudgetContext`, `TestSessionStartBudget`, `TestBuildBudgetContextHotAlways`

*(No framework install needed — existing `go test` infrastructure covers all cases)*

## Sources

### Primary (HIGH confidence)
- Direct codebase inspection — all files read and verified:
  - `internal/graph/bead.go` — Bead struct, ClosedAt *time.Time confirmed present
  - `internal/graph/query.go` — QueryByLabel, ListReady, GetBead implementations confirmed
  - `internal/graph/update.go` — ClosePlan, UpdateBeadMetadata, AddLabel — extension points confirmed
  - `internal/graph/client.go` — run/runWrite patterns confirmed
  - `internal/hook/session_start.go` — buildSessionContext, formatSessionContext — refactor targets confirmed
  - `internal/mcp/execute_wave.go` — dep resolution loop at lines 118-127 confirmed
  - `internal/mcp/tools.go` — 17 tools registered, tool 18 slot available
  - `.planning/phases/09-token-context/09-CONTEXT.md` — all 12 decisions read
- `go test -race ./...` — baseline 164 tests, 7 packages, all passing (confirmed 2026-03-21)

### Secondary (MEDIUM confidence)
- 1 token ≈ 4 bytes heuristic: widely documented OpenAI/Anthropic rule of thumb for English prose, consistent with `tiktoken` benchmarks showing ~4 bytes/token for code-heavy content.
- `Metadata map[string]any` as extension point: confirmed in `bead.go` — safe for arbitrary string values, already used for `gsd_phase`, `gsd_plan`, `gsd_compact` additions will follow same pattern.

### Tertiary (LOW confidence)
- Claude Code's context window remaining budget: not exposed to hooks via any documented hook input field. The 2000-token default for SessionStart is a conservative estimate, not a measured value.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — stdlib only, confirmed by go.mod, no new deps
- Architecture: HIGH — all integration points read directly from source, patterns derived from existing Phase 6/7 code
- Pitfalls: HIGH — most derived from existing key-decisions in STATE.md and SUMMARY files
- Budget default (2000 tokens): MEDIUM — reasonable heuristic, not validated against Claude Code internals

**Research date:** 2026-03-21
**Valid until:** 2026-04-21 (stable Go stdlib, stable bd CLI interface)
