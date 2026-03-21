package logging

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
)

func TestInitWritesToStderr(t *testing.T) {
	// Save original stderr and slog default
	origStderr := os.Stderr

	// Create a pipe to capture stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	err = Init("info", "text")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	slog.Info("test message")

	// Close write end and restore stderr
	w.Close()
	os.Stderr = origStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	if buf.Len() == 0 {
		t.Error("expected stderr to have output after slog.Info, but got empty")
	}
}

func TestInitNeverWritesToStdout(t *testing.T) {
	// Save original stdout and slog default
	origStdout := os.Stdout

	// Create a pipe to capture stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	err = Init("info", "text")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	slog.Info("test message stdout check")

	// Close write end and restore stdout
	w.Close()
	os.Stdout = origStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	if buf.Len() != 0 {
		t.Errorf("expected stdout to be empty after slog.Info, but got: %q", buf.String())
	}
}

func TestInitJSONFormat(t *testing.T) {
	// Save original stderr
	origStderr := os.Stderr

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	err = Init("info", "json")
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	slog.Info("json format test")

	w.Close()
	os.Stderr = origStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	output := buf.String()
	if !strings.Contains(output, "{") {
		t.Errorf("expected JSON output containing '{', got: %q", output)
	}
}
