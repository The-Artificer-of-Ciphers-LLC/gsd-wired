# Phase 2: Graph Primitives - Research

**Researched:** 2026-03-21
**Domain:** Go CLI wrapper architecture, bd JSON API, local index patterns, tree-format CLI output
**Confidence:** HIGH — all critical findings verified by direct bd CLI execution on the actual binary

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- D-01: bd is an implementation detail — gsdw is the interface. Users never need to think about bd query syntax, label conventions, or metadata keys.
- D-02: Developers interact with gsdw commands only. bd mechanics happen behind the scenes.
- D-03: Phases use `--type epic`, plans use `--type task` with `--parent <phase-bead-id>`
- D-04: Success criteria stored in `--acceptance`
- D-05: Requirement IDs stored as labels via `-l REQ-ID,REQ-ID` on create; `--add-label REQ-ID` on update
- D-06: GSD role markers as labels — `gsd:phase` and `gsd:plan`
- D-07: Inter-plan dependencies via `--deps <bead-id>`
- D-08: Human-readable context in `--context`, structured machine data in `--metadata` JSON
- D-09: Native bd fields first; metadata JSON only for GSD-specific extras (gsd_phase, gsd_plan, gsd_wave, gsd_files_modified, gsd_req_ids)
- D-10: Hybrid wave model — wave numbers in metadata for reporting, `bd ready` for actual execution ordering
- D-11: Wave cache invalidated after every `bd close`
- D-12: `gsdw ready` shows tree format (human) and `--json` (machine)
- D-13: On task close, gsdw notifies what became ready: "Closed bd-a3f → 2 new tasks ready: bd-c9d, bd-e1f"
- D-14: Own simpler interface for `gsdw ready` with `--phase N`, `--json`. `gsdw bd ready` passthrough as escape hatch.
- D-15: "Queued" terminology for dependency-waiting tasks
- D-16: Progress shown as "3 ready │ 4 queued │ 7 remaining"
- D-17: Metadata is canonical lookup — `gsd_phase: 3`, `gsd_plan: "02-01"` in metadata JSON
- D-18: Local index at `.gsdw/index.json`, rebuildable from `bd list --json` if stale
- D-19: GSD names only in output; bd IDs are implementation details
- D-20: gsdw enforces ownership — warn if bd mutates gsd-labeled beads directly

### Claude's Discretion
- bd wrapper Go architecture (client struct, exec.Command patterns, error handling)
- Bead creation timing (batch at plan-phase vs incremental at each step)
- Claiming mechanics (gsdw claim vs auto-claim in gsdw ready)
- Index file schema and rebuild strategy
- JSON parsing approach for bd --json output
- Error messages when bd is not installed or database not initialized

### Deferred Ideas (OUT OF SCOPE)
- Direct Go import of beads library instead of CLI wrapper — Phase 6/optimization path
- MCP tool registration for graph operations — Phase 3
- Hook-triggered automatic bead state updates — Phase 4
- Token-aware context injection from bead data — Phase 9
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INFRA-03 | bd CLI wrapper layer shells out to `bd --json` for all graph operations | Verified exact JSON schema, error patterns, exit codes from live bd binary |
| MAP-01 | Phase maps to epic bead with metadata (phase number, goal, success criteria) | Confirmed: `--type epic`, `--acceptance`, `--metadata '{"gsd_phase": N}'` |
| MAP-02 | Plan maps to task bead with parent-child relationship to phase epic | Confirmed: `--type task --parent <epic-id>`, child ID format: `epic-id.N` |
| MAP-03 | Wave computed dynamically from dependency graph via `bd ready` | Confirmed: `bd ready --json` returns only unblocked tasks; `bd blocked` shows queued; `--parent` filter works |
| MAP-04 | Success criteria stored as extensible fields on task beads | Confirmed: `--acceptance` is a native bd field, appears as `acceptance_criteria` in JSON |
| MAP-05 | Requirement IDs stored as bead tags for traceability | Confirmed: `-l REQ-ID` on create; `--add-label REQ-ID` on update; `label=REQ-ID` in bd query |
| MAP-06 | GSD-specific metadata stored via bd's extensible fields | Confirmed: `--metadata '{"gsd_phase": N, "gsd_wave": N}'` stored as JSON object; queryable via `--metadata-field key=value` |
</phase_requirements>

## Summary

Phase 2 builds the `internal/graph/` package — a Go client struct wrapping bd CLI via `exec.Command` — and delivers the `gsdw ready` subcommand. The bd CLI (v0.61.0) exposes a clean `--json` flag that produces consistent JSON for all read/write operations. Error handling follows a two-tier pattern: pre-connection errors go to stderr only (check exit code), post-connection errors appear as `{"error": "..."}` on stdout with exit 1.

The GSD-to-beads mapping is straightforward. Phases are epic beads, plans are task beads with parent IDs. bd's native ID scheme creates hierarchical IDs automatically: a child of `bd-proj-c4l` becomes `bd-proj-c4l.1`, `bd-proj-c4l.2`, etc. Labels (not metadata keys) are the correct mechanism for requirement IDs since `bd query label=INFRA-03` is supported natively. The `bd ready --parent <epic-id>` query provides phase-scoped wave computation without client-side filtering.

The local index at `.gsdw/index.json` should be a thin lookup table mapping GSD identifiers (phase numbers, plan IDs) to bd bead IDs. It should never be treated as the source of truth — only as a lookup cache for avoiding repeated `bd list` calls. The index is invalidated after any write operation (create/close) and rebuilt from `bd list --all --label gsd:phase --json` and `bd list --all --label gsd:plan --json`.

**Primary recommendation:** Build `internal/graph/` as a `Client` struct with injected `bdPath string` and `beadsDir string`, all operations using `exec.CommandContext` with captured stdout/stderr pipes, and a typed `Bead` struct matching the verified JSON schema. Never parse stderr — only stdout JSON.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `os/exec` (stdlib) | Go 1.26.1 | Subprocess bd CLI calls | Already established in Phase 1 (bd.go passthrough); no external dep needed |
| `encoding/json` (stdlib) | Go 1.26.1 | Decode bd --json output | Already established in hook/dispatcher.go; zero deps |
| `os` (stdlib) | Go 1.26.1 | Atomic index file writes, BEADS_DIR env | Already used throughout codebase |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `context` (stdlib) | Go 1.26.1 | Timeout/cancellation for bd invocations | Wrap every exec.Command with context; prevents hung processes |
| `sync` (stdlib) | Go 1.26.1 | Index file mutex | Concurrent read/write protection on index.json |
| `log/slog` (stdlib) | Go 1.26.1 | Debug logging for bd invocations | Already established — log args/exit code to stderr |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `encoding/json` manual struct | `gjson` or `tidwall/gjson` | gjson is faster for single-key lookups but adds a dep; not worth it for the call volume of this wrapper |
| `os.WriteFile` atomic via temp | `fsync`-safe write library | For index.json, write-to-temp + rename is sufficient; no third-party lib needed |

**Installation:**
```bash
# No new dependencies needed for Phase 2 — all stdlib
go mod tidy
```

**Version verification:** bd v0.61.0 confirmed at `~/.local/bin/bd`. Go 1.26.1 at `/opt/homebrew/bin/go`. No package versions to verify — pure stdlib.

## Architecture Patterns

### Recommended Project Structure
```
internal/
├── graph/               # New package: bd client and GSD domain mapping
│   ├── client.go        # Client struct, exec.Command pattern, BEADS_DIR injection
│   ├── bead.go          # Bead/Dependency typed structs matching bd JSON schema
│   ├── create.go        # CreatePhase(), CreatePlan() — bd create wrappers
│   ├── query.go         # ListReady(), ListBlocked(), GetBead(), QueryByLabel()
│   ├── update.go        # ClaimBead(), CloseBead(), AddLabel() — bd update/close wrappers
│   ├── index.go         # Index struct, Load/Save/Rebuild, .gsdw/index.json management
│   └── graph_test.go    # Tests using fake bd binary (build tag or test helper)
└── cli/
    ├── ready.go         # New: gsdw ready subcommand consuming internal/graph
    └── root.go          # Register ready subcommand (existing)
.gsdw/
└── index.json           # GSD-name → bd-id mapping (created by gsdw, not committed)
```

### Pattern 1: Client Struct with Injected Configuration

**What:** A `Client` struct holds the bd binary path, BEADS_DIR, and a context. All bd operations are methods on Client.
**When to use:** Always — never call exec.Command directly from CLI layer.

```go
// Source: verified pattern from internal/cli/bd.go + Phase 1 decisions
package graph

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "os/exec"
)

type Client struct {
    bdPath   string  // resolved at construction via exec.LookPath
    beadsDir string  // path to .beads/ directory (sets BEADS_DIR env)
}

func NewClient(beadsDir string) (*Client, error) {
    bdPath, err := exec.LookPath("bd")
    if err != nil {
        return nil, fmt.Errorf("bd not found on PATH — install beads first: %w", err)
    }
    return &Client{bdPath: bdPath, beadsDir: beadsDir}, nil
}

// run executes a bd command and returns parsed JSON output.
// Stdout is expected to be JSON. Stderr is captured for debug logging only.
func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) {
    args = append(args, "--json")
    cmd := exec.CommandContext(ctx, c.bdPath, args...)
    cmd.Env = append(os.Environ(), "BEADS_DIR="+c.beadsDir)

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        slog.Debug("bd command failed", "args", args, "stderr", stderr.String())
        // Check if stdout has a JSON error object (post-connection failure)
        if stdout.Len() > 0 {
            var bdErr struct{ Error string `json:"error"` }
            if jsonErr := json.Unmarshal(stdout.Bytes(), &bdErr); jsonErr == nil && bdErr.Error != "" {
                return nil, fmt.Errorf("bd error: %s", bdErr.Error)
            }
        }
        return nil, fmt.Errorf("bd %v exited %w: %s", args[0], err, stderr.String())
    }
    return stdout.Bytes(), nil
}
```

### Pattern 2: Typed Bead Struct Matching Verified JSON Schema

**What:** A `Bead` struct with all fields confirmed from live bd output. Omitempty on optional fields.
**When to use:** All bd JSON responses decode into `[]Bead` (list/ready/query) or `Bead` (single show).

```go
// Source: verified from live bd create/list/show/ready --json output (bd v0.61.0)
package graph

import "time"

type Bead struct {
    ID                string            `json:"id"`
    Title             string            `json:"title"`
    Description       string            `json:"description,omitempty"`
    AcceptanceCriteria string           `json:"acceptance_criteria,omitempty"`
    Status            string            `json:"status"`           // open, in_progress, blocked, deferred, closed
    Priority          int               `json:"priority"`
    IssueType         string            `json:"issue_type"`       // task, epic, chore, etc.
    Assignee          string            `json:"assignee,omitempty"`
    Owner             string            `json:"owner,omitempty"`
    CreatedAt         time.Time         `json:"created_at"`
    CreatedBy         string            `json:"created_by,omitempty"`
    UpdatedAt         time.Time         `json:"updated_at"`
    ClosedAt          *time.Time        `json:"closed_at,omitempty"`
    CloseReason       string            `json:"close_reason,omitempty"`
    Metadata          map[string]any    `json:"metadata,omitempty"`
    Labels            []string          `json:"labels,omitempty"`
    Dependencies      []Dependency      `json:"dependencies,omitempty"`
    Dependents        []BeadSummary     `json:"dependents,omitempty"`
    DependencyCount   int               `json:"dependency_count,omitempty"`
    DependentCount    int               `json:"dependent_count,omitempty"`
    CommentCount      int               `json:"comment_count,omitempty"`
    Parent            string            `json:"parent,omitempty"`
}

type Dependency struct {
    IssueID     string    `json:"issue_id"`
    DependsOnID string    `json:"depends_on_id"`
    Type        string    `json:"type"`    // "parent-child", "blocks", etc.
    CreatedAt   time.Time `json:"created_at"`
    CreatedBy   string    `json:"created_by"`
    // Note: "metadata" in dependency is a JSON string, not object — use string not map
    Metadata    string    `json:"metadata"`
}

// BeadSummary appears in "dependents" array from bd show --json
type BeadSummary struct {
    ID             string    `json:"id"`
    Title          string    `json:"title"`
    Status         string    `json:"status"`
    IssueType      string    `json:"issue_type"`
    DependencyType string    `json:"dependency_type"`
}
```

### Pattern 3: Index File — GSD Name to bd ID Mapping

**What:** A thin JSON file at `.gsdw/index.json` mapping human GSD identifiers to bd bead IDs. Write via temp+rename for atomicity.
**When to use:** Every CreatePhase/CreatePlan writes an entry. Every Close/Claim reads an entry. Rebuild by calling `bd list --all --label gsd:phase --json` + `bd list --all --label gsd:plan --json`.

```go
// Source: established pattern from Phase 1 (injected I/O), stdlib idiom
package graph

import (
    "encoding/json"
    "os"
    "path/filepath"
)

type Index struct {
    // PhaseToID maps "phase-2" → "bd-proj-c4l"
    PhaseToID map[string]string `json:"phase_to_id"`
    // PlanToID maps "02-01" → "bd-proj-c4l.1"
    PlanToID  map[string]string `json:"plan_to_id"`
}

func (idx *Index) Save(dir string) error {
    path := filepath.Join(dir, "index.json")
    tmp := path + ".tmp"
    data, err := json.MarshalIndent(idx, "", "  ")
    if err != nil {
        return err
    }
    if err := os.WriteFile(tmp, data, 0644); err != nil {
        return err
    }
    return os.Rename(tmp, path)  // atomic on same filesystem
}
```

### Pattern 4: gsdw ready Output — Tree Format

**What:** Render `bd ready --json` output as a tree grouped by parent phase.
**When to use:** `gsdw ready` subcommand; filter by `--phase N` reduces to one phase's tasks.

```
Ready Work (3 tasks, 7 remaining)

  Phase 2: Graph Primitives
  ├─ Plan 02-01: bd CLI wrapper      [INFRA-03]
  └─ Plan 02-02: Domain mapping      [MAP-01, MAP-02]

  Phase 10: Coexistence
  └─ Plan 10-01: .planning/ reader   [COMPAT-01]

Total: 3 ready │ 4 queued │ 7 remaining
```

Implementation: group `[]Bead` from `bd ready --label gsd:plan` by `bead.Metadata["gsd_phase"]`, look up phase names from index, extract labels matching `[A-Z]+-[0-9]+` for the bracket display. Counts: ready = len(bd ready result), queued = len(bd blocked --label gsd:plan), remaining = queued + in_progress.

### Anti-Patterns to Avoid

- **Parsing bd stderr:** bd error output changes format. Only parse stdout JSON. Exit code + stdout JSON error field is the contract.
- **Using `bd list --ready`:** This is NOT equivalent to `bd ready`. Per bd docs: "bd list --ready only filters by status=open" — it misses dependency-blocked tasks. Always use `bd ready` subcommand for wave computation.
- **Storing bd IDs as the primary key in index.json:** bd IDs are stable but opaque. Index.json must be keyed by GSD names (phase numbers, plan IDs), with bd IDs as values.
- **Treating `dependency_count` as blocked count:** `dependency_count` in list output includes the parent-child relationship. A task with `dependency_count: 1` is NOT necessarily blocked — it may only have a parent link. Use `bd blocked` for actual blocked/queued count.
- **Not passing context to exec.Command:** Use `exec.CommandContext(ctx, ...)` always. bd can hang if dolt server is unresponsive.
- **Calling `bd label` command:** Labels are set at create time with `-l` (comma-separated, no spaces). Updates use `--add-label` or `--remove-label`. There is no separate `bd label add` workflow needed for gsdw's create path.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Wave/dependency computation | Custom graph traversal for blocked/ready states | `bd ready --json` + `bd blocked --json` | bd uses GetReadyWork API with blocker-aware semantics; naive graph traversal misses edge cases (in_progress, deferred, hooked) |
| Parent-child querying | Recursive tree traversal via list | `bd list --parent <id> --json` or `bd ready --parent <id>` | Native server-side filter; handles all depth levels |
| Label-based querying | Client-side label array scanning | `bd query "label=INFRA-03" --json` | Server-side indexed; scales to many beads |
| Claim semantics | Optimistic concurrency with CAS loop | `bd update --claim` | Native atomic claim (fails if already claimed); implementing correctly requires distributed locking |
| Bead count | Scanning full list and counting | `bd count --json` | Returns `{"count": N}` directly |

**Key insight:** bd's server-side filtering is always preferable to loading all beads and filtering in Go. The only time to load all is for the index rebuild.

## Common Pitfalls

### Pitfall 1: Labels Not Appearing in Create Response
**What goes wrong:** `bd create ... --labels "gsd:phase,INFRA-03" --json` returns the new bead without labels in the JSON response.
**Why it happens:** Verified in testing — labels ARE applied but are NOT included in the `bd create --json` response. Labels only appear in `bd list --json`, `bd show --json`, and `bd ready --json`.
**How to avoid:** Never read labels from the create response. Always use `bd show <id> --json` or trust the index if you need to verify labels were set.
**Warning signs:** Bead struct has empty `Labels` field after decode of create response.

**UPDATE FROM LIVE TESTING:** Actually observed that create returns the bead WITHOUT labels field. Confirmed labels appear in list/show/ready. Don't assert labels in post-create tests.

### Pitfall 2: Two-Tier Error Handling
**What goes wrong:** Wrapping `cmd.Run()` error and not checking stdout is insufficient. When bd connects to dolt but an operation fails, the error is a JSON object on stdout with exit code 1 — stderr is empty.
**Why it happens:** bd uses two error channels: pre-connection failures go to stderr (human text), post-connection failures go to stdout as `{"error": "..."}` JSON.
**How to avoid:** After `cmd.Run()` fails: (1) if stdout is non-empty, try to unmarshal as `{"error": "..."}` and use that message; (2) if stdout is empty, use stderr text.
**Warning signs:** Error messages that say "exit status 1" with no detail.

### Pitfall 3: BEADS_DIR vs. --db Flag
**What goes wrong:** Using `--db /path/to/.beads` flag does NOT work correctly — it tries to connect to a different dolt server and fails with "database not found".
**Why it happens:** `--db` is for specifying which database name to use on the connected server, not which .beads directory to use. The dolt server is process-scoped to a specific data directory.
**How to avoid:** Use `BEADS_DIR=/path/to/.beads` environment variable on the exec.Command. Confirmed working in testing.
**Warning signs:** "database X not found on Dolt server" errors when bd command should work.

### Pitfall 4: Labels on create use `-l` / `--labels` (comma-separated), updates use `--add-label`
**What goes wrong:** Using `--add-label` on `bd create` fails with "unknown flag".
**Why it happens:** `--add-label` is an `update`-only flag. `create` uses `-l` or `--labels` with comma-separated values.
**How to avoid:** In `CreatePhase`/`CreatePlan`, pass labels as: `--labels`, `"gsd:phase,INFRA-03"`. In `AddLabel` wrapper, use `bd update <id> --add-label <label>`.
**Warning signs:** "unknown flag: --add-label" on create call.

### Pitfall 5: Child ID Format is Hierarchical
**What goes wrong:** Assuming bd IDs are always flat hashes. Child beads get IDs like `bd-proj-c4l.1`, `bd-proj-c4l.2` — with the parent ID as prefix.
**Why it happens:** bd creates hierarchical IDs for child beads. This is by design.
**How to avoid:** Never parse bd IDs for structure. Use `bead.Parent` field from JSON to discover parentage. Index.json stores the full ID as-is.
**Warning signs:** ID string splitting logic, substring matching on IDs.

### Pitfall 6: `bd ready` Does Not Support `--label` Filter for Queued Count
**What goes wrong:** `bd ready --label gsd:plan` returns an empty array even when there are open gsd:plan tasks.
**Why it happens:** When all gsd:plan tasks are either `in_progress` or blocked by in_progress tasks, `bd ready` correctly returns empty. The label filter is applied AFTER the ready semantics filter.
**How to avoid:** For "queued" count, use `bd blocked --json` (returns dependency-blocked tasks) and filter client-side for `gsd:plan` label. For "remaining", sum ready + in_progress + queued.
**Warning signs:** Queued count always showing 0 when tasks exist.

## Code Examples

Verified patterns from live bd testing:

### Creating a Phase (Epic Bead)
```go
// Source: verified bd create --json output, bd v0.61.0
func (c *Client) CreatePhase(ctx context.Context, phaseNum int, title, goal, acceptance string, reqIDs []string) (*Bead, error) {
    meta := map[string]any{"gsd_phase": phaseNum}
    metaJSON, _ := json.Marshal(meta)

    labels := "gsd:phase"
    for _, rid := range reqIDs {
        labels += "," + rid
    }

    out, err := c.run(ctx,
        "create", title,
        "--type", "epic",
        "--acceptance", acceptance,
        "--context", goal,
        "--metadata", string(metaJSON),
        "--labels", labels,
    )
    if err != nil {
        return nil, err
    }
    var bead Bead
    return &bead, json.Unmarshal(out, &bead)
}
```

### Creating a Plan (Task Bead)
```go
// Source: verified bd create --parent --json output
func (c *Client) CreatePlan(ctx context.Context, planID string, phaseNum int, parentBeadID, title, acceptance, context string, reqIDs []string, depBeadIDs []string) (*Bead, error) {
    meta := map[string]any{"gsd_phase": phaseNum, "gsd_plan": planID}
    metaJSON, _ := json.Marshal(meta)

    labels := "gsd:plan"
    for _, rid := range reqIDs {
        labels += "," + rid
    }

    args := []string{
        "create", title,
        "--type", "task",
        "--parent", parentBeadID,
        "--acceptance", acceptance,
        "--context", context,
        "--metadata", string(metaJSON),
        "--labels", labels,
    }
    if len(depBeadIDs) > 0 {
        args = append(args, "--deps", strings.Join(depBeadIDs, ","))
    }

    out, err := c.run(ctx, args...)
    if err != nil {
        return nil, err
    }
    var bead Bead
    return &bead, json.Unmarshal(out, &bead)
}
```

### Getting Ready Tasks for a Phase
```go
// Source: verified bd ready --parent --json; returns [] not null for empty
func (c *Client) ReadyForPhase(ctx context.Context, phaseBeadID string) ([]Bead, error) {
    out, err := c.run(ctx, "ready",
        "--parent", phaseBeadID,
        "--limit", "0",  // unlimited
    )
    if err != nil {
        return nil, err
    }
    var beads []Bead
    return beads, json.Unmarshal(out, &beads)
}
```

### Close with Post-Close Ready List (for D-13 notifications)
```go
// Source: bd close --json returns only the closed bead; must call bd ready separately
func (c *Client) ClosePlan(ctx context.Context, beadID, reason string) (*Bead, []Bead, error) {
    // Step 1: Get ready list BEFORE close to compute diff
    prevReady, err := c.ListReady(ctx)
    if err != nil {
        return nil, nil, fmt.Errorf("pre-close ready snapshot: %w", err)
    }
    prevReadyIDs := make(map[string]bool, len(prevReady))
    for _, b := range prevReady {
        prevReadyIDs[b.ID] = true
    }

    // Step 2: Close the bead
    args := []string{"close", beadID}
    if reason != "" {
        args = append(args, "--reason", reason)
    }
    out, err := c.run(ctx, args...)
    if err != nil {
        return nil, nil, err
    }
    var closed []Bead
    if err := json.Unmarshal(out, &closed); err != nil {
        return nil, nil, err
    }

    // Step 3: Get post-close ready list; new entries = newly unblocked
    newReady, err := c.ListReady(ctx)
    if err != nil {
        return &closed[0], nil, nil  // close succeeded; notification is best-effort
    }
    var newlyUnblocked []Bead
    for _, b := range newReady {
        if !prevReadyIDs[b.ID] {
            newlyUnblocked = append(newlyUnblocked, b)
        }
    }

    return &closed[0], newlyUnblocked, nil
}
```

### Index Rebuild
```go
// Source: verified bd list --all --label --json; labels filter works
func (c *Client) RebuildIndex(ctx context.Context) (*Index, error) {
    phases, err := c.run(ctx, "list", "--all", "--label", "gsd:phase", "--limit", "0")
    if err != nil {
        return nil, err
    }
    plans, err := c.run(ctx, "list", "--all", "--label", "gsd:plan", "--limit", "0")
    if err != nil {
        return nil, err
    }

    idx := &Index{
        PhaseToID: make(map[string]string),
        PlanToID:  make(map[string]string),
    }

    var phaseBeads []Bead
    json.Unmarshal(phases, &phaseBeads)
    for _, b := range phaseBeads {
        if phaseNum, ok := b.Metadata["gsd_phase"]; ok {
            key := fmt.Sprintf("phase-%v", phaseNum)
            idx.PhaseToID[key] = b.ID
        }
    }

    var planBeads []Bead
    json.Unmarshal(plans, &planBeads)
    for _, b := range planBeads {
        if planID, ok := b.Metadata["gsd_plan"]; ok {
            idx.PlanToID[fmt.Sprint(planID)] = b.ID
        }
    }

    return idx, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| bd SQLite backend | Dolt-only backend | bd v0.61.0 | Cannot use SQLite; dolt must be running; BEADS_DIR env var is the targeting mechanism |
| `bd list --ready` | `bd ready` subcommand | Current | These are NOT equivalent; `bd ready` uses GetReadyWork API with blocker semantics |
| `--storage sqlite` flag | `--backend sqlite` flag (deprecated) | bd v0.61.0 | SQLite backend fully removed; use Dolt only |

**Deprecated/outdated:**
- `--storage sqlite`: Removed. `--backend sqlite` also removed (prints deprecation notice and exits 1).
- `dolt_server_port` in metadata.json: Deprecated. Use `.beads/dolt-server.port` file instead.
- `bd list --ready`: Still exists but semantically weaker than `bd ready`; never use for wave computation.

## Open Questions

1. **Label inheritance behavior on child beads**
   - What we know: `bd create ... --labels "gsd:phase,INFRA-03"` results in only those labels on the created bead, but the list response for child tasks showed the parent's `gsd:phase` label even though tasks were created with `gsd:plan`.
   - What's unclear: Does `--no-inherit-labels` need to be passed to prevent parent label inheritance bleeding into child beads? In testing, child tasks inherited parent's `gsd:phase` label plus had their own `gsd:plan` label.
   - Recommendation: Pass `--no-inherit-labels` when creating task beads to keep label semantics clean. Verify in tests that `gsd:phase` appears only on epics, `gsd:plan` only on tasks.

2. **bd ready --limit default is 10**
   - What we know: `bd ready --help` shows `--limit int Maximum issues to show (default 10)`.
   - What's unclear: Whether the wrapper should always pass `--limit 0` for unlimited results.
   - Recommendation: Always pass `--limit 0` in all `bd ready` and `bd list` calls from the wrapper to prevent silent truncation. This is load-bearing — a truncated wave would cause tasks to appear "missing."

3. **Dolt server auto-start behavior**
   - What we know: bd auto-starts dolt when `BEADS_DIR` is set and no explicit port is configured. When an explicit port is in config, auto-start is suppressed.
   - What's unclear: How to ensure the right dolt server is running for the correct project when BEADS_DIR switches between projects.
   - Recommendation: gsdw should always set `BEADS_DIR` to `.gsdw/../.beads` (relative to the CWD of the process) and not pin a port. Let bd manage server lifecycle.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | `testing` (stdlib) + `go test` |
| Config file | none — `go test ./...` discovers all |
| Quick run command | `go test ./internal/graph/... -race -count=1` |
| Full suite command | `go test ./... -race -count=1` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INFRA-03 | Client.run() captures stdout JSON, handles exit 1 + JSON error on stdout | unit | `go test ./internal/graph/... -run TestClient -race` | ❌ Wave 0 |
| INFRA-03 | Client.run() returns error when bd not found (LookPath fail) | unit | `go test ./internal/graph/... -run TestNewClient -race` | ❌ Wave 0 |
| MAP-01 | CreatePhase() produces epic bead with correct fields | integration | `go test ./internal/graph/... -run TestCreatePhase -race` | ❌ Wave 0 |
| MAP-02 | CreatePlan() produces task bead with parent field set | integration | `go test ./internal/graph/... -run TestCreatePlan -race` | ❌ Wave 0 |
| MAP-03 | ReadyForPhase() returns only unblocked tasks | integration | `go test ./internal/graph/... -run TestReadyForPhase -race` | ❌ Wave 0 |
| MAP-04 | AcceptanceCriteria field populated in CreatePlan | unit | `go test ./internal/graph/... -run TestCreatePlanFields -race` | ❌ Wave 0 |
| MAP-05 | Labels include REQ-IDs after create | integration | `go test ./internal/graph/... -run TestLabels -race` | ❌ Wave 0 |
| MAP-06 | Metadata JSON contains gsd_phase, gsd_plan, gsd_wave | unit | `go test ./internal/graph/... -run TestMetadata -race` | ❌ Wave 0 |
| INFRA-03 | gsdw ready subcommand prints tree format to stdout | integration | `go test ./internal/cli/... -run TestReadyCmd -race` | ❌ Wave 0 |
| INFRA-03 | gsdw ready --json outputs valid JSON array | integration | `go test ./internal/cli/... -run TestReadyCmdJSON -race` | ❌ Wave 0 |

### Test Strategy Notes

Integration tests for `internal/graph/` require a live bd + dolt environment. Two approaches:

**Option A (recommended): Fake bd binary.** Create `internal/graph/testdata/fake_bd.go` that compiles to a `bd` binary returning canned JSON. Use `t.TempDir()` + `go build` in `TestMain` to produce the fake binary. Inject its path into `NewClient`. This matches the Phase 1 pattern of injected dependencies and avoids dolt dependency in CI.

**Option B: Skip tag.** Tag integration tests requiring live dolt with `//go:build integration` and run them separately. Use `t.Skip()` if `BD_TEST_BEADS_DIR` env var not set.

Recommendation: Use Option A for unit-like tests (JSON parsing, error handling), Option B for end-to-end `gsdw ready` smoke tests.

### Sampling Rate
- **Per task commit:** `go test ./internal/graph/... -race -count=1`
- **Per wave merge:** `go test ./... -race -count=1`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/graph/graph_test.go` — covers INFRA-03, MAP-01 through MAP-06
- [ ] `internal/graph/testdata/fake_bd/main.go` — fake bd binary for unit tests
- [ ] `internal/cli/ready_test.go` — covers gsdw ready subcommand
- [ ] `.gsdw/` directory (created at runtime, not in git — `.gitignore` entry needed)

## Sources

### Primary (HIGH confidence)
- Live bd v0.61.0 binary at `~/.local/bin/bd` — JSON schema verified by direct execution
- `bd create --json`, `bd list --json`, `bd show --json`, `bd ready --json`, `bd update --json`, `bd close --json`, `bd blocked --json`, `bd count --json` — all responses verified
- `bd --help`, `bd create --help`, `bd list --help`, `bd update --help`, `bd ready --help`, `bd close --help`, `bd blocked --help` — all flags verified
- Phase 1 SUMMARY.md — established patterns (injected I/O, stderr-only logging, exec.CommandContext)
- CONTEXT.md D-01 through D-20 — locked decisions

### Secondary (MEDIUM confidence)
- bd doctor output confirming Dolt-only backend, metadata.json deprecations
- BEADS_DIR env var behavior verified by live testing

### Tertiary (LOW confidence)
- Label inheritance behavior (needs targeted test to fully confirm)
- Dolt server auto-start behavior under concurrent multi-project scenarios

## Metadata

**Confidence breakdown:**
- JSON schema: HIGH — all fields verified from live bd v0.61.0 output
- bd flag names and behavior: HIGH — verified from --help + live execution
- Architecture patterns: HIGH — follows Phase 1 established patterns
- Test strategy: MEDIUM — fake bd approach is sound but requires Wave 0 scaffolding
- Label inheritance: LOW — observed but not exhaustively tested

**Research date:** 2026-03-21
**Valid until:** 2026-06-21 (stable bd API; bd version pinned at 0.61.0 on this machine)
