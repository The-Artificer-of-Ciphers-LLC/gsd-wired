package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"
)

// handleSessionStart is the handler for SessionStart hook events.
// It emits additionalContext containing project state from the beads graph.
// On error, it degrades gracefully — never crashes, always emits valid JSON.
func handleSessionStart(ctx context.Context, raw json.RawMessage, hs *hookState, w io.Writer) error {
	var input SessionStartInput
	if err := json.Unmarshal(raw, &input); err != nil {
		// Decode failed — emit empty output (degraded mode)
		slog.Warn("sessionStart: failed to decode input", "err", err)
		return writeOutput(w, HookOutput{})
	}

	// Fast path: check if .beads/ exists at the CWD.
	beadsPath := filepath.Join(input.CWD, ".beads")
	if _, err := os.Stat(beadsPath); os.IsNotExist(err) {
		hint := "gsd-wired: No .beads/ directory found. Run /gsd-wired:init to initialize."
		return writeOutput(w, HookOutput{AdditionalContext: hint})
	}

	// Set beadsDir from input CWD and initialize the graph client.
	hs.beadsDir = input.CWD
	if err := hs.init(ctx); err != nil {
		slog.Warn("sessionStart: graph client init failed, degrading", "err", err)
		return writeOutput(w, HookOutput{})
	}

	// Use a timeout context leaving 500ms headroom within the 2s budget.
	queryCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	// Build context markdown from graph queries.
	contextStr := buildSessionContext(queryCtx, hs.client)

	return writeOutput(w, HookOutput{AdditionalContext: contextStr})
}

// buildSessionContext queries the graph and builds a markdown context string.
// It always returns a string (possibly empty) and never returns an error —
// partial results are better than nothing.
func buildSessionContext(ctx context.Context, c *graph.Client) string {
	// Query for open phase beads.
	phaseBeads, err := c.QueryByLabel(ctx, "gsd:phase")
	if err != nil {
		slog.Warn("sessionStart: failed to query phase beads", "err", err)
	}

	// Find the current phase: open status + highest gsd_phase metadata value.
	var currentPhase *graph.Bead
	var currentPhaseNum float64 = -1
	for i := range phaseBeads {
		b := &phaseBeads[i]
		if b.Status != "open" {
			continue
		}
		if b.Metadata == nil {
			continue
		}
		phaseNum, ok := phaseNumAsFloat(b.Metadata["gsd_phase"])
		if !ok {
			continue
		}
		if currentPhase == nil || phaseNum > currentPhaseNum {
			currentPhase = b
			currentPhaseNum = phaseNum
		}
	}

	// Query for ready tasks.
	readyBeads, err := c.ListReady(ctx)
	if err != nil {
		slog.Warn("sessionStart: failed to list ready beads", "err", err)
	}

	// Build the markdown string.
	return formatSessionContext(currentPhase, readyBeads)
}

// phaseNumAsFloat extracts a phase number as float64 from a metadata value.
// Handles both float64 (JSON unmarshal) and int (direct construction in tests).
func phaseNumAsFloat(v any) (float64, bool) {
	if v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}

// formatSessionContext builds the markdown string from phase and ready task data.
func formatSessionContext(phase *graph.Bead, ready []graph.Bead) string {
	var sb []byte

	sb = append(sb, "## GSD Project State\n\n"...)

	if phase != nil {
		phaseNum, _ := phaseNumAsFloat(phase.Metadata["gsd_phase"])
		sb = append(sb, fmt.Sprintf("**Current Phase:** %s (Phase %d)\n", phase.Title, int(phaseNum))...)
		sb = append(sb, fmt.Sprintf("**Status:** %s\n\n", phase.Status)...)
	} else {
		sb = append(sb, "**Current Phase:** (none found)\n\n"...)
	}

	sb = append(sb, "### Ready Tasks\n"...)
	if len(ready) == 0 {
		sb = append(sb, "(none)\n"...)
	} else {
		for _, t := range ready {
			sb = append(sb, fmt.Sprintf("- [%s] %s\n", t.ID, t.Title)...)
		}
	}

	if phase != nil && phase.Description != "" {
		sb = append(sb, "\n### Phase Goal\n"...)
		sb = append(sb, phase.Description...)
		sb = append(sb, '\n')
	}

	return string(sb)
}
