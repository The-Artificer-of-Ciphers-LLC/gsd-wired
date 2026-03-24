package graph

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/connection"
)

// TestClientRunInjectsConnEnvVarsRemoteMode: Client with remote connConfig passes
// BEADS_DOLT_SERVER_HOST and BEADS_DOLT_SERVER_PORT to bd subprocess.
// In remote mode, bd cannot auto-discover the server, so env vars are needed.
func TestClientRunInjectsConnEnvVarsRemoteMode(t *testing.T) {
	root := t.TempDir()
	gsdwDir := filepath.Join(root, ".gsdw")
	beadsDir := filepath.Join(root, ".beads")
	if err := os.MkdirAll(gsdwDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &connection.Config{
		ActiveMode: "remote",
		Remote:     connection.RemoteConfig{Host: "10.0.0.5", Port: connection.FlexPort("3307"), User: "dev"},
		Configured: "2026-01-01T00:00:00Z",
	}
	if err := connection.SaveConnection(gsdwDir, cfg); err != nil {
		t.Fatal(err)
	}

	captureFile := filepath.Join(root, "env_capture.json")
	t.Setenv("FAKE_BD_ENV_CAPTURE_FILE", captureFile)

	c := NewClientWithPath(fakeBdPath, beadsDir)
	c.connConfig = cfg

	ctx := context.Background()
	_, _ = c.run(ctx, "echo-env")

	data, err := os.ReadFile(captureFile)
	if err != nil {
		t.Fatalf("env capture file not written: %v", err)
	}
	var envMap map[string]string
	if err := json.Unmarshal(data, &envMap); err != nil {
		t.Fatalf("env capture file not valid JSON: %v", err)
	}

	if envMap["BEADS_DOLT_SERVER_HOST"] != "10.0.0.5" {
		t.Errorf("BEADS_DOLT_SERVER_HOST: got %q, want %q", envMap["BEADS_DOLT_SERVER_HOST"], "10.0.0.5")
	}
	if envMap["BEADS_DOLT_SERVER_PORT"] != "3307" {
		t.Errorf("BEADS_DOLT_SERVER_PORT: got %q, want %q", envMap["BEADS_DOLT_SERVER_PORT"], "3307")
	}
}

// TestClientRunLocalModeInjectsEnvVars: Client with local connConfig passes
// BEADS_DOLT_SERVER_HOST/PORT to bd — local mode uses a container-managed server
// that bd doesn't auto-discover, so env vars are needed.
func TestClientRunLocalModeInjectsEnvVars(t *testing.T) {
	root := t.TempDir()
	gsdwDir := filepath.Join(root, ".gsdw")
	beadsDir := filepath.Join(root, ".beads")
	if err := os.MkdirAll(gsdwDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &connection.Config{
		ActiveMode: "local",
		Local:      connection.LocalConfig{Host: "127.0.0.1", Port: connection.FlexPort("3307")},
		Configured: "2026-01-01T00:00:00Z",
	}
	if err := connection.SaveConnection(gsdwDir, cfg); err != nil {
		t.Fatal(err)
	}

	captureFile := filepath.Join(root, "env_capture.json")
	t.Setenv("FAKE_BD_ENV_CAPTURE_FILE", captureFile)

	c := NewClientWithPath(fakeBdPath, beadsDir)
	c.connConfig = cfg

	ctx := context.Background()
	_, _ = c.run(ctx, "echo-env")

	data, err := os.ReadFile(captureFile)
	if err != nil {
		t.Fatalf("env capture file not written: %v", err)
	}
	var envMap map[string]string
	if err := json.Unmarshal(data, &envMap); err != nil {
		t.Fatalf("env capture file not valid JSON: %v", err)
	}

	if envMap["BEADS_DOLT_SERVER_HOST"] != "127.0.0.1" {
		t.Errorf("BEADS_DOLT_SERVER_HOST: got %q, want %q", envMap["BEADS_DOLT_SERVER_HOST"], "127.0.0.1")
	}
	if envMap["BEADS_DOLT_SERVER_PORT"] != "3307" {
		t.Errorf("BEADS_DOLT_SERVER_PORT: got %q, want %q", envMap["BEADS_DOLT_SERVER_PORT"], "3307")
	}
}

// TestClientRunNoConfigNoInjection: Client with nil connConfig does NOT inject connection env vars.
func TestClientRunNoConfigNoInjection(t *testing.T) {
	root := t.TempDir()
	beadsDir := filepath.Join(root, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}

	captureFile := filepath.Join(root, "env_capture.json")
	t.Setenv("FAKE_BD_ENV_CAPTURE_FILE", captureFile)

	c := NewClientWithPath(fakeBdPath, beadsDir)
	// connConfig is nil — no .gsdw/connection.json exists.

	ctx := context.Background()
	_, _ = c.run(ctx, "echo-env")

	data, err := os.ReadFile(captureFile)
	if err != nil {
		t.Fatalf("env capture file not written: %v", err)
	}
	var envMap map[string]string
	if err := json.Unmarshal(data, &envMap); err != nil {
		t.Fatalf("env capture file not valid JSON: %v", err)
	}

	if _, ok := envMap["BEADS_DOLT_SERVER_HOST"]; ok {
		t.Errorf("BEADS_DOLT_SERVER_HOST should not be injected when connConfig is nil, but got %q", envMap["BEADS_DOLT_SERVER_HOST"])
	}
	if _, ok := envMap["BEADS_DOLT_SERVER_PORT"]; ok {
		t.Errorf("BEADS_DOLT_SERVER_PORT should not be injected when connConfig is nil, but got %q", envMap["BEADS_DOLT_SERVER_PORT"])
	}
}

// TestClientConnConfigFromGsdwDir: NewClientWithPath with .gsdw/connection.json loads the config.
func TestClientConnConfigFromGsdwDir(t *testing.T) {
	root := t.TempDir()
	gsdwDir := filepath.Join(root, ".gsdw")
	beadsDir := filepath.Join(root, ".beads")
	if err := os.MkdirAll(gsdwDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &connection.Config{
		ActiveMode: "local",
		Local:      connection.LocalConfig{Host: "127.0.0.1", Port: connection.FlexPort("3307")},
		Configured: "2026-01-01T00:00:00Z",
	}
	if err := connection.SaveConnection(gsdwDir, cfg); err != nil {
		t.Fatal(err)
	}

	// NewClientWithPath derives .gsdw from parent of beadsDir.
	c := NewClientWithPath(fakeBdPath, beadsDir)
	if c.connConfig == nil {
		t.Error("connConfig should be non-nil when .gsdw/connection.json exists")
	}
}

// TestClientConnConfigMissing: NewClientWithPath without .gsdw/connection.json leaves connConfig nil.
func TestClientConnConfigMissing(t *testing.T) {
	root := t.TempDir()
	beadsDir := filepath.Join(root, ".beads")
	if err := os.MkdirAll(beadsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// No .gsdw/connection.json — only .beads/ exists.

	c := NewClientWithPath(fakeBdPath, beadsDir)
	if c.connConfig != nil {
		t.Errorf("connConfig should be nil when no .gsdw/connection.json, got %+v", c.connConfig)
	}
}
