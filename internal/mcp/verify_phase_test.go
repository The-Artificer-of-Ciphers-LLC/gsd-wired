package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// fakePhaseBead returns a JSON array with one phase epic whose acceptance criteria
// contains the given criteria lines joined by newlines.
func fakePhaseBead(phaseNum int, criteria string) []byte {
	bead := map[string]any{
		"id":                  "bd-phase-verify",
		"title":               "Phase 7",
		"status":              "open",
		"acceptance_criteria": criteria,
		"metadata": map[string]any{
			"gsd_phase": phaseNum,
		},
		"labels": []string{"gsd:phase"},
	}
	data, _ := json.Marshal([]any{bead})
	return data
}

// setupVerifyPhaseState sets up a serverState with fake_bd returning a custom phase epic.
// The phase epic has acceptance_criteria set to `criteria`.
// Returns the temp dir and a ready-to-use serverState.
func setupVerifyPhaseState(t *testing.T, phaseNum int, criteria string) (*serverState, string) {
	t.Helper()
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	// Write a custom phase bead to a temp file and wire up fake_bd query to return it.
	phaseData := fakePhaseBead(phaseNum, criteria)
	phaseFile := filepath.Join(tmpDir, "phase.json")
	if err := os.WriteFile(phaseFile, phaseData, 0644); err != nil {
		t.Fatal(err)
	}

	// We need fake_bd's query subcommand to return our custom phase bead.
	// Use FAKE_BD_QUERY_PHASE_RESPONSE env var approach via query subcommand support.
	// Since fake_bd doesn't support this yet, we override the canned bead via
	// writing the file and using FAKE_BD_SHOW_RESPONSE is not right here.
	//
	// Instead, use the existing cannedPhaseBead pattern: fake_bd already returns
	// cannedPhaseBead (phase_num=1) for "query label=gsd:phase". For phase 1 tests
	// this works. For testing "no phase found", use phase_num=99.
	//
	// For full acceptance criteria testing, we inject via a custom fake_bd query file.
	// Since fake_bd doesn't support FAKE_BD_QUERY_RESPONSE, we write a wrapper binary.
	// Simpler: use FAKE_BD_QUERY_PHASE_RESPONSE env var — added to fake_bd in this task.
	t.Setenv("FAKE_BD_QUERY_PHASE_RESPONSE", phaseFile)

	state := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	if err := state.init(context.Background()); err != nil {
		t.Fatalf("state.init() failed: %v", err)
	}
	return state, tmpDir
}

// TestVerifyPhase verifies that verify_phase with a file-exists criterion for an
// existing file returns passed=true for that criterion.
func TestVerifyPhase(t *testing.T) {
	// Create a temp project dir with a known file.
	projectDir := t.TempDir()
	testFile := filepath.Join(projectDir, "internal/mcp/execute_wave.go")
	if err := os.MkdirAll(filepath.Dir(testFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(testFile, []byte("package mcp"), 0644); err != nil {
		t.Fatal(err)
	}

	criteria := "internal/mcp/execute_wave.go exists and is implemented"
	state, _ := setupVerifyPhaseState(t, 1, criteria)
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "verify_phase",
		Arguments: map[string]any{
			"phase_num":   1,
			"project_dir": projectDir,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(verify_phase) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(verify_phase) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp verifyPhaseResult
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("verify_phase response is not valid JSON: %v, text: %s", err, text)
	}

	if resp.PhaseNum != 1 {
		t.Errorf("verify_phase PhaseNum = %d, want 1", resp.PhaseNum)
	}
	if len(resp.Results) == 0 {
		t.Fatalf("verify_phase returned no results")
	}
	// The file exists so the criterion should pass.
	if !resp.Results[0].Passed {
		t.Errorf("verify_phase criterion should pass (file exists), got passed=false, detail: %s", resp.Results[0].Detail)
	}
	if resp.Results[0].Method != "file_exists" {
		t.Errorf("verify_phase method = %q, want \"file_exists\"", resp.Results[0].Method)
	}
}

// TestVerifyPhaseFileCheck verifies that a missing file returns passed=false with method=file_exists.
func TestVerifyPhaseFileCheck(t *testing.T) {
	projectDir := t.TempDir()
	// Do NOT create the file — it should be missing.
	criteria := "skills/execute/SKILL.md exists and is implemented"
	state, _ := setupVerifyPhaseState(t, 1, criteria)
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "verify_phase",
		Arguments: map[string]any{
			"phase_num":   1,
			"project_dir": projectDir,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(verify_phase, file-check) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(verify_phase, file-check) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp verifyPhaseResult
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("verify_phase file-check response is not valid JSON: %v, text: %s", err, text)
	}

	if len(resp.Results) == 0 {
		t.Fatalf("verify_phase returned no results")
	}
	cr := resp.Results[0]
	if cr.Passed {
		t.Errorf("verify_phase criterion should fail (file missing), got passed=true")
	}
	if cr.Method != "file_exists" {
		t.Errorf("verify_phase method = %q, want \"file_exists\"", cr.Method)
	}
}

// TestVerifyPhaseGoTest verifies that a criterion containing "test" uses go_test method.
func TestVerifyPhaseGoTest(t *testing.T) {
	// Use a temp project dir with no Go files — go test ./... should fail.
	projectDir := t.TempDir()

	criteria := "all tests pass with -race flag"
	state, _ := setupVerifyPhaseState(t, 1, criteria)
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "verify_phase",
		Arguments: map[string]any{
			"phase_num":   1,
			"project_dir": projectDir,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(verify_phase, go-test) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(verify_phase, go-test) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp verifyPhaseResult
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("verify_phase go-test response is not valid JSON: %v, text: %s", err, text)
	}

	if len(resp.Results) == 0 {
		t.Fatalf("verify_phase returned no results")
	}
	cr := resp.Results[0]
	if cr.Method != "go_test" {
		t.Errorf("verify_phase method = %q, want \"go_test\"", cr.Method)
	}
	// Empty dir has no Go files — go test ./... should return non-zero.
	// (either "no Go files" error or test failures)
}

// TestVerifyPhaseFailures verifies that when criteria fail, the Failed array contains
// the criterion text.
func TestVerifyPhaseFailures(t *testing.T) {
	projectDir := t.TempDir()

	// Create one file that exists (passes) and reference one that doesn't (fails).
	existingFile := filepath.Join(projectDir, "internal/mcp/execute_wave.go")
	if err := os.MkdirAll(filepath.Dir(existingFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(existingFile, []byte("package mcp"), 0644); err != nil {
		t.Fatal(err)
	}

	// Two criteria: first passes (file exists), second fails (file missing).
	criteria := "internal/mcp/execute_wave.go exists\ninternal/mcp/missing_file.go exists"
	state, _ := setupVerifyPhaseState(t, 1, criteria)
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "verify_phase",
		Arguments: map[string]any{
			"phase_num":   1,
			"project_dir": projectDir,
		},
	})
	if err != nil {
		t.Fatalf("CallTool(verify_phase, failures) returned error: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(verify_phase, failures) returned IsError=true: %v", contentText(result))
	}

	text := contentText(result)
	var resp verifyPhaseResult
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		t.Fatalf("verify_phase failures response is not valid JSON: %v, text: %s", err, text)
	}

	if resp.Passed {
		t.Errorf("verify_phase should not be Passed when some criteria fail")
	}
	if len(resp.Failed) != 1 {
		t.Errorf("verify_phase Failed array length = %d, want 1; failed: %v", len(resp.Failed), resp.Failed)
	}
	if len(resp.Failed) > 0 && !contains(resp.Failed[0], "missing_file.go") {
		t.Errorf("verify_phase Failed[0] = %q, want it to contain 'missing_file.go'", resp.Failed[0])
	}
}

// --- hasUppercaseIdentifier unit tests ---

func TestHasUppercaseIdentifier_WithGoType(t *testing.T) {
	if !hasUppercaseIdentifier("The HandleExecuteWave function exists") {
		t.Error("expected true for Go type name 'HandleExecuteWave'")
	}
}

func TestHasUppercaseIdentifier_WithoutIdentifier(t *testing.T) {
	if hasUppercaseIdentifier("all tests pass with no failures") {
		t.Error("expected false for lowercase-only text")
	}
}

func TestHasUppercaseIdentifier_SingleCharWord(t *testing.T) {
	if hasUppercaseIdentifier("A b c") {
		t.Error("expected false for single-char uppercase word 'A'")
	}
}

func TestHasUppercaseIdentifier_WithPunctuation(t *testing.T) {
	if !hasUppercaseIdentifier("(ServerState) is initialized") {
		t.Error("expected true for 'ServerState' wrapped in parens")
	}
}

func TestHasUppercaseIdentifier_EmptyString(t *testing.T) {
	if hasUppercaseIdentifier("") {
		t.Error("expected false for empty string")
	}
}

// --- extractFilePath unit tests ---

func TestExtractFilePath_GoFile(t *testing.T) {
	got := extractFilePath("internal/mcp/server.go exists and compiles")
	if got != "internal/mcp/server.go" {
		t.Errorf("extractFilePath() = %q, want \"internal/mcp/server.go\"", got)
	}
}

func TestExtractFilePath_NoFile(t *testing.T) {
	got := extractFilePath("all tests pass")
	if got != "" {
		t.Errorf("extractFilePath() = %q, want empty string", got)
	}
}

func TestExtractFilePath_MarkdownFile(t *testing.T) {
	got := extractFilePath("skills/init/SKILL.md is present")
	if got != "skills/init/SKILL.md" {
		t.Errorf("extractFilePath() = %q, want \"skills/init/SKILL.md\"", got)
	}
}

func TestExtractFilePath_WithTrailingPunctuation(t *testing.T) {
	got := extractFilePath("Check internal/cli/root.go, it should exist.")
	if got != "internal/cli/root.go" {
		t.Errorf("extractFilePath() = %q, want \"internal/cli/root.go\"", got)
	}
}

// TestVerifyPhaseNoPhase verifies that a non-existent phase_num returns toolError.
func TestVerifyPhaseNoPhase(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	state := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	if err := state.init(context.Background()); err != nil {
		t.Fatalf("state.init() failed: %v", err)
	}
	cs := connectInProcess(t, state)

	// phase_num=99 does not exist in fake_bd (cannedPhaseBead has gsd_phase:1).
	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "verify_phase",
		Arguments: map[string]any{"phase_num": 99},
	})
	if err != nil {
		t.Fatalf("CallTool(verify_phase, no-phase) returned protocol error: %v", err)
	}
	if !result.IsError {
		t.Fatalf("verify_phase with non-existent phase_num should return IsError=true, got: %s", contentText(result))
	}
	text := contentText(result)
	if !contains(text, "no phase epic found") {
		t.Errorf("verify_phase error should contain 'no phase epic found', got: %s", text)
	}
}
