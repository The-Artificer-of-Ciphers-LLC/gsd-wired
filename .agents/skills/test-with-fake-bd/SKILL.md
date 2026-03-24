---
name: test-with-fake-bd
description: Writes tests using the fake_bd test binary pattern from internal/graph/testdata/fake_bd/. Covers building fake binary, setting FAKE_BD_* env vars for canned responses, capture files for arg/env verification, and serverState/hookState setup. Use when user says 'write tests', 'add test', 'test coverage', or needs to test graph/mcp/hook code that calls bd. Do NOT use for tests that don't invoke the bd CLI binary (e.g., pure utility functions, config parsing, CLI flag tests).
---
# Testing with fake_bd

## Critical

- **Never call a real `bd` binary in tests.** All tests that exercise graph, MCP, or hook code MUST use the fake_bd binary at `internal/graph/testdata/fake_bd/main.go`.
- **Create `.beads/` directory in tmpDir** before initializing `serverState`. If missing, `state.init()` runs `bd init` against the fake binary, which may produce unexpected side effects.
- **Use `t.Setenv()` for all env vars** — it auto-restores after the test. Never use `os.Setenv` directly.
- **One `TestMain` per package** builds the binary once. Do not rebuild per-test unless you're in the hook package (which uses the `buildFakeBd(t)` helper pattern instead).

## Instructions

### Step 1: Determine which package you're testing

| Package | Binary var | Build location | Client mode |
|---------|-----------|----------------|-------------|
| `internal/graph` | `fakeBdPath` | `TestMain` builds `./testdata/fake_bd` | `NewClientWithPath` (non-batch) or `NewClientWithPathBatch` (batch) |
| `internal/mcp` | `fakeBdPathMCP` | `TestMain` builds `./internal/graph/testdata/fake_bd` from module root | `serverState{bdPath: fakeBdPathMCP}` → batch client |
| `internal/hook` | per-test via `buildFakeBd(t)` | Helper builds from repo root each test | `hookState{bdPath: path}` → non-batch client |

Verify: Check if a `TestMain` already exists in the package's `*_test.go` files. If yes, reuse its binary variable. If not, add one.

### Step 2: Add TestMain (if the package doesn't have one)

**For packages at module root (like `internal/graph`):**
```go
var fakeBdPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "fake_bd_*")
	if err != nil {
		panic("failed to create temp dir for fake_bd: " + err.Error())
	}
	defer os.RemoveAll(dir)

	fakeBdPath = filepath.Join(dir, "fake_bd")
	cmd := exec.Command("go", "build", "-o", fakeBdPath, "./testdata/fake_bd")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build fake_bd: " + err.Error())
	}
	os.Exit(m.Run())
}
```

**For packages outside `internal/graph` (like `internal/mcp`, `internal/hook`):**
```go
var fakeBdPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "fake_bd_*")
	if err != nil {
		panic("failed to create temp dir for fake_bd: " + err.Error())
	}
	defer os.RemoveAll(dir)

	fakeBdPath = filepath.Join(dir, "fake_bd")
	moduleRoot, err := findModRoot()
	if err != nil {
		panic("failed to find module root: " + err.Error())
	}
	cmd := exec.Command("go", "build", "-o", fakeBdPath, "./internal/graph/testdata/fake_bd")
	cmd.Dir = moduleRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build fake_bd: " + err.Error())
	}
	os.Exit(m.Run())
}

func findModRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
```

Verify: Run `go test -run TestMain -v ./your/package/` — it should build fake_bd without error.

### Step 3: Write the test function

**Standard test skeleton:**
```go
func TestYourFeature(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	// Set env vars for fake_bd behavior (pick what you need)
	captureFile := filepath.Join(tmpDir, "capture.json")
	t.Setenv("FAKE_BD_CAPTURE_FILE", captureFile)

	// Create client or state
	c := graph.NewClientWithPath(fakeBdPath, tmpDir)
	ctx := context.Background()

	// Call the function under test
	result, err := c.YourMethod(ctx, args...)
	if err != nil {
		t.Fatalf("YourMethod() error: %v", err)
	}

	// Verify captured args
	data, err := os.ReadFile(captureFile)
	if err != nil {
		t.Fatalf("reading capture file: %v", err)
	}
	var args []string
	json.Unmarshal(data, &args)
	mustContain(t, args, "expected-flag", "YourMethod args")
}
```

Verify: The test compiles with `go vet ./your/package/`.

### Step 4: Choose the right env vars for your test goal

| Goal | Env var | Value | What fake_bd does |
|------|---------|-------|-------------------|
| Verify CLI args passed to bd | `FAKE_BD_CAPTURE_FILE` | path to write JSON `[]string` of args | Writes `os.Args` as JSON array |
| Verify env vars passed to bd subprocess | `FAKE_BD_ENV_CAPTURE_FILE` | path to write JSON `map[string]string` | Writes all env vars as JSON map |
| Custom "ready" response | `FAKE_BD_READY_RESPONSE` | path to JSON file with bead array | Reads and returns file contents |
| Custom "show" response | `FAKE_BD_SHOW_RESPONSE` | path to JSON file with bead object | Reads and returns file contents |
| Custom phase query response | `FAKE_BD_QUERY_PHASE_RESPONSE` | path to JSON file | Reads and returns file contents |
| Custom tiered query response | `FAKE_BD_QUERY_TIERED_RESPONSE` | path to JSON file | Reads and returns file contents |

**To supply custom responses**, write the JSON file first:
```go
respData, _ := json.Marshal([]map[string]any{{"id": "bd-001", "title": "Test"}})
respFile := filepath.Join(tmpDir, "ready.json")
os.WriteFile(respFile, respData, 0644)
t.Setenv("FAKE_BD_READY_RESPONSE", respFile)
```

### Step 5: For MCP tool tests, use serverState + connectInProcess

```go
func TestMCPTool(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755)

	state := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	if err := state.init(context.Background()); err != nil {
		t.Fatalf("state.init() failed: %v", err)
	}
	cs := connectInProcess(t, state)

	result, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name:      "your_tool",
		Arguments: map[string]any{"key": "value"},
	})
	if err != nil {
		t.Fatalf("CallTool error: %v", err)
	}
	text := contentText(result)
	// Assert on text...
}
```

### Step 6: For hook tests, use hookState + buildFakeBd

```go
func TestHookHandler(t *testing.T) {
	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755)

	hs := &hookState{bdPath: buildFakeBd(t), beadsDir: tmpDir}
	raw, _ := json.Marshal(YourInput{...})
	var buf bytes.Buffer

	err := handleYourHook(context.Background(), raw, hs, &buf)
	if err != nil {
		t.Fatalf("handler error: %v", err)
	}
	// Assert on buf.String()...
}
```

### Step 7: Use project helper functions for assertions

```go
// Verify an exact arg is present
mustContain(t, args, "--type", "context string")

// Verify a substring appears in any arg
mustContainSubstring(t, args, "gsd:phase", "context string")

// Extract text from MCP result
text := contentText(result)
```

Verify: Run `go test -run TestYourFeature -v ./your/package/` passes.

## Examples

**User says:** "Add a test for the new `UpdateBeadMetadata` graph client method"

**Actions:**
1. Open `internal/graph/client_test.go` — confirm `TestMain` exists with `fakeBdPath`
2. Add test:
```go
func TestUpdateBeadMetadata(t *testing.T) {
	dir := t.TempDir()
	captureFile := filepath.Join(dir, "capture.json")
	t.Setenv("FAKE_BD_CAPTURE_FILE", captureFile)

	c := NewClientWithPath(fakeBdPath, dir)
	_, err := c.UpdateBeadMetadata(context.Background(), "bd-123", map[string]string{"status": "done"})
	if err != nil {
		t.Fatalf("UpdateBeadMetadata() error: %v", err)
	}

	data, _ := os.ReadFile(captureFile)
	var args []string
	json.Unmarshal(data, &args)
	mustContain(t, args, "update", "UpdateBeadMetadata args")
	mustContain(t, args, "bd-123", "UpdateBeadMetadata bead ID")
}
```
3. Run `go test -run TestUpdateBeadMetadata -v ./internal/graph/`

**Result:** Test passes, verifying the method calls `bd update bd-123 ...` with correct args.

## Common Issues

**`panic: failed to build fake_bd` in TestMain:**
1. Verify `internal/graph/testdata/fake_bd/main.go` exists and compiles: `go build ./internal/graph/testdata/fake_bd`
2. If outside `internal/graph`, ensure `cmd.Dir` is set to module root via `findModRoot()`

**`state.init() failed: ... exit status 1`:**
You forgot to create `.beads/` in tmpDir. Add: `os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755)` before `state.init()`.

**Capture file is empty or missing:**
1. Confirm `t.Setenv("FAKE_BD_CAPTURE_FILE", path)` is called BEFORE the client method that triggers the bd call
2. Verify your test is using the fake binary (not a real `bd` from PATH) — check you're passing `fakeBdPath` to `NewClientWithPath`

**`sync.Once` caches errors permanently:**
`serverState.init()` and `hookState.init()` use `sync.Once`. If init fails once, all subsequent calls return the same error. Create a fresh state struct per test or subtest — never share across tests that might fail.

**Test passes locally but fails in CI:**
The fake_bd build requires the Go toolchain. Ensure CI has Go installed and the module cache is warm. Run `go mod download` before tests.

**`mustContain` or `mustContainSubstring` not found:**
These helpers are defined per-package in existing test files. If adding a new test package, copy them from `internal/graph/graph_test.go`.