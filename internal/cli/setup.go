package cli

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/deps"
)

// NewSetupCmd creates the "gsdw setup" subcommand.
// It runs check-deps first, then offers install options for missing deps,
// re-verifies, and guides users to next steps.
func NewSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Interactive setup wizard — install missing dependencies",
		Long: `Interactive setup wizard for gsd-wired.

Runs dependency checks first, then offers install options (brew, go install,
or binary download) for any missing deps. Setup never installs anything
automatically — it shows you the command to run.

After installing, press Enter to re-check and confirm your environment is ready.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, brewAvailable := exec.LookPath("brew")
			return runSetup(cmd.InOrStdin(), cmd.OutOrStdout(), deps.CheckAll, brewAvailable == nil)
		},
	}
	return cmd
}

// installOptions holds the available install methods for a dependency.
type installOptions struct {
	brew     string // brew install command, empty if not applicable
	goInstall string // go install command, empty if not applicable
	download  string // download URL or curl command, empty if not applicable
}

// depInstallOptions maps binary names to their install method details.
var depInstallOptions = map[string]installOptions{
	"bd": {
		brew:      "brew install steveyegge/tap/beads",
		goInstall: "go install github.com/steveyegge/beads/cmd/bd@latest",
		download:  "",
	},
	"dolt": {
		brew:      "brew install dolthub/tap/dolt",
		goInstall: "",
		download:  "curl -L https://github.com/dolthub/dolt/releases/latest/download/install.sh | bash",
	},
	"go": {
		brew:      "brew install go",
		goInstall: "",
		download:  "https://go.dev/dl/",
	},
	"docker": {
		brew:      "brew install --cask docker",
		goInstall: "",
		download:  "https://docs.docker.com/get-docker/",
	},
}

// runSetup is the testable core of the setup wizard.
//
//   - in: user input (stdin or test reader)
//   - out: output destination
//   - checkFn: dependency checker (injected for tests)
//   - brewAvailable: whether brew is on PATH (injected for tests)
func runSetup(in io.Reader, out io.Writer, checkFn func() deps.CheckResult, brewAvailable bool) error {
	reader := bufio.NewReader(in)

	// Phase 1: Check current state and display results.
	fmt.Fprintln(out, "Checking dependencies...")
	fmt.Fprintln(out)
	result := checkFn()
	renderCheckDeps(out, result)
	fmt.Fprintln(out)

	if result.AllOK {
		fmt.Fprintln(out, "All dependencies satisfied. Environment ready.")
		fmt.Fprintln(out)
		printNextSteps(out)
		return nil
	}

	// Phase 2: Offer install methods for each missing dep.
	for _, d := range result.Deps {
		if d.Status != deps.StatusFail {
			continue
		}
		if err := offerInstall(reader, out, d, brewAvailable); err != nil {
			return err
		}
		fmt.Fprintln(out)
	}

	// Phase 3: Re-verify after installs.
	fmt.Fprint(out, "Press Enter after installing to re-check...")
	_, _ = reader.ReadString('\n')
	fmt.Fprintln(out)

	fmt.Fprintln(out, "Re-checking dependencies...")
	fmt.Fprintln(out)
	result2 := checkFn()
	renderCheckDeps(out, result2)
	fmt.Fprintln(out)

	if !result2.AllOK {
		fmt.Fprintln(out, "Some dependencies are still missing. Re-run `gsdw setup` after installing.")
		fmt.Fprintln(out)
	}

	// Phase 4: Next steps guidance.
	printNextSteps(out)
	return nil
}

// offerInstall presents install options for a single missing dep and reads user's choice.
// It prints the selected command for the user to run — it does NOT run it automatically.
func offerInstall(reader *bufio.Reader, out io.Writer, d deps.Dep, brewAvailable bool) error {
	fmt.Fprintf(out, "Missing: %s\n", d.Name)
	fmt.Fprintf(out, "Install %s:\n", d.Name)

	opts := depInstallOptions[d.Binary]

	// Build numbered options list.
	type option struct {
		label   string
		command string
	}
	var options []option

	if brewAvailable && opts.brew != "" {
		options = append(options, option{label: "brew", command: opts.brew})
	}
	if opts.goInstall != "" {
		options = append(options, option{label: "go install", command: opts.goInstall})
	}
	if opts.download != "" {
		options = append(options, option{label: "download", command: opts.download})
	}

	for i, o := range options {
		fmt.Fprintf(out, "  [%d] %s\n", i+1, o.command)
	}
	fmt.Fprintln(out, "  [s] Skip")
	fmt.Fprint(out, "Choice: ")

	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return fmt.Errorf("reading choice: %w", err)
	}
	choice := strings.TrimSpace(line)

	switch {
	case choice == "s" || choice == "S" || choice == "":
		fmt.Fprintf(out, "Skipping %s.\n", d.Name)
	default:
		// Parse numeric choice.
		var idx int
		if _, parseErr := fmt.Sscanf(choice, "%d", &idx); parseErr != nil || idx < 1 || idx > len(options) {
			fmt.Fprintf(out, "Invalid choice %q — skipping %s.\n", choice, d.Name)
			return nil
		}
		selected := options[idx-1]
		fmt.Fprintf(out, "\nRun this command to install %s:\n  %s\n", d.Name, selected.command)
	}

	return nil
}

// printNextSteps prints the standard next steps guidance section (per D-08).
func printNextSteps(out io.Writer) {
	fmt.Fprintln(out, "Next steps:")
	fmt.Fprintln(out, "  - Container runtime: gsdw container setup (Phase 13)")
	fmt.Fprintln(out, "  - Connection config:  gsdw connect (Phase 14)")
	fmt.Fprintln(out, "  - Health check:       gsdw doctor")
}
