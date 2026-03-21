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
// PROJECT.md and .gsdw/config.json template files.
// Note: gsdw init will skip the bd init step if .beads/ is already absent and bd is not
// on PATH in test environments — the test only verifies file creation behavior.
func TestInitCmdWritesFiles(t *testing.T) {
	// Set up a temp directory to act as the project root.
	tmpDir := t.TempDir()

	// Restore cwd after test.
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

	// Run init command — skip-bd mode via env so no real bd is required.
	// The init cmd will detect no bd on PATH and skip bd init gracefully.
	cmd := NewInitCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Execute in the temp dir; bd init will fail since bd may not be on PATH
	// but file creation should still proceed.
	err = cmd.Execute()
	// We accept both nil error (bd found) or an error only if it's about bd.
	// The file creation is what we test.
	if err != nil && !strings.Contains(err.Error(), "bd") && !strings.Contains(err.Error(), "beads") {
		t.Fatalf("init command failed with unexpected error: %v", err)
	}

	// PROJECT.md must exist.
	projectMD := filepath.Join(tmpDir, "PROJECT.md")
	if _, statErr := os.Stat(projectMD); statErr != nil {
		t.Errorf("expected PROJECT.md to exist at %s, got: %v", projectMD, statErr)
	}

	// .gsdw/config.json must exist.
	configJSON := filepath.Join(tmpDir, ".gsdw", "config.json")
	if _, statErr := os.Stat(configJSON); statErr != nil {
		t.Errorf("expected .gsdw/config.json to exist at %s, got: %v", configJSON, statErr)
	}
}
