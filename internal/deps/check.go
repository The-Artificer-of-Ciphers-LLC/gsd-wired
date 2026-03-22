package deps

// Status represents the detection result for a dependency.
type Status string

const (
	StatusOK   Status = "ok"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

// Dep holds the detection result for a single dependency.
type Dep struct {
	Name        string // human-readable name (e.g. "bd", "Go")
	Binary      string // binary name looked up (e.g. "bd", "go")
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

// CheckAll detects bd, dolt, Go, and container runtime (docker/podman).
// It returns a CheckResult with the status of each dependency.
// For bd and dolt, it falls back to $(go env GOPATH)/bin when not on PATH.
func CheckAll() CheckResult {
	panic("not implemented")
}
