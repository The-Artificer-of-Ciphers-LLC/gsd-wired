package logging

import (
	"log/slog"
	"os"
)

// Init configures the global slog logger with the specified level and format.
// CRITICAL: The handler writer is ALWAYS os.Stderr — never os.Stdout.
// This ensures no log output ever appears on stdout, which would corrupt MCP stdio protocol.
func Init(level, format string) error {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	default:
		lvl = slog.LevelError
	}

	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler

	if format == "json" {
		handler = slog.NewJSONHandler(os.Stderr, opts) // ALWAYS os.Stderr
	} else {
		handler = slog.NewTextHandler(os.Stderr, opts) // ALWAYS os.Stderr
	}

	slog.SetDefault(slog.New(handler))
	return nil
}
