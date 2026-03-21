package graph

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBatchFlagOnWrite verifies that write operations (CreatePhase) with batch mode enabled
// prepend --dolt-auto-commit=batch before the subcommand in the bd args.
func TestBatchFlagOnWrite(t *testing.T) {
	dir := t.TempDir()
	captureFile := filepath.Join(dir, "capture.json")
	t.Setenv("FAKE_BD_CAPTURE_FILE", captureFile)

	c := NewClientWithPathBatch(fakeBdPath, t.TempDir())
	ctx := context.Background()

	_, err := c.CreatePhase(ctx, 1, "Test Phase", "goal", "acceptance", []string{})
	if err != nil {
		t.Fatalf("CreatePhase() returned error: %v", err)
	}

	data, _ := os.ReadFile(captureFile)
	var args []string
	if err := json.Unmarshal(data, &args); err != nil {
		t.Fatalf("capture file is not valid JSON: %v", err)
	}

	// --dolt-auto-commit=batch must appear BEFORE the subcommand ("create").
	batchIdx := -1
	createIdx := -1
	for i, a := range args {
		if a == "--dolt-auto-commit=batch" {
			batchIdx = i
		}
		if a == "create" {
			createIdx = i
		}
	}

	if batchIdx == -1 {
		t.Errorf("TestBatchFlagOnWrite: args %v do not contain '--dolt-auto-commit=batch'", args)
	}
	if createIdx == -1 {
		t.Errorf("TestBatchFlagOnWrite: args %v do not contain 'create' subcommand", args)
	}
	if batchIdx != -1 && createIdx != -1 && batchIdx >= createIdx {
		t.Errorf("TestBatchFlagOnWrite: --dolt-auto-commit=batch (idx %d) must appear BEFORE 'create' (idx %d)", batchIdx, createIdx)
	}
}

// TestBatchFlagNotOnRead verifies that read operations (ListReady) do NOT include
// --dolt-auto-commit=batch even when batch mode is enabled.
func TestBatchFlagNotOnRead(t *testing.T) {
	dir := t.TempDir()
	captureFile := filepath.Join(dir, "capture.json")
	t.Setenv("FAKE_BD_CAPTURE_FILE", captureFile)

	c := NewClientWithPathBatch(fakeBdPath, t.TempDir())
	ctx := context.Background()

	_, err := c.ListReady(ctx)
	if err != nil {
		t.Fatalf("ListReady() returned error: %v", err)
	}

	data, _ := os.ReadFile(captureFile)
	var args []string
	if err := json.Unmarshal(data, &args); err != nil {
		t.Fatalf("capture file is not valid JSON: %v", err)
	}

	for _, a := range args {
		if strings.Contains(a, "dolt-auto-commit") {
			t.Errorf("TestBatchFlagNotOnRead: read operation args contain batch flag: %v", args)
		}
	}
}

// TestFlushWrites verifies that FlushWrites calls "bd dolt commit" with the expected args.
func TestFlushWrites(t *testing.T) {
	dir := t.TempDir()
	captureFile := filepath.Join(dir, "capture.json")
	t.Setenv("FAKE_BD_CAPTURE_FILE", captureFile)

	c := NewClientWithPathBatch(fakeBdPath, t.TempDir())
	ctx := context.Background()

	if err := c.FlushWrites(ctx); err != nil {
		t.Fatalf("FlushWrites() returned error: %v", err)
	}

	data, _ := os.ReadFile(captureFile)
	var args []string
	if err := json.Unmarshal(data, &args); err != nil {
		t.Fatalf("capture file is not valid JSON: %v", err)
	}

	mustContain(t, args, "dolt", "FlushWrites args")
	mustContain(t, args, "commit", "FlushWrites args")
	mustContain(t, args, "--message", "FlushWrites args")
}

// TestFlushWritesError verifies that FlushWrites propagates errors from bd.
func TestFlushWritesError(t *testing.T) {
	// Use a path that points to a binary that always fails.
	dir := t.TempDir()

	// Create a shell script that always exits with an error.
	failScript := filepath.Join(dir, "fail_bd")
	if err := os.WriteFile(failScript, []byte("#!/bin/sh\necho 'bd: dolt commit failed' >&2\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}

	c := NewClientWithPathBatch(failScript, t.TempDir())
	ctx := context.Background()

	err := c.FlushWrites(ctx)
	if err == nil {
		t.Fatal("FlushWrites() expected error from failing bd, got nil")
	}
}
