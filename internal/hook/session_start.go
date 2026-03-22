package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/compat"
	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"
)

// syncPendingSnapshot checks for a pending precompact-snapshot.json and, if found,
// syncs it to the active phase bead in Dolt via UpdateBeadMetadata.
// On any error: logs slog.Warn and returns without blocking SessionStart.
// On success: removes the snapshot file to prevent re-sync on next session.
func syncPendingSnapshot(ctx context.Context, cwd string, c *graph.Client) {
	snapshotPath := filepath.Join(cwd, ".gsdw", "precompact-snapshot.json")

	// Fast path: snapshot doesn't exist — common case.
	if _, err := os.Stat(snapshotPath); os.IsNotExist(err) {
		return
	}

	// Read and unmarshal the snapshot.
	data, err := os.ReadFile(snapshotPath)
	if err != nil {
		slog.Warn("sessionStart: syncPendingSnapshot: failed to read snapshot", "err", err)
		return
	}

	var snapshot compactSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		slog.Warn("sessionStart: syncPendingSnapshot: failed to unmarshal snapshot", "err", err)
		return
	}

	// Find the active phase bead to update.
	phaseBeads, err := c.QueryByLabel(ctx, "gsd:phase")
	if err != nil {
		slog.Warn("sessionStart: syncPendingSnapshot: failed to query phase beads", "err", err)
		return
	}

	// Find the current active (open) phase bead.
	var phaseBead *graph.Bead
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
		if phaseBead == nil || phaseNum > currentPhaseNum {
			phaseBead = b
			currentPhaseNum = phaseNum
		}
	}

	if phaseBead == nil {
		slog.Warn("sessionStart: syncPendingSnapshot: no active phase bead found")
		return
	}

	// Update the bead with snapshot metadata.
	meta := map[string]any{
		"last_precompact":  snapshot.Timestamp,
		"last_session_id": snapshot.SessionID,
	}
	if _, err := c.UpdateBeadMetadata(ctx, phaseBead.ID, meta); err != nil {
		slog.Warn("sessionStart: syncPendingSnapshot: failed to update bead metadata", "err", err)
		return
	}

	// Remove snapshot file — successfully synced.
	if err := os.Remove(snapshotPath); err != nil {
		slog.Warn("sessionStart: syncPendingSnapshot: failed to remove snapshot file", "err", err)
	}
}

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
		// .beads/ absent — check for .planning/ fallback (COMPAT-01, D-10).
		if compat.DetectPlanning(input.CWD) {
			fb, fbErr := compat.BuildFallbackStatus(input.CWD)
			if fbErr == nil {
				ctx := formatFallbackContext(fb)
				return writeOutput(w, HookOutput{AdditionalContext: ctx})
			}
			slog.Warn("sessionStart: .planning/ fallback failed", "err", fbErr)
		}
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

	// Sync any pending precompact-snapshot.json to Dolt before loading context.
	// This is Stage 2 of INFRA-06: fast local write (PreCompact) + Dolt sync (SessionStart).
	// Best-effort: any error is logged and SessionStart continues normally.
	syncPendingSnapshot(queryCtx, input.CWD, hs.client)

	// Build context markdown from graph queries.
	contextStr := buildSessionContext(queryCtx, hs.client)

	return writeOutput(w, HookOutput{AdditionalContext: contextStr})
}

// sessionStartDefaultBudget is the default token budget for SessionStart additionalContext.
// Chosen to allow full context for an active phase while leaving room for the user's prompt.
const sessionStartDefaultBudget = 2000

// buildSessionContext queries the graph and builds a budget-aware markdown context string.
// It is a thin wrapper around buildBudgetContext using sessionStartDefaultBudget.
// It always returns a string (possibly empty) and never returns an error —
// partial results are better than nothing.
func buildSessionContext(ctx context.Context, c *graph.Client) string {
	return buildBudgetContext(ctx, c, sessionStartDefaultBudget)
}

// buildBudgetContext queries the graph and builds a budget-aware markdown context string.
// budget is the maximum number of tokens to include in the output (estimated via graph.EstimateTokens).
// Progressive degradation: hot beads always included, warm beads included if budget allows,
// cold beads omitted when over budget. Per Research Pattern 4 and D-08.
func buildBudgetContext(ctx context.Context, c *graph.Client, budget int) string {
	// Query for tiered phase beads (5 warm beads per Open Question 1 recommendation).
	hot, warm, cold, err := c.QueryTiered(ctx, "gsd:phase", 5)
	if err != nil {
		slog.Warn("sessionStart: failed to query tiered phase beads", "err", err)
	}

	var sb strings.Builder
	sb.WriteString("## GSD Project State\n\n")
	used := graph.EstimateTokens("## GSD Project State\n\n")

	// Always include hot beads (active work — never omit per Pitfall 2).
	for _, b := range hot {
		chunk := graph.FormatHot(b)
		sb.WriteString(chunk)
		used += graph.EstimateTokens(chunk)
	}

	// Include warm beads if budget allows; degrade to cold format if tight.
	for _, b := range warm {
		warmChunk := graph.FormatWarm(b)
		if used+graph.EstimateTokens(warmChunk) <= budget {
			sb.WriteString(warmChunk)
			used += graph.EstimateTokens(warmChunk)
		} else {
			// Degrade to cold: ID + title only.
			coldChunk := graph.FormatCold(b)
			if used+graph.EstimateTokens(coldChunk) <= budget {
				sb.WriteString(coldChunk)
				used += graph.EstimateTokens(coldChunk)
			}
			// If even cold doesn't fit, omit.
		}
	}

	// Include cold beads if budget allows.
	for _, b := range cold {
		coldChunk := graph.FormatCold(b)
		if used+graph.EstimateTokens(coldChunk) <= budget {
			sb.WriteString(coldChunk)
			used += graph.EstimateTokens(coldChunk)
		}
	}

	// Query for ready tasks — always included (active work).
	readyBeads, err := c.ListReady(ctx)
	if err != nil {
		slog.Warn("sessionStart: failed to list ready beads", "err", err)
	}

	sb.WriteString("\n### Ready Tasks\n")
	if len(readyBeads) == 0 {
		sb.WriteString("(none)\n")
	} else {
		for _, t := range readyBeads {
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", t.ID, t.Title))
		}
	}

	return sb.String()
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

// formatFallbackContext formats a FallbackStatus (from .planning/) into a markdown string
// suitable for additionalContext in SessionStart. Prepends the compatibility mode indicator
// per D-01. Only reads — never writes to .planning/ (D-09, COMPAT-03).
func formatFallbackContext(fb compat.FallbackStatus) string {
	var sb strings.Builder
	sb.WriteString("gsd-wired: Running in .planning/ compatibility mode\n\n")
	sb.WriteString("## GSD Project State (.planning/ mode)\n\n")

	if fb.ProjectName != "" {
		sb.WriteString(fmt.Sprintf("**Project:** %s\n", fb.ProjectName))
	}
	if fb.CoreValue != "" {
		sb.WriteString(fmt.Sprintf("**Core Value:** %s\n", fb.CoreValue))
	}

	s := fb.State
	if s.CurrentPhase > 0 {
		progress := s.Progress
		if progress == "" {
			progress = "(unknown)"
		}
		sb.WriteString(fmt.Sprintf("**Current Phase:** %d — %s\n", s.CurrentPhase, progress))
	}
	if s.CurrentPlan > 0 && s.TotalPlans > 0 {
		sb.WriteString(fmt.Sprintf("**Plan:** %d of %d\n", s.CurrentPlan, s.TotalPlans))
	}

	if len(fb.Phases) > 0 {
		sb.WriteString("\n### Phases\n")
		for _, p := range fb.Phases {
			check := "[ ]"
			if p.Complete {
				check = "[x]"
			}
			sb.WriteString(fmt.Sprintf("- %s Phase %d: %s\n", check, p.Number, p.Name))
		}
	}

	return sb.String()
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
