package cli

import (
	"strings"
	"testing"
)

// TestRootCmdHasResearch verifies that NewRootCmd registers the "research" subcommand.
func TestRootCmdHasResearch(t *testing.T) {
	root := NewRootCmd()
	for _, cmd := range root.Commands() {
		if cmd.Use == "research" {
			return // found
		}
	}
	t.Errorf("expected 'research' subcommand registered in root, but it was not found")
}

// TestResearchCmdOutput verifies that running the research subcommand returns an error
// containing "slash command" to redirect users to the SKILL.md flow.
func TestResearchCmdOutput(t *testing.T) {
	cmd := NewResearchCmd()
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected research command to return an error, got nil")
	}
	if !strings.Contains(err.Error(), "slash command") {
		t.Errorf("research command error should mention 'slash command', got: %q", err.Error())
	}
}
