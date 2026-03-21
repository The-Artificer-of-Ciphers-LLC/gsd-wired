# Phase 5: Project Initialization - Research

**Researched:** 2026-03-21
**Domain:** Claude Code Plugin Skills (slash commands), Go CLI, MCP tools, interactive questioning flow
**Confidence:** HIGH

## Summary

Phase 5 delivers the first user-facing gsd-wired workflow: `/gsd-wired:init` and `/gsd-wired:status`. The central question was how slash commands work in the Claude Code plugin system and whether multi-turn interactive questioning is achievable via MCP. The answer is straightforward: slash commands are Claude Code Skills (markdown files in `skills/` directory), and interactive questioning is handled natively by Claude reading the SKILL.md and conversing naturally — no special MCP protocol is needed.

The current `plugin.json` only has `name`, `version`, `description`, and `author`. It needs no changes for slash commands because Claude Code auto-discovers skills from the `skills/` directory at the plugin root. The namespace (`gsd-wired:`) is derived from the plugin name field. The MCP server (`.mcp.json`) and hooks (`hooks/hooks.json`) are already wired and need no changes for this phase.

The init flow produces: (1) a `gsdw init` CLI subcommand that calls `bd init` and writes PROJECT.md + config.json, (2) a `skills/init/SKILL.md` that drives the questioning conversation and calls MCP tools, and (3) a `skills/status/SKILL.md` plus `gsdw status` CLI subcommand for the status dashboard. The 30-second auto-proceed is implemented via Claude's normal conversational behavior — no timer mechanism needed.

**Primary recommendation:** Skills (SKILL.md files) are the correct mechanism for slash commands. Claude's native multi-turn conversation handles interactive questioning. MCP tools handle bead creation. CLI subcommand handles `bd init` + file writing.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** This is the first user-facing workflow phase. UX matters.
- **D-02:** gsd-wired translates developer answers to beads behind the scenes. Developer never thinks about bead structure.
- **D-03:** Replicate GSD's current questioning flow — same ~12 questions, same order, same categories.
- **D-04:** Interactive questioning — ask one question, wait for answer, ask next based on response. Not batch.
- **D-05:** Three init modes: Full (12 questions), Quick (3 questions), PR/Issue (existing project, import PR/issue).
- **D-06:** Bead granularity (category beads vs per-answer beads) at Claude's discretion.
- **D-07:** PROJECT.md and config.json written independently as human-readable files. Not derived from bead data — parallel views.
- **D-08:** After init completes: pause and ask developer if ready to proceed. Auto-proceed after 30 seconds of silence.
- **D-09:** Status: Dashboard format: project name, current phase, progress bar, ready tasks, recent activity.
- **D-10:** GSD-familiar terms throughout — phases, plans, waves. Never expose bead structure.
- **D-11:** PR/issue view in separate/debug mode, not mixed with main project status.
- **D-12:** Auto-show on session start (via SessionStart hook), but easily dismissable.

### Claude's Discretion
- Bead structure for init output (category vs per-answer, number of children)
- Quick init: which 3 questions are essential
- PR/Issue mode: questioning flow and bead structure for imported work
- Status dashboard layout and exact fields
- How "easily dismissable" is implemented (flag, config, one-time setting)
- SESSION context injection format for auto-show

### Deferred Ideas (OUT OF SCOPE)
- Roadmap generation from init (Phase 6+ handles planning)
- Requirements definition (Phase 6+)
- Research phase from init context (Phase 6)
- Token-aware status display (Phase 9)
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| INIT-01 | User can initialize a new project via `/gsd-wired:init` slash command | Skills system (SKILL.md) creates slash commands; namespace derives from plugin name |
| INIT-02 | Deep questioning flow captures project context (what, why, who, done criteria) | Claude's native multi-turn conversation within SKILL.md execution; no special protocol |
| INIT-03 | Questioning produces epic bead (project) + context beads (decisions, constraints) | MCP tools create_phase + create_plan already exist; init_project MCP tool needed |
| INIT-04 | PROJECT.md and config.json remain as human-readable files (hybrid state model) | `gsdw init` CLI subcommand writes these files directly to CWD |
| INIT-05 | `bd init` creates .beads/ directory with Dolt-backed storage | serverState.runBdInit() already exists; expose via `gsdw init` CLI subcommand |
| CMD-01 | `/gsd-wired:init` — Initialize new project with deep questioning | skills/init/SKILL.md with `disable-model-invocation: true` |
</phase_requirements>

---

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Claude Code Skills | N/A | Slash command mechanism via SKILL.md | Official plugin mechanism; auto-discovered from `skills/` dir |
| Go stdlib (os, encoding/json) | Go 1.26.1 | Write PROJECT.md and config.json | Already used throughout; no deps needed |
| Cobra (github.com/spf13/cobra) | existing | `gsdw init` and `gsdw status` CLI subcommands | Already used for all CLI commands (NewRootCmd pattern) |
| go-sdk/mcp (v1.4.1) | existing | New MCP tools for init and status | Already used for all 8 existing tools |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| graph.Client | existing | Create project epic + context beads during init | Called from new `init_project` MCP tool |
| buildSessionContext | existing | Status data from beads graph | Reused by `gsdw status` subcommand |
| hookState pattern | existing | Non-batch graph client for read-only queries | Status subcommand uses same pattern |

**Installation:** No new dependencies required.

---

## Architecture Patterns

### How Claude Code Slash Commands Work (Verified: official docs)

Skills in `skills/<name>/SKILL.md` automatically become `/plugin-name:name` slash commands. The plugin name comes from `.claude-plugin/plugin.json`'s `name` field (`"gsd-wired"`), so:

- `skills/init/SKILL.md` → `/gsd-wired:init`
- `skills/status/SKILL.md` → `/gsd-wired:status`

No registration in plugin.json is required. Skills are auto-discovered. The `hooks` and `mcpServers` components are loaded from `hooks/hooks.json` and `.mcp.json` at the plugin root.

**Critical: skills/ directory must be at the plugin ROOT, not inside `.claude-plugin/`.** Only `plugin.json` belongs in `.claude-plugin/`.

```
gsd-wired/              ← plugin root
├── .claude-plugin/
│   └── plugin.json     ← metadata only
├── skills/             ← at ROOT (not inside .claude-plugin/)
│   ├── init/
│   │   └── SKILL.md   → /gsd-wired:init
│   └── status/
│       └── SKILL.md   → /gsd-wired:status
├── hooks/
│   └── hooks.json      ← already registered
└── .mcp.json           ← already registered
```

### Pattern 1: Interactive Questioning via SKILL.md

**What:** Claude reads SKILL.md, which contains natural language instructions. Claude then conducts the questioning flow natively — asking one question, waiting for user response, asking the next. This is Claude's standard conversational ability; no special MCP protocol is needed for multi-turn.

**When to use:** All init questioning (full, quick, PR/issue modes).

**How it works:**
1. User types `/gsd-wired:init [mode]`
2. Claude receives the SKILL.md content as its prompt
3. SKILL.md instructs Claude to ask questions one at a time and wait
4. After collecting answers, SKILL.md instructs Claude to call MCP tools to create beads and write files
5. Claude calls `init_project` MCP tool with collected context

**Example SKILL.md structure:**
```yaml
---
name: init
description: Initialize a new gsd-wired project with guided questioning
disable-model-invocation: true
argument-hint: "[full|quick|pr]"
---

Initialize a gsd-wired project using $ARGUMENTS mode (default: full).

## Your role
You are a builder-partner. Ask questions one at a time, wait for answers.
Never batch questions. Use GSD-familiar language (phases, plans, waves).

## Full init (12 questions)
Ask in this order, waiting for each response:
1. What are you building? (the what)
2. Why are you building it? (the motivation)
...

## After questioning
Call the `init_project` MCP tool with the collected context.
Then write PROJECT.md and config.json using the `write_init_files` MCP tool.
Display the "ready to proceed" message. Auto-proceed after 30 seconds.
```

### Pattern 2: MCP Tool for Init Operations

**What:** A new `init_project` MCP tool handles bead creation (epic + context beads), and a new `write_init_files` MCP tool handles PROJECT.md and config.json writing. Alternatively, these can be a single `init_project` tool that does both.

**Design choice (Claude's discretion):** Combine into one `init_project` MCP tool that:
1. Runs `bd init` if .beads/ does not exist (calls serverState.runBdInit logic)
2. Creates project epic bead with all context
3. Creates context child beads (decisions, constraints, tech stack, done criteria)
4. Writes PROJECT.md and config.json to CWD

This is the cleaner approach — the SKILL.md calls one tool with the full context JSON.

**JSON Schema:**
```json
{
  "type": "object",
  "properties": {
    "project_name": {"type": "string"},
    "what": {"type": "string", "description": "What the project builds"},
    "why": {"type": "string", "description": "Why it exists"},
    "who": {"type": "string", "description": "Target users"},
    "done_criteria": {"type": "string", "description": "What done looks like"},
    "tech_stack": {"type": "string"},
    "constraints": {"type": "string"},
    "risks": {"type": "string"},
    "mode": {"type": "string", "enum": ["full", "quick", "pr"]},
    "pr_url": {"type": "string", "description": "PR/issue URL for pr mode"}
  },
  "required": ["project_name", "what", "why", "done_criteria", "mode"],
  "additionalProperties": false
}
```

### Pattern 3: Status Command Architecture

**What:** `/gsd-wired:status` calls the existing `buildSessionContext` logic but formats it as a dashboard instead of hook additionalContext.

**Two-layer approach:**
1. `skills/status/SKILL.md` — minimal SKILL.md that instructs Claude to call the `get_status` MCP tool and render the result
2. New `get_status` MCP tool — queries graph (reuses `buildSessionContext` logic), returns structured JSON
3. Claude renders the JSON as a dashboard with GSD-familiar terms

**Alternatively:** `gsdw status` CLI subcommand that prints directly to stdout (simpler, but requires skills to call bash). The MCP tool approach is cleaner since the MCP server is already running.

**Recommended:** New `get_status` MCP tool + `skills/status/SKILL.md` that calls it.

### Pattern 4: `gsdw init` CLI Subcommand

**What:** `gsdw init` is a CLI subcommand (not used by the slash command) that handles the bd init + file writing as a standalone operation. Separate from the SKILL.md flow.

**When to use:** Direct CLI usage, or called by the `init_project` MCP tool internally.

**Pattern follows `ready.go`:**
```go
// internal/cli/init.go
func NewInitCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "init",
        Short: "Initialize a new gsd-wired project",
        RunE: func(cmd *cobra.Command, args []string) error {
            // 1. Ensure .beads/ exists (call bd init if needed)
            // 2. Write PROJECT.md template
            // 3. Write .gsdw/config.json
            return nil
        },
    }
    return cmd
}
```

Register in `NewRootCmd()`: `root.AddCommand(..., NewInitCmd(), NewStatusCmd())`

### Anti-Patterns to Avoid

- **Registering slash commands in plugin.json `commands` field:** The `commands` field in plugin.json is for additional markdown files outside the default `skills/` and `commands/` directories. Skills in `skills/` are auto-discovered without any plugin.json entry.
- **Trying to implement multi-turn questioning in Go/MCP:** Claude's conversation handles this natively. Don't try to implement a state machine in Go for question sequencing.
- **Calling `buildSessionContext` from within the MCP server:** The MCP server runs on stdio; CLI subcommands must handle their own graph client init. The `get_status` MCP tool reuses the `graph.Client` methods, not the hook-specific `buildSessionContext` function signature.
- **30-second timer via time.Sleep in MCP tool:** Not feasible. The 30-second auto-proceed is a Claude behavior: SKILL.md instructs Claude to wait 30 seconds and proceed if no response. This is a natural language instruction to Claude.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Slash command registration | Custom JSON fields in plugin.json `commands` array | `skills/init/SKILL.md` auto-discovery | Skills are auto-discovered; `commands` field is for non-default paths only |
| Multi-turn conversation state | MCP state machine with question sequencing | Claude's native conversation in SKILL.md | Claude handles question-wait-question natively when instructed |
| `bd init` logic | Shell exec in SKILL.md via `!` backtick injection | `init_project` MCP tool (reuses serverState.runBdInit) | runBdInit already implemented in mcp/init.go with proper error handling |
| Status data formatting | Duplicate of buildSessionContext in CLI | `get_status` MCP tool + SKILL.md rendering | Reuse existing graph query patterns; avoid divergence |
| Timer mechanism | time.AfterFunc or goroutine countdown | Natural language instruction in SKILL.md | SKILL.md instructs Claude: "auto-proceed after 30s of silence" |

**Key insight:** The questioning flow, conversational waiting, and conditional branching (full vs quick vs pr mode) are all natural language instructions in SKILL.md. Claude's reasoning handles this. Go/MCP only needs to handle durable operations: bead creation, file writing, graph queries.

---

## Common Pitfalls

### Pitfall 1: skills/ directory placed inside .claude-plugin/
**What goes wrong:** Skills are not discovered, slash commands don't appear.
**Why it happens:** Confusing the plugin metadata directory (`.claude-plugin/`) with the plugin root. Only `plugin.json` belongs inside `.claude-plugin/`.
**How to avoid:** Place `skills/` at the plugin root: `gsd-wired/skills/`, not `gsd-wired/.claude-plugin/skills/`.
**Warning signs:** `/gsd-wired:init` not appearing in the `/` autocomplete menu after plugin installation.

### Pitfall 2: Using plugin.json `commands` field instead of skills/ directory
**What goes wrong:** Need to manually list every command file path in plugin.json.
**Why it happens:** Misreading the docs — `commands` in plugin.json supplements the default auto-discovery, it doesn't replace it.
**How to avoid:** Just create `skills/init/SKILL.md` at the plugin root. No plugin.json entry needed.
**Warning signs:** Trying to add `"commands": ["./skills/init/SKILL.md"]` to plugin.json.

### Pitfall 3: `init_project` MCP tool running `bd init` without CWD context
**What goes wrong:** `bd init` runs in wrong directory; .beads/ created in wrong location.
**Why it happens:** MCP tools don't automatically know the project CWD — must use same pattern as serverState: get CWD from os.Getwd() or accept beadsDir parameter.
**How to avoid:** `init_project` tool accepts optional `cwd` parameter; falls back to os.Getwd(). Follows serverState.init() pattern exactly.
**Warning signs:** .beads/ appearing in user's home directory instead of project root.

### Pitfall 4: Status SKILL.md duplicating buildSessionContext logic
**What goes wrong:** Status data drifts from SessionStart context over time.
**Why it happens:** Copy-pasting graph queries into SKILL.md instead of calling MCP tool.
**How to avoid:** Status SKILL.md calls `get_status` MCP tool. The tool uses graph.Client methods (same as buildSessionContext). One source of truth.
**Warning signs:** `/gsd-wired:status` shows different data than SessionStart additionalContext.

### Pitfall 5: stdout pollution from gsdw init/status CLI subcommands
**What goes wrong:** slog output goes to stdout, breaks MCP stdio protocol if MCP is running.
**Why it happens:** slog default handler writes to stderr, but if logging.Init() is not called, slog defaults to os.Stderr — this is actually fine. The risk is any fmt.Println in init/status that bypasses cmd.OutOrStdout().
**How to avoid:** Use `cmd.OutOrStdout()` for all output (established pattern in ready.go). Never use fmt.Println directly. slog goes to stderr per logging.Init() setup.
**Warning signs:** JSON parse errors in MCP communication after `gsdw init` runs.

### Pitfall 6: SKILL.md calling `gsdw init` via `!` backtick injection
**What goes wrong:** `bd init` timeout (30s) blocks Claude's response. User sees no output.
**Why it happens:** `!` backtick commands in SKILL.md run synchronously before Claude sees the content. A slow `bd init` (dolt initialization can take 5-10s) blocks the skill from rendering.
**How to avoid:** Do NOT use `!` backtick injection for `bd init`. Instead, the `init_project` MCP tool handles this asynchronously. Claude calls the tool (which blocks internally) and reports back. This gives proper feedback: "Initializing project... done."
**Warning signs:** SKILL.md hangs on `!` command for several seconds with no output.

---

## Code Examples

Verified patterns from project codebase:

### SKILL.md frontmatter for init (disable-model-invocation required)
```yaml
---
name: init
description: Initialize a new gsd-wired project with deep questioning. Use when starting a new project.
disable-model-invocation: true
argument-hint: "[full|quick|pr]"
---
```
`disable-model-invocation: true` prevents Claude from auto-invoking init — user must explicitly run `/gsd-wired:init`.

### New MCP tool registration (follows existing tools.go pattern)
```go
// Source: internal/mcp/tools.go (existing pattern)
server.AddTool(&mcpsdk.Tool{
    Name:        "init_project",
    Description: "Initialize a new gsd-wired project: runs bd init, creates project epic bead, writes PROJECT.md and config.json.",
    InputSchema: json.RawMessage(`{"type":"object","properties":{...},"required":["project_name","what","why","done_criteria","mode"],"additionalProperties":false}`),
}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
    if err := state.init(ctx); err != nil {  // lazy init including bd init
        return toolError(err.Error()), nil
    }
    // unmarshal args, create beads, write files
    ...
})
```

The critical insight: `state.init(ctx)` already runs `bd init` if .beads/ is absent (see mcp/init.go:runBdInit). The `init_project` tool can rely on this — calling `state.init(ctx)` IS the bd init step. No separate bd init logic needed.

### New CLI subcommand (follows ready.go pattern)
```go
// Source: internal/cli/root.go pattern
func NewInitCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "init",
        Short: "Initialize beads directory and project files",
        RunE: func(cmd *cobra.Command, args []string) error {
            // For CLI use only — SKILL.md uses MCP tool, not CLI
            cwd, _ := os.Getwd()
            // write PROJECT.md template to cwd
            // write .gsdw/config.json
            return nil
        },
    }
}
```

Register in root.go: `root.AddCommand(NewVersionCmd(), NewServeCmd(), NewHookCmd(), NewBdCmd(), NewReadyCmd(), NewInitCmd(), NewStatusCmd())`

### Status MCP tool (reuses buildSessionContext logic)
```go
// New get_status tool: queries graph and returns structured JSON
server.AddTool(&mcpsdk.Tool{
    Name: "get_status",
    Description: "Returns current project status from beads graph: current phase, ready tasks, recent activity.",
    InputSchema: json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`),
}, func(ctx context.Context, req *mcpsdk.CallToolRequest) (*mcpsdk.CallToolResult, error) {
    if err := state.init(ctx); err != nil {
        return toolError(err.Error()), nil
    }
    // Call same graph queries as buildSessionContext:
    // state.client.QueryByLabel(ctx, "gsd:phase")
    // state.client.ListReady(ctx)
    // Return structured JSON for Claude to render as dashboard
    ...
})
```

### PROJECT.md template (D-07: human-readable, not derived from beads)
```go
// Written by init_project MCP tool handler
const projectMDTemplate = `# %s

## What
%s

## Why
%s

## Who
%s

## Done Criteria
%s

## Tech Stack
%s

## Constraints
%s

## Risks
%s

---
*Initialized: %s*
*Mode: %s*
`
```

---

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) + testify/assert |
| Config file | none (go test ./... from module root) |
| Quick run command | `go test ./internal/mcp/... ./internal/cli/... -run TestInit -v` |
| Full suite command | `go test ./... -race` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| INIT-01 | `/gsd-wired:init` slash command exists (skills/init/SKILL.md) | smoke | `ls skills/init/SKILL.md` | ❌ Wave 0 |
| INIT-02 | Questioning flow structure in SKILL.md (correct categories present) | manual | Review SKILL.md content | N/A |
| INIT-03 | `init_project` MCP tool creates project epic bead + context beads | unit | `go test ./internal/mcp/... -run TestToolCallInitProject` | ❌ Wave 0 |
| INIT-04 | `init_project` writes PROJECT.md and config.json to temp dir | unit | `go test ./internal/mcp/... -run TestInitProjectWritesFiles` | ❌ Wave 0 |
| INIT-05 | `bd init` runs when .beads/ absent (via state.init) | unit | existing TestServerStateLazyInit pattern | existing |
| CMD-01 | `gsdw init` subcommand registered in root | unit | `go test ./internal/cli/... -run TestRootCmdHasInit` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/mcp/... ./internal/cli/... -race`
- **Per wave merge:** `go test ./... -race`
- **Phase gate:** Full suite green before `/gsd:verify-work`

### Wave 0 Gaps
- [ ] `internal/mcp/tools_test.go` — add TestToolCallInitProject, TestInitProjectWritesFiles
- [ ] `internal/cli/init_test.go` — TestRootCmdHasInit, TestInitCmdRegistered
- [ ] `internal/cli/status_test.go` — TestStatusCmdRegistered
- [ ] `skills/init/SKILL.md` — created as part of init work, tested by ls smoke check
- [ ] `skills/status/SKILL.md` — created as part of status work

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| plugin.json `commands` field for slash commands | Skills (`skills/<name>/SKILL.md`) auto-discovered | Early 2025 | Commands still work but skills are recommended; both use same slash syntax |
| Separate MCP server config file | Inline in plugin.json or `.mcp.json` | 2025 | `.mcp.json` at plugin root is the default location (already used) |

**Still current:**
- SKILL.md files at `skills/<name>/SKILL.md` with YAML frontmatter is the current standard
- `disable-model-invocation: true` is essential for workflow commands user wants to control
- `${CLAUDE_PLUGIN_ROOT}` env var available in hook commands and MCP server configs
- Plugin namespace = plugin `name` field from plugin.json → `/gsd-wired:init` namespace is already set

---

## Open Questions

1. **`gsdw status` CLI subcommand vs status MCP tool only**
   - What we know: SKILL.md can call MCP tools (preferred) or shell via `!` backtick injection (complex)
   - What's unclear: Whether a `gsdw status` CLI subcommand adds value given the MCP tool approach
   - Recommendation: Implement `get_status` MCP tool + `skills/status/SKILL.md` calling it. Add `gsdw status` CLI subcommand as a convenience that reuses the same graph queries, separate from MCP path.

2. **"Easily dismissable" status auto-show (D-12)**
   - What we know: SessionStart already emits `additionalContext` with project state; D-12 says auto-show but easily dismissable
   - What's unclear: Dismissable means what exactly? "Just don't reply to it" (implicit), or a flag/config option?
   - Recommendation: Dismissable = the additionalContext is injected but Claude does NOT say it aloud unless the user references it. SKILL.md for status is separate from the SessionStart context. Users type `/gsd-wired:status` when they want the full dashboard.

3. **init_project tool: separate bead creation from file writing?**
   - What we know: D-07 says PROJECT.md and config.json are independent of bead data (parallel views)
   - What's unclear: One tool or two? (init_project for beads, write_init_files for markdown)
   - Recommendation: Single `init_project` tool that does both. Simpler SKILL.md. If file writing fails, bead was still created (eventual consistency acceptable for init).

---

## Sources

### Primary (HIGH confidence)
- Official Claude Code plugin docs (https://code.claude.ai/docs/en/plugins-reference) — plugin.json schema, skills auto-discovery, component locations
- Official Claude Code skills docs (https://code.claude.ai/docs/en/skills) — SKILL.md format, frontmatter fields, argument substitution, disable-model-invocation
- Existing project codebase — internal/mcp/init.go (runBdInit), internal/mcp/tools.go (tool pattern), internal/hook/session_start.go (buildSessionContext), internal/cli/ready.go (CLI pattern)

### Secondary (MEDIUM confidence)
- GSD new-project.md workflow (https://~/.claude/get-shit-done/workflows/new-project.md) — 12-question flow structure, AskUserQuestion pattern, auto-proceed behavior
- Existing plugin.json files in user's plugin cache — confirmed plugin.json minimal structure

### Tertiary (LOW confidence)
- None — all critical claims verified against official docs or codebase

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new libraries required; existing patterns verified in codebase
- Architecture: HIGH — slash command mechanism verified against official docs; plugin structure confirmed via cached plugins
- Pitfalls: HIGH — pitfalls derived from official docs warnings + existing established patterns in this codebase
- Validation: HIGH — test patterns established in phases 1-4 (go test ./... -race is proven)

**Research date:** 2026-03-21
**Valid until:** 2026-06-01 (skills API is stable; unlikely to change within 60 days)
