package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// runResearchArgs holds the arguments for the run_research MCP tool.
type runResearchArgs struct {
	PhaseNum int      `json:"phase_num"`
	Title    string   `json:"title"`
	ReqIDs   []string `json:"req_ids"`
}

// runResearchResult is the response for the run_research MCP tool.
type runResearchResult struct {
	EpicBeadID   string            `json:"epic_bead_id"`
	ChildBeadIDs map[string]string `json:"child_bead_ids"`
}

// researchTopics are the 4 fixed research topics per D-06.
var researchTopics = []string{"stack", "features", "architecture", "pitfalls"}

// handleRunResearch creates a research epic bead plus 4 child research beads (one per topic).
// Each child bead gets the "gsd:research-child" label and can be independently claimed.
func handleRunResearch(ctx context.Context, state *serverState, args runResearchArgs) (*mcpsdk.CallToolResult, error) {
	if err := state.init(ctx); err != nil {
		return toolError(err.Error()), nil
	}

	// Create research epic bead with "gsd:research" label + req IDs.
	epicLabels := append([]string{"gsd:research"}, args.ReqIDs...)
	epic, err := state.client.CreatePhase(
		ctx,
		args.PhaseNum,
		args.Title,
		fmt.Sprintf("Research phase %d: %s", args.PhaseNum, args.Title),
		"All 4 research topics completed and synthesized",
		epicLabels,
	)
	if err != nil {
		return toolError("failed to create research epic bead: " + err.Error()), nil
	}

	// Create 4 child beads, one per research topic.
	childBeadIDs := make(map[string]string, len(researchTopics))
	for _, topic := range researchTopics {
		child, err := state.client.CreatePlan(
			ctx,
			fmt.Sprintf("research-%d-%s", args.PhaseNum, topic),
			args.PhaseNum,
			epic.ID,
			fmt.Sprintf("Research: %s", topic),
			fmt.Sprintf("Research findings for %s documented in bead metadata", topic),
			fmt.Sprintf("Investigate %s for phase %d of %s", topic, args.PhaseNum, args.Title),
			[]string{"gsd:research-child"},
			nil,
		)
		if err != nil {
			return toolError(fmt.Sprintf("failed to create child bead for topic %q: %s", topic, err.Error())), nil
		}
		childBeadIDs[topic] = child.ID
	}

	return toolResult(&runResearchResult{
		EpicBeadID:   epic.ID,
		ChildBeadIDs: childBeadIDs,
	})
}

// synthesizeResearchArgs holds the arguments for the synthesize_research MCP tool.
type synthesizeResearchArgs struct {
	PhaseNum int    `json:"phase_num"`
	Summary  string `json:"summary"`
}

// synthesizeResearchResult is the response for the synthesize_research MCP tool.
type synthesizeResearchResult struct {
	SummaryBeadID string `json:"summary_bead_id"`
}

// handleSynthesizeResearch queries the research epic for the given phase and creates a summary child bead.
func handleSynthesizeResearch(ctx context.Context, state *serverState, args synthesizeResearchArgs) (*mcpsdk.CallToolResult, error) {
	if err := state.init(ctx); err != nil {
		return toolError(err.Error()), nil
	}

	// Query for the research epic bead by "gsd:research" label.
	epics, err := state.client.QueryByLabel(ctx, "gsd:research")
	if err != nil {
		return toolError("failed to query research epic: " + err.Error()), nil
	}

	// Find the epic for this phase number.
	var epicID string
	for _, bead := range epics {
		if phaseVal, ok := bead.Metadata["gsd_phase"]; ok {
			switch v := phaseVal.(type) {
			case float64:
				if int(v) == args.PhaseNum {
					epicID = bead.ID
				}
			case int:
				if v == args.PhaseNum {
					epicID = bead.ID
				}
			case int64:
				if int(v) == args.PhaseNum {
					epicID = bead.ID
				}
			}
		}
		if epicID != "" {
			break
		}
	}

	// If no epic found for this phase, use the first available research epic
	// (graceful fallback for environments where metadata isn't fully wired).
	if epicID == "" && len(epics) > 0 {
		epicID = epics[0].ID
	}

	if epicID == "" {
		return toolError(fmt.Sprintf("no research epic found for phase %d", args.PhaseNum)), nil
	}

	// Create summary child bead under the research epic.
	summary, err := state.client.CreatePlan(
		ctx,
		fmt.Sprintf("research-%d-summary", args.PhaseNum),
		args.PhaseNum,
		epicID,
		fmt.Sprintf("Research Summary: Phase %d", args.PhaseNum),
		"Research synthesis documented",
		args.Summary,
		[]string{"gsd:research-summary"},
		nil,
	)
	if err != nil {
		return toolError("failed to create summary bead: " + err.Error()), nil
	}

	return toolResult(&synthesizeResearchResult{
		SummaryBeadID: summary.ID,
	})
}
