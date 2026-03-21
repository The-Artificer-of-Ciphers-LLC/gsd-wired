package mcp

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/The-Artificer-of-Ciphers-LLC/gsd-wired/internal/graph"
)

// serverState holds the lazily-initialized graph client.
// All MCP tool handlers call s.init(ctx) before using s.client.
// sync.Once ensures initialization runs exactly once (D-06, D-07).
type serverState struct {
	once        sync.Once
	client      *graph.Client
	err         error
	beadsDir    string // project root (parent of .beads/)
	bdPath      string // optional override for testing; empty = LookPath
	initTimeout int    // optional override for bd init timeout in ms; 0 = use default 30s
}

// init lazily initializes the graph client. First call blocks until done.
// Subsequent calls return immediately with the stored client or error.
// On failure, the error is permanent (sync.Once does not retry) per D-10.
func (s *serverState) init(ctx context.Context) error {
	s.once.Do(func() {
		dir := s.beadsDir
		if dir == "" {
			var err error
			dir, err = os.Getwd()
			if err != nil {
				s.err = fmt.Errorf("failed to determine working directory: %w", err)
				return
			}
			s.beadsDir = dir
		}

		// Check if .beads/ exists; if not, run bd init (D-08).
		beadsPath := filepath.Join(dir, ".beads")
		if _, statErr := os.Stat(beadsPath); os.IsNotExist(statErr) {
			if err := s.runBdInit(ctx, dir); err != nil {
				s.err = err
				return
			}
		}

		// Create graph client with batch mode enabled (INFRA-10).
		if s.bdPath != "" {
			s.client = graph.NewClientWithPathBatch(s.bdPath, dir)
		} else {
			c, err := graph.NewClientBatch(dir)
			if err != nil {
				s.err = err
				return
			}
			s.client = c
		}
	})
	return s.err
}

// runBdInit runs `bd init` with a timeout (default 30s, configurable via initTimeout field).
// Uses --quiet for non-interactive operation.
func (s *serverState) runBdInit(ctx context.Context, dir string) error {
	timeout := 30 * time.Second
	if s.initTimeout > 0 {
		timeout = time.Duration(s.initTimeout) * time.Millisecond
	}

	initCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	bdPath := s.bdPath
	if bdPath == "" {
		var err error
		bdPath, err = exec.LookPath("bd")
		if err != nil {
			return fmt.Errorf("bd not found on PATH: %w\nFix: install beads (https://beads.dev)", err)
		}
	}

	cmd := exec.CommandContext(initCtx, bdPath, "init", "--quiet", "--skip-hooks", "--skip-agents")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "BEADS_DIR="+dir)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd init failed: %w\n%s\nFix: ensure dolt sql-server is running on port 3307", err, string(out))
	}
	return nil
}
