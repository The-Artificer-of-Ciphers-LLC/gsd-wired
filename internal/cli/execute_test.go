package cli

import (
	"strings"
	"testing"
)

// TestRootCmdHasExecute verifies that NewRootCmd registers the "execute" subcommand.
func TestRootCmdHasExecute(t *testing.T) {
	root := NewRootCmd()
	for _, cmd := range root.Commands() {
		if cmd.Use == "execute" {
			return // found
		}
	}
	t.Error("root command missing 'execute' subcommand")
}

// TestExecuteCmdOutput verifies that running the execute subcommand returns an error
// containing "/gsd-wired:execute" to redirect users to the SKILL.md flow.
func TestExecuteCmdOutput(t *testing.T) {
	cmd := NewExecuteCmd()
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error from execute stub")
	}
	if !strings.Contains(err.Error(), "/gsd-wired:execute") {
		t.Errorf("error should mention /gsd-wired:execute, got: %s", err.Error())
	}
}
