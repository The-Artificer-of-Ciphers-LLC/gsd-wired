package main_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// buildBinary builds the gsdw binary to a temp directory and returns its path.
// The temp directory is cleaned up by the test framework when the test completes.
func buildBinary(t *testing.T) string {
	t.Helper()
	root := repoRoot(t)
	binPath := filepath.Join(t.TempDir(), "gsdw")

	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/gsdw")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}

	if _, err := os.Stat(binPath); err != nil {
		t.Fatalf("binary not found after build at %s: %v", binPath, err)
	}

	return binPath
}

// TestBinaryBuilds verifies that gsdw compiles without errors.
func TestBinaryBuilds(t *testing.T) {
	bin := buildBinary(t)
	if bin == "" {
		t.Fatal("buildBinary returned empty path")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("binary file missing: %v", err)
	}
}

// TestVersionOutput verifies that `gsdw version` exits 0 and produces output
// matching the format `0.1.0 (hash)\n`.
func TestVersionOutput(t *testing.T) {
	bin := buildBinary(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "version")
	stdout, err := cmd.Output()
	if err != nil {
		t.Fatalf("gsdw version failed: %v", err)
	}

	out := string(stdout)
	pattern := regexp.MustCompile(`^0\.1\.0 \(.+\)\n$`)
	if !pattern.MatchString(out) {
		t.Errorf("gsdw version output %q does not match pattern %q", out, pattern.String())
	}
}

// TestHookStdoutPurity verifies that `gsdw hook SessionStart` with valid JSON
// on stdin produces only valid JSON on stdout and nothing on stderr.
func TestHookStdoutPurity(t *testing.T) {
	bin := buildBinary(t)

	input := `{"session_id":"test","transcript_path":"/tmp/t.jsonl","cwd":"/tmp","hook_event_name":"SessionStart"}`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "hook", "SessionStart")
	cmd.Stdin = strings.NewReader(input)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		t.Fatalf("gsdw hook SessionStart failed: %v\nstderr: %s", err, stderrBuf.String())
	}

	// stdout must be valid JSON
	stdoutBytes := stdoutBuf.Bytes()
	var result map[string]any
	if err := json.Unmarshal(stdoutBytes, &result); err != nil {
		t.Errorf("stdout is not valid JSON: %v\ngot: %q", err, string(stdoutBytes))
	}

	// stdout must contain ONLY the JSON response (allow trailing newline, no leading bytes)
	trimmed := bytes.TrimRight(stdoutBytes, "\n")
	var check map[string]any
	if err := json.Unmarshal(trimmed, &check); err != nil {
		t.Errorf("stdout after trimming trailing newline is not clean JSON: %v\ngot: %q", err, string(trimmed))
	}

	// stderr must be empty at default log level (error)
	if got := stderrBuf.String(); got != "" {
		t.Errorf("expected empty stderr at default log level, got: %q", got)
	}
}

// TestHookInvalidEvent verifies that `gsdw hook FakeEvent` with valid JSON
// on stdin exits with code 2, writes an error to stderr, and nothing to stdout.
func TestHookInvalidEvent(t *testing.T) {
	bin := buildBinary(t)

	input := `{"session_id":"test","transcript_path":"/tmp/t.jsonl","cwd":"/tmp","hook_event_name":"FakeEvent"}`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "hook", "FakeEvent")
	cmd.Stdin = strings.NewReader(input)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()
	if err == nil {
		t.Fatal("gsdw hook FakeEvent should have exited non-zero")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T: %v", err, err)
	}
	if exitErr.ExitCode() != 2 {
		t.Errorf("expected exit code 2 for unknown event, got %d", exitErr.ExitCode())
	}

	// stderr must mention the error
	stderrStr := stderrBuf.String()
	if !strings.Contains(stderrStr, "unknown") && !strings.Contains(stderrStr, "error") && !strings.Contains(stderrStr, "FakeEvent") {
		t.Errorf("stderr should contain error information about unknown event, got: %q", stderrStr)
	}

	// stdout must be empty (no partial JSON on error)
	if got := stdoutBuf.String(); got != "" {
		t.Errorf("stdout must be empty on hook error, got: %q", got)
	}
}

// TestServeRespondesToInitialize verifies that `gsdw serve` responds to an
// MCP initialize request with a valid JSON-RPC 2.0 response containing
// the server name "gsd-wired".
func TestServeRespondesToInitialize(t *testing.T) {
	bin := buildBinary(t)

	initRequest := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}` + "\n"

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, bin, "serve")

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("create stdin pipe: %v", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		t.Fatalf("start gsdw serve: %v", err)
	}

	// Write the initialize request
	if _, err := stdinPipe.Write([]byte(initRequest)); err != nil {
		t.Fatalf("write initialize request: %v", err)
	}

	// Wait for a response line to appear (poll with timeout)
	deadline := time.Now().Add(8 * time.Second)
	var responseLine string
	for time.Now().Before(deadline) {
		data := stdoutBuf.Bytes()
		if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
			responseLine = string(data[:idx])
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if responseLine == "" {
		stdinPipe.Close()
		cmd.Wait()
		t.Fatalf("no response line from gsdw serve within timeout\nstderr: %s", stderrBuf.String())
	}

	// Validate the response is JSON-RPC 2.0
	var response map[string]any
	if err := json.Unmarshal([]byte(responseLine), &response); err != nil {
		t.Fatalf("response line is not valid JSON: %v\ngot: %q", err, responseLine)
	}

	if got, want := response["jsonrpc"], "2.0"; got != want {
		t.Errorf("response jsonrpc = %q, want %q", got, want)
	}

	// id must be 1
	id, ok := response["id"].(float64)
	if !ok || id != 1 {
		t.Errorf("response id = %v (%T), want 1", response["id"], response["id"])
	}

	// result must contain serverInfo with name "gsd-wired"
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("response result is not an object, got %T", response["result"])
	}

	serverInfo, ok := result["serverInfo"].(map[string]any)
	if !ok {
		t.Fatalf("response result.serverInfo is not an object, got %T", result["serverInfo"])
	}

	if got, want := serverInfo["name"], "gsd-wired"; got != want {
		t.Errorf("response result.serverInfo.name = %q, want %q", got, want)
	}

	// Close stdin to signal EOF — server should exit cleanly
	stdinPipe.Close()

	// Wait for the process to exit
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			// Exit errors are acceptable on EOF (some servers exit non-zero on disconnect)
			t.Logf("gsdw serve exited with: %v (acceptable on stdin EOF)", err)
		}
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		t.Error("gsdw serve did not exit within 5s after stdin closed")
	}
}
