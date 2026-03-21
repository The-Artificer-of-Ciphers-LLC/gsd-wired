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

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"
)

// contextInjectTools are tools that modify state and benefit from bead context injection.
// Read-class tools (Read, Glob, Grep, WebFetch) are excluded — fast path for them.
var contextInjectTools = map[string]bool{
	"Write": true,
	"Edit":  true,
	"Bash":  true,
	"Agent": true,
}

// handlePreToolUse is the handler for PreToolUse hook events.
// Write-class tools get bead context injected; read-class tools get immediate allow.
// Must complete within 500ms latency budget (D-11).
func handlePreToolUse(ctx context.Context, raw json.RawMessage, hs *hookState, w io.Writer) error {
	var input PreToolUseInput
	if err := json.Unmarshal(raw, &input); err != nil {
		slog.Warn("preToolUse: failed to decode input", "err", err)
		return writeOutput(w, preToolUseAllow(""))
	}

	// Fast path: read-class tools get immediate allow with no graph queries.
	if !contextInjectTools[input.ToolName] {
		return writeOutput(w, preToolUseAllow(""))
	}

	// Write-class tool: attempt to build context from local index.
	context := buildPreToolUseContext(ctx, input.CWD, hs)

	return writeOutput(w, preToolUseAllow(context))
}

// buildPreToolUseContext attempts to load bead context for a write-class tool.
// Tries local index first (<1ms), then attempts graph query with strict timeout.
// Always returns a string — partial context is better than nothing.
func buildPreToolUseContext(ctx context.Context, cwd string, hs *hookState) string {
	var parts []string

	// Step 1: Try local index for fast, near-zero-latency context.
	gsdwDir := filepath.Join(cwd, ".gsdw")
	idx, err := graph.LoadIndex(gsdwDir)
	if err == nil && len(idx.PhaseToID) > 0 {
		// Build phase summary from index.
		var phases []string
		for phaseKey := range idx.PhaseToID {
			phases = append(phases, phaseKey)
		}
		parts = append(parts, fmt.Sprintf("Active phases: %s", strings.Join(phases, ", ")))
	}

	// Step 2: If .beads/ exists, try live graph query with strict timeout (400ms).
	beadsPath := filepath.Join(cwd, ".beads")
	if _, statErr := os.Stat(beadsPath); os.IsNotExist(statErr) {
		// No .beads/ — return index-only context (or empty).
		return strings.Join(parts, "\n")
	}

	// Initialize graph client with the project CWD.
	hs.beadsDir = cwd
	if initErr := hs.init(ctx); initErr != nil {
		slog.Warn("preToolUse: graph client init failed", "err", initErr)
		return strings.Join(parts, "\n")
	}

	// Query ready tasks with strict timeout — must not exceed 400ms.
	queryCtx, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
	defer cancel()

	ready, queryErr := hs.client.ListReady(queryCtx)
	if queryErr != nil {
		slog.Warn("preToolUse: list ready failed", "err", queryErr)
		return strings.Join(parts, "\n")
	}

	if len(ready) > 0 {
		var titles []string
		for _, b := range ready {
			titles = append(titles, fmt.Sprintf("[%s] %s", b.ID, b.Title))
		}
		parts = append(parts, "Ready tasks:\n"+strings.Join(titles, "\n"))
	}

	return strings.Join(parts, "\n")
}

// preToolUseAllow builds a HookOutput that allows the tool with optional context.
func preToolUseAllow(additionalContext string) HookOutput {
	return HookOutput{
		HookSpecificOutput: PreToolUseHookOutput{
			HookEventName:      "PreToolUse",
			PermissionDecision: "allow",
			AdditionalContext:  additionalContext,
		},
	}
}
