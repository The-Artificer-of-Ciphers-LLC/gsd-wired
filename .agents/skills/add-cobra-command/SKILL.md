---
name: add-cobra-command
description: Creates a new Cobra CLI command following the project's New*Cmd() pattern in internal/cli/. Handles command registration in root.go, flag setup, output rendering, and test scaffolding. Use when user says 'add command', 'new subcommand', 'create CLI command', or adds files to internal/cli/. Do NOT use for MCP tools or hooks.
---
# Add Cobra Command

## Critical

- **stdout is sacred**: stdout is reserved for structured output (text/JSON). ALL logging goes to stderr via `slog`. This is required for MCP stdio protocol compliance.
- **SilenceUsage/SilenceErrors**: The root command sets both to `true`. Never override these in subcommands.
- **Error wrapping**: Always use `fmt.Errorf("context: %w", err)` — never discard error chains.
- **No `os.Exit()` in commands**: Return errors from `RunE`. The root `Execute()` handles exit codes. Exception: `hook.go` uses `os.Exit(2)` for hook protocol.

## Instructions

### Step 1: Create the command file

Create `internal/cli/{command}.go` following the `New*Cmd()` constructor pattern:

```go
package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func New{Command}Cmd() *cobra.Command {
	var jsonMode bool

	cmd := &cobra.Command{
		Use:   "{command-name}",
		Short: "One-line description",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use cmd.OutOrStdout() for output, never os.Stdout directly
			return run{Command}(cmd.OutOrStdout(), jsonMode)
		},
	}

	cmd.Flags().BoolVar(&jsonMode, "json", false, "Output as JSON")
	return cmd
}

func run{Command}(w io.Writer, jsonMode bool) error {
	// Testable business logic here
	return nil
}

func render{Command}(w io.Writer, data []graph.Bead) {
	fmt.Fprintln(w, "Output here")
}
```

**Key decisions:**
- If the command needs database access: use `findBeadsDir()` then `graph.NewClient(beadsDir)`
- If the command needs testable IO: create a `{command}Opts` struct with function fields for dependency injection (see `connect.go` pattern)
- If the command is a stub for a slash command: return `errors.New("... must be run through /gsd-wired:{name} slash command (requires Claude Code)")`

**Verify:** File compiles with `go build ./internal/cli/`

### Step 2: Register in root.go

Add the new command to the `root.AddCommand(...)` call in `internal/cli/root.go`:

```go
root.AddCommand(NewVersionCmd(), NewServeCmd(), /* ...existing... */, New{Command}Cmd())
```

Append to the end of the existing list. Do NOT reorder existing commands.

**Verify:** `go build ./cmd/gsdw/ && ./gsdw {command-name} --help` shows the new command

### Step 3: Implement output rendering

Follow the project's rendering pattern — separate render functions that accept `io.Writer`:

- **Plain text**: Use `fmt.Fprintf(w, ...)` with aligned columns
- **JSON mode**: Use `json.MarshalIndent(data, "", "  ")` then `fmt.Fprintln(w, string(data))`
- **Tree format**: Use `|--` for intermediate items, `+--` for last item (see `ready.go`)
- **Empty results**: For JSON, emit `[]` not `null` — guard with `if data == nil { data = []Type{} }`

**Verify:** Run command and confirm output goes to stdout, logs to stderr: `./gsdw {command} 2>/dev/null` shows clean output

### Step 4: Create test file

Create `internal/cli/{command}_test.go` with these three test categories:

```go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

// 1. Registration test — verifies root knows about the command
func TestRootCmdHas{Command}(t *testing.T) {
	root := NewRootCmd()
	for _, cmd := range root.Commands() {
		if cmd.Use == "{command-name}" {
			return
		}
	}
	t.Errorf("expected '{command-name}' subcommand registered in root")
}

// 2. Render tests — test output formatting directly
func TestRender{Command}(t *testing.T) {
	var buf bytes.Buffer
	render{Command}(&buf, testData)

	out := buf.String()
	if !strings.Contains(out, "expected text") {
		t.Errorf("expected 'expected text' in output, got:\n%s", out)
	}
}

// 3. Business logic tests — test run{Command}() with injected deps
func TestRun{Command}(t *testing.T) {
	var buf bytes.Buffer
	err := run{Command}(&buf, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

Use existing test helpers like `testBead()` and `testPhaseBead()` if your command works with beads.

**Verify:** `go test ./internal/cli/ -run Test.*{Command} -v` passes

### Step 5: Add dependency injection (if command has external deps)

For commands that call external services, file system, or network:

```go
type {command}Opts struct {
	in  io.Reader
	out io.Writer
	// Replace real dependencies with function fields
	loadConfigFn func(dir string) (*connection.Config, error)
	findBeadsDirFn func() (string, error)
}
```

The `New{Command}Cmd()` wires real implementations; tests supply fakes.

**Verify:** Tests don't touch filesystem, network, or database

## Examples

**User says:** "Add a `gsdw wave` command that shows the current execution wave"

**Actions:**
1. Create `internal/cli/wave.go` with `NewWaveCmd()` returning `*cobra.Command`
2. Implement `runWave(w io.Writer, jsonMode bool) error` using `findBeadsDir()` + `graph.NewClient()`
3. Add `renderWave(w io.Writer, beads []graph.Bead)` and `renderWaveJSON(w io.Writer, beads []graph.Bead) error`
4. Register `NewWaveCmd()` in `root.go`'s `AddCommand` call
5. Create `internal/cli/wave_test.go` with `TestRootCmdHasWave`, `TestRenderWave`, `TestRenderWaveJSON`
6. Run `go test ./internal/cli/ -run TestWave -v` and `go build ./cmd/gsdw/`

## Common Issues

**"no beads database found — run gsdw init first"**: The command calls `findBeadsDir()` but there's no `.beads/` directory. Either:
1. Run `gsdw init` in the project root first
2. Set `BEADS_DIR` env var: `export BEADS_DIR=/path/to/.beads`

**Output polluted with log lines**: You're writing logs to stdout. Use `slog.Info()` / `slog.Debug()` (writes to stderr) instead of `fmt.Println()`. For command output, use `fmt.Fprintf(cmd.OutOrStdout(), ...)`.

**Test fails with "expected subcommand registered in root"**: You forgot to add `New{Command}Cmd()` to the `root.AddCommand(...)` call in `root.go`.

**`go build` fails with import cycle**: `internal/cli/` must not import from `cmd/gsdw/`. Business logic that other packages need belongs in `internal/graph/`, `internal/connection/`, or a new `internal/{pkg}/`.

**JSON output emits `null` instead of `[]`**: Guard nil slices before marshaling: `if data == nil { data = []Type{} }`