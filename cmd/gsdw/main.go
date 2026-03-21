package main

import (
	"log/slog"
	"os"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/cli"
)

func main() {
	// CRITICAL: Set stderr-only slog default BEFORE anything else.
	// This prevents the default logger from writing to stdout if any code logs
	// before PersistentPreRunE runs (avoids stdout pollution in MCP serve mode).
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))

	os.Exit(cli.Execute())
}
