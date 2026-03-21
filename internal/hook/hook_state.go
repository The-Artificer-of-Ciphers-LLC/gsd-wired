package hook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"
)

// hookState holds the lazily-initialized graph client for hook handlers.
// Unlike serverState (MCP), hooks use non-batch client and do NOT run bd init.
// Hooks read the graph; they don't create .beads/.
type hookState struct {
	once     sync.Once
	client   *graph.Client
	err      error
	beadsDir string // project root (parent of .beads/)
	bdPath   string // optional override for testing; empty = LookPath
}

// init lazily initializes the graph client. First call blocks until done.
// Subsequent calls return immediately with the stored client or error.
// On failure, the error is permanent (sync.Once does not retry).
func (h *hookState) init(ctx context.Context) error {
	h.once.Do(func() {
		dir := h.beadsDir
		if dir == "" {
			var err error
			dir, err = os.Getwd()
			if err != nil {
				h.err = fmt.Errorf("hookState: failed to determine working directory: %w", err)
				return
			}
			h.beadsDir = dir
		}

		// Create non-batch graph client (hooks read-only, no batch needed).
		if h.bdPath != "" {
			// Validate the binary exists before constructing the client.
			if _, statErr := os.Stat(h.bdPath); statErr != nil {
				h.err = fmt.Errorf("hookState: bd binary not found at %s: %w", h.bdPath, statErr)
				return
			}
			h.client = graph.NewClientWithPath(h.bdPath, dir)
		} else {
			c, err := graph.NewClient(dir)
			if err != nil {
				h.err = err
				return
			}
			h.client = c
		}
	})
	return h.err
}

// resolvedBdPath returns bdPath or looks up "bd" on PATH.
// Used only for tests or diagnostics — init() handles the real lookup.
func (h *hookState) resolvedBdPath() (string, error) {
	if h.bdPath != "" {
		return h.bdPath, nil
	}
	return exec.LookPath("bd")
}

// writeOutput encodes out as JSON to w via json.NewEncoder.
// This is the single exit path for all hook handlers, ensuring stdout purity.
func writeOutput(w io.Writer, out HookOutput) error {
	return json.NewEncoder(w).Encode(out)
}
