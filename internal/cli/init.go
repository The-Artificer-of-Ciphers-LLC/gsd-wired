package cli

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/connection"
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
			// Failure is non-fatal — plugin scaffolding must proceed regardless.
			beadsPath := filepath.Join(cwd, ".beads")
			if _, statErr := os.Stat(beadsPath); os.IsNotExist(statErr) {
				bdPath, lookErr := exec.LookPath("bd")
				if lookErr != nil {
					fmt.Fprintln(cmd.OutOrStdout(), "bd not found on PATH — skipping .beads/ init (install beads to enable graph storage)")
				} else {
					bdCmd := exec.Command(bdPath, "init", "--force", "--backend", "dolt", "--quiet", "--skip-hooks", "--skip-agents")
					bdCmd.Env = append(os.Environ(), "BEADS_DIR="+beadsPath)
					bdCmd.Stdout = cmd.OutOrStdout()
					bdCmd.Stderr = cmd.ErrOrStderr()
					if runErr := bdCmd.Run(); runErr != nil {
						// bd may create .beads/ before reporting "already initialized" — check if it exists now.
						if _, postStat := os.Stat(beadsPath); postStat == nil {
							fmt.Fprintln(cmd.OutOrStdout(), "Using existing .beads/ directory")
						} else {
							fmt.Fprintf(cmd.ErrOrStderr(), "Warning: bd init failed: %v (continuing with plugin setup)\n", runErr)
						}
					} else {
						fmt.Fprintln(cmd.OutOrStdout(), "Initialized .beads/ directory")
					}
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

			// Step 4: Auto-configure connection from the actual Dolt server port.
			// Read the real port from .beads/dolt-server.port (written by bd init),
			// then verify the server is reachable before saving.
			connPath := filepath.Join(gsdwDir, "connection.json")
			{
				port := connection.ReadServerPort(beadsPath)
				if port == "" {
					port = "3307"
				}
				// Always update connection.json to match the running server port.
				// This handles both fresh init and re-init where the port changed.
				needsUpdate := true
				if _, statErr := os.Stat(connPath); statErr == nil {
					// connection.json exists — check if port matches
					if existing, loadErr := connection.LoadConnection(gsdwDir); loadErr == nil && existing != nil {
						_, existingPort := existing.ActiveHostPort()
						if existingPort == port {
							needsUpdate = false
						}
					}
				}
				if needsUpdate {
					conn, dialErr := net.DialTimeout("tcp", "127.0.0.1:"+port, 2*time.Second)
					if dialErr == nil {
						conn.Close()
						cfg := &connection.Config{
							ActiveMode: "local",
							Local:      connection.LocalConfig{Host: "127.0.0.1", Port: connection.FlexPort(port)},
							Configured: time.Now().UTC().Format(time.RFC3339),
						}
						if saveErr := connection.SaveConnection(gsdwDir, cfg); saveErr == nil {
							fmt.Fprintf(cmd.OutOrStdout(), "Connected to local Dolt server on 127.0.0.1:%s\n", port)
						}
					}
				}
			}

			// Step 5: Scaffold Claude Code plugin files so /gsd-wired:* slash commands appear.
			if err := scaffoldPluginFiles(cwd, cmd); err != nil {
				return err
			}

			return nil
		},
	}
	return cmd
}

// scaffoldPluginFiles writes project-local files (.mcp.json, hooks/) and registers
// /gsd-wired:* slash commands in ~/.claude/commands/gsd-wired/ so Claude Code discovers them.
func scaffoldPluginFiles(cwd string, cmd *cobra.Command) error {
	out := cmd.OutOrStdout()

	// .mcp.json — project-local, tells Claude Code to start the MCP server.
	mcpPath := filepath.Join(cwd, ".mcp.json")
	if _, statErr := os.Stat(mcpPath); os.IsNotExist(statErr) {
		if writeErr := os.WriteFile(mcpPath, []byte(mcpJSON), 0o644); writeErr != nil {
			return fmt.Errorf("cannot write .mcp.json: %w", writeErr)
		}
		fmt.Fprintln(out, "Created .mcp.json")
	}

	// hooks/hooks.json — project-local hook dispatchers.
	hooksDir := filepath.Join(cwd, "hooks")
	hooksPath := filepath.Join(hooksDir, "hooks.json")
	if _, statErr := os.Stat(hooksPath); os.IsNotExist(statErr) {
		if mkErr := os.MkdirAll(hooksDir, 0o755); mkErr != nil {
			return fmt.Errorf("cannot create hooks/ directory: %w", mkErr)
		}
		if writeErr := os.WriteFile(hooksPath, []byte(hooksJSON), 0o644); writeErr != nil {
			return fmt.Errorf("cannot write hooks/hooks.json: %w", writeErr)
		}
		fmt.Fprintln(out, "Created hooks/hooks.json (4 hook dispatchers)")
	}

	// ~/.claude/commands/gsd-wired/ — global command registration for /gsd-wired:* slash commands.
	// This is how Claude Code discovers slash commands (same mechanism as GSD vanilla).
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	cmdDir := filepath.Join(homeDir, ".claude", "commands", "gsd-wired")
	if mkErr := os.MkdirAll(cmdDir, 0o755); mkErr != nil {
		return fmt.Errorf("cannot create ~/.claude/commands/gsd-wired/: %w", mkErr)
	}

	cmdsCreated := 0
	for _, def := range commandDefs {
		cmdPath := filepath.Join(cmdDir, def.name+".md")
		content := buildCommandMD(def)
		// Always overwrite — ensures commands stay in sync with installed gsdw version.
		if writeErr := os.WriteFile(cmdPath, []byte(content), 0o644); writeErr != nil {
			return fmt.Errorf("cannot write command %s: %w", def.name, writeErr)
		}
		cmdsCreated++
	}
	if cmdsCreated > 0 {
		fmt.Fprintf(out, "Registered %d /gsd-wired:* slash commands\n", cmdsCreated)
	}

	return nil
}
