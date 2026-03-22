package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/connection"
)

// makeConnectOpts returns a connectOpts with sensible defaults for testing.
// detectServerFn: returns nil (server found) by default.
// healthCheckFn: returns nil (healthy) by default.
// loadConfigFn: returns nil,nil (no existing config) by default.
// saveConfigFn: captures saved config to savedCfg pointer.
// startContainerFn: returns nil (success) by default.
// findGsdwDirFn: returns t.TempDir() by default.
func makeConnectOpts(
	t *testing.T,
	input string,
	savedCfg **connection.Config,
	detectErr error,
	healthErr error,
	loadCfg *connection.Config,
	startErr error,
	gsdwDir string,
) connectOpts {
	t.Helper()
	return connectOpts{
		in:  strings.NewReader(input),
		out: &bytes.Buffer{},
		detectServerFn: func(host, port string) error {
			return detectErr
		},
		healthCheckFn: func(host, port, user, password string) error {
			return healthErr
		},
		loadConfigFn: func(dir string) (*connection.Config, error) {
			return loadCfg, nil
		},
		saveConfigFn: func(dir string, cfg *connection.Config) error {
			if savedCfg != nil {
				*savedCfg = cfg
			}
			return nil
		},
		startContainerFn: func() error {
			return startErr
		},
		findGsdwDirFn: func() string {
			return gsdwDir
		},
	}
}

// TestConnectAutoDetectFound: detectServerFn returns nil (server found), user answers "Y" ->
// saves local config with host 127.0.0.1 port 3307, output contains "Found local Dolt server".
func TestConnectAutoDetectFound(t *testing.T) {
	var saved *connection.Config
	tmpDir := t.TempDir()
	opts := makeConnectOpts(t, "Y\n", &saved, nil, nil, nil, nil, tmpDir)

	if err := runConnect(opts); err != nil {
		t.Fatalf("runConnect returned error: %v", err)
	}

	out := opts.out.(*bytes.Buffer).String()
	if !strings.Contains(out, "Found local Dolt server") {
		t.Errorf("expected 'Found local Dolt server' in output, got:\n%s", out)
	}
	if saved == nil {
		t.Fatal("expected config to be saved, got nil")
	}
	if saved.ActiveMode != "local" {
		t.Errorf("expected ActiveMode='local', got %q", saved.ActiveMode)
	}
	if saved.Local.Host != "127.0.0.1" {
		t.Errorf("expected Local.Host='127.0.0.1', got %q", saved.Local.Host)
	}
	if saved.Local.Port != "3307" {
		t.Errorf("expected Local.Port='3307', got %q", saved.Local.Port)
	}
}

// TestConnectAutoDetectFound_Decline: detectServerFn returns nil, user answers "n" ->
// falls through to choice menu.
func TestConnectAutoDetectFound_Decline(t *testing.T) {
	var saved *connection.Config
	tmpDir := t.TempDir()
	// "n" to auto-detect, then "3" to cancel at choices
	opts := makeConnectOpts(t, "n\n3\n", &saved, nil, nil, nil, nil, tmpDir)

	if err := runConnect(opts); err != nil {
		t.Fatalf("runConnect returned error: %v", err)
	}

	out := opts.out.(*bytes.Buffer).String()
	// Should see choice menu
	if !strings.Contains(out, "Start local container") && !strings.Contains(out, "1)") {
		t.Errorf("expected choice menu in output, got:\n%s", out)
	}
	// No config saved (cancelled)
	if saved != nil {
		t.Errorf("expected no saved config on cancel, got %+v", saved)
	}
}

// TestConnectNoServer_StartContainer: detectServerFn returns error, user picks "1" (start container),
// startContainerFn succeeds -> saves local config, output contains "Container started".
func TestConnectNoServer_StartContainer(t *testing.T) {
	var saved *connection.Config
	tmpDir := t.TempDir()
	detectErr := fmt.Errorf("connection refused")
	opts := makeConnectOpts(t, "1\n", &saved, detectErr, nil, nil, nil, tmpDir)

	if err := runConnect(opts); err != nil {
		t.Fatalf("runConnect returned error: %v", err)
	}

	out := opts.out.(*bytes.Buffer).String()
	if !strings.Contains(out, "Container started") {
		t.Errorf("expected 'Container started' in output, got:\n%s", out)
	}
	if saved == nil {
		t.Fatal("expected config to be saved, got nil")
	}
	if saved.ActiveMode != "local" {
		t.Errorf("expected ActiveMode='local', got %q", saved.ActiveMode)
	}
}

// TestConnectNoServer_ConfigureRemote: detectServerFn returns error, user picks "2",
// enters host/port/user, healthCheckFn succeeds -> saves remote config.
func TestConnectNoServer_ConfigureRemote(t *testing.T) {
	var saved *connection.Config
	tmpDir := t.TempDir()
	detectErr := fmt.Errorf("connection refused")
	// choice=2, host=db.example.com, port=3306, user=dev
	input := "2\ndb.example.com\n3306\ndev\n"
	opts := makeConnectOpts(t, input, &saved, detectErr, nil, nil, nil, tmpDir)

	if err := runConnect(opts); err != nil {
		t.Fatalf("runConnect returned error: %v", err)
	}

	if saved == nil {
		t.Fatal("expected config to be saved, got nil")
	}
	if saved.ActiveMode != "remote" {
		t.Errorf("expected ActiveMode='remote', got %q", saved.ActiveMode)
	}
	if saved.Remote.Host != "db.example.com" {
		t.Errorf("expected Remote.Host='db.example.com', got %q", saved.Remote.Host)
	}
	if saved.Remote.Port != "3306" {
		t.Errorf("expected Remote.Port='3306', got %q", saved.Remote.Port)
	}
	if saved.Remote.User != "dev" {
		t.Errorf("expected Remote.User='dev', got %q", saved.Remote.User)
	}
}

// TestConnectNoServer_Cancel: detectServerFn returns error, user picks "3" ->
// returns nil (no error, no save).
func TestConnectNoServer_Cancel(t *testing.T) {
	var saved *connection.Config
	tmpDir := t.TempDir()
	detectErr := fmt.Errorf("connection refused")
	opts := makeConnectOpts(t, "3\n", &saved, detectErr, nil, nil, nil, tmpDir)

	if err := runConnect(opts); err != nil {
		t.Fatalf("runConnect returned unexpected error: %v", err)
	}
	if saved != nil {
		t.Errorf("expected no saved config on cancel, got %+v", saved)
	}
}

// TestConnectExistingConfig_KeepCurrent: loadConfigFn returns existing config,
// healthCheckFn succeeds, user answers "N" to reconfigure -> returns nil without saving,
// output contains "[OK]".
func TestConnectExistingConfig_KeepCurrent(t *testing.T) {
	var saved *connection.Config
	tmpDir := t.TempDir()
	existing := &connection.Config{
		ActiveMode: "local",
		Local:      connection.LocalConfig{Host: "127.0.0.1", Port: connection.FlexPort("3307")},
	}
	opts := makeConnectOpts(t, "N\n", &saved, nil, nil, existing, nil, tmpDir)

	if err := runConnect(opts); err != nil {
		t.Fatalf("runConnect returned error: %v", err)
	}

	out := opts.out.(*bytes.Buffer).String()
	if !strings.Contains(out, "[OK]") {
		t.Errorf("expected '[OK]' in output, got:\n%s", out)
	}
	if saved != nil {
		t.Errorf("expected no saved config when user declines reconfigure, got %+v", saved)
	}
}

// TestConnectExistingConfig_Reconfigure: loadConfigFn returns existing config,
// user answers "y" to reconfigure -> proceeds to auto-detect flow.
func TestConnectExistingConfig_Reconfigure(t *testing.T) {
	var saved *connection.Config
	tmpDir := t.TempDir()
	existing := &connection.Config{
		ActiveMode: "local",
		Local:      connection.LocalConfig{Host: "127.0.0.1", Port: connection.FlexPort("3307")},
	}
	// "y" to reconfigure, "Y" to use auto-detected server
	opts := makeConnectOpts(t, "y\nY\n", &saved, nil, nil, existing, nil, tmpDir)

	if err := runConnect(opts); err != nil {
		t.Fatalf("runConnect returned error: %v", err)
	}

	// Should have proceeded to auto-detect
	out := opts.out.(*bytes.Buffer).String()
	if !strings.Contains(out, "Scanning") {
		t.Errorf("expected 'Scanning' in output after reconfigure, got:\n%s", out)
	}
}

// TestConnectRemoteFallback_UserConfirms: healthCheckFn returns error for remote,
// user answers "y" to fallback, startContainerFn succeeds, user answers "y" to make default ->
// saves local config as active_mode.
func TestConnectRemoteFallback_UserConfirms(t *testing.T) {
	var saved *connection.Config
	tmpDir := t.TempDir()
	detectErr := fmt.Errorf("connection refused")
	healthErr := fmt.Errorf("connection refused")
	// choice=2, host, port, user, then "y" to fallback, "y" to make default
	input := "2\ndb.example.com\n3306\ndev\ny\ny\n"
	opts := makeConnectOpts(t, input, &saved, detectErr, healthErr, nil, nil, tmpDir)

	if err := runConnect(opts); err != nil {
		t.Fatalf("runConnect returned error: %v", err)
	}

	if saved == nil {
		t.Fatal("expected config to be saved")
	}
	if saved.ActiveMode != "local" {
		t.Errorf("expected ActiveMode='local' after fallback+make-default, got %q", saved.ActiveMode)
	}
}

// TestConnectRemoteFallback_UserDeclines: healthCheckFn returns error for remote,
// user answers "n" to fallback -> returns error (connection failed).
func TestConnectRemoteFallback_UserDeclines(t *testing.T) {
	tmpDir := t.TempDir()
	detectErr := fmt.Errorf("connection refused")
	healthErr := fmt.Errorf("connection refused")
	// choice=2, host, port, user, then "n" to fallback
	input := "2\ndb.example.com\n3306\ndev\nn\n"
	opts := makeConnectOpts(t, input, nil, detectErr, healthErr, nil, nil, tmpDir)

	err := runConnect(opts)
	if err == nil {
		t.Error("expected error when user declines fallback, got nil")
	}
}

// TestConnectFallback_SessionOnly: healthCheckFn returns error for remote, user answers "y"
// to fallback, startContainerFn succeeds, user answers "n" to make default ->
// saves config with active_mode still "remote" but session uses local.
func TestConnectFallback_SessionOnly(t *testing.T) {
	var saved *connection.Config
	tmpDir := t.TempDir()
	detectErr := fmt.Errorf("connection refused")
	healthErr := fmt.Errorf("connection refused")
	// choice=2, host, port, user, "y" fallback, "n" make-default
	input := "2\ndb.example.com\n3306\ndev\ny\nn\n"
	opts := makeConnectOpts(t, input, &saved, detectErr, healthErr, nil, nil, tmpDir)

	if err := runConnect(opts); err != nil {
		t.Fatalf("runConnect returned error: %v", err)
	}

	if saved == nil {
		t.Fatal("expected config to be saved")
	}
	// active_mode stays "remote" when user chooses not to make local the default
	if saved.ActiveMode != "remote" {
		t.Errorf("expected ActiveMode='remote' for session-only fallback, got %q", saved.ActiveMode)
	}
}

// TestConnectRemoteDefaultPort: user enters empty port for remote -> defaults to "3306".
func TestConnectRemoteDefaultPort(t *testing.T) {
	var saved *connection.Config
	tmpDir := t.TempDir()
	detectErr := fmt.Errorf("connection refused")
	// choice=2, host=db.example.com, port="" (empty -> default 3306), user=dev
	input := "2\ndb.example.com\n\ndev\n"
	opts := makeConnectOpts(t, input, &saved, detectErr, nil, nil, nil, tmpDir)

	if err := runConnect(opts); err != nil {
		t.Fatalf("runConnect returned error: %v", err)
	}

	if saved == nil {
		t.Fatal("expected config to be saved")
	}
	if saved.Remote.Port != "3306" {
		t.Errorf("expected default Remote.Port='3306', got %q", saved.Remote.Port)
	}
}

// TestConnectNoGsdwDir: findGsdwDirFn returns "" -> returns error "no .gsdw/ directory found".
func TestConnectNoGsdwDir(t *testing.T) {
	opts := makeConnectOpts(t, "", nil, nil, nil, nil, nil, "")

	err := runConnect(opts)
	if err == nil {
		t.Fatal("expected error when gsdwDir is empty, got nil")
	}
	if !strings.Contains(err.Error(), ".gsdw/") {
		t.Errorf("expected error about .gsdw/, got: %v", err)
	}
}
