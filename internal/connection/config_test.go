package connection

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestConfigRoundTrip: marshal/unmarshal Config to JSON and back produces identical struct.
func TestConfigRoundTrip(t *testing.T) {
	cfg := Config{
		ActiveMode: "local",
		Local:      LocalConfig{Host: "127.0.0.1", Port: "3307"},
		Remote:     RemoteConfig{Host: "db.example.com", Port: "3306", User: "dev"},
		Configured: "2026-03-22T00:00:00Z",
	}

	dir := t.TempDir()
	if err := SaveConnection(dir, &cfg); err != nil {
		t.Fatalf("SaveConnection: %v", err)
	}
	got, err := LoadConnection(dir)
	if err != nil {
		t.Fatalf("LoadConnection: %v", err)
	}
	if got == nil {
		t.Fatal("LoadConnection returned nil for existing file")
	}
	if got.ActiveMode != cfg.ActiveMode {
		t.Errorf("ActiveMode: got %q, want %q", got.ActiveMode, cfg.ActiveMode)
	}
	if got.Local.Host != cfg.Local.Host {
		t.Errorf("Local.Host: got %q, want %q", got.Local.Host, cfg.Local.Host)
	}
	if got.Local.Port != cfg.Local.Port {
		t.Errorf("Local.Port: got %q, want %q", got.Local.Port, cfg.Local.Port)
	}
	if got.Remote.Host != cfg.Remote.Host {
		t.Errorf("Remote.Host: got %q, want %q", got.Remote.Host, cfg.Remote.Host)
	}
	if got.Remote.Port != cfg.Remote.Port {
		t.Errorf("Remote.Port: got %q, want %q", got.Remote.Port, cfg.Remote.Port)
	}
	if got.Remote.User != cfg.Remote.User {
		t.Errorf("Remote.User: got %q, want %q", got.Remote.User, cfg.Remote.User)
	}
	if got.Configured != cfg.Configured {
		t.Errorf("Configured: got %q, want %q", got.Configured, cfg.Configured)
	}
}

// TestActiveHostPort_Local: local mode returns Local.Host and Local.Port.
func TestActiveHostPort_Local(t *testing.T) {
	cfg := Config{ActiveMode: "local", Local: LocalConfig{Host: "127.0.0.1", Port: "3307"}}
	host, port := cfg.ActiveHostPort()
	if host != "127.0.0.1" {
		t.Errorf("host: got %q, want %q", host, "127.0.0.1")
	}
	if port != "3307" {
		t.Errorf("port: got %q, want %q", port, "3307")
	}
}

// TestActiveHostPort_Remote: remote mode returns Remote.Host and Remote.Port.
func TestActiveHostPort_Remote(t *testing.T) {
	cfg := Config{ActiveMode: "remote", Remote: RemoteConfig{Host: "db.example.com", Port: "3306"}}
	host, port := cfg.ActiveHostPort()
	if host != "db.example.com" {
		t.Errorf("host: got %q, want %q", host, "db.example.com")
	}
	if port != "3306" {
		t.Errorf("port: got %q, want %q", port, "3306")
	}
}

// TestActiveHostPort_LocalDefaults: zero-value local config returns defaults.
func TestActiveHostPort_LocalDefaults(t *testing.T) {
	cfg := Config{ActiveMode: "local", Local: LocalConfig{}}
	host, port := cfg.ActiveHostPort()
	if host != "127.0.0.1" {
		t.Errorf("host: got %q, want default %q", host, "127.0.0.1")
	}
	if port != "3307" {
		t.Errorf("port: got %q, want default %q", port, "3307")
	}
}

// TestSaveConnectionAtomic: writes to temp dir, connection.json exists, no .tmp file remains.
func TestSaveConnectionAtomic(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{ActiveMode: "local", Local: LocalConfig{Host: "127.0.0.1", Port: "3307"}}

	if err := SaveConnection(dir, cfg); err != nil {
		t.Fatalf("SaveConnection: %v", err)
	}

	// connection.json must exist
	connPath := filepath.Join(dir, "connection.json")
	if _, err := os.Stat(connPath); os.IsNotExist(err) {
		t.Error("connection.json does not exist after SaveConnection")
	}

	// no .tmp file should remain
	tmpPath := connPath + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("connection.json.tmp file remains after SaveConnection")
	}
}

// TestLoadConnection_Missing: returns nil, nil when file does not exist.
func TestLoadConnection_Missing(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadConnection(dir)
	if err != nil {
		t.Fatalf("LoadConnection on missing file returned error: %v", err)
	}
	if cfg != nil {
		t.Errorf("LoadConnection on missing file returned non-nil config: %+v", cfg)
	}
}

// TestLoadConnection_Valid: SaveConnection then LoadConnection returns identical config.
func TestLoadConnection_Valid(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		ActiveMode: "remote",
		Remote:     RemoteConfig{Host: "db.example.com", Port: "3306", User: "admin"},
		Configured: "2026-01-01T00:00:00Z",
	}
	if err := SaveConnection(dir, cfg); err != nil {
		t.Fatalf("SaveConnection: %v", err)
	}
	got, err := LoadConnection(dir)
	if err != nil {
		t.Fatalf("LoadConnection: %v", err)
	}
	if got == nil {
		t.Fatal("LoadConnection returned nil")
	}
	if got.ActiveMode != cfg.ActiveMode {
		t.Errorf("ActiveMode: got %q, want %q", got.ActiveMode, cfg.ActiveMode)
	}
	if got.Remote.Host != cfg.Remote.Host {
		t.Errorf("Remote.Host: got %q, want %q", got.Remote.Host, cfg.Remote.Host)
	}
}

// TestClassifyTCPError_Refused: connection refused error mentions "gsdw container start".
func TestClassifyTCPError_Refused(t *testing.T) {
	err := classifyTCPError(fmt.Errorf("dial tcp: connection refused"), "localhost", "3307")
	if err == nil {
		t.Fatal("classifyTCPError returned nil")
	}
	if !strings.Contains(err.Error(), "gsdw container start") {
		t.Errorf("error %q does not contain 'gsdw container start'", err.Error())
	}
}

// TestClassifyTCPError_DNS: no such host error mentions hostname resolution.
func TestClassifyTCPError_DNS(t *testing.T) {
	err := classifyTCPError(fmt.Errorf("dial tcp: lookup db.example.com: no such host"), "db.example.com", "3306")
	if err == nil {
		t.Fatal("classifyTCPError returned nil")
	}
	if !strings.Contains(err.Error(), "hostname is spelled correctly") {
		t.Errorf("error %q does not contain 'hostname is spelled correctly'", err.Error())
	}
}

// TestClassifyTCPError_Timeout: timeout error mentions VPN/firewall.
func TestClassifyTCPError_Timeout(t *testing.T) {
	err := classifyTCPError(fmt.Errorf("dial tcp: i/o timeout"), "remote.host", "3306")
	if err == nil {
		t.Fatal("classifyTCPError returned nil")
	}
	if !strings.Contains(err.Error(), "VPN/firewall") {
		t.Errorf("error %q does not contain 'VPN/firewall'", err.Error())
	}
}

// TestBuildDSN_UserPassword: user+password format.
func TestBuildDSN_UserPassword(t *testing.T) {
	want := "dev:pass@tcp(host:3306)/"
	got := buildDSN("dev", "pass", "host", "3306")
	if got != want {
		t.Errorf("buildDSN: got %q, want %q", got, want)
	}
}

// TestBuildDSN_UserOnly: user without password format.
func TestBuildDSN_UserOnly(t *testing.T) {
	want := "dev@tcp(host:3306)/"
	got := buildDSN("dev", "", "host", "3306")
	if got != want {
		t.Errorf("buildDSN: got %q, want %q", got, want)
	}
}

// TestBuildDSN_NoAuth: no auth format.
func TestBuildDSN_NoAuth(t *testing.T) {
	want := "tcp(host:3306)/"
	got := buildDSN("", "", "host", "3306")
	if got != want {
		t.Errorf("buildDSN: got %q, want %q", got, want)
	}
}
