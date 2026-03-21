package hook

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"
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

	return writeOutput(w, HookOutput{})
}
