package mcp

import (
	"context"
	"fmt"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// advancePhaseArgs holds the arguments for the advance_phase MCP tool.
type advancePhaseArgs struct {
	PhaseNum int    `json:"phase_num"`
	Reason   string `json:"reason"`
}

// advancePhaseResult is the response for the advance_phase MCP tool.
type advancePhaseResult struct {
	Closed    *graph.Bead `json:"closed"`
	Unblocked []graph.Bead `json:"unblocked"`
	NextPhase *phaseInfo  `json:"next_phase"`
}

// handleAdvancePhase implements the advance_phase MCP tool.
// It closes a phase epic bead via ClosePlan and surfaces the next unblocked phase (per D-06, D-07).
func handleAdvancePhase(ctx context.Context, state *serverState, args advancePhaseArgs) (*mcpsdk.CallToolResult, error) {
	if err := state.init(ctx); err != nil {
		return toolError(err.Error()), nil
	}

	// Query all phase beads to find the target.
	phaseBeads, err := state.client.QueryByLabel(ctx, "gsd:phase")
	if err != nil {
		return toolError("failed to query phase beads: " + err.Error()), nil
	}

	// Find the target phase epic bead by phase number in metadata.
	var targetID string
	for i := range phaseBeads {
		if phaseNumFromMeta(phaseBeads[i].Metadata) == args.PhaseNum {
			targetID = phaseBeads[i].ID
			break
		}
	}
	if targetID == "" {
		return toolError(fmt.Sprintf("no phase epic found for phase %d", args.PhaseNum)), nil
	}

	// Close the phase epic bead using ClosePlan (reuse existing close pattern per D-06).
	closed, unblocked, err := state.client.ClosePlan(ctx, targetID, args.Reason)
	if err != nil {
		return toolError("failed to close phase: " + err.Error()), nil
	}

	// Ensure unblocked is not nil for clean JSON output.
	if unblocked == nil {
		unblocked = []graph.Bead{}
	}

	// Find the next open phase: lowest phase number with status "open" and phaseNum > args.PhaseNum.
	var nextPhase *phaseInfo
	nextPhaseNum := -1
	for i := range phaseBeads {
		b := &phaseBeads[i]
		if b.Status != "open" {
			continue
		}
		pn := phaseNumFromMeta(b.Metadata)
		if pn <= args.PhaseNum {
			continue
		}
		if nextPhaseNum == -1 || pn < nextPhaseNum {
			nextPhaseNum = pn
			nextPhase = &phaseInfo{
				BeadID:   b.ID,
				Title:    b.Title,
				PhaseNum: pn,
				Status:   b.Status,
				Goal:     b.Description,
			}
		}
	}

	return toolResult(&advancePhaseResult{
		Closed:    closed,
		Unblocked: unblocked,
		NextPhase: nextPhase,
	})
}
