package cli

import (
	"strings"
	"testing"
)

// TestRootCmdHasShip verifies that NewRootCmd registers the "ship" subcommand.
func TestRootCmdHasShip(t *testing.T) {
	root := NewRootCmd()
	for _, cmd := range root.Commands() {
		if cmd.Use == "ship" {
			return // found
		}
	}
	t.Error("root command missing 'ship' subcommand")
}

// TestShipCmdUse verifies that the ship command Use field is "ship".
func TestShipCmdUse(t *testing.T) {
	cmd := NewShipCmd()
	if cmd.Use != "ship" {
		t.Errorf("expected Use='ship', got: %s", cmd.Use)
	}
}

// TestShipCmdOutput verifies that running the ship subcommand returns an error
// containing "slash command" to redirect users to the SKILL.md flow.
func TestShipCmdOutput(t *testing.T) {
	cmd := NewShipCmd()
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error from ship stub")
	}
	if !strings.Contains(err.Error(), "slash command") {
		t.Errorf("error should mention slash command, got: %s", err.Error())
	}
}
