package cli

import (
	"strings"
	"testing"
)

// TestRootCmdHasVerify verifies that NewRootCmd registers the "verify" subcommand.
func TestRootCmdHasVerify(t *testing.T) {
	root := NewRootCmd()
	for _, cmd := range root.Commands() {
		if cmd.Use == "verify" {
			return // found
		}
	}
	t.Error("root command missing 'verify' subcommand")
}

// TestVerifyCmdOutput verifies that running the verify subcommand returns an error
// containing "/gsd-wired:verify" to redirect users to the SKILL.md flow.
func TestVerifyCmdOutput(t *testing.T) {
	cmd := NewVerifyCmd()
	err := cmd.RunE(cmd, nil)
	if err == nil {
		t.Fatal("expected error from verify stub")
	}
	if !strings.Contains(err.Error(), "/gsd-wired:verify") {
		t.Errorf("error should mention /gsd-wired:verify, got: %s", err.Error())
	}
}
