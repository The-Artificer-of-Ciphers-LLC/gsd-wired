package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/container"
)

// --- Fake runtimes for testing ---

type fakeRuntime struct {
	name      string
	binary    string
	startArgs []string
	stopArgs  []string
}

func (f *fakeRuntime) Name() string                              { return f.name }
func (f *fakeRuntime) Binary() string                           { return f.binary }
func (f *fakeRuntime) StartArgs(_ container.ContainerConfig) []string { return f.startArgs }
func (f *fakeRuntime) StopArgs() []string                       { return f.stopArgs }
func (f *fakeRuntime) IsRunningArgs() []string                  { return nil }

// --- Helper: build startOpts with all functions injected ---

func makeStartOpts(rt container.Runtime, detectErr error) startOpts {
	return startOpts{
		force:        false,
		port:         "3307",
		beadsDoltDir: ".beads/dolt",
		detectFn: func(_ container.DetectOpts) (container.Runtime, error) {
			return rt, detectErr
		},
		composeFn: func(dir string, cfg container.ContainerConfig, force bool) (string, error) {
			return dir + "/gsdw.compose.yaml", nil
		},
		execFn: func(name string, args ...string) error {
			return nil
		},
		checkPort: func(port string) error {
			return nil
		},
		statFn: func(path string) error {
			return nil // beads dir exists
		},
	}
}

// --- NewContainerCmd structure tests ---

func TestContainerCmdHasStartAndStop(t *testing.T) {
	cmd := NewContainerCmd()

	if cmd.Use != "container" {
		t.Errorf("expected Use=container, got %q", cmd.Use)
	}

	var hasStart, hasStop bool
	for _, sub := range cmd.Commands() {
		switch sub.Use {
		case "start":
			hasStart = true
		case "stop":
			hasStop = true
		}
	}

	if !hasStart {
		t.Error("container command missing 'start' subcommand")
	}
	if !hasStop {
		t.Error("container command missing 'stop' subcommand")
	}
}

func TestContainerStartHasForceFlag(t *testing.T) {
	cmd := NewContainerCmd()

	for _, sub := range cmd.Commands() {
		if sub.Use == "start" {
			f := sub.Flags().Lookup("force")
			if f == nil {
				t.Error("start command missing --force flag")
			}
			return
		}
	}
	t.Error("start subcommand not found")
}

func TestContainerStartHasPortFlag(t *testing.T) {
	cmd := NewContainerCmd()
	for _, sub := range cmd.Commands() {
		if sub.Use == "start" {
			f := sub.Flags().Lookup("port")
			if f == nil {
				t.Fatal("start command missing --port flag")
			}
			if f.DefValue != "3307" {
				t.Errorf("expected default port 3307, got %q", f.DefValue)
			}
			return
		}
	}
	t.Error("start subcommand not found")
}

// --- runContainerStart tests ---

func TestRunContainerStart_DetectsRuntime(t *testing.T) {
	rt := &fakeRuntime{name: "docker", binary: "/usr/local/bin/docker", startArgs: []string{"run"}}
	opts := makeStartOpts(rt, nil)

	var buf bytes.Buffer
	if err := runContainerStart(&buf, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Using runtime: docker") {
		t.Errorf("expected 'Using runtime: docker' in output, got:\n%s", out)
	}
}

func TestRunContainerStart_NoRuntime_ReturnsError(t *testing.T) {
	opts := makeStartOpts(nil, container.ErrNoRuntime)

	var buf bytes.Buffer
	err := runContainerStart(&buf, opts)
	if err == nil {
		t.Fatal("expected error when no runtime available")
	}

	// Should contain install guidance
	combined := buf.String() + err.Error()
	if !strings.Contains(combined, "install") && !strings.Contains(combined, "Docker") {
		t.Errorf("expected install guidance in output or error, got:\n%s\n%v", combined, err)
	}
}

func TestRunContainerStart_MissingBeadsDir_ReturnsError(t *testing.T) {
	rt := &fakeRuntime{name: "docker", binary: "/usr/local/bin/docker"}
	opts := makeStartOpts(rt, nil)
	opts.statFn = func(path string) error {
		return os.ErrNotExist
	}

	var buf bytes.Buffer
	err := runContainerStart(&buf, opts)
	if err == nil {
		t.Fatal("expected error when .beads/dolt/ missing")
	}

	if !strings.Contains(err.Error(), "bd init") {
		t.Errorf("expected 'bd init' guidance, got: %v", err)
	}
}

func TestRunContainerStart_PortOccupied_ReturnsError(t *testing.T) {
	rt := &fakeRuntime{name: "docker", binary: "/usr/local/bin/docker"}
	opts := makeStartOpts(rt, nil)
	opts.checkPort = func(port string) error {
		return fmt.Errorf("port %s already in use", port)
	}

	var buf bytes.Buffer
	err := runContainerStart(&buf, opts)
	if err == nil {
		t.Fatal("expected error when port occupied")
	}

	if !strings.Contains(err.Error(), "3307") {
		t.Errorf("expected port 3307 in error, got: %v", err)
	}
}

func TestRunContainerStart_DockerCallsComposeFragment(t *testing.T) {
	rt := &fakeRuntime{name: "docker", binary: "/usr/local/bin/docker", startArgs: []string{"run"}}
	opts := makeStartOpts(rt, nil)

	composeCalled := false
	opts.composeFn = func(dir string, cfg container.ContainerConfig, force bool) (string, error) {
		composeCalled = true
		return dir + "/gsdw.compose.yaml", nil
	}

	var buf bytes.Buffer
	if err := runContainerStart(&buf, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !composeCalled {
		t.Error("expected composeFn to be called for docker runtime")
	}
}

func TestRunContainerStart_PodmanCallsComposeFragment(t *testing.T) {
	rt := &fakeRuntime{name: "podman", binary: "/usr/local/bin/podman", startArgs: []string{"run"}}
	opts := makeStartOpts(rt, nil)

	composeCalled := false
	opts.composeFn = func(dir string, cfg container.ContainerConfig, force bool) (string, error) {
		composeCalled = true
		return dir + "/gsdw.compose.yaml", nil
	}

	var buf bytes.Buffer
	if err := runContainerStart(&buf, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !composeCalled {
		t.Error("expected composeFn to be called for podman runtime")
	}
}

func TestRunContainerStart_AppleContainerSkipsComposeFragment(t *testing.T) {
	rt := &fakeRuntime{name: "apple-container", binary: "/usr/local/bin/container", startArgs: []string{"run"}}
	opts := makeStartOpts(rt, nil)

	composeCalled := false
	opts.composeFn = func(dir string, cfg container.ContainerConfig, force bool) (string, error) {
		composeCalled = true
		return "", nil
	}

	var buf bytes.Buffer
	if err := runContainerStart(&buf, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if composeCalled {
		t.Error("composeFn should NOT be called for apple-container runtime")
	}
}

func TestRunContainerStart_ForcePassedToCompose(t *testing.T) {
	rt := &fakeRuntime{name: "docker", binary: "/usr/local/bin/docker", startArgs: []string{"run"}}
	opts := makeStartOpts(rt, nil)
	opts.force = true

	forceReceived := false
	opts.composeFn = func(dir string, cfg container.ContainerConfig, force bool) (string, error) {
		forceReceived = force
		return dir + "/gsdw.compose.yaml", nil
	}

	var buf bytes.Buffer
	if err := runContainerStart(&buf, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !forceReceived {
		t.Error("expected force=true to be passed to composeFn")
	}
}

func TestRunContainerStart_PrintsStartCommand(t *testing.T) {
	rt := &fakeRuntime{
		name:      "docker",
		binary:    "/usr/local/bin/docker",
		startArgs: []string{"run", "-d", "--name", "gsdw-dolt"},
	}
	opts := makeStartOpts(rt, nil)

	var buf bytes.Buffer
	if err := runContainerStart(&buf, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	// Should print the binary + startArgs
	if !strings.Contains(out, "docker") {
		t.Errorf("expected docker command in output, got:\n%s", out)
	}
}

func TestRunContainerStart_PrintsSuccessInfo(t *testing.T) {
	rt := &fakeRuntime{name: "docker", binary: "/usr/local/bin/docker", startArgs: []string{"run"}}
	opts := makeStartOpts(rt, nil)

	var buf bytes.Buffer
	if err := runContainerStart(&buf, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "3307") {
		t.Errorf("expected port 3307 in output, got:\n%s", out)
	}
	if !strings.Contains(out, ".beads/dolt") {
		t.Errorf("expected beads dir in output, got:\n%s", out)
	}
}

// --- runContainerStop tests ---

func makeStopOpts(rt container.Runtime, detectErr error) stopOpts {
	return stopOpts{
		detectFn: func(_ container.DetectOpts) (container.Runtime, error) {
			return rt, detectErr
		},
		execFn: func(name string, args ...string) error {
			return nil
		},
	}
}

func TestRunContainerStop_CallsStopArgs(t *testing.T) {
	rt := &fakeRuntime{
		name:     "docker",
		binary:   "/usr/local/bin/docker",
		stopArgs: []string{"stop", "gsdw-dolt"},
	}
	opts := makeStopOpts(rt, nil)

	execArgs := []string{}
	opts.execFn = func(name string, args ...string) error {
		execArgs = args
		return nil
	}

	var buf bytes.Buffer
	if err := runContainerStop(&buf, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(execArgs) == 0 {
		t.Error("expected execFn to be called with stop args")
	}
	if execArgs[0] != "stop" {
		t.Errorf("expected first arg to be 'stop', got %q", execArgs[0])
	}
}

func TestRunContainerStop_PrintsStopped(t *testing.T) {
	rt := &fakeRuntime{name: "docker", binary: "/usr/local/bin/docker", stopArgs: []string{"stop", "gsdw-dolt"}}
	opts := makeStopOpts(rt, nil)

	var buf bytes.Buffer
	if err := runContainerStop(&buf, opts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "stopped") {
		t.Errorf("expected 'stopped' in output, got:\n%s", out)
	}
}

func TestRunContainerStop_NoRuntime_ReturnsError(t *testing.T) {
	opts := makeStopOpts(nil, container.ErrNoRuntime)

	var buf bytes.Buffer
	err := runContainerStop(&buf, opts)
	if err == nil {
		t.Fatal("expected error when no runtime")
	}
}

func TestRunContainerStop_ExecError_ReturnsError(t *testing.T) {
	rt := &fakeRuntime{name: "docker", binary: "/usr/local/bin/docker", stopArgs: []string{"stop", "gsdw-dolt"}}
	opts := makeStopOpts(rt, nil)
	opts.execFn = func(name string, args ...string) error {
		return errors.New("container not found")
	}

	var buf bytes.Buffer
	err := runContainerStop(&buf, opts)
	if err == nil {
		t.Fatal("expected error on exec failure")
	}
}
