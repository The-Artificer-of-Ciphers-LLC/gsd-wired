package container

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// ErrNoRuntime is returned when no container runtime is detected.
var ErrNoRuntime = errors.New("no container runtime found; install Docker (brew install --cask docker), Podman (brew install podman), or Apple Container (macOS 26 + Apple Silicon)")

const containerName = "gsdw-dolt"

// Runtime represents a container runtime (Docker, Podman, Apple Container).
type Runtime interface {
	Name() string                          // "docker", "podman", "apple-container"
	Binary() string                        // resolved path to binary
	StartArgs(cfg ContainerConfig) []string // args for exec.Command
	StopArgs() []string
	RemoveArgs() []string // args to remove a stopped container
	IsRunningArgs() []string
}

// ContainerConfig holds configuration for launching the Dolt container.
type ContainerConfig struct {
	BeadsDoltDir string // absolute path to .beads/dolt/ on host
	HostPort     string // default "3307"
}

// DetectOpts allows injecting test doubles for platform detection.
type DetectOpts struct {
	LookPath     func(string) (string, error)
	MacOSVersion func() (int, int, error)
	Arch         func() string
}

// DetectRuntime returns the first available runtime in priority order:
// Apple Container (macOS 26+ arm64) -> Docker -> Podman.
func DetectRuntime(opts DetectOpts) (Runtime, error) {
	lookPath := opts.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}
	macOSVersion := opts.MacOSVersion
	if macOSVersion == nil {
		macOSVersion = defaultMacOSVersion
	}
	arch := opts.Arch
	if arch == nil {
		arch = func() string { return runtime.GOARCH }
	}

	// Try Apple Container first (macOS 26+ and arm64 only).
	if major, _, err := macOSVersion(); err == nil && major >= 26 && arch() == "arm64" {
		if p, err := lookPath("container"); err == nil {
			return &AppleContainerRuntime{path: p}, nil
		}
	}

	// Try Docker.
	if p, err := lookPath("docker"); err == nil {
		return &DockerRuntime{path: p}, nil
	}

	// Try Podman.
	if p, err := lookPath("podman"); err == nil {
		return &PodmanRuntime{path: p}, nil
	}

	return nil, ErrNoRuntime
}

// defaultMacOSVersion runs sw_vers -productVersion and parses major.minor.
func defaultMacOSVersion() (int, int, error) {
	var out bytes.Buffer
	cmd := exec.Command("sw_vers", "-productVersion")
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return 0, 0, fmt.Errorf("sw_vers: %w", err)
	}
	ver := strings.TrimSpace(out.String())
	parts := strings.SplitN(ver, ".", 3)
	if len(parts) < 1 {
		return 0, 0, fmt.Errorf("unexpected sw_vers output: %q", ver)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("parse major version from %q: %w", ver, err)
	}
	minor := 0
	if len(parts) >= 2 {
		minor, _ = strconv.Atoi(parts[1])
	}
	return major, minor, nil
}

// DockerRuntime implements Runtime for Docker.
type DockerRuntime struct {
	path string
}

func (d *DockerRuntime) Name() string { return "docker" }
func (d *DockerRuntime) Binary() string { return d.path }

func (d *DockerRuntime) StartArgs(cfg ContainerConfig) []string {
	hostPort := cfg.HostPort
	if hostPort == "" {
		hostPort = "3307"
	}
	return []string{
		"run", "-d",
		"--name", containerName,
		"-p", fmt.Sprintf("127.0.0.1:%s:3306", hostPort),
		"-e", "DOLT_ROOT_HOST=%",
		"-v", fmt.Sprintf("%s:/var/lib/dolt", cfg.BeadsDoltDir),
		"dolthub/dolt-sql-server:latest",
	}
}

func (d *DockerRuntime) StopArgs() []string {
	return []string{"stop", containerName}
}

func (d *DockerRuntime) RemoveArgs() []string {
	return []string{"rm", containerName}
}

func (d *DockerRuntime) IsRunningArgs() []string {
	return []string{"inspect", "--format", "{{.State.Running}}", containerName}
}

// PodmanRuntime implements Runtime for Podman.
// Note: Podman on macOS requires 0.0.0.0 port binding (Pitfall 6).
type PodmanRuntime struct {
	path string
}

func (p *PodmanRuntime) Name() string { return "podman" }
func (p *PodmanRuntime) Binary() string { return p.path }

func (p *PodmanRuntime) StartArgs(cfg ContainerConfig) []string {
	hostPort := cfg.HostPort
	if hostPort == "" {
		hostPort = "3307"
	}
	return []string{
		"run", "-d",
		"--name", containerName,
		"-p", fmt.Sprintf("0.0.0.0:%s:3306", hostPort),
		"-e", "DOLT_ROOT_HOST=%",
		"-v", fmt.Sprintf("%s:/var/lib/dolt", cfg.BeadsDoltDir),
		"dolthub/dolt-sql-server:latest",
	}
}

func (p *PodmanRuntime) StopArgs() []string {
	return []string{"stop", containerName}
}

func (p *PodmanRuntime) RemoveArgs() []string {
	return []string{"rm", containerName}
}

func (p *PodmanRuntime) IsRunningArgs() []string {
	return []string{"inspect", "--format", "{{.State.Running}}", containerName}
}

// AppleContainerRuntime implements Runtime for Apple Container (macOS 26+ arm64).
type AppleContainerRuntime struct {
	path string
}

func (a *AppleContainerRuntime) Name() string { return "apple-container" }
func (a *AppleContainerRuntime) Binary() string { return a.path }

func (a *AppleContainerRuntime) StartArgs(cfg ContainerConfig) []string {
	hostPort := cfg.HostPort
	if hostPort == "" {
		hostPort = "3307"
	}
	return []string{
		"run", "-d",
		"--name", containerName,
		"-p", fmt.Sprintf("127.0.0.1:%s:3306", hostPort),
		"-e", "DOLT_ROOT_HOST=%",
		"-v", fmt.Sprintf("%s:/var/lib/dolt", cfg.BeadsDoltDir),
		"dolthub/dolt-sql-server:latest",
	}
}

func (a *AppleContainerRuntime) StopArgs() []string {
	return []string{"stop", containerName}
}

func (a *AppleContainerRuntime) RemoveArgs() []string {
	return []string{"rm", containerName}
}

func (a *AppleContainerRuntime) IsRunningArgs() []string {
	return []string{"inspect", containerName}
}
