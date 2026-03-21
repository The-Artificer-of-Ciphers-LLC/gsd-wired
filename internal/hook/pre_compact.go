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

// compactSnapshot is the local buffer written by PreCompact before compaction.
// It captures enough state to resume after the context is compacted.
type compactSnapshot struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	Trigger        string `json:"trigger"`
	Timestamp      string `json:"timestamp"` // RFC3339 UTC
	CWD            string `json:"cwd"`
}

// handlePreCompact is the handler for PreCompact hook events.
// It saves session state to .gsdw/precompact-snapshot.json atomically and returns
// immediately with empty HookOutput. This must NEVER block compaction (per D-07).
//
// CRITICAL: No goroutines. Process exits after compaction; goroutines would be killed
// before completing any async Dolt sync (research Pitfall 2).
func handlePreCompact(ctx context.Context, raw json.RawMessage, hs *hookState, w io.Writer) error {
	var input PreCompactInput
	if err := json.Unmarshal(raw, &input); err != nil {
		// Decode failed — log and continue. PreCompact must not block.
		slog.Error("precompact: failed to decode input", "err", err)
		return writeOutput(w, HookOutput{})
	}

	// Build snapshot with current timestamp.
	snapshot := compactSnapshot{
		SessionID:      input.SessionID,
		TranscriptPath: input.TranscriptPath,
		Trigger:        input.Trigger,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		CWD:            input.CWD,
	}

	data, err := json.Marshal(snapshot)
	if err != nil {
		slog.Error("precompact: failed to marshal snapshot", "err", err)
		return writeOutput(w, HookOutput{})
	}

	// Write atomically to .gsdw/precompact-snapshot.json via temp+rename.
	// On any write error, log and continue — write is best-effort.
	gsdwDir := filepath.Join(input.CWD, ".gsdw")
	if mkErr := os.MkdirAll(gsdwDir, 0755); mkErr != nil {
		slog.Error("precompact: local write failed", "err", mkErr)
		return writeOutput(w, HookOutput{})
	}

	bufferPath := filepath.Join(gsdwDir, "precompact-snapshot.json")
	tmp := bufferPath + ".tmp"

	if writeErr := os.WriteFile(tmp, data, 0644); writeErr != nil {
		slog.Error("precompact: local write failed", "err", writeErr)
		return writeOutput(w, HookOutput{})
	}

	if renameErr := os.Rename(tmp, bufferPath); renameErr != nil {
		slog.Error("precompact: local write failed", "err", renameErr)
		// Best-effort: the .tmp file exists but rename failed. Clean up silently.
		_ = os.Remove(tmp)
		return writeOutput(w, HookOutput{})
	}

	// Return empty output — PreCompact does not inject context.
	return writeOutput(w, HookOutput{})
}
