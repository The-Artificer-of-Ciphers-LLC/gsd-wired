package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/deps"
)

// mockCheckResult builds a CheckResult with the given deps for test injection.
func mockCheckResult(ds []deps.Dep) deps.CheckResult {
	allOK := true
	for _, d := range ds {
		if d.Status == deps.StatusFail {
			allOK = false
			break
		}
	}
	return deps.CheckResult{Deps: ds, AllOK: allOK}
}

func allOKResult() deps.CheckResult {
	return mockCheckResult([]deps.Dep{
		{Name: "bd", Binary: "bd", Status: deps.StatusOK, Version: "1.4.2", Path: "/usr/bin/bd"},
		{Name: "dolt", Binary: "dolt", Status: deps.StatusOK, Version: "1.2.0", Path: "/usr/bin/dolt"},
		{Name: "Go", Binary: "go", Status: deps.StatusOK, Version: "1.22.4", Path: "/usr/local/go/bin/go"},
		{Name: "Container Runtime", Binary: "docker", Status: deps.StatusOK, Version: "24.0.0", Path: "/usr/bin/docker"},
	})
}

func missingBdResult() deps.CheckResult {
	return mockCheckResult([]deps.Dep{
		{Name: "bd", Binary: "bd", Status: deps.StatusFail, InstallHelp: "go install github.com/steveyegge/beads/cmd/bd@latest\n  or: brew install steveyegge/tap/beads"},
		{Name: "dolt", Binary: "dolt", Status: deps.StatusOK, Version: "1.2.0", Path: "/usr/bin/dolt"},
		{Name: "Go", Binary: "go", Status: deps.StatusOK, Version: "1.22.4", Path: "/usr/local/go/bin/go"},
		{Name: "Container Runtime", Binary: "docker", Status: deps.StatusOK, Version: "24.0.0", Path: "/usr/bin/docker"},
	})
}

func missingMultipleResult() deps.CheckResult {
	return mockCheckResult([]deps.Dep{
		{Name: "bd", Binary: "bd", Status: deps.StatusFail, InstallHelp: "go install github.com/steveyegge/beads/cmd/bd@latest\n  or: brew install steveyegge/tap/beads"},
		{Name: "dolt", Binary: "dolt", Status: deps.StatusFail, InstallHelp: "brew install dolthub/tap/dolt"},
		{Name: "Go", Binary: "go", Status: deps.StatusOK, Version: "1.22.4", Path: "/usr/local/go/bin/go"},
		{Name: "Container Runtime", Binary: "docker", Status: deps.StatusOK, Version: "24.0.0", Path: "/usr/bin/docker"},
	})
}

// TestSetupAllOK verifies that when all deps are present, setup exits cleanly
// and prints a satisfaction message without prompting.
func TestSetupAllOK(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("") // no user input needed

	err := runSetup(in, &out, func() deps.CheckResult { return allOKResult() }, false)
	if err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "All dependencies satisfied") {
		t.Errorf("expected 'All dependencies satisfied' in output, got:\n%s", output)
	}
}

// TestSetupAllOK_ShowsCheckDepsFirst verifies that setup displays check results before anything else.
func TestSetupAllOK_ShowsCheckDepsFirst(t *testing.T) {
	var out bytes.Buffer
	in := strings.NewReader("")

	err := runSetup(in, &out, func() deps.CheckResult { return allOKResult() }, false)
	if err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}

	output := out.String()
	// Must show dep status lines before the satisfaction message
	okIdx := strings.Index(output, "[OK]")
	satIdx := strings.Index(output, "All dependencies satisfied")
	if okIdx < 0 {
		t.Errorf("expected [OK] status lines in output, got:\n%s", output)
	}
	if satIdx < 0 {
		t.Errorf("expected 'All dependencies satisfied' in output, got:\n%s", output)
	}
	if okIdx > satIdx {
		t.Errorf("expected dep status lines before satisfaction message, got:\n%s", output)
	}
}

// TestSetupMissingDep_ShowsInstallOptions verifies install method choices for missing dep.
func TestSetupMissingDep_ShowsInstallOptions(t *testing.T) {
	// Simulate user choosing 's' (skip) for bd
	input := "s\n" + "\n" // skip bd, then press Enter for re-check prompt
	var out bytes.Buffer

	err := runSetup(strings.NewReader(input), &out, func() deps.CheckResult { return missingBdResult() }, false)
	if err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}

	output := out.String()
	// Should mention bd is missing
	if !strings.Contains(output, "bd") {
		t.Errorf("expected 'bd' in output, got:\n%s", output)
	}
	// Should show skip option
	if !strings.Contains(output, "[s]") && !strings.Contains(output, "Skip") {
		t.Errorf("expected skip option in output, got:\n%s", output)
	}
}

// TestSetupMissingDep_BrewAvailable verifies brew option appears when brew is available.
func TestSetupMissingDep_BrewAvailable(t *testing.T) {
	input := "s\n\n" // skip, then Enter
	var out bytes.Buffer

	err := runSetup(strings.NewReader(input), &out, func() deps.CheckResult { return missingBdResult() }, true)
	if err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "brew") {
		t.Errorf("expected brew option in output when brew is available, got:\n%s", output)
	}
}

// TestSetupMissingDep_NoBrewHideBrewOption verifies brew option absent when brew not on PATH.
func TestSetupMissingDep_NoBrewHideBrewOption(t *testing.T) {
	input := "s\n\n" // skip, then Enter
	var out bytes.Buffer

	err := runSetup(strings.NewReader(input), &out, func() deps.CheckResult { return missingBdResult() }, false)
	if err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}

	output := out.String()
	// brew option should not appear for the install menu when brew is unavailable
	// (the word "brew" may still appear in InstallHelp text, but numbered menu option should not)
	if strings.Contains(output, "[1] brew") {
		t.Errorf("expected no '[1] brew' option when brew unavailable, got:\n%s", output)
	}
}

// TestSetupUserSelectsGoInstall verifies that when user picks go install, the command is printed.
func TestSetupUserSelectsGoInstall(t *testing.T) {
	// brew not available → [1] is go install, [s] is skip
	// Choose 1 for go install
	input := "1\n\n"
	var out bytes.Buffer

	err := runSetup(strings.NewReader(input), &out, func() deps.CheckResult { return missingBdResult() }, false)
	if err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "go install") {
		t.Errorf("expected 'go install' command in output, got:\n%s", output)
	}
}

// TestSetupUserSelectsBrewInstall verifies brew install command printed when brew available.
func TestSetupUserSelectsBrewInstall(t *testing.T) {
	// brew available → [1] is brew install, [2] is go install
	input := "1\n\n"
	var out bytes.Buffer

	err := runSetup(strings.NewReader(input), &out, func() deps.CheckResult { return missingBdResult() }, true)
	if err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "brew install") {
		t.Errorf("expected 'brew install' command in output, got:\n%s", output)
	}
}

// TestSetupDoesNotAutoInstall verifies setup prints command but does not silently run it.
// We verify by checking that user sees the command (not just a success message).
func TestSetupDoesNotAutoInstall(t *testing.T) {
	input := "1\n\n"
	var out bytes.Buffer

	// Track if go install was actually executed — we can't intercept exec.Command here,
	// so instead we verify the output contains "Run:" or similar instruction language.
	err := runSetup(strings.NewReader(input), &out, func() deps.CheckResult { return missingBdResult() }, false)
	if err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}

	output := out.String()
	// Must show a command to run rather than "installed successfully" (auto-install)
	if strings.Contains(output, "installed successfully") || strings.Contains(output, "Installing ") {
		t.Errorf("setup appears to auto-install, expected only to print command. Got:\n%s", output)
	}
}

// TestSetupReChecksAfterInstall verifies setup re-runs check after presenting install options.
func TestSetupReChecksAfterInstall(t *testing.T) {
	input := "s\n\n" // skip, then Enter to trigger re-check

	callCount := 0
	var out bytes.Buffer

	checkFn := func() deps.CheckResult {
		callCount++
		if callCount == 1 {
			return missingBdResult()
		}
		return allOKResult() // second call: all good after "install"
	}

	err := runSetup(strings.NewReader(input), &out, checkFn, false)
	if err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}

	if callCount < 2 {
		t.Errorf("expected checkFn to be called at least twice (initial + re-check), got %d calls", callCount)
	}
}

// TestSetupNextStepsGuidance verifies next steps are printed at end.
func TestSetupNextStepsGuidance(t *testing.T) {
	input := "s\n\n"
	var out bytes.Buffer

	err := runSetup(strings.NewReader(input), &out, func() deps.CheckResult { return missingBdResult() }, false)
	if err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Next steps") {
		t.Errorf("expected 'Next steps' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "doctor") {
		t.Errorf("expected 'gsdw doctor' mention in next steps, got:\n%s", output)
	}
}

// TestSetupNextStepsGuidance_AllOK verifies next steps appear even when all deps are OK.
func TestSetupNextStepsGuidance_AllOK(t *testing.T) {
	var out bytes.Buffer
	err := runSetup(strings.NewReader(""), &out, func() deps.CheckResult { return allOKResult() }, false)
	if err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "Next steps") {
		t.Errorf("expected 'Next steps' guidance even when all OK, got:\n%s", output)
	}
}

// TestSetupCommandRegistered verifies NewSetupCmd returns a valid command.
func TestSetupCommandRegistered(t *testing.T) {
	cmd := NewSetupCmd()
	if cmd == nil {
		t.Fatal("NewSetupCmd() returned nil")
	}
	if cmd.Use != "setup" {
		t.Errorf("expected Use='setup', got %q", cmd.Use)
	}
}

// TestSetupMultipleMissingDeps verifies each missing dep gets its own install menu.
func TestSetupMultipleMissingDeps(t *testing.T) {
	// Both bd and dolt missing — skip both, then re-check
	input := "s\ns\n\n"
	var out bytes.Buffer

	err := runSetup(strings.NewReader(input), &out, func() deps.CheckResult { return missingMultipleResult() }, false)
	if err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}

	output := out.String()
	// Both missing deps should be mentioned
	if !strings.Contains(output, "bd") {
		t.Errorf("expected 'bd' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "dolt") {
		t.Errorf("expected 'dolt' in output, got:\n%s", output)
	}
}

// TestSetupContainerRuntimeGuidance verifies container and connection guidance in next steps (D-08).
func TestSetupContainerRuntimeGuidance(t *testing.T) {
	var out bytes.Buffer
	err := runSetup(strings.NewReader(""), &out, func() deps.CheckResult { return allOKResult() }, false)
	if err != nil {
		t.Fatalf("runSetup returned error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "container") && !strings.Contains(output, "Container") {
		t.Errorf("expected container setup mention in next steps, got:\n%s", output)
	}
}
