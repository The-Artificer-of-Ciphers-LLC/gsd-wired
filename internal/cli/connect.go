package cli

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/connection"
	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/container"
)

// connectOpts holds all injectable dependencies for the connect wizard.
type connectOpts struct {
	in  io.Reader
	out io.Writer

	// detectServerFn probes whether a Dolt server is listening at host:port.
	detectServerFn func(host, port string) error

	// healthCheckFn performs a full two-phase connectivity check.
	healthCheckFn func(host, port, user, password string) error

	// loadConfigFn loads the connection config from gsdwDir (nil,nil if missing).
	loadConfigFn func(gsdwDir string) (*connection.Config, error)

	// saveConfigFn persists the connection config to gsdwDir.
	saveConfigFn func(gsdwDir string, cfg *connection.Config) error

	// startContainerFn starts the Dolt container using the detected runtime.
	startContainerFn func() error

	// findGsdwDirFn locates the .gsdw/ directory (returns "" if not found).
	findGsdwDirFn func() string
}

// NewConnectCmd creates the "gsdw connect" command.
// It runs an interactive wizard to configure how gsdw reaches the Dolt server.
// The wizard auto-detects running local servers, offers container start or remote
// configuration, and handles fallback when a remote host is unreachable.
func NewConnectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "connect",
		Short: "Configure connection to Dolt server",
		Long: `Interactive wizard to configure how gsdw connects to the Dolt database server.

The wizard:
  1. Auto-detects a running local Dolt server on 127.0.0.1:3307
  2. If none found, offers three choices:
       1) Start local container
       2) Configure remote host
       3) Cancel
  3. For remote mode, collects host, port, and optional username.
     Password is read from the GSDW_DB_PASSWORD environment variable.
  4. If reconfiguring, shows current status and asks before overwriting.

On remote unreachable, falls back to local container with a blocking Y/N prompt.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := connectOpts{
				in:  cmd.InOrStdin(),
				out: cmd.OutOrStdout(),
				detectServerFn: func(host, port string) error {
					// TCP-only probe for discovery — SQL auth not required to detect a server.
					addr := net.JoinHostPort(host, port)
					conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
					if err != nil {
						return err
					}
					conn.Close()
					return nil
				},
				healthCheckFn: func(host, port, user, password string) error {
					return connection.CheckConnectivity(host, port, user, password, 2*time.Second)
				},
				loadConfigFn:     connection.LoadConnection,
				saveConfigFn:     connection.SaveConnection,
				startContainerFn: defaultStartContainer,
				findGsdwDirFn:    findGsdwDir,
			}
			return runConnect(opts)
		},
	}
}

// runConnect is the testable core of the connect wizard.
func runConnect(opts connectOpts) error {
	reader := bufio.NewReader(opts.in)

	// Phase 1: Locate .gsdw/ (prerequisite).
	gsdwDir := opts.findGsdwDirFn()
	if gsdwDir == "" {
		return fmt.Errorf("no .gsdw/ directory found — run gsdw init first")
	}

	// Phase 2: Check existing config (D-05).
	existing, _ := opts.loadConfigFn(gsdwDir)
	if existing != nil {
		host, port := existing.ActiveHostPort()
		fmt.Fprintf(opts.out, "Current connection: %s (%s:%s)\n", existing.ActiveMode, host, port)

		healthErr := opts.healthCheckFn(host, port, existing.Remote.User, os.Getenv("GSDW_DB_PASSWORD"))
		if healthErr != nil {
			fmt.Fprintf(opts.out, "  [FAIL] %v\n", healthErr)
		} else {
			fmt.Fprintln(opts.out, "  [OK]   Server responding")
		}

		fmt.Fprint(opts.out, "Reconfigure? [y/N]: ")
		answer := readLine(reader)
		if strings.ToLower(answer) != "y" {
			return nil
		}
	}

	// Phase 3: Auto-detect local server (D-01).
	fmt.Fprintln(opts.out, "Scanning for running Dolt server on 127.0.0.1:3307...")
	if opts.detectServerFn("127.0.0.1", "3307") == nil {
		fmt.Fprintln(opts.out, "Found local Dolt server on 127.0.0.1:3307")
		fmt.Fprint(opts.out, "Use it? [Y/n]: ")
		answer := readLine(reader)
		if answer == "" || strings.ToLower(answer) == "y" {
			return doSaveLocalConfig(opts, gsdwDir, "127.0.0.1", "3307")
		}
	}

	// Phase 4: No server found — offer choices (D-02).
	fmt.Fprintln(opts.out, "No running Dolt server found.")
	fmt.Fprintln(opts.out)
	fmt.Fprintln(opts.out, "  1) Start local container")
	fmt.Fprintln(opts.out, "  2) Configure remote host")
	fmt.Fprintln(opts.out, "  3) Cancel")
	fmt.Fprint(opts.out, "Choose [1-3]: ")
	choice := readLine(reader)

	switch choice {
	case "1":
		return handleStartContainer(opts, gsdwDir)
	case "2":
		return handleConfigureRemote(opts, gsdwDir, reader)
	default:
		// "3" or anything else = cancel
		return nil
	}
}

// handleStartContainer starts the Dolt container and saves a local connection config.
func handleStartContainer(opts connectOpts, gsdwDir string) error {
	if err := opts.startContainerFn(); err != nil {
		return fmt.Errorf("container start failed: %w", err)
	}
	fmt.Fprintln(opts.out, "Container started")
	return doSaveLocalConfig(opts, gsdwDir, "127.0.0.1", "3307")
}

// handleConfigureRemote collects remote host/port/user and verifies connectivity (D-04).
func handleConfigureRemote(opts connectOpts, gsdwDir string, reader *bufio.Reader) error {
	fmt.Fprint(opts.out, "Host: ")
	host := readLine(reader)

	fmt.Fprint(opts.out, "Port [3306]: ")
	port := readLine(reader)
	if port == "" {
		port = "3306"
	}

	fmt.Fprint(opts.out, "Username (optional): ")
	user := strings.TrimSpace(readLine(reader))

	fmt.Fprintf(opts.out, "  (password from GSDW_DB_PASSWORD env var)\n")

	password := os.Getenv("GSDW_DB_PASSWORD")
	if err := opts.healthCheckFn(host, port, user, password); err != nil {
		return handleRemoteFallback(opts, gsdwDir, reader, host, port, user, err)
	}

	fmt.Fprintf(opts.out, "[OK] Connected to %s:%s\n", host, port)
	return doSaveRemoteConfig(opts, gsdwDir, host, port, user)
}

// handleRemoteFallback handles the case when the remote host is unreachable (D-09/D-10/D-11).
func handleRemoteFallback(opts connectOpts, gsdwDir string, reader *bufio.Reader, host, port, user string, healthErr error) error {
	fmt.Fprintf(opts.out, "[FAIL] %v\n", healthErr)
	fmt.Fprint(opts.out, "Fall back to local container? [y/N]: ")
	answer := readLine(reader)

	if strings.ToLower(answer) != "y" {
		return fmt.Errorf("connection to %s:%s failed: %w", host, port, healthErr)
	}

	// D-10: Start local container.
	if err := opts.startContainerFn(); err != nil {
		return fmt.Errorf("container start failed: %w", err)
	}

	// D-11: Ask whether to make local the default.
	fmt.Fprint(opts.out, "Make this the default? [y/N]: ")
	makeDefault := readLine(reader)

	if strings.ToLower(makeDefault) == "y" {
		// Save local as active mode, preserve remote settings.
		cfg := &connection.Config{
			ActiveMode: "local",
			Local:      connection.LocalConfig{Host: "127.0.0.1", Port: connection.FlexPort("3307")},
			Remote:     connection.RemoteConfig{Host: host, Port: connection.FlexPort(port), User: user},
			Configured: time.Now().UTC().Format(time.RFC3339),
		}
		if err := opts.saveConfigFn(gsdwDir, cfg); err != nil {
			return fmt.Errorf("save connection config: %w", err)
		}
		fmt.Fprintln(opts.out, "Connection saved to .gsdw/connection.json")
		return nil
	}

	// Session-only: keep active_mode="remote" but local container is running.
	cfg := &connection.Config{
		ActiveMode: "remote",
		Local:      connection.LocalConfig{Host: "127.0.0.1", Port: connection.FlexPort("3307")},
		Remote:     connection.RemoteConfig{Host: host, Port: connection.FlexPort(port), User: user},
		Configured: time.Now().UTC().Format(time.RFC3339),
	}
	if err := opts.saveConfigFn(gsdwDir, cfg); err != nil {
		return fmt.Errorf("save connection config: %w", err)
	}
	fmt.Fprintln(opts.out, "Local container started (session only — remote remains default).")
	fmt.Fprintln(opts.out, "Connection saved to .gsdw/connection.json")
	return nil
}

// doSaveLocalConfig builds and saves a local connection config.
func doSaveLocalConfig(opts connectOpts, gsdwDir, host, port string) error {
	cfg := &connection.Config{
		ActiveMode: "local",
		Local:      connection.LocalConfig{Host: host, Port: connection.FlexPort(port)},
		Configured: time.Now().UTC().Format(time.RFC3339),
	}
	if err := opts.saveConfigFn(gsdwDir, cfg); err != nil {
		return fmt.Errorf("save connection config: %w", err)
	}
	fmt.Fprintln(opts.out, "Connection saved to .gsdw/connection.json")
	return nil
}

// doSaveRemoteConfig builds and saves a remote connection config.
func doSaveRemoteConfig(opts connectOpts, gsdwDir, host, port, user string) error {
	cfg := &connection.Config{
		ActiveMode: "remote",
		Remote:     connection.RemoteConfig{Host: host, Port: connection.FlexPort(port), User: user},
		Configured: time.Now().UTC().Format(time.RFC3339),
	}
	if err := opts.saveConfigFn(gsdwDir, cfg); err != nil {
		return fmt.Errorf("save connection config: %w", err)
	}
	fmt.Fprintln(opts.out, "Connection saved to .gsdw/connection.json")
	return nil
}

// defaultStartContainer is the real container start for the connect wizard.
// It detects the available runtime and starts gsdw-dolt with default settings.
func defaultStartContainer() error {
	rt, err := container.DetectRuntime(container.DetectOpts{})
	if err != nil {
		return fmt.Errorf("no container runtime found — install Docker or Podman")
	}

	cfg := container.ContainerConfig{
		HostPort: "3307",
	}
	args := rt.StartArgs(cfg)
	cmd := exec.Command(rt.Binary(), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// readLine reads a line from reader, trims whitespace, and returns it.
// Returns "" on EOF or error.
func readLine(reader *bufio.Reader) string {
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}
