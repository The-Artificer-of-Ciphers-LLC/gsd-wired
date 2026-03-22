package cli

import (
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/container"
)

// startOpts holds all injectable dependencies for runContainerStart.
type startOpts struct {
	force        bool
	port         string
	beadsDoltDir string

	// detectFn detects the available container runtime.
	detectFn func(container.DetectOpts) (container.Runtime, error)

	// composeFn writes the gsdw.compose.yaml fragment.
	composeFn func(dir string, cfg container.ContainerConfig, force bool) (string, error)

	// execFn executes the container binary with given args.
	execFn func(name string, args ...string) error

	// checkPort returns nil if port is available, error if occupied.
	checkPort func(port string) error

	// statFn checks whether a path exists (injected for tests).
	statFn func(path string) error
}

// stopOpts holds all injectable dependencies for runContainerStop.
type stopOpts struct {
	detectFn func(container.DetectOpts) (container.Runtime, error)
	execFn   func(name string, args ...string) error
}

// NewContainerCmd creates the "gsdw container" command with start/stop subcommands.
func NewContainerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "container",
		Short: "Manage Dolt server container",
		Long:  `Start and stop the Dolt database container used as the beads graph backend.`,
	}

	cmd.AddCommand(NewContainerStartCmd(), NewContainerStopCmd())
	return cmd
}

// NewContainerStartCmd creates the "gsdw container start" subcommand.
func NewContainerStartCmd() *cobra.Command {
	var force bool
	var port string
	var beadsDoltDir string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the Dolt container",
		Long: `Detect the available container runtime (Apple Container, Docker, or Podman)
and launch the Dolt database server container.

Pre-flight checks:
  - .beads/dolt/ directory exists (run 'bd init --backend dolt' first)
  - Port is available (default 3307)

For Docker/Podman, also writes a gsdw.compose.yaml fragment for compose workflows.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := startOpts{
				force:        force,
				port:         port,
				beadsDoltDir: beadsDoltDir,
				detectFn:     container.DetectRuntime,
				composeFn:    container.WriteComposeFragment,
				execFn:       defaultExecFn,
				checkPort:    defaultCheckPort,
				statFn:       defaultStatFn,
			}
			return runContainerStart(cmd.OutOrStdout(), opts)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing gsdw.compose.yaml")
	cmd.Flags().StringVar(&port, "port", "3307", "Host port to bind Dolt MySQL interface")
	cmd.Flags().StringVar(&beadsDoltDir, "beads-dir", ".beads/dolt", "Path to beads Dolt data directory")

	return cmd
}

// NewContainerStopCmd creates the "gsdw container stop" subcommand.
func NewContainerStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the Dolt container",
		Long:  `Stop the gsdw-dolt container using the detected container runtime.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := stopOpts{
				detectFn: container.DetectRuntime,
				execFn:   defaultExecFn,
			}
			return runContainerStop(cmd.OutOrStdout(), opts)
		},
	}

	return cmd
}

// runContainerStart implements the container start logic with injected dependencies.
func runContainerStart(out io.Writer, opts startOpts) error {
	// 1. Detect runtime.
	rt, err := opts.detectFn(container.DetectOpts{})
	if err != nil {
		fmt.Fprintln(out, "No container runtime found.")
		fmt.Fprintln(out, "Install one of:")
		fmt.Fprintln(out, "  - Docker:            brew install --cask docker")
		fmt.Fprintln(out, "  - Podman:            brew install podman")
		fmt.Fprintln(out, "  - Apple Container:   macOS 26 + Apple Silicon required")
		return err
	}
	fmt.Fprintf(out, "Using runtime: %s\n", rt.Name())

	// 2. Pre-flight: check .beads/dolt/ exists (Pitfall 2).
	if err := opts.statFn(opts.beadsDoltDir); err != nil {
		return fmt.Errorf("`%s` not found — run `bd init --backend dolt` first", opts.beadsDoltDir)
	}

	// 3. Pre-flight: check port availability (Pitfall 3).
	if err := opts.checkPort(opts.port); err != nil {
		return fmt.Errorf("Port %s already in use — stop the existing Dolt server or use --port", opts.port)
	}

	// 4. Resolve absolute path for beads dolt dir.
	absDir, err := filepath.Abs(opts.beadsDoltDir)
	if err != nil {
		return fmt.Errorf("resolve beads-dir path: %w", err)
	}

	cfg := container.ContainerConfig{
		BeadsDoltDir: absDir,
		HostPort:     opts.port,
	}

	// 5. Write compose fragment for Docker/Podman (not Apple Container).
	if rt.Name() == "docker" || rt.Name() == "podman" {
		composePath, err := opts.composeFn(absDir, cfg, opts.force)
		if err != nil {
			return fmt.Errorf("write compose fragment: %w", err)
		}
		fmt.Fprintf(out, "Compose fragment written: %s\n", composePath)
	}

	// 6. Build start command and print it.
	startArgs := rt.StartArgs(cfg)
	cmdLine := rt.Binary() + " " + strings.Join(startArgs, " ")
	fmt.Fprintf(out, "Running: %s\n", cmdLine)

	// 7. Execute.
	if err := opts.execFn(rt.Binary(), startArgs...); err != nil {
		return fmt.Errorf("container start failed: %w", err)
	}

	// 8. Print success.
	fmt.Fprintf(out, "Dolt container started on 127.0.0.1:%s\n", opts.port)
	fmt.Fprintf(out, "Data persisted at %s\n", opts.beadsDoltDir)

	return nil
}

// runContainerStop implements the container stop logic with injected dependencies.
func runContainerStop(out io.Writer, opts stopOpts) error {
	// 1. Detect runtime.
	rt, err := opts.detectFn(container.DetectOpts{})
	if err != nil {
		return fmt.Errorf("detect runtime: %w", err)
	}

	// 2. Build and print stop command.
	stopArgs := rt.StopArgs()
	cmdLine := rt.Binary() + " " + strings.Join(stopArgs, " ")
	fmt.Fprintf(out, "Running: %s\n", cmdLine)

	// 3. Execute stop.
	if err := opts.execFn(rt.Binary(), stopArgs...); err != nil {
		return fmt.Errorf("container stop failed: %w", err)
	}

	fmt.Fprintln(out, "Dolt container stopped")
	return nil
}

// defaultCheckPort attempts to listen on the port. Returns nil if port is free, error if occupied.
func defaultCheckPort(port string) error {
	l, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		return err
	}
	l.Close()
	return nil
}

// defaultStatFn checks whether a path exists.
func defaultStatFn(path string) error {
	_, err := os.Stat(path)
	return err
}

// defaultExecFn runs the named binary with args.
func defaultExecFn(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
