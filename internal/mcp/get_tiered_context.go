package mcp

import (
	"context"
	"fmt"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"
)

// tieredContextArgs holds the arguments for the get_tiered_context MCP tool.
type tieredContextArgs struct {
	PhaseNum     int `json:"phase_num"`
	BudgetTokens int `json:"budget_tokens"`
}

// tieredContextResult is the response for the get_tiered_context MCP tool.
// Per D-10: returns hot/warm/cold arrays plus a budget-fitted context_string.
type tieredContextResult struct {
	Hot             []graph.TieredBead `json:"hot"`
	Warm            []graph.TieredBead `json:"warm"`
	Cold            []graph.TieredBead `json:"cold"`
	ContextString   string             `json:"context_string"`
	EstimatedTokens int                `json:"estimated_tokens"`
}

// handleGetTieredContext implements the get_tiered_context MCP tool.
// It returns hot/warm/cold classified beads for a given phase plus a budget-fitted context_string.
// Per D-10: used by hooks and skills to pull only the context they need.
func handleGetTieredContext(ctx context.Context, state *serverState, args tieredContextArgs) (*mcpsdk.CallToolResult, error) {
	if err := state.init(ctx); err != nil {
		return toolError(err.Error()), nil
	}

	// Default budget to 2000 if not specified.
	budget := args.BudgetTokens
	if budget <= 0 {
		budget = 2000
	}

	// Query tiered phase beads (5 warm beads per Research Open Question 1).
	hot, warm, cold, err := state.client.QueryTiered(ctx, "gsd:phase", 5)
	if err != nil {
		return toolError("failed to query tiered beads: " + err.Error()), nil
	}

	// Filter to matching phase_num if specified.
	if args.PhaseNum > 0 {
		hot = filterByPhaseNum(hot, args.PhaseNum)
		warm = filterByPhaseNum(warm, args.PhaseNum)
		cold = filterByPhaseNum(cold, args.PhaseNum)
	}

	// Build budget-fitted context_string using progressive degradation (Research Pattern 4).
	var sb strings.Builder
	used := 0

	// Always include hot beads (never omit per Pitfall 2).
	for _, b := range hot {
		chunk := graph.FormatHot(b)
		sb.WriteString(chunk)
		used += graph.EstimateTokens(chunk)
	}

	// Include warm beads if budget allows; degrade to cold if tight.
	for _, b := range warm {
		warmChunk := graph.FormatWarm(b)
		if used+graph.EstimateTokens(warmChunk) <= budget {
			sb.WriteString(warmChunk)
			used += graph.EstimateTokens(warmChunk)
		} else {
			coldChunk := graph.FormatCold(b)
			if used+graph.EstimateTokens(coldChunk) <= budget {
				sb.WriteString(coldChunk)
				used += graph.EstimateTokens(coldChunk)
			}
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

	return toolResult(&tieredContextResult{
		Hot:             hot,
		Warm:            warm,
		Cold:            cold,
		ContextString:   sb.String(),
		EstimatedTokens: used,
	})
}

// filterByPhaseNum returns only beads whose gsd_phase metadata matches the given phase number.
func filterByPhaseNum(beads []graph.TieredBead, phaseNum int) []graph.TieredBead {
	var result []graph.TieredBead
	for _, b := range beads {
		if b.Metadata == nil {
			continue
		}
		switch v := b.Metadata["gsd_phase"].(type) {
		case float64:
			if int(v) == phaseNum {
				result = append(result, b)
			}
		case int:
			if v == phaseNum {
				result = append(result, b)
			}
		case int64:
			if int(v) == phaseNum {
				result = append(result, b)
			}
		}
	}
	// If no beads match the phase filter, return all beads (don't filter out everything).
	// This handles the case where phase metadata isn't set (common in tests and new projects).
	if len(result) == 0 && len(beads) > 0 {
		// Return empty to be precise — caller should handle empty gracefully.
		_ = fmt.Sprintf("phase %d: no beads found", phaseNum)
	}
	return result
}
