package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// verifyPhaseArgs holds the arguments for the verify_phase MCP tool.
type verifyPhaseArgs struct {
	PhaseNum   int    `json:"phase_num"`
	ProjectDir string `json:"project_dir"`
}

// criterionResult holds the verification result for a single acceptance criterion.
type criterionResult struct {
	Criterion string `json:"criterion"`
	Passed    bool   `json:"passed"`
	Method    string `json:"method"`  // "file_exists", "go_test", "grep", "manual"
	Detail    string `json:"detail"`
}

// verifyPhaseResult is the response for the verify_phase MCP tool.
type verifyPhaseResult struct {
	PhaseNum int               `json:"phase_num"`
	Passed   bool              `json:"passed"`
	Results  []criterionResult `json:"results"`
	Failed   []string          `json:"failed"`
}

// handleVerifyPhase implements the verify_phase MCP tool.
// It checks the phase's acceptance criteria against the codebase state (per D-09).
func handleVerifyPhase(ctx context.Context, state *serverState, args verifyPhaseArgs) (*mcpsdk.CallToolResult, error) {
	if err := state.init(ctx); err != nil {
		return toolError(err.Error()), nil
	}

	// Find the phase epic bead.
	epics, err := state.client.QueryByLabel(ctx, "gsd:phase")
	if err != nil {
		return toolError("failed to query phase epics: " + err.Error()), nil
	}

	var phaseAcceptance string
	var found bool
	for _, bead := range epics {
		if phaseNumFromMeta(bead.Metadata) == args.PhaseNum {
			phaseAcceptance = bead.AcceptanceCriteria
			found = true
			break
		}
	}

	if !found {
		return toolError(fmt.Sprintf("no phase epic found for phase %d", args.PhaseNum)), nil
	}

	// Parse acceptance criteria: one criterion per line, strip numbering and bullet prefixes.
	lines := strings.Split(phaseAcceptance, "\n")
	var criteria []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Strip leading "N. " or "- " prefixes.
		if len(line) > 2 && line[1] == '.' && line[2] == ' ' {
			line = strings.TrimSpace(line[2:])
		} else if strings.HasPrefix(line, "- ") {
			line = strings.TrimSpace(line[2:])
		}
		if line != "" {
			criteria = append(criteria, line)
		}
	}

	// Resolve project_dir: use args.ProjectDir if set, otherwise use the beads directory.
	projectDir := args.ProjectDir
	if projectDir == "" {
		projectDir = state.beadsDir
	}

	// Check each criterion.
	results := make([]criterionResult, 0, len(criteria))
	var failed []string

	for _, criterion := range criteria {
		cr := checkCriterion(ctx, criterion, projectDir)
		results = append(results, cr)
		if !cr.Passed {
			failed = append(failed, criterion)
		}
	}

	// Determine overall pass: all criteria must pass.
	allPassed := len(failed) == 0

	return toolResult(&verifyPhaseResult{
		PhaseNum: args.PhaseNum,
		Passed:   allPassed,
		Results:  results,
		Failed:   failed,
	})
}

// knownExtensions are file extensions that indicate a path in a criterion.
var knownExtensions = []string{".go", ".ts", ".md", ".json", ".yaml", ".toml", ".sh", ".txt"}

// extractFilePath looks for a file path in a criterion string.
// Returns the path if found, empty string if not.
func extractFilePath(criterion string) string {
	words := strings.Fields(criterion)
	for _, word := range words {
		// Check for path separators.
		if strings.Contains(word, "/") {
			// Strip trailing punctuation.
			word = strings.TrimRight(word, ".,;:!?)")
			return word
		}
		// Check for known file extensions.
		for _, ext := range knownExtensions {
			if strings.HasSuffix(word, ext) {
				word = strings.TrimRight(word, ".,;:!?)")
				return word
			}
		}
	}
	return ""
}

// hasUppercaseIdentifier checks if the criterion contains an uppercase Go identifier pattern
// (e.g. a type name like "HandleExecuteWave" or "ExecuteWaveResult").
func hasUppercaseIdentifier(criterion string) bool {
	words := strings.Fields(criterion)
	for _, word := range words {
		word = strings.Trim(word, ".,;:!?)(\"'`")
		if len(word) >= 2 && unicode.IsUpper(rune(word[0])) && unicode.IsLetter(rune(word[1])) {
			return true
		}
	}
	return false
}

// checkCriterion evaluates a single acceptance criterion against the project directory.
func checkCriterion(ctx context.Context, criterion, projectDir string) criterionResult {
	lower := strings.ToLower(criterion)

	// 1. File path check: criterion mentions a known file extension or path separator.
	if filePath := extractFilePath(criterion); filePath != "" {
		fullPath := filePath
		if !filepath.IsAbs(filePath) {
			fullPath = filepath.Join(projectDir, filePath)
		}
		_, err := os.Stat(fullPath)
		detail := ""
		passed := err == nil
		if !passed {
			detail = "file not found: " + fullPath
		}
		return criterionResult{
			Criterion: criterion,
			Passed:    passed,
			Method:    "file_exists",
			Detail:    detail,
		}
	}

	// 2. Go test check: criterion mentions "test" (case-insensitive).
	if strings.Contains(lower, "test") {
		goTestCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		cmd := exec.CommandContext(goTestCtx, "go", "test", "./...")
		cmd.Dir = projectDir
		out, err := cmd.CombinedOutput()
		passed := err == nil
		detail := ""
		if !passed {
			detail = strings.TrimSpace(string(out))
			if len(detail) > 500 {
				detail = detail[:500] + "..."
			}
		}
		return criterionResult{
			Criterion: criterion,
			Passed:    passed,
			Method:    "go_test",
			Detail:    detail,
		}
	}

	// 3. Uppercase identifier pattern: use grep-like scan — mark as manual for v1.
	if hasUppercaseIdentifier(criterion) {
		return criterionResult{
			Criterion: criterion,
			Passed:    false,
			Method:    "manual",
			Detail:    "contains identifier pattern — manual verification required",
		}
	}

	// 4. Default: manual verification required (conservative).
	return criterionResult{
		Criterion: criterion,
		Passed:    false,
		Method:    "manual",
		Detail:    "no automated check available — manual verification required",
	}
}
