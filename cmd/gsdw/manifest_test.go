package main_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// repoRoot walks up from the test's working directory to find the go.mod file,
// returning the directory that contains it.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find go.mod — are we in the right directory?")
		}
		dir = parent
	}
}

// TestPluginManifestValid verifies .claude-plugin/plugin.json exists and
// contains the required fields with correct values.
func TestPluginManifestValid(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, ".claude-plugin", "plugin.json")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("unmarshal plugin.json: %v", err)
	}

	// Required top-level keys
	for _, key := range []string{"name", "version", "description", "author"} {
		if _, ok := manifest[key]; !ok {
			t.Errorf("plugin.json missing required key %q", key)
		}
	}

	// Correct values
	if got, want := manifest["name"], "gsd-wired"; got != want {
		t.Errorf("plugin.json name = %q, want %q", got, want)
	}
	if got, want := manifest["version"], "0.1.0"; got != want {
		t.Errorf("plugin.json version = %q, want %q", got, want)
	}
	if got, want := manifest["description"], "Token-efficient development lifecycle on a versioned graph"; got != want {
		t.Errorf("plugin.json description = %q, want %q", got, want)
	}

	// Author must be an object
	author, ok := manifest["author"].(map[string]any)
	if !ok {
		t.Fatalf("plugin.json author is not an object, got %T", manifest["author"])
	}
	if _, ok := author["name"]; !ok {
		t.Error("plugin.json author missing 'name' field")
	}

	// Must NOT contain minAppVersion (per D-12)
	if _, ok := manifest["minAppVersion"]; ok {
		t.Error("plugin.json must not contain 'minAppVersion' field (per D-12)")
	}
}

// TestMcpJsonValid verifies .mcp.json exists at the repo root and correctly
// registers gsdw serve as the MCP server.
func TestMcpJsonValid(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, ".mcp.json")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var mcpConfig map[string]any
	if err := json.Unmarshal(data, &mcpConfig); err != nil {
		t.Fatalf("unmarshal .mcp.json: %v", err)
	}

	// Top-level mcpServers key
	servers, ok := mcpConfig["mcpServers"].(map[string]any)
	if !ok {
		t.Fatalf(".mcp.json missing 'mcpServers' object, got %T", mcpConfig["mcpServers"])
	}

	// gsd-wired server entry
	entry, ok := servers["gsd-wired"].(map[string]any)
	if !ok {
		t.Fatalf(".mcp.json mcpServers missing 'gsd-wired' entry, got %T", servers["gsd-wired"])
	}

	// command must be gsdw
	if got, want := entry["command"], "gsdw"; got != want {
		t.Errorf(".mcp.json mcpServers.gsd-wired.command = %q, want %q", got, want)
	}

	// args must contain "serve"
	args, ok := entry["args"].([]any)
	if !ok {
		t.Fatalf(".mcp.json mcpServers.gsd-wired.args is not an array, got %T", entry["args"])
	}
	foundServe := false
	for _, arg := range args {
		if arg == "serve" {
			foundServe = true
			break
		}
	}
	if !foundServe {
		t.Errorf(".mcp.json mcpServers.gsd-wired.args does not contain 'serve', got %v", args)
	}
}

// TestHooksJsonValid verifies hooks/hooks.json exists at the repo root and
// registers all four hook events, each pointing to a gsdw hook command.
func TestHooksJsonValid(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "hooks", "hooks.json")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var hooksConfig map[string]any
	if err := json.Unmarshal(data, &hooksConfig); err != nil {
		t.Fatalf("unmarshal hooks/hooks.json: %v", err)
	}

	// Top-level hooks key
	hooks, ok := hooksConfig["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks/hooks.json missing 'hooks' object, got %T", hooksConfig["hooks"])
	}

	// All four required event names must be present
	requiredEvents := []string{"SessionStart", "PreCompact", "PreToolUse", "PostToolUse"}
	for _, eventName := range requiredEvents {
		entries, ok := hooks[eventName].([]any)
		if !ok {
			t.Errorf("hooks/hooks.json missing event %q or wrong type (got %T)", eventName, hooks[eventName])
			continue
		}

		// Each event should have at least one hook entry with a command containing "gsdw hook"
		foundGsdwHook := false
		for _, entry := range entries {
			entryMap, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			innerHooks, ok := entryMap["hooks"].([]any)
			if !ok {
				continue
			}
			for _, h := range innerHooks {
				hMap, ok := h.(map[string]any)
				if !ok {
					continue
				}
				cmd, _ := hMap["command"].(string)
				if len(cmd) >= 9 && cmd[:9] == "gsdw hook" {
					foundGsdwHook = true
				}
			}
		}
		if !foundGsdwHook {
			t.Errorf("hooks/hooks.json event %q does not reference 'gsdw hook' command", eventName)
		}
	}
}

// TestHooksNotInsideClaudePlugin verifies that hooks/hooks.json is NOT placed
// inside .claude-plugin/ — a common mistake per RESEARCH.md Pitfall 3.
func TestHooksNotInsideClaudePlugin(t *testing.T) {
	root := repoRoot(t)
	wrongPath := filepath.Join(root, ".claude-plugin", "hooks", "hooks.json")

	if _, err := os.Stat(wrongPath); err == nil {
		t.Errorf("hooks/hooks.json must NOT be inside .claude-plugin/ — found at %s (common mistake per Pitfall 3)", wrongPath)
	}
}
