package hook

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"
)

// beadUpdateTools are tools whose invocations are recorded to the local JSONL log.
// Agent is excluded from PostToolUse recording (per plan spec: Write, Edit, Bash only).
var beadUpdateTools = map[string]bool{
	"Write": true,
	"Edit":  true,
	"Bash":  true,
}

// toolEvent is a single record appended to .gsdw/tool-events.jsonl.
type toolEvent struct {
	ToolName  string `json:"tool_name"`
	ToolUseID string `json:"tool_use_id"`
	Timestamp string `json:"timestamp"` // RFC3339 UTC
	SessionID string `json:"session_id"`
}

// handlePostToolUse is the handler for PostToolUse hook events.
// Write-class tools get a tool event appended to .gsdw/tool-events.jsonl.
// Read-class tools are no-ops (returns empty HookOutput immediately).
// PostToolUse does NOT inject additionalContext for v1 (deferred to v2/TOKEN-A01).
func handlePostToolUse(ctx context.Context, raw json.RawMessage, hs *hookState, w io.Writer) error {
	var input PostToolUseInput
	if err := json.Unmarshal(raw, &input); err != nil {
		slog.Warn("postToolUse: failed to decode input", "err", err)
		return writeOutput(w, HookOutput{})
	}

	// Fast path: non-write tools are no-ops.
	if !beadUpdateTools[input.ToolName] {
		return writeOutput(w, HookOutput{})
	}

	// Write-class tool: append event record to .gsdw/tool-events.jsonl.
	event := toolEvent{
		ToolName:  input.ToolName,
		ToolUseID: input.ToolUseID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		SessionID: input.SessionID,
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		slog.Warn("postToolUse: failed to marshal event", "err", err)
		return writeOutput(w, HookOutput{})
	}

	// Ensure .gsdw/ directory exists.
	gsdwDir := filepath.Join(input.CWD, ".gsdw")
	if mkErr := os.MkdirAll(gsdwDir, 0755); mkErr != nil {
		slog.Warn("postToolUse: failed to create .gsdw dir", "err", mkErr)
		return writeOutput(w, HookOutput{})
	}

	// Append JSON line to tool-events.jsonl.
	eventsPath := filepath.Join(gsdwDir, "tool-events.jsonl")
	f, err := os.OpenFile(eventsPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Warn("postToolUse: failed to open tool-events.jsonl", "err", err)
		return writeOutput(w, HookOutput{})
	}
	defer f.Close()

	// Write JSON line + newline terminator.
	line := append(eventData, '\n')
	if _, writeErr := f.Write(line); writeErr != nil {
		slog.Warn("postToolUse: failed to write tool event", "err", writeErr)
	}

	// Best-effort bead state update: add gsd:tool-use label to the active bead.
	// This satisfies INFRA-08 (bead state update after tool execution).
	// Failure must never affect JSONL write or hook output.
	updateBeadOnToolUse(ctx, input.CWD, hs)

	return writeOutput(w, HookOutput{})
}

// updateBeadOnToolUse adds the gsd:tool-use label to the active bead in the local index.
// Best-effort: any error is logged and silently swallowed — JSONL is the reliable path.
func (hs *hookState) beadUpdateTimeoutDuration() time.Duration {
	if hs.beadUpdateTimeout > 0 {
		return time.Duration(hs.beadUpdateTimeout) * time.Millisecond
	}
	return 400 * time.Millisecond
}

func updateBeadOnToolUse(ctx context.Context, cwd string, hs *hookState) {
	// Fast path: skip if .beads/ doesn't exist (uninitialized project).
	beadsPath := filepath.Join(cwd, ".beads")
	if _, err := os.Stat(beadsPath); os.IsNotExist(err) {
		return
	}

	// Initialize the graph client (sets beadsDir if not already set).
	hs.beadsDir = cwd
	if err := hs.init(ctx); err != nil {
		slog.Warn("postToolUse: graph client init failed", "err", err)
		return
	}

	// Use a configurable timeout for the graph call (default 400ms).
	updateCtx, cancel := context.WithTimeout(ctx, hs.beadUpdateTimeoutDuration())
	defer cancel()

	// Load local index to find active bead (cheapest path, <1ms).
	gsdwDir := filepath.Join(cwd, ".gsdw")
	idx, err := graph.LoadIndex(gsdwDir)
	if err != nil {
		slog.Warn("postToolUse: failed to load index", "err", err)
		return
	}

	// Find the first plan bead ID in the index as the active bead.
	// The index maps plan keys (e.g., "04-03") to bead IDs.
	var activeBeadID string
	for _, beadID := range idx.PlanToID {
		activeBeadID = beadID
		break
	}
	if activeBeadID == "" {
		// No active plan bead — skip silently (common for projects without active tasks).
		return
	}

	// Add gsd:tool-use label to mark bead as having received tool activity.
	if _, err := hs.client.AddLabel(updateCtx, activeBeadID, "gsd:tool-use"); err != nil {
		slog.Warn("postToolUse: failed to add gsd:tool-use label", "err", err)
	}
}
