package mcp

import (
	"context"
	"log/slog"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"
)

// completedPhaseInfo holds ship-specific data about a completed phase.
type completedPhaseInfo struct {
	PhaseNum    int        `json:"phase_num"`
	Title       string     `json:"title"`
	CloseReason string     `json:"close_reason"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
}

// statusResult is the response for the get_status MCP tool.
type statusResult struct {
	ProjectName     string               `json:"project_name"`
	CurrentPhase    *phaseInfo           `json:"current_phase"`
	ReadyTasks      []taskInfo           `json:"ready_tasks"`
	TotalPhases     int                  `json:"total_phases"`
	OpenPhases      int                  `json:"open_phases"`
	CompletedPhases []completedPhaseInfo `json:"completed_phases"`
}

// phaseInfo holds information about a phase bead.
type phaseInfo struct {
	BeadID   string `json:"bead_id"`
	Title    string `json:"title"`
	PhaseNum int    `json:"phase_num"`
	Status   string `json:"status"`
	Goal     string `json:"goal"`
}

// taskInfo holds information about a ready task bead.
type taskInfo struct {
	BeadID string `json:"bead_id"`
	Title  string `json:"title"`
	PlanID string `json:"plan_id"`
}

// handleGetStatus queries the beads graph and returns a structured status dashboard.
// Follows buildSessionContext's degradation pattern: if any query fails, logs slog.Warn
// and continues with partial data. Never returns IsError=true for query failures.
func handleGetStatus(ctx context.Context, state *serverState) (*mcpsdk.CallToolResult, error) {
	if err := state.init(ctx); err != nil {
		return toolError(err.Error()), nil
	}

	result := &statusResult{
		ReadyTasks:      []taskInfo{},           // initialize to empty slice (not nil) for clean JSON
		CompletedPhases: []completedPhaseInfo{}, // initialize to empty slice (not nil) for clean JSON
	}

	// Query project bead for the project name.
	projectBeads, err := state.client.QueryByLabel(ctx, "gsd:project")
	if err != nil {
		slog.Warn("get_status: failed to query project beads", "err", err)
	} else if len(projectBeads) > 0 {
		result.ProjectName = projectBeads[0].Title
	}

	// Query phase beads to find the current phase (highest open phase number).
	phaseBeads, err := state.client.QueryByLabel(ctx, "gsd:phase")
	if err != nil {
		slog.Warn("get_status: failed to query phase beads", "err", err)
	} else {
		result.TotalPhases = len(phaseBeads)
		var currentPhase *graph.Bead
		var currentPhaseNum float64 = -1

		for i := range phaseBeads {
			b := &phaseBeads[i]
			if b.Status == "open" {
				result.OpenPhases++
			}
			// Collect completed (non-open) phases for ship-specific enrichment (D-08).
			if b.Status != "open" {
				var phaseNum int
				if b.Metadata != nil {
					if pn, ok := phaseNumFromMetadata(b.Metadata["gsd_phase"]); ok {
						phaseNum = int(pn)
					}
				}
				cpi := completedPhaseInfo{
					PhaseNum:    phaseNum,
					Title:       b.Title,
					CloseReason: b.CloseReason,
					ClosedAt:    b.ClosedAt,
				}
				result.CompletedPhases = append(result.CompletedPhases, cpi)
				continue
			}
			if b.Metadata == nil {
				continue
			}
			phaseNum, ok := phaseNumFromMetadata(b.Metadata["gsd_phase"])
			if !ok {
				continue
			}
			if currentPhase == nil || phaseNum > currentPhaseNum {
				currentPhase = b
				currentPhaseNum = phaseNum
			}
		}

		if currentPhase != nil {
			phaseNum, _ := phaseNumFromMetadata(currentPhase.Metadata["gsd_phase"])
			result.CurrentPhase = &phaseInfo{
				BeadID:   currentPhase.ID,
				Title:    currentPhase.Title,
				PhaseNum: int(phaseNum),
				Status:   currentPhase.Status,
				Goal:     currentPhase.Description,
			}
		}
	}

	// Query ready tasks.
	readyBeads, err := state.client.ListReady(ctx)
	if err != nil {
		slog.Warn("get_status: failed to list ready beads", "err", err)
	} else {
		for _, b := range readyBeads {
			planID := ""
			if b.Metadata != nil {
				if pid, ok := b.Metadata["gsd_plan"].(string); ok {
					planID = pid
				}
			}
			result.ReadyTasks = append(result.ReadyTasks, taskInfo{
				BeadID: b.ID,
				Title:  b.Title,
				PlanID: planID,
			})
		}
	}

	return toolResult(result)
}

// phaseNumFromMetadata extracts a phase number as float64 from bead metadata.
// Handles both float64 (JSON unmarshal) and int variants (direct construction in tests).
func phaseNumFromMetadata(v any) (float64, bool) {
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
