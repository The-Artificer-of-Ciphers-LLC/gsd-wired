// Package compat provides pure-function parsers for .planning/ markdown files.
// These parsers extract bead-equivalent data from GSD's STATE.md, ROADMAP.md,
// and PROJECT.md — the fallback path when .beads/ doesn't exist (COMPAT-01/02).
//
// All parse functions are read-only. No write operations exist in this package (D-09).
package compat

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ---- Compiled patterns (package-level, per D-08 / reqLabelPattern convention) ----

var (
	// STATE.md patterns
	statePhasePattern    = regexp.MustCompile(`Phase:\s+(\d+)\s+of\s+(\d+)`)
	statePlanPattern     = regexp.MustCompile(`Plan:\s+(\d+)\s+of\s+(\d+)`)
	stateProgressPattern = regexp.MustCompile(`Progress:\s+\[.*?\]\s+(\d+%)`)
	stateActivityPattern = regexp.MustCompile(`Last activity:\s+(.+)`)

	// ROADMAP.md patterns
	roadmapPhasePattern = regexp.MustCompile(`-\s+\[(x| )\]\s+\*\*Phase\s+(\d+):\s+(.+?)\*\*`)
	roadmapGoalPattern  = regexp.MustCompile(`\*\*Goal\*\*:\s+(.+)`)
	roadmapPlansPattern = regexp.MustCompile(`\*\*Plans\*\*:\s+(\d+)\s+plans?`)
	phaseDetailHeading  = regexp.MustCompile(`###\s+Phase\s+(\d+):`)

	// PROJECT.md patterns
	projectNamePattern      = regexp.MustCompile(`^#\s+(.+)`)
	projectCoreValueHeading = regexp.MustCompile(`^##\s+Core Value`)
	projectNextHeading      = regexp.MustCompile(`^##\s+`)
)

// ---- Types ----

// ProjectState holds the parsed current position from STATE.md.
type ProjectState struct {
	CurrentPhase int    // 0 if not found
	CurrentPlan  int    // 0 if not found
	TotalPlans   int    // 0 if not found
	LastActivity string // empty if not found
	Progress     string // e.g. "87%" — empty if not found
}

// PhaseEntry holds one phase row from ROADMAP.md.
type PhaseEntry struct {
	Number   int
	Name     string
	Goal     string // empty if not found in Phase Details
	Complete bool   // from [x] checkbox
	Plans    int    // plan count, 0 if not found
}

// FallbackStatus is the combined project state read from .planning/ files.
type FallbackStatus struct {
	ProjectName string
	CoreValue   string
	State       ProjectState
	Phases      []PhaseEntry
}

// ---- ParseState ----

// ParseState extracts current phase, plan progress, last activity, and progress
// bar percentage from STATE.md content. Returns a zero-value ProjectState on
// empty input. Never returns an error — partial results are returned on malformed input.
func ParseState(content string) ProjectState {
	var s ProjectState

	if m := statePhasePattern.FindStringSubmatch(content); m != nil {
		s.CurrentPhase, _ = strconv.Atoi(m[1])
	}
	if m := statePlanPattern.FindStringSubmatch(content); m != nil {
		s.CurrentPlan, _ = strconv.Atoi(m[1])
		s.TotalPlans, _ = strconv.Atoi(m[2])
	}
	if m := stateProgressPattern.FindStringSubmatch(content); m != nil {
		s.Progress = m[1]
	}
	if m := stateActivityPattern.FindStringSubmatch(content); m != nil {
		s.LastActivity = strings.TrimSpace(m[1])
	}

	return s
}

// ---- ParseRoadmap ----

// ParseRoadmap extracts the list of phases from ROADMAP.md content.
// Returns an empty slice on empty input. Never panics on malformed input.
// Goals are populated by scanning the Phase Details section below the phase list.
func ParseRoadmap(content string) []PhaseEntry {
	if content == "" {
		return nil
	}

	// First pass: extract all phase checkbox rows.
	phaseMatches := roadmapPhasePattern.FindAllStringSubmatch(content, -1)
	if len(phaseMatches) == 0 {
		return nil
	}

	// Build phase slice preserving order.
	phases := make([]PhaseEntry, 0, len(phaseMatches))
	indexByNum := make(map[int]int, len(phaseMatches)) // number → slice index
	for _, m := range phaseMatches {
		complete := m[1] == "x"
		num, _ := strconv.Atoi(m[2])
		name := strings.TrimSpace(m[3])
		// Strip trailing description after " - " if present (e.g. "Binary Scaffold - Go binary...")
		if idx := strings.Index(name, " - "); idx >= 0 {
			name = name[:idx]
		}
		indexByNum[num] = len(phases)
		phases = append(phases, PhaseEntry{
			Number:   num,
			Name:     name,
			Complete: complete,
		})
	}

	// Second pass: scan Phase Details sections for Goal and Plans count.
	// Process line by line, tracking which phase detail we are inside.
	lines := strings.Split(content, "\n")
	currentDetailPhase := -1
	inGoal := false
	_ = inGoal

	for _, line := range lines {
		// Detect "### Phase N:" headings that start a detail block.
		if m := phaseDetailHeading.FindStringSubmatch(line); m != nil {
			num, _ := strconv.Atoi(m[1])
			if _, ok := indexByNum[num]; ok {
				currentDetailPhase = num
			} else {
				currentDetailPhase = -1
			}
			continue
		}

		if currentDetailPhase < 0 {
			continue
		}

		idx, ok := indexByNum[currentDetailPhase]
		if !ok {
			continue
		}

		// Extract Goal.
		if phases[idx].Goal == "" {
			if m := roadmapGoalPattern.FindStringSubmatch(line); m != nil {
				phases[idx].Goal = strings.TrimSpace(m[1])
				continue
			}
		}

		// Extract Plans count.
		if phases[idx].Plans == 0 {
			if m := roadmapPlansPattern.FindStringSubmatch(line); m != nil {
				phases[idx].Plans, _ = strconv.Atoi(m[1])
				continue
			}
		}
	}

	return phases
}

// ---- ParseProject ----

// ParseProject extracts the project name (first # heading) and the Core Value
// section content from PROJECT.md. Returns empty strings on empty input or if
// sections are not found. Never panics.
func ParseProject(content string) (name string, coreValue string) {
	if content == "" {
		return "", ""
	}

	lines := strings.Split(content, "\n")
	inCoreValue := false
	var coreValueLines []string

	for _, line := range lines {
		// Extract project name from first H1.
		if name == "" {
			if m := projectNamePattern.FindStringSubmatch(line); m != nil {
				name = strings.TrimSpace(m[1])
				continue
			}
		}

		// Detect Core Value section start.
		if projectCoreValueHeading.MatchString(line) {
			inCoreValue = true
			continue
		}

		// Detect next H2 heading — end of Core Value section.
		if inCoreValue && projectNextHeading.MatchString(line) {
			inCoreValue = false
			continue
		}

		// Accumulate non-empty lines inside Core Value.
		if inCoreValue {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				coreValueLines = append(coreValueLines, trimmed)
			}
		}
	}

	coreValue = strings.Join(coreValueLines, " ")
	return name, coreValue
}

// ---- DetectPlanning ----

// DetectPlanning returns true if dir/.planning/ exists as a directory.
// Returns false if the directory doesn't exist or if an error occurs.
func DetectPlanning(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".planning"))
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ---- BuildFallbackStatus ----

// BuildFallbackStatus reads STATE.md, ROADMAP.md, and PROJECT.md from
// dir/.planning/ and returns a combined FallbackStatus. Missing files are
// non-fatal — a partial result is returned. Returns an error only if the
// .planning directory cannot be accessed at all.
func BuildFallbackStatus(dir string) (FallbackStatus, error) {
	var fs FallbackStatus

	planningDir := filepath.Join(dir, ".planning")

	// Read and parse STATE.md.
	if raw, err := os.ReadFile(filepath.Join(planningDir, "STATE.md")); err == nil {
		fs.State = ParseState(string(raw))
	}

	// Read and parse ROADMAP.md.
	if raw, err := os.ReadFile(filepath.Join(planningDir, "ROADMAP.md")); err == nil {
		fs.Phases = ParseRoadmap(string(raw))
	}

	// Read and parse PROJECT.md.
	if raw, err := os.ReadFile(filepath.Join(planningDir, "PROJECT.md")); err == nil {
		fs.ProjectName, fs.CoreValue = ParseProject(string(raw))
	}

	return fs, nil
}
