package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// executeWaveArgs holds the arguments for the execute_wave MCP tool.
type executeWaveArgs struct {
	PhaseNum int `json:"phase_num"`
}

// taskContext holds the full context chain for a single task in a wave.
// Per D-04: minimal prompt — all context pre-computed so execution agents need only the bead ID.
type taskContext struct {
	BeadID             string   `json:"bead_id"`
	PlanID             string   `json:"plan_id"`
	Title              string   `json:"title"`
	AcceptanceCriteria string   `json:"acceptance_criteria"`
	Description        string   `json:"description"`
	ParentSummary      string   `json:"parent_summary"`
	DepSummaries       []string `json:"dep_summaries"`
}

// executeWaveResult is the response for the execute_wave MCP tool.
type executeWaveResult struct {
	Wave  int           `json:"wave"`
	Tasks []taskContext `json:"tasks"`
}

// phaseNumFromMeta extracts gsd_phase from bead metadata using a type switch.
// JSON unmarshal produces float64 for numbers; direct construction in tests may use int/int64.
func phaseNumFromMeta(meta map[string]any) int {
	if meta == nil {
		return 0
	}
	v, ok := meta["gsd_phase"]
	if !ok {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}

// planIDFromMeta extracts gsd_plan from bead metadata.
func planIDFromMeta(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	v, ok := meta["gsd_plan"]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

// handleExecuteWave implements the execute_wave MCP tool.
// It pre-computes the full context chain for all ready tasks in a phase (per D-04).
func handleExecuteWave(ctx context.Context, state *serverState, args executeWaveArgs) (*mcpsdk.CallToolResult, error) {
	if err := state.init(ctx); err != nil {
		return toolError(err.Error()), nil
	}

	// Find all phase epics and locate the one matching phase_num.
	epics, err := state.client.QueryByLabel(ctx, "gsd:phase")
	if err != nil {
		return toolError("failed to query phase epics: " + err.Error()), nil
	}

	var phaseBeadID string
	var parentSummary string
	for _, bead := range epics {
		if phaseNumFromMeta(bead.Metadata) == args.PhaseNum {
			phaseBeadID = bead.ID
			// Phase goal is stored in Description field (per CreatePhase which stores goal in --context).
			parentSummary = bead.Description
			break
		}
	}

	if phaseBeadID == "" {
		return toolError(fmt.Sprintf("no phase epic found for phase %d", args.PhaseNum)), nil
	}

	// Get all ready tasks scoped to this phase.
	readyBeads, err := state.client.ReadyForPhase(ctx, phaseBeadID)
	if err != nil {
		return toolError("failed to get ready tasks: " + err.Error()), nil
	}

	// Build taskContext for each ready bead, resolving dependency summaries.
	tasks := make([]taskContext, 0, len(readyBeads))
	for _, bead := range readyBeads {
		tc := taskContext{
			BeadID:             bead.ID,
			PlanID:             planIDFromMeta(bead.Metadata),
			Title:              bead.Title,
			AcceptanceCriteria: bead.AcceptanceCriteria,
			Description:        bead.Description,
			ParentSummary:      parentSummary,
			DepSummaries:       []string{},
		}

		// Resolve each dependency: fetch the dep bead and extract its CloseReason.
		for _, dep := range bead.Dependencies {
			depBead, err := state.client.GetBead(ctx, dep.DependsOnID)
			if err != nil {
				// Best effort: skip unavailable dep beads rather than failing the whole call.
				continue
			}
			if depBead.CloseReason != "" {
				tc.DepSummaries = append(tc.DepSummaries, depBead.CloseReason)
			}
		}

		tasks = append(tasks, tc)
	}

	return toolResult(&executeWaveResult{
		Wave:  1, // Wave number is the current wave; v1 always reports 1 (dynamic computation is v2).
		Tasks: tasks,
	})
}
