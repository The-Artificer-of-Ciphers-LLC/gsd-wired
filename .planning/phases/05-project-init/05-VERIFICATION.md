---
phase: 05-project-init
verified: 2026-03-21T22:00:00Z
status: passed
score: 10/10 must-haves verified
human_verification:
  - test: "Run /gsd-wired:init full mode in a live Claude Code session"
    expected: "Claude asks one question at a time (12 questions), waits for each answer, then calls init_project MCP tool and displays 'Project initialized: {name}'"
    why_human: "SKILL.md drives interactive Claude behavior — one-question-at-a-time discipline (D-04) cannot be verified by grep alone"
  - test: "Run /gsd-wired:status immediately after /gsd-wired:init"
    expected: "Dashboard shows the project name, current phase (if any), and ready tasks using GSD terminology — no bead IDs or graph internals visible"
    why_human: "Post-init status display across two slash commands requires live Claude session and plugin loaded"
  - test: "Verify auto-proceed after 30 seconds of post-init silence"
    expected: "Claude auto-proceeds to suggest /gsd-wired:status and first-phase planning after 30 seconds with no developer response"
    why_human: "Timing behavior in SKILL.md (D-08) cannot be verified statically"
---

# Phase 5: Project Initialization Verification Report

**Phase Goal:** Users can initialize a new gsd-wired project through guided questioning that produces a beads graph
**Verified:** 2026-03-21T22:00:00Z
**Status:** PASSED (with human verification items)
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Running `/gsd-wired:init` launches a deep questioning flow | VERIFIED | `skills/init/SKILL.md` exists at plugin root with 12-question full flow, 3-question quick mode, and PR/issue mode. `disable-model-invocation: true` prevents accidental trigger. |
| 2 | Questioning produces an epic bead (project) with child context beads | VERIFIED | `handleInitProject` in `init_project.go:43` calls `state.client.CreatePhase(ctx, 0, ...)` for project epic, then `state.client.CreatePlan(...)` for up to 4 context child beads (done-criteria, constraints, tech-stack, risks). |
| 3 | PROJECT.md and config.json created as human-readable files alongside the beads graph | VERIFIED | `init_project.go:87-109` writes `PROJECT.md` via `buildProjectMD()` and `.gsdw/config.json` with project_name, initialized timestamp, and mode. Both `TestToolCallInitProjectWritesFiles` tests confirm actual file creation. |
| 4 | `bd init` creates .beads/ directory with Dolt-backed storage during initialization | VERIFIED | `handleInitProject` calls `state.init(ctx)` at line 38 — this triggers `bd init` if `.beads/` is absent (via `serverState.init` established in Phase 3). CLI `NewInitCmd` also runs `bd init --quiet --skip-hooks --skip-agents`. |
| 5 | Running `/gsd-wired:status` after init shows the project state from the beads graph | VERIFIED | `skills/status/SKILL.md` calls `get_status` MCP tool; `handleGetStatus` queries `gsd:project`, `gsd:phase`, and `ListReady` to return structured JSON; `renderStatus` in `status.go` renders GSD-familiar dashboard (never exposes bead IDs). |
| 6 | init_project MCP tool is registered and wired (10 total tools) | VERIFIED | `tools.go` has 10 `AddTool` calls; `init_project` registered at line 217, `get_status` at line 230. `TestToolsRegistered` passes. |
| 7 | SKILL.md instructs one-question-at-a-time discipline | VERIFIED | `skills/init/SKILL.md:12` — "Ask questions one at a time, wait for each answer before asking the next. Never batch multiple questions." |
| 8 | Three init modes supported (full, quick, PR/issue) | VERIFIED | `skills/init/SKILL.md` contains distinct sections: "Full init (12 questions)", "Quick init (3 questions)", "PR/Issue mode". `initProjectArgs.Mode` enum validates `"full"|"quick"|"pr"` in tool schema. |
| 9 | gsdw init CLI subcommand creates .beads/ + template files | VERIFIED | `NewInitCmd()` in `internal/cli/init.go` runs bd init, writes PROJECT.md template with placeholders, writes `.gsdw/config.json`. `TestInitCmdWritesFiles` confirms actual file creation in temp dir. |
| 10 | gsdw status CLI subcommand shows project state from graph | VERIFIED | `NewStatusCmd()` in `internal/cli/status.go` calls `QueryByLabel("gsd:phase")` and `ListReady()`, passes to `renderStatus()`. `TestStatusCmdOutput` and `TestStatusCmdNoProject` both pass. |

**Score:** 10/10 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/mcp/init_project.go` | `func handleInitProject` with bead creation and file writing | VERIFIED | 152 lines; `handleInitProject`, `initProjectArgs`, `initProjectResult`, `buildProjectMD` all present and substantive |
| `internal/mcp/get_status.go` | `func handleGetStatus` with graph queries | VERIFIED | 138 lines; `handleGetStatus`, `statusResult`, `phaseInfo`, `taskInfo`, `phaseNumFromMetadata` all present |
| `internal/mcp/tools.go` | 10 tools registered (8 existing + init_project + get_status) | VERIFIED | Exactly 10 `AddTool` calls; `init_project` and `get_status` added at lines 217 and 230 |
| `skills/init/SKILL.md` | `/gsd-wired:init` slash command with questioning flow | VERIFIED | Exists at plugin root; `disable-model-invocation: true`, `argument-hint: "[full|quick|pr]"`, 12-question full flow present |
| `skills/status/SKILL.md` | `/gsd-wired:status` slash command calling get_status | VERIFIED | Exists at plugin root; references `get_status` MCP tool; GSD-familiar rendering instructions present |
| `internal/cli/init.go` | `func NewInitCmd` CLI subcommand | VERIFIED | `NewInitCmd()` present; runs bd init, writes PROJECT.md template, writes `.gsdw/config.json` |
| `internal/cli/status.go` | `func NewStatusCmd` CLI subcommand | VERIFIED | `NewStatusCmd()` and `renderStatus()` present; uses `findBeadsDir`, `phaseNumFromBead`, `planIDFromBead` from `ready.go` |
| `internal/cli/root.go` | Updated root with NewInitCmd and NewStatusCmd | VERIFIED | Line 31: `root.AddCommand(..., NewInitCmd(), NewStatusCmd())` |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|-----|--------|---------|
| `internal/mcp/init_project.go` | `graph.Client.CreatePhase` | `state.client.CreatePhase` for project epic | WIRED | Line 43: `state.client.CreatePhase(ctx, 0, args.ProjectName, args.What, args.DoneCriteria, []string{"gsd:project"})` |
| `internal/mcp/init_project.go` | `graph.Client.CreatePlan` | `state.client.CreatePlan` for child context beads | WIRED | Line 68: `state.client.CreatePlan(ctx, cb.planID, 0, projectBead.ID, ...)` in loop |
| `internal/mcp/get_status.go` | `graph.Client.QueryByLabel` | `state.client.QueryByLabel` for phase lookup | WIRED | Line 50: `QueryByLabel(ctx, "gsd:project")` and line 58: `QueryByLabel(ctx, "gsd:phase")` |
| `skills/init/SKILL.md` | `init_project` MCP tool | Claude calls init_project after questioning | WIRED | Line 50: "call the `init_project` MCP tool with the collected context as a single JSON object" |
| `skills/status/SKILL.md` | `get_status` MCP tool | Claude calls get_status for dashboard data | WIRED | Line 7: "Show the current project status by calling the `get_status` MCP tool." |
| `internal/cli/root.go` | `internal/cli/init.go` | `root.AddCommand(NewInitCmd())` | WIRED | Line 31: `NewInitCmd()` included in `AddCommand` chain |
| `internal/cli/status.go` | `internal/cli/ready.go` | `findBeadsDir`, `phaseNumFromBead`, `planIDFromBead` | WIRED | `status.go` calls all three package-level functions defined in `ready.go` |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| INIT-01 | 05-02-PLAN.md | User can initialize a new project via `/gsd-wired:init` slash command | SATISFIED | `skills/init/SKILL.md` exists at plugin root; auto-discovered as `/gsd-wired:init` slash command |
| INIT-02 | 05-02-PLAN.md | Deep questioning flow captures project context (what, why, who, done criteria) | SATISFIED | `skills/init/SKILL.md` has 12-question full flow covering what, why, who, done criteria, tech stack, constraints, prior art, risks, v1 scope, out-of-scope, hard constraints, catch-all |
| INIT-03 | 05-01-PLAN.md | Questioning produces epic bead (project) + context beads (decisions, constraints) | SATISFIED | `handleInitProject` creates project epic bead (phaseNum=0, label gsd:project) + up to 4 category context child beads |
| INIT-04 | 05-01-PLAN.md | PROJECT.md and config.json remain as human-readable files (hybrid state model) | SATISFIED | Both files written by `handleInitProject` and `NewInitCmd`; content is human-readable Markdown and JSON |
| INIT-05 | 05-01-PLAN.md | `bd init` creates .beads/ directory with Dolt-backed storage | SATISFIED | `state.init(ctx)` triggers bd init if `.beads/` absent; `NewInitCmd` runs `bd init --quiet --skip-hooks --skip-agents` |
| CMD-01 | 05-02-PLAN.md | `/gsd-wired:init` — Initialize new project with deep questioning | SATISFIED | `skills/init/SKILL.md` provides the slash command with full questioning flow |

**Note:** REQUIREMENTS.md traceability table and checkbox entries for INIT-01, INIT-02, and CMD-01 still show status "Pending" / `[ ]`. The implementation satisfies these requirements but the requirements file was not updated to mark them complete. This is a documentation inconsistency — not an implementation gap.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `internal/cli/init.go` | 15 | Comment mentions "placeholders" — intentional in PROJECT.md template | Info | Correct behavior: the CLI `gsdw init` writes a placeholder template; the `/gsd-wired:init` slash command populates real content via MCP tool. Not a stub. |

No blockers. No implementation stubs. The comment at `init.go:15` describes intentional behavior — the PROJECT.md template written by `gsdw init` is deliberately minimal (developer fills it in, or `/gsd-wired:init` populates it via the MCP tool). This is documented design decision D-07.

### Human Verification Required

#### 1. Interactive /gsd-wired:init Questioning Flow

**Test:** In a live Claude Code session with the plugin loaded, run `/gsd-wired:init` with no arguments (defaults to full mode).
**Expected:** Claude asks "What are you building?" first. After answering, Claude asks the second question only. This continues for all 12 questions in order. After all answers are collected, Claude calls the `init_project` MCP tool, then displays "Project initialized: {project_name}" followed by "Ready to proceed? (auto-continuing in 30 seconds...)".
**Why human:** The one-question-at-a-time discipline (D-04) and 30-second auto-proceed (D-08) are behavioral constraints on Claude's execution of `SKILL.md` — they cannot be verified by static analysis.

#### 2. /gsd-wired:status After /gsd-wired:init

**Test:** After running `/gsd-wired:init`, immediately run `/gsd-wired:status` in the same session.
**Expected:** Dashboard displays project name as header, current phase info (if phases exist), ready tasks as bullet list — all using phases/plans/waves terminology with no bead IDs or graph internals visible.
**Why human:** Cross-command state continuity and rendered output quality require a live session.

#### 3. Post-Init Auto-Proceed Timer

**Test:** After `/gsd-wired:init` completes and displays "Ready to proceed?", wait 30+ seconds without responding.
**Expected:** Claude auto-proceeds and suggests running `/gsd-wired:status` and planning the first phase.
**Why human:** Timing behavior cannot be verified statically and requires live session observation.

### Gaps Summary

No implementation gaps found. All 10 must-have truths are verified, all artifacts are substantive and wired, all 6 required key links are connected, and all 6 phase requirements (INIT-01 through INIT-05, CMD-01) are satisfied by the implementation.

The only outstanding item is a documentation inconsistency: REQUIREMENTS.md still marks INIT-01, INIT-02, and CMD-01 as "Pending" — these should be updated to "Complete" and the checkboxes should be checked. This is a bookkeeping gap in the planning documents, not a code gap.

**Test suite:** All 7 packages pass `go test ./... -count=1 -race` with no failures.

---

_Verified: 2026-03-21T22:00:00Z_
_Verifier: Claude (gsd-verifier)_
