// fake_bd is a test binary that simulates the bd CLI for unit tests.
// It reads os.Args to determine what to return, allowing tests to run without
// a real bd installation or dolt server.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Canned single bead JSON for create/show/update responses.
const cannedBead = `{"id":"bd-test-abc","title":"Test Bead","status":"open","priority":3,"issue_type":"task","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}`

// Canned bead array for close response.
const cannedBeadArray = `[{"id":"bd-test-abc","title":"Test Bead","status":"closed","priority":3,"issue_type":"task","created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`

// Canned phase bead for list --label gsd:phase responses.
const cannedPhaseBead = `[{"id":"bd-phase-1","title":"Phase 1","status":"open","priority":3,"issue_type":"epic","metadata":{"gsd_phase":1},"labels":["gsd:phase"],"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`

// Canned plan bead for list --label gsd:plan responses.
const cannedPlanBead = `[{"id":"bd-plan-01","title":"Plan 01","status":"open","priority":3,"issue_type":"task","metadata":{"gsd_plan":"02-01"},"labels":["gsd:plan"],"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`

// Canned research epic bead for query label=gsd:research responses.
const cannedResearchBead = `[{"id":"bd-research-1","title":"Research Phase 6","status":"open","priority":3,"issue_type":"epic","metadata":{"gsd_phase":6},"labels":["gsd:research"],"created_at":"2026-01-01T00:00:00Z","updated_at":"2026-01-01T00:00:00Z"}]`

func main() {
	args := os.Args[1:] // strip program name

	// Handle capture mode: write args to FAKE_BD_CAPTURE_FILE if set, then process normally.
	captureFile := os.Getenv("FAKE_BD_CAPTURE_FILE")
	if captureFile != "" {
		data, _ := json.Marshal(args)
		_ = os.WriteFile(captureFile, data, 0644)
	}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "fake_bd: no command given")
		os.Exit(1)
	}

	// Handle global flag: --dolt-auto-commit=batch is a global bd flag that comes
	// before the subcommand. Strip it before dispatching.
	for len(args) > 0 && args[0] != "" && args[0][0] == '-' && args[0] != "--json" {
		args = args[1:]
	}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "fake_bd: no command after global flags")
		os.Exit(1)
	}

	// First arg is the subcommand (before --json is stripped by the real bd).
	// In our case, run() always appends --json so we look at args[0].
	subcmd := args[0]

	switch subcmd {
	case "create":
		fmt.Print(cannedBead)
		os.Exit(0)

	case "list":
		// Check if --label flag is present and what value follows.
		for i, a := range args {
			if a == "--label" && i+1 < len(args) {
				switch args[i+1] {
				case "gsd:phase":
					fmt.Print(cannedPhaseBead)
					os.Exit(0)
				case "gsd:plan":
					fmt.Print(cannedPlanBead)
					os.Exit(0)
				}
			}
		}
		fmt.Print(`[]`)
		os.Exit(0)

	case "ready":
		// Check if FAKE_BD_READY_RESPONSE env var is set to a file path.
		readyFile := os.Getenv("FAKE_BD_READY_RESPONSE")
		if readyFile != "" {
			data, err := os.ReadFile(readyFile)
			if err == nil {
				fmt.Print(string(data))
				os.Exit(0)
			}
		}
		fmt.Print(`[]`)
		os.Exit(0)


	case "show":
		// Check if FAKE_BD_SHOW_RESPONSE env var is set to a file path.
		showFile := os.Getenv("FAKE_BD_SHOW_RESPONSE")
		if showFile != "" {
			data, err := os.ReadFile(showFile)
			if err == nil {
				fmt.Print(string(data))
				os.Exit(0)
			}
		}
		fmt.Print(cannedBead)
		os.Exit(0)

	case "close":
		fmt.Print(cannedBeadArray)
		os.Exit(0)

	case "blocked":
		fmt.Print(`[]`)
		os.Exit(0)

	case "update":
		fmt.Print(cannedBead)
		os.Exit(0)

	case "query":
		// Check if querying by label.
		for i, a := range args {
			if a == "label=gsd:phase" || (i > 0 && args[i-1] == "label" && a == "gsd:phase") {
				fmt.Print(cannedPhaseBead)
				os.Exit(0)
			}
			if strings.HasPrefix(a, "label=gsd:phase") {
				fmt.Print(cannedPhaseBead)
				os.Exit(0)
			}
			if strings.HasPrefix(a, "label=gsd:research") {
				fmt.Print(cannedResearchBead)
				os.Exit(0)
			}
		}
		fmt.Print(`[]`)
		os.Exit(0)

	case "error-json":
		fmt.Print(`{"error":"test error message"}`)
		os.Exit(1)

	case "error-stderr":
		fmt.Fprintln(os.Stderr, "bd: something went wrong")
		os.Exit(1)

	case "dolt":
		// Handle dolt subcommands (e.g., "dolt commit" used by FlushWrites).
		// Writes args to FAKE_BD_CAPTURE_FILE if set, then returns success.
		fmt.Print(`{"status":"ok"}`)
		os.Exit(0)

	case "echo-args":
		data, _ := json.Marshal(args)
		fmt.Print(string(data))
		os.Exit(0)

	case "echo-env":
		fmt.Print(os.Getenv("BEADS_DIR"))
		os.Exit(0)

	case "init":
		// Handle bd init (used by serverState.runBdInit).
		fmt.Print(`{"status":"ok"}`)
		os.Exit(0)

	default:
		fmt.Fprintf(os.Stderr, "fake_bd: unknown command %q\n", subcmd)
		os.Exit(1)
	}
}
