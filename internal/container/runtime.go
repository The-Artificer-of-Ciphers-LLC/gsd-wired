package container

import "errors"

// ErrNoRuntime is returned when no container runtime is detected.
var ErrNoRuntime = errors.New("no container runtime found")

// Runtime represents a container runtime (Docker, Podman, Apple Container).
type Runtime interface {
	Name() string
	Binary() string
	StartArgs(cfg ContainerConfig) []string
	StopArgs() []string
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
	return nil, ErrNoRuntime
}
