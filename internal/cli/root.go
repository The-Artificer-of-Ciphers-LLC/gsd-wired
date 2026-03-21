package cli

import (
	"log/slog"

	"github.com/spf13/cobra"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/logging"
)

var (
	logLevel  string
	logFormat string
)

// NewRootCmd creates the root gsdw command with all subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "gsdw",
		Short:         "GSD Wired - token-efficient development lifecycle",
		SilenceUsage:  true, // CRITICAL: prevents usage dump to stdout on error
		SilenceErrors: true, // CRITICAL: prevents error text to stdout on error
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return logging.Init(logLevel, logFormat)
		},
	}

	root.PersistentFlags().StringVar(&logLevel, "log-level", "error", "Log level (error|info|debug)")
	root.PersistentFlags().StringVar(&logFormat, "log-format", "text", "Log format (text|json)")

	root.AddCommand(NewVersionCmd(), NewServeCmd(), NewHookCmd(), NewBdCmd(), NewReadyCmd(), NewInitCmd(), NewStatusCmd(), NewResearchCmd(), NewPlanCmd(), NewExecuteCmd(), NewVerifyCmd(), NewShipCmd())

	return root
}

// Execute creates the root command and runs it, returning an exit code.
func Execute() int {
	root := NewRootCmd()
	if err := root.Execute(); err != nil {
		slog.Error("command failed", "err", err)
		return 1
	}
	return 0
}
