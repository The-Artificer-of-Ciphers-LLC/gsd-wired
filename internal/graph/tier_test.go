package graph

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- classifyTier tests ---

func TestClassifyTier_OpenIsHot(t *testing.T) {
	b := Bead{ID: "b1", Status: "open"}
	got := classifyTier(b, nil)
	if got != TierHot {
		t.Errorf("classifyTier(open) = %q, want %q", got, TierHot)
	}
}

func TestClassifyTier_InProgressIsHot(t *testing.T) {
	b := Bead{ID: "b1", Status: "in_progress"}
	got := classifyTier(b, nil)
	if got != TierHot {
		t.Errorf("classifyTier(in_progress) = %q, want %q", got, TierHot)
	}
}

func TestClassifyTier_ClosedInWarmSetIsWarm(t *testing.T) {
	now := time.Now()
	b := Bead{ID: "b1", Status: "closed", ClosedAt: &now}
	warmIDs := map[string]bool{"b1": true}
	got := classifyTier(b, warmIDs)
	if got != TierWarm {
		t.Errorf("classifyTier(closed, in warmIDs) = %q, want %q", got, TierWarm)
	}
}

func TestClassifyTier_ClosedNotInWarmSetIsCold(t *testing.T) {
	now := time.Now()
	b := Bead{ID: "b1", Status: "closed", ClosedAt: &now}
	warmIDs := map[string]bool{"other-id": true}
	got := classifyTier(b, warmIDs)
	if got != TierCold {
		t.Errorf("classifyTier(closed, not in warmIDs) = %q, want %q", got, TierCold)
	}
}

func TestClassifyTier_ClosedNilClosedAtIsCold(t *testing.T) {
	b := Bead{ID: "b1", Status: "closed", ClosedAt: nil}
	got := classifyTier(b, nil)
	if got != TierCold {
		t.Errorf("classifyTier(closed, nil ClosedAt) = %q, want %q", got, TierCold)
	}
}

func TestClassifyTier_EmptyWarmIDs(t *testing.T) {
	now := time.Now()
	b := Bead{ID: "b1", Status: "closed", ClosedAt: &now}
	got := classifyTier(b, map[string]bool{})
	if got != TierCold {
		t.Errorf("classifyTier(closed, empty warmIDs) = %q, want %q", got, TierCold)
	}
}

// --- estimateTokens tests ---

func TestEstimateTokens_EmptyString(t *testing.T) {
	got := estimateTokens("")
	if got != 0 {
		t.Errorf("estimateTokens(\"\") = %d, want 0", got)
	}
}

func TestEstimateTokens_FourBytes(t *testing.T) {
	got := estimateTokens("abcd") // 4 bytes -> 1 token
	if got != 1 {
		t.Errorf("estimateTokens(\"abcd\") = %d, want 1", got)
	}
}

func TestEstimateTokens_MinOneForNonEmpty(t *testing.T) {
	got := estimateTokens("ab") // 2 bytes -> 0 raw, min 1
	if got != 1 {
		t.Errorf("estimateTokens(\"ab\") = %d, want 1 (min 1 for non-empty)", got)
	}
}

func TestEstimateTokens_FourHundredBytes(t *testing.T) {
	s := strings.Repeat("a", 400)
	got := estimateTokens(s) // 400/4 = 100
	if got != 100 {
		t.Errorf("estimateTokens(400 bytes) = %d, want 100", got)
	}
}

func TestEstimateTokens_SingleByte(t *testing.T) {
	got := estimateTokens("a") // 1 byte -> 0 raw, min 1
	if got != 1 {
		t.Errorf("estimateTokens(\"a\") = %d, want 1 (min 1 for non-empty)", got)
	}
}

// --- TieredBead tests ---

func TestTieredBead_EmbedsBead(t *testing.T) {
	b := Bead{ID: "b1", Title: "My Task", Status: "open"}
	tb := TieredBead{Bead: b, Tier: TierHot}
	if tb.ID != "b1" {
		t.Errorf("TieredBead.ID = %q, want \"b1\"", tb.ID)
	}
	if tb.Title != "My Task" {
		t.Errorf("TieredBead.Title = %q, want \"My Task\"", tb.Title)
	}
	if tb.Tier != TierHot {
		t.Errorf("TieredBead.Tier = %q, want %q", tb.Tier, TierHot)
	}
}

func TestTieredBead_JSONMarshal(t *testing.T) {
	b := Bead{ID: "b1", Title: "Task", Status: "open"}
	tb := TieredBead{Bead: b, Tier: TierHot, Compact: ""}

	data, err := json.Marshal(tb)
	if err != nil {
		t.Fatalf("json.Marshal(TieredBead) error: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, `"tier"`) {
		t.Errorf("TieredBead JSON missing \"tier\" field: %s", s)
	}
	if !strings.Contains(s, `"hot"`) {
		t.Errorf("TieredBead JSON missing tier value \"hot\": %s", s)
	}
}

func TestTieredBead_CompactOmitEmptyInJSON(t *testing.T) {
	b := Bead{ID: "b1", Title: "Task", Status: "open"}
	tb := TieredBead{Bead: b, Tier: TierHot, Compact: ""}

	data, err := json.Marshal(tb)
	if err != nil {
		t.Fatalf("json.Marshal(TieredBead) error: %v", err)
	}
	s := string(data)
	// compact field should be omitted when empty
	if strings.Contains(s, `"compact"`) {
		t.Errorf("TieredBead JSON should omit empty compact field: %s", s)
	}
}

// --- formatHot tests ---

func TestFormatHot_Basic(t *testing.T) {
	b := Bead{ID: "bd-abc", Title: "My Plan", Status: "open"}
	tb := TieredBead{Bead: b, Tier: TierHot}
	got := formatHot(tb)
	if !strings.Contains(got, "[HOT]") {
		t.Errorf("formatHot() missing [HOT]: %q", got)
	}
	if !strings.Contains(got, "My Plan") {
		t.Errorf("formatHot() missing title: %q", got)
	}
	if !strings.Contains(got, "bd-abc") {
		t.Errorf("formatHot() missing id: %q", got)
	}
}

func TestFormatHot_WithDescriptionAndCriteria(t *testing.T) {
	b := Bead{
		ID:                 "bd-abc",
		Title:              "My Plan",
		Status:             "open",
		Description:        "Build the thing",
		AcceptanceCriteria: "Tests pass",
	}
	tb := TieredBead{Bead: b, Tier: TierHot}
	got := formatHot(tb)
	if !strings.Contains(got, "Build the thing") {
		t.Errorf("formatHot() missing description: %q", got)
	}
	if !strings.Contains(got, "Tests pass") {
		t.Errorf("formatHot() missing acceptance criteria: %q", got)
	}
}

func TestFormatHot_StartsWithHeaderLine(t *testing.T) {
	b := Bead{ID: "id1", Title: "Title", Status: "open"}
	tb := TieredBead{Bead: b, Tier: TierHot}
	got := formatHot(tb)
	if !strings.HasPrefix(got, "## [HOT]") {
		t.Errorf("formatHot() should start with '## [HOT]', got: %q", got)
	}
}

// --- formatWarm tests ---

func TestFormatWarm(t *testing.T) {
	closedAt := time.Now().Add(-24 * time.Hour)
	b := Bead{ID: "bd-xyz", Title: "Old Plan", Status: "closed", ClosedAt: &closedAt, CloseReason: "done successfully"}
	tb := TieredBead{Bead: b, Tier: TierWarm}
	got := formatWarm(tb)
	if !strings.Contains(got, "[WARM]") {
		t.Errorf("formatWarm() missing [WARM]: %q", got)
	}
	if !strings.Contains(got, "Old Plan") {
		t.Errorf("formatWarm() missing title: %q", got)
	}
	if !strings.Contains(got, "done successfully") {
		t.Errorf("formatWarm() missing close_reason: %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("formatWarm() should end with newline: %q", got)
	}
}

// --- formatCold tests ---

func TestFormatCold(t *testing.T) {
	b := Bead{ID: "bd-old", Title: "Ancient Plan", Status: "closed"}
	tb := TieredBead{Bead: b, Tier: TierCold}
	got := formatCold(tb)
	if !strings.Contains(got, "bd-old") {
		t.Errorf("formatCold() missing id: %q", got)
	}
	if !strings.Contains(got, "Ancient Plan") {
		t.Errorf("formatCold() missing title: %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("formatCold() should end with newline: %q", got)
	}
	// Should have format "- [id] title\n"
	if !strings.HasPrefix(got, "- [") {
		t.Errorf("formatCold() should start with '- [', got: %q", got)
	}
}

// --- Exported Format wrapper tests (FormatHot/FormatWarm/FormatCold) ---

func TestFormatHot_Exported(t *testing.T) {
	b := Bead{ID: "bd-exp", Title: "Export Test", Status: "open", Description: "Build it", AcceptanceCriteria: "Tests pass"}
	tb := TieredBead{Bead: b, Tier: TierHot}
	got := FormatHot(tb)
	if !strings.Contains(got, "[HOT]") {
		t.Errorf("FormatHot() missing [HOT]: %q", got)
	}
	if !strings.Contains(got, "Export Test") {
		t.Errorf("FormatHot() missing title: %q", got)
	}
	if !strings.Contains(got, "Build it") {
		t.Errorf("FormatHot() missing description: %q", got)
	}
}

func TestFormatWarm_Exported(t *testing.T) {
	now := time.Now()
	b := Bead{ID: "bd-warm-exp", Title: "Warm Export", Status: "closed", ClosedAt: &now, CloseReason: "completed"}
	tb := TieredBead{Bead: b, Tier: TierWarm}
	got := FormatWarm(tb)
	if !strings.Contains(got, "[WARM]") {
		t.Errorf("FormatWarm() missing [WARM]: %q", got)
	}
	if !strings.Contains(got, "completed") {
		t.Errorf("FormatWarm() missing close reason: %q", got)
	}
}

func TestFormatCold_Exported(t *testing.T) {
	b := Bead{ID: "bd-cold-exp", Title: "Cold Export", Status: "closed"}
	tb := TieredBead{Bead: b, Tier: TierCold}
	got := FormatCold(tb)
	if !strings.Contains(got, "bd-cold-exp") {
		t.Errorf("FormatCold() missing id: %q", got)
	}
	if !strings.Contains(got, "Cold Export") {
		t.Errorf("FormatCold() missing title: %q", got)
	}
}

// --- EstimateTokens exported wrapper test ---

func TestEstimateTokens_Exported(t *testing.T) {
	got := EstimateTokens("hello world!")
	if got < 1 {
		t.Errorf("EstimateTokens(\"hello world!\") = %d, want >= 1", got)
	}
}

// --- classifyTier edge cases ---

func TestClassifyTier_UnknownStatus(t *testing.T) {
	b := Bead{ID: "b1", Status: "unknown_status"}
	got := classifyTier(b, nil)
	if got != TierCold {
		t.Errorf("classifyTier(unknown status) = %q, want %q", got, TierCold)
	}
}

// --- compactSummary tests ---

func TestCompactSummary_WithCloseReason(t *testing.T) {
	b := &Bead{Title: "My Plan", CloseReason: "completed feature X"}
	got := compactSummary(b)
	want := "My Plan: completed feature X"
	if got != want {
		t.Errorf("compactSummary(with reason) = %q, want %q", got, want)
	}
}

func TestCompactSummary_WithoutCloseReason(t *testing.T) {
	b := &Bead{Title: "My Plan", CloseReason: ""}
	got := compactSummary(b)
	want := "My Plan"
	if got != want {
		t.Errorf("compactSummary(no reason) = %q, want %q", got, want)
	}
}

// --- CompactBead tests (needs fake_bd) ---

func TestCompactBead(t *testing.T) {
	dir := t.TempDir()
	captureFile := filepath.Join(dir, "capture.json")
	t.Setenv("FAKE_BD_CAPTURE_FILE", captureFile)

	c := NewClientWithPath(fakeBdPath, t.TempDir())
	ctx := context.Background()

	bead, err := c.CompactBead(ctx, "bd-test-abc", "My Plan: completed")
	if err != nil {
		t.Fatalf("CompactBead() returned error: %v", err)
	}
	if bead == nil {
		t.Fatal("CompactBead() returned nil bead")
	}

	data, _ := os.ReadFile(captureFile)
	var args []string
	json.Unmarshal(data, &args)

	mustContain(t, args, "update", "CompactBead args")
	mustContain(t, args, "bd-test-abc", "CompactBead args")
	mustContain(t, args, "--metadata", "CompactBead args")
	mustContainSubstring(t, args, "gsd:compact", "CompactBead metadata JSON")
}

// --- QueryTiered tests (needs fake_bd) ---

func TestQueryTiered_ClassifiesBeads(t *testing.T) {
	// Set up a fake query response with mixed beads.
	dir := t.TempDir()

	now := time.Now()
	closedRecent := now.Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	closedOld := now.Add(-100 * 24 * time.Hour).UTC().Format(time.RFC3339)

	// Mix: 1 open (hot), 2 closed (warm and cold based on count).
	queryResponse := `[
		{"id":"b-open","title":"Open Task","status":"open","priority":3,"issue_type":"task","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"},
		{"id":"b-recent","title":"Recent Task","status":"closed","priority":3,"issue_type":"task","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","closed_at":"` + closedRecent + `","close_reason":"done"},
		{"id":"b-old","title":"Old Task","status":"closed","priority":3,"issue_type":"task","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","closed_at":"` + closedOld + `","close_reason":"archived"}
	]`

	queryFile := filepath.Join(dir, "query_response.json")
	os.WriteFile(queryFile, []byte(queryResponse), 0644)
	t.Setenv("FAKE_BD_QUERY_TIERED_RESPONSE", queryFile)

	c := NewClientWithPath(fakeBdPath, t.TempDir())
	ctx := context.Background()

	// warmCount=1: only the most recently closed bead is warm, the rest are cold.
	hot, warm, cold, err := c.QueryTiered(ctx, "gsd:plan", 1)
	if err != nil {
		t.Fatalf("QueryTiered() returned error: %v", err)
	}

	if len(hot) != 1 {
		t.Errorf("QueryTiered() hot count = %d, want 1", len(hot))
	}
	if len(hot) > 0 && hot[0].ID != "b-open" {
		t.Errorf("QueryTiered() hot[0].ID = %q, want \"b-open\"", hot[0].ID)
	}

	if len(warm) != 1 {
		t.Errorf("QueryTiered() warm count = %d, want 1", len(warm))
	}
	if len(warm) > 0 && warm[0].ID != "b-recent" {
		t.Errorf("QueryTiered() warm[0].ID = %q, want \"b-recent\"", warm[0].ID)
	}

	if len(cold) != 1 {
		t.Errorf("QueryTiered() cold count = %d, want 1", len(cold))
	}
	if len(cold) > 0 && cold[0].ID != "b-old" {
		t.Errorf("QueryTiered() cold[0].ID = %q, want \"b-old\"", cold[0].ID)
	}
}

func TestQueryTiered_WarmCountLimitRespected(t *testing.T) {
	// Set up a fake query response with many closed beads.
	dir := t.TempDir()

	now := time.Now()
	// Build 5 closed beads with varying timestamps.
	beads := make([]string, 5)
	for i := 0; i < 5; i++ {
		closedAt := now.Add(time.Duration(-i) * time.Hour).UTC().Format(time.RFC3339)
		beads[i] = `{"id":"b-` + string(rune('a'+i)) + `","title":"Closed ` + string(rune('A'+i)) + `","status":"closed","priority":3,"issue_type":"task","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","closed_at":"` + closedAt + `","close_reason":"done"}`
	}
	queryResponse := `[` + strings.Join(beads, ",") + `]`

	queryFile := filepath.Join(dir, "query_response.json")
	os.WriteFile(queryFile, []byte(queryResponse), 0644)
	t.Setenv("FAKE_BD_QUERY_TIERED_RESPONSE", queryFile)

	c := NewClientWithPath(fakeBdPath, t.TempDir())
	ctx := context.Background()

	// warmCount=2: only top 2 most recent closed beads should be warm.
	_, warm, cold, err := c.QueryTiered(ctx, "gsd:plan", 2)
	if err != nil {
		t.Fatalf("QueryTiered() returned error: %v", err)
	}

	if len(warm) != 2 {
		t.Errorf("QueryTiered(warmCount=2) warm count = %d, want 2", len(warm))
	}
	if len(cold) != 3 {
		t.Errorf("QueryTiered(warmCount=2) cold count = %d, want 3", len(cold))
	}
}

func TestQueryTiered_WarmBeadsHaveCompactSet(t *testing.T) {
	// Set up a fake query response with a warm bead that has gsd:compact in metadata.
	dir := t.TempDir()

	now := time.Now()
	closedAt := now.Add(-1 * time.Hour).UTC().Format(time.RFC3339)
	queryResponse := `[
		{"id":"b-warm","title":"Warm Task","status":"closed","priority":3,"issue_type":"task","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z","closed_at":"` + closedAt + `","close_reason":"feature done","metadata":{"gsd:compact":"Warm Task: feature done"}}
	]`

	queryFile := filepath.Join(dir, "query_response.json")
	os.WriteFile(queryFile, []byte(queryResponse), 0644)
	t.Setenv("FAKE_BD_QUERY_TIERED_RESPONSE", queryFile)

	c := NewClientWithPath(fakeBdPath, t.TempDir())
	ctx := context.Background()

	_, warm, _, err := c.QueryTiered(ctx, "gsd:plan", 5)
	if err != nil {
		t.Fatalf("QueryTiered() returned error: %v", err)
	}

	if len(warm) != 1 {
		t.Fatalf("QueryTiered() warm count = %d, want 1", len(warm))
	}
	// Should use the gsd:compact metadata value.
	if warm[0].Compact != "Warm Task: feature done" {
		t.Errorf("QueryTiered() warm[0].Compact = %q, want \"Warm Task: feature done\"", warm[0].Compact)
	}
}
