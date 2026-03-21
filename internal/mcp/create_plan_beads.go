package mcp

import (
	"context"
	"fmt"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// planTask represents a single task in a plan, matching the research JSON schema.
type planTask struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Acceptance string   `json:"acceptance"`
	Context    string   `json:"context"`
	ReqIDs     []string `json:"req_ids"`
	DependsOn  []string `json:"depends_on"`
	Complexity string   `json:"complexity"`
	Files      []string `json:"files"`
}

// createPlanBeadsArgs holds the arguments for the create_plan_beads MCP tool.
type createPlanBeadsArgs struct {
	PhaseNum   int        `json:"phase_num"`
	EpicBeadID string     `json:"epic_bead_id"`
	Tasks      []planTask `json:"tasks"`
}

// createPlanBeadsResult is the response for the create_plan_beads MCP tool.
type createPlanBeadsResult struct {
	TaskBeadIDs map[string]string `json:"task_bead_ids"`
}

// handleCreatePlanBeads batch-creates task beads from a structured plan in topological order.
// Local task IDs (e.g. "06-01") are resolved to actual bead IDs for dependency wiring.
// Per Pitfall 3: uses iterative topological sort, not recursive.
func handleCreatePlanBeads(ctx context.Context, state *serverState, args createPlanBeadsArgs) (*mcpsdk.CallToolResult, error) {
	if err := state.init(ctx); err != nil {
		return toolError(err.Error()), nil
	}

	if args.EpicBeadID == "" {
		return toolError("epic_bead_id is required"), nil
	}

	// Build adjacency: task ID -> list of task IDs it depends on.
	// Also build a task lookup map.
	taskMap := make(map[string]planTask, len(args.Tasks))
	for _, task := range args.Tasks {
		taskMap[task.ID] = task
	}

	// Iterative topological sort.
	// processed tracks which local IDs have been created.
	localToBead := make(map[string]string, len(args.Tasks))
	processed := make(map[string]bool, len(args.Tasks))
	taskBeadIDs := make(map[string]string, len(args.Tasks))

	// Keep iterating until all tasks are processed.
	// Each pass processes tasks whose dependencies are all resolved.
	remaining := make([]planTask, len(args.Tasks))
	copy(remaining, args.Tasks)

	for len(remaining) > 0 {
		progress := false
		nextRemaining := remaining[:0]

		for _, task := range remaining {
			// Check if all dependencies are resolved.
			allResolved := true
			for _, depID := range task.DependsOn {
				if depID != "" && !processed[depID] {
					allResolved = false
					break
				}
			}

			if !allResolved {
				nextRemaining = append(nextRemaining, task)
				continue
			}

			// Resolve dep bead IDs from local IDs.
			var depBeadIDs []string
			for _, depID := range task.DependsOn {
				if depID == "" {
					continue
				}
				beadID, ok := localToBead[depID]
				if !ok {
					return toolError(fmt.Sprintf("dependency %q not found for task %q", depID, task.ID)), nil
				}
				depBeadIDs = append(depBeadIDs, beadID)
			}

			// Build metadata with complexity, files, and gsd_plan.
			// We pass these as req_ids-style via the planID argument to embed gsd_plan,
			// and rely on CreatePlan's metadata field for complexity + files.
			// Extend req IDs to include gsd labels.
			reqIDs := task.ReqIDs

			// Create the task bead via graph.Client.CreatePlan.
			bead, err := state.client.CreatePlanWithMeta(ctx,
				task.ID,
				args.PhaseNum,
				args.EpicBeadID,
				task.Title,
				task.Acceptance,
				task.Context,
				task.Complexity,
				task.Files,
				reqIDs,
				depBeadIDs,
			)
			if err != nil {
				return toolError(fmt.Sprintf("failed to create task bead %q: %s", task.ID, err.Error())), nil
			}

			localToBead[task.ID] = bead.ID
			taskBeadIDs[task.ID] = bead.ID
			processed[task.ID] = true
			progress = true
		}

		remaining = nextRemaining

		if !progress && len(remaining) > 0 {
			// Circular dependency or unresolvable dependency.
			ids := make([]string, 0, len(remaining))
			for _, t := range remaining {
				ids = append(ids, t.ID)
			}
			return toolError(fmt.Sprintf("circular or unresolvable dependencies detected in tasks: %v", ids)), nil
		}
	}

	return toolResult(&createPlanBeadsResult{
		TaskBeadIDs: taskBeadIDs,
	})
}
