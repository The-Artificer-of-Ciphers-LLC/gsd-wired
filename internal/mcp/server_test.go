package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestServeRespondsToInitialize(t *testing.T) {
	// Build the binary first
	tmpDir := t.TempDir()
	binaryName := "gsdw-test"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(tmpDir, binaryName)

	// Find the module root (go up from this file's location)
	moduleRoot := findModuleRoot(t)

	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/gsdw")
	buildCmd.Dir = moduleRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build gsdw: %v\n%s", err, out)
	}

	// Start gsdw serve as a subprocess
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "serve")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("failed to get stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("failed to get stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start gsdw serve: %v", err)
	}
	defer cmd.Wait()

	// Send initialize JSON-RPC request
	initRequest := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}` + "\n"
	if _, err := io.WriteString(stdin, initRequest); err != nil {
		t.Fatalf("failed to write initialize request: %v", err)
	}

	// Read the response (first complete JSON line)
	responseChan := make(chan []byte, 1)
	errChan := make(chan error, 1)
	go func() {
		var buf bytes.Buffer
		tmp := make([]byte, 4096)
		for {
			n, err := stdout.Read(tmp)
			if n > 0 {
				buf.Write(tmp[:n])
				// Check if we have a complete JSON object
				data := buf.Bytes()
				// Find end of first JSON object
				depth := 0
				for i, b := range data {
					if b == '{' {
						depth++
					} else if b == '}' {
						depth--
						if depth == 0 {
							responseChan <- data[:i+1]
							return
						}
					}
				}
			}
			if err != nil {
				if buf.Len() > 0 {
					responseChan <- buf.Bytes()
				} else {
					errChan <- err
				}
				return
			}
		}
	}()

	var responseBytes []byte
	select {
	case responseBytes = <-responseChan:
	case err := <-errChan:
		t.Fatalf("failed to read response: %v", err)
	case <-ctx.Done():
		t.Fatalf("timeout waiting for initialize response")
	}

	// Close stdin to terminate the server
	stdin.Close()

	// Parse and validate the response
	var response map[string]interface{}
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		t.Fatalf("response is not valid JSON: %v, raw: %q", err, string(responseBytes))
	}

	// Assert jsonrpc version
	if v, ok := response["jsonrpc"]; !ok || v != "2.0" {
		t.Errorf("expected jsonrpc=2.0, got: %v", response["jsonrpc"])
	}

	// Assert id matches
	if id, ok := response["id"]; !ok {
		t.Error("response missing 'id' field")
	} else {
		// JSON numbers decode as float64
		if fmt.Sprintf("%v", id) != "1" {
			t.Errorf("expected id=1, got: %v", id)
		}
	}

	// Assert server info contains gsd-wired
	result, ok := response["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("response missing 'result' object, response: %v", response)
	}

	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("result missing 'serverInfo' object, result: %v", result)
	}

	responseStr := string(responseBytes)
	if !strings.Contains(responseStr, "gsd-wired") {
		t.Errorf("expected serverInfo to contain 'gsd-wired', serverInfo: %v, full response: %s", serverInfo, responseStr)
	}
}

// findModuleRoot finds the Go module root by looking for go.mod.
func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find go.mod from %s", dir)
		}
		dir = parent
	}
}
