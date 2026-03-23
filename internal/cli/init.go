package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

// projectMDTemplate is the minimal PROJECT.md template written by gsdw init.
// Fields are intentionally left as placeholders — the developer fills them in,
// or the /gsd-wired:init slash command populates them via the init_project MCP tool.
const projectMDTemplate = `# Project Name

## What
(describe what you are building)

## Why
(describe the motivation and problem this solves)

## Who
(describe the target users and audience)

## Done Criteria
(describe what success looks like — how do you know when it is done?)

## Tech Stack
(languages, frameworks, infrastructure)

## Constraints
(time, budget, team size, platform requirements)

## Risks
(technical, market, dependency risks)

---
*Initialized: %s*
`

// gsdwConfigJSON is the .gsdw/config.json structure for CLI-initialized projects.
type gsdwConfig struct {
	ProjectName string `json:"project_name"`
	Initialized string `json:"initialized"`
	Mode        string `json:"mode"`
}

// NewInitCmd creates the "gsdw init" subcommand.
// This is for direct CLI use. The /gsd-wired:init slash command uses the init_project
// MCP tool instead for deep questioning and bead creation.
func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize beads directory and project files",
		Long: `Initialize a new gsd-wired project by creating the .beads/ directory and writing
PROJECT.md and .gsdw/config.json template files.

This is for direct CLI use. For the interactive guided questioning flow, use the
/gsd-wired:init slash command which asks questions one at a time and calls the
init_project MCP tool to create project context in the beads graph.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("cannot determine working directory: %w", err)
			}

			// Step 1: Initialize .beads/ directory via bd init, if not already present.
			beadsPath := filepath.Join(cwd, ".beads")
			if _, statErr := os.Stat(beadsPath); os.IsNotExist(statErr) {
				bdPath, lookErr := exec.LookPath("bd")
				if lookErr != nil {
					fmt.Fprintln(cmd.OutOrStdout(), "bd not found on PATH — skipping .beads/ init (install beads to enable graph storage)")
				} else {
					bdCmd := exec.Command(bdPath, "init", "--backend", "dolt", "--quiet", "--skip-hooks", "--skip-agents")
					bdCmd.Env = append(os.Environ(), "BEADS_DIR="+cwd)
					bdCmd.Stdout = cmd.OutOrStdout()
					bdCmd.Stderr = cmd.ErrOrStderr()
					if runErr := bdCmd.Run(); runErr != nil {
						return fmt.Errorf("bd init failed: %w", runErr)
					}
					fmt.Fprintln(cmd.OutOrStdout(), "Initialized .beads/ directory")
				}
			}

			// Step 2: Write PROJECT.md template if it doesn't exist.
			projectMDPath := filepath.Join(cwd, "PROJECT.md")
			if _, statErr := os.Stat(projectMDPath); os.IsNotExist(statErr) {
				timestamp := time.Now().UTC().Format("2006-01-02")
				content := fmt.Sprintf(projectMDTemplate, timestamp)
				if writeErr := os.WriteFile(projectMDPath, []byte(content), 0o644); writeErr != nil {
					return fmt.Errorf("cannot write PROJECT.md: %w", writeErr)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Created PROJECT.md")
			}

			// Step 3: Create .gsdw/ directory and write config.json if it doesn't exist.
			gsdwDir := filepath.Join(cwd, ".gsdw")
			if mkdirErr := os.MkdirAll(gsdwDir, 0o755); mkdirErr != nil {
				return fmt.Errorf("cannot create .gsdw/ directory: %w", mkdirErr)
			}

			configPath := filepath.Join(gsdwDir, "config.json")
			if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
				cfg := gsdwConfig{
					ProjectName: "",
					Initialized: time.Now().UTC().Format(time.RFC3339),
					Mode:        "cli",
				}
				data, marshalErr := json.MarshalIndent(cfg, "", "  ")
				if marshalErr != nil {
					return fmt.Errorf("cannot marshal config: %w", marshalErr)
				}
				if writeErr := os.WriteFile(configPath, data, 0o644); writeErr != nil {
					return fmt.Errorf("cannot write .gsdw/config.json: %w", writeErr)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Created .gsdw/config.json")
			}

			// Step 4: Scaffold Claude Code plugin files so /gsd-wired:* slash commands appear.
			if err := scaffoldPluginFiles(cwd, cmd); err != nil {
				return err
			}

			return nil
		},
	}
	return cmd
}

// scaffoldPluginFiles writes .claude-plugin/plugin.json, .mcp.json, and skills/*/SKILL.md
// into the project directory so that Claude Code discovers the gsd-wired plugin and
// registers the /gsd-wired:* slash commands. Files are only written if they don't already exist.
func scaffoldPluginFiles(cwd string, cmd *cobra.Command) error {
	out := cmd.OutOrStdout()

	// .claude-plugin/plugin.json
	pluginDir := filepath.Join(cwd, ".claude-plugin")
	pluginPath := filepath.Join(pluginDir, "plugin.json")
	if _, statErr := os.Stat(pluginPath); os.IsNotExist(statErr) {
		if mkErr := os.MkdirAll(pluginDir, 0o755); mkErr != nil {
			return fmt.Errorf("cannot create .claude-plugin/ directory: %w", mkErr)
		}
		if writeErr := os.WriteFile(pluginPath, []byte(pluginJSON), 0o644); writeErr != nil {
			return fmt.Errorf("cannot write .claude-plugin/plugin.json: %w", writeErr)
		}
		fmt.Fprintln(out, "Created .claude-plugin/plugin.json")
	}

	// .mcp.json
	mcpPath := filepath.Join(cwd, ".mcp.json")
	if _, statErr := os.Stat(mcpPath); os.IsNotExist(statErr) {
		if writeErr := os.WriteFile(mcpPath, []byte(mcpJSON), 0o644); writeErr != nil {
			return fmt.Errorf("cannot write .mcp.json: %w", writeErr)
		}
		fmt.Fprintln(out, "Created .mcp.json")
	}

	// skills/*/SKILL.md
	skillsCreated := 0
	for relPath, content := range skillFiles {
		absPath := filepath.Join(cwd, relPath)
		if _, statErr := os.Stat(absPath); os.IsNotExist(statErr) {
			dir := filepath.Dir(absPath)
			if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
				return fmt.Errorf("cannot create directory %s: %w", dir, mkErr)
			}
			if writeErr := os.WriteFile(absPath, []byte(content), 0o644); writeErr != nil {
				return fmt.Errorf("cannot write %s: %w", relPath, writeErr)
			}
			skillsCreated++
		}
	}
	if skillsCreated > 0 {
		fmt.Fprintf(out, "Created skills/ (%d slash commands)\n", skillsCreated)
	}

	return nil
}
