package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"
)

// TestRootCmdHasStatus verifies that NewRootCmd registers the "status" subcommand.
func TestRootCmdHasStatus(t *testing.T) {
	root := NewRootCmd()
	for _, cmd := range root.Commands() {
		if cmd.Use == "status" {
			return // found
		}
	}
	t.Errorf("expected 'status' subcommand registered in root, but it was not found")
}

// testPhaseBead creates a minimal Bead representing a GSD phase (epic) for test scenarios.
func testPhaseBead(title string, phaseNum float64) graph.Bead {
	return graph.Bead{
		ID:        "bd-phase-test",
		Title:     title,
		Status:    "open",
		IssueType: "epic",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata: map[string]any{
			"gsd_phase": phaseNum,
		},
		Labels: []string{"gsd:phase"},
	}
}

// TestStatusCmdOutput verifies renderStatus produces GSD-familiar output with phase and ready tasks.
func TestStatusCmdOutput(t *testing.T) {
	phases := []graph.Bead{
		testPhaseBead("Binary Scaffold", 1),
	}
	ready := []graph.Bead{
		testBead("Init command", "05-02", 5, []string{"gsd:plan", "INIT-01"}),
		testBead("MCP tools", "05-01", 5, []string{"gsd:plan", "INIT-03"}),
	}

	var buf bytes.Buffer
	renderStatus(&buf, phases, ready)

	out := buf.String()
	t.Logf("status output:\n%s", out)

	// Must contain GSD Status header
	if !strings.Contains(out, "GSD Project Status") {
		t.Errorf("expected 'GSD Project Status' in output, got:\n%s", out)
	}

	// Must contain current phase info with GSD terminology
	if !strings.Contains(out, "Phase") {
		t.Errorf("expected 'Phase' in output, got:\n%s", out)
	}

	// Must contain phase title
	if !strings.Contains(out, "Binary Scaffold") {
		t.Errorf("expected phase title 'Binary Scaffold' in output, got:\n%s", out)
	}

	// Must list ready tasks using plan IDs
	if !strings.Contains(out, "Plan 05-02") {
		t.Errorf("expected 'Plan 05-02' in output, got:\n%s", out)
	}

	// Must NOT contain bead IDs (e.g., "bd-")
	if strings.Contains(out, "bd-") {
		t.Errorf("output must not contain bead IDs, got:\n%s", out)
	}
}

// TestStatusCmdNoProject verifies renderStatus shows helpful message when no phases exist.
func TestStatusCmdNoProject(t *testing.T) {
	var buf bytes.Buffer
	renderStatus(&buf, []graph.Bead{}, []graph.Bead{})

	out := buf.String()
	t.Logf("no project output:\n%s", out)

	if !strings.Contains(out, "No project initialized") {
		t.Errorf("expected 'No project initialized' message when no phases exist, got:\n%s", out)
	}
}
