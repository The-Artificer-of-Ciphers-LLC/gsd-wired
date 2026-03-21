package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// initProjectArgs holds the arguments for the init_project MCP tool.
type initProjectArgs struct {
	ProjectName  string `json:"project_name"`
	What         string `json:"what"`
	Why          string `json:"why"`
	Who          string `json:"who,omitempty"`
	DoneCriteria string `json:"done_criteria"`
	TechStack    string `json:"tech_stack,omitempty"`
	Constraints  string `json:"constraints,omitempty"`
	Risks        string `json:"risks,omitempty"`
	Mode         string `json:"mode"`
	PRURL        string `json:"pr_url,omitempty"`
}

// initProjectResult is the response for the init_project MCP tool.
type initProjectResult struct {
	ProjectBeadID  string   `json:"project_bead_id"`
	ContextBeadIDs []string `json:"context_bead_ids"`
	FilesWritten   []string `json:"files_written"`
}

// handleInitProject creates the project epic bead, context child beads, and writes PROJECT.md + config.json.
func handleInitProject(ctx context.Context, state *serverState, args initProjectArgs) (*mcpsdk.CallToolResult, error) {
	// Trigger bd init if .beads/ is absent (INIT-05).
	if err := state.init(ctx); err != nil {
		return toolError(err.Error()), nil
	}

	// Create project epic bead using phaseNum=0 (convention: project-level, not a real phase).
	projectBead, err := state.client.CreatePhase(ctx, 0, args.ProjectName, args.What, args.DoneCriteria, []string{"gsd:project"})
	if err != nil {
		return toolError("failed to create project bead: " + err.Error()), nil
	}

	// Create context child beads for each non-empty category (D-06: category beads).
	var contextBeadIDs []string

	type contextBead struct {
		planID  string
		title   string
		context string
	}

	contextBeads := []contextBead{
		{"init-done-criteria", "Done Criteria", args.DoneCriteria},
		{"init-decisions", "Constraints", args.Constraints},
		{"init-tech", "Tech Stack", args.TechStack},
		{"init-risks", "Risks", args.Risks},
	}

	for _, cb := range contextBeads {
		if cb.context == "" {
			continue
		}
		bead, err := state.client.CreatePlan(
			ctx,
			cb.planID,
			0, // phase 0 = project level
			projectBead.ID,
			cb.title,
			"", // no acceptance criteria for context beads
			cb.context,
			nil,
			nil,
		)
		if err != nil {
			// Non-fatal: log and continue (partial context is acceptable).
			continue
		}
		contextBeadIDs = append(contextBeadIDs, bead.ID)
	}

	// Write PROJECT.md to state.beadsDir (D-07: human-readable, parallel to beads).
	projectMDPath := filepath.Join(state.beadsDir, "PROJECT.md")
	projectMDContent := buildProjectMD(args)
	if err := os.WriteFile(projectMDPath, []byte(projectMDContent), 0644); err != nil {
		return toolError("failed to write PROJECT.md: " + err.Error()), nil
	}

	// Create .gsdw/ directory and write config.json.
	gsdwDir := filepath.Join(state.beadsDir, ".gsdw")
	if err := os.MkdirAll(gsdwDir, 0755); err != nil {
		return toolError("failed to create .gsdw/ directory: " + err.Error()), nil
	}
	configPath := filepath.Join(gsdwDir, "config.json")
	configData, err := json.Marshal(map[string]string{
		"project_name": args.ProjectName,
		"initialized":  time.Now().UTC().Format(time.RFC3339),
		"mode":         args.Mode,
	})
	if err != nil {
		return toolError("failed to marshal config.json: " + err.Error()), nil
	}
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return toolError("failed to write .gsdw/config.json: " + err.Error()), nil
	}

	return toolResult(&initProjectResult{
		ProjectBeadID:  projectBead.ID,
		ContextBeadIDs: contextBeadIDs,
		FilesWritten:   []string{projectMDPath, configPath},
	})
}

// buildProjectMD constructs the PROJECT.md content from init args.
// Sections with empty values are skipped (per D-05: quick mode omits optional fields).
func buildProjectMD(args initProjectArgs) string {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	var out string

	out += fmt.Sprintf("# %s\n", args.ProjectName)

	if args.What != "" {
		out += fmt.Sprintf("\n## What\n%s\n", args.What)
	}
	if args.Why != "" {
		out += fmt.Sprintf("\n## Why\n%s\n", args.Why)
	}
	if args.Who != "" {
		out += fmt.Sprintf("\n## Who\n%s\n", args.Who)
	}
	if args.DoneCriteria != "" {
		out += fmt.Sprintf("\n## Done Criteria\n%s\n", args.DoneCriteria)
	}
	if args.TechStack != "" {
		out += fmt.Sprintf("\n## Tech Stack\n%s\n", args.TechStack)
	}
	if args.Constraints != "" {
		out += fmt.Sprintf("\n## Constraints\n%s\n", args.Constraints)
	}
	if args.Risks != "" {
		out += fmt.Sprintf("\n## Risks\n%s\n", args.Risks)
	}

	out += fmt.Sprintf("\n---\n*Initialized: %s*\n*Mode: %s*\n", timestamp, args.Mode)

	return out
}
