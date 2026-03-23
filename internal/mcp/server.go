package mcp

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/version"
)

// Serve starts the MCP stdio server and blocks until the context is cancelled
// or the client disconnects.
// serverState is created lazily — no graph init happens here, so initialize
// responds instantly (D-06). Tools are registered but not initialized.
func Serve(ctx context.Context) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "gsd-wired",
		Version: version.String(),
	}, nil)

	state := &serverState{}
	registerTools(server, state)

	slog.Debug("mcp server starting on stdio", "tools", 19)

	return server.Run(ctx, &mcp.StdioTransport{})
}
