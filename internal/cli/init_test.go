package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRootCmdHasInit verifies that NewRootCmd registers the "init" subcommand.
func TestRootCmdHasInit(t *testing.T) {
	root := NewRootCmd()
	for _, cmd := range root.Commands() {
		if cmd.Use == "init" {
			return // found
		}
	}
	t.Errorf("expected 'init' subcommand registered in root, but it was not found")
}

// TestInitCmdWritesFiles verifies that running gsdw init in a temp dir creates
// PROJECT.md, .gsdw/config.json, .mcp.json, hooks/hooks.json, and registers
// commands in ~/.claude/commands/gsd-wired/.
func TestInitCmdWritesFiles(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get cwd: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: could not restore cwd: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("cannot chdir to temp dir: %v", err)
	}

	cmd := NewInitCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	err = cmd.Execute()
	if err != nil && !strings.Contains(err.Error(), "bd") && !strings.Contains(err.Error(), "beads") {
		t.Fatalf("init command failed with unexpected error: %v", err)
	}

	// PROJECT.md must exist.
	if _, statErr := os.Stat(filepath.Join(tmpDir, "PROJECT.md")); statErr != nil {
		t.Errorf("expected PROJECT.md to exist, got: %v", statErr)
	}

	// .gsdw/config.json must exist.
	if _, statErr := os.Stat(filepath.Join(tmpDir, ".gsdw", "config.json")); statErr != nil {
		t.Errorf("expected .gsdw/config.json to exist, got: %v", statErr)
	}

	// .mcp.json must exist.
	if _, statErr := os.Stat(filepath.Join(tmpDir, ".mcp.json")); statErr != nil {
		t.Errorf("expected .mcp.json to exist, got: %v", statErr)
	}

	// hooks/hooks.json must exist.
	if _, statErr := os.Stat(filepath.Join(tmpDir, "hooks", "hooks.json")); statErr != nil {
		t.Errorf("expected hooks/hooks.json to exist, got: %v", statErr)
	}

	// ~/.claude/commands/gsd-wired/ must have 8 command files.
	homeDir, _ := os.UserHomeDir()
	cmdDir := filepath.Join(homeDir, ".claude", "commands", "gsd-wired")
	cmdNames := []string{"init", "status", "research", "plan", "execute", "verify", "ready", "ship"}
	for _, name := range cmdNames {
		cmdPath := filepath.Join(cmdDir, name+".md")
		if _, statErr := os.Stat(cmdPath); statErr != nil {
			t.Errorf("expected command %s.md to exist at %s, got: %v", name, cmdPath, statErr)
		}
	}

	// Verify output mentions command registration.
	output := buf.String()
	if !strings.Contains(output, "Registered") && !strings.Contains(output, "slash commands") {
		t.Errorf("expected output to mention command registration, got: %s", output)
	}
}

// TestInitCmdCommandFrontmatter verifies command files have correct frontmatter format.
func TestInitCmdCommandFrontmatter(t *testing.T) {
	homeDir, _ := os.UserHomeDir()
	cmdPath := filepath.Join(homeDir, ".claude", "commands", "gsd-wired", "init.md")
	data, err := os.ReadFile(cmdPath)
	if err != nil {
		t.Skipf("command file not found (run TestInitCmdWritesFiles first): %v", err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		t.Errorf("command file should start with --- frontmatter")
	}
	if !strings.Contains(content, "name: gsd-wired:init") {
		t.Errorf("command file should contain 'name: gsd-wired:init', got: %s", content[:200])
	}
	if !strings.Contains(content, "description:") {
		t.Errorf("command file should contain 'description:', got: %s", content[:200])
	}
}
