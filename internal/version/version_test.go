package version

import (
	"encoding/json"
	"regexp"
	"testing"
)

func TestStringFormat(t *testing.T) {
	result := String()
	matched, err := regexp.MatchString(`^\d+\.\d+\.\d+ \(.+\)$`, result)
	if err != nil {
		t.Fatalf("regexp error: %v", err)
	}
	if !matched {
		t.Errorf("version.String() = %q, does not match expected format ^\\d+\\.\\d+\\.\\d+ \\(.+\\)$", result)
	}
}

func TestStringContainsFallback(t *testing.T) {
	result := String()
	if len(result) < 5 || result[:5] != "0.1.0" {
		t.Errorf("version.String() = %q, expected to start with '0.1.0'", result)
	}
}

// TestGetInfoFallback verifies GetInfo() returns non-empty values via ReadBuildInfo fallback.
func TestGetInfoFallback(t *testing.T) {
	info := GetInfo()
	if info.Version == "" {
		t.Error("GetInfo().Version should not be empty")
	}
	if info.GoVersion == "" {
		t.Error("GetInfo().GoVersion should not be empty")
	}
	if info.Platform == "" {
		t.Error("GetInfo().Platform should not be empty")
	}
}

// TestBuildInfoString verifies BuildInfo.String() returns "VERSION (COMMIT)" format.
func TestBuildInfoString(t *testing.T) {
	bi := BuildInfo{
		Version:   "1.2.3",
		Commit:    "abc1234",
		Date:      "2026-03-22",
		GoVersion: "go1.26.1",
		Platform:  "darwin/arm64",
	}
	result := bi.String()
	expected := "1.2.3 (abc1234)"
	if result != expected {
		t.Errorf("BuildInfo.String() = %q, want %q", result, expected)
	}
}

// TestBuildInfoStringUnknownCommit verifies String() handles empty commit gracefully.
func TestBuildInfoStringUnknownCommit(t *testing.T) {
	bi := BuildInfo{
		Version: "1.2.3",
		Commit:  "",
	}
	result := bi.String()
	if result == "" {
		t.Error("BuildInfo.String() should not be empty")
	}
}

// TestBuildInfoJSON verifies BuildInfo.JSON() produces valid JSON with all required keys.
func TestBuildInfoJSON(t *testing.T) {
	bi := BuildInfo{
		Version:   "1.2.3",
		Commit:    "abc1234",
		Date:      "2026-03-22",
		GoVersion: "go1.26.1",
		Platform:  "darwin/arm64",
	}
	jsonStr, err := bi.JSON()
	if err != nil {
		t.Fatalf("BuildInfo.JSON() returned error: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("BuildInfo.JSON() output is not valid JSON: %v", err)
	}

	requiredKeys := []string{"version", "commit", "date", "goVersion", "platform"}
	for _, k := range requiredKeys {
		if _, ok := parsed[k]; !ok {
			t.Errorf("BuildInfo.JSON() output missing key %q", k)
		}
	}

	if parsed["version"] != "1.2.3" {
		t.Errorf("expected version=1.2.3, got %v", parsed["version"])
	}
	if parsed["commit"] != "abc1234" {
		t.Errorf("expected commit=abc1234, got %v", parsed["commit"])
	}
}

// TestLdflagsOverride verifies that when ldflags vars are set, GetInfo() uses them.
func TestLdflagsOverride(t *testing.T) {
	// Save original values
	origVersion := version
	origCommit := commit
	origDate := date
	defer func() {
		version = origVersion
		commit = origCommit
		date = origDate
	}()

	version = "9.8.7"
	commit = "deadbeef"
	date = "2026-03-22"

	info := GetInfo()
	if info.Version != "9.8.7" {
		t.Errorf("expected Version=9.8.7, got %s", info.Version)
	}
	if info.Commit != "deadbeef" {
		t.Errorf("expected Commit=deadbeef, got %s", info.Commit)
	}
	if info.Date != "2026-03-22" {
		t.Errorf("expected Date=2026-03-22, got %s", info.Date)
	}
}
