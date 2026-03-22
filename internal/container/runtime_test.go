package container_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/container"
)

// makeFakeBinary creates a minimal executable at dir/name that exits 0.
func makeFakeBinary(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	content := "#!/bin/sh\necho \"" + name + " version 1.0.0\"\n"
	if runtime.GOOS == "windows" {
		content = "@echo off\necho " + name + " version 1.0.0\n"
		p += ".bat"
	}
	if err := os.WriteFile(p, []byte(content), 0o755); err != nil {
		t.Fatalf("makeFakeBinary(%s): %v", name, err)
	}
	return p
}

// defaultOpts returns DetectOpts with the given lookPath, os version and arch injected.
func defaultOpts(lookPath func(string) (string, error), major, minor int, arch string) container.DetectOpts {
	return container.DetectOpts{
		LookPath:      lookPath,
		MacOSVersion:  func() (int, int, error) { return major, minor, nil },
		Arch:          func() string { return arch },
	}
}

// makeLookPath returns a LookPath function that searches only dir for binaries.
func makeLookPath(dir string) func(string) (string, error) {
	return func(binary string) (string, error) {
		p := filepath.Join(dir, binary)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		return "", fmt.Errorf("%s not found", binary)
	}
}

// TestDetectRuntime_AppleContainer verifies that Apple Container is returned on macOS 26+ arm64.
func TestDetectRuntime_AppleContainer(t *testing.T) {
	dir := t.TempDir()
	makeFakeBinary(t, dir, "container")

	opts := defaultOpts(makeLookPath(dir), 26, 0, "arm64")
	rt, err := container.DetectRuntime(opts)
	if err != nil {
		t.Fatalf("DetectRuntime: unexpected error: %v", err)
	}
	if rt.Name() != "apple-container" {
		t.Errorf("expected name='apple-container', got %q", rt.Name())
	}
}

// TestDetectRuntime_Docker verifies Docker is returned when Apple Container is unavailable.
func TestDetectRuntime_Docker(t *testing.T) {
	dir := t.TempDir()
	// No 'container' binary — only docker.
	makeFakeBinary(t, dir, "docker")

	opts := defaultOpts(makeLookPath(dir), 15, 0, "amd64")
	rt, err := container.DetectRuntime(opts)
	if err != nil {
		t.Fatalf("DetectRuntime: unexpected error: %v", err)
	}
	if rt.Name() != "docker" {
		t.Errorf("expected name='docker', got %q", rt.Name())
	}
}

// TestDetectRuntime_Podman verifies Podman is returned when Docker is also unavailable.
func TestDetectRuntime_Podman(t *testing.T) {
	dir := t.TempDir()
	// Neither 'container' nor 'docker' — only podman.
	makeFakeBinary(t, dir, "podman")

	opts := defaultOpts(makeLookPath(dir), 15, 0, "amd64")
	rt, err := container.DetectRuntime(opts)
	if err != nil {
		t.Fatalf("DetectRuntime: unexpected error: %v", err)
	}
	if rt.Name() != "podman" {
		t.Errorf("expected name='podman', got %q", rt.Name())
	}
}

// TestDetectRuntime_NoRuntime verifies an error is returned when nothing is available.
func TestDetectRuntime_NoRuntime(t *testing.T) {
	dir := t.TempDir()
	// Empty dir — no binaries.
	opts := defaultOpts(makeLookPath(dir), 15, 0, "amd64")
	_, err := container.DetectRuntime(opts)
	if err == nil {
		t.Fatal("expected error when no runtime available, got nil")
	}
	// Should mention installing Docker or Podman.
	if !strings.Contains(err.Error(), "no container runtime") {
		t.Errorf("error %q should mention 'no container runtime'", err.Error())
	}
}

// TestDetectRuntime_AppleContainerGate_PreMacOS26 verifies that on macOS < 26, Apple Container
// is skipped even when the 'container' binary is present.
func TestDetectRuntime_AppleContainerGate_PreMacOS26(t *testing.T) {
	dir := t.TempDir()
	makeFakeBinary(t, dir, "container")
	makeFakeBinary(t, dir, "docker")

	// macOS 15 — should skip Apple Container and use Docker.
	opts := defaultOpts(makeLookPath(dir), 15, 0, "arm64")
	rt, err := container.DetectRuntime(opts)
	if err != nil {
		t.Fatalf("DetectRuntime: unexpected error: %v", err)
	}
	if rt.Name() != "docker" {
		t.Errorf("on macOS 15, expected fallback to 'docker', got %q", rt.Name())
	}
}

// TestDetectRuntime_AppleContainerGate_NonArm64 verifies Apple Container requires arm64.
func TestDetectRuntime_AppleContainerGate_NonArm64(t *testing.T) {
	dir := t.TempDir()
	makeFakeBinary(t, dir, "container")
	makeFakeBinary(t, dir, "docker")

	// macOS 26 but NOT arm64 — should skip Apple Container.
	opts := defaultOpts(makeLookPath(dir), 26, 0, "amd64")
	rt, err := container.DetectRuntime(opts)
	if err != nil {
		t.Fatalf("DetectRuntime: unexpected error: %v", err)
	}
	if rt.Name() != "docker" {
		t.Errorf("on non-arm64, expected fallback to 'docker', got %q", rt.Name())
	}
}

// cfg returns a ContainerConfig for use in StartArgs tests.
func cfg() container.ContainerConfig {
	return container.ContainerConfig{
		BeadsDoltDir: "/home/user/.beads/dolt",
		HostPort:     "3307",
	}
}

// TestStartArgs_Docker verifies DockerRuntime.StartArgs produces correct docker run args.
func TestStartArgs_Docker(t *testing.T) {
	dir := t.TempDir()
	makeFakeBinary(t, dir, "docker")

	opts := defaultOpts(makeLookPath(dir), 15, 0, "amd64")
	rt, err := container.DetectRuntime(opts)
	if err != nil {
		t.Fatalf("DetectRuntime: %v", err)
	}
	if rt.Name() != "docker" {
		t.Skipf("need docker runtime, got %s", rt.Name())
	}

	args := rt.StartArgs(cfg())

	// Must start with "run".
	if len(args) == 0 || args[0] != "run" {
		t.Errorf("StartArgs[0] should be 'run', got %v", args)
	}
	joined := strings.Join(args, " ")

	checks := []string{
		"-d",
		"--name gsdw-dolt",
		"127.0.0.1:3307:3306",
		"DOLT_ROOT_HOST=%",
		"/home/user/.beads/dolt:/var/lib/dolt",
		"dolthub/dolt-sql-server:latest",
	}
	for _, want := range checks {
		if !strings.Contains(joined, want) {
			t.Errorf("StartArgs missing %q\nfull args: %v", want, args)
		}
	}
}

// TestStartArgs_Podman verifies PodmanRuntime.StartArgs uses 0.0.0.0 binding.
func TestStartArgs_Podman(t *testing.T) {
	dir := t.TempDir()
	makeFakeBinary(t, dir, "podman")

	opts := defaultOpts(makeLookPath(dir), 15, 0, "amd64")
	rt, err := container.DetectRuntime(opts)
	if err != nil {
		t.Fatalf("DetectRuntime: %v", err)
	}
	if rt.Name() != "podman" {
		t.Skipf("need podman runtime, got %s", rt.Name())
	}

	args := rt.StartArgs(cfg())
	joined := strings.Join(args, " ")

	// Podman requires 0.0.0.0 binding (Pitfall 6).
	if !strings.Contains(joined, "0.0.0.0:3307:3306") {
		t.Errorf("PodmanRuntime.StartArgs should use 0.0.0.0:3307:3306, got: %v", args)
	}
	if !strings.Contains(joined, "dolthub/dolt-sql-server:latest") {
		t.Errorf("PodmanRuntime.StartArgs should reference dolthub/dolt-sql-server:latest, got: %v", args)
	}
}

// TestStartArgs_AppleContainer verifies AppleContainerRuntime.StartArgs.
func TestStartArgs_AppleContainer(t *testing.T) {
	dir := t.TempDir()
	makeFakeBinary(t, dir, "container")

	opts := defaultOpts(makeLookPath(dir), 26, 0, "arm64")
	rt, err := container.DetectRuntime(opts)
	if err != nil {
		t.Fatalf("DetectRuntime: %v", err)
	}
	if rt.Name() != "apple-container" {
		t.Skipf("need apple-container runtime, got %s", rt.Name())
	}

	args := rt.StartArgs(cfg())
	joined := strings.Join(args, " ")

	checks := []string{
		"run",
		"-d",
		"--name gsdw-dolt",
		"127.0.0.1:3307:3306",
		"/home/user/.beads/dolt:/var/lib/dolt",
		"dolthub/dolt-sql-server:latest",
	}
	for _, want := range checks {
		if !strings.Contains(joined, want) {
			t.Errorf("AppleContainer.StartArgs missing %q\nfull args: %v", want, args)
		}
	}
}

// TestStopArgs verifies all runtimes produce correct stop args.
func TestStopArgs_AllRuntimes(t *testing.T) {
	cases := []struct {
		binary string
		name   string
		major  int
		arch   string
	}{
		{"docker", "docker", 15, "amd64"},
		{"podman", "podman", 15, "amd64"},
		{"container", "apple-container", 26, "arm64"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			makeFakeBinary(t, dir, tc.binary)
			opts := defaultOpts(makeLookPath(dir), tc.major, 0, tc.arch)
			rt, err := container.DetectRuntime(opts)
			if err != nil {
				t.Fatalf("DetectRuntime: %v", err)
			}
			args := rt.StopArgs()
			joined := strings.Join(args, " ")
			if !strings.Contains(joined, "stop") {
				t.Errorf("StopArgs should contain 'stop', got: %v", args)
			}
			if !strings.Contains(joined, "gsdw-dolt") {
				t.Errorf("StopArgs should contain 'gsdw-dolt', got: %v", args)
			}
		})
	}
}

// TestIsRunningArgs verifies all runtimes produce inspect/is-running args.
func TestIsRunningArgs_AllRuntimes(t *testing.T) {
	cases := []struct {
		binary string
		name   string
		major  int
		arch   string
	}{
		{"docker", "docker", 15, "amd64"},
		{"podman", "podman", 15, "amd64"},
		{"container", "apple-container", 26, "arm64"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			makeFakeBinary(t, dir, tc.binary)
			opts := defaultOpts(makeLookPath(dir), tc.major, 0, tc.arch)
			rt, err := container.DetectRuntime(opts)
			if err != nil {
				t.Fatalf("DetectRuntime: %v", err)
			}
			args := rt.IsRunningArgs()
			joined := strings.Join(args, " ")
			if !strings.Contains(joined, "gsdw-dolt") {
				t.Errorf("IsRunningArgs should contain 'gsdw-dolt', got: %v", args)
			}
		})
	}
}
