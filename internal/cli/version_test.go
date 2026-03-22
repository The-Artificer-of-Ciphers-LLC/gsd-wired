package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestVersionCmdRegistered verifies the version subcommand is registered on root.
func TestVersionCmdRegistered(t *testing.T) {
	root := NewRootCmd()
	for _, cmd := range root.Commands() {
		if cmd.Use == "version" {
			return
		}
	}
	t.Error("root command missing 'version' subcommand")
}

// TestVersionCmdWithoutJSON verifies that `gsdw version` prints a human-readable string.
func TestVersionCmdWithoutJSON(t *testing.T) {
	cmd := NewVersionCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("version command returned error: %v", err)
	}
	output := strings.TrimSpace(buf.String())
	if output == "" {
		t.Error("version command produced no output")
	}
	// Should match "VERSION (HASH)" pattern
	if !strings.Contains(output, "(") || !strings.Contains(output, ")") {
		t.Errorf("version output %q does not look like 'VERSION (HASH)' format", output)
	}
}

// TestVersionCmdWithJSON verifies that `gsdw version --json` prints valid JSON with required keys.
func TestVersionCmdWithJSON(t *testing.T) {
	cmd := NewVersionCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	if err := cmd.Flags().Set("json", "true"); err != nil {
		t.Fatalf("failed to set --json flag: %v", err)
	}
	if err := cmd.Execute(); err != nil {
		t.Fatalf("version --json returned error: %v", err)
	}
	output := buf.String()
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("version --json output is not valid JSON: %v\nOutput: %s", err, output)
	}
	requiredKeys := []string{"version", "commit", "date", "goVersion", "platform"}
	for _, k := range requiredKeys {
		if _, ok := parsed[k]; !ok {
			t.Errorf("version --json output missing key %q", k)
		}
	}
}
