package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/deps"
)

// makeOKResult returns a CheckResult with all deps OK.
func makeOKResult() deps.CheckResult {
	return deps.CheckResult{
		AllOK: true,
		Deps: []deps.Dep{
			{Name: "bd", Binary: "bd", Status: deps.StatusOK, Version: "1.4.2", Path: "/opt/homebrew/bin/bd"},
			{Name: "dolt", Binary: "dolt", Status: deps.StatusOK, Version: "1.40.0", Path: "/opt/homebrew/bin/dolt"},
			{Name: "Go", Binary: "go", Status: deps.StatusOK, Version: "1.22.4", Path: "/usr/local/go/bin/go"},
			{Name: "Container Runtime", Binary: "docker", Status: deps.StatusOK, Version: "24.0.5", Path: "/usr/local/bin/docker"},
		},
	}
}

// makeFailResult returns a CheckResult with dolt missing.
func makeFailResult() deps.CheckResult {
	return deps.CheckResult{
		AllOK: false,
		Deps: []deps.Dep{
			{Name: "bd", Binary: "bd", Status: deps.StatusOK, Version: "1.4.2", Path: "/opt/homebrew/bin/bd"},
			{Name: "dolt", Binary: "dolt", Status: deps.StatusFail, InstallHelp: "brew install dolthub/tap/dolt\n  or: curl -L https://github.com/dolthub/dolt/releases/latest/download/install.sh | bash"},
			{Name: "Go", Binary: "go", Status: deps.StatusOK, Version: "1.22.4", Path: "/usr/local/go/bin/go"},
			{Name: "Container Runtime", Binary: "docker", Status: deps.StatusOK, Version: "24.0.5", Path: "/usr/local/bin/docker"},
		},
	}
}

// TestCheckDepsCmd_Registered verifies check-deps subcommand is registered on root.
func TestCheckDepsCmd_Registered(t *testing.T) {
	root := NewRootCmd()
	for _, cmd := range root.Commands() {
		if cmd.Use == "check-deps" {
			return
		}
	}
	t.Error("root command missing 'check-deps' subcommand")
}

// TestRenderCheckDeps_OK verifies [OK] output format for all-found deps.
func TestRenderCheckDeps_OK(t *testing.T) {
	result := makeOKResult()
	var buf bytes.Buffer
	renderCheckDeps(&buf, result)
	out := buf.String()

	t.Logf("check-deps output:\n%s", out)

	// All deps must show [OK]
	for _, d := range result.Deps {
		if !strings.Contains(out, "[OK]") {
			t.Errorf("expected '[OK]' in output for dep %q, got:\n%s", d.Name, out)
		}
	}

	// Must show version and path for found deps
	if !strings.Contains(out, "1.4.2") {
		t.Errorf("expected bd version '1.4.2' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "1.40.0") {
		t.Errorf("expected dolt version '1.40.0' in output, got:\n%s", out)
	}
}

// TestRenderCheckDeps_Fail verifies [FAIL] output format with install help for missing deps.
func TestRenderCheckDeps_Fail(t *testing.T) {
	result := makeFailResult()
	var buf bytes.Buffer
	renderCheckDeps(&buf, result)
	out := buf.String()

	t.Logf("check-deps fail output:\n%s", out)

	// Must show [FAIL] for dolt
	if !strings.Contains(out, "[FAIL]") {
		t.Errorf("expected '[FAIL]' in output, got:\n%s", out)
	}

	// Must show install help for dolt
	if !strings.Contains(out, "brew install dolthub") {
		t.Errorf("expected install help 'brew install dolthub' in output, got:\n%s", out)
	}

	// Must show dolt not found message
	if !strings.Contains(out, "dolt") {
		t.Errorf("expected 'dolt' in output, got:\n%s", out)
	}

	// Must still show [OK] for bd, Go, Container Runtime
	if !strings.Contains(out, "[OK]") {
		t.Errorf("expected '[OK]' for found deps in output, got:\n%s", out)
	}
}

// TestRenderCheckDeps_Format verifies the exact output format per plan spec.
func TestRenderCheckDeps_Format(t *testing.T) {
	result := makeFailResult()
	var buf bytes.Buffer
	renderCheckDeps(&buf, result)
	out := buf.String()

	// Verify install help is indented below the [FAIL] line
	lines := strings.Split(out, "\n")
	var failIdx int = -1
	for i, line := range lines {
		if strings.Contains(line, "[FAIL]") {
			failIdx = i
			break
		}
	}
	if failIdx == -1 {
		t.Fatal("no [FAIL] line found in output")
	}
	// Install help should appear after the FAIL line
	if failIdx+1 >= len(lines) {
		t.Error("expected install help after [FAIL] line, but no more lines")
	}
	nextLine := lines[failIdx+1]
	if !strings.Contains(nextLine, "Install") && !strings.Contains(nextLine, "brew") && !strings.Contains(nextLine, "curl") {
		t.Errorf("expected install hint on line after [FAIL], got: %q", nextLine)
	}
}

// TestCheckDepsCmd_JSON verifies --json flag outputs valid structured JSON.
func TestCheckDepsCmd_JSON(t *testing.T) {
	result := makeOKResult()
	var buf bytes.Buffer
	if err := renderCheckDepsJSON(&buf, result); err != nil {
		t.Fatalf("renderCheckDepsJSON returned error: %v", err)
	}
	out := buf.String()
	t.Logf("check-deps JSON output:\n%s", out)

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, out)
	}

	// Must have "allOK" and "deps" keys
	if _, ok := parsed["allOK"]; !ok {
		t.Error("JSON output missing 'allOK' key")
	}
	if _, ok := parsed["deps"]; !ok {
		t.Error("JSON output missing 'deps' key")
	}

	// allOK must be true for makeOKResult
	allOK, _ := parsed["allOK"].(bool)
	if !allOK {
		t.Errorf("expected allOK=true in JSON, got %v", parsed["allOK"])
	}
}

// TestCheckDepsCmd_JSONFail verifies --json flag shows allOK=false for missing deps.
func TestCheckDepsCmd_JSONFail(t *testing.T) {
	result := makeFailResult()
	var buf bytes.Buffer
	if err := renderCheckDepsJSON(&buf, result); err != nil {
		t.Fatalf("renderCheckDepsJSON returned error: %v", err)
	}
	out := buf.String()

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, out)
	}

	allOK, _ := parsed["allOK"].(bool)
	if allOK {
		t.Errorf("expected allOK=false in JSON for fail result, got true")
	}
}
