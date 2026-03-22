package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDoctorCmd_Registered verifies doctor subcommand is registered on root.
func TestDoctorCmd_Registered(t *testing.T) {
	root := NewRootCmd()
	for _, cmd := range root.Commands() {
		if cmd.Use == "doctor" {
			return
		}
	}
	t.Error("root command missing 'doctor' subcommand")
}

// TestRenderDoctor_DepsSection verifies doctor output includes the Dependencies section.
func TestRenderDoctor_DepsSection(t *testing.T) {
	result := makeOKResult()
	var buf bytes.Buffer
	renderDoctor(&buf, result, "", "")
	out := buf.String()

	t.Logf("doctor output:\n%s", out)

	if !strings.Contains(out, "Dependencies") {
		t.Errorf("expected 'Dependencies' section header in output, got:\n%s", out)
	}
	if !strings.Contains(out, "[OK]") {
		t.Errorf("expected '[OK]' for deps in doctor output, got:\n%s", out)
	}
}

// TestRenderDoctor_ProjectSection verifies doctor output includes the Project section.
func TestRenderDoctor_ProjectSection(t *testing.T) {
	result := makeOKResult()
	var buf bytes.Buffer
	renderDoctor(&buf, result, "", "")
	out := buf.String()

	t.Logf("doctor project section output:\n%s", out)

	if !strings.Contains(out, "Project") {
		t.Errorf("expected 'Project:' section in doctor output, got:\n%s", out)
	}
}

// TestRenderDoctor_BeadsDirFound verifies [OK] for .beads/ when it exists.
func TestRenderDoctor_BeadsDirFound(t *testing.T) {
	result := makeOKResult()
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	var buf bytes.Buffer
	renderDoctor(&buf, result, beadsDir, "")
	out := buf.String()

	t.Logf("doctor .beads found output:\n%s", out)

	if !strings.Contains(out, ".beads") {
		t.Errorf("expected '.beads' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "[OK]") {
		t.Errorf("expected '[OK]' for .beads/ found, got:\n%s", out)
	}
}

// TestRenderDoctor_BeadsDirMissing verifies [WARN] for .beads/ when not found.
func TestRenderDoctor_BeadsDirMissing(t *testing.T) {
	result := makeOKResult()
	var buf bytes.Buffer
	// Pass empty beadsDir to indicate not found
	renderDoctor(&buf, result, "", "")
	out := buf.String()

	t.Logf("doctor .beads missing output:\n%s", out)

	if !strings.Contains(out, "[WARN]") {
		t.Errorf("expected '[WARN]' for missing .beads/ in doctor output, got:\n%s", out)
	}
	if !strings.Contains(out, ".beads") {
		t.Errorf("expected '.beads' mention in doctor output, got:\n%s", out)
	}
}

// TestRenderDoctor_GsdwDirFound verifies [OK] for .gsdw/ when it exists.
func TestRenderDoctor_GsdwDirFound(t *testing.T) {
	result := makeOKResult()
	tmpDir := t.TempDir()
	gsdwDir := filepath.Join(tmpDir, ".gsdw")
	if err := os.MkdirAll(gsdwDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	var buf bytes.Buffer
	renderDoctor(&buf, result, "", gsdwDir)
	out := buf.String()

	t.Logf("doctor .gsdw found output:\n%s", out)

	if !strings.Contains(out, ".gsdw") {
		t.Errorf("expected '.gsdw' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "[OK]") {
		t.Errorf("expected '[OK]' for .gsdw/ found, got:\n%s", out)
	}
}

// TestRenderDoctor_GsdwDirMissing verifies [WARN] for .gsdw/ when not found.
func TestRenderDoctor_GsdwDirMissing(t *testing.T) {
	result := makeOKResult()
	var buf bytes.Buffer
	renderDoctor(&buf, result, "", "")
	out := buf.String()

	// Should have WARN for missing .gsdw/
	if !strings.Contains(out, "[WARN]") {
		t.Errorf("expected '[WARN]' for missing .gsdw/ in output, got:\n%s", out)
	}
	if !strings.Contains(out, "gsdw init") {
		t.Errorf("expected 'gsdw init' hint in output, got:\n%s", out)
	}
}

// TestRenderDoctor_NoFileModification verifies doctor does not create any files.
func TestRenderDoctor_NoFileModification(t *testing.T) {
	result := makeOKResult()
	tmpDir := t.TempDir()

	// Count files before
	before := countFiles(t, tmpDir)

	// Capture current dir, temporarily chdir to tmpDir
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	var buf bytes.Buffer
	renderDoctor(&buf, result, "", "")

	// Count files after
	after := countFiles(t, tmpDir)

	if after != before {
		t.Errorf("doctor created files: before=%d, after=%d", before, after)
	}
}

// countFiles counts all files (not dirs) in a directory tree.
func countFiles(t *testing.T, dir string) int {
	t.Helper()
	var count int
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}

// TestRenderDoctor_FailDepsShowInstallHelp verifies install help appears in doctor output for missing deps.
func TestRenderDoctor_FailDepsShowInstallHelp(t *testing.T) {
	result := makeFailResult()
	var buf bytes.Buffer
	renderDoctor(&buf, result, "", "")
	out := buf.String()

	t.Logf("doctor fail deps output:\n%s", out)

	if !strings.Contains(out, "[FAIL]") {
		t.Errorf("expected '[FAIL]' for missing dolt, got:\n%s", out)
	}
	if !strings.Contains(out, "brew install dolthub") {
		t.Errorf("expected install help for dolt in doctor output, got:\n%s", out)
	}
}
