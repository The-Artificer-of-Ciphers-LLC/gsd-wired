package cli

import (
	"strings"
	"testing"
)

// TestRootCmdHasPlan verifies that NewRootCmd registers the "plan" subcommand.
func TestRootCmdHasPlan(t *testing.T) {
	root := NewRootCmd()
	for _, cmd := range root.Commands() {
		if cmd.Use == "plan" {
			return // found
		}
	}
	t.Errorf("expected 'plan' subcommand registered in root, but it was not found")
}

// TestPlanCmdOutput verifies that running the plan subcommand returns an error
// containing "slash command" to redirect users to the SKILL.md flow.
func TestPlanCmdOutput(t *testing.T) {
	cmd := NewPlanCmd()
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected plan command to return an error, got nil")
	}
	if !strings.Contains(err.Error(), "slash command") {
		t.Errorf("plan command error should mention 'slash command', got: %q", err.Error())
	}
}
