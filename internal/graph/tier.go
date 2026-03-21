package graph

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Tier represents a context tier assignment for a bead.
// Tier constants drive how much context is included for a bead in token-aware routing.
type Tier string

const (
	// TierHot is assigned to active beads (open, in_progress).
	// Hot beads receive full context: description, acceptance criteria, notes, metadata.
	TierHot Tier = "hot"
	// TierWarm is assigned to recently closed beads (count-based: last N closed).
	// Warm beads receive summary context: title + close reason (~50 tokens).
	TierWarm Tier = "warm"
	// TierCold is assigned to old closed beads.
	// Cold beads receive minimal context: ID + title only (~10 tokens).
	TierCold Tier = "cold"
)

// TieredBead wraps a Bead with its computed tier and a pre-rendered compact summary.
// Hot beads: Compact == "" (caller uses full fields).
// Warm beads: Compact == "<title>: <close_reason>" (~50 tokens).
// Cold beads: Compact == "<id> <title>" (~10 tokens).
type TieredBead struct {
	Bead
	Tier    Tier   `json:"tier"`
	Compact string `json:"compact,omitempty"`
}

// classifyTier assigns a tier based on bead status and warm set membership.
// Uses count-based warm classification (Research Open Question 1 recommendation):
//   - open or in_progress → hot
//   - closed and ID is in warmIDs → warm
//   - closed otherwise (not in warmIDs, or nil ClosedAt) → cold
//
// warmIDs is the set of bead IDs to treat as warm (usually last N closed by ClosedAt desc).
// Pass nil or empty map to treat all closed beads as cold.
func classifyTier(b Bead, warmIDs map[string]bool) Tier {
	if b.Status == "open" || b.Status == "in_progress" {
		return TierHot
	}
	if warmIDs[b.ID] {
		return TierWarm
	}
	return TierCold
}

// estimateTokens returns a conservative token estimate for a string.
// Uses the 1 token ≈ 4 bytes heuristic (D-07).
// Result is always >= 1 for non-empty strings to prevent undercount.
func estimateTokens(s string) int {
	if len(s) == 0 {
		return 0
	}
	n := len(s) / 4
	if n == 0 {
		return 1
	}
	return n
}

// compactSummary returns the summary text for compaction.
// Format: "<title>: <close_reason>" if close_reason is set, else just "<title>".
func compactSummary(b *Bead) string {
	if b.CloseReason != "" {
		return b.Title + ": " + b.CloseReason
	}
	return b.Title
}

// formatHot returns full bead context for hot-tier beads.
// Format: "## [HOT] {title} ({id})\n" with Goal and Done-when lines if present.
func formatHot(b TieredBead) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "## [HOT] %s (%s)\n", b.Title, b.ID)
	if b.Description != "" {
		fmt.Fprintf(&sb, "**Goal:** %s\n", b.Description)
	}
	if b.AcceptanceCriteria != "" {
		fmt.Fprintf(&sb, "**Done when:** %s\n", b.AcceptanceCriteria)
	}
	return sb.String()
}

// formatWarm returns summary context for warm-tier beads (~50 tokens).
// Format: "- [WARM] {title}: {close_reason}\n"
func formatWarm(b TieredBead) string {
	return fmt.Sprintf("- [WARM] %s: %s\n", b.Title, b.CloseReason)
}

// formatCold returns ID + title only for cold-tier beads (~10 tokens).
// Format: "- [{id}] {title}\n"
func formatCold(b TieredBead) string {
	return fmt.Sprintf("- [%s] %s\n", b.ID, b.Title)
}

// CompactBead writes the compact summary to bead metadata under the "gsd:compact" key.
// Best-effort: errors are returned but callers (ClosePlan) should log and continue.
// Per Research Pattern 5 and D-12.
func (c *Client) CompactBead(ctx context.Context, beadID string, summary string) (*Bead, error) {
	meta := map[string]any{"gsd:compact": summary}
	return c.UpdateBeadMetadata(ctx, beadID, meta)
}

// QueryTiered queries all beads with the given label, classifies them into hot/warm/cold tiers,
// and returns pre-rendered TieredBead slices. Uses count-based warm classification:
// the warmCount most recently closed beads (by ClosedAt desc) are warm; all others are cold.
//
// Compact field is populated for warm and cold beads:
//   - Uses Metadata["gsd:compact"] if present (write-time compaction from ClosePlan)
//   - Falls back to compactSummary (title + close_reason) for beads not yet compacted
//
// Per Research Pattern 2 and Open Question 1 (count-based warm, default 5).
func (c *Client) QueryTiered(ctx context.Context, label string, warmCount int) (hot, warm, cold []TieredBead, err error) {
	beads, err := c.QueryByLabel(ctx, label)
	if err != nil {
		return nil, nil, nil, err
	}

	// Separate open and closed beads.
	var openBeads []Bead
	var closedBeads []Bead
	for _, b := range beads {
		if b.Status == "open" || b.Status == "in_progress" {
			openBeads = append(openBeads, b)
		} else {
			closedBeads = append(closedBeads, b)
		}
	}

	// Sort closed beads by ClosedAt descending (most recent first).
	sort.Slice(closedBeads, func(i, j int) bool {
		ti := closedBeads[i].ClosedAt
		tj := closedBeads[j].ClosedAt
		if ti == nil && tj == nil {
			return false
		}
		if ti == nil {
			return false // nil ClosedAt sorts last
		}
		if tj == nil {
			return true
		}
		return ti.After(*tj)
	})

	// Build warm set: first warmCount closed beads by recency.
	warmIDs := make(map[string]bool)
	for i, b := range closedBeads {
		if i >= warmCount {
			break
		}
		warmIDs[b.ID] = true
	}

	// All open beads are hot.
	for _, b := range openBeads {
		hot = append(hot, TieredBead{Bead: b, Tier: TierHot})
	}

	// Classify closed beads using warmIDs.
	for _, b := range closedBeads {
		tier := classifyTier(b, warmIDs)
		tb := TieredBead{Bead: b, Tier: tier}

		// Populate Compact field for warm/cold from metadata or fallback.
		compact := extractCompactFromBead(b)
		tb.Compact = compact

		switch tier {
		case TierWarm:
			warm = append(warm, tb)
		default:
			cold = append(cold, tb)
		}
	}

	return hot, warm, cold, nil
}

// extractCompactFromBead returns the compact summary from metadata["gsd:compact"] if present,
// falling back to compactSummary (title + close_reason).
func extractCompactFromBead(b Bead) string {
	if b.Metadata != nil {
		if v, ok := b.Metadata["gsd:compact"]; ok {
			if s, ok := v.(string); ok && s != "" {
				return s
			}
		}
	}
	return compactSummary(&b)
}
