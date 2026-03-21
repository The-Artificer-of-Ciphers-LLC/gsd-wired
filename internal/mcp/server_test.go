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

// TestToolsListed verifies that the MCP server responds to tools/list with all 10 tool names.
// This is a subprocess integration test: lazy init means no bd/Dolt is needed for tools/list.
func TestToolsListed(t *testing.T) {
	// Build the binary first.
	tmpDir := t.TempDir()
	binaryName := "gsdw-test-tools"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(tmpDir, binaryName)

	moduleRoot := findModuleRoot(t)
	buildCmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/gsdw")
	buildCmd.Dir = moduleRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to build gsdw: %v\n%s", err, out)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	// readNextJSON reads the next complete JSON object from the reader.
	readNextJSON := func() (map[string]any, error) {
		var buf bytes.Buffer
		tmp := make([]byte, 4096)
		for {
			n, err := stdout.Read(tmp)
			if n > 0 {
				buf.Write(tmp[:n])
				data := buf.Bytes()
				depth := 0
				for i, b := range data {
					if b == '{' {
						depth++
					} else if b == '}' {
						depth--
						if depth == 0 {
							var obj map[string]any
							if jsonErr := json.Unmarshal(data[:i+1], &obj); jsonErr != nil {
								return nil, jsonErr
							}
							return obj, nil
						}
					}
				}
			}
			if err != nil {
				return nil, err
			}
		}
	}

	// Step 1: Send initialize.
	initRequest := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}` + "\n"
	if _, err := io.WriteString(stdin, initRequest); err != nil {
		t.Fatalf("failed to write initialize request: %v", err)
	}

	// Read initialize response.
	initRespCh := make(chan map[string]any, 1)
	initErrCh := make(chan error, 1)
	go func() {
		obj, err := readNextJSON()
		if err != nil {
			initErrCh <- err
		} else {
			initRespCh <- obj
		}
	}()
	select {
	case <-initRespCh:
		// Got initialize response — proceed.
	case err := <-initErrCh:
		t.Fatalf("failed to read initialize response: %v", err)
	case <-ctx.Done():
		t.Fatalf("timeout waiting for initialize response")
	}

	// Step 2: Send notifications/initialized (required by MCP protocol).
	initializedNotif := `{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}` + "\n"
	if _, err := io.WriteString(stdin, initializedNotif); err != nil {
		t.Fatalf("failed to write initialized notification: %v", err)
	}

	// Step 3: Send tools/list.
	listRequest := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}` + "\n"
	if _, err := io.WriteString(stdin, listRequest); err != nil {
		t.Fatalf("failed to write tools/list request: %v", err)
	}

	// Read tools/list response.
	listRespCh := make(chan map[string]any, 1)
	listErrCh := make(chan error, 1)
	go func() {
		obj, err := readNextJSON()
		if err != nil {
			listErrCh <- err
		} else {
			listRespCh <- obj
		}
	}()

	var listResp map[string]any
	select {
	case listResp = <-listRespCh:
	case err := <-listErrCh:
		t.Fatalf("failed to read tools/list response: %v", err)
	case <-ctx.Done():
		t.Fatalf("timeout waiting for tools/list response")
	}

	stdin.Close()

	// Validate response ID.
	if id := fmt.Sprintf("%v", listResp["id"]); id != "2" {
		t.Errorf("expected id=2, got %v", listResp["id"])
	}

	// Extract tools array.
	result, ok := listResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("tools/list response missing 'result' object: %v", listResp)
	}
	toolsRaw, ok := result["tools"].([]any)
	if !ok {
		t.Fatalf("result missing 'tools' array: %v", result)
	}

	// Verify count.
	if len(toolsRaw) != 10 {
		names := make([]string, 0, len(toolsRaw))
		for _, ti := range toolsRaw {
			if tm, ok := ti.(map[string]any); ok {
				names = append(names, fmt.Sprintf("%v", tm["name"]))
			}
		}
		t.Errorf("expected 10 tools, got %d: %v", len(toolsRaw), names)
	}

	// Verify all expected tool names are present.
	wantNames := []string{
		"create_phase", "create_plan", "get_bead", "list_ready",
		"query_by_label", "claim_bead", "close_plan", "flush_writes",
		"init_project", "get_status",
	}
	toolMap := make(map[string]map[string]any)
	for _, ti := range toolsRaw {
		if tm, ok := ti.(map[string]any); ok {
			if name, ok := tm["name"].(string); ok {
				toolMap[name] = tm
			}
		}
	}
	for _, name := range wantNames {
		tool, found := toolMap[name]
		if !found {
			t.Errorf("tool %q not found in tools/list response", name)
			continue
		}
		// Verify each tool has a non-empty inputSchema with type:object.
		schema, ok := tool["inputSchema"].(map[string]any)
		if !ok {
			t.Errorf("tool %q missing inputSchema object", name)
			continue
		}
		if schema["type"] != "object" {
			t.Errorf("tool %q inputSchema type = %q, want \"object\"", name, schema["type"])
		}
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
