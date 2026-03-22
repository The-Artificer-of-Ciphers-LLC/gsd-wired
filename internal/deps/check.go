package deps

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Status represents the detection result for a dependency.
type Status string

const (
	StatusOK   Status = "ok"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

// Dep holds the detection result for a single dependency.
type Dep struct {
	Name        string // human-readable name (e.g. "bd", "Go", "Container Runtime")
	Binary      string // binary name looked up (e.g. "bd", "go", "docker")
	Status      Status // ok, warn, or fail
	Version     string // version string extracted from binary output
	Path        string // resolved path to binary
	InstallHelp string // actionable install instructions when status is fail
}

// CheckResult holds the full result of CheckAll.
type CheckResult struct {
	Deps  []Dep
	AllOK bool
}

// installHelp contains actionable install instructions per binary.
var installHelp = map[string]string{
	"bd":        "go install github.com/steveyegge/beads/cmd/bd@latest\n  or: brew install steveyegge/tap/beads",
	"dolt":      "brew install dolthub/tap/dolt\n  or: curl -L https://github.com/dolthub/dolt/releases/latest/download/install.sh | bash",
	"go":        "brew install go\n  or: download from https://go.dev/dl/",
	"docker":    "brew install --cask docker\n  or: brew install podman",
	"container": "Requires macOS 26 Tahoe + Apple Silicon. See https://github.com/apple/container",
}

// CheckAll detects bd, dolt, Go, and container runtime (docker/podman).
// It returns a CheckResult with the status of each dependency.
// For bd and dolt, it falls back to $(go env GOPATH)/bin when not on PATH.
func CheckAll() CheckResult {
	var result CheckResult

	// Check bd (with GOPATH/bin fallback)
	bd := checkBinary("bd", "bd", installHelp["bd"], true)
	result.Deps = append(result.Deps, bd)

	// Check dolt (with GOPATH/bin fallback)
	dolt := checkBinary("dolt", "dolt", installHelp["dolt"], true)
	result.Deps = append(result.Deps, dolt)

	// Check Go (no GOPATH fallback — go itself resolves GOPATH)
	goDep := checkBinary("Go", "go", installHelp["go"], false)
	result.Deps = append(result.Deps, goDep)

	// Check container runtime: try docker first, then podman
	crt := checkContainerRuntime()
	result.Deps = append(result.Deps, crt)

	// Compute AllOK
	result.AllOK = true
	for _, d := range result.Deps {
		if d.Status == StatusFail {
			result.AllOK = false
			break
		}
	}

	return result
}

// checkBinary checks for a binary by name, with optional GOPATH/bin fallback.
// Returns a populated Dep struct with status, version, path, and install help.
func checkBinary(name, binary, help string, tryGoPath bool) Dep {
	dep := Dep{
		Name:   name,
		Binary: binary,
	}

	// First try PATH via exec.LookPath.
	p, err := exec.LookPath(binary)
	if err != nil && tryGoPath {
		// Fallback: check GOPATH/bin.
		p, err = lookInGoPath(binary)
	}

	if err != nil {
		dep.Status = StatusFail
		dep.InstallHelp = help
		return dep
	}

	dep.Path = p
	dep.Version = extractVersion(binary, p)
	dep.Status = StatusOK
	return dep
}

// checkContainerRuntime tries Apple Container first (macOS 26+), then docker, then podman.
// Returns a Dep with Name="Container Runtime".
func checkContainerRuntime() Dep {
	dep := Dep{Name: "Container Runtime"}

	for _, binary := range []string{"container", "docker", "podman"} {
		p, err := exec.LookPath(binary)
		if err != nil {
			continue
		}
		// Apple Container requires macOS 26+. Skip on older versions.
		if binary == "container" && !isMacOS26OrNewer() {
			continue
		}
		dep.Binary = binary
		dep.Path = p
		dep.Version = extractVersion(binary, p)
		dep.Status = StatusOK
		return dep
	}

	dep.Binary = "docker"
	dep.Status = StatusFail
	dep.InstallHelp = installHelp["docker"]
	return dep
}

// isMacOS26OrNewer returns true if the current macOS major version is >= 26.
// GSDW_MOCK_MACOS_MAJOR env var overrides the version check for testing.
func isMacOS26OrNewer() bool {
	// Test injection point.
	if mock := os.Getenv("GSDW_MOCK_MACOS_MAJOR"); mock != "" {
		major, err := strconv.Atoi(mock)
		if err == nil {
			return major >= 26
		}
	}

	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err != nil {
		return false
	}
	ver := strings.TrimSpace(string(out))
	parts := strings.SplitN(ver, ".", 2)
	if len(parts) < 1 {
		return false
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	return major >= 26
}

// lookInGoPath attempts to find binary in $(go env GOPATH)/bin.
// Returns the full path if found, or an error if not.
// Per SETUP-04: avoids false negatives after `go install`.
func lookInGoPath(binary string) (string, error) {
	// Resolve go via PATH so tests can inject a fake go binary.
	goPath, err := exec.LookPath("go")
	if err != nil {
		return "", fmt.Errorf("go not found on PATH, cannot determine GOPATH: %w", err)
	}
	out, err := exec.Command(goPath, "env", "GOPATH").Output()
	if err != nil {
		return "", fmt.Errorf("cannot determine GOPATH: %w", err)
	}
	gopath := strings.TrimSpace(string(out))
	if gopath == "" {
		return "", fmt.Errorf("GOPATH is empty")
	}

	candidate := filepath.Join(gopath, "bin", binary)
	if _, err := os.Stat(candidate); err != nil {
		return "", fmt.Errorf("%s not found in GOPATH/bin (%s)", binary, candidate)
	}
	return candidate, nil
}

// extractVersion runs `binary version` and returns the version string.
// Returns empty string if the command fails or produces no output.
func extractVersion(binary, path string) string {
	var args []string
	if binary == "go" {
		args = []string{"version"}
	} else {
		args = []string{"version"}
	}

	var stdout bytes.Buffer
	cmd := exec.Command(path, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return ""
	}

	line := strings.TrimSpace(stdout.String())
	// Most tools output "name version X.Y.Z ..." — extract the version token.
	// Try to find a token that looks like a version number (contains a dot).
	parts := strings.Fields(line)
	for i, part := range parts {
		if part == "version" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	// Fallback: return the full line.
	if line != "" {
		return line
	}
	return ""
}
