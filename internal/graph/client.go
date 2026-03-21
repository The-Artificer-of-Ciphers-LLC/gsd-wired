package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
)

// Client wraps the bd CLI via exec.Command for all beads graph operations.
// All operations set BEADS_DIR env and append --json flag automatically.
type Client struct {
	bdPath    string // resolved at construction via exec.LookPath
	beadsDir  string // path to .beads/ directory (sets BEADS_DIR env)
	batchMode bool   // if true, write ops prepend --dolt-auto-commit=batch (INFRA-10)
}

// NewClient creates a new Client by looking up bd on PATH.
// Returns an error if bd is not found.
func NewClient(beadsDir string) (*Client, error) {
	bdPath, err := exec.LookPath("bd")
	if err != nil {
		return nil, fmt.Errorf("bd not found on PATH — install beads first: %w", err)
	}
	return &Client{bdPath: bdPath, beadsDir: beadsDir}, nil
}

// NewClientBatch creates a new Client with batch write mode enabled.
// Write operations will prepend --dolt-auto-commit=batch to batch Dolt commits.
// Returns an error if bd is not found.
func NewClientBatch(beadsDir string) (*Client, error) {
	bdPath, err := exec.LookPath("bd")
	if err != nil {
		return nil, fmt.Errorf("bd not found on PATH — install beads first: %w", err)
	}
	return &Client{bdPath: bdPath, beadsDir: beadsDir, batchMode: true}, nil
}

// NewClientWithPath creates a Client with an explicit bd binary path.
// Intended for testing — bypasses exec.LookPath.
func NewClientWithPath(bdPath, beadsDir string) *Client {
	return &Client{bdPath: bdPath, beadsDir: beadsDir}
}

// NewClientWithPathBatch creates a Client with an explicit bd binary path and batch mode enabled.
// Intended for testing — bypasses exec.LookPath.
func NewClientWithPathBatch(bdPath, beadsDir string) *Client {
	return &Client{bdPath: bdPath, beadsDir: beadsDir, batchMode: true}
}

// runWrite executes a mutating bd command. When batchMode is true, it prepends
// --dolt-auto-commit=batch as the first arg (a global bd flag that must precede
// the subcommand). When batchMode is false, it delegates directly to run().
func (c *Client) runWrite(ctx context.Context, args ...string) ([]byte, error) {
	if c.batchMode {
		// Prepend global flag before the subcommand: bd --dolt-auto-commit=batch create ...
		batchArgs := make([]string, 0, len(args)+1)
		batchArgs = append(batchArgs, "--dolt-auto-commit=batch")
		batchArgs = append(batchArgs, args...)
		return c.run(ctx, batchArgs...)
	}
	return c.run(ctx, args...)
}

// FlushWrites commits all accumulated batch writes to Dolt by running
// `bd dolt commit`. This must be called after a series of batched write
// operations to persist them. FlushWrites uses run() not runWrite() because
// the commit itself is not a batched operation.
func (c *Client) FlushWrites(ctx context.Context) error {
	_, err := c.run(ctx, "dolt", "commit", "--message", "gsdw: batch flush")
	return err
}

// run executes a bd command and returns stdout bytes.
// --json is always appended to args. BEADS_DIR env var is set on the command.
// Two-tier error handling:
//   - If bd exits non-zero and stdout contains {"error":"..."}, returns that message.
//   - Otherwise, returns stderr text as the error.
func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) {
	args = append(args, "--json")

	cmd := exec.CommandContext(ctx, c.bdPath, args...)
	cmd.Env = append(os.Environ(), "BEADS_DIR="+c.beadsDir)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
	}

	slog.Debug("bd command", "args", args, "exit_code", exitCode)

	if err != nil {
		// Two-tier error: check if stdout has a JSON error object (post-connection failure)
		if stdout.Len() > 0 {
			var bdErr struct {
				Error string `json:"error"`
			}
			if jsonErr := json.Unmarshal(stdout.Bytes(), &bdErr); jsonErr == nil && bdErr.Error != "" {
				return nil, fmt.Errorf("bd error: %s", bdErr.Error)
			}
		}
		// Pre-connection failure: use stderr text
		stderrText := stderr.String()
		if stderrText == "" {
			stderrText = err.Error()
		}
		return nil, fmt.Errorf("bd %v: %s", args[0], stderrText)
	}

	return stdout.Bytes(), nil
}
