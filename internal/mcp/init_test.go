package mcp

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var fakeBdPathMCP string

// TestMain builds the fake_bd binary once for all tests in this package.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "fake_bd_mcp_*")
	if err != nil {
		panic("failed to create temp dir for fake_bd: " + err.Error())
	}
	defer os.RemoveAll(dir)

	fakeBdPathMCP = filepath.Join(dir, "fake_bd")

	// Find the module root to build from correct location.
	moduleRoot, err := findModRoot()
	if err != nil {
		panic("failed to find module root: " + err.Error())
	}

	cmd := exec.Command("go", "build", "-o", fakeBdPathMCP, "./internal/graph/testdata/fake_bd")
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

// TestLazyInitCreatesClient verifies that serverState.init() with existing .beads/ dir
// creates a non-nil graph.Client.
func TestLazyInitCreatesClient(t *testing.T) {
	tmpDir := t.TempDir()
	// Create .beads/ directory so bd init is not triggered.
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	s := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	ctx := context.Background()

	if err := s.init(ctx); err != nil {
		t.Fatalf("serverState.init() returned error: %v", err)
	}
	if s.client == nil {
		t.Fatal("serverState.init() did not create graph.Client (s.client is nil)")
	}
}

// TestLazyInitRunsBdInit verifies that serverState.init() without .beads/ dir
// runs "bd init" with --quiet flag (verify via arg capture).
func TestLazyInitRunsBdInit(t *testing.T) {
	tmpDir := t.TempDir()
	// No .beads/ directory — should trigger bd init.

	captureFile := filepath.Join(tmpDir, "capture.json")
	t.Setenv("FAKE_BD_CAPTURE_FILE", captureFile)

	s := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	ctx := context.Background()

	if err := s.init(ctx); err != nil {
		t.Fatalf("serverState.init() returned error: %v", err)
	}

	data, err := os.ReadFile(captureFile)
	if err != nil {
		t.Fatalf("capture file not written by bd init: %v", err)
	}

	var args []string
	if err := json.Unmarshal(data, &args); err != nil {
		t.Fatalf("capture file is not valid JSON: %v", err)
	}

	// Verify that bd init was called.
	initFound := false
	for _, a := range args {
		if a == "init" {
			initFound = true
			break
		}
	}
	if !initFound {
		t.Errorf("TestLazyInitRunsBdInit: bd init not called; captured args: %v", args)
	}

	// Verify --quiet flag is present.
	quietFound := false
	for _, a := range args {
		if a == "--quiet" {
			quietFound = true
			break
		}
	}
	if !quietFound {
		t.Errorf("TestLazyInitRunsBdInit: --quiet flag not passed to bd init; captured args: %v", args)
	}
}

// TestLazyInitOnlyOnce verifies that calling serverState.init() twice returns the same client pointer.
func TestLazyInitOnlyOnce(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	s := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	ctx := context.Background()

	if err := s.init(ctx); err != nil {
		t.Fatalf("first s.init() returned error: %v", err)
	}
	firstClient := s.client

	if err := s.init(ctx); err != nil {
		t.Fatalf("second s.init() returned error: %v", err)
	}
	secondClient := s.client

	if firstClient != secondClient {
		t.Error("TestLazyInitOnlyOnce: client pointers differ on second call — sync.Once not working")
	}
}

// TestLazyInitErrorStored verifies that serverState.init() with failing bd stores the error
// and returns it on subsequent calls.
func TestLazyInitErrorStored(t *testing.T) {
	tmpDir := t.TempDir()
	// No .beads/ directory — will try to run bd init.

	// Point to a non-existent binary so bd init fails.
	s := &serverState{beadsDir: tmpDir, bdPath: "/nonexistent/bd"}
	ctx := context.Background()

	err := s.init(ctx)
	if err == nil {
		t.Fatal("s.init() expected error with non-existent bd, got nil")
	}
	firstErr := err

	// Second call must return the same stored error.
	err2 := s.init(ctx)
	if err2 == nil {
		t.Fatal("second s.init() expected stored error, got nil")
	}
	if err2.Error() != firstErr.Error() {
		t.Errorf("second s.init() returned different error: got %q, want %q", err2, firstErr)
	}
	if s.client != nil {
		t.Error("s.client should be nil after failed init")
	}
}

// TestLazyInitTimeout verifies that serverState uses a timeout for bd init subprocess.
// We test this indirectly: the runBdInit method uses context.WithTimeout(30s).
// We verify init returns an error when bd hangs past the deadline by using a very short timeout.
func TestLazyInitTimeout(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake bd that sleeps longer than our timeout.
	sleepScript := filepath.Join(tmpDir, "slow_bd")
	if err := os.WriteFile(sleepScript, []byte("#!/bin/sh\nsleep 60\n"), 0755); err != nil {
		t.Fatal(err)
	}

	s := &serverState{beadsDir: tmpDir, bdPath: sleepScript, initTimeout: 100} // 100ms timeout
	ctx := context.Background()

	err := s.init(ctx)
	if err == nil {
		t.Fatal("TestLazyInitTimeout: expected timeout error from slow bd, got nil")
	}
}

// TestLazyInitBatchMode verifies that serverState.client has batchMode=true.
// We verify this by calling a write operation and checking that --dolt-auto-commit=batch appears.
func TestLazyInitBatchMode(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755); err != nil {
		t.Fatal(err)
	}

	captureFile := filepath.Join(tmpDir, "capture.json")
	t.Setenv("FAKE_BD_CAPTURE_FILE", captureFile)

	s := &serverState{beadsDir: tmpDir, bdPath: fakeBdPathMCP}
	ctx := context.Background()

	if err := s.init(ctx); err != nil {
		t.Fatalf("s.init() returned error: %v", err)
	}

	// Call a write operation to verify batch flag is present.
	_, err := s.client.CreatePhase(ctx, 1, "Test Phase", "goal", "acceptance", []string{})
	if err != nil {
		t.Fatalf("CreatePhase() returned error: %v", err)
	}

	data, _ := os.ReadFile(captureFile)
	var args []string
	if err := json.Unmarshal(data, &args); err != nil {
		t.Fatalf("capture file is not valid JSON: %v", err)
	}

	batchFound := false
	for _, a := range args {
		if a == "--dolt-auto-commit=batch" {
			batchFound = true
			break
		}
	}
	if !batchFound {
		t.Errorf("TestLazyInitBatchMode: --dolt-auto-commit=batch not in args: %v", args)
	}
}
