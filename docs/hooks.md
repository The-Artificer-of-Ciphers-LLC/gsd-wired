# Hooks Reference

gsd-wired registers 4 Claude Code hooks that run automatically during sessions. Hooks are dispatched via `gsdw hook <EventName>` and receive JSON on stdin.

## SessionStart

**Trigger:** Session startup, resume, /clear, or context compaction.

Loads active project state from the beads graph into Claude's context window. Includes current phase, ready tasks, and recent decisions. Uses token-aware tiering to fit within budget.

**Key behavior:**
- Syncs any pending precompact snapshots from previous sessions
- Classifies beads as hot/warm/cold based on recency
- Budget-aware: fits context within ~2000 tokens by default
- Falls back to .planning/ files if beads graph unavailable

## PreToolUse

**Trigger:** Before any tool call.

Fast path for read-class tools (Read, Glob, Grep) — exits immediately with zero overhead. For write-class tools (Write, Edit, Bash), injects relevant bead context from the local index.

**Key behavior:**
- Loads .gsdw/index.json as cheap context source (<1ms)
- Falls back to graph query if index stale (400ms)
- Agent tool calls excluded from tracking

## PostToolUse

**Trigger:** After Write, Edit, or Bash tool calls.

Records tool outcomes to a JSONL audit log for bead state updates. Labels active beads with `gsd:tool-use` for activity tracking.

**Key behavior:**
- Agent tool calls excluded
- JSONL append (not graph write) for minimal overhead
- Bead label update is best-effort

## PreCompact

**Trigger:** Before context compaction (manual or automatic).

Saves in-progress state to `.gsdw/precompact-snapshot.json` atomically (temp file + rename). The next SessionStart syncs this snapshot back to the graph.

**Key behavior:**
- Atomic write prevents corruption on crash
- No goroutines — synchronous write
- Snapshot deleted after successful sync
