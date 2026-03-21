package mcp

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/version"
)

// Serve starts the MCP stdio server and blocks until the context is cancelled
// or the client disconnects.
// No tools are registered at Phase 1 — the manifest grows per phase (D-09).
func Serve(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "gsd-wired",
		Version: version.String(),
	}, nil)

	slog.Debug("mcp server starting on stdio")

	return server.Run(ctx, &mcp.StdioTransport{})
}
