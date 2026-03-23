package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"
)

// testBead creates a minimal Bead with GSD metadata for test scenarios.
func testBead(title, planID string, phaseNum float64, labels []string) graph.Bead {
	return graph.Bead{
		ID:        "bd-test-" + planID,
		Title:     title,
		Status:    "open",
		IssueType: "task",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata: map[string]any{
			"gsd_phase": phaseNum,
			"gsd_plan":  planID,
		},
		Labels: labels,
	}
}

// TestReadyCmd_TreeFormat verifies basic tree output grouping by phase with plan titles and footer.
func TestReadyCmd_TreeFormat(t *testing.T) {
	ready := []graph.Bead{
		testBead("bd CLI wrapper", "02-01", 2, []string{"gsd:plan", "INFRA-03"}),
		testBead("Domain mapping", "02-02", 2, []string{"gsd:plan", "MAP-01", "MAP-02"}),
	}
	blocked := []graph.Bead{
		testBead("MCP server", "03-01", 3, []string{"gsd:plan"}),
		testBead("Session hook", "04-01", 4, []string{"gsd:plan"}),
	}

	var buf bytes.Buffer
	if err := renderReadyTree(&buf, ready, blocked, 0); err != nil {
		t.Fatalf("renderReadyTree returned error: %v", err)
	}

	out := buf.String()
	t.Logf("tree output:\n%s", out)

	// Must show phase group header
	if !strings.Contains(out, "Phase 2:") {
		t.Errorf("expected 'Phase 2:' in output, got:\n%s", out)
	}

	// Must show plan names (not bd IDs)
	if !strings.Contains(out, "Plan 02-01:") {
		t.Errorf("expected 'Plan 02-01:' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Plan 02-02:") {
		t.Errorf("expected 'Plan 02-02:' in output, got:\n%s", out)
	}

	// Must show requirement labels in brackets
	if !strings.Contains(out, "[INFRA-03]") {
		t.Errorf("expected '[INFRA-03]' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "[MAP-01, MAP-02]") {
		t.Errorf("expected '[MAP-01, MAP-02]' in output, got:\n%s", out)
	}

	// Must use tree chars (|-- and +--)
	if !strings.Contains(out, "|--") && !strings.Contains(out, "+--") {
		t.Errorf("expected ASCII tree chars (|-- or +--) in output, got:\n%s", out)
	}

	// Must contain footer with ready/queued/remaining
	if !strings.Contains(out, "ready") {
		t.Errorf("expected 'ready' in footer, got:\n%s", out)
	}
	if !strings.Contains(out, "queued") {
		t.Errorf("expected 'queued' in footer, got:\n%s", out)
	}
	if !strings.Contains(out, "remaining") {
		t.Errorf("expected 'remaining' in footer, got:\n%s", out)
	}

	// Must not contain bd IDs in output (per D-19)
	if strings.Contains(out, "bd-test-") {
		t.Errorf("output must not contain bd IDs, got:\n%s", out)
	}
}

// TestReadyCmd_JSON verifies --json outputs a valid JSON array of Bead objects.
func TestReadyCmd_JSON(t *testing.T) {
	ready := []graph.Bead{
		testBead("bd CLI wrapper", "02-01", 2, []string{"gsd:plan", "INFRA-03"}),
		testBead("Domain mapping", "02-02", 2, []string{"gsd:plan", "MAP-01"}),
	}

	var buf bytes.Buffer
	if err := renderReadyJSON(&buf, ready); err != nil {
		t.Fatalf("renderReadyJSON returned error: %v", err)
	}

	out := buf.String()
	t.Logf("JSON output:\n%s", out)

	// Must be valid JSON array
	var decoded []graph.Bead
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output is not valid JSON array: %v\noutput:\n%s", err, out)
	}

	if len(decoded) != 2 {
		t.Errorf("expected 2 beads in JSON output, got %d", len(decoded))
	}
}

// TestReadyCmd_PhaseFilter verifies --phase N only shows tasks from that phase.
func TestReadyCmd_PhaseFilter(t *testing.T) {
	ready := []graph.Bead{
		testBead("bd CLI wrapper", "02-01", 2, []string{"gsd:plan", "INFRA-03"}),
		testBead("MCP server", "03-01", 3, []string{"gsd:plan", "MCP-01"}),
		testBead("Another phase 2 plan", "02-02", 2, []string{"gsd:plan", "MAP-01"}),
	}
	blocked := []graph.Bead{}

	var buf bytes.Buffer
	if err := renderReadyTree(&buf, ready, blocked, 2); err != nil {
		t.Fatalf("renderReadyTree returned error: %v", err)
	}

	out := buf.String()
	t.Logf("phase filtered output:\n%s", out)

	// Phase 2 items must appear
	if !strings.Contains(out, "Phase 2:") {
		t.Errorf("expected 'Phase 2:' in filtered output, got:\n%s", out)
	}
	if !strings.Contains(out, "Plan 02-01:") {
		t.Errorf("expected 'Plan 02-01:' in filtered output, got:\n%s", out)
	}

	// Phase 3 items must NOT appear
	if strings.Contains(out, "Phase 3:") {
		t.Errorf("expected 'Phase 3:' to be filtered out, got:\n%s", out)
	}
	if strings.Contains(out, "Plan 03-01:") {
		t.Errorf("expected 'Plan 03-01:' to be filtered out, got:\n%s", out)
	}
}

// TestReadyCmd_EmptyReady verifies "No ready work" message when there are no ready tasks.
func TestReadyCmd_EmptyReady(t *testing.T) {
	ready := []graph.Bead{}
	blocked := []graph.Bead{
		testBead("Some blocked task", "03-01", 3, []string{"gsd:plan"}),
	}

	var buf bytes.Buffer
	if err := renderReadyTree(&buf, ready, blocked, 0); err != nil {
		t.Fatalf("renderReadyTree returned error: %v", err)
	}

	out := buf.String()
	t.Logf("empty ready output:\n%s", out)

	if !strings.Contains(out, "No ready work") {
		t.Errorf("expected 'No ready work' in output, got:\n%s", out)
	}
}

// TestReadyCmd_GSDNames verifies output uses GSD names (Phase N, Plan XX-YY) not bd IDs.
func TestReadyCmd_GSDNames(t *testing.T) {
	ready := []graph.Bead{
		testBead("Graph Primitives Plan 1", "02-01", 2, []string{"gsd:plan"}),
	}
	blocked := []graph.Bead{}

	var buf bytes.Buffer
	if err := renderReadyTree(&buf, ready, blocked, 0); err != nil {
		t.Fatalf("renderReadyTree returned error: %v", err)
	}

	out := buf.String()

	// GSD names must appear
	if !strings.Contains(out, "Phase 2:") {
		t.Errorf("expected GSD-style 'Phase 2:' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Plan 02-01:") {
		t.Errorf("expected GSD-style 'Plan 02-01:' in output, got:\n%s", out)
	}

	// bd IDs must NOT appear (they start with bd-)
	if strings.Contains(out, "bd-test-") {
		t.Errorf("output must not contain bd IDs, got:\n%s", out)
	}
}

// TestPhaseNumFromBead_NilMetadata verifies phaseNumFromBead returns 0 for nil metadata.
func TestPhaseNumFromBead_NilMetadata(t *testing.T) {
	b := graph.Bead{ID: "test-1"}
	if got := phaseNumFromBead(b); got != 0 {
		t.Errorf("phaseNumFromBead(nil meta) = %d, want 0", got)
	}
}

// TestPhaseNumFromBead_Float64 verifies JSON-unmarshaled float64 is correctly converted.
func TestPhaseNumFromBead_Float64(t *testing.T) {
	b := graph.Bead{Metadata: map[string]any{"gsd_phase": float64(5)}}
	if got := phaseNumFromBead(b); got != 5 {
		t.Errorf("phaseNumFromBead(float64(5)) = %d, want 5", got)
	}
}

// TestPhaseNumFromBead_Int verifies direct int construction works.
func TestPhaseNumFromBead_Int(t *testing.T) {
	b := graph.Bead{Metadata: map[string]any{"gsd_phase": 9}}
	if got := phaseNumFromBead(b); got != 9 {
		t.Errorf("phaseNumFromBead(int 9) = %d, want 9", got)
	}
}

// TestPhaseNumFromBead_Int64 verifies int64 type is handled.
func TestPhaseNumFromBead_Int64(t *testing.T) {
	b := graph.Bead{Metadata: map[string]any{"gsd_phase": int64(12)}}
	if got := phaseNumFromBead(b); got != 12 {
		t.Errorf("phaseNumFromBead(int64(12)) = %d, want 12", got)
	}
}

// TestPhaseNumFromBead_WrongType verifies non-numeric type returns 0.
func TestPhaseNumFromBead_WrongType(t *testing.T) {
	b := graph.Bead{Metadata: map[string]any{"gsd_phase": "not-a-number"}}
	if got := phaseNumFromBead(b); got != 0 {
		t.Errorf("phaseNumFromBead(string) = %d, want 0", got)
	}
}

// TestPlanIDFromBead_NilMetadata verifies planIDFromBead returns empty for nil metadata.
func TestPlanIDFromBead_NilMetadata(t *testing.T) {
	b := graph.Bead{ID: "test-1"}
	if got := planIDFromBead(b); got != "" {
		t.Errorf("planIDFromBead(nil meta) = %q, want empty", got)
	}
}

// TestPlanIDFromBead_Valid verifies planIDFromBead extracts gsd_plan string.
func TestPlanIDFromBead_Valid(t *testing.T) {
	b := graph.Bead{Metadata: map[string]any{"gsd_plan": "07-02"}}
	if got := planIDFromBead(b); got != "07-02" {
		t.Errorf("planIDFromBead = %q, want %q", got, "07-02")
	}
}

// TestPlanIDFromBead_WrongType verifies non-string type returns empty.
func TestPlanIDFromBead_WrongType(t *testing.T) {
	b := graph.Bead{Metadata: map[string]any{"gsd_plan": 42}}
	if got := planIDFromBead(b); got != "" {
		t.Errorf("planIDFromBead(int) = %q, want empty", got)
	}
}

// TestPlanIDFromBead_MissingKey verifies missing key returns empty.
func TestPlanIDFromBead_MissingKey(t *testing.T) {
	b := graph.Bead{Metadata: map[string]any{"other": "val"}}
	if got := planIDFromBead(b); got != "" {
		t.Errorf("planIDFromBead(missing key) = %q, want empty", got)
	}
}

// TestFindBeadsDir_EnvVar verifies BEADS_DIR environment variable is returned when set.
func TestFindBeadsDir_EnvVar(t *testing.T) {
	t.Setenv("BEADS_DIR", "/custom/beads")
	got, err := findBeadsDir()
	if err != nil {
		t.Fatalf("findBeadsDir() error: %v", err)
	}
	if got != "/custom/beads" {
		t.Errorf("findBeadsDir() = %q, want %q", got, "/custom/beads")
	}
}

// TestFindBeadsDir_WalkUp verifies findBeadsDir locates .beads/ in a parent directory.
func TestFindBeadsDir_WalkUp(t *testing.T) {
	t.Setenv("BEADS_DIR", "") // ensure env var is not used
	tmpDir := resolveSymlinks(t, t.TempDir())

	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	childDir := filepath.Join(tmpDir, "sub", "deep")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(childDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	got, err := findBeadsDir()
	if err != nil {
		t.Fatalf("findBeadsDir() error: %v", err)
	}
	if got != beadsDir {
		t.Errorf("findBeadsDir() = %q, want %q", got, beadsDir)
	}
}

// TestFindBeadsDir_NotFound verifies error when .beads/ doesn't exist anywhere.
func TestFindBeadsDir_NotFound(t *testing.T) {
	t.Setenv("BEADS_DIR", "") // ensure env var is not used
	tmpDir := t.TempDir()

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	_, err := findBeadsDir()
	if err == nil {
		t.Fatal("findBeadsDir() expected error when .beads/ not found, got nil")
	}
	if !strings.Contains(err.Error(), "no beads database") {
		t.Errorf("error %q should mention 'no beads database'", err.Error())
	}
}

// TestFindBeadsDir_InCwd verifies findBeadsDir locates .beads/ directly in cwd.
func TestFindBeadsDir_InCwd(t *testing.T) {
	t.Setenv("BEADS_DIR", "") // ensure env var is not used
	tmpDir := resolveSymlinks(t, t.TempDir())

	beadsDir := filepath.Join(tmpDir, ".beads")
	if err := os.MkdirAll(beadsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	got, err := findBeadsDir()
	if err != nil {
		t.Fatalf("findBeadsDir() error: %v", err)
	}
	if got != beadsDir {
		t.Errorf("findBeadsDir() = %q, want %q", got, beadsDir)
	}
}

// TestReadyCmd_ReqLabels verifies requirement labels ([A-Z]+-[0-9]+) appear in brackets.
func TestReadyCmd_ReqLabels(t *testing.T) {
	ready := []graph.Bead{
		testBead("bd CLI wrapper", "02-01", 2, []string{"gsd:plan", "INFRA-03", "MAP-01"}),
	}
	blocked := []graph.Bead{}

	var buf bytes.Buffer
	if err := renderReadyTree(&buf, ready, blocked, 0); err != nil {
		t.Fatalf("renderReadyTree returned error: %v", err)
	}

	out := buf.String()
	t.Logf("req labels output:\n%s", out)

	// Both req labels must appear in brackets
	if !strings.Contains(out, "[INFRA-03") {
		t.Errorf("expected '[INFRA-03' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "MAP-01") {
		t.Errorf("expected 'MAP-01' in output, got:\n%s", out)
	}

	// Internal GSD labels (gsd:plan, gsd:phase) must NOT appear in tree output
	if strings.Contains(out, "gsd:plan") {
		t.Errorf("output must not contain internal 'gsd:plan' label, got:\n%s", out)
	}
}
